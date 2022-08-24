package wallet

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/transactions"
)

func NewAPI(s *Service) *API {
	reader := NewReader(s)
	router := NewRouter(s)
	return &API{s, reader, router}
}

// API is class with methods available over RPC.
type API struct {
	s      *Service
	reader *Reader
	router *Router
}

func (api *API) StartWallet(ctx context.Context, chainIDs []uint64) error {
	return api.reader.Start(ctx, chainIDs)
}

func (api *API) GetWallet(ctx context.Context, chainIDs []uint64) (*Wallet, error) {
	return api.reader.GetWallet(ctx, chainIDs)
}

type DerivedAddress struct {
	Address        common.Address `json:"address"`
	Path           string         `json:"path"`
	HasActivity    bool           `json:"hasActivity"`
	AlreadyCreated bool           `json:"alreadyCreated"`
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
	log.Debug("[WalletAPI:: GetTransfersByAddressAndChainIDs] get transfers for an address", "address", address)
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

func (api *API) DiscoverToken(ctx context.Context, chainID uint64, address common.Address) (*Token, error) {
	log.Debug("call to get discover token")
	token, err := api.s.tokenManager.discoverToken(ctx, chainID, address)
	return token, err
}

func (api *API) GetVisibleTokens(chainIDs []uint64) (map[uint64][]*Token, error) {
	log.Debug("call to get visible tokens")
	rst, err := api.s.tokenManager.getVisible(chainIDs)
	log.Debug("result from database for visible tokens", "len", len(rst))
	return rst, err
}

func (api *API) ToggleVisibleToken(ctx context.Context, chainID uint64, address common.Address) (bool, error) {
	log.Debug("call to toggle visible tokens")
	err := api.s.tokenManager.toggle(chainID, address)
	if err != nil {
		return false, err
	}
	return true, nil
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

func (api *API) GetSavedAddresses(ctx context.Context) ([]SavedAddress, error) {
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
	rst, err := api.s.transactionManager.getAllPendings([]uint64{api.s.rpcClient.UpstreamChainID})
	log.Debug("result from database for pending transactions", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingTransactionsByChainIDs(ctx context.Context, chainIDs []uint64) ([]*PendingTransaction, error) {
	log.Debug("call to get pending transactions")
	rst, err := api.s.transactionManager.getAllPendings(chainIDs)
	log.Debug("result from database for pending transactions", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingOutboundTransactionsByAddress(ctx context.Context, address common.Address) ([]*PendingTransaction, error) {
	log.Debug("call to get pending outbound transactions by address")
	rst, err := api.s.transactionManager.getPendingByAddress([]uint64{api.s.rpcClient.UpstreamChainID}, address)
	log.Debug("result from database for pending transactions by address", "len", len(rst))
	return rst, err
}

func (api *API) GetPendingOutboundTransactionsByAddressAndChainID(ctx context.Context, chainIDs []uint64, address common.Address) ([]*PendingTransaction, error) {
	log.Debug("call to get pending outbound transactions by address")
	rst, err := api.s.transactionManager.getPendingByAddress(chainIDs, address)
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

func (api *API) GetFavourites(ctx context.Context) ([]Favourite, error) {
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

func (api *API) FetchPrices(ctx context.Context, symbols []string, currency string) (map[string]float64, error) {
	log.Debug("call to FetchPrices")
	return fetchCryptoComparePrices(symbols, currency)
}

func (api *API) FetchMarketValues(ctx context.Context, symbols []string, currency string) (map[string]MarketCoinValues, error) {
	log.Debug("call to FetchMarketValues")
	return fetchTokenMarketValues(symbols, currency)
}

func (api *API) FetchTokenDetails(ctx context.Context, symbols []string) (map[string]Coin, error) {
	log.Debug("call to FetchTokenDetails")
	return fetchCryptoCompareTokenDetails(symbols)
}

func (api *API) GetSuggestedFees(ctx context.Context, chainID uint64) (*SuggestedFees, error) {
	log.Debug("call to GetSuggestedFees")
	return api.s.feesManager.suggestedFees(ctx, chainID)
}

func (api *API) GetTransactionEstimatedTime(ctx context.Context, chainID uint64, maxFeePerGas float64) (TransactionEstimation, error) {
	log.Debug("call to getTransactionEstimatedTime")
	return api.s.feesManager.transactionEstimatedTime(ctx, chainID, maxFeePerGas), nil
}

func (api *API) GetSuggestedRoutes(ctx context.Context, account common.Address, amount float64, tokenSymbol string, disabledChainIDs []uint64) (*SuggestedRoutes, error) {
	log.Debug("call to GetSuggestedRoutes")
	return api.router.suggestedRoutes(ctx, account, amount, tokenSymbol, disabledChainIDs)
}

func (api *API) GetDerivedAddressesForPath(ctx context.Context, password string, derivedFrom string, path string, pageSize int, pageNumber int) ([]*DerivedAddress, error) {
	info, err := api.s.gethManager.AccountsGenerator().LoadAccount(derivedFrom, password)
	if err != nil {
		return nil, err
	}

	return api.getDerivedAddresses(ctx, info.ID, path, pageSize, pageNumber)
}

func (api *API) GetDerivedAddressesForMnemonicWithPath(ctx context.Context, mnemonic string, path string, pageSize int, pageNumber int) ([]*DerivedAddress, error) {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	info, err := api.s.gethManager.AccountsGenerator().ImportMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return nil, err
	}

	return api.getDerivedAddresses(ctx, info.ID, path, pageSize, pageNumber)
}

func (api *API) GetDerivedAddressForPrivateKey(ctx context.Context, privateKey string) ([]*DerivedAddress, error) {
	var derivedAddresses = make([]*DerivedAddress, 0)
	info, err := api.s.gethManager.AccountsGenerator().ImportPrivateKey(privateKey)
	if err != nil {
		return derivedAddresses, err
	}

	addressExists, err := api.s.accountsDB.AddressExists(types.Address(common.HexToAddress(info.Address)))
	if err != nil {
		return derivedAddresses, err
	}
	if addressExists {
		return derivedAddresses, fmt.Errorf("account already exists")
	}

	transactions, err := api.s.transferController.GetTransfersByAddress(ctx, api.s.rpcClient.UpstreamChainID, common.HexToAddress(info.Address), nil, (*hexutil.Big)(big.NewInt(1)), false)

	if err != nil {
		return derivedAddresses, err
	}

	hasActivity := int64(len(transactions)) > 0

	derivedAddress := &DerivedAddress{
		Address:        common.HexToAddress(info.Address),
		Path:           "",
		HasActivity:    hasActivity,
		AlreadyCreated: addressExists,
	}
	derivedAddresses = append(derivedAddresses, derivedAddress)

	return derivedAddresses, nil
}

func (api *API) getDerivedAddresses(ctx context.Context, id string, path string, pageSize int, pageNumber int) ([]*DerivedAddress, error) {
	var (
		group                     = async.NewAtomicGroup(ctx)
		derivedAddresses          = make([]*DerivedAddress, 0)
		unorderedDerivedAddresses = map[int]*DerivedAddress{}
		err                       error
	)

	splitPathValues := strings.Split(path, "/")
	if len(splitPathValues) == 6 {
		derivedAddress, err := api.getDerivedAddress(id, path)
		if err != nil {
			return nil, err
		}
		derivedAddresses = append(derivedAddresses, derivedAddress)
	} else {

		if pageNumber <= 0 || pageSize <= 0 {
			return nil, fmt.Errorf("pageSize and pageNumber should be greater than 0")
		}

		var startIndex = ((pageNumber - 1) * pageSize)
		var endIndex = (pageNumber * pageSize)

		for i := startIndex; i < endIndex; i++ {
			derivedPath := fmt.Sprint(path, "/", i)
			index := i
			group.Add(func(parent context.Context) error {
				derivedAddress, err := api.getDerivedAddress(id, derivedPath)
				if err != nil {
					return err
				}
				unorderedDerivedAddresses[index] = derivedAddress
				return nil
			})
		}
		select {
		case <-group.WaitAsync():
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		for i := startIndex; i < endIndex; i++ {
			derivedAddresses = append(derivedAddresses, unorderedDerivedAddresses[i])
		}
		err = group.Error()
	}
	return derivedAddresses, err
}

func (api *API) getDerivedAddress(id string, derivedPath string) (*DerivedAddress, error) {
	addedAccounts, err := api.s.accountsDB.GetAccounts()
	if err != nil {
		return nil, err
	}

	info, err := api.s.gethManager.AccountsGenerator().DeriveAddresses(id, []string{derivedPath})
	if err != nil {
		return nil, err
	}

	alreadyExists := false
	for _, account := range addedAccounts {
		if types.Address(common.HexToAddress(info[derivedPath].Address)) == account.Address {
			alreadyExists = true
			break
		}
	}

	var ctx context.Context
	transactions, err := api.s.transferController.GetTransfersByAddress(ctx, api.s.rpcClient.UpstreamChainID, common.HexToAddress(info[derivedPath].Address), nil, (*hexutil.Big)(big.NewInt(1)), false)

	if err != nil {
		return nil, err
	}

	hasActivity := int64(len(transactions)) > 0

	address := &DerivedAddress{
		Address:        common.HexToAddress(info[derivedPath].Address),
		Path:           derivedPath,
		HasActivity:    hasActivity,
		AlreadyCreated: alreadyExists,
	}

	return address, nil
}

func (api *API) CreateMultiTransaction(ctx context.Context, multiTransaction *MultiTransaction, data map[uint64][]transactions.SendTxArgs, password string) (*MultiTransactionResult, error) {
	log.Debug("[WalletAPI:: CreateMultiTransaction] create multi transaction")
	return api.s.transactionManager.createMultiTransaction(ctx, multiTransaction, data, password)
}
