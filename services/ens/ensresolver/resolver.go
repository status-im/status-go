package ensresolver

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/wealdtech/go-ens/v3"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/resolver"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func NewEnsResolver(rpcClient *rpc.Client) *EnsResolver {
	return &EnsResolver{
		contractMaker: &contracts.ContractMaker{
			RPCClient: rpcClient,
		},
		addrPerChain: make(map[uint64]common.Address),

		quit: make(chan struct{}),
	}
}

type EnsResolver struct {
	contractMaker *contracts.ContractMaker

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

func (e *EnsResolver) Resolver(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	err := walletCommon.ValidateENSUsername(username)
	if err != nil {
		return nil, err
	}

	registry, err := e.contractMaker.NewRegistry(chainID)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	resolver, err := registry.Resolver(callOpts, walletCommon.NameHash(username))
	if err != nil {
		return nil, err
	}

	return &resolver, nil
}

func (e *EnsResolver) GetName(ctx context.Context, chainID uint64, address common.Address) (string, error) {
	backend, err := e.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return "", err
	}
	return ens.ReverseResolve(backend, address)
}

func (e *EnsResolver) OwnerOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	err := walletCommon.ValidateENSUsername(username)
	if err != nil {
		return nil, err
	}

	registry, err := e.contractMaker.NewRegistry(chainID)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	owner, err := registry.Owner(callOpts, walletCommon.NameHash(username))
	if err != nil {
		return nil, err
	}

	return &owner, nil
}

func (e *EnsResolver) ContentHash(ctx context.Context, chainID uint64, username string) ([]byte, error) {
	err := walletCommon.ValidateENSUsername(username)
	if err != nil {
		return nil, err
	}

	resolverAddress, err := e.Resolver(ctx, chainID, username)
	if err != nil {
		return nil, err
	}

	resolver, err := e.contractMaker.NewPublicResolver(chainID, resolverAddress)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contentHash, err := resolver.Contenthash(callOpts, walletCommon.NameHash(username))
	if err != nil {
		return nil, nil
	}

	return contentHash, nil
}

func (e *EnsResolver) PublicKeyOf(ctx context.Context, chainID uint64, username string) (string, error) {
	err := walletCommon.ValidateENSUsername(username)
	if err != nil {
		return "", err
	}

	resolverAddress, err := e.Resolver(ctx, chainID, username)
	if err != nil {
		return "", err
	}

	resolver, err := e.contractMaker.NewPublicResolver(chainID, resolverAddress)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	pubKey, err := resolver.Pubkey(callOpts, walletCommon.NameHash(username))
	if err != nil {
		return "", err
	}
	return "0x04" + hex.EncodeToString(pubKey.X[:]) + hex.EncodeToString(pubKey.Y[:]), nil
}

func (e *EnsResolver) AddressOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	err := walletCommon.ValidateENSUsername(username)
	if err != nil {
		return nil, err
	}

	resolverAddress, err := e.Resolver(ctx, chainID, username)
	if err != nil {
		return nil, err
	}

	resolver, err := e.contractMaker.NewPublicResolver(chainID, resolverAddress)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	addr, err := resolver.Addr(callOpts, walletCommon.NameHash(username))
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
