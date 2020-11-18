package common

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"sync"
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
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
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

// SentMessage reprent a message that has been passed to the transport layer
type SentMessage struct {
	PublicKey  *ecdsa.PublicKey
	Spec       *encryption.ProtocolMessageSpec
	MessageIDs [][]byte
}

type MessageProcessor struct {
	identity  *ecdsa.PrivateKey
	datasync  *datasync.DataSync
	protocol  *encryption.Protocol
	transport transport.Transport
	logger    *zap.Logger

	// ephemeralKeys is a map that contains the ephemeral keys of the client, used
	// to decrypt messages
	ephemeralKeys      map[string]*ecdsa.PrivateKey
	ephemeralKeysMutex sync.Mutex

	// sentMessagesSubscriptions contains all the subscriptions for sent messages
	sentMessagesSubscriptions []chan<- *SentMessage
	// sentMessagesSubscriptions contains all the subscriptions for scheduled messages
	scheduledMessagesSubscriptions []chan<- *RawMessage

	featureFlags FeatureFlags

	// handleSharedSecrets is a callback that is called every time a new shared secret is negotiated
	handleSharedSecrets func([]*sharedsecret.Secret) error
}

func NewMessageProcessor(
	identity *ecdsa.PrivateKey,
	database *sql.DB,
	enc *encryption.Protocol,
	transport transport.Transport,
	logger *zap.Logger,
	features FeatureFlags,
) (*MessageProcessor, error) {
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
	ds := datasync.New(dataSyncNode, dataSyncTransport, features.Datasync, logger)

	p := &MessageProcessor{
		identity:      identity,
		datasync:      ds,
		protocol:      enc,
		transport:     transport,
		logger:        logger,
		ephemeralKeys: make(map[string]*ecdsa.PrivateKey),
		featureFlags:  features,
	}

	// Initializing DataSync is required to encrypt and send messages.
	// With DataSync enabled, messages are added to the DataSync
	// but actual encrypt and send calls are postponed.
	// sendDataSync is responsible for encrypting and sending postponed messages.
	if features.Datasync {
		// We set the max message size to 3/4 of the allowed message size, to leave
		// room for encryption.
		// Messages will be tried to send in any case, even if they exceed this
		// value
		ds.Init(p.sendDataSync, transport.MaxMessageSize()/4*3, logger)
		ds.Start(datasync.DatasyncTicker)
	}

	return p, nil
}

func (p *MessageProcessor) Stop() {
	for _, c := range p.sentMessagesSubscriptions {
		close(c)
	}
	p.sentMessagesSubscriptions = nil
	p.datasync.Stop() // idempotent op
}

func (p *MessageProcessor) SetHandleSharedSecrets(handler func([]*sharedsecret.Secret) error) {
	p.handleSharedSecrets = handler
}

// SendPrivate takes encoded data, encrypts it and sends through the wire.
func (p *MessageProcessor) SendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage RawMessage,
) ([]byte, error) {
	p.logger.Debug(
		"sending a private message",
		zap.String("public-key", types.EncodeHex(crypto.FromECDSAPub(recipient))),
		zap.String("site", "SendPrivate"),
	)
	// Currently we don't support sending through datasync and setting custom waku fields,
	// as the datasync interface is not rich enough to propagate that information, so we
	// would have to add some complexity to handle this.
	if rawMessage.ResendAutomatically && (rawMessage.Sender != nil || rawMessage.SkipEncryption) {
		return nil, errors.New("setting identity, skip-encryption and datasync not supported")
	}

	// Set sender identity if not specified
	if rawMessage.Sender == nil {
		rawMessage.Sender = p.identity
	}

	return p.sendPrivate(ctx, recipient, &rawMessage)
}

// SendGroup takes encoded data, encrypts it and sends through the wire,
// always return the messageID
func (p *MessageProcessor) SendGroup(
	ctx context.Context,
	recipients []*ecdsa.PublicKey,
	rawMessage RawMessage,
) ([]byte, error) {
	p.logger.Debug(
		"sending a private group message",
		zap.String("site", "SendGroup"),
	)
	// Set sender if not specified
	if rawMessage.Sender == nil {
		rawMessage.Sender = p.identity
	}

	// Calculate messageID first and set on raw message
	wrappedMessage, err := p.wrapMessageV1(&rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}
	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	rawMessage.ID = types.EncodeHex(messageID)

	// Send to each recipients
	for _, recipient := range recipients {
		_, err = p.sendPrivate(ctx, recipient, &rawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send message")
		}
	}
	return messageID, nil
}

