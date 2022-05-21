// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package directory

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

// DirectoryABI is the input ABI used to generate the binding from.
const DirectoryABI = "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"community\",\"type\":\"bytes\"}],\"name\":\"addCommunity\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"communities\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCommunities\",\"outputs\":[{\"internalType\":\"bytes[]\",\"name\":\"\",\"type\":\"bytes[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"community\",\"type\":\"bytes\"}],\"name\":\"isCommunityInDirectory\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"community\",\"type\":\"bytes\"}],\"name\":\"removeCommunity\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// DirectoryFuncSigs maps the 4-byte function signature to its string representation.
var DirectoryFuncSigs = map[string]string{
	"74837935": "addCommunity(bytes)",
	"e590b56a": "communities(uint256)",
	"c251b565": "getCommunities()",
	"b3dbb52a": "isCommunityInDirectory(bytes)",
	"3c01b93c": "removeCommunity(bytes)",
}

// DirectoryBin is the compiled bytecode used for deploying new contracts.
var DirectoryBin = "0x608060405234801561001057600080fd5b5061033b806100206000396000f3fe608060405234801561001057600080fd5b50600436106100575760003560e01c80633c01b93c1461005c578063748379351461005c578063b3dbb52a14610070578063c251b5651461009b578063e590b56a146100aa575b600080fd5b61006e61006a366004610176565b5050565b005b61008661007e366004610176565b600092915050565b60405190151581526020015b60405180910390f35b60606040516100929190610235565b6100bd6100b8366004610297565b6100ca565b60405161009291906102b0565b600081815481106100da57600080fd5b9060005260206000200160009150905080546100f5906102ca565b80601f0160208091040260200160405190810160405280929190818152602001828054610121906102ca565b801561016e5780601f106101435761010080835404028352916020019161016e565b820191906000526020600020905b81548152906001019060200180831161015157829003601f168201915b505050505081565b6000806020838503121561018957600080fd5b823567ffffffffffffffff808211156101a157600080fd5b818501915085601f8301126101b557600080fd5b8135818111156101c457600080fd5b8660208285010111156101d657600080fd5b60209290920196919550909350505050565b6000815180845260005b8181101561020e576020818501810151868301820152016101f2565b81811115610220576000602083870101525b50601f01601f19169290920160200192915050565b6000602080830181845280855180835260408601915060408160051b870101925083870160005b8281101561028a57603f198886030184526102788583516101e8565b9450928501929085019060010161025c565b5092979650505050505050565b6000602082840312156102a957600080fd5b5035919050565b6020815260006102c360208301846101e8565b9392505050565b600181811c908216806102de57607f821691505b602082108114156102ff57634e487b7160e01b600052602260045260246000fd5b5091905056fea2646970667358221220f39ece68bf28d27ae1776133faa6ead8c1a3f73d975864e0cf6295808b53284664736f6c634300080b0033"

