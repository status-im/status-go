// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package cryptokitties

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

// CryptoKittiesMetaData contains all meta data concerning the CryptoKitties contract.
var CryptoKittiesMetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[{\"name\":\"_interfaceID\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"cfoAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_tokenId\",\"type\":\"uint256\"},{\"name\":\"_preferredTransport\",\"type\":\"string\"}],\"name\":\"tokenMetadata\",\"outputs\":[{\"name\":\"infoUrl\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"promoCreatedCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_tokenId\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"ceoAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GEN0_STARTING_PRICE\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"setSiringAuctionAddress\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"pregnantKitties\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_kittyId\",\"type\":\"uint256\"}],\"name\":\"isPregnant\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GEN0_AUCTION_DURATION\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"siringAuction\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_tokenId\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"setGeneScienceAddress\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newCEO\",\"type\":\"address\"}],\"name\":\"setCEO\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newCOO\",\"type\":\"address\"}],\"name\":\"setCOO\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_kittyId\",\"type\":\"uint256\"},{\"name\":\"_startingPrice\",\"type\":\"uint256\"},{\"name\":\"_endingPrice\",\"type\":\"uint256\"},{\"name\":\"_duration\",\"type\":\"uint256\"}],\"name\":\"createSaleAuction\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sireAllowedToAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_matronId\",\"type\":\"uint256\"},{\"name\":\"_sireId\",\"type\":\"uint256\"}],\"name\":\"canBreedWith\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"kittyIndexToApproved\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_kittyId\",\"type\":\"uint256\"},{\"name\":\"_startingPrice\",\"type\":\"uint256\"},{\"name\":\"_endingPrice\",\"type\":\"uint256\"},{\"name\":\"_duration\",\"type\":\"uint256\"}],\"name\":\"createSiringAuction\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"setAutoBirthFee\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_addr\",\"type\":\"address\"},{\"name\":\"_sireId\",\"type\":\"uint256\"}],\"name\":\"approveSiring\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newCFO\",\"type\":\"address\"}],\"name\":\"setCFO\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_genes\",\"type\":\"uint256\"},{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"createPromoKitty\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"secs\",\"type\":\"uint256\"}],\"name\":\"setSecondsPerBlock\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"withdrawBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_tokenId\",\"type\":\"uint256\"}],\"name\":\"ownerOf\",\"outputs\":[{\"name\":\"owner\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GEN0_CREATION_LIMIT\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"newContractAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"setSaleAuctionAddress\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"count\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_v2Address\",\"type\":\"address\"}],\"name\":\"setNewAddress\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"secondsPerBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"tokensOfOwner\",\"outputs\":[{\"name\":\"ownerTokens\",\"type\":\"uint256[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_matronId\",\"type\":\"uint256\"}],\"name\":\"giveBirth\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"withdrawAuctionBalances\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"cooldowns\",\"outputs\":[{\"name\":\"\",\"type\":\"uint32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"kittyIndexToOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_tokenId\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"cooAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"autoBirthFee\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"erc721Metadata\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_genes\",\"type\":\"uint256\"}],\"name\":\"createGen0Auction\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_kittyId\",\"type\":\"uint256\"}],\"name\":\"isReadyToBreed\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"PROMO_CREATION_LIMIT\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_contractAddress\",\"type\":\"address\"}],\"name\":\"setMetadataAddress\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"saleAuction\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_id\",\"type\":\"uint256\"}],\"name\":\"getKitty\",\"outputs\":[{\"name\":\"isGestating\",\"type\":\"bool\"},{\"name\":\"isReady\",\"type\":\"bool\"},{\"name\":\"cooldownIndex\",\"type\":\"uint256\"},{\"name\":\"nextActionAt\",\"type\":\"uint256\"},{\"name\":\"siringWithId\",\"type\":\"uint256\"},{\"name\":\"birthTime\",\"type\":\"uint256\"},{\"name\":\"matronId\",\"type\":\"uint256\"},{\"name\":\"sireId\",\"type\":\"uint256\"},{\"name\":\"generation\",\"type\":\"uint256\"},{\"name\":\"genes\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_sireId\",\"type\":\"uint256\"},{\"name\":\"_matronId\",\"type\":\"uint256\"}],\"name\":\"bidOnSiringAuction\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"gen0CreatedCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"geneScience\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_matronId\",\"type\":\"uint256\"},{\"name\":\"_sireId\",\"type\":\"uint256\"}],\"name\":\"breedWithAuto\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"matronId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"sireId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"cooldownEndBlock\",\"type\":\"uint256\"}],\"name\":\"Pregnant\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"approved\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"kittyId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"matronId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"sireId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"genes\",\"type\":\"uint256\"}],\"name\":\"Birth\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"newContract\",\"type\":\"address\"}],\"name\":\"ContractUpgrade\",\"type\":\"event\"}]",
}

// CryptoKittiesABI is the input ABI used to generate the binding from.
// Deprecated: Use CryptoKittiesMetaData.ABI instead.
var CryptoKittiesABI = CryptoKittiesMetaData.ABI

// CryptoKitties is an auto generated Go binding around an Ethereum contract.
type CryptoKitties struct {
	CryptoKittiesCaller     // Read-only binding to the contract
	CryptoKittiesTransactor // Write-only binding to the contract
	CryptoKittiesFilterer   // Log filterer for contract events
}

// CryptoKittiesCaller is an auto generated read-only Go binding around an Ethereum contract.
type CryptoKittiesCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptoKittiesTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CryptoKittiesTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptoKittiesFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CryptoKittiesFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptoKittiesSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CryptoKittiesSession struct {
	Contract     *CryptoKitties    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CryptoKittiesCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CryptoKittiesCallerSession struct {
	Contract *CryptoKittiesCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// CryptoKittiesTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CryptoKittiesTransactorSession struct {
	Contract     *CryptoKittiesTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// CryptoKittiesRaw is an auto generated low-level Go binding around an Ethereum contract.
type CryptoKittiesRaw struct {
	Contract *CryptoKitties // Generic contract binding to access the raw methods on
}

// CryptoKittiesCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CryptoKittiesCallerRaw struct {
	Contract *CryptoKittiesCaller // Generic read-only contract binding to access the raw methods on
}

// CryptoKittiesTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CryptoKittiesTransactorRaw struct {
	Contract *CryptoKittiesTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCryptoKitties creates a new instance of CryptoKitties, bound to a specific deployed contract.
func NewCryptoKitties(address common.Address, backend bind.ContractBackend) (*CryptoKitties, error) {
	contract, err := bindCryptoKitties(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CryptoKitties{CryptoKittiesCaller: CryptoKittiesCaller{contract: contract}, CryptoKittiesTransactor: CryptoKittiesTransactor{contract: contract}, CryptoKittiesFilterer: CryptoKittiesFilterer{contract: contract}}, nil
}

// NewCryptoKittiesCaller creates a new read-only instance of CryptoKitties, bound to a specific deployed contract.
func NewCryptoKittiesCaller(address common.Address, caller bind.ContractCaller) (*CryptoKittiesCaller, error) {
	contract, err := bindCryptoKitties(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CryptoKittiesCaller{contract: contract}, nil
}

// NewCryptoKittiesTransactor creates a new write-only instance of CryptoKitties, bound to a specific deployed contract.
func NewCryptoKittiesTransactor(address common.Address, transactor bind.ContractTransactor) (*CryptoKittiesTransactor, error) {
	contract, err := bindCryptoKitties(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CryptoKittiesTransactor{contract: contract}, nil
}

// NewCryptoKittiesFilterer creates a new log filterer instance of CryptoKitties, bound to a specific deployed contract.
func NewCryptoKittiesFilterer(address common.Address, filterer bind.ContractFilterer) (*CryptoKittiesFilterer, error) {
	contract, err := bindCryptoKitties(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CryptoKittiesFilterer{contract: contract}, nil
}

// bindCryptoKitties binds a generic wrapper to an already deployed contract.
func bindCryptoKitties(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CryptoKittiesMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CryptoKitties *CryptoKittiesRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CryptoKitties.Contract.CryptoKittiesCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CryptoKitties *CryptoKittiesRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CryptoKittiesTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CryptoKitties *CryptoKittiesRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CryptoKittiesTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CryptoKitties *CryptoKittiesCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CryptoKitties.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CryptoKitties *CryptoKittiesTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoKitties.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CryptoKitties *CryptoKittiesTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CryptoKitties.Contract.contract.Transact(opts, method, params...)
}

// GEN0AUCTIONDURATION is a free data retrieval call binding the contract method 0x19c2f201.
//
// Solidity: function GEN0_AUCTION_DURATION() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) GEN0AUCTIONDURATION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "GEN0_AUCTION_DURATION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GEN0AUCTIONDURATION is a free data retrieval call binding the contract method 0x19c2f201.
//
// Solidity: function GEN0_AUCTION_DURATION() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) GEN0AUCTIONDURATION() (*big.Int, error) {
	return _CryptoKitties.Contract.GEN0AUCTIONDURATION(&_CryptoKitties.CallOpts)
}

// GEN0AUCTIONDURATION is a free data retrieval call binding the contract method 0x19c2f201.
//
// Solidity: function GEN0_AUCTION_DURATION() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) GEN0AUCTIONDURATION() (*big.Int, error) {
	return _CryptoKitties.Contract.GEN0AUCTIONDURATION(&_CryptoKitties.CallOpts)
}

// GEN0CREATIONLIMIT is a free data retrieval call binding the contract method 0x680eba27.
//
// Solidity: function GEN0_CREATION_LIMIT() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) GEN0CREATIONLIMIT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "GEN0_CREATION_LIMIT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GEN0CREATIONLIMIT is a free data retrieval call binding the contract method 0x680eba27.
//
// Solidity: function GEN0_CREATION_LIMIT() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) GEN0CREATIONLIMIT() (*big.Int, error) {
	return _CryptoKitties.Contract.GEN0CREATIONLIMIT(&_CryptoKitties.CallOpts)
}

// GEN0CREATIONLIMIT is a free data retrieval call binding the contract method 0x680eba27.
//
// Solidity: function GEN0_CREATION_LIMIT() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) GEN0CREATIONLIMIT() (*big.Int, error) {
	return _CryptoKitties.Contract.GEN0CREATIONLIMIT(&_CryptoKitties.CallOpts)
}

// GEN0STARTINGPRICE is a free data retrieval call binding the contract method 0x0e583df0.
//
// Solidity: function GEN0_STARTING_PRICE() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) GEN0STARTINGPRICE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "GEN0_STARTING_PRICE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GEN0STARTINGPRICE is a free data retrieval call binding the contract method 0x0e583df0.
//
// Solidity: function GEN0_STARTING_PRICE() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) GEN0STARTINGPRICE() (*big.Int, error) {
	return _CryptoKitties.Contract.GEN0STARTINGPRICE(&_CryptoKitties.CallOpts)
}

// GEN0STARTINGPRICE is a free data retrieval call binding the contract method 0x0e583df0.
//
// Solidity: function GEN0_STARTING_PRICE() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) GEN0STARTINGPRICE() (*big.Int, error) {
	return _CryptoKitties.Contract.GEN0STARTINGPRICE(&_CryptoKitties.CallOpts)
}

