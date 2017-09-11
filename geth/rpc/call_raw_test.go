package rpc

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSuccessResponse(t *testing.T) {
	res := []byte(`"3434=done"`)
	got := newSuccessResponse(res)

	expected := `{"jsonrpc":"2.0","result":"3434=done"}`
	require.Equal(t, expected, got)

	res = []byte(`{"field": "value"}`)
	got = newSuccessResponse(res)

	expected = `{"jsonrpc":"2.0","result":{"field":"value"}}`
	require.Equal(t, expected, got)
}

func TestNewErrorResponse(t *testing.T) {
	got := newErrorResponse(-32601, errors.New("Method not found"))

	expected := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	require.Equal(t, expected, got)
}

func TestUnmarshalMessage(t *testing.T) {
	body := `{"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend": 23, "minuend": 42}}`
	got, err := unmarshalMessage(body)
	require.NoError(t, err)

	expected := &jsonrpcMessage{
		Version: "2.0",
		Method:  "subtract",
		Params:  json.RawMessage(`{"subtrahend": 23, "minuend": 42}`),
	}
	require.Equal(t, expected, got)
}
