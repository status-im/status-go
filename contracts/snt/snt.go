// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package snt

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

// ContributionWalletABI is the input ABI used to generate the binding from.
const ContributionWalletABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"endBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"multisig\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"contribution\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_multisig\",\"type\":\"address\"},{\"name\":\"_endBlock\",\"type\":\"uint256\"},{\"name\":\"_contribution\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"}]"

// ContributionWalletFuncSigs maps the 4-byte function signature to its string representation.
var ContributionWalletFuncSigs = map[string]string{
	"50520b1f": "contribution()",
	"083c6323": "endBlock()",
	"4783c35b": "multisig()",
	"3ccfd60b": "withdraw()",
}

// ContributionWalletBin is the compiled bytecode used for deploying new contracts.
var ContributionWalletBin = "0x608060405234801561001057600080fd5b506040516060806102f2833981016040908152815160208301519190920151600160a060020a038316151561004457600080fd5b600160a060020a038116151561005957600080fd5b811580159061006b5750623d09008211155b151561007657600080fd5b60008054600160a060020a03948516600160a060020a0319918216179091556001929092556002805491909316911617905561023b806100b76000396000f3006080604052600436106100615763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663083c632381146100635780633ccfd60b1461008a5780634783c35b1461009f57806350520b1f146100d0575b005b34801561006f57600080fd5b506100786100e5565b60408051918252519081900360200190f35b34801561009657600080fd5b506100616100eb565b3480156100ab57600080fd5b506100b46101f1565b60408051600160a060020a039092168252519081900360200190f35b3480156100dc57600080fd5b506100b4610200565b60015481565b600054600160a060020a0316331461010257600080fd5b6001544311806101a85750600260009054906101000a9004600160a060020a0316600160a060020a0316634084c3ab6040518163ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401602060405180830381600087803b15801561017957600080fd5b505af115801561018d573d6000803e3d6000fd5b505050506040513d60208110156101a357600080fd5b505115155b15156101b357600080fd5b60008054604051600160a060020a0390911691303180156108fc02929091818181858888f193505050501580156101ee573d6000803e3d6000fd5b50565b600054600160a060020a031681565b600254600160a060020a0316815600a165627a7a7230582056f60400b31557ebe53e444ebec3f314a0749cf9087fb8ff277f2e83cf277bed0029"

// DeployContributionWallet deploys a new Ethereum contract, binding an instance of ContributionWallet to it.
func DeployContributionWallet(auth *bind.TransactOpts, backend bind.ContractBackend, _multisig common.Address, _endBlock *big.Int, _contribution common.Address) (common.Address, *types.Transaction, *ContributionWallet, error) {
	parsed, err := abi.JSON(strings.NewReader(ContributionWalletABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ContributionWalletBin), backend, _multisig, _endBlock, _contribution)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ContributionWallet{ContributionWalletCaller: ContributionWalletCaller{contract: contract}, ContributionWalletTransactor: ContributionWalletTransactor{contract: contract}, ContributionWalletFilterer: ContributionWalletFilterer{contract: contract}}, nil
}

// ContributionWallet is an auto generated Go binding around an Ethereum contract.
type ContributionWallet struct {
	ContributionWalletCaller     // Read-only binding to the contract
	ContributionWalletTransactor // Write-only binding to the contract
	ContributionWalletFilterer   // Log filterer for contract events
}

// ContributionWalletCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContributionWalletCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContributionWalletTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContributionWalletTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContributionWalletFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContributionWalletFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContributionWalletSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContributionWalletSession struct {
	Contract     *ContributionWallet // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ContributionWalletCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContributionWalletCallerSession struct {
	Contract *ContributionWalletCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// ContributionWalletTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContributionWalletTransactorSession struct {
	Contract     *ContributionWalletTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// ContributionWalletRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContributionWalletRaw struct {
	Contract *ContributionWallet // Generic contract binding to access the raw methods on
}

// ContributionWalletCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContributionWalletCallerRaw struct {
	Contract *ContributionWalletCaller // Generic read-only contract binding to access the raw methods on
}

// ContributionWalletTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContributionWalletTransactorRaw struct {
	Contract *ContributionWalletTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContributionWallet creates a new instance of ContributionWallet, bound to a specific deployed contract.
