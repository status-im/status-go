// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

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

// RLNMetaData contains all meta data concerning the RLN contract.
var RLNMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"membershipDeposit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"depth\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_poseidonHasher\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"DuplicateIdCommitment\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyBatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FullBatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"required\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"provided\",\"type\":\"uint256\"}],\"name\":\"InsufficientDeposit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"InvalidWithdrawalAddress\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"}],\"name\":\"MemberHasNoStake\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"}],\"name\":\"MemberNotRegistered\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"givenSecretsLen\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"givenReceiversLen\",\"type\":\"uint256\"}],\"name\":\"MismatchedBatchSize\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"MemberRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"}],\"name\":\"MemberWithdrawn\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEPTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MEMBERSHIP_DEPOSIT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SET_SIZE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"idCommitmentIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"members\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poseidonHasher\",\"outputs\":[{\"internalType\":\"contractIPoseidonHasher\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"}],\"name\":\"register\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"idCommitments\",\"type\":\"uint256[]\"}],\"name\":\"registerBatch\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"stakedAmounts\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"secret\",\"type\":\"uint256\"},{\"internalType\":\"addresspayable\",\"name\":\"receiver\",\"type\":\"address\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"secrets\",\"type\":\"uint256[]\"},{\"internalType\":\"addresspayable[]\",\"name\":\"receivers\",\"type\":\"address[]\"}],\"name\":\"withdrawBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60e06040523480156200001157600080fd5b50604051620012c0380380620012c0833981810160405281019062000037919062000142565b82608081815250508160a08181525050816001901b60c0818152505080600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505050506200019e565b600080fd5b6000819050919050565b620000b781620000a2565b8114620000c357600080fd5b50565b600081519050620000d781620000ac565b92915050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006200010a82620000dd565b9050919050565b6200011c81620000fd565b81146200012857600080fd5b50565b6000815190506200013c8162000111565b92915050565b6000806000606084860312156200015e576200015d6200009d565b5b60006200016e86828701620000c6565b93505060206200018186828701620000c6565b925050604062000194868287016200012b565b9150509250925092565b60805160a05160c0516110cf620001f1600039600081816104140152818161058501526109070152600061054301526000818161047d015281816105a9015281816105d0015261063c01526110cf6000f3fe60806040526004361061009b5760003560e01c806398366e351161006457806398366e3514610176578063ae74552a146101a1578063bc499128146101cc578063d0383d6814610209578063f207564e14610234578063f220b9ec146102505761009b565b8062f714ce146100a0578063331b6ab3146100c957806340070712146100f45780635daf08ca1461011d57806369e4863f1461015a575b600080fd5b3480156100ac57600080fd5b506100c760048036038101906100c29190610b3f565b61027b565b005b3480156100d557600080fd5b506100de610289565b6040516100eb9190610bde565b60405180910390f35b34801561010057600080fd5b5061011b60048036038101906101169190610cb4565b6102af565b005b34801561012957600080fd5b50610144600480360381019061013f9190610d35565b6103b0565b6040516101519190610d7d565b60405180910390f35b610174600480360381019061016f9190610d98565b6103d0565b005b34801561018257600080fd5b5061018b610541565b6040516101989190610df4565b60405180910390f35b3480156101ad57600080fd5b506101b6610565565b6040516101c39190610df4565b60405180910390f35b3480156101d857600080fd5b506101f360048036038101906101ee9190610d35565b61056b565b6040516102009190610df4565b60405180910390f35b34801561021557600080fd5b5061021e610583565b60405161022b9190610df4565b60405180910390f35b61024e60048036038101906102499190610d35565b6105a7565b005b34801561025c57600080fd5b5061026561063a565b6040516102729190610df4565b60405180910390f35b610285828261065e565b5050565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000848490509050600081036102f1576040517fc2e5347d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8282905081146103405784849050838390506040517f727c6a75000000000000000000000000000000000000000000000000000000008152600401610337929190610e0f565b60405180910390fd5b60005b818110156103a85761039586868381811061036157610360610e38565b5b9050602002013585858481811061037b5761037a610e38565b5b90506020020160208101906103909190610e67565b61065e565b80806103a090610ec3565b915050610343565b505050505050565b60026020528060005260406000206000915054906101000a900460ff1681565b600082829050905060008103610412576040517fc2e5347d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b7f0000000000000000000000000000000000000000000000000000000000000000816000546104419190610f0b565b10610478576040517f75eb4dbe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000817f00000000000000000000000000000000000000000000000000000000000000006104a69190610f61565b90508034146104ee5780346040517f25c3f46e0000000000000000000000000000000000000000000000000000000081526004016104e5929190610e0f565b60405180910390fd5b60005b8281101561053a5761052785858381811061050f5761050e610e38565b5b9050602002013584346105229190610fea565b6108ae565b808061053290610ec3565b9150506104f1565b5050505050565b7f000000000000000000000000000000000000000000000000000000000000000081565b60005481565b60016020528060005260406000206000915090505481565b7f000000000000000000000000000000000000000000000000000000000000000081565b7f0000000000000000000000000000000000000000000000000000000000000000341461062d577f0000000000000000000000000000000000000000000000000000000000000000346040517f25c3f46e000000000000000000000000000000000000000000000000000000008152600401610624929190610e0f565b60405180910390fd5b61063781346108ae565b50565b7f000000000000000000000000000000000000000000000000000000000000000081565b3073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614806106c45750600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16145b1561070657806040517f21680a040000000000000000000000000000000000000000000000000000000081526004016106fd919061103c565b60405180910390fd5b6000610711836109fc565b90506002600082815260200190815260200160002060009054906101000a900460ff1661077557806040517f5a971ebb00000000000000000000000000000000000000000000000000000000815260040161076c9190610df4565b60405180910390fd5b60006001600083815260200190815260200160002054036107cd57806040517faabeeba50000000000000000000000000000000000000000000000000000000081526004016107c49190610df4565b60405180910390fd5b60006001600083815260200190815260200160002054905060006002600084815260200190815260200160002060006101000a81548160ff021916908315150217905550600060016000848152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f19350505050158015610870573d6000803e3d6000fd5b507fad2d771c5ad1c1e6f50cc769e53ec1e194002c29f28c3dd2af5639b60d8072a6826040516108a09190610df4565b60405180910390a150505050565b6002600083815260200190815260200160002060009054906101000a900460ff1615610905576040517e0a60f700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b7f000000000000000000000000000000000000000000000000000000000000000060005410610960576040517f75eb4dbe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60016002600084815260200190815260200160002060006101000a81548160ff0219169083151502179055508060016000848152602001908152602001600020819055507f5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2826000546040516109d7929190610e0f565b60405180910390a160016000808282546109f19190610f0b565b925050819055505050565b6000600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663b189fd4c836040518263ffffffff1660e01b8152600401610a599190610df4565b602060405180830381865afa158015610a76573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610a9a919061106c565b9050919050565b600080fd5b600080fd5b6000819050919050565b610abe81610aab565b8114610ac957600080fd5b50565b600081359050610adb81610ab5565b92915050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610b0c82610ae1565b9050919050565b610b1c81610b01565b8114610b2757600080fd5b50565b600081359050610b3981610b13565b92915050565b60008060408385031215610b5657610b55610aa1565b5b6000610b6485828601610acc565b9250506020610b7585828601610b2a565b9150509250929050565b6000819050919050565b6000610ba4610b9f610b9a84610ae1565b610b7f565b610ae1565b9050919050565b6000610bb682610b89565b9050919050565b6000610bc882610bab565b9050919050565b610bd881610bbd565b82525050565b6000602082019050610bf36000830184610bcf565b92915050565b600080fd5b600080fd5b600080fd5b60008083601f840112610c1e57610c1d610bf9565b5b8235905067ffffffffffffffff811115610c3b57610c3a610bfe565b5b602083019150836020820283011115610c5757610c56610c03565b5b9250929050565b60008083601f840112610c7457610c73610bf9565b5b8235905067ffffffffffffffff811115610c9157610c90610bfe565b5b602083019150836020820283011115610cad57610cac610c03565b5b9250929050565b60008060008060408587031215610cce57610ccd610aa1565b5b600085013567ffffffffffffffff811115610cec57610ceb610aa6565b5b610cf887828801610c08565b9450945050602085013567ffffffffffffffff811115610d1b57610d1a610aa6565b5b610d2787828801610c5e565b925092505092959194509250565b600060208284031215610d4b57610d4a610aa1565b5b6000610d5984828501610acc565b91505092915050565b60008115159050919050565b610d7781610d62565b82525050565b6000602082019050610d926000830184610d6e565b92915050565b60008060208385031215610daf57610dae610aa1565b5b600083013567ffffffffffffffff811115610dcd57610dcc610aa6565b5b610dd985828601610c08565b92509250509250929050565b610dee81610aab565b82525050565b6000602082019050610e096000830184610de5565b92915050565b6000604082019050610e246000830185610de5565b610e316020830184610de5565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600060208284031215610e7d57610e7c610aa1565b5b6000610e8b84828501610b2a565b91505092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610ece82610aab565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610f0057610eff610e94565b5b600182019050919050565b6000610f1682610aab565b9150610f2183610aab565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff03821115610f5657610f55610e94565b5b828201905092915050565b6000610f6c82610aab565b9150610f7783610aab565b9250817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615610fb057610faf610e94565b5b828202905092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b6000610ff582610aab565b915061100083610aab565b9250826110105761100f610fbb565b5b828204905092915050565b600061102682610bab565b9050919050565b6110368161101b565b82525050565b6000602082019050611051600083018461102d565b92915050565b60008151905061106681610ab5565b92915050565b60006020828403121561108257611081610aa1565b5b600061109084828501611057565b9150509291505056fea26469706673582212201eba48f9e121352ff146b50c2c2a98d3cbb52bcd90bf3d5e555aa60d7c6778f064736f6c634300080f0033",
}

