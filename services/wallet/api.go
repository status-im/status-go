package wallet

import (
	"context"
	"math/big"
)

type API struct {
	db *Database
}

// GetTransfers returns transfers in range of blocks. If `end` is nil all transfers from `start` will be returned.
func (api *API) GetTransfers(ctx context.Context, start, end *big.Int) ([]Transfer, error) {
	return api.db.GetTransfers(start, end)
}
