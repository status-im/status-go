package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/status-im/status-go/protocol/datasync"
	datasyncpeer "github.com/status-im/status-go/protocol/datasync/peer"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	transport "github.com/status-im/status-go/protocol/transport/whisper"
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	datasyncnode "github.com/vacp2p/mvds/node"
	datasyncproto "github.com/vacp2p/mvds/protobuf"
	"go.uber.org/zap"
)

// Whisper message properties.
const (
	whisperTTL     = 15
	whisperPoW     = 0.002
	whisperPoWTime = 5
)

type messageHandler interface {
	HandleMembershipUpdate(m v1protocol.MembershipUpdateMessage) error
}

type messageProcessor struct {
	identity  *ecdsa.PrivateKey
	datasync  *datasync.DataSync
	protocol  *encryption.Protocol
	transport *transport.WhisperServiceTransport
	handler   messageHandler
	logger    *zap.Logger

	featureFlags featureFlags
}

func newMessageProcessor(
	identity *ecdsa.PrivateKey,
	database *sql.DB,
	enc *encryption.Protocol,
	transport *transport.WhisperServiceTransport,
	handler messageHandler,
	logger *zap.Logger,
	features featureFlags,
) (*messageProcessor, error) {
	dataSyncTransport := datasync.NewDataSyncNodeTransport()
	dataSyncNode, err := datasyncnode.NewPersistentNode(
		database,
		dataSyncTransport,
		datasyncpeer.PublicKeyToPeerID(identity.PublicKey),
		datasyncnode.BATCH,
		datasync.CalculateSendTime,
		logger,
	)
	if err != nil {
		return nil, err
	}
	ds := datasync.New(dataSyncNode, dataSyncTransport, features.datasync, logger)

	p := &messageProcessor{
		identity:     identity,
		datasync:     ds,
		protocol:     enc,
		transport:    transport,
		handler:      handler,
		logger:       logger,
		featureFlags: features,
	}

	// Initializing DataSync is required to encrypt and send messages.
	// With DataSync enabled, messages are added to the DataSync
	// but actual encrypt and send calls are postponed.
	// sendDataSync is responsible for encrypting and sending postponed messages.
	if features.datasync {
		ds.Init(p.sendDataSync)
		ds.Start(300 * time.Millisecond)
	}

	return p, nil
}

func (p *messageProcessor) Stop() {
	p.datasync.Stop() // idempotent op
}

func (p *messageProcessor) SendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	chatID string,
	data []byte,
	clock int64,
) ([]byte, *v1protocol.Message, error) {
	message := v1protocol.CreatePrivateTextMessage(data, clock, chatID)
	encodedMessage, err := p.encodeMessage(message)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to encode message")
	}
	messageID, err := p.sendPrivate(ctx, recipient, encodedMessage)
	if err != nil {
		return nil, nil, err
	}
	return messageID, &message, nil
}

// SendPrivateRaw takes encoded data, encrypts it and sends through the wire.
func (p *messageProcessor) SendPrivateRaw(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	data []byte,
) ([]byte, error) {
	p.logger.Debug(
		"sending a private message",
		zap.Binary("public-key", crypto.FromECDSAPub(recipient)),
		zap.String("site", "SendPrivateRaw"),
	)
	return p.sendPrivate(ctx, recipient, data)
}

// sendPrivate sends data to the recipient identifying with a given public key.
func (p *messageProcessor) sendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	data []byte,
) ([]byte, error) {
	p.logger.Debug("sending private message", zap.Binary("recipient", crypto.FromECDSAPub(recipient)))

	wrappedMessage, err := p.tryWrapMessageV1(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageID := v1protocol.MessageID(&p.identity.PublicKey, wrappedMessage)

	if p.featureFlags.datasync {
		if err := p.addToDataSync(recipient, wrappedMessage); err != nil {
			return nil, errors.Wrap(err, "failed to send message with datasync")
		}

		// No need to call transport tracking.
		// It is done in a data sync dispatch step.
	} else {
		messageSpec, err := p.protocol.BuildDirectMessage(p.identity, recipient, wrappedMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encrypt message")
		}

		hash, newMessage, err := p.sendMessageSpec(ctx, recipient, messageSpec)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send a message spec")
		}

		p.transport.Track([][]byte{messageID}, hash, newMessage)
	}

	return messageID, nil
}