// RLNABI is the input ABI used to generate the binding from.
// Deprecated: Use RLNMetaData.ABI instead.
var RLNABI = RLNMetaData.ABI

// RLNBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RLNMetaData.Bin instead.
var RLNBin = RLNMetaData.Bin

// DeployRLN deploys a new Ethereum contract, binding an instance of RLN to it.
func DeployRLN(auth *bind.TransactOpts, backend bind.ContractBackend, membershipDeposit *big.Int, depth *big.Int, _poseidonHasher common.Address) (common.Address, *types.Transaction, *RLN, error) {
	parsed, err := RLNMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RLNBin), backend, membershipDeposit, depth, _poseidonHasher)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &RLN{RLNCaller: RLNCaller{contract: contract}, RLNTransactor: RLNTransactor{contract: contract}, RLNFilterer: RLNFilterer{contract: contract}}, nil
}

// RLN is an auto generated Go binding around an Ethereum contract.
type RLN struct {
	RLNCaller     // Read-only binding to the contract
	RLNTransactor // Write-only binding to the contract
	RLNFilterer   // Log filterer for contract events
}

// RLNCaller is an auto generated read-only Go binding around an Ethereum contract.
type RLNCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RLNTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RLNTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RLNFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RLNFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RLNSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RLNSession struct {
	Contract     *RLN              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RLNCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RLNCallerSession struct {
	Contract *RLNCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// RLNTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RLNTransactorSession struct {
	Contract     *RLNTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RLNRaw is an auto generated low-level Go binding around an Ethereum contract.
type RLNRaw struct {
	Contract *RLN // Generic contract binding to access the raw methods on
}

// RLNCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RLNCallerRaw struct {
	Contract *RLNCaller // Generic read-only contract binding to access the raw methods on
}

// RLNTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RLNTransactorRaw struct {
	Contract *RLNTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRLN creates a new instance of RLN, bound to a specific deployed contract.
func NewRLN(address common.Address, backend bind.ContractBackend) (*RLN, error) {
	contract, err := bindRLN(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RLN{RLNCaller: RLNCaller{contract: contract}, RLNTransactor: RLNTransactor{contract: contract}, RLNFilterer: RLNFilterer{contract: contract}}, nil
}

// NewRLNCaller creates a new read-only instance of RLN, bound to a specific deployed contract.
func NewRLNCaller(address common.Address, caller bind.ContractCaller) (*RLNCaller, error) {
	contract, err := bindRLN(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RLNCaller{contract: contract}, nil
}

// NewRLNTransactor creates a new write-only instance of RLN, bound to a specific deployed contract.
func NewRLNTransactor(address common.Address, transactor bind.ContractTransactor) (*RLNTransactor, error) {
	contract, err := bindRLN(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RLNTransactor{contract: contract}, nil
}

// NewRLNFilterer creates a new log filterer instance of RLN, bound to a specific deployed contract.
func NewRLNFilterer(address common.Address, filterer bind.ContractFilterer) (*RLNFilterer, error) {
	contract, err := bindRLN(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RLNFilterer{contract: contract}, nil
}

// bindRLN binds a generic wrapper to an already deployed contract.
func bindRLN(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RLNMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RLN *RLNRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RLN.Contract.RLNCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RLN *RLNRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RLN.Contract.RLNTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RLN *RLNRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RLN.Contract.RLNTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RLN *RLNCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RLN.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RLN *RLNTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RLN.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RLN *RLNTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RLN.Contract.contract.Transact(opts, method, params...)
}

// DEPTH is a free data retrieval call binding the contract method 0x98366e35.
//
// Solidity: function DEPTH() view returns(uint256)
func (_RLN *RLNCaller) DEPTH(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "DEPTH")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DEPTH is a free data retrieval call binding the contract method 0x98366e35.
//
// Solidity: function DEPTH() view returns(uint256)
func (_RLN *RLNSession) DEPTH() (*big.Int, error) {
	return _RLN.Contract.DEPTH(&_RLN.CallOpts)
}

// DEPTH is a free data retrieval call binding the contract method 0x98366e35.
//
// Solidity: function DEPTH() view returns(uint256)
func (_RLN *RLNCallerSession) DEPTH() (*big.Int, error) {
	return _RLN.Contract.DEPTH(&_RLN.CallOpts)
}

// MEMBERSHIPDEPOSIT is a free data retrieval call binding the contract method 0xf220b9ec.
//
// Solidity: function MEMBERSHIP_DEPOSIT() view returns(uint256)
func (_RLN *RLNCaller) MEMBERSHIPDEPOSIT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "MEMBERSHIP_DEPOSIT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MEMBERSHIPDEPOSIT is a free data retrieval call binding the contract method 0xf220b9ec.
//
// Solidity: function MEMBERSHIP_DEPOSIT() view returns(uint256)
func (_RLN *RLNSession) MEMBERSHIPDEPOSIT() (*big.Int, error) {
	return _RLN.Contract.MEMBERSHIPDEPOSIT(&_RLN.CallOpts)
}

// MEMBERSHIPDEPOSIT is a free data retrieval call binding the contract method 0xf220b9ec.
//
// Solidity: function MEMBERSHIP_DEPOSIT() view returns(uint256)
func (_RLN *RLNCallerSession) MEMBERSHIPDEPOSIT() (*big.Int, error) {
	return _RLN.Contract.MEMBERSHIPDEPOSIT(&_RLN.CallOpts)
}

// SETSIZE is a free data retrieval call binding the contract method 0xd0383d68.
//
// Solidity: function SET_SIZE() view returns(uint256)
func (_RLN *RLNCaller) SETSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "SET_SIZE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SETSIZE is a free data retrieval call binding the contract method 0xd0383d68.
//
// Solidity: function SET_SIZE() view returns(uint256)
func (_RLN *RLNSession) SETSIZE() (*big.Int, error) {
	return _RLN.Contract.SETSIZE(&_RLN.CallOpts)
}

// SETSIZE is a free data retrieval call binding the contract method 0xd0383d68.
//
// Solidity: function SET_SIZE() view returns(uint256)
func (_RLN *RLNCallerSession) SETSIZE() (*big.Int, error) {
	return _RLN.Contract.SETSIZE(&_RLN.CallOpts)
}

// IdCommitmentIndex is a free data retrieval call binding the contract method 0xae74552a.
//
// Solidity: function idCommitmentIndex() view returns(uint256)
func (_RLN *RLNCaller) IdCommitmentIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "idCommitmentIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// IdCommitmentIndex is a free data retrieval call binding the contract method 0xae74552a.
//
// Solidity: function idCommitmentIndex() view returns(uint256)
func (_RLN *RLNSession) IdCommitmentIndex() (*big.Int, error) {
	return _RLN.Contract.IdCommitmentIndex(&_RLN.CallOpts)
}

// IdCommitmentIndex is a free data retrieval call binding the contract method 0xae74552a.
//
// Solidity: function idCommitmentIndex() view returns(uint256)
func (_RLN *RLNCallerSession) IdCommitmentIndex() (*big.Int, error) {
	return _RLN.Contract.IdCommitmentIndex(&_RLN.CallOpts)
}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(bool)
func (_RLN *RLNCaller) Members(opts *bind.CallOpts, arg0 *big.Int) (bool, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "members", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(bool)
func (_RLN *RLNSession) Members(arg0 *big.Int) (bool, error) {
	return _RLN.Contract.Members(&_RLN.CallOpts, arg0)
}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(bool)
func (_RLN *RLNCallerSession) Members(arg0 *big.Int) (bool, error) {
	return _RLN.Contract.Members(&_RLN.CallOpts, arg0)
}

// PoseidonHasher is a free data retrieval call binding the contract method 0x331b6ab3.
//
// Solidity: function poseidonHasher() view returns(address)
func (_RLN *RLNCaller) PoseidonHasher(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "poseidonHasher")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoseidonHasher is a free data retrieval call binding the contract method 0x331b6ab3.
//
// Solidity: function poseidonHasher() view returns(address)
func (_RLN *RLNSession) PoseidonHasher() (common.Address, error) {
	return _RLN.Contract.PoseidonHasher(&_RLN.CallOpts)
}

// PoseidonHasher is a free data retrieval call binding the contract method 0x331b6ab3.
//
// Solidity: function poseidonHasher() view returns(address)
func (_RLN *RLNCallerSession) PoseidonHasher() (common.Address, error) {
	return _RLN.Contract.PoseidonHasher(&_RLN.CallOpts)
}

// StakedAmounts is a free data retrieval call binding the contract method 0xbc499128.
//
// Solidity: function stakedAmounts(uint256 ) view returns(uint256)
func (_RLN *RLNCaller) StakedAmounts(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "stakedAmounts", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StakedAmounts is a free data retrieval call binding the contract method 0xbc499128.
//
// Solidity: function stakedAmounts(uint256 ) view returns(uint256)
func (_RLN *RLNSession) StakedAmounts(arg0 *big.Int) (*big.Int, error) {
	return _RLN.Contract.StakedAmounts(&_RLN.CallOpts, arg0)
}

// StakedAmounts is a free data retrieval call binding the contract method 0xbc499128.
//
// Solidity: function stakedAmounts(uint256 ) view returns(uint256)
func (_RLN *RLNCallerSession) StakedAmounts(arg0 *big.Int) (*big.Int, error) {
	return _RLN.Contract.StakedAmounts(&_RLN.CallOpts, arg0)
}

// Register is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 idCommitment) payable returns()
func (_RLN *RLNTransactor) Register(opts *bind.TransactOpts, idCommitment *big.Int) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "register", idCommitment)
}

// Register is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 idCommitment) payable returns()
func (_RLN *RLNSession) Register(idCommitment *big.Int) (*types.Transaction, error) {
	return _RLN.Contract.Register(&_RLN.TransactOpts, idCommitment)
}

// Register is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 idCommitment) payable returns()
func (_RLN *RLNTransactorSession) Register(idCommitment *big.Int) (*types.Transaction, error) {
	return _RLN.Contract.Register(&_RLN.TransactOpts, idCommitment)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x69e4863f.
//
// Solidity: function registerBatch(uint256[] idCommitments) payable returns()
func (_RLN *RLNTransactor) RegisterBatch(opts *bind.TransactOpts, idCommitments []*big.Int) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "registerBatch", idCommitments)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x69e4863f.
//
// Solidity: function registerBatch(uint256[] idCommitments) payable returns()
func (_RLN *RLNSession) RegisterBatch(idCommitments []*big.Int) (*types.Transaction, error) {
	return _RLN.Contract.RegisterBatch(&_RLN.TransactOpts, idCommitments)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x69e4863f.
//
// Solidity: function registerBatch(uint256[] idCommitments) payable returns()
func (_RLN *RLNTransactorSession) RegisterBatch(idCommitments []*big.Int) (*types.Transaction, error) {
	return _RLN.Contract.RegisterBatch(&_RLN.TransactOpts, idCommitments)
}

// Withdraw is a paid mutator transaction binding the contract method 0x00f714ce.
//
// Solidity: function withdraw(uint256 secret, address receiver) returns()
func (_RLN *RLNTransactor) Withdraw(opts *bind.TransactOpts, secret *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "withdraw", secret, receiver)
}

// Withdraw is a paid mutator transaction binding the contract method 0x00f714ce.
//
// Solidity: function withdraw(uint256 secret, address receiver) returns()
func (_RLN *RLNSession) Withdraw(secret *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _RLN.Contract.Withdraw(&_RLN.TransactOpts, secret, receiver)
}

// Withdraw is a paid mutator transaction binding the contract method 0x00f714ce.
//
// Solidity: function withdraw(uint256 secret, address receiver) returns()
func (_RLN *RLNTransactorSession) Withdraw(secret *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _RLN.Contract.Withdraw(&_RLN.TransactOpts, secret, receiver)
}

// WithdrawBatch is a paid mutator transaction binding the contract method 0x40070712.
//
// Solidity: function withdrawBatch(uint256[] secrets, address[] receivers) returns()
func (_RLN *RLNTransactor) WithdrawBatch(opts *bind.TransactOpts, secrets []*big.Int, receivers []common.Address) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "withdrawBatch", secrets, receivers)
}

// WithdrawBatch is a paid mutator transaction binding the contract method 0x40070712.
//
// Solidity: function withdrawBatch(uint256[] secrets, address[] receivers) returns()
func (_RLN *RLNSession) WithdrawBatch(secrets []*big.Int, receivers []common.Address) (*types.Transaction, error) {
	return _RLN.Contract.WithdrawBatch(&_RLN.TransactOpts, secrets, receivers)
}

// WithdrawBatch is a paid mutator transaction binding the contract method 0x40070712.
//
// Solidity: function withdrawBatch(uint256[] secrets, address[] receivers) returns()
func (_RLN *RLNTransactorSession) WithdrawBatch(secrets []*big.Int, receivers []common.Address) (*types.Transaction, error) {
	return _RLN.Contract.WithdrawBatch(&_RLN.TransactOpts, secrets, receivers)
}

// RLNMemberRegisteredIterator is returned from FilterMemberRegistered and is used to iterate over the raw logs and unpacked data for MemberRegistered events raised by the RLN contract.
type RLNMemberRegisteredIterator struct {
	Event *RLNMemberRegistered // Event containing the contract specifics and raw log

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
func (it *RLNMemberRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RLNMemberRegistered)
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
		it.Event = new(RLNMemberRegistered)
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
func (it *RLNMemberRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RLNMemberRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RLNMemberRegistered represents a MemberRegistered event raised by the RLN contract.
type RLNMemberRegistered struct {
	IdCommitment *big.Int
	Index        *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterMemberRegistered is a free log retrieval operation binding the contract event 0x5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2.
//
// Solidity: event MemberRegistered(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) FilterMemberRegistered(opts *bind.FilterOpts) (*RLNMemberRegisteredIterator, error) {

	logs, sub, err := _RLN.contract.FilterLogs(opts, "MemberRegistered")
	if err != nil {
		return nil, err
	}
	return &RLNMemberRegisteredIterator{contract: _RLN.contract, event: "MemberRegistered", logs: logs, sub: sub}, nil
}

// WatchMemberRegistered is a free log subscription operation binding the contract event 0x5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2.
//
// Solidity: event MemberRegistered(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) WatchMemberRegistered(opts *bind.WatchOpts, sink chan<- *RLNMemberRegistered) (event.Subscription, error) {

	logs, sub, err := _RLN.contract.WatchLogs(opts, "MemberRegistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RLNMemberRegistered)
				if err := _RLN.contract.UnpackLog(event, "MemberRegistered", log); err != nil {
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

// ParseMemberRegistered is a log parse operation binding the contract event 0x5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2.
//
// Solidity: event MemberRegistered(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) ParseMemberRegistered(log types.Log) (*RLNMemberRegistered, error) {
	event := new(RLNMemberRegistered)
	if err := _RLN.contract.UnpackLog(event, "MemberRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RLNMemberWithdrawnIterator is returned from FilterMemberWithdrawn and is used to iterate over the raw logs and unpacked data for MemberWithdrawn events raised by the RLN contract.
type RLNMemberWithdrawnIterator struct {
	Event *RLNMemberWithdrawn // Event containing the contract specifics and raw log

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
func (it *RLNMemberWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RLNMemberWithdrawn)
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
		it.Event = new(RLNMemberWithdrawn)
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
func (it *RLNMemberWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RLNMemberWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RLNMemberWithdrawn represents a MemberWithdrawn event raised by the RLN contract.
type RLNMemberWithdrawn struct {
	IdCommitment *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterMemberWithdrawn is a free log retrieval operation binding the contract event 0xad2d771c5ad1c1e6f50cc769e53ec1e194002c29f28c3dd2af5639b60d8072a6.
//
// Solidity: event MemberWithdrawn(uint256 idCommitment)
func (_RLN *RLNFilterer) FilterMemberWithdrawn(opts *bind.FilterOpts) (*RLNMemberWithdrawnIterator, error) {

	logs, sub, err := _RLN.contract.FilterLogs(opts, "MemberWithdrawn")
	if err != nil {
		return nil, err
	}
	return &RLNMemberWithdrawnIterator{contract: _RLN.contract, event: "MemberWithdrawn", logs: logs, sub: sub}, nil
}

// WatchMemberWithdrawn is a free log subscription operation binding the contract event 0xad2d771c5ad1c1e6f50cc769e53ec1e194002c29f28c3dd2af5639b60d8072a6.
//
// Solidity: event MemberWithdrawn(uint256 idCommitment)
func (_RLN *RLNFilterer) WatchMemberWithdrawn(opts *bind.WatchOpts, sink chan<- *RLNMemberWithdrawn) (event.Subscription, error) {

	logs, sub, err := _RLN.contract.WatchLogs(opts, "MemberWithdrawn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RLNMemberWithdrawn)
				if err := _RLN.contract.UnpackLog(event, "MemberWithdrawn", log); err != nil {
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

// ParseMemberWithdrawn is a log parse operation binding the contract event 0xad2d771c5ad1c1e6f50cc769e53ec1e194002c29f28c3dd2af5639b60d8072a6.
//
// Solidity: event MemberWithdrawn(uint256 idCommitment)
func (_RLN *RLNFilterer) ParseMemberWithdrawn(log types.Log) (*RLNMemberWithdrawn, error) {
	event := new(RLNMemberWithdrawn)
	if err := _RLN.contract.UnpackLog(event, "MemberWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
