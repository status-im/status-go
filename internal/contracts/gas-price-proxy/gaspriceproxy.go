// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package gaspriceproxy

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

// GaspriceproxyMetaData contains all meta data concerning the Gaspriceproxy contract.
var GaspriceproxyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"DecimalsUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"GasPriceUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"L1BaseFeeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"OverheadUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"ScalarUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"admin\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"adminCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gasPrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getAdmin\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"adminAddress\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"getL1Fee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"getL1GasUsed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"initPayload\",\"type\":\"bytes\"}],\"name\":\"init\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"overhead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"scalar\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"adminAddress\",\"type\":\"address\"}],\"name\":\"setAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_decimals\",\"type\":\"uint256\"}],\"name\":\"setDecimals\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_gasPrice\",\"type\":\"uint256\"}],\"name\":\"setGasPrice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_baseFee\",\"type\":\"uint256\"}],\"name\":\"setL1BaseFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_overhead\",\"type\":\"uint256\"}],\"name\":\"setOverhead\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_scalar\",\"type\":\"uint256\"}],\"name\":\"setScalar\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// GaspriceproxyABI is the input ABI used to generate the binding from.
// Deprecated: Use GaspriceproxyMetaData.ABI instead.
var GaspriceproxyABI = GaspriceproxyMetaData.ABI

// Gaspriceproxy is an auto generated Go binding around an Ethereum contract.
type Gaspriceproxy struct {
	GaspriceproxyCaller     // Read-only binding to the contract
	GaspriceproxyTransactor // Write-only binding to the contract
	GaspriceproxyFilterer   // Log filterer for contract events
}

// GaspriceproxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type GaspriceproxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GaspriceproxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GaspriceproxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GaspriceproxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GaspriceproxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GaspriceproxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GaspriceproxySession struct {
	Contract     *Gaspriceproxy    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GaspriceproxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GaspriceproxyCallerSession struct {
	Contract *GaspriceproxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// GaspriceproxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GaspriceproxyTransactorSession struct {
	Contract     *GaspriceproxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// GaspriceproxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type GaspriceproxyRaw struct {
	Contract *Gaspriceproxy // Generic contract binding to access the raw methods on
}

// GaspriceproxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GaspriceproxyCallerRaw struct {
	Contract *GaspriceproxyCaller // Generic read-only contract binding to access the raw methods on
}

// GaspriceproxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GaspriceproxyTransactorRaw struct {
	Contract *GaspriceproxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGaspriceproxy creates a new instance of Gaspriceproxy, bound to a specific deployed contract.
func NewGaspriceproxy(address common.Address, backend bind.ContractBackend) (*Gaspriceproxy, error) {
	contract, err := bindGaspriceproxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Gaspriceproxy{GaspriceproxyCaller: GaspriceproxyCaller{contract: contract}, GaspriceproxyTransactor: GaspriceproxyTransactor{contract: contract}, GaspriceproxyFilterer: GaspriceproxyFilterer{contract: contract}}, nil
}

// NewGaspriceproxyCaller creates a new read-only instance of Gaspriceproxy, bound to a specific deployed contract.
func NewGaspriceproxyCaller(address common.Address, caller bind.ContractCaller) (*GaspriceproxyCaller, error) {
	contract, err := bindGaspriceproxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GaspriceproxyCaller{contract: contract}, nil
}

// NewGaspriceproxyTransactor creates a new write-only instance of Gaspriceproxy, bound to a specific deployed contract.
func NewGaspriceproxyTransactor(address common.Address, transactor bind.ContractTransactor) (*GaspriceproxyTransactor, error) {
	contract, err := bindGaspriceproxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GaspriceproxyTransactor{contract: contract}, nil
}

// NewGaspriceproxyFilterer creates a new log filterer instance of Gaspriceproxy, bound to a specific deployed contract.
func NewGaspriceproxyFilterer(address common.Address, filterer bind.ContractFilterer) (*GaspriceproxyFilterer, error) {
	contract, err := bindGaspriceproxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GaspriceproxyFilterer{contract: contract}, nil
}

