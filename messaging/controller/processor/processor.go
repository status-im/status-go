package processor

import (
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/crypto"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/common"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/encryption/sharedsecret"
	"github.com/status-im/status-go/messaging/layers/segmentation"
	"github.com/status-im/status-go/messaging/types"
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
}

func NewProcessor(
	identity *ecdsa.PrivateKey,
	stack *common.MessagingStack,
	messageConfirmationStorage common.MessageConfirmationPersistence,
	hashRatchetStorage common.HashRatchetPersistence,
	logger *zap.Logger,
) *Processor {
	return &Processor{
		identity:                   identity,
		stack:                      stack,
		messageConfirmationStorage: messageConfirmationStorage,
		hashRatchetStorage:         hashRatchetStorage,
		ephemeralKeysManager:       NewEphemeralKeysManager(maxNumOfEphemeralKeys),
		publisher:                  pubsub.NewPublisher(),
		logger:                     logger.Named("processor"),
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

func (r *Processor) ProcessMessage(msg *types.ReceivedMessage) (*types.HandleMessageResponse, error) {
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

	return &types.HandleMessageResponse{
		Messages:        response.messages,
		AckedMessageIDs: response.ackedMessageIDs,
	}, nil
}

type processMessageResponse struct {
	messages        []*types.Message
	ackedMessageIDs []cryptotypes.HexBytes
}

func (r *Processor) processMessage(m *types.ReceivedMessage) (*processMessageResponse, error) {
	logger := r.logger.With(zap.Stringer("hash", cryptotypes.HexBytes(m.Hash)))
	logger.Debug("processing received message")

	responseMessage := &types.Message{}

	response := &processMessageResponse{
		messages:        []*types.Message{responseMessage},
		ackedMessageIDs: []cryptotypes.HexBytes{},
	}

	err := processTransportLayer(responseMessage, m)
	if err != nil {
		logger.Error("failed to process transport layer", zap.Error(err))
		return nil, err
	}

	isSegmentMessage, completed, err := r.processSegmentationLayer(responseMessage)
	if err != nil {
		return nil, err
	}

	// Segments not completed yet, stop processing
	if isSegmentMessage && !completed {
		return nil, nil
	}

	err = r.processEncryptionLayer(responseMessage, logger)
	if err != nil {
		logger.Debug("failed to process encryption layer", zap.Error(err))

		// Hash ratchet with a group id not found yet, save the message for future processing
		if err == encryption.ErrHashRatchetGroupIDNotFound && len(responseMessage.EncryptionLayer.HashRatchetInfo) == 1 {
			info := responseMessage.EncryptionLayer.HashRatchetInfo[0]
			return nil, r.hashRatchetStorage.SaveMessage(info.GroupID, info.KeyID, m)
		}
	}

	messages, ackedMessageIDs, err := r.processReliabilityLayer(responseMessage, logger)
	if err == nil {
		response.messages = messages
		response.ackedMessageIDs = ackedMessageIDs
	} else {
		logger.Debug("failed to process reliability layer", zap.Error(err))
	}

	return response, nil
}

func (r *Processor) processQueuedHashRatchetMessages(hashRatchetInfos []*types.HashRatchetInfo) (*processMessageResponse, error) {
	response := &processMessageResponse{
		messages:        []*types.Message{},
		ackedMessageIDs: []cryptotypes.HexBytes{},
	}

	for _, hashRatchetInfo := range hashRatchetInfos {
		messages, err := r.hashRatchetStorage.GetMessages(hashRatchetInfo.KeyID)
		if err != nil {
			return nil, err
		}

		var processedIds [][]byte
		for _, message := range messages {
			logger := r.logger.With(zap.String("hash", cryptotypes.EncodeHex(message.Hash)))
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

func processTransportLayer(m *types.Message, receivedMessage *types.ReceivedMessage) error {
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

func (r *Processor) processSegmentationLayer(m *types.Message) (segmented, completed bool, err error) {
	var reconstructedPayload []byte
	reconstructedPayload, err = r.stack.Segmentation.Reconstruct(m.TransportLayer.Payload, m.TransportLayer.SigPubKey)

	switch err {
	case nil:
		m.TransportLayer.Payload = reconstructedPayload
		segmented = true
		completed = true
	case segmentation.ErrIncomplete:
		segmented = true
		completed = false
		err = nil
	case segmentation.ErrInvalidPayload:
		segmented = false
		completed = false
		err = nil
	}

	return
}

func (r *Processor) processEncryptionLayer(m *types.Message, logger *zap.Logger) error {
	logger = logger.Named("processEncryptionLayer")

	// As we handle non-encrypted messages, we make sure that DecryptPayload
	// is set regardless of whether this step is successful
	m.EncryptionLayer.Payload = m.TransportLayer.Payload

	// if it's an ephemeral key, we don't negotiate a topic
	ephemeralKey := r.ephemeralKeysManager.GetPrivateKeyFor(m.TransportLayer.Dst)
	if ephemeralKey != nil {
		return nil
	}

	var protocolMessage encryption.ProtocolMessage
	err := proto.Unmarshal(m.TransportLayer.Payload, &protocolMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProtocolMessage")
	}

	response, err := r.stack.Encryption.HandleMessage(
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

func (r *Processor) ProcessSharedSecrets(secrets []*sharedsecret.Secret) error {
	for _, secret := range secrets {
		_, err := r.stack.Transport.ProcessNegotiatedSecret(ethtypes.NegotiatedSecret{
			PublicKey: secret.Identity,
			Key:       secret.Key,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Processor) processReliabilityLayer(m *types.Message, logger *zap.Logger) ([]*types.Message, []cryptotypes.HexBytes, error) {
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

	var statusMessages []*types.Message
	for _, ds := range datasyncMessage.Messages {
		message, err := m.Clone()
		if err != nil {
			return nil, nil, err
		}
		message.EncryptionLayer.Payload = ds.Body
		statusMessages = append(statusMessages, message)
	}

	ackedMessageIDs := make([]cryptotypes.HexBytes, 0, len(datasyncMessage.Acks))
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
