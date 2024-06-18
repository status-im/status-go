package node

import (
	"reflect"
	"unicode"

	"github.com/ethereum/go-ethereum/p2p"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/common"
)

// firstCharToLower converts to first character of name to lowercase.
func firstCharToLower(name string) string {
	ret := []rune(name)
	if len(ret) > 0 {
		ret[0] = unicode.ToLower(ret[0])
	}
	return string(ret)
}

// addSuitableCallbacks iterates over the methods of the given type and adds them to
// the methods list
// This is taken from go-ethereum services
func addSuitableCallbacks(receiver reflect.Value, namespace string, methods map[string]bool) {
	typ := receiver.Type()
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if method.PkgPath != "" {
			continue // method not exported
		}
		name := firstCharToLower(method.Name)
		methods[namespace+"_"+name] = true
	}
}

func NewRPCAPI(node *StatusNode) common.StatusService {
	return &RPCAPI{
		node: node,
	}

}
func NewRPCService(node *StatusNode) *RPCService {
	return &RPCService{
		node: node,
	}
}

type RPCService struct {
	node *StatusNode
}

func (s *RPCAPI) Protocols() []p2p.Protocol {
	return nil
}

func (s *RPCAPI) Start() error {
	return nil
}

func (s *RPCAPI) Stop() error {
	return nil
}

func (s *RPCAPI) APIs() []gethrpc.API {
	return []gethrpc.API{
		{
			Namespace: "statustest",
			Version:   "0.1.0",
			Service:   NewRPCService(s.node),
		},
	}
}

type RPCAPI struct {
	node *StatusNode
}

func (r *RPCService) CallRPC(inputJSON string) (string, error) {
	return r.node.CallRPC(inputJSON)

}
