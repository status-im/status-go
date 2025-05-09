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
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/healthmanager"
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
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
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
	reader ReaderInterface
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

func (api *API) FetchOrGetCachedWalletBalances(ctx context.Context, addresses []common.Address, forceRefresh bool) (map[common.Address][]tokenTypes.StorageToken, error) {
	activeNetworks, err := api.s.rpcClient.NetworkManager.GetActiveNetworks()
	if err != nil {
		return nil, err
	}

	chainIDs := wcommon.NetworksToChainIDs(activeNetworks)
	clients, err := api.s.rpcClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	return api.reader.FetchOrGetCachedWalletBalances(ctx, clients, addresses, forceRefresh)
}

type DerivedAddress struct {
	Address        common.Address `json:"address"`
	PublicKey      types.HexBytes `json:"public-key,omitempty"`
	Path           string         `json:"path"`
	HasActivity    bool           `json:"hasActivity"`
	AlreadyCreated bool           `json:"alreadyCreated"`
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

func (api *API) GetTokensAvailableForBridgeOnChain(ctx context.Context, chainID uint64) []*tokenTypes.Token {
	logutils.ZapLogger().Debug("call to get tokens available for bridge on chain")
	return api.s.router.GetTokensAvailableForBridgeOnChain(chainID)
}

// @deprecated
func (api *API) GetTokens(ctx context.Context, chainID uint64) ([]*tokenTypes.Token, error) {
	logutils.ZapLogger().Debug("call to get tokens")
	rst, err := api.s.tokenManager.GetTokens(chainID)
	logutils.ZapLogger().Debug("result from token store", zap.Int("len", len(rst)))
	return rst, err
}

// @deprecated
func (api *API) GetCustomTokens(ctx context.Context) ([]*tokenTypes.Token, error) {
	logutils.ZapLogger().Debug("call to get custom tokens")
	rst, err := api.s.tokenManager.GetCustoms(true)
	logutils.ZapLogger().Debug("result from database for custom tokens", zap.Int("len", len(rst)))
	return rst, err
}

func (api *API) DiscoverToken(ctx context.Context, chainID uint64, address common.Address) (*tokenTypes.Token, error) {
	logutils.ZapLogger().Debug("call to get discover token")
	token, err := api.s.tokenManager.DiscoverToken(ctx, chainID, address)
	return token, err
}

func (api *API) AddCustomToken(ctx context.Context, token tokenTypes.Token) error {
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

// @deprecated: Custom networks not currently supported. Change settings using specific API functions.
func (api *API) AddEthereumChain(ctx context.Context, network params.Network) error {
	logutils.ZapLogger().Debug("call to AddEthereumChain")
	return api.s.rpcClient.NetworkManager.Upsert(&network)
}

// @deprecated: Custom networks not currently supported. Change settings using specific API functions.
func (api *API) DeleteEthereumChain(ctx context.Context, chainID uint64) error {
	logutils.ZapLogger().Debug("call to DeleteEthereumChain")
	return api.s.rpcClient.NetworkManager.Delete(chainID)
}

func (api *API) SetChainUserRpcProviders(ctx context.Context, chainID uint64, rpcProviders []params.RpcProvider) error {
	logutils.ZapLogger().Debug("call to SetChainUserRpcProviders")
	return api.s.rpcClient.NetworkManager.SetUserRpcProviders(chainID, rpcProviders)
}

// Active chains are the ones that are available for selection across the whole application
// Providers are expected to be accessed only for active chains.
func (api *API) SetChainActive(ctx context.Context, chainID uint64, active bool) error {
	logutils.ZapLogger().Debug("call to SetChainActive")
	return api.s.rpcClient.NetworkManager.SetActive(chainID, active)
}

// Enabled chains are the ones taken into account when displaying balances, collectibles, activity, etc.
func (api *API) SetChainEnabled(ctx context.Context, chainID uint64, enabled bool) error {
	logutils.ZapLogger().Debug("call to SetChainEnabled")
	return api.s.rpcClient.NetworkManager.SetEnabled(chainID, enabled)
}

// @deprecated: Combined networks are not used anymore, use GetFlatEthereumChains instead
func (api *API) GetEthereumChains(ctx context.Context) ([]*network.CombinedNetwork, error) {
	logutils.ZapLogger().Debug("call to GetEthereumChains")
	return api.s.rpcClient.NetworkManager.GetCombinedNetworks()
}

func (api *API) GetFlatEthereumChains(ctx context.Context) ([]*params.Network, error) {
	logutils.ZapLogger().Debug("call to GetFlatEthereumChains")
	return api.s.rpcClient.NetworkManager.GetAll()
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

// @deprecated we should remove it once clients fully switched to wallet router, `GetSuggestedRoutesAsync` should be used instead
func (api *API) GetSuggestedFees(ctx context.Context, chainID uint64) (*fees.SuggestedFeesGwei, error) {
	logutils.ZapLogger().Debug("call to GetSuggestedFees")
	return api.s.router.GetFeesManager().SuggestedFeesGwei(ctx, chainID)
}

func (api *API) GetEstimatedLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error) {
	logutils.ZapLogger().Debug("call to GetEstimatedLatestBlockNumber", zap.Uint64("chainID", chainID))
	return api.s.blockChainState.GetEstimatedLatestBlockNumber(ctx, chainID)
}

// @deprecated we should remove it once clients fully switched to wallet router, `GetSuggestedRoutesAsync` should be used instead
func (api *API) GetTransactionEstimatedTime(ctx context.Context, chainID uint64, maxFeePerGas *big.Float) (fees.TransactionEstimation, error) {
	logutils.ZapLogger().Debug("call to getTransactionEstimatedTime")
	return api.s.router.GetFeesManager().TransactionEstimatedTime(ctx, chainID, gweiToWei(maxFeePerGas)), nil
}

func (api *API) GetTransactionEstimatedTimeV2(ctx context.Context, chainID uint64, maxFeePerGas *hexutil.Big, maxPriorityFeePerGas *hexutil.Big) (uint, error) {
	logutils.ZapLogger().Debug("call to getTransactionEstimatedTimeV2")
	return api.s.router.GetFeesManager().TransactionEstimatedTimeV2(ctx, chainID, maxFeePerGas.ToInt(), maxPriorityFeePerGas.ToInt()), nil
}

func gweiToWei(val *big.Float) *big.Int {
	res, _ := new(big.Float).Mul(val, big.NewFloat(1000000000)).Int(nil)
	return res
}

func (api *API) GetSuggestedRoutes(ctx context.Context, input *requests.RouteInputParams) (*router.SuggestedRoutes, error) {
	logutils.ZapLogger().Debug("call to GetSuggestedRoutes")

	api.s.routeExecutionManager.ClearLocalRouteData()

	return api.s.router.SuggestedRoutes(ctx, input)
}

func (api *API) GetSuggestedRoutesAsync(ctx context.Context, input *requests.RouteInputParams) {
	logutils.ZapLogger().Debug("call to GetSuggestedRoutesAsync")

	api.s.routeExecutionManager.ClearLocalRouteData()

	api.s.router.SuggestedRoutesAsync(input)
}

func (api *API) StopSuggestedRoutesAsyncCalculation(ctx context.Context) {
	logutils.ZapLogger().Debug("call to StopSuggestedRoutesAsyncCalculation")

	api.s.router.StopSuggestedRoutesAsyncCalculation()
}

// GetBlockchainHealthStatus returns the status of rpc clients
func (api *API) GetBlockchainHealthStatus(ctx context.Context) healthmanager.BlockchainFullStatus {
	logutils.ZapLogger().Debug("call to GetBlockchainHealthStatus")
	return api.s.GetRPCClient().GetHealthManagerFullStatus()
}

func (api *API) StopSuggestedRoutesCalculation(ctx context.Context) {
	logutils.ZapLogger().Debug("call to StopSuggestedRoutesCalculation")

	api.s.router.StopSuggestedRoutesCalculation()
}

// SetFeeMode sets the fee mode for the provided path it should be used for setting predefined fee modes `GasFeeLow`, `GasFeeMedium` and `GasFeeHigh`
// in case of setting custom fee use `SetCustomTxDetails` function
func (api *API) SetFeeMode(ctx context.Context, pathTxIdentity *requests.PathTxIdentity, feeMode fees.GasFeeMode) error {
	logutils.ZapLogger().Debug("call to SetFeeMode")

	return api.s.router.SetFeeMode(ctx, pathTxIdentity, feeMode)
}

// SetCustomTxDetails sets custom tx details for the provided path, in case of setting predefined fee modes use `SetFeeMode` function
func (api *API) SetCustomTxDetails(ctx context.Context, pathTxIdentity *requests.PathTxIdentity, pathTxCustomParams *requests.PathTxCustomParams) error {
	logutils.ZapLogger().Debug("call to SetCustomTxDetails")

	return api.s.router.SetCustomTxDetails(ctx, pathTxIdentity, pathTxCustomParams)
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

func (api *API) BuildTransactionsFromRoute(ctx context.Context, uuid string) {
	logutils.ZapLogger().Debug("[WalletAPI::BuildTransactionsFromRoute] builds transactions from the generated best route", zap.String("uuid", uuid))
	api.s.routeExecutionManager.BuildTransactionsFromRoute(ctx, uuid)
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

// ReevaluateRouterPath reevaluates the tx-fields from the router path that matches the provided pathTxIdentity and sends signal.SuggestedRoutes.
func (api *API) ReevaluateRouterPath(ctx context.Context, pathTxIdentity *requests.PathTxIdentity) error {
	logutils.ZapLogger().Debug("wallet.api.ReevaluateRouterPath")
	return api.s.routeExecutionManager.ReevaluateRouterPath(ctx, pathTxIdentity)
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
		logutils.ZapLogger().Error("failed to query db for a given address", zap.String("address", gocommon.TruncateWithDot(address)), zap.Error(err))
		return nil, err
	}

	if !exists {
		logutils.ZapLogger().Error("failed to get a selected account", zap.Error(wallettypes.ErrInvalidTxSender))
		return nil, wallettypes.ErrAccountDoesntExist
	}

	keyStoreDir := api.s.Config().KeyStoreDir
	key, err := api.s.gethManager.VerifyAccountPassword(keyStoreDir, address, password)
	if err != nil {
		logutils.ZapLogger().Error("failed to verify account", zap.String("account", gocommon.TruncateWithDot(address)), zap.Error(err))
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
