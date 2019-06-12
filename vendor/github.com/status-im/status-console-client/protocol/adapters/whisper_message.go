package adapters

import (
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-console-client/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"
)

type newMessage struct {
	whisper.NewMessage
	keys keysManager
}

func newNewMessage(keys keysManager, data []byte) (*newMessage, error) {
	sigKey, err := keys.AddOrGetKeyPair(keys.PrivateKey())
	if err != nil {
		return nil, err
	}

	return &newMessage{
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

func (m *newMessage) ToWhisper() whisper.NewMessage {
	return m.NewMessage
}

func (m *newMessage) updateForPrivate(name string, recipient *ecdsa.PublicKey) (err error) {
	m.Topic, err = ToTopic(name)
	if err != nil {
		return
	}

	m.PublicKey = crypto.FromECDSAPub(recipient)

	return
}

func (m *newMessage) updateForPublicGroup(name string) (err error) {
	m.Topic, err = ToTopic(name)
	if err != nil {
		return
	}

	m.SymKeyID, err = m.keys.AddOrGetSymKeyFromPassword(name)
	return
}

func updateNewMessageFromSendOptions(m *newMessage, options protocol.SendOptions) error {
	if options.Recipient != nil && options.ChatName != "" {
		return m.updateForPrivate(options.ChatName, options.Recipient)
	} else if options.ChatName != "" {
		return m.updateForPublicGroup(options.ChatName)
	} else {
		return errors.New("unrecognized options")
	}
}
