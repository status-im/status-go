package whisper

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncMailRequestValidate(t *testing.T) {
	testCases := []struct {
		Name  string
		Req   SyncMailRequest
		Error string
	}{
		{
			Name:  "invalid zero Limit",
			Req:   SyncMailRequest{},
			Error: "invalid 'Limit' value, expected value greater than 0",
		},
		{
			Name:  "invalid large Limit",
			Req:   SyncMailRequest{Limit: 1e6},
			Error: "invalid 'Limit' value, expected value lower than 1000",
		},
		{
			Name:  "invalid Lower",
			Req:   SyncMailRequest{Limit: 10, Lower: 10, Upper: 5},
			Error: "invalid 'Lower' value, can't be greater than 'Upper'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Req.Validate()
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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
