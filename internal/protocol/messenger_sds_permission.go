package protocol

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

type sdsChannelScope int

const (
	sdsChannelScopeInvalid sdsChannelScope = iota
	sdsChannelScopeCommunityChat
	sdsChannelScopeControl
	sdsChannelScopeCommunityTopic
)

func (m *Messenger) validateSDSCommunityMessage(
	payload []byte,
	channelID string,
	envelopeSigner *ecdsa.PublicKey,
	envelopeTopic messagingtypes.ContentTopic,
) (messagingtypes.SDSMessageDisposition, error) {
	if len(payload) == 0 {
		return messagingtypes.SDSMessageDispositionReject, fmt.Errorf("decode SDS application message: empty payload")
	}
	applicationMessage, err := protobuf.Unmarshal(payload)
	if err != nil {
		return messagingtypes.SDSMessageDispositionReject, fmt.Errorf("decode SDS application message: %w", err)
	}

	communityID, scope := sdsTargetCommunityID(channelID)
	if scope == sdsChannelScopeInvalid {
		if m.isCommunityTopic(envelopeTopic) {
			return messagingtypes.SDSMessageDispositionReject, nil
		}
		_, messageType, supported, err := sdsCommunityChatScope(applicationMessage)
		if err != nil {
			return messagingtypes.SDSMessageDispositionReject, err
		}
		if supported && messageType == protobuf.MessageType_COMMUNITY_CHAT {
			return messagingtypes.SDSMessageDispositionReject, nil
		}
		return messagingtypes.SDSMessageDispositionProcess, nil
	}

	expectedTopicName := ""
	switch scope {
	case sdsChannelScopeCommunityChat:
		expectedTopicName = communityID + "-memberUpdate"
	case sdsChannelScopeControl:
		expectedTopicName = sdsControlTopicName(communityID)
	case sdsChannelScopeCommunityTopic:
		expectedTopicName = communityID
	}
	if expectedTopicName == "" {
		return messagingtypes.SDSMessageDispositionReject, nil
	}
	expectedTopic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(expectedTopicName))
	if envelopeTopic != expectedTopic {
		return messagingtypes.SDSMessageDispositionReject, nil
	}

	applicationSigner, err := common.RecoverKey(applicationMessage)
	if err != nil || applicationSigner == nil {
		if err == nil {
			return messagingtypes.SDSMessageDispositionReject, nil
		}
		return messagingtypes.SDSMessageDispositionReject, fmt.Errorf("recover SDS application signer: %w", err)
	}
	if envelopeSigner == nil || !crypto.IsPubKeyEqual(envelopeSigner, applicationSigner) {
		return messagingtypes.SDSMessageDispositionReject, nil
	}

	switch scope {
	case sdsChannelScopeCommunityChat:
		chatID, messageType, supported, err := sdsCommunityChatScope(applicationMessage)
		if err != nil {
			return messagingtypes.SDSMessageDispositionReject, err
		}
		if !supported || messageType != protobuf.MessageType_COMMUNITY_CHAT || chatID == "" {
			return messagingtypes.SDSMessageDispositionReject, nil
		}
		chat, ok := m.allChats.Load(chatID)
		if !ok || chat.ChatType != ChatTypeCommunityChat || chat.CommunityID != communityID {
			return messagingtypes.SDSMessageDispositionReject, nil
		}
		allowed, err := m.communitiesManager.CanPost(
			envelopeSigner,
			communityID,
			chat.CommunityChatID(),
			applicationMessage.Type,
		)
		if err != nil {
			return messagingtypes.SDSMessageDispositionReject, err
		}
		if !allowed {
			return messagingtypes.SDSMessageDispositionReject, nil
		}
		return messagingtypes.SDSMessageDispositionProcess, nil
	case sdsChannelScopeControl:
		targetCommunityID, supported, err := sdsControlMessageCommunityID(applicationMessage)
		if err != nil {
			return messagingtypes.SDSMessageDispositionReject, err
		}
		if !supported || targetCommunityID != communityID {
			return messagingtypes.SDSMessageDispositionReject, nil
		}
		return messagingtypes.SDSMessageDispositionApplicationOnly, nil
	case sdsChannelScopeCommunityTopic:
		targetCommunityID, supported, err := sdsCommunityTopicMessageCommunityID(applicationMessage)
		if err != nil {
			return messagingtypes.SDSMessageDispositionReject, err
		}
		if !supported || targetCommunityID != communityID {
			return messagingtypes.SDSMessageDispositionReject, nil
		}
		return messagingtypes.SDSMessageDispositionApplicationOnly, nil
	}

	return messagingtypes.SDSMessageDispositionReject, nil
}

