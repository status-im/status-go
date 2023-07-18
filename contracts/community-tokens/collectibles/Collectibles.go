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
const CollectiblesABI = "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"_maxSupply\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"_remoteBurnable\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"_transferable\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"_baseTokenURI\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"_ownerToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_masterToken\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"approved\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"baseTokenURI\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"getApproved\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"masterToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"addresses\",\"type\":\"address[]\"}],\"name\":\"mintTo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"mintedCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"ownerOf\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ownerToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"tokenIds\",\"type\":\"uint256[]\"}],\"name\":\"remoteBurn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"remoteBurnable\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"setApprovalForAll\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newMaxSupply\",\"type\":\"uint256\"}],\"name\":\"setMaxSupply\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"tokenByIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"tokenOfOwnerByIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"tokenURI\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"transferable\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// CollectiblesBin is the compiled bytecode used for deploying new contracts.
var CollectiblesBin = "0x60806040523480156200001157600080fd5b50604051620026a1380380620026a183398101604081905262000034916200023b565b8787878787878787878760006200004c8382620003b0565b5060016200005b8282620003b0565b505050600b869055600d805461ffff60a01b1916600160a01b8715150260ff60a81b191617600160a81b86151502179055600e6200009a8482620003b0565b50600c80546001600160a01b038085166001600160a01b03199283168117909355600d805491851691909216179055151580620000e15750600d546001600160a01b031615155b620001325760405162461bcd60e51b815260206004820152601f60248201527f6f776e6572206f72206d617374657220746f6b656e7320726571756972656400604482015260640160405180910390fd5b505050505050505050505050505050506200047c565b634e487b7160e01b600052604160045260246000fd5b600082601f8301126200017057600080fd5b81516001600160401b03808211156200018d576200018d62000148565b604051601f8301601f19908116603f01168101908282118183101715620001b857620001b862000148565b81604052838152602092508683858801011115620001d557600080fd5b600091505b83821015620001f95785820183015181830184015290820190620001da565b600093810190920192909252949350505050565b805180151581146200021e57600080fd5b919050565b80516001600160a01b03811681146200021e57600080fd5b600080600080600080600080610100898b0312156200025957600080fd5b88516001600160401b03808211156200027157600080fd5b6200027f8c838d016200015e565b995060208b01519150808211156200029657600080fd5b620002a48c838d016200015e565b985060408b01519750620002bb60608c016200020d565b9650620002cb60808c016200020d565b955060a08b0151915080821115620002e257600080fd5b50620002f18b828c016200015e565b9350506200030260c08a0162000223565b91506200031260e08a0162000223565b90509295985092959890939650565b600181811c908216806200033657607f821691505b6020821081036200035757634e487b7160e01b600052602260045260246000fd5b50919050565b601f821115620003ab57600081815260208120601f850160051c81016020861015620003865750805b601f850160051c820191505b81811015620003a75782815560010162000392565b5050505b505050565b81516001600160401b03811115620003cc57620003cc62000148565b620003e481620003dd845462000321565b846200035d565b602080601f8311600181146200041c5760008415620004035750858301515b600019600386901b1c1916600185901b178555620003a7565b600085815260208120601f198616915b828110156200044d578886015182559484019460019091019084016200042c565b50858210156200046c5787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b612215806200048c6000396000f3fe608060405234801561001057600080fd5b506004361061018e5760003560e01c806365371883116100de578063b88d4fde11610097578063cf721b1511610071578063cf721b151461035c578063d547cfb714610364578063d5abeb011461036c578063e985e9c51461037557600080fd5b8063b88d4fde14610323578063c87b56dd14610336578063ce7c8b491461034957600080fd5b806365371883146102bb5780636f8b44b0146102ce57806370a08231146102e157806392ff0d31146102f457806395d89b4114610308578063a22cb4651461031057600080fd5b806323b872dd1161014b57806342842e0e1161012557806342842e0e1461026f5780634f6ccce7146102825780634fb95e02146102955780636352211e146102a857600080fd5b806323b872dd146102365780632bb5e31e146102495780632f745c591461025c57600080fd5b806301ffc9a71461019357806306fdde03146101bb578063081812fc146101d0578063095ea7b3146101fb578063101639f51461021057806318160ddd14610224575b600080fd5b6101a66101a1366004611b83565b6103b1565b60405190151581526020015b60405180910390f35b6101c36103c2565b6040516101b29190611bf0565b6101e36101de366004611c03565b610454565b6040516001600160a01b0390911681526020016101b2565b61020e610209366004611c38565b61047b565b005b600d546101a690600160a01b900460ff1681565b6008545b6040519081526020016101b2565b61020e610244366004611c62565b610595565b600d546101e3906001600160a01b031681565b61022861026a366004611c38565b6105c6565b61020e61027d366004611c62565b61065c565b610228610290366004611c03565b610677565b61020e6102a3366004611d09565b61070a565b6101e36102b6366004611c03565b6108c3565b600c546101e3906001600160a01b031681565b61020e6102dc366004611c03565b610923565b6102286102ef366004611d9f565b610aab565b600d546101a690600160a81b900460ff1681565b6101c3610b31565b61020e61031e366004611dba565b610b40565b61020e610331366004611df6565b610b4b565b6101c3610344366004611c03565b610b83565b61020e610357366004611eb6565b610bea565b610228610d71565b6101c3610d81565b610228600b5481565b6101a6610383366004611f43565b6001600160a01b03918216600090815260056020908152604080832093909416825291909152205460ff1690565b60006103bc82610e0f565b92915050565b6060600080546103d190611f76565b80601f01602080910402602001604051908101604052809291908181526020018280546103fd90611f76565b801561044a5780601f1061041f5761010080835404028352916020019161044a565b820191906000526020600020905b81548152906001019060200180831161042d57829003601f168201915b5050505050905090565b600061045f82610e34565b506000908152600460205260409020546001600160a01b031690565b6000610486826108c3565b9050806001600160a01b0316836001600160a01b0316036104f85760405162461bcd60e51b815260206004820152602160248201527f4552433732313a20617070726f76616c20746f2063757272656e74206f776e656044820152603960f91b60648201526084015b60405180910390fd5b336001600160a01b038216148061051457506105148133610383565b6105865760405162461bcd60e51b815260206004820152603d60248201527f4552433732313a20617070726f76652063616c6c6572206973206e6f7420746f60448201527f6b656e206f776e6572206f7220617070726f76656420666f7220616c6c00000060648201526084016104ef565b6105908383610e93565b505050565b61059f3382610f01565b6105bb5760405162461bcd60e51b81526004016104ef90611fb0565b610590838383610f80565b60006105d183610aab565b82106106335760405162461bcd60e51b815260206004820152602b60248201527f455243373231456e756d657261626c653a206f776e657220696e646578206f7560448201526a74206f6620626f756e647360a81b60648201526084016104ef565b506001600160a01b03919091166000908152600660209081526040808320938352929052205490565b61059083838360405180602001604052806000815250610b4b565b600061068260085490565b82106106e55760405162461bcd60e51b815260206004820152602c60248201527f455243373231456e756d657261626c653a20676c6f62616c20696e646578206f60448201526b7574206f6620626f756e647360a01b60648201526084016104ef565b600882815481106106f8576106f8611ffd565b90600052602060002001549050919050565b600c546001600160a01b0316158061078c5750600c546040516370a0823160e01b81523360048201526000916001600160a01b0316906370a0823190602401602060405180830381865afa158015610766573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061078a9190612013565b115b806108145750600d546001600160a01b031615806108145750600d546040516370a0823160e01b81523360048201526000916001600160a01b0316906370a0823190602401602060405180830381865afa1580156107ee573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108129190612013565b115b6108305760405162461bcd60e51b81526004016104ef9061202c565b600d54600160a01b900460ff1661087f5760405162461bcd60e51b81526020600482015260136024820152724e4f545f52454d4f54455f4255524e41424c4560681b60448201526064016104ef565b60005b81518110156108bf576108ad8282815181106108a0576108a0611ffd565b60200260200101516110f1565b806108b78161206a565b915050610882565b5050565b6000818152600260205260408120546001600160a01b0316806103bc5760405162461bcd60e51b8152602060048201526018602482015277115490cdcc8c4e881a5b9d985b1a59081d1bdad95b88125160421b60448201526064016104ef565b600c546001600160a01b031615806109a55750600c546040516370a0823160e01b81523360048201526000916001600160a01b0316906370a0823190602401602060405180830381865afa15801561097f573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109a39190612013565b115b80610a2d5750600d546001600160a01b03161580610a2d5750600d546040516370a0823160e01b81523360048201526000916001600160a01b0316906370a0823190602401602060405180830381865afa158015610a07573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610a2b9190612013565b115b610a495760405162461bcd60e51b81526004016104ef9061202c565b600854811015610aa65760405162461bcd60e51b815260206004820152602260248201527f4d41585f535550504c595f4c4f5745525f5448414e5f544f54414c5f535550506044820152614c5960f01b60648201526084016104ef565b600b55565b60006001600160a01b038216610b155760405162461bcd60e51b815260206004820152602960248201527f4552433732313a2061646472657373207a65726f206973206e6f7420612076616044820152683634b21037bbb732b960b91b60648201526084016104ef565b506001600160a01b031660009081526003602052604090205490565b6060600180546103d190611f76565b6108bf338383611194565b610b553383610f01565b610b715760405162461bcd60e51b81526004016104ef90611fb0565b610b7d84848484611262565b50505050565b6060610b8e82610e34565b6000610b98611295565b90506000815111610bb85760405180602001604052806000815250610be3565b80610bc2846112a4565b604051602001610bd3929190612083565b6040516020818303038152906040525b9392505050565b600c546001600160a01b03161580610c6c5750600c546040516370a0823160e01b81523360048201526000916001600160a01b0316906370a0823190602401602060405180830381865afa158015610c46573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610c6a9190612013565b115b80610cf45750600d546001600160a01b03161580610cf45750600d546040516370a0823160e01b81523360048201526000916001600160a01b0316906370a0823190602401602060405180830381865afa158015610cce573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610cf29190612013565b115b610d105760405162461bcd60e51b81526004016104ef9061202c565b600b548151600a54610d2291906120b2565b1115610d655760405162461bcd60e51b815260206004820152601260248201527113505617d4d55414131657d4915050d2115160721b60448201526064016104ef565b610d6e81611337565b50565b6000610d7c600a5490565b905090565b600e8054610d8e90611f76565b80601f0160208091040260200160405190810160405280929190818152602001828054610dba90611f76565b8015610e075780601f10610ddc57610100808354040283529160200191610e07565b820191906000526020600020905b815481529060010190602001808311610dea57829003601f168201915b505050505081565b60006001600160e01b0319821663780e9d6360e01b14806103bc57506103bc8261139e565b6000818152600260205260409020546001600160a01b0316610d6e5760405162461bcd60e51b8152602060048201526018602482015277115490cdcc8c4e881a5b9d985b1a59081d1bdad95b88125160421b60448201526064016104ef565b600081815260046020526040902080546001600160a01b0319166001600160a01b0384169081179091558190610ec8826108c3565b6001600160a01b03167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92560405160405180910390a45050565b600080610f0d836108c3565b9050806001600160a01b0316846001600160a01b03161480610f5457506001600160a01b0380821660009081526005602090815260408083209388168352929052205460ff165b80610f785750836001600160a01b0316610f6d84610454565b6001600160a01b0316145b949350505050565b826001600160a01b0316610f93826108c3565b6001600160a01b031614610fb95760405162461bcd60e51b81526004016104ef906120c5565b6001600160a01b03821661101b5760405162461bcd60e51b8152602060048201526024808201527f4552433732313a207472616e7366657220746f20746865207a65726f206164646044820152637265737360e01b60648201526084016104ef565b61102883838360016113ee565b826001600160a01b031661103b826108c3565b6001600160a01b0316146110615760405162461bcd60e51b81526004016104ef906120c5565b600081815260046020908152604080832080546001600160a01b03199081169091556001600160a01b0387811680865260038552838620805460001901905590871680865283862080546001019055868652600290945282852080549092168417909155905184937fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef91a4505050565b60006110fc826108c3565b905061110c8160008460016113ee565b611115826108c3565b600083815260046020908152604080832080546001600160a01b03199081169091556001600160a01b0385168085526003845282852080546000190190558785526002909352818420805490911690555192935084927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef908390a45050565b816001600160a01b0316836001600160a01b0316036111f55760405162461bcd60e51b815260206004820152601960248201527f4552433732313a20617070726f766520746f2063616c6c65720000000000000060448201526064016104ef565b6001600160a01b03838116600081815260056020908152604080832094871680845294825291829020805460ff191686151590811790915591519182527f17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31910160405180910390a3505050565b61126d848484610f80565b61127984848484611470565b610b7d5760405162461bcd60e51b81526004016104ef9061210a565b6060600e80546103d190611f76565b606060006112b183611571565b600101905060008167ffffffffffffffff8111156112d1576112d1611c9e565b6040519080825280601f01601f1916602001820160405280156112fb576020820181803683370190505b5090508181016020015b600019016f181899199a1a9b1b9c1cb0b131b232b360811b600a86061a8153600a850494508461130557509392505050565b60005b81518110156108bf5761137e82828151811061135857611358611ffd565b6020026020010151611369600a5490565b60405180602001604052806000815250611649565b61138c600a80546001019055565b806113968161206a565b91505061133a565b60006001600160e01b031982166380ac58cd60e01b14806113cf57506001600160e01b03198216635b5e139f60e01b145b806103bc57506301ffc9a760e01b6001600160e01b03198316146103bc565b6001600160a01b0384161580159061140e57506001600160a01b03831615155b80156114245750600d54600160a81b900460ff16155b156114645760405162461bcd60e51b815260206004820152601060248201526f6e6f74207472616e7366657261626c6560801b60448201526064016104ef565b610b7d8484848461167c565b60006001600160a01b0384163b1561156657604051630a85bd0160e11b81526001600160a01b0385169063150b7a02906114b490339089908890889060040161215c565b6020604051808303816000875af19250505080156114ef575060408051601f3d908101601f191682019092526114ec91810190612199565b60015b61154c573d80801561151d576040519150601f19603f3d011682016040523d82523d6000602084013e611522565b606091505b5080516000036115445760405162461bcd60e51b81526004016104ef9061210a565b805181602001fd5b6001600160e01b031916630a85bd0160e11b149050610f78565b506001949350505050565b60008072184f03e93ff9f4daa797ed6e38ed64bf6a1f0160401b83106115b05772184f03e93ff9f4daa797ed6e38ed64bf6a1f0160401b830492506040015b6d04ee2d6d415b85acef810000000083106115dc576d04ee2d6d415b85acef8100000000830492506020015b662386f26fc1000083106115fa57662386f26fc10000830492506010015b6305f5e1008310611612576305f5e100830492506008015b612710831061162657612710830492506004015b60648310611638576064830492506002015b600a83106103bc5760010192915050565b61165383836117bc565b6116606000848484611470565b6105905760405162461bcd60e51b81526004016104ef9061210a565b61168884848484611955565b60018111156116f75760405162461bcd60e51b815260206004820152603560248201527f455243373231456e756d657261626c653a20636f6e7365637574697665207472604482015274185b9cd9995c9cc81b9bdd081cdd5c1c1bdc9d1959605a1b60648201526084016104ef565b816001600160a01b0385166117535761174e81600880546000838152600960205260408120829055600182018355919091527ff3f7a9fe364faab93b216da50a3214154f22a0a2b415b23a84c8169e8b636ee30155565b611776565b836001600160a01b0316856001600160a01b0316146117765761177685826119dd565b6001600160a01b0384166117925761178d81611a7a565b6117b5565b846001600160a01b0316846001600160a01b0316146117b5576117b58482611b29565b5050505050565b6001600160a01b0382166118125760405162461bcd60e51b815260206004820181905260248201527f4552433732313a206d696e7420746f20746865207a65726f206164647265737360448201526064016104ef565b6000818152600260205260409020546001600160a01b0316156118775760405162461bcd60e51b815260206004820152601c60248201527f4552433732313a20746f6b656e20616c7265616479206d696e7465640000000060448201526064016104ef565b6118856000838360016113ee565b6000818152600260205260409020546001600160a01b0316156118ea5760405162461bcd60e51b815260206004820152601c60248201527f4552433732313a20746f6b656e20616c7265616479206d696e7465640000000060448201526064016104ef565b6001600160a01b038216600081815260036020908152604080832080546001019055848352600290915280822080546001600160a01b0319168417905551839291907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef908290a45050565b6001811115610b7d576001600160a01b0384161561199b576001600160a01b038416600090815260036020526040812080548392906119959084906121b6565b90915550505b6001600160a01b03831615610b7d576001600160a01b038316600090815260036020526040812080548392906119d29084906120b2565b909155505050505050565b600060016119ea84610aab565b6119f491906121b6565b600083815260076020526040902054909150808214611a47576001600160a01b03841660009081526006602090815260408083208584528252808320548484528184208190558352600790915290208190555b5060009182526007602090815260408084208490556001600160a01b039094168352600681528383209183525290812055565b600854600090611a8c906001906121b6565b60008381526009602052604081205460088054939450909284908110611ab457611ab4611ffd565b906000526020600020015490508060088381548110611ad557611ad5611ffd565b6000918252602080832090910192909255828152600990915260408082208490558582528120556008805480611b0d57611b0d6121c9565b6001900381819060005260206000200160009055905550505050565b6000611b3483610aab565b6001600160a01b039093166000908152600660209081526040808320868452825280832085905593825260079052919091209190915550565b6001600160e01b031981168114610d6e57600080fd5b600060208284031215611b9557600080fd5b8135610be381611b6d565b60005b83811015611bbb578181015183820152602001611ba3565b50506000910152565b60008151808452611bdc816020860160208601611ba0565b601f01601f19169290920160200192915050565b602081526000610be36020830184611bc4565b600060208284031215611c1557600080fd5b5035919050565b80356001600160a01b0381168114611c3357600080fd5b919050565b60008060408385031215611c4b57600080fd5b611c5483611c1c565b946020939093013593505050565b600080600060608486031215611c7757600080fd5b611c8084611c1c565b9250611c8e60208501611c1c565b9150604084013590509250925092565b634e487b7160e01b600052604160045260246000fd5b604051601f8201601f1916810167ffffffffffffffff81118282101715611cdd57611cdd611c9e565b604052919050565b600067ffffffffffffffff821115611cff57611cff611c9e565b5060051b60200190565b60006020808385031215611d1c57600080fd5b823567ffffffffffffffff811115611d3357600080fd5b8301601f81018513611d4457600080fd5b8035611d57611d5282611ce5565b611cb4565b81815260059190911b82018301908381019087831115611d7657600080fd5b928401925b82841015611d9457833582529284019290840190611d7b565b979650505050505050565b600060208284031215611db157600080fd5b610be382611c1c565b60008060408385031215611dcd57600080fd5b611dd683611c1c565b915060208301358015158114611deb57600080fd5b809150509250929050565b60008060008060808587031215611e0c57600080fd5b611e1585611c1c565b93506020611e24818701611c1c565b935060408601359250606086013567ffffffffffffffff80821115611e4857600080fd5b818801915088601f830112611e5c57600080fd5b813581811115611e6e57611e6e611c9e565b611e80601f8201601f19168501611cb4565b91508082528984828501011115611e9657600080fd5b808484018584013760008482840101525080935050505092959194509250565b60006020808385031215611ec957600080fd5b823567ffffffffffffffff811115611ee057600080fd5b8301601f81018513611ef157600080fd5b8035611eff611d5282611ce5565b81815260059190911b82018301908381019087831115611f1e57600080fd5b928401925b82841015611d9457611f3484611c1c565b82529284019290840190611f23565b60008060408385031215611f5657600080fd5b611f5f83611c1c565b9150611f6d60208401611c1c565b90509250929050565b600181811c90821680611f8a57607f821691505b602082108103611faa57634e487b7160e01b600052602260045260246000fd5b50919050565b6020808252602d908201527f4552433732313a2063616c6c6572206973206e6f7420746f6b656e206f776e6560408201526c1c881bdc88185c1c1c9bdd9959609a1b606082015260800190565b634e487b7160e01b600052603260045260246000fd5b60006020828403121561202557600080fd5b5051919050565b6020808252600e908201526d139bdd08185d5d1a1bdc9a5e995960921b604082015260600190565b634e487b7160e01b600052601160045260246000fd5b60006001820161207c5761207c612054565b5060010190565b60008351612095818460208801611ba0565b8351908301906120a9818360208801611ba0565b01949350505050565b808201808211156103bc576103bc612054565b60208082526025908201527f4552433732313a207472616e736665722066726f6d20696e636f72726563742060408201526437bbb732b960d91b606082015260800190565b60208082526032908201527f4552433732313a207472616e7366657220746f206e6f6e20455243373231526560408201527131b2b4bb32b91034b6b83632b6b2b73a32b960711b606082015260800190565b6001600160a01b038581168252841660208201526040810183905260806060820181905260009061218f90830184611bc4565b9695505050505050565b6000602082840312156121ab57600080fd5b8151610be381611b6d565b818103818111156103bc576103bc612054565b634e487b7160e01b600052603160045260246000fdfea264697066735822122086a4861cc4cd7011cfd13ba461d6fc5087ae7a6d4798fad1c6352203bccf0a4d64736f6c63430008110033"

