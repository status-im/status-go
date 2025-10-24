package types

import (
	"crypto/ecdsa"
	"encoding/json"

	"github.com/jinzhu/copier"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

// TransportLayer is the lowest layer and represents waku message.
type TransportLayer struct {
	// Payload as received from the transport layer
	Payload   []byte           `json:"-"`
	Hash      []byte           `json:"-"`
	SigPubKey *ecdsa.PublicKey `json:"-"`
	Dst       *ecdsa.PublicKey
	Message   *ReceivedMessage `json:"message"`
}

// EncryptionLayer handles optional encryption.
// It is not mandatory and can be omitted,
// also its presence does not guarantee encryption.
type EncryptionLayer struct {
	// Payload after having been processed by the encryption layer
	Payload         []byte `json:"-"`
	Installations   []*Installation
	HashRatchetInfo []*HashRatchetInfo
}

// ApplicationLayer is the topmost layer and represents the application message.
type ApplicationLayer struct {
	// Payload after having been unwrapped from the application layer
	Payload   []byte                                   `json:"-"`
	ID        types.HexBytes                           `json:"id"`
	SigPubKey *ecdsa.PublicKey                         `json:"-"`
	Type      protobuf.ApplicationMetadataMessage_Type `json:"-"`
}

// Message encapsulates all layers of the protocol
type Message struct {
	TransportLayer   TransportLayer   `json:"transportLayer"`
	EncryptionLayer  EncryptionLayer  `json:"encryptionLayer"`
	ApplicationLayer ApplicationLayer `json:"applicationLayer"`
}

// Temporary JSON marshaling for those messages that are not yet processed
// by the go code
func (m *Message) MarshalJSON() ([]byte, error) {
	item := struct {
		ID        types.HexBytes `json:"id"`
		Payload   string         `json:"payload"`
		From      types.HexBytes `json:"from"`
		Timestamp uint32         `json:"timestamp"`
	}{
		ID:        m.ApplicationLayer.ID,
		Payload:   string(m.ApplicationLayer.Payload),
		Timestamp: m.TransportLayer.Message.Timestamp,
		From:      m.TransportLayer.Message.Sig,
	}
	return json.Marshal(item)
}

// SigPubKey returns the most important signature, from the application layer to transport
func (m *Message) SigPubKey() *ecdsa.PublicKey {
	if m.ApplicationLayer.SigPubKey != nil {
		return m.ApplicationLayer.SigPubKey
	}

	return m.TransportLayer.SigPubKey
}

func (m *Message) Clone() (*Message, error) {
	copy := &Message{}

	err := copier.Copy(&copy, m)
	return copy, err
}

func MessageID(author *ecdsa.PublicKey, data []byte) types.HexBytes {
	keyBytes := crypto.FromECDSAPub(author)
	return types.HexBytes(crypto.Keccak256(append(keyBytes, data...)))
}
