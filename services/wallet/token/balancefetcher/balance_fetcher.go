package balancefetcher

import (
	"context"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/ethscan"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
)

var NativeChainAddress = common.HexToAddress("0x")
var requestTimeout = 20 * time.Second

var (
	errScanningContract = errors.New("error scanning contract")
)

const (
	tokenChunkSize = 500
)

type BalanceFetcher interface {
	GetTokenBalanceAt(ctx context.Context, client chain.ClientInterface, account common.Address, token common.Address, blockNumber *big.Int) (*big.Int, error)
	GetBalancesAtByChain(parent context.Context, clients map[uint64]chain.ClientInterface, accounts, tokens []common.Address, atBlocks map[uint64]*big.Int) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error)
	GetBalancesByChain(parent context.Context, clients map[uint64]chain.ClientInterface, accounts, tokens []common.Address) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error)
	GetBalance(ctx context.Context, client chain.ClientInterface, account common.Address, token common.Address) (*big.Int, error)
	GetChainBalance(ctx context.Context, client chain.ClientInterface, account common.Address) (*big.Int, error)
}

type DefaultBalanceFetcher struct {
	contractMaker contracts.ContractMakerIface
}

func NewDefaultBalanceFetcher(contractMaker contracts.ContractMakerIface) *DefaultBalanceFetcher {
	return &DefaultBalanceFetcher{
		contractMaker: contractMaker,
	}
}

func (bf *DefaultBalanceFetcher) fetchBalancesForChain(parent context.Context, client chain.ClientInterface, accounts, tokens []common.Address, atBlock *big.Int) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	var (
		group = async.NewAtomicGroup(parent)
		mu    sync.Mutex
	)

	balances := make(map[common.Address]map[common.Address]*hexutil.Big)
	updateBalance := func(accTokenBalance map[common.Address]map[common.Address]*hexutil.Big) {
		mu.Lock()
		defer mu.Unlock()

		for account, tokenBalance := range accTokenBalance {
			if _, ok := balances[account]; !ok {
				balances[account] = make(map[common.Address]*hexutil.Big)
			}

			for token, balance := range tokenBalance {
				balances[account][token] = balance
			}
		}
	}

	ethScanContract, availableAtBlock, err := bf.contractMaker.NewEthScan(client.NetworkID())
	if err != nil {
		return nil, errors.Wrap(err, errScanningContract.Error())
	}

	fetchChainBalance := false

	for _, token := range tokens {
		if token == NativeChainAddress {
			fetchChainBalance = true
		}
	}
	if fetchChainBalance {
		group.Add(func(parent context.Context) error {
			balances, err := bf.FetchChainBalances(parent, accounts, ethScanContract, atBlock)
			if err != nil {
				return err
			}

			updateBalance(balances)
			return nil
		})
	}

	tokenChunks := splitTokensToChunks(tokens, tokenChunkSize)
	for accountIdx := range accounts {
		// Keep the reference to the account. DO NOT USE A LOOP, the account will be overridden in the coroutine
		account := accounts[accountIdx]
		for idx := range tokenChunks {
			// Keep the reference to the chunk. DO NOT USE A LOOP, the chunk will be overridden in the coroutine
			chunk := tokenChunks[idx]

			group.Add(func(parent context.Context) error {
				ctx, cancel := context.WithTimeout(parent, requestTimeout)
				defer cancel()

				var accTokenBalance map[common.Address]map[common.Address]*hexutil.Big
				var err error
				if atBlock == nil || big.NewInt(int64(availableAtBlock)).Cmp(atBlock) < 0 {
					accTokenBalance, err = bf.FetchTokenBalancesWithScanContract(ctx, ethScanContract, account, chunk, atBlock)
				} else {
					accTokenBalance, err = bf.fetchTokenBalancesWithTokenContracts(ctx, client, account, chunk, atBlock)
				}

				if err != nil {
					return err
				}

				updateBalance(accTokenBalance)
				return nil
			})
		}
	}

	select {
	case <-group.WaitAsync():
	case <-parent.Done():
		return nil, parent.Err()
	}
	return balances, group.Error()
}

func (bf *DefaultBalanceFetcher) FetchChainBalances(parent context.Context, accounts []common.Address, ethScanContract ethscan.BalanceScannerIface, atBlock *big.Int) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	accTokenBalance := make(map[common.Address]map[common.Address]*hexutil.Big)

	ctx, cancel := context.WithTimeout(parent, requestTimeout)
	defer cancel()

	res, err := ethScanContract.EtherBalances(&bind.CallOpts{
		Context:     ctx,
		BlockNumber: atBlock,
	}, accounts)
	if err != nil {
		logutils.ZapLogger().Error("can't fetch chain balance 5", zap.Error(err))
		return nil, err
	}
	for idx, account := range accounts {
		balance := new(big.Int)
		balance.SetBytes(res[idx].Data)

		if _, ok := accTokenBalance[account]; !ok {
			accTokenBalance[account] = make(map[common.Address]*hexutil.Big)
		}

		accTokenBalance[account][NativeChainAddress] = (*hexutil.Big)(balance)
	}

	return accTokenBalance, nil
}