// PROMOCREATIONLIMIT is a free data retrieval call binding the contract method 0xdefb9584.
//
// Solidity: function PROMO_CREATION_LIMIT() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) PROMOCREATIONLIMIT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "PROMO_CREATION_LIMIT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PROMOCREATIONLIMIT is a free data retrieval call binding the contract method 0xdefb9584.
//
// Solidity: function PROMO_CREATION_LIMIT() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) PROMOCREATIONLIMIT() (*big.Int, error) {
	return _CryptoKitties.Contract.PROMOCREATIONLIMIT(&_CryptoKitties.CallOpts)
}

// PROMOCREATIONLIMIT is a free data retrieval call binding the contract method 0xdefb9584.
//
// Solidity: function PROMO_CREATION_LIMIT() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) PROMOCREATIONLIMIT() (*big.Int, error) {
	return _CryptoKitties.Contract.PROMOCREATIONLIMIT(&_CryptoKitties.CallOpts)
}

// AutoBirthFee is a free data retrieval call binding the contract method 0xb0c35c05.
//
// Solidity: function autoBirthFee() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) AutoBirthFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "autoBirthFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AutoBirthFee is a free data retrieval call binding the contract method 0xb0c35c05.
//
// Solidity: function autoBirthFee() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) AutoBirthFee() (*big.Int, error) {
	return _CryptoKitties.Contract.AutoBirthFee(&_CryptoKitties.CallOpts)
}

// AutoBirthFee is a free data retrieval call binding the contract method 0xb0c35c05.
//
// Solidity: function autoBirthFee() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) AutoBirthFee() (*big.Int, error) {
	return _CryptoKitties.Contract.AutoBirthFee(&_CryptoKitties.CallOpts)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 count)
func (_CryptoKitties *CryptoKittiesCaller) BalanceOf(opts *bind.CallOpts, _owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "balanceOf", _owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 count)
func (_CryptoKitties *CryptoKittiesSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _CryptoKitties.Contract.BalanceOf(&_CryptoKitties.CallOpts, _owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 count)
func (_CryptoKitties *CryptoKittiesCallerSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _CryptoKitties.Contract.BalanceOf(&_CryptoKitties.CallOpts, _owner)
}

// CanBreedWith is a free data retrieval call binding the contract method 0x46d22c70.
//
// Solidity: function canBreedWith(uint256 _matronId, uint256 _sireId) view returns(bool)
func (_CryptoKitties *CryptoKittiesCaller) CanBreedWith(opts *bind.CallOpts, _matronId *big.Int, _sireId *big.Int) (bool, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "canBreedWith", _matronId, _sireId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CanBreedWith is a free data retrieval call binding the contract method 0x46d22c70.
//
// Solidity: function canBreedWith(uint256 _matronId, uint256 _sireId) view returns(bool)
func (_CryptoKitties *CryptoKittiesSession) CanBreedWith(_matronId *big.Int, _sireId *big.Int) (bool, error) {
	return _CryptoKitties.Contract.CanBreedWith(&_CryptoKitties.CallOpts, _matronId, _sireId)
}

// CanBreedWith is a free data retrieval call binding the contract method 0x46d22c70.
//
// Solidity: function canBreedWith(uint256 _matronId, uint256 _sireId) view returns(bool)
func (_CryptoKitties *CryptoKittiesCallerSession) CanBreedWith(_matronId *big.Int, _sireId *big.Int) (bool, error) {
	return _CryptoKitties.Contract.CanBreedWith(&_CryptoKitties.CallOpts, _matronId, _sireId)
}

// CeoAddress is a free data retrieval call binding the contract method 0x0a0f8168.
//
// Solidity: function ceoAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) CeoAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "ceoAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CeoAddress is a free data retrieval call binding the contract method 0x0a0f8168.
//
// Solidity: function ceoAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesSession) CeoAddress() (common.Address, error) {
	return _CryptoKitties.Contract.CeoAddress(&_CryptoKitties.CallOpts)
}

// CeoAddress is a free data retrieval call binding the contract method 0x0a0f8168.
//
// Solidity: function ceoAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) CeoAddress() (common.Address, error) {
	return _CryptoKitties.Contract.CeoAddress(&_CryptoKitties.CallOpts)
}

// CfoAddress is a free data retrieval call binding the contract method 0x0519ce79.
//
// Solidity: function cfoAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) CfoAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "cfoAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CfoAddress is a free data retrieval call binding the contract method 0x0519ce79.
//
// Solidity: function cfoAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesSession) CfoAddress() (common.Address, error) {
	return _CryptoKitties.Contract.CfoAddress(&_CryptoKitties.CallOpts)
}

// CfoAddress is a free data retrieval call binding the contract method 0x0519ce79.
//
// Solidity: function cfoAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) CfoAddress() (common.Address, error) {
	return _CryptoKitties.Contract.CfoAddress(&_CryptoKitties.CallOpts)
}

// CooAddress is a free data retrieval call binding the contract method 0xb047fb50.
//
// Solidity: function cooAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) CooAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "cooAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CooAddress is a free data retrieval call binding the contract method 0xb047fb50.
//
// Solidity: function cooAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesSession) CooAddress() (common.Address, error) {
	return _CryptoKitties.Contract.CooAddress(&_CryptoKitties.CallOpts)
}

// CooAddress is a free data retrieval call binding the contract method 0xb047fb50.
//
// Solidity: function cooAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) CooAddress() (common.Address, error) {
	return _CryptoKitties.Contract.CooAddress(&_CryptoKitties.CallOpts)
}

// Cooldowns is a free data retrieval call binding the contract method 0x9d6fac6f.
//
// Solidity: function cooldowns(uint256 ) view returns(uint32)
func (_CryptoKitties *CryptoKittiesCaller) Cooldowns(opts *bind.CallOpts, arg0 *big.Int) (uint32, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "cooldowns", arg0)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// Cooldowns is a free data retrieval call binding the contract method 0x9d6fac6f.
//
// Solidity: function cooldowns(uint256 ) view returns(uint32)
func (_CryptoKitties *CryptoKittiesSession) Cooldowns(arg0 *big.Int) (uint32, error) {
	return _CryptoKitties.Contract.Cooldowns(&_CryptoKitties.CallOpts, arg0)
}

// Cooldowns is a free data retrieval call binding the contract method 0x9d6fac6f.
//
// Solidity: function cooldowns(uint256 ) view returns(uint32)
func (_CryptoKitties *CryptoKittiesCallerSession) Cooldowns(arg0 *big.Int) (uint32, error) {
	return _CryptoKitties.Contract.Cooldowns(&_CryptoKitties.CallOpts, arg0)
}

