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

	expected := `{"jsonrpc":"2.0","id":0,"result":"3434=done"}`
	require.Equal(t, expected, got)

	res = []byte(`{"field": "value"}`)
	got = newSuccessResponse(res)

	expected = `{"jsonrpc":"2.0","id":0,"result":{"field":"value"}}`
	require.Equal(t, expected, got)
}

func TestNewErrorResponse(t *testing.T) {
	got := newErrorResponse(-32601, errors.New("Method not found"))

	expected := `{"jsonrpc":"2.0","id":0,"error":{"code":-32601,"message":"Method not found"}}`
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

func TestMethodAndParamsFromBody(t *testing.T) {
	body := `{"jsonrpc": "2.0", "method": "subtract", "params": [{"subtrahend": 23, "minuend": 42}]}`
	paramsExpect := []interface{}{
		map[string]interface{}{
			"subtrahend": float64(23),
			"minuend":    float64(42),
		},
	}

	method, params, err := methodAndParamsFromBody(body)
	require.NoError(t, err)
	require.Equal(t, "subtract", method)
	require.Equal(t, paramsExpect, params)

	body = `{"jsonrpc": "2.0", "method": "test", "params": []}`
	method, params, err = methodAndParamsFromBody(body)
	require.NoError(t, err)
	require.Equal(t, "test", method)
	require.Equal(t, []interface{}{}, params)

	body = `{"jsonrpc": "2.0", "method": "test"}`
	method, params, err = methodAndParamsFromBody(body)
	require.NoError(t, err)
	require.Equal(t, "test", method)
	require.Equal(t, []interface{}{}, params)
}
