package common

import (
	"context"
	"crypto/ecdsa"
	"math/rand"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	mvdsnode "github.com/status-im/mvds/node"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"

	utils "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/crypto"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/adapters"
	messagingevents "github.com/status-im/status-go/messaging/events"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/encryption/sharedsecret"
	"github.com/status-im/status-go/messaging/layers/reliability"
	"github.com/status-im/status-go/messaging/layers/segmentation"
	"github.com/status-im/status-go/messaging/layers/transport"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/messaging/waku/types"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

// Whisper message properties.
const (
	maxMessageSenderEphemeralKeys = 3
)

// RekeyCompatibility indicates whether we should be sending
// keys in 1-to-1 messages as well as in the newer format
var RekeyCompatibility = true

type MessageSender struct {
	identity    *ecdsa.PrivateKey
	transport   *transport.Transport
	segmenter   *segmentation.Segmenter
	protocol    *encryption.Protocol
	reliability *reliability.Reliability
	logger      *zap.Logger
	persistence messagingtypes.MessageSenderPersistence
	publisher   *pubsub.Publisher

	// ephemeralKeys is a map that contains the ephemeral keys of the client, used
	// to decrypt messages
	ephemeralKeys      map[string]*ecdsa.PrivateKey
	ephemeralKeysMutex sync.Mutex
}

func NewMessageSender(
	identity *ecdsa.PrivateKey,
	persistence messagingtypes.MessageSenderPersistence,
	mvdsPersistence mvdsnode.Persistence,
	segmentationPersistence segmentation.Persistence,
	transport *transport.Transport,
	encryptor *encryption.Protocol,
	logger *zap.Logger,
) (*MessageSender, error) {
	logger = logger.Named("message_sender")

	p := &MessageSender{
		identity:      identity,
		transport:     transport,
		segmenter:     segmentation.NewSegmenter(segmentationPersistence, logger),
		protocol:      encryptor,
		reliability:   reliability.NewReliability(mvdsPersistence, identity, logger),
		persistence:   persistence,
		publisher:     pubsub.NewPublisher(),
		logger:        logger,
		ephemeralKeys: make(map[string]*ecdsa.PrivateKey),
	}

	return p, nil
}

func (s *MessageSender) Stop() {
	s.publisher.Close()
	s.StopReliability()
}

// SendPrivate takes encoded data, encrypts it and sends through the wire.
func (s *MessageSender) SendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage *messagingtypes.RawMessage,
) ([]byte, error) {
	s.logger.Debug(
		"sending a private message",
		zap.String("public-key", cryptotypes.EncodeHex(crypto.FromECDSAPub(recipient))),
		zap.String("site", "SendPrivate"),
	)
	// Currently we don't support sending through datasync and setting custom waku fields,
	// as the datasync interface is not rich enough to propagate that information, so we
	// would have to add some complexity to handle this.
	if rawMessage.ResendType == messagingtypes.ResendTypeDataSync && (rawMessage.Sender != nil || rawMessage.SkipEncryptionLayer || rawMessage.SendOnPersonalTopic) {
		return nil, errors.New("setting identity, skip-encryption or personal topic and datasync not supported")
	}

	// Set sender identity if not specified
	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	return s.sendPrivate(ctx, recipient, rawMessage)
}

// SendCommunityMessage takes encoded data, encrypts it and sends through the wire
// using the community topic and their key
func (s *MessageSender) SendCommunityMessage(
	ctx context.Context,
	rawMessage *messagingtypes.RawMessage,
) ([]byte, error) {
	s.logger.Debug(
		"sending a community message",
		zap.String("communityId", cryptotypes.EncodeHex(rawMessage.CommunityID)),
		zap.String("site", "SendCommunityMessage"),
	)
	rawMessage.Sender = s.identity

	return s.sendCommunity(ctx, rawMessage)
}

// SendPubsubTopicKey sends the protected topic key for a community to a list of recipients
func (s *MessageSender) SendPubsubTopicKey(
	ctx context.Context,
	rawMessage *messagingtypes.RawMessage,
) ([]byte, error) {
	s.logger.Debug(
		"sending the protected topic key for a community",
		zap.String("communityId", cryptotypes.EncodeHex(rawMessage.CommunityID)),
		zap.String("site", "SendPubsubTopicKey"),
	)
	rawMessage.Sender = s.identity
	messageID, err := s.getMessageID(rawMessage)
	if err != nil {
		return nil, err
	}

	if err = s.setMessageID(messageID, rawMessage); err != nil {
		return nil, err
	}

	// Notify before dispatching, otherwise the dispatch subscription might happen
	// earlier than the scheduled
	s.notifyOnScheduledMessage(nil, rawMessage)

	// Send to each recipients
	for _, recipient := range rawMessage.Recipients {
		_, err = s.sendPrivate(ctx, recipient, rawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send message")
		}
	}
	return messageID, nil

}

