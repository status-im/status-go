package statusproto

import (
	"crypto/ecdsa"
	"github.com/pkg/errors"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/copier"
	"github.com/status-im/status-protocol-go/applicationmetadata"
	"github.com/status-im/status-protocol-go/datasync"
	"github.com/status-im/status-protocol-go/encryption"
	whisper "github.com/status-im/whisper/whisperv6"
)

// StatusMessage is any Status Protocol message.
type StatusMessage struct {
	// TransportMessage is the parsed message received from the trasport layer, i.e the input
	TransportMessage *whisper.Message
	// ParsedMessage is the parsed message by the application layer, i.e the output
	ParsedMessage interface{}

	// TransportPayload is the payload as received from the transport layer
	TransportPayload []byte
	// DecryptedPayload is the payload after having been processed by the encryption layer
	DecryptedPayload []byte

	// ID is the canonical ID of the message
	ID []byte
	// Hash is the transport layer hash
	Hash []byte

	// TransportLayerSigPubKey contains the public key provided by the transport layer
	TransportLayerSigPubKey *ecdsa.PublicKey
	// ApplicationMetadataLayerPubKey contains the public key provided by the application metadata layer
	ApplicationMetadataLayerSigPubKey *ecdsa.PublicKey
}

// SigPubKey returns the most important signature, from the application layer to transport
func (s *StatusMessage) SigPubKey() *ecdsa.PublicKey {
	if s.ApplicationMetadataLayerSigPubKey != nil {
		return s.ApplicationMetadataLayerSigPubKey
	}

	return s.TransportLayerSigPubKey
}

func (s *StatusMessage) Clone() (*StatusMessage, error) {
	copy := &StatusMessage{}

	err := copier.Copy(&copy, s)
	return copy, err
}

func (m *StatusMessage) HandleTransport(shhMessage *whisper.Message) error {
	publicKey, err := crypto.UnmarshalPubkey(shhMessage.Sig)
	if err != nil {
		return errors.Wrap(err, "failed to get signature")
	}

	m.TransportMessage = shhMessage
	m.Hash = shhMessage.Hash
	m.TransportLayerSigPubKey = publicKey
	m.TransportPayload = shhMessage.Payload

	return nil
}

func (m *StatusMessage) HandleEncryption(myKey *ecdsa.PrivateKey, senderKey *ecdsa.PublicKey, enc *encryption.Protocol) error {
	// As we handle non-encrypted messages, we make sure that DecryptPayload
	// is set regardless of whether this step is successful
	m.DecryptedPayload = m.TransportPayload

	var protocolMessage encryption.ProtocolMessage
	err := proto.Unmarshal(m.TransportPayload, &protocolMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProtocolMessage")
	}

	payload, err := enc.HandleMessage(
		myKey,
		senderKey,
		&protocolMessage,
		m.Hash,
	)

	if err != nil {
		return errors.Wrap(err, "failed to handle Encryption message")
	}

	m.DecryptedPayload = payload
	return nil
}

func (m *StatusMessage) HandleDatasync(datasync *datasync.DataSync) ([]*StatusMessage, error) {
	var statusMessages []*StatusMessage
	payloads := datasync.Handle(
		m.SigPubKey(),
		m.DecryptedPayload,
	)
	for _, payload := range payloads {
		message, err := m.Clone()
		if err != nil {
			return nil, err
		}
		message.DecryptedPayload = payload
		statusMessages = append(statusMessages, message)
	}
	return statusMessages, nil
}

func (m *StatusMessage) HandleApplicationMetadata() error {
	message, err := applicationmetadata.Unmarshal(m.DecryptedPayload)
	// Not an applicationmetadata message, calculate ID using the previous
	// signature
	if err != nil {
		m.ID = MessageID(m.SigPubKey(), m.DecryptedPayload)
		return nil
	}

	recoveredKey, err := message.RecoverKey()
	if err != nil {
		return err
	}
	m.ApplicationMetadataLayerSigPubKey = recoveredKey
	m.DecryptedPayload = message.Payload
	m.ID = MessageID(m.SigPubKey(), m.DecryptedPayload)
	return nil

}

func (m *StatusMessage) HandleApplication() error {
	value, err := decodeTransitMessage(m.DecryptedPayload)
	if err != nil {
		log.Printf("[message::DecodeMessage] could not decode message: %#x", m.Hash)
		return err
	}
	m.ParsedMessage = value

	return nil
}
