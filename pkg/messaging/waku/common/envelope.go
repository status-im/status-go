package common

import (
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

// Envelope contains information about the pubsub topic of a WakuMessage
// and a hash used to identify a message based on the bytes of a WakuMessage
// protobuffer
type Envelope interface {
	Message() *pb.WakuMessage
	PubsubTopic() string
	Hash() pb.MessageHash
}

type WakuEnvelope struct {
	msg   *pb.WakuMessage
	topic string
	hash  pb.MessageHash
}

func NewWakuEnvelope(msg *pb.WakuMessage, topic string, hash pb.MessageHash) *WakuEnvelope {
	return &WakuEnvelope{
		msg:   msg,
		topic: topic,
		hash:  hash,
	}
}

func (e *WakuEnvelope) Message() *pb.WakuMessage {
	return e.msg
}

func (e *WakuEnvelope) PubsubTopic() string {
	return e.topic
}

func (e *WakuEnvelope) Hash() pb.MessageHash {
	return e.hash
}
