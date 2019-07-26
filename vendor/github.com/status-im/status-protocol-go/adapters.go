package statusproto

import (
	"context"
	"crypto/ecdsa"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-protocol-go/encryption/sharedsecret"
	"github.com/status-im/status-protocol-go/transport/whisper/filter"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/status-im/status-protocol-go/encryption"
	"github.com/status-im/status-protocol-go/encryption/multidevice"
	transport "github.com/status-im/status-protocol-go/transport/whisper"
	protocol "github.com/status-im/status-protocol-go/v1"

	"github.com/status-im/status-protocol-go/datasync"
	datasyncpeer "github.com/status-im/status-protocol-go/datasync/peer"
)

// Whisper message properties.
const (
	whisperTTL     = 15
	whisperPoW     = 0.002
	whisperPoWTime = 5
)

// whisperAdapter is a bridge between encryption and transport
// layers.
type whisperAdapter struct {
	privateKey *ecdsa.PrivateKey
	transport  *transport.WhisperServiceTransport
	protocol   *encryption.Protocol
	datasync   *datasync.DataSync
	logger     *zap.Logger

	featureFlags featureFlags
}

func newWhisperAdapter(
	pk *ecdsa.PrivateKey,
	t *transport.WhisperServiceTransport,
	p *encryption.Protocol,
	d *datasync.DataSync,
	featureFlags featureFlags,
	logger *zap.Logger,
) *whisperAdapter {
	if logger == nil {
		logger = zap.NewNop()
	}

	adapter := &whisperAdapter{
		privateKey:   pk,
		transport:    t,
		protocol:     p,
		datasync:     d,
		featureFlags: featureFlags,
		logger:       logger.With(zap.Namespace("whisperAdapter")),
	}

	if featureFlags.datasync {
		// We pass our encryption/transport handling to the datasync
		// so it's correctly encrypted.
		d.Init(adapter.encryptAndSend)
	}

	return adapter
}

func (a *whisperAdapter) JoinPublic(chatID string) error {
	return a.transport.JoinPublic(chatID)
}

func (a *whisperAdapter) LeavePublic(chatID string) error {
	return a.transport.LeavePublic(chatID)
}

func (a *whisperAdapter) JoinPrivate(publicKey *ecdsa.PublicKey) error {
	return a.transport.JoinPrivate(publicKey)
}

func (a *whisperAdapter) LeavePrivate(publicKey *ecdsa.PublicKey) error {
	return a.transport.LeavePrivate(publicKey)
}

type ChatMessages struct {
	Messages []*protocol.Message
	Public   bool
	ChatID   string
}

func (a *whisperAdapter) RetrieveAllMessages() ([]ChatMessages, error) {
	chatMessages, err := a.transport.RetrieveAllMessages()
	if err != nil {
		return nil, err
	}

	var result []ChatMessages
	for _, messages := range chatMessages {
		protoMessages, err := a.handleRetrievedMessages(messages.Messages)
		if err != nil {
			return nil, err
		}

		result = append(result, ChatMessages{
			Messages: protoMessages,
			Public:   messages.Public,
			ChatID:   messages.ChatID,
		})
	}
	return result, nil
}

// RetrievePublicMessages retrieves the collected public messages.
// It implies joining a chat if it has not been joined yet.
func (a *whisperAdapter) RetrievePublicMessages(chatID string) ([]*protocol.Message, error) {
	messages, err := a.transport.RetrievePublicMessages(chatID)
	if err != nil {
		return nil, err
	}

	return a.handleRetrievedMessages(messages)
}

// RetrievePrivateMessages retrieves the collected private messages.
// It implies joining a chat if it has not been joined yet.
func (a *whisperAdapter) RetrievePrivateMessages(publicKey *ecdsa.PublicKey) ([]*protocol.Message, error) {
	messages, err := a.transport.RetrievePrivateMessages(publicKey)
	if err != nil {
		return nil, err
	}

	return a.handleRetrievedMessages(messages)
}

