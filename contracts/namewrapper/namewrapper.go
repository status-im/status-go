// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package namewrapper

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

// NamewrapperMetaData contains all meta data concerning the Namewrapper contract.
var NamewrapperMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"expiry\",\"type\":\"uint64\"}],\"name\":\"ExpiryExtended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"fuses\",\"type\":\"uint32\"}],\"name\":\"FusesSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NameUnwrapped\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"name\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"fuses\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"expiry\",\"type\":\"uint64\"}],\"name\":\"NameWrapped\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"}],\"name\":\"TransferBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"TransferSingle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"value\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"URI\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"fuseMask\",\"type\":\"uint32\"}],\"name\":\"allFusesBurned\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"accounts\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"}],\"name\":\"balanceOfBatch\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"canModifyName\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ens\",\"outputs\":[{\"internalType\":\"contractENS\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"labelhash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"expiry\",\"type\":\"uint64\"}],\"name\":\"extendExpiry\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"getApproved\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"getData\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"},{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isWrapped\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isWrapped\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"metadataService\",\"outputs\":[{\"internalType\":\"contractIMetadataService\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"names\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"ownerOf\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"label\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"wrappedOwner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"duration\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"resolver\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"ownerControlledFuses\",\"type\":\"uint16\"}],\"name\":\"registerAndWrapETH2LD\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"registrarExpiry\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"registrar\",\"outputs\":[{\"internalType\":\"contractIBaseRegistrar\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"labelHash\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"duration\",\"type\":\"uint256\"}],\"name\":\"renew\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"expires\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"amounts\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"safeBatchTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"setApprovalForAll\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"parentNode\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"labelhash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"fuses\",\"type\":\"uint32\"},{\"internalType\":\"uint64\",\"name\":\"expiry\",\"type\":\"uint64\"}],\"name\":\"setChildFuses\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"uint16\",\"name\":\"ownerControlledFuses\",\"type\":\"uint16\"}],\"name\":\"setFuses\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"newFuses\",\"type\":\"uint32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIMetadataService\",\"name\":\"_metadataService\",\"type\":\"address\"}],\"name\":\"setMetadataService\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"resolver\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"ttl\",\"type\":\"uint64\"}],\"name\":\"setRecord\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"resolver\",\"type\":\"address\"}],\"name\":\"setResolver\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"label\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"fuses\",\"type\":\"uint32\"},{\"internalType\":\"uint64\",\"name\":\"expiry\",\"type\":\"uint64\"}],\"name\":\"setSubnodeOwner\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"label\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"resolver\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"ttl\",\"type\":\"uint64\"},{\"internalType\":\"uint32\",\"name\":\"fuses\",\"type\":\"uint32\"},{\"internalType\":\"uint64\",\"name\":\"expiry\",\"type\":\"uint64\"}],\"name\":\"setSubnodeRecord\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"ttl\",\"type\":\"uint64\"}],\"name\":\"setTTL\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractINameWrapperUpgrade\",\"name\":\"_upgradeAddress\",\"type\":\"address\"}],\"name\":\"setUpgradeContract\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceID\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"node\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"label\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"unwrap\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"label\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"newRegistrant\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newController\",\"type\":\"address\"}],\"name\":\"unwrapETH2LD\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"name\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"upgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"upgradeContract\",\"outputs\":[{\"internalType\":\"contractINameWrapperUpgrade\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"uri\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"name\",\"type\":\"bytes\"},{\"internalType\":\"address\",\"name\":\"wrappedOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"resolver\",\"type\":\"address\"}],\"name\":\"wrap\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"label\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"wrappedOwner\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"ownerControlledFuses\",\"type\":\"uint16\"},{\"internalType\":\"address\",\"name\":\"resolver\",\"type\":\"address\"}],\"name\":\"wrapETH2LD\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"expires\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// NamewrapperABI is the input ABI used to generate the binding from.
// Deprecated: Use NamewrapperMetaData.ABI instead.
var NamewrapperABI = NamewrapperMetaData.ABI

// Namewrapper is an auto generated Go binding around an Ethereum contract.
type Namewrapper struct {
	NamewrapperCaller     // Read-only binding to the contract
	NamewrapperTransactor // Write-only binding to the contract
	NamewrapperFilterer   // Log filterer for contract events
}

// NamewrapperCaller is an auto generated read-only Go binding around an Ethereum contract.
type NamewrapperCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NamewrapperTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NamewrapperTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NamewrapperFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NamewrapperFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NamewrapperSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NamewrapperSession struct {
	Contract     *Namewrapper      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NamewrapperCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NamewrapperCallerSession struct {
	Contract *NamewrapperCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// NamewrapperTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NamewrapperTransactorSession struct {
	Contract     *NamewrapperTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// NamewrapperRaw is an auto generated low-level Go binding around an Ethereum contract.
type NamewrapperRaw struct {
	Contract *Namewrapper // Generic contract binding to access the raw methods on
}

// NamewrapperCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NamewrapperCallerRaw struct {
	Contract *NamewrapperCaller // Generic read-only contract binding to access the raw methods on
}

// NamewrapperTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NamewrapperTransactorRaw struct {
	Contract *NamewrapperTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNamewrapper creates a new instance of Namewrapper, bound to a specific deployed contract.
func NewNamewrapper(address common.Address, backend bind.ContractBackend) (*Namewrapper, error) {
	contract, err := bindNamewrapper(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Namewrapper{NamewrapperCaller: NamewrapperCaller{contract: contract}, NamewrapperTransactor: NamewrapperTransactor{contract: contract}, NamewrapperFilterer: NamewrapperFilterer{contract: contract}}, nil
}

// NewNamewrapperCaller creates a new read-only instance of Namewrapper, bound to a specific deployed contract.
func NewNamewrapperCaller(address common.Address, caller bind.ContractCaller) (*NamewrapperCaller, error) {
	contract, err := bindNamewrapper(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NamewrapperCaller{contract: contract}, nil
}

// NewNamewrapperTransactor creates a new write-only instance of Namewrapper, bound to a specific deployed contract.
func NewNamewrapperTransactor(address common.Address, transactor bind.ContractTransactor) (*NamewrapperTransactor, error) {
	contract, err := bindNamewrapper(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NamewrapperTransactor{contract: contract}, nil
}

// NewNamewrapperFilterer creates a new log filterer instance of Namewrapper, bound to a specific deployed contract.
func NewNamewrapperFilterer(address common.Address, filterer bind.ContractFilterer) (*NamewrapperFilterer, error) {
	contract, err := bindNamewrapper(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NamewrapperFilterer{contract: contract}, nil
}

// bindNamewrapper binds a generic wrapper to an already deployed contract.
func bindNamewrapper(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NamewrapperMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Namewrapper *NamewrapperRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Namewrapper.Contract.NamewrapperCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Namewrapper *NamewrapperRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Namewrapper.Contract.NamewrapperTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Namewrapper *NamewrapperRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Namewrapper.Contract.NamewrapperTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Namewrapper *NamewrapperCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Namewrapper.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Namewrapper *NamewrapperTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Namewrapper.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Namewrapper *NamewrapperTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Namewrapper.Contract.contract.Transact(opts, method, params...)
}

// AllFusesBurned is a free data retrieval call binding the contract method 0xadf4960a.
//
// Solidity: function allFusesBurned(bytes32 node, uint32 fuseMask) view returns(bool)
func (_Namewrapper *NamewrapperCaller) AllFusesBurned(opts *bind.CallOpts, node [32]byte, fuseMask uint32) (bool, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "allFusesBurned", node, fuseMask)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllFusesBurned is a free data retrieval call binding the contract method 0xadf4960a.
//
// Solidity: function allFusesBurned(bytes32 node, uint32 fuseMask) view returns(bool)
func (_Namewrapper *NamewrapperSession) AllFusesBurned(node [32]byte, fuseMask uint32) (bool, error) {
	return _Namewrapper.Contract.AllFusesBurned(&_Namewrapper.CallOpts, node, fuseMask)
}

// AllFusesBurned is a free data retrieval call binding the contract method 0xadf4960a.
//
// Solidity: function allFusesBurned(bytes32 node, uint32 fuseMask) view returns(bool)
func (_Namewrapper *NamewrapperCallerSession) AllFusesBurned(node [32]byte, fuseMask uint32) (bool, error) {
	return _Namewrapper.Contract.AllFusesBurned(&_Namewrapper.CallOpts, node, fuseMask)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address account, uint256 id) view returns(uint256)
func (_Namewrapper *NamewrapperCaller) BalanceOf(opts *bind.CallOpts, account common.Address, id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "balanceOf", account, id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address account, uint256 id) view returns(uint256)
func (_Namewrapper *NamewrapperSession) BalanceOf(account common.Address, id *big.Int) (*big.Int, error) {
	return _Namewrapper.Contract.BalanceOf(&_Namewrapper.CallOpts, account, id)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address account, uint256 id) view returns(uint256)
func (_Namewrapper *NamewrapperCallerSession) BalanceOf(account common.Address, id *big.Int) (*big.Int, error) {
	return _Namewrapper.Contract.BalanceOf(&_Namewrapper.CallOpts, account, id)
}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] accounts, uint256[] ids) view returns(uint256[])
func (_Namewrapper *NamewrapperCaller) BalanceOfBatch(opts *bind.CallOpts, accounts []common.Address, ids []*big.Int) ([]*big.Int, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "balanceOfBatch", accounts, ids)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] accounts, uint256[] ids) view returns(uint256[])
func (_Namewrapper *NamewrapperSession) BalanceOfBatch(accounts []common.Address, ids []*big.Int) ([]*big.Int, error) {
	return _Namewrapper.Contract.BalanceOfBatch(&_Namewrapper.CallOpts, accounts, ids)
}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] accounts, uint256[] ids) view returns(uint256[])
func (_Namewrapper *NamewrapperCallerSession) BalanceOfBatch(accounts []common.Address, ids []*big.Int) ([]*big.Int, error) {
	return _Namewrapper.Contract.BalanceOfBatch(&_Namewrapper.CallOpts, accounts, ids)
}

