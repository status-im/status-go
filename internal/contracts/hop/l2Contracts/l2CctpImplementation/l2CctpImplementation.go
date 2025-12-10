// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package hopL2CctpImplementation

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

// IAMMExactInputParams is an auto generated low-level Go binding around an user-defined struct.
type IAMMExactInputParams struct {
	Path             []byte
	Recipient        common.Address
	AmountIn         *big.Int
	AmountOutMinimum *big.Int
}

// HopL2CctpImplementationMetaData contains all meta data concerning the HopL2CctpImplementation contract.
var HopL2CctpImplementationMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"nativeTokenAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"cctpAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"feeCollectorAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"minBonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256[]\",\"name\":\"chainIds\",\"type\":\"uint256[]\"},{\"internalType\":\"uint32[]\",\"name\":\"domains\",\"type\":\"uint32[]\"},{\"internalType\":\"address\",\"name\":\"bridgedTokenAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"ammAddress\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"cctpNonce\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"}],\"name\":\"CCTPTransferSent\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"activeChainIds\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"amm\",\"outputs\":[{\"internalType\":\"contractIAMM\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridgedToken\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"cctp\",\"outputs\":[{\"internalType\":\"contractICCTP\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"destinationDomains\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeCollectorAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minBonderFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nativeToken\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"}],\"name\":\"send\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"path\",\"type\":\"bytes\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMinimum\",\"type\":\"uint256\"}],\"internalType\":\"structIAMM.ExactInputParams\",\"name\":\"swapParams\",\"type\":\"tuple\"}],\"name\":\"swapAndSend\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// HopL2CctpImplementationABI is the input ABI used to generate the binding from.
// Deprecated: Use HopL2CctpImplementationMetaData.ABI instead.
var HopL2CctpImplementationABI = HopL2CctpImplementationMetaData.ABI

// HopL2CctpImplementation is an auto generated Go binding around an Ethereum contract.
type HopL2CctpImplementation struct {
	HopL2CctpImplementationCaller     // Read-only binding to the contract
	HopL2CctpImplementationTransactor // Write-only binding to the contract
	HopL2CctpImplementationFilterer   // Log filterer for contract events
}

// HopL2CctpImplementationCaller is an auto generated read-only Go binding around an Ethereum contract.
type HopL2CctpImplementationCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2CctpImplementationTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HopL2CctpImplementationTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2CctpImplementationFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HopL2CctpImplementationFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2CctpImplementationSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HopL2CctpImplementationSession struct {
	Contract     *HopL2CctpImplementation // Generic contract binding to set the session for
	CallOpts     bind.CallOpts            // Call options to use throughout this session
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// HopL2CctpImplementationCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HopL2CctpImplementationCallerSession struct {
	Contract *HopL2CctpImplementationCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                  // Call options to use throughout this session
}

// HopL2CctpImplementationTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HopL2CctpImplementationTransactorSession struct {
	Contract     *HopL2CctpImplementationTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                  // Transaction auth options to use throughout this session
}

// HopL2CctpImplementationRaw is an auto generated low-level Go binding around an Ethereum contract.
type HopL2CctpImplementationRaw struct {
	Contract *HopL2CctpImplementation // Generic contract binding to access the raw methods on
}

// HopL2CctpImplementationCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HopL2CctpImplementationCallerRaw struct {
	Contract *HopL2CctpImplementationCaller // Generic read-only contract binding to access the raw methods on
}

// HopL2CctpImplementationTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HopL2CctpImplementationTransactorRaw struct {
	Contract *HopL2CctpImplementationTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHopL2CctpImplementation creates a new instance of HopL2CctpImplementation, bound to a specific deployed contract.
