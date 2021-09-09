package wallet

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/network"
	"github.com/status-im/status-go/services/wallet/transfer"
)

func NewAPI(s *Service) *API {
	return &API{s}
}

// API is class with methods available over RPC.
type API struct {
	s *Service
}

// SetInitialBlocksRange sets initial blocks range
func (api *API) SetInitialBlocksRange(ctx context.Context) error {
	return api.s.transferController.SetInitialBlocksRange([]uint64{api.s.legacyChainID})
}

func (api *API) SetInitialBlocksRange2(ctx context.Context, chainIDs []uint64) error {
	return api.s.transferController.SetInitialBlocksRange(chainIDs)
}

func (api *API) CheckRecentHistory(ctx context.Context, addresses []common.Address) error {
	return api.s.transferController.CheckRecentHistory([]uint64{api.s.legacyChainID}, addresses)
}

func (api *API) CheckRecentHistory2(ctx context.Context, chainIDs []uint64, addresses []common.Address) error {
	return api.s.transferController.CheckRecentHistory(chainIDs, addresses)
}

// GetTransfersByAddress returns transfers for a single address
func (api *API) GetTransfersByAddress(ctx context.Context, address common.Address, toBlock, limit *hexutil.Big, fetchMore bool) ([]transfer.View, error) {
	log.Debug("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address, "block", toBlock, "limit", limit)
	return api.s.transferController.GetTransfersByAddress(ctx, api.s.legacyChainID, address, toBlock, limit, fetchMore)
}

func (api *API) GetTransfersByAddress2(ctx context.Context, chainID uint64, address common.Address, toBlock, limit *hexutil.Big, fetchMore bool) ([]transfer.View, error) {
	log.Debug("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address, "block", toBlock, "limit", limit)
	return api.s.transferController.GetTransfersByAddress(ctx, chainID, address, toBlock, limit, fetchMore)
}

func (api *API) GetCachedBalances(ctx context.Context, addresses []common.Address) ([]transfer.LastKnownBlockView, error) {
	return api.s.transferController.GetCachedBalances(ctx, api.s.legacyChainID, addresses)
}

func (api *API) GetCachedBalances2(ctx context.Context, chainID uint64, addresses []common.Address) ([]transfer.LastKnownBlockView, error) {
	return api.s.transferController.GetCachedBalances(ctx, chainID, addresses)
}

// GetTokensBalances return mapping of token balances for every account.
func (api *API) GetTokensBalances(ctx context.Context, accounts, addresses []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	client, err := api.s.networkManager.GetChainClient(api.s.legacyChainID)
	if err == nil {
		return nil, err
	}
	return api.s.tokenManager.getBalances(ctx, []*network.ChainClient{client}, accounts, addresses)
}

func (api *API) GetTokensBalances2(ctx context.Context, chainIDs []uint64, accounts, addresses []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	clients, err := api.s.networkManager.GetChainClients(chainIDs)
	if err == nil {
		return nil, err
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
	chainClient, err := api.s.networkManager.GetChainClient(api.s.legacyChainID)
	if err == nil {
		return err
	}

	return api.s.transactionManager.watch(ctx, transactionHash, chainClient)
}

func (api *API) WatchTransaction2(ctx context.Context, chainID uint64, transactionHash common.Hash) error {
	chainClient, err := api.s.networkManager.GetChainClient(chainID)
	if err == nil {
		return err
	}

	return api.s.transactionManager.watch(ctx, transactionHash, chainClient)
}

func (api *API) GetFavourites(ctx context.Context) ([]*Favourite, error) {
	log.Debug("call to get favourites")
	rst, err := api.s.favouriteManager.GetFavourites()
	log.Debug("result from database for favourites", "len", len(rst))
	return rst, err
}

func (api *API) AddFavourite(ctx context.Context, favourite Favourite) error {
	log.Debug("call to create or update favourites")
	err := api.s.favouriteManager.AddFavourite(favourite)
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

func (api *API) AddEthereumChain(ctx context.Context, network network.Network) error {
	log.Debug("call to AddEthereumChain")
	return api.s.networkManager.Upsert(&network)
}

func (api *API) DeleteEthereumChain(ctx context.Context, chainID uint64) error {
	log.Debug("call to DeleteEthereumChain")
	return api.s.networkManager.Delete(chainID)
}

func (api *API) GetEthereumChains(ctx context.Context, onlyEnabled bool) ([]*network.Network, error) {
	log.Debug("call to GetEthereumChains")
	return api.s.networkManager.Get(onlyEnabled)
}