// SendGroup takes encoded data, encrypts it and sends through the wire,
// always return the messageID
func (s *MessageSender) SendGroup(
	ctx context.Context,
	recipients []*ecdsa.PublicKey,
	rawMessage messagingtypes.RawMessage,
) ([]byte, error) {
	s.logger.Debug(
		"sending a private group message",
		zap.String("site", "SendGroup"),
	)
	// Set sender if not specified
	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	// Calculate messageID first and set on raw message
	messageID, err := s.getMessageID(&rawMessage)
	if err != nil {
		return nil, err
	}

	if err = s.setMessageID(messageID, &rawMessage); err != nil {
		return nil, err
	}

	// We call it only once, and we nil the function after so it doesn't get called again
	if rawMessage.BeforeDispatch != nil {
		if err := rawMessage.BeforeDispatch(&rawMessage); err != nil {
			return nil, err
		}
	}

	// Send to each recipients
	for _, recipient := range recipients {
		_, err = s.sendPrivate(ctx, recipient, &rawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send message")
		}
	}
	return messageID, nil
}

func (s *MessageSender) getMessageID(rawMessage *messagingtypes.RawMessage) (cryptotypes.HexBytes, error) {
	wrappedMessage, err := s.wrapMessageV1(rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	return messageID, nil
}

func (s *MessageSender) ValidateRawMessage(rawMessage *messagingtypes.RawMessage) error {
	id, err := s.getMessageID(rawMessage)
	if err != nil {
		return err
	}
	messageID := cryptotypes.EncodeHex(id)

	return s.validateMessageID(messageID, rawMessage)

}

func (s *MessageSender) validateMessageID(messageID string, rawMessage *messagingtypes.RawMessage) error {
	if len(rawMessage.ID) > 0 && rawMessage.ID != messageID {
		s.logger.Error("failed to validate message ID, RawMessage content was modified",
			zap.String("prevID", rawMessage.ID),
			zap.String("newID", messageID),
			zap.Any("contentType", rawMessage.MessageType))
		return messagingtypes.ErrModifiedRawMessage
	}
	return nil
}

func (s *MessageSender) setMessageID(messageID cryptotypes.HexBytes, rawMessage *messagingtypes.RawMessage) error {
	msgID := cryptotypes.EncodeHex(messageID)

	if err := s.validateMessageID(msgID, rawMessage); err != nil {
		return err
	}

	rawMessage.ID = msgID
	return nil
}

func ShouldCommunityMessageBeEncrypted(msgType protobuf.ApplicationMetadataMessage_Type) bool {
	return msgType == protobuf.ApplicationMetadataMessage_CHAT_MESSAGE ||
		msgType == protobuf.ApplicationMetadataMessage_EDIT_MESSAGE ||
		msgType == protobuf.ApplicationMetadataMessage_DELETE_MESSAGE ||
		msgType == protobuf.ApplicationMetadataMessage_PIN_MESSAGE ||
		msgType == protobuf.ApplicationMetadataMessage_EMOJI_REACTION
}

// sendCommunity sends a message that's to be sent in a community
// If it's a chat message, it will go to the respective topic derived by the
// chat id, if it's not a chat message, it will go to the community topic.
func (s *MessageSender) sendCommunity(
	ctx context.Context,
	rawMessage *messagingtypes.RawMessage,
) ([]byte, error) {
	s.logger.Debug("sending community message", zap.String("recipient", cryptotypes.EncodeHex(crypto.FromECDSAPub(&rawMessage.Sender.PublicKey))))

	// Set sender
	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	messageID, err := s.getMessageID(rawMessage)
	if err != nil {
		return nil, err
	}

	if err = s.setMessageID(messageID, rawMessage); err != nil {
		return nil, err
	}

	if rawMessage.BeforeDispatch != nil {
		if err := rawMessage.BeforeDispatch(rawMessage); err != nil {
			return nil, err
		}
	}
	// Notify before dispatching, otherwise the dispatch subscription might happen
	// earlier than the scheduled
	s.notifyOnScheduledMessage(nil, rawMessage)

	var hashes [][]byte
	var newMessages []*wakutypes.NewMessage

	forceRekey := rawMessage.CommunityKeyExMsgType == messagingtypes.KeyExMsgRekey

	// Check if it's a key exchange message. In this case we send it
	// to all the recipients
	if rawMessage.CommunityKeyExMsgType != messagingtypes.KeyExMsgNone {
		// If rekeycompatibility is on, we always
		// want to execute below, otherwise we execute
		// only when we want to fill up old keys to a given user
		if RekeyCompatibility || !forceRekey {
			keyExMessageSpecs, err := s.protocol.GetKeyExMessageSpecs(rawMessage.HashRatchetGroupID, s.identity, rawMessage.Recipients, forceRekey)
			if err != nil {
				return nil, err
			}

			for i, spec := range keyExMessageSpecs {
				recipient := rawMessage.Recipients[i]
				_, _, err = s.sendMessageSpec(ctx, recipient, spec, [][]byte{messageID})
				if err != nil {
					return nil, err
				}
			}
		}
	}

	wrappedMessage, err := s.wrapMessageV1(rawMessage)
	if err != nil {
		return nil, err
	}

	// If it's a chat message, we send it on the community chat topic
	if ShouldCommunityMessageBeEncrypted(rawMessage.MessageType) {
		messageSpec, err := s.protocol.BuildHashRatchetMessage(rawMessage.HashRatchetGroupID, wrappedMessage)
		if err != nil {
			return nil, err
		}

		payload, err := proto.Marshal(messageSpec.Message)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal")
		}
		hashes, newMessages, err = s.dispatchCommunityChatMessage(ctx, rawMessage, payload, forceRekey)
		if err != nil {
			return nil, err
		}

		sentMessage := &messagingevents.SentMessage{
			Spec:       messageSpec,
			MessageIDs: [][]byte{messageID},
		}

		s.notifyOnSentMessage(sentMessage)

	} else {

		pubkey, err := crypto.DecompressPubkey(rawMessage.CommunityID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decompress pubkey")
		}
		hashes, newMessages, err = s.dispatchCommunityMessage(ctx, pubkey, wrappedMessage, rawMessage.PubsubTopic, forceRekey, rawMessage)
		if err != nil {
			s.logger.Error("failed to send a community message", zap.Error(err))
			return nil, errors.Wrap(err, "failed to send a message spec")
		}
	}

	s.logger.Debug("sent-message: community ",
		zap.Strings("recipient", crypto.PubkeysToHex(rawMessage.Recipients)),
		zap.String("messageID", messageID.String()),
		zap.String("messageType", "community"),
		zap.Any("contentType", rawMessage.MessageType),
		zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)))
	s.transport.Track(messageID, hashes, newMessages)
	s.notifyOnSentRawMessage(rawMessage)

	return messageID, nil
}