// Erc721Metadata is a free data retrieval call binding the contract method 0xbc4006f5.
//
// Solidity: function erc721Metadata() view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) Erc721Metadata(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "erc721Metadata")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Erc721Metadata is a free data retrieval call binding the contract method 0xbc4006f5.
//
// Solidity: function erc721Metadata() view returns(address)
func (_CryptoKitties *CryptoKittiesSession) Erc721Metadata() (common.Address, error) {
	return _CryptoKitties.Contract.Erc721Metadata(&_CryptoKitties.CallOpts)
}

// Erc721Metadata is a free data retrieval call binding the contract method 0xbc4006f5.
//
// Solidity: function erc721Metadata() view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) Erc721Metadata() (common.Address, error) {
	return _CryptoKitties.Contract.Erc721Metadata(&_CryptoKitties.CallOpts)
}

// Gen0CreatedCount is a free data retrieval call binding the contract method 0xf1ca9410.
//
// Solidity: function gen0CreatedCount() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) Gen0CreatedCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "gen0CreatedCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Gen0CreatedCount is a free data retrieval call binding the contract method 0xf1ca9410.
//
// Solidity: function gen0CreatedCount() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) Gen0CreatedCount() (*big.Int, error) {
	return _CryptoKitties.Contract.Gen0CreatedCount(&_CryptoKitties.CallOpts)
}

// Gen0CreatedCount is a free data retrieval call binding the contract method 0xf1ca9410.
//
// Solidity: function gen0CreatedCount() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) Gen0CreatedCount() (*big.Int, error) {
	return _CryptoKitties.Contract.Gen0CreatedCount(&_CryptoKitties.CallOpts)
}

// GeneScience is a free data retrieval call binding the contract method 0xf2b47d52.
//
// Solidity: function geneScience() view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) GeneScience(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "geneScience")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GeneScience is a free data retrieval call binding the contract method 0xf2b47d52.
//
// Solidity: function geneScience() view returns(address)
func (_CryptoKitties *CryptoKittiesSession) GeneScience() (common.Address, error) {
	return _CryptoKitties.Contract.GeneScience(&_CryptoKitties.CallOpts)
}

// GeneScience is a free data retrieval call binding the contract method 0xf2b47d52.
//
// Solidity: function geneScience() view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) GeneScience() (common.Address, error) {
	return _CryptoKitties.Contract.GeneScience(&_CryptoKitties.CallOpts)
}

// GetKitty is a free data retrieval call binding the contract method 0xe98b7f4d.
//
// Solidity: function getKitty(uint256 _id) view returns(bool isGestating, bool isReady, uint256 cooldownIndex, uint256 nextActionAt, uint256 siringWithId, uint256 birthTime, uint256 matronId, uint256 sireId, uint256 generation, uint256 genes)
func (_CryptoKitties *CryptoKittiesCaller) GetKitty(opts *bind.CallOpts, _id *big.Int) (struct {
	IsGestating   bool
	IsReady       bool
	CooldownIndex *big.Int
	NextActionAt  *big.Int
	SiringWithId  *big.Int
	BirthTime     *big.Int
	MatronId      *big.Int
	SireId        *big.Int
	Generation    *big.Int
	Genes         *big.Int
}, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "getKitty", _id)

	outstruct := new(struct {
		IsGestating   bool
		IsReady       bool
		CooldownIndex *big.Int
		NextActionAt  *big.Int
		SiringWithId  *big.Int
		BirthTime     *big.Int
		MatronId      *big.Int
		SireId        *big.Int
		Generation    *big.Int
		Genes         *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.IsGestating = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.IsReady = *abi.ConvertType(out[1], new(bool)).(*bool)
	outstruct.CooldownIndex = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.NextActionAt = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.SiringWithId = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.BirthTime = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.MatronId = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.SireId = *abi.ConvertType(out[7], new(*big.Int)).(**big.Int)
	outstruct.Generation = *abi.ConvertType(out[8], new(*big.Int)).(**big.Int)
	outstruct.Genes = *abi.ConvertType(out[9], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetKitty is a free data retrieval call binding the contract method 0xe98b7f4d.
//
// Solidity: function getKitty(uint256 _id) view returns(bool isGestating, bool isReady, uint256 cooldownIndex, uint256 nextActionAt, uint256 siringWithId, uint256 birthTime, uint256 matronId, uint256 sireId, uint256 generation, uint256 genes)
func (_CryptoKitties *CryptoKittiesSession) GetKitty(_id *big.Int) (struct {
	IsGestating   bool
	IsReady       bool
	CooldownIndex *big.Int
	NextActionAt  *big.Int
	SiringWithId  *big.Int
	BirthTime     *big.Int
	MatronId      *big.Int
	SireId        *big.Int
	Generation    *big.Int
	Genes         *big.Int
}, error) {
	return _CryptoKitties.Contract.GetKitty(&_CryptoKitties.CallOpts, _id)
}

// GetKitty is a free data retrieval call binding the contract method 0xe98b7f4d.
//
// Solidity: function getKitty(uint256 _id) view returns(bool isGestating, bool isReady, uint256 cooldownIndex, uint256 nextActionAt, uint256 siringWithId, uint256 birthTime, uint256 matronId, uint256 sireId, uint256 generation, uint256 genes)
func (_CryptoKitties *CryptoKittiesCallerSession) GetKitty(_id *big.Int) (struct {
	IsGestating   bool
	IsReady       bool
	CooldownIndex *big.Int
	NextActionAt  *big.Int
	SiringWithId  *big.Int
	BirthTime     *big.Int
	MatronId      *big.Int
	SireId        *big.Int
	Generation    *big.Int
	Genes         *big.Int
}, error) {
	return _CryptoKitties.Contract.GetKitty(&_CryptoKitties.CallOpts, _id)
}

// IsPregnant is a free data retrieval call binding the contract method 0x1940a936.
//
// Solidity: function isPregnant(uint256 _kittyId) view returns(bool)
func (_CryptoKitties *CryptoKittiesCaller) IsPregnant(opts *bind.CallOpts, _kittyId *big.Int) (bool, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "isPregnant", _kittyId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPregnant is a free data retrieval call binding the contract method 0x1940a936.
//
// Solidity: function isPregnant(uint256 _kittyId) view returns(bool)
func (_CryptoKitties *CryptoKittiesSession) IsPregnant(_kittyId *big.Int) (bool, error) {
	return _CryptoKitties.Contract.IsPregnant(&_CryptoKitties.CallOpts, _kittyId)
}

// IsPregnant is a free data retrieval call binding the contract method 0x1940a936.
//
// Solidity: function isPregnant(uint256 _kittyId) view returns(bool)
func (_CryptoKitties *CryptoKittiesCallerSession) IsPregnant(_kittyId *big.Int) (bool, error) {
	return _CryptoKitties.Contract.IsPregnant(&_CryptoKitties.CallOpts, _kittyId)
}

// IsReadyToBreed is a free data retrieval call binding the contract method 0xd3e6f49f.
//
// Solidity: function isReadyToBreed(uint256 _kittyId) view returns(bool)
func (_CryptoKitties *CryptoKittiesCaller) IsReadyToBreed(opts *bind.CallOpts, _kittyId *big.Int) (bool, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "isReadyToBreed", _kittyId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsReadyToBreed is a free data retrieval call binding the contract method 0xd3e6f49f.
//
// Solidity: function isReadyToBreed(uint256 _kittyId) view returns(bool)
func (_CryptoKitties *CryptoKittiesSession) IsReadyToBreed(_kittyId *big.Int) (bool, error) {
	return _CryptoKitties.Contract.IsReadyToBreed(&_CryptoKitties.CallOpts, _kittyId)
}

// IsReadyToBreed is a free data retrieval call binding the contract method 0xd3e6f49f.
//
// Solidity: function isReadyToBreed(uint256 _kittyId) view returns(bool)
func (_CryptoKitties *CryptoKittiesCallerSession) IsReadyToBreed(_kittyId *big.Int) (bool, error) {
	return _CryptoKitties.Contract.IsReadyToBreed(&_CryptoKitties.CallOpts, _kittyId)
}

// KittyIndexToApproved is a free data retrieval call binding the contract method 0x481af3d3.
//
// Solidity: function kittyIndexToApproved(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) KittyIndexToApproved(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "kittyIndexToApproved", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// KittyIndexToApproved is a free data retrieval call binding the contract method 0x481af3d3.
//
// Solidity: function kittyIndexToApproved(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesSession) KittyIndexToApproved(arg0 *big.Int) (common.Address, error) {
	return _CryptoKitties.Contract.KittyIndexToApproved(&_CryptoKitties.CallOpts, arg0)
}

// KittyIndexToApproved is a free data retrieval call binding the contract method 0x481af3d3.
//
// Solidity: function kittyIndexToApproved(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) KittyIndexToApproved(arg0 *big.Int) (common.Address, error) {
	return _CryptoKitties.Contract.KittyIndexToApproved(&_CryptoKitties.CallOpts, arg0)
}

// KittyIndexToOwner is a free data retrieval call binding the contract method 0xa45f4bfc.
//
// Solidity: function kittyIndexToOwner(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) KittyIndexToOwner(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "kittyIndexToOwner", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// KittyIndexToOwner is a free data retrieval call binding the contract method 0xa45f4bfc.
//
// Solidity: function kittyIndexToOwner(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesSession) KittyIndexToOwner(arg0 *big.Int) (common.Address, error) {
	return _CryptoKitties.Contract.KittyIndexToOwner(&_CryptoKitties.CallOpts, arg0)
}

// KittyIndexToOwner is a free data retrieval call binding the contract method 0xa45f4bfc.
//
// Solidity: function kittyIndexToOwner(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) KittyIndexToOwner(arg0 *big.Int) (common.Address, error) {
	return _CryptoKitties.Contract.KittyIndexToOwner(&_CryptoKitties.CallOpts, arg0)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_CryptoKitties *CryptoKittiesCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_CryptoKitties *CryptoKittiesSession) Name() (string, error) {
	return _CryptoKitties.Contract.Name(&_CryptoKitties.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_CryptoKitties *CryptoKittiesCallerSession) Name() (string, error) {
	return _CryptoKitties.Contract.Name(&_CryptoKitties.CallOpts)
}

// NewContractAddress is a free data retrieval call binding the contract method 0x6af04a57.
//
// Solidity: function newContractAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) NewContractAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "newContractAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NewContractAddress is a free data retrieval call binding the contract method 0x6af04a57.
//
// Solidity: function newContractAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesSession) NewContractAddress() (common.Address, error) {
	return _CryptoKitties.Contract.NewContractAddress(&_CryptoKitties.CallOpts)
}

// NewContractAddress is a free data retrieval call binding the contract method 0x6af04a57.
//
// Solidity: function newContractAddress() view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) NewContractAddress() (common.Address, error) {
	return _CryptoKitties.Contract.NewContractAddress(&_CryptoKitties.CallOpts)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 _tokenId) view returns(address owner)
