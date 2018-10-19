// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package eip712example

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ExampleABI is the input ABI used to generate the binding from.
const ExampleABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"test\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// ExampleBin is the compiled bytecode used for deploying new contracts.
const ExampleBin = `0x608060405234801561001057600080fd5b506040805160c081018252600a608082019081527f4574686572204d61696c0000000000000000000000000000000000000000000060a083015281528151808301835260018082527f31000000000000000000000000000000000000000000000000000000000000006020838101919091528301919091529181019190915273cccccccccccccccccccccccccccccccccccccccc60608201526100bb906401000000006100c3810204565b6000556102aa565b600060405180807f454950373132446f6d61696e28737472696e67206e616d652c737472696e672081526020017f76657273696f6e2c75696e7432353620636861696e49642c616464726573732081526020017f766572696679696e67436f6e74726163742900000000000000000000000000008152506052019050604051809103902082600001516040518082805190602001908083835b6020831061017b5780518252601f19909201916020918201910161015c565b51815160209384036101000a6000190180199092169116179052604051919093018190038120888401518051919650945090928392508401908083835b602083106101d75780518252601f1990920191602091820191016101b8565b51815160209384036101000a6000190180199092169116179052604080519290940182900382208a8501516060808d01518585019b909b5284870199909952978301526080820196909652600160a060020a0390961660a0808801919091528251808803909101815260c0909601918290525084519093849350850191508083835b602083106102785780518252601f199092019160209182019101610259565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b6106e4806102b96000396000f30060806040526004361061004b5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416633644e5158114610050578063f8a8fd6d14610077575b600080fd5b34801561005c57600080fd5b506100656100a0565b60408051918252519081900360200190f35b34801561008357600080fd5b5061008c6100a6565b604080519115158252519081900360200190f35b60005481565b60006100b0610672565b506040805160e081018252600360a082018181527f436f77000000000000000000000000000000000000000000000000000000000060c0840152606083810191825273cd2a3d9f938e13cd947ec05abc7fe734df8dd826608080860191909152918452845191820185528185019283527f426f6200000000000000000000000000000000000000000000000000000000009082015290815273bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb6020828101919091528083019190915282518084018452600b81527f48656c6c6f2c20426f62210000000000000000000000000000000000000000009181019190915291810191909152600054601c907f4355c47d63924e8a72e509b65029052eb6c299d53a04e167c5775fd466751c9d907f07299936d304c153f6443dfa05f40ff007d72911b6f72307f996231605b91562907ff2cee375fa42b42143804025fc449deafd50cc031ca257e0b194a650a912090f1461021957fe5b61022284610269565b7fc52c0ee5d84264471806290a3f2c4cecfc5490626bf912d01f240d7a274b371e1461024a57fe5b610256848484846103f1565b151561025e57fe5b600194505050505090565b604080517f4d61696c28506572736f6e2066726f6d2c506572736f6e20746f2c737472696e81527f6720636f6e74656e747329506572736f6e28737472696e67206e616d652c616460208201527f64726573732077616c6c6574290000000000000000000000000000000000000081830152905190819003604d0190208151600091906102f590610543565b6103028460200151610543565b84604001516040518082805190602001908083835b602083106103365780518252601f199092019160209182019101610317565b51815160001960209485036101000a019081169019919091161790526040805194909201849003842084820199909952838201979097526060830195909552506080808201969096528351808203909601865260a001928390525050825190918291908401908083835b602083106103bf5780518252601f1990920191602091820191016103a0565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b60008060005461040087610269565b604080517f19010000000000000000000000000000000000000000000000000000000000006020808301919091526022820194909452604280820193909352815180820390930183526062019081905281519192909182918401908083835b6020831061047e5780518252601f19909201916020918201910161045f565b51815160209384036101000a6000190180199092169116179052604080519290940182900382208c51820151600080855284840180885283905260ff8e1685880152606085018d9052608085018c9052955191985073ffffffffffffffffffffffffffffffffffffffff1696506001955060a080840195929450601f198201938290030191865af1158015610517573d6000803e3d6000fd5b5050506020604051035173ffffffffffffffffffffffffffffffffffffffff1614915050949350505050565b600060405180807f506572736f6e28737472696e67206e616d652c616464726573732077616c6c6581526020017f74290000000000000000000000000000000000000000000000000000000000008152506022019050604051809103902082600001516040518082805190602001908083835b602083106105d55780518252601f1990920191602091820191016105b6565b51815160001960209485036101000a019081169019919091161790526040805194909201849003842089820151858301989098528483015273ffffffffffffffffffffffffffffffffffffffff90961660608085019190915281518085039091018152608090930190819052825192959094508493508501919050808383602083106103bf5780518252601f1990920191602091820191016103a0565b60a0604051908101604052806106866106a0565b81526020016106936106a0565b8152602001606081525090565b604080518082019091526060815260006020820152905600a165627a7a72305820c1968508d71edfda0c4259f288d0c6e186819f90ce99caa7617bd2dadc0492630029`