// CanModifyName is a free data retrieval call binding the contract method 0x41415eab.
//
// Solidity: function canModifyName(bytes32 node, address addr) view returns(bool)
func (_Namewrapper *NamewrapperCaller) CanModifyName(opts *bind.CallOpts, node [32]byte, addr common.Address) (bool, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "canModifyName", node, addr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CanModifyName is a free data retrieval call binding the contract method 0x41415eab.
//
// Solidity: function canModifyName(bytes32 node, address addr) view returns(bool)
func (_Namewrapper *NamewrapperSession) CanModifyName(node [32]byte, addr common.Address) (bool, error) {
	return _Namewrapper.Contract.CanModifyName(&_Namewrapper.CallOpts, node, addr)
}

// CanModifyName is a free data retrieval call binding the contract method 0x41415eab.
//
// Solidity: function canModifyName(bytes32 node, address addr) view returns(bool)
func (_Namewrapper *NamewrapperCallerSession) CanModifyName(node [32]byte, addr common.Address) (bool, error) {
	return _Namewrapper.Contract.CanModifyName(&_Namewrapper.CallOpts, node, addr)
}

// Ens is a free data retrieval call binding the contract method 0x3f15457f.
//
// Solidity: function ens() view returns(address)
func (_Namewrapper *NamewrapperCaller) Ens(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "ens")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Ens is a free data retrieval call binding the contract method 0x3f15457f.
//
// Solidity: function ens() view returns(address)
func (_Namewrapper *NamewrapperSession) Ens() (common.Address, error) {
	return _Namewrapper.Contract.Ens(&_Namewrapper.CallOpts)
}

// Ens is a free data retrieval call binding the contract method 0x3f15457f.
//
// Solidity: function ens() view returns(address)
func (_Namewrapper *NamewrapperCallerSession) Ens() (common.Address, error) {
	return _Namewrapper.Contract.Ens(&_Namewrapper.CallOpts)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Namewrapper *NamewrapperCaller) GetApproved(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "getApproved", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Namewrapper *NamewrapperSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _Namewrapper.Contract.GetApproved(&_Namewrapper.CallOpts, tokenId)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Namewrapper *NamewrapperCallerSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _Namewrapper.Contract.GetApproved(&_Namewrapper.CallOpts, tokenId)
}

// GetData is a free data retrieval call binding the contract method 0x0178fe3f.
//
// Solidity: function getData(uint256 id) view returns(address, uint32, uint64)
func (_Namewrapper *NamewrapperCaller) GetData(opts *bind.CallOpts, id *big.Int) (common.Address, uint32, uint64, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "getData", id)

	if err != nil {
		return *new(common.Address), *new(uint32), *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	out1 := *abi.ConvertType(out[1], new(uint32)).(*uint32)
	out2 := *abi.ConvertType(out[2], new(uint64)).(*uint64)

	return out0, out1, out2, err

}

// GetData is a free data retrieval call binding the contract method 0x0178fe3f.
//
// Solidity: function getData(uint256 id) view returns(address, uint32, uint64)
func (_Namewrapper *NamewrapperSession) GetData(id *big.Int) (common.Address, uint32, uint64, error) {
	return _Namewrapper.Contract.GetData(&_Namewrapper.CallOpts, id)
}

// GetData is a free data retrieval call binding the contract method 0x0178fe3f.
//
// Solidity: function getData(uint256 id) view returns(address, uint32, uint64)
func (_Namewrapper *NamewrapperCallerSession) GetData(id *big.Int) (common.Address, uint32, uint64, error) {
	return _Namewrapper.Contract.GetData(&_Namewrapper.CallOpts, id)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address account, address operator) view returns(bool)
func (_Namewrapper *NamewrapperCaller) IsApprovedForAll(opts *bind.CallOpts, account common.Address, operator common.Address) (bool, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "isApprovedForAll", account, operator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address account, address operator) view returns(bool)
func (_Namewrapper *NamewrapperSession) IsApprovedForAll(account common.Address, operator common.Address) (bool, error) {
	return _Namewrapper.Contract.IsApprovedForAll(&_Namewrapper.CallOpts, account, operator)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address account, address operator) view returns(bool)
func (_Namewrapper *NamewrapperCallerSession) IsApprovedForAll(account common.Address, operator common.Address) (bool, error) {
	return _Namewrapper.Contract.IsApprovedForAll(&_Namewrapper.CallOpts, account, operator)
}

// IsWrapped is a free data retrieval call binding the contract method 0xd9a50c12.
//
// Solidity: function isWrapped(bytes32 , bytes32 ) view returns(bool)
func (_Namewrapper *NamewrapperCaller) IsWrapped(opts *bind.CallOpts, arg0 [32]byte, arg1 [32]byte) (bool, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "isWrapped", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsWrapped is a free data retrieval call binding the contract method 0xd9a50c12.
//
// Solidity: function isWrapped(bytes32 , bytes32 ) view returns(bool)
func (_Namewrapper *NamewrapperSession) IsWrapped(arg0 [32]byte, arg1 [32]byte) (bool, error) {
	return _Namewrapper.Contract.IsWrapped(&_Namewrapper.CallOpts, arg0, arg1)
}

// IsWrapped is a free data retrieval call binding the contract method 0xd9a50c12.
//
// Solidity: function isWrapped(bytes32 , bytes32 ) view returns(bool)
func (_Namewrapper *NamewrapperCallerSession) IsWrapped(arg0 [32]byte, arg1 [32]byte) (bool, error) {
	return _Namewrapper.Contract.IsWrapped(&_Namewrapper.CallOpts, arg0, arg1)
}

// IsWrapped0 is a free data retrieval call binding the contract method 0xfd0cd0d9.
//
// Solidity: function isWrapped(bytes32 ) view returns(bool)
func (_Namewrapper *NamewrapperCaller) IsWrapped0(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "isWrapped0", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsWrapped0 is a free data retrieval call binding the contract method 0xfd0cd0d9.
//
// Solidity: function isWrapped(bytes32 ) view returns(bool)
func (_Namewrapper *NamewrapperSession) IsWrapped0(arg0 [32]byte) (bool, error) {
	return _Namewrapper.Contract.IsWrapped0(&_Namewrapper.CallOpts, arg0)
}

// IsWrapped0 is a free data retrieval call binding the contract method 0xfd0cd0d9.
//
// Solidity: function isWrapped(bytes32 ) view returns(bool)
func (_Namewrapper *NamewrapperCallerSession) IsWrapped0(arg0 [32]byte) (bool, error) {
	return _Namewrapper.Contract.IsWrapped0(&_Namewrapper.CallOpts, arg0)
}

// MetadataService is a free data retrieval call binding the contract method 0x53095467.
//
// Solidity: function metadataService() view returns(address)
func (_Namewrapper *NamewrapperCaller) MetadataService(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "metadataService")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MetadataService is a free data retrieval call binding the contract method 0x53095467.
//
// Solidity: function metadataService() view returns(address)
func (_Namewrapper *NamewrapperSession) MetadataService() (common.Address, error) {
	return _Namewrapper.Contract.MetadataService(&_Namewrapper.CallOpts)
}

// MetadataService is a free data retrieval call binding the contract method 0x53095467.
//
// Solidity: function metadataService() view returns(address)
func (_Namewrapper *NamewrapperCallerSession) MetadataService() (common.Address, error) {
	return _Namewrapper.Contract.MetadataService(&_Namewrapper.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Namewrapper *NamewrapperCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Namewrapper *NamewrapperSession) Name() (string, error) {
	return _Namewrapper.Contract.Name(&_Namewrapper.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Namewrapper *NamewrapperCallerSession) Name() (string, error) {
	return _Namewrapper.Contract.Name(&_Namewrapper.CallOpts)
}

// Names is a free data retrieval call binding the contract method 0x20c38e2b.
//
// Solidity: function names(bytes32 ) view returns(bytes)
func (_Namewrapper *NamewrapperCaller) Names(opts *bind.CallOpts, arg0 [32]byte) ([]byte, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "names", arg0)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Names is a free data retrieval call binding the contract method 0x20c38e2b.
//
// Solidity: function names(bytes32 ) view returns(bytes)
func (_Namewrapper *NamewrapperSession) Names(arg0 [32]byte) ([]byte, error) {
	return _Namewrapper.Contract.Names(&_Namewrapper.CallOpts, arg0)
}

// Names is a free data retrieval call binding the contract method 0x20c38e2b.
//
// Solidity: function names(bytes32 ) view returns(bytes)
func (_Namewrapper *NamewrapperCallerSession) Names(arg0 [32]byte) ([]byte, error) {
	return _Namewrapper.Contract.Names(&_Namewrapper.CallOpts, arg0)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 id) view returns(address owner)
func (_Namewrapper *NamewrapperCaller) OwnerOf(opts *bind.CallOpts, id *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "ownerOf", id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 id) view returns(address owner)
func (_Namewrapper *NamewrapperSession) OwnerOf(id *big.Int) (common.Address, error) {
	return _Namewrapper.Contract.OwnerOf(&_Namewrapper.CallOpts, id)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 id) view returns(address owner)
func (_Namewrapper *NamewrapperCallerSession) OwnerOf(id *big.Int) (common.Address, error) {
	return _Namewrapper.Contract.OwnerOf(&_Namewrapper.CallOpts, id)
}

// Registrar is a free data retrieval call binding the contract method 0x2b20e397.
//
// Solidity: function registrar() view returns(address)
func (_Namewrapper *NamewrapperCaller) Registrar(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "registrar")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Registrar is a free data retrieval call binding the contract method 0x2b20e397.
//
// Solidity: function registrar() view returns(address)
func (_Namewrapper *NamewrapperSession) Registrar() (common.Address, error) {
	return _Namewrapper.Contract.Registrar(&_Namewrapper.CallOpts)
}

// Registrar is a free data retrieval call binding the contract method 0x2b20e397.
//
// Solidity: function registrar() view returns(address)
func (_Namewrapper *NamewrapperCallerSession) Registrar() (common.Address, error) {
	return _Namewrapper.Contract.Registrar(&_Namewrapper.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceID) view returns(bool)
func (_Namewrapper *NamewrapperCaller) SupportsInterface(opts *bind.CallOpts, interfaceID [4]byte) (bool, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "supportsInterface", interfaceID)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceID) view returns(bool)
func (_Namewrapper *NamewrapperSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _Namewrapper.Contract.SupportsInterface(&_Namewrapper.CallOpts, interfaceID)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceID) view returns(bool)
func (_Namewrapper *NamewrapperCallerSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _Namewrapper.Contract.SupportsInterface(&_Namewrapper.CallOpts, interfaceID)
}

// UpgradeContract is a free data retrieval call binding the contract method 0x1f4e1504.
//
// Solidity: function upgradeContract() view returns(address)
func (_Namewrapper *NamewrapperCaller) UpgradeContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "upgradeContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UpgradeContract is a free data retrieval call binding the contract method 0x1f4e1504.
//
// Solidity: function upgradeContract() view returns(address)
func (_Namewrapper *NamewrapperSession) UpgradeContract() (common.Address, error) {
	return _Namewrapper.Contract.UpgradeContract(&_Namewrapper.CallOpts)
}

// UpgradeContract is a free data retrieval call binding the contract method 0x1f4e1504.
//
// Solidity: function upgradeContract() view returns(address)
func (_Namewrapper *NamewrapperCallerSession) UpgradeContract() (common.Address, error) {
	return _Namewrapper.Contract.UpgradeContract(&_Namewrapper.CallOpts)
}

// Uri is a free data retrieval call binding the contract method 0x0e89341c.
//
// Solidity: function uri(uint256 tokenId) view returns(string)
func (_Namewrapper *NamewrapperCaller) Uri(opts *bind.CallOpts, tokenId *big.Int) (string, error) {
	var out []interface{}
	err := _Namewrapper.contract.Call(opts, &out, "uri", tokenId)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Uri is a free data retrieval call binding the contract method 0x0e89341c.
//
// Solidity: function uri(uint256 tokenId) view returns(string)
func (_Namewrapper *NamewrapperSession) Uri(tokenId *big.Int) (string, error) {
	return _Namewrapper.Contract.Uri(&_Namewrapper.CallOpts, tokenId)
}

// Uri is a free data retrieval call binding the contract method 0x0e89341c.
//
// Solidity: function uri(uint256 tokenId) view returns(string)
func (_Namewrapper *NamewrapperCallerSession) Uri(tokenId *big.Int) (string, error) {
	return _Namewrapper.Contract.Uri(&_Namewrapper.CallOpts, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Namewrapper *NamewrapperTransactor) Approve(opts *bind.TransactOpts, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "approve", to, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Namewrapper *NamewrapperSession) Approve(to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Namewrapper.Contract.Approve(&_Namewrapper.TransactOpts, to, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Namewrapper *NamewrapperTransactorSession) Approve(to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Namewrapper.Contract.Approve(&_Namewrapper.TransactOpts, to, tokenId)
}

// ExtendExpiry is a paid mutator transaction binding the contract method 0x6e5d6ad2.
//
// Solidity: function extendExpiry(bytes32 node, bytes32 labelhash, uint64 expiry) returns(uint64)
func (_Namewrapper *NamewrapperTransactor) ExtendExpiry(opts *bind.TransactOpts, node [32]byte, labelhash [32]byte, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "extendExpiry", node, labelhash, expiry)
}

// ExtendExpiry is a paid mutator transaction binding the contract method 0x6e5d6ad2.
//
// Solidity: function extendExpiry(bytes32 node, bytes32 labelhash, uint64 expiry) returns(uint64)
func (_Namewrapper *NamewrapperSession) ExtendExpiry(node [32]byte, labelhash [32]byte, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.ExtendExpiry(&_Namewrapper.TransactOpts, node, labelhash, expiry)
}

// ExtendExpiry is a paid mutator transaction binding the contract method 0x6e5d6ad2.
//
// Solidity: function extendExpiry(bytes32 node, bytes32 labelhash, uint64 expiry) returns(uint64)
func (_Namewrapper *NamewrapperTransactorSession) ExtendExpiry(node [32]byte, labelhash [32]byte, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.ExtendExpiry(&_Namewrapper.TransactOpts, node, labelhash, expiry)
}

// RegisterAndWrapETH2LD is a paid mutator transaction binding the contract method 0xa4014982.
//
// Solidity: function registerAndWrapETH2LD(string label, address wrappedOwner, uint256 duration, address resolver, uint16 ownerControlledFuses) returns(uint256 registrarExpiry)
func (_Namewrapper *NamewrapperTransactor) RegisterAndWrapETH2LD(opts *bind.TransactOpts, label string, wrappedOwner common.Address, duration *big.Int, resolver common.Address, ownerControlledFuses uint16) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "registerAndWrapETH2LD", label, wrappedOwner, duration, resolver, ownerControlledFuses)
}

// RegisterAndWrapETH2LD is a paid mutator transaction binding the contract method 0xa4014982.
//
// Solidity: function registerAndWrapETH2LD(string label, address wrappedOwner, uint256 duration, address resolver, uint16 ownerControlledFuses) returns(uint256 registrarExpiry)
func (_Namewrapper *NamewrapperSession) RegisterAndWrapETH2LD(label string, wrappedOwner common.Address, duration *big.Int, resolver common.Address, ownerControlledFuses uint16) (*types.Transaction, error) {
	return _Namewrapper.Contract.RegisterAndWrapETH2LD(&_Namewrapper.TransactOpts, label, wrappedOwner, duration, resolver, ownerControlledFuses)
}

// RegisterAndWrapETH2LD is a paid mutator transaction binding the contract method 0xa4014982.
//
// Solidity: function registerAndWrapETH2LD(string label, address wrappedOwner, uint256 duration, address resolver, uint16 ownerControlledFuses) returns(uint256 registrarExpiry)
func (_Namewrapper *NamewrapperTransactorSession) RegisterAndWrapETH2LD(label string, wrappedOwner common.Address, duration *big.Int, resolver common.Address, ownerControlledFuses uint16) (*types.Transaction, error) {
	return _Namewrapper.Contract.RegisterAndWrapETH2LD(&_Namewrapper.TransactOpts, label, wrappedOwner, duration, resolver, ownerControlledFuses)
}

// Renew is a paid mutator transaction binding the contract method 0xc475abff.
//
// Solidity: function renew(uint256 labelHash, uint256 duration) returns(uint256 expires)
func (_Namewrapper *NamewrapperTransactor) Renew(opts *bind.TransactOpts, labelHash *big.Int, duration *big.Int) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "renew", labelHash, duration)
}

// Renew is a paid mutator transaction binding the contract method 0xc475abff.
//
// Solidity: function renew(uint256 labelHash, uint256 duration) returns(uint256 expires)
func (_Namewrapper *NamewrapperSession) Renew(labelHash *big.Int, duration *big.Int) (*types.Transaction, error) {
	return _Namewrapper.Contract.Renew(&_Namewrapper.TransactOpts, labelHash, duration)
}

// Renew is a paid mutator transaction binding the contract method 0xc475abff.
//
// Solidity: function renew(uint256 labelHash, uint256 duration) returns(uint256 expires)
func (_Namewrapper *NamewrapperTransactorSession) Renew(labelHash *big.Int, duration *big.Int) (*types.Transaction, error) {
	return _Namewrapper.Contract.Renew(&_Namewrapper.TransactOpts, labelHash, duration)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0x2eb2c2d6.
//
// Solidity: function safeBatchTransferFrom(address from, address to, uint256[] ids, uint256[] amounts, bytes data) returns()
func (_Namewrapper *NamewrapperTransactor) SafeBatchTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, ids []*big.Int, amounts []*big.Int, data []byte) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "safeBatchTransferFrom", from, to, ids, amounts, data)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0x2eb2c2d6.
//
// Solidity: function safeBatchTransferFrom(address from, address to, uint256[] ids, uint256[] amounts, bytes data) returns()
func (_Namewrapper *NamewrapperSession) SafeBatchTransferFrom(from common.Address, to common.Address, ids []*big.Int, amounts []*big.Int, data []byte) (*types.Transaction, error) {
	return _Namewrapper.Contract.SafeBatchTransferFrom(&_Namewrapper.TransactOpts, from, to, ids, amounts, data)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0x2eb2c2d6.
//
// Solidity: function safeBatchTransferFrom(address from, address to, uint256[] ids, uint256[] amounts, bytes data) returns()
func (_Namewrapper *NamewrapperTransactorSession) SafeBatchTransferFrom(from common.Address, to common.Address, ids []*big.Int, amounts []*big.Int, data []byte) (*types.Transaction, error) {
	return _Namewrapper.Contract.SafeBatchTransferFrom(&_Namewrapper.TransactOpts, from, to, ids, amounts, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 amount, bytes data) returns()
func (_Namewrapper *NamewrapperTransactor) SafeTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, id *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "safeTransferFrom", from, to, id, amount, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 amount, bytes data) returns()
func (_Namewrapper *NamewrapperSession) SafeTransferFrom(from common.Address, to common.Address, id *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Namewrapper.Contract.SafeTransferFrom(&_Namewrapper.TransactOpts, from, to, id, amount, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 amount, bytes data) returns()
func (_Namewrapper *NamewrapperTransactorSession) SafeTransferFrom(from common.Address, to common.Address, id *big.Int, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Namewrapper.Contract.SafeTransferFrom(&_Namewrapper.TransactOpts, from, to, id, amount, data)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Namewrapper *NamewrapperTransactor) SetApprovalForAll(opts *bind.TransactOpts, operator common.Address, approved bool) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setApprovalForAll", operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Namewrapper *NamewrapperSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetApprovalForAll(&_Namewrapper.TransactOpts, operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Namewrapper *NamewrapperTransactorSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetApprovalForAll(&_Namewrapper.TransactOpts, operator, approved)
}

// SetChildFuses is a paid mutator transaction binding the contract method 0x33c69ea9.
//
// Solidity: function setChildFuses(bytes32 parentNode, bytes32 labelhash, uint32 fuses, uint64 expiry) returns()
func (_Namewrapper *NamewrapperTransactor) SetChildFuses(opts *bind.TransactOpts, parentNode [32]byte, labelhash [32]byte, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setChildFuses", parentNode, labelhash, fuses, expiry)
}

// SetChildFuses is a paid mutator transaction binding the contract method 0x33c69ea9.
//
// Solidity: function setChildFuses(bytes32 parentNode, bytes32 labelhash, uint32 fuses, uint64 expiry) returns()
func (_Namewrapper *NamewrapperSession) SetChildFuses(parentNode [32]byte, labelhash [32]byte, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetChildFuses(&_Namewrapper.TransactOpts, parentNode, labelhash, fuses, expiry)
}

// SetChildFuses is a paid mutator transaction binding the contract method 0x33c69ea9.
//
// Solidity: function setChildFuses(bytes32 parentNode, bytes32 labelhash, uint32 fuses, uint64 expiry) returns()
func (_Namewrapper *NamewrapperTransactorSession) SetChildFuses(parentNode [32]byte, labelhash [32]byte, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetChildFuses(&_Namewrapper.TransactOpts, parentNode, labelhash, fuses, expiry)
}

// SetFuses is a paid mutator transaction binding the contract method 0x402906fc.
//
// Solidity: function setFuses(bytes32 node, uint16 ownerControlledFuses) returns(uint32 newFuses)
func (_Namewrapper *NamewrapperTransactor) SetFuses(opts *bind.TransactOpts, node [32]byte, ownerControlledFuses uint16) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setFuses", node, ownerControlledFuses)
}

// SetFuses is a paid mutator transaction binding the contract method 0x402906fc.
//
// Solidity: function setFuses(bytes32 node, uint16 ownerControlledFuses) returns(uint32 newFuses)
func (_Namewrapper *NamewrapperSession) SetFuses(node [32]byte, ownerControlledFuses uint16) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetFuses(&_Namewrapper.TransactOpts, node, ownerControlledFuses)
}

// SetFuses is a paid mutator transaction binding the contract method 0x402906fc.
//
// Solidity: function setFuses(bytes32 node, uint16 ownerControlledFuses) returns(uint32 newFuses)
func (_Namewrapper *NamewrapperTransactorSession) SetFuses(node [32]byte, ownerControlledFuses uint16) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetFuses(&_Namewrapper.TransactOpts, node, ownerControlledFuses)
}

