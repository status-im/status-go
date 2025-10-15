package tokenbalances

//go:generate go tool mockgen -package=mock_tokenbalances -source=storage.go -destination=mock/storage/storage.go

import (
	"context"
	"math/big"
)

type Storage interface {
	GetBalances(ctx context.Context, chainID uint64, tokenAddresses []ContractAddress, accountAddresses []AccountAddress) (map[AccountAddress]map[ContractAddress]*big.Int, error)
}
