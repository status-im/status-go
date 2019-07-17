package statusproto

import (
	"context"
	"crypto/ecdsa"
	"log"
	"time"

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
	transport  transport.WhisperTransport
	protocol   *encryption.Protocol
}

func newWhisperAdapter(pk *ecdsa.PrivateKey, t transport.WhisperTransport, p *encryption.Protocol) *whisperAdapter {
	return &whisperAdapter{
		privateKey: pk,
		transport:  t,
		protocol:   p,
	}
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

// RetrievePublicMessages retrieves the collected public messages.
// It implies joining a chat if it has not been joined yet.
func (a *whisperAdapter) RetrievePublicMessages(chatID string) ([]*protocol.Message, error) {
	messages, err := a.transport.RetrievePublicMessages(chatID)
	if err != nil {
		return nil, err
	}

	decodedMessages := make([]*protocol.Message, 0, len(messages))
	for _, item := range messages {
		shhMessage := whisper.ToWhisperMessage(item)

		log.Printf("[whisperAdapter::RetrievePublicMessages] received a public message: %#x", shhMessage.Hash)

		statusMessage, err := a.decodeMessage(shhMessage)
		if err != nil {
			log.Printf("failed to decode message %#x", shhMessage.Hash)
			continue
		}

		switch m := statusMessage.Message.(type) {
		case protocol.Message:
			m.ID = statusMessage.ID
			m.SigPubKey = statusMessage.SigPubKey
			decodedMessages = append(decodedMessages, &m)
		default:
			log.Printf("skipped a public message of unsupported type")
		}
	}
	return decodedMessages, nil
}

// RetrievePrivateMessages retrieves the collected private messages.
// It implies joining a chat if it has not been joined yet.
func (a *whisperAdapter) RetrievePrivateMessages(publicKey *ecdsa.PublicKey) ([]*protocol.Message, error) {
	messages, err := a.transport.RetrievePrivateMessages(publicKey)
	if err != nil {
		return nil, err
	}

	decodedMessages := make([]*protocol.Message, 0, len(messages))
	for _, item := range messages {
		shhMessage := whisper.ToWhisperMessage(item)

		log.Printf("[whisperAdapter::RetrievePrivateMessages] received a private message: %#x", shhMessage.Hash)

		err := a.decryptMessage(context.Background(), shhMessage)
		if err != nil {
			log.Printf("[whisperAdapter::RetrievePrivateMessages] failed to decrypt a message %#x: %v", shhMessage.Hash, err)
		}

		statusMessage, err := a.decodeMessage(shhMessage)
		if err != nil {
			log.Printf("[whisperAdapter::RetrievePrivateMessages] failed to decode a message %#x: %v", shhMessage.Hash, err)
			continue
		}

		switch m := statusMessage.Message.(type) {
		case protocol.Message:
			m.ID = statusMessage.ID
			m.SigPubKey = statusMessage.SigPubKey
			decodedMessages = append(decodedMessages, &m)
		case protocol.PairMessage:
			fromOurDevice := isPubKeyEqual(statusMessage.SigPubKey, &a.privateKey.PublicKey)
			if !fromOurDevice {
				log.Printf("received PairMessage from not our device, skipping")
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
		}
	}
	return decodedMessages, nil
}

func (a *whisperAdapter) RetrieveAllRaw() (map[filter.Chat][]*whisper.Message, error) {
	chatWithMessages, err := a.transport.RetrieveAllRaw()
	if err != nil {
		return nil, err
	}

	result := make(map[filter.Chat][]*whisper.Message)

	for chat, messages := range chatWithMessages {
		for _, message := range messages {
			shhMessage := whisper.ToWhisperMessage(message)
			err := a.decryptMessage(context.Background(), shhMessage)
			if err != nil {
				log.Printf("[whisperAdapter::RetrievePrivateMessages] failed to decrypt a message %#x: %v", shhMessage.Hash, err)
			}
			result[chat] = append(result[chat], shhMessage)
		}
	}

	return result, nil
}

func (a *whisperAdapter) decodeMessage(message *whisper.Message) (*protocol.StatusMessage, error) {
	publicKey, err := crypto.UnmarshalPubkey(message.Sig)
	if err != nil {
		return nil, err
	}

	decoded, err := protocol.DecodeMessage(message.Payload)
	if err != nil {
		return nil, err
	}
	decoded.ID = message.Hash
	decoded.SigPubKey = publicKey

	return &decoded, nil
}

func (a *whisperAdapter) decryptMessage(ctx context.Context, message *whisper.Message) error {
	publicKey, err := crypto.UnmarshalPubkey(message.Sig)
	if err != nil {
		return errors.Wrap(err, "failed to get signature")
	}

	var protocolMessage encryption.ProtocolMessage

	err = proto.Unmarshal(message.Payload, &protocolMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProtocolMessage")
	}

	payload, err := a.protocol.HandleMessage(
		a.privateKey,
		publicKey,
		&protocolMessage,
		message.Hash,
	)
	if err == encryption.ErrDeviceNotFound {
		err := a.handleErrDeviceNotFound(ctx, publicKey)
		if err != nil {
			log.Printf("failed to handle ErrDeviceNotFound: %v", err)
		}
	}
	if err != nil {
		return errors.Wrap(err, "failed to process an encrypted message")
	}

	message.Payload = payload
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

func (a *whisperAdapter) SendPublic(ctx context.Context, chatName, chatID string, data []byte, clock int64) ([]byte, error) {
	log.Printf("[whisperAdapter::SendPublic] sending a public message to %s", chatName)

	message := protocol.CreatePublicTextMessage(data, clock, chatName)

	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode message")
	}

	newMessage := whisper.NewMessage{
		TTL:       whisperTTL,
		Payload:   encodedMessage,
		PowTarget: whisperPoW,
		PowTime:   whisperPoWTime,
	}

	return a.transport.SendPublic(ctx, newMessage, chatName)
}

func (a *whisperAdapter) SendContactCode(ctx context.Context, messageSpec *encryption.ProtocolMessageSpec) ([]byte, error) {
	newMessage, err := a.messageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, err
	}

	return a.transport.SendPublic(ctx, *newMessage, filter.ContactCodeTopic(&a.privateKey.PublicKey))
}

// SendPrivate sends a one-to-one message. It needs to return it
// because the registered Whisper filter handles only incoming messages.
func (a *whisperAdapter) SendPrivate(
	ctx context.Context,
	publicKey *ecdsa.PublicKey,
	chatID string,
	data []byte,
	clock int64,
) ([]byte, *protocol.Message, error) {
	log.Printf("[whisperAdapter::SendPrivate] sending a private mesage to %#x", crypto.FromECDSAPub(publicKey))

	message := protocol.CreatePrivateTextMessage(data, clock, chatID)

	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to encode message")
	}

	messageSpec, err := a.protocol.BuildDirectMessage(a.privateKey, publicKey, encodedMessage)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to encrypt message")
	}

	hash, err := a.sendMessageSpec(ctx, publicKey, messageSpec)
	if err != nil {
		return nil, nil, err
	}
	return hash, &message, nil
}

