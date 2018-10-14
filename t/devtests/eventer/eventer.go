// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package eventer

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

// EventerABI is the input ABI used to generate the binding from.
const EventerABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"crt\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"topic\",\"type\":\"bytes32\"}],\"name\":\"emit\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"topic\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"crt\",\"type\":\"int256\"}],\"name\":\"Message\",\"type\":\"event\"}]"

// EventerBin is the compiled bytecode used for deploying new contracts.
const EventerBin = `0x608060405234801561001057600080fd5b5060f28061001f6000396000f30060806040526004361060485763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630868aad68114604d578063eee17820146071575b600080fd5b348015605857600080fd5b50605f6088565b60408051918252519081900360200190f35b348015607c57600080fd5b506086600435608e565b005b60005481565b60008054600101808255604051909183917fc3dab353f3a8451adb4c3071c9df72eebc7e900383c3295d66fe939bba21e1c99190a3505600a165627a7a72305820121895a4c6e091225d36a33443bf84f57560aa732841830ddf282def477ba9570029`

// DeployEventer deploys a new Ethereum contract, binding an instance of Eventer to it.
func DeployEventer(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Eventer, error) {
	parsed, err := abi.JSON(strings.NewReader(EventerABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(EventerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Eventer{EventerCaller: EventerCaller{contract: contract}, EventerTransactor: EventerTransactor{contract: contract}, EventerFilterer: EventerFilterer{contract: contract}}, nil
}

// Eventer is an auto generated Go binding around an Ethereum contract.
type Eventer struct {
	EventerCaller     // Read-only binding to the contract
	EventerTransactor // Write-only binding to the contract
	EventerFilterer   // Log filterer for contract events
}

// EventerCaller is an auto generated read-only Go binding around an Ethereum contract.
type EventerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EventerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EventerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EventerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EventerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EventerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EventerSession struct {
	Contract     *Eventer          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// EventerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EventerCallerSession struct {
	Contract *EventerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// EventerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EventerTransactorSession struct {
	Contract     *EventerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// EventerRaw is an auto generated low-level Go binding around an Ethereum contract.
type EventerRaw struct {
	Contract *Eventer // Generic contract binding to access the raw methods on
}

// EventerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EventerCallerRaw struct {
	Contract *EventerCaller // Generic read-only contract binding to access the raw methods on
}

// EventerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EventerTransactorRaw struct {
	Contract *EventerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEventer creates a new instance of Eventer, bound to a specific deployed contract.
func NewEventer(address common.Address, backend bind.ContractBackend) (*Eventer, error) {
	contract, err := bindEventer(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Eventer{EventerCaller: EventerCaller{contract: contract}, EventerTransactor: EventerTransactor{contract: contract}, EventerFilterer: EventerFilterer{contract: contract}}, nil
}

// NewEventerCaller creates a new read-only instance of Eventer, bound to a specific deployed contract.
func NewEventerCaller(address common.Address, caller bind.ContractCaller) (*EventerCaller, error) {
	contract, err := bindEventer(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EventerCaller{contract: contract}, nil
}

// NewEventerTransactor creates a new write-only instance of Eventer, bound to a specific deployed contract.
func NewEventerTransactor(address common.Address, transactor bind.ContractTransactor) (*EventerTransactor, error) {
	contract, err := bindEventer(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EventerTransactor{contract: contract}, nil
}

// NewEventerFilterer creates a new log filterer instance of Eventer, bound to a specific deployed contract.
func NewEventerFilterer(address common.Address, filterer bind.ContractFilterer) (*EventerFilterer, error) {
	contract, err := bindEventer(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EventerFilterer{contract: contract}, nil
}

// bindEventer binds a generic wrapper to an already deployed contract.
func bindEventer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(EventerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Eventer *EventerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Eventer.Contract.EventerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Eventer *EventerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Eventer.Contract.EventerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Eventer *EventerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Eventer.Contract.EventerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Eventer *EventerCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Eventer.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Eventer *EventerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Eventer.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Eventer *EventerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Eventer.Contract.contract.Transact(opts, method, params...)
}

// Crt is a free data retrieval call binding the contract method 0x0868aad6.
//
// Solidity: function crt() constant returns(int256)
func (_Eventer *EventerCaller) Crt(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Eventer.contract.Call(opts, out, "crt")
	return *ret0, err
}

// Crt is a free data retrieval call binding the contract method 0x0868aad6.
//
// Solidity: function crt() constant returns(int256)
func (_Eventer *EventerSession) Crt() (*big.Int, error) {
	return _Eventer.Contract.Crt(&_Eventer.CallOpts)
}

// Crt is a free data retrieval call binding the contract method 0x0868aad6.
//
// Solidity: function crt() constant returns(int256)
func (_Eventer *EventerCallerSession) Crt() (*big.Int, error) {
	return _Eventer.Contract.Crt(&_Eventer.CallOpts)
}

// Emit is a paid mutator transaction binding the contract method 0xeee17820.
//
// Solidity: function emit(topic bytes32) returns()
func (_Eventer *EventerTransactor) Emit(opts *bind.TransactOpts, topic [32]byte) (*types.Transaction, error) {
	return _Eventer.contract.Transact(opts, "emit", topic)
}

// Emit is a paid mutator transaction binding the contract method 0xeee17820.
//
// Solidity: function emit(topic bytes32) returns()
func (_Eventer *EventerSession) Emit(topic [32]byte) (*types.Transaction, error) {
	return _Eventer.Contract.Emit(&_Eventer.TransactOpts, topic)
}

// Emit is a paid mutator transaction binding the contract method 0xeee17820.
//
// Solidity: function emit(topic bytes32) returns()
func (_Eventer *EventerTransactorSession) Emit(topic [32]byte) (*types.Transaction, error) {
	return _Eventer.Contract.Emit(&_Eventer.TransactOpts, topic)
}

// EventerMessageIterator is returned from FilterMessage and is used to iterate over the raw logs and unpacked data for Message events raised by the Eventer contract.
type EventerMessageIterator struct {
	Event *EventerMessage // Event containing the contract specifics and raw log

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
func (it *EventerMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EventerMessage)
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
		it.Event = new(EventerMessage)
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
func (it *EventerMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EventerMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EventerMessage represents a Message event raised by the Eventer contract.
type EventerMessage struct {
	Topic [32]byte
	Crt   *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterMessage is a free log retrieval operation binding the contract event 0xc3dab353f3a8451adb4c3071c9df72eebc7e900383c3295d66fe939bba21e1c9.
//
// Solidity: e Message(topic indexed bytes32, crt indexed int256)
func (_Eventer *EventerFilterer) FilterMessage(opts *bind.FilterOpts, topic [][32]byte, crt []*big.Int) (*EventerMessageIterator, error) {

	var topicRule []interface{}
	for _, topicItem := range topic {
		topicRule = append(topicRule, topicItem)
	}
	var crtRule []interface{}
	for _, crtItem := range crt {
		crtRule = append(crtRule, crtItem)
	}

	logs, sub, err := _Eventer.contract.FilterLogs(opts, "Message", topicRule, crtRule)
	if err != nil {
		return nil, err
	}
	return &EventerMessageIterator{contract: _Eventer.contract, event: "Message", logs: logs, sub: sub}, nil
}

// WatchMessage is a free log subscription operation binding the contract event 0xc3dab353f3a8451adb4c3071c9df72eebc7e900383c3295d66fe939bba21e1c9.
//
// Solidity: e Message(topic indexed bytes32, crt indexed int256)
func (_Eventer *EventerFilterer) WatchMessage(opts *bind.WatchOpts, sink chan<- *EventerMessage, topic [][32]byte, crt []*big.Int) (event.Subscription, error) {

	var topicRule []interface{}
	for _, topicItem := range topic {
		topicRule = append(topicRule, topicItem)
	}
	var crtRule []interface{}
	for _, crtItem := range crt {
		crtRule = append(crtRule, crtItem)
	}

	logs, sub, err := _Eventer.contract.WatchLogs(opts, "Message", topicRule, crtRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EventerMessage)
				if err := _Eventer.contract.UnpackLog(event, "Message", log); err != nil {
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

