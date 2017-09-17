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
	cases := []struct {
		name       string
		body       string
		params     []interface{}
		method     string
		id         json.RawMessage
		shouldFail bool
	}{
		{
			"params_array",
			`{"jsonrpc": "2.0", "id": 42, "method": "subtract", "params": [{"subtrahend": 23, "minuend": 42}]}`,
			[]interface{}{
				map[string]interface{}{
					"subtrahend": float64(23),
					"minuend":    float64(42),
				},
			},
			"subtract",
			json.RawMessage(`42`),
			false,
		},
		{
			"params_empty_array",
			`{"jsonrpc": "2.0", "method": "test", "params": []}`,
			[]interface{}{},
			"test",
			nil,
			false,
		},
		{
			"params_none",
			`{"jsonrpc": "2.0", "method": "test"}`,
			[]interface{}{},
			"test",
			nil,
			false,
		},
		{
			"getFilterMessage",
			`{"jsonrpc":"2.0","id":44,"method":"shh_getFilterMessages","params":["3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e"]}`,
			[]interface{}{string("3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e")},
			"shh_getFilterMessages",
			json.RawMessage(`44`),
			false,
		},
		{
			"getFilterMessage_array",
			`[{"jsonrpc":"2.0","id":44,"method":"shh_getFilterMessages","params":["3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e"]}]`,
			[]interface{}{string("3de6a8867aeb75be74d68478b853b4b0e063704d30f8231c45d0fcbd97af207e")},
			"shh_getFilterMessages",
			json.RawMessage(`44`),
			false,
		},
		{
			"empty_array",
			`[]`,
			[]interface{}{},
			"",
			nil,
			true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			method, params, id, err := methodAndParamsFromBody(test.body)
			if test.shouldFail {
				require.Error(t, err)
			}
			require.NoError(t, err)
			require.Equal(t, test.method, method)
			require.Equal(t, test.params, params)
			require.EqualValues(t, test.id, id)
		})
	}
}
