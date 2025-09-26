// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package multicall3

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// IMulticall3Call is an auto generated low-level Go binding around an user-defined struct.
type IMulticall3Call struct {
	Target   common.Address
	CallData []byte
}

// IMulticall3Call3 is an auto generated low-level Go binding around an user-defined struct.
type IMulticall3Call3 struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

// IMulticall3Call3Value is an auto generated low-level Go binding around an user-defined struct.
type IMulticall3Call3Value struct {
	Target       common.Address
	AllowFailure bool
	Value        *big.Int
	CallData     []byte
}

// IMulticall3Result is an auto generated low-level Go binding around an user-defined struct.
type IMulticall3Result struct {
	Success    bool
	ReturnData []byte
}

// Multicall3MetaData contains all meta data concerning the Multicall3 contract.
var Multicall3MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"aggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes[]\",\"name\":\"returnData\",\"type\":\"bytes[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"allowFailure\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Call3[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"aggregate3\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"allowFailure\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Call3Value[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"aggregate3Value\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"blockAndAggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBasefee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"basefee\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chainid\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockCoinbase\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"coinbase\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockDifficulty\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"difficulty\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockGasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gaslimit\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"getEthBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"requireSuccess\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"tryAggregate\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"requireSuccess\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"tryBlockAndAggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structIMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// Multicall3ABI is the input ABI used to generate the binding from.
// Deprecated: Use Multicall3MetaData.ABI instead.
var Multicall3ABI = Multicall3MetaData.ABI

// Multicall3 is an auto generated Go binding around an Ethereum contract.
type Multicall3 struct {
	Multicall3Caller     // Read-only binding to the contract
	Multicall3Transactor // Write-only binding to the contract
	Multicall3Filterer   // Log filterer for contract events
}

// Multicall3Caller is an auto generated read-only Go binding around an Ethereum contract.
type Multicall3Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Multicall3Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Multicall3Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Multicall3Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Multicall3Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Multicall3Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Multicall3Session struct {
	Contract     *Multicall3       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Multicall3CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Multicall3CallerSession struct {
	Contract *Multicall3Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// Multicall3TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Multicall3TransactorSession struct {
	Contract     *Multicall3Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// Multicall3Raw is an auto generated low-level Go binding around an Ethereum contract.
type Multicall3Raw struct {
	Contract *Multicall3 // Generic contract binding to access the raw methods on
}

// Multicall3CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Multicall3CallerRaw struct {
	Contract *Multicall3Caller // Generic read-only contract binding to access the raw methods on
}

// Multicall3TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Multicall3TransactorRaw struct {
	Contract *Multicall3Transactor // Generic write-only contract binding to access the raw methods on
}

// NewMulticall3 creates a new instance of Multicall3, bound to a specific deployed contract.
func NewMulticall3(address common.Address, backend bind.ContractBackend) (*Multicall3, error) {
	contract, err := bindMulticall3(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Multicall3{Multicall3Caller: Multicall3Caller{contract: contract}, Multicall3Transactor: Multicall3Transactor{contract: contract}, Multicall3Filterer: Multicall3Filterer{contract: contract}}, nil
}

// NewMulticall3Caller creates a new read-only instance of Multicall3, bound to a specific deployed contract.
func NewMulticall3Caller(address common.Address, caller bind.ContractCaller) (*Multicall3Caller, error) {
	contract, err := bindMulticall3(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Multicall3Caller{contract: contract}, nil
}

// NewMulticall3Transactor creates a new write-only instance of Multicall3, bound to a specific deployed contract.
func NewMulticall3Transactor(address common.Address, transactor bind.ContractTransactor) (*Multicall3Transactor, error) {
	contract, err := bindMulticall3(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Multicall3Transactor{contract: contract}, nil
}

// NewMulticall3Filterer creates a new log filterer instance of Multicall3, bound to a specific deployed contract.
func NewMulticall3Filterer(address common.Address, filterer bind.ContractFilterer) (*Multicall3Filterer, error) {
	contract, err := bindMulticall3(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Multicall3Filterer{contract: contract}, nil
}

// bindMulticall3 binds a generic wrapper to an already deployed contract.
func bindMulticall3(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Multicall3MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Multicall3 *Multicall3Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Multicall3.Contract.Multicall3Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Multicall3 *Multicall3Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Multicall3.Contract.Multicall3Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Multicall3 *Multicall3Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Multicall3.Contract.Multicall3Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Multicall3 *Multicall3CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Multicall3.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Multicall3 *Multicall3TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Multicall3.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Multicall3 *Multicall3TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Multicall3.Contract.contract.Transact(opts, method, params...)
}

// GetBasefee is a free data retrieval call binding the contract method 0x3e64a696.
//
// Solidity: function getBasefee() view returns(uint256 basefee)
func (_Multicall3 *Multicall3Caller) GetBasefee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getBasefee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBasefee is a free data retrieval call binding the contract method 0x3e64a696.
//
// Solidity: function getBasefee() view returns(uint256 basefee)
func (_Multicall3 *Multicall3Session) GetBasefee() (*big.Int, error) {
	return _Multicall3.Contract.GetBasefee(&_Multicall3.CallOpts)
}

// GetBasefee is a free data retrieval call binding the contract method 0x3e64a696.
//
// Solidity: function getBasefee() view returns(uint256 basefee)
func (_Multicall3 *Multicall3CallerSession) GetBasefee() (*big.Int, error) {
	return _Multicall3.Contract.GetBasefee(&_Multicall3.CallOpts)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_Multicall3 *Multicall3Caller) GetBlockHash(opts *bind.CallOpts, blockNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getBlockHash", blockNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_Multicall3 *Multicall3Session) GetBlockHash(blockNumber *big.Int) ([32]byte, error) {
	return _Multicall3.Contract.GetBlockHash(&_Multicall3.CallOpts, blockNumber)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_Multicall3 *Multicall3CallerSession) GetBlockHash(blockNumber *big.Int) ([32]byte, error) {
	return _Multicall3.Contract.GetBlockHash(&_Multicall3.CallOpts, blockNumber)
}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_Multicall3 *Multicall3Caller) GetBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_Multicall3 *Multicall3Session) GetBlockNumber() (*big.Int, error) {
	return _Multicall3.Contract.GetBlockNumber(&_Multicall3.CallOpts)
}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_Multicall3 *Multicall3CallerSession) GetBlockNumber() (*big.Int, error) {
	return _Multicall3.Contract.GetBlockNumber(&_Multicall3.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainid)
func (_Multicall3 *Multicall3Caller) GetChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getChainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainid)
func (_Multicall3 *Multicall3Session) GetChainId() (*big.Int, error) {
	return _Multicall3.Contract.GetChainId(&_Multicall3.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainid)
func (_Multicall3 *Multicall3CallerSession) GetChainId() (*big.Int, error) {
	return _Multicall3.Contract.GetChainId(&_Multicall3.CallOpts)
}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_Multicall3 *Multicall3Caller) GetCurrentBlockCoinbase(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getCurrentBlockCoinbase")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_Multicall3 *Multicall3Session) GetCurrentBlockCoinbase() (common.Address, error) {
	return _Multicall3.Contract.GetCurrentBlockCoinbase(&_Multicall3.CallOpts)
}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_Multicall3 *Multicall3CallerSession) GetCurrentBlockCoinbase() (common.Address, error) {
	return _Multicall3.Contract.GetCurrentBlockCoinbase(&_Multicall3.CallOpts)
}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_Multicall3 *Multicall3Caller) GetCurrentBlockDifficulty(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getCurrentBlockDifficulty")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_Multicall3 *Multicall3Session) GetCurrentBlockDifficulty() (*big.Int, error) {
	return _Multicall3.Contract.GetCurrentBlockDifficulty(&_Multicall3.CallOpts)
}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_Multicall3 *Multicall3CallerSession) GetCurrentBlockDifficulty() (*big.Int, error) {
	return _Multicall3.Contract.GetCurrentBlockDifficulty(&_Multicall3.CallOpts)
}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_Multicall3 *Multicall3Caller) GetCurrentBlockGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getCurrentBlockGasLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_Multicall3 *Multicall3Session) GetCurrentBlockGasLimit() (*big.Int, error) {
	return _Multicall3.Contract.GetCurrentBlockGasLimit(&_Multicall3.CallOpts)
}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_Multicall3 *Multicall3CallerSession) GetCurrentBlockGasLimit() (*big.Int, error) {
	return _Multicall3.Contract.GetCurrentBlockGasLimit(&_Multicall3.CallOpts)
}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_Multicall3 *Multicall3Caller) GetCurrentBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getCurrentBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_Multicall3 *Multicall3Session) GetCurrentBlockTimestamp() (*big.Int, error) {
	return _Multicall3.Contract.GetCurrentBlockTimestamp(&_Multicall3.CallOpts)
}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_Multicall3 *Multicall3CallerSession) GetCurrentBlockTimestamp() (*big.Int, error) {
	return _Multicall3.Contract.GetCurrentBlockTimestamp(&_Multicall3.CallOpts)
}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_Multicall3 *Multicall3Caller) GetEthBalance(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getEthBalance", addr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_Multicall3 *Multicall3Session) GetEthBalance(addr common.Address) (*big.Int, error) {
	return _Multicall3.Contract.GetEthBalance(&_Multicall3.CallOpts, addr)
}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_Multicall3 *Multicall3CallerSession) GetEthBalance(addr common.Address) (*big.Int, error) {
	return _Multicall3.Contract.GetEthBalance(&_Multicall3.CallOpts, addr)
}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_Multicall3 *Multicall3Caller) GetLastBlockHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Multicall3.contract.Call(opts, &out, "getLastBlockHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_Multicall3 *Multicall3Session) GetLastBlockHash() ([32]byte, error) {
	return _Multicall3.Contract.GetLastBlockHash(&_Multicall3.CallOpts)
}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_Multicall3 *Multicall3CallerSession) GetLastBlockHash() ([32]byte, error) {
	return _Multicall3.Contract.GetLastBlockHash(&_Multicall3.CallOpts)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes[] returnData)