func NewHopL2CctpImplementation(address common.Address, backend bind.ContractBackend) (*HopL2CctpImplementation, error) {
	contract, err := bindHopL2CctpImplementation(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HopL2CctpImplementation{HopL2CctpImplementationCaller: HopL2CctpImplementationCaller{contract: contract}, HopL2CctpImplementationTransactor: HopL2CctpImplementationTransactor{contract: contract}, HopL2CctpImplementationFilterer: HopL2CctpImplementationFilterer{contract: contract}}, nil
}

// NewHopL2CctpImplementationCaller creates a new read-only instance of HopL2CctpImplementation, bound to a specific deployed contract.
func NewHopL2CctpImplementationCaller(address common.Address, caller bind.ContractCaller) (*HopL2CctpImplementationCaller, error) {
	contract, err := bindHopL2CctpImplementation(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HopL2CctpImplementationCaller{contract: contract}, nil
}

// NewHopL2CctpImplementationTransactor creates a new write-only instance of HopL2CctpImplementation, bound to a specific deployed contract.
func NewHopL2CctpImplementationTransactor(address common.Address, transactor bind.ContractTransactor) (*HopL2CctpImplementationTransactor, error) {
	contract, err := bindHopL2CctpImplementation(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HopL2CctpImplementationTransactor{contract: contract}, nil
}

// NewHopL2CctpImplementationFilterer creates a new log filterer instance of HopL2CctpImplementation, bound to a specific deployed contract.
func NewHopL2CctpImplementationFilterer(address common.Address, filterer bind.ContractFilterer) (*HopL2CctpImplementationFilterer, error) {
	contract, err := bindHopL2CctpImplementation(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HopL2CctpImplementationFilterer{contract: contract}, nil
}

// bindHopL2CctpImplementation binds a generic wrapper to an already deployed contract.
func bindHopL2CctpImplementation(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HopL2CctpImplementationMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL2CctpImplementation *HopL2CctpImplementationRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL2CctpImplementation.Contract.HopL2CctpImplementationCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL2CctpImplementation *HopL2CctpImplementationRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2CctpImplementation.Contract.HopL2CctpImplementationTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL2CctpImplementation *HopL2CctpImplementationRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL2CctpImplementation.Contract.HopL2CctpImplementationTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL2CctpImplementation.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL2CctpImplementation *HopL2CctpImplementationTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2CctpImplementation.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL2CctpImplementation *HopL2CctpImplementationTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL2CctpImplementation.Contract.contract.Transact(opts, method, params...)
}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2CctpImplementation *HopL2CctpImplementationCaller) ActiveChainIds(opts *bind.CallOpts, arg0 *big.Int) (bool, error) {
	var out []interface{}
	err := _HopL2CctpImplementation.contract.Call(opts, &out, "activeChainIds", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) ActiveChainIds(arg0 *big.Int) (bool, error) {
	return _HopL2CctpImplementation.Contract.ActiveChainIds(&_HopL2CctpImplementation.CallOpts, arg0)
}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerSession) ActiveChainIds(arg0 *big.Int) (bool, error) {
	return _HopL2CctpImplementation.Contract.ActiveChainIds(&_HopL2CctpImplementation.CallOpts, arg0)
}

// Amm is a free data retrieval call binding the contract method 0x2a943945.
//
// Solidity: function amm() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCaller) Amm(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2CctpImplementation.contract.Call(opts, &out, "amm")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Amm is a free data retrieval call binding the contract method 0x2a943945.
//
// Solidity: function amm() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) Amm() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.Amm(&_HopL2CctpImplementation.CallOpts)
}

// Amm is a free data retrieval call binding the contract method 0x2a943945.
//
// Solidity: function amm() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerSession) Amm() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.Amm(&_HopL2CctpImplementation.CallOpts)
}

// BridgedToken is a free data retrieval call binding the contract method 0xee383937.
//
// Solidity: function bridgedToken() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCaller) BridgedToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2CctpImplementation.contract.Call(opts, &out, "bridgedToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BridgedToken is a free data retrieval call binding the contract method 0xee383937.
//
// Solidity: function bridgedToken() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) BridgedToken() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.BridgedToken(&_HopL2CctpImplementation.CallOpts)
}

// BridgedToken is a free data retrieval call binding the contract method 0xee383937.
//
// Solidity: function bridgedToken() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerSession) BridgedToken() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.BridgedToken(&_HopL2CctpImplementation.CallOpts)
}

// Cctp is a free data retrieval call binding the contract method 0xe3329e32.
//
// Solidity: function cctp() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCaller) Cctp(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2CctpImplementation.contract.Call(opts, &out, "cctp")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Cctp is a free data retrieval call binding the contract method 0xe3329e32.
//
// Solidity: function cctp() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) Cctp() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.Cctp(&_HopL2CctpImplementation.CallOpts)
}

// Cctp is a free data retrieval call binding the contract method 0xe3329e32.
//
// Solidity: function cctp() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerSession) Cctp() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.Cctp(&_HopL2CctpImplementation.CallOpts)
}

// DestinationDomains is a free data retrieval call binding the contract method 0x89aad5dc.
//
// Solidity: function destinationDomains(uint256 ) view returns(uint32)
func (_HopL2CctpImplementation *HopL2CctpImplementationCaller) DestinationDomains(opts *bind.CallOpts, arg0 *big.Int) (uint32, error) {
	var out []interface{}
	err := _HopL2CctpImplementation.contract.Call(opts, &out, "destinationDomains", arg0)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// DestinationDomains is a free data retrieval call binding the contract method 0x89aad5dc.
//
// Solidity: function destinationDomains(uint256 ) view returns(uint32)
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) DestinationDomains(arg0 *big.Int) (uint32, error) {
	return _HopL2CctpImplementation.Contract.DestinationDomains(&_HopL2CctpImplementation.CallOpts, arg0)
}

