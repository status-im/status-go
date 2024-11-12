package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"
	abi_spec "github.com/status-im/status-go/abi-spec"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/services/wallet/activity"
	"github.com/status-im/status-go/services/wallet/collectibles"
	wcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/currency"
	"github.com/status-im/status-go/services/wallet/history"
	"github.com/status-im/status-go/services/wallet/onramp"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletconnect"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

func NewAPI(s *Service) *API {
	return &API{s, s.reader}
}

// API is class with methods available over RPC.
type API struct {
	s      *Service
	reader *Reader
}

func (api *API) StartWallet(ctx context.Context) error {
	return api.reader.Start()
}

func (api *API) StopWallet(ctx context.Context) error {
	return api.s.Stop()
}

func (api *API) GetPairingsJSONFileContent() ([]byte, error) {
	return api.s.keycardPairings.GetPairingsJSONFileContent()
}

func (api *API) SetPairingsJSONFileContent(content []byte) error {
	return api.s.keycardPairings.SetPairingsJSONFileContent(content)
}

// Used by mobile
func (api *API) GetWalletToken(ctx context.Context, addresses []common.Address) (map[common.Address][]token.StorageToken, error) {
	currency, err := api.s.accountsDB.GetCurrency()
	if err != nil {
		return nil, err
	}

	activeNetworks, err := api.s.rpcClient.NetworkManager.GetActiveNetworks()
	if err != nil {
		return nil, err
	}

	chainIDs := wcommon.NetworksToChainIDs(activeNetworks)
	clients, err := api.s.rpcClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	return api.reader.GetWalletToken(ctx, clients, addresses, currency)
}

func (api *API) GetLastWalletTokenUpdate() map[common.Address]int64 {
	return api.reader.GetLastTokenUpdateTimestamps()
}

// GetBalancesByChain return a map with key as chain id and value as map of account address and map of token address and balance
// [chainID][account][token]balance
func (api *API) GetBalancesByChain(ctx context.Context, chainIDs []uint64, addresses, tokens []common.Address) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	clients, err := api.s.rpcClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	return api.s.tokenManager.GetBalancesByChain(ctx, clients, addresses, tokens)
}

func (api *API) FetchOrGetCachedWalletBalances(ctx context.Context, addresses []common.Address) (map[common.Address][]token.StorageToken, error) {
	activeNetworks, err := api.s.rpcClient.NetworkManager.GetActiveNetworks()
	if err != nil {
		return nil, err
	}

	chainIDs := wcommon.NetworksToChainIDs(activeNetworks)
	clients, err := api.s.rpcClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	return api.reader.FetchOrGetCachedWalletBalances(ctx, clients, addresses, false)
}

type DerivedAddress struct {
	Address        common.Address `json:"address"`
	PublicKey      types.HexBytes `json:"public-key,omitempty"`
	Path           string         `json:"path"`
	HasActivity    bool           `json:"hasActivity"`
	AlreadyCreated bool           `json:"alreadyCreated"`
}

// @deprecated
func (api *API) CheckRecentHistory(ctx context.Context, addresses []common.Address) error {
	return api.s.transferController.CheckRecentHistory([]uint64{api.s.rpcClient.UpstreamChainID}, addresses)
}

// @deprecated
func (api *API) CheckRecentHistoryForChainIDs(ctx context.Context, chainIDs []uint64, addresses []common.Address) error {
	return api.s.transferController.CheckRecentHistory(chainIDs, addresses)
}

func hexBigToBN(hexBig *hexutil.Big) *big.Int {
	var bN *big.Int
	if hexBig != nil {
		bN = hexBig.ToInt()
	}
	return bN
}

// @deprecated
// GetTransfersByAddress returns transfers for a single address
func (api *API) GetTransfersByAddress(ctx context.Context, address common.Address, toBlock, limit *hexutil.Big, fetchMore bool) ([]transfer.View, error) {
	logutils.ZapLogger().Debug("[WalletAPI:: GetTransfersByAddress] get transfers for an address", zap.Stringer("address", address))
	var intLimit = int64(1)
	if limit != nil {
		intLimit = limit.ToInt().Int64()
	}
	return api.s.transferController.GetTransfersByAddress(ctx, api.s.rpcClient.UpstreamChainID, address, hexBigToBN(toBlock), intLimit, fetchMore)
}

// @deprecated
// LoadTransferByHash loads transfer to the database
// Only used by status-mobile
func (api *API) LoadTransferByHash(ctx context.Context, address common.Address, hash common.Hash) error {
	logutils.ZapLogger().Debug("[WalletAPI:: LoadTransferByHash] get transfer by hash", zap.Stringer("address", address), zap.Stringer("hash", hash))
	return api.s.transferController.LoadTransferByHash(ctx, api.s.rpcClient, address, hash)
}

// @deprecated
func (api *API) GetTransfersByAddressAndChainID(ctx context.Context, chainID uint64, address common.Address, toBlock, limit *hexutil.Big, fetchMore bool) ([]transfer.View, error) {
	logutils.ZapLogger().Debug("[WalletAPI:: GetTransfersByAddressAndChainIDs] get transfers for an address", zap.Stringer("address", address))
	return api.s.transferController.GetTransfersByAddress(ctx, chainID, address, hexBigToBN(toBlock), limit.ToInt().Int64(), fetchMore)
}

// @deprecated
func (api *API) GetTransfersForIdentities(ctx context.Context, identities []transfer.TransactionIdentity) ([]transfer.View, error) {
	logutils.ZapLogger().Debug("wallet.api.GetTransfersForIdentities", zap.Int("identities.len", len(identities)))

	return api.s.transferController.GetTransfersForIdentities(ctx, identities)
}