func (_Multicall3 *Multicall3Transactor) Aggregate(opts *bind.TransactOpts, calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.contract.Transact(opts, "aggregate", calls)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes[] returnData)
func (_Multicall3 *Multicall3Session) Aggregate(calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.Contract.Aggregate(&_Multicall3.TransactOpts, calls)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes[] returnData)
func (_Multicall3 *Multicall3TransactorSession) Aggregate(calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.Contract.Aggregate(&_Multicall3.TransactOpts, calls)
}

// Aggregate3 is a paid mutator transaction binding the contract method 0x82ad56cb.
//
// Solidity: function aggregate3((address,bool,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Transactor) Aggregate3(opts *bind.TransactOpts, calls []IMulticall3Call3) (*types.Transaction, error) {
	return _Multicall3.contract.Transact(opts, "aggregate3", calls)
}

// Aggregate3 is a paid mutator transaction binding the contract method 0x82ad56cb.
//
// Solidity: function aggregate3((address,bool,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Session) Aggregate3(calls []IMulticall3Call3) (*types.Transaction, error) {
	return _Multicall3.Contract.Aggregate3(&_Multicall3.TransactOpts, calls)
}

// Aggregate3 is a paid mutator transaction binding the contract method 0x82ad56cb.
//
// Solidity: function aggregate3((address,bool,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3TransactorSession) Aggregate3(calls []IMulticall3Call3) (*types.Transaction, error) {
	return _Multicall3.Contract.Aggregate3(&_Multicall3.TransactOpts, calls)
}

