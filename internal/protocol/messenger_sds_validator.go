package protocol

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/waku-org/sds-go-bindings/sds"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
)

type communitySDSValidator struct {
	messenger *Messenger
}

func (v communitySDSValidator) Validate(message sds.ValidationMessage) (bool, error) {
	m := v.messenger
	communityID, topic, _ := strings.Cut(message.ChannelID, ":")
	// Community IDs are 0x-prefixed, hex-encoded compressed public keys.
	communityChannel := strings.HasPrefix(communityID, "0x") && len(communityID) == 68

	if len(message.Content) == 0 {
		if !communityChannel {
			return true, nil
		}
		community, err := m.communitiesManager.GetByIDString(communityID)
		if err != nil {
			return false, err
		}
		if topic != "" && topic != community.UniversalChatID() {
			return false, nil
		}
		sender, err := crypto.UnmarshalPubkey(message.ValidationContext)
		if err != nil {
			return false, fmt.Errorf("invalid SDS sync sender: %w", err)
		}
		return !community.IsBanned(sender) &&
			(community.HasMember(sender) || crypto.IsPubKeyEqual(sender, community.ControlNode())), nil
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
	default:
		// Older clients wrap control traffic on the separate bare-ID channel.
		// Never allow it to authorize metadata on the chat reliability channel.
		if communityChannel && topic == "" && common.IsCommunityControlMessage(application.Type) {
			if application.SigPubKey == nil {
				return false, fmt.Errorf("legacy SDS community control message has no application signature")
			}
			return true, nil
		}
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
	community, err := m.communitiesManager.GetByIDString(chat.CommunityID)
	if err != nil {
		return false, err
	}
	if message.ChannelID != chat.CommunityID &&
		message.ChannelID != reliability.BuildChannelID(chat.CommunityID, community.UniversalChatID()) {
		return false, nil
	}
	if community.IsBanned(application.SigPubKey) {
		return false, nil
	}
	return community.CanPost(application.SigPubKey, chat.CommunityChatID(), application.Type)
}
