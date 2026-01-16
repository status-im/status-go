package wallet

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
)

// FetchTokenBalances return a map with key as chain id and value as map of account address and map of token address and balance
// [chainID][account][tokenAddress]balance
func (api *API) FetchTokenBalances(ctx context.Context, addresses []common.Address, tokenKeys []string) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	logutils.ZapLogger().Debug("wallet.api.FetchTokenBalances", zap.Any("addresses", addresses), zap.Any("tokenKeys", tokenKeys))
	ret := make(map[uint64]map[common.Address]map[common.Address]*hexutil.Big)

	tokensPerChain := make(map[uint64][]common.Address)
	for _, tokenKey := range tokenKeys {
		token, err := api.s.tokenManager.GetTokenByKey(tokenKey)
		if err != nil {
			return nil, err
		}
		tokensPerChain[token.ChainID] = append(tokensPerChain[token.ChainID], token.Address)
	}

	for chainID, tokens := range tokensPerChain {
		ret[chainID] = make(map[common.Address]map[common.Address]*hexutil.Big)
		fetchResults, err := api.s.tokenBalancesFetcher.Fetch(ctx, chainID, tokens, addresses)
		if err != nil {
			return nil, err
		}
		for account, tokenBalances := range fetchResults {
			ret[chainID][account] = make(map[common.Address]*hexutil.Big)
			for token, balance := range tokenBalances {
				// Only report positive balances
				if balance.Sign() <= 0 {
					continue
				}
				ret[chainID][account][token] = (*hexutil.Big)(balance)
			}
		}
	}

	return ret, nil
}

// GetAllBalances returns cached balances along with fetching state information
// This method retrieves cached balances from storage without triggering a new fetch
type balancesState struct {
	State         tokenbalances.LoaderState `json:"state"`
	AtBlockNumber *big.Int                  `json:"atBlockNumber"`
	AtBlockHash   common.Hash               `json:"atBlockHash"`
	FetchedAt     int64                     `json:"fetchedAt"`
}

type balancesWithState struct {
	Balances map[tokenbalances.ContractAddress]*hexutil.Big `json:"balances"`
	State    balancesState                                  `json:"state"`
}

type balancesWithStatePerChainID map[uint64]balancesWithState

type balancesWithStatePerAddressAndChainID map[tokenbalances.AccountAddress]balancesWithStatePerChainID

func (api *API) GetAllTokenBalances(ctx context.Context, chainIDs []uint64, addresses []common.Address) (balancesWithStatePerAddressAndChainID, error) {
	logutils.ZapLogger().Debug("wallet.api.GetAllTokenBalances", zap.Any("chainIDs", chainIDs), zap.Any("addresses", addresses))
	// Get cached balances from storage (no fetch)
	tmpBalancesWithState, err := api.s.tokenBalancesService.GetAllBalancesWithState(ctx, chainIDs, addresses)
	if err != nil {
		return nil, err
	}

	result := make(balancesWithStatePerAddressAndChainID)
	for address, tmpChainIDBalances := range tmpBalancesWithState {
		result[address] = make(balancesWithStatePerChainID)
		for chainID, tmpBalancesWithState := range tmpChainIDBalances {
			result[address][chainID] = balancesWithState{
				Balances: make(map[tokenbalances.ContractAddress]*hexutil.Big),
				State: balancesState{
					State:         tmpBalancesWithState.State.State,
					AtBlockNumber: tmpBalancesWithState.State.AtBlockNumber,
					AtBlockHash:   tmpBalancesWithState.State.AtBlockHash,
					FetchedAt:     tmpBalancesWithState.State.FetchedAt,
				},
			}
			for token, balance := range tmpBalancesWithState.Balances {
				result[address][chainID].Balances[token] = (*hexutil.Big)(balance)
			}
		}
	}

	return result, nil
}

func (api *API) RefetchAllTokenBalances(ctx context.Context) error {
	logutils.ZapLogger().Debug("wallet.api.RefetchAllTokenBalances")
	return api.s.tokenBalancesService.ForceRefresh()
}
