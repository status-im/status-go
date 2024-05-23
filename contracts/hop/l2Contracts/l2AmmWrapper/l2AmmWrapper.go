// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package hopL2AmmWrapper

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

// HopL2AmmWrapperMetaData contains all meta data concerning the HopL2AmmWrapper contract.
var HopL2AmmWrapperMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractL2_Bridge\",\"name\":\"_bridge\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"_l2CanonicalToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"_l2CanonicalTokenIsEth\",\"type\":\"bool\"},{\"internalType\":\"contractIERC20\",\"name\":\"_hToken\",\"type\":\"address\"},{\"internalType\":\"contractSwap\",\"name\":\"_exchangeAddress\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"attemptSwap\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractL2_Bridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"exchangeAddress\",\"outputs\":[{\"internalType\":\"contractSwap\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"hToken\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2CanonicalToken\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2CanonicalTokenIsEth\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"destinationAmountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"destinationDeadline\",\"type\":\"uint256\"}],\"name\":\"swapAndSend\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
}

// HopL2AmmWrapperABI is the input ABI used to generate the binding from.
// Deprecated: Use HopL2AmmWrapperMetaData.ABI instead.
var HopL2AmmWrapperABI = HopL2AmmWrapperMetaData.ABI

// HopL2AmmWrapper is an auto generated Go binding around an Ethereum contract.
type HopL2AmmWrapper struct {
	HopL2AmmWrapperCaller     // Read-only binding to the contract
	HopL2AmmWrapperTransactor // Write-only binding to the contract
	HopL2AmmWrapperFilterer   // Log filterer for contract events
}

// HopL2AmmWrapperCaller is an auto generated read-only Go binding around an Ethereum contract.
type HopL2AmmWrapperCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2AmmWrapperTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HopL2AmmWrapperTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2AmmWrapperFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HopL2AmmWrapperFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2AmmWrapperSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HopL2AmmWrapperSession struct {
	Contract     *HopL2AmmWrapper  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// HopL2AmmWrapperCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HopL2AmmWrapperCallerSession struct {
	Contract *HopL2AmmWrapperCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// HopL2AmmWrapperTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HopL2AmmWrapperTransactorSession struct {
	Contract     *HopL2AmmWrapperTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// HopL2AmmWrapperRaw is an auto generated low-level Go binding around an Ethereum contract.
type HopL2AmmWrapperRaw struct {
	Contract *HopL2AmmWrapper // Generic contract binding to access the raw methods on
}

// HopL2AmmWrapperCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HopL2AmmWrapperCallerRaw struct {
	Contract *HopL2AmmWrapperCaller // Generic read-only contract binding to access the raw methods on
}

// HopL2AmmWrapperTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HopL2AmmWrapperTransactorRaw struct {
	Contract *HopL2AmmWrapperTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHopL2AmmWrapper creates a new instance of HopL2AmmWrapper, bound to a specific deployed contract.
