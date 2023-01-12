// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package collectibles

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

// CollectiblesABI is the input ABI used to generate the binding from.
const CollectiblesABI = "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y\",\"type\":\"uint256\"}],\"name\":\"multiply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]"

// CollectiblesBin is the compiled bytecode used for deploying new contracts.
var CollectiblesBin = "0x608060405234801561001057600080fd5b506101c2806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063165c4a1614610030575b600080fd5b61004a600480360381019061004591906100b1565b610060565b6040516100579190610100565b60405180910390f35b6000818361006e919061014a565b905092915050565b600080fd5b6000819050919050565b61008e8161007b565b811461009957600080fd5b50565b6000813590506100ab81610085565b92915050565b600080604083850312156100c8576100c7610076565b5b60006100d68582860161009c565b92505060206100e78582860161009c565b9150509250929050565b6100fa8161007b565b82525050565b600060208201905061011560008301846100f1565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60006101558261007b565b91506101608361007b565b925082820261016e8161007b565b915082820484148315176101855761018461011b565b5b509291505056fea264697066735822122060dd77a889afe7f9a40e7826c21de50df92115ecf261489dd7cd9725195d9ab564736f6c63430008110033"

// DeployCollectibles deploys a new Ethereum contract, binding an instance of Collectibles to it.
func DeployCollectibles(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Collectibles, error) {
	parsed, err := abi.JSON(strings.NewReader(CollectiblesABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(CollectiblesBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Collectibles{CollectiblesCaller: CollectiblesCaller{contract: contract}, CollectiblesTransactor: CollectiblesTransactor{contract: contract}, CollectiblesFilterer: CollectiblesFilterer{contract: contract}}, nil
}

// Collectibles is an auto generated Go binding around an Ethereum contract.
type Collectibles struct {
	CollectiblesCaller     // Read-only binding to the contract
	CollectiblesTransactor // Write-only binding to the contract
	CollectiblesFilterer   // Log filterer for contract events
}

// CollectiblesCaller is an auto generated read-only Go binding around an Ethereum contract.
type CollectiblesCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CollectiblesTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CollectiblesTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CollectiblesFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CollectiblesFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CollectiblesSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CollectiblesSession struct {
	Contract     *Collectibles     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CollectiblesCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CollectiblesCallerSession struct {
	Contract *CollectiblesCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// CollectiblesTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CollectiblesTransactorSession struct {
	Contract     *CollectiblesTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// CollectiblesRaw is an auto generated low-level Go binding around an Ethereum contract.
type CollectiblesRaw struct {
	Contract *Collectibles // Generic contract binding to access the raw methods on
}

// CollectiblesCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CollectiblesCallerRaw struct {
	Contract *CollectiblesCaller // Generic read-only contract binding to access the raw methods on
}

// CollectiblesTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CollectiblesTransactorRaw struct {
	Contract *CollectiblesTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCollectibles creates a new instance of Collectibles, bound to a specific deployed contract.
func NewCollectibles(address common.Address, backend bind.ContractBackend) (*Collectibles, error) {
	contract, err := bindCollectibles(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Collectibles{CollectiblesCaller: CollectiblesCaller{contract: contract}, CollectiblesTransactor: CollectiblesTransactor{contract: contract}, CollectiblesFilterer: CollectiblesFilterer{contract: contract}}, nil
}

// NewCollectiblesCaller creates a new read-only instance of Collectibles, bound to a specific deployed contract.
func NewCollectiblesCaller(address common.Address, caller bind.ContractCaller) (*CollectiblesCaller, error) {
	contract, err := bindCollectibles(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CollectiblesCaller{contract: contract}, nil
}

// NewCollectiblesTransactor creates a new write-only instance of Collectibles, bound to a specific deployed contract.
func NewCollectiblesTransactor(address common.Address, transactor bind.ContractTransactor) (*CollectiblesTransactor, error) {
	contract, err := bindCollectibles(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CollectiblesTransactor{contract: contract}, nil
}

// NewCollectiblesFilterer creates a new log filterer instance of Collectibles, bound to a specific deployed contract.
func NewCollectiblesFilterer(address common.Address, filterer bind.ContractFilterer) (*CollectiblesFilterer, error) {
	contract, err := bindCollectibles(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CollectiblesFilterer{contract: contract}, nil
}

// bindCollectibles binds a generic wrapper to an already deployed contract.
func bindCollectibles(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CollectiblesABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Collectibles *CollectiblesRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Collectibles.Contract.CollectiblesCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Collectibles *CollectiblesRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Collectibles.Contract.CollectiblesTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Collectibles *CollectiblesRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Collectibles.Contract.CollectiblesTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Collectibles *CollectiblesCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Collectibles.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Collectibles *CollectiblesTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Collectibles.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Collectibles *CollectiblesTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Collectibles.Contract.contract.Transact(opts, method, params...)
}

// Multiply is a free data retrieval call binding the contract method 0x165c4a16.
//
// Solidity: function multiply(uint256 x, uint256 y) pure returns(uint256)
func (_Collectibles *CollectiblesCaller) Multiply(opts *bind.CallOpts, x *big.Int, y *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "multiply", x, y)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Multiply is a free data retrieval call binding the contract method 0x165c4a16.
//
// Solidity: function multiply(uint256 x, uint256 y) pure returns(uint256)
func (_Collectibles *CollectiblesSession) Multiply(x *big.Int, y *big.Int) (*big.Int, error) {
	return _Collectibles.Contract.Multiply(&_Collectibles.CallOpts, x, y)
}

// Multiply is a free data retrieval call binding the contract method 0x165c4a16.
//
// Solidity: function multiply(uint256 x, uint256 y) pure returns(uint256)
func (_Collectibles *CollectiblesCallerSession) Multiply(x *big.Int, y *big.Int) (*big.Int, error) {
	return _Collectibles.Contract.Multiply(&_Collectibles.CallOpts, x, y)
}