func (_CryptoKitties *CryptoKittiesCaller) OwnerOf(opts *bind.CallOpts, _tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "ownerOf", _tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 _tokenId) view returns(address owner)
func (_CryptoKitties *CryptoKittiesSession) OwnerOf(_tokenId *big.Int) (common.Address, error) {
	return _CryptoKitties.Contract.OwnerOf(&_CryptoKitties.CallOpts, _tokenId)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 _tokenId) view returns(address owner)
func (_CryptoKitties *CryptoKittiesCallerSession) OwnerOf(_tokenId *big.Int) (common.Address, error) {
	return _CryptoKitties.Contract.OwnerOf(&_CryptoKitties.CallOpts, _tokenId)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_CryptoKitties *CryptoKittiesCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_CryptoKitties *CryptoKittiesSession) Paused() (bool, error) {
	return _CryptoKitties.Contract.Paused(&_CryptoKitties.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_CryptoKitties *CryptoKittiesCallerSession) Paused() (bool, error) {
	return _CryptoKitties.Contract.Paused(&_CryptoKitties.CallOpts)
}

// PregnantKitties is a free data retrieval call binding the contract method 0x183a7947.
//
// Solidity: function pregnantKitties() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) PregnantKitties(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "pregnantKitties")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PregnantKitties is a free data retrieval call binding the contract method 0x183a7947.
//
// Solidity: function pregnantKitties() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) PregnantKitties() (*big.Int, error) {
	return _CryptoKitties.Contract.PregnantKitties(&_CryptoKitties.CallOpts)
}

// PregnantKitties is a free data retrieval call binding the contract method 0x183a7947.
//
// Solidity: function pregnantKitties() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) PregnantKitties() (*big.Int, error) {
	return _CryptoKitties.Contract.PregnantKitties(&_CryptoKitties.CallOpts)
}

// PromoCreatedCount is a free data retrieval call binding the contract method 0x05e45546.
//
// Solidity: function promoCreatedCount() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) PromoCreatedCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "promoCreatedCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PromoCreatedCount is a free data retrieval call binding the contract method 0x05e45546.
//
// Solidity: function promoCreatedCount() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) PromoCreatedCount() (*big.Int, error) {
	return _CryptoKitties.Contract.PromoCreatedCount(&_CryptoKitties.CallOpts)
}

// PromoCreatedCount is a free data retrieval call binding the contract method 0x05e45546.
//
// Solidity: function promoCreatedCount() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) PromoCreatedCount() (*big.Int, error) {
	return _CryptoKitties.Contract.PromoCreatedCount(&_CryptoKitties.CallOpts)
}

// SaleAuction is a free data retrieval call binding the contract method 0xe6cbe351.
//
// Solidity: function saleAuction() view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) SaleAuction(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "saleAuction")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SaleAuction is a free data retrieval call binding the contract method 0xe6cbe351.
//
// Solidity: function saleAuction() view returns(address)
func (_CryptoKitties *CryptoKittiesSession) SaleAuction() (common.Address, error) {
	return _CryptoKitties.Contract.SaleAuction(&_CryptoKitties.CallOpts)
}

// SaleAuction is a free data retrieval call binding the contract method 0xe6cbe351.
//
// Solidity: function saleAuction() view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) SaleAuction() (common.Address, error) {
	return _CryptoKitties.Contract.SaleAuction(&_CryptoKitties.CallOpts)
}

// SecondsPerBlock is a free data retrieval call binding the contract method 0x7a7d4937.
//
// Solidity: function secondsPerBlock() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) SecondsPerBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "secondsPerBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SecondsPerBlock is a free data retrieval call binding the contract method 0x7a7d4937.
//
// Solidity: function secondsPerBlock() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) SecondsPerBlock() (*big.Int, error) {
	return _CryptoKitties.Contract.SecondsPerBlock(&_CryptoKitties.CallOpts)
}

// SecondsPerBlock is a free data retrieval call binding the contract method 0x7a7d4937.
//
// Solidity: function secondsPerBlock() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) SecondsPerBlock() (*big.Int, error) {
	return _CryptoKitties.Contract.SecondsPerBlock(&_CryptoKitties.CallOpts)
}

// SireAllowedToAddress is a free data retrieval call binding the contract method 0x46116e6f.
//
// Solidity: function sireAllowedToAddress(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) SireAllowedToAddress(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "sireAllowedToAddress", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SireAllowedToAddress is a free data retrieval call binding the contract method 0x46116e6f.
//
// Solidity: function sireAllowedToAddress(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesSession) SireAllowedToAddress(arg0 *big.Int) (common.Address, error) {
	return _CryptoKitties.Contract.SireAllowedToAddress(&_CryptoKitties.CallOpts, arg0)
}

// SireAllowedToAddress is a free data retrieval call binding the contract method 0x46116e6f.
//
// Solidity: function sireAllowedToAddress(uint256 ) view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) SireAllowedToAddress(arg0 *big.Int) (common.Address, error) {
	return _CryptoKitties.Contract.SireAllowedToAddress(&_CryptoKitties.CallOpts, arg0)
}

// SiringAuction is a free data retrieval call binding the contract method 0x21717ebf.
//
// Solidity: function siringAuction() view returns(address)
func (_CryptoKitties *CryptoKittiesCaller) SiringAuction(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "siringAuction")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SiringAuction is a free data retrieval call binding the contract method 0x21717ebf.
//
// Solidity: function siringAuction() view returns(address)
func (_CryptoKitties *CryptoKittiesSession) SiringAuction() (common.Address, error) {
	return _CryptoKitties.Contract.SiringAuction(&_CryptoKitties.CallOpts)
}

// SiringAuction is a free data retrieval call binding the contract method 0x21717ebf.
//
// Solidity: function siringAuction() view returns(address)
func (_CryptoKitties *CryptoKittiesCallerSession) SiringAuction() (common.Address, error) {
	return _CryptoKitties.Contract.SiringAuction(&_CryptoKitties.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceID) view returns(bool)
func (_CryptoKitties *CryptoKittiesCaller) SupportsInterface(opts *bind.CallOpts, _interfaceID [4]byte) (bool, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "supportsInterface", _interfaceID)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceID) view returns(bool)
func (_CryptoKitties *CryptoKittiesSession) SupportsInterface(_interfaceID [4]byte) (bool, error) {
	return _CryptoKitties.Contract.SupportsInterface(&_CryptoKitties.CallOpts, _interfaceID)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 _interfaceID) view returns(bool)
