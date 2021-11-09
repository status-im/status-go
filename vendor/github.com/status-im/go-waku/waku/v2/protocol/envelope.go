package protocol

import "github.com/status-im/go-waku/waku/v2/protocol/pb"

// Envelope contains information about the pubsub topic of a WakuMessage
// and a hash used to identify a message based on the bytes of a WakuMessage
// protobuffer
type Envelope struct {
	msg         *pb.WakuMessage
	pubsubTopic string
	size        int
	hash        []byte
}

// NewEnvelope creates a new Envelope that contains a WakuMessage
// It's used as a way to know to which Pubsub topic belongs a WakuMessage
// as well as generating a hash based on the bytes that compose the message
func NewEnvelope(msg *pb.WakuMessage, pubSubTopic string) *Envelope {
	data, _ := msg.Marshal()
	return &Envelope{
		msg:         msg,
		pubsubTopic: pubSubTopic,
		size:        len(data),
		hash:        pb.Hash(data),
	}
}

// Message returns the WakuMessage associated to an Envelope
func (e *Envelope) Message() *pb.WakuMessage {
	return e.msg
}

// PubsubTopic returns the topic on which a WakuMessage was received
func (e *Envelope) PubsubTopic() string {
	return e.pubsubTopic
}

// Hash returns a 32 byte hash calculated from the WakuMessage bytes
func (e *Envelope) Hash() []byte {
	return e.hash
}

// Size returns the byte size of the WakuMessage
func (e *Envelope) Size() int {
	return e.size
}