// bindGaspriceproxy binds a generic wrapper to an already deployed contract.
func bindGaspriceproxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := GaspriceproxyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Gaspriceproxy *GaspriceproxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Gaspriceproxy.Contract.GaspriceproxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Gaspriceproxy *GaspriceproxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.GaspriceproxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Gaspriceproxy *GaspriceproxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.GaspriceproxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Gaspriceproxy *GaspriceproxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Gaspriceproxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Gaspriceproxy *GaspriceproxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Gaspriceproxy *GaspriceproxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.contract.Transact(opts, method, params...)
}

// Admin is a free data retrieval call binding the contract method 0xf851a440.
//
// Solidity: function admin() view returns(address)
func (_Gaspriceproxy *GaspriceproxyCaller) Admin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "admin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Admin is a free data retrieval call binding the contract method 0xf851a440.
//
// Solidity: function admin() view returns(address)
func (_Gaspriceproxy *GaspriceproxySession) Admin() (common.Address, error) {
	return _Gaspriceproxy.Contract.Admin(&_Gaspriceproxy.CallOpts)
}

// Admin is a free data retrieval call binding the contract method 0xf851a440.
//
// Solidity: function admin() view returns(address)
func (_Gaspriceproxy *GaspriceproxyCallerSession) Admin() (common.Address, error) {
	return _Gaspriceproxy.Contract.Admin(&_Gaspriceproxy.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCaller) Decimals(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxySession) Decimals() (*big.Int, error) {
	return _Gaspriceproxy.Contract.Decimals(&_Gaspriceproxy.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCallerSession) Decimals() (*big.Int, error) {
	return _Gaspriceproxy.Contract.Decimals(&_Gaspriceproxy.CallOpts)
}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCaller) GasPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "gasPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxySession) GasPrice() (*big.Int, error) {
	return _Gaspriceproxy.Contract.GasPrice(&_Gaspriceproxy.CallOpts)
}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCallerSession) GasPrice() (*big.Int, error) {
	return _Gaspriceproxy.Contract.GasPrice(&_Gaspriceproxy.CallOpts)
}

// GetAdmin is a free data retrieval call binding the contract method 0x6e9960c3.
//
// Solidity: function getAdmin() view returns(address adminAddress)
func (_Gaspriceproxy *GaspriceproxyCaller) GetAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "getAdmin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetAdmin is a free data retrieval call binding the contract method 0x6e9960c3.
//
// Solidity: function getAdmin() view returns(address adminAddress)
func (_Gaspriceproxy *GaspriceproxySession) GetAdmin() (common.Address, error) {
	return _Gaspriceproxy.Contract.GetAdmin(&_Gaspriceproxy.CallOpts)
}

// GetAdmin is a free data retrieval call binding the contract method 0x6e9960c3.
//
// Solidity: function getAdmin() view returns(address adminAddress)
func (_Gaspriceproxy *GaspriceproxyCallerSession) GetAdmin() (common.Address, error) {
	return _Gaspriceproxy.Contract.GetAdmin(&_Gaspriceproxy.CallOpts)
}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCaller) GetL1Fee(opts *bind.CallOpts, _data []byte) (*big.Int, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "getL1Fee", _data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_Gaspriceproxy *GaspriceproxySession) GetL1Fee(_data []byte) (*big.Int, error) {
	return _Gaspriceproxy.Contract.GetL1Fee(&_Gaspriceproxy.CallOpts, _data)
}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCallerSession) GetL1Fee(_data []byte) (*big.Int, error) {
	return _Gaspriceproxy.Contract.GetL1Fee(&_Gaspriceproxy.CallOpts, _data)
}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCaller) GetL1GasUsed(opts *bind.CallOpts, _data []byte) (*big.Int, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "getL1GasUsed", _data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_Gaspriceproxy *GaspriceproxySession) GetL1GasUsed(_data []byte) (*big.Int, error) {
	return _Gaspriceproxy.Contract.GetL1GasUsed(&_Gaspriceproxy.CallOpts, _data)
}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCallerSession) GetL1GasUsed(_data []byte) (*big.Int, error) {
	return _Gaspriceproxy.Contract.GetL1GasUsed(&_Gaspriceproxy.CallOpts, _data)
}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCaller) L1BaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "l1BaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxySession) L1BaseFee() (*big.Int, error) {
	return _Gaspriceproxy.Contract.L1BaseFee(&_Gaspriceproxy.CallOpts)
}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCallerSession) L1BaseFee() (*big.Int, error) {
	return _Gaspriceproxy.Contract.L1BaseFee(&_Gaspriceproxy.CallOpts)
}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCaller) Overhead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "overhead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxySession) Overhead() (*big.Int, error) {
	return _Gaspriceproxy.Contract.Overhead(&_Gaspriceproxy.CallOpts)
}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCallerSession) Overhead() (*big.Int, error) {
	return _Gaspriceproxy.Contract.Overhead(&_Gaspriceproxy.CallOpts)
}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCaller) Scalar(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Gaspriceproxy.contract.Call(opts, &out, "scalar")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxySession) Scalar() (*big.Int, error) {
	return _Gaspriceproxy.Contract.Scalar(&_Gaspriceproxy.CallOpts)
}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_Gaspriceproxy *GaspriceproxyCallerSession) Scalar() (*big.Int, error) {
	return _Gaspriceproxy.Contract.Scalar(&_Gaspriceproxy.CallOpts)
}

