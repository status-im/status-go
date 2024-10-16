package rpc

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSuccessResponse(t *testing.T) {
	cases := []struct {
		name     string
		result   json.RawMessage
		id       json.RawMessage
		expected string
	}{
		{"string", json.RawMessage(`"3434=done"`), nil, `{"jsonrpc":"2.0","id":0,"result":"3434=done"}`},
		{"struct_nil_id", json.RawMessage(`{"field": "value"}`), nil, `{"jsonrpc":"2.0","id":0,"result":{"field":"value"}}`},
		{"struct_non_nil_id", json.RawMessage(`{"field": "value"}`), json.RawMessage(`42`), `{"jsonrpc":"2.0","id":42,"result":{"field":"value"}}`},
		{"null", json.RawMessage(`null`), json.RawMessage(`7`), `{"jsonrpc":"2.0","id":7,"result":null}`},
		{"null_nil", nil, json.RawMessage(`7`), `{"jsonrpc":"2.0","id":7,"result":null}`},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := newSuccessResponse(test.result, test.id)
			require.Equal(t, test.expected, got)
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	got := newErrorResponse(-32601, errors.New("Method not found"), json.RawMessage(`42`))

	expected := `{"jsonrpc":"2.0","id":42,"error":{"code":-32601,"message":"Method not found"}}`
	require.Equal(t, expected, got)
}

func TestUnmarshalMessage(t *testing.T) {
	body := json.RawMessage(`{"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend": 23, "minuend": 42}}`)
	got, err := unmarshalMessage(body)
	require.NoError(t, err)

	expected := &jsonrpcRequest{
		jsonrpcMessage: jsonrpcMessage{Version: "2.0"},
		Method:         "subtract",
		Params:         json.RawMessage(`{"subtrahend": 23, "minuend": 42}`),
	}
	require.Equal(t, expected, got)
}

func TestMethodAndParamsFromBody(t *testing.T) {
	cases := []struct {
		name       string
		body       json.RawMessage
		params     []interface{}
		method     string
		id         json.RawMessage
		chainID    uint64
		shouldFail bool
		timeout    uint64
	}{
		{
			"params_array",
			json.RawMessage(`{"jsonrpc": "2.0", "id": 42, "method": "subtract", "params": [{"subtrahend": 23, "minuend": 42}]}`),
			[]interface{}{
				map[string]interface{}{
					"subtrahend": float64(23),
					"minuend":    float64(42),
				},
			},
			"subtract",
			json.RawMessage(`42`),
			0,
			false,
			0,
		},
		{
			"params_empty_array",
			json.RawMessage(`{"jsonrpc": "2.0", "method": "test", "params": []}`),
			[]interface{}{},
			"test",
			nil,
			0,
			false,
			0,
		},
		{
			"params_none",
			json.RawMessage(`{"jsonrpc": "2.0", "method": "test"}`),
			[]interface{}{},
			"test",
			nil,
			0,
			false,
			0,
		},
		{
			"params_chain_id",
			json.RawMessage(`{"jsonrpc": "2.0", "chainId": 2, "method": "test"}`),
			[]interface{}{},
			"test",
			nil,
			2,
			false,
			0,
		},
		{
			"getFilterMessage",
			json.RawMessage(`{"jsonrpc":"2.0","id":44,"method":"shh_getFilterMessages","params":["3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e"]}`),
			[]interface{}{string("3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e")},
			"shh_getFilterMessages",
			json.RawMessage(`44`),
			0,
			false,
			0,
		},
		{
			"getFilterMessage_array",
			json.RawMessage(`[{"jsonrpc":"2.0","id":44,"method":"shh_getFilterMessages","params":["3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e"]}]`),
			[]interface{}{},
			"",
			nil,
			0,
			true,
			0,
		},
		{
			"empty_array",
			json.RawMessage(`[]`),
			[]interface{}{},
			"",
			nil,
			0,
			true,
			0,
		},
		{
			"timeout",
			json.RawMessage(`{"jsonrpc": "2.0", "timeout": 2000, "method": "test"}`),
			[]interface{}{},
			"test",
			nil,
			0,
			false,
			2000,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			response, err := methodAndParamsFromBody(test.body)
			if test.shouldFail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.timeout, response.Timeout)
			require.Equal(t, test.chainID, response.ChainID)
			require.Equal(t, test.method, response.Method)
			require.Equal(t, test.params, response.Params)
			require.EqualValues(t, test.id, response.ID)
		})
	}
}

func TestIsBatch(t *testing.T) {
	cases := []struct {
		name     string
		body     json.RawMessage
		expected bool
	}{
		{"single", json.RawMessage(`{"jsonrpc":"2.0","id":44,"method":"shh_getFilterMessages","params":["3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e"]}`), false},
		{"array", json.RawMessage(`[{"jsonrpc":"2.0","id":44,"method":"shh_getFilterMessages","params":["3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e"]}]`), true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := isBatch(test.body)
			require.Equal(t, test.expected, got)
		})
	}
}
