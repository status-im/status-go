package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	datasyncnode "github.com/vacp2p/mvds/node"
	datasyncproto "github.com/vacp2p/mvds/protobuf"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/datasync"
	datasyncpeer "github.com/status-im/status-go/protocol/datasync/peer"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

// Whisper message properties.
const (
	whisperTTL        = 15
	whisperDefaultPoW = 0.002
	// whisperLargeSizePoW is the PoWTarget for larger payload sizes
	whisperLargeSizePoW = 0.000002
	// largeSizeInBytes is when should we be using a lower POW.
	// Roughly this is 50KB
	largeSizeInBytes = 50000
	whisperPoWTime   = 5
)

type messageProcessor struct {
	identity  *ecdsa.PrivateKey
	datasync  *datasync.DataSync
	protocol  *encryption.Protocol
	transport transport.Transport
	logger    *zap.Logger

	featureFlags featureFlags
}

func newMessageProcessor(
	identity *ecdsa.PrivateKey,
	database *sql.DB,
	enc *encryption.Protocol,
	transport transport.Transport,
	logger *zap.Logger,
	features featureFlags,
) (*messageProcessor, error) {
	dataSyncTransport := datasync.NewNodeTransport()
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

// SendPrivateRaw takes encoded data, encrypts it and sends through the wire.
func (p *messageProcessor) SendPrivateRaw(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	data []byte,
	messageType protobuf.ApplicationMetadataMessage_Type,
) ([]byte, error) {
	p.logger.Debug(
		"sending a private message",
		zap.Binary("public-key", crypto.FromECDSAPub(recipient)),
		zap.String("site", "SendPrivateRaw"),
	)
	return p.sendPrivate(ctx, recipient, data, messageType)
}

// SendGroupRaw takes encoded data, encrypts it and sends through the wire,
// always return the messageID
func (p *messageProcessor) SendGroupRaw(
	ctx context.Context,
	recipients []*ecdsa.PublicKey,
	data []byte,
	messageType protobuf.ApplicationMetadataMessage_Type,
) ([]byte, error) {
	p.logger.Debug(
		"sending a private group message",
		zap.String("site", "SendGroupRaw"),
	)
	// Calculate messageID first
	wrappedMessage, err := p.wrapMessageV1(data, messageType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageID := v1protocol.MessageID(&p.identity.PublicKey, wrappedMessage)

	for _, recipient := range recipients {
		_, err = p.sendPrivate(ctx, recipient, data, messageType)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send message")
		}
	}
	return messageID, nil
}

// sendPrivate sends data to the recipient identifying with a given public key.
func (p *messageProcessor) sendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	data []byte,
	messageType protobuf.ApplicationMetadataMessage_Type,
) ([]byte, error) {
	p.logger.Debug("sending private message", zap.Binary("recipient", crypto.FromECDSAPub(recipient)))

	wrappedMessage, err := p.wrapMessageV1(data, messageType)
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

// sendPairInstallation sends data to the recipients, using DH
func (p *messageProcessor) SendPairInstallation(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	data []byte,
	messageType protobuf.ApplicationMetadataMessage_Type,
) ([]byte, error) {
	p.logger.Debug("sending private message", zap.Binary("recipient", crypto.FromECDSAPub(recipient)))

	wrappedMessage, err := p.wrapMessageV1(data, messageType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageSpec, err := p.protocol.BuildDHMessage(p.identity, recipient, wrappedMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt message")
	}

	hash, newMessage, err := p.sendMessageSpec(ctx, recipient, messageSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send a message spec")
	}

	messageID := v1protocol.MessageID(&p.identity.PublicKey, wrappedMessage)
	p.transport.Track([][]byte{messageID}, hash, newMessage)

	return messageID, nil
}

// EncodeMembershipUpdate takes a group and an optional chat message and returns the protobuf representation to be sent on the wire.
// All the events in a group are encoded and added to the payload
func (p *messageProcessor) EncodeMembershipUpdate(
	group *v1protocol.Group,
	chatMessage *protobuf.ChatMessage,
) ([]byte, error) {

	message := v1protocol.MembershipUpdateMessage{
		ChatID:  group.ChatID(),
		Events:  group.Events(),
		Message: chatMessage,
	}
	encodedMessage, err := v1protocol.EncodeMembershipUpdateMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode membership update message")
	}

	return encodedMessage, nil
}

// SendPublicRaw takes encoded data, encrypts it and sends through the wire.
func (p *messageProcessor) SendPublicRaw(
	ctx context.Context,
	chatName string,
	data []byte,
	messageType protobuf.ApplicationMetadataMessage_Type,
) ([]byte, error) {
	var newMessage *types.NewMessage

	wrappedMessage, err := p.wrapMessageV1(data, messageType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	newMessage = &types.NewMessage{
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

// handleMessages expects a whisper message as input, and it will go through
// a series of transformations until the message is parsed into an application
// layer message, or in case of Raw methods, the processing stops at the layer
// before.
// It returns an error only if the processing of required steps failed.
func (p *messageProcessor) handleMessages(shhMessage *types.Message, applicationLayer bool) ([]*v1protocol.StatusMessage, error) {
	logger := p.logger.With(zap.String("site", "handleMessages"))
	hlogger := logger.With(zap.ByteString("hash", shhMessage.Hash))
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

func (p *messageProcessor) wrapMessageV1(encodedMessage []byte, messageType protobuf.ApplicationMetadataMessage_Type) ([]byte, error) {
	wrappedMessage, err := v1protocol.WrapMessageV1(encodedMessage, messageType, p.identity)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}
	return wrappedMessage, nil
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
func (p *messageProcessor) sendMessageSpec(ctx context.Context, publicKey *ecdsa.PublicKey, messageSpec *encryption.ProtocolMessageSpec) ([]byte, *types.NewMessage, error) {
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
	default:
		logger.Debug("sending partitioned topic")
		hash, err = p.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
	}
	if err != nil {
		return nil, nil, err
	}

	return hash, newMessage, nil
}

func messageSpecToWhisper(spec *encryption.ProtocolMessageSpec) (*types.NewMessage, error) {
	var newMessage *types.NewMessage

	payload, err := proto.Marshal(spec.Message)
	if err != nil {
		return newMessage, err
	}

	newMessage = &types.NewMessage{
		TTL:       whisperTTL,
		Payload:   payload,
		PowTarget: calculatePoW(payload),
		PowTime:   whisperPoWTime,
	}
	return newMessage, nil
}

// calculatePoW returns the PoWTarget to be used.
// We check the size and arbitrarely set it to a lower PoW if the packet is
// greater than 50KB. We do this as the defaultPoW is to high for clients to send
// large messages.
func calculatePoW(payload []byte) float64 {
	if len(payload) > largeSizeInBytes {
		return whisperLargeSizePoW
	}
	return whisperDefaultPoW
}

// isPubKeyEqual checks that two public keys are equal
func isPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	// the curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