// DeployDirectory deploys a new Ethereum contract, binding an instance of Directory to it.
func DeployDirectory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Directory, error) {
	parsed, err := abi.JSON(strings.NewReader(DirectoryABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(DirectoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Directory{DirectoryCaller: DirectoryCaller{contract: contract}, DirectoryTransactor: DirectoryTransactor{contract: contract}, DirectoryFilterer: DirectoryFilterer{contract: contract}}, nil
}

// Directory is an auto generated Go binding around an Ethereum contract.
type Directory struct {
	DirectoryCaller     // Read-only binding to the contract
	DirectoryTransactor // Write-only binding to the contract
	DirectoryFilterer   // Log filterer for contract events
}

// DirectoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type DirectoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DirectoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DirectoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DirectoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DirectoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DirectorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DirectorySession struct {
	Contract     *Directory        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DirectoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DirectoryCallerSession struct {
	Contract *DirectoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// DirectoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DirectoryTransactorSession struct {
	Contract     *DirectoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// DirectoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type DirectoryRaw struct {
	Contract *Directory // Generic contract binding to access the raw methods on
}

// DirectoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DirectoryCallerRaw struct {
	Contract *DirectoryCaller // Generic read-only contract binding to access the raw methods on
}

// DirectoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DirectoryTransactorRaw struct {
	Contract *DirectoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDirectory creates a new instance of Directory, bound to a specific deployed contract.
func NewDirectory(address common.Address, backend bind.ContractBackend) (*Directory, error) {
	contract, err := bindDirectory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Directory{DirectoryCaller: DirectoryCaller{contract: contract}, DirectoryTransactor: DirectoryTransactor{contract: contract}, DirectoryFilterer: DirectoryFilterer{contract: contract}}, nil
}

// NewDirectoryCaller creates a new read-only instance of Directory, bound to a specific deployed contract.
func NewDirectoryCaller(address common.Address, caller bind.ContractCaller) (*DirectoryCaller, error) {
	contract, err := bindDirectory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DirectoryCaller{contract: contract}, nil
}

// NewDirectoryTransactor creates a new write-only instance of Directory, bound to a specific deployed contract.
func NewDirectoryTransactor(address common.Address, transactor bind.ContractTransactor) (*DirectoryTransactor, error) {
	contract, err := bindDirectory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DirectoryTransactor{contract: contract}, nil
}

// NewDirectoryFilterer creates a new log filterer instance of Directory, bound to a specific deployed contract.
func NewDirectoryFilterer(address common.Address, filterer bind.ContractFilterer) (*DirectoryFilterer, error) {
	contract, err := bindDirectory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DirectoryFilterer{contract: contract}, nil
}

// bindDirectory binds a generic wrapper to an already deployed contract.
func bindDirectory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DirectoryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Directory *DirectoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Directory.Contract.DirectoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Directory *DirectoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Directory.Contract.DirectoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Directory *DirectoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Directory.Contract.DirectoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Directory *DirectoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Directory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Directory *DirectoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Directory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Directory *DirectoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Directory.Contract.contract.Transact(opts, method, params...)
}

// Communities is a free data retrieval call binding the contract method 0xe590b56a.
//
// Solidity: function communities(uint256 ) view returns(bytes)
func (_Directory *DirectoryCaller) Communities(opts *bind.CallOpts, arg0 *big.Int) ([]byte, error) {
	var out []interface{}
	err := _Directory.contract.Call(opts, &out, "communities", arg0)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Communities is a free data retrieval call binding the contract method 0xe590b56a.
//
// Solidity: function communities(uint256 ) view returns(bytes)
func (_Directory *DirectorySession) Communities(arg0 *big.Int) ([]byte, error) {
	return _Directory.Contract.Communities(&_Directory.CallOpts, arg0)
}

// Communities is a free data retrieval call binding the contract method 0xe590b56a.
//
// Solidity: function communities(uint256 ) view returns(bytes)
func (_Directory *DirectoryCallerSession) Communities(arg0 *big.Int) ([]byte, error) {
	return _Directory.Contract.Communities(&_Directory.CallOpts, arg0)
}

// GetCommunities is a free data retrieval call binding the contract method 0xc251b565.
//
// Solidity: function getCommunities() view returns(bytes[])
func (_Directory *DirectoryCaller) GetCommunities(opts *bind.CallOpts) ([][]byte, error) {
	var out []interface{}
	err := _Directory.contract.Call(opts, &out, "getCommunities")

	if err != nil {
		return *new([][]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][]byte)).(*[][]byte)

	return out0, err

}

// GetCommunities is a free data retrieval call binding the contract method 0xc251b565.
//
// Solidity: function getCommunities() view returns(bytes[])
func (_Directory *DirectorySession) GetCommunities() ([][]byte, error) {
	return _Directory.Contract.GetCommunities(&_Directory.CallOpts)
}

// GetCommunities is a free data retrieval call binding the contract method 0xc251b565.
//
// Solidity: function getCommunities() view returns(bytes[])
func (_Directory *DirectoryCallerSession) GetCommunities() ([][]byte, error) {
	return _Directory.Contract.GetCommunities(&_Directory.CallOpts)
}

// IsCommunityInDirectory is a free data retrieval call binding the contract method 0xb3dbb52a.
//
// Solidity: function isCommunityInDirectory(bytes community) view returns(bool)
func (_Directory *DirectoryCaller) IsCommunityInDirectory(opts *bind.CallOpts, community []byte) (bool, error) {
	var out []interface{}
	err := _Directory.contract.Call(opts, &out, "isCommunityInDirectory", community)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsCommunityInDirectory is a free data retrieval call binding the contract method 0xb3dbb52a.
//
// Solidity: function isCommunityInDirectory(bytes community) view returns(bool)
func (_Directory *DirectorySession) IsCommunityInDirectory(community []byte) (bool, error) {
	return _Directory.Contract.IsCommunityInDirectory(&_Directory.CallOpts, community)
}

// IsCommunityInDirectory is a free data retrieval call binding the contract method 0xb3dbb52a.
//
// Solidity: function isCommunityInDirectory(bytes community) view returns(bool)
func (_Directory *DirectoryCallerSession) IsCommunityInDirectory(community []byte) (bool, error) {
	return _Directory.Contract.IsCommunityInDirectory(&_Directory.CallOpts, community)
}

// AddCommunity is a paid mutator transaction binding the contract method 0x74837935.
//
// Solidity: function addCommunity(bytes community) returns()
func (_Directory *DirectoryTransactor) AddCommunity(opts *bind.TransactOpts, community []byte) (*types.Transaction, error) {
	return _Directory.contract.Transact(opts, "addCommunity", community)
}

// AddCommunity is a paid mutator transaction binding the contract method 0x74837935.
//
// Solidity: function addCommunity(bytes community) returns()
func (_Directory *DirectorySession) AddCommunity(community []byte) (*types.Transaction, error) {
	return _Directory.Contract.AddCommunity(&_Directory.TransactOpts, community)
}

// AddCommunity is a paid mutator transaction binding the contract method 0x74837935.
//
// Solidity: function addCommunity(bytes community) returns()
func (_Directory *DirectoryTransactorSession) AddCommunity(community []byte) (*types.Transaction, error) {
	return _Directory.Contract.AddCommunity(&_Directory.TransactOpts, community)
}

// RemoveCommunity is a paid mutator transaction binding the contract method 0x3c01b93c.
//
// Solidity: function removeCommunity(bytes community) returns()
func (_Directory *DirectoryTransactor) RemoveCommunity(opts *bind.TransactOpts, community []byte) (*types.Transaction, error) {
	return _Directory.contract.Transact(opts, "removeCommunity", community)
}

// RemoveCommunity is a paid mutator transaction binding the contract method 0x3c01b93c.
//
// Solidity: function removeCommunity(bytes community) returns()
func (_Directory *DirectorySession) RemoveCommunity(community []byte) (*types.Transaction, error) {
	return _Directory.Contract.RemoveCommunity(&_Directory.TransactOpts, community)
}

// RemoveCommunity is a paid mutator transaction binding the contract method 0x3c01b93c.
//
// Solidity: function removeCommunity(bytes community) returns()
func (_Directory *DirectoryTransactorSession) RemoveCommunity(community []byte) (*types.Transaction, error) {
	return _Directory.Contract.RemoveCommunity(&_Directory.TransactOpts, community)
}
