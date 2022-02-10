package wallet

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/chain"
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
	return api.s.transferController.SetInitialBlocksRange([]uint64{api.s.rpcClient.UpstreamChainID})
}

func (api *API) SetInitialBlocksRangeForChainIDs(ctx context.Context, chainIDs []uint64) error {
	return api.s.transferController.SetInitialBlocksRange(chainIDs)
}

func (api *API) CheckRecentHistory(ctx context.Context, addresses []common.Address) error {
	return api.s.transferController.CheckRecentHistory([]uint64{api.s.rpcClient.UpstreamChainID}, addresses)
}

func (api *API) CheckRecentHistoryForChainIDs(ctx context.Context, chainIDs []uint64, addresses []common.Address) error {
	return api.s.transferController.CheckRecentHistory(chainIDs, addresses)
}

// GetTransfersByAddress returns transfers for a single address
func (api *API) GetTransfersByAddress(ctx context.Context, address common.Address, toBlock, limit *hexutil.Big, fetchMore bool) ([]transfer.View, error) {
	log.Debug("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address)
	return api.s.transferController.GetTransfersByAddress(ctx, api.s.rpcClient.UpstreamChainID, address, toBlock, limit, fetchMore)
}

// LoadTransferByHash loads transfer to the database
func (api *API) LoadTransferByHash(ctx context.Context, address common.Address, hash common.Hash) error {
	log.Debug("[WalletAPI:: LoadTransferByHash] get transfer by hash", "address", address, "hash", hash)
	return api.s.transferController.LoadTransferByHash(ctx, api.s.rpcClient, address, hash)
}

func (api *API) GetTransfersByAddressAndChainID(ctx context.Context, chainID uint64, address common.Address, toBlock, limit *hexutil.Big, fetchMore bool) ([]transfer.View, error) {
	log.Debug("[WalletAPI:: GetTransfersByAddressAndChainID] get transfers for an address", "address", address)
	return api.s.transferController.GetTransfersByAddress(ctx, chainID, address, toBlock, limit, fetchMore)
}

func (api *API) GetCachedBalances(ctx context.Context, addresses []common.Address) ([]transfer.LastKnownBlockView, error) {
	return api.s.transferController.GetCachedBalances(ctx, api.s.rpcClient.UpstreamChainID, addresses)
}

func (api *API) GetCachedBalancesbyChainID(ctx context.Context, chainID uint64, addresses []common.Address) ([]transfer.LastKnownBlockView, error) {
	return api.s.transferController.GetCachedBalances(ctx, chainID, addresses)
}

// GetTokensBalances return mapping of token balances for every account.
func (api *API) GetTokensBalances(ctx context.Context, accounts, addresses []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	chainClient, err := chain.NewLegacyClient(api.s.rpcClient)
	if err != nil {
		return nil, err
	}
	return api.s.tokenManager.getBalances(ctx, []*chain.Client{chainClient}, accounts, addresses)
}

func (api *API) GetTokensBalancesForChainIDs(ctx context.Context, chainIDs []uint64, accounts, addresses []common.Address) (map[common.Address]map[common.Address]*hexutil.Big, error) {
	clients, err := chain.NewClients(api.s.rpcClient, chainIDs)
	if err != nil {
		return nil, err
	}
	return api.s.tokenManager.getBalances(ctx, clients, accounts, addresses)
}

func (api *API) GetTokens(ctx context.Context, chainID uint64) ([]*Token, error) {
	log.Debug("call to get tokens")
	rst, err := api.s.tokenManager.getTokens(chainID)
	log.Debug("result from token store", "len", len(rst))
	return rst, err
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
		token.ChainID = api.s.rpcClient.UpstreamChainID
	}
	err := api.s.tokenManager.upsertCustom(token)
	log.Debug("result from database for create or edit custom token", "err", err)
	return err
}

func (api *API) DeleteCustomToken(ctx context.Context, address common.Address) error {
	log.Debug("call to remove custom token")
	err := api.s.tokenManager.deleteCustom(api.s.rpcClient.UpstreamChainID, address)
	log.Debug("result from database for remove custom token", "err", err)
	return err
}

func (api *API) DeleteCustomTokenByChainID(ctx context.Context, chainID uint64, address common.Address) error {
	log.Debug("call to remove custom token")
	err := api.s.tokenManager.deleteCustom(chainID, address)
	log.Debug("result from database for remove custom token", "err", err)
	return err
}

func (api *API) GetSavedAddresses(ctx context.Context) ([]*SavedAddress, error) {
	log.Debug("call to get saved addresses")
	rst, err := api.s.savedAddressesManager.GetSavedAddresses(api.s.rpcClient.UpstreamChainID)
	log.Debug("result from database for saved addresses", "len", len(rst))
	return rst, err
}

