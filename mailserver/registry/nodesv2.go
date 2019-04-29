// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registry

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
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// NodesV2ABI is the input ABI used to generate the binding from.
const NodesV2ABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"getCurrentSession\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"publicKey\",\"type\":\"bytes\"}],\"name\":\"publicKeyToAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getNode\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"},{\"name\":\"\",\"type\":\"uint32\"},{\"name\":\"\",\"type\":\"uint16\"},{\"name\":\"\",\"type\":\"uint32\"},{\"name\":\"\",\"type\":\"uint32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"publicKey\",\"type\":\"bytes\"}],\"name\":\"registered\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"publicKey\",\"type\":\"bytes\"},{\"name\":\"ip\",\"type\":\"uint32\"},{\"name\":\"port\",\"type\":\"uint16\"}],\"name\":\"registerNode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"inactiveNodeCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"inactiveNodes\",\"outputs\":[{\"name\":\"publicKey\",\"type\":\"bytes\"},{\"name\":\"ip\",\"type\":\"uint32\"},{\"name\":\"port\",\"type\":\"uint16\"},{\"name\":\"joinVotes\",\"type\":\"uint8\"},{\"name\":\"removeVotes\",\"type\":\"uint8\"},{\"name\":\"lastTimeHasVoted\",\"type\":\"uint256\"},{\"name\":\"lastTimeHasBeenVoted\",\"type\":\"uint256\"},{\"name\":\"joiningSession\",\"type\":\"uint32\"},{\"name\":\"activeSession\",\"type\":\"uint32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"activeNodeCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getInactiveNode\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"},{\"name\":\"\",\"type\":\"uint32\"},{\"name\":\"\",\"type\":\"uint16\"},{\"name\":\"\",\"type\":\"uint32\"},{\"name\":\"\",\"type\":\"uint32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"activeNodes\",\"outputs\":[{\"name\":\"publicKey\",\"type\":\"bytes\"},{\"name\":\"ip\",\"type\":\"uint32\"},{\"name\":\"port\",\"type\":\"uint16\"},{\"name\":\"joinVotes\",\"type\":\"uint8\"},{\"name\":\"removeVotes\",\"type\":\"uint8\"},{\"name\":\"lastTimeHasVoted\",\"type\":\"uint256\"},{\"name\":\"lastTimeHasBeenVoted\",\"type\":\"uint256\"},{\"name\":\"joiningSession\",\"type\":\"uint32\"},{\"name\":\"activeSession\",\"type\":\"uint32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"joinNodes\",\"type\":\"address[]\"},{\"name\":\"removeNodes\",\"type\":\"address[]\"}],\"name\":\"vote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"currentSession\",\"outputs\":[{\"name\":\"\",\"type\":\"uint32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"publicKey\",\"type\":\"bytes\"},{\"name\":\"ip\",\"type\":\"uint32\"},{\"name\":\"port\",\"type\":\"uint16\"}],\"name\":\"addActiveNode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_blockPerSession\",\"type\":\"uint16\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"}]"