// sendPrivate sends data to the recipient identifying with a given public key.
func (s *MessageSender) sendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage *messagingtypes.RawMessage,
) ([]byte, error) {
	s.logger.Debug("sending private message", zap.String("recipient", cryptotypes.EncodeHex(crypto.FromECDSAPub(recipient))))

	var wrappedMessage []byte
	var err error
	if rawMessage.SkipApplicationWrap {
		wrappedMessage = rawMessage.Payload
	} else {
		wrappedMessage, err = s.wrapMessageV1(rawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap message")
		}
	}

	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)

	if err = s.setMessageID(messageID, rawMessage); err != nil {
		return nil, err
	}

	if rawMessage.BeforeDispatch != nil {
		if err := rawMessage.BeforeDispatch(rawMessage); err != nil {
			return nil, err
		}
	}

	// Notify before dispatching, otherwise the dispatch subscription might happen
	// earlier than the scheduled
	s.notifyOnScheduledMessage(recipient, rawMessage)

	if rawMessage.ResendType == messagingtypes.ResendTypeDataSync {
		err = s.sendWithReliability(recipient, messageID, wrappedMessage)
		if err != nil {
			s.logger.Error("failed to send a private message with reliability", zap.Error(err))
		}
	} else if rawMessage.SkipEncryptionLayer {
		messageBytes := wrappedMessage
		if rawMessage.CommunityKeyExMsgType == messagingtypes.KeyExMsgReuse {
			groupID := rawMessage.HashRatchetGroupID

			ratchets, err := s.protocol.GetKeysForGroup(groupID)
			if err != nil {
				return nil, err
			}

			message, err := s.protocol.BuildHashRatchetKeyExchangeMessageWithPayload(s.identity, recipient, groupID, ratchets, wrappedMessage)
			if err != nil {
				return nil, err
			}

			messageBytes, err = proto.Marshal(message.Message)
			if err != nil {
				return nil, err
			}
		}

		// When SkipProtocolLayer is set we don't pass the message to the encryption layer
		hashes, newMessages, err := s.sendPrivateRawMessage(ctx, rawMessage, recipient, messageBytes)
		if err != nil {
			s.logger.Error("failed to send a private message", zap.Error(err))
			return nil, errors.Wrap(err, "failed to send a message spec")
		}

		s.logger.Debug("sent-message: private skipProtocolLayer",
			zap.String("recipient", crypto.PubkeyToHex(recipient)),
			zap.Stringer("messageID", messageID),
			zap.String("messageType", "private"),
			zap.Any("contentType", rawMessage.MessageType),
			zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)))
		s.transport.Track(messageID, hashes, newMessages)
	} else {
		err := s.sendPrivateEncryptedMessage(ctx, rawMessage.Sender, recipient, wrappedMessage, []cryptotypes.HexBytes{messageID})
		if err != nil {
			return nil, errors.Wrap(err, "failed to send private encrypted message")
		}
	}

	s.notifyOnSentRawMessage(rawMessage)

	return messageID, nil
}

func (s *MessageSender) sendPrivateEncryptedMessage(
	ctx context.Context,
	sender *ecdsa.PrivateKey,
	recipient *ecdsa.PublicKey,
	payload []byte,
	messageIDs []cryptotypes.HexBytes,
) error {
	messageSpec, err := s.protocol.BuildEncryptedMessage(sender, recipient, payload)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt message")
	}

	byteMessageIDs := make([][]byte, len(messageIDs))
	for i, id := range messageIDs {
		byteMessageIDs[i] = []byte(id)
	}

	hashes, newMessages, err := s.sendMessageSpec(ctx, recipient, messageSpec, byteMessageIDs)
	if err != nil {
		return errors.Wrap(err, "failed to send a message spec")
	}

	s.logger.Debug("sent-message: private encrypted",
		zap.String("recipient", crypto.PubkeyToHex(recipient)),
		zap.Stringers("messageID", messageIDs),
		zap.String("messageType", "private"),
		zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)))

	s.transport.TrackMany(byteMessageIDs, hashes, newMessages)

	return nil
}

