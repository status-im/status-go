package node

import (
	"github.com/status-im/status-go/geth/common/geth"
	"sync"
)

type rpc struct {
	geth.RPCClient // reference to RPC client
	*sync.RWMutex
}

func newRPC() *rpc {
	m := &sync.RWMutex{}
	return &rpc{RWMutex: m}
}
