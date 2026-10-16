package ensresolver

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/contracts"
	"github.com/status-im/status-go/internal/contracts/registrar"
	"github.com/status-im/status-go/internal/contracts/universalresolver"
	"github.com/status-im/status-go/internal/rpc"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

// ensRecordABI is the minimal subset of the ENS public-resolver interface used to
// encode/decode the record calls executed through the Universal Resolver.
const ensRecordABI = `[` +
	`{"name":"addr","inputs":[{"type":"bytes32"}],"outputs":[{"type":"address"}],"stateMutability":"view","type":"function"},` +
	`{"name":"pubkey","inputs":[{"type":"bytes32"}],"outputs":[{"name":"x","type":"bytes32"},{"name":"y","type":"bytes32"}],"stateMutability":"view","type":"function"},` +
	`{"name":"contenthash","inputs":[{"type":"bytes32"}],"outputs":[{"type":"bytes"}],"stateMutability":"view","type":"function"}` +
	`]`

var ensRecordMethods = mustParseABI(ensRecordABI)

// ensRegistryABI is the subset of the ENS registry used to find a name's resolver.
const ensRegistryABI = `[{"name":"resolver","inputs":[{"type":"bytes32"}],"outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}]`

var ensRegistryMethods = mustParseABI(ensRegistryABI)

// ensRegistryAddress is the ENS registry on Ethereum mainnet and its testnets.
// source: https://docs.ens.domains/learn/deployments
var ensRegistryAddress = common.HexToAddress("0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e")

var errNoRegistryResolver = errors.New("no resolver set in the ENS registry")

func mustParseABI(definition string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		panic(err) // the definitions are compile-time constants
	}
	return parsed
}

func NewEnsResolver(rpcClient *rpc.Client) *EnsResolver {
	return &EnsResolver{
		contractMaker:   contracts.NewContractMaker(rpcClient),
		ethClientGetter: rpcClient,

		quit: make(chan struct{}),
	}
}

type EnsResolver struct {
	contractMaker   *contracts.ContractMaker
	ethClientGetter rpc.EthClientGetter

	quitOnce sync.Once
	quit     chan struct{}
}

func (e *EnsResolver) Stop() {
	e.quitOnce.Do(func() {
		close(e.quit)
	})
}

func (e *EnsResolver) GetRegistrarAddress(ctx context.Context, chainID uint64) (common.Address, error) {
	return registrar.ContractAddress(chainID)
}

// resolveENSRecord executes a public-resolver record call (e.g. `addr`, `pubkey`,
// `contenthash`) for the given username and returns the decoded output values.
func (e *EnsResolver) resolveENSRecord(ctx context.Context, chainID uint64, username, method string) ([]interface{}, error) {
	result, _, err := e.resolve(ctx, chainID, username, method)
	if err != nil {
		return nil, err
	}

	return ensRecordMethods.Unpack(method, result)
}

// resolve executes a record call through the ENS Universal Resolver. Status usernames
// it cannot resolve are read from the ENS registry instead.
func (e *EnsResolver) resolve(ctx context.Context, chainID uint64, username, method string) ([]byte, common.Address, error) {
	if err := walletCommon.ValidateENSUsername(username); err != nil {
		return nil, common.Address{}, err
	}

	universalResolver, err := e.contractMaker.NewUniversalResolver(chainID)
	if err != nil {
		return nil, common.Address{}, err
	}

	node := walletCommon.NameHash(username)
	data, err := ensRecordMethods.Pack(method, [32]byte(node))
	if err != nil {
		return nil, common.Address{}, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	result, resolverAddress, err := universalResolver.Resolve(callOpts, universalresolver.DNSEncode(username), data)
	if err == nil || !isStatusUsername(username) {
		return result, resolverAddress, err
	}

	result, resolverAddress, registryErr := e.resolveThroughRegistry(ctx, chainID, node, data)
	if registryErr != nil {
		return nil, common.Address{}, err
	}
	return result, resolverAddress, nil
}

// resolveThroughRegistry executes a record call on the resolver the ENS registry
// holds for the node.
func (e *EnsResolver) resolveThroughRegistry(ctx context.Context, chainID uint64, node common.Hash, data []byte) ([]byte, common.Address, error) {
	backend, err := e.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, common.Address{}, err
	}

	registryCall, err := ensRegistryMethods.Pack("resolver", [32]byte(node))
	if err != nil {
		return nil, common.Address{}, err
	}
	output, err := backend.CallContract(ctx, ethereum.CallMsg{To: &ensRegistryAddress, Data: registryCall}, nil)
	if err != nil {
		return nil, common.Address{}, err
	}
	values, err := ensRegistryMethods.Unpack("resolver", output)
	if err != nil {
		return nil, common.Address{}, err
	}
	resolverAddress := *abi.ConvertType(values[0], new(common.Address)).(*common.Address)
	if resolverAddress == (common.Address{}) {
		return nil, common.Address{}, errNoRegistryResolver
	}

	result, err := backend.CallContract(ctx, ethereum.CallMsg{To: &resolverAddress, Data: data}, nil)
	if err != nil {
		return nil, common.Address{}, err
	}
	return result, resolverAddress, nil
}

