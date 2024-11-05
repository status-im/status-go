package wallettypes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
)

func TestSendTxArgsValidity(t *testing.T) {
	// 1. If only data fields is set, valid and return data

	bytes1 := types.HexBytes([]byte{0xAA, 0xBB, 0xCC, 0xDD})
	bytes2 := types.HexBytes([]byte{0x00, 0x01, 0x02})

	bytesEmpty := types.HexBytes([]byte{})

	doSendTxValidityTest(t, SendTxArgs{}, true, nil)
	doSendTxValidityTest(t, SendTxArgs{Input: bytes1}, true, bytes1)
	doSendTxValidityTest(t, SendTxArgs{Data: bytes1}, true, bytes1)
	doSendTxValidityTest(t, SendTxArgs{Input: bytes1, Data: bytes1}, true, bytes1)
	doSendTxValidityTest(t, SendTxArgs{Input: bytes1, Data: bytes2}, false, nil)
	doSendTxValidityTest(t, SendTxArgs{Input: bytes1, Data: bytesEmpty}, true, bytes1)
	doSendTxValidityTest(t, SendTxArgs{Input: bytesEmpty, Data: bytes1}, true, bytes1)
	doSendTxValidityTest(t, SendTxArgs{Input: bytesEmpty, Data: bytesEmpty}, true, bytesEmpty)
}

func doSendTxValidityTest(t *testing.T, args SendTxArgs, expectValid bool, expectValue types.HexBytes) {
	assert.Equal(t, expectValid, args.Valid(), "Valid() returned unexpected value")
	if expectValid {
		assert.Equal(t, expectValue, args.GetInput(), "GetInput() returned unexpected value")
	}
}