func (_CryptoKitties *CryptoKittiesCallerSession) SupportsInterface(_interfaceID [4]byte) (bool, error) {
	return _CryptoKitties.Contract.SupportsInterface(&_CryptoKitties.CallOpts, _interfaceID)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_CryptoKitties *CryptoKittiesCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_CryptoKitties *CryptoKittiesSession) Symbol() (string, error) {
	return _CryptoKitties.Contract.Symbol(&_CryptoKitties.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_CryptoKitties *CryptoKittiesCallerSession) Symbol() (string, error) {
	return _CryptoKitties.Contract.Symbol(&_CryptoKitties.CallOpts)
}

// TokenMetadata is a free data retrieval call binding the contract method 0x0560ff44.
//
// Solidity: function tokenMetadata(uint256 _tokenId, string _preferredTransport) view returns(string infoUrl)
func (_CryptoKitties *CryptoKittiesCaller) TokenMetadata(opts *bind.CallOpts, _tokenId *big.Int, _preferredTransport string) (string, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "tokenMetadata", _tokenId, _preferredTransport)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// TokenMetadata is a free data retrieval call binding the contract method 0x0560ff44.
//
// Solidity: function tokenMetadata(uint256 _tokenId, string _preferredTransport) view returns(string infoUrl)
func (_CryptoKitties *CryptoKittiesSession) TokenMetadata(_tokenId *big.Int, _preferredTransport string) (string, error) {
	return _CryptoKitties.Contract.TokenMetadata(&_CryptoKitties.CallOpts, _tokenId, _preferredTransport)
}

// TokenMetadata is a free data retrieval call binding the contract method 0x0560ff44.
//
// Solidity: function tokenMetadata(uint256 _tokenId, string _preferredTransport) view returns(string infoUrl)
func (_CryptoKitties *CryptoKittiesCallerSession) TokenMetadata(_tokenId *big.Int, _preferredTransport string) (string, error) {
	return _CryptoKitties.Contract.TokenMetadata(&_CryptoKitties.CallOpts, _tokenId, _preferredTransport)
}

// TokensOfOwner is a free data retrieval call binding the contract method 0x8462151c.
//
// Solidity: function tokensOfOwner(address _owner) view returns(uint256[] ownerTokens)
func (_CryptoKitties *CryptoKittiesCaller) TokensOfOwner(opts *bind.CallOpts, _owner common.Address) ([]*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "tokensOfOwner", _owner)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// TokensOfOwner is a free data retrieval call binding the contract method 0x8462151c.
//
// Solidity: function tokensOfOwner(address _owner) view returns(uint256[] ownerTokens)
func (_CryptoKitties *CryptoKittiesSession) TokensOfOwner(_owner common.Address) ([]*big.Int, error) {
	return _CryptoKitties.Contract.TokensOfOwner(&_CryptoKitties.CallOpts, _owner)
}

// TokensOfOwner is a free data retrieval call binding the contract method 0x8462151c.
//
// Solidity: function tokensOfOwner(address _owner) view returns(uint256[] ownerTokens)
func (_CryptoKitties *CryptoKittiesCallerSession) TokensOfOwner(_owner common.Address) ([]*big.Int, error) {
	return _CryptoKitties.Contract.TokensOfOwner(&_CryptoKitties.CallOpts, _owner)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CryptoKitties.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) TotalSupply() (*big.Int, error) {
	return _CryptoKitties.Contract.TotalSupply(&_CryptoKitties.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_CryptoKitties *CryptoKittiesCallerSession) TotalSupply() (*big.Int, error) {
	return _CryptoKitties.Contract.TotalSupply(&_CryptoKitties.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesTransactor) Approve(opts *bind.TransactOpts, _to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "approve", _to, _tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesSession) Approve(_to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.Approve(&_CryptoKitties.TransactOpts, _to, _tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) Approve(_to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.Approve(&_CryptoKitties.TransactOpts, _to, _tokenId)
}

// ApproveSiring is a paid mutator transaction binding the contract method 0x4dfff04f.
//
// Solidity: function approveSiring(address _addr, uint256 _sireId) returns()
func (_CryptoKitties *CryptoKittiesTransactor) ApproveSiring(opts *bind.TransactOpts, _addr common.Address, _sireId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "approveSiring", _addr, _sireId)
}

// ApproveSiring is a paid mutator transaction binding the contract method 0x4dfff04f.
//
// Solidity: function approveSiring(address _addr, uint256 _sireId) returns()
func (_CryptoKitties *CryptoKittiesSession) ApproveSiring(_addr common.Address, _sireId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.ApproveSiring(&_CryptoKitties.TransactOpts, _addr, _sireId)
}

// ApproveSiring is a paid mutator transaction binding the contract method 0x4dfff04f.
//
// Solidity: function approveSiring(address _addr, uint256 _sireId) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) ApproveSiring(_addr common.Address, _sireId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.ApproveSiring(&_CryptoKitties.TransactOpts, _addr, _sireId)
}

// BidOnSiringAuction is a paid mutator transaction binding the contract method 0xed60ade6.
//
// Solidity: function bidOnSiringAuction(uint256 _sireId, uint256 _matronId) payable returns()
func (_CryptoKitties *CryptoKittiesTransactor) BidOnSiringAuction(opts *bind.TransactOpts, _sireId *big.Int, _matronId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "bidOnSiringAuction", _sireId, _matronId)
}

// BidOnSiringAuction is a paid mutator transaction binding the contract method 0xed60ade6.
//
// Solidity: function bidOnSiringAuction(uint256 _sireId, uint256 _matronId) payable returns()
func (_CryptoKitties *CryptoKittiesSession) BidOnSiringAuction(_sireId *big.Int, _matronId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.BidOnSiringAuction(&_CryptoKitties.TransactOpts, _sireId, _matronId)
}

// BidOnSiringAuction is a paid mutator transaction binding the contract method 0xed60ade6.
//
// Solidity: function bidOnSiringAuction(uint256 _sireId, uint256 _matronId) payable returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) BidOnSiringAuction(_sireId *big.Int, _matronId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.BidOnSiringAuction(&_CryptoKitties.TransactOpts, _sireId, _matronId)
}

// BreedWithAuto is a paid mutator transaction binding the contract method 0xf7d8c883.
//
// Solidity: function breedWithAuto(uint256 _matronId, uint256 _sireId) payable returns()
func (_CryptoKitties *CryptoKittiesTransactor) BreedWithAuto(opts *bind.TransactOpts, _matronId *big.Int, _sireId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "breedWithAuto", _matronId, _sireId)
}

// BreedWithAuto is a paid mutator transaction binding the contract method 0xf7d8c883.
//
// Solidity: function breedWithAuto(uint256 _matronId, uint256 _sireId) payable returns()
func (_CryptoKitties *CryptoKittiesSession) BreedWithAuto(_matronId *big.Int, _sireId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.BreedWithAuto(&_CryptoKitties.TransactOpts, _matronId, _sireId)
}

// BreedWithAuto is a paid mutator transaction binding the contract method 0xf7d8c883.
//
// Solidity: function breedWithAuto(uint256 _matronId, uint256 _sireId) payable returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) BreedWithAuto(_matronId *big.Int, _sireId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.BreedWithAuto(&_CryptoKitties.TransactOpts, _matronId, _sireId)
}

// CreateGen0Auction is a paid mutator transaction binding the contract method 0xc3bea9af.
//
// Solidity: function createGen0Auction(uint256 _genes) returns()
func (_CryptoKitties *CryptoKittiesTransactor) CreateGen0Auction(opts *bind.TransactOpts, _genes *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "createGen0Auction", _genes)
}

// CreateGen0Auction is a paid mutator transaction binding the contract method 0xc3bea9af.
//
// Solidity: function createGen0Auction(uint256 _genes) returns()
func (_CryptoKitties *CryptoKittiesSession) CreateGen0Auction(_genes *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CreateGen0Auction(&_CryptoKitties.TransactOpts, _genes)
}

// CreateGen0Auction is a paid mutator transaction binding the contract method 0xc3bea9af.
//
// Solidity: function createGen0Auction(uint256 _genes) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) CreateGen0Auction(_genes *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CreateGen0Auction(&_CryptoKitties.TransactOpts, _genes)
}

// CreatePromoKitty is a paid mutator transaction binding the contract method 0x56129134.
//
// Solidity: function createPromoKitty(uint256 _genes, address _owner) returns()
func (_CryptoKitties *CryptoKittiesTransactor) CreatePromoKitty(opts *bind.TransactOpts, _genes *big.Int, _owner common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "createPromoKitty", _genes, _owner)
}

// CreatePromoKitty is a paid mutator transaction binding the contract method 0x56129134.
//
// Solidity: function createPromoKitty(uint256 _genes, address _owner) returns()
func (_CryptoKitties *CryptoKittiesSession) CreatePromoKitty(_genes *big.Int, _owner common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CreatePromoKitty(&_CryptoKitties.TransactOpts, _genes, _owner)
}

// CreatePromoKitty is a paid mutator transaction binding the contract method 0x56129134.
//
// Solidity: function createPromoKitty(uint256 _genes, address _owner) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) CreatePromoKitty(_genes *big.Int, _owner common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CreatePromoKitty(&_CryptoKitties.TransactOpts, _genes, _owner)
}