func isStatusUsername(username string) bool {
	return strings.HasSuffix(username, "."+walletCommon.StatusDomain)
}

func (e *EnsResolver) Resolver(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	// Resolve an addr() record to discover the resolver that answers for this name.
	_, resolverAddress, err := e.resolve(ctx, chainID, username, "addr")
	if err != nil {
		return nil, err
	}

	return &resolverAddress, nil
}

func (e *EnsResolver) GetName(ctx context.Context, chainID uint64, address common.Address) (string, error) {
	universalResolver, err := e.contractMaker.NewUniversalResolver(chainID)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	// coinType 60 = ETH (ENSIP-11/19), the default Ethereum reverse record.
	name, _, _, err := universalResolver.Reverse(callOpts, address.Bytes(), big.NewInt(60))
	if err != nil {
		return "", err
	}

	return name, nil
}

func (e *EnsResolver) OwnerOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	err := walletCommon.ValidateENSUsername(username)
	if err != nil {
		return nil, err
	}

	// Status usernames are tracked by the username registrar itself, so ownership
	// is answered there without going through the ENS registry.
	if isStatusUsername(username) {
		return e.statusUsernameOwner(ctx, chainID, username)
	}

	// For any other name, ownership is read from the NameWrapper: wrapped names
	// report their true owner, unwrapped ones the zero address (treated as unowned).
	nameWrapperAddress, err := NameWrapperContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	return e.resolveWrappedOwner(ctx, chainID, walletCommon.NameHash(username), nameWrapperAddress)
}

// statusUsernameOwner returns the owner of a `*.stateofus.eth` username as recorded
// by the Status username registrar (zero address if the username is not registered).
func (e *EnsResolver) statusUsernameOwner(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	registrarAddr, err := registrar.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	usernameRegistrar, err := e.contractMaker.NewUsernameRegistrar(chainID, registrarAddr)
	if err != nil {
		return nil, err
	}

	label := walletCommon.UsernameToLabel(strings.TrimSuffix(username, "."+walletCommon.StatusDomain))
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	owner, err := usernameRegistrar.GetAccountOwner(callOpts, label)
	if err != nil {
		return nil, err
	}

	return &owner, nil
}

// resolveWrappedOwner resolves the ENS owner recorded by the NameWrapper contract.
func (e *EnsResolver) resolveWrappedOwner(ctx context.Context, chainID uint64, nameHash common.Hash, wrapperAddr common.Address) (*common.Address, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}

	nameWrapper, err := e.contractMaker.NewNameWrapper(chainID, &wrapperAddr)
	if err != nil {
		return nil, err
	}

	// Convert the namehash to tokenId (uint256)
	tokenId := new(big.Int).SetBytes(nameHash.Bytes())

	owner, err := nameWrapper.OwnerOf(callOpts, tokenId)
	if err != nil {
		return nil, nil // treat as unowned
	}

	return &owner, nil
}

func (e *EnsResolver) ContentHash(ctx context.Context, chainID uint64, username string) ([]byte, error) {
	values, err := e.resolveENSRecord(ctx, chainID, username, "contenthash")
	if err != nil {
		// Mirror previous behavior: treat a missing record / failed resolution as "no content hash".
		return nil, nil
	}

	contentHash := *abi.ConvertType(values[0], new([]byte)).(*[]byte)
	return contentHash, nil
}

func (e *EnsResolver) PublicKeyOf(ctx context.Context, chainID uint64, username string) (string, error) {
	values, err := e.resolveENSRecord(ctx, chainID, username, "pubkey")
	if err != nil {
		return "", err
	}

	x := *abi.ConvertType(values[0], new([32]byte)).(*[32]byte)
	y := *abi.ConvertType(values[1], new([32]byte)).(*[32]byte)
	return "0x04" + hex.EncodeToString(x[:]) + hex.EncodeToString(y[:]), nil
}

func (e *EnsResolver) AddressOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	values, err := e.resolveENSRecord(ctx, chainID, username, "addr")
	if err != nil {
		return nil, err
	}

	addr := *abi.ConvertType(values[0], new(common.Address)).(*common.Address)
	return &addr, nil
}

func (e *EnsResolver) ExpireAt(ctx context.Context, chainID uint64, username string) (string, error) {
	registrarAddr, err := registrar.ContractAddress(chainID)
	if err != nil {
		return "", err
	}

	usernameRegistrar, err := e.contractMaker.NewUsernameRegistrar(chainID, registrarAddr)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	expTime, err := usernameRegistrar.GetExpirationTime(callOpts, walletCommon.UsernameToLabel(username))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", expTime), nil
}

func (e *EnsResolver) Price(ctx context.Context, chainID uint64) (string, error) {
	registrarAddr, err := registrar.ContractAddress(chainID)
	if err != nil {
		return "", err
	}

	usernameRegistrar, err := e.contractMaker.NewUsernameRegistrar(chainID, registrarAddr)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	price, err := usernameRegistrar.GetPrice(callOpts)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", price), nil
}