func (a *whisperAdapter) handleRetrievedMessages(messages []*whisper.ReceivedMessage) ([]*protocol.Message, error) {
	logger := a.logger.With(zap.String("site", "handleRetrievedMessages"))

	decodedMessages := make([]*protocol.Message, 0, len(messages))
	for _, item := range messages {
		shhMessage := whisper.ToWhisperMessage(item)

		hlogger := logger.With(zap.Binary("hash", shhMessage.Hash))
		hlogger.Debug("handling a received message")

		statusMessages, err := a.handleMessages(shhMessage, true)
		if err != nil {
			hlogger.Info("failed to decode messages", zap.Error(err))
			continue
		}

		for _, statusMessage := range statusMessages {
			switch m := statusMessage.ParsedMessage.(type) {
			case protocol.Message:
				m.ID = statusMessage.ID
				m.SigPubKey = statusMessage.SigPubKey()
				decodedMessages = append(decodedMessages, &m)
			case protocol.PairMessage:
				fromOurDevice := isPubKeyEqual(statusMessage.SigPubKey(), &a.privateKey.PublicKey)
				if !fromOurDevice {
					hlogger.Debug("received PairMessage from not our device, skipping")
					break
				}

				metadata := &multidevice.InstallationMetadata{
					Name:       m.Name,
					FCMToken:   m.FCMToken,
					DeviceType: m.DeviceType,
				}
				err := a.protocol.SetInstallationMetadata(&a.privateKey.PublicKey, m.InstallationID, metadata)
				if err != nil {
					return nil, err
				}
			default:
				hlogger.Error("skipped a public message of unsupported type")
			}
		}
	}
	return decodedMessages, nil
}

// DEPRECATED
func (a *whisperAdapter) RetrieveRawAll() (map[filter.Chat][]*protocol.StatusMessage, error) {
	chatWithMessages, err := a.transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}

	logger := a.logger.With(zap.String("site", "RetrieveRawAll"))
	result := make(map[filter.Chat][]*protocol.StatusMessage)

	for chat, messages := range chatWithMessages {
		for _, message := range messages {
			shhMessage := whisper.ToWhisperMessage(message)
			statusMessages, err := a.handleMessages(shhMessage, false)
			if err != nil {
				logger.Info("failed to decode messages", zap.Error(err))
				continue
			}

			result[chat] = append(result[chat], statusMessages...)

		}
	}

	return result, nil
}

// handleMessages expects a whisper message as input, and it will go through
// a series of transformations until the message is parsed into an application
// layer message, or in case of Raw methods, the processing stops at the layer
// before
func (a *whisperAdapter) handleMessages(shhMessage *whisper.Message, applicationLayer bool) ([]*protocol.StatusMessage, error) {
	logger := a.logger.With(zap.String("site", "handleMessages"))
	hlogger := logger.With(zap.Binary("hash", shhMessage.Hash))
	var statusMessage protocol.StatusMessage

	err := statusMessage.HandleTransport(shhMessage)
	if err != nil {
		hlogger.Error("failed to handle transport layer message", zap.Error(err))
		return nil, err
	}

	err = a.handleEncryptionLayer(context.Background(), &statusMessage)
	if err != nil {
		hlogger.Debug("failed to handle an encryption message", zap.Error(err))
	}

	statusMessages, err := statusMessage.HandleDatasync(a.datasync)
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
				hlogger.Error("failed to handle application layer message")
			}
		}
	}

	return statusMessages, nil
}

func (a *whisperAdapter) handleEncryptionLayer(ctx context.Context, message *protocol.StatusMessage) error {
	publicKey := message.SigPubKey()

	logger := a.logger.With(zap.String("site", "handleEncryptionLayer"))

	err := message.HandleEncryption(a.privateKey, publicKey, a.protocol)
	if err == encryption.ErrDeviceNotFound {
		handleErr := a.handleErrDeviceNotFound(ctx, publicKey)
		if handleErr != nil {
			logger.Error("failed to handle error", zap.Error(err), zap.NamedError("handleErr", handleErr))
		}
	}
	if err != nil {
		return errors.Wrap(err, "failed to process an encrypted message")
	}

	return nil
}

func (a *whisperAdapter) handleErrDeviceNotFound(ctx context.Context, publicKey *ecdsa.PublicKey) error {
	now := time.Now().Unix()
	advertise, err := a.protocol.ShouldAdvertiseBundle(publicKey, now)
	if err != nil {
		return err
	}
	if !advertise {
		return nil
	}

	messageSpec, err := a.protocol.BuildBundleAdvertiseMessage(a.privateKey, publicKey)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_, err = a.sendMessageSpec(ctx, publicKey, messageSpec)
	if err != nil {
		return err
	}

	a.protocol.ConfirmBundleAdvertisement(publicKey, now)

	return nil
}