// NodesV2Bin is the compiled bytecode used for deploying new contracts.
const NodesV2Bin = `0x60806040526003805463ffff00001916631770000017905534801561002357600080fd5b50604051602080611f278339810180604052602081101561004357600080fd5b5051600080546001600160a01b031916331790556005805463ffffffff19169055436004556003805461ffff9092166401000000000265ffff0000000019909216919091179055611e8e806100996000396000f3fe6080604052600436106100c25760003560e01c806372460fa81161007f57806396f9d9831161005957806396f9d98314610562578063a19e39e81461058c578063d4166763146106bc578063dad7bcee146106ea576100c2565b806372460fa81461042d578063753408151461052357806393696e1a14610538576100c2565b80631401795f146100cf57806343ae656c146100f65780634f0f4aa9146101c35780635aca952e1461029357806363cd6e18146103585780636d1c76c214610418575b36156100cd57600080fd5b005b3480156100db57600080fd5b506100e46107aa565b60408051918252519081900360200190f35b34801561010257600080fd5b506101a76004803603602081101561011957600080fd5b810190602081018135600160201b81111561013357600080fd5b82018360208201111561014557600080fd5b803590602001918460018302840111600160201b8311171561016657600080fd5b91908080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152509295506107dc945050505050565b604080516001600160a01b039092168252519081900360200190f35b3480156101cf57600080fd5b506101ed600480360360208110156101e657600080fd5b50356107ee565b6040805163ffffffff80871660208084019190915261ffff87169383019390935284811660608301528316608082015260a080825287519082015286519091829160c083019189019080838360005b8381101561025457818101518382015260200161023c565b50505050905090810190601f1680156102815780820380516001836020036101000a031916815260200191505b50965050505050505060405180910390f35b34801561029f57600080fd5b50610344600480360360208110156102b657600080fd5b810190602081018135600160201b8111156102d057600080fd5b8201836020820111156102e257600080fd5b803590602001918460018302840111600160201b8311171561030357600080fd5b91908080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525092955061095e945050505050565b604080519115158252519081900360200190f35b34801561036457600080fd5b506100cd6004803603606081101561037b57600080fd5b810190602081018135600160201b81111561039557600080fd5b8201836020820111156103a757600080fd5b803590602001918460018302840111600160201b831117156103c857600080fd5b91908080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152509295505050813563ffffffff169250506020013561ffff166109b1565b34801561042457600080fd5b506100e4610bbc565b34801561043957600080fd5b506104576004803603602081101561045057600080fd5b5035610bc2565b6040805163ffffffff808b1660208084019190915261ffff8b169383019390935260ff808a1660608401528816608083015260a0820187905260c0820186905284811660e083015283166101008201526101208082528b51908201528a51909182916101408301918d019080838360005b838110156104e05781810151838201526020016104c8565b50505050905090810190601f16801561050d5780820380516001836020036101000a031916815260200191505b509a505050505050505050505060405180910390f35b34801561052f57600080fd5b506100e4610cbe565b34801561054457600080fd5b506101ed6004803603602081101561055b57600080fd5b5035610cc4565b34801561056e57600080fd5b506104576004803603602081101561058557600080fd5b5035610ce1565b34801561059857600080fd5b506100cd600480360360408110156105af57600080fd5b810190602081018135600160201b8111156105c957600080fd5b8201836020820111156105db57600080fd5b803590602001918460208302840111600160201b831117156105fc57600080fd5b9190808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152509295949360208101935035915050600160201b81111561064b57600080fd5b82018360208201111561065d57600080fd5b803590602001918460208302840111600160201b8311171561067e57600080fd5b919080806020026020016040519081016040528093929190818152602001838360200280828437600092019190915250929550610cee945050505050565b3480156106c857600080fd5b506106d1611249565b6040805163ffffffff9092168252519081900360200190f35b3480156106f657600080fd5b506100cd6004803603606081101561070d57600080fd5b810190602081018135600160201b81111561072757600080fd5b82018360208201111561073957600080fd5b803590602001918460018302840111600160201b8311171561075a57600080fd5b91908080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152509295505050813563ffffffff169250506020013561ffff16611255565b60006107b4611371565b156107ce575060055463ffffffff908116600101166107d9565b5060055463ffffffff165b90565b8051602090910120606090811b901c90565b60606000806000806107fe611cc2565b6001878154811061080b57fe5b600091825260209182902060408051600593909302909101805460026001821615610100026000190190911604601f810185900490940283016101409081019092526101208301848152929390928492909184918401828280156108b05780601f10610885576101008083540402835291602001916108b0565b820191906000526020600020905b81548152906001019060200180831161089357829003601f168201915b5050509183525050600182015463ffffffff808216602080850191909152600160201b80840461ffff16604080870191909152600160301b850460ff9081166060880152600160381b9095049094166080860152600286015460a0860152600386015460c086015260049095015480831660e0808701919091529590049091166101009384015284519085015191850151938501519490920151919b909a5091985091965090945092505050565b60008061096a836107dc565b6001600160a01b0381166000908152600660205260409020549091501580156109a957506001600160a01b038116600090815260076020526040902054155b159392505050565b6109ba836107dc565b6001600160a01b0316336001600160a01b0316146109d757600080fd5b33600090815260066020526040902054158015610a01575033600090815260076020526040902054155b610a0a57600080fd5b6003546002546001546201000090920461ffff16910110610a2a57600080fd5b610a32611cc2565b50604080516101208101825284815263ffffffff80851660208084019190915261ffff8516938301939093526000606083018190526080830181905260a0830181905260c083018190526005805490921660e0840152610100830181905260028054600181018083559190925283518051949591948694939093027f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace0192610add9284920190611d0e565b506020828101516001830180546040808701516060880151608089015163ffffffff1994851663ffffffff9788161765ffff000000001916600160201b61ffff90941684021766ff0000000000001916600160301b60ff938416021767ff000000000000001916600160381b92909116919091021790935560a087015160028088019190915560c0880151600388015560e08801516004909701805461010090990151989093169685169690961767ffffffff000000001916969093169091029490941790935590543360009081526007909252919020555050505050565b60025490565b60028181548110610bcf57fe5b60009182526020918290206005919091020180546040805160026001841615610100026000190190931692909204601f810185900485028301850190915280825291935091839190830182828015610c685780601f10610c3d57610100808354040283529160200191610c68565b820191906000526020600020905b815481529060010190602001808311610c4b57829003601f168201915b50505050600183015460028401546003850154600490950154939463ffffffff80841695600160201b80860461ffff169650600160301b860460ff90811696600160381b90041694939280831692919091041689565b60015490565b6060600080600080610cd4611cc2565b6002878154811061080b57fe5b60018181548110610bcf57fe5b610cf6611371565b15610d0357610d0361138a565b3360009081526006602052604090205480610d1d57600080fd5b6005546001805463ffffffff909216916000198401908110610d3b57fe5b9060005260206000209060050201600201541415610d5857600080fd5b600180820381548110610d6757fe5b6000918252602090912060059091020160040154600160201b900463ffffffff16158015610dd257506005546001805463ffffffff909216916000198401908110610dae57fe5b6000918252602090912060059091020160040154600160201b900463ffffffff1614155b610ddb57600080fd5b6005546001805463ffffffff909216916000198401908110610df957fe5b600091825260208220600260059092020101919091555b8251811015610f3257600060066000858481518110610e2b57fe5b60200260200101516001600160a01b03166001600160a01b0316815260200190815260200160002054905080600014610f29576000600180830381548110610e6f57fe5b90600052602060002090600502019050600560009054906101000a900463ffffffff1663ffffffff16816003015414610ece5760055463ffffffff16600382015560018101805467ff000000000000001916600160381b179055610efc565b6001808201805460ff600160381b80830482169094011690920267ff00000000000000199092169190911790555b600354600182015460ff600160381b9091041661ffff9091161415610f2757610f27600183036113cf565b505b50600101610e10565b5060005b8351811015611243576000848281518110610f4d57fe5b6020908102919091018101516001600160a01b03811660009081526007909252604090912054909150801561123957600060026001830381548110610f8e57fe5b6000918252602091829020600590910201805460408051601f600260001961010060018716150201909416939093049283018590048502810185019091528181529193506110349284919083018282801561102a5780601f10610fff5761010080835404028352916020019161102a565b820191906000526020600020905b81548152906001019060200180831161100d57829003601f168201915b50505050506107dc565b6001600160a01b0316836001600160a01b03161461105157600080fd5b600554600382015463ffffffff909116146110915760055463ffffffff16600382015560018101805466ff0000000000001916600160301b1790556110be565b6001808201805460ff600160301b80830482169094011690920266ff000000000000199092169190911790555b600354600182015460ff600160301b9091041661ffff9091161415611237576110e5611cc2565b600260018403815481106110f557fe5b600091825260209182902060408051600593909302909101805460026001821615610100026000190190911604601f8101859004909402830161014090810190925261012083018481529293909284929091849184018282801561119a5780601f1061116f5761010080835404028352916020019161119a565b820191906000526020600020905b81548152906001019060200180831161117d57829003601f168201915b5050509183525050600182015463ffffffff8082166020840152600160201b80830461ffff166040850152600160301b830460ff9081166060860152600160381b9093049092166080840152600284015460a0840152600384015460c084015260049093015480841660e08401520482166101009182015260055490911690820152905061122b60001984016117b6565b6112358482611b80565b505b505b5050600101610f36565b50505050565b60055463ffffffff1681565b6000546001600160a01b0316331461126c57600080fd5b6000611277846107dc565b6001600160a01b0381166000908152600660205260409020549091501580156112b657506001600160a01b038116600090815260076020526040902054155b6112bf57600080fd5b6003546002546001546201000090920461ffff169101106112df57600080fd5b6112e7611cc2565b50604080516101208101825285815263ffffffff808616602083015261ffff8516928201929092526000606082018190526080820181905260a0820181905260c082015260055490911660e082018190526101008201526113488282611b80565b60015461135490611cae565b6003805461ffff191661ffff929092169190911790555050505050565b600354600454600160201b90910461ffff160143101590565b60015461139690611cae565b6003805461ffff191661ffff929092169190911790556005805463ffffffff19811663ffffffff91821660010190911617905543600455565b60015481106113dd57600080fd5b6113e5611cc2565b600182815481106113f257fe5b600091825260209182902060408051600593909302909101805460026001821615610100026000190190911604601f810185900490940283016101409081019092526101208301848152929390928492909184918401828280156114975780601f1061146c57610100808354040283529160200191611497565b820191906000526020600020905b81548152906001019060200180831161147a57829003601f168201915b5050509183525050600182015463ffffffff8082166020840152600160201b80830461ffff166040850152600160301b830460ff9081166060860152600160381b9093049092166080840152600284015460a0840152600384015460c084015260049093015480841660e0840152049091166101009091015280519091506000906006908290611526906107dc565b6001600160a01b0316815260208101919091526040016000205560018054600019810190811061155257fe5b90600052602060002090600502016001838154811061156d57fe5b9060005260206000209060050201600082018160000190805460018160011615610100020316600290046115a2929190611d8c565b5060018281018054838301805463ffffffff92831663ffffffff1991821617808355845465ffff0000000019909116600160201b9182900461ffff16820217808455855466ff00000000000019909116600160301b9182900460ff90811690920217808555955467ff0000000000000019909616600160381b96879004909116909502949094179091556002808701549086015560038087015490860155600495860180549690950180549683169690911695909517808655935467ffffffff00000000199094169382900416029190911790915580548061168057fe5b600082815260208120600019909201916005830201906116a08282611e01565b506001818101805467ffffffffffffffff1990811690915560006002840181905560038401556004909201805490921690915591555482146117b2576060600183815481106116eb57fe5b6000918252602091829020600590910201805460408051601f600260001961010060018716150201909416939093049283018590048502810185019091528181529283018282801561177e5780601f106117535761010080835404028352916020019161177e565b820191906000526020600020905b81548152906001019060200180831161176157829003601f168201915b505050505090508260010160066000611796846107dc565b6001600160a01b03168152602081019190915260400160002055505b5050565b60025481106117c457600080fd5b6117cc611cc2565b600282815481106117d957fe5b600091825260209182902060408051600593909302909101805460026001821615610100026000190190911604601f8101859004909402830161014090810190925261012083018481529293909284929091849184018282801561187e5780601f106118535761010080835404028352916020019161187e565b820191906000526020600020905b81548152906001019060200180831161186157829003601f168201915b5050509183525050600182015463ffffffff8082166020840152600160201b80830461ffff166040850152600160301b830460ff9081166060860152600160381b9093049092166080840152600284015460a0840152600384015460c084015260049093015480841660e084015204909116610100909101528051909150600090600790829061190d906107dc565b6001600160a01b0316815260208101919091526040016000205560028054600019810190811061193957fe5b90600052602060002090600502016002838154811061195457fe5b906000526020600020906005020160008201816000019080546001816001161561010002031660029004611989929190611d8c565b5060018281018054918301805463ffffffff93841663ffffffff1991821617808355835465ffff0000000019909116600160201b9182900461ffff16820217808455845466ff00000000000019909116600160301b9182900460ff90811690920217808555945467ff0000000000000019909516600160381b95869004909116909402939093179091556002808601548186015560038087015490860155600495860180549690950180549685169690921695909517808255935467ffffffff00000000199094169382900490921602919091179055805480611a6857fe5b60008281526020812060001990920191600583020190611a888282611e01565b5060018101805467ffffffffffffffff1990811690915560006002808401829055600384019190915560049092018054909116905591555482146117b257606060028381548110611ad557fe5b6000918252602091829020600590910201805460408051601f6002600019610100600187161502019094169390930492830185900485028101850190915281815292830182828015611b685780601f10611b3d57610100808354040283529160200191611b68565b820191906000526020600020905b815481529060010190602001808311611b4b57829003601f168201915b505050505090508260010160076000611796846107dc565b600180548082018083556000929092528251805184926005027fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf60191611bcb91839160200190611d0e565b5060208281015160018381018054604080880151606089015160808a015163ffffffff1994851663ffffffff9889161765ffff000000001916600160201b61ffff90941684021766ff0000000000001916600160301b60ff938416021767ff000000000000001916600160381b92909116919091021790935560a0880151600288015560c0880151600388015560e08801516004909701805461010090990151989092169685169690961767ffffffff000000001916969093160294909417905591546001600160a01b0395909516600090815260069092529020929092555050565b6000600261ffff8316046001019050919050565b6040805161012081018252606080825260006020830181905292820183905281018290526080810182905260a0810182905260c0810182905260e0810182905261010081019190915290565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10611d4f57805160ff1916838001178555611d7c565b82800160010185558215611d7c579182015b82811115611d7c578251825591602001919060010190611d61565b50611d88929150611e48565b5090565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10611dc55780548555611d7c565b82800160010185558215611d7c57600052602060002091601f016020900482015b82811115611d7c578254825591600101919060010190611de6565b50805460018160011615610100020316600290046000825580601f10611e275750611e45565b601f016020900490600052602060002090810190611e459190611e48565b50565b6107d991905b80821115611d885760008155600101611e4e56fea165627a7a72305820d86f2f26f79475dfc2993934f6e720204c7d9b84299fa64ad2b339cabd00e94a0029`