// SetMetadataService is a paid mutator transaction binding the contract method 0x1534e177.
//
// Solidity: function setMetadataService(address _metadataService) returns()
func (_Namewrapper *NamewrapperTransactor) SetMetadataService(opts *bind.TransactOpts, _metadataService common.Address) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setMetadataService", _metadataService)
}

// SetMetadataService is a paid mutator transaction binding the contract method 0x1534e177.
//
// Solidity: function setMetadataService(address _metadataService) returns()
func (_Namewrapper *NamewrapperSession) SetMetadataService(_metadataService common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetMetadataService(&_Namewrapper.TransactOpts, _metadataService)
}

// SetMetadataService is a paid mutator transaction binding the contract method 0x1534e177.
//
// Solidity: function setMetadataService(address _metadataService) returns()
func (_Namewrapper *NamewrapperTransactorSession) SetMetadataService(_metadataService common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetMetadataService(&_Namewrapper.TransactOpts, _metadataService)
}

// SetRecord is a paid mutator transaction binding the contract method 0xcf408823.
//
// Solidity: function setRecord(bytes32 node, address owner, address resolver, uint64 ttl) returns()
func (_Namewrapper *NamewrapperTransactor) SetRecord(opts *bind.TransactOpts, node [32]byte, owner common.Address, resolver common.Address, ttl uint64) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setRecord", node, owner, resolver, ttl)
}