func NewHopL2AmmWrapper(address common.Address, backend bind.ContractBackend) (*HopL2AmmWrapper, error) {
	contract, err := bindHopL2AmmWrapper(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HopL2AmmWrapper{HopL2AmmWrapperCaller: HopL2AmmWrapperCaller{contract: contract}, HopL2AmmWrapperTransactor: HopL2AmmWrapperTransactor{contract: contract}, HopL2AmmWrapperFilterer: HopL2AmmWrapperFilterer{contract: contract}}, nil
}

// NewHopL2AmmWrapperCaller creates a new read-only instance of HopL2AmmWrapper, bound to a specific deployed contract.
func NewHopL2AmmWrapperCaller(address common.Address, caller bind.ContractCaller) (*HopL2AmmWrapperCaller, error) {
	contract, err := bindHopL2AmmWrapper(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HopL2AmmWrapperCaller{contract: contract}, nil
}

// NewHopL2AmmWrapperTransactor creates a new write-only instance of HopL2AmmWrapper, bound to a specific deployed contract.
func NewHopL2AmmWrapperTransactor(address common.Address, transactor bind.ContractTransactor) (*HopL2AmmWrapperTransactor, error) {
	contract, err := bindHopL2AmmWrapper(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HopL2AmmWrapperTransactor{contract: contract}, nil
}

// NewHopL2AmmWrapperFilterer creates a new log filterer instance of HopL2AmmWrapper, bound to a specific deployed contract.
func NewHopL2AmmWrapperFilterer(address common.Address, filterer bind.ContractFilterer) (*HopL2AmmWrapperFilterer, error) {
	contract, err := bindHopL2AmmWrapper(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HopL2AmmWrapperFilterer{contract: contract}, nil
}

// bindHopL2AmmWrapper binds a generic wrapper to an already deployed contract.
func bindHopL2AmmWrapper(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HopL2AmmWrapperMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL2AmmWrapper *HopL2AmmWrapperRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL2AmmWrapper.Contract.HopL2AmmWrapperCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL2AmmWrapper *HopL2AmmWrapperRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.HopL2AmmWrapperTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL2AmmWrapper *HopL2AmmWrapperRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.HopL2AmmWrapperTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL2AmmWrapper *HopL2AmmWrapperCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL2AmmWrapper.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL2AmmWrapper *HopL2AmmWrapperTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL2AmmWrapper *HopL2AmmWrapperTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.contract.Transact(opts, method, params...)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2AmmWrapper.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperSession) Bridge() (common.Address, error) {
	return _HopL2AmmWrapper.Contract.Bridge(&_HopL2AmmWrapper.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperCallerSession) Bridge() (common.Address, error) {
	return _HopL2AmmWrapper.Contract.Bridge(&_HopL2AmmWrapper.CallOpts)
}

// ExchangeAddress is a free data retrieval call binding the contract method 0x9cd01605.
//
// Solidity: function exchangeAddress() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperCaller) ExchangeAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2AmmWrapper.contract.Call(opts, &out, "exchangeAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ExchangeAddress is a free data retrieval call binding the contract method 0x9cd01605.
//
// Solidity: function exchangeAddress() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperSession) ExchangeAddress() (common.Address, error) {
	return _HopL2AmmWrapper.Contract.ExchangeAddress(&_HopL2AmmWrapper.CallOpts)
}

// ExchangeAddress is a free data retrieval call binding the contract method 0x9cd01605.
//
// Solidity: function exchangeAddress() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperCallerSession) ExchangeAddress() (common.Address, error) {
	return _HopL2AmmWrapper.Contract.ExchangeAddress(&_HopL2AmmWrapper.CallOpts)
}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperCaller) HToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2AmmWrapper.contract.Call(opts, &out, "hToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperSession) HToken() (common.Address, error) {
	return _HopL2AmmWrapper.Contract.HToken(&_HopL2AmmWrapper.CallOpts)
}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperCallerSession) HToken() (common.Address, error) {
	return _HopL2AmmWrapper.Contract.HToken(&_HopL2AmmWrapper.CallOpts)
}

// L2CanonicalToken is a free data retrieval call binding the contract method 0x1ee1bf67.
//
// Solidity: function l2CanonicalToken() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperCaller) L2CanonicalToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2AmmWrapper.contract.Call(opts, &out, "l2CanonicalToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2CanonicalToken is a free data retrieval call binding the contract method 0x1ee1bf67.
//
// Solidity: function l2CanonicalToken() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperSession) L2CanonicalToken() (common.Address, error) {
	return _HopL2AmmWrapper.Contract.L2CanonicalToken(&_HopL2AmmWrapper.CallOpts)
}

// L2CanonicalToken is a free data retrieval call binding the contract method 0x1ee1bf67.
//
// Solidity: function l2CanonicalToken() view returns(address)
func (_HopL2AmmWrapper *HopL2AmmWrapperCallerSession) L2CanonicalToken() (common.Address, error) {
	return _HopL2AmmWrapper.Contract.L2CanonicalToken(&_HopL2AmmWrapper.CallOpts)
}