func (api *API) FetchDecodedTxData(ctx context.Context, data string) (*thirdparty.DataParsed, error) {
	logutils.ZapLogger().Debug("[Wallet: FetchDecodedTxData]")

	return api.s.decoder.Decode(data)
}

// GetBalanceHistory retrieves token balance history for token identity on multiple chains
func (api *API) GetBalanceHistory(ctx context.Context, chainIDs []uint64, addresses []common.Address, tokenSymbol string, currencySymbol string, timeInterval history.TimeInterval) ([]*history.ValuePoint, error) {
	logutils.ZapLogger().Debug("wallet.api.GetBalanceHistory",
		zap.Uint64s("chainIDs", chainIDs),
		zap.Stringers("address", addresses),
		zap.String("tokenSymbol", tokenSymbol),
		zap.String("currencySymbol", currencySymbol),
		zap.Int("timeInterval", int(timeInterval)),
	)

	var fromTimestamp uint64
	now := uint64(time.Now().UTC().Unix())
	switch timeInterval {
	case history.BalanceHistoryAllTime:
		fromTimestamp = 0
	case history.BalanceHistory1Year:
		fallthrough
	case history.BalanceHistory6Months:
		fallthrough
	case history.BalanceHistory1Month:
		fallthrough
	case history.BalanceHistory7Days:
		fromTimestamp = now - history.TimeIntervalDurationSecs(timeInterval)
	default:
		return nil, fmt.Errorf("unknown time interval: %v", timeInterval)
	}

	return api.GetBalanceHistoryRange(ctx, chainIDs, addresses, tokenSymbol, currencySymbol, fromTimestamp, now)
}

// GetBalanceHistoryRange retrieves token balance history for token identity on multiple chains for a time range
// 'toTimestamp' is ignored for now, but will be used in the future to limit the range of the history
func (api *API) GetBalanceHistoryRange(ctx context.Context, chainIDs []uint64, addresses []common.Address, tokenSymbol string, currencySymbol string, fromTimestamp uint64, _ uint64) ([]*history.ValuePoint, error) {
	logutils.ZapLogger().Debug("wallet.api.GetBalanceHistoryRange",
		zap.Uint64s("chainIDs", chainIDs),
		zap.Stringers("address", addresses),
		zap.String("tokenSymbol", tokenSymbol),
		zap.String("currencySymbol", currencySymbol),
		zap.Uint64("fromTimestamp", fromTimestamp),
	)
	return api.s.history.GetBalanceHistory(ctx, chainIDs, addresses, tokenSymbol, currencySymbol, fromTimestamp)
}

func (api *API) GetTokenList(ctx context.Context) (*token.ListWrapper, error) {
	logutils.ZapLogger().Debug("call to get token list")
	rst := api.s.tokenManager.GetList()
	logutils.ZapLogger().Debug("result from token list", zap.Int("len", len(rst.Data)))
	return rst, nil
}

// @deprecated
func (api *API) GetTokens(ctx context.Context, chainID uint64) ([]*token.Token, error) {
	logutils.ZapLogger().Debug("call to get tokens")
	rst, err := api.s.tokenManager.GetTokens(chainID)
	logutils.ZapLogger().Debug("result from token store", zap.Int("len", len(rst)))
	return rst, err
}

// @deprecated
func (api *API) GetCustomTokens(ctx context.Context) ([]*token.Token, error) {
	logutils.ZapLogger().Debug("call to get custom tokens")
	rst, err := api.s.tokenManager.GetCustoms(true)
	logutils.ZapLogger().Debug("result from database for custom tokens", zap.Int("len", len(rst)))
	return rst, err
}

func (api *API) DiscoverToken(ctx context.Context, chainID uint64, address common.Address) (*token.Token, error) {
	logutils.ZapLogger().Debug("call to get discover token")
	token, err := api.s.tokenManager.DiscoverToken(ctx, chainID, address)
	return token, err
}

func (api *API) AddCustomToken(ctx context.Context, token token.Token) error {
	logutils.ZapLogger().Debug("call to create or edit custom token")
	if token.ChainID == 0 {
		token.ChainID = api.s.rpcClient.UpstreamChainID
	}
	err := api.s.tokenManager.UpsertCustom(token)
	logutils.ZapLogger().Debug("result from database for create or edit custom token", zap.Error(err))
	return err
}

// @deprecated
func (api *API) DeleteCustomToken(ctx context.Context, address common.Address) error {
	logutils.ZapLogger().Debug("call to remove custom token")
	err := api.s.tokenManager.DeleteCustom(api.s.rpcClient.UpstreamChainID, address)
	logutils.ZapLogger().Debug("result from database for remove custom token", zap.Error(err))
	return err
}

func (api *API) DeleteCustomTokenByChainID(ctx context.Context, chainID uint64, address common.Address) error {
	logutils.ZapLogger().Debug("call to remove custom token")
	err := api.s.tokenManager.DeleteCustom(chainID, address)
	logutils.ZapLogger().Debug("result from database for remove custom token", zap.Error(err))
	return err
}

// @deprecated
// Not used by status-desktop anymore
func (api *API) GetPendingTransactions(ctx context.Context) ([]*transactions.PendingTransaction, error) {
	logutils.ZapLogger().Debug("wallet.api.GetPendingTransactions")
	rst, err := api.s.pendingTxManager.GetAllPending()
	logutils.ZapLogger().Debug("wallet.api.GetPendingTransactions RESULT", zap.Int("len", len(rst)))
	return rst, err
}

