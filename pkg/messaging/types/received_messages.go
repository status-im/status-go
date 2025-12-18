package types

// ReceivedMessage represents a message received from the transport filter,
// roughly corresponding to a Waku message.
type ReceivedMessage struct {
	Sig          []byte       `json:"sig,omitempty"`
	Timestamp    uint32       `json:"timestamp"`
	Topic        ContentTopic `json:"topic"`
	Payload      []byte       `json:"payload"`
	Padding      []byte       `json:"padding"`
	Hash         []byte       `json:"hash"`
	Dst          []byte       `json:"recipientPublicKey,omitempty"`
	ThirdPartyID string       `json:"thirdPartyId,omitempty"`
}

type ReceivedMessages struct {
	Filter     ChatFilter
	SHHMessage *ReceivedMessage
	Messages   []*Message
}
