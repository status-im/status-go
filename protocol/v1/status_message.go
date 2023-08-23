package protocol

import (
	"crypto/ecdsa"
	"encoding/json"

	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
	"github.com/status-im/status-go/protocol/protobuf"
)

type StatusMessageT int

// StatusMessage is any Status Protocol message.
type StatusMessage struct {
	// TransportMessage is the parsed message received from the transport layer, i.e the input
	TransportMessage *types.Message `json:"transportMessage"`
	// Type is the type of application message contained
	Type protobuf.ApplicationMetadataMessage_Type `json:"-"`

	// TransportPayload is the payload as received from the transport layer
	TransportPayload []byte `json:"-"`
	// DecryptedPayload is the payload after having been processed by the encryption layer
	DecryptedPayload []byte `json:"decryptedPayload"`
	// UnwrappedPayload is the payload after having been unwrapped from the applicaition metadata layer
	UnwrappedPayload []byte `json:"unwrappedPayload"`

	// ID is the canonical ID of the message
	ID types.HexBytes `json:"id"`
	// Hash is the transport layer hash
	Hash []byte `json:"-"`

	// Dst is the targeted public key
	Dst *ecdsa.PublicKey

	// TransportLayerSigPubKey contains the public key provided by the transport layer
	TransportLayerSigPubKey *ecdsa.PublicKey `json:"-"`
	// ApplicationMetadataLayerPubKey contains the public key provided by the application metadata layer
	ApplicationMetadataLayerSigPubKey *ecdsa.PublicKey `json:"-"`

	// Installations is the new installations returned by the encryption layer
	Installations []*multidevice.Installation
	// SharedSecret is the shared secret returned by the encryption layer
	SharedSecrets []*sharedsecret.Secret

	// HashRatchetInfo is the information about a new hash ratchet group/key pair
	HashRatchetInfo []*encryption.HashRatchetInfo
}

// Temporary JSON marshaling for those messages that are not yet processed
// by the go code
func (m *StatusMessage) MarshalJSON() ([]byte, error) {
	item := struct {
		ID        types.HexBytes `json:"id"`
		Payload   string         `json:"payload"`
		From      types.HexBytes `json:"from"`
		Timestamp uint32         `json:"timestamp"`
	}{
		ID:        m.ID,
		Payload:   string(m.UnwrappedPayload),
		Timestamp: m.TransportMessage.Timestamp,
		From:      m.TransportMessage.Sig,
	}
	return json.Marshal(item)
}

// SigPubKey returns the most important signature, from the application layer to transport
func (m *StatusMessage) SigPubKey() *ecdsa.PublicKey {
	if m.ApplicationMetadataLayerSigPubKey != nil {
		return m.ApplicationMetadataLayerSigPubKey
	}

	return m.TransportLayerSigPubKey
}

func (m *StatusMessage) Clone() (*StatusMessage, error) {
	copy := &StatusMessage{}

	err := copier.Copy(&copy, m)
	return copy, err
}

func (m *StatusMessage) HandleTransport(shhMessage *types.Message) error {
	publicKey, err := crypto.UnmarshalPubkey(shhMessage.Sig)
	if err != nil {
		return errors.Wrap(err, "failed to get signature")
	}

	m.TransportMessage = shhMessage
	m.Hash = shhMessage.Hash
	m.TransportLayerSigPubKey = publicKey
	m.TransportPayload = shhMessage.Payload

	if shhMessage.Dst != nil {
		publicKey, err := crypto.UnmarshalPubkey(shhMessage.Dst)
		if err != nil {
			return err
		}
		m.Dst = publicKey
	}

	return nil
}

func (m *StatusMessage) HandleEncryption(myKey *ecdsa.PrivateKey, senderKey *ecdsa.PublicKey, enc *encryption.Protocol, skipNegotiation bool) error {
	// As we handle non-encrypted messages, we make sure that DecryptPayload
	// is set regardless of whether this step is successful
	m.DecryptedPayload = m.TransportPayload
	// Nothing to do
	if skipNegotiation {
		return nil
	}

	var protocolMessage encryption.ProtocolMessage
	err := proto.Unmarshal(m.TransportPayload, &protocolMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProtocolMessage")
	}

	response, err := enc.HandleMessage(
		myKey,
		senderKey,
		&protocolMessage,
		m.Hash,
	)

	if err == encryption.ErrHashRatchetGroupIDNotFound {

		if response != nil {
			m.HashRatchetInfo = response.HashRatchetInfo
		}
		return err
	}

	if err != nil {
		return errors.Wrap(err, "failed to handle Encryption message")
	}

	m.DecryptedPayload = response.DecryptedMessage
	m.Installations = response.Installations
	m.SharedSecrets = response.SharedSecrets
	m.HashRatchetInfo = response.HashRatchetInfo
	return nil
}

func (m *StatusMessage) HandleApplicationMetadata() error {
	message, err := protobuf.Unmarshal(m.DecryptedPayload)
	if err != nil {
		return err
	}

	recoveredKey, err := message.RecoverKey()
	if err != nil {
		return err
	}
	m.ApplicationMetadataLayerSigPubKey = recoveredKey
	// Calculate ID using the wrapped record
	m.ID = MessageID(recoveredKey, m.DecryptedPayload)
	log.Debug("calculated ID for envelope", "envelopeHash", hexutil.Encode(m.Hash), "messageId", hexutil.Encode(m.ID))

	m.UnwrappedPayload = message.Payload
	m.Type = message.Type
	return nil

}