func (p *messageProcessor) SendGroup(
	ctx context.Context,
	recipients []*ecdsa.PublicKey,
	chatID string,
	data []byte,
	clock int64,
) ([][]byte, []*v1protocol.Message, error) {
	p.logger.Debug("sending a group message", zap.Int("membersCount", len(recipients)))

	message := v1protocol.CreatePrivateGroupTextMessage(data, clock, chatID)
	encodedMessage, err := p.encodeMessage(message)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to encode message")
	}

	var resultIDs [][]byte
	for _, recipient := range recipients {
		messageID, err := p.sendPrivate(ctx, recipient, encodedMessage)
		if err != nil {
			return nil, nil, err
		}
		resultIDs = append(resultIDs, messageID)
	}
	return resultIDs, nil, nil
}

func (p *messageProcessor) SendMembershipUpdate(
	ctx context.Context,
	recipients []*ecdsa.PublicKey,
	chatID string,
	updates []v1protocol.MembershipUpdate,
	clock int64,
) ([][]byte, error) {
	p.logger.Debug("sending a membership update", zap.Int("membersCount", len(recipients)))

	message := v1protocol.MembershipUpdateMessage{
		ChatID:  chatID,
		Updates: updates,
	}
	encodedMessage, err := v1protocol.EncodeMembershipUpdateMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode membership update message")
	}

	var resultIDs [][]byte
	for _, recipient := range recipients {
		messageID, err := p.sendPrivate(ctx, recipient, encodedMessage)
		if err != nil {
			return nil, err
		}
		resultIDs = append(resultIDs, messageID)
	}
	return resultIDs, nil
}

func (p *messageProcessor) SendPublic(ctx context.Context, chatID string, data []byte, clock int64) ([]byte, error) {
	message := v1protocol.CreatePublicTextMessage(data, clock, chatID)

	encodedMessage, err := p.encodeMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode message")
	}

	wrappedMessage, err := p.tryWrapMessageV1(encodedMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageSpec, err := p.protocol.BuildPublicMessage(p.identity, wrappedMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build public message")
	}

	newMessage, err := messageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, err
	}

	hash, err := p.transport.SendPublic(ctx, newMessage, chatID)
	if err != nil {
		return nil, err
	}

	messageID := v1protocol.MessageID(&p.identity.PublicKey, wrappedMessage)

	p.transport.Track([][]byte{messageID}, hash, newMessage)

	return messageID, nil
}