// AdminCall is a paid mutator transaction binding the contract method 0xbf64a82d.
//
// Solidity: function adminCall(address target, bytes data) payable returns()
func (_Gaspriceproxy *GaspriceproxyTransactor) AdminCall(opts *bind.TransactOpts, target common.Address, data []byte) (*types.Transaction, error) {
	return _Gaspriceproxy.contract.Transact(opts, "adminCall", target, data)
}

// AdminCall is a paid mutator transaction binding the contract method 0xbf64a82d.
//
// Solidity: function adminCall(address target, bytes data) payable returns()
func (_Gaspriceproxy *GaspriceproxySession) AdminCall(target common.Address, data []byte) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.AdminCall(&_Gaspriceproxy.TransactOpts, target, data)
}

// AdminCall is a paid mutator transaction binding the contract method 0xbf64a82d.
//
// Solidity: function adminCall(address target, bytes data) payable returns()
func (_Gaspriceproxy *GaspriceproxyTransactorSession) AdminCall(target common.Address, data []byte) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.AdminCall(&_Gaspriceproxy.TransactOpts, target, data)
}

// Init is a paid mutator transaction binding the contract method 0x4ddf47d4.
//
// Solidity: function init(bytes initPayload) returns(bytes4)
func (_Gaspriceproxy *GaspriceproxyTransactor) Init(opts *bind.TransactOpts, initPayload []byte) (*types.Transaction, error) {
	return _Gaspriceproxy.contract.Transact(opts, "init", initPayload)
}

// Init is a paid mutator transaction binding the contract method 0x4ddf47d4.
//
// Solidity: function init(bytes initPayload) returns(bytes4)
func (_Gaspriceproxy *GaspriceproxySession) Init(initPayload []byte) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.Init(&_Gaspriceproxy.TransactOpts, initPayload)
}

// Init is a paid mutator transaction binding the contract method 0x4ddf47d4.
//
// Solidity: function init(bytes initPayload) returns(bytes4)
func (_Gaspriceproxy *GaspriceproxyTransactorSession) Init(initPayload []byte) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.Init(&_Gaspriceproxy.TransactOpts, initPayload)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x704b6c02.
//
// Solidity: function setAdmin(address adminAddress) returns()
func (_Gaspriceproxy *GaspriceproxyTransactor) SetAdmin(opts *bind.TransactOpts, adminAddress common.Address) (*types.Transaction, error) {
	return _Gaspriceproxy.contract.Transact(opts, "setAdmin", adminAddress)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x704b6c02.
//
// Solidity: function setAdmin(address adminAddress) returns()
func (_Gaspriceproxy *GaspriceproxySession) SetAdmin(adminAddress common.Address) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetAdmin(&_Gaspriceproxy.TransactOpts, adminAddress)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x704b6c02.
//
// Solidity: function setAdmin(address adminAddress) returns()
func (_Gaspriceproxy *GaspriceproxyTransactorSession) SetAdmin(adminAddress common.Address) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetAdmin(&_Gaspriceproxy.TransactOpts, adminAddress)
}

// SetDecimals is a paid mutator transaction binding the contract method 0x8c8885c8.
//
// Solidity: function setDecimals(uint256 _decimals) returns()
func (_Gaspriceproxy *GaspriceproxyTransactor) SetDecimals(opts *bind.TransactOpts, _decimals *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.contract.Transact(opts, "setDecimals", _decimals)
}

// SetDecimals is a paid mutator transaction binding the contract method 0x8c8885c8.
//
// Solidity: function setDecimals(uint256 _decimals) returns()
func (_Gaspriceproxy *GaspriceproxySession) SetDecimals(_decimals *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetDecimals(&_Gaspriceproxy.TransactOpts, _decimals)
}