func (m *Messenger) isCommunityTopic(topic messagingtypes.ContentTopic) bool {
	isCommunityTopic := false
	m.allChats.Range(func(_ string, chat *Chat) bool {
		if chat.ChatType != ChatTypeCommunityChat || chat.CommunityID == "" {
			return true
		}

		for _, topicName := range []string{
			chat.CommunityID + "-memberUpdate",
			chat.CommunityID,
			sdsControlTopicName(chat.CommunityID),
		} {
			if topicName != "" && topic == messagingtypes.BytesToContentTopic(messaging.ToContentTopic(topicName)) {
				isCommunityTopic = true
				return false
			}
		}
		return true
	})
	return isCommunityTopic
}

func sdsTargetCommunityID(channelID string) (string, sdsChannelScope) {
	communityID, suffix, hasSuffix := strings.Cut(channelID, ":")
	encodedID, err := cryptotypes.DecodeHex(communityID)
	if err != nil || len(encodedID) != 33 {
		return "", sdsChannelScopeInvalid
	}
	publicKey, err := crypto.DecompressPubkey(encodedID)
	if err != nil {
		return "", sdsChannelScopeInvalid
	}
	canonicalID := cryptotypes.EncodeHex(crypto.CompressPubkey(publicKey))
	if communityID != canonicalID {
		return "", sdsChannelScopeInvalid
	}
	if !hasSuffix {
		return canonicalID, sdsChannelScopeControl
	}
	switch suffix {
	case communityID + "-memberUpdate":
		return canonicalID, sdsChannelScopeCommunityChat
	case communityID:
		return canonicalID, sdsChannelScopeCommunityTopic
	default:
		return "", sdsChannelScopeInvalid
	}
}

func sdsControlTopicName(communityID string) string {
	encodedID, err := cryptotypes.DecodeHex(communityID)
	if err != nil {
		return ""
	}
	publicKey, err := crypto.DecompressPubkey(encodedID)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(cryptotypes.EncodeHex(crypto.FromECDSAPub(publicKey)), "0x")
}

func sdsControlMessageCommunityID(
	message *protobuf.ApplicationMetadataMessage,
) (string, bool, error) {
	var communityID []byte
	switch message.Type {
	case protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN:
		payload := &protobuf.CommunityRequestToJoin{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS community join request: %w", err)
		}
		communityID = payload.GetCommunityId()
	case protobuf.ApplicationMetadataMessage_COMMUNITY_CANCEL_REQUEST_TO_JOIN:
		payload := &protobuf.CommunityCancelRequestToJoin{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS community join cancellation: %w", err)
		}
		communityID = payload.GetCommunityId()
	case protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_LEAVE:
		payload := &protobuf.CommunityRequestToLeave{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS community leave request: %w", err)
		}
		communityID = payload.GetCommunityId()
	case protobuf.ApplicationMetadataMessage_COMMUNITY_EDIT_SHARED_ADDRESSES:
		payload := &protobuf.CommunityEditSharedAddresses{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS shared-address update: %w", err)
		}
		communityID = payload.GetCommunityId()
	case protobuf.ApplicationMetadataMessage_COMMUNITY_REEVALUATE_PERMISSIONS_REQUEST:
		payload := &protobuf.CommunityReevaluatePermissionsRequest{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS permission reevaluation request: %w", err)
		}
		communityID = payload.GetCommunityId()
	case protobuf.ApplicationMetadataMessage_COMMUNITY_ENCRYPTION_KEYS_REQUEST:
		payload := &protobuf.CommunityEncryptionKeysRequest{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS encryption keys request: %w", err)
		}
		communityID = payload.GetCommunityId()
	case protobuf.ApplicationMetadataMessage_COMMUNITY_SHARED_ADDRESSES_REQUEST:
		payload := &protobuf.CommunitySharedAddressesRequest{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS shared-address request: %w", err)
		}
		communityID = payload.GetCommunityId()
	default:
		return "", false, nil
	}
	return sdsCommunityIDFromBytes(communityID)
}