// SetRecord is a paid mutator transaction binding the contract method 0xcf408823.
//
// Solidity: function setRecord(bytes32 node, address owner, address resolver, uint64 ttl) returns()
func (_Namewrapper *NamewrapperSession) SetRecord(node [32]byte, owner common.Address, resolver common.Address, ttl uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetRecord(&_Namewrapper.TransactOpts, node, owner, resolver, ttl)
}

// SetRecord is a paid mutator transaction binding the contract method 0xcf408823.
//
// Solidity: function setRecord(bytes32 node, address owner, address resolver, uint64 ttl) returns()
func (_Namewrapper *NamewrapperTransactorSession) SetRecord(node [32]byte, owner common.Address, resolver common.Address, ttl uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetRecord(&_Namewrapper.TransactOpts, node, owner, resolver, ttl)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(bytes32 node, address resolver) returns()
func (_Namewrapper *NamewrapperTransactor) SetResolver(opts *bind.TransactOpts, node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setResolver", node, resolver)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(bytes32 node, address resolver) returns()
func (_Namewrapper *NamewrapperSession) SetResolver(node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetResolver(&_Namewrapper.TransactOpts, node, resolver)
}

// SetResolver is a paid mutator transaction binding the contract method 0x1896f70a.
//
// Solidity: function setResolver(bytes32 node, address resolver) returns()
func (_Namewrapper *NamewrapperTransactorSession) SetResolver(node [32]byte, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetResolver(&_Namewrapper.TransactOpts, node, resolver)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0xc658e086.
//
// Solidity: function setSubnodeOwner(bytes32 node, string label, address newOwner, uint32 fuses, uint64 expiry) returns(bytes32)
func (_Namewrapper *NamewrapperTransactor) SetSubnodeOwner(opts *bind.TransactOpts, node [32]byte, label string, newOwner common.Address, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setSubnodeOwner", node, label, newOwner, fuses, expiry)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0xc658e086.
//
// Solidity: function setSubnodeOwner(bytes32 node, string label, address newOwner, uint32 fuses, uint64 expiry) returns(bytes32)
func (_Namewrapper *NamewrapperSession) SetSubnodeOwner(node [32]byte, label string, newOwner common.Address, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetSubnodeOwner(&_Namewrapper.TransactOpts, node, label, newOwner, fuses, expiry)
}

// SetSubnodeOwner is a paid mutator transaction binding the contract method 0xc658e086.
//
// Solidity: function setSubnodeOwner(bytes32 node, string label, address newOwner, uint32 fuses, uint64 expiry) returns(bytes32)
func (_Namewrapper *NamewrapperTransactorSession) SetSubnodeOwner(node [32]byte, label string, newOwner common.Address, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetSubnodeOwner(&_Namewrapper.TransactOpts, node, label, newOwner, fuses, expiry)
}

// SetSubnodeRecord is a paid mutator transaction binding the contract method 0x24c1af44.
//
// Solidity: function setSubnodeRecord(bytes32 node, string label, address owner, address resolver, uint64 ttl, uint32 fuses, uint64 expiry) returns(bytes32)
func (_Namewrapper *NamewrapperTransactor) SetSubnodeRecord(opts *bind.TransactOpts, node [32]byte, label string, owner common.Address, resolver common.Address, ttl uint64, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setSubnodeRecord", node, label, owner, resolver, ttl, fuses, expiry)
}

// SetSubnodeRecord is a paid mutator transaction binding the contract method 0x24c1af44.
//
// Solidity: function setSubnodeRecord(bytes32 node, string label, address owner, address resolver, uint64 ttl, uint32 fuses, uint64 expiry) returns(bytes32)
func (_Namewrapper *NamewrapperSession) SetSubnodeRecord(node [32]byte, label string, owner common.Address, resolver common.Address, ttl uint64, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetSubnodeRecord(&_Namewrapper.TransactOpts, node, label, owner, resolver, ttl, fuses, expiry)
}

// SetSubnodeRecord is a paid mutator transaction binding the contract method 0x24c1af44.
//
// Solidity: function setSubnodeRecord(bytes32 node, string label, address owner, address resolver, uint64 ttl, uint32 fuses, uint64 expiry) returns(bytes32)
func (_Namewrapper *NamewrapperTransactorSession) SetSubnodeRecord(node [32]byte, label string, owner common.Address, resolver common.Address, ttl uint64, fuses uint32, expiry uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetSubnodeRecord(&_Namewrapper.TransactOpts, node, label, owner, resolver, ttl, fuses, expiry)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(bytes32 node, uint64 ttl) returns()
func (_Namewrapper *NamewrapperTransactor) SetTTL(opts *bind.TransactOpts, node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setTTL", node, ttl)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(bytes32 node, uint64 ttl) returns()
func (_Namewrapper *NamewrapperSession) SetTTL(node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetTTL(&_Namewrapper.TransactOpts, node, ttl)
}

// SetTTL is a paid mutator transaction binding the contract method 0x14ab9038.
//
// Solidity: function setTTL(bytes32 node, uint64 ttl) returns()
func (_Namewrapper *NamewrapperTransactorSession) SetTTL(node [32]byte, ttl uint64) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetTTL(&_Namewrapper.TransactOpts, node, ttl)
}

// SetUpgradeContract is a paid mutator transaction binding the contract method 0xb6bcad26.
//
// Solidity: function setUpgradeContract(address _upgradeAddress) returns()
func (_Namewrapper *NamewrapperTransactor) SetUpgradeContract(opts *bind.TransactOpts, _upgradeAddress common.Address) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "setUpgradeContract", _upgradeAddress)
}

// SetUpgradeContract is a paid mutator transaction binding the contract method 0xb6bcad26.
//
// Solidity: function setUpgradeContract(address _upgradeAddress) returns()
func (_Namewrapper *NamewrapperSession) SetUpgradeContract(_upgradeAddress common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetUpgradeContract(&_Namewrapper.TransactOpts, _upgradeAddress)
}

// SetUpgradeContract is a paid mutator transaction binding the contract method 0xb6bcad26.
//
// Solidity: function setUpgradeContract(address _upgradeAddress) returns()
func (_Namewrapper *NamewrapperTransactorSession) SetUpgradeContract(_upgradeAddress common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.SetUpgradeContract(&_Namewrapper.TransactOpts, _upgradeAddress)
}

// Unwrap is a paid mutator transaction binding the contract method 0xd8c9921a.
//
// Solidity: function unwrap(bytes32 node, bytes32 label, address owner) returns()
func (_Namewrapper *NamewrapperTransactor) Unwrap(opts *bind.TransactOpts, node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "unwrap", node, label, owner)
}

// Unwrap is a paid mutator transaction binding the contract method 0xd8c9921a.
//
// Solidity: function unwrap(bytes32 node, bytes32 label, address owner) returns()
func (_Namewrapper *NamewrapperSession) Unwrap(node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.Unwrap(&_Namewrapper.TransactOpts, node, label, owner)
}

// Unwrap is a paid mutator transaction binding the contract method 0xd8c9921a.
//
// Solidity: function unwrap(bytes32 node, bytes32 label, address owner) returns()
func (_Namewrapper *NamewrapperTransactorSession) Unwrap(node [32]byte, label [32]byte, owner common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.Unwrap(&_Namewrapper.TransactOpts, node, label, owner)
}

// UnwrapETH2LD is a paid mutator transaction binding the contract method 0x8b4dfa75.
//
// Solidity: function unwrapETH2LD(bytes32 label, address newRegistrant, address newController) returns()
func (_Namewrapper *NamewrapperTransactor) UnwrapETH2LD(opts *bind.TransactOpts, label [32]byte, newRegistrant common.Address, newController common.Address) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "unwrapETH2LD", label, newRegistrant, newController)
}