// DestinationDomains is a free data retrieval call binding the contract method 0x89aad5dc.
//
// Solidity: function destinationDomains(uint256 ) view returns(uint32)
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerSession) DestinationDomains(arg0 *big.Int) (uint32, error) {
	return _HopL2CctpImplementation.Contract.DestinationDomains(&_HopL2CctpImplementation.CallOpts, arg0)
}

// FeeCollectorAddress is a free data retrieval call binding the contract method 0xf108e225.
//
// Solidity: function feeCollectorAddress() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCaller) FeeCollectorAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2CctpImplementation.contract.Call(opts, &out, "feeCollectorAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeCollectorAddress is a free data retrieval call binding the contract method 0xf108e225.
//
// Solidity: function feeCollectorAddress() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) FeeCollectorAddress() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.FeeCollectorAddress(&_HopL2CctpImplementation.CallOpts)
}

// FeeCollectorAddress is a free data retrieval call binding the contract method 0xf108e225.
//
// Solidity: function feeCollectorAddress() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerSession) FeeCollectorAddress() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.FeeCollectorAddress(&_HopL2CctpImplementation.CallOpts)
}

// MinBonderFee is a free data retrieval call binding the contract method 0x50fc2401.
//
// Solidity: function minBonderFee() view returns(uint256)
func (_HopL2CctpImplementation *HopL2CctpImplementationCaller) MinBonderFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2CctpImplementation.contract.Call(opts, &out, "minBonderFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinBonderFee is a free data retrieval call binding the contract method 0x50fc2401.
//
// Solidity: function minBonderFee() view returns(uint256)
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) MinBonderFee() (*big.Int, error) {
	return _HopL2CctpImplementation.Contract.MinBonderFee(&_HopL2CctpImplementation.CallOpts)
}

// MinBonderFee is a free data retrieval call binding the contract method 0x50fc2401.
//
// Solidity: function minBonderFee() view returns(uint256)
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerSession) MinBonderFee() (*big.Int, error) {
	return _HopL2CctpImplementation.Contract.MinBonderFee(&_HopL2CctpImplementation.CallOpts)
}

// NativeToken is a free data retrieval call binding the contract method 0xe1758bd8.
//
// Solidity: function nativeToken() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCaller) NativeToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2CctpImplementation.contract.Call(opts, &out, "nativeToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NativeToken is a free data retrieval call binding the contract method 0xe1758bd8.
//
// Solidity: function nativeToken() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) NativeToken() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.NativeToken(&_HopL2CctpImplementation.CallOpts)
}

// NativeToken is a free data retrieval call binding the contract method 0xe1758bd8.
//
// Solidity: function nativeToken() view returns(address)
func (_HopL2CctpImplementation *HopL2CctpImplementationCallerSession) NativeToken() (common.Address, error) {
	return _HopL2CctpImplementation.Contract.NativeToken(&_HopL2CctpImplementation.CallOpts)
}

// Send is a paid mutator transaction binding the contract method 0xa134ce5b.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee) returns()
func (_HopL2CctpImplementation *HopL2CctpImplementationTransactor) Send(opts *bind.TransactOpts, chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2CctpImplementation.contract.Transact(opts, "send", chainId, recipient, amount, bonderFee)
}

// Send is a paid mutator transaction binding the contract method 0xa134ce5b.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee) returns()
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) Send(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2CctpImplementation.Contract.Send(&_HopL2CctpImplementation.TransactOpts, chainId, recipient, amount, bonderFee)
}

// Send is a paid mutator transaction binding the contract method 0xa134ce5b.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee) returns()
func (_HopL2CctpImplementation *HopL2CctpImplementationTransactorSession) Send(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2CctpImplementation.Contract.Send(&_HopL2CctpImplementation.TransactOpts, chainId, recipient, amount, bonderFee)
}

// SwapAndSend is a paid mutator transaction binding the contract method 0x070d46e4.
//
// Solidity: function swapAndSend(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, (bytes,address,uint256,uint256) swapParams) returns()
func (_HopL2CctpImplementation *HopL2CctpImplementationTransactor) SwapAndSend(opts *bind.TransactOpts, chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, swapParams IAMMExactInputParams) (*types.Transaction, error) {
	return _HopL2CctpImplementation.contract.Transact(opts, "swapAndSend", chainId, recipient, amount, bonderFee, swapParams)
}

// SwapAndSend is a paid mutator transaction binding the contract method 0x070d46e4.
//
// Solidity: function swapAndSend(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, (bytes,address,uint256,uint256) swapParams) returns()
func (_HopL2CctpImplementation *HopL2CctpImplementationSession) SwapAndSend(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, swapParams IAMMExactInputParams) (*types.Transaction, error) {
	return _HopL2CctpImplementation.Contract.SwapAndSend(&_HopL2CctpImplementation.TransactOpts, chainId, recipient, amount, bonderFee, swapParams)
}

