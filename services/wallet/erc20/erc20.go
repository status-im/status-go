// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package erc20

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
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// ERC20TransferABI is the input ABI used to generate the binding from.
const ERC20TransferABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"}]"

// ERC20TransferBin is the compiled bytecode used for deploying new contracts.
const ERC20TransferBin = `0x6080604052348015600f57600080fd5b5060c88061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063a9059cbb14602d575b600080fd5b605660048036036040811015604157600080fd5b506001600160a01b0381351690602001356058565b005b6040805182815290516001600160a01b0384169133917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a3505056fea165627a7a723058205a3b651d160745f3384583bfcd779db1e53bc7e335198ba8a8503fc353ab2a630029`

// DeployERC20Transfer deploys a new Ethereum contract, binding an instance of ERC20Transfer to it.
func DeployERC20Transfer(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ERC20Transfer, error) {
	parsed, err := abi.JSON(strings.NewReader(ERC20TransferABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ERC20TransferBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ERC20Transfer{ERC20TransferCaller: ERC20TransferCaller{contract: contract}, ERC20TransferTransactor: ERC20TransferTransactor{contract: contract}, ERC20TransferFilterer: ERC20TransferFilterer{contract: contract}}, nil
}

// ERC20Transfer is an auto generated Go binding around an Ethereum contract.
type ERC20Transfer struct {
	ERC20TransferCaller     // Read-only binding to the contract
	ERC20TransferTransactor // Write-only binding to the contract
	ERC20TransferFilterer   // Log filterer for contract events
}

// ERC20TransferCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC20TransferCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TransferTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC20TransferTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TransferFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC20TransferFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TransferSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC20TransferSession struct {
	Contract     *ERC20Transfer    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ERC20TransferCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC20TransferCallerSession struct {
	Contract *ERC20TransferCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ERC20TransferTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC20TransferTransactorSession struct {
	Contract     *ERC20TransferTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ERC20TransferRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC20TransferRaw struct {
	Contract *ERC20Transfer // Generic contract binding to access the raw methods on
}

// ERC20TransferCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC20TransferCallerRaw struct {
	Contract *ERC20TransferCaller // Generic read-only contract binding to access the raw methods on
}

// ERC20TransferTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC20TransferTransactorRaw struct {
	Contract *ERC20TransferTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC20Transfer creates a new instance of ERC20Transfer, bound to a specific deployed contract.
func NewERC20Transfer(address common.Address, backend bind.ContractBackend) (*ERC20Transfer, error) {
	contract, err := bindERC20Transfer(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC20Transfer{ERC20TransferCaller: ERC20TransferCaller{contract: contract}, ERC20TransferTransactor: ERC20TransferTransactor{contract: contract}, ERC20TransferFilterer: ERC20TransferFilterer{contract: contract}}, nil
}

// NewERC20TransferCaller creates a new read-only instance of ERC20Transfer, bound to a specific deployed contract.
func NewERC20TransferCaller(address common.Address, caller bind.ContractCaller) (*ERC20TransferCaller, error) {
	contract, err := bindERC20Transfer(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20TransferCaller{contract: contract}, nil
}

// NewERC20TransferTransactor creates a new write-only instance of ERC20Transfer, bound to a specific deployed contract.
func NewERC20TransferTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC20TransferTransactor, error) {
	contract, err := bindERC20Transfer(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20TransferTransactor{contract: contract}, nil
}

// NewERC20TransferFilterer creates a new log filterer instance of ERC20Transfer, bound to a specific deployed contract.
func NewERC20TransferFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC20TransferFilterer, error) {
	contract, err := bindERC20Transfer(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC20TransferFilterer{contract: contract}, nil
}

// bindERC20Transfer binds a generic wrapper to an already deployed contract.
func bindERC20Transfer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ERC20TransferABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Transfer *ERC20TransferRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ERC20Transfer.Contract.ERC20TransferCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Transfer *ERC20TransferRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Transfer.Contract.ERC20TransferTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Transfer *ERC20TransferRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Transfer.Contract.ERC20TransferTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Transfer *ERC20TransferCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _ERC20Transfer.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Transfer *ERC20TransferTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Transfer.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Transfer *ERC20TransferTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Transfer.Contract.contract.Transact(opts, method, params...)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns()
func (_ERC20Transfer *ERC20TransferTransactor) Transfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Transfer.contract.Transact(opts, "transfer", to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns()
func (_ERC20Transfer *ERC20TransferSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Transfer.Contract.Transfer(&_ERC20Transfer.TransactOpts, to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns()
func (_ERC20Transfer *ERC20TransferTransactorSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Transfer.Contract.Transfer(&_ERC20Transfer.TransactOpts, to, value)
}

// ERC20TransferTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the ERC20Transfer contract.
type ERC20TransferTransferIterator struct {
	Event *ERC20TransferTransfer // Event containing the contract specifics and raw log

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
func (it *ERC20TransferTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20TransferTransfer)
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
		it.Event = new(ERC20TransferTransfer)
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
func (it *ERC20TransferTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20TransferTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20TransferTransfer represents a Transfer event raised by the ERC20Transfer contract.
type ERC20TransferTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Transfer *ERC20TransferFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*ERC20TransferTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ERC20Transfer.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &ERC20TransferTransferIterator{contract: _ERC20Transfer.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Transfer *ERC20TransferFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *ERC20TransferTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ERC20Transfer.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20TransferTransfer)
				if err := _ERC20Transfer.contract.UnpackLog(event, "Transfer", log); err != nil {
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

