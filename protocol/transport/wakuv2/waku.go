package wakuv2

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
