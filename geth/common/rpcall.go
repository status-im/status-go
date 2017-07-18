package common

import (
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// RPCCall represents a unit of a rpc request which is to be executed.
type RPCCall struct {
	ID     int64
	Method string
	Params []interface{}
}

// ParseFromAddress returns the address associated with the RPCCall.
func (r RPCCall) ParseFromAddress() gethcommon.Address {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return gethcommon.HexToAddress("0x")
	}

	from, ok := params["from"].(string)
	if !ok {
		from = "0x"
	}

	return gethcommon.HexToAddress(from)
}

// ParseToAddress returns the gethcommon.Address associated with the call.
func (r RPCCall) ParseToAddress() *gethcommon.Address {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil
	}

	to, ok := params["to"].(string)
	if !ok {
		return nil
	}

	address := gethcommon.HexToAddress(to)
	return &address
}

// ParseData returns the bytes associated with the call.
func (r RPCCall) ParseData() hexutil.Bytes {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return hexutil.Bytes("0x")
	}

	data, ok := params["data"].(string)
	if !ok {
		data = "0x"
	}

	byteCode, err := hexutil.Decode(data)
	if err != nil {
		byteCode = hexutil.Bytes(data)
	}

	return byteCode
}

// ParseValue returns the hex big associated with the call.
// nolint: dupl
func (r RPCCall) ParseValue() *hexutil.Big {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil
		//return (*hexutil.Big)(big.NewInt("0x0"))
	}

	inputValue, ok := params["value"].(string)
	if !ok {
		return nil
	}

	parsedValue, err := hexutil.DecodeBig(inputValue)
	if err != nil {
		return nil
	}

	return (*hexutil.Big)(parsedValue)
}

// ParseGas returns the hex big associated with the call.
// nolint: dupl
func (r RPCCall) ParseGas() *hexutil.Big {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil
	}

	inputValue, ok := params["gas"].(string)
	if !ok {
		return nil
	}

	parsedValue, err := hexutil.DecodeBig(inputValue)
	if err != nil {
		return nil
	}

	return (*hexutil.Big)(parsedValue)
}

// ParseGasPrice returns the hex big associated with the call.
// nolint: dupl
func (r RPCCall) ParseGasPrice() *hexutil.Big {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil
	}

	inputValue, ok := params["gasPrice"].(string)
	if !ok {
		return nil
	}

	parsedValue, err := hexutil.DecodeBig(inputValue)
	if err != nil {
		return nil
	}

	return (*hexutil.Big)(parsedValue)
}
