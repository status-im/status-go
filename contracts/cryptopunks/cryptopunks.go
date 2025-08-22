// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package cryptopunks

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

// CryptoPunksMetaData contains all meta data concerning the CryptoPunks contract.
var CryptoPunksMetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"punksOfferedForSale\",\"outputs\":[{\"name\":\"isForSale\",\"type\":\"bool\"},{\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"name\":\"seller\",\"type\":\"address\"},{\"name\":\"minValue\",\"type\":\"uint256\"},{\"name\":\"onlySellTo\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"enterBidForPunk\",\"outputs\":[],\"payable\":true,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"name\":\"minPrice\",\"type\":\"uint256\"}],\"name\":\"acceptBidForPunk\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"addresses\",\"type\":\"address[]\"},{\"name\":\"indices\",\"type\":\"uint256[]\"}],\"name\":\"setInitialOwners\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"imageHash\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"nextPunkIndexToAssign\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"punkIndexToAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"standard\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"punkBids\",\"outputs\":[{\"name\":\"hasBid\",\"type\":\"bool\"},{\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"name\":\"bidder\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"allInitialOwnersAssigned\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"allPunksAssigned\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"buyPunk\",\"outputs\":[],\"payable\":true,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"transferPunk\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"withdrawBidForPunk\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"setInitialOwner\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"name\":\"minSalePriceInWei\",\"type\":\"uint256\"},{\"name\":\"toAddress\",\"type\":\"address\"}],\"name\":\"offerPunkForSaleToAddress\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"punksRemainingToAssign\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"name\":\"minSalePriceInWei\",\"type\":\"uint256\"}],\"name\":\"offerPunkForSale\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"getPunk\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"pendingWithdrawals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"punkNoLongerForSale\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"inputs\":[],\"payable\":true,\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"Assign\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"PunkTransfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"minValue\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"toAddress\",\"type\":\"address\"}],\"name\":\"PunkOffered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"fromAddress\",\"type\":\"address\"}],\"name\":\"PunkBidEntered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"fromAddress\",\"type\":\"address\"}],\"name\":\"PunkBidWithdrawn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"punkIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"fromAddress\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"toAddress\",\"type\":\"address\"}],\"name\":\"PunkBought\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"punkIndex\",\"type\":\"uint256\"}],\"name\":\"PunkNoLongerForSale\",\"type\":\"event\"}]",
}

// CryptoPunksABI is the input ABI used to generate the binding from.
// Deprecated: Use CryptoPunksMetaData.ABI instead.
var CryptoPunksABI = CryptoPunksMetaData.ABI

// CryptoPunks is an auto generated Go binding around an Ethereum contract.
type CryptoPunks struct {
	CryptoPunksCaller     // Read-only binding to the contract
	CryptoPunksTransactor // Write-only binding to the contract
	CryptoPunksFilterer   // Log filterer for contract events
}

