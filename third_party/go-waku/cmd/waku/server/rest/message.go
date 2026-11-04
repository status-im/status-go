package rest

import (
	"errors"

	"github.com/waku-org/go-waku/cmd/waku/server"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

type RestWakuMessage struct {
	Payload      server.Base64URLByte `json:"payload"`
	ContentTopic string               `json:"contentTopic"`
	Version      *uint32              `json:"version,omitempty"`
	Timestamp    *int64               `json:"timestamp,omitempty"`
	Meta         []byte               `json:"meta,omitempty"`
	Ephemeral    *bool                `json:"ephemeral"`
}

func (r *RestWakuMessage) FromProto(input *pb.WakuMessage) error {
	if err := input.Validate(); err != nil {
		return err
	}

	r.Payload = input.Payload
	r.ContentTopic = input.ContentTopic
	r.Timestamp = input.Timestamp
	r.Version = input.Version
	r.Meta = input.Meta
	r.Ephemeral = input.Ephemeral

	return nil
}

func (r *RestWakuMessage) ToProto() (*pb.WakuMessage, error) {
	if r == nil {
		return nil, errors.New("wakumessage is missing")
	}

	msg := &pb.WakuMessage{
		Payload:      r.Payload,
		ContentTopic: r.ContentTopic,
		Version:      r.Version,
		Timestamp:    r.Timestamp,
		Meta:         r.Meta,
		Ephemeral:    r.Ephemeral,
	}

	return msg, nil
}
