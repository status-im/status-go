package geth

import (
	"context"

	"github.com/status-im/status-go/geth/rpc"
)

type RPCClient interface {
	Call(result interface{}, method string, args ...interface{}) error
	CallRaw(body string) string
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	RegisterHandler(method string, handler rpc.Handler)
}