// @deprecated
// Not used by status-desktop anymore
func (api *API) GetPendingTransactionsForIdentities(ctx context.Context, identities []transfer.TransactionIdentity) (
	result []*transactions.PendingTransaction, err error) {

	logutils.ZapLogger().Debug("wallet.api.GetPendingTransactionsForIdentities")

	result = make([]*transactions.PendingTransaction, 0, len(identities))
	var pt *transactions.PendingTransaction
	for _, identity := range identities {
		pt, err = api.s.pendingTxManager.GetPendingEntry(identity.ChainID, identity.Hash)
		result = append(result, pt)
	}

	logutils.ZapLogger().Debug("wallet.api.GetPendingTransactionsForIdentities RES", zap.Int("len", len(result)))
	return
}

// @deprecated
// TODO - #11861: Remove this and replace with EventPendingTransactionStatusChanged event and Delete to confirm the transaction where it is needed
func (api *API) WatchTransactionByChainID(ctx context.Context, chainID uint64, transactionHash common.Hash) (err error) {
	logutils.ZapLogger().Debug("wallet.api.WatchTransactionByChainID", zap.Uint64("chainID", chainID), zap.Stringer("transactionHash", transactionHash))
	defer func() {
		logutils.ZapLogger().Debug("wallet.api.WatchTransactionByChainID",
			zap.Error(err),
			zap.Uint64("chainID", chainID),
			zap.Stringer("transactionHash", transactionHash),
		)
	}()

	return api.s.transactionManager.WatchTransaction(ctx, chainID, transactionHash)
}

func (api *API) GetCryptoOnRamps(ctx context.Context) ([]onramp.CryptoOnRamp, error) {
	logutils.ZapLogger().Debug("call to GetCryptoOnRamps")
	return api.s.cryptoOnRampManager.GetProviders(ctx)
}

func (api *API) GetCryptoOnRampURL(ctx context.Context, providerID string, parameters onramp.Parameters) (string, error) {
	logutils.ZapLogger().Debug("call to GetCryptoOnRampURL")
	return api.s.cryptoOnRampManager.GetURL(ctx, providerID, parameters)
}

/*
   Collectibles API Start
*/

func (api *API) FetchCachedBalancesByOwnerAndContractAddress(ctx context.Context, chainID wcommon.ChainID, ownerAddress common.Address, contractAddresses []common.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	logutils.ZapLogger().Debug("call to FetchCachedBalancesByOwnerAndContractAddress")

	return api.s.collectiblesManager.FetchCachedBalancesByOwnerAndContractAddress(ctx, chainID, ownerAddress, contractAddresses)
}

func (api *API) FetchBalancesByOwnerAndContractAddress(ctx context.Context, chainID wcommon.ChainID, ownerAddress common.Address, contractAddresses []common.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	logutils.ZapLogger().Debug("call to FetchBalancesByOwnerAndContractAddress")

	return api.s.collectiblesManager.FetchBalancesByOwnerAndContractAddress(ctx, chainID, ownerAddress, contractAddresses)
}

func (api *API) GetCollectibleOwnership(id thirdparty.CollectibleUniqueID) ([]thirdparty.AccountBalance, error) {
	return api.s.collectiblesManager.GetCollectibleOwnership(id)
}

func (api *API) RefetchOwnedCollectibles() error {
	logutils.ZapLogger().Debug("wallet.api.RefetchOwnedCollectibles")

	api.s.collectibles.RefetchOwnedCollectibles()
	return nil
}

func (api *API) GetOwnedCollectiblesAsync(requestID int32, chainIDs []wcommon.ChainID, addresses []common.Address, filter collectibles.Filter, offset int, limit int, dataType collectibles.CollectibleDataType, fetchCriteria collectibles.FetchCriteria) error {
	logutils.ZapLogger().Debug("wallet.api.GetOwnedCollectiblesAsync",
		zap.Int32("requestID", requestID),
		zap.Int("chainIDs.count", len(chainIDs)),
		zap.Int("addr.count", len(addresses)),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.Any("dataType", dataType),
		zap.Any("fetchCriteria", fetchCriteria),
	)

	api.s.collectibles.GetOwnedCollectiblesAsync(requestID, chainIDs, addresses, filter, offset, limit, dataType, fetchCriteria)
	return nil
}

func (api *API) GetCollectiblesByUniqueIDAsync(requestID int32, uniqueIDs []thirdparty.CollectibleUniqueID, dataType collectibles.CollectibleDataType) error {
	logutils.ZapLogger().Debug("wallet.api.GetCollectiblesByUniqueIDAsync",
		zap.Int32("requestID", requestID),
		zap.Int("uniqueIDs.count", len(uniqueIDs)),
		zap.Any("dataType", dataType),
	)

	api.s.collectibles.GetCollectiblesByUniqueIDAsync(requestID, uniqueIDs, dataType)
	return nil
}

func (api *API) FetchCollectionSocialsAsync(contractID thirdparty.ContractID) error {
	logutils.ZapLogger().Debug("wallet.api.FetchCollectionSocialsAsync", zap.Any("contractID", contractID))

	return api.s.collectiblesManager.FetchCollectionSocialsAsync(contractID)
}

func (api *API) GetCollectibleOwnersByContractAddress(ctx context.Context, chainID wcommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	logutils.ZapLogger().Debug("call to GetCollectibleOwnersByContractAddress")
	return api.s.collectiblesManager.FetchCollectibleOwnersByContractAddress(ctx, chainID, contractAddress)
}

