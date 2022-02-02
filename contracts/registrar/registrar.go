// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registrar

import (
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
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// ApproveAndCallFallBackABI is the input ABI used to generate the binding from.
const ApproveAndCallFallBackABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"from\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"},{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"receiveApproval\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// ApproveAndCallFallBackFuncSigs maps the 4-byte function signature to its string representation.
var ApproveAndCallFallBackFuncSigs = map[string]string{
	"8f4ffcb1": "receiveApproval(address,uint256,address,bytes)",
}

// ApproveAndCallFallBack is an auto generated Go binding around an Ethereum contract.
type ApproveAndCallFallBack struct {
	ApproveAndCallFallBackCaller     // Read-only binding to the contract
	ApproveAndCallFallBackTransactor // Write-only binding to the contract
	ApproveAndCallFallBackFilterer   // Log filterer for contract events
}

// ApproveAndCallFallBackCaller is an auto generated read-only Go binding around an Ethereum contract.
type ApproveAndCallFallBackCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ApproveAndCallFallBackTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ApproveAndCallFallBackTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ApproveAndCallFallBackFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ApproveAndCallFallBackFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ApproveAndCallFallBackSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ApproveAndCallFallBackSession struct {
	Contract     *ApproveAndCallFallBack // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// ApproveAndCallFallBackCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ApproveAndCallFallBackCallerSession struct {
	Contract *ApproveAndCallFallBackCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// ApproveAndCallFallBackTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ApproveAndCallFallBackTransactorSession struct {
	Contract     *ApproveAndCallFallBackTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// ApproveAndCallFallBackRaw is an auto generated low-level Go binding around an Ethereum contract.
type ApproveAndCallFallBackRaw struct {
	Contract *ApproveAndCallFallBack // Generic contract binding to access the raw methods on
}

// ApproveAndCallFallBackCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ApproveAndCallFallBackCallerRaw struct {
	Contract *ApproveAndCallFallBackCaller // Generic read-only contract binding to access the raw methods on
}

// ApproveAndCallFallBackTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ApproveAndCallFallBackTransactorRaw struct {
	Contract *ApproveAndCallFallBackTransactor // Generic write-only contract binding to access the raw methods on
}