func (bf *DefaultBalanceFetcher) FetchTokenBalancesWithScanContract(ctx context.Context, ethScanContract ethscan.BalanceScannerIface, account common.Address, chunk []common.Address, atBlock *big.Int) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	accTokenBalance := make(map[common.Address]map[common.Address]*hexutil.Big)
	res, err := ethScanContract.TokensBalance(&bind.CallOpts{
		Context:     ctx,
		BlockNumber: atBlock,
	}, account, chunk)
	if err != nil {
		logutils.ZapLogger().Error("can't fetch erc20 token balance 6", zap.Stringer("account", account), zap.Error(err))
		return nil, err
	}

	if len(res) != len(chunk) {
		logutils.ZapLogger().Error("can't fetch erc20 token balance 7",
			zap.Stringer("account", account),
			zap.Error(errors.New("response not complete")),
			zap.Int("expected", len(chunk)),
			zap.Int("got", len(res)),
		)
		return nil, errors.New("response not complete")
	}

	for idx, token := range chunk {
		if !res[idx].Success {
			continue
		}
		balance := new(big.Int)
		balance.SetBytes(res[idx].Data)

		if _, ok := accTokenBalance[account]; !ok {
			accTokenBalance[account] = make(map[common.Address]*hexutil.Big)
		}

		accTokenBalance[account][token] = (*hexutil.Big)(balance)
	}
	return accTokenBalance, nil
}

func (bf *DefaultBalanceFetcher) fetchTokenBalancesWithTokenContracts(ctx context.Context, client chain.ClientInterface, account common.Address, chunk []common.Address, atBlock *big.Int) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	accTokenBalance := make(map[common.Address]map[common.Address]*hexutil.Big)
	for _, token := range chunk {
		balance, err := bf.GetTokenBalanceAt(ctx, client, account, token, atBlock)
		if err != nil {
			if err != bind.ErrNoCode {
				logutils.ZapLogger().Error("can't fetch erc20 token balance 8",
					zap.Stringer("account", account),
					zap.Stringer("token", token),
					zap.Error(errors.New("on fetching token balance")),
				)
				return nil, err
			}
		}

		if _, ok := accTokenBalance[account]; !ok {
			accTokenBalance[account] = make(map[common.Address]*hexutil.Big)
		}

		accTokenBalance[account][token] = (*hexutil.Big)(balance)
	}

	return accTokenBalance, nil
}

func (bf *DefaultBalanceFetcher) GetTokenBalanceAt(ctx context.Context, client chain.ClientInterface, account common.Address, token common.Address, blockNumber *big.Int) (*big.Int, error) {
	caller, err := bf.contractMaker.NewERC20Caller(client.NetworkID(), token)
	if err != nil {
		return nil, err
	}

	balance, err := caller.BalanceOf(&bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}, account)

	if err != nil {
		if err != bind.ErrNoCode {
			return nil, err
		}
		balance = big.NewInt(0)
	}

	return balance, nil
}

func splitTokensToChunks(tokens []common.Address, chunkSize int) [][]common.Address {
	tokenChunks := make([][]common.Address, 0)
	for i := 0; i < len(tokens); i += chunkSize {
		end := i + chunkSize
		if end > len(tokens) {
			end = len(tokens)
		}

		tokenChunks = append(tokenChunks, tokens[i:end])
	}

	return tokenChunks
}

func (tm *DefaultBalanceFetcher) GetTokenBalance(ctx context.Context, client chain.ClientInterface, account common.Address, token common.Address) (*big.Int, error) {
	caller, err := ierc20.NewIERC20Caller(token, client)
	if err != nil {
		return nil, err
	}

	return caller.BalanceOf(&bind.CallOpts{
		Context: ctx,
	}, account)
}

func (bf *DefaultBalanceFetcher) GetChainBalance(ctx context.Context, client chain.ClientInterface, account common.Address) (*big.Int, error) {
	return client.BalanceAt(ctx, account, nil)
}

func (bf *DefaultBalanceFetcher) GetBalance(ctx context.Context, client chain.ClientInterface, account common.Address, token common.Address) (*big.Int, error) {
	if token == NativeChainAddress {
		return bf.GetChainBalance(ctx, client, account)
	}

	return bf.GetTokenBalance(ctx, client, account, token)
}

func (bf *DefaultBalanceFetcher) GetBalancesByChain(parent context.Context, clients map[uint64]chain.ClientInterface, accounts, tokens []common.Address) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	return bf.GetBalancesAtByChain(parent, clients, accounts, tokens, nil)
}

func (bf *DefaultBalanceFetcher) GetBalancesAtByChain(parent context.Context, clients map[uint64]chain.ClientInterface, accounts, tokens []common.Address, atBlocks map[uint64]*big.Int) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	var (
		group    = async.NewAtomicGroup(parent)
		mu       sync.Mutex
		response = map[uint64]map[common.Address]map[common.Address]*hexutil.Big{}
	)

	updateBalance := func(chainID uint64, accTokenBalance map[common.Address]map[common.Address]*hexutil.Big) {
		mu.Lock()
		defer mu.Unlock()

		if _, ok := response[chainID]; !ok {
			response[chainID] = map[common.Address]map[common.Address]*hexutil.Big{}
		}

		for account, tokenBalance := range accTokenBalance {
			response[chainID][account] = tokenBalance
		}
	}

	for clientIdx := range clients {
		// Keep the reference to the client. DO NOT USE A LOOP, the client will be overridden in the coroutine
		client := clients[clientIdx]

		group.Add(func(parent context.Context) error {
			balances, err := bf.fetchBalancesForChain(parent, client, accounts, tokens, atBlocks[client.NetworkID()])
			if err != nil {
				return err
			}
			updateBalance(client.NetworkID(), balances)
			return nil
		})
	}
	select {
	case <-group.WaitAsync():
	case <-parent.Done():
		return nil, parent.Err()
	}
	if group.Error() != nil {
		logutils.ZapLogger().Error("failed to get balances by chain", zap.Error(group.Error()))
		return nil, group.Error()
	}
	return response, nil
}