// CreateSaleAuction is a paid mutator transaction binding the contract method 0x3d7d3f5a.
//
// Solidity: function createSaleAuction(uint256 _kittyId, uint256 _startingPrice, uint256 _endingPrice, uint256 _duration) returns()
func (_CryptoKitties *CryptoKittiesTransactor) CreateSaleAuction(opts *bind.TransactOpts, _kittyId *big.Int, _startingPrice *big.Int, _endingPrice *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "createSaleAuction", _kittyId, _startingPrice, _endingPrice, _duration)
}

// CreateSaleAuction is a paid mutator transaction binding the contract method 0x3d7d3f5a.
//
// Solidity: function createSaleAuction(uint256 _kittyId, uint256 _startingPrice, uint256 _endingPrice, uint256 _duration) returns()
func (_CryptoKitties *CryptoKittiesSession) CreateSaleAuction(_kittyId *big.Int, _startingPrice *big.Int, _endingPrice *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CreateSaleAuction(&_CryptoKitties.TransactOpts, _kittyId, _startingPrice, _endingPrice, _duration)
}

// CreateSaleAuction is a paid mutator transaction binding the contract method 0x3d7d3f5a.
//
// Solidity: function createSaleAuction(uint256 _kittyId, uint256 _startingPrice, uint256 _endingPrice, uint256 _duration) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) CreateSaleAuction(_kittyId *big.Int, _startingPrice *big.Int, _endingPrice *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CreateSaleAuction(&_CryptoKitties.TransactOpts, _kittyId, _startingPrice, _endingPrice, _duration)
}

// CreateSiringAuction is a paid mutator transaction binding the contract method 0x4ad8c938.
//
// Solidity: function createSiringAuction(uint256 _kittyId, uint256 _startingPrice, uint256 _endingPrice, uint256 _duration) returns()
func (_CryptoKitties *CryptoKittiesTransactor) CreateSiringAuction(opts *bind.TransactOpts, _kittyId *big.Int, _startingPrice *big.Int, _endingPrice *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "createSiringAuction", _kittyId, _startingPrice, _endingPrice, _duration)
}

// CreateSiringAuction is a paid mutator transaction binding the contract method 0x4ad8c938.
//
// Solidity: function createSiringAuction(uint256 _kittyId, uint256 _startingPrice, uint256 _endingPrice, uint256 _duration) returns()
func (_CryptoKitties *CryptoKittiesSession) CreateSiringAuction(_kittyId *big.Int, _startingPrice *big.Int, _endingPrice *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CreateSiringAuction(&_CryptoKitties.TransactOpts, _kittyId, _startingPrice, _endingPrice, _duration)
}

// CreateSiringAuction is a paid mutator transaction binding the contract method 0x4ad8c938.
//
// Solidity: function createSiringAuction(uint256 _kittyId, uint256 _startingPrice, uint256 _endingPrice, uint256 _duration) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) CreateSiringAuction(_kittyId *big.Int, _startingPrice *big.Int, _endingPrice *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.CreateSiringAuction(&_CryptoKitties.TransactOpts, _kittyId, _startingPrice, _endingPrice, _duration)
}

// GiveBirth is a paid mutator transaction binding the contract method 0x88c2a0bf.
//
// Solidity: function giveBirth(uint256 _matronId) returns(uint256)
func (_CryptoKitties *CryptoKittiesTransactor) GiveBirth(opts *bind.TransactOpts, _matronId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "giveBirth", _matronId)
}

// GiveBirth is a paid mutator transaction binding the contract method 0x88c2a0bf.
//
// Solidity: function giveBirth(uint256 _matronId) returns(uint256)
func (_CryptoKitties *CryptoKittiesSession) GiveBirth(_matronId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.GiveBirth(&_CryptoKitties.TransactOpts, _matronId)
}

// GiveBirth is a paid mutator transaction binding the contract method 0x88c2a0bf.
//
// Solidity: function giveBirth(uint256 _matronId) returns(uint256)
func (_CryptoKitties *CryptoKittiesTransactorSession) GiveBirth(_matronId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.GiveBirth(&_CryptoKitties.TransactOpts, _matronId)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_CryptoKitties *CryptoKittiesTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_CryptoKitties *CryptoKittiesSession) Pause() (*types.Transaction, error) {
	return _CryptoKitties.Contract.Pause(&_CryptoKitties.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) Pause() (*types.Transaction, error) {
	return _CryptoKitties.Contract.Pause(&_CryptoKitties.TransactOpts)
}

// SetAutoBirthFee is a paid mutator transaction binding the contract method 0x4b85fd55.
//
// Solidity: function setAutoBirthFee(uint256 val) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetAutoBirthFee(opts *bind.TransactOpts, val *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setAutoBirthFee", val)
}

// SetAutoBirthFee is a paid mutator transaction binding the contract method 0x4b85fd55.
//
// Solidity: function setAutoBirthFee(uint256 val) returns()
func (_CryptoKitties *CryptoKittiesSession) SetAutoBirthFee(val *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetAutoBirthFee(&_CryptoKitties.TransactOpts, val)
}

// SetAutoBirthFee is a paid mutator transaction binding the contract method 0x4b85fd55.
//
// Solidity: function setAutoBirthFee(uint256 val) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetAutoBirthFee(val *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetAutoBirthFee(&_CryptoKitties.TransactOpts, val)
}

// SetCEO is a paid mutator transaction binding the contract method 0x27d7874c.
//
// Solidity: function setCEO(address _newCEO) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetCEO(opts *bind.TransactOpts, _newCEO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setCEO", _newCEO)
}

// SetCEO is a paid mutator transaction binding the contract method 0x27d7874c.
//
// Solidity: function setCEO(address _newCEO) returns()
func (_CryptoKitties *CryptoKittiesSession) SetCEO(_newCEO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetCEO(&_CryptoKitties.TransactOpts, _newCEO)
}

// SetCEO is a paid mutator transaction binding the contract method 0x27d7874c.
//
// Solidity: function setCEO(address _newCEO) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetCEO(_newCEO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetCEO(&_CryptoKitties.TransactOpts, _newCEO)
}

// SetCFO is a paid mutator transaction binding the contract method 0x4e0a3379.
//
// Solidity: function setCFO(address _newCFO) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetCFO(opts *bind.TransactOpts, _newCFO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setCFO", _newCFO)
}

// SetCFO is a paid mutator transaction binding the contract method 0x4e0a3379.
//
// Solidity: function setCFO(address _newCFO) returns()
func (_CryptoKitties *CryptoKittiesSession) SetCFO(_newCFO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetCFO(&_CryptoKitties.TransactOpts, _newCFO)
}

// SetCFO is a paid mutator transaction binding the contract method 0x4e0a3379.
//
// Solidity: function setCFO(address _newCFO) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetCFO(_newCFO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetCFO(&_CryptoKitties.TransactOpts, _newCFO)
}

// SetCOO is a paid mutator transaction binding the contract method 0x2ba73c15.
//
// Solidity: function setCOO(address _newCOO) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetCOO(opts *bind.TransactOpts, _newCOO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setCOO", _newCOO)
}

// SetCOO is a paid mutator transaction binding the contract method 0x2ba73c15.
//
// Solidity: function setCOO(address _newCOO) returns()
func (_CryptoKitties *CryptoKittiesSession) SetCOO(_newCOO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetCOO(&_CryptoKitties.TransactOpts, _newCOO)
}

// SetCOO is a paid mutator transaction binding the contract method 0x2ba73c15.
//
// Solidity: function setCOO(address _newCOO) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetCOO(_newCOO common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetCOO(&_CryptoKitties.TransactOpts, _newCOO)
}

// SetGeneScienceAddress is a paid mutator transaction binding the contract method 0x24e7a38a.
//
// Solidity: function setGeneScienceAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetGeneScienceAddress(opts *bind.TransactOpts, _address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setGeneScienceAddress", _address)
}

// SetGeneScienceAddress is a paid mutator transaction binding the contract method 0x24e7a38a.
//
// Solidity: function setGeneScienceAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesSession) SetGeneScienceAddress(_address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetGeneScienceAddress(&_CryptoKitties.TransactOpts, _address)
}

// SetGeneScienceAddress is a paid mutator transaction binding the contract method 0x24e7a38a.
//
// Solidity: function setGeneScienceAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetGeneScienceAddress(_address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetGeneScienceAddress(&_CryptoKitties.TransactOpts, _address)
}

// SetMetadataAddress is a paid mutator transaction binding the contract method 0xe17b25af.
//
// Solidity: function setMetadataAddress(address _contractAddress) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetMetadataAddress(opts *bind.TransactOpts, _contractAddress common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setMetadataAddress", _contractAddress)
}

// SetMetadataAddress is a paid mutator transaction binding the contract method 0xe17b25af.
//
// Solidity: function setMetadataAddress(address _contractAddress) returns()
func (_CryptoKitties *CryptoKittiesSession) SetMetadataAddress(_contractAddress common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetMetadataAddress(&_CryptoKitties.TransactOpts, _contractAddress)
}

// SetMetadataAddress is a paid mutator transaction binding the contract method 0xe17b25af.
//
// Solidity: function setMetadataAddress(address _contractAddress) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetMetadataAddress(_contractAddress common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetMetadataAddress(&_CryptoKitties.TransactOpts, _contractAddress)
}