// SetDecimals is a paid mutator transaction binding the contract method 0x8c8885c8.
//
// Solidity: function setDecimals(uint256 _decimals) returns()
func (_Gaspriceproxy *GaspriceproxyTransactorSession) SetDecimals(_decimals *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetDecimals(&_Gaspriceproxy.TransactOpts, _decimals)
}

// SetGasPrice is a paid mutator transaction binding the contract method 0xbf1fe420.
//
// Solidity: function setGasPrice(uint256 _gasPrice) returns()
func (_Gaspriceproxy *GaspriceproxyTransactor) SetGasPrice(opts *bind.TransactOpts, _gasPrice *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.contract.Transact(opts, "setGasPrice", _gasPrice)
}

// SetGasPrice is a paid mutator transaction binding the contract method 0xbf1fe420.
//
// Solidity: function setGasPrice(uint256 _gasPrice) returns()
func (_Gaspriceproxy *GaspriceproxySession) SetGasPrice(_gasPrice *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetGasPrice(&_Gaspriceproxy.TransactOpts, _gasPrice)
}

// SetGasPrice is a paid mutator transaction binding the contract method 0xbf1fe420.
//
// Solidity: function setGasPrice(uint256 _gasPrice) returns()
func (_Gaspriceproxy *GaspriceproxyTransactorSession) SetGasPrice(_gasPrice *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetGasPrice(&_Gaspriceproxy.TransactOpts, _gasPrice)
}

// SetL1BaseFee is a paid mutator transaction binding the contract method 0xbede39b5.
//
// Solidity: function setL1BaseFee(uint256 _baseFee) returns()
func (_Gaspriceproxy *GaspriceproxyTransactor) SetL1BaseFee(opts *bind.TransactOpts, _baseFee *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.contract.Transact(opts, "setL1BaseFee", _baseFee)
}

// SetL1BaseFee is a paid mutator transaction binding the contract method 0xbede39b5.
//
// Solidity: function setL1BaseFee(uint256 _baseFee) returns()
func (_Gaspriceproxy *GaspriceproxySession) SetL1BaseFee(_baseFee *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetL1BaseFee(&_Gaspriceproxy.TransactOpts, _baseFee)
}

// SetL1BaseFee is a paid mutator transaction binding the contract method 0xbede39b5.
//
// Solidity: function setL1BaseFee(uint256 _baseFee) returns()
func (_Gaspriceproxy *GaspriceproxyTransactorSession) SetL1BaseFee(_baseFee *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetL1BaseFee(&_Gaspriceproxy.TransactOpts, _baseFee)
}

// SetOverhead is a paid mutator transaction binding the contract method 0x3577afc5.
//
// Solidity: function setOverhead(uint256 _overhead) returns()
func (_Gaspriceproxy *GaspriceproxyTransactor) SetOverhead(opts *bind.TransactOpts, _overhead *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.contract.Transact(opts, "setOverhead", _overhead)
}

// SetOverhead is a paid mutator transaction binding the contract method 0x3577afc5.
//
// Solidity: function setOverhead(uint256 _overhead) returns()
func (_Gaspriceproxy *GaspriceproxySession) SetOverhead(_overhead *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetOverhead(&_Gaspriceproxy.TransactOpts, _overhead)
}

// SetOverhead is a paid mutator transaction binding the contract method 0x3577afc5.
//
// Solidity: function setOverhead(uint256 _overhead) returns()
func (_Gaspriceproxy *GaspriceproxyTransactorSession) SetOverhead(_overhead *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetOverhead(&_Gaspriceproxy.TransactOpts, _overhead)
}

// SetScalar is a paid mutator transaction binding the contract method 0x70465597.
//
// Solidity: function setScalar(uint256 _scalar) returns()
func (_Gaspriceproxy *GaspriceproxyTransactor) SetScalar(opts *bind.TransactOpts, _scalar *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.contract.Transact(opts, "setScalar", _scalar)
}

// SetScalar is a paid mutator transaction binding the contract method 0x70465597.
//
// Solidity: function setScalar(uint256 _scalar) returns()
func (_Gaspriceproxy *GaspriceproxySession) SetScalar(_scalar *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetScalar(&_Gaspriceproxy.TransactOpts, _scalar)
}

