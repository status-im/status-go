package common

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
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

type nwakuMessage struct {
	Payload        []byte  `json:"payload,omitempty"`
	ContentTopic   string  `json:"contentTopic,omitempty"`
	Version        *uint32 `json:"version,omitempty"`
	Timestamp      *int64  `json:"timestamp,omitempty"`
	Meta           []byte  `json:"meta,omitempty"`
	Ephemeral      *bool   `json:"ephemeral,omitempty"`
	RateLimitProof []byte  `json:"proof,omitempty"`
}

type nwakuEnvelope struct {
	WakuMessage nwakuMessage `json:"wakuMessage"`
	PubsubTopic string       `json:"pubsubTopic"`
	MessageHash string       `json:"messageHash"`
}

func NewWakuEnvelope(msg *pb.WakuMessage, topic string, hash pb.MessageHash) *WakuEnvelope {
	return &WakuEnvelope{
		msg:   msg,
		topic: topic,
		hash:  hash,
	}

}

// NewEnvelope creates a new Envelope from a json string generated in nwaku
func (e *WakuEnvelope) UnmarshalJSON(input []byte) error {
	nwakuEnvelope := nwakuEnvelope{}
	err := json.Unmarshal(input, &nwakuEnvelope)
	if err != nil {
		return err
	}

	hash, err := hexutil.Decode(nwakuEnvelope.MessageHash)
	if err != nil {
		return err
	}

	// Modify the receiver instead of returning a new instance
	e.msg = &pb.WakuMessage{
		Payload:        nwakuEnvelope.WakuMessage.Payload,
		ContentTopic:   nwakuEnvelope.WakuMessage.ContentTopic,
		Version:        nwakuEnvelope.WakuMessage.Version,
		Timestamp:      nwakuEnvelope.WakuMessage.Timestamp,
		Meta:           nwakuEnvelope.WakuMessage.Meta,
		Ephemeral:      nwakuEnvelope.WakuMessage.Ephemeral,
		RateLimitProof: nwakuEnvelope.WakuMessage.RateLimitProof,
	}
	e.topic = nwakuEnvelope.PubsubTopic
	e.hash = pb.ToMessageHash(hash)

	return nil
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
