// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package assets

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

// AssetsMetaData contains all meta data concerning the Assets contract.
var AssetsMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"_decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"_maxSupply\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"CommunityERC20_MaxSupplyLowerThanTotalSupply\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"CommunityERC20_MaxSupplyReached\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"CommunityERC20_MismatchingAddressesAndAmountsLengths\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"addresses\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"amounts\",\"type\":\"uint256[]\"}],\"name\":\"mintTo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newMaxSupply\",\"type\":\"uint256\"}],\"name\":\"setMaxSupply\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a06040523480156200001157600080fd5b50604051620015b6380380620015b6833981016040819052620000349162000188565b8383620000413362000073565b60046200004f8382620002a2565b5060056200005e8282620002a2565b50505060065560ff16608052506200036e9050565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b634e487b7160e01b600052604160045260246000fd5b600082601f830112620000eb57600080fd5b81516001600160401b0380821115620001085762000108620000c3565b604051601f8301601f19908116603f01168101908282118183101715620001335762000133620000c3565b816040528381526020925086838588010111156200015057600080fd5b600091505b8382101562000174578582018301518183018401529082019062000155565b600093810190920192909252949350505050565b600080600080608085870312156200019f57600080fd5b84516001600160401b0380821115620001b757600080fd5b620001c588838901620000d9565b95506020870151915080821115620001dc57600080fd5b50620001eb87828801620000d9565b935050604085015160ff811681146200020357600080fd5b6060959095015193969295505050565b600181811c908216806200022857607f821691505b6020821081036200024957634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156200029d57600081815260208120601f850160051c81016020861015620002785750805b601f850160051c820191505b81811015620002995782815560010162000284565b5050505b505050565b81516001600160401b03811115620002be57620002be620000c3565b620002d681620002cf845462000213565b846200024f565b602080601f8311600181146200030e5760008415620002f55750858301515b600019600386901b1c1916600185901b17855562000299565b600085815260208120601f198616915b828110156200033f578886015182559484019460019091019084016200031e565b50858210156200035e5787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b60805161122c6200038a600039600061018d015261122c6000f3fe608060405234801561001057600080fd5b506004361061011b5760003560e01c806370a08231116100b2578063a457c2d711610081578063d5abeb0111610066578063d5abeb0114610286578063dd62ed3e1461028f578063f2fde38b146102d557600080fd5b8063a457c2d714610260578063a9059cbb1461027357600080fd5b806370a08231146101f2578063715018a6146102285780638da5cb5b1461023057806395d89b411461025857600080fd5b8063313ce567116100ee578063313ce5671461018657806339509351146101b757806369add11d146101ca5780636f8b44b0146101df57600080fd5b806306fdde0314610120578063095ea7b31461013e57806318160ddd1461016157806323b872dd14610173575b600080fd5b6101286102e8565b6040516101359190610ded565b60405180910390f35b61015161014c366004610e82565b61037a565b6040519015158152602001610135565b6003545b604051908152602001610135565b610151610181366004610eac565b610394565b60405160ff7f0000000000000000000000000000000000000000000000000000000000000000168152602001610135565b6101516101c5366004610e82565b6103b8565b6101dd6101d8366004610ff5565b610404565b005b6101dd6101ed3660046110b5565b6104fb565b6101656102003660046110ce565b73ffffffffffffffffffffffffffffffffffffffff1660009081526001602052604090205490565b6101dd610544565b60005460405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610135565b610128610558565b61015161026e366004610e82565b610567565b610151610281366004610e82565b61063d565b61016560065481565b61016561029d3660046110f0565b73ffffffffffffffffffffffffffffffffffffffff918216600090815260026020908152604080832093909416825291909152205490565b6101dd6102e33660046110ce565b61064b565b6060600480546102f790611123565b80601f016020809104026020016040519081016040528092919081815260200182805461032390611123565b80156103705780601f1061034557610100808354040283529160200191610370565b820191906000526020600020905b81548152906001019060200180831161035357829003601f168201915b5050505050905090565b600033610388818585610702565b60019150505b92915050565b6000336103a28582856108b5565b6103ad85858561098c565b506001949350505050565b33600081815260026020908152604080832073ffffffffffffffffffffffffffffffffffffffff8716845290915281205490919061038890829086906103ff9087906111a5565b610702565b61040c610c02565b8051825114610447576040517f825caa1d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b82518110156104f6576000828281518110610467576104676111b8565b602002602001015190506006548161047e60035490565b61048891906111a5565b11156104c0576040517fb9da758f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6104e38483815181106104d5576104d56111b8565b602002602001015182610c83565b50806104ee816111e7565b91505061044a565b505050565b610503610c02565b60035481101561053f576040517f5716872300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600655565b61054c610c02565b6105566000610d78565b565b6060600580546102f790611123565b33600081815260026020908152604080832073ffffffffffffffffffffffffffffffffffffffff8716845290915281205490919083811015610630576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f00000000000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b6103ad8286868403610702565b60003361038881858561098c565b610653610c02565b73ffffffffffffffffffffffffffffffffffffffff81166106f6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f64647265737300000000000000000000000000000000000000000000000000006064820152608401610627565b6106ff81610d78565b50565b73ffffffffffffffffffffffffffffffffffffffff83166107a4576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f72657373000000000000000000000000000000000000000000000000000000006064820152608401610627565b73ffffffffffffffffffffffffffffffffffffffff8216610847576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f73730000000000000000000000000000000000000000000000000000000000006064820152608401610627565b73ffffffffffffffffffffffffffffffffffffffff83811660008181526002602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925910160405180910390a3505050565b73ffffffffffffffffffffffffffffffffffffffff8381166000908152600260209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81146109865781811015610979576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e63650000006044820152606401610627565b6109868484848403610702565b50505050565b73ffffffffffffffffffffffffffffffffffffffff8316610a2f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f64726573730000000000000000000000000000000000000000000000000000006064820152608401610627565b73ffffffffffffffffffffffffffffffffffffffff8216610ad2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f65737300000000000000000000000000000000000000000000000000000000006064820152608401610627565b73ffffffffffffffffffffffffffffffffffffffff831660009081526001602052604090205481811015610b88576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e636500000000000000000000000000000000000000000000000000006064820152608401610627565b73ffffffffffffffffffffffffffffffffffffffff80851660008181526001602052604080822086860390559286168082529083902080548601905591517fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef90610bf59086815260200190565b60405180910390a3610986565b60005473ffffffffffffffffffffffffffffffffffffffff163314610556576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401610627565b73ffffffffffffffffffffffffffffffffffffffff8216610d00576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f2061646472657373006044820152606401610627565b8060036000828254610d1291906111a5565b909155505073ffffffffffffffffffffffffffffffffffffffff82166000818152600160209081526040808320805486019055518481527fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef910160405180910390a35050565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b600060208083528351808285015260005b81811015610e1a57858101830151858201604001528201610dfe565b5060006040828601015260407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f8301168501019250505092915050565b803573ffffffffffffffffffffffffffffffffffffffff81168114610e7d57600080fd5b919050565b60008060408385031215610e9557600080fd5b610e9e83610e59565b946020939093013593505050565b600080600060608486031215610ec157600080fd5b610eca84610e59565b9250610ed860208501610e59565b9150604084013590509250925092565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff81118282101715610f5e57610f5e610ee8565b604052919050565b600067ffffffffffffffff821115610f8057610f80610ee8565b5060051b60200190565b600082601f830112610f9b57600080fd5b81356020610fb0610fab83610f66565b610f17565b82815260059290921b84018101918181019086841115610fcf57600080fd5b8286015b84811015610fea5780358352918301918301610fd3565b509695505050505050565b6000806040838503121561100857600080fd5b823567ffffffffffffffff8082111561102057600080fd5b818501915085601f83011261103457600080fd5b81356020611044610fab83610f66565b82815260059290921b8401810191818101908984111561106357600080fd5b948201945b838610156110885761107986610e59565b82529482019490820190611068565b9650508601359250508082111561109e57600080fd5b506110ab85828601610f8a565b9150509250929050565b6000602082840312156110c757600080fd5b5035919050565b6000602082840312156110e057600080fd5b6110e982610e59565b9392505050565b6000806040838503121561110357600080fd5b61110c83610e59565b915061111a60208401610e59565b90509250929050565b600181811c9082168061113757607f821691505b602082108103611170577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8082018082111561038e5761038e611176565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361121857611218611176565b506001019056fea164736f6c6343000811000a",
}