func (api *API) FetchCollectibleOwnersByContractAddress(ctx context.Context, chainID wcommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	logutils.ZapLogger().Debug("call to FetchCollectibleOwnersByContractAddress")
	return api.s.collectiblesManager.FetchCollectibleOwnersByContractAddress(ctx, chainID, contractAddress)
}

func (api *API) SearchCollectibles(ctx context.Context, chainID wcommon.ChainID, text string, cursor string, limit int, providerID string) (*thirdparty.FullCollectibleDataContainer, error) {
	logutils.ZapLogger().Debug("call to SearchCollectibles")
	return api.s.collectiblesManager.SearchCollectibles(ctx, chainID, text, cursor, limit, providerID)
}

func (api *API) SearchCollections(ctx context.Context, chainID wcommon.ChainID, text string, cursor string, limit int, providerID string) (*thirdparty.CollectionDataContainer, error) {
	logutils.ZapLogger().Debug("call to SearchCollections")
	return api.s.collectiblesManager.SearchCollections(ctx, chainID, text, cursor, limit, providerID)
}

/*
   Collectibles API End
*/

func (api *API) AddEthereumChain(ctx context.Context, network params.Network) error {
	logutils.ZapLogger().Debug("call to AddEthereumChain")
	return api.s.rpcClient.NetworkManager.Upsert(&network)
}

func (api *API) DeleteEthereumChain(ctx context.Context, chainID uint64) error {
	logutils.ZapLogger().Debug("call to DeleteEthereumChain")
	return api.s.rpcClient.NetworkManager.Delete(chainID)
}

func (api *API) GetEthereumChains(ctx context.Context) ([]*network.CombinedNetwork, error) {
	logutils.ZapLogger().Debug("call to GetEthereumChains")
	return api.s.rpcClient.NetworkManager.GetCombinedNetworks()
}

// @deprecated
func (api *API) FetchPrices(ctx context.Context, symbols []string, currencies []string) (map[string]map[string]float64, error) {
	logutils.ZapLogger().Debug("call to FetchPrices")
	return api.s.marketManager.FetchPrices(symbols, currencies)
}

// @deprecated
func (api *API) FetchMarketValues(ctx context.Context, symbols []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	logutils.ZapLogger().Debug("call to FetchMarketValues")
	return api.s.marketManager.FetchTokenMarketValues(symbols, currency)
}

func (api *API) GetHourlyMarketValues(ctx context.Context, symbol string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	logutils.ZapLogger().Debug("call to GetHourlyMarketValues")
	return api.s.marketManager.FetchHistoricalHourlyPrices(symbol, currency, limit, aggregate)
}

func (api *API) GetDailyMarketValues(ctx context.Context, symbol string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	logutils.ZapLogger().Debug("call to GetDailyMarketValues")
	return api.s.marketManager.FetchHistoricalDailyPrices(symbol, currency, limit, allData, aggregate)
}

// @deprecated
func (api *API) FetchTokenDetails(ctx context.Context, symbols []string) (map[string]thirdparty.TokenDetails, error) {
	logutils.ZapLogger().Debug("call to FetchTokenDetails")
	return api.s.marketManager.FetchTokenDetails(symbols)
}

func (api *API) GetSuggestedFees(ctx context.Context, chainID uint64) (*fees.SuggestedFeesGwei, error) {
	logutils.ZapLogger().Debug("call to GetSuggestedFees")
	return api.s.router.GetFeesManager().SuggestedFeesGwei(ctx, chainID)
}

func (api *API) GetEstimatedLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error) {
	logutils.ZapLogger().Debug("call to GetEstimatedLatestBlockNumber", zap.Uint64("chainID", chainID))
	return api.s.blockChainState.GetEstimatedLatestBlockNumber(ctx, chainID)
}

func (api *API) GetTransactionEstimatedTime(ctx context.Context, chainID uint64, maxFeePerGas *big.Float) (fees.TransactionEstimation, error) {
	logutils.ZapLogger().Debug("call to getTransactionEstimatedTime")
	return api.s.router.GetFeesManager().TransactionEstimatedTime(ctx, chainID, gweiToWei(maxFeePerGas)), nil
}

func gweiToWei(val *big.Float) *big.Int {
	res, _ := new(big.Float).Mul(val, big.NewFloat(1000000000)).Int(nil)
	return res
}

func (api *API) GetSuggestedRoutes(ctx context.Context, input *requests.RouteInputParams) (*router.SuggestedRoutes, error) {
	logutils.ZapLogger().Debug("call to GetSuggestedRoutes")

	return api.s.router.SuggestedRoutes(ctx, input)
}

func (api *API) GetSuggestedRoutesAsync(ctx context.Context, input *requests.RouteInputParams) {
	logutils.ZapLogger().Debug("call to GetSuggestedRoutesAsync")

	api.s.router.SuggestedRoutesAsync(input)
}

func (api *API) StopSuggestedRoutesAsyncCalculation(ctx context.Context) {
	logutils.ZapLogger().Debug("call to StopSuggestedRoutesAsyncCalculation")

	api.s.router.StopSuggestedRoutesAsyncCalculation()
}

func (api *API) StopSuggestedRoutesCalculation(ctx context.Context) {
	logutils.ZapLogger().Debug("call to StopSuggestedRoutesCalculation")

	api.s.router.StopSuggestedRoutesCalculation()
}

// Generates addresses for the provided paths, response doesn't include `HasActivity` value (if you need it check `GetAddressDetails` function)
func (api *API) GetDerivedAddresses(ctx context.Context, password string, derivedFrom string, paths []string) ([]*DerivedAddress, error) {
	info, err := api.s.gethManager.AccountsGenerator().LoadAccount(derivedFrom, password)
	if err != nil {
		return nil, err
	}

	return api.getDerivedAddresses(info.ID, paths)
}

