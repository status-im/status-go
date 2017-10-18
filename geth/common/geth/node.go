package geth

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/node"
)

// Node interface for geth.Node.
type Node interface {
	Register(constructor node.ServiceConstructor) error
	Start() error
	Stop() error
	Wait()
	Restart() error
	Attach() (*rpc.Client, error)
	Server() *p2p.Server
	Service(service interface{}) error
	DataDir() string
	InstanceDir() string
	AccountManager() *accounts.Manager
	IPCEndpoint() string
	HTTPEndpoint() string
	WSEndpoint() string
	EventMux() *event.TypeMux
	OpenDatabase(name string, cache, handles int) (ethdb.Database, error)
	ResolvePath(x string) string
}
