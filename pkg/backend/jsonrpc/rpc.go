package jsonrpc

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
