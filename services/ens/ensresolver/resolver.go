package ensresolver

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/wealdtech/go-ens/v3"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/contracts/resolver"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
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

func (e *EnsResolver) Release(ctx context.Context, chainID uint64, registryAddress common.Address, txArgs wallettypes.SendTxArgs, username string, signFn bind.SignerFn) (*types.Transaction, error) {
	registrar, err := e.contractMaker.NewUsernameRegistrar(chainID, registryAddress)
	if err != nil {
		return nil, err
	}

	txOpts := txArgs.ToTransactOpts(signFn)
	return registrar.Release(txOpts, walletCommon.UsernameToLabel(username))
}

func (e *EnsResolver) ReleaseEstimate(ctx context.Context, chainID uint64, callMsg ethereum.CallMsg) (uint64, error) {
	ethClient, err := e.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	estimate, err := ethClient.EstimateGas(ctx, callMsg)
	if err != nil {
		return 0, err
	}
	return estimate + 1000, nil
}

func (e *EnsResolver) Register(ctx context.Context, chainID uint64, registryAddress common.Address, txArgs wallettypes.SendTxArgs, username string, pubkey string, signFn bind.SignerFn) (*types.Transaction, error) {
	snt, err := e.contractMaker.NewSNT(chainID)
	if err != nil {
		return nil, err
	}

	priceHex, err := e.Price(ctx, chainID)
	if err != nil {
		return nil, err
	}
	price := new(big.Int)
	price.SetString(priceHex, 16)

	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return nil, err
	}

	x, y := walletCommon.ExtractCoordinates(pubkey)
	extraData, err := registrarABI.Pack("register", walletCommon.UsernameToLabel(username), common.Address(txArgs.From), x, y)
	if err != nil {
		return nil, err
	}

	txOpts := txArgs.ToTransactOpts(signFn)
	return snt.ApproveAndCall(
		txOpts,
		registryAddress,
		price,
		extraData,
	)
}

func (e *EnsResolver) RegisterEstimate(ctx context.Context, chainID uint64, callMsg ethereum.CallMsg) (uint64, error) {
	ethClient, err := e.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	estimate, err := ethClient.EstimateGas(ctx, callMsg)
	if err != nil {
		return 0, err
	}
	return estimate + 1000, nil
}

func (e *EnsResolver) SetPubKey(ctx context.Context, chainID uint64, resolverAddress *common.Address, txArgs wallettypes.SendTxArgs, username string, pubkey string, signFn bind.SignerFn) (*types.Transaction, error) {
	err := walletCommon.ValidateENSUsername(username)
	if err != nil {
		return nil, err
	}

	resolver, err := e.contractMaker.NewPublicResolver(chainID, resolverAddress)
	if err != nil {
		return nil, err
	}

	x, y := walletCommon.ExtractCoordinates(pubkey)
	txOpts := txArgs.ToTransactOpts(signFn)
	return resolver.SetPubkey(txOpts, walletCommon.NameHash(username), x, y)
}

func (e *EnsResolver) SetPubKeyEstimate(ctx context.Context, chainID uint64, callMsg ethereum.CallMsg) (uint64, error) {
	ethClient, err := e.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	estimate, err := ethClient.EstimateGas(ctx, callMsg)
	if err != nil {
		return 0, err
	}
	return estimate + 1000, nil
}