// SendPublic sends a public message passing chat name to the transport layer.
//
// Be aware that this method returns a message ID using protocol.MessageID
// instead of Whisper message hash.
func (a *whisperAdapter) SendPublic(ctx context.Context, chatName, chatID string, data []byte, clock int64) ([]byte, error) {
	logger := a.logger.With(zap.String("site", "SendPublic"))

	logger.Debug("sending a public message", zap.String("chat-name", chatName))

	message := protocol.CreatePublicTextMessage(data, clock, chatName)

	encodedMessage, err := a.encodeMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode message")
	}

	wrappedMessage, err := a.tryWrapMessageV1(encodedMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message")
	}

	messageSpec, err := a.protocol.BuildPublicMessage(a.privateKey, wrappedMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build public message")
	}

	newMessage, err := a.messageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, err
	}

	_, err = a.transport.SendPublic(ctx, newMessage, chatName)
	if err != nil {
		return nil, err
	}

	return protocol.MessageID(&a.privateKey.PublicKey, wrappedMessage), nil
}

// SendPublicRaw takes encoded data, encrypts it and sends through the wire.
// DEPRECATED
func (a *whisperAdapter) SendPublicRaw(ctx context.Context, chatName string, data []byte) ([]byte, whisper.NewMessage, error) {

	var newMessage whisper.NewMessage

	wrappedMessage, err := a.tryWrapMessageV1(data)
	if err != nil {
		return nil, newMessage, errors.Wrap(err, "failed to wrap message")
	}

	newMessage = whisper.NewMessage{
		TTL:       whisperTTL,
		Payload:   wrappedMessage,
		PowTarget: whisperPoW,
		PowTime:   whisperPoWTime,
	}

	hash, err := a.transport.SendPublic(ctx, newMessage, chatName)
	return hash, newMessage, err
}

func (a *whisperAdapter) SendContactCode(ctx context.Context, messageSpec *encryption.ProtocolMessageSpec) ([]byte, error) {
	newMessage, err := a.messageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, err
	}

	return a.transport.SendPublic(ctx, newMessage, filter.ContactCodeTopic(&a.privateKey.PublicKey))
}

func (a *whisperAdapter) tryWrapMessageV1(encodedMessage []byte) ([]byte, error) {
	if a.featureFlags.sendV1Messages {
		wrappedMessage, err := protocol.WrapMessageV1(encodedMessage, a.privateKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap message")
		}

		return wrappedMessage, nil

	}

	return encodedMessage, nil
}

func (a *whisperAdapter) encodeMessage(message protocol.Message) ([]byte, error) {
	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode message")
	}
	return encodedMessage, nil
}

// SendPrivate sends a one-to-one message. It needs to return it
// because the registered Whisper filter handles only incoming messages
// and our own messages need to be handled manually.
//
// This might be not true if a shared secret is used because it relies on
// symmetric encryption.
//
// Be aware that this method returns a message ID using protocol.MessageID
// instead of Whisper message hash.
func (a *whisperAdapter) SendPrivate(
	ctx context.Context,
	publicKey *ecdsa.PublicKey,
	chatID string,
	data []byte,
	clock int64,
) ([]byte, *protocol.Message, error) {
	logger := a.logger.With(zap.String("site", "SendPrivate"))

	logger.Debug("sending a private message", zap.Binary("public-key", crypto.FromECDSAPub(publicKey)))

	message := protocol.CreatePrivateTextMessage(data, clock, chatID)

	encodedMessage, err := a.encodeMessage(message)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to encode message")
	}

	wrappedMessage, err := a.tryWrapMessageV1(encodedMessage)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to wrap message")
	}

	if a.featureFlags.datasync {
		if err := a.sendWithDataSync(publicKey, wrappedMessage); err != nil {
			return nil, nil, errors.Wrap(err, "failed to send message with datasync")
		}
	} else {
		err = a.encryptAndSend(ctx, publicKey, wrappedMessage)
		if err != nil {
			return nil, nil, err
		}
	}

	return protocol.MessageID(&a.privateKey.PublicKey, wrappedMessage), &message, nil
}

func (a *whisperAdapter) sendWithDataSync(publicKey *ecdsa.PublicKey, message []byte) error {
	groupID := datasync.ToOneToOneGroupID(&a.privateKey.PublicKey, publicKey)
	peerID := datasyncpeer.PublicKeyToPeerID(*publicKey)
	exist, err := a.datasync.IsPeerInGroup(groupID, peerID)
	if err != nil {
		return errors.Wrap(err, "failed to check if peer is in group")
	}
	if !exist {
		if err := a.datasync.AddPeer(groupID, peerID); err != nil {
			return errors.Wrap(err, "failed to add peer")
		}
	}
	_, err = a.datasync.AppendMessage(groupID, message)
	if err != nil {
		return errors.Wrap(err, "failed to append message to datasync")
	}

	return nil
}