// DeployCollectibles deploys a new Ethereum contract, binding an instance of Collectibles to it.
func DeployCollectibles(auth *bind.TransactOpts, backend bind.ContractBackend, _name string, _symbol string, _maxSupply *big.Int, _remoteBurnable bool, _transferable bool, _baseTokenURI string, _ownerToken common.Address, _masterToken common.Address) (common.Address, *types.Transaction, *Collectibles, error) {
	parsed, err := abi.JSON(strings.NewReader(CollectiblesABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(CollectiblesBin), backend, _name, _symbol, _maxSupply, _remoteBurnable, _transferable, _baseTokenURI, _ownerToken, _masterToken)
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

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_Collectibles *CollectiblesCaller) BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "balanceOf", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_Collectibles *CollectiblesSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _Collectibles.Contract.BalanceOf(&_Collectibles.CallOpts, owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_Collectibles *CollectiblesCallerSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _Collectibles.Contract.BalanceOf(&_Collectibles.CallOpts, owner)
}

// BaseTokenURI is a free data retrieval call binding the contract method 0xd547cfb7.
//
// Solidity: function baseTokenURI() view returns(string)
func (_Collectibles *CollectiblesCaller) BaseTokenURI(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "baseTokenURI")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// BaseTokenURI is a free data retrieval call binding the contract method 0xd547cfb7.
//
// Solidity: function baseTokenURI() view returns(string)
func (_Collectibles *CollectiblesSession) BaseTokenURI() (string, error) {
	return _Collectibles.Contract.BaseTokenURI(&_Collectibles.CallOpts)
}

// BaseTokenURI is a free data retrieval call binding the contract method 0xd547cfb7.
//
// Solidity: function baseTokenURI() view returns(string)
func (_Collectibles *CollectiblesCallerSession) BaseTokenURI() (string, error) {
	return _Collectibles.Contract.BaseTokenURI(&_Collectibles.CallOpts)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Collectibles *CollectiblesCaller) GetApproved(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "getApproved", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Collectibles *CollectiblesSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _Collectibles.Contract.GetApproved(&_Collectibles.CallOpts, tokenId)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_Collectibles *CollectiblesCallerSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _Collectibles.Contract.GetApproved(&_Collectibles.CallOpts, tokenId)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_Collectibles *CollectiblesCaller) IsApprovedForAll(opts *bind.CallOpts, owner common.Address, operator common.Address) (bool, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "isApprovedForAll", owner, operator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_Collectibles *CollectiblesSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _Collectibles.Contract.IsApprovedForAll(&_Collectibles.CallOpts, owner, operator)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_Collectibles *CollectiblesCallerSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _Collectibles.Contract.IsApprovedForAll(&_Collectibles.CallOpts, owner, operator)
}

// MasterToken is a free data retrieval call binding the contract method 0x2bb5e31e.
//
// Solidity: function masterToken() view returns(address)
func (_Collectibles *CollectiblesCaller) MasterToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "masterToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MasterToken is a free data retrieval call binding the contract method 0x2bb5e31e.
//
// Solidity: function masterToken() view returns(address)
func (_Collectibles *CollectiblesSession) MasterToken() (common.Address, error) {
	return _Collectibles.Contract.MasterToken(&_Collectibles.CallOpts)
}

// MasterToken is a free data retrieval call binding the contract method 0x2bb5e31e.
//
// Solidity: function masterToken() view returns(address)
func (_Collectibles *CollectiblesCallerSession) MasterToken() (common.Address, error) {
	return _Collectibles.Contract.MasterToken(&_Collectibles.CallOpts)
}

// MaxSupply is a free data retrieval call binding the contract method 0xd5abeb01.
//
// Solidity: function maxSupply() view returns(uint256)
func (_Collectibles *CollectiblesCaller) MaxSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "maxSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxSupply is a free data retrieval call binding the contract method 0xd5abeb01.
//
// Solidity: function maxSupply() view returns(uint256)
func (_Collectibles *CollectiblesSession) MaxSupply() (*big.Int, error) {
	return _Collectibles.Contract.MaxSupply(&_Collectibles.CallOpts)
}

// MaxSupply is a free data retrieval call binding the contract method 0xd5abeb01.
//
// Solidity: function maxSupply() view returns(uint256)
func (_Collectibles *CollectiblesCallerSession) MaxSupply() (*big.Int, error) {
	return _Collectibles.Contract.MaxSupply(&_Collectibles.CallOpts)
}

// MintedCount is a free data retrieval call binding the contract method 0xcf721b15.
//
// Solidity: function mintedCount() view returns(uint256)
func (_Collectibles *CollectiblesCaller) MintedCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "mintedCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MintedCount is a free data retrieval call binding the contract method 0xcf721b15.
//
// Solidity: function mintedCount() view returns(uint256)
func (_Collectibles *CollectiblesSession) MintedCount() (*big.Int, error) {
	return _Collectibles.Contract.MintedCount(&_Collectibles.CallOpts)
}

// MintedCount is a free data retrieval call binding the contract method 0xcf721b15.
//
// Solidity: function mintedCount() view returns(uint256)
func (_Collectibles *CollectiblesCallerSession) MintedCount() (*big.Int, error) {
	return _Collectibles.Contract.MintedCount(&_Collectibles.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Collectibles *CollectiblesCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Collectibles *CollectiblesSession) Name() (string, error) {
	return _Collectibles.Contract.Name(&_Collectibles.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Collectibles *CollectiblesCallerSession) Name() (string, error) {
	return _Collectibles.Contract.Name(&_Collectibles.CallOpts)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_Collectibles *CollectiblesCaller) OwnerOf(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "ownerOf", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_Collectibles *CollectiblesSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _Collectibles.Contract.OwnerOf(&_Collectibles.CallOpts, tokenId)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_Collectibles *CollectiblesCallerSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _Collectibles.Contract.OwnerOf(&_Collectibles.CallOpts, tokenId)
}

// OwnerToken is a free data retrieval call binding the contract method 0x65371883.
//
// Solidity: function ownerToken() view returns(address)
func (_Collectibles *CollectiblesCaller) OwnerToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "ownerToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerToken is a free data retrieval call binding the contract method 0x65371883.
//
// Solidity: function ownerToken() view returns(address)
func (_Collectibles *CollectiblesSession) OwnerToken() (common.Address, error) {
	return _Collectibles.Contract.OwnerToken(&_Collectibles.CallOpts)
}

// OwnerToken is a free data retrieval call binding the contract method 0x65371883.
//
// Solidity: function ownerToken() view returns(address)
func (_Collectibles *CollectiblesCallerSession) OwnerToken() (common.Address, error) {
	return _Collectibles.Contract.OwnerToken(&_Collectibles.CallOpts)
}

// RemoteBurnable is a free data retrieval call binding the contract method 0x101639f5.
//
// Solidity: function remoteBurnable() view returns(bool)
func (_Collectibles *CollectiblesCaller) RemoteBurnable(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "remoteBurnable")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// RemoteBurnable is a free data retrieval call binding the contract method 0x101639f5.
//
// Solidity: function remoteBurnable() view returns(bool)
func (_Collectibles *CollectiblesSession) RemoteBurnable() (bool, error) {
	return _Collectibles.Contract.RemoteBurnable(&_Collectibles.CallOpts)
}

// RemoteBurnable is a free data retrieval call binding the contract method 0x101639f5.
//
// Solidity: function remoteBurnable() view returns(bool)
func (_Collectibles *CollectiblesCallerSession) RemoteBurnable() (bool, error) {
	return _Collectibles.Contract.RemoteBurnable(&_Collectibles.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Collectibles *CollectiblesCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Collectibles *CollectiblesSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Collectibles.Contract.SupportsInterface(&_Collectibles.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Collectibles *CollectiblesCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Collectibles.Contract.SupportsInterface(&_Collectibles.CallOpts, interfaceId)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Collectibles *CollectiblesCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Collectibles *CollectiblesSession) Symbol() (string, error) {
	return _Collectibles.Contract.Symbol(&_Collectibles.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Collectibles *CollectiblesCallerSession) Symbol() (string, error) {
	return _Collectibles.Contract.Symbol(&_Collectibles.CallOpts)
}

// TokenByIndex is a free data retrieval call binding the contract method 0x4f6ccce7.
//
// Solidity: function tokenByIndex(uint256 index) view returns(uint256)
func (_Collectibles *CollectiblesCaller) TokenByIndex(opts *bind.CallOpts, index *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "tokenByIndex", index)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokenByIndex is a free data retrieval call binding the contract method 0x4f6ccce7.
//
// Solidity: function tokenByIndex(uint256 index) view returns(uint256)
func (_Collectibles *CollectiblesSession) TokenByIndex(index *big.Int) (*big.Int, error) {
	return _Collectibles.Contract.TokenByIndex(&_Collectibles.CallOpts, index)
}

// TokenByIndex is a free data retrieval call binding the contract method 0x4f6ccce7.
//
// Solidity: function tokenByIndex(uint256 index) view returns(uint256)
func (_Collectibles *CollectiblesCallerSession) TokenByIndex(index *big.Int) (*big.Int, error) {
	return _Collectibles.Contract.TokenByIndex(&_Collectibles.CallOpts, index)
}

// TokenOfOwnerByIndex is a free data retrieval call binding the contract method 0x2f745c59.
//
// Solidity: function tokenOfOwnerByIndex(address owner, uint256 index) view returns(uint256)
func (_Collectibles *CollectiblesCaller) TokenOfOwnerByIndex(opts *bind.CallOpts, owner common.Address, index *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "tokenOfOwnerByIndex", owner, index)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokenOfOwnerByIndex is a free data retrieval call binding the contract method 0x2f745c59.
//
// Solidity: function tokenOfOwnerByIndex(address owner, uint256 index) view returns(uint256)
func (_Collectibles *CollectiblesSession) TokenOfOwnerByIndex(owner common.Address, index *big.Int) (*big.Int, error) {
	return _Collectibles.Contract.TokenOfOwnerByIndex(&_Collectibles.CallOpts, owner, index)
}

// TokenOfOwnerByIndex is a free data retrieval call binding the contract method 0x2f745c59.
//
// Solidity: function tokenOfOwnerByIndex(address owner, uint256 index) view returns(uint256)
func (_Collectibles *CollectiblesCallerSession) TokenOfOwnerByIndex(owner common.Address, index *big.Int) (*big.Int, error) {
	return _Collectibles.Contract.TokenOfOwnerByIndex(&_Collectibles.CallOpts, owner, index)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_Collectibles *CollectiblesCaller) TokenURI(opts *bind.CallOpts, tokenId *big.Int) (string, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "tokenURI", tokenId)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_Collectibles *CollectiblesSession) TokenURI(tokenId *big.Int) (string, error) {
	return _Collectibles.Contract.TokenURI(&_Collectibles.CallOpts, tokenId)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_Collectibles *CollectiblesCallerSession) TokenURI(tokenId *big.Int) (string, error) {
	return _Collectibles.Contract.TokenURI(&_Collectibles.CallOpts, tokenId)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Collectibles *CollectiblesCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Collectibles *CollectiblesSession) TotalSupply() (*big.Int, error) {
	return _Collectibles.Contract.TotalSupply(&_Collectibles.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Collectibles *CollectiblesCallerSession) TotalSupply() (*big.Int, error) {
	return _Collectibles.Contract.TotalSupply(&_Collectibles.CallOpts)
}

// Transferable is a free data retrieval call binding the contract method 0x92ff0d31.
//
// Solidity: function transferable() view returns(bool)
func (_Collectibles *CollectiblesCaller) Transferable(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Collectibles.contract.Call(opts, &out, "transferable")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Transferable is a free data retrieval call binding the contract method 0x92ff0d31.
//
// Solidity: function transferable() view returns(bool)
func (_Collectibles *CollectiblesSession) Transferable() (bool, error) {
	return _Collectibles.Contract.Transferable(&_Collectibles.CallOpts)
}

// Transferable is a free data retrieval call binding the contract method 0x92ff0d31.
//
// Solidity: function transferable() view returns(bool)
func (_Collectibles *CollectiblesCallerSession) Transferable() (bool, error) {
	return _Collectibles.Contract.Transferable(&_Collectibles.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesTransactor) Approve(opts *bind.TransactOpts, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.contract.Transact(opts, "approve", to, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesSession) Approve(to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.Approve(&_Collectibles.TransactOpts, to, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesTransactorSession) Approve(to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.Approve(&_Collectibles.TransactOpts, to, tokenId)
}

// MintTo is a paid mutator transaction binding the contract method 0xce7c8b49.
//
// Solidity: function mintTo(address[] addresses) returns()
func (_Collectibles *CollectiblesTransactor) MintTo(opts *bind.TransactOpts, addresses []common.Address) (*types.Transaction, error) {
	return _Collectibles.contract.Transact(opts, "mintTo", addresses)
}

// MintTo is a paid mutator transaction binding the contract method 0xce7c8b49.
//
// Solidity: function mintTo(address[] addresses) returns()
func (_Collectibles *CollectiblesSession) MintTo(addresses []common.Address) (*types.Transaction, error) {
	return _Collectibles.Contract.MintTo(&_Collectibles.TransactOpts, addresses)
}

// MintTo is a paid mutator transaction binding the contract method 0xce7c8b49.
//
// Solidity: function mintTo(address[] addresses) returns()
func (_Collectibles *CollectiblesTransactorSession) MintTo(addresses []common.Address) (*types.Transaction, error) {
	return _Collectibles.Contract.MintTo(&_Collectibles.TransactOpts, addresses)
}

// RemoteBurn is a paid mutator transaction binding the contract method 0x4fb95e02.
//
// Solidity: function remoteBurn(uint256[] tokenIds) returns()
func (_Collectibles *CollectiblesTransactor) RemoteBurn(opts *bind.TransactOpts, tokenIds []*big.Int) (*types.Transaction, error) {
	return _Collectibles.contract.Transact(opts, "remoteBurn", tokenIds)
}

// RemoteBurn is a paid mutator transaction binding the contract method 0x4fb95e02.
//
// Solidity: function remoteBurn(uint256[] tokenIds) returns()
func (_Collectibles *CollectiblesSession) RemoteBurn(tokenIds []*big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.RemoteBurn(&_Collectibles.TransactOpts, tokenIds)
}

// RemoteBurn is a paid mutator transaction binding the contract method 0x4fb95e02.
//
// Solidity: function remoteBurn(uint256[] tokenIds) returns()
func (_Collectibles *CollectiblesTransactorSession) RemoteBurn(tokenIds []*big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.RemoteBurn(&_Collectibles.TransactOpts, tokenIds)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesTransactor) SafeTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.contract.Transact(opts, "safeTransferFrom", from, to, tokenId)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesSession) SafeTransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.SafeTransferFrom(&_Collectibles.TransactOpts, from, to, tokenId)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesTransactorSession) SafeTransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.SafeTransferFrom(&_Collectibles.TransactOpts, from, to, tokenId)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_Collectibles *CollectiblesTransactor) SafeTransferFrom0(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Collectibles.contract.Transact(opts, "safeTransferFrom0", from, to, tokenId, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_Collectibles *CollectiblesSession) SafeTransferFrom0(from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Collectibles.Contract.SafeTransferFrom0(&_Collectibles.TransactOpts, from, to, tokenId, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_Collectibles *CollectiblesTransactorSession) SafeTransferFrom0(from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Collectibles.Contract.SafeTransferFrom0(&_Collectibles.TransactOpts, from, to, tokenId, data)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Collectibles *CollectiblesTransactor) SetApprovalForAll(opts *bind.TransactOpts, operator common.Address, approved bool) (*types.Transaction, error) {
	return _Collectibles.contract.Transact(opts, "setApprovalForAll", operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Collectibles *CollectiblesSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _Collectibles.Contract.SetApprovalForAll(&_Collectibles.TransactOpts, operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Collectibles *CollectiblesTransactorSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _Collectibles.Contract.SetApprovalForAll(&_Collectibles.TransactOpts, operator, approved)
}

// SetMaxSupply is a paid mutator transaction binding the contract method 0x6f8b44b0.
//
// Solidity: function setMaxSupply(uint256 newMaxSupply) returns()
func (_Collectibles *CollectiblesTransactor) SetMaxSupply(opts *bind.TransactOpts, newMaxSupply *big.Int) (*types.Transaction, error) {
	return _Collectibles.contract.Transact(opts, "setMaxSupply", newMaxSupply)
}

// SetMaxSupply is a paid mutator transaction binding the contract method 0x6f8b44b0.
//
// Solidity: function setMaxSupply(uint256 newMaxSupply) returns()
func (_Collectibles *CollectiblesSession) SetMaxSupply(newMaxSupply *big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.SetMaxSupply(&_Collectibles.TransactOpts, newMaxSupply)
}

// SetMaxSupply is a paid mutator transaction binding the contract method 0x6f8b44b0.
//
// Solidity: function setMaxSupply(uint256 newMaxSupply) returns()
func (_Collectibles *CollectiblesTransactorSession) SetMaxSupply(newMaxSupply *big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.SetMaxSupply(&_Collectibles.TransactOpts, newMaxSupply)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.contract.Transact(opts, "transferFrom", from, to, tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesSession) TransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.TransferFrom(&_Collectibles.TransactOpts, from, to, tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_Collectibles *CollectiblesTransactorSession) TransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _Collectibles.Contract.TransferFrom(&_Collectibles.TransactOpts, from, to, tokenId)
}

// CollectiblesApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the Collectibles contract.
type CollectiblesApprovalIterator struct {
	Event *CollectiblesApproval // Event containing the contract specifics and raw log

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
func (it *CollectiblesApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CollectiblesApproval)
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
		it.Event = new(CollectiblesApproval)
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
func (it *CollectiblesApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CollectiblesApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CollectiblesApproval represents a Approval event raised by the Collectibles contract.
type CollectiblesApproval struct {
	Owner    common.Address
	Approved common.Address
	TokenId  *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_Collectibles *CollectiblesFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, approved []common.Address, tokenId []*big.Int) (*CollectiblesApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _Collectibles.contract.FilterLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &CollectiblesApprovalIterator{contract: _Collectibles.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_Collectibles *CollectiblesFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *CollectiblesApproval, owner []common.Address, approved []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _Collectibles.contract.WatchLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CollectiblesApproval)
				if err := _Collectibles.contract.UnpackLog(event, "Approval", log); err != nil {
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
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_Collectibles *CollectiblesFilterer) ParseApproval(log types.Log) (*CollectiblesApproval, error) {
	event := new(CollectiblesApproval)
	if err := _Collectibles.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CollectiblesApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the Collectibles contract.
type CollectiblesApprovalForAllIterator struct {
	Event *CollectiblesApprovalForAll // Event containing the contract specifics and raw log

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
func (it *CollectiblesApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CollectiblesApprovalForAll)
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
		it.Event = new(CollectiblesApprovalForAll)
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
func (it *CollectiblesApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CollectiblesApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CollectiblesApprovalForAll represents a ApprovalForAll event raised by the Collectibles contract.
type CollectiblesApprovalForAll struct {
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Collectibles *CollectiblesFilterer) FilterApprovalForAll(opts *bind.FilterOpts, owner []common.Address, operator []common.Address) (*CollectiblesApprovalForAllIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Collectibles.contract.FilterLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &CollectiblesApprovalForAllIterator{contract: _Collectibles.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Collectibles *CollectiblesFilterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *CollectiblesApprovalForAll, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Collectibles.contract.WatchLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CollectiblesApprovalForAll)
				if err := _Collectibles.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
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
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Collectibles *CollectiblesFilterer) ParseApprovalForAll(log types.Log) (*CollectiblesApprovalForAll, error) {
	event := new(CollectiblesApprovalForAll)
	if err := _Collectibles.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CollectiblesTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the Collectibles contract.
type CollectiblesTransferIterator struct {
	Event *CollectiblesTransfer // Event containing the contract specifics and raw log

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
func (it *CollectiblesTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CollectiblesTransfer)
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
		it.Event = new(CollectiblesTransfer)
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
func (it *CollectiblesTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CollectiblesTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CollectiblesTransfer represents a Transfer event raised by the Collectibles contract.
type CollectiblesTransfer struct {
	From    common.Address
	To      common.Address
	TokenId *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_Collectibles *CollectiblesFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address, tokenId []*big.Int) (*CollectiblesTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _Collectibles.contract.FilterLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &CollectiblesTransferIterator{contract: _Collectibles.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_Collectibles *CollectiblesFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *CollectiblesTransfer, from []common.Address, to []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _Collectibles.contract.WatchLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CollectiblesTransfer)
				if err := _Collectibles.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_Collectibles *CollectiblesFilterer) ParseTransfer(log types.Log) (*CollectiblesTransfer, error) {
	event := new(CollectiblesTransfer)
	if err := _Collectibles.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
