package node

import (
	"sync"

	enode "github.com/ethereum/go-ethereum/node"
	"github.com/status-im/status-go/geth/common/geth"
	"github.com/status-im/status-go/geth/params"
	erpc "github.com/status-im/status-go/geth/rpc"
)

type rpc struct {
	geth.RPCClient // reference to RPC client
	*sync.RWMutex
}

type rpcAccess interface {
	Init(node *enode.Node, upstream params.UpstreamRPCConfig) error
	Client() geth.RPCClient
}

func (r *rpc) Init(node *enode.Node, upstream params.UpstreamRPCConfig) error {
	var err error

	r.Lock()
	r.RPCClient, err = erpc.NewClient(node, upstream)
	r.Unlock()

	return err
}

func (r *rpc) Client() geth.RPCClient {
	r.Lock()
	defer r.Unlock()

	return r.RPCClient
}

func newRPC() *rpc {
	m := &sync.RWMutex{}
	return &rpc{RWMutex: m}
}