// UnwrapETH2LD is a paid mutator transaction binding the contract method 0x8b4dfa75.
//
// Solidity: function unwrapETH2LD(bytes32 label, address newRegistrant, address newController) returns()
func (_Namewrapper *NamewrapperSession) UnwrapETH2LD(label [32]byte, newRegistrant common.Address, newController common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.UnwrapETH2LD(&_Namewrapper.TransactOpts, label, newRegistrant, newController)
}

// UnwrapETH2LD is a paid mutator transaction binding the contract method 0x8b4dfa75.
//
// Solidity: function unwrapETH2LD(bytes32 label, address newRegistrant, address newController) returns()
func (_Namewrapper *NamewrapperTransactorSession) UnwrapETH2LD(label [32]byte, newRegistrant common.Address, newController common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.UnwrapETH2LD(&_Namewrapper.TransactOpts, label, newRegistrant, newController)
}

// Upgrade is a paid mutator transaction binding the contract method 0xc93ab3fd.
//
// Solidity: function upgrade(bytes name, bytes extraData) returns()
func (_Namewrapper *NamewrapperTransactor) Upgrade(opts *bind.TransactOpts, name []byte, extraData []byte) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "upgrade", name, extraData)
}

// Upgrade is a paid mutator transaction binding the contract method 0xc93ab3fd.
//
// Solidity: function upgrade(bytes name, bytes extraData) returns()
func (_Namewrapper *NamewrapperSession) Upgrade(name []byte, extraData []byte) (*types.Transaction, error) {
	return _Namewrapper.Contract.Upgrade(&_Namewrapper.TransactOpts, name, extraData)
}

