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
const ExampleABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"MAIL\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"v\",\"type\":\"uint8\"},{\"name\":\"r\",\"type\":\"bytes32\"},{\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"verify\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// ExampleBin is the compiled bytecode used for deploying new contracts.
const ExampleBin = `0x60036101208181527f436f77000000000000000000000000000000000000000000000000000000000061014090815260e091825273cd2a3d9f938e13cd947ec05abc7fe734df8dd8266101005260808281526101a08481527f426f6200000000000000000000000000000000000000000000000000000000006101c05261016090815273bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb6101805260a052610220604052600b6101e09081527f48656c6c6f2c20426f62210000000000000000000000000000000000000000006102005260c05292600092918391620000e8918391620008eb565b506020918201516001919091018054600160a060020a031916600160a060020a03909216919091179055828101518051805191926002850192620001309284920190620008eb565b506020918201516001919091018054600160a060020a031916600160a060020a0390921691909117905560408301518051620001739260048501920190620008eb565b5050503480156200018357600080fd5b506040805160c081018252600a608082019081527f4574686572204d61696c0000000000000000000000000000000000000000000060a083015281528151808301835260018082527f31000000000000000000000000000000000000000000000000000000000000006020838101919091528301919091529181019190915273cccccccccccccccccccccccccccccccccccccccc6060820152620002309064010000000062000463810204565b600555604080516000805460c06020601f60026101006001861615026000190190941693909304928301819004028401810190945260a083018181526200045a9484926060840192859284928491870182828015620002d35780601f10620002a757610100808354040283529160200191620002d3565b820191906000526020600020905b815481529060010190602001808311620002b557829003601f168201915b5050509183525050600191820154600160a060020a0316602091820152918352604080516002868101805461010095811615959095026000190190941604601f810185900485028201606090810184529282018181529590940194909384928491908401828280156200038a5780601f106200035e576101008083540402835291602001916200038a565b820191906000526020600020905b8154815290600101906020018083116200036c57829003601f168201915b5050509183525050600191820154600160a060020a031660209182015291835260048401805460408051600261010095841615959095026000190190921693909304601f810185900485028201850190935282815293830193929091908301828280156200043c5780601f1062000410576101008083540402835291602001916200043c565b820191906000526020600020905b8154815290600101906020018083116200041e57829003601f168201915b50505050508152505062000650640100000000026401000000009004565b60065562000990565b600060405180807f454950373132446f6d61696e28737472696e67206e616d652c737472696e672081526020017f76657273696f6e2c75696e7432353620636861696e49642c616464726573732081526020017f766572696679696e67436f6e74726163742900000000000000000000000000008152506052019050604051809103902082600001516040518082805190602001908083835b602083106200051d5780518252601f199092019160209182019101620004fc565b51815160209384036101000a6000190180199092169116179052604051919093018190038120888401518051919650945090928392508401908083835b602083106200057b5780518252601f1990920191602091820191016200055a565b51815160209384036101000a6000190180199092169116179052604080519290940182900382208a8501516060808d01518585019b909b5284870199909952978301526080820196909652600160a060020a0390961660a0808801919091528251808803909101815260c0909601918290525084519093849350850191508083835b602083106200061e5780518252601f199092019160209182019101620005fd565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b604080517f4d61696c28506572736f6e2066726f6d2c506572736f6e20746f2c737472696e81527f6720636f6e74656e747329506572736f6e28737472696e67206e616d652c616460208201527f64726573732077616c6c6574290000000000000000000000000000000000000081830152905190819003604d019020815160009190620006e790640100000000620007c5810204565b620007058460200151620007c5640100000000026401000000009004565b84604001516040518082805190602001908083835b602083106200073b5780518252601f1990920191602091820191016200071a565b51815160001960209485036101000a019081169019919091161790526040805194909201849003842084820199909952838201979097526060830195909552506080808201969096528351808203909601865260a00192839052505082519091829190840190808383602083106200061e5780518252601f199092019160209182019101620005fd565b600060405180807f506572736f6e28737472696e67206e616d652c616464726573732077616c6c6581526020017f74290000000000000000000000000000000000000000000000000000000000008152506022019050604051809103902082600001516040518082805190602001908083835b60208310620008595780518252601f19909201916020918201910162000838565b51815160001960209485036101000a0190811690199190911617905260408051949092018490038420898201518583019890985284830152600160a060020a0390961660608085019190915281518085039091018152608090930190819052825192959094508493508501919050808383602083106200061e5780518252601f199092019160209182019101620005fd565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106200092e57805160ff19168380011785556200095e565b828001600101855582156200095e579182015b828111156200095e57825182559160200191906001019062000941565b506200096c92915062000970565b5090565b6200098d91905b808211156200096c576000815560010162000977565b90565b61072180620009a06000396000f3006080604052600436106100565763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416633644e515811461005b57806387d093da14610082578063e245452214610097575b600080fd5b34801561006757600080fd5b506100706100cc565b60408051918252519081900360200190f35b34801561008e57600080fd5b506100706100d2565b3480156100a357600080fd5b506100b860ff600435166024356044356100d8565b604080519115158252519081900360200190f35b60055481565b60065481565b600554604080516000805460c06020601f60026000196101006001871615020190941693909304928301819004028401810190945260a0830181815291948594909361030a939092869284926060840192859284929184918701828280156101815780601f1061015657610100808354040283529160200191610181565b820191906000526020600020905b81548152906001019060200180831161016457829003601f168201915b505050918352505060019182015473ffffffffffffffffffffffffffffffffffffffff16602091820152918352604080516002868101805461010095811615959095026000190190941604601f810185900485028201606090810184529282018181529590940194909384928491908401828280156102415780601f1061021657610100808354040283529160200191610241565b820191906000526020600020905b81548152906001019060200180831161022457829003601f168201915b505050918352505060019182015473ffffffffffffffffffffffffffffffffffffffff1660209182015291835260048401805460408051600261010095841615959095026000190190921693909304601f810185900485028201850190935282815293830193929091908301828280156102fc5780601f106102d1576101008083540402835291602001916102fc565b820191906000526020600020905b8154815290600101906020018083116102df57829003601f168201915b50505050508152505061043e565b604080517f19010000000000000000000000000000000000000000000000000000000000006020808301919091526022820194909452604280820193909352815180820390930183526062019081905281519192909182918401908083835b602083106103885780518252601f199092019160209182019101610369565b51815160209384036101000a600019018019909216911617905260408051929094018290038220600080845283830180875282905260ff8d1684870152606084018c9052608084018b905294519097503396506001955060a080840195929450601f198201938290030191865af1158015610407573d6000803e3d6000fd5b5050506020604051035173ffffffffffffffffffffffffffffffffffffffff1614151561043357600080fd5b506001949350505050565b604080517f4d61696c28506572736f6e2066726f6d2c506572736f6e20746f2c737472696e81527f6720636f6e74656e747329506572736f6e28737472696e67206e616d652c616460208201527f64726573732077616c6c6574290000000000000000000000000000000000000081830152905190819003604d0190208151600091906104ca906105c6565b6104d784602001516105c6565b84604001516040518082805190602001908083835b6020831061050b5780518252601f1990920191602091820191016104ec565b51815160001960209485036101000a019081169019919091161790526040805194909201849003842084820199909952838201979097526060830195909552506080808201969096528351808203909601865260a001928390525050825190918291908401908083835b602083106105945780518252601f199092019160209182019101610575565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b600060405180807f506572736f6e28737472696e67206e616d652c616464726573732077616c6c6581526020017f74290000000000000000000000000000000000000000000000000000000000008152506022019050604051809103902082600001516040518082805190602001908083835b602083106106585780518252601f199092019160209182019101610639565b51815160001960209485036101000a019081169019919091161790526040805194909201849003842089820151858301989098528483015273ffffffffffffffffffffffffffffffffffffffff90961660608085019190915281518085039091018152608090930190819052825192959094508493508501919050808383602083106105945780518252601f1990920191602091820191016105755600a165627a7a723058207bf5da9df54e2af341db2c954e10c236cac514fb0c6cf892cd4c7e0d609d7ee70029`

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