// CryptoPunksCaller is an auto generated read-only Go binding around an Ethereum contract.
type CryptoPunksCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptoPunksTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CryptoPunksTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptoPunksFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CryptoPunksFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptoPunksSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CryptoPunksSession struct {
	Contract     *CryptoPunks      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CryptoPunksCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CryptoPunksCallerSession struct {
	Contract *CryptoPunksCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// CryptoPunksTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CryptoPunksTransactorSession struct {
	Contract     *CryptoPunksTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// CryptoPunksRaw is an auto generated low-level Go binding around an Ethereum contract.
type CryptoPunksRaw struct {
	Contract *CryptoPunks // Generic contract binding to access the raw methods on
}

// CryptoPunksCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CryptoPunksCallerRaw struct {
	Contract *CryptoPunksCaller // Generic read-only contract binding to access the raw methods on
}

// CryptoPunksTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CryptoPunksTransactorRaw struct {
	Contract *CryptoPunksTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCryptoPunks creates a new instance of CryptoPunks, bound to a specific deployed contract.
func NewCryptoPunks(address common.Address, backend bind.ContractBackend) (*CryptoPunks, error) {
	contract, err := bindCryptoPunks(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CryptoPunks{CryptoPunksCaller: CryptoPunksCaller{contract: contract}, CryptoPunksTransactor: CryptoPunksTransactor{contract: contract}, CryptoPunksFilterer: CryptoPunksFilterer{contract: contract}}, nil
}

// NewCryptoPunksCaller creates a new read-only instance of CryptoPunks, bound to a specific deployed contract.
func NewCryptoPunksCaller(address common.Address, caller bind.ContractCaller) (*CryptoPunksCaller, error) {
	contract, err := bindCryptoPunks(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksCaller{contract: contract}, nil
}

// NewCryptoPunksTransactor creates a new write-only instance of CryptoPunks, bound to a specific deployed contract.
func NewCryptoPunksTransactor(address common.Address, transactor bind.ContractTransactor) (*CryptoPunksTransactor, error) {
	contract, err := bindCryptoPunks(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksTransactor{contract: contract}, nil
}

// NewCryptoPunksFilterer creates a new log filterer instance of CryptoPunks, bound to a specific deployed contract.
func NewCryptoPunksFilterer(address common.Address, filterer bind.ContractFilterer) (*CryptoPunksFilterer, error) {
	contract, err := bindCryptoPunks(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksFilterer{contract: contract}, nil
}

// bindCryptoPunks binds a generic wrapper to an already deployed contract.
func bindCryptoPunks(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CryptoPunksMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CryptoPunks *CryptoPunksRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CryptoPunks.Contract.CryptoPunksCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CryptoPunks *CryptoPunksRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoPunks.Contract.CryptoPunksTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CryptoPunks *CryptoPunksRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CryptoPunks.Contract.CryptoPunksTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CryptoPunks *CryptoPunksCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CryptoPunks.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CryptoPunks *CryptoPunksTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoPunks.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CryptoPunks *CryptoPunksTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CryptoPunks.Contract.contract.Transact(opts, method, params...)
}

// AllPunksAssigned is a free data retrieval call binding the contract method 0x8126c38a.
//
// Solidity: function allPunksAssigned() returns(bool)
func (_CryptoPunks *CryptoPunksCaller) AllPunksAssigned(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "allPunksAssigned")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllPunksAssigned is a free data retrieval call binding the contract method 0x8126c38a.
//
// Solidity: function allPunksAssigned() returns(bool)
func (_CryptoPunks *CryptoPunksSession) AllPunksAssigned() (bool, error) {
	return _CryptoPunks.Contract.AllPunksAssigned(&_CryptoPunks.CallOpts)
}

// AllPunksAssigned is a free data retrieval call binding the contract method 0x8126c38a.
//
// Solidity: function allPunksAssigned() returns(bool)
func (_CryptoPunks *CryptoPunksCallerSession) AllPunksAssigned() (bool, error) {
	return _CryptoPunks.Contract.AllPunksAssigned(&_CryptoPunks.CallOpts)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) returns(uint256)
func (_CryptoPunks *CryptoPunksCaller) BalanceOf(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "balanceOf", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) returns(uint256)
func (_CryptoPunks *CryptoPunksSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _CryptoPunks.Contract.BalanceOf(&_CryptoPunks.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) returns(uint256)
func (_CryptoPunks *CryptoPunksCallerSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _CryptoPunks.Contract.BalanceOf(&_CryptoPunks.CallOpts, arg0)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() returns(uint8)
func (_CryptoPunks *CryptoPunksCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() returns(uint8)
func (_CryptoPunks *CryptoPunksSession) Decimals() (uint8, error) {
	return _CryptoPunks.Contract.Decimals(&_CryptoPunks.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() returns(uint8)
func (_CryptoPunks *CryptoPunksCallerSession) Decimals() (uint8, error) {
	return _CryptoPunks.Contract.Decimals(&_CryptoPunks.CallOpts)
}

// ImageHash is a free data retrieval call binding the contract method 0x51605d80.
//
// Solidity: function imageHash() returns(string)
func (_CryptoPunks *CryptoPunksCaller) ImageHash(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "imageHash")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ImageHash is a free data retrieval call binding the contract method 0x51605d80.
//
// Solidity: function imageHash() returns(string)
func (_CryptoPunks *CryptoPunksSession) ImageHash() (string, error) {
	return _CryptoPunks.Contract.ImageHash(&_CryptoPunks.CallOpts)
}

// ImageHash is a free data retrieval call binding the contract method 0x51605d80.
//
// Solidity: function imageHash() returns(string)
func (_CryptoPunks *CryptoPunksCallerSession) ImageHash() (string, error) {
	return _CryptoPunks.Contract.ImageHash(&_CryptoPunks.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() returns(string)
func (_CryptoPunks *CryptoPunksCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() returns(string)
func (_CryptoPunks *CryptoPunksSession) Name() (string, error) {
	return _CryptoPunks.Contract.Name(&_CryptoPunks.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() returns(string)
func (_CryptoPunks *CryptoPunksCallerSession) Name() (string, error) {
	return _CryptoPunks.Contract.Name(&_CryptoPunks.CallOpts)
}

// NextPunkIndexToAssign is a free data retrieval call binding the contract method 0x52f29a25.
//
// Solidity: function nextPunkIndexToAssign() returns(uint256)
func (_CryptoPunks *CryptoPunksCaller) NextPunkIndexToAssign(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "nextPunkIndexToAssign")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextPunkIndexToAssign is a free data retrieval call binding the contract method 0x52f29a25.
//
// Solidity: function nextPunkIndexToAssign() returns(uint256)
func (_CryptoPunks *CryptoPunksSession) NextPunkIndexToAssign() (*big.Int, error) {
	return _CryptoPunks.Contract.NextPunkIndexToAssign(&_CryptoPunks.CallOpts)
}

// NextPunkIndexToAssign is a free data retrieval call binding the contract method 0x52f29a25.
//
// Solidity: function nextPunkIndexToAssign() returns(uint256)
func (_CryptoPunks *CryptoPunksCallerSession) NextPunkIndexToAssign() (*big.Int, error) {
	return _CryptoPunks.Contract.NextPunkIndexToAssign(&_CryptoPunks.CallOpts)
}

// PendingWithdrawals is a free data retrieval call binding the contract method 0xf3f43703.
//
// Solidity: function pendingWithdrawals(address ) returns(uint256)
func (_CryptoPunks *CryptoPunksCaller) PendingWithdrawals(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "pendingWithdrawals", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PendingWithdrawals is a free data retrieval call binding the contract method 0xf3f43703.
//
// Solidity: function pendingWithdrawals(address ) returns(uint256)
func (_CryptoPunks *CryptoPunksSession) PendingWithdrawals(arg0 common.Address) (*big.Int, error) {
	return _CryptoPunks.Contract.PendingWithdrawals(&_CryptoPunks.CallOpts, arg0)
}

// PendingWithdrawals is a free data retrieval call binding the contract method 0xf3f43703.
//
// Solidity: function pendingWithdrawals(address ) returns(uint256)
func (_CryptoPunks *CryptoPunksCallerSession) PendingWithdrawals(arg0 common.Address) (*big.Int, error) {
	return _CryptoPunks.Contract.PendingWithdrawals(&_CryptoPunks.CallOpts, arg0)
}

// PunkBids is a free data retrieval call binding the contract method 0x6e743fa9.
//
// Solidity: function punkBids(uint256 ) returns(bool hasBid, uint256 punkIndex, address bidder, uint256 value)
func (_CryptoPunks *CryptoPunksCaller) PunkBids(opts *bind.CallOpts, arg0 *big.Int) (struct {
	HasBid    bool
	PunkIndex *big.Int
	Bidder    common.Address
	Value     *big.Int
}, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "punkBids", arg0)

	outstruct := new(struct {
		HasBid    bool
		PunkIndex *big.Int
		Bidder    common.Address
		Value     *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.HasBid = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.PunkIndex = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Bidder = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.Value = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PunkBids is a free data retrieval call binding the contract method 0x6e743fa9.
//
// Solidity: function punkBids(uint256 ) returns(bool hasBid, uint256 punkIndex, address bidder, uint256 value)
func (_CryptoPunks *CryptoPunksSession) PunkBids(arg0 *big.Int) (struct {
	HasBid    bool
	PunkIndex *big.Int
	Bidder    common.Address
	Value     *big.Int
}, error) {
	return _CryptoPunks.Contract.PunkBids(&_CryptoPunks.CallOpts, arg0)
}

// PunkBids is a free data retrieval call binding the contract method 0x6e743fa9.
//
// Solidity: function punkBids(uint256 ) returns(bool hasBid, uint256 punkIndex, address bidder, uint256 value)
func (_CryptoPunks *CryptoPunksCallerSession) PunkBids(arg0 *big.Int) (struct {
	HasBid    bool
	PunkIndex *big.Int
	Bidder    common.Address
	Value     *big.Int
}, error) {
	return _CryptoPunks.Contract.PunkBids(&_CryptoPunks.CallOpts, arg0)
}

// PunkIndexToAddress is a free data retrieval call binding the contract method 0x58178168.
//
// Solidity: function punkIndexToAddress(uint256 ) returns(address)
func (_CryptoPunks *CryptoPunksCaller) PunkIndexToAddress(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "punkIndexToAddress", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PunkIndexToAddress is a free data retrieval call binding the contract method 0x58178168.
//
// Solidity: function punkIndexToAddress(uint256 ) returns(address)
func (_CryptoPunks *CryptoPunksSession) PunkIndexToAddress(arg0 *big.Int) (common.Address, error) {
	return _CryptoPunks.Contract.PunkIndexToAddress(&_CryptoPunks.CallOpts, arg0)
}

// PunkIndexToAddress is a free data retrieval call binding the contract method 0x58178168.
//
// Solidity: function punkIndexToAddress(uint256 ) returns(address)
func (_CryptoPunks *CryptoPunksCallerSession) PunkIndexToAddress(arg0 *big.Int) (common.Address, error) {
	return _CryptoPunks.Contract.PunkIndexToAddress(&_CryptoPunks.CallOpts, arg0)
}

// PunksOfferedForSale is a free data retrieval call binding the contract method 0x088f11f3.
//
// Solidity: function punksOfferedForSale(uint256 ) returns(bool isForSale, uint256 punkIndex, address seller, uint256 minValue, address onlySellTo)
func (_CryptoPunks *CryptoPunksCaller) PunksOfferedForSale(opts *bind.CallOpts, arg0 *big.Int) (struct {
	IsForSale  bool
	PunkIndex  *big.Int
	Seller     common.Address
	MinValue   *big.Int
	OnlySellTo common.Address
}, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "punksOfferedForSale", arg0)

	outstruct := new(struct {
		IsForSale  bool
		PunkIndex  *big.Int
		Seller     common.Address
		MinValue   *big.Int
		OnlySellTo common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.IsForSale = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.PunkIndex = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Seller = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.MinValue = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.OnlySellTo = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// PunksOfferedForSale is a free data retrieval call binding the contract method 0x088f11f3.
//
// Solidity: function punksOfferedForSale(uint256 ) returns(bool isForSale, uint256 punkIndex, address seller, uint256 minValue, address onlySellTo)
func (_CryptoPunks *CryptoPunksSession) PunksOfferedForSale(arg0 *big.Int) (struct {
	IsForSale  bool
	PunkIndex  *big.Int
	Seller     common.Address
	MinValue   *big.Int
	OnlySellTo common.Address
}, error) {
	return _CryptoPunks.Contract.PunksOfferedForSale(&_CryptoPunks.CallOpts, arg0)
}

// PunksOfferedForSale is a free data retrieval call binding the contract method 0x088f11f3.
//
// Solidity: function punksOfferedForSale(uint256 ) returns(bool isForSale, uint256 punkIndex, address seller, uint256 minValue, address onlySellTo)
func (_CryptoPunks *CryptoPunksCallerSession) PunksOfferedForSale(arg0 *big.Int) (struct {
	IsForSale  bool
	PunkIndex  *big.Int
	Seller     common.Address
	MinValue   *big.Int
	OnlySellTo common.Address
}, error) {
	return _CryptoPunks.Contract.PunksOfferedForSale(&_CryptoPunks.CallOpts, arg0)
}

// PunksRemainingToAssign is a free data retrieval call binding the contract method 0xc0d6ce63.
//
// Solidity: function punksRemainingToAssign() returns(uint256)
func (_CryptoPunks *CryptoPunksCaller) PunksRemainingToAssign(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "punksRemainingToAssign")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PunksRemainingToAssign is a free data retrieval call binding the contract method 0xc0d6ce63.
//
// Solidity: function punksRemainingToAssign() returns(uint256)
func (_CryptoPunks *CryptoPunksSession) PunksRemainingToAssign() (*big.Int, error) {
	return _CryptoPunks.Contract.PunksRemainingToAssign(&_CryptoPunks.CallOpts)
}

// PunksRemainingToAssign is a free data retrieval call binding the contract method 0xc0d6ce63.
//
// Solidity: function punksRemainingToAssign() returns(uint256)
func (_CryptoPunks *CryptoPunksCallerSession) PunksRemainingToAssign() (*big.Int, error) {
	return _CryptoPunks.Contract.PunksRemainingToAssign(&_CryptoPunks.CallOpts)
}

// Standard is a free data retrieval call binding the contract method 0x5a3b7e42.
//
// Solidity: function standard() returns(string)
func (_CryptoPunks *CryptoPunksCaller) Standard(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "standard")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Standard is a free data retrieval call binding the contract method 0x5a3b7e42.
//
// Solidity: function standard() returns(string)
func (_CryptoPunks *CryptoPunksSession) Standard() (string, error) {
	return _CryptoPunks.Contract.Standard(&_CryptoPunks.CallOpts)
}

// Standard is a free data retrieval call binding the contract method 0x5a3b7e42.
//
// Solidity: function standard() returns(string)
func (_CryptoPunks *CryptoPunksCallerSession) Standard() (string, error) {
	return _CryptoPunks.Contract.Standard(&_CryptoPunks.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() returns(string)
func (_CryptoPunks *CryptoPunksCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() returns(string)
func (_CryptoPunks *CryptoPunksSession) Symbol() (string, error) {
	return _CryptoPunks.Contract.Symbol(&_CryptoPunks.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() returns(string)
func (_CryptoPunks *CryptoPunksCallerSession) Symbol() (string, error) {
	return _CryptoPunks.Contract.Symbol(&_CryptoPunks.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() returns(uint256)
func (_CryptoPunks *CryptoPunksCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoPunks.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() returns(uint256)
func (_CryptoPunks *CryptoPunksSession) TotalSupply() (*big.Int, error) {
	return _CryptoPunks.Contract.TotalSupply(&_CryptoPunks.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() returns(uint256)
func (_CryptoPunks *CryptoPunksCallerSession) TotalSupply() (*big.Int, error) {
	return _CryptoPunks.Contract.TotalSupply(&_CryptoPunks.CallOpts)
}

// AcceptBidForPunk is a paid mutator transaction binding the contract method 0x23165b75.
//
// Solidity: function acceptBidForPunk(uint256 punkIndex, uint256 minPrice) returns()
func (_CryptoPunks *CryptoPunksTransactor) AcceptBidForPunk(opts *bind.TransactOpts, punkIndex *big.Int, minPrice *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "acceptBidForPunk", punkIndex, minPrice)
}

// AcceptBidForPunk is a paid mutator transaction binding the contract method 0x23165b75.
//
// Solidity: function acceptBidForPunk(uint256 punkIndex, uint256 minPrice) returns()
func (_CryptoPunks *CryptoPunksSession) AcceptBidForPunk(punkIndex *big.Int, minPrice *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.AcceptBidForPunk(&_CryptoPunks.TransactOpts, punkIndex, minPrice)
}

// AcceptBidForPunk is a paid mutator transaction binding the contract method 0x23165b75.
//
// Solidity: function acceptBidForPunk(uint256 punkIndex, uint256 minPrice) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) AcceptBidForPunk(punkIndex *big.Int, minPrice *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.AcceptBidForPunk(&_CryptoPunks.TransactOpts, punkIndex, minPrice)
}

// AllInitialOwnersAssigned is a paid mutator transaction binding the contract method 0x7ecedac9.
//
// Solidity: function allInitialOwnersAssigned() returns()
func (_CryptoPunks *CryptoPunksTransactor) AllInitialOwnersAssigned(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "allInitialOwnersAssigned")
}

// AllInitialOwnersAssigned is a paid mutator transaction binding the contract method 0x7ecedac9.
//
// Solidity: function allInitialOwnersAssigned() returns()
func (_CryptoPunks *CryptoPunksSession) AllInitialOwnersAssigned() (*types.Transaction, error) {
	return _CryptoPunks.Contract.AllInitialOwnersAssigned(&_CryptoPunks.TransactOpts)
}

// AllInitialOwnersAssigned is a paid mutator transaction binding the contract method 0x7ecedac9.
//
// Solidity: function allInitialOwnersAssigned() returns()
func (_CryptoPunks *CryptoPunksTransactorSession) AllInitialOwnersAssigned() (*types.Transaction, error) {
	return _CryptoPunks.Contract.AllInitialOwnersAssigned(&_CryptoPunks.TransactOpts)
}

// BuyPunk is a paid mutator transaction binding the contract method 0x8264fe98.
//
// Solidity: function buyPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactor) BuyPunk(opts *bind.TransactOpts, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "buyPunk", punkIndex)
}

// BuyPunk is a paid mutator transaction binding the contract method 0x8264fe98.
//
// Solidity: function buyPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksSession) BuyPunk(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.BuyPunk(&_CryptoPunks.TransactOpts, punkIndex)
}

// BuyPunk is a paid mutator transaction binding the contract method 0x8264fe98.
//
// Solidity: function buyPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) BuyPunk(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.BuyPunk(&_CryptoPunks.TransactOpts, punkIndex)
}

// EnterBidForPunk is a paid mutator transaction binding the contract method 0x091dbfd2.
//
// Solidity: function enterBidForPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactor) EnterBidForPunk(opts *bind.TransactOpts, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "enterBidForPunk", punkIndex)
}

// EnterBidForPunk is a paid mutator transaction binding the contract method 0x091dbfd2.
//
// Solidity: function enterBidForPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksSession) EnterBidForPunk(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.EnterBidForPunk(&_CryptoPunks.TransactOpts, punkIndex)
}

// EnterBidForPunk is a paid mutator transaction binding the contract method 0x091dbfd2.
//
// Solidity: function enterBidForPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) EnterBidForPunk(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.EnterBidForPunk(&_CryptoPunks.TransactOpts, punkIndex)
}

// GetPunk is a paid mutator transaction binding the contract method 0xc81d1d5b.
//
// Solidity: function getPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactor) GetPunk(opts *bind.TransactOpts, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "getPunk", punkIndex)
}

// GetPunk is a paid mutator transaction binding the contract method 0xc81d1d5b.
//
// Solidity: function getPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksSession) GetPunk(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.GetPunk(&_CryptoPunks.TransactOpts, punkIndex)
}

// GetPunk is a paid mutator transaction binding the contract method 0xc81d1d5b.
//
// Solidity: function getPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) GetPunk(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.GetPunk(&_CryptoPunks.TransactOpts, punkIndex)
}

// OfferPunkForSale is a paid mutator transaction binding the contract method 0xc44193c3.
//
// Solidity: function offerPunkForSale(uint256 punkIndex, uint256 minSalePriceInWei) returns()
func (_CryptoPunks *CryptoPunksTransactor) OfferPunkForSale(opts *bind.TransactOpts, punkIndex *big.Int, minSalePriceInWei *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "offerPunkForSale", punkIndex, minSalePriceInWei)
}

// OfferPunkForSale is a paid mutator transaction binding the contract method 0xc44193c3.
//
// Solidity: function offerPunkForSale(uint256 punkIndex, uint256 minSalePriceInWei) returns()
func (_CryptoPunks *CryptoPunksSession) OfferPunkForSale(punkIndex *big.Int, minSalePriceInWei *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.OfferPunkForSale(&_CryptoPunks.TransactOpts, punkIndex, minSalePriceInWei)
}

// OfferPunkForSale is a paid mutator transaction binding the contract method 0xc44193c3.
//
// Solidity: function offerPunkForSale(uint256 punkIndex, uint256 minSalePriceInWei) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) OfferPunkForSale(punkIndex *big.Int, minSalePriceInWei *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.OfferPunkForSale(&_CryptoPunks.TransactOpts, punkIndex, minSalePriceInWei)
}

// OfferPunkForSaleToAddress is a paid mutator transaction binding the contract method 0xbf31196f.
//
// Solidity: function offerPunkForSaleToAddress(uint256 punkIndex, uint256 minSalePriceInWei, address toAddress) returns()
func (_CryptoPunks *CryptoPunksTransactor) OfferPunkForSaleToAddress(opts *bind.TransactOpts, punkIndex *big.Int, minSalePriceInWei *big.Int, toAddress common.Address) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "offerPunkForSaleToAddress", punkIndex, minSalePriceInWei, toAddress)
}

// OfferPunkForSaleToAddress is a paid mutator transaction binding the contract method 0xbf31196f.
//
// Solidity: function offerPunkForSaleToAddress(uint256 punkIndex, uint256 minSalePriceInWei, address toAddress) returns()
func (_CryptoPunks *CryptoPunksSession) OfferPunkForSaleToAddress(punkIndex *big.Int, minSalePriceInWei *big.Int, toAddress common.Address) (*types.Transaction, error) {
	return _CryptoPunks.Contract.OfferPunkForSaleToAddress(&_CryptoPunks.TransactOpts, punkIndex, minSalePriceInWei, toAddress)
}

// OfferPunkForSaleToAddress is a paid mutator transaction binding the contract method 0xbf31196f.
//
// Solidity: function offerPunkForSaleToAddress(uint256 punkIndex, uint256 minSalePriceInWei, address toAddress) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) OfferPunkForSaleToAddress(punkIndex *big.Int, minSalePriceInWei *big.Int, toAddress common.Address) (*types.Transaction, error) {
	return _CryptoPunks.Contract.OfferPunkForSaleToAddress(&_CryptoPunks.TransactOpts, punkIndex, minSalePriceInWei, toAddress)
}

// PunkNoLongerForSale is a paid mutator transaction binding the contract method 0xf6eeff1e.
//
// Solidity: function punkNoLongerForSale(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactor) PunkNoLongerForSale(opts *bind.TransactOpts, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "punkNoLongerForSale", punkIndex)
}

// PunkNoLongerForSale is a paid mutator transaction binding the contract method 0xf6eeff1e.
//
// Solidity: function punkNoLongerForSale(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksSession) PunkNoLongerForSale(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.PunkNoLongerForSale(&_CryptoPunks.TransactOpts, punkIndex)
}

// PunkNoLongerForSale is a paid mutator transaction binding the contract method 0xf6eeff1e.
//
// Solidity: function punkNoLongerForSale(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) PunkNoLongerForSale(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.PunkNoLongerForSale(&_CryptoPunks.TransactOpts, punkIndex)
}

// SetInitialOwner is a paid mutator transaction binding the contract method 0xa75a9049.
//
// Solidity: function setInitialOwner(address to, uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactor) SetInitialOwner(opts *bind.TransactOpts, to common.Address, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "setInitialOwner", to, punkIndex)
}

// SetInitialOwner is a paid mutator transaction binding the contract method 0xa75a9049.
//
// Solidity: function setInitialOwner(address to, uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksSession) SetInitialOwner(to common.Address, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.SetInitialOwner(&_CryptoPunks.TransactOpts, to, punkIndex)
}

// SetInitialOwner is a paid mutator transaction binding the contract method 0xa75a9049.
//
// Solidity: function setInitialOwner(address to, uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) SetInitialOwner(to common.Address, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.SetInitialOwner(&_CryptoPunks.TransactOpts, to, punkIndex)
}

// SetInitialOwners is a paid mutator transaction binding the contract method 0x39c5dde6.
//
// Solidity: function setInitialOwners(address[] addresses, uint256[] indices) returns()
func (_CryptoPunks *CryptoPunksTransactor) SetInitialOwners(opts *bind.TransactOpts, addresses []common.Address, indices []*big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "setInitialOwners", addresses, indices)
}

// SetInitialOwners is a paid mutator transaction binding the contract method 0x39c5dde6.
//
// Solidity: function setInitialOwners(address[] addresses, uint256[] indices) returns()
func (_CryptoPunks *CryptoPunksSession) SetInitialOwners(addresses []common.Address, indices []*big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.SetInitialOwners(&_CryptoPunks.TransactOpts, addresses, indices)
}

// SetInitialOwners is a paid mutator transaction binding the contract method 0x39c5dde6.
//
// Solidity: function setInitialOwners(address[] addresses, uint256[] indices) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) SetInitialOwners(addresses []common.Address, indices []*big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.SetInitialOwners(&_CryptoPunks.TransactOpts, addresses, indices)
}

// TransferPunk is a paid mutator transaction binding the contract method 0x8b72a2ec.
//
// Solidity: function transferPunk(address to, uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactor) TransferPunk(opts *bind.TransactOpts, to common.Address, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "transferPunk", to, punkIndex)
}

// TransferPunk is a paid mutator transaction binding the contract method 0x8b72a2ec.
//
// Solidity: function transferPunk(address to, uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksSession) TransferPunk(to common.Address, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.TransferPunk(&_CryptoPunks.TransactOpts, to, punkIndex)
}

// TransferPunk is a paid mutator transaction binding the contract method 0x8b72a2ec.
//
// Solidity: function transferPunk(address to, uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) TransferPunk(to common.Address, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.TransferPunk(&_CryptoPunks.TransactOpts, to, punkIndex)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_CryptoPunks *CryptoPunksTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_CryptoPunks *CryptoPunksSession) Withdraw() (*types.Transaction, error) {
	return _CryptoPunks.Contract.Withdraw(&_CryptoPunks.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_CryptoPunks *CryptoPunksTransactorSession) Withdraw() (*types.Transaction, error) {
	return _CryptoPunks.Contract.Withdraw(&_CryptoPunks.TransactOpts)
}

// WithdrawBidForPunk is a paid mutator transaction binding the contract method 0x979bc638.
//
// Solidity: function withdrawBidForPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactor) WithdrawBidForPunk(opts *bind.TransactOpts, punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.contract.Transact(opts, "withdrawBidForPunk", punkIndex)
}

// WithdrawBidForPunk is a paid mutator transaction binding the contract method 0x979bc638.
//
// Solidity: function withdrawBidForPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksSession) WithdrawBidForPunk(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.WithdrawBidForPunk(&_CryptoPunks.TransactOpts, punkIndex)
}

// WithdrawBidForPunk is a paid mutator transaction binding the contract method 0x979bc638.
//
// Solidity: function withdrawBidForPunk(uint256 punkIndex) returns()
func (_CryptoPunks *CryptoPunksTransactorSession) WithdrawBidForPunk(punkIndex *big.Int) (*types.Transaction, error) {
	return _CryptoPunks.Contract.WithdrawBidForPunk(&_CryptoPunks.TransactOpts, punkIndex)
}

// CryptoPunksAssignIterator is returned from FilterAssign and is used to iterate over the raw logs and unpacked data for Assign events raised by the CryptoPunks contract.
type CryptoPunksAssignIterator struct {
	Event *CryptoPunksAssign // Event containing the contract specifics and raw log

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
func (it *CryptoPunksAssignIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoPunksAssign)
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
		it.Event = new(CryptoPunksAssign)
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
func (it *CryptoPunksAssignIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoPunksAssignIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoPunksAssign represents a Assign event raised by the CryptoPunks contract.
type CryptoPunksAssign struct {
	To        common.Address
	PunkIndex *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAssign is a free log retrieval operation binding the contract event 0x8a0e37b73a0d9c82e205d4d1a3ff3d0b57ce5f4d7bccf6bac03336dc101cb7ba.
//
// Solidity: event Assign(address indexed to, uint256 punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) FilterAssign(opts *bind.FilterOpts, to []common.Address) (*CryptoPunksAssignIterator, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _CryptoPunks.contract.FilterLogs(opts, "Assign", toRule)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksAssignIterator{contract: _CryptoPunks.contract, event: "Assign", logs: logs, sub: sub}, nil
}

// WatchAssign is a free log subscription operation binding the contract event 0x8a0e37b73a0d9c82e205d4d1a3ff3d0b57ce5f4d7bccf6bac03336dc101cb7ba.
//
// Solidity: event Assign(address indexed to, uint256 punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) WatchAssign(opts *bind.WatchOpts, sink chan<- *CryptoPunksAssign, to []common.Address) (event.Subscription, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _CryptoPunks.contract.WatchLogs(opts, "Assign", toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoPunksAssign)
				if err := _CryptoPunks.contract.UnpackLog(event, "Assign", log); err != nil {
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

// ParseAssign is a log parse operation binding the contract event 0x8a0e37b73a0d9c82e205d4d1a3ff3d0b57ce5f4d7bccf6bac03336dc101cb7ba.
//
// Solidity: event Assign(address indexed to, uint256 punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) ParseAssign(log types.Log) (*CryptoPunksAssign, error) {
	event := new(CryptoPunksAssign)
	if err := _CryptoPunks.contract.UnpackLog(event, "Assign", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoPunksPunkBidEnteredIterator is returned from FilterPunkBidEntered and is used to iterate over the raw logs and unpacked data for PunkBidEntered events raised by the CryptoPunks contract.
type CryptoPunksPunkBidEnteredIterator struct {
	Event *CryptoPunksPunkBidEntered // Event containing the contract specifics and raw log

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
func (it *CryptoPunksPunkBidEnteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoPunksPunkBidEntered)
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
		it.Event = new(CryptoPunksPunkBidEntered)
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
func (it *CryptoPunksPunkBidEnteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoPunksPunkBidEnteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoPunksPunkBidEntered represents a PunkBidEntered event raised by the CryptoPunks contract.
type CryptoPunksPunkBidEntered struct {
	PunkIndex   *big.Int
	Value       *big.Int
	FromAddress common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterPunkBidEntered is a free log retrieval operation binding the contract event 0x5b859394fabae0c1ba88baffe67e751ab5248d2e879028b8c8d6897b0519f56a.
//
// Solidity: event PunkBidEntered(uint256 indexed punkIndex, uint256 value, address indexed fromAddress)
func (_CryptoPunks *CryptoPunksFilterer) FilterPunkBidEntered(opts *bind.FilterOpts, punkIndex []*big.Int, fromAddress []common.Address) (*CryptoPunksPunkBidEnteredIterator, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	var fromAddressRule []interface{}
	for _, fromAddressItem := range fromAddress {
		fromAddressRule = append(fromAddressRule, fromAddressItem)
	}

	logs, sub, err := _CryptoPunks.contract.FilterLogs(opts, "PunkBidEntered", punkIndexRule, fromAddressRule)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksPunkBidEnteredIterator{contract: _CryptoPunks.contract, event: "PunkBidEntered", logs: logs, sub: sub}, nil
}

// WatchPunkBidEntered is a free log subscription operation binding the contract event 0x5b859394fabae0c1ba88baffe67e751ab5248d2e879028b8c8d6897b0519f56a.
//
// Solidity: event PunkBidEntered(uint256 indexed punkIndex, uint256 value, address indexed fromAddress)
func (_CryptoPunks *CryptoPunksFilterer) WatchPunkBidEntered(opts *bind.WatchOpts, sink chan<- *CryptoPunksPunkBidEntered, punkIndex []*big.Int, fromAddress []common.Address) (event.Subscription, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	var fromAddressRule []interface{}
	for _, fromAddressItem := range fromAddress {
		fromAddressRule = append(fromAddressRule, fromAddressItem)
	}

	logs, sub, err := _CryptoPunks.contract.WatchLogs(opts, "PunkBidEntered", punkIndexRule, fromAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoPunksPunkBidEntered)
				if err := _CryptoPunks.contract.UnpackLog(event, "PunkBidEntered", log); err != nil {
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

// ParsePunkBidEntered is a log parse operation binding the contract event 0x5b859394fabae0c1ba88baffe67e751ab5248d2e879028b8c8d6897b0519f56a.
//
// Solidity: event PunkBidEntered(uint256 indexed punkIndex, uint256 value, address indexed fromAddress)
func (_CryptoPunks *CryptoPunksFilterer) ParsePunkBidEntered(log types.Log) (*CryptoPunksPunkBidEntered, error) {
	event := new(CryptoPunksPunkBidEntered)
	if err := _CryptoPunks.contract.UnpackLog(event, "PunkBidEntered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoPunksPunkBidWithdrawnIterator is returned from FilterPunkBidWithdrawn and is used to iterate over the raw logs and unpacked data for PunkBidWithdrawn events raised by the CryptoPunks contract.
type CryptoPunksPunkBidWithdrawnIterator struct {
	Event *CryptoPunksPunkBidWithdrawn // Event containing the contract specifics and raw log

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
func (it *CryptoPunksPunkBidWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoPunksPunkBidWithdrawn)
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
		it.Event = new(CryptoPunksPunkBidWithdrawn)
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
func (it *CryptoPunksPunkBidWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoPunksPunkBidWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoPunksPunkBidWithdrawn represents a PunkBidWithdrawn event raised by the CryptoPunks contract.
type CryptoPunksPunkBidWithdrawn struct {
	PunkIndex   *big.Int
	Value       *big.Int
	FromAddress common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterPunkBidWithdrawn is a free log retrieval operation binding the contract event 0x6f30e1ee4d81dcc7a8a478577f65d2ed2edb120565960ac45fe7c50551c87932.
//
// Solidity: event PunkBidWithdrawn(uint256 indexed punkIndex, uint256 value, address indexed fromAddress)
func (_CryptoPunks *CryptoPunksFilterer) FilterPunkBidWithdrawn(opts *bind.FilterOpts, punkIndex []*big.Int, fromAddress []common.Address) (*CryptoPunksPunkBidWithdrawnIterator, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	var fromAddressRule []interface{}
	for _, fromAddressItem := range fromAddress {
		fromAddressRule = append(fromAddressRule, fromAddressItem)
	}

	logs, sub, err := _CryptoPunks.contract.FilterLogs(opts, "PunkBidWithdrawn", punkIndexRule, fromAddressRule)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksPunkBidWithdrawnIterator{contract: _CryptoPunks.contract, event: "PunkBidWithdrawn", logs: logs, sub: sub}, nil
}

// WatchPunkBidWithdrawn is a free log subscription operation binding the contract event 0x6f30e1ee4d81dcc7a8a478577f65d2ed2edb120565960ac45fe7c50551c87932.
//
// Solidity: event PunkBidWithdrawn(uint256 indexed punkIndex, uint256 value, address indexed fromAddress)
func (_CryptoPunks *CryptoPunksFilterer) WatchPunkBidWithdrawn(opts *bind.WatchOpts, sink chan<- *CryptoPunksPunkBidWithdrawn, punkIndex []*big.Int, fromAddress []common.Address) (event.Subscription, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	var fromAddressRule []interface{}
	for _, fromAddressItem := range fromAddress {
		fromAddressRule = append(fromAddressRule, fromAddressItem)
	}

	logs, sub, err := _CryptoPunks.contract.WatchLogs(opts, "PunkBidWithdrawn", punkIndexRule, fromAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoPunksPunkBidWithdrawn)
				if err := _CryptoPunks.contract.UnpackLog(event, "PunkBidWithdrawn", log); err != nil {
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

// ParsePunkBidWithdrawn is a log parse operation binding the contract event 0x6f30e1ee4d81dcc7a8a478577f65d2ed2edb120565960ac45fe7c50551c87932.
//
// Solidity: event PunkBidWithdrawn(uint256 indexed punkIndex, uint256 value, address indexed fromAddress)
func (_CryptoPunks *CryptoPunksFilterer) ParsePunkBidWithdrawn(log types.Log) (*CryptoPunksPunkBidWithdrawn, error) {
	event := new(CryptoPunksPunkBidWithdrawn)
	if err := _CryptoPunks.contract.UnpackLog(event, "PunkBidWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoPunksPunkBoughtIterator is returned from FilterPunkBought and is used to iterate over the raw logs and unpacked data for PunkBought events raised by the CryptoPunks contract.
type CryptoPunksPunkBoughtIterator struct {
	Event *CryptoPunksPunkBought // Event containing the contract specifics and raw log

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
func (it *CryptoPunksPunkBoughtIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoPunksPunkBought)
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
		it.Event = new(CryptoPunksPunkBought)
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
func (it *CryptoPunksPunkBoughtIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoPunksPunkBoughtIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoPunksPunkBought represents a PunkBought event raised by the CryptoPunks contract.
type CryptoPunksPunkBought struct {
	PunkIndex   *big.Int
	Value       *big.Int
	FromAddress common.Address
	ToAddress   common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterPunkBought is a free log retrieval operation binding the contract event 0x58e5d5a525e3b40bc15abaa38b5882678db1ee68befd2f60bafe3a7fd06db9e3.
//
// Solidity: event PunkBought(uint256 indexed punkIndex, uint256 value, address indexed fromAddress, address indexed toAddress)
func (_CryptoPunks *CryptoPunksFilterer) FilterPunkBought(opts *bind.FilterOpts, punkIndex []*big.Int, fromAddress []common.Address, toAddress []common.Address) (*CryptoPunksPunkBoughtIterator, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	var fromAddressRule []interface{}
	for _, fromAddressItem := range fromAddress {
		fromAddressRule = append(fromAddressRule, fromAddressItem)
	}
	var toAddressRule []interface{}
	for _, toAddressItem := range toAddress {
		toAddressRule = append(toAddressRule, toAddressItem)
	}

	logs, sub, err := _CryptoPunks.contract.FilterLogs(opts, "PunkBought", punkIndexRule, fromAddressRule, toAddressRule)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksPunkBoughtIterator{contract: _CryptoPunks.contract, event: "PunkBought", logs: logs, sub: sub}, nil
}

// WatchPunkBought is a free log subscription operation binding the contract event 0x58e5d5a525e3b40bc15abaa38b5882678db1ee68befd2f60bafe3a7fd06db9e3.
//
// Solidity: event PunkBought(uint256 indexed punkIndex, uint256 value, address indexed fromAddress, address indexed toAddress)
func (_CryptoPunks *CryptoPunksFilterer) WatchPunkBought(opts *bind.WatchOpts, sink chan<- *CryptoPunksPunkBought, punkIndex []*big.Int, fromAddress []common.Address, toAddress []common.Address) (event.Subscription, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	var fromAddressRule []interface{}
	for _, fromAddressItem := range fromAddress {
		fromAddressRule = append(fromAddressRule, fromAddressItem)
	}
	var toAddressRule []interface{}
	for _, toAddressItem := range toAddress {
		toAddressRule = append(toAddressRule, toAddressItem)
	}

	logs, sub, err := _CryptoPunks.contract.WatchLogs(opts, "PunkBought", punkIndexRule, fromAddressRule, toAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoPunksPunkBought)
				if err := _CryptoPunks.contract.UnpackLog(event, "PunkBought", log); err != nil {
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

// ParsePunkBought is a log parse operation binding the contract event 0x58e5d5a525e3b40bc15abaa38b5882678db1ee68befd2f60bafe3a7fd06db9e3.
//
// Solidity: event PunkBought(uint256 indexed punkIndex, uint256 value, address indexed fromAddress, address indexed toAddress)
func (_CryptoPunks *CryptoPunksFilterer) ParsePunkBought(log types.Log) (*CryptoPunksPunkBought, error) {
	event := new(CryptoPunksPunkBought)
	if err := _CryptoPunks.contract.UnpackLog(event, "PunkBought", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoPunksPunkNoLongerForSaleIterator is returned from FilterPunkNoLongerForSale and is used to iterate over the raw logs and unpacked data for PunkNoLongerForSale events raised by the CryptoPunks contract.
type CryptoPunksPunkNoLongerForSaleIterator struct {
	Event *CryptoPunksPunkNoLongerForSale // Event containing the contract specifics and raw log

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
func (it *CryptoPunksPunkNoLongerForSaleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoPunksPunkNoLongerForSale)
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
		it.Event = new(CryptoPunksPunkNoLongerForSale)
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
func (it *CryptoPunksPunkNoLongerForSaleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoPunksPunkNoLongerForSaleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoPunksPunkNoLongerForSale represents a PunkNoLongerForSale event raised by the CryptoPunks contract.
type CryptoPunksPunkNoLongerForSale struct {
	PunkIndex *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterPunkNoLongerForSale is a free log retrieval operation binding the contract event 0xb0e0a660b4e50f26f0b7ce75c24655fc76cc66e3334a54ff410277229fa10bd4.
//
// Solidity: event PunkNoLongerForSale(uint256 indexed punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) FilterPunkNoLongerForSale(opts *bind.FilterOpts, punkIndex []*big.Int) (*CryptoPunksPunkNoLongerForSaleIterator, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	logs, sub, err := _CryptoPunks.contract.FilterLogs(opts, "PunkNoLongerForSale", punkIndexRule)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksPunkNoLongerForSaleIterator{contract: _CryptoPunks.contract, event: "PunkNoLongerForSale", logs: logs, sub: sub}, nil
}

// WatchPunkNoLongerForSale is a free log subscription operation binding the contract event 0xb0e0a660b4e50f26f0b7ce75c24655fc76cc66e3334a54ff410277229fa10bd4.
//
// Solidity: event PunkNoLongerForSale(uint256 indexed punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) WatchPunkNoLongerForSale(opts *bind.WatchOpts, sink chan<- *CryptoPunksPunkNoLongerForSale, punkIndex []*big.Int) (event.Subscription, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	logs, sub, err := _CryptoPunks.contract.WatchLogs(opts, "PunkNoLongerForSale", punkIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoPunksPunkNoLongerForSale)
				if err := _CryptoPunks.contract.UnpackLog(event, "PunkNoLongerForSale", log); err != nil {
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

// ParsePunkNoLongerForSale is a log parse operation binding the contract event 0xb0e0a660b4e50f26f0b7ce75c24655fc76cc66e3334a54ff410277229fa10bd4.
//
// Solidity: event PunkNoLongerForSale(uint256 indexed punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) ParsePunkNoLongerForSale(log types.Log) (*CryptoPunksPunkNoLongerForSale, error) {
	event := new(CryptoPunksPunkNoLongerForSale)
	if err := _CryptoPunks.contract.UnpackLog(event, "PunkNoLongerForSale", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoPunksPunkOfferedIterator is returned from FilterPunkOffered and is used to iterate over the raw logs and unpacked data for PunkOffered events raised by the CryptoPunks contract.
type CryptoPunksPunkOfferedIterator struct {
	Event *CryptoPunksPunkOffered // Event containing the contract specifics and raw log

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
func (it *CryptoPunksPunkOfferedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoPunksPunkOffered)
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
		it.Event = new(CryptoPunksPunkOffered)
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
func (it *CryptoPunksPunkOfferedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoPunksPunkOfferedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoPunksPunkOffered represents a PunkOffered event raised by the CryptoPunks contract.
type CryptoPunksPunkOffered struct {
	PunkIndex *big.Int
	MinValue  *big.Int
	ToAddress common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterPunkOffered is a free log retrieval operation binding the contract event 0x3c7b682d5da98001a9b8cbda6c647d2c63d698a4184fd1d55e2ce7b66f5d21eb.
//
// Solidity: event PunkOffered(uint256 indexed punkIndex, uint256 minValue, address indexed toAddress)
func (_CryptoPunks *CryptoPunksFilterer) FilterPunkOffered(opts *bind.FilterOpts, punkIndex []*big.Int, toAddress []common.Address) (*CryptoPunksPunkOfferedIterator, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	var toAddressRule []interface{}
	for _, toAddressItem := range toAddress {
		toAddressRule = append(toAddressRule, toAddressItem)
	}

	logs, sub, err := _CryptoPunks.contract.FilterLogs(opts, "PunkOffered", punkIndexRule, toAddressRule)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksPunkOfferedIterator{contract: _CryptoPunks.contract, event: "PunkOffered", logs: logs, sub: sub}, nil
}

// WatchPunkOffered is a free log subscription operation binding the contract event 0x3c7b682d5da98001a9b8cbda6c647d2c63d698a4184fd1d55e2ce7b66f5d21eb.
//
// Solidity: event PunkOffered(uint256 indexed punkIndex, uint256 minValue, address indexed toAddress)
func (_CryptoPunks *CryptoPunksFilterer) WatchPunkOffered(opts *bind.WatchOpts, sink chan<- *CryptoPunksPunkOffered, punkIndex []*big.Int, toAddress []common.Address) (event.Subscription, error) {

	var punkIndexRule []interface{}
	for _, punkIndexItem := range punkIndex {
		punkIndexRule = append(punkIndexRule, punkIndexItem)
	}

	var toAddressRule []interface{}
	for _, toAddressItem := range toAddress {
		toAddressRule = append(toAddressRule, toAddressItem)
	}

	logs, sub, err := _CryptoPunks.contract.WatchLogs(opts, "PunkOffered", punkIndexRule, toAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoPunksPunkOffered)
				if err := _CryptoPunks.contract.UnpackLog(event, "PunkOffered", log); err != nil {
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

// ParsePunkOffered is a log parse operation binding the contract event 0x3c7b682d5da98001a9b8cbda6c647d2c63d698a4184fd1d55e2ce7b66f5d21eb.
//
// Solidity: event PunkOffered(uint256 indexed punkIndex, uint256 minValue, address indexed toAddress)
func (_CryptoPunks *CryptoPunksFilterer) ParsePunkOffered(log types.Log) (*CryptoPunksPunkOffered, error) {
	event := new(CryptoPunksPunkOffered)
	if err := _CryptoPunks.contract.UnpackLog(event, "PunkOffered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoPunksPunkTransferIterator is returned from FilterPunkTransfer and is used to iterate over the raw logs and unpacked data for PunkTransfer events raised by the CryptoPunks contract.
type CryptoPunksPunkTransferIterator struct {
	Event *CryptoPunksPunkTransfer // Event containing the contract specifics and raw log

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
func (it *CryptoPunksPunkTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoPunksPunkTransfer)
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
		it.Event = new(CryptoPunksPunkTransfer)
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
func (it *CryptoPunksPunkTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoPunksPunkTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoPunksPunkTransfer represents a PunkTransfer event raised by the CryptoPunks contract.
type CryptoPunksPunkTransfer struct {
	From      common.Address
	To        common.Address
	PunkIndex *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterPunkTransfer is a free log retrieval operation binding the contract event 0x05af636b70da6819000c49f85b21fa82081c632069bb626f30932034099107d8.
//
// Solidity: event PunkTransfer(address indexed from, address indexed to, uint256 punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) FilterPunkTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*CryptoPunksPunkTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _CryptoPunks.contract.FilterLogs(opts, "PunkTransfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksPunkTransferIterator{contract: _CryptoPunks.contract, event: "PunkTransfer", logs: logs, sub: sub}, nil
}

// WatchPunkTransfer is a free log subscription operation binding the contract event 0x05af636b70da6819000c49f85b21fa82081c632069bb626f30932034099107d8.
//
// Solidity: event PunkTransfer(address indexed from, address indexed to, uint256 punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) WatchPunkTransfer(opts *bind.WatchOpts, sink chan<- *CryptoPunksPunkTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _CryptoPunks.contract.WatchLogs(opts, "PunkTransfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoPunksPunkTransfer)
				if err := _CryptoPunks.contract.UnpackLog(event, "PunkTransfer", log); err != nil {
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

// ParsePunkTransfer is a log parse operation binding the contract event 0x05af636b70da6819000c49f85b21fa82081c632069bb626f30932034099107d8.
//
// Solidity: event PunkTransfer(address indexed from, address indexed to, uint256 punkIndex)
func (_CryptoPunks *CryptoPunksFilterer) ParsePunkTransfer(log types.Log) (*CryptoPunksPunkTransfer, error) {
	event := new(CryptoPunksPunkTransfer)
	if err := _CryptoPunks.contract.UnpackLog(event, "PunkTransfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoPunksTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the CryptoPunks contract.
type CryptoPunksTransferIterator struct {
	Event *CryptoPunksTransfer // Event containing the contract specifics and raw log

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
func (it *CryptoPunksTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoPunksTransfer)
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
		it.Event = new(CryptoPunksTransfer)
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
func (it *CryptoPunksTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoPunksTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoPunksTransfer represents a Transfer event raised by the CryptoPunks contract.
type CryptoPunksTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_CryptoPunks *CryptoPunksFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*CryptoPunksTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _CryptoPunks.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &CryptoPunksTransferIterator{contract: _CryptoPunks.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_CryptoPunks *CryptoPunksFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *CryptoPunksTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _CryptoPunks.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoPunksTransfer)
				if err := _CryptoPunks.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_CryptoPunks *CryptoPunksFilterer) ParseTransfer(log types.Log) (*CryptoPunksTransfer, error) {
	event := new(CryptoPunksTransfer)
	if err := _CryptoPunks.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