// SetNewAddress is a paid mutator transaction binding the contract method 0x71587988.
//
// Solidity: function setNewAddress(address _v2Address) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetNewAddress(opts *bind.TransactOpts, _v2Address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setNewAddress", _v2Address)
}

// SetNewAddress is a paid mutator transaction binding the contract method 0x71587988.
//
// Solidity: function setNewAddress(address _v2Address) returns()
func (_CryptoKitties *CryptoKittiesSession) SetNewAddress(_v2Address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetNewAddress(&_CryptoKitties.TransactOpts, _v2Address)
}

// SetNewAddress is a paid mutator transaction binding the contract method 0x71587988.
//
// Solidity: function setNewAddress(address _v2Address) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetNewAddress(_v2Address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetNewAddress(&_CryptoKitties.TransactOpts, _v2Address)
}

// SetSaleAuctionAddress is a paid mutator transaction binding the contract method 0x6fbde40d.
//
// Solidity: function setSaleAuctionAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetSaleAuctionAddress(opts *bind.TransactOpts, _address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setSaleAuctionAddress", _address)
}

// SetSaleAuctionAddress is a paid mutator transaction binding the contract method 0x6fbde40d.
//
// Solidity: function setSaleAuctionAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesSession) SetSaleAuctionAddress(_address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetSaleAuctionAddress(&_CryptoKitties.TransactOpts, _address)
}

// SetSaleAuctionAddress is a paid mutator transaction binding the contract method 0x6fbde40d.
//
// Solidity: function setSaleAuctionAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetSaleAuctionAddress(_address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetSaleAuctionAddress(&_CryptoKitties.TransactOpts, _address)
}

// SetSecondsPerBlock is a paid mutator transaction binding the contract method 0x5663896e.
//
// Solidity: function setSecondsPerBlock(uint256 secs) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetSecondsPerBlock(opts *bind.TransactOpts, secs *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setSecondsPerBlock", secs)
}

// SetSecondsPerBlock is a paid mutator transaction binding the contract method 0x5663896e.
//
// Solidity: function setSecondsPerBlock(uint256 secs) returns()
func (_CryptoKitties *CryptoKittiesSession) SetSecondsPerBlock(secs *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetSecondsPerBlock(&_CryptoKitties.TransactOpts, secs)
}

// SetSecondsPerBlock is a paid mutator transaction binding the contract method 0x5663896e.
//
// Solidity: function setSecondsPerBlock(uint256 secs) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetSecondsPerBlock(secs *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetSecondsPerBlock(&_CryptoKitties.TransactOpts, secs)
}

// SetSiringAuctionAddress is a paid mutator transaction binding the contract method 0x14001f4c.
//
// Solidity: function setSiringAuctionAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesTransactor) SetSiringAuctionAddress(opts *bind.TransactOpts, _address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "setSiringAuctionAddress", _address)
}

// SetSiringAuctionAddress is a paid mutator transaction binding the contract method 0x14001f4c.
//
// Solidity: function setSiringAuctionAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesSession) SetSiringAuctionAddress(_address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetSiringAuctionAddress(&_CryptoKitties.TransactOpts, _address)
}