func sdsCommunityTopicMessageCommunityID(
	message *protobuf.ApplicationMetadataMessage,
) (string, bool, error) {
	switch message.Type {
	case protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION:
		payload := &protobuf.CommunityDescription{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS community description: %w", err)
		}
		communityID, err := cryptotypes.DecodeHex(payload.GetID())
		if err != nil || len(communityID) != 33 {
			return "", false, nil
		}
		publicKey, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			return "", false, nil
		}
		return cryptotypes.EncodeHex(crypto.CompressPubkey(publicKey)), true, nil
	case protobuf.ApplicationMetadataMessage_COMMUNITY_EVENTS_MESSAGE:
		payload := &protobuf.CommunityEventsMessage{}
		if err := proto.Unmarshal(message.Payload, payload); err != nil {
			return "", false, fmt.Errorf("decode SDS community events: %w", err)
		}
		return sdsCommunityIDFromBytes(payload.GetCommunityId())
	default:
		return "", false, nil
	}
}

func sdsCommunityIDFromBytes(communityID []byte) (string, bool, error) {
	if len(communityID) != 33 {
		return "", false, nil
	}
	publicKey, err := crypto.DecompressPubkey(communityID)
	if err != nil {
		return "", false, nil
	}
	return cryptotypes.EncodeHex(crypto.CompressPubkey(publicKey)), true, nil
}

func sdsCommunityChatScope(
	message *protobuf.ApplicationMetadataMessage,
) (chatID string, messageType protobuf.MessageType, supported bool, err error) {
	switch message.Type {
	case protobuf.ApplicationMetadataMessage_CHAT_MESSAGE:
		payload := &protobuf.ChatMessage{}
		err = proto.Unmarshal(message.Payload, payload)
		if err == nil {
			chatID, messageType, supported = payload.GetChatId(), payload.GetMessageType(), true
		}
	case protobuf.ApplicationMetadataMessage_EMOJI_REACTION:
		payload := &protobuf.EmojiReaction{}
		err = proto.Unmarshal(message.Payload, payload)
		if err == nil {
			chatID, messageType, supported = payload.GetChatId(), payload.GetMessageType(), true
		}
	case protobuf.ApplicationMetadataMessage_PIN_MESSAGE:
		payload := &protobuf.PinMessage{}
		err = proto.Unmarshal(message.Payload, payload)
		if err == nil {
			chatID, messageType, supported = payload.GetChatId(), payload.GetMessageType(), true
		}
	case protobuf.ApplicationMetadataMessage_THREAD_METADATA:
		payload := &protobuf.ThreadMetadata{}
		err = proto.Unmarshal(message.Payload, payload)
		if err == nil {
			chatID, messageType, supported = payload.GetChatId(), payload.GetMessageType(), true
		}
	case protobuf.ApplicationMetadataMessage_EDIT_MESSAGE:
		payload := &protobuf.EditMessage{}
		err = proto.Unmarshal(message.Payload, payload)
		if err == nil {
			chatID, messageType, supported = payload.GetChatId(), payload.GetMessageType(), true
		}
	case protobuf.ApplicationMetadataMessage_DELETE_MESSAGE:
		payload := &protobuf.DeleteMessage{}
		err = proto.Unmarshal(message.Payload, payload)
		if err == nil {
			chatID, messageType, supported = payload.GetChatId(), payload.GetMessageType(), true
		}
	}
	if err != nil {
		return "", 0, false, fmt.Errorf("decode SDS community message payload: %w", err)
	}
	return chatID, messageType, supported, nil
}
