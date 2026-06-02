package types

import (
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/crypto/types"
)

const (
	// TopicLength is the expected length of the topic, in bytes
	TopicLength = 4
)

// TopicType represents a cryptographically secure, probabilistic partial
// classifications of a message, determined as the first (left) 4 bytes of the
// SHA3 hash of some arbitrary data given by the original author of the message.
type TopicType [TopicLength]byte

// BytesToTopic converts from the byte array representation of a topic
// into the TopicType type.
func BytesToTopic(b []byte) (t TopicType) {
	sz := TopicLength
	if x := len(b); x < TopicLength {
		sz = x
	}
	for i := 0; i < sz; i++ {
		t[i] = b[i]
	}
	return t
}

// String converts a topic byte array to a string representation.
func (t TopicType) String() string {
	return types.EncodeHex(t[:])
}

func (t TopicType) Bytes() []byte {
	return TopicTypeToByteArray(t)
}

func TopicTypeToByteArray(t TopicType) []byte {
	return t[:4]
}

// ContentTopic returns the 23/WAKU2-TOPICS string representation of the topic.
//
// TODO(status-im/status-go#7509): this duplicates common.TopicType.ContentTopic
// because TopicType itself is duplicated between waku/types and waku/common.
// Remove once the two TopicType definitions are unified.
func (t TopicType) ContentTopic() string {
	return "/waku/1/" + hexutil.Encode(t[:]) + "/rfc26"
}
