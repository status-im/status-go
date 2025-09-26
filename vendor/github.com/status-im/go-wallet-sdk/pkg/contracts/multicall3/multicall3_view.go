// Multicall3 defines aggregate functions as payable since they can be used for both
// read and write operations, so abigen doesn't generate read-only wrappers for them in Multicall3Caller.
// We manually define "View" versions of them here.

package multicall3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// ViewAggregate is a free data retrieval call binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes[] returnData)
func (_Multicall3 *Multicall3Caller) ViewAggregate(opts *bind.CallOpts, calls []IMulticall3Call) (*big.Int, []byte, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "aggregate", calls)

	if err != nil {
		return *new(*big.Int), *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new([]byte)).(*[]byte)

	return out0, out1, err

}

func (_Multicall3 *Multicall3Session) ViewAggregate(calls []IMulticall3Call) (*big.Int, []byte, error) {
	return _Multicall3.Contract.ViewAggregate(&_Multicall3.CallOpts, calls)
}

func (_Multicall3 *Multicall3CallerSession) ViewAggregate(calls []IMulticall3Call) (*big.Int, []byte, error) {
	return _Multicall3.Contract.ViewAggregate(&_Multicall3.CallOpts, calls)
}

// ViewBlockAndAggregate is a free data retrieval call binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Caller) ViewBlockAndAggregate(opts *bind.CallOpts, calls []IMulticall3Call) (*big.Int, [32]byte, []IMulticall3Result, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "blockAndAggregate", calls)

	if err != nil {
		return *new(*big.Int), *new([32]byte), *new([]IMulticall3Result), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	out2 := *abi.ConvertType(out[2], new([]IMulticall3Result)).(*[]IMulticall3Result)

	return out0, out1, out2, err

}

func (_Multicall3 *Multicall3Session) ViewBlockAndAggregate(calls []IMulticall3Call) (*big.Int, [32]byte, []IMulticall3Result, error) {
	return _Multicall3.Contract.ViewBlockAndAggregate(&_Multicall3.CallOpts, calls)
}

func (_Multicall3 *Multicall3CallerSession) ViewBlockAndAggregate(calls []IMulticall3Call) (*big.Int, [32]byte, []IMulticall3Result, error) {
	return _Multicall3.Contract.ViewBlockAndAggregate(&_Multicall3.CallOpts, calls)
}

// ViewTryAggregate is a free data retrieval call binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Caller) ViewTryAggregate(opts *bind.CallOpts, requireSuccess bool, calls []IMulticall3Call) ([]IMulticall3Result, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "tryAggregate", requireSuccess, calls)

	if err != nil {
		return *new([]IMulticall3Result), err
	}

	out0 := *abi.ConvertType(out[0], new([]IMulticall3Result)).(*[]IMulticall3Result)

	return out0, err
}

func (_Multicall3 *Multicall3Session) ViewTryAggregate(requireSuccess bool, calls []IMulticall3Call) ([]IMulticall3Result, error) {
	return _Multicall3.Contract.ViewTryAggregate(&_Multicall3.CallOpts, requireSuccess, calls)
}

func (_Multicall3 *Multicall3CallerSession) ViewTryAggregate(requireSuccess bool, calls []IMulticall3Call) ([]IMulticall3Result, error) {
	return _Multicall3.Contract.ViewTryAggregate(&_Multicall3.CallOpts, requireSuccess, calls)
}

// ViewTryBlockAndAggregate is a free data retrieval call binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Caller) ViewTryBlockAndAggregate(opts *bind.CallOpts, requireSuccess bool, calls []IMulticall3Call) (*big.Int, [32]byte, []IMulticall3Result, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "tryBlockAndAggregate", requireSuccess, calls)

	if err != nil {
		return *new(*big.Int), *new([32]byte), *new([]IMulticall3Result), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	out2 := *abi.ConvertType(out[2], new([]IMulticall3Result)).(*[]IMulticall3Result)

	return out0, out1, out2, err
}

func (_Multicall3 *Multicall3Session) ViewTryBlockAndAggregate(requireSuccess bool, calls []IMulticall3Call) (*big.Int, [32]byte, []IMulticall3Result, error) {
	return _Multicall3.Contract.ViewTryBlockAndAggregate(&_Multicall3.CallOpts, requireSuccess, calls)
}

func (_Multicall3 *Multicall3CallerSession) ViewTryBlockAndAggregate(requireSuccess bool, calls []IMulticall3Call) (*big.Int, [32]byte, []IMulticall3Result, error) {
	return _Multicall3.Contract.ViewTryBlockAndAggregate(&_Multicall3.CallOpts, requireSuccess, calls)
}

// ViewAggregate3 is a free data retrieval call binding the contract method 0x82ad56cb.
//
// Solidity: function aggregate3((address,bool,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Caller) ViewAggregate3(opts *bind.CallOpts, calls []IMulticall3Call3) ([]IMulticall3Result, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "aggregate3", calls)

	if err != nil {
		return *new([]IMulticall3Result), err
	}

	out0 := *abi.ConvertType(out[0], new([]IMulticall3Result)).(*[]IMulticall3Result)

	return out0, err

}

func (_Multicall3 *Multicall3Session) ViewAggregate3(calls []IMulticall3Call3) ([]IMulticall3Result, error) {
	return _Multicall3.Contract.ViewAggregate3(&_Multicall3.CallOpts, calls)
}

func (_Multicall3 *Multicall3CallerSession) ViewAggregate3(calls []IMulticall3Call3) ([]IMulticall3Result, error) {
	return _Multicall3.Contract.ViewAggregate3(&_Multicall3.CallOpts, calls)
}

// ViewAggregate3Value is a free data retrieval call binding the contract method 0x174dea71.
//
// Solidity: function aggregate3Value((address,bool,uint256,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Caller) ViewAggregate3Value(opts *bind.CallOpts, calls []IMulticall3Call3Value) ([]IMulticall3Result, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "aggregate3Value", calls)

	if err != nil {
		return *new([]IMulticall3Result), err
	}

	out0 := *abi.ConvertType(out[0], new([]IMulticall3Result)).(*[]IMulticall3Result)

	return out0, err

}

func (_Multicall3 *Multicall3Session) ViewAggregate3Value(calls []IMulticall3Call3Value) ([]IMulticall3Result, error) {
	return _Multicall3.Contract.ViewAggregate3Value(&_Multicall3.CallOpts, calls)
}

func (_Multicall3 *Multicall3CallerSession) ViewAggregate3Value(calls []IMulticall3Call3Value) ([]IMulticall3Result, error) {
	return _Multicall3.Contract.ViewAggregate3Value(&_Multicall3.CallOpts, calls)
}