// sendPairInstallation sends data to the recipients, using DH
func (s *MessageSender) SendPairInstallation(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage messagingtypes.RawMessage,
) ([]byte, error) {
	s.logger.Debug("sending private message", zap.String("recipient", cryptotypes.EncodeHex(crypto.FromECDSAPub(recipient))))

	wrappedMessage, err := s.wrapMessageV1(&rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageSpec, err := s.protocol.BuildDHMessage(s.identity, recipient, wrappedMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt message")
	}

	messageID := v1protocol.MessageID(&s.identity.PublicKey, wrappedMessage)

	hashes, newMessages, err := s.sendMessageSpec(ctx, recipient, messageSpec, [][]byte{messageID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to send a message spec")
	}

	s.transport.Track(messageID, hashes, newMessages)
	s.notifyOnSentRawMessage(&rawMessage)

	return messageID, nil
}

func (s *MessageSender) dispatchCommunityChatMessage(ctx context.Context, rawMessage *messagingtypes.RawMessage, wrappedMessage []byte, rekey bool) ([][]byte, []*wakutypes.NewMessage, error) {
	payload := wrappedMessage
	var err error
	if rekey && len(rawMessage.HashRatchetGroupID) != 0 {

		var ratchet *encryption.HashRatchetKeyCompatibility
		// We have just rekeyed, pull the latest
		if RekeyCompatibility {
			ratchet, err = s.protocol.GetCurrentKeyForGroup(rawMessage.HashRatchetGroupID)
			if err != nil {
				return nil, nil, err
			}

		}
		// We send the message over the community topic
		spec, err := s.protocol.BuildHashRatchetReKeyGroupMessage(s.identity, rawMessage.Recipients, rawMessage.HashRatchetGroupID, wrappedMessage, ratchet)
		if err != nil {
			return nil, nil, err
		}
		payload, err = proto.Marshal(spec.Message)
		if err != nil {
			return nil, nil, err
		}
	}

	newMessage := &wakutypes.NewMessage{
		Payload:     payload,
		PubsubTopic: rawMessage.PubsubTopic,
	}

	if rawMessage.BeforeDispatch != nil {
		if err := rawMessage.BeforeDispatch(rawMessage); err != nil {
			return nil, nil, err
		}
	}

	// notify before dispatching
	s.notifyOnScheduledMessage(nil, rawMessage)

	newMessages, err := s.segmentMessage(newMessage)
	if err != nil {
		return nil, nil, err
	}

	hashes := make([][]byte, 0, len(newMessages))
	for _, newMessage := range newMessages {
		hash, err := s.transport.SendPublic(ctx, newMessage, rawMessage.ContentTopic)
		if err != nil {
			return nil, nil, err
		}
		hashes = append(hashes, hash)
	}

	return hashes, newMessages, nil
}

// SendPublic takes encoded data, encrypts it and sends through the wire.
func (s *MessageSender) SendPublic(
	ctx context.Context,
	chatName string,
	rawMessage messagingtypes.RawMessage,
) ([]byte, error) {
	// Set sender
	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	var wrappedMessage []byte
	var err error
	if rawMessage.SkipApplicationWrap {
		wrappedMessage = rawMessage.Payload
	} else {
		wrappedMessage, err = s.wrapMessageV1(&rawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap message")
		}
	}

	var newMessage *wakutypes.NewMessage

	messageSpec, err := s.protocol.BuildPublicMessage(s.identity, wrappedMessage)
	if err != nil {
		s.logger.Error("failed to send a public message", zap.Error(err))
		return nil, errors.Wrap(err, "failed to wrap a public message in the encryption layer")
	}

	if len(rawMessage.HashRatchetGroupID) != 0 {

		var ratchet *encryption.HashRatchetKeyCompatibility
		var err error
		// We have just rekeyed, pull the latest
		ratchet, err = s.protocol.GetCurrentKeyForGroup(rawMessage.HashRatchetGroupID)
		if err != nil {
			return nil, err
		}

		keyID, err := ratchet.GetKeyID()
		if err != nil {
			return nil, err
		}
		s.logger.Debug("adding key id to message", zap.String("keyid", cryptotypes.Bytes2Hex(keyID)))
		// We send the message over the community topic
		spec, err := s.protocol.BuildHashRatchetReKeyGroupMessage(s.identity, rawMessage.Recipients, rawMessage.HashRatchetGroupID, wrappedMessage, ratchet)
		if err != nil {
			return nil, err
		}
		newMessage, err = MessageSpecToWhisper(spec)
		if err != nil {
			return nil, err
		}

	} else if !rawMessage.SkipEncryptionLayer {
		newMessage, err = MessageSpecToWhisper(messageSpec)
		if err != nil {
			return nil, err
		}
	} else {
		newMessage = &wakutypes.NewMessage{
			Payload: wrappedMessage,
		}
	}

	newMessage.Ephemeral = rawMessage.Ephemeral
	newMessage.PubsubTopic = rawMessage.PubsubTopic
	newMessage.Priority = rawMessage.Priority

	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)

	if err = s.setMessageID(messageID, &rawMessage); err != nil {
		return nil, err
	}

	if rawMessage.BeforeDispatch != nil {
		if err := rawMessage.BeforeDispatch(&rawMessage); err != nil {
			return nil, err
		}
	}

	// notify before dispatching
	s.notifyOnScheduledMessage(nil, &rawMessage)

	newMessages, err := s.segmentMessage(newMessage)
	if err != nil {
		return nil, err
	}

	hashes := make([][]byte, 0, len(newMessages))
	for _, newMessage := range newMessages {
		hash, err := s.transport.SendPublic(ctx, newMessage, chatName)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}

	sentMessage := &messagingevents.SentMessage{
		Spec:       messageSpec,
		MessageIDs: [][]byte{messageID},
	}

	s.notifyOnSentMessage(sentMessage)

	s.logger.Debug("sent-message: public message",
		zap.Strings("recipient", crypto.PubkeysToHex(rawMessage.Recipients)),
		zap.String("messageID", messageID.String()),
		zap.Any("contentType", rawMessage.MessageType),
		zap.String("messageType", "public"),
		zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)))
	s.transport.Track(messageID, hashes, newMessages)
	s.notifyOnSentRawMessage(&rawMessage)

	return messageID, nil
}