// Generates addresses for the provided paths derived from the provided mnemonic, response doesn't include `HasActivity` value (if you need it check `GetAddressDetails` function)
func (api *API) GetDerivedAddressesForMnemonic(ctx context.Context, mnemonic string, paths []string) ([]*DerivedAddress, error) {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	info, err := api.s.gethManager.AccountsGenerator().ImportMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return nil, err
	}

	return api.getDerivedAddresses(info.ID, paths)
}

// Generates addresses for the provided paths, response doesn't include `HasActivity` value (if you need it check `GetAddressDetails` function)
func (api *API) getDerivedAddresses(id string, paths []string) ([]*DerivedAddress, error) {
	addedAccounts, err := api.s.accountsDB.GetActiveAccounts()
	if err != nil {
		return nil, err
	}

	info, err := api.s.gethManager.AccountsGenerator().DeriveAddresses(id, paths)
	if err != nil {
		return nil, err
	}

	derivedAddresses := make([]*DerivedAddress, 0)
	for accPath, acc := range info {

		derivedAddress := &DerivedAddress{
			Address:   common.HexToAddress(acc.Address),
			PublicKey: types.Hex2Bytes(acc.PublicKey),
			Path:      accPath,
		}

		for _, account := range addedAccounts {
			if types.Address(derivedAddress.Address) == account.Address {
				derivedAddress.AlreadyCreated = true
				break
			}
		}

		derivedAddresses = append(derivedAddresses, derivedAddress)
	}

	return derivedAddresses, nil
}

func (api *API) AddressExists(ctx context.Context, address types.Address) (bool, error) {
	return api.s.accountsDB.AddressExists(address)
}

// AddressDetails returns details for passed params (passed address, chains to check, timeout for the call to complete)
// if chainIDs is empty, it will use all active chains
// if timeout is zero, it will wait until the call completes
// response doesn't include derivation path
func (api *API) AddressDetails(ctx context.Context, params *requests.AddressDetails) (*DerivedAddress, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	result := &DerivedAddress{
		Address: common.HexToAddress(params.Address),
	}
	addressExists, err := api.s.accountsDB.AddressExists(types.Address(result.Address))
	if err != nil {
		return result, err
	}

	result.AlreadyCreated = addressExists

	chainIDs := params.ChainIDs
	if len(chainIDs) == 0 {
		activeNetworks, err := api.s.rpcClient.NetworkManager.GetActiveNetworks()
		if err != nil {
			return nil, err
		}

		chainIDs = wcommon.NetworksToChainIDs(activeNetworks)
	}

	clients, err := api.s.rpcClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	if params.TimeoutInMilliseconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(params.TimeoutInMilliseconds)*time.Millisecond)
		defer cancel()
	}

	for _, client := range clients {
		balance, err := api.s.tokenManager.GetChainBalance(ctx, client, result.Address)
		if err != nil {
			if err != nil && errors.Is(err, context.DeadlineExceeded) {
				return result, nil
			}
			return result, err
		}

		result.HasActivity = balance.Cmp(big.NewInt(0)) != 0
		if result.HasActivity {
			break
		}
	}

	return result, nil
}

// @deprecated replaced by AddressDetails
// GetAddressDetails returns details for the passed address (response doesn't include derivation path)
func (api *API) GetAddressDetails(ctx context.Context, chainID uint64, address string) (*DerivedAddress, error) {
	result := &DerivedAddress{
		Address: common.HexToAddress(address),
	}
	addressExists, err := api.s.accountsDB.AddressExists(types.Address(result.Address))
	if err != nil {
		return result, err
	}

	result.AlreadyCreated = addressExists

	chainClient, err := api.s.rpcClient.EthClient(chainID)
	if err != nil {
		return result, err
	}

	balance, err := api.s.tokenManager.GetChainBalance(ctx, chainClient, result.Address)
	if err != nil {
		return result, err
	}

	result.HasActivity = balance.Cmp(big.NewInt(0)) != 0
	return result, nil
}

func (api *API) SignMessage(ctx context.Context, message types.HexBytes, address common.Address, password string) (string, error) {
	logutils.ZapLogger().Debug("[WalletAPI::SignMessage]", zap.Stringer("message", message), zap.Stringer("address", address))

	selectedAccount, err := api.s.gethManager.VerifyAccountPassword(api.s.Config().KeyStoreDir, address.Hex(), password)
	if err != nil {
		return "", err
	}

	return api.s.transactionManager.SignMessage(message, selectedAccount)
}

func (api *API) BuildTransaction(ctx context.Context, chainID uint64, sendTxArgsJSON string) (response *transfer.TxResponse, err error) {
	logutils.ZapLogger().Debug("[WalletAPI::BuildTransaction]", zap.Uint64("chainID", chainID), zap.String("sendTxArgsJSON", sendTxArgsJSON))
	var params wallettypes.SendTxArgs
	err = json.Unmarshal([]byte(sendTxArgsJSON), &params)
	if err != nil {
		return nil, err
	}
	return api.s.transactionManager.BuildTransaction(chainID, params)
}

func (api *API) BuildRawTransaction(ctx context.Context, chainID uint64, sendTxArgsJSON string, signature string) (response *transfer.TxResponse, err error) {
	logutils.ZapLogger().Debug("[WalletAPI::BuildRawTransaction]", zap.Uint64("chainID", chainID), zap.String("sendTxArgsJSON", sendTxArgsJSON), zap.String("signature", signature))

	sig, err := hex.DecodeString(signature)
	if err != nil {
		return nil, err
	}

	var params wallettypes.SendTxArgs
	err = json.Unmarshal([]byte(sendTxArgsJSON), &params)
	if err != nil {
		return nil, err
	}

	return api.s.transactionManager.BuildRawTransaction(chainID, params, sig)
}

