package wallet

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
)

// API is class with methods available over RPC.
type API struct {
	s *Service
}

// GetTransfers returns transfers in range of blocks. If `end` is nil all transfers from `start` will be returned.
// TODO(dshulyak) benchmark loading many transfers from database. We can avoid json unmarshal/marshal if we will
// read header, tx and receipt as a raw json.
func (api *API) GetTransfers(ctx context.Context, start, end *big.Int) ([]Transfer, error) {
	log.Debug("call to get transfers", "start", start, "end", end)
	if api.s.db == nil {
		return nil, errors.New("wallet service is not initialized")
	}
	rst, err := api.s.db.GetTransfers(start, end)
	if err != nil {
		return nil, err
	}
	log.Debug("result from database for transfers", "start", start, "end", end, "len", len(rst))
	return rst, nil
}