// SetSiringAuctionAddress is a paid mutator transaction binding the contract method 0x14001f4c.
//
// Solidity: function setSiringAuctionAddress(address _address) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) SetSiringAuctionAddress(_address common.Address) (*types.Transaction, error) {
	return _CryptoKitties.Contract.SetSiringAuctionAddress(&_CryptoKitties.TransactOpts, _address)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesTransactor) Transfer(opts *bind.TransactOpts, _to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "transfer", _to, _tokenId)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesSession) Transfer(_to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.Transfer(&_CryptoKitties.TransactOpts, _to, _tokenId)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) Transfer(_to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.Transfer(&_CryptoKitties.TransactOpts, _to, _tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesTransactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "transferFrom", _from, _to, _tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesSession) TransferFrom(_from common.Address, _to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.TransferFrom(&_CryptoKitties.TransactOpts, _from, _to, _tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _tokenId) returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) TransferFrom(_from common.Address, _to common.Address, _tokenId *big.Int) (*types.Transaction, error) {
	return _CryptoKitties.Contract.TransferFrom(&_CryptoKitties.TransactOpts, _from, _to, _tokenId)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_CryptoKitties *CryptoKittiesTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_CryptoKitties *CryptoKittiesSession) Unpause() (*types.Transaction, error) {
	return _CryptoKitties.Contract.Unpause(&_CryptoKitties.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) Unpause() (*types.Transaction, error) {
	return _CryptoKitties.Contract.Unpause(&_CryptoKitties.TransactOpts)
}

// WithdrawAuctionBalances is a paid mutator transaction binding the contract method 0x91876e57.
//
// Solidity: function withdrawAuctionBalances() returns()
func (_CryptoKitties *CryptoKittiesTransactor) WithdrawAuctionBalances(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "withdrawAuctionBalances")
}

// WithdrawAuctionBalances is a paid mutator transaction binding the contract method 0x91876e57.
//
// Solidity: function withdrawAuctionBalances() returns()
func (_CryptoKitties *CryptoKittiesSession) WithdrawAuctionBalances() (*types.Transaction, error) {
	return _CryptoKitties.Contract.WithdrawAuctionBalances(&_CryptoKitties.TransactOpts)
}

// WithdrawAuctionBalances is a paid mutator transaction binding the contract method 0x91876e57.
//
// Solidity: function withdrawAuctionBalances() returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) WithdrawAuctionBalances() (*types.Transaction, error) {
	return _CryptoKitties.Contract.WithdrawAuctionBalances(&_CryptoKitties.TransactOpts)
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_CryptoKitties *CryptoKittiesTransactor) WithdrawBalance(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptoKitties.contract.Transact(opts, "withdrawBalance")
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_CryptoKitties *CryptoKittiesSession) WithdrawBalance() (*types.Transaction, error) {
	return _CryptoKitties.Contract.WithdrawBalance(&_CryptoKitties.TransactOpts)
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) WithdrawBalance() (*types.Transaction, error) {
	return _CryptoKitties.Contract.WithdrawBalance(&_CryptoKitties.TransactOpts)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_CryptoKitties *CryptoKittiesTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _CryptoKitties.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_CryptoKitties *CryptoKittiesSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _CryptoKitties.Contract.Fallback(&_CryptoKitties.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_CryptoKitties *CryptoKittiesTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _CryptoKitties.Contract.Fallback(&_CryptoKitties.TransactOpts, calldata)
}

// CryptoKittiesApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the CryptoKitties contract.
type CryptoKittiesApprovalIterator struct {
	Event *CryptoKittiesApproval // Event containing the contract specifics and raw log

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
func (it *CryptoKittiesApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoKittiesApproval)
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
		it.Event = new(CryptoKittiesApproval)
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
func (it *CryptoKittiesApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoKittiesApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoKittiesApproval represents a Approval event raised by the CryptoKitties contract.
type CryptoKittiesApproval struct {
	Owner    common.Address
	Approved common.Address
	TokenId  *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address owner, address approved, uint256 tokenId)
func (_CryptoKitties *CryptoKittiesFilterer) FilterApproval(opts *bind.FilterOpts) (*CryptoKittiesApprovalIterator, error) {

	logs, sub, err := _CryptoKitties.contract.FilterLogs(opts, "Approval")
	if err != nil {
		return nil, err
	}
	return &CryptoKittiesApprovalIterator{contract: _CryptoKitties.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address owner, address approved, uint256 tokenId)
func (_CryptoKitties *CryptoKittiesFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *CryptoKittiesApproval) (event.Subscription, error) {

	logs, sub, err := _CryptoKitties.contract.WatchLogs(opts, "Approval")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoKittiesApproval)
				if err := _CryptoKitties.contract.UnpackLog(event, "Approval", log); err != nil {
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
// Solidity: event Approval(address owner, address approved, uint256 tokenId)
func (_CryptoKitties *CryptoKittiesFilterer) ParseApproval(log types.Log) (*CryptoKittiesApproval, error) {
	event := new(CryptoKittiesApproval)
	if err := _CryptoKitties.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoKittiesBirthIterator is returned from FilterBirth and is used to iterate over the raw logs and unpacked data for Birth events raised by the CryptoKitties contract.
type CryptoKittiesBirthIterator struct {
	Event *CryptoKittiesBirth // Event containing the contract specifics and raw log

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
func (it *CryptoKittiesBirthIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoKittiesBirth)
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
		it.Event = new(CryptoKittiesBirth)
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
func (it *CryptoKittiesBirthIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoKittiesBirthIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoKittiesBirth represents a Birth event raised by the CryptoKitties contract.
type CryptoKittiesBirth struct {
	Owner    common.Address
	KittyId  *big.Int
	MatronId *big.Int
	SireId   *big.Int
	Genes    *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterBirth is a free log retrieval operation binding the contract event 0x0a5311bd2a6608f08a180df2ee7c5946819a649b204b554bb8e39825b2c50ad5.
//
// Solidity: event Birth(address owner, uint256 kittyId, uint256 matronId, uint256 sireId, uint256 genes)
func (_CryptoKitties *CryptoKittiesFilterer) FilterBirth(opts *bind.FilterOpts) (*CryptoKittiesBirthIterator, error) {

	logs, sub, err := _CryptoKitties.contract.FilterLogs(opts, "Birth")
	if err != nil {
		return nil, err
	}
	return &CryptoKittiesBirthIterator{contract: _CryptoKitties.contract, event: "Birth", logs: logs, sub: sub}, nil
}

// WatchBirth is a free log subscription operation binding the contract event 0x0a5311bd2a6608f08a180df2ee7c5946819a649b204b554bb8e39825b2c50ad5.
//
// Solidity: event Birth(address owner, uint256 kittyId, uint256 matronId, uint256 sireId, uint256 genes)
func (_CryptoKitties *CryptoKittiesFilterer) WatchBirth(opts *bind.WatchOpts, sink chan<- *CryptoKittiesBirth) (event.Subscription, error) {

	logs, sub, err := _CryptoKitties.contract.WatchLogs(opts, "Birth")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoKittiesBirth)
				if err := _CryptoKitties.contract.UnpackLog(event, "Birth", log); err != nil {
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

// ParseBirth is a log parse operation binding the contract event 0x0a5311bd2a6608f08a180df2ee7c5946819a649b204b554bb8e39825b2c50ad5.
//
// Solidity: event Birth(address owner, uint256 kittyId, uint256 matronId, uint256 sireId, uint256 genes)
func (_CryptoKitties *CryptoKittiesFilterer) ParseBirth(log types.Log) (*CryptoKittiesBirth, error) {
	event := new(CryptoKittiesBirth)
	if err := _CryptoKitties.contract.UnpackLog(event, "Birth", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoKittiesContractUpgradeIterator is returned from FilterContractUpgrade and is used to iterate over the raw logs and unpacked data for ContractUpgrade events raised by the CryptoKitties contract.
type CryptoKittiesContractUpgradeIterator struct {
	Event *CryptoKittiesContractUpgrade // Event containing the contract specifics and raw log

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
func (it *CryptoKittiesContractUpgradeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoKittiesContractUpgrade)
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
		it.Event = new(CryptoKittiesContractUpgrade)
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
func (it *CryptoKittiesContractUpgradeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoKittiesContractUpgradeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoKittiesContractUpgrade represents a ContractUpgrade event raised by the CryptoKitties contract.
type CryptoKittiesContractUpgrade struct {
	NewContract common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterContractUpgrade is a free log retrieval operation binding the contract event 0x450db8da6efbe9c22f2347f7c2021231df1fc58d3ae9a2fa75d39fa446199305.
//
// Solidity: event ContractUpgrade(address newContract)
func (_CryptoKitties *CryptoKittiesFilterer) FilterContractUpgrade(opts *bind.FilterOpts) (*CryptoKittiesContractUpgradeIterator, error) {

	logs, sub, err := _CryptoKitties.contract.FilterLogs(opts, "ContractUpgrade")
	if err != nil {
		return nil, err
	}
	return &CryptoKittiesContractUpgradeIterator{contract: _CryptoKitties.contract, event: "ContractUpgrade", logs: logs, sub: sub}, nil
}

// WatchContractUpgrade is a free log subscription operation binding the contract event 0x450db8da6efbe9c22f2347f7c2021231df1fc58d3ae9a2fa75d39fa446199305.
//
// Solidity: event ContractUpgrade(address newContract)
func (_CryptoKitties *CryptoKittiesFilterer) WatchContractUpgrade(opts *bind.WatchOpts, sink chan<- *CryptoKittiesContractUpgrade) (event.Subscription, error) {

	logs, sub, err := _CryptoKitties.contract.WatchLogs(opts, "ContractUpgrade")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoKittiesContractUpgrade)
				if err := _CryptoKitties.contract.UnpackLog(event, "ContractUpgrade", log); err != nil {
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

// ParseContractUpgrade is a log parse operation binding the contract event 0x450db8da6efbe9c22f2347f7c2021231df1fc58d3ae9a2fa75d39fa446199305.
//
// Solidity: event ContractUpgrade(address newContract)
func (_CryptoKitties *CryptoKittiesFilterer) ParseContractUpgrade(log types.Log) (*CryptoKittiesContractUpgrade, error) {
	event := new(CryptoKittiesContractUpgrade)
	if err := _CryptoKitties.contract.UnpackLog(event, "ContractUpgrade", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoKittiesPregnantIterator is returned from FilterPregnant and is used to iterate over the raw logs and unpacked data for Pregnant events raised by the CryptoKitties contract.
type CryptoKittiesPregnantIterator struct {
	Event *CryptoKittiesPregnant // Event containing the contract specifics and raw log

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
func (it *CryptoKittiesPregnantIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoKittiesPregnant)
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
		it.Event = new(CryptoKittiesPregnant)
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
func (it *CryptoKittiesPregnantIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoKittiesPregnantIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoKittiesPregnant represents a Pregnant event raised by the CryptoKitties contract.
type CryptoKittiesPregnant struct {
	Owner            common.Address
	MatronId         *big.Int
	SireId           *big.Int
	CooldownEndBlock *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterPregnant is a free log retrieval operation binding the contract event 0x241ea03ca20251805084d27d4440371c34a0b85ff108f6bb5611248f73818b80.
//
// Solidity: event Pregnant(address owner, uint256 matronId, uint256 sireId, uint256 cooldownEndBlock)
func (_CryptoKitties *CryptoKittiesFilterer) FilterPregnant(opts *bind.FilterOpts) (*CryptoKittiesPregnantIterator, error) {

	logs, sub, err := _CryptoKitties.contract.FilterLogs(opts, "Pregnant")
	if err != nil {
		return nil, err
	}
	return &CryptoKittiesPregnantIterator{contract: _CryptoKitties.contract, event: "Pregnant", logs: logs, sub: sub}, nil
}

// WatchPregnant is a free log subscription operation binding the contract event 0x241ea03ca20251805084d27d4440371c34a0b85ff108f6bb5611248f73818b80.
//
// Solidity: event Pregnant(address owner, uint256 matronId, uint256 sireId, uint256 cooldownEndBlock)
func (_CryptoKitties *CryptoKittiesFilterer) WatchPregnant(opts *bind.WatchOpts, sink chan<- *CryptoKittiesPregnant) (event.Subscription, error) {

	logs, sub, err := _CryptoKitties.contract.WatchLogs(opts, "Pregnant")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoKittiesPregnant)
				if err := _CryptoKitties.contract.UnpackLog(event, "Pregnant", log); err != nil {
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

// ParsePregnant is a log parse operation binding the contract event 0x241ea03ca20251805084d27d4440371c34a0b85ff108f6bb5611248f73818b80.
//
// Solidity: event Pregnant(address owner, uint256 matronId, uint256 sireId, uint256 cooldownEndBlock)
func (_CryptoKitties *CryptoKittiesFilterer) ParsePregnant(log types.Log) (*CryptoKittiesPregnant, error) {
	event := new(CryptoKittiesPregnant)
	if err := _CryptoKitties.contract.UnpackLog(event, "Pregnant", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptoKittiesTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the CryptoKitties contract.
type CryptoKittiesTransferIterator struct {
	Event *CryptoKittiesTransfer // Event containing the contract specifics and raw log

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
func (it *CryptoKittiesTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CryptoKittiesTransfer)
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
		it.Event = new(CryptoKittiesTransfer)
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
func (it *CryptoKittiesTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CryptoKittiesTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CryptoKittiesTransfer represents a Transfer event raised by the CryptoKitties contract.
type CryptoKittiesTransfer struct {
	From    common.Address
	To      common.Address
	TokenId *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address from, address to, uint256 tokenId)
func (_CryptoKitties *CryptoKittiesFilterer) FilterTransfer(opts *bind.FilterOpts) (*CryptoKittiesTransferIterator, error) {

	logs, sub, err := _CryptoKitties.contract.FilterLogs(opts, "Transfer")
	if err != nil {
		return nil, err
	}
	return &CryptoKittiesTransferIterator{contract: _CryptoKitties.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address from, address to, uint256 tokenId)
func (_CryptoKitties *CryptoKittiesFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *CryptoKittiesTransfer) (event.Subscription, error) {

	logs, sub, err := _CryptoKitties.contract.WatchLogs(opts, "Transfer")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CryptoKittiesTransfer)
				if err := _CryptoKitties.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address from, address to, uint256 tokenId)
func (_CryptoKitties *CryptoKittiesFilterer) ParseTransfer(log types.Log) (*CryptoKittiesTransfer, error) {
	event := new(CryptoKittiesTransfer)
	if err := _CryptoKitties.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
