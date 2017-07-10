package jail

// import (
// 	"context"

// 	gethcommon "github.com/ethereum/go-ethereum/common"
// 	"github.com/ethereum/go-ethereum/common/hexutil"
// 	"github.com/ethereum/go-ethereum/les/status"
// 	"github.com/ethereum/go-ethereum/rpc"
// 	"github.com/robertkrimen/otto"
// 	"github.com/status-im/status-go/geth/common"
// )

// const (
// 	// SendTransactionRequest is triggered on send transaction request
// 	SendTransactionRequest = "eth_sendTransaction"
// )

// // RequestManager represents interface to manage jailed requests.
// // Whenever some request passed to a Jail, needs to be pre/post processed,
// // request manager is the right place for that.
// type RequestManager struct {
// 	nodeManager common.NodeManager
// }

// func NewRequestManager(nodeManager common.NodeManager) *RequestManager {
// 	return &RequestManager{
// 		nodeManager: nodeManager,
// 	}
// }

// // PreProcessRequest pre-processes a given RPC call to a given Otto VM
// func (m *RequestManager) PreProcessRequest(vm *otto.Otto, req RPCCall) (string, error) {
// 	messageID := currentMessageID(vm.Context())

// 	return messageID, nil
// }

// // PostProcessRequest post-processes a given RPC call to a given Otto VM
// func (m *RequestManager) PostProcessRequest(vm *otto.Otto, req RPCCall, messageID string) {
// 	if len(messageID) > 0 {
// 		vm.Call("addContext", nil, messageID, common.MessageIDKey, messageID) // nolint: errcheck
// 	}

// 	// set extra markers for queued transaction requests
// 	if req.Method == SendTransactionRequest {
// 		vm.Call("addContext", nil, messageID, SendTransactionRequest, true) // nolint: errcheck
// 	}
// }

// // ProcessSendTransactionRequest processes send transaction request.
// // Both pre and post processing happens within this function. Pre-processing
// // happens before transaction is send to backend, and post processing occurs
// // when backend notifies that transaction sending is complete (either successfully
// // or with error)
// func (m *RequestManager) ProcessSendTransactionRequest(vm *otto.Otto, req RPCCall) (gethcommon.Hash, error) {
// 	lightEthereum, err := m.nodeManager.LightEthereumService()
// 	if err != nil {
// 		return gethcommon.Hash{}, err
// 	}

// 	backend := lightEthereum.StatusBackend

// 	messageID, err := m.PreProcessRequest(vm, req)
// 	if err != nil {
// 		return gethcommon.Hash{}, err
// 	}

// 	// onSendTransactionRequest() will use context to obtain and release ticket
// 	ctx := context.Background()
// 	ctx = context.WithValue(ctx, common.MessageIDKey, messageID)

// 	//  this call blocks, up until Complete Transaction is called
// 	txHash, err := backend.SendTransaction(ctx, sendTxArgsFromRPCCall(req))
// 	if err != nil {
// 		return gethcommon.Hash{}, err
// 	}

// 	// invoke post processing
// 	m.PostProcessRequest(vm, req, messageID)

// 	return txHash, nil
// }

// // RPCClient returns RPC client instance, creating it if necessary.
// func (m *RequestManager) RPCClient() (*rpc.Client, error) {
// 	return m.nodeManager.RPCClient()
// }

// // RPCCall represents RPC call parameters
// type RPCCall struct {
// 	ID     int64
// 	Method string
// 	Params []interface{}
// }

// func sendTxArgsFromRPCCall(req RPCCall) status.SendTxArgs {
// 	if req.Method != SendTransactionRequest { // no need to persist extra state for other requests
// 		return status.SendTxArgs{}
// 	}

// 	return status.SendTxArgs{
// 		From:     req.parseFromAddress(),
// 		To:       req.parseToAddress(),
// 		Value:    req.parseValue(),
// 		Data:     req.parseData(),
// 		Gas:      req.parseGas(),
// 		GasPrice: req.parseGasPrice(),
// 	}
// }

// func (r RPCCall) parseFromAddress() gethcommon.Address {
// 	params, ok := r.Params[0].(map[string]interface{})
// 	if !ok {
// 		return gethcommon.HexToAddress("0x")
// 	}

// 	from, ok := params["from"].(string)
// 	if !ok {
// 		from = "0x"
// 	}

// 	return gethcommon.HexToAddress(from)
// }

// func (r RPCCall) parseToAddress() *gethcommon.Address {
// 	params, ok := r.Params[0].(map[string]interface{})
// 	if !ok {
// 		return nil
// 	}

// 	to, ok := params["to"].(string)
// 	if !ok {
// 		return nil
// 	}

// 	address := gethcommon.HexToAddress(to)
// 	return &address
// }

// func (r RPCCall) parseData() hexutil.Bytes {
// 	params, ok := r.Params[0].(map[string]interface{})
// 	if !ok {
// 		return hexutil.Bytes("0x")
// 	}

// 	data, ok := params["data"].(string)
// 	if !ok {
// 		data = "0x"
// 	}

// 	byteCode, err := hexutil.Decode(data)
// 	if err != nil {
// 		byteCode = hexutil.Bytes(data)
// 	}

// 	return byteCode
// }

// // nolint: dupl
// func (r RPCCall) parseValue() *hexutil.Big {
// 	params, ok := r.Params[0].(map[string]interface{})
// 	if !ok {
// 		return nil
// 		//return (*hexutil.Big)(big.NewInt("0x0"))
// 	}

// 	inputValue, ok := params["value"].(string)
// 	if !ok {
// 		return nil
// 	}

// 	parsedValue, err := hexutil.DecodeBig(inputValue)
// 	if err != nil {
// 		return nil
// 	}

// 	return (*hexutil.Big)(parsedValue)
// }

// // nolint: dupl
// func (r RPCCall) parseGas() *hexutil.Big {
// 	params, ok := r.Params[0].(map[string]interface{})
// 	if !ok {
// 		return nil
// 	}

// 	inputValue, ok := params["gas"].(string)
// 	if !ok {
// 		return nil
// 	}

// 	parsedValue, err := hexutil.DecodeBig(inputValue)
// 	if err != nil {
// 		return nil
// 	}

// 	return (*hexutil.Big)(parsedValue)
// }

// // nolint: dupl
// func (r RPCCall) parseGasPrice() *hexutil.Big {
// 	params, ok := r.Params[0].(map[string]interface{})
// 	if !ok {
// 		return nil
// 	}

// 	inputValue, ok := params["gasPrice"].(string)
// 	if !ok {
// 		return nil
// 	}

// 	parsedValue, err := hexutil.DecodeBig(inputValue)
// 	if err != nil {
// 		return nil
// 	}

// 	return (*hexutil.Big)(parsedValue)
// }

// // currentMessageID looks for `status.message_id` variable in current JS context
// func currentMessageID(ctx otto.Context) string {
// 	if statusObj, ok := ctx.Symbols["status"]; ok {
// 		messageID, err := statusObj.Object().Get("message_id")
// 		if err != nil {
// 			return ""
// 		}
// 		if messageID, err := messageID.ToString(); err == nil {
// 			return messageID
// 		}
// 	}

// 	return ""
// }
