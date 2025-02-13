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

type envelopeImpl struct {
	msg   *pb.WakuMessage
	topic string
	hash  pb.MessageHash
}

type tmpWakuMessageJson struct {
	Payload        []byte  `json:"payload,omitempty"`
	ContentTopic   string  `json:"contentTopic,omitempty"`
	Version        *uint32 `json:"version,omitempty"`
	Timestamp      *int64  `json:"timestamp,omitempty"`
	Meta           []byte  `json:"meta,omitempty"`
	Ephemeral      *bool   `json:"ephemeral,omitempty"`
	RateLimitProof []byte  `json:"proof,omitempty"`
}

type tmpEnvelopeStruct struct {
	WakuMessage tmpWakuMessageJson `json:"wakuMessage"`
	PubsubTopic string             `json:"pubsubTopic"`
	MessageHash string             `json:"messageHash"`
}

// NewEnvelope creates a new Envelope from a json string generated in nwaku
func NewEnvelope(jsonEventStr string) (Envelope, error) {
	tmpEnvelopeStruct := tmpEnvelopeStruct{}
	err := json.Unmarshal([]byte(jsonEventStr), &tmpEnvelopeStruct)
	if err != nil {
		return nil, err
	}

	hash, err := hexutil.Decode(tmpEnvelopeStruct.MessageHash)
	if err != nil {
		return nil, err
	}

	return &envelopeImpl{
		msg: &pb.WakuMessage{
			Payload:        tmpEnvelopeStruct.WakuMessage.Payload,
			ContentTopic:   tmpEnvelopeStruct.WakuMessage.ContentTopic,
			Version:        tmpEnvelopeStruct.WakuMessage.Version,
			Timestamp:      tmpEnvelopeStruct.WakuMessage.Timestamp,
			Meta:           tmpEnvelopeStruct.WakuMessage.Meta,
			Ephemeral:      tmpEnvelopeStruct.WakuMessage.Ephemeral,
			RateLimitProof: tmpEnvelopeStruct.WakuMessage.RateLimitProof,
		},
		topic: tmpEnvelopeStruct.PubsubTopic,
		hash:  pb.ToMessageHash(hash),
	}, nil
}

func (e *envelopeImpl) Message() *pb.WakuMessage {
	return e.msg
}

func (e *envelopeImpl) PubsubTopic() string {
	return e.topic
}

func (e *envelopeImpl) Hash() pb.MessageHash {
	return e.hash
}
