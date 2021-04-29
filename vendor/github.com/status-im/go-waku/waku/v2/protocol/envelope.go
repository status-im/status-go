package protocol

import "github.com/status-im/go-waku/waku/v2/protocol/pb"

type Envelope struct {
	msg         *pb.WakuMessage
	pubsubTopic string
	size        int
	hash        []byte
}

func NewEnvelope(msg *pb.WakuMessage, pubSubTopic string) *Envelope {
	data, _ := msg.Marshal()
	return &Envelope{
		msg:         msg,
		pubsubTopic: pubSubTopic,
		size:        len(data),
		hash:        pb.Hash(data),
	}
}

func (e *Envelope) Message() *pb.WakuMessage {
	return e.msg
}

func (e *Envelope) PubsubTopic() string {
	return e.pubsubTopic
}

func (e *Envelope) Hash() []byte {
	return e.hash
}

func (e *Envelope) Size() int {
	return e.size
}
