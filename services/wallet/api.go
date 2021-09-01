package wallet

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

func NewAPI(s *Service) *API {
	return &API{s}
}

// API is class with methods available over RPC.
type API struct {
	s *Service
}

type LastKnownBlockView struct {
	Address common.Address `json:"address"`
	Number  *big.Int       `json:"blockNumber"`
	Balance BigInt         `json:"balance"`
	Nonce   *int64         `json:"nonce"`
}

func blocksToViews(blocks map[common.Address]*LastKnownBlock) []LastKnownBlockView {
	blocksViews := []LastKnownBlockView{}
	for address, block := range blocks {
		view := LastKnownBlockView{
			Address: address,
			Number:  block.Number,
			Balance: BigInt{block.Balance},
			Nonce:   block.Nonce,
		}
		blocksViews = append(blocksViews, view)
	}

	return blocksViews
}

// SetInitialBlocksRange sets initial blocks range
func (api *API) SetInitialBlocksRange(ctx context.Context) error {
	return api.s.SetInitialBlocksRange(api.s.legacyChainID)
}

func (api *API) CheckRecentHistory(ctx context.Context, addresses []common.Address) error {
	if len(addresses) == 0 {
		log.Info("no addresses provided")
		return nil
	}
	err := api.s.MergeBlocksRanges(addresses, api.s.legacyChainID)
	if err != nil {
		return err
	}

	return api.s.StartReactor(
		addresses,
		api.s.legacyChainID,
	)
}