func (api *API) AddSavedAddress(ctx context.Context, sa SavedAddress) error {
	log.Debug("call to create or edit saved address")
	if sa.ChainID == 0 {
		sa.ChainID = api.s.rpcClient.UpstreamChainID
	}
	err := api.s.savedAddressesManager.AddSavedAddress(sa)
	log.Debug("result from database for create or edit saved address", "err", err)
	return err
}

func (api *API) DeleteSavedAddress(ctx context.Context, address common.Address) error {
	log.Debug("call to remove saved address")
	err := api.s.savedAddressesManager.DeleteSavedAddress(api.s.rpcClient.UpstreamChainID, address)
	log.Debug("result from database for remove saved address", "err", err)
	return err
}

func (api *API) GetPendingTransactions(ctx context.Context) ([]*PendingTransaction, error) {
	log.Debug("call to get pending transactions")
	rst, err := api.s.transactionManager.getAllPendings(api.s.rpcClient.UpstreamChainID)
	log.Debug("result from database for pending transactions", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingTransactionsByChainID(ctx context.Context, chainID uint64) ([]*PendingTransaction, error) {
	log.Debug("call to get pending transactions")
	rst, err := api.s.transactionManager.getAllPendings(chainID)
	log.Debug("result from database for pending transactions", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingOutboundTransactionsByAddress(ctx context.Context, address common.Address) ([]*PendingTransaction, error) {
	log.Debug("call to get pending outbound transactions by address")
	rst, err := api.s.transactionManager.getPendingByAddress(api.s.rpcClient.UpstreamChainID, address)
	log.Debug("result from database for pending transactions by address", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingOutboundTransactionsByAddressAndChainID(ctx context.Context, chainID uint64, address common.Address) ([]*PendingTransaction, error) {
	log.Debug("call to get pending outbound transactions by address")
	rst, err := api.s.transactionManager.getPendingByAddress(chainID, address)
	log.Debug("result from database for pending transactions by address", "len", len(rst))
	return rst, err
}

func (api *API) StorePendingTransaction(ctx context.Context, trx PendingTransaction) error {
	log.Debug("call to create or edit pending transaction")
	if trx.ChainID == 0 {
		trx.ChainID = api.s.rpcClient.UpstreamChainID
	}
	err := api.s.transactionManager.addPending(trx)
	log.Debug("result from database for creating or editing a pending transaction", "err", err)
	return err
}

func (api *API) DeletePendingTransaction(ctx context.Context, transactionHash common.Hash) error {
	log.Debug("call to remove pending transaction")
	err := api.s.transactionManager.deletePending(api.s.rpcClient.UpstreamChainID, transactionHash)
	log.Debug("result from database for remove pending transaction", "err", err)
	return err
}

func (api *API) DeletePendingTransactionByChainID(ctx context.Context, chainID uint64, transactionHash common.Hash) error {
	log.Debug("call to remove pending transaction")
	err := api.s.transactionManager.deletePending(chainID, transactionHash)
	log.Debug("result from database for remove pending transaction", "err", err)
	return err
}

func (api *API) WatchTransaction(ctx context.Context, transactionHash common.Hash) error {
	chainClient, err := chain.NewLegacyClient(api.s.rpcClient)
	if err != nil {
		return err
	}
	return api.s.transactionManager.watch(ctx, transactionHash, chainClient)
}

func (api *API) WatchTransactionByChainID(ctx context.Context, chainID uint64, transactionHash common.Hash) error {
	chainClient, err := chain.NewClient(api.s.rpcClient, chainID)
	if err != nil {
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

func (api *API) GetOpenseaCollectionsByOwner(ctx context.Context, chainID uint64, owner common.Address) ([]OpenseaCollection, error) {
	log.Debug("call to get opensea collections")
	client, err := newOpenseaClient(chainID, api.s.openseaAPIKey)
	if err != nil {
		return nil, err
	}

	return client.fetchAllCollectionsByOwner(owner)
}

func (api *API) GetOpenseaAssetsByOwnerAndCollection(ctx context.Context, chainID uint64, owner common.Address, collectionSlug string, limit int) ([]OpenseaAsset, error) {
	log.Debug("call to get opensea assets")
	client, err := newOpenseaClient(chainID, api.s.openseaAPIKey)
	if err != nil {
		return nil, err
	}

	return client.fetchAllAssetsByOwnerAndCollection(owner, collectionSlug, limit)
}

func (api *API) AddEthereumChain(ctx context.Context, network params.Network) error {
	log.Debug("call to AddEthereumChain")
	return api.s.rpcClient.NetworkManager.Upsert(&network)
}

func (api *API) DeleteEthereumChain(ctx context.Context, chainID uint64) error {
	log.Debug("call to DeleteEthereumChain")
	return api.s.rpcClient.NetworkManager.Delete(chainID)
}

func (api *API) GetEthereumChains(ctx context.Context, onlyEnabled bool) ([]*params.Network, error) {
	log.Debug("call to GetEthereumChains")
	return api.s.rpcClient.NetworkManager.Get(onlyEnabled)
}
