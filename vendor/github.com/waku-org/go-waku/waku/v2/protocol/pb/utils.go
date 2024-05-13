package pb

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/waku-org/go-waku/waku/v2/hash"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// MessageHash represents an unique identifier for a message within a pubsub topic
type MessageHash [32]byte

func (h MessageHash) String() string {
	return hexutil.Encode(h[:])
}

func (h MessageHash) Bytes() []byte {
	return h[:]
}

// ToMessageHash converts a byte slice into a MessageHash
func ToMessageHash(b []byte) MessageHash {
	var result MessageHash
	copy(result[:], b)
	return result
}

// Hash calculates the hash of a waku message
func (msg *WakuMessage) Hash(pubsubTopic string) MessageHash {
	hash := hash.SHA256([]byte(pubsubTopic), msg.Payload, []byte(msg.ContentTopic), msg.Meta, toBytes(msg.GetTimestamp()))
	return ToMessageHash(hash)
}

func toBytes(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func (msg *WakuMessage) LogFields(pubsubTopic string) []zapcore.Field {
	return []zapcore.Field{
		zap.Stringer("hash", msg.Hash(pubsubTopic)),
		zap.String("pubsubTopic", pubsubTopic),
		zap.String("contentTopic", msg.ContentTopic),
		zap.Int64("timestamp", msg.GetTimestamp()),
	}
}

func (msg *WakuMessage) Logger(logger *zap.Logger, pubsubTopic string) *zap.Logger {
	return logger.With(msg.LogFields(pubsubTopic)...)
}
