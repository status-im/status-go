package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/waku-org/sds-go-bindings/sds"

	"github.com/status-im/status-go/internal/crypto"
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
	communityChannel := strings.HasPrefix(communityID, "0x") && len(communityID) == 68

	if len(message.Content) == 0 {
		if !communityChannel {
			return true, nil
		}
		if message.ChannelID != reliability.BuildChannelID(communityID) {
			return false, nil
		}
		community, err := m.communitiesManager.GetByIDString(communityID)
		if err != nil {
			return false, err
		}
		sender, groupID, err := reliability.ParseSDSAuthorizationContext(message.AuthorizationContext)
		if err != nil {
			return false, err
		}
		if community.Encrypted() && !bytes.Equal(community.ID(), groupID) {
			return false, nil
		}
		return isAuthorizedSDSCommunitySender(community, sender), nil
	}

	application, err := common.ParseApplicationLayer(message.Content)
	if err != nil {
		return false, fmt.Errorf("invalid SDS application message: %w", err)
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
		return false, fmt.Errorf("invalid SDS chat message: %w", err)
	}
	if chatMessage.GetMessageType() != protobuf.MessageType_COMMUNITY_CHAT {
		return !communityChannel, nil
	}
	if application.SigPubKey == nil {
		return false, fmt.Errorf("SDS community message has no application signature")
	}
	chat, ok := m.allChats.Load(chatMessage.GetChatId())
	if !ok || chat.ChatType != ChatTypeCommunityChat || chat.CommunityID == "" {
		return false, fmt.Errorf("SDS community message targets an unknown community chat")
	}
	if message.ChannelID != reliability.BuildChannelID(chat.CommunityID) {
		return false, nil
	}
	community, err := m.communitiesManager.GetByIDString(chat.CommunityID)
	if err != nil {
		return false, err
	}
	// AuthorizationContext is the Waku-authenticated signer of the full payload,
	// including SDS IDs, dependencies, and acknowledgements. Require that signer
	// to be a community participant, while allowing member repair forwarding.
	transportSender, _, err := reliability.ParseSDSAuthorizationContext(message.AuthorizationContext)
	if err != nil {
		return false, err
	}
	if !isAuthorizedSDSCommunitySender(community, transportSender) {
		return false, nil
	}
	if community.IsBanned(application.SigPubKey) {
		return false, nil
	}
	return community.CanPost(application.SigPubKey, chat.CommunityChatID(), application.Type)
}

func isAuthorizedSDSCommunitySender(community *communities.Community, sender *ecdsa.PublicKey) bool {
	return !community.IsBanned(sender) &&
		(community.HasMember(sender) || crypto.IsPubKeyEqual(sender, community.ControlNode()))
}