func (a *whisperAdapter) sendMessageSpec(ctx context.Context, publicKey *ecdsa.PublicKey, messageSpec *encryption.ProtocolMessageSpec) ([]byte, error) {
	newMessage, err := a.messageSpecToWhisper(messageSpec)
	if err != nil {
		return nil, err
	}

	if messageSpec.SharedSecret != nil {
		log.Printf("[whisperAdapter::sendMessageSpec] sending using shared secret")
		return a.transport.SendPrivateWithSharedSecret(ctx, *newMessage, publicKey, messageSpec.SharedSecret)
	} else if messageSpec.PartitionedTopicMode() == encryption.PartitionTopicV1 {
		log.Printf("[whisperAdapter::sendMessageSpec] sending partitioned topic")
		return a.transport.SendPrivateWithPartitioned(ctx, *newMessage, publicKey)
	} else {
		log.Printf("[whisperAdapter::sendMessageSpec] sending using discovery topic")
		return a.transport.SendPrivateOnDiscovery(ctx, *newMessage, publicKey)
	}
}

func (a *whisperAdapter) messageSpecToWhisper(spec *encryption.ProtocolMessageSpec) (*whisper.NewMessage, error) {
	payload, err := proto.Marshal(spec.Message)
	if err != nil {
		return nil, err
	}

	newMessage := whisper.NewMessage{
		TTL:       whisperTTL,
		Payload:   payload,
		PowTarget: whisperPoW,
		PowTime:   whisperPoWTime,
	}

	return &newMessage, nil
}

func (a *whisperAdapter) handleSharedSecrets(secrets []*sharedsecret.Secret) error {
	for _, secret := range secrets {
		log.Printf(
			"[whisperAdapter::handleSharedSecrets] received shared secret with identity %#x",
			crypto.FromECDSAPub(secret.Identity),
		)

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
