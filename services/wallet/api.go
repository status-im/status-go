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

// GetTransfersByAddress returns transfers for a single address
func (api *API) GetTransfersByAddress(ctx context.Context, address common.Address, toBlock, limit *hexutil.Big) ([]TransferView, error) {
	log.Debug("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address, "block", toBlock, "limit", limit)
	if api.s.db == nil {
		log.Error("[WalletAPI:: GetTransfersByAddress] db is not initialized")
		return nil, ErrServiceNotInitialized
	}

	var toBlockBN *big.Int
	if toBlock != nil {
		toBlockBN = toBlock.ToInt()
	}

	rst, err := api.s.db.GetTransfersByAddress(address, toBlockBN, limit.ToInt().Int64())
	if err != nil {
		log.Error("[WalletAPI:: GetTransfersByAddress] can't fetch transfers", "err", err)
		return nil, err
	}

	transfersCount := big.NewInt(int64(len(rst)))
	if limit.ToInt().Cmp(transfersCount) == 1 {
		block, err := api.s.db.GetFirstKnownBlock(address)
		if err != nil {
			return nil, err
		}

		if block == nil {
			return castToTransferViews(rst), nil
		}

		from, err := findFirstRange(ctx, address, block, api.s.client)
		if err != nil {
			return nil, err
		}
		fromByAddress := map[common.Address]*big.Int{address: from}
		toByAddress := map[common.Address]*big.Int{address: block}

		balanceCache := newBalanceCache()
		blocksCommand := &findAndCheckBlockRangeCommand{
			accounts:      []common.Address{address},
			db:            api.s.db,
			chain:         api.s.reactor.chain,
			client:        api.s.client,
			balanceCache:  balanceCache,
			feed:          api.s.feed,
			fromByAddress: fromByAddress,
			toByAddress:   toByAddress,
		}

		if err = blocksCommand.Command()(ctx); err != nil {
			return nil, err
		}

		blocks, err := api.s.db.GetBlocksByAddress(address, numberOfBlocksCheckedPerIteration)
		if err != nil {
			return nil, err
		}

		log.Info("checking blocks again", "blocks", len(blocks))
		if len(blocks) > 0 {
			txCommand := &loadTransfersCommand{
				accounts: []common.Address{address},
				db:       api.s.db,
				chain:    api.s.reactor.chain,
				client:   api.s.client,
			}

			err = txCommand.Command()(ctx)
			if err != nil {
				return nil, err
			}
			rst, err = api.s.db.GetTransfersByAddress(address, toBlockBN, limit.ToInt().Int64())
			if err != nil {
				return nil, err
			}
		}
	}

	return castToTransferViews(rst), nil
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
