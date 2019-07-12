package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// ErrServiceNotInitialized returned when wallet is not initialized/started,.
	ErrServiceNotInitialized = errors.New("wallet service is not initialized")
)

func NewAPI(s *Service) *API {
	return &API{s}
}

// API is class with methods available over RPC.
type API struct {
	s *Service
}

// GetTransfers returns transfers in range of blocks. If `end` is nil all transfers from `start` will be returned.
// TODO(dshulyak) benchmark loading many transfers from database. We can avoid json unmarshal/marshal if we will
// read header, tx and receipt as a raw json.
func (api *API) GetTransfers(ctx context.Context, start, end *hexutil.Big) ([]Transfer, error) {
	log.Debug("call to get transfers", "start", start, "end", end)
	if start == nil {
		return nil, errors.New("start of the query must be provided. use 0 if you want to load all transfers")
	}
	if api.s.db == nil {
		return nil, ErrServiceNotInitialized
	}
	rst, err := api.s.db.GetTransfers((*big.Int)(start), (*big.Int)(end))
	if err != nil {
		return nil, err
	}
	log.Debug("result from database for transfers", "start", start, "end", end, "len", len(rst))
	return rst, nil
}

// GetTransfersByAddress returns transfers for a single address between two blocks.
func (api *API) GetTransfersByAddress(ctx context.Context, address common.Address, start, end *hexutil.Big) ([]Transfer, error) {
	log.Debug("call to get transfers for an address", "address", address, "start", start, "end", end)
	if start == nil {
		return nil, errors.New("start of the query must be provided. use 0 if you want to load all transfers")
	}
	if api.s.db == nil {
		return nil, ErrServiceNotInitialized
	}
	rst, err := api.s.db.GetTransfersByAddress(address, (*big.Int)(start), (*big.Int)(end))
	if err != nil {
		return nil, err
	}
	log.Debug("result from database for address", "address", address, "start", start, "end", end, "len", len(rst))
	return rst, nil
}

// GetTokensBalances return mapping of token balances for every account.
func (api *API) GetTokensBalances(ctx context.Context, accounts, tokens []common.Address) (map[common.Address]map[common.Address]*big.Int, error) {
	if api.s.client == nil {
		return nil, ErrServiceNotInitialized
	}
	return GetTokensBalances(ctx, api.s.client, accounts, tokens)
}

func (api *API) AddBrowser(ctx context.Context, browser Browser) error {
	if api.s.db == nil {
		return ErrServiceNotInitialized
	}
	return api.s.db.InsertBrowser(browser)
}

func (api *API) GetBrowsers(ctx context.Context) ([]Browser, error) {
	if api.s.db == nil {
		return nil, ErrServiceNotInitialized
	}
	return api.s.db.GetBrowsers()
}

func (api *API) GetBrowsersTransit(ctx context.Context) (json.RawMessage, error) {
	browsers, err := api.GetBrowsers(ctx)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	enc := NewEncoder(buf)
	err = enc.Encode(browsers)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(buf.Bytes()), nil

}

func (api *API) DeleteBrowser(ctx context.Context, id string) error {
	if api.s.db == nil {
		return ErrServiceNotInitialized
	}
	return api.s.db.DeleteBrowser(id)
}
