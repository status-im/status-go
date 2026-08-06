package processor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	otelattribute "go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	adapters "github.com/status-im/status-go/pkg/messaging/adapters"
	common "github.com/status-im/status-go/pkg/messaging/common"
	"github.com/status-im/status-go/pkg/messaging/controller/utils"
	encryption "github.com/status-im/status-go/pkg/messaging/layers/encryption"
	"github.com/status-im/status-go/pkg/messaging/layers/encryption/sharedsecret"
	"github.com/status-im/status-go/pkg/messaging/layers/segmentation"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

const (
	maxNumOfEphemeralKeys = 3
)

var errReliabilityNotStarted = errors.New("reliability not started")

type Processor struct {
	identity *ecdsa.PrivateKey
	stack    *common.MessagingStack

	messageConfirmationStorage common.MessageConfirmationPersistence
	hashRatchetStorage         common.HashRatchetPersistence

	ephemeralKeysManager *EphemeralKeysManager

	publisher *pubsub.Publisher
	logger    *zap.Logger

	tracer trace.Tracer
}

func NewProcessor(
	identity *ecdsa.PrivateKey,
	stack *common.MessagingStack,
	messageConfirmationStorage common.MessageConfirmationPersistence,
	hashRatchetStorage common.HashRatchetPersistence,
	logger *zap.Logger,
	tracer trace.Tracer,
) *Processor {
	return &Processor{
		identity:                   identity,
		stack:                      stack,
		messageConfirmationStorage: messageConfirmationStorage,
		hashRatchetStorage:         hashRatchetStorage,
		ephemeralKeysManager:       NewEphemeralKeysManager(maxNumOfEphemeralKeys),
		publisher:                  pubsub.NewPublisher(),
		logger:                     logger.Named("processor"),
		tracer:                     tracer,
	}
}

func (r *Processor) Publisher() *pubsub.Publisher {
	return r.publisher
}