// L2CanonicalTokenIsEth is a free data retrieval call binding the contract method 0x28555125.
//
// Solidity: function l2CanonicalTokenIsEth() view returns(bool)
func (_HopL2AmmWrapper *HopL2AmmWrapperCaller) L2CanonicalTokenIsEth(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _HopL2AmmWrapper.contract.Call(opts, &out, "l2CanonicalTokenIsEth")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// L2CanonicalTokenIsEth is a free data retrieval call binding the contract method 0x28555125.
//
// Solidity: function l2CanonicalTokenIsEth() view returns(bool)
func (_HopL2AmmWrapper *HopL2AmmWrapperSession) L2CanonicalTokenIsEth() (bool, error) {
	return _HopL2AmmWrapper.Contract.L2CanonicalTokenIsEth(&_HopL2AmmWrapper.CallOpts)
}

// L2CanonicalTokenIsEth is a free data retrieval call binding the contract method 0x28555125.
//
// Solidity: function l2CanonicalTokenIsEth() view returns(bool)
func (_HopL2AmmWrapper *HopL2AmmWrapperCallerSession) L2CanonicalTokenIsEth() (bool, error) {
	return _HopL2AmmWrapper.Contract.L2CanonicalTokenIsEth(&_HopL2AmmWrapper.CallOpts)
}

// AttemptSwap is a paid mutator transaction binding the contract method 0x676c5ef6.
//
// Solidity: function attemptSwap(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperTransactor) AttemptSwap(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2AmmWrapper.contract.Transact(opts, "attemptSwap", recipient, amount, amountOutMin, deadline)
}

// AttemptSwap is a paid mutator transaction binding the contract method 0x676c5ef6.
//
// Solidity: function attemptSwap(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperSession) AttemptSwap(recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.AttemptSwap(&_HopL2AmmWrapper.TransactOpts, recipient, amount, amountOutMin, deadline)
}

// AttemptSwap is a paid mutator transaction binding the contract method 0x676c5ef6.
//
// Solidity: function attemptSwap(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperTransactorSession) AttemptSwap(recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.AttemptSwap(&_HopL2AmmWrapper.TransactOpts, recipient, amount, amountOutMin, deadline)
}

// SwapAndSend is a paid mutator transaction binding the contract method 0xeea0d7b2.
//
// Solidity: function swapAndSend(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, uint256 destinationAmountOutMin, uint256 destinationDeadline) payable returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperTransactor) SwapAndSend(opts *bind.TransactOpts, chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, destinationAmountOutMin *big.Int, destinationDeadline *big.Int) (*types.Transaction, error) {
	return _HopL2AmmWrapper.contract.Transact(opts, "swapAndSend", chainId, recipient, amount, bonderFee, amountOutMin, deadline, destinationAmountOutMin, destinationDeadline)
}

// SwapAndSend is a paid mutator transaction binding the contract method 0xeea0d7b2.
//
// Solidity: function swapAndSend(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, uint256 destinationAmountOutMin, uint256 destinationDeadline) payable returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperSession) SwapAndSend(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, destinationAmountOutMin *big.Int, destinationDeadline *big.Int) (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.SwapAndSend(&_HopL2AmmWrapper.TransactOpts, chainId, recipient, amount, bonderFee, amountOutMin, deadline, destinationAmountOutMin, destinationDeadline)
}

// SwapAndSend is a paid mutator transaction binding the contract method 0xeea0d7b2.
//
// Solidity: function swapAndSend(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, uint256 destinationAmountOutMin, uint256 destinationDeadline) payable returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperTransactorSession) SwapAndSend(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, destinationAmountOutMin *big.Int, destinationDeadline *big.Int) (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.SwapAndSend(&_HopL2AmmWrapper.TransactOpts, chainId, recipient, amount, bonderFee, amountOutMin, deadline, destinationAmountOutMin, destinationDeadline)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2AmmWrapper.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperSession) Receive() (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.Receive(&_HopL2AmmWrapper.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_HopL2AmmWrapper *HopL2AmmWrapperTransactorSession) Receive() (*types.Transaction, error) {
	return _HopL2AmmWrapper.Contract.Receive(&_HopL2AmmWrapper.TransactOpts)
}
