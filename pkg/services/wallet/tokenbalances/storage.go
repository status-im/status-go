package tokenbalances

//go:generate go tool mockgen -package=mock_tokenbalances -source=storage.go -destination=mock/storage/storage.go

import (
	"context"
	"math/big"

	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

type Storage interface {
	// GetBalances returns the balances for the given tokens and account addresses, grouped by chainID.
	// A token is absent when its (chainID, account) was never fetched, and present with a nil
	// balance when it was fetched but the fetch did not answer for that token.
	GetBalances(ctx context.Context, tokens []*tokentypes.Token, accountAddresses []AccountAddress) (map[uint64]map[AccountAddress]map[ContractAddress]*big.Int, error)
}
