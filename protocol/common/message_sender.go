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

type MessageSender struct {
	identity    *ecdsa.PrivateKey
	datasync    *datasync.DataSync
	protocol    *encryption.Protocol
	transport   *transport.Transport
	logger      *zap.Logger
	persistence *RawMessagesPersistence

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

func NewMessageSender(
	identity *ecdsa.PrivateKey,
	database *sql.DB,
	enc *encryption.Protocol,
	transport *transport.Transport,
	logger *zap.Logger,
	features FeatureFlags,
) (*MessageSender, error) {
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

	p := &MessageSender{
		identity:      identity,
		datasync:      ds,
		protocol:      enc,
		persistence:   NewRawMessagesPersistence(database),
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

func (s *MessageSender) Stop() {
	for _, c := range s.sentMessagesSubscriptions {
		close(c)
	}
	s.sentMessagesSubscriptions = nil
	s.datasync.Stop() // idempotent op
}

func (s *MessageSender) SetHandleSharedSecrets(handler func([]*sharedsecret.Secret) error) {
	s.handleSharedSecrets = handler
}

// SendPrivate takes encoded data, encrypts it and sends through the wire.
func (s *MessageSender) SendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage *RawMessage,
) ([]byte, error) {
	s.logger.Debug(
		"sending a private message",
		zap.String("public-key", types.EncodeHex(crypto.FromECDSAPub(recipient))),
		zap.String("site", "SendPrivate"),
	)
	// Currently we don't support sending through datasync and setting custom waku fields,
	// as the datasync interface is not rich enough to propagate that information, so we
	// would have to add some complexity to handle this.
	if rawMessage.ResendAutomatically && (rawMessage.Sender != nil || rawMessage.SkipEncryption || rawMessage.SendOnPersonalTopic) {
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
	recipient *ecdsa.PublicKey,
	rawMessage RawMessage,
) ([]byte, error) {
	s.logger.Debug(
		"sending a community message",
		zap.String("public-key", types.EncodeHex(crypto.FromECDSAPub(recipient))),
		zap.String("site", "SendPrivate"),
	)
	rawMessage.Sender = s.identity

	return s.sendCommunity(ctx, recipient, &rawMessage)
}

// SendGroup takes encoded data, encrypts it and sends through the wire,
// always return the messageID
func (s *MessageSender) SendGroup(
	ctx context.Context,
	recipients []*ecdsa.PublicKey,
	rawMessage RawMessage,
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
	wrappedMessage, err := s.wrapMessageV1(&rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}
	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	rawMessage.ID = types.EncodeHex(messageID)

	// Send to each recipients
	for _, recipient := range recipients {
		_, err = s.sendPrivate(ctx, recipient, &rawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send message")
		}
	}
	return messageID, nil
}

// sendCommunity sends data to the recipient identifying with a given public key.
func (s *MessageSender) sendCommunity(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage *RawMessage,
) ([]byte, error) {
	s.logger.Debug("sending community message", zap.String("recipient", types.EncodeHex(crypto.FromECDSAPub(recipient))))

	wrappedMessage, err := s.wrapMessageV1(rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	rawMessage.ID = types.EncodeHex(messageID)

	// Notify before dispatching, otherwise the dispatch subscription might happen
	// earlier than the scheduled
	s.notifyOnScheduledMessage(rawMessage)

	messageIDs := [][]byte{messageID}
	hash, newMessage, err := s.sendCommunityRawMessage(ctx, recipient, wrappedMessage, messageIDs)
	if err != nil {
		s.logger.Error("failed to send a community message", zap.Error(err))
		return nil, errors.Wrap(err, "failed to send a message spec")
	}

	s.transport.Track(messageIDs, hash, newMessage)

	return messageID, nil
}

// sendPrivate sends data to the recipient identifying with a given public key.
func (s *MessageSender) sendPrivate(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage *RawMessage,
) ([]byte, error) {
	s.logger.Debug("sending private message", zap.String("recipient", types.EncodeHex(crypto.FromECDSAPub(recipient))))

	wrappedMessage, err := s.wrapMessageV1(rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageID := v1protocol.MessageID(&rawMessage.Sender.PublicKey, wrappedMessage)
	rawMessage.ID = types.EncodeHex(messageID)

	// Notify before dispatching, otherwise the dispatch subscription might happen
	// earlier than the scheduled
	s.notifyOnScheduledMessage(rawMessage)

	if s.featureFlags.Datasync && rawMessage.ResendAutomatically {
		// No need to call transport tracking.
		// It is done in a data sync dispatch step.
		datasyncID, err := s.addToDataSync(recipient, wrappedMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send message with datasync")
		}
		// We don't need to receive confirmations from our own devices
		if !IsPubKeyEqual(recipient, &s.identity.PublicKey) {
			confirmation := &RawMessageConfirmation{
				DataSyncID: datasyncID,
				MessageID:  messageID,
				PublicKey:  crypto.CompressPubkey(recipient),
			}

			err = s.persistence.InsertPendingConfirmation(confirmation)
			if err != nil {
				return nil, err
			}
		}
	} else if rawMessage.SkipEncryption {
		// When SkipEncryption is set we don't pass the message to the encryption layer
		messageIDs := [][]byte{messageID}
		hash, newMessage, err := s.sendPrivateRawMessage(ctx, rawMessage, recipient, wrappedMessage, messageIDs)
		if err != nil {
			s.logger.Error("failed to send a private message", zap.Error(err))
			return nil, errors.Wrap(err, "failed to send a message spec")
		}

		s.transport.Track(messageIDs, hash, newMessage)

	} else {
		messageSpec, err := s.protocol.BuildDirectMessage(rawMessage.Sender, recipient, wrappedMessage)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encrypt message")
		}

		// The shared secret needs to be handle before we send a message
		// otherwise the topic might not be set up before we receive a message
		if s.handleSharedSecrets != nil {
			err := s.handleSharedSecrets([]*sharedsecret.Secret{messageSpec.SharedSecret})
			if err != nil {
				return nil, err
			}

		}

		messageIDs := [][]byte{messageID}
		hash, newMessage, err := s.sendMessageSpec(ctx, recipient, messageSpec, messageIDs)
		if err != nil {
			s.logger.Error("failed to send a private message", zap.Error(err))
			return nil, errors.Wrap(err, "failed to send a message spec")
		}

		s.transport.Track(messageIDs, hash, newMessage)
	}

	return messageID, nil
}

// sendPairInstallation sends data to the recipients, using DH
func (s *MessageSender) SendPairInstallation(
	ctx context.Context,
	recipient *ecdsa.PublicKey,
	rawMessage RawMessage,
) ([]byte, error) {
	s.logger.Debug("sending private message", zap.String("recipient", types.EncodeHex(crypto.FromECDSAPub(recipient))))

	wrappedMessage, err := s.wrapMessageV1(&rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageSpec, err := s.protocol.BuildDHMessage(s.identity, recipient, wrappedMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt message")
	}

	messageID := v1protocol.MessageID(&s.identity.PublicKey, wrappedMessage)
	messageIDs := [][]byte{messageID}

	hash, newMessage, err := s.sendMessageSpec(ctx, recipient, messageSpec, messageIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send a message spec")
	}

	s.transport.Track(messageIDs, hash, newMessage)

	return messageID, nil
}

func (s *MessageSender) encodeMembershipUpdate(
	message v1protocol.MembershipUpdateMessage,
	chatEntity ChatEntity,
) ([]byte, error) {

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

// EncodeMembershipUpdate takes a group and an optional chat message and returns the protobuf representation to be sent on the wire.
// All the events in a group are encoded and added to the payload
func (s *MessageSender) EncodeMembershipUpdate(
	group *v1protocol.Group,
	chatEntity ChatEntity,
) ([]byte, error) {
	message := v1protocol.MembershipUpdateMessage{
		ChatID: group.ChatID(),
		Events: group.Events(),
	}

	return s.encodeMembershipUpdate(message, chatEntity)
}

// EncodeAbridgedMembershipUpdate takes a group and an optional chat message and returns the protobuf representation to be sent on the wire.
// Only the events relevant to the sender are encoded
func (s *MessageSender) EncodeAbridgedMembershipUpdate(
	group *v1protocol.Group,
	chatEntity ChatEntity,
) ([]byte, error) {
	message := v1protocol.MembershipUpdateMessage{
		ChatID: group.ChatID(),
		Events: group.AbridgedEvents(&s.identity.PublicKey),
	}
	return s.encodeMembershipUpdate(message, chatEntity)
}

// SendPublic takes encoded data, encrypts it and sends through the wire.
func (s *MessageSender) SendPublic(
	ctx context.Context,
	chatName string,
	rawMessage RawMessage,
) ([]byte, error) {
	// Set sender
	if rawMessage.Sender == nil {
		rawMessage.Sender = s.identity
	}

	wrappedMessage, err := s.wrapMessageV1(&rawMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	var newMessage *types.NewMessage

	messageSpec, err := s.protocol.BuildPublicMessage(s.identity, wrappedMessage)
	if err != nil {
		s.logger.Error("failed to send a public message", zap.Error(err))
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
	s.notifyOnScheduledMessage(&rawMessage)

	hash, err := s.transport.SendPublic(ctx, newMessage, chatName)
	if err != nil {
		return nil, err
	}

	sentMessage := &SentMessage{
		Spec:       messageSpec,
		MessageIDs: [][]byte{messageID},
	}

	s.notifyOnSentMessage(sentMessage)

	s.transport.Track([][]byte{messageID}, hash, newMessage)

	return messageID, nil
}

// unwrapDatasyncMessage tries to unwrap message as datasync one and in case of success
// returns cloned messages with replaced payloads
func unwrapDatasyncMessage(m *v1protocol.StatusMessage, datasync *datasync.DataSync) ([]*v1protocol.StatusMessage, [][]byte, error) {
	var statusMessages []*v1protocol.StatusMessage

	payloads, acks, err := datasync.UnwrapPayloadsAndAcks(
		m.SigPubKey(),
		m.DecryptedPayload,
	)
	if err != nil {
		return nil, nil, err
	}

	for _, payload := range payloads {
		message, err := m.Clone()
		if err != nil {
			return nil, nil, err
		}
		message.DecryptedPayload = payload
		statusMessages = append(statusMessages, message)
	}
	return statusMessages, acks, nil
}

// HandleMessages expects a whisper message as input, and it will go through
// a series of transformations until the message is parsed into an application
// layer message, or in case of Raw methods, the processing stops at the layer
// before.
// It returns an error only if the processing of required steps failed.
func (s *MessageSender) HandleMessages(shhMessage *types.Message, applicationLayer bool) ([]*v1protocol.StatusMessage, [][]byte, error) {
	logger := s.logger.With(zap.String("site", "handleMessages"))
	hlogger := logger.With(zap.ByteString("hash", shhMessage.Hash))
	var statusMessage v1protocol.StatusMessage
	var statusMessages []*v1protocol.StatusMessage

	err := statusMessage.HandleTransport(shhMessage)
	if err != nil {
		hlogger.Error("failed to handle transport layer message", zap.Error(err))
		return nil, nil, err
	}

	err = s.handleEncryptionLayer(context.Background(), &statusMessage)
	if err != nil {
		hlogger.Debug("failed to handle an encryption message", zap.Error(err))
	}

	statusMessages, acks, err := unwrapDatasyncMessage(&statusMessage, s.datasync)
	if err != nil {
		hlogger.Debug("failed to handle datasync message", zap.Error(err))
		//that wasn't a datasync message, so use the original payload
		statusMessages = append(statusMessages, &statusMessage)
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

	return statusMessages, acks, nil
}

// fetchDecryptionKey returns the private key associated with this public key, and returns true if it's an ephemeral key
func (s *MessageSender) fetchDecryptionKey(destination *ecdsa.PublicKey) (*ecdsa.PrivateKey, bool) {
	destinationID := types.EncodeHex(crypto.FromECDSAPub(destination))

	s.ephemeralKeysMutex.Lock()
	decryptionKey, ok := s.ephemeralKeys[destinationID]
	s.ephemeralKeysMutex.Unlock()

	// the key is not there, fallback on identity
	if !ok {
		return s.identity, false
	}
	return decryptionKey, true
}

func (s *MessageSender) handleEncryptionLayer(ctx context.Context, message *v1protocol.StatusMessage) error {
	logger := s.logger.With(zap.String("site", "handleEncryptionLayer"))
	publicKey := message.SigPubKey()

	// if it's an ephemeral key, we don't negotiate a topic
	decryptionKey, skipNegotiation := s.fetchDecryptionKey(message.Dst)

	err := message.HandleEncryption(decryptionKey, publicKey, s.protocol, skipNegotiation)

	// if it's an ephemeral key, we don't have to handle a device not found error
	if err == encryption.ErrDeviceNotFound && !skipNegotiation {
		if err := s.handleErrDeviceNotFound(ctx, publicKey); err != nil {
			logger.Error("failed to handle ErrDeviceNotFound", zap.Error(err))
		}
	}
	if err != nil {
		return errors.Wrap(err, "failed to process an encrypted message")
	}

	return nil
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

func (s *MessageSender) wrapMessageV1(rawMessage *RawMessage) ([]byte, error) {
	wrappedMessage, err := v1protocol.WrapMessageV1(rawMessage.Payload, rawMessage.MessageType, rawMessage.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}
	return wrappedMessage, nil
}

func (s *MessageSender) addToDataSync(publicKey *ecdsa.PublicKey, message []byte) ([]byte, error) {
	groupID := datasync.ToOneToOneGroupID(&s.identity.PublicKey, publicKey)
	peerID := datasyncpeer.PublicKeyToPeerID(*publicKey)
	exist, err := s.datasync.IsPeerInGroup(groupID, peerID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if peer is in group")
	}
	if !exist {
		if err := s.datasync.AddPeer(groupID, peerID); err != nil {
			return nil, errors.Wrap(err, "failed to add peer")
		}
	}
	id, err := s.datasync.AppendMessage(groupID, message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to append message to datasync")
	}

	return id[:], nil
}

// sendDataSync sends a message scheduled by the data sync layer.
// Data Sync layer calls this method "dispatch" function.
func (s *MessageSender) sendDataSync(ctx context.Context, publicKey *ecdsa.PublicKey, marshalledDatasyncPayload []byte, payload *datasyncproto.Payload) error {
	// Calculate the messageIDs
	messageIDs := make([][]byte, 0, len(payload.Messages))
	for _, payload := range payload.Messages {
		messageIDs = append(messageIDs, v1protocol.MessageID(&s.identity.PublicKey, payload.Body))
	}

	messageSpec, err := s.protocol.BuildDirectMessage(s.identity, publicKey, marshalledDatasyncPayload)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt message")
	}

	// The shared secret needs to be handle before we send a message
	// otherwise the topic might not be set up before we receive a message
	if s.handleSharedSecrets != nil {
		err := s.handleSharedSecrets([]*sharedsecret.Secret{messageSpec.SharedSecret})
		if err != nil {
			return err
		}

	}

	hash, newMessage, err := s.sendMessageSpec(ctx, publicKey, messageSpec, messageIDs)
	if err != nil {
		s.logger.Error("failed to send a datasync message", zap.Error(err))
		return err
	}

	s.transport.Track(messageIDs, hash, newMessage)

	return nil
}

// sendPrivateRawMessage sends a message not wrapped in an encryption layer
func (s *MessageSender) sendPrivateRawMessage(ctx context.Context, rawMessage *RawMessage, publicKey *ecdsa.PublicKey, payload []byte, messageIDs [][]byte) ([]byte, *types.NewMessage, error) {
	newMessage := &types.NewMessage{
		TTL:       whisperTTL,
		Payload:   payload,
		PowTarget: calculatePoW(payload),
		PowTime:   whisperPoWTime,
	}
	var hash []byte
	var err error

	if rawMessage.SendOnPersonalTopic {
		hash, err = s.transport.SendPrivateOnPersonalTopic(ctx, newMessage, publicKey)
	} else {
		hash, err = s.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
	}
	if err != nil {
		return nil, nil, err
	}

	return hash, newMessage, nil
}

// sendCommunityRawMessage sends a message not wrapped in an encryption layer
// to a community
func (s *MessageSender) sendCommunityRawMessage(ctx context.Context, publicKey *ecdsa.PublicKey, payload []byte, messageIDs [][]byte) ([]byte, *types.NewMessage, error) {
	newMessage := &types.NewMessage{
		TTL:       whisperTTL,
		Payload:   payload,
		PowTarget: calculatePoW(payload),
		PowTime:   whisperPoWTime,
	}

	hash, err := s.transport.SendCommunityMessage(ctx, newMessage, publicKey)
	if err != nil {
		return nil, nil, err
	}

	return hash, newMessage, nil
}

// sendMessageSpec analyses the spec properties and selects a proper transport method.
func (s *MessageSender) sendMessageSpec(ctx context.Context, publicKey *ecdsa.PublicKey, messageSpec *encryption.ProtocolMessageSpec, messageIDs [][]byte) ([]byte, *types.NewMessage, error) {
	newMessage, err := MessageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, nil, err
	}

	logger := s.logger.With(zap.String("site", "sendMessageSpec"))

	var hash []byte

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

	sentMessage := &SentMessage{
		PublicKey:  publicKey,
		Spec:       messageSpec,
		MessageIDs: messageIDs,
	}

	s.notifyOnSentMessage(sentMessage)

	return hash, newMessage, nil
}

// SubscribeToSentMessages returns a channel where we publish every time a message is sent
func (s *MessageSender) SubscribeToSentMessages() <-chan *SentMessage {
	c := make(chan *SentMessage, 100)
	s.sentMessagesSubscriptions = append(s.sentMessagesSubscriptions, c)
	return c
}

func (s *MessageSender) notifyOnSentMessage(sentMessage *SentMessage) {
	// Publish on channels, drop if buffer is full
	for _, c := range s.sentMessagesSubscriptions {
		select {
		case c <- sentMessage:
		default:
			s.logger.Warn("sent messages subscription channel full, dropping message")
		}
	}

}

// SubscribeToScheduledMessages returns a channel where we publish every time a message is scheduled for sending
func (s *MessageSender) SubscribeToScheduledMessages() <-chan *RawMessage {
	c := make(chan *RawMessage, 100)
	s.scheduledMessagesSubscriptions = append(s.scheduledMessagesSubscriptions, c)
	return c
}

func (s *MessageSender) notifyOnScheduledMessage(message *RawMessage) {
	// Publish on channels, drop if buffer is full
	for _, c := range s.scheduledMessagesSubscriptions {
		select {
		case c <- message:
		default:
			s.logger.Warn("scheduled messages subscription channel full, dropping message")
		}
	}
}

func (s *MessageSender) JoinPublic(id string) (*transport.Filter, error) {
	return s.transport.JoinPublic(id)
}

// AddEphemeralKey adds an ephemeral key that we will be listening to
// note that we never removed them from now, as waku/whisper does not
// recalculate topics on removal, so effectively there's no benefit.
// On restart they will be gone.
func (s *MessageSender) AddEphemeralKey(privateKey *ecdsa.PrivateKey) (*transport.Filter, error) {
	s.ephemeralKeysMutex.Lock()
	s.ephemeralKeys[types.EncodeHex(crypto.FromECDSAPub(&privateKey.PublicKey))] = privateKey
	s.ephemeralKeysMutex.Unlock()
	return s.transport.LoadKeyFilters(privateKey)
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
