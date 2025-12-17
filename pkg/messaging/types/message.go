package types

import (
	"crypto/ecdsa"
	"encoding/json"

	"github.com/jinzhu/copier"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
)

// TransportLayer is the lowest layer and represents waku message.
type TransportLayer struct {
	// Payload as received from the transport layer
	Payload   []byte           `json:"-"`
	Hash      []byte           `json:"-"`
	SigPubKey *ecdsa.PublicKey `json:"-"`
	Dst       *ecdsa.PublicKey `json:"-"`
	Message   *ReceivedMessage `json:"message"`
}

type SegmentationLayer struct {
	Segmented bool     `json:"segmented"`
	Completed bool     `json:"completed"`
	Hashes    [][]byte `json:"hashes"`
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

// Message encapsulates layers of the protocol
type Message struct {
	TransportLayer    TransportLayer    `json:"transportLayer"`
	SegmentationLayer SegmentationLayer `json:"segmentationLayer"`
	EncryptionLayer   EncryptionLayer   `json:"encryptionLayer"`
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
		ID:        MessageID(m.SigPubKey(), m.EncryptionLayer.Payload),
		Payload:   string(m.EncryptionLayer.Payload),
		Timestamp: m.TransportLayer.Message.Timestamp,
		From:      m.TransportLayer.Message.Sig,
	}
	return json.Marshal(item)
}

// SigPubKey returns the most important signature, from the application layer to transport
func (m *Message) SigPubKey() *ecdsa.PublicKey {
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
