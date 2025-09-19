package api

import (
	"context"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
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
	tokenManager *token.Manager
}

func NewCommunitiesTokenBalanceManager(tm *token.Manager) *CommunitiesTokenBalanceManager {
	return &CommunitiesTokenBalanceManager{tokenManager: tm}
}

func (m *CommunitiesTokenBalanceManager) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (communities.BalancesByChain, error) {
	chainClients, err := m.tokenManager.RPCClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	resp, err := m.tokenManager.GetBalancesByChain(context.Background(), chainClients, accounts, tokenAddresses)
	return resp, err
}

func (m *CommunitiesTokenBalanceManager) GetCachedBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (communities.BalancesByChain, error) {
	resp, err := m.tokenManager.GetCachedBalancesByChain(accounts, tokenAddresses, chainIDs)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
