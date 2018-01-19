package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

// use an interface for the go-ethereum CallContext method, for ease of
// switching between upstream and remote, and to facilitate injection of mock
// nodes in route_test.go.
type callContext interface {
	CallContext(ctx context.Context, result interface{}, method string,
		args ...interface{}) error
}

// router implements logic for routing
// JSON-RPC requests either to Upstream or
// Local node.
type router struct {
	upstreamEnabled bool

	local    callContext
	upstream callContext

	handlersMx sync.RWMutex       // mx guards handlers
	handlers   map[string]Handler // locally registered handlers
}

// newRouter inits new router.
func newRouter(local callContext, upstream callContext, upstreamEnabled bool) *router {
	r := &router{
		upstreamEnabled: upstreamEnabled,
		local:           local,
		upstream:        upstream,
		handlers:        make(map[string]Handler),
	}

	return r
}

// callContext performs a JSON-RPC call with the given arguments. If the context is
// canceled before the call has successfully returned, callContext returns immediately.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
//
// It uses custom routing scheme for calls, as determined by route.go.
func (r *router) callContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	route, routed := RoutingTable[method]

	if !routed {
		log.Warn("Unrecognized RPC method " + method + ". Routing to" +
			" local node.")
		return r.local.CallContext(ctx, result, method, args...)
	}

	switch route {
	case localHandler:
		if handler, ok := r.handler(method); ok {
			return r.callHandler(ctx, result, handler, args...)
		}
		return errors.New("No handler registered for method " + method)

	case upstreamNode:
		if r.upstreamEnabled {
			return r.upstream.CallContext(ctx, result, method,
				args...)
		}
		return r.local.CallContext(ctx, result, method, args...)

	case localNode:
		return r.local.CallContext(ctx, result, method, args...)

	default:
		return errors.New("Unknown RPC destination '" + method + "'")
	}
}

// registerHandler registers local handler for specific RPC method.
//
// If method handler is registered, and if it is permitted by
// RoutingTable, then it will be executed with given handler and never
// routed to the upstream or local servers.
func (r *router) registerHandler(method string, handler Handler) {
	r.handlersMx.Lock()
	defer r.handlersMx.Unlock()

	r.handlers[method] = handler
}

// callHandler calls registered RPC handler with given args and pointer to result.
// It handles proper params and result converting
//
// TODO(divan): use cancellation via context here?
func (r *router) callHandler(ctx context.Context, result interface{}, handler Handler, args ...interface{}) error {
	response, err := handler(ctx, args...)
	if err != nil {
		return err
	}

	// if result is nil, just ignore result -
	// the same way as gethrpc.CallContext() caller would expect
	if result == nil {
		return nil
	}

	if err := setResultFromRPCResponse(result, response); err != nil {
		return err
	}

	return nil
}

// handler is a concurrently safe method to get registered handler by name.
func (r *router) handler(method string) (Handler, bool) {
	r.handlersMx.RLock()
	defer r.handlersMx.RUnlock()
	handler, ok := r.handlers[method]
	return handler, ok
}

// setResultFromRPCResponse tries to set result value from response using reflection
// as concrete types are unknown.
func setResultFromRPCResponse(result, response interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid result type: %s", r)
		}
	}()

	responseValue := reflect.ValueOf(response)

	// If it is called via CallRaw, result has type json.RawMessage and
	// we should marshal the response before setting it.
	// Otherwise, it is called with CallContext and result is of concrete type,
	// thus we should try to set it as it is.
	// If response type and result type are incorrect, an error should be returned.
	// TODO(divan): add additional checks for result underlying value, if needed:
	// some example: https://golang.org/src/encoding/json/decode.go#L596
	switch reflect.ValueOf(result).Elem().Type() {
	case reflect.TypeOf(json.RawMessage{}), reflect.TypeOf([]byte{}):
		data, err := json.Marshal(response)
		if err != nil {
			return err
		}

		responseValue = reflect.ValueOf(data)
	}

	value := reflect.ValueOf(result).Elem()
	if !value.CanSet() {
		return errors.New("can't assign value to result")
	}
	value.Set(responseValue)

	return nil
}

const upstreamNode = "UpstreamNode"
const localNode = "LocalNode"
const localHandler = "LocalHandler"

