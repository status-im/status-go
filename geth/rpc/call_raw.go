package rpc

import (
	"context"
	"encoding/json"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth/log"
)

const (
	jsonrpcVersion        = "2.0"
	errInvalidMessageCode = -32700 // from go-ethereum/rpc/errors.go
)

// for JSON-RPC responses obtained via CallRaw(), we have no way
// to know ID field from actual response. web3.js (primary and
// only user of CallRaw()) will validate response by checking
// ID field for being a number:
// https://github.com/ethereum/web3.js/blob/develop/lib/web3/jsonrpc.js#L66
// thus, we will use zero ID as a workaround of this limitation
var defaultMsgID = json.RawMessage(`0`)

// CallRaw performs a JSON-RPC call with already crafted JSON-RPC body. It
// returns string in JSON format with response (successul or error).
func (c *Client) CallRaw(body string) string {
	ctx := context.Background()
	return c.callRawContext(ctx, body)
}

// jsonrpcMessage represents JSON-RPC request, notification, successful response or
// error response.
type jsonrpcMessage struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// jsonError represents Error message for JSON-RPC responses.
type jsonError struct {
	Code    int         `json:"code,omitempty"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// callRawContext performs a JSON-RPC call with already crafted JSON-RPC body and
// given context. It returns string in JSON format with response (successul or error).
//
// TODO(divan): this function exists for compatibility and uses default
// go-ethereum's RPC client under the hood. It adds some unnecessary overhead
// by first marshalling JSON string into object to use with normal Call,
// which is then umarshalled back to the same JSON. The same goes with response.
// This is waste of CPU and memory and should be avoided if possible,
// either by changing exported API (provide only Call, not CallRaw) or
// refactoring go-ethereum's client to allow using raw JSON directly.
func (c *Client) callRawContext(ctx context.Context, body string) string {
	// unmarshal JSON body into json-rpc request
	method, params, id, err := methodAndParamsFromBody(body)
	if err != nil {
		return newErrorResponse(errInvalidMessageCode, err, id)
	}

	// route and execute
	var result json.RawMessage
	err = c.CallContext(ctx, &result, method, params...)

	// as we have to return original JSON, we have to
	// analyze returned error and reconstruct original
	// JSON error response.
	if err != nil && err != gethrpc.ErrNoResult {
		if er, ok := err.(gethrpc.Error); ok {
			return newErrorResponse(er.ErrorCode(), err, id)
		}

		return newErrorResponse(errInvalidMessageCode, err, id)
	}

	// finally, marshal answer
	return newSuccessResponse(result, id)
}

// methodAndParamsFromBody extracts Method and Params of
// JSON-RPC body into values ready to use with ethereum-go's
// RPC client Call() function. A lot of empty interface usage is
// due to the underlying code design :/
func methodAndParamsFromBody(body string) (string, []interface{}, json.RawMessage, error) {
	msg, err := unmarshalMessage(body)
	if err != nil {
		return "", nil, nil, err
	}

	params := []interface{}{}
	if msg.Params != nil {
		err = json.Unmarshal(msg.Params, &params)
		if err != nil {
			log.Error("unmarshal params", "error", err)
			return "", nil, nil, err
		}
	}

	id := msg.ID
	if id == nil {
		id = defaultMsgID
	}

	return msg.Method, params, id, nil
}

func unmarshalMessage(body string) (*jsonrpcMessage, error) {
	var msg jsonrpcMessage
	err := json.Unmarshal([]byte(body), &msg)
	return &msg, err
}

func newSuccessResponse(result json.RawMessage, id json.RawMessage) string {
	msg := &jsonrpcMessage{
		ID:      id,
		Version: jsonrpcVersion,
		Result:  result,
	}
	data, _ := json.Marshal(msg)
	return string(data)
}

func newErrorResponse(code int, err error, id json.RawMessage) string {
	errMsg := &jsonrpcMessage{
		Version: jsonrpcVersion,
		ID:      id,
		Error: &jsonError{
			Code:    code,
			Message: err.Error(),
		},
	}

	data, _ := json.Marshal(errMsg)
	return string(data)
}
