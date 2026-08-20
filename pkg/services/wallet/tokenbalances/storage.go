package tokenbalances

//go:generate go tool mockgen -package=mock_tokenbalances -source=storage.go -destination=mock/storage/storage.go

import (
	"context"
	"math/big"

	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

type Storage interface {
	// GetBalances returns the balances for the given tokens and account addresses, grouped by chainID.
	GetBalances(ctx context.Context, tokens []*tokentypes.Token, accountAddresses []AccountAddress) (map[uint64]map[AccountAddress]map[ContractAddress]*big.Int, error)
}