func NewContributionWallet(address common.Address, backend bind.ContractBackend) (*ContributionWallet, error) {
	contract, err := bindContributionWallet(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ContributionWallet{ContributionWalletCaller: ContributionWalletCaller{contract: contract}, ContributionWalletTransactor: ContributionWalletTransactor{contract: contract}, ContributionWalletFilterer: ContributionWalletFilterer{contract: contract}}, nil
}

// NewContributionWalletCaller creates a new read-only instance of ContributionWallet, bound to a specific deployed contract.
func NewContributionWalletCaller(address common.Address, caller bind.ContractCaller) (*ContributionWalletCaller, error) {
	contract, err := bindContributionWallet(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContributionWalletCaller{contract: contract}, nil
}

// NewContributionWalletTransactor creates a new write-only instance of ContributionWallet, bound to a specific deployed contract.
func NewContributionWalletTransactor(address common.Address, transactor bind.ContractTransactor) (*ContributionWalletTransactor, error) {
	contract, err := bindContributionWallet(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContributionWalletTransactor{contract: contract}, nil
}

// NewContributionWalletFilterer creates a new log filterer instance of ContributionWallet, bound to a specific deployed contract.
func NewContributionWalletFilterer(address common.Address, filterer bind.ContractFilterer) (*ContributionWalletFilterer, error) {
	contract, err := bindContributionWallet(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContributionWalletFilterer{contract: contract}, nil
}

// bindContributionWallet binds a generic wrapper to an already deployed contract.
func bindContributionWallet(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ContributionWalletABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ContributionWallet *ContributionWalletRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ContributionWallet.Contract.ContributionWalletCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ContributionWallet *ContributionWalletRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ContributionWallet.Contract.ContributionWalletTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ContributionWallet *ContributionWalletRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ContributionWallet.Contract.ContributionWalletTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ContributionWallet *ContributionWalletCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ContributionWallet.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ContributionWallet *ContributionWalletTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ContributionWallet.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ContributionWallet *ContributionWalletTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ContributionWallet.Contract.contract.Transact(opts, method, params...)
}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_ContributionWallet *ContributionWalletCaller) Contribution(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ContributionWallet.contract.Call(opts, &out, "contribution")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_ContributionWallet *ContributionWalletSession) Contribution() (common.Address, error) {
	return _ContributionWallet.Contract.Contribution(&_ContributionWallet.CallOpts)
}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_ContributionWallet *ContributionWalletCallerSession) Contribution() (common.Address, error) {
	return _ContributionWallet.Contract.Contribution(&_ContributionWallet.CallOpts)
}

// EndBlock is a free data retrieval call binding the contract method 0x083c6323.
//
// Solidity: function endBlock() view returns(uint256)
func (_ContributionWallet *ContributionWalletCaller) EndBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ContributionWallet.contract.Call(opts, &out, "endBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EndBlock is a free data retrieval call binding the contract method 0x083c6323.
//
// Solidity: function endBlock() view returns(uint256)
func (_ContributionWallet *ContributionWalletSession) EndBlock() (*big.Int, error) {
	return _ContributionWallet.Contract.EndBlock(&_ContributionWallet.CallOpts)
}

// EndBlock is a free data retrieval call binding the contract method 0x083c6323.
//
// Solidity: function endBlock() view returns(uint256)
func (_ContributionWallet *ContributionWalletCallerSession) EndBlock() (*big.Int, error) {
	return _ContributionWallet.Contract.EndBlock(&_ContributionWallet.CallOpts)
}

// Multisig is a free data retrieval call binding the contract method 0x4783c35b.
//
// Solidity: function multisig() view returns(address)
func (_ContributionWallet *ContributionWalletCaller) Multisig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ContributionWallet.contract.Call(opts, &out, "multisig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Multisig is a free data retrieval call binding the contract method 0x4783c35b.
//
// Solidity: function multisig() view returns(address)
func (_ContributionWallet *ContributionWalletSession) Multisig() (common.Address, error) {
	return _ContributionWallet.Contract.Multisig(&_ContributionWallet.CallOpts)
}

// Multisig is a free data retrieval call binding the contract method 0x4783c35b.
//
// Solidity: function multisig() view returns(address)
func (_ContributionWallet *ContributionWalletCallerSession) Multisig() (common.Address, error) {
	return _ContributionWallet.Contract.Multisig(&_ContributionWallet.CallOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_ContributionWallet *ContributionWalletTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ContributionWallet.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_ContributionWallet *ContributionWalletSession) Withdraw() (*types.Transaction, error) {
	return _ContributionWallet.Contract.Withdraw(&_ContributionWallet.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_ContributionWallet *ContributionWalletTransactorSession) Withdraw() (*types.Transaction, error) {
	return _ContributionWallet.Contract.Withdraw(&_ContributionWallet.TransactOpts)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_ContributionWallet *ContributionWalletTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _ContributionWallet.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_ContributionWallet *ContributionWalletSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _ContributionWallet.Contract.Fallback(&_ContributionWallet.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_ContributionWallet *ContributionWalletTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _ContributionWallet.Contract.Fallback(&_ContributionWallet.TransactOpts, calldata)
}

// ControlledABI is the input ABI used to generate the binding from.
const ControlledABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_newController\",\"type\":\"address\"}],\"name\":\"changeController\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"controller\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// ControlledFuncSigs maps the 4-byte function signature to its string representation.
var ControlledFuncSigs = map[string]string{
	"3cebb823": "changeController(address)",
	"f77c4791": "controller()",
}

// ControlledBin is the compiled bytecode used for deploying new contracts.
var ControlledBin = "0x608060405234801561001057600080fd5b5060008054600160a060020a03191633179055610166806100326000396000f30060806040526004361061004b5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416633cebb8238114610050578063f77c479114610080575b600080fd5b34801561005c57600080fd5b5061007e73ffffffffffffffffffffffffffffffffffffffff600435166100be565b005b34801561008c57600080fd5b5061009561011e565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b60005473ffffffffffffffffffffffffffffffffffffffff1633146100e257600080fd5b6000805473ffffffffffffffffffffffffffffffffffffffff191673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b60005473ffffffffffffffffffffffffffffffffffffffff16815600a165627a7a72305820773b4baa99eb83da8b0f1c3125c1559fbc27d756dbd58f2399064a6ac0fa619b0029"

// DeployControlled deploys a new Ethereum contract, binding an instance of Controlled to it.
func DeployControlled(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Controlled, error) {
	parsed, err := abi.JSON(strings.NewReader(ControlledABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ControlledBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Controlled{ControlledCaller: ControlledCaller{contract: contract}, ControlledTransactor: ControlledTransactor{contract: contract}, ControlledFilterer: ControlledFilterer{contract: contract}}, nil
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

// DevTokensHolderABI is the input ABI used to generate the binding from.
const DevTokensHolderABI = "[{\"constant\":false,\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"collectTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"changeOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"newOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_contribution\",\"type\":\"address\"},{\"name\":\"_snt\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_token\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_controller\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"ClaimedTokens\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_holder\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"TokensWithdrawn\",\"type\":\"event\"}]"

// DevTokensHolderFuncSigs maps the 4-byte function signature to its string representation.
var DevTokensHolderFuncSigs = map[string]string{
	"79ba5097": "acceptOwnership()",
	"a6f9dae1": "changeOwner(address)",
	"df8de3e7": "claimTokens(address)",
	"8433acd1": "collectTokens()",
	"d4ee1d90": "newOwner()",
	"8da5cb5b": "owner()",
}

// DevTokensHolderBin is the compiled bytecode used for deploying new contracts.
var DevTokensHolderBin = "0x608060405234801561001057600080fd5b506040516060806108128339810160409081528151602083015191909201516000805433600160a060020a0319918216178116600160a060020a0395861617825560038054821694861694909417909355600480549093169390911692909217905561079090819061008290396000f3006080604052600436106100775763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166379ba5097811461007c5780638433acd1146100935780638da5cb5b146100a8578063a6f9dae1146100d9578063d4ee1d90146100fa578063df8de3e71461010f575b600080fd5b34801561008857600080fd5b50610091610130565b005b34801561009f57600080fd5b50610091610175565b3480156100b457600080fd5b506100bd61047c565b60408051600160a060020a039092168252519081900360200190f35b3480156100e557600080fd5b50610091600160a060020a036004351661048b565b34801561010657600080fd5b506100bd6104d1565b34801561011b57600080fd5b50610091600160a060020a03600435166104e0565b600154600160a060020a0316331415610173576001546000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a039092169190911790555b565b60008054819081908190600160a060020a0316331461019357600080fd5b60048054604080517f70a08231000000000000000000000000000000000000000000000000000000008152309381019390935251600160a060020a03909116916370a082319160248083019260209291908290030181600087803b1580156101fa57600080fd5b505af115801561020e573d6000803e3d6000fd5b505050506040513d602081101561022457600080fd5b505160025490945061023c908563ffffffff6106e216565b9250600360009054906101000a9004600160a060020a0316600160a060020a031663fe67a1896040518163ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401602060405180830381600087803b1580156102aa57600080fd5b505af11580156102be573d6000803e3d6000fd5b505050506040513d60208110156102d457600080fd5b5051915060008211801561030757506102fd6102f060066106f8565b839063ffffffff6106e216565b610305610713565b115b151561031257600080fd5b61035361031f60186106f8565b61034761033a8561032e610713565b9063ffffffff61071716565b869063ffffffff61072916565b9063ffffffff61074d16565b905061036a6002548261071790919063ffffffff16565b9050838111156103775750825b60025461038a908263ffffffff6106e216565b6002556004805460008054604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a0392831695810195909552602485018690525192169263a9059cbb9260448083019360209383900390910190829087803b15801561040057600080fd5b505af1158015610414573d6000803e3d6000fd5b505050506040513d602081101561042a57600080fd5b5051151561043457fe5b600054604080518381529051600160a060020a03909216917f6352c5382c4a4578e712449ca65e83cdb392d045dfcf1cad9615189db2da244b9181900360200190a250505050565b600054600160a060020a031681565b600054600160a060020a031633146104a257600080fd5b6001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600154600160a060020a031681565b600080548190600160a060020a031633146104fa57600080fd5b600454600160a060020a038481169116141561051557600080fd5b600160a060020a03831615156105665760008054604051600160a060020a0390911691303180156108fc02929091818181858888f19350505050158015610560573d6000803e3d6000fd5b506106dd565b604080517f70a082310000000000000000000000000000000000000000000000000000000081523060048201529051849350600160a060020a038416916370a082319160248083019260209291908290030181600087803b1580156105ca57600080fd5b505af11580156105de573d6000803e3d6000fd5b505050506040513d60208110156105f457600080fd5b505160008054604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a0392831660048201526024810185905290519394509085169263a9059cbb92604480840193602093929083900390910190829087803b15801561066a57600080fd5b505af115801561067e573d6000803e3d6000fd5b505050506040513d602081101561069457600080fd5b5050600054604080518381529051600160a060020a03928316928616917ff931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c919081900360200190a35b505050565b6000828201838110156106f157fe5b9392505050565b600061070d8262278d0063ffffffff61072916565b92915050565b4290565b60008282111561072357fe5b50900390565b6000828202831580610745575082848281151561074257fe5b04145b15156106f157fe5b600080828481151561075b57fe5b049493505050505600a165627a7a72305820094ec8d50d00ecb45491ea1f8d2e62e58432ae38006f9e50e910ad8b8a30fd540029"

// DeployDevTokensHolder deploys a new Ethereum contract, binding an instance of DevTokensHolder to it.
func DeployDevTokensHolder(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address, _contribution common.Address, _snt common.Address) (common.Address, *types.Transaction, *DevTokensHolder, error) {
	parsed, err := abi.JSON(strings.NewReader(DevTokensHolderABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(DevTokensHolderBin), backend, _owner, _contribution, _snt)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DevTokensHolder{DevTokensHolderCaller: DevTokensHolderCaller{contract: contract}, DevTokensHolderTransactor: DevTokensHolderTransactor{contract: contract}, DevTokensHolderFilterer: DevTokensHolderFilterer{contract: contract}}, nil
}

// DevTokensHolder is an auto generated Go binding around an Ethereum contract.
type DevTokensHolder struct {
	DevTokensHolderCaller     // Read-only binding to the contract
	DevTokensHolderTransactor // Write-only binding to the contract
	DevTokensHolderFilterer   // Log filterer for contract events
}

// DevTokensHolderCaller is an auto generated read-only Go binding around an Ethereum contract.
type DevTokensHolderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DevTokensHolderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DevTokensHolderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DevTokensHolderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DevTokensHolderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DevTokensHolderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DevTokensHolderSession struct {
	Contract     *DevTokensHolder  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DevTokensHolderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DevTokensHolderCallerSession struct {
	Contract *DevTokensHolderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// DevTokensHolderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DevTokensHolderTransactorSession struct {
	Contract     *DevTokensHolderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// DevTokensHolderRaw is an auto generated low-level Go binding around an Ethereum contract.
type DevTokensHolderRaw struct {
	Contract *DevTokensHolder // Generic contract binding to access the raw methods on
}

// DevTokensHolderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DevTokensHolderCallerRaw struct {
	Contract *DevTokensHolderCaller // Generic read-only contract binding to access the raw methods on
}

// DevTokensHolderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DevTokensHolderTransactorRaw struct {
	Contract *DevTokensHolderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDevTokensHolder creates a new instance of DevTokensHolder, bound to a specific deployed contract.
func NewDevTokensHolder(address common.Address, backend bind.ContractBackend) (*DevTokensHolder, error) {
	contract, err := bindDevTokensHolder(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DevTokensHolder{DevTokensHolderCaller: DevTokensHolderCaller{contract: contract}, DevTokensHolderTransactor: DevTokensHolderTransactor{contract: contract}, DevTokensHolderFilterer: DevTokensHolderFilterer{contract: contract}}, nil
}

// NewDevTokensHolderCaller creates a new read-only instance of DevTokensHolder, bound to a specific deployed contract.
func NewDevTokensHolderCaller(address common.Address, caller bind.ContractCaller) (*DevTokensHolderCaller, error) {
	contract, err := bindDevTokensHolder(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DevTokensHolderCaller{contract: contract}, nil
}

// NewDevTokensHolderTransactor creates a new write-only instance of DevTokensHolder, bound to a specific deployed contract.
func NewDevTokensHolderTransactor(address common.Address, transactor bind.ContractTransactor) (*DevTokensHolderTransactor, error) {
	contract, err := bindDevTokensHolder(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DevTokensHolderTransactor{contract: contract}, nil
}

// NewDevTokensHolderFilterer creates a new log filterer instance of DevTokensHolder, bound to a specific deployed contract.
func NewDevTokensHolderFilterer(address common.Address, filterer bind.ContractFilterer) (*DevTokensHolderFilterer, error) {
	contract, err := bindDevTokensHolder(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DevTokensHolderFilterer{contract: contract}, nil
}

// bindDevTokensHolder binds a generic wrapper to an already deployed contract.
func bindDevTokensHolder(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DevTokensHolderABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DevTokensHolder *DevTokensHolderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DevTokensHolder.Contract.DevTokensHolderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DevTokensHolder *DevTokensHolderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DevTokensHolder.Contract.DevTokensHolderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DevTokensHolder *DevTokensHolderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DevTokensHolder.Contract.DevTokensHolderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DevTokensHolder *DevTokensHolderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DevTokensHolder.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DevTokensHolder *DevTokensHolderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DevTokensHolder.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DevTokensHolder *DevTokensHolderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DevTokensHolder.Contract.contract.Transact(opts, method, params...)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_DevTokensHolder *DevTokensHolderCaller) NewOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DevTokensHolder.contract.Call(opts, &out, "newOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_DevTokensHolder *DevTokensHolderSession) NewOwner() (common.Address, error) {
	return _DevTokensHolder.Contract.NewOwner(&_DevTokensHolder.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_DevTokensHolder *DevTokensHolderCallerSession) NewOwner() (common.Address, error) {
	return _DevTokensHolder.Contract.NewOwner(&_DevTokensHolder.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DevTokensHolder *DevTokensHolderCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DevTokensHolder.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DevTokensHolder *DevTokensHolderSession) Owner() (common.Address, error) {
	return _DevTokensHolder.Contract.Owner(&_DevTokensHolder.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DevTokensHolder *DevTokensHolderCallerSession) Owner() (common.Address, error) {
	return _DevTokensHolder.Contract.Owner(&_DevTokensHolder.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_DevTokensHolder *DevTokensHolderTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DevTokensHolder.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_DevTokensHolder *DevTokensHolderSession) AcceptOwnership() (*types.Transaction, error) {
	return _DevTokensHolder.Contract.AcceptOwnership(&_DevTokensHolder.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_DevTokensHolder *DevTokensHolderTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _DevTokensHolder.Contract.AcceptOwnership(&_DevTokensHolder.TransactOpts)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_DevTokensHolder *DevTokensHolderTransactor) ChangeOwner(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _DevTokensHolder.contract.Transact(opts, "changeOwner", _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_DevTokensHolder *DevTokensHolderSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _DevTokensHolder.Contract.ChangeOwner(&_DevTokensHolder.TransactOpts, _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_DevTokensHolder *DevTokensHolderTransactorSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _DevTokensHolder.Contract.ChangeOwner(&_DevTokensHolder.TransactOpts, _newOwner)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_DevTokensHolder *DevTokensHolderTransactor) ClaimTokens(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _DevTokensHolder.contract.Transact(opts, "claimTokens", _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_DevTokensHolder *DevTokensHolderSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _DevTokensHolder.Contract.ClaimTokens(&_DevTokensHolder.TransactOpts, _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_DevTokensHolder *DevTokensHolderTransactorSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _DevTokensHolder.Contract.ClaimTokens(&_DevTokensHolder.TransactOpts, _token)
}

// CollectTokens is a paid mutator transaction binding the contract method 0x8433acd1.
//
// Solidity: function collectTokens() returns()
func (_DevTokensHolder *DevTokensHolderTransactor) CollectTokens(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DevTokensHolder.contract.Transact(opts, "collectTokens")
}

// CollectTokens is a paid mutator transaction binding the contract method 0x8433acd1.
//
// Solidity: function collectTokens() returns()
func (_DevTokensHolder *DevTokensHolderSession) CollectTokens() (*types.Transaction, error) {
	return _DevTokensHolder.Contract.CollectTokens(&_DevTokensHolder.TransactOpts)
}

// CollectTokens is a paid mutator transaction binding the contract method 0x8433acd1.
//
// Solidity: function collectTokens() returns()
func (_DevTokensHolder *DevTokensHolderTransactorSession) CollectTokens() (*types.Transaction, error) {
	return _DevTokensHolder.Contract.CollectTokens(&_DevTokensHolder.TransactOpts)
}

// DevTokensHolderClaimedTokensIterator is returned from FilterClaimedTokens and is used to iterate over the raw logs and unpacked data for ClaimedTokens events raised by the DevTokensHolder contract.
type DevTokensHolderClaimedTokensIterator struct {
	Event *DevTokensHolderClaimedTokens // Event containing the contract specifics and raw log

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
func (it *DevTokensHolderClaimedTokensIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DevTokensHolderClaimedTokens)
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
		it.Event = new(DevTokensHolderClaimedTokens)
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
func (it *DevTokensHolderClaimedTokensIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DevTokensHolderClaimedTokensIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DevTokensHolderClaimedTokens represents a ClaimedTokens event raised by the DevTokensHolder contract.
type DevTokensHolderClaimedTokens struct {
	Token      common.Address
	Controller common.Address
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterClaimedTokens is a free log retrieval operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_DevTokensHolder *DevTokensHolderFilterer) FilterClaimedTokens(opts *bind.FilterOpts, _token []common.Address, _controller []common.Address) (*DevTokensHolderClaimedTokensIterator, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _DevTokensHolder.contract.FilterLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return &DevTokensHolderClaimedTokensIterator{contract: _DevTokensHolder.contract, event: "ClaimedTokens", logs: logs, sub: sub}, nil
}

// WatchClaimedTokens is a free log subscription operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_DevTokensHolder *DevTokensHolderFilterer) WatchClaimedTokens(opts *bind.WatchOpts, sink chan<- *DevTokensHolderClaimedTokens, _token []common.Address, _controller []common.Address) (event.Subscription, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _DevTokensHolder.contract.WatchLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DevTokensHolderClaimedTokens)
				if err := _DevTokensHolder.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
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

// ParseClaimedTokens is a log parse operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_DevTokensHolder *DevTokensHolderFilterer) ParseClaimedTokens(log types.Log) (*DevTokensHolderClaimedTokens, error) {
	event := new(DevTokensHolderClaimedTokens)
	if err := _DevTokensHolder.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DevTokensHolderTokensWithdrawnIterator is returned from FilterTokensWithdrawn and is used to iterate over the raw logs and unpacked data for TokensWithdrawn events raised by the DevTokensHolder contract.
type DevTokensHolderTokensWithdrawnIterator struct {
	Event *DevTokensHolderTokensWithdrawn // Event containing the contract specifics and raw log

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
func (it *DevTokensHolderTokensWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DevTokensHolderTokensWithdrawn)
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
		it.Event = new(DevTokensHolderTokensWithdrawn)
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
func (it *DevTokensHolderTokensWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DevTokensHolderTokensWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DevTokensHolderTokensWithdrawn represents a TokensWithdrawn event raised by the DevTokensHolder contract.
type DevTokensHolderTokensWithdrawn struct {
	Holder common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTokensWithdrawn is a free log retrieval operation binding the contract event 0x6352c5382c4a4578e712449ca65e83cdb392d045dfcf1cad9615189db2da244b.
//
// Solidity: event TokensWithdrawn(address indexed _holder, uint256 _amount)
func (_DevTokensHolder *DevTokensHolderFilterer) FilterTokensWithdrawn(opts *bind.FilterOpts, _holder []common.Address) (*DevTokensHolderTokensWithdrawnIterator, error) {

	var _holderRule []interface{}
	for _, _holderItem := range _holder {
		_holderRule = append(_holderRule, _holderItem)
	}

	logs, sub, err := _DevTokensHolder.contract.FilterLogs(opts, "TokensWithdrawn", _holderRule)
	if err != nil {
		return nil, err
	}
	return &DevTokensHolderTokensWithdrawnIterator{contract: _DevTokensHolder.contract, event: "TokensWithdrawn", logs: logs, sub: sub}, nil
}

// WatchTokensWithdrawn is a free log subscription operation binding the contract event 0x6352c5382c4a4578e712449ca65e83cdb392d045dfcf1cad9615189db2da244b.
//
// Solidity: event TokensWithdrawn(address indexed _holder, uint256 _amount)
func (_DevTokensHolder *DevTokensHolderFilterer) WatchTokensWithdrawn(opts *bind.WatchOpts, sink chan<- *DevTokensHolderTokensWithdrawn, _holder []common.Address) (event.Subscription, error) {

	var _holderRule []interface{}
	for _, _holderItem := range _holder {
		_holderRule = append(_holderRule, _holderItem)
	}

	logs, sub, err := _DevTokensHolder.contract.WatchLogs(opts, "TokensWithdrawn", _holderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DevTokensHolderTokensWithdrawn)
				if err := _DevTokensHolder.contract.UnpackLog(event, "TokensWithdrawn", log); err != nil {
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

// ParseTokensWithdrawn is a log parse operation binding the contract event 0x6352c5382c4a4578e712449ca65e83cdb392d045dfcf1cad9615189db2da244b.
//
// Solidity: event TokensWithdrawn(address indexed _holder, uint256 _amount)
func (_DevTokensHolder *DevTokensHolderFilterer) ParseTokensWithdrawn(log types.Log) (*DevTokensHolderTokensWithdrawn, error) {
	event := new(DevTokensHolderTokensWithdrawn)
	if err := _DevTokensHolder.contract.UnpackLog(event, "TokensWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DynamicCeilingABI is the input ABI used to generate the binding from.
const DynamicCeilingABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"curves\",\"outputs\":[{\"name\":\"hash\",\"type\":\"bytes32\"},{\"name\":\"limit\",\"type\":\"uint256\"},{\"name\":\"slopeFactor\",\"type\":\"uint256\"},{\"name\":\"collectMinimum\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"currentIndex\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"nCurves\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"allRevealed\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"contribution\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_curveHashes\",\"type\":\"bytes32[]\"}],\"name\":\"setHiddenCurves\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_limits\",\"type\":\"uint256[]\"},{\"name\":\"_slopeFactors\",\"type\":\"uint256[]\"},{\"name\":\"_collectMinimums\",\"type\":\"uint256[]\"},{\"name\":\"_lasts\",\"type\":\"bool[]\"},{\"name\":\"_salts\",\"type\":\"bytes32[]\"}],\"name\":\"revealMulti\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_limit\",\"type\":\"uint256\"},{\"name\":\"_slopeFactor\",\"type\":\"uint256\"},{\"name\":\"_collectMinimum\",\"type\":\"uint256\"},{\"name\":\"_last\",\"type\":\"bool\"},{\"name\":\"_salt\",\"type\":\"bytes32\"}],\"name\":\"revealCurve\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"revealedCurves\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_limit\",\"type\":\"uint256\"},{\"name\":\"_slopeFactor\",\"type\":\"uint256\"},{\"name\":\"_collectMinimum\",\"type\":\"uint256\"},{\"name\":\"_last\",\"type\":\"bool\"},{\"name\":\"_salt\",\"type\":\"bytes32\"}],\"name\":\"calculateHash\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"collected\",\"type\":\"uint256\"}],\"name\":\"toCollect\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"changeOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"moveTo\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"newOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_contribution\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// DynamicCeilingFuncSigs maps the 4-byte function signature to its string representation.
var DynamicCeilingFuncSigs = map[string]string{
	"79ba5097": "acceptOwnership()",
	"4b28bdc2": "allRevealed()",
	"7ab7d55b": "calculateHash(uint256,uint256,uint256,bool,bytes32)",
	"a6f9dae1": "changeOwner(address)",
	"50520b1f": "contribution()",
	"26987b60": "currentIndex()",
	"1bf7d749": "curves(uint256)",
	"cdd63344": "moveTo(uint256)",
	"3a47e629": "nCurves()",
	"d4ee1d90": "newOwner()",
	"8da5cb5b": "owner()",
	"65f594a7": "revealCurve(uint256,uint256,uint256,bool,bytes32)",
	"627adaa6": "revealMulti(uint256[],uint256[],uint256[],bool[],bytes32[])",
	"6e4e5c1d": "revealedCurves()",
	"54657f0a": "setHiddenCurves(bytes32[])",
	"86bb1e03": "toCollect(uint256)",
}

// DynamicCeilingBin is the compiled bytecode used for deploying new contracts.
var DynamicCeilingBin = "0x608060405234801561001057600080fd5b50604051604080610bc683398101604052805160209091015160008054600160a060020a03938416600160a060020a0319918216331782161790915560028054939092169216919091179055610b5b8061006b6000396000f3006080604052600436106100e55763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416631bf7d74981146100ea57806326987b60146101285780633a47e6291461014f5780634b28bdc21461016457806350520b1f1461018d57806354657f0a146101be578063627adaa61461021557806365f594a71461034e5780636e4e5c1d1461037457806379ba5097146103895780637ab7d55b1461039e57806386bb1e03146103c45780638da5cb5b146103dc578063a6f9dae1146103f1578063cdd6334414610412578063d4ee1d901461042a575b600080fd5b3480156100f657600080fd5b5061010260043561043f565b604080519485526020850193909352838301919091526060830152519081900360800190f35b34801561013457600080fd5b5061013d610477565b60408051918252519081900360200190f35b34801561015b57600080fd5b5061013d61047d565b34801561017057600080fd5b50610179610484565b604080519115158252519081900360200190f35b34801561019957600080fd5b506101a261048d565b60408051600160a060020a039092168252519081900360200190f35b3480156101ca57600080fd5b50604080516020600480358082013583810280860185019096528085526102139536959394602494938501929182918501908490808284375094975061049c9650505050505050565b005b34801561022157600080fd5b506040805160206004803580820135838102808601850190965280855261021395369593946024949385019291829185019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750506040805187358901803560208181028481018201909552818452989b9a9989019892975090820195509350839250850190849080828437509497506105319650505050505050565b34801561035a57600080fd5b506102136004356024356044356064351515608435610625565b34801561038057600080fd5b5061013d610785565b34801561039557600080fd5b5061021361078b565b3480156103aa57600080fd5b5061013d60043560243560443560643515156084356107d0565b3480156103d057600080fd5b5061013d600435610822565b3480156103e857600080fd5b506101a26109df565b3480156103fd57600080fd5b50610213600160a060020a03600435166109ee565b34801561041e57600080fd5b50610213600435610a34565b34801561043657600080fd5b506101a2610a7e565b600380548290811061044d57fe5b60009182526020909120600490910201805460018201546002830154600390930154919350919084565b60045481565b6003545b90565b60065460ff1681565b600254600160a060020a031681565b60008054600160a060020a031633146104b457600080fd5b600354156104c157600080fd5b81516104ce600382610acc565b50600090505b815181101561052d5781818151811015156104eb57fe5b9060200190602002015160038281548110151561050457fe5b600091825260209091206004909102015561052681600163ffffffff610a8d16565b90506104d4565b5050565b60008551600014158015610546575084518651145b8015610553575083518651145b8015610560575082518651145b801561056d575081518651145b151561057857600080fd5b5060005b855181101561061d57610605868281518110151561059657fe5b9060200190602002015186838151811015156105ae57fe5b9060200190602002015186848151811015156105c657fe5b9060200190602002015186858151811015156105de57fe5b9060200190602002015186868151811015156105f657fe5b90602001906020020151610625565b61061681600163ffffffff610a8d16565b905061057c565b505050505050565b60065460ff161561063557600080fd5b61064285858585856107d0565b60055460038054909190811061065457fe5b60009182526020909120600490910201541461066f57600080fd5b841580159061067d57508315155b801561068857508215155b151561069357600080fd5b600060055411156106df576005546003906106b590600163ffffffff610aa316565b815481106106bf57fe5b90600052602060002090600402016001015485101515156106df57600080fd5b8460036005548154811015156106f157fe5b90600052602060002090600402016001018190555083600360055481548110151561071857fe5b90600052602060002090600402016002018190555082600360055481548110151561073f57fe5b600091825260209091206003600490920201015560055461076790600163ffffffff610a8d16565b600555811561077e576006805460ff191660011790555b5050505050565b60055481565b600154600160a060020a03163314156107ce576001546000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a039092169190911790555b565b604080519586526020860194909452848401929092527f010000000000000000000000000000000000000000000000000000000000000090151502606084015260618301525190819003608101902090565b600254600090819081908190600160a060020a0316331461084257600080fd5b600554151561085457600093506109d7565b600360045481548110151561086557fe5b906000526020600020906004020160010154851015156108de5760045461089390600163ffffffff610a8d16565b60055490935083106108a857600093506109d7565b600483905560038054849081106108bb57fe5b906000526020600020906004020160010154851015156108de57600093506109d7565b6109138560036004548154811015156108f357fe5b906000526020600020906004020160010154610aa390919063ffffffff16565b915061094a600360045481548110151561092957fe5b90600052602060002090600402016002015483610ab590919063ffffffff16565b9050600360045481548110151561095d57fe5b906000526020600020906004020160030154811115156109d357600360045481548110151561098857fe5b9060005260206000209060040201600301548211156109cb5760036004548154811015156109b257fe5b90600052602060002090600402016003015493506109d7565b8193506109d7565b8093505b505050919050565b600054600160a060020a031681565b600054600160a060020a03163314610a0557600080fd5b6001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600054600160a060020a03163314610a4b57600080fd5b60055481108015610a6e5750600454610a6b90600163ffffffff610a8d16565b81145b1515610a7957600080fd5b600455565b600154600160a060020a031681565b600082820183811015610a9c57fe5b9392505050565b600082821115610aaf57fe5b50900390565b6000808284811515610ac357fe5b04949350505050565b815481835581811115610af857600402816004028360005260206000209182019101610af89190610afd565b505050565b61048191905b80821115610b2b57600080825560018201819055600282018190556003820155600401610b03565b50905600a165627a7a72305820054b8b9b15a16ab0a22bfd0ada809f1a16bd1cc9b8db03c628bff263a84248b30029"

// DeployDynamicCeiling deploys a new Ethereum contract, binding an instance of DynamicCeiling to it.
func DeployDynamicCeiling(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address, _contribution common.Address) (common.Address, *types.Transaction, *DynamicCeiling, error) {
	parsed, err := abi.JSON(strings.NewReader(DynamicCeilingABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(DynamicCeilingBin), backend, _owner, _contribution)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DynamicCeiling{DynamicCeilingCaller: DynamicCeilingCaller{contract: contract}, DynamicCeilingTransactor: DynamicCeilingTransactor{contract: contract}, DynamicCeilingFilterer: DynamicCeilingFilterer{contract: contract}}, nil
}

// DynamicCeiling is an auto generated Go binding around an Ethereum contract.
type DynamicCeiling struct {
	DynamicCeilingCaller     // Read-only binding to the contract
	DynamicCeilingTransactor // Write-only binding to the contract
	DynamicCeilingFilterer   // Log filterer for contract events
}

// DynamicCeilingCaller is an auto generated read-only Go binding around an Ethereum contract.
type DynamicCeilingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DynamicCeilingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DynamicCeilingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DynamicCeilingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DynamicCeilingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DynamicCeilingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DynamicCeilingSession struct {
	Contract     *DynamicCeiling   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DynamicCeilingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DynamicCeilingCallerSession struct {
	Contract *DynamicCeilingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// DynamicCeilingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DynamicCeilingTransactorSession struct {
	Contract     *DynamicCeilingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// DynamicCeilingRaw is an auto generated low-level Go binding around an Ethereum contract.
type DynamicCeilingRaw struct {
	Contract *DynamicCeiling // Generic contract binding to access the raw methods on
}

// DynamicCeilingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DynamicCeilingCallerRaw struct {
	Contract *DynamicCeilingCaller // Generic read-only contract binding to access the raw methods on
}

// DynamicCeilingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DynamicCeilingTransactorRaw struct {
	Contract *DynamicCeilingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDynamicCeiling creates a new instance of DynamicCeiling, bound to a specific deployed contract.
func NewDynamicCeiling(address common.Address, backend bind.ContractBackend) (*DynamicCeiling, error) {
	contract, err := bindDynamicCeiling(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DynamicCeiling{DynamicCeilingCaller: DynamicCeilingCaller{contract: contract}, DynamicCeilingTransactor: DynamicCeilingTransactor{contract: contract}, DynamicCeilingFilterer: DynamicCeilingFilterer{contract: contract}}, nil
}

// NewDynamicCeilingCaller creates a new read-only instance of DynamicCeiling, bound to a specific deployed contract.
func NewDynamicCeilingCaller(address common.Address, caller bind.ContractCaller) (*DynamicCeilingCaller, error) {
	contract, err := bindDynamicCeiling(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DynamicCeilingCaller{contract: contract}, nil
}

// NewDynamicCeilingTransactor creates a new write-only instance of DynamicCeiling, bound to a specific deployed contract.
func NewDynamicCeilingTransactor(address common.Address, transactor bind.ContractTransactor) (*DynamicCeilingTransactor, error) {
	contract, err := bindDynamicCeiling(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DynamicCeilingTransactor{contract: contract}, nil
}

// NewDynamicCeilingFilterer creates a new log filterer instance of DynamicCeiling, bound to a specific deployed contract.
func NewDynamicCeilingFilterer(address common.Address, filterer bind.ContractFilterer) (*DynamicCeilingFilterer, error) {
	contract, err := bindDynamicCeiling(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DynamicCeilingFilterer{contract: contract}, nil
}

// bindDynamicCeiling binds a generic wrapper to an already deployed contract.
func bindDynamicCeiling(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DynamicCeilingABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DynamicCeiling *DynamicCeilingRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DynamicCeiling.Contract.DynamicCeilingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DynamicCeiling *DynamicCeilingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.DynamicCeilingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DynamicCeiling *DynamicCeilingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.DynamicCeilingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DynamicCeiling *DynamicCeilingCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DynamicCeiling.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DynamicCeiling *DynamicCeilingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DynamicCeiling *DynamicCeilingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.contract.Transact(opts, method, params...)
}

// AllRevealed is a free data retrieval call binding the contract method 0x4b28bdc2.
//
// Solidity: function allRevealed() view returns(bool)
func (_DynamicCeiling *DynamicCeilingCaller) AllRevealed(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "allRevealed")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllRevealed is a free data retrieval call binding the contract method 0x4b28bdc2.
//
// Solidity: function allRevealed() view returns(bool)
func (_DynamicCeiling *DynamicCeilingSession) AllRevealed() (bool, error) {
	return _DynamicCeiling.Contract.AllRevealed(&_DynamicCeiling.CallOpts)
}

// AllRevealed is a free data retrieval call binding the contract method 0x4b28bdc2.
//
// Solidity: function allRevealed() view returns(bool)
func (_DynamicCeiling *DynamicCeilingCallerSession) AllRevealed() (bool, error) {
	return _DynamicCeiling.Contract.AllRevealed(&_DynamicCeiling.CallOpts)
}

// CalculateHash is a free data retrieval call binding the contract method 0x7ab7d55b.
//
// Solidity: function calculateHash(uint256 _limit, uint256 _slopeFactor, uint256 _collectMinimum, bool _last, bytes32 _salt) view returns(bytes32)
func (_DynamicCeiling *DynamicCeilingCaller) CalculateHash(opts *bind.CallOpts, _limit *big.Int, _slopeFactor *big.Int, _collectMinimum *big.Int, _last bool, _salt [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "calculateHash", _limit, _slopeFactor, _collectMinimum, _last, _salt)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateHash is a free data retrieval call binding the contract method 0x7ab7d55b.
//
// Solidity: function calculateHash(uint256 _limit, uint256 _slopeFactor, uint256 _collectMinimum, bool _last, bytes32 _salt) view returns(bytes32)
func (_DynamicCeiling *DynamicCeilingSession) CalculateHash(_limit *big.Int, _slopeFactor *big.Int, _collectMinimum *big.Int, _last bool, _salt [32]byte) ([32]byte, error) {
	return _DynamicCeiling.Contract.CalculateHash(&_DynamicCeiling.CallOpts, _limit, _slopeFactor, _collectMinimum, _last, _salt)
}

// CalculateHash is a free data retrieval call binding the contract method 0x7ab7d55b.
//
// Solidity: function calculateHash(uint256 _limit, uint256 _slopeFactor, uint256 _collectMinimum, bool _last, bytes32 _salt) view returns(bytes32)
func (_DynamicCeiling *DynamicCeilingCallerSession) CalculateHash(_limit *big.Int, _slopeFactor *big.Int, _collectMinimum *big.Int, _last bool, _salt [32]byte) ([32]byte, error) {
	return _DynamicCeiling.Contract.CalculateHash(&_DynamicCeiling.CallOpts, _limit, _slopeFactor, _collectMinimum, _last, _salt)
}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_DynamicCeiling *DynamicCeilingCaller) Contribution(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "contribution")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_DynamicCeiling *DynamicCeilingSession) Contribution() (common.Address, error) {
	return _DynamicCeiling.Contract.Contribution(&_DynamicCeiling.CallOpts)
}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_DynamicCeiling *DynamicCeilingCallerSession) Contribution() (common.Address, error) {
	return _DynamicCeiling.Contract.Contribution(&_DynamicCeiling.CallOpts)
}

// CurrentIndex is a free data retrieval call binding the contract method 0x26987b60.
//
// Solidity: function currentIndex() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingCaller) CurrentIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "currentIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentIndex is a free data retrieval call binding the contract method 0x26987b60.
//
// Solidity: function currentIndex() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingSession) CurrentIndex() (*big.Int, error) {
	return _DynamicCeiling.Contract.CurrentIndex(&_DynamicCeiling.CallOpts)
}

// CurrentIndex is a free data retrieval call binding the contract method 0x26987b60.
//
// Solidity: function currentIndex() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingCallerSession) CurrentIndex() (*big.Int, error) {
	return _DynamicCeiling.Contract.CurrentIndex(&_DynamicCeiling.CallOpts)
}

// Curves is a free data retrieval call binding the contract method 0x1bf7d749.
//
// Solidity: function curves(uint256 ) view returns(bytes32 hash, uint256 limit, uint256 slopeFactor, uint256 collectMinimum)
func (_DynamicCeiling *DynamicCeilingCaller) Curves(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Hash           [32]byte
	Limit          *big.Int
	SlopeFactor    *big.Int
	CollectMinimum *big.Int
}, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "curves", arg0)

	outstruct := new(struct {
		Hash           [32]byte
		Limit          *big.Int
		SlopeFactor    *big.Int
		CollectMinimum *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Hash = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Limit = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.SlopeFactor = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.CollectMinimum = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Curves is a free data retrieval call binding the contract method 0x1bf7d749.
//
// Solidity: function curves(uint256 ) view returns(bytes32 hash, uint256 limit, uint256 slopeFactor, uint256 collectMinimum)
func (_DynamicCeiling *DynamicCeilingSession) Curves(arg0 *big.Int) (struct {
	Hash           [32]byte
	Limit          *big.Int
	SlopeFactor    *big.Int
	CollectMinimum *big.Int
}, error) {
	return _DynamicCeiling.Contract.Curves(&_DynamicCeiling.CallOpts, arg0)
}

// Curves is a free data retrieval call binding the contract method 0x1bf7d749.
//
// Solidity: function curves(uint256 ) view returns(bytes32 hash, uint256 limit, uint256 slopeFactor, uint256 collectMinimum)
func (_DynamicCeiling *DynamicCeilingCallerSession) Curves(arg0 *big.Int) (struct {
	Hash           [32]byte
	Limit          *big.Int
	SlopeFactor    *big.Int
	CollectMinimum *big.Int
}, error) {
	return _DynamicCeiling.Contract.Curves(&_DynamicCeiling.CallOpts, arg0)
}

// NCurves is a free data retrieval call binding the contract method 0x3a47e629.
//
// Solidity: function nCurves() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingCaller) NCurves(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "nCurves")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NCurves is a free data retrieval call binding the contract method 0x3a47e629.
//
// Solidity: function nCurves() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingSession) NCurves() (*big.Int, error) {
	return _DynamicCeiling.Contract.NCurves(&_DynamicCeiling.CallOpts)
}

// NCurves is a free data retrieval call binding the contract method 0x3a47e629.
//
// Solidity: function nCurves() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingCallerSession) NCurves() (*big.Int, error) {
	return _DynamicCeiling.Contract.NCurves(&_DynamicCeiling.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_DynamicCeiling *DynamicCeilingCaller) NewOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "newOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_DynamicCeiling *DynamicCeilingSession) NewOwner() (common.Address, error) {
	return _DynamicCeiling.Contract.NewOwner(&_DynamicCeiling.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_DynamicCeiling *DynamicCeilingCallerSession) NewOwner() (common.Address, error) {
	return _DynamicCeiling.Contract.NewOwner(&_DynamicCeiling.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DynamicCeiling *DynamicCeilingCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DynamicCeiling *DynamicCeilingSession) Owner() (common.Address, error) {
	return _DynamicCeiling.Contract.Owner(&_DynamicCeiling.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DynamicCeiling *DynamicCeilingCallerSession) Owner() (common.Address, error) {
	return _DynamicCeiling.Contract.Owner(&_DynamicCeiling.CallOpts)
}

// RevealedCurves is a free data retrieval call binding the contract method 0x6e4e5c1d.
//
// Solidity: function revealedCurves() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingCaller) RevealedCurves(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DynamicCeiling.contract.Call(opts, &out, "revealedCurves")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RevealedCurves is a free data retrieval call binding the contract method 0x6e4e5c1d.
//
// Solidity: function revealedCurves() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingSession) RevealedCurves() (*big.Int, error) {
	return _DynamicCeiling.Contract.RevealedCurves(&_DynamicCeiling.CallOpts)
}

// RevealedCurves is a free data retrieval call binding the contract method 0x6e4e5c1d.
//
// Solidity: function revealedCurves() view returns(uint256)
func (_DynamicCeiling *DynamicCeilingCallerSession) RevealedCurves() (*big.Int, error) {
	return _DynamicCeiling.Contract.RevealedCurves(&_DynamicCeiling.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_DynamicCeiling *DynamicCeilingTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DynamicCeiling.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_DynamicCeiling *DynamicCeilingSession) AcceptOwnership() (*types.Transaction, error) {
	return _DynamicCeiling.Contract.AcceptOwnership(&_DynamicCeiling.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_DynamicCeiling *DynamicCeilingTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _DynamicCeiling.Contract.AcceptOwnership(&_DynamicCeiling.TransactOpts)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_DynamicCeiling *DynamicCeilingTransactor) ChangeOwner(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _DynamicCeiling.contract.Transact(opts, "changeOwner", _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_DynamicCeiling *DynamicCeilingSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.ChangeOwner(&_DynamicCeiling.TransactOpts, _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_DynamicCeiling *DynamicCeilingTransactorSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.ChangeOwner(&_DynamicCeiling.TransactOpts, _newOwner)
}

// MoveTo is a paid mutator transaction binding the contract method 0xcdd63344.
//
// Solidity: function moveTo(uint256 _index) returns()
func (_DynamicCeiling *DynamicCeilingTransactor) MoveTo(opts *bind.TransactOpts, _index *big.Int) (*types.Transaction, error) {
	return _DynamicCeiling.contract.Transact(opts, "moveTo", _index)
}

// MoveTo is a paid mutator transaction binding the contract method 0xcdd63344.
//
// Solidity: function moveTo(uint256 _index) returns()
func (_DynamicCeiling *DynamicCeilingSession) MoveTo(_index *big.Int) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.MoveTo(&_DynamicCeiling.TransactOpts, _index)
}

// MoveTo is a paid mutator transaction binding the contract method 0xcdd63344.
//
// Solidity: function moveTo(uint256 _index) returns()
func (_DynamicCeiling *DynamicCeilingTransactorSession) MoveTo(_index *big.Int) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.MoveTo(&_DynamicCeiling.TransactOpts, _index)
}

// RevealCurve is a paid mutator transaction binding the contract method 0x65f594a7.
//
// Solidity: function revealCurve(uint256 _limit, uint256 _slopeFactor, uint256 _collectMinimum, bool _last, bytes32 _salt) returns()
func (_DynamicCeiling *DynamicCeilingTransactor) RevealCurve(opts *bind.TransactOpts, _limit *big.Int, _slopeFactor *big.Int, _collectMinimum *big.Int, _last bool, _salt [32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.contract.Transact(opts, "revealCurve", _limit, _slopeFactor, _collectMinimum, _last, _salt)
}

// RevealCurve is a paid mutator transaction binding the contract method 0x65f594a7.
//
// Solidity: function revealCurve(uint256 _limit, uint256 _slopeFactor, uint256 _collectMinimum, bool _last, bytes32 _salt) returns()
func (_DynamicCeiling *DynamicCeilingSession) RevealCurve(_limit *big.Int, _slopeFactor *big.Int, _collectMinimum *big.Int, _last bool, _salt [32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.RevealCurve(&_DynamicCeiling.TransactOpts, _limit, _slopeFactor, _collectMinimum, _last, _salt)
}

// RevealCurve is a paid mutator transaction binding the contract method 0x65f594a7.
//
// Solidity: function revealCurve(uint256 _limit, uint256 _slopeFactor, uint256 _collectMinimum, bool _last, bytes32 _salt) returns()
func (_DynamicCeiling *DynamicCeilingTransactorSession) RevealCurve(_limit *big.Int, _slopeFactor *big.Int, _collectMinimum *big.Int, _last bool, _salt [32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.RevealCurve(&_DynamicCeiling.TransactOpts, _limit, _slopeFactor, _collectMinimum, _last, _salt)
}

// RevealMulti is a paid mutator transaction binding the contract method 0x627adaa6.
//
// Solidity: function revealMulti(uint256[] _limits, uint256[] _slopeFactors, uint256[] _collectMinimums, bool[] _lasts, bytes32[] _salts) returns()
func (_DynamicCeiling *DynamicCeilingTransactor) RevealMulti(opts *bind.TransactOpts, _limits []*big.Int, _slopeFactors []*big.Int, _collectMinimums []*big.Int, _lasts []bool, _salts [][32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.contract.Transact(opts, "revealMulti", _limits, _slopeFactors, _collectMinimums, _lasts, _salts)
}

// RevealMulti is a paid mutator transaction binding the contract method 0x627adaa6.
//
// Solidity: function revealMulti(uint256[] _limits, uint256[] _slopeFactors, uint256[] _collectMinimums, bool[] _lasts, bytes32[] _salts) returns()
func (_DynamicCeiling *DynamicCeilingSession) RevealMulti(_limits []*big.Int, _slopeFactors []*big.Int, _collectMinimums []*big.Int, _lasts []bool, _salts [][32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.RevealMulti(&_DynamicCeiling.TransactOpts, _limits, _slopeFactors, _collectMinimums, _lasts, _salts)
}

// RevealMulti is a paid mutator transaction binding the contract method 0x627adaa6.
//
// Solidity: function revealMulti(uint256[] _limits, uint256[] _slopeFactors, uint256[] _collectMinimums, bool[] _lasts, bytes32[] _salts) returns()
func (_DynamicCeiling *DynamicCeilingTransactorSession) RevealMulti(_limits []*big.Int, _slopeFactors []*big.Int, _collectMinimums []*big.Int, _lasts []bool, _salts [][32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.RevealMulti(&_DynamicCeiling.TransactOpts, _limits, _slopeFactors, _collectMinimums, _lasts, _salts)
}

// SetHiddenCurves is a paid mutator transaction binding the contract method 0x54657f0a.
//
// Solidity: function setHiddenCurves(bytes32[] _curveHashes) returns()
func (_DynamicCeiling *DynamicCeilingTransactor) SetHiddenCurves(opts *bind.TransactOpts, _curveHashes [][32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.contract.Transact(opts, "setHiddenCurves", _curveHashes)
}

// SetHiddenCurves is a paid mutator transaction binding the contract method 0x54657f0a.
//
// Solidity: function setHiddenCurves(bytes32[] _curveHashes) returns()
func (_DynamicCeiling *DynamicCeilingSession) SetHiddenCurves(_curveHashes [][32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.SetHiddenCurves(&_DynamicCeiling.TransactOpts, _curveHashes)
}

// SetHiddenCurves is a paid mutator transaction binding the contract method 0x54657f0a.
//
// Solidity: function setHiddenCurves(bytes32[] _curveHashes) returns()
func (_DynamicCeiling *DynamicCeilingTransactorSession) SetHiddenCurves(_curveHashes [][32]byte) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.SetHiddenCurves(&_DynamicCeiling.TransactOpts, _curveHashes)
}

// ToCollect is a paid mutator transaction binding the contract method 0x86bb1e03.
//
// Solidity: function toCollect(uint256 collected) returns(uint256)
func (_DynamicCeiling *DynamicCeilingTransactor) ToCollect(opts *bind.TransactOpts, collected *big.Int) (*types.Transaction, error) {
	return _DynamicCeiling.contract.Transact(opts, "toCollect", collected)
}

// ToCollect is a paid mutator transaction binding the contract method 0x86bb1e03.
//
// Solidity: function toCollect(uint256 collected) returns(uint256)
func (_DynamicCeiling *DynamicCeilingSession) ToCollect(collected *big.Int) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.ToCollect(&_DynamicCeiling.TransactOpts, collected)
}

// ToCollect is a paid mutator transaction binding the contract method 0x86bb1e03.
//
// Solidity: function toCollect(uint256 collected) returns(uint256)
func (_DynamicCeiling *DynamicCeilingTransactorSession) ToCollect(collected *big.Int) (*types.Transaction, error) {
	return _DynamicCeiling.Contract.ToCollect(&_DynamicCeiling.TransactOpts, collected)
}

// ERC20TokenABI is the input ABI used to generate the binding from.
const ERC20TokenABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"remaining\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"}]"

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
// Solidity: function totalSupply() view returns(uint256)
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
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Token *ERC20TokenSession) TotalSupply() (*big.Int, error) {
	return _ERC20Token.Contract.TotalSupply(&_ERC20Token.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
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

// MiniMeTokenABI is the input ABI used to generate the binding from.
const MiniMeTokenABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"creationBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newController\",\"type\":\"address\"}],\"name\":\"changeController\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"balanceOfAt\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_cloneTokenName\",\"type\":\"string\"},{\"name\":\"_cloneDecimalUnits\",\"type\":\"uint8\"},{\"name\":\"_cloneTokenSymbol\",\"type\":\"string\"},{\"name\":\"_snapshotBlock\",\"type\":\"uint256\"},{\"name\":\"_transfersEnabled\",\"type\":\"bool\"}],\"name\":\"createCloneToken\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"parentToken\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"generateTokens\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"totalSupplyAt\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"transfersEnabled\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"parentSnapShotBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"},{\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"approveAndCall\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"destroyTokens\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"remaining\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"tokenFactory\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_transfersEnabled\",\"type\":\"bool\"}],\"name\":\"enableTransfers\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"controller\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_tokenFactory\",\"type\":\"address\"},{\"name\":\"_parentToken\",\"type\":\"address\"},{\"name\":\"_parentSnapShotBlock\",\"type\":\"uint256\"},{\"name\":\"_tokenName\",\"type\":\"string\"},{\"name\":\"_decimalUnits\",\"type\":\"uint8\"},{\"name\":\"_tokenSymbol\",\"type\":\"string\"},{\"name\":\"_transfersEnabled\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_token\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_controller\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"ClaimedTokens\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_cloneToken\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_snapshotBlock\",\"type\":\"uint256\"}],\"name\":\"NewCloneToken\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"}]"

// MiniMeTokenFuncSigs maps the 4-byte function signature to its string representation.
var MiniMeTokenFuncSigs = map[string]string{
	"dd62ed3e": "allowance(address,address)",
	"095ea7b3": "approve(address,uint256)",
	"cae9ca51": "approveAndCall(address,uint256,bytes)",
	"70a08231": "balanceOf(address)",
	"4ee2cd7e": "balanceOfAt(address,uint256)",
	"3cebb823": "changeController(address)",
	"df8de3e7": "claimTokens(address)",
	"f77c4791": "controller()",
	"6638c087": "createCloneToken(string,uint8,string,uint256,bool)",
	"17634514": "creationBlock()",
	"313ce567": "decimals()",
	"d3ce77fe": "destroyTokens(address,uint256)",
	"f41e60c5": "enableTransfers(bool)",
	"827f32c0": "generateTokens(address,uint256)",
	"06fdde03": "name()",
	"c5bcc4f1": "parentSnapShotBlock()",
	"80a54001": "parentToken()",
	"95d89b41": "symbol()",
	"e77772fe": "tokenFactory()",
	"18160ddd": "totalSupply()",
	"981b24d0": "totalSupplyAt(uint256)",
	"a9059cbb": "transfer(address,uint256)",
	"23b872dd": "transferFrom(address,address,uint256)",
	"bef97c87": "transfersEnabled()",
	"54fd4d50": "version()",
}

// MiniMeTokenBin is the compiled bytecode used for deploying new contracts.
var MiniMeTokenBin = "0x60c0604052600760808190527f4d4d545f302e310000000000000000000000000000000000000000000000000060a09081526200004091600491906200015b565b503480156200004e57600080fd5b5060405162001b4638038062001b468339810160409081528151602080840151928401516060850151608086015160a087015160c088015160008054600160a060020a03191633179055600b8054600160a060020a0389166101000261010060a860020a031990911617905592880180519698949690959294919091019291620000de916001918701906200015b565b506002805460ff191660ff85161790558151620001039060039060208501906200015b565b5060058054600160a060020a031916600160a060020a0388161790556006859055600b805460ff19168215151790556200014564010000000062000156810204565b60075550620001fd95505050505050565b435b90565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106200019e57805160ff1916838001178555620001ce565b82800160010185558215620001ce579182015b82811115620001ce578251825591602001919060010190620001b1565b50620001dc929150620001e0565b5090565b6200015891905b80821115620001dc5760008155600101620001e7565b611939806200020d6000396000f30060806040526004361061012f5763ffffffff60e060020a60003504166306fdde0381146101f3578063095ea7b31461027d57806317634514146102b557806318160ddd146102dc57806323b872dd146102f1578063313ce5671461031b5780633cebb823146103465780634ee2cd7e1461036757806354fd4d501461038b5780636638c087146103a057806370a082311461046357806380a5400114610484578063827f32c01461049957806395d89b41146104bd578063981b24d0146104d2578063a9059cbb146104ea578063bef97c871461050e578063c5bcc4f114610523578063cae9ca5114610538578063d3ce77fe146105a1578063dd62ed3e146105c5578063df8de3e7146105ec578063e77772fe1461060d578063f41e60c514610622578063f77c47911461063c575b60005461014490600160a060020a0316610651565b156101ec57600054604080517ff48c30540000000000000000000000000000000000000000000000000000000081523360048201529051600160a060020a039092169163f48c3054913491602480830192602092919082900301818588803b1580156101af57600080fd5b505af11580156101c3573d6000803e3d6000fd5b50505050506040513d60208110156101da57600080fd5b505115156101e757600080fd5b6101f1565b600080fd5b005b3480156101ff57600080fd5b5061020861067e565b6040805160208082528351818301528351919283929083019185019080838360005b8381101561024257818101518382015260200161022a565b50505050905090810190601f16801561026f5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561028957600080fd5b506102a1600160a060020a036004351660243561070b565b604080519115158252519081900360200190f35b3480156102c157600080fd5b506102ca61088b565b60408051918252519081900360200190f35b3480156102e857600080fd5b506102ca610891565b3480156102fd57600080fd5b506102a1600160a060020a03600435811690602435166044356108a9565b34801561032757600080fd5b50610330610940565b6040805160ff9092168252519081900360200190f35b34801561035257600080fd5b506101f1600160a060020a0360043516610949565b34801561037357600080fd5b506102ca600160a060020a036004351660243561098f565b34801561039757600080fd5b50610208610adc565b3480156103ac57600080fd5b506040805160206004803580820135601f810184900484028501840190955284845261044794369492936024939284019190819084018382808284375050604080516020601f818a01358b0180359182018390048302840183018552818452989b60ff8b35169b909a909994019750919550918201935091508190840183828082843750949750508435955050505050602001351515610b37565b60408051600160a060020a039092168252519081900360200190f35b34801561046f57600080fd5b506102ca600160a060020a0360043516610d98565b34801561049057600080fd5b50610447610db3565b3480156104a557600080fd5b506102a1600160a060020a0360043516602435610dc2565b3480156104c957600080fd5b50610208610e98565b3480156104de57600080fd5b506102ca600435610ef3565b3480156104f657600080fd5b506102a1600160a060020a0360043516602435610fe7565b34801561051a57600080fd5b506102a1611006565b34801561052f57600080fd5b506102ca61100f565b34801561054457600080fd5b50604080516020600460443581810135601f81018490048402850184019095528484526102a1948235600160a060020a03169460248035953695946064949201919081908401838280828437509497506110159650505050505050565b3480156105ad57600080fd5b506102a1600160a060020a0360043516602435611130565b3480156105d157600080fd5b506102ca600160a060020a03600435811690602435166111fd565b3480156105f857600080fd5b506101f1600160a060020a0360043516611228565b34801561061957600080fd5b5061044761140f565b34801561062e57600080fd5b506101f16004351515611423565b34801561064857600080fd5b5061044761144d565b600080600160a060020a038316151561066d5760009150610678565b823b90506000811191505b50919050565b60018054604080516020600284861615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b820191906000526020600020905b8154815290600101906020018083116106e657829003601f168201915b505050505081565b600b5460009060ff16151561071f57600080fd5b81158015906107505750336000908152600960209081526040808320600160a060020a038716845290915290205415155b1561075a57600080fd5b60005461076f90600160a060020a0316610651565b156108235760008054604080517fda682aeb000000000000000000000000000000000000000000000000000000008152336004820152600160a060020a038781166024830152604482018790529151919092169263da682aeb92606480820193602093909283900390910190829087803b1580156107ec57600080fd5b505af1158015610800573d6000803e3d6000fd5b505050506040513d602081101561081657600080fd5b5051151561082357600080fd5b336000818152600960209081526040808320600160a060020a03881680855290835292819020869055805186815290519293927f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925929181900390910190a35060015b92915050565b60075481565b60006108a361089e61145c565b610ef3565b90505b90565b60008054600160a060020a0316331461092b57600b5460ff1615156108cd57600080fd5b600160a060020a038416600090815260096020908152604080832033845290915290205482111561090057506000610939565b600160a060020a03841660009081526009602090815260408083203384529091529020805483900390555b610936848484611460565b90505b9392505050565b60025460ff1681565b600054600160a060020a0316331461096057600080fd5b6000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600160a060020a03821660009081526008602052604081205415806109eb5750600160a060020a0383166000908152600860205260408120805484929081106109d457fe5b6000918252602090912001546001608060020a0316115b15610ab357600554600160a060020a031615610aab57600554600654600160a060020a0390911690634ee2cd7e908590610a26908690611659565b6040518363ffffffff1660e060020a0281526004018083600160a060020a0316600160a060020a0316815260200182815260200192505050602060405180830381600087803b158015610a7857600080fd5b505af1158015610a8c573d6000803e3d6000fd5b505050506040513d6020811015610aa257600080fd5b50519050610885565b506000610885565b600160a060020a0383166000908152600860205260409020610ad5908361166f565b9050610885565b6004805460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b600080831515610b4c57610b4961145c565b93505b600b546040517f5b7b72c100000000000000000000000000000000000000000000000000000000815230600482018181526024830188905260ff8a16606484015286151560a484015260c0604484019081528b5160c48501528b51610100909504600160a060020a031694635b7b72c1948a938e938e938e938d939291608482019160e40190602089019080838360005b83811015610bf5578181015183820152602001610bdd565b50505050905090810190601f168015610c225780820380516001836020036101000a031916815260200191505b50838103825285518152855160209182019187019080838360005b83811015610c55578181015183820152602001610c3d565b50505050905090810190601f168015610c825780820380516001836020036101000a031916815260200191505b5098505050505050505050602060405180830381600087803b158015610ca757600080fd5b505af1158015610cbb573d6000803e3d6000fd5b505050506040513d6020811015610cd157600080fd5b5051604080517f3cebb8230000000000000000000000000000000000000000000000000000000081523360048201529051919250600160a060020a03831691633cebb8239160248082019260009290919082900301818387803b158015610d3757600080fd5b505af1158015610d4b573d6000803e3d6000fd5b5050604080518781529051600160a060020a03851693507f086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade92509081900360200190a29695505050505050565b6000610dab82610da661145c565b61098f565b90505b919050565b600554600160a060020a031681565b6000805481908190600160a060020a03163314610dde57600080fd5b610df0600a610deb61145c565b61166f565b9150818483011015610e0157600080fd5b610e0e600a8584016117ce565b610e1785610d98565b9050808482011015610e2857600080fd5b600160a060020a0385166000908152600860205260409020610e4c908286016117ce565b604080518581529051600160a060020a038716916000917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a3506001949350505050565b6003805460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b600a546000901580610f28575081600a6000815481101515610f1157fe5b6000918252602090912001546001608060020a0316115b15610fd557600554600160a060020a031615610fcd57600554600654600160a060020a039091169063981b24d090610f61908590611659565b6040518263ffffffff1660e060020a02815260040180828152602001915050602060405180830381600087803b158015610f9a57600080fd5b505af1158015610fae573d6000803e3d6000fd5b505050506040513d6020811015610fc457600080fd5b50519050610dae565b506000610dae565b610fe0600a8361166f565b9050610dae565b600b5460009060ff161515610ffb57600080fd5b610939338484611460565b600b5460ff1681565b60065481565b6000611021848461070b565b151561102c57600080fd5b6040517f8f4ffcb10000000000000000000000000000000000000000000000000000000081523360048201818152602483018690523060448401819052608060648501908152865160848601528651600160a060020a038a1695638f4ffcb195948a94938a939192909160a490910190602085019080838360005b838110156110bf5781810151838201526020016110a7565b50505050905090810190601f1680156110ec5780820380516001836020036101000a031916815260200191505b5095505050505050600060405180830381600087803b15801561110e57600080fd5b505af1158015611122573d6000803e3d6000fd5b506001979650505050505050565b6000805481908190600160a060020a0316331461114c57600080fd5b611159600a610deb61145c565b91508382101561116857600080fd5b611175600a8584036117ce565b61117e85610d98565b90508381101561118d57600080fd5b600160a060020a03851660009081526008602052604090206111b1908583036117ce565b604080518581529051600091600160a060020a038816917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a3506001949350505050565b600160a060020a03918216600090815260096020908152604080832093909416825291909152205490565b600080548190600160a060020a0316331461124257600080fd5b600160a060020a03831615156112935760008054604051600160a060020a0390911691303180156108fc02929091818181858888f1935050505015801561128d573d6000803e3d6000fd5b5061140a565b604080517f70a082310000000000000000000000000000000000000000000000000000000081523060048201529051849350600160a060020a038416916370a082319160248083019260209291908290030181600087803b1580156112f757600080fd5b505af115801561130b573d6000803e3d6000fd5b505050506040513d602081101561132157600080fd5b505160008054604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a0392831660048201526024810185905290519394509085169263a9059cbb92604480840193602093929083900390910190829087803b15801561139757600080fd5b505af11580156113ab573d6000803e3d6000fd5b505050506040513d60208110156113c157600080fd5b5050600054604080518381529051600160a060020a03928316928616917ff931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c919081900360200190a35b505050565b600b546101009004600160a060020a031681565b600054600160a060020a0316331461143a57600080fd5b600b805460ff1916911515919091179055565b600054600160a060020a031681565b4390565b600080808315156114745760019250611650565b61147c61145c565b6006541061148957600080fd5b600160a060020a03851615806114a75750600160a060020a03851630145b156114b157600080fd5b6114bd86610da661145c565b9150838210156114d05760009250611650565b6000546114e590600160a060020a0316610651565b1561159b5760008054604080517f4a393149000000000000000000000000000000000000000000000000000000008152600160a060020a038a8116600483015289811660248301526044820189905291519190921692634a39314992606480820193602093909283900390910190829087803b15801561156457600080fd5b505af1158015611578573d6000803e3d6000fd5b505050506040513d602081101561158e57600080fd5b5051151561159b57600080fd5b600160a060020a03861660009081526008602052604090206115bf908584036117ce565b6115cb85610da661145c565b90508084820110156115dc57600080fd5b600160a060020a0385166000908152600860205260409020611600908286016117ce565b84600160a060020a031686600160a060020a03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef866040518082815260200191505060405180910390a3600192505b50509392505050565b60008183106116685781610939565b5090919050565b60008060008085805490506000141561168b57600093506117c5565b85548690600019810190811061169d57fe5b6000918252602090912001546001608060020a031685106116fa578554869060001981019081106116ca57fe5b60009182526020909120015470010000000000000000000000000000000090046001608060020a031693506117c5565b85600081548110151561170957fe5b6000918252602090912001546001608060020a031685101561172e57600093506117c5565b8554600093506000190191505b8282111561178b57600260018385010104905084868281548110151561175d57fe5b6000918252602090912001546001608060020a03161161177f57809250611786565b6001810391505b61173b565b858381548110151561179957fe5b60009182526020909120015470010000000000000000000000000000000090046001608060020a031693505b50505092915050565b81546000908190158061180d57506117e461145c565b8454859060001981019081106117f657fe5b6000918252602090912001546001608060020a0316105b15611885578354849061182382600183016118d0565b8154811061182d57fe5b90600052602060002001915061184161145c565b82546fffffffffffffffffffffffffffffffff19166001608060020a03918216178116700100000000000000000000000000000000918516919091021782556118ca565b83548490600019810190811061189757fe5b600091825260209091200180546001608060020a0380861670010000000000000000000000000000000002911617815590505b50505050565b81548183558181111561140a5760008381526020902061140a9181019083016108a691905b8082111561190957600081556001016118f5565b50905600a165627a7a723058203b4b8b2b078f541b06a353e7d147625bc703b2c60c6f9da487c4ffcaa076fdbe0029"

// DeployMiniMeToken deploys a new Ethereum contract, binding an instance of MiniMeToken to it.
func DeployMiniMeToken(auth *bind.TransactOpts, backend bind.ContractBackend, _tokenFactory common.Address, _parentToken common.Address, _parentSnapShotBlock *big.Int, _tokenName string, _decimalUnits uint8, _tokenSymbol string, _transfersEnabled bool) (common.Address, *types.Transaction, *MiniMeToken, error) {
	parsed, err := abi.JSON(strings.NewReader(MiniMeTokenABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(MiniMeTokenBin), backend, _tokenFactory, _parentToken, _parentSnapShotBlock, _tokenName, _decimalUnits, _tokenSymbol, _transfersEnabled)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MiniMeToken{MiniMeTokenCaller: MiniMeTokenCaller{contract: contract}, MiniMeTokenTransactor: MiniMeTokenTransactor{contract: contract}, MiniMeTokenFilterer: MiniMeTokenFilterer{contract: contract}}, nil
}

// MiniMeToken is an auto generated Go binding around an Ethereum contract.
type MiniMeToken struct {
	MiniMeTokenCaller     // Read-only binding to the contract
	MiniMeTokenTransactor // Write-only binding to the contract
	MiniMeTokenFilterer   // Log filterer for contract events
}

// MiniMeTokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type MiniMeTokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MiniMeTokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MiniMeTokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MiniMeTokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MiniMeTokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MiniMeTokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MiniMeTokenSession struct {
	Contract     *MiniMeToken      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MiniMeTokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MiniMeTokenCallerSession struct {
	Contract *MiniMeTokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// MiniMeTokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MiniMeTokenTransactorSession struct {
	Contract     *MiniMeTokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// MiniMeTokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type MiniMeTokenRaw struct {
	Contract *MiniMeToken // Generic contract binding to access the raw methods on
}

// MiniMeTokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MiniMeTokenCallerRaw struct {
	Contract *MiniMeTokenCaller // Generic read-only contract binding to access the raw methods on
}

// MiniMeTokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MiniMeTokenTransactorRaw struct {
	Contract *MiniMeTokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMiniMeToken creates a new instance of MiniMeToken, bound to a specific deployed contract.
func NewMiniMeToken(address common.Address, backend bind.ContractBackend) (*MiniMeToken, error) {
	contract, err := bindMiniMeToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MiniMeToken{MiniMeTokenCaller: MiniMeTokenCaller{contract: contract}, MiniMeTokenTransactor: MiniMeTokenTransactor{contract: contract}, MiniMeTokenFilterer: MiniMeTokenFilterer{contract: contract}}, nil
}

// NewMiniMeTokenCaller creates a new read-only instance of MiniMeToken, bound to a specific deployed contract.
func NewMiniMeTokenCaller(address common.Address, caller bind.ContractCaller) (*MiniMeTokenCaller, error) {
	contract, err := bindMiniMeToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenCaller{contract: contract}, nil
}

// NewMiniMeTokenTransactor creates a new write-only instance of MiniMeToken, bound to a specific deployed contract.
func NewMiniMeTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*MiniMeTokenTransactor, error) {
	contract, err := bindMiniMeToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenTransactor{contract: contract}, nil
}

// NewMiniMeTokenFilterer creates a new log filterer instance of MiniMeToken, bound to a specific deployed contract.
func NewMiniMeTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*MiniMeTokenFilterer, error) {
	contract, err := bindMiniMeToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenFilterer{contract: contract}, nil
}

// bindMiniMeToken binds a generic wrapper to an already deployed contract.
func bindMiniMeToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MiniMeTokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MiniMeToken *MiniMeTokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MiniMeToken.Contract.MiniMeTokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MiniMeToken *MiniMeTokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MiniMeToken.Contract.MiniMeTokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MiniMeToken *MiniMeTokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MiniMeToken.Contract.MiniMeTokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MiniMeToken *MiniMeTokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MiniMeToken.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MiniMeToken *MiniMeTokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MiniMeToken.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MiniMeToken *MiniMeTokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MiniMeToken.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_MiniMeToken *MiniMeTokenCaller) Allowance(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "allowance", _owner, _spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_MiniMeToken *MiniMeTokenSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _MiniMeToken.Contract.Allowance(&_MiniMeToken.CallOpts, _owner, _spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_MiniMeToken *MiniMeTokenCallerSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _MiniMeToken.Contract.Allowance(&_MiniMeToken.CallOpts, _owner, _spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_MiniMeToken *MiniMeTokenCaller) BalanceOf(opts *bind.CallOpts, _owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "balanceOf", _owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_MiniMeToken *MiniMeTokenSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _MiniMeToken.Contract.BalanceOf(&_MiniMeToken.CallOpts, _owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_MiniMeToken *MiniMeTokenCallerSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _MiniMeToken.Contract.BalanceOf(&_MiniMeToken.CallOpts, _owner)
}

// BalanceOfAt is a free data retrieval call binding the contract method 0x4ee2cd7e.
//
// Solidity: function balanceOfAt(address _owner, uint256 _blockNumber) view returns(uint256)
func (_MiniMeToken *MiniMeTokenCaller) BalanceOfAt(opts *bind.CallOpts, _owner common.Address, _blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "balanceOfAt", _owner, _blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOfAt is a free data retrieval call binding the contract method 0x4ee2cd7e.
//
// Solidity: function balanceOfAt(address _owner, uint256 _blockNumber) view returns(uint256)
func (_MiniMeToken *MiniMeTokenSession) BalanceOfAt(_owner common.Address, _blockNumber *big.Int) (*big.Int, error) {
	return _MiniMeToken.Contract.BalanceOfAt(&_MiniMeToken.CallOpts, _owner, _blockNumber)
}

// BalanceOfAt is a free data retrieval call binding the contract method 0x4ee2cd7e.
//
// Solidity: function balanceOfAt(address _owner, uint256 _blockNumber) view returns(uint256)
func (_MiniMeToken *MiniMeTokenCallerSession) BalanceOfAt(_owner common.Address, _blockNumber *big.Int) (*big.Int, error) {
	return _MiniMeToken.Contract.BalanceOfAt(&_MiniMeToken.CallOpts, _owner, _blockNumber)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_MiniMeToken *MiniMeTokenCaller) Controller(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "controller")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_MiniMeToken *MiniMeTokenSession) Controller() (common.Address, error) {
	return _MiniMeToken.Contract.Controller(&_MiniMeToken.CallOpts)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_MiniMeToken *MiniMeTokenCallerSession) Controller() (common.Address, error) {
	return _MiniMeToken.Contract.Controller(&_MiniMeToken.CallOpts)
}

// CreationBlock is a free data retrieval call binding the contract method 0x17634514.
//
// Solidity: function creationBlock() view returns(uint256)
func (_MiniMeToken *MiniMeTokenCaller) CreationBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "creationBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CreationBlock is a free data retrieval call binding the contract method 0x17634514.
//
// Solidity: function creationBlock() view returns(uint256)
func (_MiniMeToken *MiniMeTokenSession) CreationBlock() (*big.Int, error) {
	return _MiniMeToken.Contract.CreationBlock(&_MiniMeToken.CallOpts)
}

// CreationBlock is a free data retrieval call binding the contract method 0x17634514.
//
// Solidity: function creationBlock() view returns(uint256)
func (_MiniMeToken *MiniMeTokenCallerSession) CreationBlock() (*big.Int, error) {
	return _MiniMeToken.Contract.CreationBlock(&_MiniMeToken.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MiniMeToken *MiniMeTokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MiniMeToken *MiniMeTokenSession) Decimals() (uint8, error) {
	return _MiniMeToken.Contract.Decimals(&_MiniMeToken.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MiniMeToken *MiniMeTokenCallerSession) Decimals() (uint8, error) {
	return _MiniMeToken.Contract.Decimals(&_MiniMeToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MiniMeToken *MiniMeTokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MiniMeToken *MiniMeTokenSession) Name() (string, error) {
	return _MiniMeToken.Contract.Name(&_MiniMeToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MiniMeToken *MiniMeTokenCallerSession) Name() (string, error) {
	return _MiniMeToken.Contract.Name(&_MiniMeToken.CallOpts)
}

// ParentSnapShotBlock is a free data retrieval call binding the contract method 0xc5bcc4f1.
//
// Solidity: function parentSnapShotBlock() view returns(uint256)
func (_MiniMeToken *MiniMeTokenCaller) ParentSnapShotBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "parentSnapShotBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ParentSnapShotBlock is a free data retrieval call binding the contract method 0xc5bcc4f1.
//
// Solidity: function parentSnapShotBlock() view returns(uint256)
func (_MiniMeToken *MiniMeTokenSession) ParentSnapShotBlock() (*big.Int, error) {
	return _MiniMeToken.Contract.ParentSnapShotBlock(&_MiniMeToken.CallOpts)
}

// ParentSnapShotBlock is a free data retrieval call binding the contract method 0xc5bcc4f1.
//
// Solidity: function parentSnapShotBlock() view returns(uint256)
func (_MiniMeToken *MiniMeTokenCallerSession) ParentSnapShotBlock() (*big.Int, error) {
	return _MiniMeToken.Contract.ParentSnapShotBlock(&_MiniMeToken.CallOpts)
}

// ParentToken is a free data retrieval call binding the contract method 0x80a54001.
//
// Solidity: function parentToken() view returns(address)
func (_MiniMeToken *MiniMeTokenCaller) ParentToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "parentToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ParentToken is a free data retrieval call binding the contract method 0x80a54001.
//
// Solidity: function parentToken() view returns(address)
func (_MiniMeToken *MiniMeTokenSession) ParentToken() (common.Address, error) {
	return _MiniMeToken.Contract.ParentToken(&_MiniMeToken.CallOpts)
}

// ParentToken is a free data retrieval call binding the contract method 0x80a54001.
//
// Solidity: function parentToken() view returns(address)
func (_MiniMeToken *MiniMeTokenCallerSession) ParentToken() (common.Address, error) {
	return _MiniMeToken.Contract.ParentToken(&_MiniMeToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MiniMeToken *MiniMeTokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MiniMeToken *MiniMeTokenSession) Symbol() (string, error) {
	return _MiniMeToken.Contract.Symbol(&_MiniMeToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MiniMeToken *MiniMeTokenCallerSession) Symbol() (string, error) {
	return _MiniMeToken.Contract.Symbol(&_MiniMeToken.CallOpts)
}

// TokenFactory is a free data retrieval call binding the contract method 0xe77772fe.
//
// Solidity: function tokenFactory() view returns(address)
func (_MiniMeToken *MiniMeTokenCaller) TokenFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "tokenFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TokenFactory is a free data retrieval call binding the contract method 0xe77772fe.
//
// Solidity: function tokenFactory() view returns(address)
func (_MiniMeToken *MiniMeTokenSession) TokenFactory() (common.Address, error) {
	return _MiniMeToken.Contract.TokenFactory(&_MiniMeToken.CallOpts)
}

// TokenFactory is a free data retrieval call binding the contract method 0xe77772fe.
//
// Solidity: function tokenFactory() view returns(address)
func (_MiniMeToken *MiniMeTokenCallerSession) TokenFactory() (common.Address, error) {
	return _MiniMeToken.Contract.TokenFactory(&_MiniMeToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MiniMeToken *MiniMeTokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MiniMeToken *MiniMeTokenSession) TotalSupply() (*big.Int, error) {
	return _MiniMeToken.Contract.TotalSupply(&_MiniMeToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MiniMeToken *MiniMeTokenCallerSession) TotalSupply() (*big.Int, error) {
	return _MiniMeToken.Contract.TotalSupply(&_MiniMeToken.CallOpts)
}

// TotalSupplyAt is a free data retrieval call binding the contract method 0x981b24d0.
//
// Solidity: function totalSupplyAt(uint256 _blockNumber) view returns(uint256)
func (_MiniMeToken *MiniMeTokenCaller) TotalSupplyAt(opts *bind.CallOpts, _blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "totalSupplyAt", _blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupplyAt is a free data retrieval call binding the contract method 0x981b24d0.
//
// Solidity: function totalSupplyAt(uint256 _blockNumber) view returns(uint256)
func (_MiniMeToken *MiniMeTokenSession) TotalSupplyAt(_blockNumber *big.Int) (*big.Int, error) {
	return _MiniMeToken.Contract.TotalSupplyAt(&_MiniMeToken.CallOpts, _blockNumber)
}

// TotalSupplyAt is a free data retrieval call binding the contract method 0x981b24d0.
//
// Solidity: function totalSupplyAt(uint256 _blockNumber) view returns(uint256)
func (_MiniMeToken *MiniMeTokenCallerSession) TotalSupplyAt(_blockNumber *big.Int) (*big.Int, error) {
	return _MiniMeToken.Contract.TotalSupplyAt(&_MiniMeToken.CallOpts, _blockNumber)
}

// TransfersEnabled is a free data retrieval call binding the contract method 0xbef97c87.
//
// Solidity: function transfersEnabled() view returns(bool)
func (_MiniMeToken *MiniMeTokenCaller) TransfersEnabled(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "transfersEnabled")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// TransfersEnabled is a free data retrieval call binding the contract method 0xbef97c87.
//
// Solidity: function transfersEnabled() view returns(bool)
func (_MiniMeToken *MiniMeTokenSession) TransfersEnabled() (bool, error) {
	return _MiniMeToken.Contract.TransfersEnabled(&_MiniMeToken.CallOpts)
}

// TransfersEnabled is a free data retrieval call binding the contract method 0xbef97c87.
//
// Solidity: function transfersEnabled() view returns(bool)
func (_MiniMeToken *MiniMeTokenCallerSession) TransfersEnabled() (bool, error) {
	return _MiniMeToken.Contract.TransfersEnabled(&_MiniMeToken.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_MiniMeToken *MiniMeTokenCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _MiniMeToken.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_MiniMeToken *MiniMeTokenSession) Version() (string, error) {
	return _MiniMeToken.Contract.Version(&_MiniMeToken.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_MiniMeToken *MiniMeTokenCallerSession) Version() (string, error) {
	return _MiniMeToken.Contract.Version(&_MiniMeToken.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenTransactor) Approve(opts *bind.TransactOpts, _spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "approve", _spender, _amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenSession) Approve(_spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.Approve(&_MiniMeToken.TransactOpts, _spender, _amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenTransactorSession) Approve(_spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.Approve(&_MiniMeToken.TransactOpts, _spender, _amount)
}

// ApproveAndCall is a paid mutator transaction binding the contract method 0xcae9ca51.
//
// Solidity: function approveAndCall(address _spender, uint256 _amount, bytes _extraData) returns(bool success)
func (_MiniMeToken *MiniMeTokenTransactor) ApproveAndCall(opts *bind.TransactOpts, _spender common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "approveAndCall", _spender, _amount, _extraData)
}

// ApproveAndCall is a paid mutator transaction binding the contract method 0xcae9ca51.
//
// Solidity: function approveAndCall(address _spender, uint256 _amount, bytes _extraData) returns(bool success)
func (_MiniMeToken *MiniMeTokenSession) ApproveAndCall(_spender common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _MiniMeToken.Contract.ApproveAndCall(&_MiniMeToken.TransactOpts, _spender, _amount, _extraData)
}

// ApproveAndCall is a paid mutator transaction binding the contract method 0xcae9ca51.
//
// Solidity: function approveAndCall(address _spender, uint256 _amount, bytes _extraData) returns(bool success)
func (_MiniMeToken *MiniMeTokenTransactorSession) ApproveAndCall(_spender common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _MiniMeToken.Contract.ApproveAndCall(&_MiniMeToken.TransactOpts, _spender, _amount, _extraData)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_MiniMeToken *MiniMeTokenTransactor) ChangeController(opts *bind.TransactOpts, _newController common.Address) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "changeController", _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_MiniMeToken *MiniMeTokenSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _MiniMeToken.Contract.ChangeController(&_MiniMeToken.TransactOpts, _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_MiniMeToken *MiniMeTokenTransactorSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _MiniMeToken.Contract.ChangeController(&_MiniMeToken.TransactOpts, _newController)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_MiniMeToken *MiniMeTokenTransactor) ClaimTokens(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "claimTokens", _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_MiniMeToken *MiniMeTokenSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _MiniMeToken.Contract.ClaimTokens(&_MiniMeToken.TransactOpts, _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_MiniMeToken *MiniMeTokenTransactorSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _MiniMeToken.Contract.ClaimTokens(&_MiniMeToken.TransactOpts, _token)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x6638c087.
//
// Solidity: function createCloneToken(string _cloneTokenName, uint8 _cloneDecimalUnits, string _cloneTokenSymbol, uint256 _snapshotBlock, bool _transfersEnabled) returns(address)
func (_MiniMeToken *MiniMeTokenTransactor) CreateCloneToken(opts *bind.TransactOpts, _cloneTokenName string, _cloneDecimalUnits uint8, _cloneTokenSymbol string, _snapshotBlock *big.Int, _transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "createCloneToken", _cloneTokenName, _cloneDecimalUnits, _cloneTokenSymbol, _snapshotBlock, _transfersEnabled)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x6638c087.
//
// Solidity: function createCloneToken(string _cloneTokenName, uint8 _cloneDecimalUnits, string _cloneTokenSymbol, uint256 _snapshotBlock, bool _transfersEnabled) returns(address)
func (_MiniMeToken *MiniMeTokenSession) CreateCloneToken(_cloneTokenName string, _cloneDecimalUnits uint8, _cloneTokenSymbol string, _snapshotBlock *big.Int, _transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeToken.Contract.CreateCloneToken(&_MiniMeToken.TransactOpts, _cloneTokenName, _cloneDecimalUnits, _cloneTokenSymbol, _snapshotBlock, _transfersEnabled)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x6638c087.
//
// Solidity: function createCloneToken(string _cloneTokenName, uint8 _cloneDecimalUnits, string _cloneTokenSymbol, uint256 _snapshotBlock, bool _transfersEnabled) returns(address)
func (_MiniMeToken *MiniMeTokenTransactorSession) CreateCloneToken(_cloneTokenName string, _cloneDecimalUnits uint8, _cloneTokenSymbol string, _snapshotBlock *big.Int, _transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeToken.Contract.CreateCloneToken(&_MiniMeToken.TransactOpts, _cloneTokenName, _cloneDecimalUnits, _cloneTokenSymbol, _snapshotBlock, _transfersEnabled)
}

// DestroyTokens is a paid mutator transaction binding the contract method 0xd3ce77fe.
//
// Solidity: function destroyTokens(address _owner, uint256 _amount) returns(bool)
func (_MiniMeToken *MiniMeTokenTransactor) DestroyTokens(opts *bind.TransactOpts, _owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "destroyTokens", _owner, _amount)
}

// DestroyTokens is a paid mutator transaction binding the contract method 0xd3ce77fe.
//
// Solidity: function destroyTokens(address _owner, uint256 _amount) returns(bool)
func (_MiniMeToken *MiniMeTokenSession) DestroyTokens(_owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.DestroyTokens(&_MiniMeToken.TransactOpts, _owner, _amount)
}

// DestroyTokens is a paid mutator transaction binding the contract method 0xd3ce77fe.
//
// Solidity: function destroyTokens(address _owner, uint256 _amount) returns(bool)
func (_MiniMeToken *MiniMeTokenTransactorSession) DestroyTokens(_owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.DestroyTokens(&_MiniMeToken.TransactOpts, _owner, _amount)
}

// EnableTransfers is a paid mutator transaction binding the contract method 0xf41e60c5.
//
// Solidity: function enableTransfers(bool _transfersEnabled) returns()
func (_MiniMeToken *MiniMeTokenTransactor) EnableTransfers(opts *bind.TransactOpts, _transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "enableTransfers", _transfersEnabled)
}

// EnableTransfers is a paid mutator transaction binding the contract method 0xf41e60c5.
//
// Solidity: function enableTransfers(bool _transfersEnabled) returns()
func (_MiniMeToken *MiniMeTokenSession) EnableTransfers(_transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeToken.Contract.EnableTransfers(&_MiniMeToken.TransactOpts, _transfersEnabled)
}

// EnableTransfers is a paid mutator transaction binding the contract method 0xf41e60c5.
//
// Solidity: function enableTransfers(bool _transfersEnabled) returns()
func (_MiniMeToken *MiniMeTokenTransactorSession) EnableTransfers(_transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeToken.Contract.EnableTransfers(&_MiniMeToken.TransactOpts, _transfersEnabled)
}

// GenerateTokens is a paid mutator transaction binding the contract method 0x827f32c0.
//
// Solidity: function generateTokens(address _owner, uint256 _amount) returns(bool)
func (_MiniMeToken *MiniMeTokenTransactor) GenerateTokens(opts *bind.TransactOpts, _owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "generateTokens", _owner, _amount)
}

// GenerateTokens is a paid mutator transaction binding the contract method 0x827f32c0.
//
// Solidity: function generateTokens(address _owner, uint256 _amount) returns(bool)
func (_MiniMeToken *MiniMeTokenSession) GenerateTokens(_owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.GenerateTokens(&_MiniMeToken.TransactOpts, _owner, _amount)
}

// GenerateTokens is a paid mutator transaction binding the contract method 0x827f32c0.
//
// Solidity: function generateTokens(address _owner, uint256 _amount) returns(bool)
func (_MiniMeToken *MiniMeTokenTransactorSession) GenerateTokens(_owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.GenerateTokens(&_MiniMeToken.TransactOpts, _owner, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenTransactor) Transfer(opts *bind.TransactOpts, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "transfer", _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenSession) Transfer(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.Transfer(&_MiniMeToken.TransactOpts, _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenTransactorSession) Transfer(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.Transfer(&_MiniMeToken.TransactOpts, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenTransactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.contract.Transact(opts, "transferFrom", _from, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenSession) TransferFrom(_from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.TransferFrom(&_MiniMeToken.TransactOpts, _from, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool success)
func (_MiniMeToken *MiniMeTokenTransactorSession) TransferFrom(_from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _MiniMeToken.Contract.TransferFrom(&_MiniMeToken.TransactOpts, _from, _to, _amount)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_MiniMeToken *MiniMeTokenTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _MiniMeToken.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_MiniMeToken *MiniMeTokenSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _MiniMeToken.Contract.Fallback(&_MiniMeToken.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_MiniMeToken *MiniMeTokenTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _MiniMeToken.Contract.Fallback(&_MiniMeToken.TransactOpts, calldata)
}

// MiniMeTokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the MiniMeToken contract.
type MiniMeTokenApprovalIterator struct {
	Event *MiniMeTokenApproval // Event containing the contract specifics and raw log

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
func (it *MiniMeTokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MiniMeTokenApproval)
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
		it.Event = new(MiniMeTokenApproval)
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
func (it *MiniMeTokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MiniMeTokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MiniMeTokenApproval represents a Approval event raised by the MiniMeToken contract.
type MiniMeTokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) FilterApproval(opts *bind.FilterOpts, _owner []common.Address, _spender []common.Address) (*MiniMeTokenApprovalIterator, error) {

	var _ownerRule []interface{}
	for _, _ownerItem := range _owner {
		_ownerRule = append(_ownerRule, _ownerItem)
	}
	var _spenderRule []interface{}
	for _, _spenderItem := range _spender {
		_spenderRule = append(_spenderRule, _spenderItem)
	}

	logs, sub, err := _MiniMeToken.contract.FilterLogs(opts, "Approval", _ownerRule, _spenderRule)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenApprovalIterator{contract: _MiniMeToken.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *MiniMeTokenApproval, _owner []common.Address, _spender []common.Address) (event.Subscription, error) {

	var _ownerRule []interface{}
	for _, _ownerItem := range _owner {
		_ownerRule = append(_ownerRule, _ownerItem)
	}
	var _spenderRule []interface{}
	for _, _spenderItem := range _spender {
		_spenderRule = append(_spenderRule, _spenderItem)
	}

	logs, sub, err := _MiniMeToken.contract.WatchLogs(opts, "Approval", _ownerRule, _spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MiniMeTokenApproval)
				if err := _MiniMeToken.contract.UnpackLog(event, "Approval", log); err != nil {
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
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) ParseApproval(log types.Log) (*MiniMeTokenApproval, error) {
	event := new(MiniMeTokenApproval)
	if err := _MiniMeToken.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MiniMeTokenClaimedTokensIterator is returned from FilterClaimedTokens and is used to iterate over the raw logs and unpacked data for ClaimedTokens events raised by the MiniMeToken contract.
type MiniMeTokenClaimedTokensIterator struct {
	Event *MiniMeTokenClaimedTokens // Event containing the contract specifics and raw log

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
func (it *MiniMeTokenClaimedTokensIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MiniMeTokenClaimedTokens)
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
		it.Event = new(MiniMeTokenClaimedTokens)
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
func (it *MiniMeTokenClaimedTokensIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MiniMeTokenClaimedTokensIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MiniMeTokenClaimedTokens represents a ClaimedTokens event raised by the MiniMeToken contract.
type MiniMeTokenClaimedTokens struct {
	Token      common.Address
	Controller common.Address
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterClaimedTokens is a free log retrieval operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) FilterClaimedTokens(opts *bind.FilterOpts, _token []common.Address, _controller []common.Address) (*MiniMeTokenClaimedTokensIterator, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _MiniMeToken.contract.FilterLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenClaimedTokensIterator{contract: _MiniMeToken.contract, event: "ClaimedTokens", logs: logs, sub: sub}, nil
}

// WatchClaimedTokens is a free log subscription operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) WatchClaimedTokens(opts *bind.WatchOpts, sink chan<- *MiniMeTokenClaimedTokens, _token []common.Address, _controller []common.Address) (event.Subscription, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _MiniMeToken.contract.WatchLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MiniMeTokenClaimedTokens)
				if err := _MiniMeToken.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
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

// ParseClaimedTokens is a log parse operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) ParseClaimedTokens(log types.Log) (*MiniMeTokenClaimedTokens, error) {
	event := new(MiniMeTokenClaimedTokens)
	if err := _MiniMeToken.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MiniMeTokenNewCloneTokenIterator is returned from FilterNewCloneToken and is used to iterate over the raw logs and unpacked data for NewCloneToken events raised by the MiniMeToken contract.
type MiniMeTokenNewCloneTokenIterator struct {
	Event *MiniMeTokenNewCloneToken // Event containing the contract specifics and raw log

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
func (it *MiniMeTokenNewCloneTokenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MiniMeTokenNewCloneToken)
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
		it.Event = new(MiniMeTokenNewCloneToken)
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
func (it *MiniMeTokenNewCloneTokenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MiniMeTokenNewCloneTokenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MiniMeTokenNewCloneToken represents a NewCloneToken event raised by the MiniMeToken contract.
type MiniMeTokenNewCloneToken struct {
	CloneToken    common.Address
	SnapshotBlock *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterNewCloneToken is a free log retrieval operation binding the contract event 0x086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade.
//
// Solidity: event NewCloneToken(address indexed _cloneToken, uint256 _snapshotBlock)
func (_MiniMeToken *MiniMeTokenFilterer) FilterNewCloneToken(opts *bind.FilterOpts, _cloneToken []common.Address) (*MiniMeTokenNewCloneTokenIterator, error) {

	var _cloneTokenRule []interface{}
	for _, _cloneTokenItem := range _cloneToken {
		_cloneTokenRule = append(_cloneTokenRule, _cloneTokenItem)
	}

	logs, sub, err := _MiniMeToken.contract.FilterLogs(opts, "NewCloneToken", _cloneTokenRule)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenNewCloneTokenIterator{contract: _MiniMeToken.contract, event: "NewCloneToken", logs: logs, sub: sub}, nil
}

// WatchNewCloneToken is a free log subscription operation binding the contract event 0x086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade.
//
// Solidity: event NewCloneToken(address indexed _cloneToken, uint256 _snapshotBlock)
func (_MiniMeToken *MiniMeTokenFilterer) WatchNewCloneToken(opts *bind.WatchOpts, sink chan<- *MiniMeTokenNewCloneToken, _cloneToken []common.Address) (event.Subscription, error) {

	var _cloneTokenRule []interface{}
	for _, _cloneTokenItem := range _cloneToken {
		_cloneTokenRule = append(_cloneTokenRule, _cloneTokenItem)
	}

	logs, sub, err := _MiniMeToken.contract.WatchLogs(opts, "NewCloneToken", _cloneTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MiniMeTokenNewCloneToken)
				if err := _MiniMeToken.contract.UnpackLog(event, "NewCloneToken", log); err != nil {
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

// ParseNewCloneToken is a log parse operation binding the contract event 0x086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade.
//
// Solidity: event NewCloneToken(address indexed _cloneToken, uint256 _snapshotBlock)
func (_MiniMeToken *MiniMeTokenFilterer) ParseNewCloneToken(log types.Log) (*MiniMeTokenNewCloneToken, error) {
	event := new(MiniMeTokenNewCloneToken)
	if err := _MiniMeToken.contract.UnpackLog(event, "NewCloneToken", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MiniMeTokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the MiniMeToken contract.
type MiniMeTokenTransferIterator struct {
	Event *MiniMeTokenTransfer // Event containing the contract specifics and raw log

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
func (it *MiniMeTokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MiniMeTokenTransfer)
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
		it.Event = new(MiniMeTokenTransfer)
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
func (it *MiniMeTokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MiniMeTokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MiniMeTokenTransfer represents a Transfer event raised by the MiniMeToken contract.
type MiniMeTokenTransfer struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) FilterTransfer(opts *bind.FilterOpts, _from []common.Address, _to []common.Address) (*MiniMeTokenTransferIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _MiniMeToken.contract.FilterLogs(opts, "Transfer", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenTransferIterator{contract: _MiniMeToken.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *MiniMeTokenTransfer, _from []common.Address, _to []common.Address) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _MiniMeToken.contract.WatchLogs(opts, "Transfer", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MiniMeTokenTransfer)
				if err := _MiniMeToken.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _amount)
func (_MiniMeToken *MiniMeTokenFilterer) ParseTransfer(log types.Log) (*MiniMeTokenTransfer, error) {
	event := new(MiniMeTokenTransfer)
	if err := _MiniMeToken.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MiniMeTokenFactoryABI is the input ABI used to generate the binding from.
const MiniMeTokenFactoryABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_parentToken\",\"type\":\"address\"},{\"name\":\"_snapshotBlock\",\"type\":\"uint256\"},{\"name\":\"_tokenName\",\"type\":\"string\"},{\"name\":\"_decimalUnits\",\"type\":\"uint8\"},{\"name\":\"_tokenSymbol\",\"type\":\"string\"},{\"name\":\"_transfersEnabled\",\"type\":\"bool\"}],\"name\":\"createCloneToken\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// MiniMeTokenFactoryFuncSigs maps the 4-byte function signature to its string representation.
var MiniMeTokenFactoryFuncSigs = map[string]string{
	"5b7b72c1": "createCloneToken(address,uint256,string,uint8,string,bool)",
}

// MiniMeTokenFactoryBin is the compiled bytecode used for deploying new contracts.
var MiniMeTokenFactoryBin = "0x608060405234801561001057600080fd5b50611e8d806100206000396000f3006080604052600436106100405763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416635b7b72c18114610045575b600080fd5b34801561005157600080fd5b50604080516020600460443581810135601f810184900484028501840190955284845261010694823573ffffffffffffffffffffffffffffffffffffffff1694602480359536959460649492019190819084018382808284375050604080516020601f818a01358b0180359182018390048302840183018552818452989b60ff8b35169b909a90999401975091955091820193509150819084018382808284375094975050505091351515925061012f915050565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b6000803088888888888861014161030b565b73ffffffffffffffffffffffffffffffffffffffff808916825287166020808301919091526040820187905260ff8516608083015282151560c083015260e0606083018181528751918401919091528651909160a084019161010085019189019080838360005b838110156101c05781810151838201526020016101a8565b50505050905090810190601f1680156101ed5780820380516001836020036101000a031916815260200191505b50838103825285518152855160209182019187019080838360005b83811015610220578181015183820152602001610208565b50505050905090810190601f16801561024d5780820380516001836020036101000a031916815260200191505b509950505050505050505050604051809103906000f080158015610275573d6000803e3d6000fd5b50604080517f3cebb823000000000000000000000000000000000000000000000000000000008152336004820152905191925073ffffffffffffffffffffffffffffffffffffffff831691633cebb8239160248082019260009290919082900301818387803b1580156102e757600080fd5b505af11580156102fb573d6000803e3d6000fd5b50929a9950505050505050505050565b604051611b468061031c83390190560060c0604052600760808190527f4d4d545f302e310000000000000000000000000000000000000000000000000060a09081526200004091600491906200015b565b503480156200004e57600080fd5b5060405162001b4638038062001b468339810160409081528151602080840151928401516060850151608086015160a087015160c088015160008054600160a060020a03191633179055600b8054600160a060020a0389166101000261010060a860020a031990911617905592880180519698949690959294919091019291620000de916001918701906200015b565b506002805460ff191660ff85161790558151620001039060039060208501906200015b565b5060058054600160a060020a031916600160a060020a0388161790556006859055600b805460ff19168215151790556200014564010000000062000156810204565b60075550620001fd95505050505050565b435b90565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106200019e57805160ff1916838001178555620001ce565b82800160010185558215620001ce579182015b82811115620001ce578251825591602001919060010190620001b1565b50620001dc929150620001e0565b5090565b6200015891905b80821115620001dc5760008155600101620001e7565b611939806200020d6000396000f30060806040526004361061012f5763ffffffff60e060020a60003504166306fdde0381146101f3578063095ea7b31461027d57806317634514146102b557806318160ddd146102dc57806323b872dd146102f1578063313ce5671461031b5780633cebb823146103465780634ee2cd7e1461036757806354fd4d501461038b5780636638c087146103a057806370a082311461046357806380a5400114610484578063827f32c01461049957806395d89b41146104bd578063981b24d0146104d2578063a9059cbb146104ea578063bef97c871461050e578063c5bcc4f114610523578063cae9ca5114610538578063d3ce77fe146105a1578063dd62ed3e146105c5578063df8de3e7146105ec578063e77772fe1461060d578063f41e60c514610622578063f77c47911461063c575b60005461014490600160a060020a0316610651565b156101ec57600054604080517ff48c30540000000000000000000000000000000000000000000000000000000081523360048201529051600160a060020a039092169163f48c3054913491602480830192602092919082900301818588803b1580156101af57600080fd5b505af11580156101c3573d6000803e3d6000fd5b50505050506040513d60208110156101da57600080fd5b505115156101e757600080fd5b6101f1565b600080fd5b005b3480156101ff57600080fd5b5061020861067e565b6040805160208082528351818301528351919283929083019185019080838360005b8381101561024257818101518382015260200161022a565b50505050905090810190601f16801561026f5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561028957600080fd5b506102a1600160a060020a036004351660243561070b565b604080519115158252519081900360200190f35b3480156102c157600080fd5b506102ca61088b565b60408051918252519081900360200190f35b3480156102e857600080fd5b506102ca610891565b3480156102fd57600080fd5b506102a1600160a060020a03600435811690602435166044356108a9565b34801561032757600080fd5b50610330610940565b6040805160ff9092168252519081900360200190f35b34801561035257600080fd5b506101f1600160a060020a0360043516610949565b34801561037357600080fd5b506102ca600160a060020a036004351660243561098f565b34801561039757600080fd5b50610208610adc565b3480156103ac57600080fd5b506040805160206004803580820135601f810184900484028501840190955284845261044794369492936024939284019190819084018382808284375050604080516020601f818a01358b0180359182018390048302840183018552818452989b60ff8b35169b909a909994019750919550918201935091508190840183828082843750949750508435955050505050602001351515610b37565b60408051600160a060020a039092168252519081900360200190f35b34801561046f57600080fd5b506102ca600160a060020a0360043516610d98565b34801561049057600080fd5b50610447610db3565b3480156104a557600080fd5b506102a1600160a060020a0360043516602435610dc2565b3480156104c957600080fd5b50610208610e98565b3480156104de57600080fd5b506102ca600435610ef3565b3480156104f657600080fd5b506102a1600160a060020a0360043516602435610fe7565b34801561051a57600080fd5b506102a1611006565b34801561052f57600080fd5b506102ca61100f565b34801561054457600080fd5b50604080516020600460443581810135601f81018490048402850184019095528484526102a1948235600160a060020a03169460248035953695946064949201919081908401838280828437509497506110159650505050505050565b3480156105ad57600080fd5b506102a1600160a060020a0360043516602435611130565b3480156105d157600080fd5b506102ca600160a060020a03600435811690602435166111fd565b3480156105f857600080fd5b506101f1600160a060020a0360043516611228565b34801561061957600080fd5b5061044761140f565b34801561062e57600080fd5b506101f16004351515611423565b34801561064857600080fd5b5061044761144d565b600080600160a060020a038316151561066d5760009150610678565b823b90506000811191505b50919050565b60018054604080516020600284861615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b820191906000526020600020905b8154815290600101906020018083116106e657829003601f168201915b505050505081565b600b5460009060ff16151561071f57600080fd5b81158015906107505750336000908152600960209081526040808320600160a060020a038716845290915290205415155b1561075a57600080fd5b60005461076f90600160a060020a0316610651565b156108235760008054604080517fda682aeb000000000000000000000000000000000000000000000000000000008152336004820152600160a060020a038781166024830152604482018790529151919092169263da682aeb92606480820193602093909283900390910190829087803b1580156107ec57600080fd5b505af1158015610800573d6000803e3d6000fd5b505050506040513d602081101561081657600080fd5b5051151561082357600080fd5b336000818152600960209081526040808320600160a060020a03881680855290835292819020869055805186815290519293927f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925929181900390910190a35060015b92915050565b60075481565b60006108a361089e61145c565b610ef3565b90505b90565b60008054600160a060020a0316331461092b57600b5460ff1615156108cd57600080fd5b600160a060020a038416600090815260096020908152604080832033845290915290205482111561090057506000610939565b600160a060020a03841660009081526009602090815260408083203384529091529020805483900390555b610936848484611460565b90505b9392505050565b60025460ff1681565b600054600160a060020a0316331461096057600080fd5b6000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600160a060020a03821660009081526008602052604081205415806109eb5750600160a060020a0383166000908152600860205260408120805484929081106109d457fe5b6000918252602090912001546001608060020a0316115b15610ab357600554600160a060020a031615610aab57600554600654600160a060020a0390911690634ee2cd7e908590610a26908690611659565b6040518363ffffffff1660e060020a0281526004018083600160a060020a0316600160a060020a0316815260200182815260200192505050602060405180830381600087803b158015610a7857600080fd5b505af1158015610a8c573d6000803e3d6000fd5b505050506040513d6020811015610aa257600080fd5b50519050610885565b506000610885565b600160a060020a0383166000908152600860205260409020610ad5908361166f565b9050610885565b6004805460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b600080831515610b4c57610b4961145c565b93505b600b546040517f5b7b72c100000000000000000000000000000000000000000000000000000000815230600482018181526024830188905260ff8a16606484015286151560a484015260c0604484019081528b5160c48501528b51610100909504600160a060020a031694635b7b72c1948a938e938e938e938d939291608482019160e40190602089019080838360005b83811015610bf5578181015183820152602001610bdd565b50505050905090810190601f168015610c225780820380516001836020036101000a031916815260200191505b50838103825285518152855160209182019187019080838360005b83811015610c55578181015183820152602001610c3d565b50505050905090810190601f168015610c825780820380516001836020036101000a031916815260200191505b5098505050505050505050602060405180830381600087803b158015610ca757600080fd5b505af1158015610cbb573d6000803e3d6000fd5b505050506040513d6020811015610cd157600080fd5b5051604080517f3cebb8230000000000000000000000000000000000000000000000000000000081523360048201529051919250600160a060020a03831691633cebb8239160248082019260009290919082900301818387803b158015610d3757600080fd5b505af1158015610d4b573d6000803e3d6000fd5b5050604080518781529051600160a060020a03851693507f086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade92509081900360200190a29695505050505050565b6000610dab82610da661145c565b61098f565b90505b919050565b600554600160a060020a031681565b6000805481908190600160a060020a03163314610dde57600080fd5b610df0600a610deb61145c565b61166f565b9150818483011015610e0157600080fd5b610e0e600a8584016117ce565b610e1785610d98565b9050808482011015610e2857600080fd5b600160a060020a0385166000908152600860205260409020610e4c908286016117ce565b604080518581529051600160a060020a038716916000917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a3506001949350505050565b6003805460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b600a546000901580610f28575081600a6000815481101515610f1157fe5b6000918252602090912001546001608060020a0316115b15610fd557600554600160a060020a031615610fcd57600554600654600160a060020a039091169063981b24d090610f61908590611659565b6040518263ffffffff1660e060020a02815260040180828152602001915050602060405180830381600087803b158015610f9a57600080fd5b505af1158015610fae573d6000803e3d6000fd5b505050506040513d6020811015610fc457600080fd5b50519050610dae565b506000610dae565b610fe0600a8361166f565b9050610dae565b600b5460009060ff161515610ffb57600080fd5b610939338484611460565b600b5460ff1681565b60065481565b6000611021848461070b565b151561102c57600080fd5b6040517f8f4ffcb10000000000000000000000000000000000000000000000000000000081523360048201818152602483018690523060448401819052608060648501908152865160848601528651600160a060020a038a1695638f4ffcb195948a94938a939192909160a490910190602085019080838360005b838110156110bf5781810151838201526020016110a7565b50505050905090810190601f1680156110ec5780820380516001836020036101000a031916815260200191505b5095505050505050600060405180830381600087803b15801561110e57600080fd5b505af1158015611122573d6000803e3d6000fd5b506001979650505050505050565b6000805481908190600160a060020a0316331461114c57600080fd5b611159600a610deb61145c565b91508382101561116857600080fd5b611175600a8584036117ce565b61117e85610d98565b90508381101561118d57600080fd5b600160a060020a03851660009081526008602052604090206111b1908583036117ce565b604080518581529051600091600160a060020a038816917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a3506001949350505050565b600160a060020a03918216600090815260096020908152604080832093909416825291909152205490565b600080548190600160a060020a0316331461124257600080fd5b600160a060020a03831615156112935760008054604051600160a060020a0390911691303180156108fc02929091818181858888f1935050505015801561128d573d6000803e3d6000fd5b5061140a565b604080517f70a082310000000000000000000000000000000000000000000000000000000081523060048201529051849350600160a060020a038416916370a082319160248083019260209291908290030181600087803b1580156112f757600080fd5b505af115801561130b573d6000803e3d6000fd5b505050506040513d602081101561132157600080fd5b505160008054604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a0392831660048201526024810185905290519394509085169263a9059cbb92604480840193602093929083900390910190829087803b15801561139757600080fd5b505af11580156113ab573d6000803e3d6000fd5b505050506040513d60208110156113c157600080fd5b5050600054604080518381529051600160a060020a03928316928616917ff931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c919081900360200190a35b505050565b600b546101009004600160a060020a031681565b600054600160a060020a0316331461143a57600080fd5b600b805460ff1916911515919091179055565b600054600160a060020a031681565b4390565b600080808315156114745760019250611650565b61147c61145c565b6006541061148957600080fd5b600160a060020a03851615806114a75750600160a060020a03851630145b156114b157600080fd5b6114bd86610da661145c565b9150838210156114d05760009250611650565b6000546114e590600160a060020a0316610651565b1561159b5760008054604080517f4a393149000000000000000000000000000000000000000000000000000000008152600160a060020a038a8116600483015289811660248301526044820189905291519190921692634a39314992606480820193602093909283900390910190829087803b15801561156457600080fd5b505af1158015611578573d6000803e3d6000fd5b505050506040513d602081101561158e57600080fd5b5051151561159b57600080fd5b600160a060020a03861660009081526008602052604090206115bf908584036117ce565b6115cb85610da661145c565b90508084820110156115dc57600080fd5b600160a060020a0385166000908152600860205260409020611600908286016117ce565b84600160a060020a031686600160a060020a03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef866040518082815260200191505060405180910390a3600192505b50509392505050565b60008183106116685781610939565b5090919050565b60008060008085805490506000141561168b57600093506117c5565b85548690600019810190811061169d57fe5b6000918252602090912001546001608060020a031685106116fa578554869060001981019081106116ca57fe5b60009182526020909120015470010000000000000000000000000000000090046001608060020a031693506117c5565b85600081548110151561170957fe5b6000918252602090912001546001608060020a031685101561172e57600093506117c5565b8554600093506000190191505b8282111561178b57600260018385010104905084868281548110151561175d57fe5b6000918252602090912001546001608060020a03161161177f57809250611786565b6001810391505b61173b565b858381548110151561179957fe5b60009182526020909120015470010000000000000000000000000000000090046001608060020a031693505b50505092915050565b81546000908190158061180d57506117e461145c565b8454859060001981019081106117f657fe5b6000918252602090912001546001608060020a0316105b15611885578354849061182382600183016118d0565b8154811061182d57fe5b90600052602060002001915061184161145c565b82546fffffffffffffffffffffffffffffffff19166001608060020a03918216178116700100000000000000000000000000000000918516919091021782556118ca565b83548490600019810190811061189757fe5b600091825260209091200180546001608060020a0380861670010000000000000000000000000000000002911617815590505b50505050565b81548183558181111561140a5760008381526020902061140a9181019083016108a691905b8082111561190957600081556001016118f5565b50905600a165627a7a723058203b4b8b2b078f541b06a353e7d147625bc703b2c60c6f9da487c4ffcaa076fdbe0029a165627a7a7230582040df622ed8c656bf809dd9a2283f5b7824ae505494e6713d4b51a925353d8cd80029"

// DeployMiniMeTokenFactory deploys a new Ethereum contract, binding an instance of MiniMeTokenFactory to it.
func DeployMiniMeTokenFactory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MiniMeTokenFactory, error) {
	parsed, err := abi.JSON(strings.NewReader(MiniMeTokenFactoryABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(MiniMeTokenFactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MiniMeTokenFactory{MiniMeTokenFactoryCaller: MiniMeTokenFactoryCaller{contract: contract}, MiniMeTokenFactoryTransactor: MiniMeTokenFactoryTransactor{contract: contract}, MiniMeTokenFactoryFilterer: MiniMeTokenFactoryFilterer{contract: contract}}, nil
}

// MiniMeTokenFactory is an auto generated Go binding around an Ethereum contract.
type MiniMeTokenFactory struct {
	MiniMeTokenFactoryCaller     // Read-only binding to the contract
	MiniMeTokenFactoryTransactor // Write-only binding to the contract
	MiniMeTokenFactoryFilterer   // Log filterer for contract events
}

// MiniMeTokenFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type MiniMeTokenFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MiniMeTokenFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MiniMeTokenFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MiniMeTokenFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MiniMeTokenFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MiniMeTokenFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MiniMeTokenFactorySession struct {
	Contract     *MiniMeTokenFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// MiniMeTokenFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MiniMeTokenFactoryCallerSession struct {
	Contract *MiniMeTokenFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// MiniMeTokenFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MiniMeTokenFactoryTransactorSession struct {
	Contract     *MiniMeTokenFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// MiniMeTokenFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type MiniMeTokenFactoryRaw struct {
	Contract *MiniMeTokenFactory // Generic contract binding to access the raw methods on
}

// MiniMeTokenFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MiniMeTokenFactoryCallerRaw struct {
	Contract *MiniMeTokenFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// MiniMeTokenFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MiniMeTokenFactoryTransactorRaw struct {
	Contract *MiniMeTokenFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMiniMeTokenFactory creates a new instance of MiniMeTokenFactory, bound to a specific deployed contract.
func NewMiniMeTokenFactory(address common.Address, backend bind.ContractBackend) (*MiniMeTokenFactory, error) {
	contract, err := bindMiniMeTokenFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenFactory{MiniMeTokenFactoryCaller: MiniMeTokenFactoryCaller{contract: contract}, MiniMeTokenFactoryTransactor: MiniMeTokenFactoryTransactor{contract: contract}, MiniMeTokenFactoryFilterer: MiniMeTokenFactoryFilterer{contract: contract}}, nil
}

// NewMiniMeTokenFactoryCaller creates a new read-only instance of MiniMeTokenFactory, bound to a specific deployed contract.
func NewMiniMeTokenFactoryCaller(address common.Address, caller bind.ContractCaller) (*MiniMeTokenFactoryCaller, error) {
	contract, err := bindMiniMeTokenFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenFactoryCaller{contract: contract}, nil
}

// NewMiniMeTokenFactoryTransactor creates a new write-only instance of MiniMeTokenFactory, bound to a specific deployed contract.
func NewMiniMeTokenFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*MiniMeTokenFactoryTransactor, error) {
	contract, err := bindMiniMeTokenFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenFactoryTransactor{contract: contract}, nil
}

// NewMiniMeTokenFactoryFilterer creates a new log filterer instance of MiniMeTokenFactory, bound to a specific deployed contract.
func NewMiniMeTokenFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*MiniMeTokenFactoryFilterer, error) {
	contract, err := bindMiniMeTokenFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MiniMeTokenFactoryFilterer{contract: contract}, nil
}

// bindMiniMeTokenFactory binds a generic wrapper to an already deployed contract.
func bindMiniMeTokenFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MiniMeTokenFactoryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MiniMeTokenFactory *MiniMeTokenFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MiniMeTokenFactory.Contract.MiniMeTokenFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MiniMeTokenFactory *MiniMeTokenFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MiniMeTokenFactory.Contract.MiniMeTokenFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MiniMeTokenFactory *MiniMeTokenFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MiniMeTokenFactory.Contract.MiniMeTokenFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MiniMeTokenFactory *MiniMeTokenFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MiniMeTokenFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MiniMeTokenFactory *MiniMeTokenFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MiniMeTokenFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MiniMeTokenFactory *MiniMeTokenFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MiniMeTokenFactory.Contract.contract.Transact(opts, method, params...)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x5b7b72c1.
//
// Solidity: function createCloneToken(address _parentToken, uint256 _snapshotBlock, string _tokenName, uint8 _decimalUnits, string _tokenSymbol, bool _transfersEnabled) returns(address)
func (_MiniMeTokenFactory *MiniMeTokenFactoryTransactor) CreateCloneToken(opts *bind.TransactOpts, _parentToken common.Address, _snapshotBlock *big.Int, _tokenName string, _decimalUnits uint8, _tokenSymbol string, _transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeTokenFactory.contract.Transact(opts, "createCloneToken", _parentToken, _snapshotBlock, _tokenName, _decimalUnits, _tokenSymbol, _transfersEnabled)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x5b7b72c1.
//
// Solidity: function createCloneToken(address _parentToken, uint256 _snapshotBlock, string _tokenName, uint8 _decimalUnits, string _tokenSymbol, bool _transfersEnabled) returns(address)
func (_MiniMeTokenFactory *MiniMeTokenFactorySession) CreateCloneToken(_parentToken common.Address, _snapshotBlock *big.Int, _tokenName string, _decimalUnits uint8, _tokenSymbol string, _transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeTokenFactory.Contract.CreateCloneToken(&_MiniMeTokenFactory.TransactOpts, _parentToken, _snapshotBlock, _tokenName, _decimalUnits, _tokenSymbol, _transfersEnabled)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x5b7b72c1.
//
// Solidity: function createCloneToken(address _parentToken, uint256 _snapshotBlock, string _tokenName, uint8 _decimalUnits, string _tokenSymbol, bool _transfersEnabled) returns(address)
func (_MiniMeTokenFactory *MiniMeTokenFactoryTransactorSession) CreateCloneToken(_parentToken common.Address, _snapshotBlock *big.Int, _tokenName string, _decimalUnits uint8, _tokenSymbol string, _transfersEnabled bool) (*types.Transaction, error) {
	return _MiniMeTokenFactory.Contract.CreateCloneToken(&_MiniMeTokenFactory.TransactOpts, _parentToken, _snapshotBlock, _tokenName, _decimalUnits, _tokenSymbol, _transfersEnabled)
}

// OwnedABI is the input ABI used to generate the binding from.
const OwnedABI = "[{\"constant\":false,\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"changeOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"newOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// OwnedFuncSigs maps the 4-byte function signature to its string representation.
var OwnedFuncSigs = map[string]string{
	"79ba5097": "acceptOwnership()",
	"a6f9dae1": "changeOwner(address)",
	"d4ee1d90": "newOwner()",
	"8da5cb5b": "owner()",
}

// OwnedBin is the compiled bytecode used for deploying new contracts.
var OwnedBin = "0x608060405234801561001057600080fd5b5060008054600160a060020a031916331790556101b9806100326000396000f3006080604052600436106100615763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166379ba509781146100665780638da5cb5b1461007d578063a6f9dae1146100ae578063d4ee1d90146100cf575b600080fd5b34801561007257600080fd5b5061007b6100e4565b005b34801561008957600080fd5b50610092610129565b60408051600160a060020a039092168252519081900360200190f35b3480156100ba57600080fd5b5061007b600160a060020a0360043516610138565b3480156100db57600080fd5b5061009261017e565b600154600160a060020a0316331415610127576001546000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a039092169190911790555b565b600054600160a060020a031681565b600054600160a060020a0316331461014f57600080fd5b6001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600154600160a060020a0316815600a165627a7a7230582005289f902d38637cac2795c372da977dc87c74f60e013557f0ccb45b8de379430029"

// DeployOwned deploys a new Ethereum contract, binding an instance of Owned to it.
func DeployOwned(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Owned, error) {
	parsed, err := abi.JSON(strings.NewReader(OwnedABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(OwnedBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Owned{OwnedCaller: OwnedCaller{contract: contract}, OwnedTransactor: OwnedTransactor{contract: contract}, OwnedFilterer: OwnedFilterer{contract: contract}}, nil
}

// Owned is an auto generated Go binding around an Ethereum contract.
type Owned struct {
	OwnedCaller     // Read-only binding to the contract
	OwnedTransactor // Write-only binding to the contract
	OwnedFilterer   // Log filterer for contract events
}

// OwnedCaller is an auto generated read-only Go binding around an Ethereum contract.
type OwnedCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnedTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OwnedTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnedFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OwnedFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnedSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OwnedSession struct {
	Contract     *Owned            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnedCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OwnedCallerSession struct {
	Contract *OwnedCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// OwnedTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OwnedTransactorSession struct {
	Contract     *OwnedTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnedRaw is an auto generated low-level Go binding around an Ethereum contract.
type OwnedRaw struct {
	Contract *Owned // Generic contract binding to access the raw methods on
}

// OwnedCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OwnedCallerRaw struct {
	Contract *OwnedCaller // Generic read-only contract binding to access the raw methods on
}

// OwnedTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OwnedTransactorRaw struct {
	Contract *OwnedTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOwned creates a new instance of Owned, bound to a specific deployed contract.
func NewOwned(address common.Address, backend bind.ContractBackend) (*Owned, error) {
	contract, err := bindOwned(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Owned{OwnedCaller: OwnedCaller{contract: contract}, OwnedTransactor: OwnedTransactor{contract: contract}, OwnedFilterer: OwnedFilterer{contract: contract}}, nil
}

// NewOwnedCaller creates a new read-only instance of Owned, bound to a specific deployed contract.
func NewOwnedCaller(address common.Address, caller bind.ContractCaller) (*OwnedCaller, error) {
	contract, err := bindOwned(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OwnedCaller{contract: contract}, nil
}

// NewOwnedTransactor creates a new write-only instance of Owned, bound to a specific deployed contract.
func NewOwnedTransactor(address common.Address, transactor bind.ContractTransactor) (*OwnedTransactor, error) {
	contract, err := bindOwned(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OwnedTransactor{contract: contract}, nil
}

// NewOwnedFilterer creates a new log filterer instance of Owned, bound to a specific deployed contract.
func NewOwnedFilterer(address common.Address, filterer bind.ContractFilterer) (*OwnedFilterer, error) {
	contract, err := bindOwned(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OwnedFilterer{contract: contract}, nil
}

// bindOwned binds a generic wrapper to an already deployed contract.
func bindOwned(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OwnedABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Owned *OwnedRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Owned.Contract.OwnedCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Owned *OwnedRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Owned.Contract.OwnedTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Owned *OwnedRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Owned.Contract.OwnedTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Owned *OwnedCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Owned.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Owned *OwnedTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Owned.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Owned *OwnedTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Owned.Contract.contract.Transact(opts, method, params...)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_Owned *OwnedCaller) NewOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Owned.contract.Call(opts, &out, "newOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_Owned *OwnedSession) NewOwner() (common.Address, error) {
	return _Owned.Contract.NewOwner(&_Owned.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_Owned *OwnedCallerSession) NewOwner() (common.Address, error) {
	return _Owned.Contract.NewOwner(&_Owned.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Owned *OwnedCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Owned.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Owned *OwnedSession) Owner() (common.Address, error) {
	return _Owned.Contract.Owner(&_Owned.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Owned *OwnedCallerSession) Owner() (common.Address, error) {
	return _Owned.Contract.Owner(&_Owned.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Owned *OwnedTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Owned.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Owned *OwnedSession) AcceptOwnership() (*types.Transaction, error) {
	return _Owned.Contract.AcceptOwnership(&_Owned.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Owned *OwnedTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _Owned.Contract.AcceptOwnership(&_Owned.TransactOpts)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_Owned *OwnedTransactor) ChangeOwner(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _Owned.contract.Transact(opts, "changeOwner", _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_Owned *OwnedSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _Owned.Contract.ChangeOwner(&_Owned.TransactOpts, _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_Owned *OwnedTransactorSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _Owned.Contract.ChangeOwner(&_Owned.TransactOpts, _newOwner)
}

// SGTExchangerABI is the input ABI used to generate the binding from.
const SGTExchangerABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"snt\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"sgt\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"collected\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"onTransfer\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"statusContribution\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"changeOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"newOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"onApprove\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalCollected\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"collect\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"proxyPayment\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_sgt\",\"type\":\"address\"},{\"name\":\"_snt\",\"type\":\"address\"},{\"name\":\"_statusContribution\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_token\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_controller\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"ClaimedTokens\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_holder\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"TokensCollected\",\"type\":\"event\"}]"

// SGTExchangerFuncSigs maps the 4-byte function signature to its string representation.
var SGTExchangerFuncSigs = map[string]string{
	"79ba5097": "acceptOwnership()",
	"a6f9dae1": "changeOwner(address)",
	"df8de3e7": "claimTokens(address)",
	"e5225381": "collect()",
	"38e43840": "collected(address)",
	"d4ee1d90": "newOwner()",
	"da682aeb": "onApprove(address,address,uint256)",
	"4a393149": "onTransfer(address,address,uint256)",
	"8da5cb5b": "owner()",
	"f48c3054": "proxyPayment(address)",
	"357a0ba2": "sgt()",
	"060eb520": "snt()",
	"52d50408": "statusContribution()",
	"e29eb836": "totalCollected()",
}

// SGTExchangerBin is the compiled bytecode used for deploying new contracts.
var SGTExchangerBin = "0x608060405234801561001057600080fd5b50604051606080610a848339810160409081528151602083015191909201516000805433600160a060020a0319918216178255600480548216600160a060020a039687161790556005805482169486169490941790935560068054909316939091169290921790556109fc90819061008890396000f3006080604052600436106100cf5763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663060eb52081146100d4578063357a0ba21461010557806338e438401461011a5780634a3931491461014d57806352d504081461018b57806379ba5097146101a05780638da5cb5b146101b7578063a6f9dae1146101cc578063d4ee1d90146101ed578063da682aeb1461014d578063df8de3e714610202578063e29eb83614610223578063e522538114610238578063f48c30541461024d575b600080fd5b3480156100e057600080fd5b506100e9610261565b60408051600160a060020a039092168252519081900360200190f35b34801561011157600080fd5b506100e9610270565b34801561012657600080fd5b5061013b600160a060020a036004351661027f565b60408051918252519081900360200190f35b34801561015957600080fd5b50610177600160a060020a0360043581169060243516604435610291565b604080519115158252519081900360200190f35b34801561019757600080fd5b506100e961029a565b3480156101ac57600080fd5b506101b56102a9565b005b3480156101c357600080fd5b506100e96102ee565b3480156101d857600080fd5b506101b5600160a060020a03600435166102fd565b3480156101f957600080fd5b506100e9610343565b34801561020e57600080fd5b506101b5600160a060020a0360043516610352565b34801561022f57600080fd5b5061013b610554565b34801561024457600080fd5b506101b561055a565b610177600160a060020a0360043516610962565b600554600160a060020a031681565b600454600160a060020a031681565b60026020526000908152604090205481565b60009392505050565b600654600160a060020a031681565b600154600160a060020a03163314156102ec576001546000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a039092169190911790555b565b600054600160a060020a031681565b600054600160a060020a0316331461031457600080fd5b6001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600154600160a060020a031681565b600080548190600160a060020a0316331461036c57600080fd5b600554600160a060020a038481169116141561038757600080fd5b600160a060020a03831615156103d85760008054604051600160a060020a0390911691303180156108fc02929091818181858888f193505050501580156103d2573d6000803e3d6000fd5b5061054f565b604080517f70a082310000000000000000000000000000000000000000000000000000000081523060048201529051849350600160a060020a038416916370a082319160248083019260209291908290030181600087803b15801561043c57600080fd5b505af1158015610450573d6000803e3d6000fd5b505050506040513d602081101561046657600080fd5b505160008054604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a0392831660048201526024810185905290519394509085169263a9059cbb92604480840193602093929083900390910190829087803b1580156104dc57600080fd5b505af11580156104f0573d6000803e3d6000fd5b505050506040513d602081101561050657600080fd5b5050600054604080518381529051600160a060020a03928316928616917ff931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c919081900360200190a35b505050565b60035481565b600080600080600660009054906101000a9004600160a060020a0316600160a060020a0316634084c3ab6040518163ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401602060405180830381600087803b1580156105cc57600080fd5b505af11580156105e0573d6000803e3d6000fd5b505050506040513d60208110156105f657600080fd5b5051935083151561060657600080fd5b8361060f610969565b1161061957600080fd5b600554604080517f70a0823100000000000000000000000000000000000000000000000000000000815230600482015290516106bb92600160a060020a0316916370a082319160248083019260209291908290030181600087803b15801561068057600080fd5b505af1158015610694573d6000803e3d6000fd5b505050506040513d60208110156106aa57600080fd5b50516003549063ffffffff61096d16565b60048054604080517f4ee2cd7e00000000000000000000000000000000000000000000000000000000815233938101939093526024830188905251929550600160a060020a031691634ee2cd7e916044808201926020929091908290030181600087803b15801561072b57600080fd5b505af115801561073f573d6000803e3d6000fd5b505050506040513d602081101561075557600080fd5b505160048054604080517f981b24d00000000000000000000000000000000000000000000000000000000081529283018890525192945061080b92600160a060020a039091169163981b24d09160248083019260209291908290030181600087803b1580156107c357600080fd5b505af11580156107d7573d6000803e3d6000fd5b505050506040513d60208110156107ed57600080fd5b50516107ff858563ffffffff61098316565b9063ffffffff6109a716565b3360009081526002602052604090205490915061082f90829063ffffffff6109be16565b90506000811161083e57600080fd5b600354610851908263ffffffff61096d16565b60035533600090815260026020526040902054610874908263ffffffff61096d16565b3360008181526002602090815260408083209490945560055484517fa9059cbb0000000000000000000000000000000000000000000000000000000081526004810194909452602484018690529351600160a060020a039094169363a9059cbb93604480820194918390030190829087803b1580156108f257600080fd5b505af1158015610906573d6000803e3d6000fd5b505050506040513d602081101561091c57600080fd5b5051151561092657fe5b60408051828152905133917f9381e53ffdc9733a6783a6f8665be3f89c231bb81a6771996ed553b4e75c0fe3919081900360200190a250505050565b6000806000fd5b4390565b60008282018381101561097c57fe5b9392505050565b600082820283158061099f575082848281151561099c57fe5b04145b151561097c57fe5b60008082848115156109b557fe5b04949350505050565b6000828211156109ca57fe5b509003905600a165627a7a7230582094bb84d3a41d1eaead9479c9c00da46ed01c6708e49829f2766ae13ef04518d50029"

// DeploySGTExchanger deploys a new Ethereum contract, binding an instance of SGTExchanger to it.
func DeploySGTExchanger(auth *bind.TransactOpts, backend bind.ContractBackend, _sgt common.Address, _snt common.Address, _statusContribution common.Address) (common.Address, *types.Transaction, *SGTExchanger, error) {
	parsed, err := abi.JSON(strings.NewReader(SGTExchangerABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SGTExchangerBin), backend, _sgt, _snt, _statusContribution)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SGTExchanger{SGTExchangerCaller: SGTExchangerCaller{contract: contract}, SGTExchangerTransactor: SGTExchangerTransactor{contract: contract}, SGTExchangerFilterer: SGTExchangerFilterer{contract: contract}}, nil
}

// SGTExchanger is an auto generated Go binding around an Ethereum contract.
type SGTExchanger struct {
	SGTExchangerCaller     // Read-only binding to the contract
	SGTExchangerTransactor // Write-only binding to the contract
	SGTExchangerFilterer   // Log filterer for contract events
}

// SGTExchangerCaller is an auto generated read-only Go binding around an Ethereum contract.
type SGTExchangerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SGTExchangerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SGTExchangerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SGTExchangerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SGTExchangerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SGTExchangerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SGTExchangerSession struct {
	Contract     *SGTExchanger     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SGTExchangerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SGTExchangerCallerSession struct {
	Contract *SGTExchangerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// SGTExchangerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SGTExchangerTransactorSession struct {
	Contract     *SGTExchangerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// SGTExchangerRaw is an auto generated low-level Go binding around an Ethereum contract.
type SGTExchangerRaw struct {
	Contract *SGTExchanger // Generic contract binding to access the raw methods on
}

// SGTExchangerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SGTExchangerCallerRaw struct {
	Contract *SGTExchangerCaller // Generic read-only contract binding to access the raw methods on
}

// SGTExchangerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SGTExchangerTransactorRaw struct {
	Contract *SGTExchangerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSGTExchanger creates a new instance of SGTExchanger, bound to a specific deployed contract.
func NewSGTExchanger(address common.Address, backend bind.ContractBackend) (*SGTExchanger, error) {
	contract, err := bindSGTExchanger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SGTExchanger{SGTExchangerCaller: SGTExchangerCaller{contract: contract}, SGTExchangerTransactor: SGTExchangerTransactor{contract: contract}, SGTExchangerFilterer: SGTExchangerFilterer{contract: contract}}, nil
}

// NewSGTExchangerCaller creates a new read-only instance of SGTExchanger, bound to a specific deployed contract.
func NewSGTExchangerCaller(address common.Address, caller bind.ContractCaller) (*SGTExchangerCaller, error) {
	contract, err := bindSGTExchanger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SGTExchangerCaller{contract: contract}, nil
}

// NewSGTExchangerTransactor creates a new write-only instance of SGTExchanger, bound to a specific deployed contract.
func NewSGTExchangerTransactor(address common.Address, transactor bind.ContractTransactor) (*SGTExchangerTransactor, error) {
	contract, err := bindSGTExchanger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SGTExchangerTransactor{contract: contract}, nil
}

// NewSGTExchangerFilterer creates a new log filterer instance of SGTExchanger, bound to a specific deployed contract.
func NewSGTExchangerFilterer(address common.Address, filterer bind.ContractFilterer) (*SGTExchangerFilterer, error) {
	contract, err := bindSGTExchanger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SGTExchangerFilterer{contract: contract}, nil
}

// bindSGTExchanger binds a generic wrapper to an already deployed contract.
func bindSGTExchanger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SGTExchangerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SGTExchanger *SGTExchangerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SGTExchanger.Contract.SGTExchangerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SGTExchanger *SGTExchangerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SGTExchanger.Contract.SGTExchangerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SGTExchanger *SGTExchangerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SGTExchanger.Contract.SGTExchangerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SGTExchanger *SGTExchangerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SGTExchanger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SGTExchanger *SGTExchangerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SGTExchanger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SGTExchanger *SGTExchangerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SGTExchanger.Contract.contract.Transact(opts, method, params...)
}

// Collected is a free data retrieval call binding the contract method 0x38e43840.
//
// Solidity: function collected(address ) view returns(uint256)
func (_SGTExchanger *SGTExchangerCaller) Collected(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _SGTExchanger.contract.Call(opts, &out, "collected", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Collected is a free data retrieval call binding the contract method 0x38e43840.
//
// Solidity: function collected(address ) view returns(uint256)
func (_SGTExchanger *SGTExchangerSession) Collected(arg0 common.Address) (*big.Int, error) {
	return _SGTExchanger.Contract.Collected(&_SGTExchanger.CallOpts, arg0)
}

// Collected is a free data retrieval call binding the contract method 0x38e43840.
//
// Solidity: function collected(address ) view returns(uint256)
func (_SGTExchanger *SGTExchangerCallerSession) Collected(arg0 common.Address) (*big.Int, error) {
	return _SGTExchanger.Contract.Collected(&_SGTExchanger.CallOpts, arg0)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_SGTExchanger *SGTExchangerCaller) NewOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SGTExchanger.contract.Call(opts, &out, "newOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_SGTExchanger *SGTExchangerSession) NewOwner() (common.Address, error) {
	return _SGTExchanger.Contract.NewOwner(&_SGTExchanger.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_SGTExchanger *SGTExchangerCallerSession) NewOwner() (common.Address, error) {
	return _SGTExchanger.Contract.NewOwner(&_SGTExchanger.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SGTExchanger *SGTExchangerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SGTExchanger.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SGTExchanger *SGTExchangerSession) Owner() (common.Address, error) {
	return _SGTExchanger.Contract.Owner(&_SGTExchanger.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SGTExchanger *SGTExchangerCallerSession) Owner() (common.Address, error) {
	return _SGTExchanger.Contract.Owner(&_SGTExchanger.CallOpts)
}

// Sgt is a free data retrieval call binding the contract method 0x357a0ba2.
//
// Solidity: function sgt() view returns(address)
func (_SGTExchanger *SGTExchangerCaller) Sgt(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SGTExchanger.contract.Call(opts, &out, "sgt")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Sgt is a free data retrieval call binding the contract method 0x357a0ba2.
//
// Solidity: function sgt() view returns(address)
func (_SGTExchanger *SGTExchangerSession) Sgt() (common.Address, error) {
	return _SGTExchanger.Contract.Sgt(&_SGTExchanger.CallOpts)
}

// Sgt is a free data retrieval call binding the contract method 0x357a0ba2.
//
// Solidity: function sgt() view returns(address)
func (_SGTExchanger *SGTExchangerCallerSession) Sgt() (common.Address, error) {
	return _SGTExchanger.Contract.Sgt(&_SGTExchanger.CallOpts)
}

// Snt is a free data retrieval call binding the contract method 0x060eb520.
//
// Solidity: function snt() view returns(address)
func (_SGTExchanger *SGTExchangerCaller) Snt(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SGTExchanger.contract.Call(opts, &out, "snt")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Snt is a free data retrieval call binding the contract method 0x060eb520.
//
// Solidity: function snt() view returns(address)
func (_SGTExchanger *SGTExchangerSession) Snt() (common.Address, error) {
	return _SGTExchanger.Contract.Snt(&_SGTExchanger.CallOpts)
}

// Snt is a free data retrieval call binding the contract method 0x060eb520.
//
// Solidity: function snt() view returns(address)
func (_SGTExchanger *SGTExchangerCallerSession) Snt() (common.Address, error) {
	return _SGTExchanger.Contract.Snt(&_SGTExchanger.CallOpts)
}

// StatusContribution is a free data retrieval call binding the contract method 0x52d50408.
//
// Solidity: function statusContribution() view returns(address)
func (_SGTExchanger *SGTExchangerCaller) StatusContribution(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SGTExchanger.contract.Call(opts, &out, "statusContribution")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StatusContribution is a free data retrieval call binding the contract method 0x52d50408.
//
// Solidity: function statusContribution() view returns(address)
func (_SGTExchanger *SGTExchangerSession) StatusContribution() (common.Address, error) {
	return _SGTExchanger.Contract.StatusContribution(&_SGTExchanger.CallOpts)
}

// StatusContribution is a free data retrieval call binding the contract method 0x52d50408.
//
// Solidity: function statusContribution() view returns(address)
func (_SGTExchanger *SGTExchangerCallerSession) StatusContribution() (common.Address, error) {
	return _SGTExchanger.Contract.StatusContribution(&_SGTExchanger.CallOpts)
}

// TotalCollected is a free data retrieval call binding the contract method 0xe29eb836.
//
// Solidity: function totalCollected() view returns(uint256)
func (_SGTExchanger *SGTExchangerCaller) TotalCollected(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SGTExchanger.contract.Call(opts, &out, "totalCollected")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalCollected is a free data retrieval call binding the contract method 0xe29eb836.
//
// Solidity: function totalCollected() view returns(uint256)
func (_SGTExchanger *SGTExchangerSession) TotalCollected() (*big.Int, error) {
	return _SGTExchanger.Contract.TotalCollected(&_SGTExchanger.CallOpts)
}

// TotalCollected is a free data retrieval call binding the contract method 0xe29eb836.
//
// Solidity: function totalCollected() view returns(uint256)
func (_SGTExchanger *SGTExchangerCallerSession) TotalCollected() (*big.Int, error) {
	return _SGTExchanger.Contract.TotalCollected(&_SGTExchanger.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SGTExchanger *SGTExchangerTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SGTExchanger.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SGTExchanger *SGTExchangerSession) AcceptOwnership() (*types.Transaction, error) {
	return _SGTExchanger.Contract.AcceptOwnership(&_SGTExchanger.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SGTExchanger *SGTExchangerTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _SGTExchanger.Contract.AcceptOwnership(&_SGTExchanger.TransactOpts)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_SGTExchanger *SGTExchangerTransactor) ChangeOwner(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _SGTExchanger.contract.Transact(opts, "changeOwner", _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_SGTExchanger *SGTExchangerSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _SGTExchanger.Contract.ChangeOwner(&_SGTExchanger.TransactOpts, _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_SGTExchanger *SGTExchangerTransactorSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _SGTExchanger.Contract.ChangeOwner(&_SGTExchanger.TransactOpts, _newOwner)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SGTExchanger *SGTExchangerTransactor) ClaimTokens(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _SGTExchanger.contract.Transact(opts, "claimTokens", _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SGTExchanger *SGTExchangerSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _SGTExchanger.Contract.ClaimTokens(&_SGTExchanger.TransactOpts, _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SGTExchanger *SGTExchangerTransactorSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _SGTExchanger.Contract.ClaimTokens(&_SGTExchanger.TransactOpts, _token)
}

// Collect is a paid mutator transaction binding the contract method 0xe5225381.
//
// Solidity: function collect() returns()
func (_SGTExchanger *SGTExchangerTransactor) Collect(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SGTExchanger.contract.Transact(opts, "collect")
}

// Collect is a paid mutator transaction binding the contract method 0xe5225381.
//
// Solidity: function collect() returns()
func (_SGTExchanger *SGTExchangerSession) Collect() (*types.Transaction, error) {
	return _SGTExchanger.Contract.Collect(&_SGTExchanger.TransactOpts)
}

// Collect is a paid mutator transaction binding the contract method 0xe5225381.
//
// Solidity: function collect() returns()
func (_SGTExchanger *SGTExchangerTransactorSession) Collect() (*types.Transaction, error) {
	return _SGTExchanger.Contract.Collect(&_SGTExchanger.TransactOpts)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address , address , uint256 ) returns(bool)
func (_SGTExchanger *SGTExchangerTransactor) OnApprove(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SGTExchanger.contract.Transact(opts, "onApprove", arg0, arg1, arg2)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address , address , uint256 ) returns(bool)
func (_SGTExchanger *SGTExchangerSession) OnApprove(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SGTExchanger.Contract.OnApprove(&_SGTExchanger.TransactOpts, arg0, arg1, arg2)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address , address , uint256 ) returns(bool)
func (_SGTExchanger *SGTExchangerTransactorSession) OnApprove(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SGTExchanger.Contract.OnApprove(&_SGTExchanger.TransactOpts, arg0, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address , address , uint256 ) returns(bool)
func (_SGTExchanger *SGTExchangerTransactor) OnTransfer(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SGTExchanger.contract.Transact(opts, "onTransfer", arg0, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address , address , uint256 ) returns(bool)
func (_SGTExchanger *SGTExchangerSession) OnTransfer(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SGTExchanger.Contract.OnTransfer(&_SGTExchanger.TransactOpts, arg0, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address , address , uint256 ) returns(bool)
func (_SGTExchanger *SGTExchangerTransactorSession) OnTransfer(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SGTExchanger.Contract.OnTransfer(&_SGTExchanger.TransactOpts, arg0, arg1, arg2)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address ) payable returns(bool)
func (_SGTExchanger *SGTExchangerTransactor) ProxyPayment(opts *bind.TransactOpts, arg0 common.Address) (*types.Transaction, error) {
	return _SGTExchanger.contract.Transact(opts, "proxyPayment", arg0)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address ) payable returns(bool)
func (_SGTExchanger *SGTExchangerSession) ProxyPayment(arg0 common.Address) (*types.Transaction, error) {
	return _SGTExchanger.Contract.ProxyPayment(&_SGTExchanger.TransactOpts, arg0)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address ) payable returns(bool)
func (_SGTExchanger *SGTExchangerTransactorSession) ProxyPayment(arg0 common.Address) (*types.Transaction, error) {
	return _SGTExchanger.Contract.ProxyPayment(&_SGTExchanger.TransactOpts, arg0)
}

// SGTExchangerClaimedTokensIterator is returned from FilterClaimedTokens and is used to iterate over the raw logs and unpacked data for ClaimedTokens events raised by the SGTExchanger contract.
type SGTExchangerClaimedTokensIterator struct {
	Event *SGTExchangerClaimedTokens // Event containing the contract specifics and raw log

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
func (it *SGTExchangerClaimedTokensIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SGTExchangerClaimedTokens)
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
		it.Event = new(SGTExchangerClaimedTokens)
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
func (it *SGTExchangerClaimedTokensIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SGTExchangerClaimedTokensIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SGTExchangerClaimedTokens represents a ClaimedTokens event raised by the SGTExchanger contract.
type SGTExchangerClaimedTokens struct {
	Token      common.Address
	Controller common.Address
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterClaimedTokens is a free log retrieval operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SGTExchanger *SGTExchangerFilterer) FilterClaimedTokens(opts *bind.FilterOpts, _token []common.Address, _controller []common.Address) (*SGTExchangerClaimedTokensIterator, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _SGTExchanger.contract.FilterLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return &SGTExchangerClaimedTokensIterator{contract: _SGTExchanger.contract, event: "ClaimedTokens", logs: logs, sub: sub}, nil
}

// WatchClaimedTokens is a free log subscription operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SGTExchanger *SGTExchangerFilterer) WatchClaimedTokens(opts *bind.WatchOpts, sink chan<- *SGTExchangerClaimedTokens, _token []common.Address, _controller []common.Address) (event.Subscription, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _SGTExchanger.contract.WatchLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SGTExchangerClaimedTokens)
				if err := _SGTExchanger.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
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

// ParseClaimedTokens is a log parse operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SGTExchanger *SGTExchangerFilterer) ParseClaimedTokens(log types.Log) (*SGTExchangerClaimedTokens, error) {
	event := new(SGTExchangerClaimedTokens)
	if err := _SGTExchanger.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SGTExchangerTokensCollectedIterator is returned from FilterTokensCollected and is used to iterate over the raw logs and unpacked data for TokensCollected events raised by the SGTExchanger contract.
type SGTExchangerTokensCollectedIterator struct {
	Event *SGTExchangerTokensCollected // Event containing the contract specifics and raw log

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
func (it *SGTExchangerTokensCollectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SGTExchangerTokensCollected)
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
		it.Event = new(SGTExchangerTokensCollected)
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
func (it *SGTExchangerTokensCollectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SGTExchangerTokensCollectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SGTExchangerTokensCollected represents a TokensCollected event raised by the SGTExchanger contract.
type SGTExchangerTokensCollected struct {
	Holder common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTokensCollected is a free log retrieval operation binding the contract event 0x9381e53ffdc9733a6783a6f8665be3f89c231bb81a6771996ed553b4e75c0fe3.
//
// Solidity: event TokensCollected(address indexed _holder, uint256 _amount)
func (_SGTExchanger *SGTExchangerFilterer) FilterTokensCollected(opts *bind.FilterOpts, _holder []common.Address) (*SGTExchangerTokensCollectedIterator, error) {

	var _holderRule []interface{}
	for _, _holderItem := range _holder {
		_holderRule = append(_holderRule, _holderItem)
	}

	logs, sub, err := _SGTExchanger.contract.FilterLogs(opts, "TokensCollected", _holderRule)
	if err != nil {
		return nil, err
	}
	return &SGTExchangerTokensCollectedIterator{contract: _SGTExchanger.contract, event: "TokensCollected", logs: logs, sub: sub}, nil
}

// WatchTokensCollected is a free log subscription operation binding the contract event 0x9381e53ffdc9733a6783a6f8665be3f89c231bb81a6771996ed553b4e75c0fe3.
//
// Solidity: event TokensCollected(address indexed _holder, uint256 _amount)
func (_SGTExchanger *SGTExchangerFilterer) WatchTokensCollected(opts *bind.WatchOpts, sink chan<- *SGTExchangerTokensCollected, _holder []common.Address) (event.Subscription, error) {

	var _holderRule []interface{}
	for _, _holderItem := range _holder {
		_holderRule = append(_holderRule, _holderItem)
	}

	logs, sub, err := _SGTExchanger.contract.WatchLogs(opts, "TokensCollected", _holderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SGTExchangerTokensCollected)
				if err := _SGTExchanger.contract.UnpackLog(event, "TokensCollected", log); err != nil {
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

// ParseTokensCollected is a log parse operation binding the contract event 0x9381e53ffdc9733a6783a6f8665be3f89c231bb81a6771996ed553b4e75c0fe3.
//
// Solidity: event TokensCollected(address indexed _holder, uint256 _amount)
func (_SGTExchanger *SGTExchangerFilterer) ParseTokensCollected(log types.Log) (*SGTExchangerTokensCollected, error) {
	event := new(SGTExchangerTokensCollected)
	if err := _SGTExchanger.contract.UnpackLog(event, "TokensCollected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SNTABI is the input ABI used to generate the binding from.
const SNTABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"creationBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newController\",\"type\":\"address\"}],\"name\":\"changeController\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"balanceOfAt\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_cloneTokenName\",\"type\":\"string\"},{\"name\":\"_cloneDecimalUnits\",\"type\":\"uint8\"},{\"name\":\"_cloneTokenSymbol\",\"type\":\"string\"},{\"name\":\"_snapshotBlock\",\"type\":\"uint256\"},{\"name\":\"_transfersEnabled\",\"type\":\"bool\"}],\"name\":\"createCloneToken\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"parentToken\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"generateTokens\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"totalSupplyAt\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"transfersEnabled\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"parentSnapShotBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"},{\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"approveAndCall\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"destroyTokens\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"remaining\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"tokenFactory\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_transfersEnabled\",\"type\":\"bool\"}],\"name\":\"enableTransfers\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"controller\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_tokenFactory\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_token\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_controller\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"ClaimedTokens\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_cloneToken\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_snapshotBlock\",\"type\":\"uint256\"}],\"name\":\"NewCloneToken\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"}]"

// SNTFuncSigs maps the 4-byte function signature to its string representation.
var SNTFuncSigs = map[string]string{
	"dd62ed3e": "allowance(address,address)",
	"095ea7b3": "approve(address,uint256)",
	"cae9ca51": "approveAndCall(address,uint256,bytes)",
	"70a08231": "balanceOf(address)",
	"4ee2cd7e": "balanceOfAt(address,uint256)",
	"3cebb823": "changeController(address)",
	"df8de3e7": "claimTokens(address)",
	"f77c4791": "controller()",
	"6638c087": "createCloneToken(string,uint8,string,uint256,bool)",
	"17634514": "creationBlock()",
	"313ce567": "decimals()",
	"d3ce77fe": "destroyTokens(address,uint256)",
	"f41e60c5": "enableTransfers(bool)",
	"827f32c0": "generateTokens(address,uint256)",
	"06fdde03": "name()",
	"c5bcc4f1": "parentSnapShotBlock()",
	"80a54001": "parentToken()",
	"95d89b41": "symbol()",
	"e77772fe": "tokenFactory()",
	"18160ddd": "totalSupply()",
	"981b24d0": "totalSupplyAt(uint256)",
	"a9059cbb": "transfer(address,uint256)",
	"23b872dd": "transferFrom(address,address,uint256)",
	"bef97c87": "transfersEnabled()",
	"54fd4d50": "version()",
}

// SNTBin is the compiled bytecode used for deploying new contracts.
var SNTBin = "0x60c0604052600760808190527f4d4d545f302e310000000000000000000000000000000000000000000000000060a090815262000040916004919062000198565b503480156200004e57600080fd5b5060405160208062001b8383398101604081815291518282018352601482527f537461747573204e6574776f726b20546f6b656e00000000000000000000000060208084019182528451808601909552600385527f534e5400000000000000000000000000000000000000000000000000000000009085015260008054600160a060020a03191633178155600b8054600160a060020a0385166101000261010060a860020a031990911617905583519294859491938493601292916001916200011a9183919062000198565b506002805460ff191660ff851617905581516200013f90600390602085019062000198565b5060058054600160a060020a031916600160a060020a0388161790556006859055600b805460ff19168215151790556200018164010000000062000193810204565b600755506200023a9650505050505050565b435b90565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10620001db57805160ff19168380011785556200020b565b828001600101855582156200020b579182015b828111156200020b578251825591602001919060010190620001ee565b50620002199291506200021d565b5090565b6200019591905b8082111562000219576000815560010162000224565b611939806200024a6000396000f30060806040526004361061012f5763ffffffff60e060020a60003504166306fdde0381146101f3578063095ea7b31461027d57806317634514146102b557806318160ddd146102dc57806323b872dd146102f1578063313ce5671461031b5780633cebb823146103465780634ee2cd7e1461036757806354fd4d501461038b5780636638c087146103a057806370a082311461046357806380a5400114610484578063827f32c01461049957806395d89b41146104bd578063981b24d0146104d2578063a9059cbb146104ea578063bef97c871461050e578063c5bcc4f114610523578063cae9ca5114610538578063d3ce77fe146105a1578063dd62ed3e146105c5578063df8de3e7146105ec578063e77772fe1461060d578063f41e60c514610622578063f77c47911461063c575b60005461014490600160a060020a0316610651565b156101ec57600054604080517ff48c30540000000000000000000000000000000000000000000000000000000081523360048201529051600160a060020a039092169163f48c3054913491602480830192602092919082900301818588803b1580156101af57600080fd5b505af11580156101c3573d6000803e3d6000fd5b50505050506040513d60208110156101da57600080fd5b505115156101e757600080fd5b6101f1565b600080fd5b005b3480156101ff57600080fd5b5061020861067e565b6040805160208082528351818301528351919283929083019185019080838360005b8381101561024257818101518382015260200161022a565b50505050905090810190601f16801561026f5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561028957600080fd5b506102a1600160a060020a036004351660243561070b565b604080519115158252519081900360200190f35b3480156102c157600080fd5b506102ca61088b565b60408051918252519081900360200190f35b3480156102e857600080fd5b506102ca610891565b3480156102fd57600080fd5b506102a1600160a060020a03600435811690602435166044356108a9565b34801561032757600080fd5b50610330610940565b6040805160ff9092168252519081900360200190f35b34801561035257600080fd5b506101f1600160a060020a0360043516610949565b34801561037357600080fd5b506102ca600160a060020a036004351660243561098f565b34801561039757600080fd5b50610208610adc565b3480156103ac57600080fd5b506040805160206004803580820135601f810184900484028501840190955284845261044794369492936024939284019190819084018382808284375050604080516020601f818a01358b0180359182018390048302840183018552818452989b60ff8b35169b909a909994019750919550918201935091508190840183828082843750949750508435955050505050602001351515610b37565b60408051600160a060020a039092168252519081900360200190f35b34801561046f57600080fd5b506102ca600160a060020a0360043516610d98565b34801561049057600080fd5b50610447610db3565b3480156104a557600080fd5b506102a1600160a060020a0360043516602435610dc2565b3480156104c957600080fd5b50610208610e98565b3480156104de57600080fd5b506102ca600435610ef3565b3480156104f657600080fd5b506102a1600160a060020a0360043516602435610fe7565b34801561051a57600080fd5b506102a1611006565b34801561052f57600080fd5b506102ca61100f565b34801561054457600080fd5b50604080516020600460443581810135601f81018490048402850184019095528484526102a1948235600160a060020a03169460248035953695946064949201919081908401838280828437509497506110159650505050505050565b3480156105ad57600080fd5b506102a1600160a060020a0360043516602435611130565b3480156105d157600080fd5b506102ca600160a060020a03600435811690602435166111fd565b3480156105f857600080fd5b506101f1600160a060020a0360043516611228565b34801561061957600080fd5b5061044761140f565b34801561062e57600080fd5b506101f16004351515611423565b34801561064857600080fd5b5061044761144d565b600080600160a060020a038316151561066d5760009150610678565b823b90506000811191505b50919050565b60018054604080516020600284861615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b820191906000526020600020905b8154815290600101906020018083116106e657829003601f168201915b505050505081565b600b5460009060ff16151561071f57600080fd5b81158015906107505750336000908152600960209081526040808320600160a060020a038716845290915290205415155b1561075a57600080fd5b60005461076f90600160a060020a0316610651565b156108235760008054604080517fda682aeb000000000000000000000000000000000000000000000000000000008152336004820152600160a060020a038781166024830152604482018790529151919092169263da682aeb92606480820193602093909283900390910190829087803b1580156107ec57600080fd5b505af1158015610800573d6000803e3d6000fd5b505050506040513d602081101561081657600080fd5b5051151561082357600080fd5b336000818152600960209081526040808320600160a060020a03881680855290835292819020869055805186815290519293927f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925929181900390910190a35060015b92915050565b60075481565b60006108a361089e61145c565b610ef3565b90505b90565b60008054600160a060020a0316331461092b57600b5460ff1615156108cd57600080fd5b600160a060020a038416600090815260096020908152604080832033845290915290205482111561090057506000610939565b600160a060020a03841660009081526009602090815260408083203384529091529020805483900390555b610936848484611460565b90505b9392505050565b60025460ff1681565b600054600160a060020a0316331461096057600080fd5b6000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600160a060020a03821660009081526008602052604081205415806109eb5750600160a060020a0383166000908152600860205260408120805484929081106109d457fe5b6000918252602090912001546001608060020a0316115b15610ab357600554600160a060020a031615610aab57600554600654600160a060020a0390911690634ee2cd7e908590610a26908690611659565b6040518363ffffffff1660e060020a0281526004018083600160a060020a0316600160a060020a0316815260200182815260200192505050602060405180830381600087803b158015610a7857600080fd5b505af1158015610a8c573d6000803e3d6000fd5b505050506040513d6020811015610aa257600080fd5b50519050610885565b506000610885565b600160a060020a0383166000908152600860205260409020610ad5908361166f565b9050610885565b6004805460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b600080831515610b4c57610b4961145c565b93505b600b546040517f5b7b72c100000000000000000000000000000000000000000000000000000000815230600482018181526024830188905260ff8a16606484015286151560a484015260c0604484019081528b5160c48501528b51610100909504600160a060020a031694635b7b72c1948a938e938e938e938d939291608482019160e40190602089019080838360005b83811015610bf5578181015183820152602001610bdd565b50505050905090810190601f168015610c225780820380516001836020036101000a031916815260200191505b50838103825285518152855160209182019187019080838360005b83811015610c55578181015183820152602001610c3d565b50505050905090810190601f168015610c825780820380516001836020036101000a031916815260200191505b5098505050505050505050602060405180830381600087803b158015610ca757600080fd5b505af1158015610cbb573d6000803e3d6000fd5b505050506040513d6020811015610cd157600080fd5b5051604080517f3cebb8230000000000000000000000000000000000000000000000000000000081523360048201529051919250600160a060020a03831691633cebb8239160248082019260009290919082900301818387803b158015610d3757600080fd5b505af1158015610d4b573d6000803e3d6000fd5b5050604080518781529051600160a060020a03851693507f086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade92509081900360200190a29695505050505050565b6000610dab82610da661145c565b61098f565b90505b919050565b600554600160a060020a031681565b6000805481908190600160a060020a03163314610dde57600080fd5b610df0600a610deb61145c565b61166f565b9150818483011015610e0157600080fd5b610e0e600a8584016117ce565b610e1785610d98565b9050808482011015610e2857600080fd5b600160a060020a0385166000908152600860205260409020610e4c908286016117ce565b604080518581529051600160a060020a038716916000917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a3506001949350505050565b6003805460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292918301828280156107035780601f106106d857610100808354040283529160200191610703565b600a546000901580610f28575081600a6000815481101515610f1157fe5b6000918252602090912001546001608060020a0316115b15610fd557600554600160a060020a031615610fcd57600554600654600160a060020a039091169063981b24d090610f61908590611659565b6040518263ffffffff1660e060020a02815260040180828152602001915050602060405180830381600087803b158015610f9a57600080fd5b505af1158015610fae573d6000803e3d6000fd5b505050506040513d6020811015610fc457600080fd5b50519050610dae565b506000610dae565b610fe0600a8361166f565b9050610dae565b600b5460009060ff161515610ffb57600080fd5b610939338484611460565b600b5460ff1681565b60065481565b6000611021848461070b565b151561102c57600080fd5b6040517f8f4ffcb10000000000000000000000000000000000000000000000000000000081523360048201818152602483018690523060448401819052608060648501908152865160848601528651600160a060020a038a1695638f4ffcb195948a94938a939192909160a490910190602085019080838360005b838110156110bf5781810151838201526020016110a7565b50505050905090810190601f1680156110ec5780820380516001836020036101000a031916815260200191505b5095505050505050600060405180830381600087803b15801561110e57600080fd5b505af1158015611122573d6000803e3d6000fd5b506001979650505050505050565b6000805481908190600160a060020a0316331461114c57600080fd5b611159600a610deb61145c565b91508382101561116857600080fd5b611175600a8584036117ce565b61117e85610d98565b90508381101561118d57600080fd5b600160a060020a03851660009081526008602052604090206111b1908583036117ce565b604080518581529051600091600160a060020a038816917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a3506001949350505050565b600160a060020a03918216600090815260096020908152604080832093909416825291909152205490565b600080548190600160a060020a0316331461124257600080fd5b600160a060020a03831615156112935760008054604051600160a060020a0390911691303180156108fc02929091818181858888f1935050505015801561128d573d6000803e3d6000fd5b5061140a565b604080517f70a082310000000000000000000000000000000000000000000000000000000081523060048201529051849350600160a060020a038416916370a082319160248083019260209291908290030181600087803b1580156112f757600080fd5b505af115801561130b573d6000803e3d6000fd5b505050506040513d602081101561132157600080fd5b505160008054604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a0392831660048201526024810185905290519394509085169263a9059cbb92604480840193602093929083900390910190829087803b15801561139757600080fd5b505af11580156113ab573d6000803e3d6000fd5b505050506040513d60208110156113c157600080fd5b5050600054604080518381529051600160a060020a03928316928616917ff931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c919081900360200190a35b505050565b600b546101009004600160a060020a031681565b600054600160a060020a0316331461143a57600080fd5b600b805460ff1916911515919091179055565b600054600160a060020a031681565b4390565b600080808315156114745760019250611650565b61147c61145c565b6006541061148957600080fd5b600160a060020a03851615806114a75750600160a060020a03851630145b156114b157600080fd5b6114bd86610da661145c565b9150838210156114d05760009250611650565b6000546114e590600160a060020a0316610651565b1561159b5760008054604080517f4a393149000000000000000000000000000000000000000000000000000000008152600160a060020a038a8116600483015289811660248301526044820189905291519190921692634a39314992606480820193602093909283900390910190829087803b15801561156457600080fd5b505af1158015611578573d6000803e3d6000fd5b505050506040513d602081101561158e57600080fd5b5051151561159b57600080fd5b600160a060020a03861660009081526008602052604090206115bf908584036117ce565b6115cb85610da661145c565b90508084820110156115dc57600080fd5b600160a060020a0385166000908152600860205260409020611600908286016117ce565b84600160a060020a031686600160a060020a03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef866040518082815260200191505060405180910390a3600192505b50509392505050565b60008183106116685781610939565b5090919050565b60008060008085805490506000141561168b57600093506117c5565b85548690600019810190811061169d57fe5b6000918252602090912001546001608060020a031685106116fa578554869060001981019081106116ca57fe5b60009182526020909120015470010000000000000000000000000000000090046001608060020a031693506117c5565b85600081548110151561170957fe5b6000918252602090912001546001608060020a031685101561172e57600093506117c5565b8554600093506000190191505b8282111561178b57600260018385010104905084868281548110151561175d57fe5b6000918252602090912001546001608060020a03161161177f57809250611786565b6001810391505b61173b565b858381548110151561179957fe5b60009182526020909120015470010000000000000000000000000000000090046001608060020a031693505b50505092915050565b81546000908190158061180d57506117e461145c565b8454859060001981019081106117f657fe5b6000918252602090912001546001608060020a0316105b15611885578354849061182382600183016118d0565b8154811061182d57fe5b90600052602060002001915061184161145c565b82546fffffffffffffffffffffffffffffffff19166001608060020a03918216178116700100000000000000000000000000000000918516919091021782556118ca565b83548490600019810190811061189757fe5b600091825260209091200180546001608060020a0380861670010000000000000000000000000000000002911617815590505b50505050565b81548183558181111561140a5760008381526020902061140a9181019083016108a691905b8082111561190957600081556001016118f5565b50905600a165627a7a723058204901d258e6f53b0c2c4d0eb1bb60ed9df90c45c6053e8116ae2fc27e92bbbcb60029"

// DeploySNT deploys a new Ethereum contract, binding an instance of SNT to it.
func DeploySNT(auth *bind.TransactOpts, backend bind.ContractBackend, _tokenFactory common.Address) (common.Address, *types.Transaction, *SNT, error) {
	parsed, err := abi.JSON(strings.NewReader(SNTABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SNTBin), backend, _tokenFactory)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SNT{SNTCaller: SNTCaller{contract: contract}, SNTTransactor: SNTTransactor{contract: contract}, SNTFilterer: SNTFilterer{contract: contract}}, nil
}

// SNT is an auto generated Go binding around an Ethereum contract.
type SNT struct {
	SNTCaller     // Read-only binding to the contract
	SNTTransactor // Write-only binding to the contract
	SNTFilterer   // Log filterer for contract events
}

// SNTCaller is an auto generated read-only Go binding around an Ethereum contract.
type SNTCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SNTTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SNTTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SNTFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SNTFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SNTSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SNTSession struct {
	Contract     *SNT              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SNTCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SNTCallerSession struct {
	Contract *SNTCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// SNTTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SNTTransactorSession struct {
	Contract     *SNTTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SNTRaw is an auto generated low-level Go binding around an Ethereum contract.
type SNTRaw struct {
	Contract *SNT // Generic contract binding to access the raw methods on
}

// SNTCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SNTCallerRaw struct {
	Contract *SNTCaller // Generic read-only contract binding to access the raw methods on
}

// SNTTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SNTTransactorRaw struct {
	Contract *SNTTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSNT creates a new instance of SNT, bound to a specific deployed contract.
func NewSNT(address common.Address, backend bind.ContractBackend) (*SNT, error) {
	contract, err := bindSNT(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SNT{SNTCaller: SNTCaller{contract: contract}, SNTTransactor: SNTTransactor{contract: contract}, SNTFilterer: SNTFilterer{contract: contract}}, nil
}

// NewSNTCaller creates a new read-only instance of SNT, bound to a specific deployed contract.
func NewSNTCaller(address common.Address, caller bind.ContractCaller) (*SNTCaller, error) {
	contract, err := bindSNT(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SNTCaller{contract: contract}, nil
}

// NewSNTTransactor creates a new write-only instance of SNT, bound to a specific deployed contract.
func NewSNTTransactor(address common.Address, transactor bind.ContractTransactor) (*SNTTransactor, error) {
	contract, err := bindSNT(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SNTTransactor{contract: contract}, nil
}

// NewSNTFilterer creates a new log filterer instance of SNT, bound to a specific deployed contract.
func NewSNTFilterer(address common.Address, filterer bind.ContractFilterer) (*SNTFilterer, error) {
	contract, err := bindSNT(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SNTFilterer{contract: contract}, nil
}

// bindSNT binds a generic wrapper to an already deployed contract.
func bindSNT(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SNTABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SNT *SNTRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SNT.Contract.SNTCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SNT *SNTRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SNT.Contract.SNTTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SNT *SNTRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SNT.Contract.SNTTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SNT *SNTCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SNT.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SNT *SNTTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SNT.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SNT *SNTTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SNT.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_SNT *SNTCaller) Allowance(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "allowance", _owner, _spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_SNT *SNTSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _SNT.Contract.Allowance(&_SNT.CallOpts, _owner, _spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_SNT *SNTCallerSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _SNT.Contract.Allowance(&_SNT.CallOpts, _owner, _spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_SNT *SNTCaller) BalanceOf(opts *bind.CallOpts, _owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "balanceOf", _owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_SNT *SNTSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _SNT.Contract.BalanceOf(&_SNT.CallOpts, _owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_SNT *SNTCallerSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _SNT.Contract.BalanceOf(&_SNT.CallOpts, _owner)
}

// BalanceOfAt is a free data retrieval call binding the contract method 0x4ee2cd7e.
//
// Solidity: function balanceOfAt(address _owner, uint256 _blockNumber) view returns(uint256)
func (_SNT *SNTCaller) BalanceOfAt(opts *bind.CallOpts, _owner common.Address, _blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "balanceOfAt", _owner, _blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOfAt is a free data retrieval call binding the contract method 0x4ee2cd7e.
//
// Solidity: function balanceOfAt(address _owner, uint256 _blockNumber) view returns(uint256)
func (_SNT *SNTSession) BalanceOfAt(_owner common.Address, _blockNumber *big.Int) (*big.Int, error) {
	return _SNT.Contract.BalanceOfAt(&_SNT.CallOpts, _owner, _blockNumber)
}

// BalanceOfAt is a free data retrieval call binding the contract method 0x4ee2cd7e.
//
// Solidity: function balanceOfAt(address _owner, uint256 _blockNumber) view returns(uint256)
func (_SNT *SNTCallerSession) BalanceOfAt(_owner common.Address, _blockNumber *big.Int) (*big.Int, error) {
	return _SNT.Contract.BalanceOfAt(&_SNT.CallOpts, _owner, _blockNumber)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_SNT *SNTCaller) Controller(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "controller")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_SNT *SNTSession) Controller() (common.Address, error) {
	return _SNT.Contract.Controller(&_SNT.CallOpts)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() view returns(address)
func (_SNT *SNTCallerSession) Controller() (common.Address, error) {
	return _SNT.Contract.Controller(&_SNT.CallOpts)
}

// CreationBlock is a free data retrieval call binding the contract method 0x17634514.
//
// Solidity: function creationBlock() view returns(uint256)
func (_SNT *SNTCaller) CreationBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "creationBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CreationBlock is a free data retrieval call binding the contract method 0x17634514.
//
// Solidity: function creationBlock() view returns(uint256)
func (_SNT *SNTSession) CreationBlock() (*big.Int, error) {
	return _SNT.Contract.CreationBlock(&_SNT.CallOpts)
}

// CreationBlock is a free data retrieval call binding the contract method 0x17634514.
//
// Solidity: function creationBlock() view returns(uint256)
func (_SNT *SNTCallerSession) CreationBlock() (*big.Int, error) {
	return _SNT.Contract.CreationBlock(&_SNT.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_SNT *SNTCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_SNT *SNTSession) Decimals() (uint8, error) {
	return _SNT.Contract.Decimals(&_SNT.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_SNT *SNTCallerSession) Decimals() (uint8, error) {
	return _SNT.Contract.Decimals(&_SNT.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_SNT *SNTCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_SNT *SNTSession) Name() (string, error) {
	return _SNT.Contract.Name(&_SNT.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_SNT *SNTCallerSession) Name() (string, error) {
	return _SNT.Contract.Name(&_SNT.CallOpts)
}

// ParentSnapShotBlock is a free data retrieval call binding the contract method 0xc5bcc4f1.
//
// Solidity: function parentSnapShotBlock() view returns(uint256)
func (_SNT *SNTCaller) ParentSnapShotBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "parentSnapShotBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ParentSnapShotBlock is a free data retrieval call binding the contract method 0xc5bcc4f1.
//
// Solidity: function parentSnapShotBlock() view returns(uint256)
func (_SNT *SNTSession) ParentSnapShotBlock() (*big.Int, error) {
	return _SNT.Contract.ParentSnapShotBlock(&_SNT.CallOpts)
}

// ParentSnapShotBlock is a free data retrieval call binding the contract method 0xc5bcc4f1.
//
// Solidity: function parentSnapShotBlock() view returns(uint256)
func (_SNT *SNTCallerSession) ParentSnapShotBlock() (*big.Int, error) {
	return _SNT.Contract.ParentSnapShotBlock(&_SNT.CallOpts)
}

// ParentToken is a free data retrieval call binding the contract method 0x80a54001.
//
// Solidity: function parentToken() view returns(address)
func (_SNT *SNTCaller) ParentToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "parentToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ParentToken is a free data retrieval call binding the contract method 0x80a54001.
//
// Solidity: function parentToken() view returns(address)
func (_SNT *SNTSession) ParentToken() (common.Address, error) {
	return _SNT.Contract.ParentToken(&_SNT.CallOpts)
}

// ParentToken is a free data retrieval call binding the contract method 0x80a54001.
//
// Solidity: function parentToken() view returns(address)
func (_SNT *SNTCallerSession) ParentToken() (common.Address, error) {
	return _SNT.Contract.ParentToken(&_SNT.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_SNT *SNTCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_SNT *SNTSession) Symbol() (string, error) {
	return _SNT.Contract.Symbol(&_SNT.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_SNT *SNTCallerSession) Symbol() (string, error) {
	return _SNT.Contract.Symbol(&_SNT.CallOpts)
}

// TokenFactory is a free data retrieval call binding the contract method 0xe77772fe.
//
// Solidity: function tokenFactory() view returns(address)
func (_SNT *SNTCaller) TokenFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "tokenFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TokenFactory is a free data retrieval call binding the contract method 0xe77772fe.
//
// Solidity: function tokenFactory() view returns(address)
func (_SNT *SNTSession) TokenFactory() (common.Address, error) {
	return _SNT.Contract.TokenFactory(&_SNT.CallOpts)
}

// TokenFactory is a free data retrieval call binding the contract method 0xe77772fe.
//
// Solidity: function tokenFactory() view returns(address)
func (_SNT *SNTCallerSession) TokenFactory() (common.Address, error) {
	return _SNT.Contract.TokenFactory(&_SNT.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_SNT *SNTCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_SNT *SNTSession) TotalSupply() (*big.Int, error) {
	return _SNT.Contract.TotalSupply(&_SNT.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_SNT *SNTCallerSession) TotalSupply() (*big.Int, error) {
	return _SNT.Contract.TotalSupply(&_SNT.CallOpts)
}

// TotalSupplyAt is a free data retrieval call binding the contract method 0x981b24d0.
//
// Solidity: function totalSupplyAt(uint256 _blockNumber) view returns(uint256)
func (_SNT *SNTCaller) TotalSupplyAt(opts *bind.CallOpts, _blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "totalSupplyAt", _blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupplyAt is a free data retrieval call binding the contract method 0x981b24d0.
//
// Solidity: function totalSupplyAt(uint256 _blockNumber) view returns(uint256)
func (_SNT *SNTSession) TotalSupplyAt(_blockNumber *big.Int) (*big.Int, error) {
	return _SNT.Contract.TotalSupplyAt(&_SNT.CallOpts, _blockNumber)
}

// TotalSupplyAt is a free data retrieval call binding the contract method 0x981b24d0.
//
// Solidity: function totalSupplyAt(uint256 _blockNumber) view returns(uint256)
func (_SNT *SNTCallerSession) TotalSupplyAt(_blockNumber *big.Int) (*big.Int, error) {
	return _SNT.Contract.TotalSupplyAt(&_SNT.CallOpts, _blockNumber)
}

// TransfersEnabled is a free data retrieval call binding the contract method 0xbef97c87.
//
// Solidity: function transfersEnabled() view returns(bool)
func (_SNT *SNTCaller) TransfersEnabled(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "transfersEnabled")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// TransfersEnabled is a free data retrieval call binding the contract method 0xbef97c87.
//
// Solidity: function transfersEnabled() view returns(bool)
func (_SNT *SNTSession) TransfersEnabled() (bool, error) {
	return _SNT.Contract.TransfersEnabled(&_SNT.CallOpts)
}

// TransfersEnabled is a free data retrieval call binding the contract method 0xbef97c87.
//
// Solidity: function transfersEnabled() view returns(bool)
func (_SNT *SNTCallerSession) TransfersEnabled() (bool, error) {
	return _SNT.Contract.TransfersEnabled(&_SNT.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SNT *SNTCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SNT.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SNT *SNTSession) Version() (string, error) {
	return _SNT.Contract.Version(&_SNT.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SNT *SNTCallerSession) Version() (string, error) {
	return _SNT.Contract.Version(&_SNT.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool success)
func (_SNT *SNTTransactor) Approve(opts *bind.TransactOpts, _spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "approve", _spender, _amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool success)
func (_SNT *SNTSession) Approve(_spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.Approve(&_SNT.TransactOpts, _spender, _amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _amount) returns(bool success)
func (_SNT *SNTTransactorSession) Approve(_spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.Approve(&_SNT.TransactOpts, _spender, _amount)
}

// ApproveAndCall is a paid mutator transaction binding the contract method 0xcae9ca51.
//
// Solidity: function approveAndCall(address _spender, uint256 _amount, bytes _extraData) returns(bool success)
func (_SNT *SNTTransactor) ApproveAndCall(opts *bind.TransactOpts, _spender common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "approveAndCall", _spender, _amount, _extraData)
}

// ApproveAndCall is a paid mutator transaction binding the contract method 0xcae9ca51.
//
// Solidity: function approveAndCall(address _spender, uint256 _amount, bytes _extraData) returns(bool success)
func (_SNT *SNTSession) ApproveAndCall(_spender common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _SNT.Contract.ApproveAndCall(&_SNT.TransactOpts, _spender, _amount, _extraData)
}

// ApproveAndCall is a paid mutator transaction binding the contract method 0xcae9ca51.
//
// Solidity: function approveAndCall(address _spender, uint256 _amount, bytes _extraData) returns(bool success)
func (_SNT *SNTTransactorSession) ApproveAndCall(_spender common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _SNT.Contract.ApproveAndCall(&_SNT.TransactOpts, _spender, _amount, _extraData)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_SNT *SNTTransactor) ChangeController(opts *bind.TransactOpts, _newController common.Address) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "changeController", _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_SNT *SNTSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _SNT.Contract.ChangeController(&_SNT.TransactOpts, _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_SNT *SNTTransactorSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _SNT.Contract.ChangeController(&_SNT.TransactOpts, _newController)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SNT *SNTTransactor) ClaimTokens(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "claimTokens", _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SNT *SNTSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _SNT.Contract.ClaimTokens(&_SNT.TransactOpts, _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SNT *SNTTransactorSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _SNT.Contract.ClaimTokens(&_SNT.TransactOpts, _token)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x6638c087.
//
// Solidity: function createCloneToken(string _cloneTokenName, uint8 _cloneDecimalUnits, string _cloneTokenSymbol, uint256 _snapshotBlock, bool _transfersEnabled) returns(address)
func (_SNT *SNTTransactor) CreateCloneToken(opts *bind.TransactOpts, _cloneTokenName string, _cloneDecimalUnits uint8, _cloneTokenSymbol string, _snapshotBlock *big.Int, _transfersEnabled bool) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "createCloneToken", _cloneTokenName, _cloneDecimalUnits, _cloneTokenSymbol, _snapshotBlock, _transfersEnabled)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x6638c087.
//
// Solidity: function createCloneToken(string _cloneTokenName, uint8 _cloneDecimalUnits, string _cloneTokenSymbol, uint256 _snapshotBlock, bool _transfersEnabled) returns(address)
func (_SNT *SNTSession) CreateCloneToken(_cloneTokenName string, _cloneDecimalUnits uint8, _cloneTokenSymbol string, _snapshotBlock *big.Int, _transfersEnabled bool) (*types.Transaction, error) {
	return _SNT.Contract.CreateCloneToken(&_SNT.TransactOpts, _cloneTokenName, _cloneDecimalUnits, _cloneTokenSymbol, _snapshotBlock, _transfersEnabled)
}

// CreateCloneToken is a paid mutator transaction binding the contract method 0x6638c087.
//
// Solidity: function createCloneToken(string _cloneTokenName, uint8 _cloneDecimalUnits, string _cloneTokenSymbol, uint256 _snapshotBlock, bool _transfersEnabled) returns(address)
func (_SNT *SNTTransactorSession) CreateCloneToken(_cloneTokenName string, _cloneDecimalUnits uint8, _cloneTokenSymbol string, _snapshotBlock *big.Int, _transfersEnabled bool) (*types.Transaction, error) {
	return _SNT.Contract.CreateCloneToken(&_SNT.TransactOpts, _cloneTokenName, _cloneDecimalUnits, _cloneTokenSymbol, _snapshotBlock, _transfersEnabled)
}

// DestroyTokens is a paid mutator transaction binding the contract method 0xd3ce77fe.
//
// Solidity: function destroyTokens(address _owner, uint256 _amount) returns(bool)
func (_SNT *SNTTransactor) DestroyTokens(opts *bind.TransactOpts, _owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "destroyTokens", _owner, _amount)
}

// DestroyTokens is a paid mutator transaction binding the contract method 0xd3ce77fe.
//
// Solidity: function destroyTokens(address _owner, uint256 _amount) returns(bool)
func (_SNT *SNTSession) DestroyTokens(_owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.DestroyTokens(&_SNT.TransactOpts, _owner, _amount)
}

// DestroyTokens is a paid mutator transaction binding the contract method 0xd3ce77fe.
//
// Solidity: function destroyTokens(address _owner, uint256 _amount) returns(bool)
func (_SNT *SNTTransactorSession) DestroyTokens(_owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.DestroyTokens(&_SNT.TransactOpts, _owner, _amount)
}

// EnableTransfers is a paid mutator transaction binding the contract method 0xf41e60c5.
//
// Solidity: function enableTransfers(bool _transfersEnabled) returns()
func (_SNT *SNTTransactor) EnableTransfers(opts *bind.TransactOpts, _transfersEnabled bool) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "enableTransfers", _transfersEnabled)
}

// EnableTransfers is a paid mutator transaction binding the contract method 0xf41e60c5.
//
// Solidity: function enableTransfers(bool _transfersEnabled) returns()
func (_SNT *SNTSession) EnableTransfers(_transfersEnabled bool) (*types.Transaction, error) {
	return _SNT.Contract.EnableTransfers(&_SNT.TransactOpts, _transfersEnabled)
}

// EnableTransfers is a paid mutator transaction binding the contract method 0xf41e60c5.
//
// Solidity: function enableTransfers(bool _transfersEnabled) returns()
func (_SNT *SNTTransactorSession) EnableTransfers(_transfersEnabled bool) (*types.Transaction, error) {
	return _SNT.Contract.EnableTransfers(&_SNT.TransactOpts, _transfersEnabled)
}

// GenerateTokens is a paid mutator transaction binding the contract method 0x827f32c0.
//
// Solidity: function generateTokens(address _owner, uint256 _amount) returns(bool)
func (_SNT *SNTTransactor) GenerateTokens(opts *bind.TransactOpts, _owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "generateTokens", _owner, _amount)
}

// GenerateTokens is a paid mutator transaction binding the contract method 0x827f32c0.
//
// Solidity: function generateTokens(address _owner, uint256 _amount) returns(bool)
func (_SNT *SNTSession) GenerateTokens(_owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.GenerateTokens(&_SNT.TransactOpts, _owner, _amount)
}

// GenerateTokens is a paid mutator transaction binding the contract method 0x827f32c0.
//
// Solidity: function generateTokens(address _owner, uint256 _amount) returns(bool)
func (_SNT *SNTTransactorSession) GenerateTokens(_owner common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.GenerateTokens(&_SNT.TransactOpts, _owner, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool success)
func (_SNT *SNTTransactor) Transfer(opts *bind.TransactOpts, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "transfer", _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool success)
func (_SNT *SNTSession) Transfer(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.Transfer(&_SNT.TransactOpts, _to, _amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _amount) returns(bool success)
func (_SNT *SNTTransactorSession) Transfer(_to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.Transfer(&_SNT.TransactOpts, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool success)
func (_SNT *SNTTransactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.contract.Transact(opts, "transferFrom", _from, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool success)
func (_SNT *SNTSession) TransferFrom(_from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.TransferFrom(&_SNT.TransactOpts, _from, _to, _amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _amount) returns(bool success)
func (_SNT *SNTTransactorSession) TransferFrom(_from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _SNT.Contract.TransferFrom(&_SNT.TransactOpts, _from, _to, _amount)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SNT *SNTTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _SNT.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SNT *SNTSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _SNT.Contract.Fallback(&_SNT.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SNT *SNTTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _SNT.Contract.Fallback(&_SNT.TransactOpts, calldata)
}

// SNTApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the SNT contract.
type SNTApprovalIterator struct {
	Event *SNTApproval // Event containing the contract specifics and raw log

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
func (it *SNTApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SNTApproval)
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
		it.Event = new(SNTApproval)
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
func (it *SNTApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SNTApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SNTApproval represents a Approval event raised by the SNT contract.
type SNTApproval struct {
	Owner   common.Address
	Spender common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _amount)
func (_SNT *SNTFilterer) FilterApproval(opts *bind.FilterOpts, _owner []common.Address, _spender []common.Address) (*SNTApprovalIterator, error) {

	var _ownerRule []interface{}
	for _, _ownerItem := range _owner {
		_ownerRule = append(_ownerRule, _ownerItem)
	}
	var _spenderRule []interface{}
	for _, _spenderItem := range _spender {
		_spenderRule = append(_spenderRule, _spenderItem)
	}

	logs, sub, err := _SNT.contract.FilterLogs(opts, "Approval", _ownerRule, _spenderRule)
	if err != nil {
		return nil, err
	}
	return &SNTApprovalIterator{contract: _SNT.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _amount)
func (_SNT *SNTFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *SNTApproval, _owner []common.Address, _spender []common.Address) (event.Subscription, error) {

	var _ownerRule []interface{}
	for _, _ownerItem := range _owner {
		_ownerRule = append(_ownerRule, _ownerItem)
	}
	var _spenderRule []interface{}
	for _, _spenderItem := range _spender {
		_spenderRule = append(_spenderRule, _spenderItem)
	}

	logs, sub, err := _SNT.contract.WatchLogs(opts, "Approval", _ownerRule, _spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SNTApproval)
				if err := _SNT.contract.UnpackLog(event, "Approval", log); err != nil {
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
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _amount)
func (_SNT *SNTFilterer) ParseApproval(log types.Log) (*SNTApproval, error) {
	event := new(SNTApproval)
	if err := _SNT.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SNTClaimedTokensIterator is returned from FilterClaimedTokens and is used to iterate over the raw logs and unpacked data for ClaimedTokens events raised by the SNT contract.
type SNTClaimedTokensIterator struct {
	Event *SNTClaimedTokens // Event containing the contract specifics and raw log

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
func (it *SNTClaimedTokensIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SNTClaimedTokens)
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
		it.Event = new(SNTClaimedTokens)
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
func (it *SNTClaimedTokensIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SNTClaimedTokensIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SNTClaimedTokens represents a ClaimedTokens event raised by the SNT contract.
type SNTClaimedTokens struct {
	Token      common.Address
	Controller common.Address
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterClaimedTokens is a free log retrieval operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SNT *SNTFilterer) FilterClaimedTokens(opts *bind.FilterOpts, _token []common.Address, _controller []common.Address) (*SNTClaimedTokensIterator, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _SNT.contract.FilterLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return &SNTClaimedTokensIterator{contract: _SNT.contract, event: "ClaimedTokens", logs: logs, sub: sub}, nil
}

// WatchClaimedTokens is a free log subscription operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SNT *SNTFilterer) WatchClaimedTokens(opts *bind.WatchOpts, sink chan<- *SNTClaimedTokens, _token []common.Address, _controller []common.Address) (event.Subscription, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _SNT.contract.WatchLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SNTClaimedTokens)
				if err := _SNT.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
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

// ParseClaimedTokens is a log parse operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SNT *SNTFilterer) ParseClaimedTokens(log types.Log) (*SNTClaimedTokens, error) {
	event := new(SNTClaimedTokens)
	if err := _SNT.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SNTNewCloneTokenIterator is returned from FilterNewCloneToken and is used to iterate over the raw logs and unpacked data for NewCloneToken events raised by the SNT contract.
type SNTNewCloneTokenIterator struct {
	Event *SNTNewCloneToken // Event containing the contract specifics and raw log

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
func (it *SNTNewCloneTokenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SNTNewCloneToken)
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
		it.Event = new(SNTNewCloneToken)
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
func (it *SNTNewCloneTokenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SNTNewCloneTokenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SNTNewCloneToken represents a NewCloneToken event raised by the SNT contract.
type SNTNewCloneToken struct {
	CloneToken    common.Address
	SnapshotBlock *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterNewCloneToken is a free log retrieval operation binding the contract event 0x086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade.
//
// Solidity: event NewCloneToken(address indexed _cloneToken, uint256 _snapshotBlock)
func (_SNT *SNTFilterer) FilterNewCloneToken(opts *bind.FilterOpts, _cloneToken []common.Address) (*SNTNewCloneTokenIterator, error) {

	var _cloneTokenRule []interface{}
	for _, _cloneTokenItem := range _cloneToken {
		_cloneTokenRule = append(_cloneTokenRule, _cloneTokenItem)
	}

	logs, sub, err := _SNT.contract.FilterLogs(opts, "NewCloneToken", _cloneTokenRule)
	if err != nil {
		return nil, err
	}
	return &SNTNewCloneTokenIterator{contract: _SNT.contract, event: "NewCloneToken", logs: logs, sub: sub}, nil
}

// WatchNewCloneToken is a free log subscription operation binding the contract event 0x086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade.
//
// Solidity: event NewCloneToken(address indexed _cloneToken, uint256 _snapshotBlock)
func (_SNT *SNTFilterer) WatchNewCloneToken(opts *bind.WatchOpts, sink chan<- *SNTNewCloneToken, _cloneToken []common.Address) (event.Subscription, error) {

	var _cloneTokenRule []interface{}
	for _, _cloneTokenItem := range _cloneToken {
		_cloneTokenRule = append(_cloneTokenRule, _cloneTokenItem)
	}

	logs, sub, err := _SNT.contract.WatchLogs(opts, "NewCloneToken", _cloneTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SNTNewCloneToken)
				if err := _SNT.contract.UnpackLog(event, "NewCloneToken", log); err != nil {
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

// ParseNewCloneToken is a log parse operation binding the contract event 0x086c875b377f900b07ce03575813022f05dd10ed7640b5282cf6d3c3fc352ade.
//
// Solidity: event NewCloneToken(address indexed _cloneToken, uint256 _snapshotBlock)
func (_SNT *SNTFilterer) ParseNewCloneToken(log types.Log) (*SNTNewCloneToken, error) {
	event := new(SNTNewCloneToken)
	if err := _SNT.contract.UnpackLog(event, "NewCloneToken", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SNTTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the SNT contract.
type SNTTransferIterator struct {
	Event *SNTTransfer // Event containing the contract specifics and raw log

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
func (it *SNTTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SNTTransfer)
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
		it.Event = new(SNTTransfer)
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
func (it *SNTTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SNTTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SNTTransfer represents a Transfer event raised by the SNT contract.
type SNTTransfer struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _amount)
func (_SNT *SNTFilterer) FilterTransfer(opts *bind.FilterOpts, _from []common.Address, _to []common.Address) (*SNTTransferIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _SNT.contract.FilterLogs(opts, "Transfer", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return &SNTTransferIterator{contract: _SNT.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _amount)
func (_SNT *SNTFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *SNTTransfer, _from []common.Address, _to []common.Address) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _SNT.contract.WatchLogs(opts, "Transfer", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SNTTransfer)
				if err := _SNT.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _amount)
func (_SNT *SNTFilterer) ParseTransfer(log types.Log) (*SNTTransfer, error) {
	event := new(SNTTransfer)
	if err := _SNT.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SNTPlaceHolderABI is the input ABI used to generate the binding from.
const SNTPlaceHolderABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"snt\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newController\",\"type\":\"address\"}],\"name\":\"changeController\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"onTransfer\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"contribution\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"changeOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"sgtExchanger\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"newOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"activationTime\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"onApprove\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"proxyPayment\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_snt\",\"type\":\"address\"},{\"name\":\"_contribution\",\"type\":\"address\"},{\"name\":\"_sgtExchanger\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_token\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_controller\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"ClaimedTokens\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_newController\",\"type\":\"address\"}],\"name\":\"ControllerChanged\",\"type\":\"event\"}]"

// SNTPlaceHolderFuncSigs maps the 4-byte function signature to its string representation.
var SNTPlaceHolderFuncSigs = map[string]string{
	"79ba5097": "acceptOwnership()",
	"da4493f6": "activationTime()",
	"3cebb823": "changeController(address)",
	"a6f9dae1": "changeOwner(address)",
	"df8de3e7": "claimTokens(address)",
	"50520b1f": "contribution()",
	"d4ee1d90": "newOwner()",
	"da682aeb": "onApprove(address,address,uint256)",
	"4a393149": "onTransfer(address,address,uint256)",
	"8da5cb5b": "owner()",
	"f48c3054": "proxyPayment(address)",
	"ad344bbe": "sgtExchanger()",
	"060eb520": "snt()",
}

// SNTPlaceHolderBin is the compiled bytecode used for deploying new contracts.
var SNTPlaceHolderBin = "0x608060405234801561001057600080fd5b506040516080806108de83398101604090815281516020830151918301516060909301516000805433600160a060020a0319918216178116600160a060020a03948516178255600280548216958516959095179094556003805485169584169590951790945560058054909316911617905561084c90819061009290396000f3006080604052600436106100c45763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663060eb52081146100c95780633cebb823146100fa5780634a3931491461011d57806350520b1f1461015b57806379ba5097146101705780638da5cb5b14610185578063a6f9dae11461019a578063ad344bbe146101bb578063d4ee1d90146101d0578063da4493f6146101e5578063da682aeb1461011d578063df8de3e71461020c578063f48c30541461022d575b600080fd5b3480156100d557600080fd5b506100de610241565b60408051600160a060020a039092168252519081900360200190f35b34801561010657600080fd5b5061011b600160a060020a0360043516610250565b005b34801561012957600080fd5b50610147600160a060020a036004358116906024351660443561031d565b604080519115158252519081900360200190f35b34801561016757600080fd5b506100de610330565b34801561017c57600080fd5b5061011b61033f565b34801561019157600080fd5b506100de610384565b3480156101a657600080fd5b5061011b600160a060020a0360043516610393565b3480156101c757600080fd5b506100de6103d9565b3480156101dc57600080fd5b506100de6103e8565b3480156101f157600080fd5b506101fa6103f7565b60408051918252519081900360200190f35b34801561021857600080fd5b5061011b600160a060020a03600435166103fd565b610147600160a060020a03600435166106fe565b600254600160a060020a031681565b600054600160a060020a0316331461026757600080fd5b600254604080517f3cebb823000000000000000000000000000000000000000000000000000000008152600160a060020a03848116600483015291519190921691633cebb82391602480830192600092919082900301818387803b1580156102ce57600080fd5b505af11580156102e2573d6000803e3d6000fd5b5050604051600160a060020a03841692507f027c3e080ed9215f564a9455a666f7e459b3edc0bb6e02a1bf842fde4d0ccfc19150600090a250565b600061032884610704565b949350505050565b600354600160a060020a031681565b600154600160a060020a0316331415610382576001546000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a039092169190911790555b565b600054600160a060020a031681565b600054600160a060020a031633146103aa57600080fd5b6001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600554600160a060020a031681565b600154600160a060020a031681565b60045481565b600080548190600160a060020a0316331461041757600080fd5b600254604080517ff77c479100000000000000000000000000000000000000000000000000000000815290513092600160a060020a03169163f77c47919160048083019260209291908290030181600087803b15801561047657600080fd5b505af115801561048a573d6000803e3d6000fd5b505050506040513d60208110156104a057600080fd5b5051600160a060020a0316141561053157600254604080517fdf8de3e7000000000000000000000000000000000000000000000000000000008152600160a060020a0386811660048301529151919092169163df8de3e791602480830192600092919082900301818387803b15801561051857600080fd5b505af115801561052c573d6000803e3d6000fd5b505050505b600160a060020a03831615156105825760008054604051600160a060020a0390911691303180156108fc02929091818181858888f1935050505015801561057c573d6000803e3d6000fd5b506106f9565b604080517f70a082310000000000000000000000000000000000000000000000000000000081523060048201529051849350600160a060020a038416916370a082319160248083019260209291908290030181600087803b1580156105e657600080fd5b505af11580156105fa573d6000803e3d6000fd5b505050506040513d602081101561061057600080fd5b505160008054604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a0392831660048201526024810185905290519394509085169263a9059cbb92604480840193602093929083900390910190829087803b15801561068657600080fd5b505af115801561069a573d6000803e3d6000fd5b505050506040513d60208110156106b057600080fd5b5050600054604080518381529051600160a060020a03928316928616917ff931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c919081900360200190a35b505050565b50600090565b600080600454600014156107d957600360009054906101000a9004600160a060020a0316600160a060020a031663fe67a1896040518163ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401602060405180830381600087803b15801561077e57600080fd5b505af1158015610792573d6000803e3d6000fd5b505050506040513d60208110156107a857600080fd5b5051905060008111156107d0576107c88162093a8063ffffffff61080616565b6004556107d9565b60009150610800565b6004546107e461081c565b11806107fd5750600554600160a060020a038481169116145b91505b50919050565b60008282018381101561081557fe5b9392505050565b42905600a165627a7a72305820ffe0a81a211756c6d5f34b7b11be5d2b3f72b2810f1f5a80a76dcff4a3a4df9c0029"

// DeploySNTPlaceHolder deploys a new Ethereum contract, binding an instance of SNTPlaceHolder to it.
func DeploySNTPlaceHolder(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address, _snt common.Address, _contribution common.Address, _sgtExchanger common.Address) (common.Address, *types.Transaction, *SNTPlaceHolder, error) {
	parsed, err := abi.JSON(strings.NewReader(SNTPlaceHolderABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SNTPlaceHolderBin), backend, _owner, _snt, _contribution, _sgtExchanger)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SNTPlaceHolder{SNTPlaceHolderCaller: SNTPlaceHolderCaller{contract: contract}, SNTPlaceHolderTransactor: SNTPlaceHolderTransactor{contract: contract}, SNTPlaceHolderFilterer: SNTPlaceHolderFilterer{contract: contract}}, nil
}

// SNTPlaceHolder is an auto generated Go binding around an Ethereum contract.
type SNTPlaceHolder struct {
	SNTPlaceHolderCaller     // Read-only binding to the contract
	SNTPlaceHolderTransactor // Write-only binding to the contract
	SNTPlaceHolderFilterer   // Log filterer for contract events
}

// SNTPlaceHolderCaller is an auto generated read-only Go binding around an Ethereum contract.
type SNTPlaceHolderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SNTPlaceHolderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SNTPlaceHolderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SNTPlaceHolderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SNTPlaceHolderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SNTPlaceHolderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SNTPlaceHolderSession struct {
	Contract     *SNTPlaceHolder   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SNTPlaceHolderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SNTPlaceHolderCallerSession struct {
	Contract *SNTPlaceHolderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// SNTPlaceHolderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SNTPlaceHolderTransactorSession struct {
	Contract     *SNTPlaceHolderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// SNTPlaceHolderRaw is an auto generated low-level Go binding around an Ethereum contract.
type SNTPlaceHolderRaw struct {
	Contract *SNTPlaceHolder // Generic contract binding to access the raw methods on
}

// SNTPlaceHolderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SNTPlaceHolderCallerRaw struct {
	Contract *SNTPlaceHolderCaller // Generic read-only contract binding to access the raw methods on
}

// SNTPlaceHolderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SNTPlaceHolderTransactorRaw struct {
	Contract *SNTPlaceHolderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSNTPlaceHolder creates a new instance of SNTPlaceHolder, bound to a specific deployed contract.
func NewSNTPlaceHolder(address common.Address, backend bind.ContractBackend) (*SNTPlaceHolder, error) {
	contract, err := bindSNTPlaceHolder(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SNTPlaceHolder{SNTPlaceHolderCaller: SNTPlaceHolderCaller{contract: contract}, SNTPlaceHolderTransactor: SNTPlaceHolderTransactor{contract: contract}, SNTPlaceHolderFilterer: SNTPlaceHolderFilterer{contract: contract}}, nil
}

// NewSNTPlaceHolderCaller creates a new read-only instance of SNTPlaceHolder, bound to a specific deployed contract.
func NewSNTPlaceHolderCaller(address common.Address, caller bind.ContractCaller) (*SNTPlaceHolderCaller, error) {
	contract, err := bindSNTPlaceHolder(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SNTPlaceHolderCaller{contract: contract}, nil
}

// NewSNTPlaceHolderTransactor creates a new write-only instance of SNTPlaceHolder, bound to a specific deployed contract.
func NewSNTPlaceHolderTransactor(address common.Address, transactor bind.ContractTransactor) (*SNTPlaceHolderTransactor, error) {
	contract, err := bindSNTPlaceHolder(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SNTPlaceHolderTransactor{contract: contract}, nil
}

// NewSNTPlaceHolderFilterer creates a new log filterer instance of SNTPlaceHolder, bound to a specific deployed contract.
func NewSNTPlaceHolderFilterer(address common.Address, filterer bind.ContractFilterer) (*SNTPlaceHolderFilterer, error) {
	contract, err := bindSNTPlaceHolder(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SNTPlaceHolderFilterer{contract: contract}, nil
}

// bindSNTPlaceHolder binds a generic wrapper to an already deployed contract.
func bindSNTPlaceHolder(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SNTPlaceHolderABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SNTPlaceHolder *SNTPlaceHolderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SNTPlaceHolder.Contract.SNTPlaceHolderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SNTPlaceHolder *SNTPlaceHolderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.SNTPlaceHolderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SNTPlaceHolder *SNTPlaceHolderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.SNTPlaceHolderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SNTPlaceHolder *SNTPlaceHolderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SNTPlaceHolder.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SNTPlaceHolder *SNTPlaceHolderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SNTPlaceHolder *SNTPlaceHolderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.contract.Transact(opts, method, params...)
}

// ActivationTime is a free data retrieval call binding the contract method 0xda4493f6.
//
// Solidity: function activationTime() view returns(uint256)
func (_SNTPlaceHolder *SNTPlaceHolderCaller) ActivationTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SNTPlaceHolder.contract.Call(opts, &out, "activationTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ActivationTime is a free data retrieval call binding the contract method 0xda4493f6.
//
// Solidity: function activationTime() view returns(uint256)
func (_SNTPlaceHolder *SNTPlaceHolderSession) ActivationTime() (*big.Int, error) {
	return _SNTPlaceHolder.Contract.ActivationTime(&_SNTPlaceHolder.CallOpts)
}

// ActivationTime is a free data retrieval call binding the contract method 0xda4493f6.
//
// Solidity: function activationTime() view returns(uint256)
func (_SNTPlaceHolder *SNTPlaceHolderCallerSession) ActivationTime() (*big.Int, error) {
	return _SNTPlaceHolder.Contract.ActivationTime(&_SNTPlaceHolder.CallOpts)
}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCaller) Contribution(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SNTPlaceHolder.contract.Call(opts, &out, "contribution")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderSession) Contribution() (common.Address, error) {
	return _SNTPlaceHolder.Contract.Contribution(&_SNTPlaceHolder.CallOpts)
}

// Contribution is a free data retrieval call binding the contract method 0x50520b1f.
//
// Solidity: function contribution() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCallerSession) Contribution() (common.Address, error) {
	return _SNTPlaceHolder.Contract.Contribution(&_SNTPlaceHolder.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCaller) NewOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SNTPlaceHolder.contract.Call(opts, &out, "newOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderSession) NewOwner() (common.Address, error) {
	return _SNTPlaceHolder.Contract.NewOwner(&_SNTPlaceHolder.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCallerSession) NewOwner() (common.Address, error) {
	return _SNTPlaceHolder.Contract.NewOwner(&_SNTPlaceHolder.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SNTPlaceHolder.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderSession) Owner() (common.Address, error) {
	return _SNTPlaceHolder.Contract.Owner(&_SNTPlaceHolder.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCallerSession) Owner() (common.Address, error) {
	return _SNTPlaceHolder.Contract.Owner(&_SNTPlaceHolder.CallOpts)
}

// SgtExchanger is a free data retrieval call binding the contract method 0xad344bbe.
//
// Solidity: function sgtExchanger() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCaller) SgtExchanger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SNTPlaceHolder.contract.Call(opts, &out, "sgtExchanger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SgtExchanger is a free data retrieval call binding the contract method 0xad344bbe.
//
// Solidity: function sgtExchanger() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderSession) SgtExchanger() (common.Address, error) {
	return _SNTPlaceHolder.Contract.SgtExchanger(&_SNTPlaceHolder.CallOpts)
}

// SgtExchanger is a free data retrieval call binding the contract method 0xad344bbe.
//
// Solidity: function sgtExchanger() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCallerSession) SgtExchanger() (common.Address, error) {
	return _SNTPlaceHolder.Contract.SgtExchanger(&_SNTPlaceHolder.CallOpts)
}

// Snt is a free data retrieval call binding the contract method 0x060eb520.
//
// Solidity: function snt() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCaller) Snt(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SNTPlaceHolder.contract.Call(opts, &out, "snt")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Snt is a free data retrieval call binding the contract method 0x060eb520.
//
// Solidity: function snt() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderSession) Snt() (common.Address, error) {
	return _SNTPlaceHolder.Contract.Snt(&_SNTPlaceHolder.CallOpts)
}

// Snt is a free data retrieval call binding the contract method 0x060eb520.
//
// Solidity: function snt() view returns(address)
func (_SNTPlaceHolder *SNTPlaceHolderCallerSession) Snt() (common.Address, error) {
	return _SNTPlaceHolder.Contract.Snt(&_SNTPlaceHolder.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SNTPlaceHolder *SNTPlaceHolderTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SNTPlaceHolder.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SNTPlaceHolder *SNTPlaceHolderSession) AcceptOwnership() (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.AcceptOwnership(&_SNTPlaceHolder.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_SNTPlaceHolder *SNTPlaceHolderTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.AcceptOwnership(&_SNTPlaceHolder.TransactOpts)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_SNTPlaceHolder *SNTPlaceHolderTransactor) ChangeController(opts *bind.TransactOpts, _newController common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.contract.Transact(opts, "changeController", _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_SNTPlaceHolder *SNTPlaceHolderSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.ChangeController(&_SNTPlaceHolder.TransactOpts, _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(address _newController) returns()
func (_SNTPlaceHolder *SNTPlaceHolderTransactorSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.ChangeController(&_SNTPlaceHolder.TransactOpts, _newController)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_SNTPlaceHolder *SNTPlaceHolderTransactor) ChangeOwner(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.contract.Transact(opts, "changeOwner", _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_SNTPlaceHolder *SNTPlaceHolderSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.ChangeOwner(&_SNTPlaceHolder.TransactOpts, _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_SNTPlaceHolder *SNTPlaceHolderTransactorSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.ChangeOwner(&_SNTPlaceHolder.TransactOpts, _newOwner)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SNTPlaceHolder *SNTPlaceHolderTransactor) ClaimTokens(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.contract.Transact(opts, "claimTokens", _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SNTPlaceHolder *SNTPlaceHolderSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.ClaimTokens(&_SNTPlaceHolder.TransactOpts, _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_SNTPlaceHolder *SNTPlaceHolderTransactorSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.ClaimTokens(&_SNTPlaceHolder.TransactOpts, _token)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address _from, address , uint256 ) returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderTransactor) OnApprove(opts *bind.TransactOpts, _from common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SNTPlaceHolder.contract.Transact(opts, "onApprove", _from, arg1, arg2)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address _from, address , uint256 ) returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderSession) OnApprove(_from common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.OnApprove(&_SNTPlaceHolder.TransactOpts, _from, arg1, arg2)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address _from, address , uint256 ) returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderTransactorSession) OnApprove(_from common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.OnApprove(&_SNTPlaceHolder.TransactOpts, _from, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address _from, address , uint256 ) returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderTransactor) OnTransfer(opts *bind.TransactOpts, _from common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SNTPlaceHolder.contract.Transact(opts, "onTransfer", _from, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address _from, address , uint256 ) returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderSession) OnTransfer(_from common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.OnTransfer(&_SNTPlaceHolder.TransactOpts, _from, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address _from, address , uint256 ) returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderTransactorSession) OnTransfer(_from common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.OnTransfer(&_SNTPlaceHolder.TransactOpts, _from, arg1, arg2)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address ) payable returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderTransactor) ProxyPayment(opts *bind.TransactOpts, arg0 common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.contract.Transact(opts, "proxyPayment", arg0)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address ) payable returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderSession) ProxyPayment(arg0 common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.ProxyPayment(&_SNTPlaceHolder.TransactOpts, arg0)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address ) payable returns(bool)
func (_SNTPlaceHolder *SNTPlaceHolderTransactorSession) ProxyPayment(arg0 common.Address) (*types.Transaction, error) {
	return _SNTPlaceHolder.Contract.ProxyPayment(&_SNTPlaceHolder.TransactOpts, arg0)
}

// SNTPlaceHolderClaimedTokensIterator is returned from FilterClaimedTokens and is used to iterate over the raw logs and unpacked data for ClaimedTokens events raised by the SNTPlaceHolder contract.
type SNTPlaceHolderClaimedTokensIterator struct {
	Event *SNTPlaceHolderClaimedTokens // Event containing the contract specifics and raw log

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
func (it *SNTPlaceHolderClaimedTokensIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SNTPlaceHolderClaimedTokens)
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
		it.Event = new(SNTPlaceHolderClaimedTokens)
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
func (it *SNTPlaceHolderClaimedTokensIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SNTPlaceHolderClaimedTokensIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SNTPlaceHolderClaimedTokens represents a ClaimedTokens event raised by the SNTPlaceHolder contract.
type SNTPlaceHolderClaimedTokens struct {
	Token      common.Address
	Controller common.Address
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterClaimedTokens is a free log retrieval operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SNTPlaceHolder *SNTPlaceHolderFilterer) FilterClaimedTokens(opts *bind.FilterOpts, _token []common.Address, _controller []common.Address) (*SNTPlaceHolderClaimedTokensIterator, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _SNTPlaceHolder.contract.FilterLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return &SNTPlaceHolderClaimedTokensIterator{contract: _SNTPlaceHolder.contract, event: "ClaimedTokens", logs: logs, sub: sub}, nil
}

// WatchClaimedTokens is a free log subscription operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SNTPlaceHolder *SNTPlaceHolderFilterer) WatchClaimedTokens(opts *bind.WatchOpts, sink chan<- *SNTPlaceHolderClaimedTokens, _token []common.Address, _controller []common.Address) (event.Subscription, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _SNTPlaceHolder.contract.WatchLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SNTPlaceHolderClaimedTokens)
				if err := _SNTPlaceHolder.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
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

// ParseClaimedTokens is a log parse operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_SNTPlaceHolder *SNTPlaceHolderFilterer) ParseClaimedTokens(log types.Log) (*SNTPlaceHolderClaimedTokens, error) {
	event := new(SNTPlaceHolderClaimedTokens)
	if err := _SNTPlaceHolder.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SNTPlaceHolderControllerChangedIterator is returned from FilterControllerChanged and is used to iterate over the raw logs and unpacked data for ControllerChanged events raised by the SNTPlaceHolder contract.
type SNTPlaceHolderControllerChangedIterator struct {
	Event *SNTPlaceHolderControllerChanged // Event containing the contract specifics and raw log

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
func (it *SNTPlaceHolderControllerChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SNTPlaceHolderControllerChanged)
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
		it.Event = new(SNTPlaceHolderControllerChanged)
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
func (it *SNTPlaceHolderControllerChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SNTPlaceHolderControllerChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SNTPlaceHolderControllerChanged represents a ControllerChanged event raised by the SNTPlaceHolder contract.
type SNTPlaceHolderControllerChanged struct {
	NewController common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterControllerChanged is a free log retrieval operation binding the contract event 0x027c3e080ed9215f564a9455a666f7e459b3edc0bb6e02a1bf842fde4d0ccfc1.
//
// Solidity: event ControllerChanged(address indexed _newController)
func (_SNTPlaceHolder *SNTPlaceHolderFilterer) FilterControllerChanged(opts *bind.FilterOpts, _newController []common.Address) (*SNTPlaceHolderControllerChangedIterator, error) {

	var _newControllerRule []interface{}
	for _, _newControllerItem := range _newController {
		_newControllerRule = append(_newControllerRule, _newControllerItem)
	}

	logs, sub, err := _SNTPlaceHolder.contract.FilterLogs(opts, "ControllerChanged", _newControllerRule)
	if err != nil {
		return nil, err
	}
	return &SNTPlaceHolderControllerChangedIterator{contract: _SNTPlaceHolder.contract, event: "ControllerChanged", logs: logs, sub: sub}, nil
}

// WatchControllerChanged is a free log subscription operation binding the contract event 0x027c3e080ed9215f564a9455a666f7e459b3edc0bb6e02a1bf842fde4d0ccfc1.
//
// Solidity: event ControllerChanged(address indexed _newController)
func (_SNTPlaceHolder *SNTPlaceHolderFilterer) WatchControllerChanged(opts *bind.WatchOpts, sink chan<- *SNTPlaceHolderControllerChanged, _newController []common.Address) (event.Subscription, error) {

	var _newControllerRule []interface{}
	for _, _newControllerItem := range _newController {
		_newControllerRule = append(_newControllerRule, _newControllerItem)
	}

	logs, sub, err := _SNTPlaceHolder.contract.WatchLogs(opts, "ControllerChanged", _newControllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SNTPlaceHolderControllerChanged)
				if err := _SNTPlaceHolder.contract.UnpackLog(event, "ControllerChanged", log); err != nil {
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

// ParseControllerChanged is a log parse operation binding the contract event 0x027c3e080ed9215f564a9455a666f7e459b3edc0bb6e02a1bf842fde4d0ccfc1.
//
// Solidity: event ControllerChanged(address indexed _newController)
func (_SNTPlaceHolder *SNTPlaceHolderFilterer) ParseControllerChanged(log types.Log) (*SNTPlaceHolderControllerChanged, error) {
	event := new(SNTPlaceHolderControllerChanged)
	if err := _SNTPlaceHolder.contract.UnpackLog(event, "ControllerChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SafeMathABI is the input ABI used to generate the binding from.
const SafeMathABI = "[]"

// SafeMathBin is the compiled bytecode used for deploying new contracts.
var SafeMathBin = "0x604c602c600b82828239805160001a60731460008114601c57601e565bfe5b5030600052607381538281f30073000000000000000000000000000000000000000030146080604052600080fd00a165627a7a723058202cc39b15e17bb02f1da11ef40b9afbebddfe2f48f4536125455736e5e0e5ca5c0029"

// DeploySafeMath deploys a new Ethereum contract, binding an instance of SafeMath to it.
func DeploySafeMath(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SafeMath, error) {
	parsed, err := abi.JSON(strings.NewReader(SafeMathABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SafeMathBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SafeMath{SafeMathCaller: SafeMathCaller{contract: contract}, SafeMathTransactor: SafeMathTransactor{contract: contract}, SafeMathFilterer: SafeMathFilterer{contract: contract}}, nil
}

// SafeMath is an auto generated Go binding around an Ethereum contract.
type SafeMath struct {
	SafeMathCaller     // Read-only binding to the contract
	SafeMathTransactor // Write-only binding to the contract
	SafeMathFilterer   // Log filterer for contract events
}

// SafeMathCaller is an auto generated read-only Go binding around an Ethereum contract.
type SafeMathCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SafeMathTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SafeMathFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeMathSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SafeMathSession struct {
	Contract     *SafeMath         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SafeMathCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SafeMathCallerSession struct {
	Contract *SafeMathCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// SafeMathTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SafeMathTransactorSession struct {
	Contract     *SafeMathTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// SafeMathRaw is an auto generated low-level Go binding around an Ethereum contract.
type SafeMathRaw struct {
	Contract *SafeMath // Generic contract binding to access the raw methods on
}

// SafeMathCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SafeMathCallerRaw struct {
	Contract *SafeMathCaller // Generic read-only contract binding to access the raw methods on
}

// SafeMathTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SafeMathTransactorRaw struct {
	Contract *SafeMathTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSafeMath creates a new instance of SafeMath, bound to a specific deployed contract.
func NewSafeMath(address common.Address, backend bind.ContractBackend) (*SafeMath, error) {
	contract, err := bindSafeMath(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SafeMath{SafeMathCaller: SafeMathCaller{contract: contract}, SafeMathTransactor: SafeMathTransactor{contract: contract}, SafeMathFilterer: SafeMathFilterer{contract: contract}}, nil
}

// NewSafeMathCaller creates a new read-only instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathCaller(address common.Address, caller bind.ContractCaller) (*SafeMathCaller, error) {
	contract, err := bindSafeMath(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SafeMathCaller{contract: contract}, nil
}

// NewSafeMathTransactor creates a new write-only instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathTransactor(address common.Address, transactor bind.ContractTransactor) (*SafeMathTransactor, error) {
	contract, err := bindSafeMath(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SafeMathTransactor{contract: contract}, nil
}

// NewSafeMathFilterer creates a new log filterer instance of SafeMath, bound to a specific deployed contract.
func NewSafeMathFilterer(address common.Address, filterer bind.ContractFilterer) (*SafeMathFilterer, error) {
	contract, err := bindSafeMath(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SafeMathFilterer{contract: contract}, nil
}

// bindSafeMath binds a generic wrapper to an already deployed contract.
func bindSafeMath(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SafeMathABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeMath *SafeMathRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SafeMath.Contract.SafeMathCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeMath *SafeMathRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeMath.Contract.SafeMathTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeMath *SafeMathRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeMath.Contract.SafeMathTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeMath *SafeMathCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SafeMath.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeMath *SafeMathTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeMath.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeMath *SafeMathTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeMath.Contract.contract.Transact(opts, method, params...)
}

// StatusContributionABI is the input ABI used to generate the binding from.
const StatusContributionABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"destEthDevs\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"endBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maxSGTSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalGuaranteedCollected\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalNormalCollected\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maxCallFrequency\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"sntController\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"failSafeLimit\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"exchangeRate\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maxGasPrice\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"finalizedBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"startBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"onTransfer\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"pauseContribution\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"finalize\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maxGuaranteedLimit\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"lastCallBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"dynamicCeiling\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"destTokensDevs\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"guaranteedBuyersBought\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"tokensIssued\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_snt\",\"type\":\"address\"},{\"name\":\"_sntController\",\"type\":\"address\"},{\"name\":\"_startBlock\",\"type\":\"uint256\"},{\"name\":\"_endBlock\",\"type\":\"uint256\"},{\"name\":\"_dynamicCeiling\",\"type\":\"address\"},{\"name\":\"_destEthDevs\",\"type\":\"address\"},{\"name\":\"_destTokensReserve\",\"type\":\"address\"},{\"name\":\"_destTokensSgt\",\"type\":\"address\"},{\"name\":\"_destTokensDevs\",\"type\":\"address\"},{\"name\":\"_sgt\",\"type\":\"address\"},{\"name\":\"_maxSGTSupply\",\"type\":\"uint256\"}],\"name\":\"initialize\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"SGT\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"guaranteedBuyersLimit\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"destTokensSgt\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"changeOwner\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"resumeContribution\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"SNT\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_th\",\"type\":\"address\"},{\"name\":\"_limit\",\"type\":\"uint256\"}],\"name\":\"setGuaranteedAddress\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"newOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"destTokensReserve\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"onApprove\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"claimTokens\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalCollected\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_th\",\"type\":\"address\"}],\"name\":\"proxyPayment\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"finalizedTime\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_token\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_controller\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"ClaimedTokens\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_th\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"_tokens\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"_guaranteed\",\"type\":\"bool\"}],\"name\":\"NewSale\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_th\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_limit\",\"type\":\"uint256\"}],\"name\":\"GuaranteedAddress\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Finalized\",\"type\":\"event\"}]"

// StatusContributionFuncSigs maps the 4-byte function signature to its string representation.
var StatusContributionFuncSigs = map[string]string{
	"9321cb7d": "SGT()",
	"c55a02a0": "SNT()",
	"79ba5097": "acceptOwnership()",
	"a6f9dae1": "changeOwner(address)",
	"df8de3e7": "claimTokens(address)",
	"01621527": "destEthDevs()",
	"5bd475fd": "destTokensDevs()",
	"d7e07d5f": "destTokensReserve()",
	"a1a7405a": "destTokensSgt()",
	"5a3c8826": "dynamicCeiling()",
	"083c6323": "endBlock()",
	"3ba0b9a9": "exchangeRate()",
	"23dc1314": "failSafeLimit()",
	"4bb278f3": "finalize()",
	"4084c3ab": "finalizedBlock()",
	"fe67a189": "finalizedTime()",
	"73aad472": "guaranteedBuyersBought(address)",
	"9752bcd3": "guaranteedBuyersLimit(address)",
	"8733f360": "initialize(address,address,uint256,uint256,address,address,address,address,address,address,uint256)",
	"548e0846": "lastCallBlock(address)",
	"17183ca3": "maxCallFrequency()",
	"3de39c11": "maxGasPrice()",
	"4d1ed74b": "maxGuaranteedLimit()",
	"092c506e": "maxSGTSupply()",
	"d4ee1d90": "newOwner()",
	"da682aeb": "onApprove(address,address,uint256)",
	"4a393149": "onTransfer(address,address,uint256)",
	"8da5cb5b": "owner()",
	"4b8adcf7": "pauseContribution()",
	"5c975abb": "paused()",
	"f48c3054": "proxyPayment(address)",
	"b681f9f6": "resumeContribution()",
	"cc9b7826": "setGuaranteedAddress(address,uint256)",
	"1b2f1109": "sntController()",
	"48cd4cb1": "startBlock()",
	"7c48bbda": "tokensIssued()",
	"e29eb836": "totalCollected()",
	"137935d5": "totalGuaranteedCollected()",
	"1517d107": "totalNormalCollected()",
}

// StatusContributionBin is the compiled bytecode used for deploying new contracts.
var StatusContributionBin = "0x608060405234801561001057600080fd5b5060008054600160a060020a031916331790556014805460ff19169055611d5e8061003c6000396000f3006080604052600436106101c95763ffffffff60e060020a6000350416630162152781146101e5578063083c632314610216578063092c506e1461023d578063137935d5146102525780631517d1071461026757806317183ca31461027c5780631b2f11091461029157806323dc1314146102a65780633ba0b9a9146102bb5780633de39c11146102d05780634084c3ab146102e557806348cd4cb1146102fa5780634a3931491461030f5780634b8adcf71461034d5780634bb278f3146103645780634d1ed74b14610379578063548e08461461038e5780635a3c8826146103af5780635bd475fd146103c45780635c975abb146103d957806373aad472146103ee57806379ba50971461040f5780637c48bbda146104245780638733f360146104395780638da5cb5b146104925780639321cb7d146104a75780639752bcd3146104bc578063a1a7405a146104dd578063a6f9dae1146104f2578063b681f9f614610513578063c55a02a014610528578063cc9b78261461053d578063d4ee1d9014610561578063d7e07d5f14610576578063da682aeb1461030f578063df8de3e71461058b578063e29eb836146105ac578063f48c3054146105c1578063fe67a189146105d5575b60145460ff16156101d957600080fd5b6101e2336105ea565b50005b3480156101f157600080fd5b506101fa6106b6565b60408051600160a060020a039092168252519081900360200190f35b34801561022257600080fd5b5061022b6106c5565b60408051918252519081900360200190f35b34801561024957600080fd5b5061022b6106cb565b34801561025e57600080fd5b5061022b6106d1565b34801561027357600080fd5b5061022b6106d7565b34801561028857600080fd5b5061022b6106dd565b34801561029d57600080fd5b506101fa6106e2565b3480156102b257600080fd5b5061022b6106f1565b3480156102c757600080fd5b5061022b6106ff565b3480156102dc57600080fd5b5061022b610705565b3480156102f157600080fd5b5061022b61070e565b34801561030657600080fd5b5061022b610714565b34801561031b57600080fd5b50610339600160a060020a036004358116906024351660443561071a565b604080519115158252519081900360200190f35b34801561035957600080fd5b50610362610723565b005b34801561037057600080fd5b50610362610749565b34801561038557600080fd5b5061022b610e95565b34801561039a57600080fd5b5061022b600160a060020a0360043516610ea3565b3480156103bb57600080fd5b506101fa610eb5565b3480156103d057600080fd5b506101fa610ec4565b3480156103e557600080fd5b50610339610ed3565b3480156103fa57600080fd5b5061022b600160a060020a0360043516610edc565b34801561041b57600080fd5b50610362610eee565b34801561043057600080fd5b5061022b610f26565b34801561044557600080fd5b50610362600160a060020a03600435811690602435811690604435906064359060843581169060a43581169060c43581169060e43581169061010435811690610124351661014435610fa0565b34801561049e57600080fd5b506101fa611388565b3480156104b357600080fd5b506101fa611397565b3480156104c857600080fd5b5061022b600160a060020a03600435166113a6565b3480156104e957600080fd5b506101fa6113b8565b3480156104fe57600080fd5b50610362600160a060020a03600435166113c7565b34801561051f57600080fd5b50610362611400565b34801561053457600080fd5b506101fa611423565b34801561054957600080fd5b50610362600160a060020a0360043516602435611432565b34801561056d57600080fd5b506101fa6114ee565b34801561058257600080fd5b506101fa6114fd565b34801561059757600080fd5b50610362600160a060020a036004351661150c565b3480156105b857600080fd5b5061022b61180d565b610339600160a060020a03600435166105ea565b3480156105e157600080fd5b5061022b61182b565b60145460009060ff16156105fd57600080fd5b600354600160a060020a0316151561061457600080fd5b60045461061f611831565b101580156106365750600554610633611831565b11155b80156106425750601154155b80156106585750600354600160a060020a031615155b151561066357600080fd5b600160a060020a038216151561067857600080fd5b600160a060020a0382166000908152600d602052604081205411156106a5576106a082611835565b6106ae565b6106ae8261190c565b506001919050565b600654600160a060020a031681565b60055481565b60095481565b600f5481565b60105481565b606481565b600c54600160a060020a031681565b693f870857a3e0e380000081565b61271081565b640ba43b740081565b60115481565b60045481565b60009392505050565b600054600160a060020a0316331461073a57600080fd5b6014805460ff19166001179055565b60035460009081908190819081908190600160a060020a0316151561076d57600080fd5b600454610778611831565b101561078357600080fd5b600054600160a060020a03163314806107a457506005546107a2611831565b115b15156107af57600080fd5b601154156107bc57600080fd5b600b60009054906101000a9004600160a060020a0316600160a060020a0316634b28bdc26040518163ffffffff1660e060020a028152600401602060405180830381600087803b15801561080f57600080fd5b505af1158015610823573d6000803e3d6000fd5b505050506040513d602081101561083957600080fd5b5051151561084657600080fd5b600554610851611831565b1161097757600b54604080517f6e4e5c1d0000000000000000000000000000000000000000000000000000000081529051600160a060020a0390921691631bf7d749916108fd916001918591636e4e5c1d916004808201926020929091908290030181600087803b1580156108c557600080fd5b505af11580156108d9573d6000803e3d6000fd5b505050506040513d60208110156108ef57600080fd5b50519063ffffffff611a7216565b6040518263ffffffff1660e060020a02815260040180828152602001915050608060405180830381600087803b15801561093657600080fd5b505af115801561094a573d6000803e3d6000fd5b505050506040513d608081101561096057600080fd5b506020015160105490965086111561097757600080fd5b61097f611831565b601155426012556009546002546040805160e060020a6318160ddd0281529051600160a060020a03909216916318160ddd916004808201926020929091908290030181600087803b1580156109d357600080fd5b505af11580156109e7573d6000803e3d6000fd5b505050506040513d60208110156109fd57600080fd5b505110610a1557610a0e600a611a84565b9450610ac2565b610abf600954610ab3600260009054906101000a9004600160a060020a0316600160a060020a03166318160ddd6040518163ffffffff1660e060020a028152600401602060405180830381600087803b158015610a7157600080fd5b505af1158015610a85573d6000803e3d6000fd5b505050506040513d6020811015610a9b57600080fd5b5051610aa7600a611a84565b9063ffffffff611aa316565b9063ffffffff611ace16565b94505b610acc6014611a84565b9350610b01610aeb86610adf600a611a84565b9063ffffffff611a7216565b610af56029611a84565b9063ffffffff611ae516565b9250610b0d601d611a84565b9150610bab83610ab3610b206064611a84565b600360009054906101000a9004600160a060020a0316600160a060020a03166318160ddd6040518163ffffffff1660e060020a028152600401602060405180830381600087803b158015610b7357600080fd5b505af1158015610b87573d6000803e3d6000fd5b505050506040513d6020811015610b9d57600080fd5b50519063ffffffff611aa316565b600354600854919250600160a060020a039081169163827f32c09116610be4610bd46064611a84565b610ab3868863ffffffff611aa316565b6040518363ffffffff1660e060020a0281526004018083600160a060020a0316600160a060020a0316815260200182815260200192505050602060405180830381600087803b158015610c3657600080fd5b505af1158015610c4a573d6000803e3d6000fd5b505050506040513d6020811015610c6057600080fd5b50511515610c6a57fe5b600354600a54600160a060020a039182169163827f32c09116610ca0610c906064611a84565b610ab3868b63ffffffff611aa316565b6040518363ffffffff1660e060020a0281526004018083600160a060020a0316600160a060020a0316815260200182815260200192505050602060405180830381600087803b158015610cf257600080fd5b505af1158015610d06573d6000803e3d6000fd5b505050506040513d6020811015610d1c57600080fd5b50511515610d2657fe5b600354600754600160a060020a039182169163827f32c09116610d5c610d4c6064611a84565b610ab3868a63ffffffff611aa316565b6040518363ffffffff1660e060020a0281526004018083600160a060020a0316600160a060020a0316815260200182815260200192505050602060405180830381600087803b158015610dae57600080fd5b505af1158015610dc2573d6000803e3d6000fd5b505050506040513d6020811015610dd857600080fd5b50511515610de257fe5b600354600c54604080517f3cebb823000000000000000000000000000000000000000000000000000000008152600160a060020a03928316600482015290519190921691633cebb82391602480830192600092919082900301818387803b158015610e4c57600080fd5b505af1158015610e60573d6000803e3d6000fd5b50506040517f6823b073d48d6e3a7d385eeb601452d680e74bb46afe3255a7d778f3a9b17681925060009150a1505050505050565b69065a4da25d3016c0000081565b60136020526000908152604090205481565b600b54600160a060020a031681565b600754600160a060020a031681565b60145460ff1681565b600e6020526000908152604090205481565b600154600160a060020a0316331415610f245760015460008054600160a060020a031916600160a060020a039092169190911790555b565b6003546040805160e060020a6318160ddd0281529051600092600160a060020a0316916318160ddd91600480830192602092919082900301818787803b158015610f6f57600080fd5b505af1158015610f83573d6000803e3d6000fd5b505050506040513d6020811015610f9957600080fd5b5051905090565b600054600160a060020a03163314610fb757600080fd5b600354600160a060020a031615610fcd57600080fd5b60038054600160a060020a031916600160a060020a038d811691909117918290556040805160e060020a6318160ddd028152905192909116916318160ddd916004808201926020929091908290030181600087803b15801561102e57600080fd5b505af1158015611042573d6000803e3d6000fd5b505050506040513d602081101561105857600080fd5b50511561106457600080fd5b600354604080517ff77c479100000000000000000000000000000000000000000000000000000000815290513092600160a060020a03169163f77c47919160048083019260209291908290030181600087803b1580156110c357600080fd5b505af11580156110d7573d6000803e3d6000fd5b505050506040513d60208110156110ed57600080fd5b5051600160a060020a03161461110257600080fd5b600360009054906101000a9004600160a060020a0316600160a060020a031663313ce5676040518163ffffffff1660e060020a028152600401602060405180830381600087803b15801561115557600080fd5b505af1158015611169573d6000803e3d6000fd5b505050506040513d602081101561117f57600080fd5b505160ff1660121461119057600080fd5b600160a060020a038a1615156111a557600080fd5b600c8054600160a060020a031916600160a060020a038c161790556111c8611831565b8910156111d457600080fd5b8789106111e057600080fd5b60048990556005889055600160a060020a03871615156111ff57600080fd5b600b8054600160a060020a031916600160a060020a03898116919091179091558616151561122c57600080fd5b60068054600160a060020a031916600160a060020a03888116919091179091558516151561125957600080fd5b60088054600160a060020a031916600160a060020a03878116919091179091558416151561128657600080fd5b600a8054600160a060020a031916600160a060020a0386811691909117909155831615156112b357600080fd5b60078054600160a060020a031916600160a060020a0385811691909117909155821615156112e057600080fd5b60028054600160a060020a031916600160a060020a0384811691909117918290556040805160e060020a6318160ddd028152905192909116916318160ddd916004808201926020929091908290030181600087803b15801561134157600080fd5b505af1158015611355573d6000803e3d6000fd5b505050506040513d602081101561136b57600080fd5b505181101561137957600080fd5b60095550505050505050505050565b600054600160a060020a031681565b600254600160a060020a031681565b600d6020526000908152604090205481565b600a54600160a060020a031681565b600054600160a060020a031633146113de57600080fd5b60018054600160a060020a031916600160a060020a0392909216919091179055565b600054600160a060020a0316331461141757600080fd5b6014805460ff19169055565b600354600160a060020a031681565b600354600160a060020a0316151561144957600080fd5b600054600160a060020a0316331461146057600080fd5b60045461146b611831565b1061147557600080fd5b60008111801561148f575069065a4da25d3016c000008111155b151561149a57600080fd5b600160a060020a0382166000818152600d6020908152604091829020849055815184815291517ff98a1a7197ad3a98801d4ebd0681918117f85926f7e920a2f5cf4437fd887d469281900390910190a25050565b600154600160a060020a031681565b600854600160a060020a031681565b600080548190600160a060020a0316331461152657600080fd5b600354604080517ff77c479100000000000000000000000000000000000000000000000000000000815290513092600160a060020a03169163f77c47919160048083019260209291908290030181600087803b15801561158557600080fd5b505af1158015611599573d6000803e3d6000fd5b505050506040513d60208110156115af57600080fd5b5051600160a060020a0316141561164057600354604080517fdf8de3e7000000000000000000000000000000000000000000000000000000008152600160a060020a0386811660048301529151919092169163df8de3e791602480830192600092919082900301818387803b15801561162757600080fd5b505af115801561163b573d6000803e3d6000fd5b505050505b600160a060020a03831615156116915760008054604051600160a060020a0390911691303180156108fc02929091818181858888f1935050505015801561168b573d6000803e3d6000fd5b50611808565b604080517f70a082310000000000000000000000000000000000000000000000000000000081523060048201529051849350600160a060020a038416916370a082319160248083019260209291908290030181600087803b1580156116f557600080fd5b505af1158015611709573d6000803e3d6000fd5b505050506040513d602081101561171f57600080fd5b505160008054604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152600160a060020a0392831660048201526024810185905290519394509085169263a9059cbb92604480840193602093929083900390910190829087803b15801561179557600080fd5b505af11580156117a9573d6000803e3d6000fd5b505050506040513d60208110156117bf57600080fd5b5050600054604080518381529051600160a060020a03928316928616917ff931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c919081900360200190a35b505050565b6000611826600f54601054611ae590919063ffffffff16565b905090565b60125481565b4390565b600160a060020a0381166000908152600d6020908152604080832054600e909252822054909190829061186e903463ffffffff611ae516565b11156118a557600160a060020a0383166000908152600e602052604090205461189e90839063ffffffff611a7216565b90506118a8565b50345b600160a060020a0383166000908152600e60205260409020546118d1908263ffffffff611ae516565b600160a060020a0384166000908152600e6020526040902055600f546118fd908263ffffffff611ae516565b600f5561180883826001611af4565b60008080640ba43b74003a111561192257600080fd5b600354600160a060020a031633141561193d57839250611941565b3392505b61194a83611d05565b1561195457600080fd5b600160a060020a03831660009081526013602052604090205460649061197c90610adf611831565b101561198757600080fd5b61198f611831565b600160a060020a03808516600090815260136020908152604080832094909455600b5460105485517f86bb1e03000000000000000000000000000000000000000000000000000000008152600481019190915294519316936386bb1e039360248083019491928390030190829087803b158015611a0b57600080fd5b505af1158015611a1f573d6000803e3d6000fd5b505050506040513d6020811015611a3557600080fd5b50519150348210611a47575034611a4a565b50805b601054611a5d908263ffffffff611ae516565b601055611a6c84826000611af4565b50505050565b600082821115611a7e57fe5b50900390565b6000611a9d82662386f26fc1000063ffffffff611aa316565b92915050565b6000828202831580611abf5750828482811515611abc57fe5b04145b1515611ac757fe5b9392505050565b6000808284811515611adc57fe5b04949350505050565b600082820183811015611ac757fe5b60008034841115611b0157fe5b693f870857a3e0e3800000611b1461180d565b1115611b1c57fe5b6000841115611c6557611b378461271063ffffffff611aa316565b600354604080517f827f32c0000000000000000000000000000000000000000000000000000000008152600160a060020a03898116600483015260248201859052915193955091169163827f32c0916044808201926020929091908290030181600087803b158015611ba857600080fd5b505af1158015611bbc573d6000803e3d6000fd5b505050506040513d6020811015611bd257600080fd5b50511515611bdc57fe5b600654604051600160a060020a039091169085156108fc029086906000818181858888f19350505050158015611c16573d6000803e3d6000fd5b506040805185815260208101849052841515818301529051600160a060020a038716917f3a8504b5d9cf48b7641ffa6ae4fbd66b0b38fa49ff67269024e5f62c41f485ab919081900360600190a25b611c75348563ffffffff611a7216565b90506000811115611cfe57600354600160a060020a0316331415611ccf57604051600160a060020a0386169082156108fc029083906000818181858888f19350505050158015611cc9573d6000803e3d6000fd5b50611cfe565b604051339082156108fc029083906000818181858888f19350505050158015611cfc573d6000803e3d6000fd5b505b5050505050565b600080600160a060020a0383161515611d215760009150611d2c565b823b90506000811191505b509190505600a165627a7a723058207f813e8d80578fb586a45b1ff6b6d8acf48b333ebaaa242e6c70a1441caaf2e10029"

// DeployStatusContribution deploys a new Ethereum contract, binding an instance of StatusContribution to it.
func DeployStatusContribution(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *StatusContribution, error) {
	parsed, err := abi.JSON(strings.NewReader(StatusContributionABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(StatusContributionBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &StatusContribution{StatusContributionCaller: StatusContributionCaller{contract: contract}, StatusContributionTransactor: StatusContributionTransactor{contract: contract}, StatusContributionFilterer: StatusContributionFilterer{contract: contract}}, nil
}

// StatusContribution is an auto generated Go binding around an Ethereum contract.
type StatusContribution struct {
	StatusContributionCaller     // Read-only binding to the contract
	StatusContributionTransactor // Write-only binding to the contract
	StatusContributionFilterer   // Log filterer for contract events
}

// StatusContributionCaller is an auto generated read-only Go binding around an Ethereum contract.
type StatusContributionCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StatusContributionTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StatusContributionTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StatusContributionFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StatusContributionFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StatusContributionSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StatusContributionSession struct {
	Contract     *StatusContribution // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// StatusContributionCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StatusContributionCallerSession struct {
	Contract *StatusContributionCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// StatusContributionTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StatusContributionTransactorSession struct {
	Contract     *StatusContributionTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// StatusContributionRaw is an auto generated low-level Go binding around an Ethereum contract.
type StatusContributionRaw struct {
	Contract *StatusContribution // Generic contract binding to access the raw methods on
}

// StatusContributionCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StatusContributionCallerRaw struct {
	Contract *StatusContributionCaller // Generic read-only contract binding to access the raw methods on
}

// StatusContributionTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StatusContributionTransactorRaw struct {
	Contract *StatusContributionTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStatusContribution creates a new instance of StatusContribution, bound to a specific deployed contract.
func NewStatusContribution(address common.Address, backend bind.ContractBackend) (*StatusContribution, error) {
	contract, err := bindStatusContribution(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StatusContribution{StatusContributionCaller: StatusContributionCaller{contract: contract}, StatusContributionTransactor: StatusContributionTransactor{contract: contract}, StatusContributionFilterer: StatusContributionFilterer{contract: contract}}, nil
}

// NewStatusContributionCaller creates a new read-only instance of StatusContribution, bound to a specific deployed contract.
func NewStatusContributionCaller(address common.Address, caller bind.ContractCaller) (*StatusContributionCaller, error) {
	contract, err := bindStatusContribution(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StatusContributionCaller{contract: contract}, nil
}

// NewStatusContributionTransactor creates a new write-only instance of StatusContribution, bound to a specific deployed contract.
func NewStatusContributionTransactor(address common.Address, transactor bind.ContractTransactor) (*StatusContributionTransactor, error) {
	contract, err := bindStatusContribution(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StatusContributionTransactor{contract: contract}, nil
}

// NewStatusContributionFilterer creates a new log filterer instance of StatusContribution, bound to a specific deployed contract.
func NewStatusContributionFilterer(address common.Address, filterer bind.ContractFilterer) (*StatusContributionFilterer, error) {
	contract, err := bindStatusContribution(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StatusContributionFilterer{contract: contract}, nil
}

// bindStatusContribution binds a generic wrapper to an already deployed contract.
func bindStatusContribution(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(StatusContributionABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StatusContribution *StatusContributionRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StatusContribution.Contract.StatusContributionCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StatusContribution *StatusContributionRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StatusContribution.Contract.StatusContributionTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StatusContribution *StatusContributionRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StatusContribution.Contract.StatusContributionTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StatusContribution *StatusContributionCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StatusContribution.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StatusContribution *StatusContributionTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StatusContribution.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StatusContribution *StatusContributionTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StatusContribution.Contract.contract.Transact(opts, method, params...)
}

// SGT is a free data retrieval call binding the contract method 0x9321cb7d.
//
// Solidity: function SGT() view returns(address)
func (_StatusContribution *StatusContributionCaller) SGT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "SGT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SGT is a free data retrieval call binding the contract method 0x9321cb7d.
//
// Solidity: function SGT() view returns(address)
func (_StatusContribution *StatusContributionSession) SGT() (common.Address, error) {
	return _StatusContribution.Contract.SGT(&_StatusContribution.CallOpts)
}

// SGT is a free data retrieval call binding the contract method 0x9321cb7d.
//
// Solidity: function SGT() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) SGT() (common.Address, error) {
	return _StatusContribution.Contract.SGT(&_StatusContribution.CallOpts)
}

// SNT is a free data retrieval call binding the contract method 0xc55a02a0.
//
// Solidity: function SNT() view returns(address)
func (_StatusContribution *StatusContributionCaller) SNT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "SNT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SNT is a free data retrieval call binding the contract method 0xc55a02a0.
//
// Solidity: function SNT() view returns(address)
func (_StatusContribution *StatusContributionSession) SNT() (common.Address, error) {
	return _StatusContribution.Contract.SNT(&_StatusContribution.CallOpts)
}

// SNT is a free data retrieval call binding the contract method 0xc55a02a0.
//
// Solidity: function SNT() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) SNT() (common.Address, error) {
	return _StatusContribution.Contract.SNT(&_StatusContribution.CallOpts)
}

// DestEthDevs is a free data retrieval call binding the contract method 0x01621527.
//
// Solidity: function destEthDevs() view returns(address)
func (_StatusContribution *StatusContributionCaller) DestEthDevs(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "destEthDevs")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DestEthDevs is a free data retrieval call binding the contract method 0x01621527.
//
// Solidity: function destEthDevs() view returns(address)
func (_StatusContribution *StatusContributionSession) DestEthDevs() (common.Address, error) {
	return _StatusContribution.Contract.DestEthDevs(&_StatusContribution.CallOpts)
}

// DestEthDevs is a free data retrieval call binding the contract method 0x01621527.
//
// Solidity: function destEthDevs() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) DestEthDevs() (common.Address, error) {
	return _StatusContribution.Contract.DestEthDevs(&_StatusContribution.CallOpts)
}

// DestTokensDevs is a free data retrieval call binding the contract method 0x5bd475fd.
//
// Solidity: function destTokensDevs() view returns(address)
func (_StatusContribution *StatusContributionCaller) DestTokensDevs(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "destTokensDevs")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DestTokensDevs is a free data retrieval call binding the contract method 0x5bd475fd.
//
// Solidity: function destTokensDevs() view returns(address)
func (_StatusContribution *StatusContributionSession) DestTokensDevs() (common.Address, error) {
	return _StatusContribution.Contract.DestTokensDevs(&_StatusContribution.CallOpts)
}

// DestTokensDevs is a free data retrieval call binding the contract method 0x5bd475fd.
//
// Solidity: function destTokensDevs() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) DestTokensDevs() (common.Address, error) {
	return _StatusContribution.Contract.DestTokensDevs(&_StatusContribution.CallOpts)
}

// DestTokensReserve is a free data retrieval call binding the contract method 0xd7e07d5f.
//
// Solidity: function destTokensReserve() view returns(address)
func (_StatusContribution *StatusContributionCaller) DestTokensReserve(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "destTokensReserve")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DestTokensReserve is a free data retrieval call binding the contract method 0xd7e07d5f.
//
// Solidity: function destTokensReserve() view returns(address)
func (_StatusContribution *StatusContributionSession) DestTokensReserve() (common.Address, error) {
	return _StatusContribution.Contract.DestTokensReserve(&_StatusContribution.CallOpts)
}

// DestTokensReserve is a free data retrieval call binding the contract method 0xd7e07d5f.
//
// Solidity: function destTokensReserve() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) DestTokensReserve() (common.Address, error) {
	return _StatusContribution.Contract.DestTokensReserve(&_StatusContribution.CallOpts)
}

// DestTokensSgt is a free data retrieval call binding the contract method 0xa1a7405a.
//
// Solidity: function destTokensSgt() view returns(address)
func (_StatusContribution *StatusContributionCaller) DestTokensSgt(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "destTokensSgt")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DestTokensSgt is a free data retrieval call binding the contract method 0xa1a7405a.
//
// Solidity: function destTokensSgt() view returns(address)
func (_StatusContribution *StatusContributionSession) DestTokensSgt() (common.Address, error) {
	return _StatusContribution.Contract.DestTokensSgt(&_StatusContribution.CallOpts)
}

// DestTokensSgt is a free data retrieval call binding the contract method 0xa1a7405a.
//
// Solidity: function destTokensSgt() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) DestTokensSgt() (common.Address, error) {
	return _StatusContribution.Contract.DestTokensSgt(&_StatusContribution.CallOpts)
}

// DynamicCeiling is a free data retrieval call binding the contract method 0x5a3c8826.
//
// Solidity: function dynamicCeiling() view returns(address)
func (_StatusContribution *StatusContributionCaller) DynamicCeiling(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "dynamicCeiling")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DynamicCeiling is a free data retrieval call binding the contract method 0x5a3c8826.
//
// Solidity: function dynamicCeiling() view returns(address)
func (_StatusContribution *StatusContributionSession) DynamicCeiling() (common.Address, error) {
	return _StatusContribution.Contract.DynamicCeiling(&_StatusContribution.CallOpts)
}

// DynamicCeiling is a free data retrieval call binding the contract method 0x5a3c8826.
//
// Solidity: function dynamicCeiling() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) DynamicCeiling() (common.Address, error) {
	return _StatusContribution.Contract.DynamicCeiling(&_StatusContribution.CallOpts)
}

// EndBlock is a free data retrieval call binding the contract method 0x083c6323.
//
// Solidity: function endBlock() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) EndBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "endBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EndBlock is a free data retrieval call binding the contract method 0x083c6323.
//
// Solidity: function endBlock() view returns(uint256)
func (_StatusContribution *StatusContributionSession) EndBlock() (*big.Int, error) {
	return _StatusContribution.Contract.EndBlock(&_StatusContribution.CallOpts)
}

// EndBlock is a free data retrieval call binding the contract method 0x083c6323.
//
// Solidity: function endBlock() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) EndBlock() (*big.Int, error) {
	return _StatusContribution.Contract.EndBlock(&_StatusContribution.CallOpts)
}

// ExchangeRate is a free data retrieval call binding the contract method 0x3ba0b9a9.
//
// Solidity: function exchangeRate() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) ExchangeRate(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "exchangeRate")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ExchangeRate is a free data retrieval call binding the contract method 0x3ba0b9a9.
//
// Solidity: function exchangeRate() view returns(uint256)
func (_StatusContribution *StatusContributionSession) ExchangeRate() (*big.Int, error) {
	return _StatusContribution.Contract.ExchangeRate(&_StatusContribution.CallOpts)
}

// ExchangeRate is a free data retrieval call binding the contract method 0x3ba0b9a9.
//
// Solidity: function exchangeRate() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) ExchangeRate() (*big.Int, error) {
	return _StatusContribution.Contract.ExchangeRate(&_StatusContribution.CallOpts)
}

// FailSafeLimit is a free data retrieval call binding the contract method 0x23dc1314.
//
// Solidity: function failSafeLimit() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) FailSafeLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "failSafeLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FailSafeLimit is a free data retrieval call binding the contract method 0x23dc1314.
//
// Solidity: function failSafeLimit() view returns(uint256)
func (_StatusContribution *StatusContributionSession) FailSafeLimit() (*big.Int, error) {
	return _StatusContribution.Contract.FailSafeLimit(&_StatusContribution.CallOpts)
}

// FailSafeLimit is a free data retrieval call binding the contract method 0x23dc1314.
//
// Solidity: function failSafeLimit() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) FailSafeLimit() (*big.Int, error) {
	return _StatusContribution.Contract.FailSafeLimit(&_StatusContribution.CallOpts)
}

// FinalizedBlock is a free data retrieval call binding the contract method 0x4084c3ab.
//
// Solidity: function finalizedBlock() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) FinalizedBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "finalizedBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FinalizedBlock is a free data retrieval call binding the contract method 0x4084c3ab.
//
// Solidity: function finalizedBlock() view returns(uint256)
func (_StatusContribution *StatusContributionSession) FinalizedBlock() (*big.Int, error) {
	return _StatusContribution.Contract.FinalizedBlock(&_StatusContribution.CallOpts)
}

// FinalizedBlock is a free data retrieval call binding the contract method 0x4084c3ab.
//
// Solidity: function finalizedBlock() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) FinalizedBlock() (*big.Int, error) {
	return _StatusContribution.Contract.FinalizedBlock(&_StatusContribution.CallOpts)
}

// FinalizedTime is a free data retrieval call binding the contract method 0xfe67a189.
//
// Solidity: function finalizedTime() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) FinalizedTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "finalizedTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FinalizedTime is a free data retrieval call binding the contract method 0xfe67a189.
//
// Solidity: function finalizedTime() view returns(uint256)
func (_StatusContribution *StatusContributionSession) FinalizedTime() (*big.Int, error) {
	return _StatusContribution.Contract.FinalizedTime(&_StatusContribution.CallOpts)
}

// FinalizedTime is a free data retrieval call binding the contract method 0xfe67a189.
//
// Solidity: function finalizedTime() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) FinalizedTime() (*big.Int, error) {
	return _StatusContribution.Contract.FinalizedTime(&_StatusContribution.CallOpts)
}

// GuaranteedBuyersBought is a free data retrieval call binding the contract method 0x73aad472.
//
// Solidity: function guaranteedBuyersBought(address ) view returns(uint256)
func (_StatusContribution *StatusContributionCaller) GuaranteedBuyersBought(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "guaranteedBuyersBought", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GuaranteedBuyersBought is a free data retrieval call binding the contract method 0x73aad472.
//
// Solidity: function guaranteedBuyersBought(address ) view returns(uint256)
func (_StatusContribution *StatusContributionSession) GuaranteedBuyersBought(arg0 common.Address) (*big.Int, error) {
	return _StatusContribution.Contract.GuaranteedBuyersBought(&_StatusContribution.CallOpts, arg0)
}

// GuaranteedBuyersBought is a free data retrieval call binding the contract method 0x73aad472.
//
// Solidity: function guaranteedBuyersBought(address ) view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) GuaranteedBuyersBought(arg0 common.Address) (*big.Int, error) {
	return _StatusContribution.Contract.GuaranteedBuyersBought(&_StatusContribution.CallOpts, arg0)
}

// GuaranteedBuyersLimit is a free data retrieval call binding the contract method 0x9752bcd3.
//
// Solidity: function guaranteedBuyersLimit(address ) view returns(uint256)
func (_StatusContribution *StatusContributionCaller) GuaranteedBuyersLimit(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "guaranteedBuyersLimit", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GuaranteedBuyersLimit is a free data retrieval call binding the contract method 0x9752bcd3.
//
// Solidity: function guaranteedBuyersLimit(address ) view returns(uint256)
func (_StatusContribution *StatusContributionSession) GuaranteedBuyersLimit(arg0 common.Address) (*big.Int, error) {
	return _StatusContribution.Contract.GuaranteedBuyersLimit(&_StatusContribution.CallOpts, arg0)
}

// GuaranteedBuyersLimit is a free data retrieval call binding the contract method 0x9752bcd3.
//
// Solidity: function guaranteedBuyersLimit(address ) view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) GuaranteedBuyersLimit(arg0 common.Address) (*big.Int, error) {
	return _StatusContribution.Contract.GuaranteedBuyersLimit(&_StatusContribution.CallOpts, arg0)
}

// LastCallBlock is a free data retrieval call binding the contract method 0x548e0846.
//
// Solidity: function lastCallBlock(address ) view returns(uint256)
func (_StatusContribution *StatusContributionCaller) LastCallBlock(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "lastCallBlock", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastCallBlock is a free data retrieval call binding the contract method 0x548e0846.
//
// Solidity: function lastCallBlock(address ) view returns(uint256)
func (_StatusContribution *StatusContributionSession) LastCallBlock(arg0 common.Address) (*big.Int, error) {
	return _StatusContribution.Contract.LastCallBlock(&_StatusContribution.CallOpts, arg0)
}

// LastCallBlock is a free data retrieval call binding the contract method 0x548e0846.
//
// Solidity: function lastCallBlock(address ) view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) LastCallBlock(arg0 common.Address) (*big.Int, error) {
	return _StatusContribution.Contract.LastCallBlock(&_StatusContribution.CallOpts, arg0)
}

// MaxCallFrequency is a free data retrieval call binding the contract method 0x17183ca3.
//
// Solidity: function maxCallFrequency() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) MaxCallFrequency(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "maxCallFrequency")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxCallFrequency is a free data retrieval call binding the contract method 0x17183ca3.
//
// Solidity: function maxCallFrequency() view returns(uint256)
func (_StatusContribution *StatusContributionSession) MaxCallFrequency() (*big.Int, error) {
	return _StatusContribution.Contract.MaxCallFrequency(&_StatusContribution.CallOpts)
}

// MaxCallFrequency is a free data retrieval call binding the contract method 0x17183ca3.
//
// Solidity: function maxCallFrequency() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) MaxCallFrequency() (*big.Int, error) {
	return _StatusContribution.Contract.MaxCallFrequency(&_StatusContribution.CallOpts)
}

// MaxGasPrice is a free data retrieval call binding the contract method 0x3de39c11.
//
// Solidity: function maxGasPrice() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) MaxGasPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "maxGasPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxGasPrice is a free data retrieval call binding the contract method 0x3de39c11.
//
// Solidity: function maxGasPrice() view returns(uint256)
func (_StatusContribution *StatusContributionSession) MaxGasPrice() (*big.Int, error) {
	return _StatusContribution.Contract.MaxGasPrice(&_StatusContribution.CallOpts)
}

// MaxGasPrice is a free data retrieval call binding the contract method 0x3de39c11.
//
// Solidity: function maxGasPrice() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) MaxGasPrice() (*big.Int, error) {
	return _StatusContribution.Contract.MaxGasPrice(&_StatusContribution.CallOpts)
}

// MaxGuaranteedLimit is a free data retrieval call binding the contract method 0x4d1ed74b.
//
// Solidity: function maxGuaranteedLimit() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) MaxGuaranteedLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "maxGuaranteedLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxGuaranteedLimit is a free data retrieval call binding the contract method 0x4d1ed74b.
//
// Solidity: function maxGuaranteedLimit() view returns(uint256)
func (_StatusContribution *StatusContributionSession) MaxGuaranteedLimit() (*big.Int, error) {
	return _StatusContribution.Contract.MaxGuaranteedLimit(&_StatusContribution.CallOpts)
}

// MaxGuaranteedLimit is a free data retrieval call binding the contract method 0x4d1ed74b.
//
// Solidity: function maxGuaranteedLimit() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) MaxGuaranteedLimit() (*big.Int, error) {
	return _StatusContribution.Contract.MaxGuaranteedLimit(&_StatusContribution.CallOpts)
}

// MaxSGTSupply is a free data retrieval call binding the contract method 0x092c506e.
//
// Solidity: function maxSGTSupply() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) MaxSGTSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "maxSGTSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxSGTSupply is a free data retrieval call binding the contract method 0x092c506e.
//
// Solidity: function maxSGTSupply() view returns(uint256)
func (_StatusContribution *StatusContributionSession) MaxSGTSupply() (*big.Int, error) {
	return _StatusContribution.Contract.MaxSGTSupply(&_StatusContribution.CallOpts)
}

// MaxSGTSupply is a free data retrieval call binding the contract method 0x092c506e.
//
// Solidity: function maxSGTSupply() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) MaxSGTSupply() (*big.Int, error) {
	return _StatusContribution.Contract.MaxSGTSupply(&_StatusContribution.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_StatusContribution *StatusContributionCaller) NewOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "newOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_StatusContribution *StatusContributionSession) NewOwner() (common.Address, error) {
	return _StatusContribution.Contract.NewOwner(&_StatusContribution.CallOpts)
}

// NewOwner is a free data retrieval call binding the contract method 0xd4ee1d90.
//
// Solidity: function newOwner() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) NewOwner() (common.Address, error) {
	return _StatusContribution.Contract.NewOwner(&_StatusContribution.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_StatusContribution *StatusContributionCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_StatusContribution *StatusContributionSession) Owner() (common.Address, error) {
	return _StatusContribution.Contract.Owner(&_StatusContribution.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) Owner() (common.Address, error) {
	return _StatusContribution.Contract.Owner(&_StatusContribution.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_StatusContribution *StatusContributionCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_StatusContribution *StatusContributionSession) Paused() (bool, error) {
	return _StatusContribution.Contract.Paused(&_StatusContribution.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_StatusContribution *StatusContributionCallerSession) Paused() (bool, error) {
	return _StatusContribution.Contract.Paused(&_StatusContribution.CallOpts)
}

// SntController is a free data retrieval call binding the contract method 0x1b2f1109.
//
// Solidity: function sntController() view returns(address)
func (_StatusContribution *StatusContributionCaller) SntController(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "sntController")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SntController is a free data retrieval call binding the contract method 0x1b2f1109.
//
// Solidity: function sntController() view returns(address)
func (_StatusContribution *StatusContributionSession) SntController() (common.Address, error) {
	return _StatusContribution.Contract.SntController(&_StatusContribution.CallOpts)
}

// SntController is a free data retrieval call binding the contract method 0x1b2f1109.
//
// Solidity: function sntController() view returns(address)
func (_StatusContribution *StatusContributionCallerSession) SntController() (common.Address, error) {
	return _StatusContribution.Contract.SntController(&_StatusContribution.CallOpts)
}

// StartBlock is a free data retrieval call binding the contract method 0x48cd4cb1.
//
// Solidity: function startBlock() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) StartBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "startBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StartBlock is a free data retrieval call binding the contract method 0x48cd4cb1.
//
// Solidity: function startBlock() view returns(uint256)
func (_StatusContribution *StatusContributionSession) StartBlock() (*big.Int, error) {
	return _StatusContribution.Contract.StartBlock(&_StatusContribution.CallOpts)
}

// StartBlock is a free data retrieval call binding the contract method 0x48cd4cb1.
//
// Solidity: function startBlock() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) StartBlock() (*big.Int, error) {
	return _StatusContribution.Contract.StartBlock(&_StatusContribution.CallOpts)
}

// TokensIssued is a free data retrieval call binding the contract method 0x7c48bbda.
//
// Solidity: function tokensIssued() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) TokensIssued(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "tokensIssued")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokensIssued is a free data retrieval call binding the contract method 0x7c48bbda.
//
// Solidity: function tokensIssued() view returns(uint256)
func (_StatusContribution *StatusContributionSession) TokensIssued() (*big.Int, error) {
	return _StatusContribution.Contract.TokensIssued(&_StatusContribution.CallOpts)
}

// TokensIssued is a free data retrieval call binding the contract method 0x7c48bbda.
//
// Solidity: function tokensIssued() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) TokensIssued() (*big.Int, error) {
	return _StatusContribution.Contract.TokensIssued(&_StatusContribution.CallOpts)
}

// TotalCollected is a free data retrieval call binding the contract method 0xe29eb836.
//
// Solidity: function totalCollected() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) TotalCollected(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "totalCollected")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalCollected is a free data retrieval call binding the contract method 0xe29eb836.
//
// Solidity: function totalCollected() view returns(uint256)
func (_StatusContribution *StatusContributionSession) TotalCollected() (*big.Int, error) {
	return _StatusContribution.Contract.TotalCollected(&_StatusContribution.CallOpts)
}

// TotalCollected is a free data retrieval call binding the contract method 0xe29eb836.
//
// Solidity: function totalCollected() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) TotalCollected() (*big.Int, error) {
	return _StatusContribution.Contract.TotalCollected(&_StatusContribution.CallOpts)
}

// TotalGuaranteedCollected is a free data retrieval call binding the contract method 0x137935d5.
//
// Solidity: function totalGuaranteedCollected() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) TotalGuaranteedCollected(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "totalGuaranteedCollected")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalGuaranteedCollected is a free data retrieval call binding the contract method 0x137935d5.
//
// Solidity: function totalGuaranteedCollected() view returns(uint256)
func (_StatusContribution *StatusContributionSession) TotalGuaranteedCollected() (*big.Int, error) {
	return _StatusContribution.Contract.TotalGuaranteedCollected(&_StatusContribution.CallOpts)
}

// TotalGuaranteedCollected is a free data retrieval call binding the contract method 0x137935d5.
//
// Solidity: function totalGuaranteedCollected() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) TotalGuaranteedCollected() (*big.Int, error) {
	return _StatusContribution.Contract.TotalGuaranteedCollected(&_StatusContribution.CallOpts)
}

// TotalNormalCollected is a free data retrieval call binding the contract method 0x1517d107.
//
// Solidity: function totalNormalCollected() view returns(uint256)
func (_StatusContribution *StatusContributionCaller) TotalNormalCollected(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StatusContribution.contract.Call(opts, &out, "totalNormalCollected")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalNormalCollected is a free data retrieval call binding the contract method 0x1517d107.
//
// Solidity: function totalNormalCollected() view returns(uint256)
func (_StatusContribution *StatusContributionSession) TotalNormalCollected() (*big.Int, error) {
	return _StatusContribution.Contract.TotalNormalCollected(&_StatusContribution.CallOpts)
}

// TotalNormalCollected is a free data retrieval call binding the contract method 0x1517d107.
//
// Solidity: function totalNormalCollected() view returns(uint256)
func (_StatusContribution *StatusContributionCallerSession) TotalNormalCollected() (*big.Int, error) {
	return _StatusContribution.Contract.TotalNormalCollected(&_StatusContribution.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_StatusContribution *StatusContributionTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_StatusContribution *StatusContributionSession) AcceptOwnership() (*types.Transaction, error) {
	return _StatusContribution.Contract.AcceptOwnership(&_StatusContribution.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_StatusContribution *StatusContributionTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _StatusContribution.Contract.AcceptOwnership(&_StatusContribution.TransactOpts)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_StatusContribution *StatusContributionTransactor) ChangeOwner(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "changeOwner", _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_StatusContribution *StatusContributionSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _StatusContribution.Contract.ChangeOwner(&_StatusContribution.TransactOpts, _newOwner)
}

// ChangeOwner is a paid mutator transaction binding the contract method 0xa6f9dae1.
//
// Solidity: function changeOwner(address _newOwner) returns()
func (_StatusContribution *StatusContributionTransactorSession) ChangeOwner(_newOwner common.Address) (*types.Transaction, error) {
	return _StatusContribution.Contract.ChangeOwner(&_StatusContribution.TransactOpts, _newOwner)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_StatusContribution *StatusContributionTransactor) ClaimTokens(opts *bind.TransactOpts, _token common.Address) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "claimTokens", _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_StatusContribution *StatusContributionSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _StatusContribution.Contract.ClaimTokens(&_StatusContribution.TransactOpts, _token)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0xdf8de3e7.
//
// Solidity: function claimTokens(address _token) returns()
func (_StatusContribution *StatusContributionTransactorSession) ClaimTokens(_token common.Address) (*types.Transaction, error) {
	return _StatusContribution.Contract.ClaimTokens(&_StatusContribution.TransactOpts, _token)
}

// Finalize is a paid mutator transaction binding the contract method 0x4bb278f3.
//
// Solidity: function finalize() returns()
func (_StatusContribution *StatusContributionTransactor) Finalize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "finalize")
}

// Finalize is a paid mutator transaction binding the contract method 0x4bb278f3.
//
// Solidity: function finalize() returns()
func (_StatusContribution *StatusContributionSession) Finalize() (*types.Transaction, error) {
	return _StatusContribution.Contract.Finalize(&_StatusContribution.TransactOpts)
}

// Finalize is a paid mutator transaction binding the contract method 0x4bb278f3.
//
// Solidity: function finalize() returns()
func (_StatusContribution *StatusContributionTransactorSession) Finalize() (*types.Transaction, error) {
	return _StatusContribution.Contract.Finalize(&_StatusContribution.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8733f360.
//
// Solidity: function initialize(address _snt, address _sntController, uint256 _startBlock, uint256 _endBlock, address _dynamicCeiling, address _destEthDevs, address _destTokensReserve, address _destTokensSgt, address _destTokensDevs, address _sgt, uint256 _maxSGTSupply) returns()
func (_StatusContribution *StatusContributionTransactor) Initialize(opts *bind.TransactOpts, _snt common.Address, _sntController common.Address, _startBlock *big.Int, _endBlock *big.Int, _dynamicCeiling common.Address, _destEthDevs common.Address, _destTokensReserve common.Address, _destTokensSgt common.Address, _destTokensDevs common.Address, _sgt common.Address, _maxSGTSupply *big.Int) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "initialize", _snt, _sntController, _startBlock, _endBlock, _dynamicCeiling, _destEthDevs, _destTokensReserve, _destTokensSgt, _destTokensDevs, _sgt, _maxSGTSupply)
}

// Initialize is a paid mutator transaction binding the contract method 0x8733f360.
//
// Solidity: function initialize(address _snt, address _sntController, uint256 _startBlock, uint256 _endBlock, address _dynamicCeiling, address _destEthDevs, address _destTokensReserve, address _destTokensSgt, address _destTokensDevs, address _sgt, uint256 _maxSGTSupply) returns()
func (_StatusContribution *StatusContributionSession) Initialize(_snt common.Address, _sntController common.Address, _startBlock *big.Int, _endBlock *big.Int, _dynamicCeiling common.Address, _destEthDevs common.Address, _destTokensReserve common.Address, _destTokensSgt common.Address, _destTokensDevs common.Address, _sgt common.Address, _maxSGTSupply *big.Int) (*types.Transaction, error) {
	return _StatusContribution.Contract.Initialize(&_StatusContribution.TransactOpts, _snt, _sntController, _startBlock, _endBlock, _dynamicCeiling, _destEthDevs, _destTokensReserve, _destTokensSgt, _destTokensDevs, _sgt, _maxSGTSupply)
}

// Initialize is a paid mutator transaction binding the contract method 0x8733f360.
//
// Solidity: function initialize(address _snt, address _sntController, uint256 _startBlock, uint256 _endBlock, address _dynamicCeiling, address _destEthDevs, address _destTokensReserve, address _destTokensSgt, address _destTokensDevs, address _sgt, uint256 _maxSGTSupply) returns()
func (_StatusContribution *StatusContributionTransactorSession) Initialize(_snt common.Address, _sntController common.Address, _startBlock *big.Int, _endBlock *big.Int, _dynamicCeiling common.Address, _destEthDevs common.Address, _destTokensReserve common.Address, _destTokensSgt common.Address, _destTokensDevs common.Address, _sgt common.Address, _maxSGTSupply *big.Int) (*types.Transaction, error) {
	return _StatusContribution.Contract.Initialize(&_StatusContribution.TransactOpts, _snt, _sntController, _startBlock, _endBlock, _dynamicCeiling, _destEthDevs, _destTokensReserve, _destTokensSgt, _destTokensDevs, _sgt, _maxSGTSupply)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address , address , uint256 ) returns(bool)
func (_StatusContribution *StatusContributionTransactor) OnApprove(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "onApprove", arg0, arg1, arg2)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address , address , uint256 ) returns(bool)
func (_StatusContribution *StatusContributionSession) OnApprove(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _StatusContribution.Contract.OnApprove(&_StatusContribution.TransactOpts, arg0, arg1, arg2)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address , address , uint256 ) returns(bool)
func (_StatusContribution *StatusContributionTransactorSession) OnApprove(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _StatusContribution.Contract.OnApprove(&_StatusContribution.TransactOpts, arg0, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address , address , uint256 ) returns(bool)
func (_StatusContribution *StatusContributionTransactor) OnTransfer(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "onTransfer", arg0, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address , address , uint256 ) returns(bool)
func (_StatusContribution *StatusContributionSession) OnTransfer(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _StatusContribution.Contract.OnTransfer(&_StatusContribution.TransactOpts, arg0, arg1, arg2)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address , address , uint256 ) returns(bool)
func (_StatusContribution *StatusContributionTransactorSession) OnTransfer(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (*types.Transaction, error) {
	return _StatusContribution.Contract.OnTransfer(&_StatusContribution.TransactOpts, arg0, arg1, arg2)
}

// PauseContribution is a paid mutator transaction binding the contract method 0x4b8adcf7.
//
// Solidity: function pauseContribution() returns()
func (_StatusContribution *StatusContributionTransactor) PauseContribution(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "pauseContribution")
}

// PauseContribution is a paid mutator transaction binding the contract method 0x4b8adcf7.
//
// Solidity: function pauseContribution() returns()
func (_StatusContribution *StatusContributionSession) PauseContribution() (*types.Transaction, error) {
	return _StatusContribution.Contract.PauseContribution(&_StatusContribution.TransactOpts)
}

// PauseContribution is a paid mutator transaction binding the contract method 0x4b8adcf7.
//
// Solidity: function pauseContribution() returns()
func (_StatusContribution *StatusContributionTransactorSession) PauseContribution() (*types.Transaction, error) {
	return _StatusContribution.Contract.PauseContribution(&_StatusContribution.TransactOpts)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address _th) payable returns(bool)
func (_StatusContribution *StatusContributionTransactor) ProxyPayment(opts *bind.TransactOpts, _th common.Address) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "proxyPayment", _th)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address _th) payable returns(bool)
func (_StatusContribution *StatusContributionSession) ProxyPayment(_th common.Address) (*types.Transaction, error) {
	return _StatusContribution.Contract.ProxyPayment(&_StatusContribution.TransactOpts, _th)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address _th) payable returns(bool)
func (_StatusContribution *StatusContributionTransactorSession) ProxyPayment(_th common.Address) (*types.Transaction, error) {
	return _StatusContribution.Contract.ProxyPayment(&_StatusContribution.TransactOpts, _th)
}

// ResumeContribution is a paid mutator transaction binding the contract method 0xb681f9f6.
//
// Solidity: function resumeContribution() returns()
func (_StatusContribution *StatusContributionTransactor) ResumeContribution(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "resumeContribution")
}

// ResumeContribution is a paid mutator transaction binding the contract method 0xb681f9f6.
//
// Solidity: function resumeContribution() returns()
func (_StatusContribution *StatusContributionSession) ResumeContribution() (*types.Transaction, error) {
	return _StatusContribution.Contract.ResumeContribution(&_StatusContribution.TransactOpts)
}

// ResumeContribution is a paid mutator transaction binding the contract method 0xb681f9f6.
//
// Solidity: function resumeContribution() returns()
func (_StatusContribution *StatusContributionTransactorSession) ResumeContribution() (*types.Transaction, error) {
	return _StatusContribution.Contract.ResumeContribution(&_StatusContribution.TransactOpts)
}

// SetGuaranteedAddress is a paid mutator transaction binding the contract method 0xcc9b7826.
//
// Solidity: function setGuaranteedAddress(address _th, uint256 _limit) returns()
func (_StatusContribution *StatusContributionTransactor) SetGuaranteedAddress(opts *bind.TransactOpts, _th common.Address, _limit *big.Int) (*types.Transaction, error) {
	return _StatusContribution.contract.Transact(opts, "setGuaranteedAddress", _th, _limit)
}

// SetGuaranteedAddress is a paid mutator transaction binding the contract method 0xcc9b7826.
//
// Solidity: function setGuaranteedAddress(address _th, uint256 _limit) returns()
func (_StatusContribution *StatusContributionSession) SetGuaranteedAddress(_th common.Address, _limit *big.Int) (*types.Transaction, error) {
	return _StatusContribution.Contract.SetGuaranteedAddress(&_StatusContribution.TransactOpts, _th, _limit)
}

// SetGuaranteedAddress is a paid mutator transaction binding the contract method 0xcc9b7826.
//
// Solidity: function setGuaranteedAddress(address _th, uint256 _limit) returns()
func (_StatusContribution *StatusContributionTransactorSession) SetGuaranteedAddress(_th common.Address, _limit *big.Int) (*types.Transaction, error) {
	return _StatusContribution.Contract.SetGuaranteedAddress(&_StatusContribution.TransactOpts, _th, _limit)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_StatusContribution *StatusContributionTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _StatusContribution.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_StatusContribution *StatusContributionSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _StatusContribution.Contract.Fallback(&_StatusContribution.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_StatusContribution *StatusContributionTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _StatusContribution.Contract.Fallback(&_StatusContribution.TransactOpts, calldata)
}

// StatusContributionClaimedTokensIterator is returned from FilterClaimedTokens and is used to iterate over the raw logs and unpacked data for ClaimedTokens events raised by the StatusContribution contract.
type StatusContributionClaimedTokensIterator struct {
	Event *StatusContributionClaimedTokens // Event containing the contract specifics and raw log

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
func (it *StatusContributionClaimedTokensIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StatusContributionClaimedTokens)
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
		it.Event = new(StatusContributionClaimedTokens)
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
func (it *StatusContributionClaimedTokensIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StatusContributionClaimedTokensIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StatusContributionClaimedTokens represents a ClaimedTokens event raised by the StatusContribution contract.
type StatusContributionClaimedTokens struct {
	Token      common.Address
	Controller common.Address
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterClaimedTokens is a free log retrieval operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_StatusContribution *StatusContributionFilterer) FilterClaimedTokens(opts *bind.FilterOpts, _token []common.Address, _controller []common.Address) (*StatusContributionClaimedTokensIterator, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _StatusContribution.contract.FilterLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return &StatusContributionClaimedTokensIterator{contract: _StatusContribution.contract, event: "ClaimedTokens", logs: logs, sub: sub}, nil
}

// WatchClaimedTokens is a free log subscription operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_StatusContribution *StatusContributionFilterer) WatchClaimedTokens(opts *bind.WatchOpts, sink chan<- *StatusContributionClaimedTokens, _token []common.Address, _controller []common.Address) (event.Subscription, error) {

	var _tokenRule []interface{}
	for _, _tokenItem := range _token {
		_tokenRule = append(_tokenRule, _tokenItem)
	}
	var _controllerRule []interface{}
	for _, _controllerItem := range _controller {
		_controllerRule = append(_controllerRule, _controllerItem)
	}

	logs, sub, err := _StatusContribution.contract.WatchLogs(opts, "ClaimedTokens", _tokenRule, _controllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StatusContributionClaimedTokens)
				if err := _StatusContribution.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
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

// ParseClaimedTokens is a log parse operation binding the contract event 0xf931edb47c50b4b4104c187b5814a9aef5f709e17e2ecf9617e860cacade929c.
//
// Solidity: event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount)
func (_StatusContribution *StatusContributionFilterer) ParseClaimedTokens(log types.Log) (*StatusContributionClaimedTokens, error) {
	event := new(StatusContributionClaimedTokens)
	if err := _StatusContribution.contract.UnpackLog(event, "ClaimedTokens", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StatusContributionFinalizedIterator is returned from FilterFinalized and is used to iterate over the raw logs and unpacked data for Finalized events raised by the StatusContribution contract.
type StatusContributionFinalizedIterator struct {
	Event *StatusContributionFinalized // Event containing the contract specifics and raw log

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
func (it *StatusContributionFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StatusContributionFinalized)
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
		it.Event = new(StatusContributionFinalized)
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
func (it *StatusContributionFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StatusContributionFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StatusContributionFinalized represents a Finalized event raised by the StatusContribution contract.
type StatusContributionFinalized struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterFinalized is a free log retrieval operation binding the contract event 0x6823b073d48d6e3a7d385eeb601452d680e74bb46afe3255a7d778f3a9b17681.
//
// Solidity: event Finalized()
func (_StatusContribution *StatusContributionFilterer) FilterFinalized(opts *bind.FilterOpts) (*StatusContributionFinalizedIterator, error) {

	logs, sub, err := _StatusContribution.contract.FilterLogs(opts, "Finalized")
	if err != nil {
		return nil, err
	}
	return &StatusContributionFinalizedIterator{contract: _StatusContribution.contract, event: "Finalized", logs: logs, sub: sub}, nil
}

// WatchFinalized is a free log subscription operation binding the contract event 0x6823b073d48d6e3a7d385eeb601452d680e74bb46afe3255a7d778f3a9b17681.
//
// Solidity: event Finalized()
func (_StatusContribution *StatusContributionFilterer) WatchFinalized(opts *bind.WatchOpts, sink chan<- *StatusContributionFinalized) (event.Subscription, error) {

	logs, sub, err := _StatusContribution.contract.WatchLogs(opts, "Finalized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StatusContributionFinalized)
				if err := _StatusContribution.contract.UnpackLog(event, "Finalized", log); err != nil {
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

// ParseFinalized is a log parse operation binding the contract event 0x6823b073d48d6e3a7d385eeb601452d680e74bb46afe3255a7d778f3a9b17681.
//
// Solidity: event Finalized()
func (_StatusContribution *StatusContributionFilterer) ParseFinalized(log types.Log) (*StatusContributionFinalized, error) {
	event := new(StatusContributionFinalized)
	if err := _StatusContribution.contract.UnpackLog(event, "Finalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StatusContributionGuaranteedAddressIterator is returned from FilterGuaranteedAddress and is used to iterate over the raw logs and unpacked data for GuaranteedAddress events raised by the StatusContribution contract.
type StatusContributionGuaranteedAddressIterator struct {
	Event *StatusContributionGuaranteedAddress // Event containing the contract specifics and raw log

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
func (it *StatusContributionGuaranteedAddressIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StatusContributionGuaranteedAddress)
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
		it.Event = new(StatusContributionGuaranteedAddress)
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
func (it *StatusContributionGuaranteedAddressIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StatusContributionGuaranteedAddressIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StatusContributionGuaranteedAddress represents a GuaranteedAddress event raised by the StatusContribution contract.
type StatusContributionGuaranteedAddress struct {
	Th    common.Address
	Limit *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterGuaranteedAddress is a free log retrieval operation binding the contract event 0xf98a1a7197ad3a98801d4ebd0681918117f85926f7e920a2f5cf4437fd887d46.
//
// Solidity: event GuaranteedAddress(address indexed _th, uint256 _limit)
func (_StatusContribution *StatusContributionFilterer) FilterGuaranteedAddress(opts *bind.FilterOpts, _th []common.Address) (*StatusContributionGuaranteedAddressIterator, error) {

	var _thRule []interface{}
	for _, _thItem := range _th {
		_thRule = append(_thRule, _thItem)
	}

	logs, sub, err := _StatusContribution.contract.FilterLogs(opts, "GuaranteedAddress", _thRule)
	if err != nil {
		return nil, err
	}
	return &StatusContributionGuaranteedAddressIterator{contract: _StatusContribution.contract, event: "GuaranteedAddress", logs: logs, sub: sub}, nil
}

// WatchGuaranteedAddress is a free log subscription operation binding the contract event 0xf98a1a7197ad3a98801d4ebd0681918117f85926f7e920a2f5cf4437fd887d46.
//
// Solidity: event GuaranteedAddress(address indexed _th, uint256 _limit)
func (_StatusContribution *StatusContributionFilterer) WatchGuaranteedAddress(opts *bind.WatchOpts, sink chan<- *StatusContributionGuaranteedAddress, _th []common.Address) (event.Subscription, error) {

	var _thRule []interface{}
	for _, _thItem := range _th {
		_thRule = append(_thRule, _thItem)
	}

	logs, sub, err := _StatusContribution.contract.WatchLogs(opts, "GuaranteedAddress", _thRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StatusContributionGuaranteedAddress)
				if err := _StatusContribution.contract.UnpackLog(event, "GuaranteedAddress", log); err != nil {
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

// ParseGuaranteedAddress is a log parse operation binding the contract event 0xf98a1a7197ad3a98801d4ebd0681918117f85926f7e920a2f5cf4437fd887d46.
//
// Solidity: event GuaranteedAddress(address indexed _th, uint256 _limit)
func (_StatusContribution *StatusContributionFilterer) ParseGuaranteedAddress(log types.Log) (*StatusContributionGuaranteedAddress, error) {
	event := new(StatusContributionGuaranteedAddress)
	if err := _StatusContribution.contract.UnpackLog(event, "GuaranteedAddress", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StatusContributionNewSaleIterator is returned from FilterNewSale and is used to iterate over the raw logs and unpacked data for NewSale events raised by the StatusContribution contract.
type StatusContributionNewSaleIterator struct {
	Event *StatusContributionNewSale // Event containing the contract specifics and raw log

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
func (it *StatusContributionNewSaleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StatusContributionNewSale)
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
		it.Event = new(StatusContributionNewSale)
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
func (it *StatusContributionNewSaleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StatusContributionNewSaleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StatusContributionNewSale represents a NewSale event raised by the StatusContribution contract.
type StatusContributionNewSale struct {
	Th         common.Address
	Amount     *big.Int
	Tokens     *big.Int
	Guaranteed bool
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNewSale is a free log retrieval operation binding the contract event 0x3a8504b5d9cf48b7641ffa6ae4fbd66b0b38fa49ff67269024e5f62c41f485ab.
//
// Solidity: event NewSale(address indexed _th, uint256 _amount, uint256 _tokens, bool _guaranteed)
func (_StatusContribution *StatusContributionFilterer) FilterNewSale(opts *bind.FilterOpts, _th []common.Address) (*StatusContributionNewSaleIterator, error) {

	var _thRule []interface{}
	for _, _thItem := range _th {
		_thRule = append(_thRule, _thItem)
	}

	logs, sub, err := _StatusContribution.contract.FilterLogs(opts, "NewSale", _thRule)
	if err != nil {
		return nil, err
	}
	return &StatusContributionNewSaleIterator{contract: _StatusContribution.contract, event: "NewSale", logs: logs, sub: sub}, nil
}

// WatchNewSale is a free log subscription operation binding the contract event 0x3a8504b5d9cf48b7641ffa6ae4fbd66b0b38fa49ff67269024e5f62c41f485ab.
//
// Solidity: event NewSale(address indexed _th, uint256 _amount, uint256 _tokens, bool _guaranteed)
func (_StatusContribution *StatusContributionFilterer) WatchNewSale(opts *bind.WatchOpts, sink chan<- *StatusContributionNewSale, _th []common.Address) (event.Subscription, error) {

	var _thRule []interface{}
	for _, _thItem := range _th {
		_thRule = append(_thRule, _thItem)
	}

	logs, sub, err := _StatusContribution.contract.WatchLogs(opts, "NewSale", _thRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StatusContributionNewSale)
				if err := _StatusContribution.contract.UnpackLog(event, "NewSale", log); err != nil {
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

// ParseNewSale is a log parse operation binding the contract event 0x3a8504b5d9cf48b7641ffa6ae4fbd66b0b38fa49ff67269024e5f62c41f485ab.
//
// Solidity: event NewSale(address indexed _th, uint256 _amount, uint256 _tokens, bool _guaranteed)
func (_StatusContribution *StatusContributionFilterer) ParseNewSale(log types.Log) (*StatusContributionNewSale, error) {
	event := new(StatusContributionNewSale)
	if err := _StatusContribution.contract.UnpackLog(event, "NewSale", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenControllerABI is the input ABI used to generate the binding from.
const TokenControllerABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"onTransfer\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"onApprove\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"proxyPayment\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"}]"

// TokenControllerFuncSigs maps the 4-byte function signature to its string representation.
var TokenControllerFuncSigs = map[string]string{
	"da682aeb": "onApprove(address,address,uint256)",
	"4a393149": "onTransfer(address,address,uint256)",
	"f48c3054": "proxyPayment(address)",
}

// TokenController is an auto generated Go binding around an Ethereum contract.
type TokenController struct {
	TokenControllerCaller     // Read-only binding to the contract
	TokenControllerTransactor // Write-only binding to the contract
	TokenControllerFilterer   // Log filterer for contract events
}

// TokenControllerCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenControllerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenControllerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenControllerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenControllerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenControllerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenControllerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenControllerSession struct {
	Contract     *TokenController  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenControllerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenControllerCallerSession struct {
	Contract *TokenControllerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// TokenControllerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenControllerTransactorSession struct {
	Contract     *TokenControllerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// TokenControllerRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenControllerRaw struct {
	Contract *TokenController // Generic contract binding to access the raw methods on
}

// TokenControllerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenControllerCallerRaw struct {
	Contract *TokenControllerCaller // Generic read-only contract binding to access the raw methods on
}

// TokenControllerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenControllerTransactorRaw struct {
	Contract *TokenControllerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTokenController creates a new instance of TokenController, bound to a specific deployed contract.
func NewTokenController(address common.Address, backend bind.ContractBackend) (*TokenController, error) {
	contract, err := bindTokenController(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TokenController{TokenControllerCaller: TokenControllerCaller{contract: contract}, TokenControllerTransactor: TokenControllerTransactor{contract: contract}, TokenControllerFilterer: TokenControllerFilterer{contract: contract}}, nil
}

// NewTokenControllerCaller creates a new read-only instance of TokenController, bound to a specific deployed contract.
func NewTokenControllerCaller(address common.Address, caller bind.ContractCaller) (*TokenControllerCaller, error) {
	contract, err := bindTokenController(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenControllerCaller{contract: contract}, nil
}

// NewTokenControllerTransactor creates a new write-only instance of TokenController, bound to a specific deployed contract.
func NewTokenControllerTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenControllerTransactor, error) {
	contract, err := bindTokenController(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenControllerTransactor{contract: contract}, nil
}

// NewTokenControllerFilterer creates a new log filterer instance of TokenController, bound to a specific deployed contract.
func NewTokenControllerFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenControllerFilterer, error) {
	contract, err := bindTokenController(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenControllerFilterer{contract: contract}, nil
}

// bindTokenController binds a generic wrapper to an already deployed contract.
func bindTokenController(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TokenControllerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TokenController *TokenControllerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TokenController.Contract.TokenControllerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TokenController *TokenControllerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TokenController.Contract.TokenControllerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TokenController *TokenControllerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TokenController.Contract.TokenControllerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TokenController *TokenControllerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TokenController.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TokenController *TokenControllerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TokenController.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TokenController *TokenControllerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TokenController.Contract.contract.Transact(opts, method, params...)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address _owner, address _spender, uint256 _amount) returns(bool)
func (_TokenController *TokenControllerTransactor) OnApprove(opts *bind.TransactOpts, _owner common.Address, _spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _TokenController.contract.Transact(opts, "onApprove", _owner, _spender, _amount)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address _owner, address _spender, uint256 _amount) returns(bool)
func (_TokenController *TokenControllerSession) OnApprove(_owner common.Address, _spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _TokenController.Contract.OnApprove(&_TokenController.TransactOpts, _owner, _spender, _amount)
}

// OnApprove is a paid mutator transaction binding the contract method 0xda682aeb.
//
// Solidity: function onApprove(address _owner, address _spender, uint256 _amount) returns(bool)
func (_TokenController *TokenControllerTransactorSession) OnApprove(_owner common.Address, _spender common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _TokenController.Contract.OnApprove(&_TokenController.TransactOpts, _owner, _spender, _amount)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address _from, address _to, uint256 _amount) returns(bool)
func (_TokenController *TokenControllerTransactor) OnTransfer(opts *bind.TransactOpts, _from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _TokenController.contract.Transact(opts, "onTransfer", _from, _to, _amount)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address _from, address _to, uint256 _amount) returns(bool)
func (_TokenController *TokenControllerSession) OnTransfer(_from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _TokenController.Contract.OnTransfer(&_TokenController.TransactOpts, _from, _to, _amount)
}

// OnTransfer is a paid mutator transaction binding the contract method 0x4a393149.
//
// Solidity: function onTransfer(address _from, address _to, uint256 _amount) returns(bool)
func (_TokenController *TokenControllerTransactorSession) OnTransfer(_from common.Address, _to common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _TokenController.Contract.OnTransfer(&_TokenController.TransactOpts, _from, _to, _amount)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address _owner) payable returns(bool)
func (_TokenController *TokenControllerTransactor) ProxyPayment(opts *bind.TransactOpts, _owner common.Address) (*types.Transaction, error) {
	return _TokenController.contract.Transact(opts, "proxyPayment", _owner)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address _owner) payable returns(bool)
func (_TokenController *TokenControllerSession) ProxyPayment(_owner common.Address) (*types.Transaction, error) {
	return _TokenController.Contract.ProxyPayment(&_TokenController.TransactOpts, _owner)
}

// ProxyPayment is a paid mutator transaction binding the contract method 0xf48c3054.
//
// Solidity: function proxyPayment(address _owner) payable returns(bool)
func (_TokenController *TokenControllerTransactorSession) ProxyPayment(_owner common.Address) (*types.Transaction, error) {
	return _TokenController.Contract.ProxyPayment(&_TokenController.TransactOpts, _owner)
}
