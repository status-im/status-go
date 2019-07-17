package whisper

import (
	whisper "github.com/status-im/whisper/whisperv6"
)

type RequestOptions struct {
	Topics   []whisper.TopicType
	Password string
	Limit    int
	From     int64 // in seconds
	To       int64 // in seconds
}

const (
	defaultPowTime = 1
)

func DefaultWhisperMessage() whisper.NewMessage {
	msg := whisper.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.002
	msg.PowTime = defaultPowTime

	return msg
}