// HandleMessages expects a whisper message as input, and it will go through
// a series of transformations until the message is parsed into an application
// layer message, or in case of Raw methods, the processing stops at the layer
// before.
// It returns an error only if the processing of required steps failed.
func (s *MessageSender) HandleMessages(msg *messagingtypes.ReceivedMessage) (*messagingtypes.HandleMessageResponse, error) {
	logger := s.logger.With(zap.String("site", "HandleMessages"))
	hlogger := logger.With(zap.String("hash", cryptotypes.HexBytes(msg.Hash).String()))

	response, err := s.handleMessage(msg)
	if err != nil {
		// Hash ratchet with a group id not found yet, save the message for future processing
		if err == encryption.ErrHashRatchetGroupIDNotFound && len(response.Message.EncryptionLayer.HashRatchetInfo) == 1 {
			info := response.Message.EncryptionLayer.HashRatchetInfo[0]
			return nil, s.persistence.SaveHashRatchetMessage(info.GroupID, info.KeyID, msg)
		}

		return nil, err
	}

	if response == nil {
		return nil, nil
	}

	// Process queued hash ratchet messages
	for _, hashRatchetInfo := range response.Message.EncryptionLayer.HashRatchetInfo {
		messages, err := s.persistence.GetHashRatchetMessages(hashRatchetInfo.KeyID)
		if err != nil {
			return nil, err
		}

		var processedIds [][]byte
		for _, message := range messages {
			hlogger.Info("handling out of order encrypted messages", zap.String("hash", cryptotypes.Bytes2Hex(message.Hash)))
			r, err := s.handleMessage(message)
			if err != nil {
				hlogger.Debug("failed to handle hash ratchet message", zap.Error(err))
				continue
			}
			response.ReliabilityMessages = append(response.toPublicResponse().StatusMessages, r.Messages()...)
			response.AckedMessageIDs = append(response.AckedMessageIDs, r.AckedMessageIDs...)

			processedIds = append(processedIds, message.Hash)
		}

		err = s.persistence.DeleteHashRatchetMessages(processedIds)
		if err != nil {
			s.logger.Warn("failed to delete hash ratchet messages", zap.Error(err))
			return nil, err
		}
	}

	return response.toPublicResponse(), nil
}

func (h *handleMessageResponse) toPublicResponse() *messagingtypes.HandleMessageResponse {
	return &messagingtypes.HandleMessageResponse{
		StatusMessages:  h.Messages(),
		AckedMessageIDs: h.AckedMessageIDs,
	}
}

type handleMessageResponse struct {
	Message             *messagingtypes.Message
	ReliabilityMessages []*messagingtypes.Message
	AckedMessageIDs     []cryptotypes.HexBytes
}

func (h *handleMessageResponse) Messages() []*messagingtypes.Message {
	if len(h.ReliabilityMessages) > 0 {
		return h.ReliabilityMessages
	}
	return []*messagingtypes.Message{h.Message}
}