// NewApproveAndCallFallBack creates a new instance of ApproveAndCallFallBack, bound to a specific deployed contract.
func NewApproveAndCallFallBack(address common.Address, backend bind.ContractBackend) (*ApproveAndCallFallBack, error) {
	contract, err := bindApproveAndCallFallBack(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ApproveAndCallFallBack{ApproveAndCallFallBackCaller: ApproveAndCallFallBackCaller{contract: contract}, ApproveAndCallFallBackTransactor: ApproveAndCallFallBackTransactor{contract: contract}, ApproveAndCallFallBackFilterer: ApproveAndCallFallBackFilterer{contract: contract}}, nil
}

// NewApproveAndCallFallBackCaller creates a new read-only instance of ApproveAndCallFallBack, bound to a specific deployed contract.
func NewApproveAndCallFallBackCaller(address common.Address, caller bind.ContractCaller) (*ApproveAndCallFallBackCaller, error) {
	contract, err := bindApproveAndCallFallBack(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ApproveAndCallFallBackCaller{contract: contract}, nil
}

// NewApproveAndCallFallBackTransactor creates a new write-only instance of ApproveAndCallFallBack, bound to a specific deployed contract.
func NewApproveAndCallFallBackTransactor(address common.Address, transactor bind.ContractTransactor) (*ApproveAndCallFallBackTransactor, error) {
	contract, err := bindApproveAndCallFallBack(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ApproveAndCallFallBackTransactor{contract: contract}, nil
}

// NewApproveAndCallFallBackFilterer creates a new log filterer instance of ApproveAndCallFallBack, bound to a specific deployed contract.
func NewApproveAndCallFallBackFilterer(address common.Address, filterer bind.ContractFilterer) (*ApproveAndCallFallBackFilterer, error) {
	contract, err := bindApproveAndCallFallBack(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ApproveAndCallFallBackFilterer{contract: contract}, nil
}

// bindApproveAndCallFallBack binds a generic wrapper to an already deployed contract.
func bindApproveAndCallFallBack(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ApproveAndCallFallBackABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ApproveAndCallFallBack *ApproveAndCallFallBackRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ApproveAndCallFallBack.Contract.ApproveAndCallFallBackCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ApproveAndCallFallBack *ApproveAndCallFallBackRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ApproveAndCallFallBack.Contract.ApproveAndCallFallBackTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ApproveAndCallFallBack *ApproveAndCallFallBackRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ApproveAndCallFallBack.Contract.ApproveAndCallFallBackTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ApproveAndCallFallBack *ApproveAndCallFallBackCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ApproveAndCallFallBack.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ApproveAndCallFallBack *ApproveAndCallFallBackTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ApproveAndCallFallBack.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ApproveAndCallFallBack *ApproveAndCallFallBackTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ApproveAndCallFallBack.Contract.contract.Transact(opts, method, params...)
}

// ReceiveApproval is a paid mutator transaction binding the contract method 0x8f4ffcb1.
//
// Solidity: function receiveApproval(address from, uint256 _amount, address _token, bytes _data) returns()
func (_ApproveAndCallFallBack *ApproveAndCallFallBackTransactor) ReceiveApproval(opts *bind.TransactOpts, from common.Address, _amount *big.Int, _token common.Address, _data []byte) (*types.Transaction, error) {
	return _ApproveAndCallFallBack.contract.Transact(opts, "receiveApproval", from, _amount, _token, _data)
}

// ReceiveApproval is a paid mutator transaction binding the contract method 0x8f4ffcb1.
//
// Solidity: function receiveApproval(address from, uint256 _amount, address _token, bytes _data) returns()
func (_ApproveAndCallFallBack *ApproveAndCallFallBackSession) ReceiveApproval(from common.Address, _amount *big.Int, _token common.Address, _data []byte) (*types.Transaction, error) {
	return _ApproveAndCallFallBack.Contract.ReceiveApproval(&_ApproveAndCallFallBack.TransactOpts, from, _amount, _token, _data)
}

// ReceiveApproval is a paid mutator transaction binding the contract method 0x8f4ffcb1.
//
// Solidity: function receiveApproval(address from, uint256 _amount, address _token, bytes _data) returns()
func (_ApproveAndCallFallBack *ApproveAndCallFallBackTransactorSession) ReceiveApproval(from common.Address, _amount *big.Int, _token common.Address, _data []byte) (*types.Transaction, error) {
	return _ApproveAndCallFallBack.Contract.ReceiveApproval(&_ApproveAndCallFallBack.TransactOpts, from, _amount, _token, _data)
}

// ControlledABI is the input ABI used to generate the binding from.
const ControlledABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_newController\",\"type\":\"address\"}],\"name\":\"changeController\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"controller\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// ControlledFuncSigs maps the 4-byte function signature to its string representation.
var ControlledFuncSigs = map[string]string{
	"3cebb823": "changeController(address)",
	"f77c4791": "controller()",
}

// Controlled is an auto generated Go binding around an Ethereum contract.
type Controlled struct {
	ControlledCaller     // Read-only binding to the contract
	ControlledTransactor // Write-only binding to the contract
	ControlledFilterer   // Log filterer for contract events
}

// ControlledCaller is an auto generated read-only Go binding around an Ethereum contract.
type ControlledCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ControlledTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ControlledTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ControlledFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ControlledFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ControlledSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ControlledSession struct {
	Contract     *Controlled       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ControlledCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ControlledCallerSession struct {
	Contract *ControlledCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// ControlledTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ControlledTransactorSession struct {
	Contract     *ControlledTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ControlledRaw is an auto generated low-level Go binding around an Ethereum contract.
type ControlledRaw struct {
	Contract *Controlled // Generic contract binding to access the raw methods on
}

// ControlledCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ControlledCallerRaw struct {
	Contract *ControlledCaller // Generic read-only contract binding to access the raw methods on
}

// ControlledTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ControlledTransactorRaw struct {
	Contract *ControlledTransactor // Generic write-only contract binding to access the raw methods on
}

// NewControlled creates a new instance of Controlled, bound to a specific deployed contract.
func NewControlled(address common.Address, backend bind.ContractBackend) (*Controlled, error) {
	contract, err := bindControlled(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Controlled{ControlledCaller: ControlledCaller{contract: contract}, ControlledTransactor: ControlledTransactor{contract: contract}, ControlledFilterer: ControlledFilterer{contract: contract}}, nil
}

// NewControlledCaller creates a new read-only instance of Controlled, bound to a specific deployed contract.
func NewControlledCaller(address common.Address, caller bind.ContractCaller) (*ControlledCaller, error) {
	contract, err := bindControlled(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ControlledCaller{contract: contract}, nil
}

// NewControlledTransactor creates a new write-only instance of Controlled, bound to a specific deployed contract.
func NewControlledTransactor(address common.Address, transactor bind.ContractTransactor) (*ControlledTransactor, error) {
	contract, err := bindControlled(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ControlledTransactor{contract: contract}, nil
}

// NewControlledFilterer creates a new log filterer instance of Controlled, bound to a specific deployed contract.
func NewControlledFilterer(address common.Address, filterer bind.ContractFilterer) (*ControlledFilterer, error) {
	contract, err := bindControlled(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ControlledFilterer{contract: contract}, nil
}

// bindControlled binds a generic wrapper to an already deployed contract.
func bindControlled(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ControlledABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Controlled *ControlledRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Controlled.Contract.ControlledCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Controlled *ControlledRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Controlled.Contract.ControlledTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Controlled *ControlledRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Controlled.Contract.ControlledTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Controlled *ControlledCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Controlled.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Controlled *ControlledTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Controlled.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Controlled *ControlledTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Controlled.Contract.contract.Transact(opts, method, params...)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_Controlled *ControlledCaller) Controller(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Controlled.contract.Call(opts, &out, "controller")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_Controlled *ControlledSession) Controller() (common.Address, error) {
	return _Controlled.Contract.Controller(&_Controlled.CallOpts)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_Controlled *ControlledCallerSession) Controller() (common.Address, error) {
	return _Controlled.Contract.Controller(&_Controlled.CallOpts)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_Controlled *ControlledTransactor) ChangeController(opts *bind.TransactOpts, _newController common.Address) (*types.Transaction, error) {
	return _Controlled.contract.Transact(opts, "changeController", _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_Controlled *ControlledSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _Controlled.Contract.ChangeController(&_Controlled.TransactOpts, _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_Controlled *ControlledTransactorSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _Controlled.Contract.ChangeController(&_Controlled.TransactOpts, _newController)
}

// ENSABI is the input ABI used to generate the binding from.
const ENSABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"resolver\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"label\",\"type\":\"bytes32\"},{\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"setSubnodeOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"ttl\",\"type\":\"uint64\"}],\"name\":\"setTTL\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"ttl\",\"outputs\":[{\"name\":\"\",\"type\":\"uint64\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"resolver\",\"type\":\"address\"}],\"name\":\"setResolver\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"setOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"label\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NewOwner\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"resolver\",\"type\":\"address\"}],\"name\":\"NewResolver\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"ttl\",\"type\":\"uint64\"}],\"name\":\"NewTTL\",\"type\":\"event\"}]"

// ENSFuncSigs maps the 4-byte function signature to its string representation.
var ENSFuncSigs = map[string]string{
	"02571be3": "owner(bytes32)",
	"0178b8bf": "resolver(bytes32)",
	"5b0fc9c3": "setOwner(bytes32,address)",
	"1896f70a": "setResolver(bytes32,address)",
	"06ab5923": "setSubnodeOwner(bytes32,bytes32,address)",
	"14ab9038": "setTTL(bytes32,uint64)",
	"16a25cbd": "ttl(bytes32)",
}

// ENS is an auto generated Go binding around an Ethereum contract.
type ENS struct {
	ENSCaller     // Read-only binding to the contract
	ENSTransactor // Write-only binding to the contract
	ENSFilterer   // Log filterer for contract events
}

// ENSCaller is an auto generated read-only Go binding around an Ethereum contract.
type ENSCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ENSTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ENSTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ENSFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ENSFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ENSSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ENSSession struct {
	Contract     *ENS              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ENSCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ENSCallerSession struct {
	Contract *ENSCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// ENSTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ENSTransactorSession struct {
	Contract     *ENSTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ENSRaw is an auto generated low-level Go binding around an Ethereum contract.
type ENSRaw struct {
	Contract *ENS // Generic contract binding to access the raw methods on
}

// ENSCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ENSCallerRaw struct {
	Contract *ENSCaller // Generic read-only contract binding to access the raw methods on
}

// ENSTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ENSTransactorRaw struct {
	Contract *ENSTransactor // Generic write-only contract binding to access the raw methods on
}

// NewENS creates a new instance of ENS, bound to a specific deployed contract.
func NewENS(address common.Address, backend bind.ContractBackend) (*ENS, error) {
	contract, err := bindENS(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ENS{ENSCaller: ENSCaller{contract: contract}, ENSTransactor: ENSTransactor{contract: contract}, ENSFilterer: ENSFilterer{contract: contract}}, nil
}

// NewENSCaller creates a new read-only instance of ENS, bound to a specific deployed contract.
func NewENSCaller(address common.Address, caller bind.ContractCaller) (*ENSCaller, error) {
	contract, err := bindENS(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ENSCaller{contract: contract}, nil
}

// NewENSTransactor creates a new write-only instance of ENS, bound to a specific deployed contract.
func NewENSTransactor(address common.Address, transactor bind.ContractTransactor) (*ENSTransactor, error) {
	contract, err := bindENS(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ENSTransactor{contract: contract}, nil
}

// NewENSFilterer creates a new log filterer instance of ENS, bound to a specific deployed contract.
func NewENSFilterer(address common.Address, filterer bind.ContractFilterer) (*ENSFilterer, error) {
	contract, err := bindENS(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ENSFilterer{contract: contract}, nil
}

// bindENS binds a generic wrapper to an already deployed contract.
func bindENS(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ENSABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ENS *ENSRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ENS.Contract.ENSCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ENS *ENSRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ENS.Contract.ENSTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ENS *ENSRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ENS.Contract.ENSTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ENS *ENSCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ENS.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ENS *ENSTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ENS.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ENS *ENSTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ENS.Contract.contract.Transact(opts, method, params...)
}

// Owner is a free data retrieval call binding the contract method 0x02571be3.
//
// Solidity: function owner(bytes32 node) view returns(address)
func (_ENS *ENSCaller) Owner(opts *bind.CallOpts, node [32]byte) (common.Address, error) {
	var out []interface{}
	err := _ENS.contract.Call(opts, &out, "owner", node)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x02571be3.
//
// Solidity: function owner(bytes32 node) view returns(address)
func (_ENS *ENSSession) Owner(node [32]byte) (common.Address, error) {
	return _ENS.Contract.Owner(&_ENS.CallOpts, node)
}

// Owner is a free data retrieval call binding the contract method 0x02571be3.
//
// Solidity: function owner(bytes32 node) view returns(address)
func (_ENS *ENSCallerSession) Owner(node [32]byte) (common.Address, error) {
	return _ENS.Contract.Owner(&_ENS.CallOpts, node)
}

// Resolver is a free data retrieval call binding the contract method 0x0178b8bf.
//
// Solidity: function resolver(bytes32 node) view returns(address)
func (_ENS *ENSCaller) Resolver(opts *bind.CallOpts, node [32]byte) (common.Address, error) {
	var out []interface{}
	err := _ENS.contract.Call(opts, &out, "resolver", node)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Resolver is a free data retrieval call binding the contract method 0x0178b8bf.
//
// Solidity: function resolver(bytes32 node) view returns(address)
func (_ENS *ENSSession) Resolver(node [32]byte) (common.Address, error) {
	return _ENS.Contract.Resolver(&_ENS.CallOpts, node)
}

// Resolver is a free data retrieval call binding the contract method 0x0178b8bf.
//
// Solidity: function resolver(bytes32 node) view returns(address)
func (_ENS *ENSCallerSession) Resolver(node [32]byte) (common.Address, error) {
	return _ENS.Contract.Resolver(&_ENS.CallOpts, node)
}

// Ttl is a free data retrieval call binding the contract method 0x16a25cbd.
//
// Solidity: function ttl(bytes32 node) view returns(uint64)
func (_ENS *ENSCaller) Ttl(opts *bind.CallOpts, node [32]byte) (uint64, error) {
	var out []interface{}
	err := _ENS.contract.Call(opts, &out, "ttl", node)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// Ttl is a free data retrieval call binding the contract method 0x16a25cbd.
//
// Solidity: function ttl(bytes32 node) view returns(uint64)
func (_ENS *ENSSession) Ttl(node [32]byte) (uint64, error) {
	return _ENS.Contract.Ttl(&_ENS.CallOpts, node)
}

// Ttl is a free data retrieval call binding the contract method 0x16a25cbd.
//
// Solidity: function ttl(bytes32 node) view returns(uint64)
func (_ENS *ENSCallerSession) Ttl(node [32]byte) (uint64, error) {
	return _ENS.Contract.Ttl(&_ENS.CallOpts, node)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(bytes32 node, address owner) returns()
func (_ENS *ENSTransactor) SetOwner(opts *bind.TransactOpts, node [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.contract.Transact(opts, "setOwner", node, owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(bytes32 node, address owner) returns()
func (_ENS *ENSSession) SetOwner(node [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetOwner(&_ENS.TransactOpts, node, owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x5b0fc9c3.
//
// Solidity: function setOwner(bytes32 node, address owner) returns()
func (_ENS *ENSTransactorSession) SetOwner(node [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetOwner(&_ENS.TransactOpts, node, owner)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(bytes32 node, address resolver) returns()
func (_ENS *ENSTransactor) SetResolver(opts *bind.TransactOpts, node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _ENS.contract.Transact(opts, "setResolver", node, resolver)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(bytes32 node, address resolver) returns()
func (_ENS *ENSSession) SetResolver(node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetResolver(&_ENS.TransactOpts, node, resolver)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(bytes32 node, address resolver) returns()
func (_ENS *ENSTransactorSession) SetResolver(node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetResolver(&_ENS.TransactOpts, node, resolver)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0x06ab5923.
//
// Solidity: function setSubnodeOwner(bytes32 node, bytes32 label, address owner) returns()
func (_ENS *ENSTransactor) SetSubnodeOwner(opts *bind.TransactOpts, node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.contract.Transact(opts, "setSubnodeOwner", node, label, owner)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0x06ab5923.
//
// Solidity: function setSubnodeOwner(bytes32 node, bytes32 label, address owner) returns()
func (_ENS *ENSSession) SetSubnodeOwner(node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetSubnodeOwner(&_ENS.TransactOpts, node, label, owner)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0x06ab5923.
//
// Solidity: function setSubnodeOwner(bytes32 node, bytes32 label, address owner) returns()
func (_ENS *ENSTransactorSession) SetSubnodeOwner(node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _ENS.Contract.SetSubnodeOwner(&_ENS.TransactOpts, node, label, owner)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(bytes32 node, uint64 ttl) returns()
func (_ENS *ENSTransactor) SetTTL(opts *bind.TransactOpts, node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _ENS.contract.Transact(opts, "setTTL", node, ttl)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(bytes32 node, uint64 ttl) returns()
func (_ENS *ENSSession) SetTTL(node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _ENS.Contract.SetTTL(&_ENS.TransactOpts, node, ttl)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(bytes32 node, uint64 ttl) returns()
func (_ENS *ENSTransactorSession) SetTTL(node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _ENS.Contract.SetTTL(&_ENS.TransactOpts, node, ttl)
}

// ENSNewOwnerIterator is returned from FilterNewOwner and is used to iterate over the raw logs and unpacked data for NewOwner events raised by the ENS contract.
type ENSNewOwnerIterator struct {
	Event *ENSNewOwner // Event containing the contract specifics and raw log

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
func (it *ENSNewOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ENSNewOwner)
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
		it.Event = new(ENSNewOwner)
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
func (it *ENSNewOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ENSNewOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ENSNewOwner represents a NewOwner event raised by the ENS contract.
type ENSNewOwner struct {
	Node  [32]byte
	Label [32]byte
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterNewOwner is a free log retrieval operation binding the contract event 0xce0457fe73731f824cc272376169235128c118b49d344817417c6d108d155e82.
//
// Solidity: event NewOwner(bytes32 indexed node, bytes32 indexed label, address owner)
func (_ENS *ENSFilterer) FilterNewOwner(opts *bind.FilterOpts, node [][32]byte, label [][32]byte) (*ENSNewOwnerIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}
	var labelRule []interface{}
	for _, labelItem := range label {
		labelRule = append(labelRule, labelItem)
	}

	logs, sub, err := _ENS.contract.FilterLogs(opts, "NewOwner", nodeRule, labelRule)
	if err != nil {
		return nil, err
	}
	return &ENSNewOwnerIterator{contract: _ENS.contract, event: "NewOwner", logs: logs, sub: sub}, nil
}

// WatchNewOwner is a free log subscription operation binding the contract event 0xce0457fe73731f824cc272376169235128c118b49d344817417c6d108d155e82.
//
// Solidity: event NewOwner(bytes32 indexed node, bytes32 indexed label, address owner)
func (_ENS *ENSFilterer) WatchNewOwner(opts *bind.WatchOpts, sink chan<- *ENSNewOwner, node [][32]byte, label [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}
	var labelRule []interface{}
	for _, labelItem := range label {
		labelRule = append(labelRule, labelItem)
	}

	logs, sub, err := _ENS.contract.WatchLogs(opts, "NewOwner", nodeRule, labelRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ENSNewOwner)
				if err := _ENS.contract.UnpackLog(event, "NewOwner", log); err != nil {
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

// ParseNewOwner is a log parse operation binding the contract event 0xce0457fe73731f824cc272376169235128c118b49d344817417c6d108d155e82.
//
// Solidity: event NewOwner(bytes32 indexed node, bytes32 indexed label, address owner)
func (_ENS *ENSFilterer) ParseNewOwner(log types.Log) (*ENSNewOwner, error) {
	event := new(ENSNewOwner)
	if err := _ENS.contract.UnpackLog(event, "NewOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ENSNewResolverIterator is returned from FilterNewResolver and is used to iterate over the raw logs and unpacked data for NewResolver events raised by the ENS contract.
type ENSNewResolverIterator struct {
	Event *ENSNewResolver // Event containing the contract specifics and raw log

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
func (it *ENSNewResolverIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ENSNewResolver)
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
		it.Event = new(ENSNewResolver)
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
func (it *ENSNewResolverIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ENSNewResolverIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ENSNewResolver represents a NewResolver event raised by the ENS contract.
type ENSNewResolver struct {
	Node     [32]byte
	Resolver common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterNewResolver is a free log retrieval operation binding the contract event 0x335721b01866dc23fbee8b6b2c7b1e14d6f05c28cd35a2c934239f94095602a0.
//
// Solidity: event NewResolver(bytes32 indexed node, address resolver)
func (_ENS *ENSFilterer) FilterNewResolver(opts *bind.FilterOpts, node [][32]byte) (*ENSNewResolverIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _ENS.contract.FilterLogs(opts, "NewResolver", nodeRule)
	if err != nil {
		return nil, err
	}
	return &ENSNewResolverIterator{contract: _ENS.contract, event: "NewResolver", logs: logs, sub: sub}, nil
}

// WatchNewResolver is a free log subscription operation binding the contract event 0x335721b01866dc23fbee8b6b2c7b1e14d6f05c28cd35a2c934239f94095602a0.
//
// Solidity: event NewResolver(bytes32 indexed node, address resolver)
func (_ENS *ENSFilterer) WatchNewResolver(opts *bind.WatchOpts, sink chan<- *ENSNewResolver, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _ENS.contract.WatchLogs(opts, "NewResolver", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ENSNewResolver)
				if err := _ENS.contract.UnpackLog(event, "NewResolver", log); err != nil {
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

// ParseNewResolver is a log parse operation binding the contract event 0x335721b01866dc23fbee8b6b2c7b1e14d6f05c28cd35a2c934239f94095602a0.
//
// Solidity: event NewResolver(bytes32 indexed node, address resolver)
func (_ENS *ENSFilterer) ParseNewResolver(log types.Log) (*ENSNewResolver, error) {
	event := new(ENSNewResolver)
	if err := _ENS.contract.UnpackLog(event, "NewResolver", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ENSNewTTLIterator is returned from FilterNewTTL and is used to iterate over the raw logs and unpacked data for NewTTL events raised by the ENS contract.
type ENSNewTTLIterator struct {
	Event *ENSNewTTL // Event containing the contract specifics and raw log

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
func (it *ENSNewTTLIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ENSNewTTL)
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
		it.Event = new(ENSNewTTL)
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
func (it *ENSNewTTLIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ENSNewTTLIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ENSNewTTL represents a NewTTL event raised by the ENS contract.
type ENSNewTTL struct {
	Node [32]byte
	Ttl  uint64
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterNewTTL is a free log retrieval operation binding the contract event 0x1d4f9bbfc9cab89d66e1a1562f2233ccbf1308cb4f63de2ead5787adddb8fa68.
//
// Solidity: event NewTTL(bytes32 indexed node, uint64 ttl)
func (_ENS *ENSFilterer) FilterNewTTL(opts *bind.FilterOpts, node [][32]byte) (*ENSNewTTLIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _ENS.contract.FilterLogs(opts, "NewTTL", nodeRule)
	if err != nil {
		return nil, err
	}
	return &ENSNewTTLIterator{contract: _ENS.contract, event: "NewTTL", logs: logs, sub: sub}, nil
}

// WatchNewTTL is a free log subscription operation binding the contract event 0x1d4f9bbfc9cab89d66e1a1562f2233ccbf1308cb4f63de2ead5787adddb8fa68.
//
// Solidity: event NewTTL(bytes32 indexed node, uint64 ttl)
func (_ENS *ENSFilterer) WatchNewTTL(opts *bind.WatchOpts, sink chan<- *ENSNewTTL, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _ENS.contract.WatchLogs(opts, "NewTTL", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ENSNewTTL)
				if err := _ENS.contract.UnpackLog(event, "NewTTL", log); err != nil {
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

// ParseNewTTL is a log parse operation binding the contract event 0x1d4f9bbfc9cab89d66e1a1562f2233ccbf1308cb4f63de2ead5787adddb8fa68.
//
// Solidity: event NewTTL(bytes32 indexed node, uint64 ttl)
func (_ENS *ENSFilterer) ParseNewTTL(log types.Log) (*ENSNewTTL, error) {
	event := new(ENSNewTTL)
	if err := _ENS.contract.UnpackLog(event, "NewTTL", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ENSTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the ENS contract.
type ENSTransferIterator struct {
	Event *ENSTransfer // Event containing the contract specifics and raw log

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
func (it *ENSTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ENSTransfer)
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
		it.Event = new(ENSTransfer)
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
func (it *ENSTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ENSTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ENSTransfer represents a Transfer event raised by the ENS contract.
type ENSTransfer struct {
	Node  [32]byte
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xd4735d920b0f87494915f556dd9b54c8f309026070caea5c737245152564d266.
//
// Solidity: event Transfer(bytes32 indexed node, address owner)
func (_ENS *ENSFilterer) FilterTransfer(opts *bind.FilterOpts, node [][32]byte) (*ENSTransferIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _ENS.contract.FilterLogs(opts, "Transfer", nodeRule)
	if err != nil {
		return nil, err
	}
	return &ENSTransferIterator{contract: _ENS.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xd4735d920b0f87494915f556dd9b54c8f309026070caea5c737245152564d266.
//
// Solidity: event Transfer(bytes32 indexed node, address owner)
func (_ENS *ENSFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *ENSTransfer, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _ENS.contract.WatchLogs(opts, "Transfer", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ENSTransfer)
				if err := _ENS.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xd4735d920b0f87494915f556dd9b54c8f309026070caea5c737245152564d266.
//
// Solidity: event Transfer(bytes32 indexed node, address owner)
func (_ENS *ENSFilterer) ParseTransfer(log types.Log) (*ENSTransfer, error) {
	event := new(ENSTransfer)
	if err := _ENS.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ERC20TokenABI is the input ABI used to generate the binding from.
const ERC20TokenABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"supply\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"remaining\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"}]"

// ERC20TokenFuncSigs maps the 4-byte function signature to its string representation.
var ERC20TokenFuncSigs = map[string]string{
	"dd62ed3e": "allowance(address,address)",
	"095ea7b3": "approve(address,uint256)",
	"70a08231": "balanceOf(address)",
	"18160ddd": "totalSupply()",
	"a9059cbb": "transfer(address,uint256)",
	"23b872dd": "transferFrom(address,address,uint256)",
}

// ERC20Token is an auto generated Go binding around an Ethereum contract.
type ERC20Token struct {
	ERC20TokenCaller     // Read-only binding to the contract
	ERC20TokenTransactor // Write-only binding to the contract
	ERC20TokenFilterer   // Log filterer for contract events
}

// ERC20TokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC20TokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC20TokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC20TokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC20TokenSession struct {
	Contract     *ERC20Token       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ERC20TokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC20TokenCallerSession struct {
	Contract *ERC20TokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// ERC20TokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC20TokenTransactorSession struct {
	Contract     *ERC20TokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ERC20TokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC20TokenRaw struct {
	Contract *ERC20Token // Generic contract binding to access the raw methods on
}

// ERC20TokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC20TokenCallerRaw struct {
	Contract *ERC20TokenCaller // Generic read-only contract binding to access the raw methods on
}

// ERC20TokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC20TokenTransactorRaw struct {
	Contract *ERC20TokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC20Token creates a new instance of ERC20Token, bound to a specific deployed contract.
func NewERC20Token(address common.Address, backend bind.ContractBackend) (*ERC20Token, error) {
	contract, err := bindERC20Token(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC20Token{ERC20TokenCaller: ERC20TokenCaller{contract: contract}, ERC20TokenTransactor: ERC20TokenTransactor{contract: contract}, ERC20TokenFilterer: ERC20TokenFilterer{contract: contract}}, nil
}

// NewERC20TokenCaller creates a new read-only instance of ERC20Token, bound to a specific deployed contract.
func NewERC20TokenCaller(address common.Address, caller bind.ContractCaller) (*ERC20TokenCaller, error) {
	contract, err := bindERC20Token(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenCaller{contract: contract}, nil
}

// NewERC20TokenTransactor creates a new write-only instance of ERC20Token, bound to a specific deployed contract.
func NewERC20TokenTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC20TokenTransactor, error) {
	contract, err := bindERC20Token(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenTransactor{contract: contract}, nil
}

// NewERC20TokenFilterer creates a new log filterer instance of ERC20Token, bound to a specific deployed contract.
func NewERC20TokenFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC20TokenFilterer, error) {
	contract, err := bindERC20Token(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenFilterer{contract: contract}, nil
}

// bindERC20Token binds a generic wrapper to an already deployed contract.
func bindERC20Token(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ERC20TokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Token *ERC20TokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC20Token.Contract.ERC20TokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Token *ERC20TokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Token.Contract.ERC20TokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Token *ERC20TokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Token.Contract.ERC20TokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Token *ERC20TokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC20Token.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Token *ERC20TokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Token.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Token *ERC20TokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Token.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_ERC20Token *ERC20TokenCaller) Allowance(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "allowance", _owner, _spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_ERC20Token *ERC20TokenSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _ERC20Token.Contract.Allowance(&_ERC20Token.CallOpts, _owner, _spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_ERC20Token *ERC20TokenCallerSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _ERC20Token.Contract.Allowance(&_ERC20Token.CallOpts, _owner, _spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_ERC20Token *ERC20TokenCaller) BalanceOf(opts *bind.CallOpts, _owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "balanceOf", _owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_ERC20Token *ERC20TokenSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _ERC20Token.Contract.BalanceOf(&_ERC20Token.CallOpts, _owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_ERC20Token *ERC20TokenCallerSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _ERC20Token.Contract.BalanceOf(&_ERC20Token.CallOpts, _owner)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256 supply)
func (_ERC20Token *ERC20TokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256 supply)
func (_ERC20Token *ERC20TokenSession) TotalSupply() (*big.Int, error) {
	return _ERC20Token.Contract.TotalSupply(&_ERC20Token.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256 supply)
func (_ERC20Token *ERC20TokenCallerSession) TotalSupply() (*big.Int, error) {
	return _ERC20Token.Contract.TotalSupply(&_ERC20Token.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenTransactor) Approve(opts *bind.TransactOpts, _spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.contract.Transact(opts, "approve", _spender, _value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenSession) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.Approve(&_ERC20Token.TransactOpts, _spender, _value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenTransactorSession) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.Approve(&_ERC20Token.TransactOpts, _spender, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenTransactor) Transfer(opts *bind.TransactOpts, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.contract.Transact(opts, "transfer", _to, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenSession) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.Transfer(&_ERC20Token.TransactOpts, _to, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenTransactorSession) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.Transfer(&_ERC20Token.TransactOpts, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenTransactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.contract.Transact(opts, "transferFrom", _from, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenSession) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.TransferFrom(&_ERC20Token.TransactOpts, _from, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_ERC20Token *ERC20TokenTransactorSession) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.TransferFrom(&_ERC20Token.TransactOpts, _from, _to, _value)
}

// ERC20TokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the ERC20Token contract.
type ERC20TokenApprovalIterator struct {
	Event *ERC20TokenApproval // Event containing the contract specifics and raw log

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
func (it *ERC20TokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20TokenApproval)
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
		it.Event = new(ERC20TokenApproval)
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
func (it *ERC20TokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20TokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20TokenApproval represents a Approval event raised by the ERC20Token contract.
type ERC20TokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _value)
func (_ERC20Token *ERC20TokenFilterer) FilterApproval(opts *bind.FilterOpts, _owner []common.Address, _spender []common.Address) (*ERC20TokenApprovalIterator, error) {

	var _ownerRule []interface{}
	for _, _ownerItem := range _owner {
		_ownerRule = append(_ownerRule, _ownerItem)
	}
	var _spenderRule []interface{}
	for _, _spenderItem := range _spender {
		_spenderRule = append(_spenderRule, _spenderItem)
	}

	logs, sub, err := _ERC20Token.contract.FilterLogs(opts, "Approval", _ownerRule, _spenderRule)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenApprovalIterator{contract: _ERC20Token.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _value)
func (_ERC20Token *ERC20TokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *ERC20TokenApproval, _owner []common.Address, _spender []common.Address) (event.Subscription, error) {

	var _ownerRule []interface{}
	for _, _ownerItem := range _owner {
		_ownerRule = append(_ownerRule, _ownerItem)
	}
	var _spenderRule []interface{}
	for _, _spenderItem := range _spender {
		_spenderRule = append(_spenderRule, _spenderItem)
	}

	logs, sub, err := _ERC20Token.contract.WatchLogs(opts, "Approval", _ownerRule, _spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20TokenApproval)
				if err := _ERC20Token.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _value)
func (_ERC20Token *ERC20TokenFilterer) ParseApproval(log types.Log) (*ERC20TokenApproval, error) {
	event := new(ERC20TokenApproval)
	if err := _ERC20Token.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ERC20TokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the ERC20Token contract.
type ERC20TokenTransferIterator struct {
	Event *ERC20TokenTransfer // Event containing the contract specifics and raw log

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
func (it *ERC20TokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20TokenTransfer)
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
		it.Event = new(ERC20TokenTransfer)
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
func (it *ERC20TokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20TokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20TokenTransfer represents a Transfer event raised by the ERC20Token contract.
type ERC20TokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _value)
func (_ERC20Token *ERC20TokenFilterer) FilterTransfer(opts *bind.FilterOpts, _from []common.Address, _to []common.Address) (*ERC20TokenTransferIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _ERC20Token.contract.FilterLogs(opts, "Transfer", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenTransferIterator{contract: _ERC20Token.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _value)
func (_ERC20Token *ERC20TokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *ERC20TokenTransfer, _from []common.Address, _to []common.Address) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _ERC20Token.contract.WatchLogs(opts, "Transfer", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20TokenTransfer)
				if err := _ERC20Token.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _value)
func (_ERC20Token *ERC20TokenFilterer) ParseTransfer(log types.Log) (*ERC20TokenTransfer, error) {
	event := new(ERC20TokenTransfer)
	if err := _ERC20Token.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MerkleProofABI is the input ABI used to generate the binding from.
const MerkleProofABI = "[]"

// MerkleProofBin is the compiled bytecode used for deploying new contracts.
var MerkleProofBin = "0x604c602c600b82828239805160001a60731460008114601c57601e565bfe5b5030600052607381538281f30073000000000000000000000000000000000000000030146080604052600080fd00a165627a7a72305820170addd6452ce06e5d52013970c73987b0324a1f6c9ecf5a2fef8923de0e62180029"

// DeployMerkleProof deploys a new Ethereum contract, binding an instance of MerkleProof to it.
func DeployMerkleProof(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MerkleProof, error) {
	parsed, err := abi.JSON(strings.NewReader(MerkleProofABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(MerkleProofBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MerkleProof{MerkleProofCaller: MerkleProofCaller{contract: contract}, MerkleProofTransactor: MerkleProofTransactor{contract: contract}, MerkleProofFilterer: MerkleProofFilterer{contract: contract}}, nil
}

// MerkleProof is an auto generated Go binding around an Ethereum contract.
type MerkleProof struct {
	MerkleProofCaller     // Read-only binding to the contract
	MerkleProofTransactor // Write-only binding to the contract
	MerkleProofFilterer   // Log filterer for contract events
}

// MerkleProofCaller is an auto generated read-only Go binding around an Ethereum contract.
type MerkleProofCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleProofTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MerkleProofTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleProofFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MerkleProofFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleProofSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MerkleProofSession struct {
	Contract     *MerkleProof      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MerkleProofCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MerkleProofCallerSession struct {
	Contract *MerkleProofCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// MerkleProofTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MerkleProofTransactorSession struct {
	Contract     *MerkleProofTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// MerkleProofRaw is an auto generated low-level Go binding around an Ethereum contract.
type MerkleProofRaw struct {
	Contract *MerkleProof // Generic contract binding to access the raw methods on
}

// MerkleProofCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MerkleProofCallerRaw struct {
	Contract *MerkleProofCaller // Generic read-only contract binding to access the raw methods on
}

// MerkleProofTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MerkleProofTransactorRaw struct {
	Contract *MerkleProofTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMerkleProof creates a new instance of MerkleProof, bound to a specific deployed contract.
func NewMerkleProof(address common.Address, backend bind.ContractBackend) (*MerkleProof, error) {
	contract, err := bindMerkleProof(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MerkleProof{MerkleProofCaller: MerkleProofCaller{contract: contract}, MerkleProofTransactor: MerkleProofTransactor{contract: contract}, MerkleProofFilterer: MerkleProofFilterer{contract: contract}}, nil
}

// NewMerkleProofCaller creates a new read-only instance of MerkleProof, bound to a specific deployed contract.
func NewMerkleProofCaller(address common.Address, caller bind.ContractCaller) (*MerkleProofCaller, error) {
	contract, err := bindMerkleProof(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MerkleProofCaller{contract: contract}, nil
}

// NewMerkleProofTransactor creates a new write-only instance of MerkleProof, bound to a specific deployed contract.
func NewMerkleProofTransactor(address common.Address, transactor bind.ContractTransactor) (*MerkleProofTransactor, error) {
	contract, err := bindMerkleProof(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MerkleProofTransactor{contract: contract}, nil
}

// NewMerkleProofFilterer creates a new log filterer instance of MerkleProof, bound to a specific deployed contract.
func NewMerkleProofFilterer(address common.Address, filterer bind.ContractFilterer) (*MerkleProofFilterer, error) {
	contract, err := bindMerkleProof(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MerkleProofFilterer{contract: contract}, nil
}

// bindMerkleProof binds a generic wrapper to an already deployed contract.
func bindMerkleProof(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MerkleProofABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MerkleProof *MerkleProofRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MerkleProof.Contract.MerkleProofCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MerkleProof *MerkleProofRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MerkleProof.Contract.MerkleProofTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MerkleProof *MerkleProofRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MerkleProof.Contract.MerkleProofTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MerkleProof *MerkleProofCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MerkleProof.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MerkleProof *MerkleProofTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MerkleProof.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MerkleProof *MerkleProofTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MerkleProof.Contract.contract.Transact(opts, method, params...)
}

// PublicResolverABI is the input ABI used to generate the binding from.
const PublicResolverABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"interfaceID\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"key\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"string\"}],\"name\":\"setText\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"contentTypes\",\"type\":\"uint256\"}],\"name\":\"ABI\",\"outputs\":[{\"name\":\"contentType\",\"type\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"x\",\"type\":\"bytes32\"},{\"name\":\"y\",\"type\":\"bytes32\"}],\"name\":\"setPubkey\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"content\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"addr\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"key\",\"type\":\"string\"}],\"name\":\"text\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"contentType\",\"type\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"setABI\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"name\",\"type\":\"string\"}],\"name\":\"setName\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"hash\",\"type\":\"bytes\"}],\"name\":\"setMultihash\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"hash\",\"type\":\"bytes32\"}],\"name\":\"setContent\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"pubkey\",\"outputs\":[{\"name\":\"x\",\"type\":\"bytes32\"},{\"name\":\"y\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"setAddr\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"multihash\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"ensAddr\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"a\",\"type\":\"address\"}],\"name\":\"AddrChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"hash\",\"type\":\"bytes32\"}],\"name\":\"ContentChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"}],\"name\":\"NameChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"contentType\",\"type\":\"uint256\"}],\"name\":\"ABIChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"x\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"y\",\"type\":\"bytes32\"}],\"name\":\"PubkeyChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"indexedKey\",\"type\":\"string\"},{\"indexed\":false,\"name\":\"key\",\"type\":\"string\"}],\"name\":\"TextChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"hash\",\"type\":\"bytes\"}],\"name\":\"MultihashChanged\",\"type\":\"event\"}]"

// PublicResolverFuncSigs maps the 4-byte function signature to its string representation.
var PublicResolverFuncSigs = map[string]string{
	"2203ab56": "ABI(bytes32,uint256)",
	"3b3b57de": "addr(bytes32)",
	"2dff6941": "content(bytes32)",
	"e89401a1": "multihash(bytes32)",
	"691f3431": "name(bytes32)",
	"c8690233": "pubkey(bytes32)",
	"623195b0": "setABI(bytes32,uint256,bytes)",
	"d5fa2b00": "setAddr(bytes32,address)",
	"c3d014d6": "setContent(bytes32,bytes32)",
	"aa4cb547": "setMultihash(bytes32,bytes)",
	"77372213": "setName(bytes32,string)",
	"29cd62ea": "setPubkey(bytes32,bytes32,bytes32)",
	"10f13a8c": "setText(bytes32,string,string)",
	"01ffc9a7": "supportsInterface(bytes4)",
	"59d1d43c": "text(bytes32,string)",
}

// PublicResolverBin is the compiled bytecode used for deploying new contracts.
var PublicResolverBin = "0x608060405234801561001057600080fd5b50604051602080611400833981016040525160008054600160a060020a03909216600160a060020a03199092169190911790556113ae806100526000396000f3006080604052600436106100da5763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166301ffc9a781146100df57806310f13a8c146101155780632203ab56146101b357806329cd62ea1461024d5780632dff69411461026b5780633b3b57de1461029557806359d1d43c146102c9578063623195b01461039c578063691f3431146103fc5780637737221314610414578063aa4cb54714610472578063c3d014d6146104d0578063c8690233146104eb578063d5fa2b001461051c578063e89401a114610540575b600080fd5b3480156100eb57600080fd5b50610101600160e060020a031960043516610558565b604080519115158252519081900360200190f35b34801561012157600080fd5b5060408051602060046024803582810135601f81018590048502860185019096528585526101b195833595369560449491939091019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a9998810197919650918201945092508291508401838280828437509497506106f99650505050505050565b005b3480156101bf57600080fd5b506101ce60043560243561091f565b6040518083815260200180602001828103825283818151815260200191508051906020019080838360005b838110156102115781810151838201526020016101f9565b50505050905090810190601f16801561023e5780820380516001836020036101000a031916815260200191505b50935050505060405180910390f35b34801561025957600080fd5b506101b1600435602435604435610a2b565b34801561027757600080fd5b50610283600435610b2b565b60408051918252519081900360200190f35b3480156102a157600080fd5b506102ad600435610b41565b60408051600160a060020a039092168252519081900360200190f35b3480156102d557600080fd5b5060408051602060046024803582810135601f8101859004850286018501909652858552610327958335953695604494919390910191908190840183828082843750949750610b5c9650505050505050565b6040805160208082528351818301528351919283929083019185019080838360005b83811015610361578181015183820152602001610349565b50505050905090810190601f16801561038e5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156103a857600080fd5b50604080516020600460443581810135601f81018490048402850184019095528484526101b1948235946024803595369594606494920191908190840183828082843750949750610c659650505050505050565b34801561040857600080fd5b50610327600435610d66565b34801561042057600080fd5b5060408051602060046024803582810135601f81018590048502860185019096528585526101b1958335953695604494919390910191908190840183828082843750949750610e0a9650505050505050565b34801561047e57600080fd5b5060408051602060046024803582810135601f81018590048502860185019096528585526101b1958335953695604494919390910191908190840183828082843750949750610f609650505050505050565b3480156104dc57600080fd5b506101b1600435602435611076565b3480156104f757600080fd5b50610503600435611157565b6040805192835260208301919091528051918290030190f35b34801561052857600080fd5b506101b1600435600160a060020a0360243516611174565b34801561054c57600080fd5b50610327600435611278565b6000600160e060020a031982167f3b3b57de0000000000000000000000000000000000000000000000000000000014806105bb5750600160e060020a031982167fd8389dc500000000000000000000000000000000000000000000000000000000145b806105ef5750600160e060020a031982167f691f343100000000000000000000000000000000000000000000000000000000145b806106235750600160e060020a031982167f2203ab5600000000000000000000000000000000000000000000000000000000145b806106575750600160e060020a031982167fc869023300000000000000000000000000000000000000000000000000000000145b8061068b5750600160e060020a031982167f59d1d43c00000000000000000000000000000000000000000000000000000000145b806106bf5750600160e060020a031982167fe89401a100000000000000000000000000000000000000000000000000000000145b806106f35750600160e060020a031982167f01ffc9a700000000000000000000000000000000000000000000000000000000145b92915050565b600080546040805160e060020a6302571be302815260048101879052905186933393600160a060020a0316926302571be39260248083019360209383900390910190829087803b15801561074c57600080fd5b505af1158015610760573d6000803e3d6000fd5b505050506040513d602081101561077657600080fd5b5051600160a060020a03161461078b57600080fd5b6000848152600160209081526040918290209151855185936005019287929182918401908083835b602083106107d25780518252601f1990920191602091820191016107b3565b51815160209384036101000a6000190180199092169116179052920194855250604051938490038101909320845161081395919491909101925090506112e7565b5083600019167fd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a75508485604051808060200180602001838103835285818151815260200191508051906020019080838360005b8381101561087d578181015183820152602001610865565b50505050905090810190601f1680156108aa5780820380516001836020036101000a031916815260200191505b50838103825284518152845160209182019186019080838360005b838110156108dd5781810151838201526020016108c5565b50505050905090810190601f16801561090a5780820380516001836020036101000a031916815260200191505b5094505050505060405180910390a250505050565b60008281526001602081905260409091206060905b838311610a1e578284161580159061096d5750600083815260068201602052604081205460026000196101006001841615020190911604115b15610a1357600083815260068201602090815260409182902080548351601f600260001961010060018616150201909316929092049182018490048402810184019094528084529091830182828015610a075780601f106109dc57610100808354040283529160200191610a07565b820191906000526020600020905b8154815290600101906020018083116109ea57829003601f168201915b50505050509150610a23565b600290920291610934565b600092505b509250929050565b600080546040805160e060020a6302571be302815260048101879052905186933393600160a060020a0316926302571be39260248083019360209383900390910190829087803b158015610a7e57600080fd5b505af1158015610a92573d6000803e3d6000fd5b505050506040513d6020811015610aa857600080fd5b5051600160a060020a031614610abd57600080fd5b604080518082018252848152602080820185815260008881526001835284902092516003840155516004909201919091558151858152908101849052815186927f1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46928290030190a250505050565b6000908152600160208190526040909120015490565b600090815260016020526040902054600160a060020a031690565b600082815260016020908152604091829020915183516060936005019285929182918401908083835b60208310610ba45780518252601f199092019160209182019101610b85565b518151600019602094850361010090810a820192831692199390931691909117909252949092019687526040805197889003820188208054601f6002600183161590980290950116959095049283018290048202880182019052818752929450925050830182828015610c585780601f10610c2d57610100808354040283529160200191610c58565b820191906000526020600020905b815481529060010190602001808311610c3b57829003601f168201915b5050505050905092915050565b600080546040805160e060020a6302571be302815260048101879052905186933393600160a060020a0316926302571be39260248083019360209383900390910190829087803b158015610cb857600080fd5b505af1158015610ccc573d6000803e3d6000fd5b505050506040513d6020811015610ce257600080fd5b5051600160a060020a031614610cf757600080fd5b6000198301831615610d0857600080fd5b600084815260016020908152604080832086845260060182529091208351610d32928501906112e7565b50604051839085907faa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe390600090a350505050565b6000818152600160208181526040928390206002908101805485516000199582161561010002959095011691909104601f81018390048302840183019094528383526060939091830182828015610dfe5780601f10610dd357610100808354040283529160200191610dfe565b820191906000526020600020905b815481529060010190602001808311610de157829003601f168201915b50505050509050919050565b600080546040805160e060020a6302571be302815260048101869052905185933393600160a060020a0316926302571be39260248083019360209383900390910190829087803b158015610e5d57600080fd5b505af1158015610e71573d6000803e3d6000fd5b505050506040513d6020811015610e8757600080fd5b5051600160a060020a031614610e9c57600080fd5b60008381526001602090815260409091208351610ec1926002909201918501906112e7565b50604080516020808252845181830152845186937fb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f79387939092839283019185019080838360005b83811015610f21578181015183820152602001610f09565b50505050905090810190601f168015610f4e5780820380516001836020036101000a031916815260200191505b509250505060405180910390a2505050565b600080546040805160e060020a6302571be302815260048101869052905185933393600160a060020a0316926302571be39260248083019360209383900390910190829087803b158015610fb357600080fd5b505af1158015610fc7573d6000803e3d6000fd5b505050506040513d6020811015610fdd57600080fd5b5051600160a060020a031614610ff257600080fd5b60008381526001602090815260409091208351611017926007909201918501906112e7565b50604080516020808252845181830152845186937fc0b0fc07269fc2749adada3221c095a1d2187b2d075b51c915857b520f3a502193879390928392830191850190808383600083811015610f21578181015183820152602001610f09565b600080546040805160e060020a6302571be302815260048101869052905185933393600160a060020a0316926302571be39260248083019360209383900390910190829087803b1580156110c957600080fd5b505af11580156110dd573d6000803e3d6000fd5b505050506040513d60208110156110f357600080fd5b5051600160a060020a03161461110857600080fd5b6000838152600160208181526040928390209091018490558151848152915185927f0424b6fe0d9c3bdbece0e7879dc241bb0c22e900be8b6c168b4ee08bd9bf83bc92908290030190a2505050565b600090815260016020526040902060038101546004909101549091565b600080546040805160e060020a6302571be302815260048101869052905185933393600160a060020a0316926302571be39260248083019360209383900390910190829087803b1580156111c757600080fd5b505af11580156111db573d6000803e3d6000fd5b505050506040513d60208110156111f157600080fd5b5051600160a060020a03161461120657600080fd5b600083815260016020908152604091829020805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0386169081179091558251908152915185927f52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd292908290030190a2505050565b60008181526001602081815260409283902060070180548451600260001995831615610100029590950190911693909304601f81018390048302840183019094528383526060939091830182828015610dfe5780601f10610dd357610100808354040283529160200191610dfe565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061132857805160ff1916838001178555611355565b82800160010185558215611355579182015b8281111561135557825182559160200191906001019061133a565b50611361929150611365565b5090565b61137f91905b80821115611361576000815560010161136b565b905600a165627a7a723058206a33cfe43406d7a114c10dfd1c93ec0606df4fbd5ff228360da4b075d19f12e00029"

// DeployPublicResolver deploys a new Ethereum contract, binding an instance of PublicResolver to it.
func DeployPublicResolver(auth *bind.TransactOpts, backend bind.ContractBackend, ensAddr common.Address) (common.Address, *types.Transaction, *PublicResolver, error) {
	parsed, err := abi.JSON(strings.NewReader(PublicResolverABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(PublicResolverBin), backend, ensAddr)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PublicResolver{PublicResolverCaller: PublicResolverCaller{contract: contract}, PublicResolverTransactor: PublicResolverTransactor{contract: contract}, PublicResolverFilterer: PublicResolverFilterer{contract: contract}}, nil
}

// PublicResolver is an auto generated Go binding around an Ethereum contract.
type PublicResolver struct {
	PublicResolverCaller     // Read-only binding to the contract
	PublicResolverTransactor // Write-only binding to the contract
	PublicResolverFilterer   // Log filterer for contract events
}

// PublicResolverCaller is an auto generated read-only Go binding around an Ethereum contract.
type PublicResolverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PublicResolverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PublicResolverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PublicResolverSession struct {
	Contract     *PublicResolver   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PublicResolverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PublicResolverCallerSession struct {
	Contract *PublicResolverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// PublicResolverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PublicResolverTransactorSession struct {
	Contract     *PublicResolverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// PublicResolverRaw is an auto generated low-level Go binding around an Ethereum contract.
type PublicResolverRaw struct {
	Contract *PublicResolver // Generic contract binding to access the raw methods on
}

// PublicResolverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PublicResolverCallerRaw struct {
	Contract *PublicResolverCaller // Generic read-only contract binding to access the raw methods on
}

// PublicResolverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PublicResolverTransactorRaw struct {
	Contract *PublicResolverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPublicResolver creates a new instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolver(address common.Address, backend bind.ContractBackend) (*PublicResolver, error) {
	contract, err := bindPublicResolver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PublicResolver{PublicResolverCaller: PublicResolverCaller{contract: contract}, PublicResolverTransactor: PublicResolverTransactor{contract: contract}, PublicResolverFilterer: PublicResolverFilterer{contract: contract}}, nil
}

// NewPublicResolverCaller creates a new read-only instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverCaller(address common.Address, caller bind.ContractCaller) (*PublicResolverCaller, error) {
	contract, err := bindPublicResolver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PublicResolverCaller{contract: contract}, nil
}

// NewPublicResolverTransactor creates a new write-only instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverTransactor(address common.Address, transactor bind.ContractTransactor) (*PublicResolverTransactor, error) {
	contract, err := bindPublicResolver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PublicResolverTransactor{contract: contract}, nil
}

// NewPublicResolverFilterer creates a new log filterer instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverFilterer(address common.Address, filterer bind.ContractFilterer) (*PublicResolverFilterer, error) {
	contract, err := bindPublicResolver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PublicResolverFilterer{contract: contract}, nil
}

// bindPublicResolver binds a generic wrapper to an already deployed contract.
func bindPublicResolver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PublicResolverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PublicResolver *PublicResolverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PublicResolver.Contract.PublicResolverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PublicResolver *PublicResolverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PublicResolver.Contract.PublicResolverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PublicResolver *PublicResolverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PublicResolver.Contract.PublicResolverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PublicResolver *PublicResolverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PublicResolver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PublicResolver *PublicResolverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PublicResolver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PublicResolver *PublicResolverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PublicResolver.Contract.contract.Transact(opts, method, params...)
}

// ABI is a free data retrieval call binding the contract method 0x2203ab56.
//
// Solidity: function ABI(bytes32 node, uint256 contentTypes) view returns(uint256 contentType, bytes data)
func (_PublicResolver *PublicResolverCaller) ABI(opts *bind.CallOpts, node [32]byte, contentTypes *big.Int) (struct {
	ContentType *big.Int
	Data        []byte
}, error) {
	var out []interface{}
	err := _PublicResolver.contract.Call(opts, &out, "ABI", node, contentTypes)

	outstruct := new(struct {
		ContentType *big.Int
		Data        []byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ContentType = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Data = *abi.ConvertType(out[1], new([]byte)).(*[]byte)

	return *outstruct, err

}

// ABI is a free data retrieval call binding the contract method 0x2203ab56.
//
// Solidity: function ABI(bytes32 node, uint256 contentTypes) view returns(uint256 contentType, bytes data)
func (_PublicResolver *PublicResolverSession) ABI(node [32]byte, contentTypes *big.Int) (struct {
	ContentType *big.Int
	Data        []byte
}, error) {
	return _PublicResolver.Contract.ABI(&_PublicResolver.CallOpts, node, contentTypes)
}

// ABI is a free data retrieval call binding the contract method 0x2203ab56.
//
// Solidity: function ABI(bytes32 node, uint256 contentTypes) view returns(uint256 contentType, bytes data)
func (_PublicResolver *PublicResolverCallerSession) ABI(node [32]byte, contentTypes *big.Int) (struct {
	ContentType *big.Int
	Data        []byte
}, error) {
	return _PublicResolver.Contract.ABI(&_PublicResolver.CallOpts, node, contentTypes)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(bytes32 node) view returns(address)
func (_PublicResolver *PublicResolverCaller) Addr(opts *bind.CallOpts, node [32]byte) (common.Address, error) {
	var out []interface{}
	err := _PublicResolver.contract.Call(opts, &out, "addr", node)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(bytes32 node) view returns(address)
func (_PublicResolver *PublicResolverSession) Addr(node [32]byte) (common.Address, error) {
	return _PublicResolver.Contract.Addr(&_PublicResolver.CallOpts, node)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(bytes32 node) view returns(address)
func (_PublicResolver *PublicResolverCallerSession) Addr(node [32]byte) (common.Address, error) {
	return _PublicResolver.Contract.Addr(&_PublicResolver.CallOpts, node)
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(bytes32 node) view returns(bytes32)
func (_PublicResolver *PublicResolverCaller) Content(opts *bind.CallOpts, node [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _PublicResolver.contract.Call(opts, &out, "content", node)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(bytes32 node) view returns(bytes32)
func (_PublicResolver *PublicResolverSession) Content(node [32]byte) ([32]byte, error) {
	return _PublicResolver.Contract.Content(&_PublicResolver.CallOpts, node)
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(bytes32 node) view returns(bytes32)
func (_PublicResolver *PublicResolverCallerSession) Content(node [32]byte) ([32]byte, error) {
	return _PublicResolver.Contract.Content(&_PublicResolver.CallOpts, node)
}

// Multihash is a free data retrieval call binding the contract method 0xe89401a1.
//
// Solidity: function multihash(bytes32 node) view returns(bytes)
func (_PublicResolver *PublicResolverCaller) Multihash(opts *bind.CallOpts, node [32]byte) ([]byte, error) {
	var out []interface{}
	err := _PublicResolver.contract.Call(opts, &out, "multihash", node)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Multihash is a free data retrieval call binding the contract method 0xe89401a1.
//
// Solidity: function multihash(bytes32 node) view returns(bytes)
func (_PublicResolver *PublicResolverSession) Multihash(node [32]byte) ([]byte, error) {
	return _PublicResolver.Contract.Multihash(&_PublicResolver.CallOpts, node)
}

// Multihash is a free data retrieval call binding the contract method 0xe89401a1.
//
// Solidity: function multihash(bytes32 node) view returns(bytes)
func (_PublicResolver *PublicResolverCallerSession) Multihash(node [32]byte) ([]byte, error) {
	return _PublicResolver.Contract.Multihash(&_PublicResolver.CallOpts, node)
}

// Name is a free data retrieval call binding the contract method 0x691f3431.
//
// Solidity: function name(bytes32 node) view returns(string)
func (_PublicResolver *PublicResolverCaller) Name(opts *bind.CallOpts, node [32]byte) (string, error) {
	var out []interface{}
	err := _PublicResolver.contract.Call(opts, &out, "name", node)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x691f3431.
//
// Solidity: function name(bytes32 node) view returns(string)
func (_PublicResolver *PublicResolverSession) Name(node [32]byte) (string, error) {
	return _PublicResolver.Contract.Name(&_PublicResolver.CallOpts, node)
}

// Name is a free data retrieval call binding the contract method 0x691f3431.
//
// Solidity: function name(bytes32 node) view returns(string)
func (_PublicResolver *PublicResolverCallerSession) Name(node [32]byte) (string, error) {
	return _PublicResolver.Contract.Name(&_PublicResolver.CallOpts, node)
}

// Pubkey is a free data retrieval call binding the contract method 0xc8690233.
//
// Solidity: function pubkey(bytes32 node) view returns(bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverCaller) Pubkey(opts *bind.CallOpts, node [32]byte) (struct {
	X [32]byte
	Y [32]byte
}, error) {
	var out []interface{}
	err := _PublicResolver.contract.Call(opts, &out, "pubkey", node)

	outstruct := new(struct {
		X [32]byte
		Y [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.X = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Y = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// Pubkey is a free data retrieval call binding the contract method 0xc8690233.
//
// Solidity: function pubkey(bytes32 node) view returns(bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverSession) Pubkey(node [32]byte) (struct {
	X [32]byte
	Y [32]byte
}, error) {
	return _PublicResolver.Contract.Pubkey(&_PublicResolver.CallOpts, node)
}

// Pubkey is a free data retrieval call binding the contract method 0xc8690233.
//
// Solidity: function pubkey(bytes32 node) view returns(bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverCallerSession) Pubkey(node [32]byte) (struct {
	X [32]byte
	Y [32]byte
}, error) {
	return _PublicResolver.Contract.Pubkey(&_PublicResolver.CallOpts, node)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceID) pure returns(bool)
func (_PublicResolver *PublicResolverCaller) SupportsInterface(opts *bind.CallOpts, interfaceID [4]byte) (bool, error) {
	var out []interface{}
	err := _PublicResolver.contract.Call(opts, &out, "supportsInterface", interfaceID)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceID) pure returns(bool)
func (_PublicResolver *PublicResolverSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _PublicResolver.Contract.SupportsInterface(&_PublicResolver.CallOpts, interfaceID)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceID) pure returns(bool)
func (_PublicResolver *PublicResolverCallerSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _PublicResolver.Contract.SupportsInterface(&_PublicResolver.CallOpts, interfaceID)
}

// Text is a free data retrieval call binding the contract method 0x59d1d43c.
//
// Solidity: function text(bytes32 node, string key) view returns(string)
func (_PublicResolver *PublicResolverCaller) Text(opts *bind.CallOpts, node [32]byte, key string) (string, error) {
	var out []interface{}
	err := _PublicResolver.contract.Call(opts, &out, "text", node, key)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Text is a free data retrieval call binding the contract method 0x59d1d43c.
//
// Solidity: function text(bytes32 node, string key) view returns(string)
func (_PublicResolver *PublicResolverSession) Text(node [32]byte, key string) (string, error) {
	return _PublicResolver.Contract.Text(&_PublicResolver.CallOpts, node, key)
}

// Text is a free data retrieval call binding the contract method 0x59d1d43c.
//
// Solidity: function text(bytes32 node, string key) view returns(string)
func (_PublicResolver *PublicResolverCallerSession) Text(node [32]byte, key string) (string, error) {
	return _PublicResolver.Contract.Text(&_PublicResolver.CallOpts, node, key)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(bytes32 node, uint256 contentType, bytes data) returns()
func (_PublicResolver *PublicResolverTransactor) SetABI(opts *bind.TransactOpts, node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setABI", node, contentType, data)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(bytes32 node, uint256 contentType, bytes data) returns()
func (_PublicResolver *PublicResolverSession) SetABI(node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetABI(&_PublicResolver.TransactOpts, node, contentType, data)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(bytes32 node, uint256 contentType, bytes data) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetABI(node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetABI(&_PublicResolver.TransactOpts, node, contentType, data)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(bytes32 node, address addr) returns()
func (_PublicResolver *PublicResolverTransactor) SetAddr(opts *bind.TransactOpts, node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setAddr", node, addr)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(bytes32 node, address addr) returns()
func (_PublicResolver *PublicResolverSession) SetAddr(node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetAddr(&_PublicResolver.TransactOpts, node, addr)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(bytes32 node, address addr) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetAddr(node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetAddr(&_PublicResolver.TransactOpts, node, addr)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(bytes32 node, bytes32 hash) returns()
func (_PublicResolver *PublicResolverTransactor) SetContent(opts *bind.TransactOpts, node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setContent", node, hash)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(bytes32 node, bytes32 hash) returns()
func (_PublicResolver *PublicResolverSession) SetContent(node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetContent(&_PublicResolver.TransactOpts, node, hash)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(bytes32 node, bytes32 hash) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetContent(node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetContent(&_PublicResolver.TransactOpts, node, hash)
}

// SetMultihash is a paid mutator transaction binding the contract method 0xaa4cb547.
//
// Solidity: function setMultihash(bytes32 node, bytes hash) returns()
func (_PublicResolver *PublicResolverTransactor) SetMultihash(opts *bind.TransactOpts, node [32]byte, hash []byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setMultihash", node, hash)
}

// SetMultihash is a paid mutator transaction binding the contract method 0xaa4cb547.
//
// Solidity: function setMultihash(bytes32 node, bytes hash) returns()
func (_PublicResolver *PublicResolverSession) SetMultihash(node [32]byte, hash []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetMultihash(&_PublicResolver.TransactOpts, node, hash)
}

// SetMultihash is a paid mutator transaction binding the contract method 0xaa4cb547.
//
// Solidity: function setMultihash(bytes32 node, bytes hash) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetMultihash(node [32]byte, hash []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetMultihash(&_PublicResolver.TransactOpts, node, hash)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(bytes32 node, string name) returns()
func (_PublicResolver *PublicResolverTransactor) SetName(opts *bind.TransactOpts, node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setName", node, name)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(bytes32 node, string name) returns()
func (_PublicResolver *PublicResolverSession) SetName(node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetName(&_PublicResolver.TransactOpts, node, name)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(bytes32 node, string name) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetName(node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetName(&_PublicResolver.TransactOpts, node, name)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(bytes32 node, bytes32 x, bytes32 y) returns()
func (_PublicResolver *PublicResolverTransactor) SetPubkey(opts *bind.TransactOpts, node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setPubkey", node, x, y)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(bytes32 node, bytes32 x, bytes32 y) returns()
func (_PublicResolver *PublicResolverSession) SetPubkey(node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetPubkey(&_PublicResolver.TransactOpts, node, x, y)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(bytes32 node, bytes32 x, bytes32 y) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetPubkey(node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetPubkey(&_PublicResolver.TransactOpts, node, x, y)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(bytes32 node, string key, string value) returns()
func (_PublicResolver *PublicResolverTransactor) SetText(opts *bind.TransactOpts, node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setText", node, key, value)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(bytes32 node, string key, string value) returns()
func (_PublicResolver *PublicResolverSession) SetText(node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetText(&_PublicResolver.TransactOpts, node, key, value)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(bytes32 node, string key, string value) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetText(node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetText(&_PublicResolver.TransactOpts, node, key, value)
}

// PublicResolverABIChangedIterator is returned from FilterABIChanged and is used to iterate over the raw logs and unpacked data for ABIChanged events raised by the PublicResolver contract.
type PublicResolverABIChangedIterator struct {
	Event *PublicResolverABIChanged // Event containing the contract specifics and raw log

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
func (it *PublicResolverABIChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverABIChanged)
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
		it.Event = new(PublicResolverABIChanged)
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
func (it *PublicResolverABIChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverABIChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverABIChanged represents a ABIChanged event raised by the PublicResolver contract.
type PublicResolverABIChanged struct {
	Node        [32]byte
	ContentType *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterABIChanged is a free log retrieval operation binding the contract event 0xaa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe3.
//
// Solidity: event ABIChanged(bytes32 indexed node, uint256 indexed contentType)
func (_PublicResolver *PublicResolverFilterer) FilterABIChanged(opts *bind.FilterOpts, node [][32]byte, contentType []*big.Int) (*PublicResolverABIChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}
	var contentTypeRule []interface{}
	for _, contentTypeItem := range contentType {
		contentTypeRule = append(contentTypeRule, contentTypeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "ABIChanged", nodeRule, contentTypeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverABIChangedIterator{contract: _PublicResolver.contract, event: "ABIChanged", logs: logs, sub: sub}, nil
}

// WatchABIChanged is a free log subscription operation binding the contract event 0xaa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe3.
//
// Solidity: event ABIChanged(bytes32 indexed node, uint256 indexed contentType)
func (_PublicResolver *PublicResolverFilterer) WatchABIChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverABIChanged, node [][32]byte, contentType []*big.Int) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}
	var contentTypeRule []interface{}
	for _, contentTypeItem := range contentType {
		contentTypeRule = append(contentTypeRule, contentTypeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "ABIChanged", nodeRule, contentTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverABIChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "ABIChanged", log); err != nil {
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

// ParseABIChanged is a log parse operation binding the contract event 0xaa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe3.
//
// Solidity: event ABIChanged(bytes32 indexed node, uint256 indexed contentType)
func (_PublicResolver *PublicResolverFilterer) ParseABIChanged(log types.Log) (*PublicResolverABIChanged, error) {
	event := new(PublicResolverABIChanged)
	if err := _PublicResolver.contract.UnpackLog(event, "ABIChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PublicResolverAddrChangedIterator is returned from FilterAddrChanged and is used to iterate over the raw logs and unpacked data for AddrChanged events raised by the PublicResolver contract.
type PublicResolverAddrChangedIterator struct {
	Event *PublicResolverAddrChanged // Event containing the contract specifics and raw log

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
func (it *PublicResolverAddrChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverAddrChanged)
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
		it.Event = new(PublicResolverAddrChanged)
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
func (it *PublicResolverAddrChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverAddrChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverAddrChanged represents a AddrChanged event raised by the PublicResolver contract.
type PublicResolverAddrChanged struct {
	Node [32]byte
	A    common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterAddrChanged is a free log retrieval operation binding the contract event 0x52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd2.
//
// Solidity: event AddrChanged(bytes32 indexed node, address a)
func (_PublicResolver *PublicResolverFilterer) FilterAddrChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverAddrChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "AddrChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverAddrChangedIterator{contract: _PublicResolver.contract, event: "AddrChanged", logs: logs, sub: sub}, nil
}

// WatchAddrChanged is a free log subscription operation binding the contract event 0x52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd2.
//
// Solidity: event AddrChanged(bytes32 indexed node, address a)
func (_PublicResolver *PublicResolverFilterer) WatchAddrChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverAddrChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "AddrChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverAddrChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "AddrChanged", log); err != nil {
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

// ParseAddrChanged is a log parse operation binding the contract event 0x52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd2.
//
// Solidity: event AddrChanged(bytes32 indexed node, address a)
func (_PublicResolver *PublicResolverFilterer) ParseAddrChanged(log types.Log) (*PublicResolverAddrChanged, error) {
	event := new(PublicResolverAddrChanged)
	if err := _PublicResolver.contract.UnpackLog(event, "AddrChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PublicResolverContentChangedIterator is returned from FilterContentChanged and is used to iterate over the raw logs and unpacked data for ContentChanged events raised by the PublicResolver contract.
type PublicResolverContentChangedIterator struct {
	Event *PublicResolverContentChanged // Event containing the contract specifics and raw log

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
func (it *PublicResolverContentChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverContentChanged)
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
		it.Event = new(PublicResolverContentChanged)
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
func (it *PublicResolverContentChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverContentChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverContentChanged represents a ContentChanged event raised by the PublicResolver contract.
type PublicResolverContentChanged struct {
	Node [32]byte
	Hash [32]byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterContentChanged is a free log retrieval operation binding the contract event 0x0424b6fe0d9c3bdbece0e7879dc241bb0c22e900be8b6c168b4ee08bd9bf83bc.
//
// Solidity: event ContentChanged(bytes32 indexed node, bytes32 hash)
func (_PublicResolver *PublicResolverFilterer) FilterContentChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverContentChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "ContentChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverContentChangedIterator{contract: _PublicResolver.contract, event: "ContentChanged", logs: logs, sub: sub}, nil
}

// WatchContentChanged is a free log subscription operation binding the contract event 0x0424b6fe0d9c3bdbece0e7879dc241bb0c22e900be8b6c168b4ee08bd9bf83bc.
//
// Solidity: event ContentChanged(bytes32 indexed node, bytes32 hash)
func (_PublicResolver *PublicResolverFilterer) WatchContentChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverContentChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "ContentChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverContentChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "ContentChanged", log); err != nil {
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

// ParseContentChanged is a log parse operation binding the contract event 0x0424b6fe0d9c3bdbece0e7879dc241bb0c22e900be8b6c168b4ee08bd9bf83bc.
//
// Solidity: event ContentChanged(bytes32 indexed node, bytes32 hash)
func (_PublicResolver *PublicResolverFilterer) ParseContentChanged(log types.Log) (*PublicResolverContentChanged, error) {
	event := new(PublicResolverContentChanged)
	if err := _PublicResolver.contract.UnpackLog(event, "ContentChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PublicResolverMultihashChangedIterator is returned from FilterMultihashChanged and is used to iterate over the raw logs and unpacked data for MultihashChanged events raised by the PublicResolver contract.
type PublicResolverMultihashChangedIterator struct {
	Event *PublicResolverMultihashChanged // Event containing the contract specifics and raw log

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
func (it *PublicResolverMultihashChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverMultihashChanged)
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
		it.Event = new(PublicResolverMultihashChanged)
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
func (it *PublicResolverMultihashChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverMultihashChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverMultihashChanged represents a MultihashChanged event raised by the PublicResolver contract.
type PublicResolverMultihashChanged struct {
	Node [32]byte
	Hash []byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterMultihashChanged is a free log retrieval operation binding the contract event 0xc0b0fc07269fc2749adada3221c095a1d2187b2d075b51c915857b520f3a5021.
//
// Solidity: event MultihashChanged(bytes32 indexed node, bytes hash)
func (_PublicResolver *PublicResolverFilterer) FilterMultihashChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverMultihashChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "MultihashChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverMultihashChangedIterator{contract: _PublicResolver.contract, event: "MultihashChanged", logs: logs, sub: sub}, nil
}

// WatchMultihashChanged is a free log subscription operation binding the contract event 0xc0b0fc07269fc2749adada3221c095a1d2187b2d075b51c915857b520f3a5021.
//
// Solidity: event MultihashChanged(bytes32 indexed node, bytes hash)
func (_PublicResolver *PublicResolverFilterer) WatchMultihashChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverMultihashChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "MultihashChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverMultihashChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "MultihashChanged", log); err != nil {
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

// ParseMultihashChanged is a log parse operation binding the contract event 0xc0b0fc07269fc2749adada3221c095a1d2187b2d075b51c915857b520f3a5021.
//
// Solidity: event MultihashChanged(bytes32 indexed node, bytes hash)
func (_PublicResolver *PublicResolverFilterer) ParseMultihashChanged(log types.Log) (*PublicResolverMultihashChanged, error) {
	event := new(PublicResolverMultihashChanged)
	if err := _PublicResolver.contract.UnpackLog(event, "MultihashChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PublicResolverNameChangedIterator is returned from FilterNameChanged and is used to iterate over the raw logs and unpacked data for NameChanged events raised by the PublicResolver contract.
type PublicResolverNameChangedIterator struct {
	Event *PublicResolverNameChanged // Event containing the contract specifics and raw log

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
func (it *PublicResolverNameChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverNameChanged)
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
		it.Event = new(PublicResolverNameChanged)
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
func (it *PublicResolverNameChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverNameChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverNameChanged represents a NameChanged event raised by the PublicResolver contract.
type PublicResolverNameChanged struct {
	Node [32]byte
	Name string
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterNameChanged is a free log retrieval operation binding the contract event 0xb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f7.
//
// Solidity: event NameChanged(bytes32 indexed node, string name)
func (_PublicResolver *PublicResolverFilterer) FilterNameChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverNameChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "NameChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverNameChangedIterator{contract: _PublicResolver.contract, event: "NameChanged", logs: logs, sub: sub}, nil
}

// WatchNameChanged is a free log subscription operation binding the contract event 0xb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f7.
//
// Solidity: event NameChanged(bytes32 indexed node, string name)
func (_PublicResolver *PublicResolverFilterer) WatchNameChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverNameChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "NameChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverNameChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "NameChanged", log); err != nil {
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

// ParseNameChanged is a log parse operation binding the contract event 0xb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f7.
//
// Solidity: event NameChanged(bytes32 indexed node, string name)
func (_PublicResolver *PublicResolverFilterer) ParseNameChanged(log types.Log) (*PublicResolverNameChanged, error) {
	event := new(PublicResolverNameChanged)
	if err := _PublicResolver.contract.UnpackLog(event, "NameChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PublicResolverPubkeyChangedIterator is returned from FilterPubkeyChanged and is used to iterate over the raw logs and unpacked data for PubkeyChanged events raised by the PublicResolver contract.
type PublicResolverPubkeyChangedIterator struct {
	Event *PublicResolverPubkeyChanged // Event containing the contract specifics and raw log

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
func (it *PublicResolverPubkeyChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverPubkeyChanged)
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
		it.Event = new(PublicResolverPubkeyChanged)
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
func (it *PublicResolverPubkeyChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverPubkeyChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverPubkeyChanged represents a PubkeyChanged event raised by the PublicResolver contract.
type PublicResolverPubkeyChanged struct {
	Node [32]byte
	X    [32]byte
	Y    [32]byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterPubkeyChanged is a free log retrieval operation binding the contract event 0x1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46.
//
// Solidity: event PubkeyChanged(bytes32 indexed node, bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverFilterer) FilterPubkeyChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverPubkeyChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "PubkeyChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverPubkeyChangedIterator{contract: _PublicResolver.contract, event: "PubkeyChanged", logs: logs, sub: sub}, nil
}

// WatchPubkeyChanged is a free log subscription operation binding the contract event 0x1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46.
//
// Solidity: event PubkeyChanged(bytes32 indexed node, bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverFilterer) WatchPubkeyChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverPubkeyChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "PubkeyChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverPubkeyChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "PubkeyChanged", log); err != nil {
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

// ParsePubkeyChanged is a log parse operation binding the contract event 0x1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46.
//
// Solidity: event PubkeyChanged(bytes32 indexed node, bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverFilterer) ParsePubkeyChanged(log types.Log) (*PublicResolverPubkeyChanged, error) {
	event := new(PublicResolverPubkeyChanged)
	if err := _PublicResolver.contract.UnpackLog(event, "PubkeyChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PublicResolverTextChangedIterator is returned from FilterTextChanged and is used to iterate over the raw logs and unpacked data for TextChanged events raised by the PublicResolver contract.
type PublicResolverTextChangedIterator struct {
	Event *PublicResolverTextChanged // Event containing the contract specifics and raw log

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
func (it *PublicResolverTextChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverTextChanged)
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
		it.Event = new(PublicResolverTextChanged)
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
func (it *PublicResolverTextChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverTextChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverTextChanged represents a TextChanged event raised by the PublicResolver contract.
type PublicResolverTextChanged struct {
	Node       [32]byte
	IndexedKey string
	Key        string
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTextChanged is a free log retrieval operation binding the contract event 0xd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a7550.
//
// Solidity: event TextChanged(bytes32 indexed node, string indexedKey, string key)
func (_PublicResolver *PublicResolverFilterer) FilterTextChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverTextChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "TextChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverTextChangedIterator{contract: _PublicResolver.contract, event: "TextChanged", logs: logs, sub: sub}, nil
}

// WatchTextChanged is a free log subscription operation binding the contract event 0xd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a7550.
//
// Solidity: event TextChanged(bytes32 indexed node, string indexedKey, string key)
func (_PublicResolver *PublicResolverFilterer) WatchTextChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverTextChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "TextChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverTextChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "TextChanged", log); err != nil {
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

// ParseTextChanged is a log parse operation binding the contract event 0xd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a7550.
//
// Solidity: event TextChanged(bytes32 indexed node, string indexedKey, string key)
func (_PublicResolver *PublicResolverFilterer) ParseTextChanged(log types.Log) (*PublicResolverTextChanged, error) {
	event := new(PublicResolverTextChanged)
	if err := _PublicResolver.contract.UnpackLog(event, "TextChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UsernameRegistrarABI is the input ABI used to generate the binding from.
const UsernameRegistrarABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"resolver\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_secret\",\"type\":\"bytes32\"}],\"name\":\"reserveSlash\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"reservedUsernamesMerkleRoot\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_beneficiary\",\"type\":\"address\"}],\"name\":\"withdrawExcessBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"}],\"name\":\"updateAccountOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newController\",\"type\":\"address\"}],\"name\":\"changeController\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_username\",\"type\":\"string\"},{\"name\":\"_offendingPos\",\"type\":\"uint256\"},{\"name\":\"_reserveSecret\",\"type\":\"uint256\"}],\"name\":\"slashInvalidUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_username\",\"type\":\"string\"},{\"name\":\"_proof\",\"type\":\"bytes32[]\"},{\"name\":\"_reserveSecret\",\"type\":\"uint256\"}],\"name\":\"slashReservedUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"reserveAmount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_resolver\",\"type\":\"address\"}],\"name\":\"setResolver\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"usernameMinLength\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"}],\"name\":\"release\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"}],\"name\":\"getCreationTime\",\"outputs\":[{\"name\":\"creationTime\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"releaseDelay\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"ensRegistry\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"},{\"name\":\"_tokenBalance\",\"type\":\"uint256\"},{\"name\":\"_creationTime\",\"type\":\"uint256\"},{\"name\":\"_accountOwner\",\"type\":\"address\"}],\"name\":\"migrateUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"}],\"name\":\"getSlashRewardPart\",\"outputs\":[{\"name\":\"partReward\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_price\",\"type\":\"uint256\"}],\"name\":\"updateRegistryPrice\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_username\",\"type\":\"string\"},{\"name\":\"_reserveSecret\",\"type\":\"uint256\"}],\"name\":\"slashAddressLikeUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"},{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"receiveApproval\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_username\",\"type\":\"string\"},{\"name\":\"_reserveSecret\",\"type\":\"uint256\"}],\"name\":\"slashSmallUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getPrice\",\"outputs\":[{\"name\":\"registryPrice\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_price\",\"type\":\"uint256\"}],\"name\":\"migrateRegistry\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"price\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"}],\"name\":\"getExpirationTime\",\"outputs\":[{\"name\":\"releaseTime\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"}],\"name\":\"getAccountOwner\",\"outputs\":[{\"name\":\"owner\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_domainHash\",\"type\":\"bytes32\"},{\"name\":\"_beneficiary\",\"type\":\"address\"}],\"name\":\"withdrawWrongNode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_price\",\"type\":\"uint256\"}],\"name\":\"activate\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"},{\"name\":\"_account\",\"type\":\"address\"},{\"name\":\"_pubkeyA\",\"type\":\"bytes32\"},{\"name\":\"_pubkeyB\",\"type\":\"bytes32\"}],\"name\":\"register\",\"outputs\":[{\"name\":\"namehash\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"accounts\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"},{\"name\":\"creationTime\",\"type\":\"uint256\"},{\"name\":\"owner\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"state\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"},{\"name\":\"_newRegistry\",\"type\":\"address\"}],\"name\":\"moveAccount\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"parentRegistry\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"ensNode\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_labels\",\"type\":\"bytes32[]\"}],\"name\":\"eraseNode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newRegistry\",\"type\":\"address\"}],\"name\":\"moveRegistry\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"}],\"name\":\"getAccountBalance\",\"outputs\":[{\"name\":\"accountBalance\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"controller\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_label\",\"type\":\"bytes32\"}],\"name\":\"dropUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_ensRegistry\",\"type\":\"address\"},{\"name\":\"_resolver\",\"type\":\"address\"},{\"name\":\"_ensNode\",\"type\":\"bytes32\"},{\"name\":\"_usernameMinLength\",\"type\":\"uint256\"},{\"name\":\"_reservedUsernamesMerkleRoot\",\"type\":\"bytes32\"},{\"name\":\"_parentRegistry\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"state\",\"type\":\"uint8\"}],\"name\":\"RegistryState\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"price\",\"type\":\"uint256\"}],\"name\":\"RegistryPrice\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"newRegistry\",\"type\":\"address\"}],\"name\":\"RegistryMoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"nameHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"UsernameOwner\",\"type\":\"event\"}]"

// UsernameRegistrarFuncSigs maps the 4-byte function signature to its string representation.
var UsernameRegistrarFuncSigs = map[string]string{
	"bc529c43": "accounts(bytes32)",
	"b260c42a": "activate(uint256)",
	"3cebb823": "changeController(address)",
	"f77c4791": "controller()",
	"f9e54282": "dropUsername(bytes32)",
	"ddbcf3a1": "ensNode()",
	"7d73b231": "ensRegistry()",
	"de10f04b": "eraseNode(bytes32[])",
	"ebf701e0": "getAccountBalance(bytes32)",
	"aacffccf": "getAccountOwner(bytes32)",
	"6f79301d": "getCreationTime(bytes32)",
	"a1454830": "getExpirationTime(bytes32)",
	"98d5fdca": "getPrice()",
	"8382b460": "getSlashRewardPart(bytes32)",
	"98f038ff": "migrateRegistry(uint256)",
	"80cd0015": "migrateUsername(bytes32,uint256,uint256,address)",
	"c23e61b9": "moveAccount(bytes32,address)",
	"e882c3ce": "moveRegistry(address)",
	"c9b84d4d": "parentRegistry()",
	"a035b1fe": "price()",
	"8f4ffcb1": "receiveApproval(address,uint256,address,bytes)",
	"b82fedbb": "register(bytes32,address,bytes32,bytes32)",
	"67d42a8b": "release(bytes32)",
	"7195bf23": "releaseDelay()",
	"4b09b72a": "reserveAmount()",
	"05c24481": "reserveSlash(bytes32)",
	"07f908cb": "reservedUsernamesMerkleRoot()",
	"04f3bcec": "resolver()",
	"4e543b26": "setResolver(address)",
	"8cf7b7a4": "slashAddressLikeUsername(string,uint256)",
	"40784ebd": "slashInvalidUsername(string,uint256,uint256)",
	"40b1ad52": "slashReservedUsername(string,bytes32[],uint256)",
	"96bba9a8": "slashSmallUsername(string,uint256)",
	"c19d93fb": "state()",
	"fc0c546a": "token()",
	"32e1ed24": "updateAccountOwner(bytes32)",
	"860e9b0f": "updateRegistryPrice(uint256)",
	"59ad0209": "usernameMinLength()",
	"307c7a0d": "withdrawExcessBalance(address,address)",
	"afe12e77": "withdrawWrongNode(bytes32,address)",
}

// UsernameRegistrarBin is the compiled bytecode used for deploying new contracts.
var UsernameRegistrarBin = "0x60806040523480156200001157600080fd5b5060405160e080620049ea83398101604090815281516020830151918301516060840151608085015160a086015160c09096015160008054600160a060020a031916331790559395929391929091600160a060020a0387161515620000d757604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601e60248201527f4e6f204552433230546f6b656e206164647265737320646566696e65642e0000604482015290519081900360640190fd5b600160a060020a03861615156200014f57604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f4e6f20454e53206164647265737320646566696e65642e000000000000000000604482015290519081900360640190fd5b600160a060020a0385161515620001c757604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601c60248201527f4e6f205265736f6c766572206164647265737320646566696e65642e00000000604482015290519081900360640190fd5b8315156200023657604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f4e6f20454e53206e6f646520646566696e65642e000000000000000000000000604482015290519081900360640190fd5b60018054600160a060020a03808a16600160a060020a03199283161790925560028054898416908316179055600380548884169083161790556009869055600785905560088490556004805492841692909116919091179055620002a46000640100000000620002b1810204565b5050505050505062000319565b600b805482919060ff19166001836002811115620002cb57fe5b02179055507fee85d4d9a9722e814f07db07f29734cd5a97e0e58781ad41ae4572193b1caea081604051808260028111156200030357fe5b60ff16815260200191505060405180910390a150565b6146c180620003296000396000f3006080604052600436106101d45763ffffffff60e060020a60003504166304f3bcec81146101d957806305c244811461020a57806307f908cb14610224578063307c7a0d1461024b57806332e1ed24146102725780633cebb8231461028a57806340784ebd146102ab57806340b1ad52146102d25780634b09b72a146103015780634e543b261461031657806359ad02091461033757806367d42a8b1461034c5780636f79301d146103645780637195bf231461037c5780637d73b2311461039157806380cd0015146103a65780638382b460146103d0578063860e9b0f146103e85780638cf7b7a4146104005780638f4ffcb11461042457806396bba9a81461049457806398d5fdca146104b857806398f038ff146104cd578063a035b1fe146104e5578063a1454830146104fa578063aacffccf14610512578063afe12e771461052a578063b260c42a1461054e578063b82fedbb14610566578063bc529c4314610590578063c19d93fb146105cf578063c23e61b914610608578063c9b84d4d1461062c578063ddbcf3a114610641578063de10f04b14610656578063e882c3ce14610676578063ebf701e014610697578063f77c4791146106af578063f9e54282146106c4578063fc0c546a146106dc575b600080fd5b3480156101e557600080fd5b506101ee6106f1565b60408051600160a060020a039092168252519081900360200190f35b34801561021657600080fd5b50610222600435610700565b005b34801561023057600080fd5b506102396107ae565b60408051918252519081900360200190f35b34801561025757600080fd5b50610222600160a060020a03600435811690602435166107b4565b34801561027e57600080fd5b50610222600435610a74565b34801561029657600080fd5b50610222600160a060020a0360043516610d60565b3480156102b757600080fd5b50610222602460048035828101929101359035604435610d99565b3480156102de57600080fd5b506102226024600480358281019290820135918135918201910135604435610f78565b34801561030d57600080fd5b506102396110a7565b34801561032257600080fd5b50610222600160a060020a03600435166110ad565b34801561034357600080fd5b506102396110e6565b34801561035857600080fd5b506102226004356110ec565b34801561037057600080fd5b506102396004356117ef565b34801561038857600080fd5b50610239611804565b34801561039d57600080fd5b506101ee61180c565b3480156103b257600080fd5b50610222600435602435604435600160a060020a036064351661181b565b3480156103dc57600080fd5b50610239600435611a09565b3480156103f457600080fd5b50610222600435611a2d565b34801561040c57600080fd5b50610222602460048035828101929101359035611ae7565b34801561043057600080fd5b50604080516020601f60643560048181013592830184900484028501840190955281845261022294600160a060020a03813581169560248035966044359093169536956084949201918190840183828082843750949750611e489650505050505050565b3480156104a057600080fd5b50610222602460048035828101929101359035612091565b3480156104c457600080fd5b5061023961212e565b3480156104d957600080fd5b50610222600435612134565b3480156104f157600080fd5b50610239612346565b34801561050657600080fd5b5061023960043561234c565b34801561051e57600080fd5b506101ee600435612372565b34801561053657600080fd5b50610222600435600160a060020a0360243516612390565b34801561055a57600080fd5b506102226004356125a7565b34801561057257600080fd5b50610239600435600160a060020a0360243516604435606435612703565b34801561059c57600080fd5b506105a860043561271b565b604080519384526020840192909252600160a060020a031682820152519081900360600190f35b3480156105db57600080fd5b506105e4612745565b604051808260028111156105f457fe5b60ff16815260200191505060405180910390f35b34801561061457600080fd5b50610222600435600160a060020a036024351661274e565b34801561063857600080fd5b506101ee612a9c565b34801561064d57600080fd5b50610239612aab565b34801561066257600080fd5b506102226004803560248101910135612ab1565b34801561068257600080fd5b50610222600160a060020a0360043516612e47565b3480156106a357600080fd5b506102396004356130bc565b3480156106bb57600080fd5b506101ee6130ce565b3480156106d057600080fd5b506102226004356130dd565b3480156106e857600080fd5b506101ee61335f565b600354600160a060020a031681565b60008181526006602052604090206001015415610767576040805160e560020a62461bcd02815260206004820152601060248201527f416c726561647920526573657276656400000000000000000000000000000000604482015290519081900360640190fd5b6040805180820182523381524360208083019182526000948552600690529190922091518254600160a060020a031916600160a060020a0390911617825551600190910155565b60085481565b600080548190600160a060020a031633146107ce57600080fd5b600160a060020a038316151561082e576040805160e560020a62461bcd02815260206004820152601160248201527f43616e6e6f74206275726e20746f6b656e000000000000000000000000000000604482015290519081900360640190fd5b600160a060020a038416151561087a57604051600160a060020a03841690303180156108fc02916000818181858888f19350505050158015610874573d6000803e3d6000fd5b50610a6e565b604080517f70a082310000000000000000000000000000000000000000000000000000000081523060048201529051859350600160a060020a038416916370a082319160248083019260209291908290030181600087803b1580156108de57600080fd5b505af11580156108f2573d6000803e3d6000fd5b505050506040513d602081101561090857600080fd5b5051600154909150600160a060020a038581169116141561098657600c54811161097c576040805160e560020a62461bcd02815260206004820152600d60248201527f4973206e6f742065786365737300000000000000000000000000000000000000604482015290519081900360640190fd5b600c5490036109de565b600081116109de576040805160e560020a62461bcd02815260206004820152600a60248201527f4e6f2062616c616e636500000000000000000000000000000000000000000000604482015290519081900360640190fd5b81600160a060020a031663a9059cbb84836040518363ffffffff1660e060020a0281526004018083600160a060020a0316600160a060020a0316815260200182815260200192505050602060405180830381600087803b158015610a4157600080fd5b505af1158015610a55573d6000803e3d6000fd5b505050506040513d6020811015610a6b57600080fd5b50505b50505050565b6009546040805160208082019390935280820184905281518082038301815260609091019182905280516000939192918291908401908083835b60208310610acd5780518252601f199092019160209182019101610aae565b51815160209384036101000a60001901801990921691161790526040805192909401829003822060025460e060020a6302571be3028452600484018290529451909750600160a060020a0390941695506302571be3945060248083019491935090918290030181600087803b158015610b4557600080fd5b505af1158015610b59573d6000803e3d6000fd5b505050506040513d6020811015610b6f57600080fd5b5051600160a060020a03163314610bd0576040805160e560020a62461bcd02815260206004820152601d60248201527f43616c6c6572206e6f74206f776e6572206f6620454e53206e6f64652e000000604482015290519081900360640190fd5b60008281526005602052604081206001015411610c37576040805160e560020a62461bcd02815260206004820152601860248201527f557365726e616d65206e6f7420726567697374657265642e0000000000000000604482015290519081900360640190fd5b6002546009546040805160e060020a6302571be30281526004810192909252513092600160a060020a0316916302571be39160248083019260209291908290030181600087803b158015610c8a57600080fd5b505af1158015610c9e573d6000803e3d6000fd5b505050506040513d6020811015610cb457600080fd5b5051600160a060020a031614610d14576040805160e560020a62461bcd02815260206004820152601f60248201527f5265676973747279206e6f74206f776e6572206f662072656769737472792e00604482015290519081900360640190fd5b6000828152600560209081526040918290206002018054600160a060020a0319163390811790915582519081529151839260008051602061467683398151915292908290030190a25050565b600054600160a060020a03163314610d7757600080fd5b60008054600160a060020a031916600160a060020a0392909216919091179055565b6060600085858080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509150838251111515610e2c576040805160e560020a62461bcd02815260206004820152601160248201527f496e76616c696420706f736974696f6e2e000000000000000000000000000000604482015290519081900360640190fd5b8184815181101515610e3a57fe5b016020015160f860020a908190040290507f3000000000000000000000000000000000000000000000000000000000000000600160f860020a0319821610801590610eaf57507f3900000000000000000000000000000000000000000000000000000000000000600160f860020a0319821611155b80610f1957507f6100000000000000000000000000000000000000000000000000000000000000600160f860020a0319821610801590610f1957507f7a00000000000000000000000000000000000000000000000000000000000000600160f860020a0319821611155b15610f6e576040805160e560020a62461bcd02815260206004820152601660248201527f4e6f7420696e76616c6964206368617261637465722e00000000000000000000604482015290519081900360640190fd5b610a6b828461336e565b606085858080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509050611047848480806020026020016040519081016040528093929190818152602001838360200280828437820191505050505050600854836040518082805190602001908083835b602083106110155780518252601f199092019160209182019101610ff6565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040518091039020613a4b565b151561109d576040805160e560020a62461bcd02815260206004820152600e60248201527f496e76616c69642050726f6f662e000000000000000000000000000000000000604482015290519081900360640190fd5b610a6b818361336e565b600c5481565b600054600160a060020a031633146110c457600080fd5b60038054600160a060020a031916600160a060020a0392909216919091179055565b60075481565b60006110f6614633565b6009546040805160208082019390935280820186905281518082038301815260609091019182905280516000939192918291908401908083835b6020831061114f5780518252601f199092019160209182019101611130565b51815160209384036101000a60001901801990921691161790526040805192909401829003822060008b81526005835285812060608501875280548552600181015493850184905260020154600160a060020a031695840195909552985090965091909111925061120d915050576040805160e560020a62461bcd02815260206004820152601860248201527f557365726e616d65206e6f7420726567697374657265642e0000000000000000604482015290519081900360640190fd5b6001600b5460ff16600281111561122057fe5b1415611368576002546040805160e060020a6302571be3028152600481018690529051600160a060020a03909216916302571be3916024808201926020929091908290030181600087803b15801561127757600080fd5b505af115801561128b573d6000803e3d6000fd5b505050506040513d60208110156112a157600080fd5b5051600160a060020a03163314611302576040805160e560020a62461bcd02815260206004820152601660248201527f4e6f74206f776e6572206f6620454e53206e6f64652e00000000000000000000604482015290519081900360640190fd5b60208201516301e13380014211611363576040805160e560020a62461bcd02815260206004820152601b60248201527f52656c6561736520706572696f64206e6f7420726561636865642e0000000000604482015290519081900360640190fd5b6113cc565b6040820151600160a060020a031633146113cc576040805160e560020a62461bcd02815260206004820152601d60248201527f4e6f742074686520666f726d6572206163636f756e74206f776e65722e000000604482015290519081900360640190fd5b6000848152600560205260408120818155600181018290556002018054600160a060020a0319169055825111156114f6578151600c80548290039055600154604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152336004820152602481019390935251600160a060020a039091169163a9059cbb9160448083019260209291908290030181600087803b15801561147457600080fd5b505af1158015611488573d6000803e3d6000fd5b505050506040513d602081101561149e57600080fd5b505115156114f6576040805160e560020a62461bcd02815260206004820152600f60248201527f5472616e73666572206661696c65640000000000000000000000000000000000604482015290519081900360640190fd5b6001600b5460ff16600281111561150957fe5b1415611664576002546009546040805160e060020a6306ab592302815260048101929092526024820187905230604483015251600160a060020a03909216916306ab59239160648082019260009290919082900301818387803b15801561156f57600080fd5b505af1158015611583573d6000803e3d6000fd5b50506002546040805160e160020a630c4b7b85028152600481018890526000602482018190529151600160a060020a039093169450631896f70a93506044808201939182900301818387803b1580156115db57600080fd5b505af11580156115ef573d6000803e3d6000fd5b50506002546040805160e060020a635b0fc9c3028152600481018890526000602482018190529151600160a060020a039093169450635b0fc9c393506044808201939182900301818387803b15801561164757600080fd5b505af115801561165b573d6000803e3d6000fd5b505050506117c4565b6002546009546040805160e060020a6302571be3028152600481019290925251600160a060020a03909216916302571be3916024808201926020929091908290030181600087803b1580156116b857600080fd5b505af11580156116cc573d6000803e3d6000fd5b505050506040513d60208110156116e257600080fd5b505160408051602480820188905282518083039091018152604490910182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167ff9e542820000000000000000000000000000000000000000000000000000000017815291518151939450600160a060020a038516936201388093829180838360005b83811015611780578181015183820152602001611768565b50505050905090810190601f1680156117ad5780820380516001836020036101000a031916815260200191505b5091505060006040518083038160008787f1505050505b604080516000815290518491600080516020614676833981519152919081900360200190a250505050565b60009081526005602052604090206001015490565b6301e1338081565b600254600160a060020a031681565b600454600160a060020a0316331461187d576040805160e560020a62461bcd02815260206004820152600f60248201527f4d6967726174696f6e206f6e6c792e0000000000000000000000000000000000604482015290519081900360640190fd5b60008311156119b05760015460048054604080517f23b872dd000000000000000000000000000000000000000000000000000000008152600160a060020a039283169381019390935230602484015260448301879052519216916323b872dd916064808201926020929091908290030181600087803b1580156118ff57600080fd5b505af1158015611913573d6000803e3d6000fd5b505050506040513d602081101561192957600080fd5b505115156119a7576040805160e560020a62461bcd02815260206004820152602560248201527f4572726f72206d6f76696e672066756e64732066726f6d206f6c64207265676960448201527f737461722e000000000000000000000000000000000000000000000000000000606482015290519081900360840190fd5b600c8054840190555b604080516060810182529384526020808501938452600160a060020a039283168583019081526000968752600590915294209251835590516001830155915160029091018054600160a060020a03191691909216179055565b60008181526005602052604081205481811115611a27576003810491505b50919050565b600054600160a060020a03163314611a4457600080fd5b6001600b5460ff166002811115611a5757fe5b14611aac576040805160e560020a62461bcd02815260206004820152601260248201527f5265676973747279206e6f74206f776e65640000000000000000000000000000604482015290519081900360640190fd5b600a8190556040805182815290517f45d3cd7c7bd7d211f00610f51660b2f114c7833e0c52ef3603c6d41ed07a74589181900360200190a150565b606060008085858080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509250600c8351111515611ba2576040805160e560020a62461bcd02815260206004820152602260248201527f546f6f20736d616c6c20746f206c6f6f6b206c696b6520616e2061646472657360448201527f732e000000000000000000000000000000000000000000000000000000000000606482015290519081900360840190fd5b82517f30000000000000000000000000000000000000000000000000000000000000009084906000908110611bd357fe5b60209101015160f860020a9081900402600160f860020a03191614611c42576040805160e560020a62461bcd02815260206004820152601c60248201527f466972737420636861726163746572206e65656420746f206265203000000000604482015290519081900360640190fd5b82517f78000000000000000000000000000000000000000000000000000000000000009084906001908110611c7357fe5b60209101015160f860020a9081900402600160f860020a03191614611ce2576040805160e560020a62461bcd02815260206004820152601d60248201527f5365636f6e6420636861726163746572206e65656420746f2062652078000000604482015290519081900360640190fd5b600291505b6007821015611e3e578282815181101515611cfe57fe5b016020015160f860020a908190040290507f3000000000000000000000000000000000000000000000000000000000000000600160f860020a0319821610801590611d7357507f3900000000000000000000000000000000000000000000000000000000000000600160f860020a0319821611155b80611ddd57507f6100000000000000000000000000000000000000000000000000000000000000600160f860020a0319821610801590611ddd57507f6600000000000000000000000000000000000000000000000000000000000000600160f860020a0319821611155b1515611e33576040805160e560020a62461bcd02815260206004820152601d60248201527f446f6573206e6f74206c6f6f6b206c696b6520616e2061646472657373000000604482015290519081900360640190fd5b600190910190611ce7565b610a6b838561336e565b6000806000806000600a5488141515611eab576040805160e560020a62461bcd02815260206004820152600b60248201527f57726f6e672076616c7565000000000000000000000000000000000000000000604482015290519081900360640190fd5b600154600160a060020a03888116911614611f10576040805160e560020a62461bcd02815260206004820152600b60248201527f57726f6e6720746f6b656e000000000000000000000000000000000000000000604482015290519081900360640190fd5b600160a060020a0387163314611f70576040805160e560020a62461bcd02815260206004820152600a60248201527f57726f6e672063616c6c00000000000000000000000000000000000000000000604482015290519081900360640190fd5b855160841015611fca576040805160e560020a62461bcd02815260206004820152601160248201527f57726f6e672064617461206c656e677468000000000000000000000000000000604482015290519081900360640190fd5b611fd386613b9a565b9398509196509450925090507fffffffff0000000000000000000000000000000000000000000000000000000085167fb82fedbb0000000000000000000000000000000000000000000000000000000014612078576040805160e560020a62461bcd02815260206004820152601560248201527f57726f6e67206d6574686f642073656c6563746f720000000000000000000000604482015290519081900360640190fd5b6120858985858585613bbe565b50505050505050505050565b606083838080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505090506007548151101515612124576040805160e560020a62461bcd02815260206004820152601560248201527f4e6f74206120736d616c6c20757365726e616d652e0000000000000000000000604482015290519081900360640190fd5b610a6e818361336e565b600a5490565b600454600160a060020a03163314612196576040805160e560020a62461bcd02815260206004820152600f60248201527f4d6967726174696f6e206f6e6c792e0000000000000000000000000000000000604482015290519081900360640190fd5b6000600b5460ff1660028111156121a957fe5b146121fe576040805160e560020a62461bcd02815260206004820152600c60248201527f4e6f7420496e6163746976650000000000000000000000000000000000000000604482015290519081900360640190fd5b6002546009546040805160e060020a6302571be30281526004810192909252513092600160a060020a0316916302571be39160248083019260209291908290030181600087803b15801561225157600080fd5b505af1158015612265573d6000803e3d6000fd5b505050506040513d602081101561227b57600080fd5b5051600160a060020a031614612301576040805160e560020a62461bcd02815260206004820152602260248201527f454e53207265676973747279206f776e6572206e6f74207472616e736665726560448201527f642e000000000000000000000000000000000000000000000000000000000000606482015290519081900360840190fd5b600a81905561231060016143aa565b6040805182815290517f45d3cd7c7bd7d211f00610f51660b2f114c7833e0c52ef3603c6d41ed07a74589181900360200190a150565b600a5481565b60008181526005602052604081206001015481811115611a27576301e133800192915050565b600090815260056020526040902060020154600160a060020a031690565b600054600160a060020a031633146123a757600080fd5b600160a060020a0381161515612407576040805160e560020a62461bcd02815260206004820152601060248201527f43616e6e6f74206275726e206e6f646500000000000000000000000000000000604482015290519081900360640190fd5b600954821415612461576040805160e560020a62461bcd02815260206004820152601960248201527f43616e6e6f74207769746864726177206d61696e206e6f646500000000000000604482015290519081900360640190fd5b6002546040805160e060020a6302571be30281526004810185905290513092600160a060020a0316916302571be39160248083019260209291908290030181600087803b1580156124b157600080fd5b505af11580156124c5573d6000803e3d6000fd5b505050506040513d60208110156124db57600080fd5b5051600160a060020a03161461253b576040805160e560020a62461bcd02815260206004820152601660248201527f4e6f74206f776e6572206f662074686973206e6f646500000000000000000000604482015290519081900360640190fd5b6002546040805160e060020a635b0fc9c302815260048101859052600160a060020a03848116602483015291519190921691635b0fc9c391604480830192600092919082900301818387803b15801561259357600080fd5b505af1158015610a6b573d6000803e3d6000fd5b600054600160a060020a031633146125be57600080fd5b6000600b5460ff1660028111156125d157fe5b14612626576040805160e560020a62461bcd02815260206004820152601e60248201527f5265676973747279207374617465206973206e6f7420496e6163746976650000604482015290519081900360640190fd5b6002546009546040805160e060020a6302571be30281526004810192909252513092600160a060020a0316916302571be39160248083019260209291908290030181600087803b15801561267957600080fd5b505af115801561268d573d6000803e3d6000fd5b505050506040513d60208110156126a357600080fd5b5051600160a060020a031614612301576040805160e560020a62461bcd02815260206004820152601e60248201527f526567697374727920646f6573206e6f74206f776e2072656769737472790000604482015290519081900360640190fd5b60006127123386868686613bbe565b95945050505050565b600560205260009081526040902080546001820154600290920154909190600160a060020a031683565b600b5460ff1681565b612756614633565b6002600b5460ff16600281111561276957fe5b146127be576040805160e560020a62461bcd02815260206004820152601460248201527f57726f6e6720636f6e7472616374207374617465000000000000000000000000604482015290519081900360640190fd5b600083815260056020526040902060020154600160a060020a0316331461282f576040805160e560020a62461bcd02815260206004820152601f60248201527f43616c6c61626c65206f6e6c79206279206163636f756e74206f776e65722e00604482015290519081900360640190fd5b6002546009546040805160e060020a6302571be3028152600481019290925251600160a060020a038086169316916302571be39160248083019260209291908290030181600087803b15801561288457600080fd5b505af1158015612898573d6000803e3d6000fd5b505050506040513d60208110156128ae57600080fd5b5051600160a060020a03161461290e576040805160e560020a62461bcd02815260206004820152600c60248201527f57726f6e67207570646174650000000000000000000000000000000000000000604482015290519081900360640190fd5b5060008281526005602081815260408084208151606081018352815481526001808301805483870152600284018054600160a060020a03808216868901528c8b529888529489905590889055600160a060020a03199093169092559054815183517f095ea7b3000000000000000000000000000000000000000000000000000000008152888716600482015260248101919091529251919594169363095ea7b393604480850194919392918390030190829087803b1580156129cf57600080fd5b505af11580156129e3573d6000803e3d6000fd5b505050506040513d60208110156129f957600080fd5b50508051602082015160408084015181517f80cd00150000000000000000000000000000000000000000000000000000000081526004810188905260248101949094526044840192909252600160a060020a03918216606484015251908416916380cd001591608480830192600092919082900301818387803b158015612a7f57600080fd5b505af1158015612a93573d6000803e3d6000fd5b50505050505050565b600454600160a060020a031681565b60095481565b80600080821515612b0c576040805160e560020a62461bcd02815260206004820152601060248201527f4e6f7468696e6720746f20657261736500000000000000000000000000000000604482015290519081900360640190fd5b84846000198501818110612b1c57fe5b6009546040805160208181019390935292820294909401358285018190528451808403860181526060909301948590528251909650919392508291908401908083835b60208310612b7e5780518252601f199092019160209182019101612b5f565b51815160209384036101000a60001901801990921691161790526040805192909401829003822060025460e060020a6302571be302845260048401829052945190975060009650600160a060020a0390941694506302571be39360248084019450919290919082900301818787803b158015612bf957600080fd5b505af1158015612c0d573d6000803e3d6000fd5b505050506040513d6020811015612c2357600080fd5b5051600160a060020a031614612ca9576040805160e560020a62461bcd02815260206004820152602760248201527f466972737420736c6173682f72656c6561736520746f70206c6576656c20737560448201527f62646f6d61696e00000000000000000000000000000000000000000000000000606482015290519081900360840190fd5b6002546009546040805160e060020a6306ab592302815260048101929092526024820185905230604483015251600160a060020a03909216916306ab59239160648082019260009290919082900301818387803b158015612d0957600080fd5b505af1158015612d1d573d6000803e3d6000fd5b505050506001831115612d6657612d6660028403868680806020026020016040519081016040528093929190818152602001838360200280828437508894506144109350505050565b6002546040805160e160020a630c4b7b85028152600481018490526000602482018190529151600160a060020a0390931692631896f70a9260448084019391929182900301818387803b158015612dbc57600080fd5b505af1158015612dd0573d6000803e3d6000fd5b50506002546040805160e060020a635b0fc9c3028152600481018690526000602482018190529151600160a060020a039093169450635b0fc9c393506044808201939182900301818387803b158015612e2857600080fd5b505af1158015612e3c573d6000803e3d6000fd5b505050505050505050565b600054600160a060020a03163314612e5e57600080fd5b600160a060020a038116301415612ebf576040805160e560020a62461bcd02815260206004820152601460248201527f43616e6e6f74206d6f766520746f2073656c662e000000000000000000000000604482015290519081900360640190fd5b6002546009546040805160e060020a6302571be30281526004810192909252513092600160a060020a0316916302571be39160248083019260209291908290030181600087803b158015612f1257600080fd5b505af1158015612f26573d6000803e3d6000fd5b505050506040513d6020811015612f3c57600080fd5b5051600160a060020a031614612f9c576040805160e560020a62461bcd02815260206004820152601b60248201527f5265676973747279206e6f74206f776e656420616e796d6f72652e0000000000604482015290519081900360640190fd5b612fa660026143aa565b6002546009546040805160e060020a635b0fc9c30281526004810192909252600160a060020a0384811660248401529051921691635b0fc9c39160448082019260009290919082900301818387803b15801561300157600080fd5b505af1158015613015573d6000803e3d6000fd5b5050505080600160a060020a03166398f038ff600a546040518263ffffffff1660e060020a02815260040180828152602001915050600060405180830381600087803b15801561306457600080fd5b505af1158015613078573d6000803e3d6000fd5b505060408051600160a060020a038516815290517fce0afb4c27dbd57a3646e2d639557521bfb05a42dc0ec50f9c1fe13d92e3e6d69350908190036020019150a150565b60009081526005602052604090205490565b600054600160a060020a031681565b600454600090600160a060020a03163314613142576040805160e560020a62461bcd02815260206004820152600f60248201527f4d6967726174696f6e206f6e6c792e0000000000000000000000000000000000604482015290519081900360640190fd5b600082815260056020526040902060010154156131a9576040805160e560020a62461bcd02815260206004820152601060248201527f416c7265616479206d6967726174656400000000000000000000000000000000604482015290519081900360640190fd5b60095460408051602080820193909352808201859052815180820383018152606090910191829052805190928291908401908083835b602083106131fe5780518252601f1990920191602091820191016131df565b5181516020939093036101000a60001901801990911692169190911790526040805191909301819003812060025460095460e060020a6306ab59230284526004840152602483018990523060448401529351909650600160a060020a0390931694506306ab59239350606480820193600093509182900301818387803b15801561328757600080fd5b505af115801561329b573d6000803e3d6000fd5b50506002546040805160e160020a630c4b7b85028152600481018690526000602482018190529151600160a060020a039093169450631896f70a93506044808201939182900301818387803b1580156132f357600080fd5b505af1158015613307573d6000803e3d6000fd5b50506002546040805160e060020a635b0fc9c3028152600481018690526000602482018190529151600160a060020a039093169450635b0fc9c393506044808201939182900301818387803b15801561259357600080fd5b600154600160a060020a031681565b600080600080600080600061338161465e565b896040518082805190602001908083835b602083106133b15780518252601f199092019160209182019101613392565b51815160209384036101000a600019018019909216911617905260408051929094018290038220600954838301528285018190528451808403860181526060909301948590528251909e509195509293508392850191508083835b6020831061342b5780518252601f19909201916020918201910161340c565b51815160209384036101000a60001901801990921691161790526040805192909401829003822060008f8152600583528581206001015460025460e060020a6302571be3028652600486018490529651929f50909d509b50600160a060020a0390941695506302571be39450602480830194919350909182900301818c87803b1580156134b757600080fd5b505af11580156134cb573d6000803e3d6000fd5b505050506040513d60208110156134e157600080fd5b505193508415156135f757600160a060020a03841615158061359c5750600254604080517f0178b8bf000000000000000000000000000000000000000000000000000000008152600481018a90529051600092600160a060020a031691630178b8bf91602480830192602092919082900301818787803b15801561356457600080fd5b505af1158015613578573d6000803e3d6000fd5b505050506040513d602081101561358e57600080fd5b5051600160a060020a031614155b15156135f2576040805160e560020a62461bcd02815260206004820152601160248201527f4e6f7468696e6720746f20736c6173682e000000000000000000000000000000604482015290519081900360640190fd5b613630565b4285141561360157fe5b6000888152600560205260408120805482825560018201929092556002018054600160a060020a031916905595505b6002546009546040805160e060020a6306ab59230281526004810192909252602482018b905230604483015251600160a060020a03909216916306ab59239160648082019260009290919082900301818387803b15801561369057600080fd5b505af11580156136a4573d6000803e3d6000fd5b50506002546040805160e160020a630c4b7b85028152600481018c90526000602482018190529151600160a060020a039093169450631896f70a93506044808201939182900301818387803b1580156136fc57600080fd5b505af1158015613710573d6000803e3d6000fd5b50506002546040805160e060020a635b0fc9c3028152600481018c90526000602482018190529151600160a060020a039093169450635b0fc9c393506044808201939182900301818387803b15801561376857600080fd5b505af115801561377c573d6000803e3d6000fd5b505050506000861115613a1a57600c805487900390556040805160208082018a905281830188905260608083018d905283518084039091018152608090920192839052815160026003909a04998a0299965091929182918401908083835b602083106137f95780518252601f1990920191602091820191016137da565b51815160209384036101000a6000190180199092169116179052604080519290940182900382206000818152600683528590208386019095528454600160a060020a0316808452600190950154918301919091529650945050151591506138ac9050576040805160e560020a62461bcd02815260206004820152600d60248201527f4e6f742072657365727665642e00000000000000000000000000000000000000604482015290519081900360640190fd5b60208101514311613907576040805160e560020a62461bcd02815260206004820152601b60248201527f43616e6e6f742072657665616c20696e2073616d6520626c6f636b0000000000604482015290519081900360640190fd5b60008281526006602090815260408083208054600160a060020a0319168155600190810184905554845182517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a039182166004820152602481018c9052925191169363a9059cbb93604480850194919392918390030190829087803b15801561399857600080fd5b505af11580156139ac573d6000803e3d6000fd5b505050506040513d60208110156139c257600080fd5b50511515613a1a576040805160e560020a62461bcd02815260206004820152601260248201527f4572726f7220696e207472616e736665722e0000000000000000000000000000604482015290519081900360640190fd5b604080516000815290518891600080516020614676833981519152919081900360200190a250505050505050505050565b60008181805b8651821015613b8d578682815181101515613a6857fe5b60209081029091010151905080831015613b0157604080516020808201869052818301849052825180830384018152606090920192839052815191929182918401908083835b60208310613acd5780518252601f199092019160209182019101613aae565b6001836020036101000a03801982511681845116808217855250505050505090500191505060405180910390209250613b82565b604080516020808201849052818301869052825180830384018152606090920192839052815191929182918401908083835b60208310613b525780518252601f199092019160209182019101613b33565b6001836020036101000a038019825116818451168082178552505050505050905001915050604051809103902092505b600190910190613a51565b5050929092149392505050565b60208101516024820151604483015160648401516084909401519294919390929091565b600080806001600b5460ff166002811115613bd557fe5b14613c2a576040805160e560020a62461bcd02815260206004820152601460248201527f5265676973747279206e6f74206163746976652e000000000000000000000000604482015290519081900360640190fd5b600954604080516020808201939093528082018a9052815180820383018152606090910191829052805190928291908401908083835b60208310613c7f5780518252601f199092019160209182019101613c60565b51815160209384036101000a60001901801990921691161790526040805192909401829003822060025460e060020a6302571be302845260048401829052945190995060009650600160a060020a0390941694506302571be39360248084019450919290919082900301818787803b158015613cfa57600080fd5b505af1158015613d0e573d6000803e3d6000fd5b505050506040513d6020811015613d2457600080fd5b5051600160a060020a031614613d84576040805160e560020a62461bcd02815260206004820152601760248201527f454e53206e6f646520616c7265616479206f776e65642e000000000000000000604482015290519081900360640190fd5b60008781526005602052604090206001015415613deb576040805160e560020a62461bcd02815260206004820152601c60248201527f557365726e616d6520616c726561647920726567697374657265642e00000000604482015290519081900360640190fd5b60408051606081018252600a80548252426020808401918252600160a060020a038d811685870190815260008e815260059093529582209451855591516001850155935160029093018054600160a060020a031916939091169290921790915554111561404d57600a54600154604080517fdd62ed3e000000000000000000000000000000000000000000000000000000008152600160a060020a038c811660048301523060248301529151919092169163dd62ed3e9160448083019260209291908290030181600087803b158015613ec357600080fd5b505af1158015613ed7573d6000803e3d6000fd5b505050506040513d6020811015613eed57600080fd5b50511015613f45576040805160e560020a62461bcd02815260206004820152601360248201527f556e616c6c6f77656420746f207370656e642e00000000000000000000000000604482015290519081900360640190fd5b600154600a54604080517f23b872dd000000000000000000000000000000000000000000000000000000008152600160a060020a038c811660048301523060248301526044820193909352905191909216916323b872dd9160648083019260209291908290030181600087803b158015613fbe57600080fd5b505af1158015613fd2573d6000803e3d6000fd5b505050506040513d6020811015613fe857600080fd5b50511515614040576040805160e560020a62461bcd02815260206004820152600f60248201527f5472616e73666572206661696c65640000000000000000000000000000000000604482015290519081900360640190fd5b600a54600c805490910190555b8415158061405a57508315155b915050600160a060020a038516151581806140725750805b156142f7576002546009546040805160e060020a6306ab59230281526004810192909252602482018a905230604483015251600160a060020a03909216916306ab59239160648082019260009290919082900301818387803b1580156140d757600080fd5b505af11580156140eb573d6000803e3d6000fd5b50506002546003546040805160e160020a630c4b7b8502815260048101899052600160a060020a0392831660248201529051919092169350631896f70a9250604480830192600092919082900301818387803b15801561414a57600080fd5b505af115801561415e573d6000803e3d6000fd5b5050505080156141ef57600354604080517fd5fa2b0000000000000000000000000000000000000000000000000000000000815260048101869052600160a060020a0389811660248301529151919092169163d5fa2b0091604480830192600092919082900301818387803b1580156141d657600080fd5b505af11580156141ea573d6000803e3d6000fd5b505050505b811561428257600354604080517f29cd62ea0000000000000000000000000000000000000000000000000000000081526004810186905260248101889052604481018790529051600160a060020a03909216916329cd62ea9160648082019260009290919082900301818387803b15801561426957600080fd5b505af115801561427d573d6000803e3d6000fd5b505050505b6002546040805160e060020a635b0fc9c302815260048101869052600160a060020a038b8116602483015291519190921691635b0fc9c391604480830192600092919082900301818387803b1580156142da57600080fd5b505af11580156142ee573d6000803e3d6000fd5b50505050614372565b6002546009546040805160e060020a6306ab59230281526004810192909252602482018a9052600160a060020a038b8116604484015290519216916306ab59239160648082019260009290919082900301818387803b15801561435957600080fd5b505af115801561436d573d6000803e3d6000fd5b505050505b60408051600160a060020a038a16815290518491600080516020614676833981519152919081900360200190a2505095945050505050565b600b805482919060ff191660018360028111156143c357fe5b02179055507fee85d4d9a9722e814f07db07f29734cd5a97e0e58781ad41ae4572193b1caea081604051808260028111156143fa57fe5b60ff16815260200191505060405180910390a150565b6002548251600091600160a060020a0316906306ab592390849086908890811061443657fe5b602090810290910101516040805160e060020a63ffffffff86160281526004810193909352602483019190915230604483015251606480830192600092919082900301818387803b15801561448a57600080fd5b505af115801561449e573d6000803e3d6000fd5b505050508183858151811015156144b157fe5b6020908102909101810151604080518084019490945283810191909152805180840382018152606090930190819052825190918291908401908083835b6020831061450d5780518252601f1990920191602091820191016144ee565b6001836020036101000a03801982511681845116808217855250505050505090500191505060405180910390209050600084111561455357614553600185038483614410565b6002546040805160e160020a630c4b7b85028152600481018490526000602482018190529151600160a060020a0390931692631896f70a9260448084019391929182900301818387803b1580156145a957600080fd5b505af11580156145bd573d6000803e3d6000fd5b50506002546040805160e060020a635b0fc9c3028152600481018690526000602482018190529151600160a060020a039093169450635b0fc9c393506044808201939182900301818387803b15801561461557600080fd5b505af1158015614629573d6000803e3d6000fd5b5050505050505050565b60606040519081016040528060008152602001600081526020016000600160a060020a031681525090565b6040805180820190915260008082526020820152905600d2da4206c3fa95b8fc1ee48627023d322b59cc7218e14cb95cf0c0fe562f2e4da165627a7a723058205994f6efc637dc93ed1eed6bdd349793d213589282d72f178a75fdf6fef07d760029"

// DeployUsernameRegistrar deploys a new Ethereum contract, binding an instance of UsernameRegistrar to it.
func DeployUsernameRegistrar(auth *bind.TransactOpts, backend bind.ContractBackend, _token common.Address, _ensRegistry common.Address, _resolver common.Address, _ensNode [32]byte, _usernameMinLength *big.Int, _reservedUsernamesMerkleRoot [32]byte, _parentRegistry common.Address) (common.Address, *types.Transaction, *UsernameRegistrar, error) {
	parsed, err := abi.JSON(strings.NewReader(UsernameRegistrarABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(UsernameRegistrarBin), backend, _token, _ensRegistry, _resolver, _ensNode, _usernameMinLength, _reservedUsernamesMerkleRoot, _parentRegistry)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &UsernameRegistrar{UsernameRegistrarCaller: UsernameRegistrarCaller{contract: contract}, UsernameRegistrarTransactor: UsernameRegistrarTransactor{contract: contract}, UsernameRegistrarFilterer: UsernameRegistrarFilterer{contract: contract}}, nil
}

// UsernameRegistrar is an auto generated Go binding around an Ethereum contract.
type UsernameRegistrar struct {
	UsernameRegistrarCaller     // Read-only binding to the contract
	UsernameRegistrarTransactor // Write-only binding to the contract
	UsernameRegistrarFilterer   // Log filterer for contract events
}

// UsernameRegistrarCaller is an auto generated read-only Go binding around an Ethereum contract.
type UsernameRegistrarCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UsernameRegistrarTransactor is an auto generated write-only Go binding around an Ethereum contract.
type UsernameRegistrarTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UsernameRegistrarFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type UsernameRegistrarFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UsernameRegistrarSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type UsernameRegistrarSession struct {
	Contract     *UsernameRegistrar // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// UsernameRegistrarCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type UsernameRegistrarCallerSession struct {
	Contract *UsernameRegistrarCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// UsernameRegistrarTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type UsernameRegistrarTransactorSession struct {
	Contract     *UsernameRegistrarTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// UsernameRegistrarRaw is an auto generated low-level Go binding around an Ethereum contract.
type UsernameRegistrarRaw struct {
	Contract *UsernameRegistrar // Generic contract binding to access the raw methods on
}

// UsernameRegistrarCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type UsernameRegistrarCallerRaw struct {
	Contract *UsernameRegistrarCaller // Generic read-only contract binding to access the raw methods on
}

// UsernameRegistrarTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type UsernameRegistrarTransactorRaw struct {
	Contract *UsernameRegistrarTransactor // Generic write-only contract binding to access the raw methods on
}

// NewUsernameRegistrar creates a new instance of UsernameRegistrar, bound to a specific deployed contract.
func NewUsernameRegistrar(address common.Address, backend bind.ContractBackend) (*UsernameRegistrar, error) {
	contract, err := bindUsernameRegistrar(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &UsernameRegistrar{UsernameRegistrarCaller: UsernameRegistrarCaller{contract: contract}, UsernameRegistrarTransactor: UsernameRegistrarTransactor{contract: contract}, UsernameRegistrarFilterer: UsernameRegistrarFilterer{contract: contract}}, nil
}

// NewUsernameRegistrarCaller creates a new read-only instance of UsernameRegistrar, bound to a specific deployed contract.
func NewUsernameRegistrarCaller(address common.Address, caller bind.ContractCaller) (*UsernameRegistrarCaller, error) {
	contract, err := bindUsernameRegistrar(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &UsernameRegistrarCaller{contract: contract}, nil
}

// NewUsernameRegistrarTransactor creates a new write-only instance of UsernameRegistrar, bound to a specific deployed contract.
func NewUsernameRegistrarTransactor(address common.Address, transactor bind.ContractTransactor) (*UsernameRegistrarTransactor, error) {
	contract, err := bindUsernameRegistrar(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &UsernameRegistrarTransactor{contract: contract}, nil
}

// NewUsernameRegistrarFilterer creates a new log filterer instance of UsernameRegistrar, bound to a specific deployed contract.
func NewUsernameRegistrarFilterer(address common.Address, filterer bind.ContractFilterer) (*UsernameRegistrarFilterer, error) {
	contract, err := bindUsernameRegistrar(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &UsernameRegistrarFilterer{contract: contract}, nil
}

// bindUsernameRegistrar binds a generic wrapper to an already deployed contract.
func bindUsernameRegistrar(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(UsernameRegistrarABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_UsernameRegistrar *UsernameRegistrarRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _UsernameRegistrar.Contract.UsernameRegistrarCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_UsernameRegistrar *UsernameRegistrarRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.UsernameRegistrarTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_UsernameRegistrar *UsernameRegistrarRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.UsernameRegistrarTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_UsernameRegistrar *UsernameRegistrarCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _UsernameRegistrar.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_UsernameRegistrar *UsernameRegistrarTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_UsernameRegistrar *UsernameRegistrarTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.contract.Transact(opts, method, params...)
}

// Accounts is a free data retrieval call binding the contract method 0xbc529c43.
//
// Solidity: function accounts(bytes32 ) view returns(uint256 balance, uint256 creationTime, address owner)
func (_UsernameRegistrar *UsernameRegistrarCaller) Accounts(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Balance      *big.Int
	CreationTime *big.Int
	Owner        common.Address
}, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "accounts", arg0)

	outstruct := new(struct {
		Balance      *big.Int
		CreationTime *big.Int
		Owner        common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Balance = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.CreationTime = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Owner = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// Accounts is a free data retrieval call binding the contract method 0xbc529c43.
//
// Solidity: function accounts(bytes32 ) view returns(uint256 balance, uint256 creationTime, address owner)
func (_UsernameRegistrar *UsernameRegistrarSession) Accounts(arg0 [32]byte) (struct {
	Balance      *big.Int
	CreationTime *big.Int
	Owner        common.Address
}, error) {
	return _UsernameRegistrar.Contract.Accounts(&_UsernameRegistrar.CallOpts, arg0)
}

// Accounts is a free data retrieval call binding the contract method 0xbc529c43.
//
// Solidity: function accounts(bytes32 ) view returns(uint256 balance, uint256 creationTime, address owner)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) Accounts(arg0 [32]byte) (struct {
	Balance      *big.Int
	CreationTime *big.Int
	Owner        common.Address
}, error) {
	return _UsernameRegistrar.Contract.Accounts(&_UsernameRegistrar.CallOpts, arg0)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCaller) Controller(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "controller")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarSession) Controller() (common.Address, error) {
	return _UsernameRegistrar.Contract.Controller(&_UsernameRegistrar.CallOpts)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) Controller() (common.Address, error) {
	return _UsernameRegistrar.Contract.Controller(&_UsernameRegistrar.CallOpts)
}

// EnsNode is a free data retrieval call binding the contract method 0xddbcf3a1.
//
// Solidity: function ensNode() view returns(bytes32)
func (_UsernameRegistrar *UsernameRegistrarCaller) EnsNode(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "ensNode")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// EnsNode is a free data retrieval call binding the contract method 0xddbcf3a1.
//
// Solidity: function ensNode() view returns(bytes32)
func (_UsernameRegistrar *UsernameRegistrarSession) EnsNode() ([32]byte, error) {
	return _UsernameRegistrar.Contract.EnsNode(&_UsernameRegistrar.CallOpts)
}

// EnsNode is a free data retrieval call binding the contract method 0xddbcf3a1.
//
// Solidity: function ensNode() view returns(bytes32)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) EnsNode() ([32]byte, error) {
	return _UsernameRegistrar.Contract.EnsNode(&_UsernameRegistrar.CallOpts)
}

// EnsRegistry is a free data retrieval call binding the contract method 0x7d73b231.
//
// Solidity: function ensRegistry() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCaller) EnsRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "ensRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// EnsRegistry is a free data retrieval call binding the contract method 0x7d73b231.
//
// Solidity: function ensRegistry() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarSession) EnsRegistry() (common.Address, error) {
	return _UsernameRegistrar.Contract.EnsRegistry(&_UsernameRegistrar.CallOpts)
}

// EnsRegistry is a free data retrieval call binding the contract method 0x7d73b231.
//
// Solidity: function ensRegistry() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) EnsRegistry() (common.Address, error) {
	return _UsernameRegistrar.Contract.EnsRegistry(&_UsernameRegistrar.CallOpts)
}

// GetAccountBalance is a free data retrieval call binding the contract method 0xebf701e0.
//
// Solidity: function getAccountBalance(bytes32 _label) view returns(uint256 accountBalance)
func (_UsernameRegistrar *UsernameRegistrarCaller) GetAccountBalance(opts *bind.CallOpts, _label [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "getAccountBalance", _label)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAccountBalance is a free data retrieval call binding the contract method 0xebf701e0.
//
// Solidity: function getAccountBalance(bytes32 _label) view returns(uint256 accountBalance)
func (_UsernameRegistrar *UsernameRegistrarSession) GetAccountBalance(_label [32]byte) (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetAccountBalance(&_UsernameRegistrar.CallOpts, _label)
}

// GetAccountBalance is a free data retrieval call binding the contract method 0xebf701e0.
//
// Solidity: function getAccountBalance(bytes32 _label) view returns(uint256 accountBalance)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) GetAccountBalance(_label [32]byte) (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetAccountBalance(&_UsernameRegistrar.CallOpts, _label)
}

// GetAccountOwner is a free data retrieval call binding the contract method 0xaacffccf.
//
// Solidity: function getAccountOwner(bytes32 _label) view returns(address owner)
func (_UsernameRegistrar *UsernameRegistrarCaller) GetAccountOwner(opts *bind.CallOpts, _label [32]byte) (common.Address, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "getAccountOwner", _label)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetAccountOwner is a free data retrieval call binding the contract method 0xaacffccf.
//
// Solidity: function getAccountOwner(bytes32 _label) view returns(address owner)
func (_UsernameRegistrar *UsernameRegistrarSession) GetAccountOwner(_label [32]byte) (common.Address, error) {
	return _UsernameRegistrar.Contract.GetAccountOwner(&_UsernameRegistrar.CallOpts, _label)
}

// GetAccountOwner is a free data retrieval call binding the contract method 0xaacffccf.
//
// Solidity: function getAccountOwner(bytes32 _label) view returns(address owner)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) GetAccountOwner(_label [32]byte) (common.Address, error) {
	return _UsernameRegistrar.Contract.GetAccountOwner(&_UsernameRegistrar.CallOpts, _label)
}

// GetCreationTime is a free data retrieval call binding the contract method 0x6f79301d.
//
// Solidity: function getCreationTime(bytes32 _label) view returns(uint256 creationTime)
func (_UsernameRegistrar *UsernameRegistrarCaller) GetCreationTime(opts *bind.CallOpts, _label [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "getCreationTime", _label)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCreationTime is a free data retrieval call binding the contract method 0x6f79301d.
//
// Solidity: function getCreationTime(bytes32 _label) view returns(uint256 creationTime)
func (_UsernameRegistrar *UsernameRegistrarSession) GetCreationTime(_label [32]byte) (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetCreationTime(&_UsernameRegistrar.CallOpts, _label)
}

// GetCreationTime is a free data retrieval call binding the contract method 0x6f79301d.
//
// Solidity: function getCreationTime(bytes32 _label) view returns(uint256 creationTime)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) GetCreationTime(_label [32]byte) (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetCreationTime(&_UsernameRegistrar.CallOpts, _label)
}

// GetExpirationTime is a free data retrieval call binding the contract method 0xa1454830.
//
// Solidity: function getExpirationTime(bytes32 _label) view returns(uint256 releaseTime)
func (_UsernameRegistrar *UsernameRegistrarCaller) GetExpirationTime(opts *bind.CallOpts, _label [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "getExpirationTime", _label)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetExpirationTime is a free data retrieval call binding the contract method 0xa1454830.
//
// Solidity: function getExpirationTime(bytes32 _label) view returns(uint256 releaseTime)
func (_UsernameRegistrar *UsernameRegistrarSession) GetExpirationTime(_label [32]byte) (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetExpirationTime(&_UsernameRegistrar.CallOpts, _label)
}

// GetExpirationTime is a free data retrieval call binding the contract method 0xa1454830.
//
// Solidity: function getExpirationTime(bytes32 _label) view returns(uint256 releaseTime)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) GetExpirationTime(_label [32]byte) (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetExpirationTime(&_UsernameRegistrar.CallOpts, _label)
}

// GetPrice is a free data retrieval call binding the contract method 0x98d5fdca.
//
// Solidity: function getPrice() view returns(uint256 registryPrice)
func (_UsernameRegistrar *UsernameRegistrarCaller) GetPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "getPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetPrice is a free data retrieval call binding the contract method 0x98d5fdca.
//
// Solidity: function getPrice() view returns(uint256 registryPrice)
func (_UsernameRegistrar *UsernameRegistrarSession) GetPrice() (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetPrice(&_UsernameRegistrar.CallOpts)
}

// GetPrice is a free data retrieval call binding the contract method 0x98d5fdca.
//
// Solidity: function getPrice() view returns(uint256 registryPrice)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) GetPrice() (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetPrice(&_UsernameRegistrar.CallOpts)
}

// GetSlashRewardPart is a free data retrieval call binding the contract method 0x8382b460.
//
// Solidity: function getSlashRewardPart(bytes32 _label) view returns(uint256 partReward)
func (_UsernameRegistrar *UsernameRegistrarCaller) GetSlashRewardPart(opts *bind.CallOpts, _label [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "getSlashRewardPart", _label)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetSlashRewardPart is a free data retrieval call binding the contract method 0x8382b460.
//
// Solidity: function getSlashRewardPart(bytes32 _label) view returns(uint256 partReward)
func (_UsernameRegistrar *UsernameRegistrarSession) GetSlashRewardPart(_label [32]byte) (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetSlashRewardPart(&_UsernameRegistrar.CallOpts, _label)
}

// GetSlashRewardPart is a free data retrieval call binding the contract method 0x8382b460.
//
// Solidity: function getSlashRewardPart(bytes32 _label) view returns(uint256 partReward)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) GetSlashRewardPart(_label [32]byte) (*big.Int, error) {
	return _UsernameRegistrar.Contract.GetSlashRewardPart(&_UsernameRegistrar.CallOpts, _label)
}

// ParentRegistry is a free data retrieval call binding the contract method 0xc9b84d4d.
//
// Solidity: function parentRegistry() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCaller) ParentRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "parentRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ParentRegistry is a free data retrieval call binding the contract method 0xc9b84d4d.
//
// Solidity: function parentRegistry() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarSession) ParentRegistry() (common.Address, error) {
	return _UsernameRegistrar.Contract.ParentRegistry(&_UsernameRegistrar.CallOpts)
}

// ParentRegistry is a free data retrieval call binding the contract method 0xc9b84d4d.
//
// Solidity: function parentRegistry() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) ParentRegistry() (common.Address, error) {
	return _UsernameRegistrar.Contract.ParentRegistry(&_UsernameRegistrar.CallOpts)
}

// Price is a free data retrieval call binding the contract method 0xa035b1fe.
//
// Solidity: function price() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarCaller) Price(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "price")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Price is a free data retrieval call binding the contract method 0xa035b1fe.
//
// Solidity: function price() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarSession) Price() (*big.Int, error) {
	return _UsernameRegistrar.Contract.Price(&_UsernameRegistrar.CallOpts)
}

// Price is a free data retrieval call binding the contract method 0xa035b1fe.
//
// Solidity: function price() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) Price() (*big.Int, error) {
	return _UsernameRegistrar.Contract.Price(&_UsernameRegistrar.CallOpts)
}

// ReleaseDelay is a free data retrieval call binding the contract method 0x7195bf23.
//
// Solidity: function releaseDelay() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarCaller) ReleaseDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "releaseDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReleaseDelay is a free data retrieval call binding the contract method 0x7195bf23.
//
// Solidity: function releaseDelay() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarSession) ReleaseDelay() (*big.Int, error) {
	return _UsernameRegistrar.Contract.ReleaseDelay(&_UsernameRegistrar.CallOpts)
}

// ReleaseDelay is a free data retrieval call binding the contract method 0x7195bf23.
//
// Solidity: function releaseDelay() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) ReleaseDelay() (*big.Int, error) {
	return _UsernameRegistrar.Contract.ReleaseDelay(&_UsernameRegistrar.CallOpts)
}

// ReserveAmount is a free data retrieval call binding the contract method 0x4b09b72a.
//
// Solidity: function reserveAmount() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarCaller) ReserveAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "reserveAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReserveAmount is a free data retrieval call binding the contract method 0x4b09b72a.
//
// Solidity: function reserveAmount() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarSession) ReserveAmount() (*big.Int, error) {
	return _UsernameRegistrar.Contract.ReserveAmount(&_UsernameRegistrar.CallOpts)
}

// ReserveAmount is a free data retrieval call binding the contract method 0x4b09b72a.
//
// Solidity: function reserveAmount() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) ReserveAmount() (*big.Int, error) {
	return _UsernameRegistrar.Contract.ReserveAmount(&_UsernameRegistrar.CallOpts)
}

// ReservedUsernamesMerkleRoot is a free data retrieval call binding the contract method 0x07f908cb.
//
// Solidity: function reservedUsernamesMerkleRoot() view returns(bytes32)
func (_UsernameRegistrar *UsernameRegistrarCaller) ReservedUsernamesMerkleRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "reservedUsernamesMerkleRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ReservedUsernamesMerkleRoot is a free data retrieval call binding the contract method 0x07f908cb.
//
// Solidity: function reservedUsernamesMerkleRoot() view returns(bytes32)
func (_UsernameRegistrar *UsernameRegistrarSession) ReservedUsernamesMerkleRoot() ([32]byte, error) {
	return _UsernameRegistrar.Contract.ReservedUsernamesMerkleRoot(&_UsernameRegistrar.CallOpts)
}

// ReservedUsernamesMerkleRoot is a free data retrieval call binding the contract method 0x07f908cb.
//
// Solidity: function reservedUsernamesMerkleRoot() view returns(bytes32)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) ReservedUsernamesMerkleRoot() ([32]byte, error) {
	return _UsernameRegistrar.Contract.ReservedUsernamesMerkleRoot(&_UsernameRegistrar.CallOpts)
}

// Resolver is a free data retrieval call binding the contract method 0x04f3bcec.
//
// Solidity: function resolver() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCaller) Resolver(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "resolver")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Resolver is a free data retrieval call binding the contract method 0x04f3bcec.
//
// Solidity: function resolver() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarSession) Resolver() (common.Address, error) {
	return _UsernameRegistrar.Contract.Resolver(&_UsernameRegistrar.CallOpts)
}

// Resolver is a free data retrieval call binding the contract method 0x04f3bcec.
//
// Solidity: function resolver() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) Resolver() (common.Address, error) {
	return _UsernameRegistrar.Contract.Resolver(&_UsernameRegistrar.CallOpts)
}

// State is a free data retrieval call binding the contract method 0xc19d93fb.
//
// Solidity: function state() view returns(uint8)
func (_UsernameRegistrar *UsernameRegistrarCaller) State(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "state")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// State is a free data retrieval call binding the contract method 0xc19d93fb.
//
// Solidity: function state() view returns(uint8)
func (_UsernameRegistrar *UsernameRegistrarSession) State() (uint8, error) {
	return _UsernameRegistrar.Contract.State(&_UsernameRegistrar.CallOpts)
}

// State is a free data retrieval call binding the contract method 0xc19d93fb.
//
// Solidity: function state() view returns(uint8)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) State() (uint8, error) {
	return _UsernameRegistrar.Contract.State(&_UsernameRegistrar.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarSession) Token() (common.Address, error) {
	return _UsernameRegistrar.Contract.Token(&_UsernameRegistrar.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) Token() (common.Address, error) {
	return _UsernameRegistrar.Contract.Token(&_UsernameRegistrar.CallOpts)
}

// UsernameMinLength is a free data retrieval call binding the contract method 0x59ad0209.
//
// Solidity: function usernameMinLength() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarCaller) UsernameMinLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _UsernameRegistrar.contract.Call(opts, &out, "usernameMinLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// UsernameMinLength is a free data retrieval call binding the contract method 0x59ad0209.
//
// Solidity: function usernameMinLength() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarSession) UsernameMinLength() (*big.Int, error) {
	return _UsernameRegistrar.Contract.UsernameMinLength(&_UsernameRegistrar.CallOpts)
}

// UsernameMinLength is a free data retrieval call binding the contract method 0x59ad0209.
//
// Solidity: function usernameMinLength() view returns(uint256)
func (_UsernameRegistrar *UsernameRegistrarCallerSession) UsernameMinLength() (*big.Int, error) {
	return _UsernameRegistrar.Contract.UsernameMinLength(&_UsernameRegistrar.CallOpts)
}

// Activate is a paid mutator transaction binding the contract method 0xb260c42a.
//
// Solidity: function activate(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) Activate(opts *bind.TransactOpts, _price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "activate", _price)
}

// Activate is a paid mutator transaction binding the contract method 0xb260c42a.
//
// Solidity: function activate(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) Activate(_price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.Activate(&_UsernameRegistrar.TransactOpts, _price)
}

// Activate is a paid mutator transaction binding the contract method 0xb260c42a.
//
// Solidity: function activate(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) Activate(_price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.Activate(&_UsernameRegistrar.TransactOpts, _price)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) ChangeController(opts *bind.TransactOpts, _newController common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "changeController", _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.ChangeController(&_UsernameRegistrar.TransactOpts, _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.ChangeController(&_UsernameRegistrar.TransactOpts, _newController)
}

// DropUsername is a paid mutator transaction binding the contract method 0xf9e54282.
//
// Solidity: function dropUsername(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) DropUsername(opts *bind.TransactOpts, _label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "dropUsername", _label)
}

// DropUsername is a paid mutator transaction binding the contract method 0xf9e54282.
//
// Solidity: function dropUsername(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) DropUsername(_label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.DropUsername(&_UsernameRegistrar.TransactOpts, _label)
}

// DropUsername is a paid mutator transaction binding the contract method 0xf9e54282.
//
// Solidity: function dropUsername(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) DropUsername(_label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.DropUsername(&_UsernameRegistrar.TransactOpts, _label)
}

// EraseNode is a paid mutator transaction binding the contract method 0xde10f04b.
//
// Solidity: function eraseNode(bytes32[] _labels) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) EraseNode(opts *bind.TransactOpts, _labels [][32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "eraseNode", _labels)
}

// EraseNode is a paid mutator transaction binding the contract method 0xde10f04b.
//
// Solidity: function eraseNode(bytes32[] _labels) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) EraseNode(_labels [][32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.EraseNode(&_UsernameRegistrar.TransactOpts, _labels)
}

// EraseNode is a paid mutator transaction binding the contract method 0xde10f04b.
//
// Solidity: function eraseNode(bytes32[] _labels) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) EraseNode(_labels [][32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.EraseNode(&_UsernameRegistrar.TransactOpts, _labels)
}

// MigrateRegistry is a paid mutator transaction binding the contract method 0x98f038ff.
//
// Solidity: function migrateRegistry(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) MigrateRegistry(opts *bind.TransactOpts, _price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "migrateRegistry", _price)
}

// MigrateRegistry is a paid mutator transaction binding the contract method 0x98f038ff.
//
// Solidity: function migrateRegistry(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) MigrateRegistry(_price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.MigrateRegistry(&_UsernameRegistrar.TransactOpts, _price)
}

// MigrateRegistry is a paid mutator transaction binding the contract method 0x98f038ff.
//
// Solidity: function migrateRegistry(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) MigrateRegistry(_price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.MigrateRegistry(&_UsernameRegistrar.TransactOpts, _price)
}

// MigrateUsername is a paid mutator transaction binding the contract method 0x80cd0015.
//
// Solidity: function migrateUsername(bytes32 _label, uint256 _tokenBalance, uint256 _creationTime, address _accountOwner) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) MigrateUsername(opts *bind.TransactOpts, _label [32]byte, _tokenBalance *big.Int, _creationTime *big.Int, _accountOwner common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "migrateUsername", _label, _tokenBalance, _creationTime, _accountOwner)
}

// MigrateUsername is a paid mutator transaction binding the contract method 0x80cd0015.
//
// Solidity: function migrateUsername(bytes32 _label, uint256 _tokenBalance, uint256 _creationTime, address _accountOwner) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) MigrateUsername(_label [32]byte, _tokenBalance *big.Int, _creationTime *big.Int, _accountOwner common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.MigrateUsername(&_UsernameRegistrar.TransactOpts, _label, _tokenBalance, _creationTime, _accountOwner)
}

// MigrateUsername is a paid mutator transaction binding the contract method 0x80cd0015.
//
// Solidity: function migrateUsername(bytes32 _label, uint256 _tokenBalance, uint256 _creationTime, address _accountOwner) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) MigrateUsername(_label [32]byte, _tokenBalance *big.Int, _creationTime *big.Int, _accountOwner common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.MigrateUsername(&_UsernameRegistrar.TransactOpts, _label, _tokenBalance, _creationTime, _accountOwner)
}

// MoveAccount is a paid mutator transaction binding the contract method 0xc23e61b9.
//
// Solidity: function moveAccount(bytes32 _label, address _newRegistry) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) MoveAccount(opts *bind.TransactOpts, _label [32]byte, _newRegistry common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "moveAccount", _label, _newRegistry)
}

// MoveAccount is a paid mutator transaction binding the contract method 0xc23e61b9.
//
// Solidity: function moveAccount(bytes32 _label, address _newRegistry) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) MoveAccount(_label [32]byte, _newRegistry common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.MoveAccount(&_UsernameRegistrar.TransactOpts, _label, _newRegistry)
}

// MoveAccount is a paid mutator transaction binding the contract method 0xc23e61b9.
//
// Solidity: function moveAccount(bytes32 _label, address _newRegistry) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) MoveAccount(_label [32]byte, _newRegistry common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.MoveAccount(&_UsernameRegistrar.TransactOpts, _label, _newRegistry)
}

// MoveRegistry is a paid mutator transaction binding the contract method 0xe882c3ce.
//
// Solidity: function moveRegistry(address _newRegistry) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) MoveRegistry(opts *bind.TransactOpts, _newRegistry common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "moveRegistry", _newRegistry)
}

// MoveRegistry is a paid mutator transaction binding the contract method 0xe882c3ce.
//
// Solidity: function moveRegistry(address _newRegistry) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) MoveRegistry(_newRegistry common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.MoveRegistry(&_UsernameRegistrar.TransactOpts, _newRegistry)
}

// MoveRegistry is a paid mutator transaction binding the contract method 0xe882c3ce.
//
// Solidity: function moveRegistry(address _newRegistry) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) MoveRegistry(_newRegistry common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.MoveRegistry(&_UsernameRegistrar.TransactOpts, _newRegistry)
}

// ReceiveApproval is a paid mutator transaction binding the contract method 0x8f4ffcb1.
//
// Solidity: function receiveApproval(address _from, uint256 _amount, address _token, bytes _data) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) ReceiveApproval(opts *bind.TransactOpts, _from common.Address, _amount *big.Int, _token common.Address, _data []byte) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "receiveApproval", _from, _amount, _token, _data)
}

// ReceiveApproval is a paid mutator transaction binding the contract method 0x8f4ffcb1.
//
// Solidity: function receiveApproval(address _from, uint256 _amount, address _token, bytes _data) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) ReceiveApproval(_from common.Address, _amount *big.Int, _token common.Address, _data []byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.ReceiveApproval(&_UsernameRegistrar.TransactOpts, _from, _amount, _token, _data)
}

// ReceiveApproval is a paid mutator transaction binding the contract method 0x8f4ffcb1.
//
// Solidity: function receiveApproval(address _from, uint256 _amount, address _token, bytes _data) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) ReceiveApproval(_from common.Address, _amount *big.Int, _token common.Address, _data []byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.ReceiveApproval(&_UsernameRegistrar.TransactOpts, _from, _amount, _token, _data)
}

// Register is a paid mutator transaction binding the contract method 0xb82fedbb.
//
// Solidity: function register(bytes32 _label, address _account, bytes32 _pubkeyA, bytes32 _pubkeyB) returns(bytes32 namehash)
func (_UsernameRegistrar *UsernameRegistrarTransactor) Register(opts *bind.TransactOpts, _label [32]byte, _account common.Address, _pubkeyA [32]byte, _pubkeyB [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "register", _label, _account, _pubkeyA, _pubkeyB)
}

// Register is a paid mutator transaction binding the contract method 0xb82fedbb.
//
// Solidity: function register(bytes32 _label, address _account, bytes32 _pubkeyA, bytes32 _pubkeyB) returns(bytes32 namehash)
func (_UsernameRegistrar *UsernameRegistrarSession) Register(_label [32]byte, _account common.Address, _pubkeyA [32]byte, _pubkeyB [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.Register(&_UsernameRegistrar.TransactOpts, _label, _account, _pubkeyA, _pubkeyB)
}

// Register is a paid mutator transaction binding the contract method 0xb82fedbb.
//
// Solidity: function register(bytes32 _label, address _account, bytes32 _pubkeyA, bytes32 _pubkeyB) returns(bytes32 namehash)
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) Register(_label [32]byte, _account common.Address, _pubkeyA [32]byte, _pubkeyB [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.Register(&_UsernameRegistrar.TransactOpts, _label, _account, _pubkeyA, _pubkeyB)
}

// Release is a paid mutator transaction binding the contract method 0x67d42a8b.
//
// Solidity: function release(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) Release(opts *bind.TransactOpts, _label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "release", _label)
}

// Release is a paid mutator transaction binding the contract method 0x67d42a8b.
//
// Solidity: function release(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) Release(_label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.Release(&_UsernameRegistrar.TransactOpts, _label)
}

// Release is a paid mutator transaction binding the contract method 0x67d42a8b.
//
// Solidity: function release(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) Release(_label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.Release(&_UsernameRegistrar.TransactOpts, _label)
}

// ReserveSlash is a paid mutator transaction binding the contract method 0x05c24481.
//
// Solidity: function reserveSlash(bytes32 _secret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) ReserveSlash(opts *bind.TransactOpts, _secret [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "reserveSlash", _secret)
}

// ReserveSlash is a paid mutator transaction binding the contract method 0x05c24481.
//
// Solidity: function reserveSlash(bytes32 _secret) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) ReserveSlash(_secret [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.ReserveSlash(&_UsernameRegistrar.TransactOpts, _secret)
}

// ReserveSlash is a paid mutator transaction binding the contract method 0x05c24481.
//
// Solidity: function reserveSlash(bytes32 _secret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) ReserveSlash(_secret [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.ReserveSlash(&_UsernameRegistrar.TransactOpts, _secret)
}

// SetResolver is a paid mutator transaction binding the contract method 0x4e543b26.
//
// Solidity: function setResolver(address _resolver) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) SetResolver(opts *bind.TransactOpts, _resolver common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "setResolver", _resolver)
}

// SetResolver is a paid mutator transaction binding the contract method 0x4e543b26.
//
// Solidity: function setResolver(address _resolver) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) SetResolver(_resolver common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SetResolver(&_UsernameRegistrar.TransactOpts, _resolver)
}

// SetResolver is a paid mutator transaction binding the contract method 0x4e543b26.
//
// Solidity: function setResolver(address _resolver) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) SetResolver(_resolver common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SetResolver(&_UsernameRegistrar.TransactOpts, _resolver)
}

// SlashAddressLikeUsername is a paid mutator transaction binding the contract method 0x8cf7b7a4.
//
// Solidity: function slashAddressLikeUsername(string _username, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) SlashAddressLikeUsername(opts *bind.TransactOpts, _username string, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "slashAddressLikeUsername", _username, _reserveSecret)
}

// SlashAddressLikeUsername is a paid mutator transaction binding the contract method 0x8cf7b7a4.
//
// Solidity: function slashAddressLikeUsername(string _username, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) SlashAddressLikeUsername(_username string, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SlashAddressLikeUsername(&_UsernameRegistrar.TransactOpts, _username, _reserveSecret)
}

// SlashAddressLikeUsername is a paid mutator transaction binding the contract method 0x8cf7b7a4.
//
// Solidity: function slashAddressLikeUsername(string _username, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) SlashAddressLikeUsername(_username string, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SlashAddressLikeUsername(&_UsernameRegistrar.TransactOpts, _username, _reserveSecret)
}

// SlashInvalidUsername is a paid mutator transaction binding the contract method 0x40784ebd.
//
// Solidity: function slashInvalidUsername(string _username, uint256 _offendingPos, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) SlashInvalidUsername(opts *bind.TransactOpts, _username string, _offendingPos *big.Int, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "slashInvalidUsername", _username, _offendingPos, _reserveSecret)
}

// SlashInvalidUsername is a paid mutator transaction binding the contract method 0x40784ebd.
//
// Solidity: function slashInvalidUsername(string _username, uint256 _offendingPos, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) SlashInvalidUsername(_username string, _offendingPos *big.Int, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SlashInvalidUsername(&_UsernameRegistrar.TransactOpts, _username, _offendingPos, _reserveSecret)
}

// SlashInvalidUsername is a paid mutator transaction binding the contract method 0x40784ebd.
//
// Solidity: function slashInvalidUsername(string _username, uint256 _offendingPos, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) SlashInvalidUsername(_username string, _offendingPos *big.Int, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SlashInvalidUsername(&_UsernameRegistrar.TransactOpts, _username, _offendingPos, _reserveSecret)
}

// SlashReservedUsername is a paid mutator transaction binding the contract method 0x40b1ad52.
//
// Solidity: function slashReservedUsername(string _username, bytes32[] _proof, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) SlashReservedUsername(opts *bind.TransactOpts, _username string, _proof [][32]byte, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "slashReservedUsername", _username, _proof, _reserveSecret)
}

// SlashReservedUsername is a paid mutator transaction binding the contract method 0x40b1ad52.
//
// Solidity: function slashReservedUsername(string _username, bytes32[] _proof, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) SlashReservedUsername(_username string, _proof [][32]byte, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SlashReservedUsername(&_UsernameRegistrar.TransactOpts, _username, _proof, _reserveSecret)
}

// SlashReservedUsername is a paid mutator transaction binding the contract method 0x40b1ad52.
//
// Solidity: function slashReservedUsername(string _username, bytes32[] _proof, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) SlashReservedUsername(_username string, _proof [][32]byte, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SlashReservedUsername(&_UsernameRegistrar.TransactOpts, _username, _proof, _reserveSecret)
}

// SlashSmallUsername is a paid mutator transaction binding the contract method 0x96bba9a8.
//
// Solidity: function slashSmallUsername(string _username, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) SlashSmallUsername(opts *bind.TransactOpts, _username string, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "slashSmallUsername", _username, _reserveSecret)
}

// SlashSmallUsername is a paid mutator transaction binding the contract method 0x96bba9a8.
//
// Solidity: function slashSmallUsername(string _username, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) SlashSmallUsername(_username string, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SlashSmallUsername(&_UsernameRegistrar.TransactOpts, _username, _reserveSecret)
}

// SlashSmallUsername is a paid mutator transaction binding the contract method 0x96bba9a8.
//
// Solidity: function slashSmallUsername(string _username, uint256 _reserveSecret) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) SlashSmallUsername(_username string, _reserveSecret *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.SlashSmallUsername(&_UsernameRegistrar.TransactOpts, _username, _reserveSecret)
}

// UpdateAccountOwner is a paid mutator transaction binding the contract method 0x32e1ed24.
//
// Solidity: function updateAccountOwner(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) UpdateAccountOwner(opts *bind.TransactOpts, _label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "updateAccountOwner", _label)
}

// UpdateAccountOwner is a paid mutator transaction binding the contract method 0x32e1ed24.
//
// Solidity: function updateAccountOwner(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) UpdateAccountOwner(_label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.UpdateAccountOwner(&_UsernameRegistrar.TransactOpts, _label)
}

// UpdateAccountOwner is a paid mutator transaction binding the contract method 0x32e1ed24.
//
// Solidity: function updateAccountOwner(bytes32 _label) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) UpdateAccountOwner(_label [32]byte) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.UpdateAccountOwner(&_UsernameRegistrar.TransactOpts, _label)
}

// UpdateRegistryPrice is a paid mutator transaction binding the contract method 0x860e9b0f.
//
// Solidity: function updateRegistryPrice(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) UpdateRegistryPrice(opts *bind.TransactOpts, _price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "updateRegistryPrice", _price)
}

// UpdateRegistryPrice is a paid mutator transaction binding the contract method 0x860e9b0f.
//
// Solidity: function updateRegistryPrice(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) UpdateRegistryPrice(_price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.UpdateRegistryPrice(&_UsernameRegistrar.TransactOpts, _price)
}

// UpdateRegistryPrice is a paid mutator transaction binding the contract method 0x860e9b0f.
//
// Solidity: function updateRegistryPrice(uint256 _price) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) UpdateRegistryPrice(_price *big.Int) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.UpdateRegistryPrice(&_UsernameRegistrar.TransactOpts, _price)
}

// WithdrawExcessBalance is a paid mutator transaction binding the contract method 0x307c7a0d.
//
// Solidity: function withdrawExcessBalance(address _token, address _beneficiary) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) WithdrawExcessBalance(opts *bind.TransactOpts, _token common.Address, _beneficiary common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "withdrawExcessBalance", _token, _beneficiary)
}

// WithdrawExcessBalance is a paid mutator transaction binding the contract method 0x307c7a0d.
//
// Solidity: function withdrawExcessBalance(address _token, address _beneficiary) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) WithdrawExcessBalance(_token common.Address, _beneficiary common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.WithdrawExcessBalance(&_UsernameRegistrar.TransactOpts, _token, _beneficiary)
}

// WithdrawExcessBalance is a paid mutator transaction binding the contract method 0x307c7a0d.
//
// Solidity: function withdrawExcessBalance(address _token, address _beneficiary) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) WithdrawExcessBalance(_token common.Address, _beneficiary common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.WithdrawExcessBalance(&_UsernameRegistrar.TransactOpts, _token, _beneficiary)
}

// WithdrawWrongNode is a paid mutator transaction binding the contract method 0xafe12e77.
//
// Solidity: function withdrawWrongNode(bytes32 _domainHash, address _beneficiary) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactor) WithdrawWrongNode(opts *bind.TransactOpts, _domainHash [32]byte, _beneficiary common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.contract.Transact(opts, "withdrawWrongNode", _domainHash, _beneficiary)
}

// WithdrawWrongNode is a paid mutator transaction binding the contract method 0xafe12e77.
//
// Solidity: function withdrawWrongNode(bytes32 _domainHash, address _beneficiary) returns()
func (_UsernameRegistrar *UsernameRegistrarSession) WithdrawWrongNode(_domainHash [32]byte, _beneficiary common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.WithdrawWrongNode(&_UsernameRegistrar.TransactOpts, _domainHash, _beneficiary)
}

// WithdrawWrongNode is a paid mutator transaction binding the contract method 0xafe12e77.
//
// Solidity: function withdrawWrongNode(bytes32 _domainHash, address _beneficiary) returns()
func (_UsernameRegistrar *UsernameRegistrarTransactorSession) WithdrawWrongNode(_domainHash [32]byte, _beneficiary common.Address) (*types.Transaction, error) {
	return _UsernameRegistrar.Contract.WithdrawWrongNode(&_UsernameRegistrar.TransactOpts, _domainHash, _beneficiary)
}

// UsernameRegistrarRegistryMovedIterator is returned from FilterRegistryMoved and is used to iterate over the raw logs and unpacked data for RegistryMoved events raised by the UsernameRegistrar contract.
type UsernameRegistrarRegistryMovedIterator struct {
	Event *UsernameRegistrarRegistryMoved // Event containing the contract specifics and raw log

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
func (it *UsernameRegistrarRegistryMovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UsernameRegistrarRegistryMoved)
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
		it.Event = new(UsernameRegistrarRegistryMoved)
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
func (it *UsernameRegistrarRegistryMovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UsernameRegistrarRegistryMovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UsernameRegistrarRegistryMoved represents a RegistryMoved event raised by the UsernameRegistrar contract.
type UsernameRegistrarRegistryMoved struct {
	NewRegistry common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterRegistryMoved is a free log retrieval operation binding the contract event 0xce0afb4c27dbd57a3646e2d639557521bfb05a42dc0ec50f9c1fe13d92e3e6d6.
//
// Solidity: event RegistryMoved(address newRegistry)
func (_UsernameRegistrar *UsernameRegistrarFilterer) FilterRegistryMoved(opts *bind.FilterOpts) (*UsernameRegistrarRegistryMovedIterator, error) {

	logs, sub, err := _UsernameRegistrar.contract.FilterLogs(opts, "RegistryMoved")
	if err != nil {
		return nil, err
	}
	return &UsernameRegistrarRegistryMovedIterator{contract: _UsernameRegistrar.contract, event: "RegistryMoved", logs: logs, sub: sub}, nil
}

// WatchRegistryMoved is a free log subscription operation binding the contract event 0xce0afb4c27dbd57a3646e2d639557521bfb05a42dc0ec50f9c1fe13d92e3e6d6.
//
// Solidity: event RegistryMoved(address newRegistry)
func (_UsernameRegistrar *UsernameRegistrarFilterer) WatchRegistryMoved(opts *bind.WatchOpts, sink chan<- *UsernameRegistrarRegistryMoved) (event.Subscription, error) {

	logs, sub, err := _UsernameRegistrar.contract.WatchLogs(opts, "RegistryMoved")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UsernameRegistrarRegistryMoved)
				if err := _UsernameRegistrar.contract.UnpackLog(event, "RegistryMoved", log); err != nil {
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

// ParseRegistryMoved is a log parse operation binding the contract event 0xce0afb4c27dbd57a3646e2d639557521bfb05a42dc0ec50f9c1fe13d92e3e6d6.
//
// Solidity: event RegistryMoved(address newRegistry)
func (_UsernameRegistrar *UsernameRegistrarFilterer) ParseRegistryMoved(log types.Log) (*UsernameRegistrarRegistryMoved, error) {
	event := new(UsernameRegistrarRegistryMoved)
	if err := _UsernameRegistrar.contract.UnpackLog(event, "RegistryMoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UsernameRegistrarRegistryPriceIterator is returned from FilterRegistryPrice and is used to iterate over the raw logs and unpacked data for RegistryPrice events raised by the UsernameRegistrar contract.
type UsernameRegistrarRegistryPriceIterator struct {
	Event *UsernameRegistrarRegistryPrice // Event containing the contract specifics and raw log

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
func (it *UsernameRegistrarRegistryPriceIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UsernameRegistrarRegistryPrice)
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
		it.Event = new(UsernameRegistrarRegistryPrice)
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
func (it *UsernameRegistrarRegistryPriceIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UsernameRegistrarRegistryPriceIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UsernameRegistrarRegistryPrice represents a RegistryPrice event raised by the UsernameRegistrar contract.
type UsernameRegistrarRegistryPrice struct {
	Price *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterRegistryPrice is a free log retrieval operation binding the contract event 0x45d3cd7c7bd7d211f00610f51660b2f114c7833e0c52ef3603c6d41ed07a7458.
//
// Solidity: event RegistryPrice(uint256 price)
func (_UsernameRegistrar *UsernameRegistrarFilterer) FilterRegistryPrice(opts *bind.FilterOpts) (*UsernameRegistrarRegistryPriceIterator, error) {

	logs, sub, err := _UsernameRegistrar.contract.FilterLogs(opts, "RegistryPrice")
	if err != nil {
		return nil, err
	}
	return &UsernameRegistrarRegistryPriceIterator{contract: _UsernameRegistrar.contract, event: "RegistryPrice", logs: logs, sub: sub}, nil
}

// WatchRegistryPrice is a free log subscription operation binding the contract event 0x45d3cd7c7bd7d211f00610f51660b2f114c7833e0c52ef3603c6d41ed07a7458.
//
// Solidity: event RegistryPrice(uint256 price)
func (_UsernameRegistrar *UsernameRegistrarFilterer) WatchRegistryPrice(opts *bind.WatchOpts, sink chan<- *UsernameRegistrarRegistryPrice) (event.Subscription, error) {

	logs, sub, err := _UsernameRegistrar.contract.WatchLogs(opts, "RegistryPrice")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UsernameRegistrarRegistryPrice)
				if err := _UsernameRegistrar.contract.UnpackLog(event, "RegistryPrice", log); err != nil {
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

// ParseRegistryPrice is a log parse operation binding the contract event 0x45d3cd7c7bd7d211f00610f51660b2f114c7833e0c52ef3603c6d41ed07a7458.
//
// Solidity: event RegistryPrice(uint256 price)
func (_UsernameRegistrar *UsernameRegistrarFilterer) ParseRegistryPrice(log types.Log) (*UsernameRegistrarRegistryPrice, error) {
	event := new(UsernameRegistrarRegistryPrice)
	if err := _UsernameRegistrar.contract.UnpackLog(event, "RegistryPrice", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UsernameRegistrarRegistryStateIterator is returned from FilterRegistryState and is used to iterate over the raw logs and unpacked data for RegistryState events raised by the UsernameRegistrar contract.
type UsernameRegistrarRegistryStateIterator struct {
	Event *UsernameRegistrarRegistryState // Event containing the contract specifics and raw log

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
func (it *UsernameRegistrarRegistryStateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UsernameRegistrarRegistryState)
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
		it.Event = new(UsernameRegistrarRegistryState)
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
func (it *UsernameRegistrarRegistryStateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UsernameRegistrarRegistryStateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UsernameRegistrarRegistryState represents a RegistryState event raised by the UsernameRegistrar contract.
type UsernameRegistrarRegistryState struct {
	State uint8
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterRegistryState is a free log retrieval operation binding the contract event 0xee85d4d9a9722e814f07db07f29734cd5a97e0e58781ad41ae4572193b1caea0.
//
// Solidity: event RegistryState(uint8 state)
func (_UsernameRegistrar *UsernameRegistrarFilterer) FilterRegistryState(opts *bind.FilterOpts) (*UsernameRegistrarRegistryStateIterator, error) {

	logs, sub, err := _UsernameRegistrar.contract.FilterLogs(opts, "RegistryState")
	if err != nil {
		return nil, err
	}
	return &UsernameRegistrarRegistryStateIterator{contract: _UsernameRegistrar.contract, event: "RegistryState", logs: logs, sub: sub}, nil
}

// WatchRegistryState is a free log subscription operation binding the contract event 0xee85d4d9a9722e814f07db07f29734cd5a97e0e58781ad41ae4572193b1caea0.
//
// Solidity: event RegistryState(uint8 state)
func (_UsernameRegistrar *UsernameRegistrarFilterer) WatchRegistryState(opts *bind.WatchOpts, sink chan<- *UsernameRegistrarRegistryState) (event.Subscription, error) {

	logs, sub, err := _UsernameRegistrar.contract.WatchLogs(opts, "RegistryState")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UsernameRegistrarRegistryState)
				if err := _UsernameRegistrar.contract.UnpackLog(event, "RegistryState", log); err != nil {
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

// ParseRegistryState is a log parse operation binding the contract event 0xee85d4d9a9722e814f07db07f29734cd5a97e0e58781ad41ae4572193b1caea0.
//
// Solidity: event RegistryState(uint8 state)
func (_UsernameRegistrar *UsernameRegistrarFilterer) ParseRegistryState(log types.Log) (*UsernameRegistrarRegistryState, error) {
	event := new(UsernameRegistrarRegistryState)
	if err := _UsernameRegistrar.contract.UnpackLog(event, "RegistryState", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UsernameRegistrarUsernameOwnerIterator is returned from FilterUsernameOwner and is used to iterate over the raw logs and unpacked data for UsernameOwner events raised by the UsernameRegistrar contract.
type UsernameRegistrarUsernameOwnerIterator struct {
	Event *UsernameRegistrarUsernameOwner // Event containing the contract specifics and raw log

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
func (it *UsernameRegistrarUsernameOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UsernameRegistrarUsernameOwner)
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
		it.Event = new(UsernameRegistrarUsernameOwner)
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
func (it *UsernameRegistrarUsernameOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UsernameRegistrarUsernameOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UsernameRegistrarUsernameOwner represents a UsernameOwner event raised by the UsernameRegistrar contract.
type UsernameRegistrarUsernameOwner struct {
	NameHash [32]byte
	Owner    common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterUsernameOwner is a free log retrieval operation binding the contract event 0xd2da4206c3fa95b8fc1ee48627023d322b59cc7218e14cb95cf0c0fe562f2e4d.
//
// Solidity: event UsernameOwner(bytes32 indexed nameHash, address owner)
func (_UsernameRegistrar *UsernameRegistrarFilterer) FilterUsernameOwner(opts *bind.FilterOpts, nameHash [][32]byte) (*UsernameRegistrarUsernameOwnerIterator, error) {

	var nameHashRule []interface{}
	for _, nameHashItem := range nameHash {
		nameHashRule = append(nameHashRule, nameHashItem)
	}

	logs, sub, err := _UsernameRegistrar.contract.FilterLogs(opts, "UsernameOwner", nameHashRule)
	if err != nil {
		return nil, err
	}
	return &UsernameRegistrarUsernameOwnerIterator{contract: _UsernameRegistrar.contract, event: "UsernameOwner", logs: logs, sub: sub}, nil
}

// WatchUsernameOwner is a free log subscription operation binding the contract event 0xd2da4206c3fa95b8fc1ee48627023d322b59cc7218e14cb95cf0c0fe562f2e4d.
//
// Solidity: event UsernameOwner(bytes32 indexed nameHash, address owner)
func (_UsernameRegistrar *UsernameRegistrarFilterer) WatchUsernameOwner(opts *bind.WatchOpts, sink chan<- *UsernameRegistrarUsernameOwner, nameHash [][32]byte) (event.Subscription, error) {

	var nameHashRule []interface{}
	for _, nameHashItem := range nameHash {
		nameHashRule = append(nameHashRule, nameHashItem)
	}

	logs, sub, err := _UsernameRegistrar.contract.WatchLogs(opts, "UsernameOwner", nameHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UsernameRegistrarUsernameOwner)
				if err := _UsernameRegistrar.contract.UnpackLog(event, "UsernameOwner", log); err != nil {
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

// ParseUsernameOwner is a log parse operation binding the contract event 0xd2da4206c3fa95b8fc1ee48627023d322b59cc7218e14cb95cf0c0fe562f2e4d.
//
// Solidity: event UsernameOwner(bytes32 indexed nameHash, address owner)
func (_UsernameRegistrar *UsernameRegistrarFilterer) ParseUsernameOwner(log types.Log) (*UsernameRegistrarUsernameOwner, error) {
	event := new(UsernameRegistrarUsernameOwner)
	if err := _UsernameRegistrar.contract.UnpackLog(event, "UsernameOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