// sendPrivate sends data to the recipient identifying with a given public key.
func (p *MessageProcessor) sendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage *RawMessage,
) ([]byte, error) {
	p.logger.Debug("sending private message", zap.String("recipient", types.EncodeHex(crypto.FromECDSAPub(recipient))))

	wrappedMessage, err := p.wrapMessageV1(rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	rawMessage.ID = types.EncodeHex(messageID)

	// Notify before dispatching, otherwise the dispatch subscription might happen
	// earlier than the scheduled
	p.notifyOnScheduledMessage(rawMessage)

	if p.featureFlags.Datasync && rawMessage.ResendAutomatically {
		// No need to call transport tracking.
		// It is done in a data sync dispatch step.
		if err := p.addToDataSync(recipient, wrappedMessage); err != nil {
			return nil, errors.Wrap(err, "failed to send message with datasync")
		}

	} else if rawMessage.SkipEncryption {
		// When SkipEncryption is set we don't pass the message to the encryption layer
		messageIDs := [][]byte{messageID}
		hash, newMessage, err := p.sendPrivateRawMessage(ctx, recipient, wrappedMessage, messageIDs)
		if err != nil {
			p.logger.Error("failed to send a private message", zap.Error(err))
			return nil, errors.Wrap(err, "failed to send a message spec")
		}

		p.transport.Track(messageIDs, hash, newMessage)

	} else {
		messageSpec, err := p.protocol.BuildDirectMessage(rawMessage.Sender, recipient, wrappedMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encrypt message")
		}

		// The shared secret needs to be handle before we send a message
		// otherwise the topic might not be set up before we receive a message
		if p.handleSharedSecrets != nil {
			err := p.handleSharedSecrets([]*sharedsecret.Secret{messageSpec.SharedSecret})
			if err != nil {
				return nil, err
			}

		}

		messageIDs := [][]byte{messageID}
		hash, newMessage, err := p.sendMessageSpec(ctx, recipient, messageSpec, messageIDs)
		if err != nil {
			p.logger.Error("failed to send a private message", zap.Error(err))
			return nil, errors.Wrap(err, "failed to send a message spec")
		}

		p.transport.Track(messageIDs, hash, newMessage)
	}

	return messageID, nil
}

// sendPairInstallation sends data to the recipients, using DH
func (p *MessageProcessor) SendPairInstallation(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage RawMessage,
) ([]byte, error) {
	p.logger.Debug("sending private message", zap.String("recipient", types.EncodeHex(crypto.FromECDSAPub(recipient))))

	wrappedMessage, err := p.wrapMessageV1(&rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageSpec, err := p.protocol.BuildDHMessage(p.identity, recipient, wrappedMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt message")
	}

	messageID := v1protocol.MessageID(&p.identity.PublicKey, wrappedMessage)
	messageIDs := [][]byte{messageID}

	hash, newMessage, err := p.sendMessageSpec(ctx, recipient, messageSpec, messageIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send a message spec")
	}

	p.transport.Track(messageIDs, hash, newMessage)

	return messageID, nil
}

// EncodeMembershipUpdate takes a group and an optional chat message and returns the protobuf representation to be sent on the wire.
// All the events in a group are encoded and added to the payload
func (p *MessageProcessor) EncodeMembershipUpdate(
	group *v1protocol.Group,
	chatEntity ChatEntity,
) ([]byte, error) {
	message := v1protocol.MembershipUpdateMessage{
		ChatID: group.ChatID(),
		Events: group.Events(),
	}

	if chatEntity != nil {
		chatEntityProtobuf := chatEntity.GetProtobuf()
		switch chatEntityProtobuf := chatEntityProtobuf.(type) {
		case *protobuf.ChatMessage:
			message.Message = chatEntityProtobuf
		case *protobuf.EmojiReaction:
			message.EmojiReaction = chatEntityProtobuf

		}
	}

	encodedMessage, err := v1protocol.EncodeMembershipUpdateMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode membership update message")
	}

	return encodedMessage, nil
}

// SendPublic takes encoded data, encrypts it and sends through the wire.
func (p *MessageProcessor) SendPublic(
	ctx context.Context,
	chatName string,
	rawMessage RawMessage,
) ([]byte, error) {
	// Set sender
	if rawMessage.Sender == nil {
		rawMessage.Sender = p.identity
	}

	wrappedMessage, err := p.wrapMessageV1(&rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	var newMessage *types.NewMessage

	messageSpec, err := p.protocol.BuildPublicMessage(p.identity, wrappedMessage)
	if err != nil {
		p.logger.Error("failed to send a public message", zap.Error(err))
		return nil, errors.Wrap(err, "failed to wrap a public message in the encryption layer")
	}

	if !rawMessage.SkipEncryption {
		newMessage, err = MessageSpecToWhisper(messageSpec)
		if err != nil {
			return nil, err
		}
	} else {
		newMessage = &types.NewMessage{
			TTL:       whisperTTL,
			Payload:   wrappedMessage,
			PowTarget: calculatePoW(wrappedMessage),
			PowTime:   whisperPoWTime,
		}
	}

	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	rawMessage.ID = types.EncodeHex(messageID)

	// notify before dispatching
	p.notifyOnScheduledMessage(&rawMessage)

	hash, err := p.transport.SendPublic(ctx, newMessage, chatName)
	if err != nil {
		return nil, err
	}

	sentMessage := &SentMessage{
		Spec:       messageSpec,
		MessageIDs: [][]byte{messageID},
	}

	p.notifyOnSentMessage(sentMessage)

	p.transport.Track([][]byte{messageID}, hash, newMessage)

	return messageID, nil
}

// HandleMessages expects a whisper message as input, and it will go through
// a series of transformations until the message is parsed into an application
// layer message, or in case of Raw methods, the processing stops at the layer
// before.
// It returns an error only if the processing of required steps failed.
func (p *MessageProcessor) HandleMessages(shhMessage *types.Message, applicationLayer bool) ([]*v1protocol.StatusMessage, error) {
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

// fetchDecryptionKey returns the private key associated with this public key, and returns true if it's an ephemeral key
func (p *MessageProcessor) fetchDecryptionKey(destination *ecdsa.PublicKey) (*ecdsa.PrivateKey, bool) {
	destinationID := types.EncodeHex(crypto.FromECDSAPub(destination))

	p.ephemeralKeysMutex.Lock()
	decryptionKey, ok := p.ephemeralKeys[destinationID]
	p.ephemeralKeysMutex.Unlock()

	// the key is not there, fallback on identity
	if !ok {
		return p.identity, false
	}
	return decryptionKey, true
}

func (p *MessageProcessor) handleEncryptionLayer(ctx context.Context, message *v1protocol.StatusMessage) error {
	logger := p.logger.With(zap.String("site", "handleEncryptionLayer"))
	publicKey := message.SigPubKey()

	// if it's an ephemeral key, we don't negotiate a topic
	decryptionKey, skipNegotiation := p.fetchDecryptionKey(message.Dst)

	err := message.HandleEncryption(decryptionKey, publicKey, p.protocol, skipNegotiation)

	// if it's an ephemeral key, we don't have to handle a device not found error
	if err == encryption.ErrDeviceNotFound && !skipNegotiation {
		if err := p.handleErrDeviceNotFound(ctx, publicKey); err != nil {
			logger.Error("failed to handle ErrDeviceNotFound", zap.Error(err))
		}
	}
	if err != nil {
		return errors.Wrap(err, "failed to process an encrypted message")
	}

	return nil
}

func (p *MessageProcessor) handleErrDeviceNotFound(ctx context.Context, publicKey *ecdsa.PublicKey) error {
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
	// We don't pass an array of messageIDs as no action needs to be taken
	// when sending a bundle
	_, _, err = p.sendMessageSpec(ctx, publicKey, messageSpec, nil)
	if err != nil {
		return err
	}

	p.protocol.ConfirmBundleAdvertisement(publicKey, now)

	return nil
}

func (p *MessageProcessor) wrapMessageV1(rawMessage *RawMessage) ([]byte, error) {
	wrappedMessage, err := v1protocol.WrapMessageV1(rawMessage.Payload, rawMessage.MessageType, rawMessage.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}
	return wrappedMessage, nil
}

func (p *MessageProcessor) addToDataSync(publicKey *ecdsa.PublicKey, message []byte) error {
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
func (p *MessageProcessor) sendDataSync(ctx context.Context, publicKey *ecdsa.PublicKey, encodedMessage []byte, payload *datasyncproto.Payload) error {
	// Calculate the messageIDs
	messageIDs := make([][]byte, 0, len(payload.Messages))
	for _, payload := range payload.Messages {
		messageIDs = append(messageIDs, v1protocol.MessageID(&p.identity.PublicKey, payload.Body))
	}

	messageSpec, err := p.protocol.BuildDirectMessage(p.identity, publicKey, encodedMessage)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt message")
	}

	// The shared secret needs to be handle before we send a message
	// otherwise the topic might not be set up before we receive a message
	if p.handleSharedSecrets != nil {
		err := p.handleSharedSecrets([]*sharedsecret.Secret{messageSpec.SharedSecret})
		if err != nil {
			return err
		}

	}

	hash, newMessage, err := p.sendMessageSpec(ctx, publicKey, messageSpec, messageIDs)
	if err != nil {
		p.logger.Error("failed to send a datasync message", zap.Error(err))
		return err
	}

	p.transport.Track(messageIDs, hash, newMessage)

	return nil
}

// sendPrivateRawMessage sends a message not wrapped in an encryption layer
func (p *MessageProcessor) sendPrivateRawMessage(ctx context.Context, publicKey *ecdsa.PublicKey, payload []byte, messageIDs [][]byte) ([]byte, *types.NewMessage, error) {
	newMessage := &types.NewMessage{
		TTL:       whisperTTL,
		Payload:   payload,
		PowTarget: calculatePoW(payload),
		PowTime:   whisperPoWTime,
	}

	hash, err := p.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
	if err != nil {
		return nil, nil, err
	}

	return hash, newMessage, nil
}

// sendMessageSpec analyses the spec properties and selects a proper transport method.
func (p *MessageProcessor) sendMessageSpec(ctx context.Context, publicKey *ecdsa.PublicKey, messageSpec *encryption.ProtocolMessageSpec, messageIDs [][]byte) ([]byte, *types.NewMessage, error) {
	newMessage, err := MessageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, nil, err
	}

	logger := p.logger.With(zap.String("site", "sendMessageSpec"))

	var hash []byte

	// process shared secret
	if messageSpec.AgreedSecret {
		logger.Debug("sending using shared secret")
		hash, err = p.transport.SendPrivateWithSharedSecret(ctx, newMessage, publicKey, messageSpec.SharedSecret.Key)
	} else {
		logger.Debug("sending partitioned topic")
		hash, err = p.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
	}
	if err != nil {
		return nil, nil, err
	}

	sentMessage := &SentMessage{
		PublicKey:  publicKey,
		Spec:       messageSpec,
		MessageIDs: messageIDs,
	}

	p.notifyOnSentMessage(sentMessage)

	return hash, newMessage, nil
}

// SubscribeToSentMessages returns a channel where we publish every time a message is sent
func (p *MessageProcessor) SubscribeToSentMessages() <-chan *SentMessage {
	c := make(chan *SentMessage, 100)
	p.sentMessagesSubscriptions = append(p.sentMessagesSubscriptions, c)
	return c
}

func (p *MessageProcessor) notifyOnSentMessage(sentMessage *SentMessage) {
	// Publish on channels, drop if buffer is full
	for _, c := range p.sentMessagesSubscriptions {
		select {
		case c <- sentMessage:
		default:
			p.logger.Warn("sent messages subscription channel full, dropping message")
		}
	}

}

// SubscribeToScheduledMessages returns a channel where we publish every time a message is scheduled for sending
func (p *MessageProcessor) SubscribeToScheduledMessages() <-chan *RawMessage {
	c := make(chan *RawMessage, 100)
	p.scheduledMessagesSubscriptions = append(p.scheduledMessagesSubscriptions, c)
	return c
}

func (p *MessageProcessor) notifyOnScheduledMessage(message *RawMessage) {
	// Publish on channels, drop if buffer is full
	for _, c := range p.scheduledMessagesSubscriptions {
		select {
		case c <- message:
		default:
			p.logger.Warn("scheduled messages subscription channel full, dropping message")
		}
	}
}

func (p *MessageProcessor) JoinPublic(chatID string) error {
	return p.transport.JoinPublic(chatID)
}

// AddEphemeralKey adds an ephemeral key that we will be listening to
// note that we never removed them from now, as waku/whisper does not
// recalculate topics on removal, so effectively there's no benefit.
// On restart they will be gone.
func (p *MessageProcessor) AddEphemeralKey(privateKey *ecdsa.PrivateKey) (*transport.Filter, error) {
	p.ephemeralKeysMutex.Lock()
	p.ephemeralKeys[types.EncodeHex(crypto.FromECDSAPub(&privateKey.PublicKey))] = privateKey
	p.ephemeralKeysMutex.Unlock()
	return p.transport.LoadKeyFilters(privateKey)
}

func MessageSpecToWhisper(spec *encryption.ProtocolMessageSpec) (*types.NewMessage, error) {
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
// We check the size and arbitrarily set it to a lower PoW if the packet is
// greater than 50KB. We do this as the defaultPoW is too high for clients to send
// large messages.
func calculatePoW(payload []byte) float64 {
	if len(payload) > largeSizeInBytes {
		return whisperLargeSizePoW
	}
	return whisperDefaultPoW
}
