package common

import (
	"encoding/json"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

// Envelope contains information about the pubsub topic of a WakuMessage
// and a hash used to identify a message based on the bytes of a WakuMessage
// protobuffer
type Envelope struct {
	msg   *pb.WakuMessage
	topic string
	hash  MessageHash
}

type wakuMessage struct {
	Payload        []byte  `json:"payload,omitempty"`
	ContentTopic   string  `json:"contentTopic,omitempty"`
	Version        *uint32 `json:"version,omitempty"`
	Timestamp      *int64  `json:"timestamp,omitempty"`
	Meta           []byte  `json:"meta,omitempty"`
	Ephemeral      *bool   `json:"ephemeral,omitempty"`
	RateLimitProof []byte  `json:"proof,omitempty"`
}

type wakuEnvelope struct {
	WakuMessage wakuMessage `json:"wakuMessage"`
	PubsubTopic string      `json:"pubsubTopic"`
	MessageHash MessageHash `json:"messageHash"`
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (e *Envelope) UnmarshalJSON(input []byte) error {
	wakuEnvelope := wakuEnvelope{}
	err := json.Unmarshal(input, &wakuEnvelope)
	if err != nil {
		return err
	}

	e.msg = &pb.WakuMessage{
		Payload:        wakuEnvelope.WakuMessage.Payload,
		ContentTopic:   wakuEnvelope.WakuMessage.ContentTopic,
		Version:        wakuEnvelope.WakuMessage.Version,
		Timestamp:      wakuEnvelope.WakuMessage.Timestamp,
		Meta:           wakuEnvelope.WakuMessage.Meta,
		Ephemeral:      wakuEnvelope.WakuMessage.Ephemeral,
		RateLimitProof: wakuEnvelope.WakuMessage.RateLimitProof,
	}
	e.topic = wakuEnvelope.PubsubTopic
	e.hash = wakuEnvelope.MessageHash

	return nil
}

func (e *Envelope) Message() *pb.WakuMessage {
	return e.msg
}

func (e *Envelope) PubsubTopic() string {
	return e.topic
}

func (e *Envelope) Hash() MessageHash {
	return e.hash
}
