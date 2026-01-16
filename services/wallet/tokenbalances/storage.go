package tokenbalances

//go:generate go tool mockgen -package=mock_tokenbalances -source=storage.go -destination=mock/storage.go

import (
	"context"
	"math/big"

	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

type State = multistandardbalance.State

type Storage interface {
	// GetBalances returns the balances for the given tokens and account addresses, grouped by chainID.
	GetBalances(ctx context.Context, tokens []*tokentypes.Token, accountAddresses []AccountAddress) (map[uint64]map[AccountAddress]map[ContractAddress]*big.Int, error)
	// GetAllBalancesForAccountAndChain returns the balances of all fetched tokens for the given account and chainID.
	GetAllBalancesForAccountAndChain(ctx context.Context, accountAddress AccountAddress, chainID uint64) (map[ContractAddress]*big.Int, State, error)
}