// AssetsABI is the input ABI used to generate the binding from.
// Deprecated: Use AssetsMetaData.ABI instead.
var AssetsABI = AssetsMetaData.ABI

// AssetsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AssetsMetaData.Bin instead.
var AssetsBin = AssetsMetaData.Bin

// DeployAssets deploys a new Ethereum contract, binding an instance of Assets to it.
func DeployAssets(auth *bind.TransactOpts, backend bind.ContractBackend, _name string, _symbol string, _decimals uint8, _maxSupply *big.Int) (common.Address, *types.Transaction, *Assets, error) {
	parsed, err := AssetsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AssetsBin), backend, _name, _symbol, _decimals, _maxSupply)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Assets{AssetsCaller: AssetsCaller{contract: contract}, AssetsTransactor: AssetsTransactor{contract: contract}, AssetsFilterer: AssetsFilterer{contract: contract}}, nil
}

// Assets is an auto generated Go binding around an Ethereum contract.
type Assets struct {
	AssetsCaller     // Read-only binding to the contract
	AssetsTransactor // Write-only binding to the contract
	AssetsFilterer   // Log filterer for contract events
}

// AssetsCaller is an auto generated read-only Go binding around an Ethereum contract.
type AssetsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssetsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AssetsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssetsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AssetsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssetsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AssetsSession struct {
	Contract     *Assets           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AssetsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AssetsCallerSession struct {
	Contract *AssetsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// AssetsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AssetsTransactorSession struct {
	Contract     *AssetsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AssetsRaw is an auto generated low-level Go binding around an Ethereum contract.
type AssetsRaw struct {
	Contract *Assets // Generic contract binding to access the raw methods on
}

// AssetsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AssetsCallerRaw struct {
	Contract *AssetsCaller // Generic read-only contract binding to access the raw methods on
}

// AssetsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AssetsTransactorRaw struct {
	Contract *AssetsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAssets creates a new instance of Assets, bound to a specific deployed contract.
func NewAssets(address common.Address, backend bind.ContractBackend) (*Assets, error) {
	contract, err := bindAssets(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Assets{AssetsCaller: AssetsCaller{contract: contract}, AssetsTransactor: AssetsTransactor{contract: contract}, AssetsFilterer: AssetsFilterer{contract: contract}}, nil
}

// NewAssetsCaller creates a new read-only instance of Assets, bound to a specific deployed contract.
func NewAssetsCaller(address common.Address, caller bind.ContractCaller) (*AssetsCaller, error) {
	contract, err := bindAssets(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AssetsCaller{contract: contract}, nil
}

// NewAssetsTransactor creates a new write-only instance of Assets, bound to a specific deployed contract.
func NewAssetsTransactor(address common.Address, transactor bind.ContractTransactor) (*AssetsTransactor, error) {
	contract, err := bindAssets(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AssetsTransactor{contract: contract}, nil
}

// NewAssetsFilterer creates a new log filterer instance of Assets, bound to a specific deployed contract.
func NewAssetsFilterer(address common.Address, filterer bind.ContractFilterer) (*AssetsFilterer, error) {
	contract, err := bindAssets(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AssetsFilterer{contract: contract}, nil
}

// bindAssets binds a generic wrapper to an already deployed contract.
func bindAssets(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AssetsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Assets *AssetsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Assets.Contract.AssetsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Assets *AssetsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Assets.Contract.AssetsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Assets *AssetsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Assets.Contract.AssetsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Assets *AssetsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Assets.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Assets *AssetsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Assets.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Assets *AssetsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Assets.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Assets *AssetsCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Assets *AssetsSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _Assets.Contract.Allowance(&_Assets.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Assets *AssetsCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _Assets.Contract.Allowance(&_Assets.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_Assets *AssetsCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_Assets *AssetsSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _Assets.Contract.BalanceOf(&_Assets.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_Assets *AssetsCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _Assets.Contract.BalanceOf(&_Assets.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Assets *AssetsCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Assets *AssetsSession) Decimals() (uint8, error) {
	return _Assets.Contract.Decimals(&_Assets.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Assets *AssetsCallerSession) Decimals() (uint8, error) {
	return _Assets.Contract.Decimals(&_Assets.CallOpts)
}

// MaxSupply is a free data retrieval call binding the contract method 0xd5abeb01.
//
// Solidity: function maxSupply() view returns(uint256)
func (_Assets *AssetsCaller) MaxSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "maxSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxSupply is a free data retrieval call binding the contract method 0xd5abeb01.
//
// Solidity: function maxSupply() view returns(uint256)
func (_Assets *AssetsSession) MaxSupply() (*big.Int, error) {
	return _Assets.Contract.MaxSupply(&_Assets.CallOpts)
}

// MaxSupply is a free data retrieval call binding the contract method 0xd5abeb01.
//
// Solidity: function maxSupply() view returns(uint256)
func (_Assets *AssetsCallerSession) MaxSupply() (*big.Int, error) {
	return _Assets.Contract.MaxSupply(&_Assets.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Assets *AssetsCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Assets *AssetsSession) Name() (string, error) {
	return _Assets.Contract.Name(&_Assets.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Assets *AssetsCallerSession) Name() (string, error) {
	return _Assets.Contract.Name(&_Assets.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Assets *AssetsCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Assets *AssetsSession) Owner() (common.Address, error) {
	return _Assets.Contract.Owner(&_Assets.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Assets *AssetsCallerSession) Owner() (common.Address, error) {
	return _Assets.Contract.Owner(&_Assets.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Assets *AssetsCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Assets *AssetsSession) Symbol() (string, error) {
	return _Assets.Contract.Symbol(&_Assets.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Assets *AssetsCallerSession) Symbol() (string, error) {
	return _Assets.Contract.Symbol(&_Assets.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Assets *AssetsCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Assets *AssetsSession) TotalSupply() (*big.Int, error) {
	return _Assets.Contract.TotalSupply(&_Assets.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Assets *AssetsCallerSession) TotalSupply() (*big.Int, error) {
	return _Assets.Contract.TotalSupply(&_Assets.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Assets *AssetsTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Assets *AssetsSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.Approve(&_Assets.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Assets *AssetsTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.Approve(&_Assets.TransactOpts, spender, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_Assets *AssetsTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_Assets *AssetsSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.DecreaseAllowance(&_Assets.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_Assets *AssetsTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.DecreaseAllowance(&_Assets.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_Assets *AssetsTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_Assets *AssetsSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.IncreaseAllowance(&_Assets.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_Assets *AssetsTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.IncreaseAllowance(&_Assets.TransactOpts, spender, addedValue)
}

// MintTo is a paid mutator transaction binding the contract method 0x69add11d.
//
// Solidity: function mintTo(address[] addresses, uint256[] amounts) returns()
func (_Assets *AssetsTransactor) MintTo(opts *bind.TransactOpts, addresses []common.Address, amounts []*big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "mintTo", addresses, amounts)
}

// MintTo is a paid mutator transaction binding the contract method 0x69add11d.
//
// Solidity: function mintTo(address[] addresses, uint256[] amounts) returns()
func (_Assets *AssetsSession) MintTo(addresses []common.Address, amounts []*big.Int) (*types.Transaction, error) {
	return _Assets.Contract.MintTo(&_Assets.TransactOpts, addresses, amounts)
}

// MintTo is a paid mutator transaction binding the contract method 0x69add11d.
//
// Solidity: function mintTo(address[] addresses, uint256[] amounts) returns()
func (_Assets *AssetsTransactorSession) MintTo(addresses []common.Address, amounts []*big.Int) (*types.Transaction, error) {
	return _Assets.Contract.MintTo(&_Assets.TransactOpts, addresses, amounts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Assets *AssetsTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Assets *AssetsSession) RenounceOwnership() (*types.Transaction, error) {
	return _Assets.Contract.RenounceOwnership(&_Assets.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Assets *AssetsTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Assets.Contract.RenounceOwnership(&_Assets.TransactOpts)
}

// SetMaxSupply is a paid mutator transaction binding the contract method 0x6f8b44b0.
//
// Solidity: function setMaxSupply(uint256 newMaxSupply) returns()
func (_Assets *AssetsTransactor) SetMaxSupply(opts *bind.TransactOpts, newMaxSupply *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "setMaxSupply", newMaxSupply)
}

// SetMaxSupply is a paid mutator transaction binding the contract method 0x6f8b44b0.
//
// Solidity: function setMaxSupply(uint256 newMaxSupply) returns()
func (_Assets *AssetsSession) SetMaxSupply(newMaxSupply *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.SetMaxSupply(&_Assets.TransactOpts, newMaxSupply)
}

// SetMaxSupply is a paid mutator transaction binding the contract method 0x6f8b44b0.
//
// Solidity: function setMaxSupply(uint256 newMaxSupply) returns()
func (_Assets *AssetsTransactorSession) SetMaxSupply(newMaxSupply *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.SetMaxSupply(&_Assets.TransactOpts, newMaxSupply)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_Assets *AssetsTransactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_Assets *AssetsSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.Transfer(&_Assets.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_Assets *AssetsTransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.Transfer(&_Assets.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_Assets *AssetsTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_Assets *AssetsSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.TransferFrom(&_Assets.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_Assets *AssetsTransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.TransferFrom(&_Assets.TransactOpts, from, to, amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Assets *AssetsTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Assets *AssetsSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Assets.Contract.TransferOwnership(&_Assets.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Assets *AssetsTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Assets.Contract.TransferOwnership(&_Assets.TransactOpts, newOwner)
}

// AssetsApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the Assets contract.
type AssetsApprovalIterator struct {
	Event *AssetsApproval // Event containing the contract specifics and raw log

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
func (it *AssetsApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssetsApproval)
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
		it.Event = new(AssetsApproval)
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
func (it *AssetsApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssetsApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssetsApproval represents a Approval event raised by the Assets contract.
type AssetsApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Assets *AssetsFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*AssetsApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Assets.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &AssetsApprovalIterator{contract: _Assets.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Assets *AssetsFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *AssetsApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Assets.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssetsApproval)
				if err := _Assets.contract.UnpackLog(event, "Approval", log); err != nil {
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
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Assets *AssetsFilterer) ParseApproval(log types.Log) (*AssetsApproval, error) {
	event := new(AssetsApproval)
	if err := _Assets.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AssetsOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Assets contract.
type AssetsOwnershipTransferredIterator struct {
	Event *AssetsOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *AssetsOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssetsOwnershipTransferred)
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
		it.Event = new(AssetsOwnershipTransferred)
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
func (it *AssetsOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssetsOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssetsOwnershipTransferred represents a OwnershipTransferred event raised by the Assets contract.
type AssetsOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Assets *AssetsFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*AssetsOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Assets.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &AssetsOwnershipTransferredIterator{contract: _Assets.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Assets *AssetsFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *AssetsOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Assets.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssetsOwnershipTransferred)
				if err := _Assets.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Assets *AssetsFilterer) ParseOwnershipTransferred(log types.Log) (*AssetsOwnershipTransferred, error) {
	event := new(AssetsOwnershipTransferred)
	if err := _Assets.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AssetsTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the Assets contract.
type AssetsTransferIterator struct {
	Event *AssetsTransfer // Event containing the contract specifics and raw log

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
func (it *AssetsTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssetsTransfer)
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
		it.Event = new(AssetsTransfer)
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
func (it *AssetsTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssetsTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssetsTransfer represents a Transfer event raised by the Assets contract.
type AssetsTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Assets *AssetsFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*AssetsTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Assets.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &AssetsTransferIterator{contract: _Assets.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Assets *AssetsFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *AssetsTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Assets.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssetsTransfer)
				if err := _Assets.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Assets *AssetsFilterer) ParseTransfer(log types.Log) (*AssetsTransfer, error) {
	event := new(AssetsTransfer)
	if err := _Assets.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
