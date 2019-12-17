package whisper

import (
	"github.com/status-im/status-go/eth-node/types"
)

type RequestOptions struct {
	Topics   []types.TopicType
	Password string
	Limit    int
	From     int64 // in seconds
	To       int64 // in seconds
}

const (
	defaultPowTime = 1
)

func DefaultWhisperMessage() types.NewMessage {
	msg := types.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.002
	msg.PowTime = defaultPowTime

	return msg
}
