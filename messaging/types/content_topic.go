package types

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/types"
)

const (
	topicLength = 4
)

type ContentTopic [topicLength]byte

func BytesToContentTopic(b []byte) (t ContentTopic) {
	sz := topicLength
	if x := len(b); x < topicLength {
		sz = x
	}
	for i := 0; i < sz; i++ {
		t[i] = b[i]
	}
	return t
}

func StringToContentTopic(s string) (t ContentTopic) {
	str, _ := hexutil.Decode(s)
	return BytesToContentTopic(str)
}

func (t ContentTopic) Bytes() []byte {
	return t[:topicLength]
}

func (t ContentTopic) String() string {
	return types.EncodeHex(t[:])
}