// SetScalar is a paid mutator transaction binding the contract method 0x70465597.
//
// Solidity: function setScalar(uint256 _scalar) returns()
func (_Gaspriceproxy *GaspriceproxyTransactorSession) SetScalar(_scalar *big.Int) (*types.Transaction, error) {
	return _Gaspriceproxy.Contract.SetScalar(&_Gaspriceproxy.TransactOpts, _scalar)
}

// GaspriceproxyDecimalsUpdatedIterator is returned from FilterDecimalsUpdated and is used to iterate over the raw logs and unpacked data for DecimalsUpdated events raised by the Gaspriceproxy contract.
type GaspriceproxyDecimalsUpdatedIterator struct {
	Event *GaspriceproxyDecimalsUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GaspriceproxyDecimalsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GaspriceproxyDecimalsUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GaspriceproxyDecimalsUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GaspriceproxyDecimalsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GaspriceproxyDecimalsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GaspriceproxyDecimalsUpdated represents a DecimalsUpdated event raised by the Gaspriceproxy contract.
type GaspriceproxyDecimalsUpdated struct {
	Arg0 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterDecimalsUpdated is a free log retrieval operation binding the contract event 0xd68112a8707e326d08be3656b528c1bcc5bbbfc47f4177e2179b14d8640838c1.
//
// Solidity: event DecimalsUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) FilterDecimalsUpdated(opts *bind.FilterOpts) (*GaspriceproxyDecimalsUpdatedIterator, error) {

	logs, sub, err := _Gaspriceproxy.contract.FilterLogs(opts, "DecimalsUpdated")
	if err != nil {
		return nil, err
	}
	return &GaspriceproxyDecimalsUpdatedIterator{contract: _Gaspriceproxy.contract, event: "DecimalsUpdated", logs: logs, sub: sub}, nil
}

// WatchDecimalsUpdated is a free log subscription operation binding the contract event 0xd68112a8707e326d08be3656b528c1bcc5bbbfc47f4177e2179b14d8640838c1.
//
// Solidity: event DecimalsUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) WatchDecimalsUpdated(opts *bind.WatchOpts, sink chan<- *GaspriceproxyDecimalsUpdated) (event.Subscription, error) {

	logs, sub, err := _Gaspriceproxy.contract.WatchLogs(opts, "DecimalsUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GaspriceproxyDecimalsUpdated)
				if err := _Gaspriceproxy.contract.UnpackLog(event, "DecimalsUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDecimalsUpdated is a log parse operation binding the contract event 0xd68112a8707e326d08be3656b528c1bcc5bbbfc47f4177e2179b14d8640838c1.
//
// Solidity: event DecimalsUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) ParseDecimalsUpdated(log types.Log) (*GaspriceproxyDecimalsUpdated, error) {
	event := new(GaspriceproxyDecimalsUpdated)
	if err := _Gaspriceproxy.contract.UnpackLog(event, "DecimalsUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GaspriceproxyGasPriceUpdatedIterator is returned from FilterGasPriceUpdated and is used to iterate over the raw logs and unpacked data for GasPriceUpdated events raised by the Gaspriceproxy contract.
type GaspriceproxyGasPriceUpdatedIterator struct {
	Event *GaspriceproxyGasPriceUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GaspriceproxyGasPriceUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GaspriceproxyGasPriceUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GaspriceproxyGasPriceUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GaspriceproxyGasPriceUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GaspriceproxyGasPriceUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GaspriceproxyGasPriceUpdated represents a GasPriceUpdated event raised by the Gaspriceproxy contract.
type GaspriceproxyGasPriceUpdated struct {
	Arg0 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterGasPriceUpdated is a free log retrieval operation binding the contract event 0xfcdccc6074c6c42e4bd578aa9870c697dc976a270968452d2b8c8dc369fae396.
//
// Solidity: event GasPriceUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) FilterGasPriceUpdated(opts *bind.FilterOpts) (*GaspriceproxyGasPriceUpdatedIterator, error) {

	logs, sub, err := _Gaspriceproxy.contract.FilterLogs(opts, "GasPriceUpdated")
	if err != nil {
		return nil, err
	}
	return &GaspriceproxyGasPriceUpdatedIterator{contract: _Gaspriceproxy.contract, event: "GasPriceUpdated", logs: logs, sub: sub}, nil
}

// WatchGasPriceUpdated is a free log subscription operation binding the contract event 0xfcdccc6074c6c42e4bd578aa9870c697dc976a270968452d2b8c8dc369fae396.
//
// Solidity: event GasPriceUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) WatchGasPriceUpdated(opts *bind.WatchOpts, sink chan<- *GaspriceproxyGasPriceUpdated) (event.Subscription, error) {

	logs, sub, err := _Gaspriceproxy.contract.WatchLogs(opts, "GasPriceUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GaspriceproxyGasPriceUpdated)
				if err := _Gaspriceproxy.contract.UnpackLog(event, "GasPriceUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseGasPriceUpdated is a log parse operation binding the contract event 0xfcdccc6074c6c42e4bd578aa9870c697dc976a270968452d2b8c8dc369fae396.
//
// Solidity: event GasPriceUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) ParseGasPriceUpdated(log types.Log) (*GaspriceproxyGasPriceUpdated, error) {
	event := new(GaspriceproxyGasPriceUpdated)
	if err := _Gaspriceproxy.contract.UnpackLog(event, "GasPriceUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GaspriceproxyL1BaseFeeUpdatedIterator is returned from FilterL1BaseFeeUpdated and is used to iterate over the raw logs and unpacked data for L1BaseFeeUpdated events raised by the Gaspriceproxy contract.
type GaspriceproxyL1BaseFeeUpdatedIterator struct {
	Event *GaspriceproxyL1BaseFeeUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GaspriceproxyL1BaseFeeUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GaspriceproxyL1BaseFeeUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GaspriceproxyL1BaseFeeUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GaspriceproxyL1BaseFeeUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GaspriceproxyL1BaseFeeUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GaspriceproxyL1BaseFeeUpdated represents a L1BaseFeeUpdated event raised by the Gaspriceproxy contract.
type GaspriceproxyL1BaseFeeUpdated struct {
	Arg0 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterL1BaseFeeUpdated is a free log retrieval operation binding the contract event 0x351fb23757bb5ea0546c85b7996ddd7155f96b939ebaa5ff7bc49c75f27f2c44.
//
// Solidity: event L1BaseFeeUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) FilterL1BaseFeeUpdated(opts *bind.FilterOpts) (*GaspriceproxyL1BaseFeeUpdatedIterator, error) {

	logs, sub, err := _Gaspriceproxy.contract.FilterLogs(opts, "L1BaseFeeUpdated")
	if err != nil {
		return nil, err
	}
	return &GaspriceproxyL1BaseFeeUpdatedIterator{contract: _Gaspriceproxy.contract, event: "L1BaseFeeUpdated", logs: logs, sub: sub}, nil
}

// WatchL1BaseFeeUpdated is a free log subscription operation binding the contract event 0x351fb23757bb5ea0546c85b7996ddd7155f96b939ebaa5ff7bc49c75f27f2c44.
//
// Solidity: event L1BaseFeeUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) WatchL1BaseFeeUpdated(opts *bind.WatchOpts, sink chan<- *GaspriceproxyL1BaseFeeUpdated) (event.Subscription, error) {

	logs, sub, err := _Gaspriceproxy.contract.WatchLogs(opts, "L1BaseFeeUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GaspriceproxyL1BaseFeeUpdated)
				if err := _Gaspriceproxy.contract.UnpackLog(event, "L1BaseFeeUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseL1BaseFeeUpdated is a log parse operation binding the contract event 0x351fb23757bb5ea0546c85b7996ddd7155f96b939ebaa5ff7bc49c75f27f2c44.
//
// Solidity: event L1BaseFeeUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) ParseL1BaseFeeUpdated(log types.Log) (*GaspriceproxyL1BaseFeeUpdated, error) {
	event := new(GaspriceproxyL1BaseFeeUpdated)
	if err := _Gaspriceproxy.contract.UnpackLog(event, "L1BaseFeeUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GaspriceproxyOverheadUpdatedIterator is returned from FilterOverheadUpdated and is used to iterate over the raw logs and unpacked data for OverheadUpdated events raised by the Gaspriceproxy contract.
type GaspriceproxyOverheadUpdatedIterator struct {
	Event *GaspriceproxyOverheadUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GaspriceproxyOverheadUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GaspriceproxyOverheadUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GaspriceproxyOverheadUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GaspriceproxyOverheadUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GaspriceproxyOverheadUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GaspriceproxyOverheadUpdated represents a OverheadUpdated event raised by the Gaspriceproxy contract.
type GaspriceproxyOverheadUpdated struct {
	Arg0 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterOverheadUpdated is a free log retrieval operation binding the contract event 0x32740b35c0ea213650f60d44366b4fb211c9033b50714e4a1d34e65d5beb9bb4.
//
// Solidity: event OverheadUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) FilterOverheadUpdated(opts *bind.FilterOpts) (*GaspriceproxyOverheadUpdatedIterator, error) {

	logs, sub, err := _Gaspriceproxy.contract.FilterLogs(opts, "OverheadUpdated")
	if err != nil {
		return nil, err
	}
	return &GaspriceproxyOverheadUpdatedIterator{contract: _Gaspriceproxy.contract, event: "OverheadUpdated", logs: logs, sub: sub}, nil
}

// WatchOverheadUpdated is a free log subscription operation binding the contract event 0x32740b35c0ea213650f60d44366b4fb211c9033b50714e4a1d34e65d5beb9bb4.
//
// Solidity: event OverheadUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) WatchOverheadUpdated(opts *bind.WatchOpts, sink chan<- *GaspriceproxyOverheadUpdated) (event.Subscription, error) {

	logs, sub, err := _Gaspriceproxy.contract.WatchLogs(opts, "OverheadUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GaspriceproxyOverheadUpdated)
				if err := _Gaspriceproxy.contract.UnpackLog(event, "OverheadUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOverheadUpdated is a log parse operation binding the contract event 0x32740b35c0ea213650f60d44366b4fb211c9033b50714e4a1d34e65d5beb9bb4.
//
// Solidity: event OverheadUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) ParseOverheadUpdated(log types.Log) (*GaspriceproxyOverheadUpdated, error) {
	event := new(GaspriceproxyOverheadUpdated)
	if err := _Gaspriceproxy.contract.UnpackLog(event, "OverheadUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GaspriceproxyScalarUpdatedIterator is returned from FilterScalarUpdated and is used to iterate over the raw logs and unpacked data for ScalarUpdated events raised by the Gaspriceproxy contract.
type GaspriceproxyScalarUpdatedIterator struct {
	Event *GaspriceproxyScalarUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GaspriceproxyScalarUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GaspriceproxyScalarUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GaspriceproxyScalarUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GaspriceproxyScalarUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GaspriceproxyScalarUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GaspriceproxyScalarUpdated represents a ScalarUpdated event raised by the Gaspriceproxy contract.
type GaspriceproxyScalarUpdated struct {
	Arg0 *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterScalarUpdated is a free log retrieval operation binding the contract event 0x3336cd9708eaf2769a0f0dc0679f30e80f15dcd88d1921b5a16858e8b85c591a.
//
// Solidity: event ScalarUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) FilterScalarUpdated(opts *bind.FilterOpts) (*GaspriceproxyScalarUpdatedIterator, error) {

	logs, sub, err := _Gaspriceproxy.contract.FilterLogs(opts, "ScalarUpdated")
	if err != nil {
		return nil, err
	}
	return &GaspriceproxyScalarUpdatedIterator{contract: _Gaspriceproxy.contract, event: "ScalarUpdated", logs: logs, sub: sub}, nil
}

// WatchScalarUpdated is a free log subscription operation binding the contract event 0x3336cd9708eaf2769a0f0dc0679f30e80f15dcd88d1921b5a16858e8b85c591a.
//
// Solidity: event ScalarUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) WatchScalarUpdated(opts *bind.WatchOpts, sink chan<- *GaspriceproxyScalarUpdated) (event.Subscription, error) {

	logs, sub, err := _Gaspriceproxy.contract.WatchLogs(opts, "ScalarUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GaspriceproxyScalarUpdated)
				if err := _Gaspriceproxy.contract.UnpackLog(event, "ScalarUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseScalarUpdated is a log parse operation binding the contract event 0x3336cd9708eaf2769a0f0dc0679f30e80f15dcd88d1921b5a16858e8b85c591a.
//
// Solidity: event ScalarUpdated(uint256 arg0)
func (_Gaspriceproxy *GaspriceproxyFilterer) ParseScalarUpdated(log types.Log) (*GaspriceproxyScalarUpdated, error) {
	event := new(GaspriceproxyScalarUpdated)
	if err := _Gaspriceproxy.contract.UnpackLog(event, "ScalarUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
