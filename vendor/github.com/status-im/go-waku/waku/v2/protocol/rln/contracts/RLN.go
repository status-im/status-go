// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

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

// IPoseidonHasherABI is the input ABI used to generate the binding from.
const IPoseidonHasherABI = "[{\"inputs\":[{\"internalType\":\"uint256[2]\",\"name\":\"input\",\"type\":\"uint256[2]\"}],\"name\":\"hash\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"result\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"identity\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]"

// IPoseidonHasherFuncSigs maps the 4-byte function signature to its string representation.
var IPoseidonHasherFuncSigs = map[string]string{
	"561558fe": "hash(uint256[2])",
	"2c159a1a": "identity()",
}

// IPoseidonHasher is an auto generated Go binding around an Ethereum contract.
type IPoseidonHasher struct {
	IPoseidonHasherCaller     // Read-only binding to the contract
	IPoseidonHasherTransactor // Write-only binding to the contract
	IPoseidonHasherFilterer   // Log filterer for contract events
}

// IPoseidonHasherCaller is an auto generated read-only Go binding around an Ethereum contract.
type IPoseidonHasherCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IPoseidonHasherTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IPoseidonHasherTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IPoseidonHasherFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IPoseidonHasherFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IPoseidonHasherSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IPoseidonHasherSession struct {
	Contract     *IPoseidonHasher  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IPoseidonHasherCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IPoseidonHasherCallerSession struct {
	Contract *IPoseidonHasherCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// IPoseidonHasherTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IPoseidonHasherTransactorSession struct {
	Contract     *IPoseidonHasherTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// IPoseidonHasherRaw is an auto generated low-level Go binding around an Ethereum contract.
type IPoseidonHasherRaw struct {
	Contract *IPoseidonHasher // Generic contract binding to access the raw methods on
}

// IPoseidonHasherCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IPoseidonHasherCallerRaw struct {
	Contract *IPoseidonHasherCaller // Generic read-only contract binding to access the raw methods on
}

// IPoseidonHasherTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IPoseidonHasherTransactorRaw struct {
	Contract *IPoseidonHasherTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIPoseidonHasher creates a new instance of IPoseidonHasher, bound to a specific deployed contract.
func NewIPoseidonHasher(address common.Address, backend bind.ContractBackend) (*IPoseidonHasher, error) {
	contract, err := bindIPoseidonHasher(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IPoseidonHasher{IPoseidonHasherCaller: IPoseidonHasherCaller{contract: contract}, IPoseidonHasherTransactor: IPoseidonHasherTransactor{contract: contract}, IPoseidonHasherFilterer: IPoseidonHasherFilterer{contract: contract}}, nil
}

// NewIPoseidonHasherCaller creates a new read-only instance of IPoseidonHasher, bound to a specific deployed contract.
func NewIPoseidonHasherCaller(address common.Address, caller bind.ContractCaller) (*IPoseidonHasherCaller, error) {
	contract, err := bindIPoseidonHasher(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IPoseidonHasherCaller{contract: contract}, nil
}

// NewIPoseidonHasherTransactor creates a new write-only instance of IPoseidonHasher, bound to a specific deployed contract.
func NewIPoseidonHasherTransactor(address common.Address, transactor bind.ContractTransactor) (*IPoseidonHasherTransactor, error) {
	contract, err := bindIPoseidonHasher(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IPoseidonHasherTransactor{contract: contract}, nil
}

// NewIPoseidonHasherFilterer creates a new log filterer instance of IPoseidonHasher, bound to a specific deployed contract.
func NewIPoseidonHasherFilterer(address common.Address, filterer bind.ContractFilterer) (*IPoseidonHasherFilterer, error) {
	contract, err := bindIPoseidonHasher(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IPoseidonHasherFilterer{contract: contract}, nil
}

// bindIPoseidonHasher binds a generic wrapper to an already deployed contract.
func bindIPoseidonHasher(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IPoseidonHasherABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IPoseidonHasher *IPoseidonHasherRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IPoseidonHasher.Contract.IPoseidonHasherCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IPoseidonHasher *IPoseidonHasherRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IPoseidonHasher.Contract.IPoseidonHasherTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IPoseidonHasher *IPoseidonHasherRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IPoseidonHasher.Contract.IPoseidonHasherTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IPoseidonHasher *IPoseidonHasherCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IPoseidonHasher.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IPoseidonHasher *IPoseidonHasherTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IPoseidonHasher.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IPoseidonHasher *IPoseidonHasherTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IPoseidonHasher.Contract.contract.Transact(opts, method, params...)
}

// Hash is a free data retrieval call binding the contract method 0x561558fe.
//
// Solidity: function hash(uint256[2] input) pure returns(uint256 result)
func (_IPoseidonHasher *IPoseidonHasherCaller) Hash(opts *bind.CallOpts, input [2]*big.Int) (*big.Int, error) {
	var out []interface{}
	err := _IPoseidonHasher.contract.Call(opts, &out, "hash", input)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Hash is a free data retrieval call binding the contract method 0x561558fe.
//
// Solidity: function hash(uint256[2] input) pure returns(uint256 result)
func (_IPoseidonHasher *IPoseidonHasherSession) Hash(input [2]*big.Int) (*big.Int, error) {
	return _IPoseidonHasher.Contract.Hash(&_IPoseidonHasher.CallOpts, input)
}

// Hash is a free data retrieval call binding the contract method 0x561558fe.
//
// Solidity: function hash(uint256[2] input) pure returns(uint256 result)
func (_IPoseidonHasher *IPoseidonHasherCallerSession) Hash(input [2]*big.Int) (*big.Int, error) {
	return _IPoseidonHasher.Contract.Hash(&_IPoseidonHasher.CallOpts, input)
}

// Identity is a free data retrieval call binding the contract method 0x2c159a1a.
//
// Solidity: function identity() pure returns(uint256)
func (_IPoseidonHasher *IPoseidonHasherCaller) Identity(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IPoseidonHasher.contract.Call(opts, &out, "identity")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Identity is a free data retrieval call binding the contract method 0x2c159a1a.
//
// Solidity: function identity() pure returns(uint256)
func (_IPoseidonHasher *IPoseidonHasherSession) Identity() (*big.Int, error) {
	return _IPoseidonHasher.Contract.Identity(&_IPoseidonHasher.CallOpts)
}

// Identity is a free data retrieval call binding the contract method 0x2c159a1a.
//
// Solidity: function identity() pure returns(uint256)
func (_IPoseidonHasher *IPoseidonHasherCallerSession) Identity() (*big.Int, error) {
	return _IPoseidonHasher.Contract.Identity(&_IPoseidonHasher.CallOpts)
}

// PoseidonHasherABI is the input ABI used to generate the binding from.
const PoseidonHasherABI = "[{\"inputs\":[{\"internalType\":\"uint256[2]\",\"name\":\"input\",\"type\":\"uint256[2]\"}],\"name\":\"hash\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"result\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"identity\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]"

// PoseidonHasherFuncSigs maps the 4-byte function signature to its string representation.
var PoseidonHasherFuncSigs = map[string]string{
	"561558fe": "hash(uint256[2])",
	"2c159a1a": "identity()",
}

// PoseidonHasherBin is the compiled bytecode used for deploying new contracts.
var PoseidonHasherBin = "0x608060405234801561001057600080fd5b50612142806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80632c159a1a1461003b578063561558fe14610055575b600080fd5b6100436100a0565b60408051918252519081900360200190f35b6100436004803603604081101561006b57600080fd5b604080518082018252918301929181830191839060029083908390808284376000920191909152509194506100af9350505050565b60006100aa6100c0565b905090565b60006100ba826100e4565b92915050565b7f2ff267fd23782a5625e6d804f0a7fa700b8dc6084e2e7a5aff7cd4b1c506d30b90565b60007f30644e72e131a029b85045b68181585d2833e84879b9709143e1f593f00000016040517f165d45ae851912f9a33800b04cc6617b184bf67db11ce904dc82601244551ae281527f10fc284d0af588165f4fc650fe7c53b1d80fbaac16d30518bf142117f42f820460208201527f06b687bd3c688aa9a03545d0835bca75ae82c434bf7d5fb065a2818b5c74814f60408201527f01057eb8e4bba26f12f4ea819251708d72e0605e6de827e990c3ba4ae13f5ecd60608201527e23779a38eb9ef4a9beaf4dc0a2ab5233a28ce6d10ad2512230a275b83017c360808201527f012e5dfdd4f34081753b70c897773f5d2987c8bbae32ad202a27cd61d7fba2fb7f0d1807f022a8d80d9304a1522087b8692dc0acf7b43fea28782d2ae672c0b11f7f17d468d0e6541de501481931453ed1e4a250e5e301f27dc91fe3b4bd522aa53c7f1ea09a4bd33f14eafd75e775d85e568fa668938fdd2f16fad1d4d2d2b9862b007f061f2e832c23bee84c2f09d63d371cc5eda2f749cdbe9a6a6d20469e9fa36e8b8851017f061f2e832c23bee84c2f09d63d371cc5eda2f749cdbe9a6a6d20469e9fa36e8b60208a0151017f061f2e832c23bee84c2f09d63d371cc5eda2f749cdbe9a6a6d20469e9fa36e8b8883840989848b83840909935089838409905089838b83840909925089828309905089828b8384090991507f0e4d154ca9b7f5111958289f43ed5bbc4d4f6118d45d9aefeb778179d921a59b9050808a60408b015184098b60208c015186098c8c518809010101818b8a85098c60808d015187098d60608e01518909010101828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991508b81820992508b818d8586090990508b84850992508b848d8586090993507f298d683000ab71c72fe4371cf6cd37bb584b6d816a653ee4bfea62518a337e079250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995508b85860992508b858d8586090994508b84850992508b848d8586090993507f2f860295bc93d694e74905913ddcae47290b9b5abb43a537fe40d9305bd1e1679250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991508b81820992508b818d8586090990508b84850992508b848d8586090993507f1dd8b95942d95c7896be7f3f455e595cbfee5e1023c5d45e6573c8513f1f3dfa9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f189aa3023aeaa7267dd3c74e2c8b9cba363546619bb9c54bcb02b22fe64c51619250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f0da6b697fd05fe54a523131d91e2a7f6ac18184e237c1b20ab1936616f08e5239250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f019df963bfafa7f0e34cf092f33ce93b38b9d130fb44f276196e59f16e63ac5a9250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f295332d5ab168bc3c5eb671528c9896b7bd5034fb02472ff2af4fa1087ae658f9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f0f423b84792458876c314228be9361a604c75010411b87add91b791df2f980fa9250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f1171744890a155b5ad9cde4f1f5c4eab878f3730a9fc71f546eb737c84abec5f9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f2a557a13928c7eacff5dc049bc603cff1d7959c3975c5c9fc035b0c67173abac9250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507ec9c858d5d1ab1622d7ea2174cd9a35b07c2655dd08dcda3f5a9f7222c021af9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f0b358e01fc7cf925b6d3ba6e80c345881b54ff289479cc532f9fec01b500b9789250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f26c26f6e5a92ff96ec05a61d5f36df4f8134f8d61ff0eaa28cf7a480ee2b7f849250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f020930074e8c6c15295104a58533a601f202d3b3f959650d7b8cfa754ae41e7f9250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f10f5f06286fdde345a5cd8ba4ad94828271ea16309f5fa906841e72eb4c330279250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f116ce0fb46b45c99fec7d369ab5c0d11d374a16b2107c5d6f21e3792d33fc9699250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f2a59da7487bf7a05c3dbdfca6084e2ec589b1baf87ddd790d9bbc9c23557255b9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f1559c4b2c04c419e948f42a7f8fecfb1525dbdf2b6830632723ff4fa77030ea89250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f030b1745767217e6d8df6885a85651f9aebefcf468003d2dd25906897c3ca92a9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f2668ff336585f1e64cd6ddf2cbcaa8a695585f0db79cbd1727becfeabe76d0449250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f158c1ceb605d6cb3e0c3a7e6650b1f297c4fccb1aeed560a09dfd474c4e47f999250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f27506f4872c8b8bb7f42921afbefd206cef54f5001f1c32efbc2a22094d172d49250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f0e915e793df4e8ee01fd861b22ee463c985c691fb2a54c8657575ed93ab92c739250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f264bd2bc0486212803e5ef729bcaf466eb14f5b6ac18b898ed94844dda28d8a99250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f2f23c5d822cd0d5b27fee9ff2c51f30a40654cdd08677543244f6aad8f2ebc8a9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f2776a03893ed5c5ce680ca5acda3c7d4b60b81269cabca38bfe060ce6eb6c4129250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f07aba324bd6d746b9f23313249f6a1809f3cfea3b1f3eb70c37951eb77120d3a9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f09c0a6b8cf7f1c457987d7bb476b8ed8ddf88dbb146f2d887821d93d453d0fa39250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f292e4542f0e5b148bc7c9188b9db100f0f2aa2a62edb236041c4e4ebf44eb1469250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f01b08e9289e3a73080f49265283832fc48dfb2da2d540503b8274d7c0e52c9b69250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f0d897c5adfe393f4c0247e718eb0b227f87226cc416b703eb2a96d61581af2e39250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f057582e2ebdd031c39c69ed08086d969d22de5d1bf79ee1be837234977e3477c9250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f0be6726fc2d3d2b9681fdc9ce4e274395f9e6dc5c6e647a3a5343e3f3fe77edf9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f0e919ea6e332df81ae88f1155eac9e3c4b7dd7d8637feb112bcb2176f45fd9429250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f2dadfa8057c413974c1c03aa2d38aac0d8c09748d08edc0e97450320c889c40d9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f2954c27dbde3ba4f887becde3fc6c47db5266d436914d50d4e93faa79e40f7789250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f1e8f4f1b4fa7cc676e0d02084ce2bbf095adf6bcee924520e57c2f2a4b43f98a9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f01892c0a3e4c3ffce3763f5d67ea86157c3b2461e08ff82ee28342ad41e636259250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f297b80afd662c439ff4837049ea3d95f565007b2defd87d981bfa45ce9ce80bf9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f28def6e44cc1e6b1826dd02d19d3383ddfd0dcdf96d5e9ce0b2834c6ad5c17b89250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f0e7efce2651888246927ee306c83f4b24982ae2a126ff3be0217ea155e850d3e9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f1b61bcbf0030070f3ca673d814ad8ea269bf0b4ddf75c4b4dd8383835be6b3809250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f2bfc5e31157cdf08d36dcba28d869c209688dcb9da74df689427efbfe7ef20409250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f2535e86274b29f682edc2f95d8fc9aa4b660b1992da99d3e3ccfd551b25280ae9250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f2e26f31955ddb83a0866535943960e6bf1d75532494e07382a1886df2eba4f2c9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f1367f3092e878d4204f8597f09a6e75e596f8f859f7c4f0e4ee3bdefb14185759250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f0786e0a89d59f9f1ec5a24b4e77471afb551993358839a2ad7f0cd30ee34a5939250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f05a623c15705f67c426798eb1aeb61d16bd599b73c618801e79ce87e4acf44439250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f216a2cd9a5dec0ce0f298c37a605d63a7a241fd04f74ee76d955ec729f75c7749250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f1ac59c277950c8cf903e1e0b04c13b6be622d88a899267ccd30957e8b24feb899250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f21181ae96428a48600806eb2d2b94b4972bc69f4b69d99fb1edcbce69d1a73029250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f2a24403e8cf5fb93ed9408b11d96e377bf1f1a2e183b0e14a18d6b794fc482909250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f13324a9a9db19a8f877c22de405a88bd39bec4696dc84721e7ccfef724a8e4f59250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f17f556db73809befc58822ad53ad5785aee4ff6891f211ae7183284b57a3910a9250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f22e4f8f9c9ae56f5254ca73dc68572e805b9406a45fb4e10a831a7619b129fe59250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f066e36c90fda2dd52aa0c05909831d1dd345f75fcb17934ecb754d74e7ce8df99250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991507f190c65fb8440d50383891cbe244a8b063f0f9afbf14adce401416de7d6c755e19250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995507f0b4fec4e028a67f9a9b932ae35a1269eeec8b98b96793d29811a766ad089ab1b9250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991508b81820992508b818d8586090990508b84850992508b848d8586090993507f1def7475769de3cd372283c7d36bd153637723ff14615542bfe8b27108ad4de79250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c860901010193508b86870992508b868d8586090995508b85860992508b858d8586090994508b84850992508b848d8586090993507f119217aded5f1ebf525a69e23a08f6a976178e5939646f6c87033299d2d3c8749250828c60408d015186098d60208e015188098e8e518a090101019150828c8b86098d60808e015188098e60608f01518a090101019050828c8886098d8a88098e8c8a0901010193508b82830992508b828d8586090991508b81820992508b818d8586090990508b84850992508b848d8586090993507f036bd8311510a8d03c5b354f03c664f9f2d23f57b1e4286d1938f8dbe4a7c28c9250828c60408d015186098d60208e015184098e8e5186090101019550828c8b86098d60808e015184098e60608f015186090101019450828c8886098d8a84098e8c86090101019350505089848509905089848b83840909935089838409905089838b83840909925089828309905089828b838409505050929b9a505050505050505050505056fea264697066735822122022fc21868d907bed4fa3abb78ab3ed4f2a0fb08b5292320b05ffb1408625623f64736f6c63430007040033"

// DeployPoseidonHasher deploys a new Ethereum contract, binding an instance of PoseidonHasher to it.
func DeployPoseidonHasher(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PoseidonHasher, error) {
	parsed, err := abi.JSON(strings.NewReader(PoseidonHasherABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(PoseidonHasherBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PoseidonHasher{PoseidonHasherCaller: PoseidonHasherCaller{contract: contract}, PoseidonHasherTransactor: PoseidonHasherTransactor{contract: contract}, PoseidonHasherFilterer: PoseidonHasherFilterer{contract: contract}}, nil
}

// PoseidonHasher is an auto generated Go binding around an Ethereum contract.
type PoseidonHasher struct {
	PoseidonHasherCaller     // Read-only binding to the contract
	PoseidonHasherTransactor // Write-only binding to the contract
	PoseidonHasherFilterer   // Log filterer for contract events
}

// PoseidonHasherCaller is an auto generated read-only Go binding around an Ethereum contract.
type PoseidonHasherCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoseidonHasherTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PoseidonHasherTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoseidonHasherFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PoseidonHasherFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoseidonHasherSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PoseidonHasherSession struct {
	Contract     *PoseidonHasher   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PoseidonHasherCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PoseidonHasherCallerSession struct {
	Contract *PoseidonHasherCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// PoseidonHasherTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PoseidonHasherTransactorSession struct {
	Contract     *PoseidonHasherTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// PoseidonHasherRaw is an auto generated low-level Go binding around an Ethereum contract.
type PoseidonHasherRaw struct {
	Contract *PoseidonHasher // Generic contract binding to access the raw methods on
}

// PoseidonHasherCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PoseidonHasherCallerRaw struct {
	Contract *PoseidonHasherCaller // Generic read-only contract binding to access the raw methods on
}

// PoseidonHasherTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PoseidonHasherTransactorRaw struct {
	Contract *PoseidonHasherTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPoseidonHasher creates a new instance of PoseidonHasher, bound to a specific deployed contract.
func NewPoseidonHasher(address common.Address, backend bind.ContractBackend) (*PoseidonHasher, error) {
	contract, err := bindPoseidonHasher(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PoseidonHasher{PoseidonHasherCaller: PoseidonHasherCaller{contract: contract}, PoseidonHasherTransactor: PoseidonHasherTransactor{contract: contract}, PoseidonHasherFilterer: PoseidonHasherFilterer{contract: contract}}, nil
}

// NewPoseidonHasherCaller creates a new read-only instance of PoseidonHasher, bound to a specific deployed contract.
func NewPoseidonHasherCaller(address common.Address, caller bind.ContractCaller) (*PoseidonHasherCaller, error) {
	contract, err := bindPoseidonHasher(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PoseidonHasherCaller{contract: contract}, nil
}

// NewPoseidonHasherTransactor creates a new write-only instance of PoseidonHasher, bound to a specific deployed contract.
func NewPoseidonHasherTransactor(address common.Address, transactor bind.ContractTransactor) (*PoseidonHasherTransactor, error) {
	contract, err := bindPoseidonHasher(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PoseidonHasherTransactor{contract: contract}, nil
}

// NewPoseidonHasherFilterer creates a new log filterer instance of PoseidonHasher, bound to a specific deployed contract.
func NewPoseidonHasherFilterer(address common.Address, filterer bind.ContractFilterer) (*PoseidonHasherFilterer, error) {
	contract, err := bindPoseidonHasher(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PoseidonHasherFilterer{contract: contract}, nil
}

// bindPoseidonHasher binds a generic wrapper to an already deployed contract.
func bindPoseidonHasher(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PoseidonHasherABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PoseidonHasher *PoseidonHasherRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PoseidonHasher.Contract.PoseidonHasherCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PoseidonHasher *PoseidonHasherRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoseidonHasher.Contract.PoseidonHasherTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PoseidonHasher *PoseidonHasherRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PoseidonHasher.Contract.PoseidonHasherTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PoseidonHasher *PoseidonHasherCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PoseidonHasher.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PoseidonHasher *PoseidonHasherTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoseidonHasher.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PoseidonHasher *PoseidonHasherTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PoseidonHasher.Contract.contract.Transact(opts, method, params...)
}

// Hash is a free data retrieval call binding the contract method 0x561558fe.
//
// Solidity: function hash(uint256[2] input) pure returns(uint256 result)
func (_PoseidonHasher *PoseidonHasherCaller) Hash(opts *bind.CallOpts, input [2]*big.Int) (*big.Int, error) {
	var out []interface{}
	err := _PoseidonHasher.contract.Call(opts, &out, "hash", input)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Hash is a free data retrieval call binding the contract method 0x561558fe.
//
// Solidity: function hash(uint256[2] input) pure returns(uint256 result)
func (_PoseidonHasher *PoseidonHasherSession) Hash(input [2]*big.Int) (*big.Int, error) {
	return _PoseidonHasher.Contract.Hash(&_PoseidonHasher.CallOpts, input)
}

// Hash is a free data retrieval call binding the contract method 0x561558fe.
//
// Solidity: function hash(uint256[2] input) pure returns(uint256 result)
func (_PoseidonHasher *PoseidonHasherCallerSession) Hash(input [2]*big.Int) (*big.Int, error) {
	return _PoseidonHasher.Contract.Hash(&_PoseidonHasher.CallOpts, input)
}

// Identity is a free data retrieval call binding the contract method 0x2c159a1a.
//
// Solidity: function identity() pure returns(uint256)
func (_PoseidonHasher *PoseidonHasherCaller) Identity(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PoseidonHasher.contract.Call(opts, &out, "identity")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Identity is a free data retrieval call binding the contract method 0x2c159a1a.
//
// Solidity: function identity() pure returns(uint256)
func (_PoseidonHasher *PoseidonHasherSession) Identity() (*big.Int, error) {
	return _PoseidonHasher.Contract.Identity(&_PoseidonHasher.CallOpts)
}

// Identity is a free data retrieval call binding the contract method 0x2c159a1a.
//
// Solidity: function identity() pure returns(uint256)
func (_PoseidonHasher *PoseidonHasherCallerSession) Identity() (*big.Int, error) {
	return _PoseidonHasher.Contract.Identity(&_PoseidonHasher.CallOpts)
}

// RLNABI is the input ABI used to generate the binding from.
const RLNABI = "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"membershipDeposit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"depth\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_poseidonHasher\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"pubkey\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"MemberRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"pubkey\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"MemberWithdrawn\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEPTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MEMBERSHIP_DEPOSIT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SET_SIZE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"members\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poseidonHasher\",\"outputs\":[{\"internalType\":\"contractIPoseidonHasher\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pubkeyIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"pubkey\",\"type\":\"uint256\"}],\"name\":\"register\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"pubkeys\",\"type\":\"uint256[]\"}],\"name\":\"registerBatch\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"secret\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_pubkeyIndex\",\"type\":\"uint256\"},{\"internalType\":\"addresspayable\",\"name\":\"receiver\",\"type\":\"address\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"secrets\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"pubkeyIndexes\",\"type\":\"uint256[]\"},{\"internalType\":\"addresspayable[]\",\"name\":\"receivers\",\"type\":\"address[]\"}],\"name\":\"withdrawBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// RLNFuncSigs maps the 4-byte function signature to its string representation.
var RLNFuncSigs = map[string]string{
	"98366e35": "DEPTH()",
	"f220b9ec": "MEMBERSHIP_DEPOSIT()",
	"d0383d68": "SET_SIZE()",
	"5daf08ca": "members(uint256)",
	"331b6ab3": "poseidonHasher()",
	"61579a93": "pubkeyIndex()",
	"f207564e": "register(uint256)",
	"69e4863f": "registerBatch(uint256[])",
	"0ad58d2f": "withdraw(uint256,uint256,address)",
	"a9d85eba": "withdrawBatch(uint256[],uint256[],address[])",
}

// RLNBin is the compiled bytecode used for deploying new contracts.
var RLNBin = "0x60e06040526000805534801561001457600080fd5b50604051610c4f380380610c4f8339818101604052606081101561003757600080fd5b5080516020820151604090920151608082905260a08390526001831b60c0819052600280546001600160a01b0319166001600160a01b0390931692909217909155909190610b936100bc6000398061037a52806105c352806105e752806106eb52508061047f5250806103f2528061065d52806106c7528061087b5250610b936000f3fe6080604052600436106100915760003560e01c806398366e351161005957806398366e35146101c7578063a9d85eba146101dc578063d0383d68146102f7578063f207564e1461030c578063f220b9ec1461032957610091565b80630ad58d2f14610096578063331b6ab3146100d75780635daf08ca1461010857806361579a931461014457806369e4863f14610159575b600080fd5b3480156100a257600080fd5b506100d5600480360360608110156100b957600080fd5b50803590602081013590604001356001600160a01b031661033e565b005b3480156100e357600080fd5b506100ec61034e565b604080516001600160a01b039092168252519081900360200190f35b34801561011457600080fd5b506101326004803603602081101561012b57600080fd5b503561035d565b60408051918252519081900360200190f35b34801561015057600080fd5b5061013261036f565b6100d56004803603602081101561016f57600080fd5b810190602081018135600160201b81111561018957600080fd5b82018360208201111561019b57600080fd5b803590602001918460208302840111600160201b831117156101bc57600080fd5b509092509050610375565b3480156101d357600080fd5b5061013261047d565b3480156101e857600080fd5b506100d5600480360360608110156101ff57600080fd5b810190602081018135600160201b81111561021957600080fd5b82018360208201111561022b57600080fd5b803590602001918460208302840111600160201b8311171561024c57600080fd5b919390929091602081019035600160201b81111561026957600080fd5b82018360208201111561027b57600080fd5b803590602001918460208302840111600160201b8311171561029c57600080fd5b919390929091602081019035600160201b8111156102b957600080fd5b8201836020820111156102cb57600080fd5b803590602001918460208302840111600160201b831117156102ec57600080fd5b5090925090506104a1565b34801561030357600080fd5b506101326105c1565b6100d56004803603602081101561032257600080fd5b50356105e5565b34801561033557600080fd5b506101326106c5565b6103498383836106e9565b505050565b6002546001600160a01b031681565b60016020526000908152604090205481565b60005481565b6000547f000000000000000000000000000000000000000000000000000000000000000090820111156103ef576040805162461bcd60e51b815260206004820152601f60248201527f524c4e2c20726567697374657242617463683a207365742069732066756c6c00604482015290519081900360640190fd5b347f000000000000000000000000000000000000000000000000000000000000000082021461044f5760405162461bcd60e51b8152600401808060200182810382526037815260200180610a566037913960400191505060405180910390fd5b60005b818110156103495761047583838381811061046957fe5b90506020020135610902565b600101610452565b7f000000000000000000000000000000000000000000000000000000000000000081565b84806104de5760405162461bcd60e51b8152600401808060200182810382526023815260200180610a336023913960400191505060405180910390fd5b80841461051c5760405162461bcd60e51b81526004018080602001828103825260368152602001806109fd6036913960400191505060405180910390fd5b80821461055a5760405162461bcd60e51b8152600401808060200182810382526031815260200180610a8d6031913960400191505060405180910390fd5b60005b818110156105b7576105af88888381811061057457fe5b9050602002013587878481811061058757fe5b9050602002013586868581811061059a57fe5b905060200201356001600160a01b03166106e9565b60010161055d565b5050505050505050565b7f000000000000000000000000000000000000000000000000000000000000000081565b7f00000000000000000000000000000000000000000000000000000000000000006000541061065b576040805162461bcd60e51b815260206004820152601a60248201527f524c4e2c2072656769737465723a207365742069732066756c6c000000000000604482015290519081900360640190fd5b7f000000000000000000000000000000000000000000000000000000000000000034146106b95760405162461bcd60e51b8152600401808060200182810382526032815260200180610abe6032913960400191505060405180910390fd5b6106c281610902565b50565b7f000000000000000000000000000000000000000000000000000000000000000081565b7f000000000000000000000000000000000000000000000000000000000000000082106107475760405162461bcd60e51b8152600401808060200182810382526024815260200180610af06024913960400191505060405180910390fd5b6000828152600160205260409020546107915760405162461bcd60e51b8152600401808060200182810382526024815260200180610b146024913960400191505060405180910390fd5b6001600160a01b0381166107d65760405162461bcd60e51b8152600401808060200182810382526026815260200180610b386026913960400191505060405180910390fd5b60006107f66040518060400160405280868152602001600081525061095b565b600084815260016020526040902054909150811461085b576040805162461bcd60e51b815260206004820152601c60248201527f524c4e2c205f77697468647261773a206e6f7420766572696669656400000000604482015290519081900360640190fd5b600083815260016020526040808220829055516001600160a01b038416917f000000000000000000000000000000000000000000000000000000000000000080156108fc02929091818181858888f193505050501580156108c0573d6000803e3d6000fd5b50604080518281526020810185905281517f62ec3a516d22a993ce5cb4e7593e878c74f4d799dde522a88dc27a994fd5a943929181900390910190a150505050565b600080548152600160209081526040808320849055915482518481529182015281517f5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2929181900390910190a150600080546001019055565b60025460408051632b0aac7f60e11b81526000926001600160a01b03169163561558fe918591600490910190819083908083838a5b838110156109a8578181015183820152602001610990565b5050505090500191505060206040518083038186803b1580156109ca57600080fd5b505afa1580156109de573d6000803e3d6000fd5b505050506040513d60208110156109f457600080fd5b50519291505056fe524c4e2c20776974686472617742617463683a2062617463682073697a65206d69736d61746368207075626b657920696e6465786573524c4e2c20776974686472617742617463683a2062617463682073697a65207a65726f524c4e2c20726567697374657242617463683a206d656d62657273686970206465706f736974206973206e6f7420736174697366696564524c4e2c20776974686472617742617463683a2062617463682073697a65206d69736d6174636820726563656976657273524c4e2c2072656769737465723a206d656d62657273686970206465706f736974206973206e6f7420736174697366696564524c4e2c205f77697468647261773a20696e76616c6964207075626b657920696e646578524c4e2c205f77697468647261773a206d656d62657220646f65736e2774206578697374524c4e2c205f77697468647261773a20656d7074792072656365697665722061646472657373a264697066735822122089b1a09f209b826ff6506cf1ad2811656094f532866b769f0bb8e9a160a21dc464736f6c63430007040033"

// DeployRLN deploys a new Ethereum contract, binding an instance of RLN to it.
func DeployRLN(auth *bind.TransactOpts, backend bind.ContractBackend, membershipDeposit *big.Int, depth *big.Int, _poseidonHasher common.Address) (common.Address, *types.Transaction, *RLN, error) {
	parsed, err := abi.JSON(strings.NewReader(RLNABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(RLNBin), backend, membershipDeposit, depth, _poseidonHasher)
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
	parsed, err := abi.JSON(strings.NewReader(RLNABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(uint256)
func (_RLN *RLNCaller) Members(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "members", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(uint256)
func (_RLN *RLNSession) Members(arg0 *big.Int) (*big.Int, error) {
	return _RLN.Contract.Members(&_RLN.CallOpts, arg0)
}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(uint256)
func (_RLN *RLNCallerSession) Members(arg0 *big.Int) (*big.Int, error) {
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

// PubkeyIndex is a free data retrieval call binding the contract method 0x61579a93.
//
// Solidity: function pubkeyIndex() view returns(uint256)
func (_RLN *RLNCaller) PubkeyIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "pubkeyIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PubkeyIndex is a free data retrieval call binding the contract method 0x61579a93.
//
// Solidity: function pubkeyIndex() view returns(uint256)
func (_RLN *RLNSession) PubkeyIndex() (*big.Int, error) {
	return _RLN.Contract.PubkeyIndex(&_RLN.CallOpts)
}

// PubkeyIndex is a free data retrieval call binding the contract method 0x61579a93.
//
// Solidity: function pubkeyIndex() view returns(uint256)
func (_RLN *RLNCallerSession) PubkeyIndex() (*big.Int, error) {
	return _RLN.Contract.PubkeyIndex(&_RLN.CallOpts)
}

// Register is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 pubkey) payable returns()
func (_RLN *RLNTransactor) Register(opts *bind.TransactOpts, pubkey *big.Int) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "register", pubkey)
}

// Register is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 pubkey) payable returns()
func (_RLN *RLNSession) Register(pubkey *big.Int) (*types.Transaction, error) {
	return _RLN.Contract.Register(&_RLN.TransactOpts, pubkey)
}

// Register is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 pubkey) payable returns()
func (_RLN *RLNTransactorSession) Register(pubkey *big.Int) (*types.Transaction, error) {
	return _RLN.Contract.Register(&_RLN.TransactOpts, pubkey)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x69e4863f.
//
// Solidity: function registerBatch(uint256[] pubkeys) payable returns()
func (_RLN *RLNTransactor) RegisterBatch(opts *bind.TransactOpts, pubkeys []*big.Int) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "registerBatch", pubkeys)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x69e4863f.
//
// Solidity: function registerBatch(uint256[] pubkeys) payable returns()
func (_RLN *RLNSession) RegisterBatch(pubkeys []*big.Int) (*types.Transaction, error) {
	return _RLN.Contract.RegisterBatch(&_RLN.TransactOpts, pubkeys)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x69e4863f.
//
// Solidity: function registerBatch(uint256[] pubkeys) payable returns()
func (_RLN *RLNTransactorSession) RegisterBatch(pubkeys []*big.Int) (*types.Transaction, error) {
	return _RLN.Contract.RegisterBatch(&_RLN.TransactOpts, pubkeys)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0ad58d2f.
//
// Solidity: function withdraw(uint256 secret, uint256 _pubkeyIndex, address receiver) returns()
func (_RLN *RLNTransactor) Withdraw(opts *bind.TransactOpts, secret *big.Int, _pubkeyIndex *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "withdraw", secret, _pubkeyIndex, receiver)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0ad58d2f.
//
// Solidity: function withdraw(uint256 secret, uint256 _pubkeyIndex, address receiver) returns()
func (_RLN *RLNSession) Withdraw(secret *big.Int, _pubkeyIndex *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _RLN.Contract.Withdraw(&_RLN.TransactOpts, secret, _pubkeyIndex, receiver)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0ad58d2f.
//
// Solidity: function withdraw(uint256 secret, uint256 _pubkeyIndex, address receiver) returns()
func (_RLN *RLNTransactorSession) Withdraw(secret *big.Int, _pubkeyIndex *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _RLN.Contract.Withdraw(&_RLN.TransactOpts, secret, _pubkeyIndex, receiver)
}

// WithdrawBatch is a paid mutator transaction binding the contract method 0xa9d85eba.
//
// Solidity: function withdrawBatch(uint256[] secrets, uint256[] pubkeyIndexes, address[] receivers) returns()
func (_RLN *RLNTransactor) WithdrawBatch(opts *bind.TransactOpts, secrets []*big.Int, pubkeyIndexes []*big.Int, receivers []common.Address) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "withdrawBatch", secrets, pubkeyIndexes, receivers)
}

// WithdrawBatch is a paid mutator transaction binding the contract method 0xa9d85eba.
//
// Solidity: function withdrawBatch(uint256[] secrets, uint256[] pubkeyIndexes, address[] receivers) returns()
func (_RLN *RLNSession) WithdrawBatch(secrets []*big.Int, pubkeyIndexes []*big.Int, receivers []common.Address) (*types.Transaction, error) {
	return _RLN.Contract.WithdrawBatch(&_RLN.TransactOpts, secrets, pubkeyIndexes, receivers)
}

// WithdrawBatch is a paid mutator transaction binding the contract method 0xa9d85eba.
//
// Solidity: function withdrawBatch(uint256[] secrets, uint256[] pubkeyIndexes, address[] receivers) returns()
func (_RLN *RLNTransactorSession) WithdrawBatch(secrets []*big.Int, pubkeyIndexes []*big.Int, receivers []common.Address) (*types.Transaction, error) {
	return _RLN.Contract.WithdrawBatch(&_RLN.TransactOpts, secrets, pubkeyIndexes, receivers)
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
	Pubkey *big.Int
	Index  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterMemberRegistered is a free log retrieval operation binding the contract event 0x5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2.
//
// Solidity: event MemberRegistered(uint256 pubkey, uint256 index)
func (_RLN *RLNFilterer) FilterMemberRegistered(opts *bind.FilterOpts) (*RLNMemberRegisteredIterator, error) {

	logs, sub, err := _RLN.contract.FilterLogs(opts, "MemberRegistered")
	if err != nil {
		return nil, err
	}
	return &RLNMemberRegisteredIterator{contract: _RLN.contract, event: "MemberRegistered", logs: logs, sub: sub}, nil
}

// WatchMemberRegistered is a free log subscription operation binding the contract event 0x5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2.
//
// Solidity: event MemberRegistered(uint256 pubkey, uint256 index)
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
// Solidity: event MemberRegistered(uint256 pubkey, uint256 index)
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
	Pubkey *big.Int
	Index  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterMemberWithdrawn is a free log retrieval operation binding the contract event 0x62ec3a516d22a993ce5cb4e7593e878c74f4d799dde522a88dc27a994fd5a943.
//
// Solidity: event MemberWithdrawn(uint256 pubkey, uint256 index)
func (_RLN *RLNFilterer) FilterMemberWithdrawn(opts *bind.FilterOpts) (*RLNMemberWithdrawnIterator, error) {

	logs, sub, err := _RLN.contract.FilterLogs(opts, "MemberWithdrawn")
	if err != nil {
		return nil, err
	}
	return &RLNMemberWithdrawnIterator{contract: _RLN.contract, event: "MemberWithdrawn", logs: logs, sub: sub}, nil
}

// WatchMemberWithdrawn is a free log subscription operation binding the contract event 0x62ec3a516d22a993ce5cb4e7593e878c74f4d799dde522a88dc27a994fd5a943.
//
// Solidity: event MemberWithdrawn(uint256 pubkey, uint256 index)
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

// ParseMemberWithdrawn is a log parse operation binding the contract event 0x62ec3a516d22a993ce5cb4e7593e878c74f4d799dde522a88dc27a994fd5a943.
//
// Solidity: event MemberWithdrawn(uint256 pubkey, uint256 index)
func (_RLN *RLNFilterer) ParseMemberWithdrawn(log types.Log) (*RLNMemberWithdrawn, error) {
	event := new(RLNMemberWithdrawn)
	if err := _RLN.contract.UnpackLog(event, "MemberWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
