package ensresolver

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/contracts"
	"github.com/status-im/status-go/internal/contracts/resolver"
	"github.com/status-im/status-go/internal/contracts/universalresolver"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/services/ens/ccipread"
	"github.com/status-im/status-go/services/ens/urlookup"
	"github.com/status-im/status-go/services/ens/validate"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

// isResolverNotFound matches the Universal Resolver's ResolverNotFound revert.
// Pre-ENSv2 the registry returned the zero address for names with no resolver
// (no error). Downstream code (e.g. processor_ens_public_key) still switches
// on ZeroAddress, so we preserve that contract here.
func isResolverNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "ResolverNotFound")
}

// errNotAnENSName is returned by lookup methods when the supplied username
// is not a plausible ENS name. The API layer guards most entry points, but
// internal callers should not rely on that.
var errNotAnENSName = errors.New("ensresolver: not a recognizable ENS name")

func NewEnsResolver(rpcClient *rpc.Client) *EnsResolver {
	return &EnsResolver{
		contractMaker:   contracts.NewContractMaker(rpcClient),
		ethClientGetter: rpcClient,
		addrPerChain:    make(map[uint64]common.Address),

		quit: make(chan struct{}),
	}
}

type EnsResolver struct {
	contractMaker   *contracts.ContractMaker
	ethClientGetter rpc.EthClientGetter

	addrPerChain      map[uint64]common.Address
	addrPerChainMutex sync.Mutex

	quitOnce sync.Once
	quit     chan struct{}
}

func (e *EnsResolver) Stop() {
	e.quitOnce.Do(func() {
		close(e.quit)
	})
}

func (e *EnsResolver) GetRegistrarAddress(ctx context.Context, chainID uint64) (common.Address, error) {
	return e.usernameRegistrarAddr(ctx, chainID)
}

// urBackend returns a CCIP-Read-aware backend and the Universal Resolver
// address for the given chain. UR.resolve / UR.reverse calls go through this
// backend so that offchain (L2 / DNS-imported) names resolve transparently.
func (e *EnsResolver) urBackend(chainID uint64) (bind.ContractBackend, common.Address, error) {
	urAddr, err := universalresolver.ContractAddress(chainID)
	if err != nil {
		return nil, common.Address{}, err
	}
	raw, err := e.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, common.Address{}, err
	}
	return ccipread.New(raw), urAddr, nil
}

func (e *EnsResolver) Resolver(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	backend, urAddr, err := e.urBackend(chainID)
	if err != nil {
		return nil, err
	}
	addr, err := urlookup.Resolver(ctx, backend, urAddr, username)
	if err != nil {
		if isResolverNotFound(err) {
			zero := common.Address{}
			return &zero, nil
		}
		return nil, err
	}
	return &addr, nil
}

func (e *EnsResolver) GetName(ctx context.Context, chainID uint64, address common.Address) (string, error) {
	backend, urAddr, err := e.urBackend(chainID)
	if err != nil {
		return "", err
	}
	return urlookup.Reverse(ctx, backend, urAddr, address)
}

func (e *EnsResolver) OwnerOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	if !validate.IsLikelyENSName(username) {
		return nil, errNotAnENSName
	}
	nameHash := walletCommon.NameHash(username)

	registry, err := e.contractMaker.NewRegistry(chainID)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	owner, err := registry.Owner(callOpts, nameHash)
	if err != nil {
		return nil, err
	}

	nameWrapperAddress, err := NameWrapperContractAddress(chainID)
	if err != nil {
		return &owner, nil
	}

	if owner != nameWrapperAddress {
		return &owner, nil
	}

	return e.resolveWrappedOwner(ctx, chainID, nameHash, nameWrapperAddress)
}