// DeployNodesV2 deploys a new Ethereum contract, binding an instance of NodesV2 to it.
func DeployNodesV2(auth *bind.TransactOpts, backend bind.ContractBackend, _blockPerSession uint16) (common.Address, *types.Transaction, *NodesV2, error) {
	parsed, err := abi.JSON(strings.NewReader(NodesV2ABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(NodesV2Bin), backend, _blockPerSession)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &NodesV2{NodesV2Caller: NodesV2Caller{contract: contract}, NodesV2Transactor: NodesV2Transactor{contract: contract}, NodesV2Filterer: NodesV2Filterer{contract: contract}}, nil
}

// NodesV2 is an auto generated Go binding around an Ethereum contract.
type NodesV2 struct {
	NodesV2Caller     // Read-only binding to the contract
	NodesV2Transactor // Write-only binding to the contract
	NodesV2Filterer   // Log filterer for contract events
}

// NodesV2Caller is an auto generated read-only Go binding around an Ethereum contract.
type NodesV2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodesV2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type NodesV2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodesV2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NodesV2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodesV2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NodesV2Session struct {
	Contract     *NodesV2          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NodesV2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NodesV2CallerSession struct {
	Contract *NodesV2Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// NodesV2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NodesV2TransactorSession struct {
	Contract     *NodesV2Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// NodesV2Raw is an auto generated low-level Go binding around an Ethereum contract.
type NodesV2Raw struct {
	Contract *NodesV2 // Generic contract binding to access the raw methods on
}

// NodesV2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NodesV2CallerRaw struct {
	Contract *NodesV2Caller // Generic read-only contract binding to access the raw methods on
}

// NodesV2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NodesV2TransactorRaw struct {
	Contract *NodesV2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewNodesV2 creates a new instance of NodesV2, bound to a specific deployed contract.
func NewNodesV2(address common.Address, backend bind.ContractBackend) (*NodesV2, error) {
	contract, err := bindNodesV2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NodesV2{NodesV2Caller: NodesV2Caller{contract: contract}, NodesV2Transactor: NodesV2Transactor{contract: contract}, NodesV2Filterer: NodesV2Filterer{contract: contract}}, nil
}

// NewNodesV2Caller creates a new read-only instance of NodesV2, bound to a specific deployed contract.
func NewNodesV2Caller(address common.Address, caller bind.ContractCaller) (*NodesV2Caller, error) {
	contract, err := bindNodesV2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NodesV2Caller{contract: contract}, nil
}

// NewNodesV2Transactor creates a new write-only instance of NodesV2, bound to a specific deployed contract.
func NewNodesV2Transactor(address common.Address, transactor bind.ContractTransactor) (*NodesV2Transactor, error) {
	contract, err := bindNodesV2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NodesV2Transactor{contract: contract}, nil
}

// NewNodesV2Filterer creates a new log filterer instance of NodesV2, bound to a specific deployed contract.
func NewNodesV2Filterer(address common.Address, filterer bind.ContractFilterer) (*NodesV2Filterer, error) {
	contract, err := bindNodesV2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NodesV2Filterer{contract: contract}, nil
}

// bindNodesV2 binds a generic wrapper to an already deployed contract.
func bindNodesV2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(NodesV2ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NodesV2 *NodesV2Raw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _NodesV2.Contract.NodesV2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NodesV2 *NodesV2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NodesV2.Contract.NodesV2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NodesV2 *NodesV2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NodesV2.Contract.NodesV2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NodesV2 *NodesV2CallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _NodesV2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NodesV2 *NodesV2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NodesV2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NodesV2 *NodesV2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NodesV2.Contract.contract.Transact(opts, method, params...)
}

// ActiveNodeCount is a free data retrieval call binding the contract method 0x75340815.
//
// Solidity: function activeNodeCount() constant returns(uint256)
func (_NodesV2 *NodesV2Caller) ActiveNodeCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _NodesV2.contract.Call(opts, out, "activeNodeCount")
	return *ret0, err
}

// ActiveNodeCount is a free data retrieval call binding the contract method 0x75340815.
//
// Solidity: function activeNodeCount() constant returns(uint256)
func (_NodesV2 *NodesV2Session) ActiveNodeCount() (*big.Int, error) {
	return _NodesV2.Contract.ActiveNodeCount(&_NodesV2.CallOpts)
}

// ActiveNodeCount is a free data retrieval call binding the contract method 0x75340815.
//
// Solidity: function activeNodeCount() constant returns(uint256)
func (_NodesV2 *NodesV2CallerSession) ActiveNodeCount() (*big.Int, error) {
	return _NodesV2.Contract.ActiveNodeCount(&_NodesV2.CallOpts)
}

// ActiveNodes is a free data retrieval call binding the contract method 0x96f9d983.
//
// Solidity: function activeNodes(uint256 ) constant returns(bytes publicKey, uint32 ip, uint16 port, uint8 joinVotes, uint8 removeVotes, uint256 lastTimeHasVoted, uint256 lastTimeHasBeenVoted, uint32 joiningSession, uint32 activeSession)
func (_NodesV2 *NodesV2Caller) ActiveNodes(opts *bind.CallOpts, arg0 *big.Int) (struct {
	PublicKey            []byte
	Ip                   uint32
	Port                 uint16
	JoinVotes            uint8
	RemoveVotes          uint8
	LastTimeHasVoted     *big.Int
	LastTimeHasBeenVoted *big.Int
	JoiningSession       uint32
	ActiveSession        uint32
}, error) {
	ret := new(struct {
		PublicKey            []byte
		Ip                   uint32
		Port                 uint16
		JoinVotes            uint8
		RemoveVotes          uint8
		LastTimeHasVoted     *big.Int
		LastTimeHasBeenVoted *big.Int
		JoiningSession       uint32
		ActiveSession        uint32
	})
	out := ret
	err := _NodesV2.contract.Call(opts, out, "activeNodes", arg0)
	return *ret, err
}

// ActiveNodes is a free data retrieval call binding the contract method 0x96f9d983.
//
// Solidity: function activeNodes(uint256 ) constant returns(bytes publicKey, uint32 ip, uint16 port, uint8 joinVotes, uint8 removeVotes, uint256 lastTimeHasVoted, uint256 lastTimeHasBeenVoted, uint32 joiningSession, uint32 activeSession)
func (_NodesV2 *NodesV2Session) ActiveNodes(arg0 *big.Int) (struct {
	PublicKey            []byte
	Ip                   uint32
	Port                 uint16
	JoinVotes            uint8
	RemoveVotes          uint8
	LastTimeHasVoted     *big.Int
	LastTimeHasBeenVoted *big.Int
	JoiningSession       uint32
	ActiveSession        uint32
}, error) {
	return _NodesV2.Contract.ActiveNodes(&_NodesV2.CallOpts, arg0)
}

// ActiveNodes is a free data retrieval call binding the contract method 0x96f9d983.
//
// Solidity: function activeNodes(uint256 ) constant returns(bytes publicKey, uint32 ip, uint16 port, uint8 joinVotes, uint8 removeVotes, uint256 lastTimeHasVoted, uint256 lastTimeHasBeenVoted, uint32 joiningSession, uint32 activeSession)
func (_NodesV2 *NodesV2CallerSession) ActiveNodes(arg0 *big.Int) (struct {
	PublicKey            []byte
	Ip                   uint32
	Port                 uint16
	JoinVotes            uint8
	RemoveVotes          uint8
	LastTimeHasVoted     *big.Int
	LastTimeHasBeenVoted *big.Int
	JoiningSession       uint32
	ActiveSession        uint32
}, error) {
	return _NodesV2.Contract.ActiveNodes(&_NodesV2.CallOpts, arg0)
}

// CurrentSession is a free data retrieval call binding the contract method 0xd4166763.
//
// Solidity: function currentSession() constant returns(uint32)
func (_NodesV2 *NodesV2Caller) CurrentSession(opts *bind.CallOpts) (uint32, error) {
	var (
		ret0 = new(uint32)
	)
	out := ret0
	err := _NodesV2.contract.Call(opts, out, "currentSession")
	return *ret0, err
}

// CurrentSession is a free data retrieval call binding the contract method 0xd4166763.
//
// Solidity: function currentSession() constant returns(uint32)
func (_NodesV2 *NodesV2Session) CurrentSession() (uint32, error) {
	return _NodesV2.Contract.CurrentSession(&_NodesV2.CallOpts)
}

// CurrentSession is a free data retrieval call binding the contract method 0xd4166763.
//
// Solidity: function currentSession() constant returns(uint32)
func (_NodesV2 *NodesV2CallerSession) CurrentSession() (uint32, error) {
	return _NodesV2.Contract.CurrentSession(&_NodesV2.CallOpts)
}

// GetCurrentSession is a free data retrieval call binding the contract method 0x1401795f.
//
// Solidity: function getCurrentSession() constant returns(uint256)
func (_NodesV2 *NodesV2Caller) GetCurrentSession(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _NodesV2.contract.Call(opts, out, "getCurrentSession")
	return *ret0, err
}

// GetCurrentSession is a free data retrieval call binding the contract method 0x1401795f.
//
// Solidity: function getCurrentSession() constant returns(uint256)
func (_NodesV2 *NodesV2Session) GetCurrentSession() (*big.Int, error) {
	return _NodesV2.Contract.GetCurrentSession(&_NodesV2.CallOpts)
}

// GetCurrentSession is a free data retrieval call binding the contract method 0x1401795f.
//
// Solidity: function getCurrentSession() constant returns(uint256)
func (_NodesV2 *NodesV2CallerSession) GetCurrentSession() (*big.Int, error) {
	return _NodesV2.Contract.GetCurrentSession(&_NodesV2.CallOpts)
}

// GetInactiveNode is a free data retrieval call binding the contract method 0x93696e1a.
//
// Solidity: function getInactiveNode(uint256 index) constant returns(bytes, uint32, uint16, uint32, uint32)
func (_NodesV2 *NodesV2Caller) GetInactiveNode(opts *bind.CallOpts, index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error) {
	var (
		ret0 = new([]byte)
		ret1 = new(uint32)
		ret2 = new(uint16)
		ret3 = new(uint32)
		ret4 = new(uint32)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
		ret3,
		ret4,
	}
	err := _NodesV2.contract.Call(opts, out, "getInactiveNode", index)
	return *ret0, *ret1, *ret2, *ret3, *ret4, err
}

// GetInactiveNode is a free data retrieval call binding the contract method 0x93696e1a.
//
// Solidity: function getInactiveNode(uint256 index) constant returns(bytes, uint32, uint16, uint32, uint32)
func (_NodesV2 *NodesV2Session) GetInactiveNode(index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error) {
	return _NodesV2.Contract.GetInactiveNode(&_NodesV2.CallOpts, index)
}

// GetInactiveNode is a free data retrieval call binding the contract method 0x93696e1a.
//
// Solidity: function getInactiveNode(uint256 index) constant returns(bytes, uint32, uint16, uint32, uint32)
func (_NodesV2 *NodesV2CallerSession) GetInactiveNode(index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error) {
	return _NodesV2.Contract.GetInactiveNode(&_NodesV2.CallOpts, index)
}

// GetNode is a free data retrieval call binding the contract method 0x4f0f4aa9.
//
// Solidity: function getNode(uint256 index) constant returns(bytes, uint32, uint16, uint32, uint32)
func (_NodesV2 *NodesV2Caller) GetNode(opts *bind.CallOpts, index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error) {
	var (
		ret0 = new([]byte)
		ret1 = new(uint32)
		ret2 = new(uint16)
		ret3 = new(uint32)
		ret4 = new(uint32)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
		ret3,
		ret4,
	}
	err := _NodesV2.contract.Call(opts, out, "getNode", index)
	return *ret0, *ret1, *ret2, *ret3, *ret4, err
}

// GetNode is a free data retrieval call binding the contract method 0x4f0f4aa9.
//
// Solidity: function getNode(uint256 index) constant returns(bytes, uint32, uint16, uint32, uint32)
func (_NodesV2 *NodesV2Session) GetNode(index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error) {
	return _NodesV2.Contract.GetNode(&_NodesV2.CallOpts, index)
}

// GetNode is a free data retrieval call binding the contract method 0x4f0f4aa9.
//
// Solidity: function getNode(uint256 index) constant returns(bytes, uint32, uint16, uint32, uint32)
func (_NodesV2 *NodesV2CallerSession) GetNode(index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error) {
	return _NodesV2.Contract.GetNode(&_NodesV2.CallOpts, index)
}

// InactiveNodeCount is a free data retrieval call binding the contract method 0x6d1c76c2.
//
// Solidity: function inactiveNodeCount() constant returns(uint256)
func (_NodesV2 *NodesV2Caller) InactiveNodeCount(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _NodesV2.contract.Call(opts, out, "inactiveNodeCount")
	return *ret0, err
}

// InactiveNodeCount is a free data retrieval call binding the contract method 0x6d1c76c2.
//
// Solidity: function inactiveNodeCount() constant returns(uint256)
func (_NodesV2 *NodesV2Session) InactiveNodeCount() (*big.Int, error) {
	return _NodesV2.Contract.InactiveNodeCount(&_NodesV2.CallOpts)
}

// InactiveNodeCount is a free data retrieval call binding the contract method 0x6d1c76c2.
//
// Solidity: function inactiveNodeCount() constant returns(uint256)
func (_NodesV2 *NodesV2CallerSession) InactiveNodeCount() (*big.Int, error) {
	return _NodesV2.Contract.InactiveNodeCount(&_NodesV2.CallOpts)
}

// InactiveNodes is a free data retrieval call binding the contract method 0x72460fa8.
//
// Solidity: function inactiveNodes(uint256 ) constant returns(bytes publicKey, uint32 ip, uint16 port, uint8 joinVotes, uint8 removeVotes, uint256 lastTimeHasVoted, uint256 lastTimeHasBeenVoted, uint32 joiningSession, uint32 activeSession)
func (_NodesV2 *NodesV2Caller) InactiveNodes(opts *bind.CallOpts, arg0 *big.Int) (struct {
	PublicKey            []byte
	Ip                   uint32
	Port                 uint16
	JoinVotes            uint8
	RemoveVotes          uint8
	LastTimeHasVoted     *big.Int
	LastTimeHasBeenVoted *big.Int
	JoiningSession       uint32
	ActiveSession        uint32
}, error) {
	ret := new(struct {
		PublicKey            []byte
		Ip                   uint32
		Port                 uint16
		JoinVotes            uint8
		RemoveVotes          uint8
		LastTimeHasVoted     *big.Int
		LastTimeHasBeenVoted *big.Int
		JoiningSession       uint32
		ActiveSession        uint32
	})
	out := ret
	err := _NodesV2.contract.Call(opts, out, "inactiveNodes", arg0)
	return *ret, err
}

// InactiveNodes is a free data retrieval call binding the contract method 0x72460fa8.
//
// Solidity: function inactiveNodes(uint256 ) constant returns(bytes publicKey, uint32 ip, uint16 port, uint8 joinVotes, uint8 removeVotes, uint256 lastTimeHasVoted, uint256 lastTimeHasBeenVoted, uint32 joiningSession, uint32 activeSession)
func (_NodesV2 *NodesV2Session) InactiveNodes(arg0 *big.Int) (struct {
	PublicKey            []byte
	Ip                   uint32
	Port                 uint16
	JoinVotes            uint8
	RemoveVotes          uint8
	LastTimeHasVoted     *big.Int
	LastTimeHasBeenVoted *big.Int
	JoiningSession       uint32
	ActiveSession        uint32
}, error) {
	return _NodesV2.Contract.InactiveNodes(&_NodesV2.CallOpts, arg0)
}

// InactiveNodes is a free data retrieval call binding the contract method 0x72460fa8.
//
// Solidity: function inactiveNodes(uint256 ) constant returns(bytes publicKey, uint32 ip, uint16 port, uint8 joinVotes, uint8 removeVotes, uint256 lastTimeHasVoted, uint256 lastTimeHasBeenVoted, uint32 joiningSession, uint32 activeSession)
func (_NodesV2 *NodesV2CallerSession) InactiveNodes(arg0 *big.Int) (struct {
	PublicKey            []byte
	Ip                   uint32
	Port                 uint16
	JoinVotes            uint8
	RemoveVotes          uint8
	LastTimeHasVoted     *big.Int
	LastTimeHasBeenVoted *big.Int
	JoiningSession       uint32
	ActiveSession        uint32
}, error) {
	return _NodesV2.Contract.InactiveNodes(&_NodesV2.CallOpts, arg0)
}

// PublicKeyToAddress is a free data retrieval call binding the contract method 0x43ae656c.
//
// Solidity: function publicKeyToAddress(bytes publicKey) constant returns(address)
func (_NodesV2 *NodesV2Caller) PublicKeyToAddress(opts *bind.CallOpts, publicKey []byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _NodesV2.contract.Call(opts, out, "publicKeyToAddress", publicKey)
	return *ret0, err
}

// PublicKeyToAddress is a free data retrieval call binding the contract method 0x43ae656c.
//
// Solidity: function publicKeyToAddress(bytes publicKey) constant returns(address)
func (_NodesV2 *NodesV2Session) PublicKeyToAddress(publicKey []byte) (common.Address, error) {
	return _NodesV2.Contract.PublicKeyToAddress(&_NodesV2.CallOpts, publicKey)
}

// PublicKeyToAddress is a free data retrieval call binding the contract method 0x43ae656c.
//
// Solidity: function publicKeyToAddress(bytes publicKey) constant returns(address)
func (_NodesV2 *NodesV2CallerSession) PublicKeyToAddress(publicKey []byte) (common.Address, error) {
	return _NodesV2.Contract.PublicKeyToAddress(&_NodesV2.CallOpts, publicKey)
}

// Registered is a free data retrieval call binding the contract method 0x5aca952e.
//
// Solidity: function registered(bytes publicKey) constant returns(bool)
func (_NodesV2 *NodesV2Caller) Registered(opts *bind.CallOpts, publicKey []byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _NodesV2.contract.Call(opts, out, "registered", publicKey)
	return *ret0, err
}

// Registered is a free data retrieval call binding the contract method 0x5aca952e.
//
// Solidity: function registered(bytes publicKey) constant returns(bool)
func (_NodesV2 *NodesV2Session) Registered(publicKey []byte) (bool, error) {
	return _NodesV2.Contract.Registered(&_NodesV2.CallOpts, publicKey)
}

// Registered is a free data retrieval call binding the contract method 0x5aca952e.
//
// Solidity: function registered(bytes publicKey) constant returns(bool)
func (_NodesV2 *NodesV2CallerSession) Registered(publicKey []byte) (bool, error) {
	return _NodesV2.Contract.Registered(&_NodesV2.CallOpts, publicKey)
}

// AddActiveNode is a paid mutator transaction binding the contract method 0xdad7bcee.
//
// Solidity: function addActiveNode(bytes publicKey, uint32 ip, uint16 port) returns()
func (_NodesV2 *NodesV2Transactor) AddActiveNode(opts *bind.TransactOpts, publicKey []byte, ip uint32, port uint16) (*types.Transaction, error) {
	return _NodesV2.contract.Transact(opts, "addActiveNode", publicKey, ip, port)
}

// AddActiveNode is a paid mutator transaction binding the contract method 0xdad7bcee.
//
// Solidity: function addActiveNode(bytes publicKey, uint32 ip, uint16 port) returns()
func (_NodesV2 *NodesV2Session) AddActiveNode(publicKey []byte, ip uint32, port uint16) (*types.Transaction, error) {
	return _NodesV2.Contract.AddActiveNode(&_NodesV2.TransactOpts, publicKey, ip, port)
}

// AddActiveNode is a paid mutator transaction binding the contract method 0xdad7bcee.
//
// Solidity: function addActiveNode(bytes publicKey, uint32 ip, uint16 port) returns()
func (_NodesV2 *NodesV2TransactorSession) AddActiveNode(publicKey []byte, ip uint32, port uint16) (*types.Transaction, error) {
	return _NodesV2.Contract.AddActiveNode(&_NodesV2.TransactOpts, publicKey, ip, port)
}

// RegisterNode is a paid mutator transaction binding the contract method 0x63cd6e18.
//
// Solidity: function registerNode(bytes publicKey, uint32 ip, uint16 port) returns()
func (_NodesV2 *NodesV2Transactor) RegisterNode(opts *bind.TransactOpts, publicKey []byte, ip uint32, port uint16) (*types.Transaction, error) {
	return _NodesV2.contract.Transact(opts, "registerNode", publicKey, ip, port)
}

// RegisterNode is a paid mutator transaction binding the contract method 0x63cd6e18.
//
// Solidity: function registerNode(bytes publicKey, uint32 ip, uint16 port) returns()
func (_NodesV2 *NodesV2Session) RegisterNode(publicKey []byte, ip uint32, port uint16) (*types.Transaction, error) {
	return _NodesV2.Contract.RegisterNode(&_NodesV2.TransactOpts, publicKey, ip, port)
}

// RegisterNode is a paid mutator transaction binding the contract method 0x63cd6e18.
//
// Solidity: function registerNode(bytes publicKey, uint32 ip, uint16 port) returns()
func (_NodesV2 *NodesV2TransactorSession) RegisterNode(publicKey []byte, ip uint32, port uint16) (*types.Transaction, error) {
	return _NodesV2.Contract.RegisterNode(&_NodesV2.TransactOpts, publicKey, ip, port)
}

// Vote is a paid mutator transaction binding the contract method 0xa19e39e8.
//
// Solidity: function vote(address[] joinNodes, address[] removeNodes) returns()
func (_NodesV2 *NodesV2Transactor) Vote(opts *bind.TransactOpts, joinNodes []common.Address, removeNodes []common.Address) (*types.Transaction, error) {
	return _NodesV2.contract.Transact(opts, "vote", joinNodes, removeNodes)
}

// Vote is a paid mutator transaction binding the contract method 0xa19e39e8.
//
// Solidity: function vote(address[] joinNodes, address[] removeNodes) returns()
func (_NodesV2 *NodesV2Session) Vote(joinNodes []common.Address, removeNodes []common.Address) (*types.Transaction, error) {
	return _NodesV2.Contract.Vote(&_NodesV2.TransactOpts, joinNodes, removeNodes)
}

// Vote is a paid mutator transaction binding the contract method 0xa19e39e8.
//
// Solidity: function vote(address[] joinNodes, address[] removeNodes) returns()
func (_NodesV2 *NodesV2TransactorSession) Vote(joinNodes []common.Address, removeNodes []common.Address) (*types.Transaction, error) {
	return _NodesV2.Contract.Vote(&_NodesV2.TransactOpts, joinNodes, removeNodes)
}