// GetTransfersByAddress returns transfers for a single address
func (api *API) GetTransfersByAddress(ctx context.Context, address common.Address, toBlock, limit *hexutil.Big, fetchMore bool) ([]TransferView, error) {
	log.Debug("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address, "block", toBlock, "limit", limit)
	var toBlockBN *big.Int
	if toBlock != nil {
		toBlockBN = toBlock.ToInt()
	}

	rst, err := api.s.db.GetTransfersByAddress(api.s.legacyChainID, address, toBlockBN, limit.ToInt().Int64())
	if err != nil {
		log.Error("[WalletAPI:: GetTransfersByAddress] can't fetch transfers", "err", err)
		return nil, err
	}

	transfersCount := big.NewInt(int64(len(rst)))
	chainClient, err := api.s.networkManager.getChainClient(api.s.legacyChainID)
	if err == nil {
		return nil, err
	}
	if fetchMore && limit.ToInt().Cmp(transfersCount) == 1 {
		block, err := api.s.db.GetFirstKnownBlock(api.s.legacyChainID, address)
		if err != nil {
			return nil, err
		}

		// if zero block was already checked there is nothing to find more
		if block == nil || big.NewInt(0).Cmp(block) == 0 {
			return castToTransferViews(rst), nil
		}

		from, err := findFirstRange(ctx, address, block, chainClient)
		if err != nil {
			if nonArchivalNodeError(err) {
				api.s.feed.Send(Event{
					Type: EventNonArchivalNodeDetected,
				})
				from = big.NewInt(0).Sub(block, big.NewInt(100))
			} else {
				log.Error("first range error", "error", err)
				return nil, err
			}
		}
		fromByAddress := map[common.Address]*LastKnownBlock{address: {
			Number: from,
		}}
		toByAddress := map[common.Address]*big.Int{address: block}

		balanceCache := newBalanceCache()
		blocksCommand := &findAndCheckBlockRangeCommand{
			accounts:      []common.Address{address},
			db:            api.s.db,
			chain:         api.s.reactor.chain,
			client:        chainClient,
			balanceCache:  balanceCache,
			feed:          api.s.feed,
			fromByAddress: fromByAddress,
			toByAddress:   toByAddress,
		}

		if err = blocksCommand.Command()(ctx); err != nil {
			return nil, err
		}

		blocks, err := api.s.db.GetBlocksByAddress(api.s.legacyChainID, address, numberOfBlocksCheckedPerIteration)
		if err != nil {
			return nil, err
		}

		log.Info("checking blocks again", "blocks", len(blocks))
		if len(blocks) > 0 {
			txCommand := &loadTransfersCommand{
				accounts: []common.Address{address},
				db:       api.s.db,
				chain:    api.s.reactor.chain,
				client:   chainClient,
			}

			err = txCommand.Command()(ctx)
			if err != nil {
				return nil, err
			}
			rst, err = api.s.db.GetTransfersByAddress(api.s.legacyChainID, address, toBlockBN, limit.ToInt().Int64())
			if err != nil {
				return nil, err
			}
		}
	}

	return castToTransferViews(rst), nil
}

func (api *API) GetCachedBalances(ctx context.Context, addresses []common.Address) ([]LastKnownBlockView, error) {
	result, error := api.s.db.getLastKnownBalances(api.s.legacyChainID, addresses)
	if error != nil {
		return nil, error
	}

	return blocksToViews(result), nil
}

// GetTokensBalances return mapping of token balances for every account.
func (api *API) GetTokensBalances(ctx context.Context, accounts, addresses []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	client, err := api.s.networkManager.getChainClient(api.s.legacyChainID)
	if err == nil {
		return nil, err
	}
	return api.s.tokenManager.getBalances(ctx, []*chainClient{client}, accounts, addresses)
}

func (api *API) GetTokensBalances2(ctx context.Context, chainIDs []uint64, accounts, addresses []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	var clients []*chainClient
	for _, chainID := range chainIDs {
		client, err := api.s.networkManager.getChainClient(chainID)
		if err == nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return api.s.tokenManager.getBalances(ctx, clients, accounts, addresses)
}

func (api *API) GetCustomTokens(ctx context.Context) ([]*Token, error) {
	log.Debug("call to get custom tokens")
	rst, err := api.s.tokenManager.getCustoms()
	log.Debug("result from database for custom tokens", "len", len(rst))
	return rst, err
}

func (api *API) AddCustomToken(ctx context.Context, token Token) error {
	log.Debug("call to create or edit custom token")
	if token.ChainID == 0 {
		token.ChainID = api.s.legacyChainID
	}
	err := api.s.tokenManager.upsertCustom(token)
	log.Debug("result from database for create or edit custom token", "err", err)
	return err
}

func (api *API) DeleteCustomToken(ctx context.Context, address common.Address) error {
	log.Debug("call to remove custom token")
	err := api.s.tokenManager.deleteCustom(api.s.legacyChainID, address)
	log.Debug("result from database for remove custom token", "err", err)
	return err
}

func (api *API) DeleteCustomToken2(ctx context.Context, chainID uint64, address common.Address) error {
	log.Debug("call to remove custom token")
	err := api.s.tokenManager.deleteCustom(chainID, address)
	log.Debug("result from database for remove custom token", "err", err)
	return err
}

func (api *API) GetPendingTransactions(ctx context.Context) ([]*PendingTransaction, error) {
	log.Debug("call to get pending transactions")
	rst, err := api.s.transactionManager.getAllPendings(api.s.legacyChainID)
	log.Debug("result from database for pending transactions", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingTransactions2(ctx context.Context, chainID uint64) ([]*PendingTransaction, error) {
	log.Debug("call to get pending transactions")
	rst, err := api.s.transactionManager.getAllPendings(chainID)
	log.Debug("result from database for pending transactions", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingOutboundTransactionsByAddress(ctx context.Context, address common.Address) ([]*PendingTransaction, error) {
	log.Debug("call to get pending outbound transactions by address")
	rst, err := api.s.transactionManager.getPendingByAddress(api.s.legacyChainID, address)
	log.Debug("result from database for pending transactions by address", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingOutboundTransactionsByAddress2(ctx context.Context, chainID uint64, address common.Address) ([]*PendingTransaction, error) {
	log.Debug("call to get pending outbound transactions by address")
	rst, err := api.s.transactionManager.getPendingByAddress(chainID, address)
	log.Debug("result from database for pending transactions by address", "len", len(rst))
	return rst, err
}

func (api *API) StorePendingTransaction(ctx context.Context, trx PendingTransaction) error {
	log.Debug("call to create or edit pending transaction")
	if trx.ChainID == 0 {
		trx.ChainID = api.s.legacyChainID
	}
	err := api.s.transactionManager.addPending(trx)
	log.Debug("result from database for creating or editing a pending transaction", "err", err)
	return err
}

func (api *API) DeletePendingTransaction(ctx context.Context, transactionHash common.Hash) error {
	log.Debug("call to remove pending transaction")
	err := api.s.transactionManager.deletePending(api.s.legacyChainID, transactionHash)
	log.Debug("result from database for remove pending transaction", "err", err)
	return err
}

func (api *API) DeletePendingTransaction2(ctx context.Context, chainID uint64, transactionHash common.Hash) error {
	log.Debug("call to remove pending transaction")
	err := api.s.transactionManager.deletePending(chainID, transactionHash)
	log.Debug("result from database for remove pending transaction", "err", err)
	return err
}

func (api *API) WatchTransaction(ctx context.Context, transactionHash common.Hash) error {
	chainClient, err := api.s.networkManager.getChainClient(api.s.legacyChainID)
	if err == nil {
		return err
	}

	return api.s.transactionManager.watch(ctx, transactionHash, chainClient, api.s.feed)
}

func (api *API) WatchTransaction2(ctx context.Context, chainID uint64, transactionHash common.Hash) error {
	chainClient, err := api.s.networkManager.getChainClient(chainID)
	if err == nil {
		return err
	}

	return api.s.transactionManager.watch(ctx, transactionHash, chainClient, api.s.feed)
}

func (api *API) GetFavourites(ctx context.Context) ([]*Favourite, error) {
	log.Debug("call to get favourites")
	rst, err := api.s.db.GetFavourites()
	log.Debug("result from database for favourites", "len", len(rst))
	return rst, err
}

func (api *API) AddFavourite(ctx context.Context, favourite Favourite) error {
	log.Debug("call to create or update favourites")
	err := api.s.db.AddFavourite(favourite)
	log.Debug("result from database for create or update favourites", "err", err)
	return err
}

func (api *API) GetCryptoOnRamps(ctx context.Context) ([]CryptoOnRamp, error) {
	return api.s.cryptoOnRampManager.Get()
}

func (api *API) GetOpenseaCollectionsByOwner(ctx context.Context, owner common.Address) ([]OpenseaCollection, error) {
	log.Debug("call to get opensea collections")
	return api.s.opensea.fetchAllCollectionsByOwner(owner)
}

func (api *API) GetOpenseaAssetsByOwnerAndCollection(ctx context.Context, owner common.Address, collectionSlug string, limit int) ([]OpenseaAsset, error) {
	log.Debug("call to get opensea assets")
	return api.s.opensea.fetchAllAssetsByOwnerAndCollection(owner, collectionSlug, limit)
}

func (api *API) AddEthereumChain(ctx context.Context, network Network) error {
	log.Debug("call to AddEthereumChain")
	return api.s.networkManager.upsert(&network)
}

func (api *API) DeleteEthereumChain(ctx context.Context, chainID uint64) error {
	log.Debug("call to DeleteEthereumChain")
	return api.s.networkManager.delete(chainID)
}

func (api *API) GetEthereumChains(ctx context.Context, onlyEnabled bool) ([]*Network, error) {
	log.Debug("call to GetEthereumChains")
	return api.s.networkManager.get(onlyEnabled)
}