// DeployExample deploys a new Ethereum contract, binding an instance of Example to it.
func DeployExample(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Example, error) {
	parsed, err := abi.JSON(strings.NewReader(ExampleABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ExampleBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Example{ExampleCaller: ExampleCaller{contract: contract}, ExampleTransactor: ExampleTransactor{contract: contract}, ExampleFilterer: ExampleFilterer{contract: contract}}, nil
}

// Example is an auto generated Go binding around an Ethereum contract.
type Example struct {
	ExampleCaller     // Read-only binding to the contract
	ExampleTransactor // Write-only binding to the contract
	ExampleFilterer   // Log filterer for contract events
}

// ExampleCaller is an auto generated read-only Go binding around an Ethereum contract.
type ExampleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExampleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ExampleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExampleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ExampleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExampleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ExampleSession struct {
	Contract     *Example          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ExampleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ExampleCallerSession struct {
	Contract *ExampleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// ExampleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ExampleTransactorSession struct {
	Contract     *ExampleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ExampleRaw is an auto generated low-level Go binding around an Ethereum contract.
type ExampleRaw struct {
	Contract *Example // Generic contract binding to access the raw methods on
}

// ExampleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ExampleCallerRaw struct {
	Contract *ExampleCaller // Generic read-only contract binding to access the raw methods on
}

// ExampleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ExampleTransactorRaw struct {
	Contract *ExampleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewExample creates a new instance of Example, bound to a specific deployed contract.
func NewExample(address common.Address, backend bind.ContractBackend) (*Example, error) {
	contract, err := bindExample(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Example{ExampleCaller: ExampleCaller{contract: contract}, ExampleTransactor: ExampleTransactor{contract: contract}, ExampleFilterer: ExampleFilterer{contract: contract}}, nil
}

// NewExampleCaller creates a new read-only instance of Example, bound to a specific deployed contract.
func NewExampleCaller(address common.Address, caller bind.ContractCaller) (*ExampleCaller, error) {
	contract, err := bindExample(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ExampleCaller{contract: contract}, nil
}

// NewExampleTransactor creates a new write-only instance of Example, bound to a specific deployed contract.
func NewExampleTransactor(address common.Address, transactor bind.ContractTransactor) (*ExampleTransactor, error) {
	contract, err := bindExample(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ExampleTransactor{contract: contract}, nil
}

// NewExampleFilterer creates a new log filterer instance of Example, bound to a specific deployed contract.
func NewExampleFilterer(address common.Address, filterer bind.ContractFilterer) (*ExampleFilterer, error) {
	contract, err := bindExample(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ExampleFilterer{contract: contract}, nil
}

// bindExample binds a generic wrapper to an already deployed contract.
func bindExample(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ExampleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Example *ExampleRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Example.Contract.ExampleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Example *ExampleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Example.Contract.ExampleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Example *ExampleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Example.Contract.ExampleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Example *ExampleCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Example.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Example *ExampleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Example.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Example *ExampleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Example.Contract.contract.Transact(opts, method, params...)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() constant returns(bytes32)
func (_Example *ExampleCaller) DOMAINSEPARATOR(opts *bind.CallOpts) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _Example.contract.Call(opts, out, "DOMAIN_SEPARATOR")
	return *ret0, err
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() constant returns(bytes32)
func (_Example *ExampleSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _Example.Contract.DOMAINSEPARATOR(&_Example.CallOpts)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() constant returns(bytes32)
func (_Example *ExampleCallerSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _Example.Contract.DOMAINSEPARATOR(&_Example.CallOpts)
}

// Test is a free data retrieval call binding the contract method 0xf8a8fd6d.
//
// Solidity: function test() constant returns(bool)
func (_Example *ExampleCaller) Test(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Example.contract.Call(opts, out, "test")
	return *ret0, err
}

// Test is a free data retrieval call binding the contract method 0xf8a8fd6d.
//
// Solidity: function test() constant returns(bool)
func (_Example *ExampleSession) Test() (bool, error) {
	return _Example.Contract.Test(&_Example.CallOpts)
}

// Test is a free data retrieval call binding the contract method 0xf8a8fd6d.
//
// Solidity: function test() constant returns(bool)
func (_Example *ExampleCallerSession) Test() (bool, error) {
	return _Example.Contract.Test(&_Example.CallOpts)
}

