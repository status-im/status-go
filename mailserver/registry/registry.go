// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registry

import (
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// RegistryABI is the input ABI used to generate the binding from.
const RegistryABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_newController\",\"type\":\"address\"}],\"name\":\"changeController\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"a\",\"type\":\"bytes\"}],\"name\":\"remove\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"a\",\"type\":\"bytes\"}],\"name\":\"exists\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"a\",\"type\":\"bytes\"}],\"name\":\"add\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"controller\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"a\",\"type\":\"bytes\"}],\"name\":\"MailServerAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"a\",\"type\":\"bytes\"}],\"name\":\"MailServerRemoved\",\"type\":\"event\"}]"

// RegistryBin is the compiled bytecode used for deploying new contracts.
const RegistryBin = `6080604052336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610511806100536000396000f30060806040526004361061006d576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680633cebb8231461007257806358edef4c146100b557806379fc09a2146100f0578063ba65811114610143578063f77c47911461017e575b600080fd5b34801561007e57600080fd5b506100b3600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506101d5565b005b3480156100c157600080fd5b506100ee600480360381019080803590602001908201803590602001919091929391929390505050610273565b005b3480156100fc57600080fd5b50610129600480360381019080803590602001908201803590602001919091929391929390505050610396565b604051808215151515815260200191505060405180910390f35b34801561014f57600080fd5b5061017c6004803603810190808035906020019082018035906020019190919293919293905050506103d3565b005b34801561018a57600080fd5b506101936104c0565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561023057600080fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156102ce57600080fd5b6001828260405180838380828437820191505092505050908152602001604051809103902060009054906101000a900460ff16151561030c57600080fd5b6001828260405180838380828437820191505092505050908152602001604051809103902060006101000a81549060ff02191690557f44e7d85a87eeb950b8bdc144d44b0b474be610d1953607251a0130edc10a222b8282604051808060200182810382528484828181526020019250808284378201915050935050505060405180910390a15050565b60006001838360405180838380828437820191505092505050908152602001604051809103902060009054906101000a900460ff16905092915050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561042e57600080fd5b600180838360405180838380828437820191505092505050908152602001604051809103902060006101000a81548160ff0219169083151502179055507fcb379cb5890ec9889055734e1561cdc353a342d46d8d650c8c3a8d66383c29cd8282604051808060200182810382528484828181526020019250808284378201915050935050505060405180910390a15050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff16815600a165627a7a723058205234376a4ed154ce31a74225ac2258d6a4a82fd5fef967aaaf17c005c40b6f370029`

// DeployRegistry deploys a new Ethereum contract, binding an instance of Registry to it.
func DeployRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Registry, error) {
	parsed, err := abi.JSON(strings.NewReader(RegistryABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(RegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// Registry is an auto generated Go binding around an Ethereum contract.
type Registry struct {
	RegistryCaller     // Read-only binding to the contract
	RegistryTransactor // Write-only binding to the contract
	RegistryFilterer   // Log filterer for contract events
}

// RegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type RegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RegistrySession struct {
	Contract     *Registry         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RegistryCallerSession struct {
	Contract *RegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// RegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RegistryTransactorSession struct {
	Contract     *RegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// RegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type RegistryRaw struct {
	Contract *Registry // Generic contract binding to access the raw methods on
}

// RegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RegistryCallerRaw struct {
	Contract *RegistryCaller // Generic read-only contract binding to access the raw methods on
}

// RegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RegistryTransactorRaw struct {
	Contract *RegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRegistry creates a new instance of Registry, bound to a specific deployed contract.
func NewRegistry(address common.Address, backend bind.ContractBackend) (*Registry, error) {
	contract, err := bindRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// NewRegistryCaller creates a new read-only instance of Registry, bound to a specific deployed contract.
func NewRegistryCaller(address common.Address, caller bind.ContractCaller) (*RegistryCaller, error) {
	contract, err := bindRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryCaller{contract: contract}, nil
}

// NewRegistryTransactor creates a new write-only instance of Registry, bound to a specific deployed contract.
func NewRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*RegistryTransactor, error) {
	contract, err := bindRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryTransactor{contract: contract}, nil
}

// NewRegistryFilterer creates a new log filterer instance of Registry, bound to a specific deployed contract.
func NewRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*RegistryFilterer, error) {
	contract, err := bindRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RegistryFilterer{contract: contract}, nil
}

// bindRegistry binds a generic wrapper to an already deployed contract.
func bindRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(RegistryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.RegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transact(opts, method, params...)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() constant returns(address)
func (_Registry *RegistryCaller) Controller(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Registry.contract.Call(opts, out, "controller")
	return *ret0, err
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() constant returns(address)
func (_Registry *RegistrySession) Controller() (common.Address, error) {
	return _Registry.Contract.Controller(&_Registry.CallOpts)
}

// Controller is a free data retrieval call binding the contract method 0xf77c4791.
//
// Solidity: function controller() constant returns(address)
func (_Registry *RegistryCallerSession) Controller() (common.Address, error) {
	return _Registry.Contract.Controller(&_Registry.CallOpts)
}

// Exists is a free data retrieval call binding the contract method 0x79fc09a2.
//
// Solidity: function exists(a bytes) constant returns(bool)
func (_Registry *RegistryCaller) Exists(opts *bind.CallOpts, a []byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Registry.contract.Call(opts, out, "exists", a)
	return *ret0, err
}

// Exists is a free data retrieval call binding the contract method 0x79fc09a2.
//
// Solidity: function exists(a bytes) constant returns(bool)
func (_Registry *RegistrySession) Exists(a []byte) (bool, error) {
	return _Registry.Contract.Exists(&_Registry.CallOpts, a)
}

// Exists is a free data retrieval call binding the contract method 0x79fc09a2.
//
// Solidity: function exists(a bytes) constant returns(bool)
func (_Registry *RegistryCallerSession) Exists(a []byte) (bool, error) {
	return _Registry.Contract.Exists(&_Registry.CallOpts, a)
}

// Add is a paid mutator transaction binding the contract method 0xba658111.
//
// Solidity: function add(a bytes) returns()
func (_Registry *RegistryTransactor) Add(opts *bind.TransactOpts, a []byte) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "add", a)
}

// Add is a paid mutator transaction binding the contract method 0xba658111.
//
// Solidity: function add(a bytes) returns()
func (_Registry *RegistrySession) Add(a []byte) (*types.Transaction, error) {
	return _Registry.Contract.Add(&_Registry.TransactOpts, a)
}

// Add is a paid mutator transaction binding the contract method 0xba658111.
//
// Solidity: function add(a bytes) returns()
func (_Registry *RegistryTransactorSession) Add(a []byte) (*types.Transaction, error) {
	return _Registry.Contract.Add(&_Registry.TransactOpts, a)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(_newController address) returns()
func (_Registry *RegistryTransactor) ChangeController(opts *bind.TransactOpts, _newController common.Address) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "changeController", _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(_newController address) returns()
func (_Registry *RegistrySession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _Registry.Contract.ChangeController(&_Registry.TransactOpts, _newController)
}

// ChangeController is a paid mutator transaction binding the contract method 0x3cebb823.
//
// Solidity: function changeController(_newController address) returns()
func (_Registry *RegistryTransactorSession) ChangeController(_newController common.Address) (*types.Transaction, error) {
	return _Registry.Contract.ChangeController(&_Registry.TransactOpts, _newController)
}

// Remove is a paid mutator transaction binding the contract method 0x58edef4c.
//
// Solidity: function remove(a bytes) returns()
func (_Registry *RegistryTransactor) Remove(opts *bind.TransactOpts, a []byte) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "remove", a)
}

// Remove is a paid mutator transaction binding the contract method 0x58edef4c.
//
// Solidity: function remove(a bytes) returns()
func (_Registry *RegistrySession) Remove(a []byte) (*types.Transaction, error) {
	return _Registry.Contract.Remove(&_Registry.TransactOpts, a)
}

// Remove is a paid mutator transaction binding the contract method 0x58edef4c.
//
// Solidity: function remove(a bytes) returns()
func (_Registry *RegistryTransactorSession) Remove(a []byte) (*types.Transaction, error) {
	return _Registry.Contract.Remove(&_Registry.TransactOpts, a)
}

// RegistryMailServerAddedIterator is returned from FilterMailServerAdded and is used to iterate over the raw logs and unpacked data for MailServerAdded events raised by the Registry contract.
type RegistryMailServerAddedIterator struct {
	Event *RegistryMailServerAdded // Event containing the contract specifics and raw log

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
func (it *RegistryMailServerAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryMailServerAdded)
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
		it.Event = new(RegistryMailServerAdded)
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
func (it *RegistryMailServerAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryMailServerAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryMailServerAdded represents a MailServerAdded event raised by the Registry contract.
type RegistryMailServerAdded struct {
	A   []byte
	Raw types.Log // Blockchain specific contextual infos
}

// FilterMailServerAdded is a free log retrieval operation binding the contract event 0xcb379cb5890ec9889055734e1561cdc353a342d46d8d650c8c3a8d66383c29cd.
//
// Solidity: e MailServerAdded(a bytes)
func (_Registry *RegistryFilterer) FilterMailServerAdded(opts *bind.FilterOpts) (*RegistryMailServerAddedIterator, error) {

	logs, sub, err := _Registry.contract.FilterLogs(opts, "MailServerAdded")
	if err != nil {
		return nil, err
	}
	return &RegistryMailServerAddedIterator{contract: _Registry.contract, event: "MailServerAdded", logs: logs, sub: sub}, nil
}

// WatchMailServerAdded is a free log subscription operation binding the contract event 0xcb379cb5890ec9889055734e1561cdc353a342d46d8d650c8c3a8d66383c29cd.
//
// Solidity: e MailServerAdded(a bytes)
func (_Registry *RegistryFilterer) WatchMailServerAdded(opts *bind.WatchOpts, sink chan<- *RegistryMailServerAdded) (event.Subscription, error) {

	logs, sub, err := _Registry.contract.WatchLogs(opts, "MailServerAdded")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryMailServerAdded)
				if err := _Registry.contract.UnpackLog(event, "MailServerAdded", log); err != nil {
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

// RegistryMailServerRemovedIterator is returned from FilterMailServerRemoved and is used to iterate over the raw logs and unpacked data for MailServerRemoved events raised by the Registry contract.
type RegistryMailServerRemovedIterator struct {
	Event *RegistryMailServerRemoved // Event containing the contract specifics and raw log

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
func (it *RegistryMailServerRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryMailServerRemoved)
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
		it.Event = new(RegistryMailServerRemoved)
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
func (it *RegistryMailServerRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryMailServerRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryMailServerRemoved represents a MailServerRemoved event raised by the Registry contract.
type RegistryMailServerRemoved struct {
	A   []byte
	Raw types.Log // Blockchain specific contextual infos
}

// FilterMailServerRemoved is a free log retrieval operation binding the contract event 0x44e7d85a87eeb950b8bdc144d44b0b474be610d1953607251a0130edc10a222b.
//
// Solidity: e MailServerRemoved(a bytes)
func (_Registry *RegistryFilterer) FilterMailServerRemoved(opts *bind.FilterOpts) (*RegistryMailServerRemovedIterator, error) {

	logs, sub, err := _Registry.contract.FilterLogs(opts, "MailServerRemoved")
	if err != nil {
		return nil, err
	}
	return &RegistryMailServerRemovedIterator{contract: _Registry.contract, event: "MailServerRemoved", logs: logs, sub: sub}, nil
}

// WatchMailServerRemoved is a free log subscription operation binding the contract event 0x44e7d85a87eeb950b8bdc144d44b0b474be610d1953607251a0130edc10a222b.
//
// Solidity: e MailServerRemoved(a bytes)
func (_Registry *RegistryFilterer) WatchMailServerRemoved(opts *bind.WatchOpts, sink chan<- *RegistryMailServerRemoved) (event.Subscription, error) {

	logs, sub, err := _Registry.contract.WatchLogs(opts, "MailServerRemoved")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryMailServerRemoved)
				if err := _Registry.contract.UnpackLog(event, "MailServerRemoved", log); err != nil {
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
