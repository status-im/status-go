package wallet

import (
	"context"
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
func (api *API) GetTransfers(ctx context.Context, start, end *hexutil.Big) ([]TransferView, error) {
	log.Info("call to get transfers", "start", start, "end", end)
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
	return castToTransferViews(rst), nil
}

// GetTransfersByAddress returns transfers for a single address between two blocks.
func (api *API) GetTransfersByAddress(ctx context.Context, address common.Address, start, end *hexutil.Big) ([]TransferView, error) {
	log.Info("call to get transfers for an address", "address", address, "start", start, "end", end)
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
	return castToTransferViews(rst), nil
}

type StatsView struct {
	BlocksStats    map[int64]int64 `json:"blocksStats"`
	TransfersCount int64           `json:"transfersCount"`
}

// GetTransfersStats
func (api *API) GetTransfersStats(ctx context.Context, address common.Address) (*StatsView, error) {
	blocksStats, transfersCount := api.s.db.GetTransfersStats(address)

	return &StatsView{
		BlocksStats:    blocksStats,
		TransfersCount: transfersCount,
	}, nil
}

// GetTransfersByAddressPage returns transfers for a single address between two blocks.
func (a *API) GetTransfersByAddressPage(ctx context.Context, address common.Address, page, pageSize *hexutil.Big) ([]TransferView, error) {
	log.Info("call to get transfers for an address", "address", address, "page", page, "pageSize", pageSize)
	if a.s.db == nil {
		return nil, ErrServiceNotInitialized
	}
	rst, err := a.s.db.GetTransfersByAddressAndPage(address, page.ToInt().Int64(), pageSize.ToInt().Int64())
	if err != nil {
		return nil, err
	}

	transfersCount := big.NewInt(int64(len(rst)))
	if pageSize.ToInt().Cmp(transfersCount) == 1 {
		log.Info("page size is bigger than number of fetched transfers, try to fetch more transfers")

		ctxt := context.Background()

		block, err := a.s.db.GetFirstKnownBlock(address, ethTransfer)
		if err != nil {
			return nil, err
		}
		from, err := findFirstRange(ctxt, address, block, a.s.client)
		fromByAddress := map[common.Address]*big.Int{address: from}
		toByAddress := map[common.Address]*big.Int{address: block}

		balanceCache := newBalanceCache()
		blocksCommand := &findAndCheckBlockRangeCommand{
			accounts:      []common.Address{address},
			db:            a.s.db,
			chain:         a.s.reactor.chain,
			client:        a.s.client,
			balanceCache:  balanceCache,
			feed:          a.s.feed,
			fromByAddress: fromByAddress,
			toByAddress:   toByAddress,
		}

		err = blocksCommand.Command()(ctxt)
		if err != nil {
			return nil, err
		}

		blocks, err := a.s.db.GetBlocksByAddress(address, 40)
		if err != nil {
			log.Info("hm error", "error", err)
			return nil, err
		}
		log.Info("checking blocks again", "blocks", len(blocks))
		if len(blocks) > 0 {
			log.Info("start loadTransfersCommand")
			txCommand := &loadTransfersCommand{
				accounts: []common.Address{address},
				db:       a.s.db,
				chain:    a.s.reactor.chain,
				client:   a.s.client,
			}

			err = txCommand.Command()(ctxt)
			if err != nil {
				return nil, err
			}
			rst, err = a.s.db.GetTransfersByAddressAndPage(address, page.ToInt().Int64(), pageSize.ToInt().Int64())
			if err != nil {
				return nil, err
			}
		}
	}

	log.Debug("result from database for address", "address", address, "page", page, "pageSize", pageSize, "len", len(rst))
	return castToTransferViews(rst), nil
}

type BlockView struct {
	Block *big.Int `json:"number"`
}

func castToBlockViews(blocks []*big.Int) []BlockView {
	views := make([]BlockView, len(blocks))
	for i, block := range blocks {
		view := BlockView{}
		view.Block = block
		views[i] = view
	}
	return views
}

// GetBlocksByAddress returns transfers for a single address between two blocks.
func (api *API) GetBlocksByAddress(ctx context.Context, address common.Address) ([]BlockView, error) {
	if api.s.db == nil {
		return nil, ErrServiceNotInitialized
	}
	rst, err := api.s.db.GetBlocksByAddress(address, 10)
	if err != nil {
		return nil, err
	}
	log.Info("result blocks from database for address", "address", address, "len", len(rst))
	return castToBlockViews(rst), nil
}

// GetTokensBalances return mapping of token balances for every account.
func (api *API) GetTokensBalances(ctx context.Context, accounts, tokens []common.Address) (map[common.Address]map[common.Address]*big.Int, error) {
	if api.s.client == nil {
		return nil, ErrServiceNotInitialized
	}
	return GetTokensBalances(ctx, api.s.client, accounts, tokens)
}

func (api *API) GetCustomTokens(ctx context.Context) ([]*Token, error) {
	log.Debug("call to get custom tokens")
	rst, err := api.s.db.GetCustomTokens()
	log.Debug("result from database for custom tokens", "len", len(rst))
	return rst, err
}

func (api *API) AddCustomToken(ctx context.Context, token Token) error {
	log.Debug("call to create or edit custom token")
	err := api.s.db.AddCustomToken(token)
	log.Debug("result from database for create or edit custom token", "err", err)
	return err
}

func (api *API) DeleteCustomToken(ctx context.Context, address common.Address) error {
	log.Debug("call to remove custom token")
	err := api.s.db.DeleteCustomToken(address)
	log.Debug("result from database for remove custom token", "err", err)
	return err
}