// Upgrade is a paid mutator transaction binding the contract method 0xc93ab3fd.
//
// Solidity: function upgrade(bytes name, bytes extraData) returns()
func (_Namewrapper *NamewrapperTransactorSession) Upgrade(name []byte, extraData []byte) (*types.Transaction, error) {
	return _Namewrapper.Contract.Upgrade(&_Namewrapper.TransactOpts, name, extraData)
}

// Wrap is a paid mutator transaction binding the contract method 0xeb8ae530.
//
// Solidity: function wrap(bytes name, address wrappedOwner, address resolver) returns()
func (_Namewrapper *NamewrapperTransactor) Wrap(opts *bind.TransactOpts, name []byte, wrappedOwner common.Address, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "wrap", name, wrappedOwner, resolver)
}

// Wrap is a paid mutator transaction binding the contract method 0xeb8ae530.
//
// Solidity: function wrap(bytes name, address wrappedOwner, address resolver) returns()
func (_Namewrapper *NamewrapperSession) Wrap(name []byte, wrappedOwner common.Address, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.Wrap(&_Namewrapper.TransactOpts, name, wrappedOwner, resolver)
}

// Wrap is a paid mutator transaction binding the contract method 0xeb8ae530.
//
// Solidity: function wrap(bytes name, address wrappedOwner, address resolver) returns()
func (_Namewrapper *NamewrapperTransactorSession) Wrap(name []byte, wrappedOwner common.Address, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.Wrap(&_Namewrapper.TransactOpts, name, wrappedOwner, resolver)
}

// WrapETH2LD is a paid mutator transaction binding the contract method 0x8cf8b41e.
//
// Solidity: function wrapETH2LD(string label, address wrappedOwner, uint16 ownerControlledFuses, address resolver) returns(uint64 expires)
func (_Namewrapper *NamewrapperTransactor) WrapETH2LD(opts *bind.TransactOpts, label string, wrappedOwner common.Address, ownerControlledFuses uint16, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.contract.Transact(opts, "wrapETH2LD", label, wrappedOwner, ownerControlledFuses, resolver)
}

// WrapETH2LD is a paid mutator transaction binding the contract method 0x8cf8b41e.
//
// Solidity: function wrapETH2LD(string label, address wrappedOwner, uint16 ownerControlledFuses, address resolver) returns(uint64 expires)
func (_Namewrapper *NamewrapperSession) WrapETH2LD(label string, wrappedOwner common.Address, ownerControlledFuses uint16, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.WrapETH2LD(&_Namewrapper.TransactOpts, label, wrappedOwner, ownerControlledFuses, resolver)
}

// WrapETH2LD is a paid mutator transaction binding the contract method 0x8cf8b41e.
//
// Solidity: function wrapETH2LD(string label, address wrappedOwner, uint16 ownerControlledFuses, address resolver) returns(uint64 expires)
func (_Namewrapper *NamewrapperTransactorSession) WrapETH2LD(label string, wrappedOwner common.Address, ownerControlledFuses uint16, resolver common.Address) (*types.Transaction, error) {
	return _Namewrapper.Contract.WrapETH2LD(&_Namewrapper.TransactOpts, label, wrappedOwner, ownerControlledFuses, resolver)
}

// NamewrapperApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the Namewrapper contract.
type NamewrapperApprovalForAllIterator struct {
	Event *NamewrapperApprovalForAll // Event containing the contract specifics and raw log

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
func (it *NamewrapperApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NamewrapperApprovalForAll)
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
		it.Event = new(NamewrapperApprovalForAll)
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
func (it *NamewrapperApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NamewrapperApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NamewrapperApprovalForAll represents a ApprovalForAll event raised by the Namewrapper contract.
type NamewrapperApprovalForAll struct {
	Account  common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed account, address indexed operator, bool approved)
func (_Namewrapper *NamewrapperFilterer) FilterApprovalForAll(opts *bind.FilterOpts, account []common.Address, operator []common.Address) (*NamewrapperApprovalForAllIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Namewrapper.contract.FilterLogs(opts, "ApprovalForAll", accountRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &NamewrapperApprovalForAllIterator{contract: _Namewrapper.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed account, address indexed operator, bool approved)
func (_Namewrapper *NamewrapperFilterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *NamewrapperApprovalForAll, account []common.Address, operator []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Namewrapper.contract.WatchLogs(opts, "ApprovalForAll", accountRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NamewrapperApprovalForAll)
				if err := _Namewrapper.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
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

// ParseApprovalForAll is a log parse operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed account, address indexed operator, bool approved)
func (_Namewrapper *NamewrapperFilterer) ParseApprovalForAll(log types.Log) (*NamewrapperApprovalForAll, error) {
	event := new(NamewrapperApprovalForAll)
	if err := _Namewrapper.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NamewrapperExpiryExtendedIterator is returned from FilterExpiryExtended and is used to iterate over the raw logs and unpacked data for ExpiryExtended events raised by the Namewrapper contract.
type NamewrapperExpiryExtendedIterator struct {
	Event *NamewrapperExpiryExtended // Event containing the contract specifics and raw log

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
func (it *NamewrapperExpiryExtendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NamewrapperExpiryExtended)
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
		it.Event = new(NamewrapperExpiryExtended)
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
func (it *NamewrapperExpiryExtendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NamewrapperExpiryExtendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NamewrapperExpiryExtended represents a ExpiryExtended event raised by the Namewrapper contract.
type NamewrapperExpiryExtended struct {
	Node   [32]byte
	Expiry uint64
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterExpiryExtended is a free log retrieval operation binding the contract event 0xf675815a0817338f93a7da433f6bd5f5542f1029b11b455191ac96c7f6a9b132.
//
// Solidity: event ExpiryExtended(bytes32 indexed node, uint64 expiry)
func (_Namewrapper *NamewrapperFilterer) FilterExpiryExtended(opts *bind.FilterOpts, node [][32]byte) (*NamewrapperExpiryExtendedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _Namewrapper.contract.FilterLogs(opts, "ExpiryExtended", nodeRule)
	if err != nil {
		return nil, err
	}
	return &NamewrapperExpiryExtendedIterator{contract: _Namewrapper.contract, event: "ExpiryExtended", logs: logs, sub: sub}, nil
}

// WatchExpiryExtended is a free log subscription operation binding the contract event 0xf675815a0817338f93a7da433f6bd5f5542f1029b11b455191ac96c7f6a9b132.
//
// Solidity: event ExpiryExtended(bytes32 indexed node, uint64 expiry)
func (_Namewrapper *NamewrapperFilterer) WatchExpiryExtended(opts *bind.WatchOpts, sink chan<- *NamewrapperExpiryExtended, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _Namewrapper.contract.WatchLogs(opts, "ExpiryExtended", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NamewrapperExpiryExtended)
				if err := _Namewrapper.contract.UnpackLog(event, "ExpiryExtended", log); err != nil {
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

// ParseExpiryExtended is a log parse operation binding the contract event 0xf675815a0817338f93a7da433f6bd5f5542f1029b11b455191ac96c7f6a9b132.
//
// Solidity: event ExpiryExtended(bytes32 indexed node, uint64 expiry)
func (_Namewrapper *NamewrapperFilterer) ParseExpiryExtended(log types.Log) (*NamewrapperExpiryExtended, error) {
	event := new(NamewrapperExpiryExtended)
	if err := _Namewrapper.contract.UnpackLog(event, "ExpiryExtended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NamewrapperFusesSetIterator is returned from FilterFusesSet and is used to iterate over the raw logs and unpacked data for FusesSet events raised by the Namewrapper contract.
type NamewrapperFusesSetIterator struct {
	Event *NamewrapperFusesSet // Event containing the contract specifics and raw log

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
func (it *NamewrapperFusesSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NamewrapperFusesSet)
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
		it.Event = new(NamewrapperFusesSet)
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
func (it *NamewrapperFusesSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NamewrapperFusesSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NamewrapperFusesSet represents a FusesSet event raised by the Namewrapper contract.
type NamewrapperFusesSet struct {
	Node  [32]byte
	Fuses uint32
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterFusesSet is a free log retrieval operation binding the contract event 0x39873f00c80f4f94b7bd1594aebcf650f003545b74824d57ddf4939e3ff3a34b.
//
// Solidity: event FusesSet(bytes32 indexed node, uint32 fuses)
func (_Namewrapper *NamewrapperFilterer) FilterFusesSet(opts *bind.FilterOpts, node [][32]byte) (*NamewrapperFusesSetIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _Namewrapper.contract.FilterLogs(opts, "FusesSet", nodeRule)
	if err != nil {
		return nil, err
	}
	return &NamewrapperFusesSetIterator{contract: _Namewrapper.contract, event: "FusesSet", logs: logs, sub: sub}, nil
}

// WatchFusesSet is a free log subscription operation binding the contract event 0x39873f00c80f4f94b7bd1594aebcf650f003545b74824d57ddf4939e3ff3a34b.
//
// Solidity: event FusesSet(bytes32 indexed node, uint32 fuses)
func (_Namewrapper *NamewrapperFilterer) WatchFusesSet(opts *bind.WatchOpts, sink chan<- *NamewrapperFusesSet, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _Namewrapper.contract.WatchLogs(opts, "FusesSet", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NamewrapperFusesSet)
				if err := _Namewrapper.contract.UnpackLog(event, "FusesSet", log); err != nil {
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

// ParseFusesSet is a log parse operation binding the contract event 0x39873f00c80f4f94b7bd1594aebcf650f003545b74824d57ddf4939e3ff3a34b.
//
// Solidity: event FusesSet(bytes32 indexed node, uint32 fuses)
func (_Namewrapper *NamewrapperFilterer) ParseFusesSet(log types.Log) (*NamewrapperFusesSet, error) {
	event := new(NamewrapperFusesSet)
	if err := _Namewrapper.contract.UnpackLog(event, "FusesSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NamewrapperNameUnwrappedIterator is returned from FilterNameUnwrapped and is used to iterate over the raw logs and unpacked data for NameUnwrapped events raised by the Namewrapper contract.
type NamewrapperNameUnwrappedIterator struct {
	Event *NamewrapperNameUnwrapped // Event containing the contract specifics and raw log

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
func (it *NamewrapperNameUnwrappedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NamewrapperNameUnwrapped)
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
		it.Event = new(NamewrapperNameUnwrapped)
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
func (it *NamewrapperNameUnwrappedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NamewrapperNameUnwrappedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NamewrapperNameUnwrapped represents a NameUnwrapped event raised by the Namewrapper contract.
type NamewrapperNameUnwrapped struct {
	Node  [32]byte
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterNameUnwrapped is a free log retrieval operation binding the contract event 0xee2ba1195c65bcf218a83d874335c6bf9d9067b4c672f3c3bf16cf40de7586c4.
//
// Solidity: event NameUnwrapped(bytes32 indexed node, address owner)
func (_Namewrapper *NamewrapperFilterer) FilterNameUnwrapped(opts *bind.FilterOpts, node [][32]byte) (*NamewrapperNameUnwrappedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _Namewrapper.contract.FilterLogs(opts, "NameUnwrapped", nodeRule)
	if err != nil {
		return nil, err
	}
	return &NamewrapperNameUnwrappedIterator{contract: _Namewrapper.contract, event: "NameUnwrapped", logs: logs, sub: sub}, nil
}

// WatchNameUnwrapped is a free log subscription operation binding the contract event 0xee2ba1195c65bcf218a83d874335c6bf9d9067b4c672f3c3bf16cf40de7586c4.
//
// Solidity: event NameUnwrapped(bytes32 indexed node, address owner)
func (_Namewrapper *NamewrapperFilterer) WatchNameUnwrapped(opts *bind.WatchOpts, sink chan<- *NamewrapperNameUnwrapped, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _Namewrapper.contract.WatchLogs(opts, "NameUnwrapped", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NamewrapperNameUnwrapped)
				if err := _Namewrapper.contract.UnpackLog(event, "NameUnwrapped", log); err != nil {
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

// ParseNameUnwrapped is a log parse operation binding the contract event 0xee2ba1195c65bcf218a83d874335c6bf9d9067b4c672f3c3bf16cf40de7586c4.
//
// Solidity: event NameUnwrapped(bytes32 indexed node, address owner)
func (_Namewrapper *NamewrapperFilterer) ParseNameUnwrapped(log types.Log) (*NamewrapperNameUnwrapped, error) {
	event := new(NamewrapperNameUnwrapped)
	if err := _Namewrapper.contract.UnpackLog(event, "NameUnwrapped", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NamewrapperNameWrappedIterator is returned from FilterNameWrapped and is used to iterate over the raw logs and unpacked data for NameWrapped events raised by the Namewrapper contract.
type NamewrapperNameWrappedIterator struct {
	Event *NamewrapperNameWrapped // Event containing the contract specifics and raw log

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
func (it *NamewrapperNameWrappedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NamewrapperNameWrapped)
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
		it.Event = new(NamewrapperNameWrapped)
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
func (it *NamewrapperNameWrappedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NamewrapperNameWrappedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NamewrapperNameWrapped represents a NameWrapped event raised by the Namewrapper contract.
type NamewrapperNameWrapped struct {
	Node   [32]byte
	Name   []byte
	Owner  common.Address
	Fuses  uint32
	Expiry uint64
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterNameWrapped is a free log retrieval operation binding the contract event 0x8ce7013e8abebc55c3890a68f5a27c67c3f7efa64e584de5fb22363c606fd340.
//
// Solidity: event NameWrapped(bytes32 indexed node, bytes name, address owner, uint32 fuses, uint64 expiry)
func (_Namewrapper *NamewrapperFilterer) FilterNameWrapped(opts *bind.FilterOpts, node [][32]byte) (*NamewrapperNameWrappedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _Namewrapper.contract.FilterLogs(opts, "NameWrapped", nodeRule)
	if err != nil {
		return nil, err
	}
	return &NamewrapperNameWrappedIterator{contract: _Namewrapper.contract, event: "NameWrapped", logs: logs, sub: sub}, nil
}

// WatchNameWrapped is a free log subscription operation binding the contract event 0x8ce7013e8abebc55c3890a68f5a27c67c3f7efa64e584de5fb22363c606fd340.
//
// Solidity: event NameWrapped(bytes32 indexed node, bytes name, address owner, uint32 fuses, uint64 expiry)
func (_Namewrapper *NamewrapperFilterer) WatchNameWrapped(opts *bind.WatchOpts, sink chan<- *NamewrapperNameWrapped, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _Namewrapper.contract.WatchLogs(opts, "NameWrapped", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NamewrapperNameWrapped)
				if err := _Namewrapper.contract.UnpackLog(event, "NameWrapped", log); err != nil {
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

// ParseNameWrapped is a log parse operation binding the contract event 0x8ce7013e8abebc55c3890a68f5a27c67c3f7efa64e584de5fb22363c606fd340.
//
// Solidity: event NameWrapped(bytes32 indexed node, bytes name, address owner, uint32 fuses, uint64 expiry)
func (_Namewrapper *NamewrapperFilterer) ParseNameWrapped(log types.Log) (*NamewrapperNameWrapped, error) {
	event := new(NamewrapperNameWrapped)
	if err := _Namewrapper.contract.UnpackLog(event, "NameWrapped", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NamewrapperTransferBatchIterator is returned from FilterTransferBatch and is used to iterate over the raw logs and unpacked data for TransferBatch events raised by the Namewrapper contract.
type NamewrapperTransferBatchIterator struct {
	Event *NamewrapperTransferBatch // Event containing the contract specifics and raw log

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
func (it *NamewrapperTransferBatchIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NamewrapperTransferBatch)
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
		it.Event = new(NamewrapperTransferBatch)
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
func (it *NamewrapperTransferBatchIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NamewrapperTransferBatchIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NamewrapperTransferBatch represents a TransferBatch event raised by the Namewrapper contract.
type NamewrapperTransferBatch struct {
	Operator common.Address
	From     common.Address
	To       common.Address
	Ids      []*big.Int
	Values   []*big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTransferBatch is a free log retrieval operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
func (_Namewrapper *NamewrapperFilterer) FilterTransferBatch(opts *bind.FilterOpts, operator []common.Address, from []common.Address, to []common.Address) (*NamewrapperTransferBatchIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Namewrapper.contract.FilterLogs(opts, "TransferBatch", operatorRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &NamewrapperTransferBatchIterator{contract: _Namewrapper.contract, event: "TransferBatch", logs: logs, sub: sub}, nil
}

// WatchTransferBatch is a free log subscription operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
func (_Namewrapper *NamewrapperFilterer) WatchTransferBatch(opts *bind.WatchOpts, sink chan<- *NamewrapperTransferBatch, operator []common.Address, from []common.Address, to []common.Address) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Namewrapper.contract.WatchLogs(opts, "TransferBatch", operatorRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NamewrapperTransferBatch)
				if err := _Namewrapper.contract.UnpackLog(event, "TransferBatch", log); err != nil {
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

// ParseTransferBatch is a log parse operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
func (_Namewrapper *NamewrapperFilterer) ParseTransferBatch(log types.Log) (*NamewrapperTransferBatch, error) {
	event := new(NamewrapperTransferBatch)
	if err := _Namewrapper.contract.UnpackLog(event, "TransferBatch", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NamewrapperTransferSingleIterator is returned from FilterTransferSingle and is used to iterate over the raw logs and unpacked data for TransferSingle events raised by the Namewrapper contract.
type NamewrapperTransferSingleIterator struct {
	Event *NamewrapperTransferSingle // Event containing the contract specifics and raw log

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
func (it *NamewrapperTransferSingleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NamewrapperTransferSingle)
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
		it.Event = new(NamewrapperTransferSingle)
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
func (it *NamewrapperTransferSingleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NamewrapperTransferSingleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NamewrapperTransferSingle represents a TransferSingle event raised by the Namewrapper contract.
type NamewrapperTransferSingle struct {
	Operator common.Address
	From     common.Address
	To       common.Address
	Id       *big.Int
	Value    *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTransferSingle is a free log retrieval operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
func (_Namewrapper *NamewrapperFilterer) FilterTransferSingle(opts *bind.FilterOpts, operator []common.Address, from []common.Address, to []common.Address) (*NamewrapperTransferSingleIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Namewrapper.contract.FilterLogs(opts, "TransferSingle", operatorRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &NamewrapperTransferSingleIterator{contract: _Namewrapper.contract, event: "TransferSingle", logs: logs, sub: sub}, nil
}

// WatchTransferSingle is a free log subscription operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
func (_Namewrapper *NamewrapperFilterer) WatchTransferSingle(opts *bind.WatchOpts, sink chan<- *NamewrapperTransferSingle, operator []common.Address, from []common.Address, to []common.Address) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Namewrapper.contract.WatchLogs(opts, "TransferSingle", operatorRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NamewrapperTransferSingle)
				if err := _Namewrapper.contract.UnpackLog(event, "TransferSingle", log); err != nil {
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

// ParseTransferSingle is a log parse operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
func (_Namewrapper *NamewrapperFilterer) ParseTransferSingle(log types.Log) (*NamewrapperTransferSingle, error) {
	event := new(NamewrapperTransferSingle)
	if err := _Namewrapper.contract.UnpackLog(event, "TransferSingle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NamewrapperURIIterator is returned from FilterURI and is used to iterate over the raw logs and unpacked data for URI events raised by the Namewrapper contract.
type NamewrapperURIIterator struct {
	Event *NamewrapperURI // Event containing the contract specifics and raw log

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
func (it *NamewrapperURIIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NamewrapperURI)
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
		it.Event = new(NamewrapperURI)
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
func (it *NamewrapperURIIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NamewrapperURIIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NamewrapperURI represents a URI event raised by the Namewrapper contract.
type NamewrapperURI struct {
	Value string
	Id    *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterURI is a free log retrieval operation binding the contract event 0x6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b.
//
// Solidity: event URI(string value, uint256 indexed id)
func (_Namewrapper *NamewrapperFilterer) FilterURI(opts *bind.FilterOpts, id []*big.Int) (*NamewrapperURIIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Namewrapper.contract.FilterLogs(opts, "URI", idRule)
	if err != nil {
		return nil, err
	}
	return &NamewrapperURIIterator{contract: _Namewrapper.contract, event: "URI", logs: logs, sub: sub}, nil
}

// WatchURI is a free log subscription operation binding the contract event 0x6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b.
//
// Solidity: event URI(string value, uint256 indexed id)
func (_Namewrapper *NamewrapperFilterer) WatchURI(opts *bind.WatchOpts, sink chan<- *NamewrapperURI, id []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Namewrapper.contract.WatchLogs(opts, "URI", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NamewrapperURI)
				if err := _Namewrapper.contract.UnpackLog(event, "URI", log); err != nil {
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

// ParseURI is a log parse operation binding the contract event 0x6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b.
//
// Solidity: event URI(string value, uint256 indexed id)
func (_Namewrapper *NamewrapperFilterer) ParseURI(log types.Log) (*NamewrapperURI, error) {
	event := new(NamewrapperURI)
	if err := _Namewrapper.contract.UnpackLog(event, "URI", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