func (s *MessageSender) handleMessage(receivedMsg *messagingtypes.ReceivedMessage) (*handleMessageResponse, error) {
	hlogger := s.logger.Named("handleMessage").With(zap.String("hash", cryptotypes.EncodeHex(receivedMsg.Hash)))

	message := &messagingtypes.Message{}

	response := &handleMessageResponse{
		Message:             message,
		ReliabilityMessages: []*messagingtypes.Message{},
		AckedMessageIDs:     []cryptotypes.HexBytes{},
	}

	err := populateMessageTransportLayer(message, receivedMsg)
	if err != nil {
		hlogger.Error("failed to handle transport layer message", zap.Error(err))
		return nil, err
	}

	isSegmentMessage, completed, err := s.handleSegmentationLayer(message)
	if err != nil {
		return nil, err
	}

	// Segments not completed yet, stop processing
	if isSegmentMessage && !completed {
		return nil, nil
	}

	err = s.handleEncryptionLayer(context.Background(), message)
	if err != nil {
		hlogger.Debug("failed to handle an encryption message", zap.Error(err))

		// Hash ratchet with a group id not found yet, stop processing
		if err == encryption.ErrHashRatchetGroupIDNotFound {
			return response, err
		}
	}

	statusMessages, ackedMessageIDs, err := s.handleReliabilityLayer(message)
	if err == nil {
		response.ReliabilityMessages = append(response.ReliabilityMessages, statusMessages...)
		response.AckedMessageIDs = append(response.AckedMessageIDs, ackedMessageIDs...)
	} else {
		hlogger.Debug("failed to handle datasync message", zap.Error(err))
	}

	for _, msg := range response.Messages() {
		err := populateMessageApplicationLayer(msg)
		if err != nil {
			hlogger.Error("failed to handle application metadata layer message", zap.Error(err))
		}
		s.logger.Debug("calculated ID for envelope",
			zap.String("envelopeHash", hexutil.Encode(msg.TransportLayer.Hash)),
			zap.String("messageId", hexutil.Encode(msg.ApplicationLayer.ID)),
		)
	}

	return response, nil
}

// fetchDecryptionKey returns the private key associated with this public key, and returns true if it's an ephemeral key
func (s *MessageSender) fetchDecryptionKey(destination *ecdsa.PublicKey) (*ecdsa.PrivateKey, bool) {
	destinationID := cryptotypes.EncodeHex(crypto.FromECDSAPub(destination))

	s.ephemeralKeysMutex.Lock()
	decryptionKey, ok := s.ephemeralKeys[destinationID]
	s.ephemeralKeysMutex.Unlock()

	// the key is not there, fallback on identity
	if !ok {
		return s.identity, false
	}
	return decryptionKey, true
}

func (s *MessageSender) handleEncryptionLayer(ctx context.Context, message *messagingtypes.Message) error {
	logger := s.logger.Named("handleEncryptionLayer")
	publicKey := message.SigPubKey()

	// if it's an ephemeral key, we don't negotiate a topic
	decryptionKey, skipNegotiation := s.fetchDecryptionKey(message.TransportLayer.Dst)

	// As we handle non-encrypted messages, we make sure that DecryptPayload
	// is set regardless of whether this step is successful
	message.EncryptionLayer.Payload = message.TransportLayer.Payload

	// Nothing to do
	if skipNegotiation {
		return nil
	}

	var protocolMessage encryption.ProtocolMessage
	err := proto.Unmarshal(message.TransportLayer.Payload, &protocolMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProtocolMessage")
	}

	response, err := s.protocol.HandleMessage(
		decryptionKey,
		publicKey,
		&protocolMessage,
		message.TransportLayer.Hash,
	)

	switch err {
	case nil:
		message.EncryptionLayer.Payload = response.DecryptedMessage
		message.EncryptionLayer.Installations = adapters.FromEncryptionInstallations(response.Installations)
		message.EncryptionLayer.HashRatchetInfo = adapters.FromEncryptionHashRatchets(response.HashRatchetInfo)

		err := s.HandleSharedSecrets(response.SharedSecrets)
		if err != nil {
			logger.Error("failed to handle shared secrets", zap.Error(err))
		}
	case encryption.ErrHashRatchetGroupIDNotFound:
		if response != nil {
			message.EncryptionLayer.HashRatchetInfo = adapters.FromEncryptionHashRatchets(response.HashRatchetInfo)
		}
	case encryption.ErrDeviceNotFound:
		err := s.handleErrDeviceNotFound(ctx, publicKey)
		if err != nil {
			logger.Error("failed to handle ErrDeviceNotFound", zap.Error(err))
		}
	}

	return err
}

func (s *MessageSender) handleErrDeviceNotFound(ctx context.Context, publicKey *ecdsa.PublicKey) error {
	now := time.Now().Unix()
	advertise, err := s.protocol.ShouldAdvertiseBundle(publicKey, now)
	if err != nil {
		return err
	}
	if !advertise {
		return nil
	}

	messageSpec, err := s.protocol.BuildBundleAdvertiseMessage(s.identity, publicKey)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// We don't pass an array of messageIDs as no action needs to be taken
	// when sending a bundle
	_, _, err = s.sendMessageSpec(ctx, publicKey, messageSpec, nil)
	if err != nil {
		return err
	}

	s.protocol.ConfirmBundleAdvertisement(publicKey, now)

	return nil
}

func (s *MessageSender) wrapMessageV1(rawMessage *messagingtypes.RawMessage) ([]byte, error) {
	wrappedMessage, err := v1protocol.WrapMessageV1(rawMessage.Payload, rawMessage.MessageType, rawMessage.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}
	return wrappedMessage, nil
}