// SendPrivateRaw takes encoded data, encrypts it and sends through the wire.
// DEPRECATED
func (a *whisperAdapter) SendPrivateRaw(
	ctx context.Context,
	publicKey *ecdsa.PublicKey,
	data []byte,
) ([]byte, whisper.NewMessage, error) {
	a.logger.Debug(
		"sending a private message",
		zap.Binary("public-key", crypto.FromECDSAPub(publicKey)),
		zap.String("site", "SendPrivateRaw"),
	)

	var newMessage whisper.NewMessage

	wrappedMessage, err := a.tryWrapMessageV1(data)
	if err != nil {
		return nil, newMessage, errors.Wrap(err, "failed to wrap message")
	}

	messageSpec, err := a.protocol.BuildDirectMessage(a.privateKey, publicKey, wrappedMessage)
	if err != nil {
		return nil, newMessage, errors.Wrap(err, "failed to encrypt message")
	}

	newMessage, err = a.messageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, newMessage, errors.Wrap(err, "failed to convert ProtocolMessageSpec to whisper.NewMessage")
	}

	if a.featureFlags.datasync {
		if err := a.sendWithDataSync(publicKey, wrappedMessage); err != nil {
			return nil, newMessage, errors.Wrap(err, "failed to send message with datasync")
		}
		return nil, newMessage, err
	}

	hash, err := a.sendMessageSpec(ctx, publicKey, messageSpec)
	return hash, newMessage, err
}

func (a *whisperAdapter) sendMessageSpec(ctx context.Context, publicKey *ecdsa.PublicKey, messageSpec *encryption.ProtocolMessageSpec) ([]byte, error) {
	newMessage, err := a.messageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, err
	}

	logger := a.logger.With(zap.String("site", "sendMessageSpec"))
	switch {
	case messageSpec.SharedSecret != nil:
		logger.Debug("sending using shared secret")
		return a.transport.SendPrivateWithSharedSecret(ctx, newMessage, publicKey, messageSpec.SharedSecret)
	case messageSpec.PartitionedTopicMode() == encryption.PartitionTopicV1:
		logger.Debug("sending partitioned topic")
		return a.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
	case !a.featureFlags.genericDiscoveryTopicEnabled:
		logger.Debug("sending partitioned topic (generic discovery topic disabled)")
		return a.transport.SendPrivateWithPartitioned(ctx, newMessage, publicKey)
	default:
		logger.Debug("sending using discovery topic")
		return a.transport.SendPrivateOnDiscovery(ctx, newMessage, publicKey)
	}
}

func (a *whisperAdapter) encryptAndSend(ctx context.Context, publicKey *ecdsa.PublicKey, encodedMessage []byte) error {
	messageSpec, err := a.protocol.BuildDirectMessage(a.privateKey, publicKey, encodedMessage)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt message")
	}
	_, err = a.sendMessageSpec(ctx, publicKey, messageSpec)
	if err != nil {
		return err
	}
	return nil
}

func (a *whisperAdapter) messageSpecToWhisper(spec *encryption.ProtocolMessageSpec) (whisper.NewMessage, error) {
	var newMessage whisper.NewMessage

	payload, err := proto.Marshal(spec.Message)
	if err != nil {
		return newMessage, err
	}

	newMessage = whisper.NewMessage{
		TTL:       whisperTTL,
		Payload:   payload,
		PowTarget: whisperPoW,
		PowTime:   whisperPoWTime,
	}
	return newMessage, nil
}

func (a *whisperAdapter) handleSharedSecrets(secrets []*sharedsecret.Secret) error {
	logger := a.logger.With(zap.String("site", "handleSharedSecrets"))
	for _, secret := range secrets {
		logger.Debug("received shared secret", zap.Binary("identity", crypto.FromECDSAPub(secret.Identity)))

		fSecret := filter.NegotiatedSecret{
			PublicKey: secret.Identity,
			Key:       secret.Key,
		}
		if err := a.transport.ProcessNegotiatedSecret(fSecret); err != nil {
			return err
		}
	}
	return nil
}

// isPubKeyEqual checks that two public keys are equal
func isPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	// the curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