func (e *EnsResolver) resolveWrappedOwner(ctx context.Context, chainID uint64, nameHash common.Hash, wrapperAddr common.Address) (*common.Address, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}

	nameWrapper, err := e.contractMaker.NewNameWrapper(chainID, &wrapperAddr)
	if err != nil {
		return nil, err
	}

	tokenId := new(big.Int).SetBytes(nameHash.Bytes())

	owner, err := nameWrapper.OwnerOf(callOpts, tokenId)
	if err != nil {
		return nil, nil
	}

	return &owner, nil
}

func (e *EnsResolver) ContentHash(ctx context.Context, chainID uint64, username string) ([]byte, error) {
	backend, urAddr, err := e.urBackend(chainID)
	if err != nil {
		return nil, err
	}
	hash, err := urlookup.Contenthash(ctx, backend, urAddr, username)
	if err != nil {
		// Pre-ENSv2 callers treat "no contenthash" as nil/nil. Preserve that
		// for the empty-record case and for names with no resolver, but
		// surface real transport/decoding errors so they can be logged.
		if errors.Is(err, urlookup.ErrEmptyResolverData) || isResolverNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return hash, nil
}

func (e *EnsResolver) PublicKeyOf(ctx context.Context, chainID uint64, username string) (string, error) {
	backend, urAddr, err := e.urBackend(chainID)
	if err != nil {
		return "", err
	}
	return urlookup.PubkeyHex(ctx, backend, urAddr, username)
}

func (e *EnsResolver) AddressOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	backend, urAddr, err := e.urBackend(chainID)
	if err != nil {
		return nil, err
	}
	addr, err := urlookup.Address(ctx, backend, urAddr, username)
	if err != nil {
		return nil, err
	}
	return &addr, nil
}

func (e *EnsResolver) usernameRegistrarAddr(ctx context.Context, chainID uint64) (common.Address, error) {
	logutils.ZapLogger().Info("obtaining username registrar address")
	e.addrPerChainMutex.Lock()
	defer e.addrPerChainMutex.Unlock()
	addr, ok := e.addrPerChain[chainID]
	if ok {
		return addr, nil
	}

	registryAddr, err := e.OwnerOf(ctx, chainID, walletCommon.StatusDomain)
	if err != nil {
		return common.Address{}, err
	}

	e.addrPerChain[chainID] = *registryAddr

	go func() {
		defer gocommon.LogOnPanic()
		registry, err := e.contractMaker.NewRegistry(chainID)
		if err != nil {
			return
		}

		logs := make(chan *resolver.ENSRegistryWithFallbackNewOwner)

		sub, err := registry.WatchNewOwner(&bind.WatchOpts{}, logs, nil, nil)
		if err != nil {
			return
		}

		for {
			select {
			case <-e.quit:
				logutils.ZapLogger().Info("quitting ens contract subscription")
				sub.Unsubscribe()
				return
			case err := <-sub.Err():
				if err != nil {
					logutils.ZapLogger().Error("ens contract subscription error: " + err.Error())
				}
				return
			case vLog := <-logs:
				e.addrPerChainMutex.Lock()
				e.addrPerChain[chainID] = vLog.Owner
				e.addrPerChainMutex.Unlock()
			}
		}
	}()

	return *registryAddr, nil
}

func (e *EnsResolver) ExpireAt(ctx context.Context, chainID uint64, username string) (string, error) {
	registryAddr, err := e.usernameRegistrarAddr(ctx, chainID)
	if err != nil {
		return "", err
	}

	registrar, err := e.contractMaker.NewUsernameRegistrar(chainID, registryAddr)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	expTime, err := registrar.GetExpirationTime(callOpts, walletCommon.UsernameToLabel(username))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", expTime), nil
}

func (e *EnsResolver) Price(ctx context.Context, chainID uint64) (string, error) {
	registryAddr, err := e.usernameRegistrarAddr(ctx, chainID)
	if err != nil {
		return "", err
	}

	registrar, err := e.contractMaker.NewUsernameRegistrar(chainID, registryAddr)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	price, err := registrar.GetPrice(callOpts)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", price), nil
}