// Aggregate3Value is a paid mutator transaction binding the contract method 0x174dea71.
//
// Solidity: function aggregate3Value((address,bool,uint256,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Transactor) Aggregate3Value(opts *bind.TransactOpts, calls []IMulticall3Call3Value) (*types.Transaction, error) {
	return _Multicall3.contract.Transact(opts, "aggregate3Value", calls)
}

// Aggregate3Value is a paid mutator transaction binding the contract method 0x174dea71.
//
// Solidity: function aggregate3Value((address,bool,uint256,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Session) Aggregate3Value(calls []IMulticall3Call3Value) (*types.Transaction, error) {
	return _Multicall3.Contract.Aggregate3Value(&_Multicall3.TransactOpts, calls)
}

// Aggregate3Value is a paid mutator transaction binding the contract method 0x174dea71.
//
// Solidity: function aggregate3Value((address,bool,uint256,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3TransactorSession) Aggregate3Value(calls []IMulticall3Call3Value) (*types.Transaction, error) {
	return _Multicall3.Contract.Aggregate3Value(&_Multicall3.TransactOpts, calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Transactor) BlockAndAggregate(opts *bind.TransactOpts, calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.contract.Transact(opts, "blockAndAggregate", calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Session) BlockAndAggregate(calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.Contract.BlockAndAggregate(&_Multicall3.TransactOpts, calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall3 *Multicall3TransactorSession) BlockAndAggregate(calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.Contract.BlockAndAggregate(&_Multicall3.TransactOpts, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Transactor) TryAggregate(opts *bind.TransactOpts, requireSuccess bool, calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.contract.Transact(opts, "tryAggregate", requireSuccess, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Session) TryAggregate(requireSuccess bool, calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.Contract.TryAggregate(&_Multicall3.TransactOpts, requireSuccess, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_Multicall3 *Multicall3TransactorSession) TryAggregate(requireSuccess bool, calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.Contract.TryAggregate(&_Multicall3.TransactOpts, requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Transactor) TryBlockAndAggregate(opts *bind.TransactOpts, requireSuccess bool, calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.contract.Transact(opts, "tryBlockAndAggregate", requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall3 *Multicall3Session) TryBlockAndAggregate(requireSuccess bool, calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.Contract.TryBlockAndAggregate(&_Multicall3.TransactOpts, requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall3 *Multicall3TransactorSession) TryBlockAndAggregate(requireSuccess bool, calls []IMulticall3Call) (*types.Transaction, error) {
	return _Multicall3.Contract.TryBlockAndAggregate(&_Multicall3.TransactOpts, requireSuccess, calls)
}
