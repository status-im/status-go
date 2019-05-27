package wallet

import (
	"context"
	"math/big"
)

// API is class with methods available over RPC.
type API struct {
	db *Database
}

// GetTransfers returns transfers in range of blocks. If `end` is nil all transfers from `start` will be returned.
// TODO(dshulyak) benchmark loading many transfers from database. We can avoid json unmarshal/marshal json if we will
// return header, tx and receipt as a raw json.
func (api *API) GetTransfers(ctx context.Context, start, end *big.Int) ([]Transfer, error) {
	return api.db.GetTransfers(start, end)
}
