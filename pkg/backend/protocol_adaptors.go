package backend

import (
	"context"
	"fmt"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/pkg/services/networks"
	"github.com/status-im/status-go/pkg/services/wallet/token"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/tokenbalances"
)

var _ communities.NetworkManager = (*CommunitiesNetworkManager)(nil)
var _ communities.TokenManager = (*CommunitiesTokenManager)(nil)
var _ communities.TokenBalanceManager = (*CommunitiesTokenBalanceManager)(nil)

type CommunitiesNetworkManager struct {
	networkManager *networks.Manager
}

func NewCommunitiesNetworkManager(nm *networks.Manager) *CommunitiesNetworkManager {
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

func (m *CommunitiesTokenManager) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address gethcommon.Address) (*tokentypes.Token, error) {
	return m.tokenManager.FindOrCreateTokenByAddress(ctx, chainID, address)
}

type CommunitiesTokenBalanceManager struct {
	tokenBalancesFetcher tokenbalances.FetcherIface
	tokenBalancesStorage tokenbalances.Storage
}

func NewCommunitiesTokenBalanceManager(f tokenbalances.FetcherIface, s tokenbalances.Storage) *CommunitiesTokenBalanceManager {
	return &CommunitiesTokenBalanceManager{tokenBalancesFetcher: f, tokenBalancesStorage: s}
}

func (m *CommunitiesTokenBalanceManager) GetBalancesByChain(ctx context.Context, accounts []gethcommon.Address, tokens []*tokentypes.Token) (communities.BalancesByChain, error) {
	if m.tokenBalancesFetcher == nil {
		return nil, fmt.Errorf("tokenBalancesFetcher is nil")
	}

	tokenAddressesPerChain := make(map[uint64][]gethcommon.Address)
	for _, token := range tokens {
		tokenAddressesPerChain[token.ChainID] = append(tokenAddressesPerChain[token.ChainID], token.Address)
	}

	ret := make(communities.BalancesByChain)
	for chainID, tokenAddresses := range tokenAddressesPerChain {
		ret[chainID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
		balances, err := m.tokenBalancesFetcher.Fetch(ctx, chainID, tokenAddresses, accounts)
		if err != nil {
			return nil, err
		}
		ret[chainID] = balancesToCommunitiesBalances(balances)
	}
	return ret, nil
}

func (m *CommunitiesTokenBalanceManager) GetCachedBalancesByChain(ctx context.Context, accounts []gethcommon.Address, tokens []*tokentypes.Token) (communities.BalancesByChain, error) {
	if m.tokenBalancesStorage == nil {
		return nil, fmt.Errorf("tokenBalancesStorage is nil")
	}

	balances, err := m.tokenBalancesStorage.GetBalances(ctx, tokens, accounts)
	if err != nil {
		return nil, err
	}

	return balancesPerChainToCommunitiesBalances(balances), nil
}

func balancesPerChainToCommunitiesBalances(balances map[uint64]map[gethcommon.Address]map[gethcommon.Address]*big.Int) map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big {
	ret := make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	for chainID, tokenBalances := range balances {
		ret[chainID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
		for account, tokenBalances := range tokenBalances {
			ret[chainID][account] = make(map[gethcommon.Address]*hexutil.Big)
			for token, balance := range tokenBalances {
				ret[chainID][account][token] = (*hexutil.Big)(balance)
			}
		}
	}
	return ret
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
