package protocol

import (
	"crypto/ecdsa"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/status-im/status-go/protocol/applicationmetadata"
	"github.com/status-im/status-go/protocol/datasync"
	"github.com/status-im/status-go/protocol/encryption"
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
)

type StatusMessageT int

const (
	MessageT StatusMessageT = iota + 1
	MembershipUpdateMessageT
	PairMessageT
)

// StatusMessage is any Status Protocol message.
type StatusMessage struct {
	// TransportMessage is the parsed message received from the transport layer, i.e the input
	TransportMessage *whispertypes.Message
	// MessageType is the type of application message contained
	MessageType StatusMessageT
	// ParsedMessage is the parsed message by the application layer, i.e the output
	ParsedMessage interface{}

	// TransportPayload is the payload as received from the transport layer
	TransportPayload []byte
	// DecryptedPayload is the payload after having been processed by the encryption layer
	DecryptedPayload []byte

	// ID is the canonical ID of the message
	ID protocol.HexBytes
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

func (m *StatusMessage) HandleTransport(shhMessage *whispertypes.Message) error {
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

// HandleDatasync processes StatusMessage through data sync layer.
// This is optional and DataSync might be nil. In such a case,
// only one payload will be returned equal to DecryptedPayload.
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
	// Calculate ID using the wrapped record
	m.ID = MessageID(recoveredKey, m.DecryptedPayload)
	m.DecryptedPayload = message.Payload
	return nil

}

func (m *StatusMessage) HandleApplication() error {
	value, err := decodeTransitMessage(m.DecryptedPayload)
	if err != nil {
		log.Printf("[message::DecodeMessage] could not decode message: %#x, err: %v", m.Hash, err.Error())
		return err
	}
	m.ParsedMessage = value
	switch m.ParsedMessage.(type) {
	case Message:
		m.MessageType = MessageT
	case MembershipUpdateMessage:
		m.MessageType = MembershipUpdateMessageT
	case PairMessage:
		m.MessageType = PairMessageT
		// By default we null the parsed message field, as
		// otherwise is populated with the raw transit and we are
		// unable to marshal in case it contains maps
		// as they have type map[interface{}]interface{}
	default:
		m.ParsedMessage = nil

	}
	return nil
}
