package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/waku-org/sds-go-bindings/sds"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
)

type communitySDSAuthorizer struct {
	messenger *Messenger
}

func (v communitySDSAuthorizer) Authorize(message sds.AuthorizationMessage) (bool, error) {
	m := v.messenger
	communityID, _, _ := strings.Cut(message.ChannelID, ":")
	// Community IDs are 0x-prefixed, hex-encoded compressed public keys.
	communityChannel := strings.HasPrefix(communityID, "0x") && len(communityID) == pkStringLength

	if len(message.Content) == 0 {
		if !communityChannel {
			return true, nil
		}
		if message.ChannelID != reliability.BuildChannelID(communityID) {
			return false, nil
		}
		sender, groupID, err := reliability.ParseSDSAuthorizationContext(message.AuthorizationContext)
		if err != nil {
			return false, nil
		}
		community, err := v.loadCommunity(communityID, communitySDSSyncTopic(communityID), false)
		if err != nil || community == nil {
			return false, err
		}
		if community.Encrypted() && !bytes.Equal(community.ID(), groupID) {
			return false, nil
		}
		return isAuthorizedSDSCommunitySender(community, sender), nil
	}

	application, err := common.ParseApplicationLayer(message.Content)
	if err != nil {
		return false, nil
	}
	var chatMessage interface {
		proto.Message
		GetChatId() string
		GetMessageType() protobuf.MessageType
	}
	switch application.Type {
	case protobuf.ApplicationMetadataMessage_CHAT_MESSAGE:
		chatMessage = &protobuf.ChatMessage{}
	case protobuf.ApplicationMetadataMessage_EMOJI_REACTION:
		chatMessage = &protobuf.EmojiReaction{}
	case protobuf.ApplicationMetadataMessage_PIN_MESSAGE:
		chatMessage = &protobuf.PinMessage{}
	case protobuf.ApplicationMetadataMessage_EDIT_MESSAGE:
		chatMessage = &protobuf.EditMessage{}
	case protobuf.ApplicationMetadataMessage_DELETE_MESSAGE:
		chatMessage = &protobuf.DeleteMessage{}
	case protobuf.ApplicationMetadataMessage_THREAD_METADATA:
		chatMessage = &protobuf.ThreadMetadata{}
	case protobuf.ApplicationMetadataMessage_DELETE_THREAD:
		chatMessage = &protobuf.DeleteThread{}
	default:
		// Control messages use their own authorization outside SDS.
		return !communityChannel, nil
	}
	if err := proto.Unmarshal(application.Payload, chatMessage); err != nil {
		return false, nil
	}
	if chatMessage.GetMessageType() != protobuf.MessageType_COMMUNITY_CHAT {
		return !communityChannel, nil
	}
	if application.SigPubKey == nil || !communityChannel || message.ChannelID != reliability.BuildChannelID(communityID) {
		return false, nil
	}
	chatID := chatMessage.GetChatId()
	if len(chatID) <= pkStringLength || !strings.HasPrefix(chatID, communityID) {
		return false, nil
	}
	chat, knownChat := m.allChats.Load(chatID)
	if knownChat && (chat.ChatType != ChatTypeCommunityChat || chat.CommunityID != communityID) {
		return false, nil
	}

	// The Waku signer authenticated the full SDS payload. An authorized member
	// may forward another member's signed application message during repair.
	transportSender, _, err := reliability.ParseSDSAuthorizationContext(message.AuthorizationContext)
	if err != nil {
		return false, nil
	}
	community, err := v.loadCommunity(communityID, communityID, knownChat)
	if err != nil || community == nil {
		return false, err
	}
	if !isAuthorizedSDSCommunitySender(community, transportSender) || community.IsBanned(application.SigPubKey) {
		return false, nil
	}
	if !knownChat {
		if community.Joined() || community.Spectated() {
			return false, fmt.Errorf("SDS community chat %q is not loaded yet", chatID)
		}
		return false, nil
	}
	return community.CanPost(application.SigPubKey, chat.CommunityChatID(), application.Type)
}

// loadCommunity permits a missing record to be retried when a local chat or
// subscription indicates that the client expects state for this community.
func (v communitySDSAuthorizer) loadCommunity(id, filterID string, knownChat bool) (*communities.Community, error) {
	decodedID, err := cryptotypes.DecodeHex(id)
	if err != nil {
		return nil, nil
	}
	community, err := v.messenger.communitiesManager.GetByID(decodedID)
	if errors.Is(err, communities.ErrOrgNotFound) && !knownChat && v.messenger.messaging.ChatFilterByChatID(filterID) == nil {
		return nil, nil
	}
	return community, err
}

func isAuthorizedSDSCommunitySender(community *communities.Community, sender *ecdsa.PublicKey) bool {
	return !community.IsBanned(sender) &&
		(community.HasMember(sender) || crypto.IsPubKeyEqual(sender, community.ControlNode()))
}