// sendPrivateRawMessage sends a message not wrapped in an encryption layer
func (s *MessageSender) sendPrivateRawMessage(ctx context.Context, rawMessage *messagingtypes.RawMessage, publicKey *ecdsa.PublicKey, payload []byte) ([][]byte, []*wakutypes.NewMessage, error) {
	newMessage := &wakutypes.NewMessage{
		Payload:     payload,
		PubsubTopic: rawMessage.PubsubTopic,
	}

	newMessages, err := s.segmentMessage(newMessage)
	if err != nil {
		return nil, nil, err
	}

	hashes := make([][]byte, 0, len(newMessages))
	var hash []byte
	for _, newMessage := range newMessages {
		if rawMessage.SendOnPersonalTopic {
			hash, err = s.transport.SendPrivateOnPersonalTopic(ctx, newMessage, publicKey)
		} else {
			hash, err = s.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
		}
		if err != nil {
			return nil, nil, err
		}
		hashes = append(hashes, hash)
	}

	return hashes, newMessages, nil
}

// sendCommunityMessage sends a message not wrapped in an encryption layer
// to a community
func (s *MessageSender) dispatchCommunityMessage(ctx context.Context, publicKey *ecdsa.PublicKey, wrappedMessage []byte, pubsubTopic string, rekey bool, rawMessage *messagingtypes.RawMessage) ([][]byte, []*wakutypes.NewMessage, error) {
	payload := wrappedMessage
	if rekey && len(rawMessage.HashRatchetGroupID) != 0 {

		var ratchet *encryption.HashRatchetKeyCompatibility
		var err error
		// We have just rekeyed, pull the latest
		if RekeyCompatibility {
			ratchet, err = s.protocol.GetCurrentKeyForGroup(rawMessage.HashRatchetGroupID)
			if err != nil {
				return nil, nil, err
			}

		}
		keyID, err := ratchet.GetKeyID()
		if err != nil {
			return nil, nil, err
		}
		s.logger.Debug("adding key id to message", zap.String("keyid", cryptotypes.Bytes2Hex(keyID)))
		// We send the message over the community topic
		spec, err := s.protocol.BuildHashRatchetReKeyGroupMessage(s.identity, rawMessage.Recipients, rawMessage.HashRatchetGroupID, wrappedMessage, ratchet)
		if err != nil {
			return nil, nil, err
		}
		payload, err = proto.Marshal(spec.Message)
		if err != nil {
			return nil, nil, err
		}
	}

	newMessage := &wakutypes.NewMessage{
		Payload:     payload,
		PubsubTopic: pubsubTopic,
	}

	newMessages, err := s.segmentMessage(newMessage)
	if err != nil {
		return nil, nil, err
	}

	hashes := make([][]byte, 0, len(newMessages))
	for _, newMessage := range newMessages {
		hash, err := s.transport.SendCommunityMessage(ctx, newMessage, publicKey)
		if err != nil {
			return nil, nil, err
		}
		hashes = append(hashes, hash)
	}

	return hashes, newMessages, nil
}

// sendMessageSpec analyses the spec properties and selects a proper transport method.
func (s *MessageSender) sendMessageSpec(ctx context.Context, publicKey *ecdsa.PublicKey, messageSpec *encryption.ProtocolMessageSpec, messageIDs [][]byte) ([][]byte, []*wakutypes.NewMessage, error) {
	logger := s.logger.With(zap.String("site", "sendMessageSpec"))

	// The shared secret needs to be handle before we send a message
	// otherwise the topic might not be set up before we receive a message
	if messageSpec.SharedSecret != nil {
		err := s.HandleSharedSecrets([]*sharedsecret.Secret{messageSpec.SharedSecret})
		if err != nil {
			return nil, nil, err
		}
	}

	newMessage, err := MessageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, nil, err
	}

	newMessages, err := s.segmentMessage(newMessage)
	if err != nil {
		return nil, nil, err
	}

	hashes := make([][]byte, 0, len(newMessages))
	var hash []byte
	for _, newMessage := range newMessages {
		// process shared secret
		if messageSpec.AgreedSecret {
			logger.Debug("sending using shared secret")
			hash, err = s.transport.SendPrivateWithSharedSecret(ctx, newMessage, publicKey, messageSpec.SharedSecret.Key)
		} else {
			logger.Debug("sending partitioned topic")
			hash, err = s.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
		}
		if err != nil {
			return nil, nil, err
		}
		hashes = append(hashes, hash)
	}

	sentMessage := &messagingevents.SentMessage{
		PublicKey:  publicKey,
		Spec:       messageSpec,
		MessageIDs: messageIDs,
	}

	s.notifyOnSentMessage(sentMessage)

	return hashes, newMessages, nil
}

func (s *MessageSender) notifyOnSentMessage(sentMessage *messagingevents.SentMessage) {
	pubsub.Publish(s.publisher, messagingevents.MessageEvent{
		Type:        messagingevents.MessageSent,
		SentMessage: sentMessage,
	})
}

func (s *MessageSender) notifyOnSentRawMessage(rawMessage *messagingtypes.RawMessage) {
	pubsub.Publish(s.publisher, messagingevents.MessageEvent{
		Type:       messagingevents.RawMessageSent,
		RawMessage: rawMessage,
	})
}

