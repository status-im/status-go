package common

import (
	"context"
	"crypto/ecdsa"
	"sync"

	"github.com/pkg/errors"
	otelattribute "go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/pkg/messaging"
	messagingevents "github.com/status-im/status-go/pkg/messaging/events"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

var ErrModifiedRawMessage = errors.New("modified rawMessage")

type MessageSender struct {
	identity  *ecdsa.PrivateKey
	messaging *messaging.API
	logger    *zap.Logger
	tracer    trace.Tracer
	publisher *pubsub.Publisher

	wg   sync.WaitGroup
	quit chan struct{}
}

func NewMessageSender(
	identity *ecdsa.PrivateKey,
	messaging *messaging.API,
	logger *zap.Logger,
	tracer trace.Tracer,
) *MessageSender {
	return &MessageSender{
		identity:  identity,
		messaging: messaging,
		logger:    logger,
		tracer:    tracer,
		publisher: pubsub.NewPublisher(),
		quit:      make(chan struct{}),
	}
}

func wrapIntoAppLayerMessage(rawMessage *RawMessage) ([]byte, error) {
	wrappedMessage, err := v1protocol.WrapIntoAppLayerMessage(rawMessage.Payload, rawMessage.MessageType, rawMessage.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}
	return wrappedMessage, nil
}

func EnsureMessageIDIntegrity(messageID string, rawMessage *RawMessage) error {
	if len(rawMessage.ID) > 0 && rawMessage.ID != messageID {
		return ErrModifiedRawMessage
	}
	return nil
}

func setMessageID(messageID types2.HexBytes, rawMessage *RawMessage) error {
	messageIDString := types2.EncodeHex(messageID)

	err := EnsureMessageIDIntegrity(messageIDString, rawMessage)
	if err != nil {
		return err
	}

	rawMessage.ID = messageIDString

	return nil
}

func (s *MessageSender) Start() {
	s.wg.Add(1)

	go func() {
		defer gocommon.LogOnPanic()
		defer s.wg.Done()

		sentMessagesSub, unsubSentMessages := pubsub.Subscribe[messagingevents.SentMessage](s.messaging.Publisher(), 100)
		defer unsubSentMessages()

		for {
			select {
			case m, ok := <-sentMessagesSub:
				if !ok {
					return
				}
				s.notifyOnSentMessage(&m)
			case <-s.quit:
				return
			}
		}
	}()
}

func (s *MessageSender) Stop() {
	close(s.quit)
	s.wg.Wait()
}

func (s *MessageSender) notifyOnScheduledMessage(recipient *ecdsa.PublicKey, message *RawMessage) {
	pubsub.Publish(s.publisher, MessageEvent{
		ScheduledMessage: &ScheduledMessageEvent{
			Recipient:  recipient,
			RawMessage: message,
		},
	})
}

func (s *MessageSender) notifyOnSentMessage(sentMessage *messagingevents.SentMessage) {
	pubsub.Publish(s.publisher, MessageEvent{
		SentMessage: sentMessage,
	})
}

func (s *MessageSender) Publisher() *pubsub.Publisher {
	return s.publisher
}

func (s *MessageSender) SendPublic(
	ctx context.Context,
	chatName string,
	rawMessage RawMessage,
) ([]byte, error) {
	ctx, span := s.tracer.Start(ctx, "MessageSender.SendPublic")
	defer span.End()

	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	if len(rawMessage.ContentTopic) == 0 {
		rawMessage.ContentTopic = chatName
	}

	var wrappedMessage []byte
	var err error
	if rawMessage.SkipApplicationWrap {
		wrappedMessage = rawMessage.Payload
	} else {
		wrappedMessage, err = wrapIntoAppLayerMessage(&rawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap message")
		}
	}

	messageID := types.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	if err = setMessageID(messageID, &rawMessage); err != nil {
		return nil, err
	}

	logger := s.logger.Named("sendPublic").With(
		zap.Stringer("messageID", messageID),
	)

	if rawMessage.BeforeDispatch != nil {
		if err := rawMessage.BeforeDispatch(&rawMessage); err != nil {
			return nil, err
		}
	}

	// notify before dispatching
	s.notifyOnScheduledMessage(nil, &rawMessage)

	var hashRatchetParams *types.SendPublicHashRatchetParams
	if len(rawMessage.HashRatchetGroupID) != 0 {
		hashRatchetParams = &types.SendPublicHashRatchetParams{
			Encrypt:   false,
			GroupID:   rawMessage.HashRatchetGroupID,
			KeyExType: rawMessage.CommunityKeyExMsgType,
			Members:   rawMessage.Recipients,
		}
	}

	err = s.messaging.SendPublic(ctx, types.SendPublicParams{
		Sender:              &rawMessage.Sender.PublicKey,
		Payload:             wrappedMessage,
		PubsubTopic:         rawMessage.PubsubTopic,
		ContentTopic:        rawMessage.ContentTopic,
		SkipEncryptionLayer: rawMessage.SkipEncryptionLayer,
		Ephemeral:           rawMessage.Ephemeral,
		Priority:            rawMessage.Priority,
		HashRatchet:         hashRatchetParams,
		CommunityID:         rawMessage.CommunityID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to send public message")
	}

	logger.Debug("sent-message",
		zap.Any("contentType", rawMessage.MessageType),
		zap.String("messageType", "public"),
	)

	if s.tracer.Enabled() {
		linkSpanWithMessage(span, &rawMessage)
	}

	s.messaging.MetricsPushSentMessage(
		rawMessage.PubsubTopic,
		rawMessage.ContentTopic,
		rawMessage.MessageType.String(),
		uint32(len(wrappedMessage)),
	)

	return messageID, nil
}

func (s *MessageSender) sendPrivate(
	ctx context.Context,
	rawMessage *RawMessage,
) ([]byte, error) {
	ctx, span := s.tracer.Start(ctx, "MessageSender.sendPrivate")
	defer span.End()

	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	var wrappedMessage []byte
	var err error
	if rawMessage.SkipApplicationWrap {
		wrappedMessage = rawMessage.Payload
	} else {
		wrappedMessage, err = wrapIntoAppLayerMessage(rawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap message")
		}
	}

	messageID := types.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	if err = setMessageID(messageID, rawMessage); err != nil {
		return nil, err
	}

	logger := s.logger.Named("sendPrivate").With(
		zap.Stringer("messageID", messageID),
	)

	logger.Debug("sending private message",
		zap.Strings("recipients", crypto.PubkeysToHex(rawMessage.Recipients)),
		zap.Stringer("contentType", rawMessage.MessageType),
	)

	if rawMessage.BeforeDispatch != nil {
		if err := rawMessage.BeforeDispatch(rawMessage); err != nil {
			return nil, err
		}
	}

	var hashRatchetGroupID []byte
	if rawMessage.CommunityKeyExMsgType == types.KeyExMsgReuse {
		hashRatchetGroupID = rawMessage.HashRatchetGroupID
	}

	for _, recipient := range rawMessage.Recipients {
		s.notifyOnScheduledMessage(recipient, rawMessage)

		err = s.messaging.SendPrivate(ctx, types.SendPrivateParams{
			Sender:              rawMessage.Sender,
			Recipient:           recipient,
			Payload:             wrappedMessage,
			PubsubTopic:         rawMessage.PubsubTopic,
			WithReliability:     rawMessage.ResendType == ResendTypeDataSync,
			SkipEncryptionLayer: rawMessage.SkipEncryptionLayer,
			SendOnPersonalTopic: rawMessage.SendOnPersonalTopic,
			HashRatchetGroupID:  hashRatchetGroupID,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to send private message")
		}
	}

	if s.tracer.Enabled() {
		linkSpanWithMessage(span, rawMessage)
	}

	s.messaging.MetricsPushSentMessage(
		rawMessage.PubsubTopic,
		rawMessage.ContentTopic,
		rawMessage.MessageType.String(),
		uint32(len(wrappedMessage)),
	)

	return messageID, nil
}

func (s *MessageSender) SendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage *RawMessage,
) ([]byte, error) {
	// Currently we don't support sending through datasync and setting custom waku fields,
	// as the datasync interface is not rich enough to propagate that information, so we
	// would have to add some complexity to handle this.
	if rawMessage.ResendType == ResendTypeDataSync && (rawMessage.Sender != nil || rawMessage.SkipEncryptionLayer || rawMessage.SendOnPersonalTopic) {
		return nil, errors.New("setting identity, skip-encryption or personal topic and datasync not supported")
	}

	rawMessage.Recipients = []*ecdsa.PublicKey{recipient}
	return s.sendPrivate(ctx, rawMessage)
}

func (s *MessageSender) SendGroup(
	ctx context.Context,
	recipients []*ecdsa.PublicKey,
	rawMessage *RawMessage,
) ([]byte, error) {
	rawMessage.Recipients = recipients
	return s.sendPrivate(ctx, rawMessage)
}

func shouldCommunityMessageBeEncrypted(msgType protobuf.ApplicationMetadataMessage_Type) bool {
	return msgType == protobuf.ApplicationMetadataMessage_CHAT_MESSAGE ||
		msgType == protobuf.ApplicationMetadataMessage_EDIT_MESSAGE ||
		msgType == protobuf.ApplicationMetadataMessage_DELETE_MESSAGE ||
		msgType == protobuf.ApplicationMetadataMessage_PIN_MESSAGE ||
		msgType == protobuf.ApplicationMetadataMessage_EMOJI_REACTION
}

func (s *MessageSender) SendCommunity(
	ctx context.Context,
	rawMessage *RawMessage,
) ([]byte, error) {
	ctx, span := s.tracer.Start(ctx, "MessageSender.SendCommunity")
	defer span.End()

	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	wrappedMessage, err := wrapIntoAppLayerMessage(rawMessage)
	if err != nil {
		return nil, err
	}

	messageID := types.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	err = setMessageID(messageID, rawMessage)
	if err != nil {
		return nil, err
	}

	logger := s.logger.Named("sendCommunity").With(
		zap.Stringer("messageID", messageID),
		zap.String("communityID", types2.EncodeHex(rawMessage.CommunityID)),
		zap.String("sender", crypto.PubkeyToHex(&rawMessage.Sender.PublicKey)),
	)

	if rawMessage.BeforeDispatch != nil {
		if err := rawMessage.BeforeDispatch(rawMessage); err != nil {
			return nil, err
		}
	}
	// Notify before dispatching, otherwise the dispatch subscription might happen
	// earlier than the scheduled
	s.notifyOnScheduledMessage(nil, rawMessage)

	// We want to fill up old keys to a given users
	if rawMessage.CommunityKeyExMsgType == types.KeyExMsgReuse {
		return messageID, s.messaging.SendPrivateHashRatchetKeys(ctx, rawMessage.Recipients, rawMessage.HashRatchetGroupID)
	}

	hashRatchetParams := &types.SendPublicHashRatchetParams{
		Encrypt:   false,
		GroupID:   rawMessage.HashRatchetGroupID,
		KeyExType: rawMessage.CommunityKeyExMsgType,
		Members:   rawMessage.Recipients,
	}

	// If it's a chat message, we send it on the community chat topic
	if shouldCommunityMessageBeEncrypted(rawMessage.MessageType) {
		if len(rawMessage.HashRatchetGroupID) == 0 {
			return nil, errors.New("missing hash ratchet group ID for community encrypted message")
		}

		hashRatchetParams.Encrypt = true

		err = s.messaging.SendPublic(ctx, types.SendPublicParams{
			Sender:       &rawMessage.Sender.PublicKey,
			Payload:      wrappedMessage,
			PubsubTopic:  rawMessage.PubsubTopic,
			ContentTopic: rawMessage.ContentTopic,
			HashRatchet:  hashRatchetParams,
			CommunityID:  rawMessage.CommunityID,
		})

	} else {
		var pubkey *ecdsa.PublicKey
		pubkey, err = crypto.DecompressPubkey(rawMessage.CommunityID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decompress pubkey")
		}

		err = s.messaging.SendPublic(ctx, types.SendPublicParams{
			Sender:             &rawMessage.Sender.PublicKey,
			Payload:            wrappedMessage,
			PubsubTopic:        rawMessage.PubsubTopic,
			ContentTopic:       rawMessage.ContentTopic,
			HashRatchet:        hashRatchetParams,
			CommunityPublicKey: pubkey,
			CommunityID:        rawMessage.CommunityID,
		})
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to send community message")
	}

	logger.Debug("sent-message",
		zap.Any("contentType", rawMessage.MessageType),
	)

	if s.tracer.Enabled() {
		linkSpanWithMessage(span, rawMessage)
	}

	s.messaging.MetricsPushSentMessage(
		rawMessage.PubsubTopic,
		rawMessage.ContentTopic,
		rawMessage.MessageType.String(),
		uint32(len(wrappedMessage)),
	)

	return messageID, nil
}

// sendPairInstallation sends data to the recipients, using DH
func (s *MessageSender) SendPairInstallation(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage RawMessage,
) ([]byte, error) {
	ctx, span := s.tracer.Start(ctx, "MessageSender.SendPairInstallation")
	defer span.End()

	s.logger.Debug("sending private message", zap.String("recipient", types2.EncodeHex(crypto.FromECDSAPub(recipient))))

	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	wrappedMessage, err := wrapIntoAppLayerMessage(&rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageID := types.MessageID(&s.identity.PublicKey, wrappedMessage)
	err = setMessageID(messageID, &rawMessage)
	if err != nil {
		return nil, err
	}

	s.notifyOnScheduledMessage(recipient, &rawMessage)

	err = s.messaging.SendPrivate(ctx, types.SendPrivateParams{
		Sender:     s.identity,
		Recipient:  recipient,
		Payload:    wrappedMessage,
		SendWithDH: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to send private DH message")
	}

	if s.tracer.Enabled() {
		linkSpanWithMessage(span, &rawMessage)
	}

	s.messaging.MetricsPushSentMessage(
		rawMessage.PubsubTopic,
		rawMessage.ContentTopic,
		rawMessage.MessageType.String(),
		uint32(len(wrappedMessage)),
	)

	return messageID, nil
}

func linkSpanWithMessage(span oteltrace.Span, message *RawMessage) {
	span.SetAttributes(
		otelattribute.String("messageID", message.ID),
		otelattribute.Stringer("messageType", message.MessageType),
		otelattribute.String("sender", crypto.PubkeyToHex(&message.Sender.PublicKey)),
		otelattribute.StringSlice("recipients", crypto.PubkeysToHex(message.Recipients)),
		otelattribute.String("contentTopic", message.ContentTopic),
		otelattribute.String("pubsubTopic", message.PubsubTopic),
		otelattribute.Int("sendCount", message.SendCount),
	)
	linkSpanCtx := trace.DeriveSpanContext([]byte(message.ID), false)
	span.AddLink(oteltrace.Link{SpanContext: linkSpanCtx})
}