func (r *Processor) GetEphemeralKey() (*ecdsa.PrivateKey, error) {
	key, err := r.ephemeralKeysManager.GetRandom()
	if err != nil {
		return nil, err
	}
	_, err = r.stack.Transport.LoadKeyFilters(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (r *Processor) ProcessMessage(msg *messagingtypes.ReceivedMessage) (*messagingtypes.HandleMessageResponse, error) {
	response, err := r.processMessage(msg)
	if response == nil || err != nil {
		return nil, err
	}

	// Process queued hash ratchet messages
	for _, message := range response.messages {
		queuedMessagesResponse, err := r.processQueuedHashRatchetMessages(message.EncryptionLayer.HashRatchetInfo)
		if err != nil {
			return nil, err
		}
		response.messages = append(response.messages, queuedMessagesResponse.messages...)
		response.ackedMessageIDs = append(response.ackedMessageIDs, queuedMessagesResponse.ackedMessageIDs...)
	}

	return &messagingtypes.HandleMessageResponse{
		Messages:        response.messages,
		AckedMessageIDs: response.ackedMessageIDs,
	}, nil
}

type processMessageResponse struct {
	messages        []*messagingtypes.Message
	ackedMessageIDs []types.HexBytes
}

func (r *Processor) processMessage(m *messagingtypes.ReceivedMessage) (*processMessageResponse, error) {
	logger := r.logger.With(zap.Stringer("hash", types.HexBytes(m.Hash)))
	logger.Debug("processing received message")

	responseMessage := &messagingtypes.Message{}

	response := &processMessageResponse{
		messages:        []*messagingtypes.Message{responseMessage},
		ackedMessageIDs: []types.HexBytes{},
	}

	err := processTransportLayer(responseMessage, m)
	if err != nil {
		logger.Error("failed to process transport layer", zap.Error(err))
		return nil, err
	}

	err = r.processSegmentationLayer(responseMessage)
	if err != nil {
		return nil, err
	}

	hashes := [][]byte{m.Hash}
	if responseMessage.SegmentationLayer.Segmented {
		// Segments not completed yet, stop processing
		if !responseMessage.SegmentationLayer.Completed {
			return nil, nil
		}
		hashes = responseMessage.SegmentationLayer.Hashes
	}

	ctx, span := r.tracer.Start(trace.DeriveRemoteContext(utils.MergeByteSlices(hashes)), "Processor.processMessage",
		oteltrace.WithAttributes(
			otelattribute.String("hash", types.EncodeHex(m.Hash)),
			otelattribute.StringSlice("hashes", types.EncodeHexes(hashes)),
		),
	)
	defer span.End()

	err = r.processEncryptionLayer(ctx, responseMessage, logger)
	if err == nil {
		span.AddEvent("encryption layer processed")
	} else {
		// Hash ratchet with a group id not found yet, save the message for future processing
		if err == encryption.ErrHashRatchetGroupIDNotFound && len(responseMessage.EncryptionLayer.HashRatchetInfo) == 1 {
			info := responseMessage.EncryptionLayer.HashRatchetInfo[0]
			span.AddEvent("hash ratchet with group id not found yet", oteltrace.WithAttributes(
				otelattribute.String("groupID", types.ToHex(info.GroupID)),
			))
			return nil, r.hashRatchetStorage.SaveMessage(info.GroupID, info.KeyID, m)
		} else {
			span.AddEvent("encryption layer not processed", oteltrace.WithAttributes(
				otelattribute.String("error", err.Error()),
			))
			logger.Debug("failed to process encryption layer", zap.Error(err))
		}
	}

	// A broken SDS layer yields a payload that silently fails to decode further up,
	// so fail the whole envelope instead: it must be retried, not marked processed.
	err = r.processSDSLayer(responseMessage)
	if err != nil {
		logger.Error("failed to unwrap payload for SDS", zap.Error(err))
		return nil, err
	}

	messages, ackedMessageIDs, err := r.processReliabilityLayer(responseMessage, logger)
	if err == nil {
		span.AddEvent("reliability layer processed")
		response.messages = messages
		response.ackedMessageIDs = ackedMessageIDs
	} else {
		span.AddEvent("reliability layer not processed")
		logger.Debug("failed to process reliability layer", zap.Error(err))
	}

	return response, nil
}

func (r *Processor) processQueuedHashRatchetMessages(hashRatchetInfos []*messagingtypes.HashRatchetInfo) (*processMessageResponse, error) {
	response := &processMessageResponse{
		messages:        []*messagingtypes.Message{},
		ackedMessageIDs: []types.HexBytes{},
	}

	for _, hashRatchetInfo := range hashRatchetInfos {
		messages, err := r.hashRatchetStorage.GetMessages(hashRatchetInfo.KeyID)
		if err != nil {
			return nil, err
		}

		var processedIds [][]byte
		for _, message := range messages {
			logger := r.logger.With(zap.String("hash", types.EncodeHex(message.Hash)))
			logger.Debug("processing queued hash ratchet message")

			r, err := r.processMessage(message)
			if err != nil {
				continue
			}

			processedIds = append(processedIds, message.Hash)

			response.messages = append(response.messages, r.messages...)
			response.ackedMessageIDs = append(response.ackedMessageIDs, r.ackedMessageIDs...)
		}

		err = r.hashRatchetStorage.DeleteMessages(processedIds)
		if err != nil {
			r.logger.Warn("failed to delete hash ratchet messages", zap.Error(err))
			return nil, err
		}
	}

	return response, nil
}

func processTransportLayer(m *messagingtypes.Message, receivedMessage *messagingtypes.ReceivedMessage) error {
	publicKey, err := crypto.UnmarshalPubkey(receivedMessage.Sig)
	if err != nil {
		return errors.Wrap(err, "failed to get signature")
	}

	m.TransportLayer.Message = receivedMessage
	m.TransportLayer.Hash = receivedMessage.Hash
	m.TransportLayer.SigPubKey = publicKey
	m.TransportLayer.Payload = receivedMessage.Payload

	if receivedMessage.Dst != nil {
		publicKey, err := crypto.UnmarshalPubkey(receivedMessage.Dst)
		if err != nil {
			return err
		}
		m.TransportLayer.Dst = publicKey
	}

	return nil
}

func (r *Processor) processSegmentationLayer(m *messagingtypes.Message) error {
	reconstructedPayload, transportIDs, err := r.stack.Segmentation.Reconstruct(
		m.TransportLayer.Payload,
		m.TransportLayer.SigPubKey,
		m.TransportLayer.Hash)

	switch err {
	case nil:
		m.TransportLayer.Payload = reconstructedPayload
		m.SegmentationLayer.Segmented = true
		m.SegmentationLayer.Completed = true
		m.SegmentationLayer.Hashes = transportIDs
	case segmentation.ErrIncomplete:
		m.SegmentationLayer.Segmented = true
		m.SegmentationLayer.Completed = false
		err = nil
	case segmentation.ErrAlreadyCompleted:
		// A duplicate segment for an already reconstructed message should be ignored.
		m.SegmentationLayer.Segmented = true
		m.SegmentationLayer.Completed = false
		err = nil
	case segmentation.ErrInvalidPayload:
		m.SegmentationLayer.Segmented = false
		m.SegmentationLayer.Completed = false
		err = nil
	}

	return err
}

func (r *Processor) processEncryptionLayer(ctx context.Context, m *messagingtypes.Message, logger *zap.Logger) error {
	logger = logger.Named("processEncryptionLayer")

	ctx, span := r.tracer.Start(ctx, "Processor.processEncryptionLayer")
	defer span.End()

	// As we handle non-encrypted messages, we make sure that DecryptPayload
	// is set regardless of whether this step is successful
	m.EncryptionLayer.Payload = m.TransportLayer.Payload

	// if it's an ephemeral key, we don't negotiate a topic
	ephemeralKey := r.ephemeralKeysManager.GetPrivateKeyFor(m.TransportLayer.Dst)
	if ephemeralKey != nil {
		span.AddEvent("targeted ephemeral key")
		return nil
	}

	var protocolMessage encryption.ProtocolMessage
	err := proto.Unmarshal(m.TransportLayer.Payload, &protocolMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProtocolMessage")
	}

	response, err := r.stack.Encryption.HandleMessage(
		ctx,
		r.identity,
		m.SigPubKey(),
		&protocolMessage,
		m.TransportLayer.Hash,
	)

	switch err {
	case nil:
		m.EncryptionLayer.Payload = response.DecryptedMessage
		m.EncryptionLayer.Installations = adapters.FromEncryptionInstallations(response.Installations)
		m.EncryptionLayer.HashRatchetInfo = adapters.FromEncryptionHashRatchets(response.HashRatchetInfo)

		err := r.ProcessSharedSecrets(response.SharedSecrets)
		if err != nil {
			logger.Error("failed to process shared secrets", zap.Error(err))
		}
	case encryption.ErrHashRatchetGroupIDNotFound:
		if response != nil {
			m.EncryptionLayer.HashRatchetInfo = adapters.FromEncryptionHashRatchets(response.HashRatchetInfo)
		}
	case encryption.ErrDeviceNotFound:
		pubsub.Publish(r.publisher, SenderUnawareOfInstallation{
			PublicKey: m.SigPubKey(),
		})
	default:
		logger.Error("failed to decrypt message", zap.Error(err))
	}

	return err
}

func (r *Processor) processSDSLayer(msg *messagingtypes.Message) error {
	if len(msg.EncryptionLayer.Payload) <= 0 {
		return nil
	}

	unwrappedPayload, err := r.stack.Reliability.UnwrapPayloadFromSDS(msg.EncryptionLayer.Payload)
	if err != nil {
		return err
	}

	msg.EncryptionLayer.Payload = unwrappedPayload
	return nil
}

func (r *Processor) ProcessSharedSecrets(secrets []*sharedsecret.Secret) error {
	for _, secret := range secrets {
		_, err := r.stack.Transport.ProcessNegotiatedSecret(messagingtypes.NegotiatedSecret{
			PublicKey: secret.Identity,
			Key:       secret.Key,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Processor) processReliabilityLayer(m *messagingtypes.Message, logger *zap.Logger) ([]*messagingtypes.Message, []types.HexBytes, error) {
	if !r.stack.Reliability.Started() {
		return nil, nil, errReliabilityNotStarted
	}

	datasyncMessage, err := r.stack.Reliability.UnwrapAndAcknowledgeMessage(
		m.SigPubKey(),
		m.EncryptionLayer.Payload,
	)
	if err != nil {
		return nil, nil, err
	}

	var statusMessages []*messagingtypes.Message
	for _, ds := range datasyncMessage.Messages {
		message, err := m.Clone()
		if err != nil {
			return nil, nil, err
		}
		message.EncryptionLayer.Payload = ds.Body
		statusMessages = append(statusMessages, message)
	}

	ackedMessageIDs := make([]types.HexBytes, 0, len(datasyncMessage.Acks))
	for _, ack := range datasyncMessage.Acks {
		messageID, err := r.messageConfirmationStorage.MarkAsConfirmed(ack, true)
		if err != nil {
			logger.Info("got datasync acknowledge for message we don't have in db", zap.String("ack", hex.EncodeToString(ack)))
			continue
		}

		r.stack.Transport.ConfirmMessageDelivered(messageID.String())

		ackedMessageIDs = append(ackedMessageIDs, messageID)
	}

	return statusMessages, ackedMessageIDs, nil
}
