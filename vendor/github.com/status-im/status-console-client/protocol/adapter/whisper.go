// Collection of common Whisper entities that can be used by any adapter.

package adapter

import (
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-console-client/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"
	"golang.org/x/crypto/sha3"
)

// MailServerPassword is a password that is required
// to request messages from a Status mail server.
const MailServerPassword = "status-offline-inbox"

// Whisper message properties.
const (
	WhisperTTL     = 15
	WhisperPoW     = 0.002
	WhisperPoWTime = 5
)

// Whisper known topics.
const (
	TopicDiscovery = "contact-discovery"
)

type keysManager interface {
	PrivateKey() *ecdsa.PrivateKey
	AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error)
	AddOrGetSymKeyFromPassword(password string) (string, error)
	GetRawSymKey(string) ([]byte, error)
}

type filter struct {
	*whisper.Filter
	keys keysManager
}

func newFilter(keys keysManager) *filter {
	return &filter{
		Filter: &whisper.Filter{
			PoW:      0,
			AllowP2P: true,
			Messages: whisper.NewMemoryMessageStore(),
		},
		keys: keys,
	}
}

func (f *filter) updateForPublicGroup(name string) error {
	topic, err := ToTopic(name)
	if err != nil {
		return err
	}
	f.Topics = append(f.Topics, topic[:])

	symKeyID, err := f.keys.AddOrGetSymKeyFromPassword(name)
	if err != nil {
		return err
	}
	symKey, err := f.keys.GetRawSymKey(symKeyID)
	if err != nil {
		return err
	}
	f.KeySym = symKey

	return nil
}

func (f *filter) updateForPrivate(name string, recipient *ecdsa.PublicKey) error {
	topic, err := ToTopic(name)
	if err != nil {
		return err
	}
	f.Topics = append(f.Topics, topic[:])

	f.KeyAsym = f.keys.PrivateKey()

	return nil
}

func updateFilterFromSubscribeOptions(f *filter, options protocol.SubscribeOptions) error {
	if options.Recipient != nil && options.ChatName != "" {
		return f.updateForPrivate(options.ChatName, options.Recipient)
	} else if options.ChatName != "" {
		return f.updateForPublicGroup(options.ChatName)
	} else {
		return errors.New("unrecognized options")
	}
}

type NewMessage struct {
	whisper.NewMessage
	keys keysManager
}

func NewNewMessage(keys keysManager, data []byte) (*NewMessage, error) {
	sigKey, err := keys.AddOrGetKeyPair(keys.PrivateKey())
	if err != nil {
		return nil, err
	}

	return &NewMessage{
		NewMessage: whisper.NewMessage{
			TTL:       WhisperTTL,
			Payload:   data,
			PowTarget: WhisperPoW,
			PowTime:   WhisperPoWTime,
			Sig:       sigKey,
		},
		keys: keys,
	}, nil
}

func (m *NewMessage) updateForPrivate(name string, recipient *ecdsa.PublicKey) (err error) {
	m.Topic, err = ToTopic(name)
	if err != nil {
		return
	}

	m.PublicKey = crypto.FromECDSAPub(recipient)

	return
}

func (m *NewMessage) updateForPublicGroup(name string) (err error) {
	m.Topic, err = ToTopic(name)
	if err != nil {
		return
	}

	m.SymKeyID, err = m.keys.AddOrGetSymKeyFromPassword(name)
	return
}

func updateNewMessageFromSendOptions(m *NewMessage, options protocol.SendOptions) error {
	if options.Recipient != nil && options.ChatName != "" {
		return m.updateForPrivate(options.ChatName, options.Recipient)
	} else if options.ChatName != "" {
		return m.updateForPublicGroup(options.ChatName)
	} else {
		return errors.New("unrecognized options")
	}
}

func ToTopic(name string) (whisper.TopicType, error) {
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write([]byte(name)); err != nil {
		return whisper.TopicType{}, err
	}
	return whisper.BytesToTopic(hash.Sum(nil)), nil
}