// MAIL is a free data retrieval call binding the contract method 0x87d093da.
//
// Solidity: function MAIL() constant returns(bytes32)
func (_Example *ExampleCaller) MAIL(opts *bind.CallOpts) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _Example.contract.Call(opts, out, "MAIL")
	return *ret0, err
}

// MAIL is a free data retrieval call binding the contract method 0x87d093da.
//
// Solidity: function MAIL() constant returns(bytes32)
func (_Example *ExampleSession) MAIL() ([32]byte, error) {
	return _Example.Contract.MAIL(&_Example.CallOpts)
}

// MAIL is a free data retrieval call binding the contract method 0x87d093da.
//
// Solidity: function MAIL() constant returns(bytes32)
func (_Example *ExampleCallerSession) MAIL() ([32]byte, error) {
	return _Example.Contract.MAIL(&_Example.CallOpts)
}

// Verify is a paid mutator transaction binding the contract method 0xe2454522.
//
// Solidity: function verify(v uint8, r bytes32, s bytes32) returns(bool)
func (_Example *ExampleTransactor) Verify(opts *bind.TransactOpts, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Example.contract.Transact(opts, "verify", v, r, s)
}

// Verify is a paid mutator transaction binding the contract method 0xe2454522.
//
// Solidity: function verify(v uint8, r bytes32, s bytes32) returns(bool)
func (_Example *ExampleSession) Verify(v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Example.Contract.Verify(&_Example.TransactOpts, v, r, s)
}

// Verify is a paid mutator transaction binding the contract method 0xe2454522.
//
// Solidity: function verify(v uint8, r bytes32, s bytes32) returns(bool)
func (_Example *ExampleTransactorSession) Verify(v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Example.Contract.Verify(&_Example.TransactOpts, v, r, s)
}

