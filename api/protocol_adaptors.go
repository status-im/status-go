package api

import (
	"context"
	"fmt"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
)

var _ communities.NetworkManager = (*CommunitiesNetworkManager)(nil)
var _ communities.TokenManager = (*CommunitiesTokenManager)(nil)
var _ communities.TokenBalanceManager = (*CommunitiesTokenBalanceManager)(nil)

type CommunitiesNetworkManager struct {
	networkManager *network.Manager
}

func NewCommunitiesNetworkManager(nm *network.Manager) *CommunitiesNetworkManager {
	return &CommunitiesNetworkManager{networkManager: nm}
}

func (m *CommunitiesNetworkManager) GetAllChainIDs() ([]uint64, error) {
	networks, err := m.networkManager.GetActiveNetworks()
	if err != nil {
		return nil, err
	}

	chainIDs := make([]uint64, 0)
	for _, network := range networks {
		chainIDs = append(chainIDs, network.ChainID)
	}
	return chainIDs, nil
}

type CommunitiesTokenManager struct {
	tokenManager *token.Manager
}

func NewCommunitiesTokenManager(tm *token.Manager) *CommunitiesTokenManager {
	return &CommunitiesTokenManager{tokenManager: tm}
}

func (m *CommunitiesTokenManager) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address gethcommon.Address) *tokenTypes.Token {
	return m.tokenManager.FindOrCreateTokenByAddress(ctx, chainID, address)
}

type CommunitiesTokenBalanceManager struct {
	tokenBalancesFetcher tokenbalances.FetcherIface
	tokenBalancesStorage tokenbalances.Storage
}

func NewCommunitiesTokenBalanceManager(f tokenbalances.FetcherIface, s tokenbalances.Storage) *CommunitiesTokenBalanceManager {
	return &CommunitiesTokenBalanceManager{tokenBalancesFetcher: f, tokenBalancesStorage: s}
}

func (m *CommunitiesTokenBalanceManager) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (communities.BalancesByChain, error) {
	if m.tokenBalancesFetcher == nil {
		return nil, fmt.Errorf("tokenBalancesFetcher is nil")
	}
	ret := make(communities.BalancesByChain)
	for _, chainID := range chainIDs {
		ret[chainID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
		balances, err := m.tokenBalancesFetcher.Fetch(ctx, chainID, tokenAddresses, accounts)
		if err != nil {
			return nil, err
		}
		ret[chainID] = balancesToCommunitiesBalances(balances)
	}
	return ret, nil
}

func (m *CommunitiesTokenBalanceManager) GetCachedBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (communities.BalancesByChain, error) {
	if m.tokenBalancesStorage == nil {
		return nil, fmt.Errorf("tokenBalancesStorage is nil")
	}
	ret := make(communities.BalancesByChain)
	for _, chainID := range chainIDs {
		balances, err := m.tokenBalancesStorage.GetBalances(ctx, chainID, tokenAddresses, accounts)
		if err != nil {
			return nil, err
		}
		ret[chainID] = balancesToCommunitiesBalances(balances)
	}
	return ret, nil
}

func balancesToCommunitiesBalances(balances map[gethcommon.Address]map[gethcommon.Address]*big.Int) map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big {
	ret := make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	for account, tokenBalances := range balances {
		ret[account] = make(map[gethcommon.Address]*hexutil.Big)
		for token, balance := range tokenBalances {
			ret[account][token] = (*hexutil.Big)(balance)
		}
	}
	return ret
}