func (s *MessageSender) notifyOnScheduledMessage(recipient *ecdsa.PublicKey, message *messagingtypes.RawMessage) {
	pubsub.Publish(s.publisher, messagingevents.MessageEvent{
		Type:       messagingevents.MessageScheduled,
		Recipient:  recipient,
		RawMessage: message,
	})
}

func (s *MessageSender) JoinPublic(id string) (*transport.Filter, error) {
	filter, err := s.transport.JoinPublic(id)
	if err != nil {
		return nil, err
	}
	return filter, nil
}

func (s *MessageSender) getRandomEphemeralKey() *ecdsa.PrivateKey {
	k := rand.Intn(len(s.ephemeralKeys)) //nolint: gosec
	for _, key := range s.ephemeralKeys {
		if k == 0 {
			return key
		}
		k--
	}
	return nil
}

func (s *MessageSender) GetEphemeralKey() (*ecdsa.PrivateKey, error) {
	s.ephemeralKeysMutex.Lock()
	if len(s.ephemeralKeys) >= maxMessageSenderEphemeralKeys {
		s.ephemeralKeysMutex.Unlock()
		return s.getRandomEphemeralKey(), nil
	}
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		s.ephemeralKeysMutex.Unlock()
		return nil, err
	}

	s.ephemeralKeys[cryptotypes.EncodeHex(crypto.FromECDSAPub(&privateKey.PublicKey))] = privateKey
	s.ephemeralKeysMutex.Unlock()
	_, err = s.transport.LoadKeyFilters(privateKey)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func MessageSpecToWhisper(spec *encryption.ProtocolMessageSpec) (*wakutypes.NewMessage, error) {
	var newMessage *wakutypes.NewMessage

	payload, err := proto.Marshal(spec.Message)
	if err != nil {
		return newMessage, err
	}

	newMessage = &wakutypes.NewMessage{
		Payload: payload,
	}
	return newMessage, nil
}

func (s *MessageSender) markAsConfirmed(dataSyncID []byte, atLeastOne bool) (messageID cryptotypes.HexBytes, err error) {
	return s.persistence.MarkAsConfirmed(dataSyncID, atLeastOne)
}

func (s *MessageSender) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *messagingtypes.ReceivedMessage) error {
	return s.persistence.SaveHashRatchetMessage(groupID, keyID, m)
}

// GetCurrentKeyForGroup returns the latest key timestampID belonging to a key group
func (s *MessageSender) GetCurrentKeyForGroup(groupID []byte) (*encryption.HashRatchetKeyCompatibility, error) {
	return s.protocol.GetCurrentKeyForGroup(groupID)
}

// GetKeyIDsForGroup returns a slice of key IDs belonging to a given group ID
func (s *MessageSender) GetKeysForGroup(groupID []byte) ([]*encryption.HashRatchetKeyCompatibility, error) {
	return s.protocol.GetKeysForGroup(groupID)
}

func (s *MessageSender) CleanupHashRatchetEncryptedMessages() error {
	monthAgo := time.Now().AddDate(0, -1, 0).Unix()

	err := s.persistence.DeleteHashRatchetMessagesOlderThan(monthAgo)
	if err != nil {
		return err
	}

	return nil
}

func (s *MessageSender) Publisher() *pubsub.Publisher {
	return s.publisher
}

func (s *MessageSender) HandleSharedSecrets(secrets []*sharedsecret.Secret) error {
	for _, secret := range secrets {
		fSecret := ethtypes.NegotiatedSecret{
			PublicKey: secret.Identity,
			Key:       secret.Key,
		}
		_, err := s.transport.ProcessNegotiatedSecret(fSecret)
		if err != nil {
			return err
		}
	}
	return nil
}

func populateMessageTransportLayer(m *messagingtypes.Message, msg *messagingtypes.ReceivedMessage) error {
	publicKey, err := crypto.UnmarshalPubkey(msg.Sig)
	if err != nil {
		return errors.Wrap(err, "failed to get signature")
	}

	m.TransportLayer.Message = msg
	m.TransportLayer.Hash = msg.Hash
	m.TransportLayer.SigPubKey = publicKey
	m.TransportLayer.Payload = msg.Payload

	if msg.Dst != nil {
		publicKey, err := crypto.UnmarshalPubkey(msg.Dst)
		if err != nil {
			return err
		}
		m.TransportLayer.Dst = publicKey
	}

	return nil
}

func populateMessageApplicationLayer(m *messagingtypes.Message) error {
	message, err := protobuf.Unmarshal(m.EncryptionLayer.Payload)
	if err != nil {
		return err
	}

	recoveredKey, err := utils.RecoverKey(message)
	if err != nil {
		return err
	}

	m.ApplicationLayer.SigPubKey = recoveredKey
	// Calculate ID using the wrapped record
	m.ApplicationLayer.ID = messagingtypes.MessageID(recoveredKey, m.EncryptionLayer.Payload)
	m.ApplicationLayer.Payload = message.Payload
	m.ApplicationLayer.Type = message.Type
	return nil
}
