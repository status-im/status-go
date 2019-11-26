package waku

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeVersionedResponse(t *testing.T) {
	response := NewMessagesResponse(common.Hash{1}, []EnvelopeError{{Code: 1}})
	bytes, err := rlp.EncodeToBytes(response)
	require.NoError(t, err)

	var mresponse MultiVersionResponse
	require.NoError(t, rlp.DecodeBytes(bytes, &mresponse))
	v1resp, err := mresponse.DecodeResponse1()
	require.NoError(t, err)
	require.Equal(t, response.Response.Hash, v1resp.Hash)
}
