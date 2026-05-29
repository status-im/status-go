package types

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