func (api *API) SendTransactionWithSignature(ctx context.Context, chainID uint64, txType transactions.PendingTrxType,
	sendTxArgsJSON string, signature string) (hash types.Hash, err error) {
	logutils.ZapLogger().Debug("[WalletAPI::SendTransactionWithSignature]",
		zap.Uint64("chainID", chainID),
		zap.String("txType", string(txType)),
		zap.String("sendTxArgsJSON", sendTxArgsJSON),
		zap.String("signature", signature),
	)
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return hash, err
	}

	var params wallettypes.SendTxArgs
	err = json.Unmarshal([]byte(sendTxArgsJSON), &params)
	if err != nil {
		return hash, err
	}
	return api.s.transactionManager.SendTransactionWithSignature(chainID, params, sig)
}

// Deprecated: `CreateMultiTransaction` is the old way of sending transactions and should not be used anymore.
//
// The flow that should be used instead:
// - call `BuildTransactionsFromRoute`
// - wait for the `wallet.router.sign-transactions` signal
// - sign received hashes using `SignMessage` call or sign on keycard
// - call `SendRouterTransactionsWithSignatures` with the signatures of signed hashes from the previous step
//
// TODO: remove this struct once mobile switches to the new approach
func (api *API) CreateMultiTransaction(ctx context.Context, multiTransactionCommand *transfer.MultiTransactionCommand, data []*pathprocessor.MultipathProcessorTxArgs, password string) (*transfer.MultiTransactionCommandResult, error) {
	logutils.ZapLogger().Debug("[WalletAPI:: CreateMultiTransaction] create multi transaction")

	cmd, err := api.s.transactionManager.CreateMultiTransactionFromCommand(multiTransactionCommand, data)
	if err != nil {
		return nil, err
	}

	if password != "" {
		selectedAccount, err := api.getVerifiedWalletAccount(multiTransactionCommand.FromAddress.Hex(), password)
		if err != nil {
			return nil, err
		}

		cmdRes, err := api.s.transactionManager.SendTransactions(ctx, cmd, data, api.s.router.GetPathProcessors(), selectedAccount)
		if err != nil {
			return nil, err
		}

		_, err = api.s.transactionManager.InsertMultiTransaction(cmd)
		if err != nil {
			logutils.ZapLogger().Error("Failed to save multi transaction", zap.Error(err)) // not critical
		}

		return cmdRes, nil
	}

	return nil, api.s.transactionManager.SendTransactionForSigningToKeycard(ctx, cmd, data, api.s.router.GetPathProcessors())
}

func (api *API) BuildTransactionsFromRoute(ctx context.Context, buildInputParams *requests.RouterBuildTransactionsParams) {
	logutils.ZapLogger().Debug("[WalletAPI::BuildTransactionsFromRoute] builds transactions from the generated best route", zap.String("uuid", buildInputParams.Uuid))
	api.s.routeExecutionManager.BuildTransactionsFromRoute(ctx, buildInputParams)
}

// Deprecated: `ProceedWithTransactionsSignatures` is the endpoint used in the old way of sending transactions and should not be used anymore.
//
// The flow that should be used instead:
// - call `BuildTransactionsFromRoute`
// - wait for the `wallet.router.sign-transactions` signal
// - sign received hashes using `SignMessage` call or sign on keycard
// - call `SendRouterTransactionsWithSignatures` with the signatures of signed hashes from the previous step
//
// TODO: remove this struct once mobile switches to the new approach
func (api *API) ProceedWithTransactionsSignatures(ctx context.Context, signatures map[string]requests.SignatureDetails) (*transfer.MultiTransactionCommandResult, error) {
	logutils.ZapLogger().Debug("[WalletAPI:: ProceedWithTransactionsSignatures] sign with signatures and send multi transaction")
	return api.s.transactionManager.ProceedWithTransactionsSignatures(ctx, signatures)
}

func (api *API) SendRouterTransactionsWithSignatures(ctx context.Context, sendInputParams *requests.RouterSendTransactionsParams) {
	logutils.ZapLogger().Debug("[WalletAPI:: SendRouterTransactionsWithSignatures] sign with signatures and send")
	api.s.routeExecutionManager.SendRouterTransactionsWithSignatures(ctx, sendInputParams)
}

func (api *API) GetMultiTransactions(ctx context.Context, transactionIDs []wcommon.MultiTransactionIDType) ([]*transfer.MultiTransaction, error) {
	logutils.ZapLogger().Debug("wallet.api.GetMultiTransactions", zap.Int("IDs.len", len(transactionIDs)))
	return api.s.transactionManager.GetMultiTransactions(ctx, transactionIDs)
}

func (api *API) GetCachedCurrencyFormats() (currency.FormatPerSymbol, error) {
	logutils.ZapLogger().Debug("call to GetCachedCurrencyFormats")
	return api.s.currency.GetCachedCurrencyFormats()
}

func (api *API) FetchAllCurrencyFormats() (currency.FormatPerSymbol, error) {
	logutils.ZapLogger().Debug("call to FetchAllCurrencyFormats")
	return api.s.currency.FetchAllCurrencyFormats()
}