// SwapAndSend is a paid mutator transaction binding the contract method 0x070d46e4.
//
// Solidity: function swapAndSend(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, (bytes,address,uint256,uint256) swapParams) returns()
func (_HopL2CctpImplementation *HopL2CctpImplementationTransactorSession) SwapAndSend(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, swapParams IAMMExactInputParams) (*types.Transaction, error) {
	return _HopL2CctpImplementation.Contract.SwapAndSend(&_HopL2CctpImplementation.TransactOpts, chainId, recipient, amount, bonderFee, swapParams)
}

// HopL2CctpImplementationCCTPTransferSentIterator is returned from FilterCCTPTransferSent and is used to iterate over the raw logs and unpacked data for CCTPTransferSent events raised by the HopL2CctpImplementation contract.
type HopL2CctpImplementationCCTPTransferSentIterator struct {
	Event *HopL2CctpImplementationCCTPTransferSent // Event containing the contract specifics and raw log

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
func (it *HopL2CctpImplementationCCTPTransferSentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2CctpImplementationCCTPTransferSent)
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
		it.Event = new(HopL2CctpImplementationCCTPTransferSent)
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
func (it *HopL2CctpImplementationCCTPTransferSentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2CctpImplementationCCTPTransferSentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2CctpImplementationCCTPTransferSent represents a CCTPTransferSent event raised by the HopL2CctpImplementation contract.
type HopL2CctpImplementationCCTPTransferSent struct {
	CctpNonce uint64
	ChainId   *big.Int
	Recipient common.Address
	Amount    *big.Int
	BonderFee *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCCTPTransferSent is a free log retrieval operation binding the contract event 0x10bf4019e09db5876a05d237bfcc676cd84eee2c23f820284906dd7cfa70d2c4.
//
// Solidity: event CCTPTransferSent(uint64 indexed cctpNonce, uint256 indexed chainId, address indexed recipient, uint256 amount, uint256 bonderFee)
func (_HopL2CctpImplementation *HopL2CctpImplementationFilterer) FilterCCTPTransferSent(opts *bind.FilterOpts, cctpNonce []uint64, chainId []*big.Int, recipient []common.Address) (*HopL2CctpImplementationCCTPTransferSentIterator, error) {

	var cctpNonceRule []interface{}
	for _, cctpNonceItem := range cctpNonce {
		cctpNonceRule = append(cctpNonceRule, cctpNonceItem)
	}
	var chainIdRule []interface{}
	for _, chainIdItem := range chainId {
		chainIdRule = append(chainIdRule, chainIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL2CctpImplementation.contract.FilterLogs(opts, "CCTPTransferSent", cctpNonceRule, chainIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &HopL2CctpImplementationCCTPTransferSentIterator{contract: _HopL2CctpImplementation.contract, event: "CCTPTransferSent", logs: logs, sub: sub}, nil
}

// WatchCCTPTransferSent is a free log subscription operation binding the contract event 0x10bf4019e09db5876a05d237bfcc676cd84eee2c23f820284906dd7cfa70d2c4.
//
// Solidity: event CCTPTransferSent(uint64 indexed cctpNonce, uint256 indexed chainId, address indexed recipient, uint256 amount, uint256 bonderFee)
func (_HopL2CctpImplementation *HopL2CctpImplementationFilterer) WatchCCTPTransferSent(opts *bind.WatchOpts, sink chan<- *HopL2CctpImplementationCCTPTransferSent, cctpNonce []uint64, chainId []*big.Int, recipient []common.Address) (event.Subscription, error) {

	var cctpNonceRule []interface{}
	for _, cctpNonceItem := range cctpNonce {
		cctpNonceRule = append(cctpNonceRule, cctpNonceItem)
	}
	var chainIdRule []interface{}
	for _, chainIdItem := range chainId {
		chainIdRule = append(chainIdRule, chainIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL2CctpImplementation.contract.WatchLogs(opts, "CCTPTransferSent", cctpNonceRule, chainIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2CctpImplementationCCTPTransferSent)
				if err := _HopL2CctpImplementation.contract.UnpackLog(event, "CCTPTransferSent", log); err != nil {
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

// ParseCCTPTransferSent is a log parse operation binding the contract event 0x10bf4019e09db5876a05d237bfcc676cd84eee2c23f820284906dd7cfa70d2c4.
//
// Solidity: event CCTPTransferSent(uint64 indexed cctpNonce, uint256 indexed chainId, address indexed recipient, uint256 amount, uint256 bonderFee)
func (_HopL2CctpImplementation *HopL2CctpImplementationFilterer) ParseCCTPTransferSent(log types.Log) (*HopL2CctpImplementationCCTPTransferSent, error) {
	event := new(HopL2CctpImplementationCCTPTransferSent)
	if err := _HopL2CctpImplementation.contract.UnpackLog(event, "CCTPTransferSent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