// RoutingTable correlates RPC methods with implementations, providing a means
// (a) to ensure that only known requests leave our zone of responsibility, and
// (b) to prevent new requests in newer Geth versions from being accidentally
// routed to the upstream.
//
// A destination of LocalHandler indicates that the method should be processed
// by a Handler that was registered via RegisterHandler.
//
// Methods destined for UpstreamNode or LocalNode are sent to the local or
// upstream Ethereum nodes, respectively.
//
// The eth_accounts method has a Handler due to sub-accounts handling.
//
// The eth_sendTransaction method has a Handler because we're calling
// eth_estimateGas, signing it locally and sending eth_sendRawTransaction
// afterwards.
//
// The eth_sign method is sent to the local node because only it has an
// injected account to sign the payload with.
//
// The eth_getCompilers, eth_compileLLL, eth_compileSolidity and
// eth_compileSerpent methods all go to the local node because there's no need
// to send them anywhere.
//
// Trying to call a method not represented in this map will result in an error.
//
// List of RPC API methods:
// https://github.com/ethereum/wiki/wiki/JSON-RPC#json-rpc-methods
//
// List of methods supported by the upstream node:
// https://github.com/INFURA/infura/blob/master/docs/source/index.html.md#supported-json-rpc-methods
var RoutingTable = map[string]string{
	"db_getHex":                               localNode,
	"db_getString":                            localNode,
	"db_putHex":                               localNode,
	"db_putString":                            localNode,
	"eth_accounts":                            localHandler,
	"eth_blockNumber":                         upstreamNode,
	"eth_call":                                upstreamNode,
	"eth_coinbase":                            upstreamNode,
	"eth_compileLLL":                          localNode,
	"eth_compileSerpent":                      localNode,
	"eth_compileSolidity":                     localNode,
	"eth_estimateGas":                         upstreamNode,
	"eth_gasPrice":                            upstreamNode,
	"eth_getBalance":                          upstreamNode,
	"eth_getBlockByHash":                      upstreamNode,
	"eth_getBlockByNumber":                    upstreamNode,
	"eth_getBlockTransactionCountByHash":      upstreamNode,
	"eth_getBlockTransactionCountByNumber":    upstreamNode,
	"eth_getCode":                             upstreamNode,
	"eth_getCompilers":                        localNode,
	"eth_getFilterChanges":                    upstreamNode,
	"eth_getFilterLogs":                       upstreamNode,
	"eth_getLogs":                             upstreamNode,
	"eth_getStorageAt":                        upstreamNode,
	"eth_getTransactionByBlockHashAndIndex":   upstreamNode,
	"eth_getTransactionByBlockNumberAndIndex": upstreamNode,
	"eth_getTransactionByHash":                upstreamNode,
	"eth_getTransactionCount":                 upstreamNode,
	"eth_getTransactionReceipt":               upstreamNode,
	"eth_getUncleByBlockHashAndIndex":         upstreamNode,
	"eth_getUncleByBlockNumberAndIndex":       upstreamNode,
	"eth_getUncleCountByBlockHash":            upstreamNode,
	"eth_getUncleCountByBlockNumber":          upstreamNode,
	"eth_getWork":                             upstreamNode,
	"eth_hashrate":                            upstreamNode,
	"eth_mining":                              upstreamNode,
	"eth_newBlockFilter":                      upstreamNode,
	"eth_newFilter":                           upstreamNode,
	"eth_newPendingTransactionFilter":         upstreamNode,
	"eth_protocolVersion":                     upstreamNode,
	"eth_sendRawTransaction":                  upstreamNode,
	"eth_sendTransaction":                     localHandler,
	"eth_sign":                                localNode,
	"eth_submitHashrate":                      upstreamNode,
	"eth_submitWork":                          upstreamNode,
	"eth_syncing":                             upstreamNode,
	"eth_uninstallFilter":                     upstreamNode,
	"net_listening":                           upstreamNode,
	"net_peerCount":                           upstreamNode,
	"net_version":                             upstreamNode,
	"shh_addToGroup":                          localNode,
	"shh_getFilterChanges":                    localNode,
	"shh_getMessages":                         localNode,
	"shh_hasIdentity":                         localNode,
	"shh_newFilter":                           localNode,
	"shh_newGroup":                            localNode,
	"shh_newIdentity":                         localNode,
	"shh_post":                                localNode,
	"shh_uninstallFilter":                     localNode,
	"shh_version":                             localNode,
	"web3_clientVersion":                      localNode,
	"web3_sha3":                               localNode,
}