// @deprecated replaced by session APIs; see #12120
func (api *API) FilterActivityAsync(requestID int32, addresses []common.Address, chainIDs []wcommon.ChainID, filter activity.Filter, offset int, limit int) error {
	logutils.ZapLogger().Debug("wallet.api.FilterActivityAsync",
		zap.Int32("requestID", requestID),
		zap.Int("addr.count", len(addresses)),
		zap.Int("chainIDs.count", len(chainIDs)),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
	)

	api.s.activity.FilterActivityAsync(requestID, addresses, chainIDs, filter, offset, limit)
	return nil
}

// @deprecated replaced by session APIs; see #12120
func (api *API) CancelActivityFilterTask(requestID int32) error {
	logutils.ZapLogger().Debug("wallet.api.CancelActivityFilterTask", zap.Int32("requestID", requestID))

	api.s.activity.CancelFilterTask(requestID)
	return nil
}

func (api *API) StartActivityFilterSession(addresses []common.Address, chainIDs []wcommon.ChainID, filter activity.Filter, firstPageCount int) (activity.SessionID, error) {
	logutils.ZapLogger().Debug("wallet.api.StartActivityFilterSession",
		zap.Int("addr.count", len(addresses)),
		zap.Int("chainIDs.count", len(chainIDs)),
		zap.Int("firstPageCount", firstPageCount),
	)

	return api.s.activity.StartFilterSession(addresses, chainIDs, filter, firstPageCount, activity.V1), nil
}

func (api *API) StartActivityFilterSessionV2(addresses []common.Address, chainIDs []wcommon.ChainID, filter activity.Filter, firstPageCount int) (activity.SessionID, error) {
	logutils.ZapLogger().Debug("wallet.api.StartActivityFilterSessionV2",
		zap.Int("addr.count", len(addresses)),
		zap.Int("chainIDs.count", len(chainIDs)),
		zap.Int("firstPageCount", firstPageCount),
	)

	return api.s.activity.StartFilterSession(addresses, chainIDs, filter, firstPageCount, activity.V2), nil
}

func (api *API) UpdateActivityFilterForSession(sessionID activity.SessionID, filter activity.Filter, firstPageCount int) error {
	logutils.ZapLogger().Debug("wallet.api.UpdateActivityFilterForSession",
		zap.Int32("sessionID", int32(sessionID)),
		zap.Int("firstPageCount", firstPageCount),
	)

	return api.s.activity.UpdateFilterForSession(sessionID, filter, firstPageCount)
}

func (api *API) ResetActivityFilterSession(id activity.SessionID, firstPageCount int) error {
	logutils.ZapLogger().Debug("wallet.api.ResetActivityFilterSession",
		zap.Int32("id", int32(id)),
		zap.Int("firstPageCount", firstPageCount),
	)

	return api.s.activity.ResetFilterSession(id, firstPageCount)
}

func (api *API) GetMoreForActivityFilterSession(id activity.SessionID, pageCount int) error {
	logutils.ZapLogger().Debug("wallet.api.GetMoreForActivityFilterSession",
		zap.Int32("id", int32(id)),
		zap.Int("pageCount", pageCount),
	)

	return api.s.activity.GetMoreForFilterSession(id, pageCount)
}

func (api *API) StopActivityFilterSession(id activity.SessionID) {
	logutils.ZapLogger().Debug("wallet.api.StopActivityFilterSession", zap.Int32("id", int32(id)))

	api.s.activity.StopFilterSession(id)
}

func (api *API) GetMultiTxDetails(ctx context.Context, multiTxID int) (*activity.EntryDetails, error) {
	logutils.ZapLogger().Debug("wallet.api.GetMultiTxDetails", zap.Int("multiTxID", multiTxID))

	return api.s.activity.GetMultiTxDetails(ctx, multiTxID)
}

func (api *API) GetTxDetails(ctx context.Context, id string) (*activity.EntryDetails, error) {
	logutils.ZapLogger().Debug("wallet.api.GetTxDetails", zap.String("id", id))

	return api.s.activity.GetTxDetails(ctx, id)
}

func (api *API) GetRecipientsAsync(requestID int32, chainIDs []wcommon.ChainID, addresses []common.Address, offset int, limit int) (ignored bool, err error) {
	logutils.ZapLogger().Debug("wallet.api.GetRecipientsAsync",
		zap.Int("addresses.len", len(addresses)),
		zap.Int("chainIDs.len", len(chainIDs)),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
	)

	ignored = api.s.activity.GetRecipientsAsync(requestID, chainIDs, addresses, offset, limit)
	return ignored, err
}

func (api *API) GetOldestActivityTimestampAsync(requestID int32, addresses []common.Address) error {
	logutils.ZapLogger().Debug("wallet.api.GetOldestActivityTimestamp", zap.Int("addresses.len", len(addresses)))

	api.s.activity.GetOldestTimestampAsync(requestID, addresses)
	return nil
}

func (api *API) GetActivityCollectiblesAsync(requestID int32, chainIDs []wcommon.ChainID, addresses []common.Address, offset int, limit int) error {
	logutils.ZapLogger().Debug("wallet.api.GetActivityCollectiblesAsync",
		zap.Int("addresses.len", len(addresses)),
		zap.Int("chainIDs.len", len(chainIDs)),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
	)

	api.s.activity.GetActivityCollectiblesAsync(requestID, chainIDs, addresses, offset, limit)

	return nil
}

func (api *API) FetchChainIDForURL(ctx context.Context, rpcURL string) (*big.Int, error) {
	logutils.ZapLogger().Debug("wallet.api.VerifyURL", zap.String("rpcURL", rpcURL))

	rpcClient, err := gethrpc.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial upstream server: %s", err)
	}
	client := ethclient.NewClient(rpcClient)
	return client.ChainID(ctx)
}

