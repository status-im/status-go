package pb

import (
	"github.com/waku-org/go-waku/waku/v2/hash"
)

// Hash calculates the hash of a waku message
func (msg *WakuMessage) Hash(pubsubTopic string) []byte {
	return hash.SHA256([]byte(pubsubTopic), msg.Payload, []byte(msg.ContentTopic), msg.Meta)
}