// SendPublicRaw takes encoded data, encrypts it and sends through the wire.
func (p *messageProcessor) SendPublicRaw(ctx context.Context, chatName string, data []byte) ([]byte, error) {
	var newMessage *whispertypes.NewMessage

	wrappedMessage, err := p.tryWrapMessageV1(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	newMessage = &whispertypes.NewMessage{
		TTL:       whisperTTL,
		Payload:   wrappedMessage,
		PowTarget: whisperPoW,
		PowTime:   whisperPoWTime,
	}

	hash, err := p.transport.SendPublic(ctx, newMessage, chatName)
	if err != nil {
		return nil, err
	}

	messageID := v1protocol.MessageID(&p.identity.PublicKey, wrappedMessage)

	p.transport.Track([][]byte{messageID}, hash, newMessage)

	return messageID, nil
}

// Process processes received Whisper messages through all the layers
// and returns decoded user messages.
// It also handled all non-user messages like PairMessage.
func (p *messageProcessor) Process(shhMessage *whispertypes.Message) ([]*v1protocol.Message, error) {
	logger := p.logger.With(zap.String("site", "Process"))

	var decodedMessages []*v1protocol.Message

	hlogger := logger.With(zap.Binary("hash", shhMessage.Hash))
	hlogger.Debug("handling a received message")

	statusMessages, err := p.handleMessages(shhMessage, true)
	if err != nil {
		return nil, err
	}

	for _, statusMessage := range statusMessages {
		switch m := statusMessage.ParsedMessage.(type) {
		case v1protocol.Message:
			m.ID = statusMessage.ID
			m.SigPubKey = statusMessage.SigPubKey()
			decodedMessages = append(decodedMessages, &m)
		case v1protocol.MembershipUpdateMessage:
			// Handle user message that can be attached to the membership update.
			userMessage := m.Message
			if userMessage != nil {
				userMessage.ID = statusMessage.ID
				userMessage.SigPubKey = statusMessage.SigPubKey()
				decodedMessages = append(decodedMessages, userMessage)
			}

			if err := p.processMembershipUpdate(m); err != nil {
				hlogger.Error("failed to process MembershipUpdateMessage", zap.Error(err))
			}
		case v1protocol.PairMessage:
			fromOurDevice := isPubKeyEqual(statusMessage.SigPubKey(), &p.identity.PublicKey)
			if !fromOurDevice {
				hlogger.Debug("received PairMessage from not our device, skipping")
				break
			}

			if err := p.processPairMessage(m); err != nil {
				hlogger.Error("failed to process PairMessage", zap.Error(err))
			}
		default:
			hlogger.Error(
				"skipped a public message of unsupported type",
				zap.Any("value", statusMessage.ParsedMessage),
			)
		}
	}

	return decodedMessages, nil
}

func (p *messageProcessor) processMembershipUpdate(m v1protocol.MembershipUpdateMessage) error {
	if err := m.Verify(); err != nil {
		return err
	}
	if p.handler != nil {
		return p.handler.HandleMembershipUpdate(m)
	}
	return errors.New("missing handler")
}

func (p *messageProcessor) processPairMessage(m v1protocol.PairMessage) error {
	metadata := &multidevice.InstallationMetadata{
		Name:       m.Name,
		FCMToken:   m.FCMToken,
		DeviceType: m.DeviceType,
	}
	return p.protocol.SetInstallationMetadata(&p.identity.PublicKey, m.InstallationID, metadata)
}

// handleMessages expects a whisper message as input, and it will go through
// a series of transformations until the message is parsed into an application
// layer message, or in case of Raw methods, the processing stops at the layer
// before.
// It returns an error only if the processing of required steps failed.
func (p *messageProcessor) handleMessages(shhMessage *whispertypes.Message, applicationLayer bool) ([]*v1protocol.StatusMessage, error) {
	logger := p.logger.With(zap.String("site", "handleMessages"))
	hlogger := logger.With(zap.Binary("hash", shhMessage.Hash))
	var statusMessage v1protocol.StatusMessage

	err := statusMessage.HandleTransport(shhMessage)
	if err != nil {
		hlogger.Error("failed to handle transport layer message", zap.Error(err))
		return nil, err
	}

	err = p.handleEncryptionLayer(context.Background(), &statusMessage)
	if err != nil {
		hlogger.Debug("failed to handle an encryption message", zap.Error(err))
	}

	statusMessages, err := statusMessage.HandleDatasync(p.datasync)
	if err != nil {
		hlogger.Debug("failed to handle datasync message", zap.Error(err))
	}

	for _, statusMessage := range statusMessages {
		err := statusMessage.HandleApplicationMetadata()
		if err != nil {
			hlogger.Error("failed to handle application metadata layer message", zap.Error(err))
		}

		if applicationLayer {
			err = statusMessage.HandleApplication()
			if err != nil {
				hlogger.Error("failed to handle application layer message", zap.Error(err))
			}
		}
	}

	return statusMessages, nil
}

func (p *messageProcessor) handleEncryptionLayer(ctx context.Context, message *v1protocol.StatusMessage) error {
	logger := p.logger.With(zap.String("site", "handleEncryptionLayer"))
	publicKey := message.SigPubKey()

	err := message.HandleEncryption(p.identity, publicKey, p.protocol)
	if err == encryption.ErrDeviceNotFound {
		if err := p.handleErrDeviceNotFound(ctx, publicKey); err != nil {
			logger.Error("failed to handle ErrDeviceNotFound", zap.Error(err))
		}
	}
	if err != nil {
		return errors.Wrap(err, "failed to process an encrypted message")
	}

	return nil
}

func (p *messageProcessor) handleErrDeviceNotFound(ctx context.Context, publicKey *ecdsa.PublicKey) error {
	now := time.Now().Unix()
	advertise, err := p.protocol.ShouldAdvertiseBundle(publicKey, now)
	if err != nil {
		return err
	}
	if !advertise {
		return nil
	}

	messageSpec, err := p.protocol.BuildBundleAdvertiseMessage(p.identity, publicKey)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_, _, err = p.sendMessageSpec(ctx, publicKey, messageSpec)
	if err != nil {
		return err
	}

	p.protocol.ConfirmBundleAdvertisement(publicKey, now)

	return nil
}

func (p *messageProcessor) encodeMessage(message v1protocol.Message) ([]byte, error) {
	encodedMessage, err := v1protocol.EncodeMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode message")
	}
	return encodedMessage, nil
}