func (api *API) getVerifiedWalletAccount(address, password string) (*account.SelectedExtKey, error) {
	exists, err := api.s.accountsDB.AddressExists(types.HexToAddress(address))
	if err != nil {
		logutils.ZapLogger().Error("failed to query db for a given address", zap.String("address", address), zap.Error(err))
		return nil, err
	}

	if !exists {
		logutils.ZapLogger().Error("failed to get a selected account", zap.Error(wallettypes.ErrInvalidTxSender))
		return nil, wallettypes.ErrAccountDoesntExist
	}

	keyStoreDir := api.s.Config().KeyStoreDir
	key, err := api.s.gethManager.VerifyAccountPassword(keyStoreDir, address, password)
	if err != nil {
		logutils.ZapLogger().Error("failed to verify account", zap.String("account", address), zap.Error(err))
		return nil, err
	}

	return &account.SelectedExtKey{
		Address:    key.Address,
		AccountKey: key,
	}, nil
}

// AddWalletConnectSession adds or updates a session wallet connect session
func (api *API) AddWalletConnectSession(ctx context.Context, session_json string) error {
	logutils.ZapLogger().Debug("wallet.api.AddWalletConnectSession", zap.Int("rpcURL", len(session_json)))
	return walletconnect.AddSession(api.s.db, api.s.config.Networks, session_json)
}

// DisconnectWalletConnectSession removes a wallet connect session
func (api *API) DisconnectWalletConnectSession(ctx context.Context, topic walletconnect.Topic) error {
	logutils.ZapLogger().Debug("wallet.api.DisconnectWalletConnectSession", zap.String("topic", string(topic)))
	return walletconnect.DisconnectSession(api.s.db, topic)
}

// GetWalletConnectActiveSessions returns all active wallet connect sessions
func (api *API) GetWalletConnectActiveSessions(ctx context.Context, validAtTimestamp int64) ([]walletconnect.DBSession, error) {
	logutils.ZapLogger().Debug("wallet.api.GetWalletConnectActiveSessions")
	return walletconnect.GetActiveSessions(api.s.db, validAtTimestamp)
}

// GetWalletConnectDapps returns all active wallet connect dapps
// Active dApp are those having active sessions (not expired and not disconnected)
func (api *API) GetWalletConnectDapps(ctx context.Context, validAtTimestamp int64, testChains bool) ([]walletconnect.DBDApp, error) {
	logutils.ZapLogger().Debug("wallet.api.GetWalletConnectDapps",
		zap.Int64("validAtTimestamp", validAtTimestamp),
		zap.Bool("testChains", testChains),
	)
	return walletconnect.GetActiveDapps(api.s.db, validAtTimestamp, testChains)
}

// HashMessageEIP191 is used for hashing dApps requests for "personal_sign" and "eth_sign"
// in a safe manner following the EIP-191 version 0x45 for signing on the client side.
func (api *API) HashMessageEIP191(ctx context.Context, message types.HexBytes) types.Hash {
	logutils.ZapLogger().Debug("wallet.api.HashMessageEIP191", zap.Int("len(data)", len(message)))
	safeMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), string(message))
	return crypto.Keccak256Hash([]byte(safeMsg))
}

// SignTypedDataV4 dApps use it to execute "eth_signTypedData_v4" requests
// the formatted typed data will be prefixed with \x19\x01 based on the EIP-712
// @deprecated
func (api *API) SignTypedDataV4(typedJson string, address string, password string) (types.HexBytes, error) {
	logutils.ZapLogger().Debug("wallet.api.SignTypedDataV4",
		zap.Int("len(typedJson)", len(typedJson)),
		zap.String("address", address),
		zap.Int("len(password)", len(password)),
	)

	account, err := api.getVerifiedWalletAccount(address, password)
	if err != nil {
		return types.HexBytes{}, err
	}
	var typed signercore.TypedData
	err = json.Unmarshal([]byte(typedJson), &typed)
	if err != nil {
		return types.HexBytes{}, err
	}

	// This is not used down the line but required by the typeddata.SignTypedDataV4 function call
	chain := new(big.Int).SetUint64(api.s.config.NetworkID)
	sig, err := typeddata.SignTypedDataV4(typed, account.AccountKey.PrivateKey, chain)
	if err != nil {
		return types.HexBytes{}, err
	}
	return types.HexBytes(sig), err
}

// SafeSignTypedDataForDApps is used to execute requests for "eth_signTypedData"
// if legacy is true else "eth_signTypedData_v4"
// the formatted typed data won't be prefixed in case of legacy calls, as the
// old dApps implementation expects
// the chain is validate for both cases
func (api *API) SafeSignTypedDataForDApps(typedJson string, address string, password string, chainID uint64, legacy bool) (types.HexBytes, error) {
	logutils.ZapLogger().Debug("wallet.api.SafeSignTypedDataForDApps",
		zap.Int("len(typedJson)", len(typedJson)),
		zap.String("address", address),
		zap.Int("len(password)", len(password)),
		zap.Uint64("chainID", chainID),
		zap.Bool("legacy", legacy),
	)

	account, err := api.getVerifiedWalletAccount(address, password)
	if err != nil {
		return types.HexBytes{}, err
	}

	return walletconnect.SafeSignTypedDataForDApps(typedJson, account.AccountKey.PrivateKey, chainID, legacy)
}

func (api *API) RestartWalletReloadTimer(ctx context.Context) error {
	return api.s.reader.Restart()
}

func (api *API) IsChecksumValidForAddress(address string) (bool, error) {
	logutils.ZapLogger().Debug("wallet.api.isChecksumValidForAddress", zap.String("address", address))
	return abi_spec.CheckAddressChecksum(address)
}
