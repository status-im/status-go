package types

import (
	"context"

	"github.com/status-im/status-go/eth-node/types"
)

// NewMessage represents a new whisper message that is posted through the RPC.
type NewMessage struct {
	SymKeyID    string    `json:"symKeyID"`
	PublicKey   []byte    `json:"pubKey"`
	SigID       string    `json:"sig"`
	PubsubTopic string    `json:"pubsubTopic"`
	Topic       TopicType `json:"topic"`
	Payload     []byte    `json:"payload"`
	Ephemeral   bool      `json:"ephemeral"`
	Priority    *int      `json:"priority"`
}

// Message is the RPC representation of a whisper message.
type Message struct {
	Sig          []byte    `json:"sig,omitempty"`
	Timestamp    uint32    `json:"timestamp"`
	Topic        TopicType `json:"topic"`
	Payload      []byte    `json:"payload"`
	Padding      []byte    `json:"padding"`
	Hash         []byte    `json:"hash"`
	Dst          []byte    `json:"recipientPublicKey,omitempty"`
	ThirdPartyID string    `json:"thirdPartyId,omitempty"`
}

// Criteria holds various filter options for inbound messages.
type Criteria struct {
	SymKeyID     string      `json:"symKeyID"`
	PrivateKeyID string      `json:"privateKeyID"`
	Sig          []byte      `json:"sig"`
	MinPow       float64     `json:"minPow"`
	PubsubTopic  string      `json:"pubsubTopic"`
	Topics       []TopicType `json:"topics"`
	AllowP2P     bool        `json:"allowP2P"`
}

// PublicWakuAPI provides the waku RPC service that can be
// use publicly without security implications.
type PublicWakuAPI interface {
	// AddPrivateKey imports the given private key.
	AddPrivateKey(ctx context.Context, privateKey types.HexBytes) (string, error)
	// GenerateSymKeyFromPassword derives a key from the given password, stores it, and returns its ID.
	GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error)
	// DeleteKeyPair removes the key with the given key if it exists.
	DeleteKeyPair(ctx context.Context, key string) (bool, error)

	// Post posts a message on the Whisper network.
	// returns the hash of the message in case of success.
	Post(ctx context.Context, req NewMessage) ([]byte, error)

	// NewMessageFilter creates a new filter that can be used to poll for
	// (new) messages that satisfy the given criteria.
	NewMessageFilter(req Criteria) (string, error)
	// GetFilterMessages returns the messages that match the filter criteria and
	// are received between the last poll and now.
	GetFilterMessages(id string) ([]*Message, error)
}