func (p *messageProcessor) tryWrapMessageV1(encodedMessage []byte) ([]byte, error) {
	if p.featureFlags.sendV1Messages {
		wrappedMessage, err := v1protocol.WrapMessageV1(encodedMessage, p.identity)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap message")
		}
		return wrappedMessage, nil
	}
	return encodedMessage, nil
}

func (p *messageProcessor) addToDataSync(publicKey *ecdsa.PublicKey, message []byte) error {
	groupID := datasync.ToOneToOneGroupID(&p.identity.PublicKey, publicKey)
	peerID := datasyncpeer.PublicKeyToPeerID(*publicKey)
	exist, err := p.datasync.IsPeerInGroup(groupID, peerID)
	if err != nil {
		return errors.Wrap(err, "failed to check if peer is in group")
	}
	if !exist {
		if err := p.datasync.AddPeer(groupID, peerID); err != nil {
			return errors.Wrap(err, "failed to add peer")
		}
	}
	_, err = p.datasync.AppendMessage(groupID, message)
	if err != nil {
		return errors.Wrap(err, "failed to append message to datasync")
	}

	return nil
}

// sendDataSync sends a message scheduled by the data sync layer.
// Data Sync layer calls this method "dispatch" function.
func (p *messageProcessor) sendDataSync(ctx context.Context, publicKey *ecdsa.PublicKey, encodedMessage []byte, payload *datasyncproto.Payload) error {
	messageIDs := make([][]byte, 0, len(payload.Messages))
	for _, payload := range payload.Messages {
		messageIDs = append(messageIDs, v1protocol.MessageID(&p.identity.PublicKey, payload.Body))
	}

	messageSpec, err := p.protocol.BuildDirectMessage(p.identity, publicKey, encodedMessage)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt message")
	}

	hash, newMessage, err := p.sendMessageSpec(ctx, publicKey, messageSpec)
	if err != nil {
		return err
	}

	p.transport.Track(messageIDs, hash, newMessage)

	return nil
}

// sendMessageSpec analyses the spec properties and selects a proper transport method.
func (p *messageProcessor) sendMessageSpec(ctx context.Context, publicKey *ecdsa.PublicKey, messageSpec *encryption.ProtocolMessageSpec) ([]byte, *whispertypes.NewMessage, error) {
	newMessage, err := messageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, nil, err
	}

	logger := p.logger.With(zap.String("site", "sendMessageSpec"))

	var hash []byte

	switch {
	case messageSpec.SharedSecret != nil:
		logger.Debug("sending using shared secret")
		hash, err = p.transport.SendPrivateWithSharedSecret(ctx, newMessage, publicKey, messageSpec.SharedSecret)
	case messageSpec.PartitionedTopicMode() == encryption.PartitionTopicV1:
		logger.Debug("sending partitioned topic")
		hash, err = p.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
	case !p.featureFlags.genericDiscoveryTopicEnabled:
		logger.Debug("sending partitioned topic (generic discovery topic disabled)")
		hash, err = p.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
	default:
		logger.Debug("sending using discovery topic")
		hash, err = p.transport.SendPrivateOnDiscovery(ctx, newMessage, publicKey)
	}
	if err != nil {
		return nil, nil, err
	}

	return hash, newMessage, nil
}

func messageSpecToWhisper(spec *encryption.ProtocolMessageSpec) (*whispertypes.NewMessage, error) {
	var newMessage *whispertypes.NewMessage

	payload, err := proto.Marshal(spec.Message)
	if err != nil {
		return newMessage, err
	}

	newMessage = &whispertypes.NewMessage{
		TTL:       whisperTTL,
		Payload:   payload,
		PowTarget: whisperPoW,
		PowTime:   whisperPoWTime,
	}
	return newMessage, nil
}

// isPubKeyEqual checks that two public keys are equal
func isPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	// the curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
