package whisper

import (
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
)

type RequestOptions struct {
	Topics   []whispertypes.TopicType
	Password string
	Limit    int
	From     int64 // in seconds
	To       int64 // in seconds
}

const (
	defaultPowTime = 1
)

func DefaultWhisperMessage() whispertypes.NewMessage {
	msg := whispertypes.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.002
	msg.PowTime = defaultPowTime

	return msg
}
