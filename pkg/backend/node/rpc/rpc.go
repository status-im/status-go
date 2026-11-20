package rpc

import (
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
	"time"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type nopConn struct{}

func (d *nopConn) Close() error {
	return nil
}

func (d *nopConn) SetWriteDeadline(time.Time) error {
	return nil
}

type SingleRequestCodec struct {
	input     string
	output    string
	done      chan struct{} // closed when encode runs
	processed atomic.Bool
}

func NewSingleRequestCodec(inputJSON string) *SingleRequestCodec {
	return &SingleRequestCodec{
		input: inputJSON,
		done:  make(chan struct{}),
	}
}

func (c *SingleRequestCodec) GethCodec() gethrpc.ServerCodec {
	return gethrpc.NewFuncCodec(&nopConn{}, c.encode, c.decode)
}

func (c *SingleRequestCodec) Output() string {
	return c.output
}

func (c *SingleRequestCodec) encode(v interface{}, isErrorResponse bool) error {
	// isErrorResponse seem to be safe to ignore.
	// It was added in go-ethereum to handle HTTP differently in some specific cases
	// https://github.com/ethereum/go-ethereum/pull/25457

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.output = string(data)
	close(c.done)
	return nil
}

func (c *SingleRequestCodec) decode(v interface{}) error {
	if !c.processed.CompareAndSwap(false, true) {
		<-c.done
		return io.EOF
	}

	output, ok := v.(*json.RawMessage)
	if !ok {
		return errors.New("expected json.RawMessage codec")
	}

	*output = json.RawMessage(c.input)
	return nil
}

// ErrorResponse returns a JSON-RPC error response to the single request in inputJSON,
// for requests that cannot reach a server.
func ErrorResponse(inputJSON string, err error) string {
	var request struct {
		ID json.RawMessage `json:"id"`
	}
	if json.Unmarshal([]byte(inputJSON), &request) != nil || len(request.ID) == 0 {
		request.ID = json.RawMessage("null")
	}

	response := struct {
		Version string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   jsonError       `json:"error"`
	}{
		Version: "2.0",
		ID:      request.ID,
		Error:   jsonError{Code: errcodeDefault, Message: err.Error()},
	}
	data, _ := json.Marshal(response)
	return string(data)
}

const errcodeDefault = -32000

type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
