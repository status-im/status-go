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
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"

	abi_spec "github.com/status-im/status-go/internal/abi-spec"
	generator2 "github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/crypto"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/healthmanager"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/services/wallet/activity"
	"github.com/status-im/status-go/services/wallet/collectibles"
	wcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/currency"
	"github.com/status-im/status-go/services/wallet/leaderboard"
	"github.com/status-im/status-go/services/wallet/onramp"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/efp"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletconnect"
	"github.com/status-im/status-go/services/wallet/wallettypes"

	"github.com/status-im/go-wallet-sdk/pkg/ethclient"
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
// [chainID][account][tokenAddress]balance
func (api *API) GetBalancesByChain(ctx context.Context, addresses []common.Address, tokenKeys []string) (map[uint64]map[common.Address]map[common.Address]*hexutil.Big, error) {
	ret := make(map[uint64]map[common.Address]map[common.Address]*hexutil.Big)

	tokensPerChain := make(map[uint64][]common.Address)
	for _, tokenKey := range tokenKeys {
		token, err := api.s.tokenManager.GetTokenByKey(tokenKey)
		if err != nil {
			return nil, err
		}
		tokensPerChain[token.ChainID] = append(tokensPerChain[token.ChainID], token.Address)
	}

	for chainID, tokens := range tokensPerChain {
		ret[chainID] = make(map[common.Address]map[common.Address]*hexutil.Big)
		fetchResults, err := api.s.tokenBalancesFetcher.Fetch(ctx, chainID, tokens, addresses)
		if err != nil {
			return nil, err
		}
		for account, tokenBalances := range fetchResults {
			ret[chainID][account] = make(map[common.Address]*hexutil.Big)
			for token, balance := range tokenBalances {
				ret[chainID][account][token] = (*hexutil.Big)(balance)
			}
		}
	}

	return ret, nil
}

// The client doesn't really need to force balance refreshes, it can just get them from the tokenbalances Storage.
// Every reason for which reader used to trigger a fetch is already handled by the multistandardbalance Controller.
// - Account addition
// - Active network change / Testnet mode toggle
// - App start
// - Periodic refresh
// The only outlier is the user-triggered refresh, which is handled in API method RestartWalletReloadTimer
// This will be fully refactored soon to avoid the duplicate storage and simplify the API, for now we just use the tokenbalances Storage
// and ignore the forceRefresh parameter.
func (api *API) FetchOrGetCachedWalletBalances(ctx context.Context, addresses []common.Address, forceRefresh bool) (map[common.Address][]tokentypes.StorageToken, error) {
	activeNetworks, err := api.s.rpcClient.GetNetworkManager().GetActiveNetworks()
	if err != nil {
		return nil, err
	}

	chainIDs := wcommon.NetworksToChainIDs(activeNetworks)

	return api.reader.GetCachedBalances(chainIDs, addresses)
}

type DerivedAddress struct {
	Address        types2.Address  `json:"address"`
	PublicKey      types2.HexBytes `json:"public-key,omitempty"`
	Path           string          `json:"path"`
	HasActivity    bool            `json:"hasActivity"`
	AlreadyCreated bool            `json:"alreadyCreated"`
}

func (api *API) FetchDecodedTxData(ctx context.Context, data string) (*thirdparty.DataParsed, error) {
	logutils.ZapLogger().Debug("[Wallet: FetchDecodedTxData]")

	return api.s.decoder.Decode(data)
}

// GetAllTokenLists returns all token lists (including native, custom, community token lists).
func (api *API) GetAllTokenLists(ctx context.Context) ([]*tokentypes.TokenList, error) {
	return api.s.tokenManager.GetAllTokenLists()
}

// GetAllTokens returns all unique tokens.
func (api *API) GetAllTokens(ctx context.Context) ([]*tokentypes.Token, error) {
	return api.s.tokenManager.GetAllTokens()
}

// GetTokensOfInterestForActiveNetworksMode returns all unique tokens that are of interest for the current active networks mode (testnet or mainnet).
func (api *API) GetTokensOfInterestForActiveNetworksMode(ctx context.Context) ([]*tokentypes.Token, error) {
	return api.s.tokenManager.GetTokensOfInterestForActiveNetworksMode()
}

// GetTokensForActiveNetworksMode returns all unique tokens for the current active networks mode (testnet or mainnet).
func (api *API) GetTokensForActiveNetworksMode(ctx context.Context) ([]*tokentypes.Token, error) {
	return api.s.tokenManager.GetTokensForActiveNetworksMode()
}

// GetTokenByChainAddress returns a token that matches the given chain ID and address.
func (api *API) GetTokenByChainAddress(chainID uint64, addr common.Address) (*tokentypes.Token, error) {
	return api.s.tokenManager.GetTokenByChainAddress(chainID, addr)
}

// GetTokensByChain returns all tokens for a specific chain.
func (api *API) GetTokensByChain(chainID uint64) ([]*tokentypes.Token, error) {
	return api.s.tokenManager.GetTokensByChain(chainID)
}

// GetTokensByKeys returns tokens that match the given keys.
func (api *API) GetTokensByKeys(keys []string) ([]*tokentypes.Token, error) {
	return api.s.tokenManager.GetTokensByKeys(keys)
}

func (api *API) TokenAvailableForBridgingViaHop(ctx context.Context, chainID uint64, address common.Address) bool {
	logutils.ZapLogger().Debug("call to get tokens available for bridge on chain")
	return api.s.router.TokenAvailableForBridgingViaHop(chainID, address)
}

func (api *API) DiscoverToken(ctx context.Context, chainID uint64, address common.Address) (*tokentypes.Token, error) {
	logutils.ZapLogger().Debug("call to get discover token")
	token, err := api.s.tokenManager.DiscoverToken(ctx, chainID, address)
	return token, err
}

// @deprecated
// Not used by status-desktop anymore
func (api *API) GetPendingTransactions(ctx context.Context) ([]*pendingtxtracker.PendingTransaction, error) {
	logutils.ZapLogger().Debug("wallet.api.GetPendingTransactions")
	rst, err := api.s.pendingTxManager.GetAllPending()
	logutils.ZapLogger().Debug("wallet.api.GetPendingTransactions RESULT", zap.Int("len", len(rst)))
	return rst, err
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
	return api.s.rpcClient.GetNetworkManager().Upsert(&network)
}

// @deprecated: Custom networks not currently supported. Change settings using specific API functions.
func (api *API) DeleteEthereumChain(ctx context.Context, chainID uint64) error {
	logutils.ZapLogger().Debug("call to DeleteEthereumChain")
	return api.s.rpcClient.GetNetworkManager().Delete(chainID)
}

func (api *API) SetChainUserRpcProviders(ctx context.Context, chainID uint64, rpcProviders []params.RpcProvider) error {
	logutils.ZapLogger().Debug("call to SetChainUserRpcProviders")
	return api.s.rpcClient.GetNetworkManager().SetUserRpcProviders(chainID, rpcProviders)
}

// Active chains are the ones that are available for selection across the whole application
// Providers are expected to be accessed only for active chains.
func (api *API) SetChainActive(ctx context.Context, chainID uint64, active bool) error {
	logutils.ZapLogger().Debug("call to SetChainActive")
	return api.s.rpcClient.GetNetworkManager().SetActive(chainID, active)
}

// Enabled chains are the ones taken into account when displaying balances, collectibles, activity, etc.
func (api *API) SetChainEnabled(ctx context.Context, chainID uint64, enabled bool) error {
	logutils.ZapLogger().Debug("call to SetChainEnabled")
	return api.s.rpcClient.GetNetworkManager().SetEnabled(chainID, enabled)
}

// @deprecated: Combined networks are not used anymore, use GetFlatEthereumChains instead
func (api *API) GetEthereumChains(ctx context.Context) ([]*network.CombinedNetwork, error) {
	logutils.ZapLogger().Debug("call to GetEthereumChains")
	return api.s.rpcClient.GetNetworkManager().GetCombinedNetworks()
}

func (api *API) GetFlatEthereumChains(ctx context.Context) ([]*params.Network, error) {
	logutils.ZapLogger().Debug("call to GetFlatEthereumChains")
	return api.s.rpcClient.GetNetworkManager().GetAll()
}

// @deprecated
// FetchPrices fetches prices for a given token keys and currencies. If no tokens are provided, all tokens of interest are fetched.
func (api *API) FetchPrices(ctx context.Context, tokensKeys []string, currencies []string) (map[string]map[string]float64, error) {
	logutils.ZapLogger().Debug("call to FetchPrices")
	return api.s.marketManager.FetchPrices(tokensKeys, currencies)
}

// @deprecated
// FetchTokenMarketValues fetches market values for a given token keys and currency. If no tokens are provided, all tokens of interest are fetched.
func (api *API) FetchMarketValues(ctx context.Context, tokensKeys []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	logutils.ZapLogger().Debug("call to FetchMarketValues")
	return api.s.marketManager.FetchTokenMarketValues(tokensKeys, currency)
}

func (api *API) GetHourlyMarketValues(ctx context.Context, tokenKey string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	logutils.ZapLogger().Debug("call to GetHourlyMarketValues")
	return api.s.marketManager.FetchHistoricalHourlyPrices(tokenKey, currency, limit, aggregate)
}

func (api *API) GetDailyMarketValues(ctx context.Context, tokenKey string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	logutils.ZapLogger().Debug("call to GetDailyMarketValues")
	return api.s.marketManager.FetchHistoricalDailyPrices(tokenKey, currency, limit, allData, aggregate)
}

// @deprecated
// FetchTokenDetails fetches token details for a given tokens. If no tokens are provided, all tokens of interest are fetched.
func (api *API) FetchTokenDetails(ctx context.Context, tokensKeys []string) (map[string]thirdparty.TokenDetails, error) {
	logutils.ZapLogger().Debug("call to FetchTokenDetails")
	return api.s.marketManager.FetchTokenDetails(tokensKeys)
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
	return api.s.router.GetFeesManager().EstimatedTimeLevel(ctx, chainID, gweiToWei(maxFeePerGas))
}

func gweiToWei(val *big.Float) *big.Int {
	res, _ := new(big.Float).Mul(val, big.NewFloat(1000000000)).Int(nil)
	return res
}

func (api *API) GetTransactionEstimatedTimeV2(ctx context.Context, chainID uint64, gasPrice *hexutil.Big, maxFeePerGas *hexutil.Big, maxPriorityFeePerGas *hexutil.Big) (uint, error) {
	logutils.ZapLogger().Debug("call to getTransactionEstimatedTimeV2")
	return api.s.router.GetFeesManager().EstimatedTime(ctx, chainID, maxFeePerGas.ToInt(), maxPriorityFeePerGas.ToInt())
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
	acc, err := api.s.gethManager.LoadAccount(types2.HexToAddress(derivedFrom), password)
	if err != nil {
		return nil, err
	}

	return api.getDerivedAddresses(acc, paths)
}

// Generates addresses for the provided paths derived from the provided mnemonic, response doesn't include `HasActivity` value (if you need it check `GetAddressDetails` function)
func (api *API) GetDerivedAddressesForMnemonic(ctx context.Context, mnemonic string, paths []string) ([]*DerivedAddress, error) {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	acc, err := generator2.CreateAccountFromMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return nil, err
	}

	return api.getDerivedAddresses(acc, paths)
}

// Generates addresses for the provided paths, response doesn't include `HasActivity` value (if you need it check `GetAddressDetails` function)
func (api *API) getDerivedAddresses(account *generator2.Account, paths []string) ([]*DerivedAddress, error) {
	addedAccounts, err := api.s.accountsDB.GetActiveAccounts()
	if err != nil {
		return nil, err
	}

	childrenAccounts, err := generator2.DeriveChildrenFromAccount(account, paths)
	if err != nil {
		return nil, err
	}

	derivedAddresses := make([]*DerivedAddress, 0)
	for accPath, childAccount := range childrenAccounts {
		accountInfo := childAccount.ToAccountInfo()

		derivedAddress := &DerivedAddress{
			Address:   types2.HexToAddress(accountInfo.Address),
			PublicKey: types2.Hex2Bytes(accountInfo.PublicKey),
			Path:      accPath,
		}

		for _, account := range addedAccounts {
			if derivedAddress.Address == account.Address {
				derivedAddress.AlreadyCreated = true
				break
			}
		}

		derivedAddresses = append(derivedAddresses, derivedAddress)
	}

	return derivedAddresses, nil
}

func (api *API) AddressExists(ctx context.Context, address types2.Address) (bool, error) {
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
		Address: types2.HexToAddress(params.Address),
	}
	addressExists, err := api.s.accountsDB.AddressExists(result.Address)
	if err != nil {
		return result, err
	}

	result.AlreadyCreated = addressExists

	chainIDs := params.ChainIDs
	if len(chainIDs) == 0 {
		activeNetworks, err := api.s.rpcClient.GetNetworkManager().GetActiveNetworks()
		if err != nil {
			return nil, err
		}

		chainIDs = wcommon.NetworksToChainIDs(activeNetworks)
	}

	if params.TimeoutInMilliseconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(params.TimeoutInMilliseconds)*time.Millisecond)
		defer cancel()
	}

	for _, chainID := range chainIDs {
		balance, err := api.s.tokenBalancesFetcher.FetchSingle(ctx, chainID, tokenbalances.NativeTokenAddress, common.Address(result.Address))
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
		Address: types2.HexToAddress(address),
	}
	addressExists, err := api.s.accountsDB.AddressExists(result.Address)
	if err != nil {
		return result, err
	}

	result.AlreadyCreated = addressExists

	balance, err := api.s.tokenBalancesFetcher.FetchSingle(ctx, chainID, tokenbalances.NativeTokenAddress, common.Address(result.Address))
	if err != nil {
		return result, err
	}

	result.HasActivity = balance.Cmp(big.NewInt(0)) != 0
	return result, nil
}

func (api *API) SignMessage(ctx context.Context, message types2.HexBytes, address types2.Address, password string) (string, error) {
	logutils.ZapLogger().Debug("[WalletAPI::SignMessage]", zap.Stringer("message", message), zap.Stringer("address", address))

	selectedAccount, err := api.s.gethManager.LoadAccount(address, password)
	if err != nil {
		return "", err
	}

	return api.s.transactionManager.SignMessage(message, selectedAccount.PrivateKey())
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

func (api *API) SendTransactionWithSignature(ctx context.Context, chainID uint64, txType pendingtxtracker.PendingTrxType,
	sendTxArgsJSON string, signature string) (hash types2.Hash, err error) {
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

func (api *API) SendRouterTransactionsWithSignatures(ctx context.Context, sendInputParams *requests.RouterSendTransactionsParams) {
	logutils.ZapLogger().Debug("[WalletAPI:: SendRouterTransactionsWithSignatures] sign with signatures and send")
	api.s.routeExecutionManager.SendRouterTransactionsWithSignatures(ctx, sendInputParams)
}

// ReevaluateRouterPath reevaluates the tx-fields from the router path that matches the provided pathTxIdentity and sends signal.SuggestedRoutes.
func (api *API) ReevaluateRouterPath(ctx context.Context, pathTxIdentity *requests.PathTxIdentity) error {
	logutils.ZapLogger().Debug("wallet.api.ReevaluateRouterPath")
	return api.s.routeExecutionManager.ReevaluateRouterPath(ctx, pathTxIdentity)
}

func (api *API) GetCachedCurrencyFormats() (currency.FormatPerKey, error) {
	logutils.ZapLogger().Debug("call to GetCachedCurrencyFormats")
	return api.s.currency.GetCachedCurrencyFormats()
}

func (api *API) FetchAllCurrencyFormats() (currency.FormatPerKey, error) {
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

func (api *API) UpdateActivityFilterForSession(sessionID activity.SessionID, filter activity.Filter) error {
	logutils.ZapLogger().Debug("wallet.api.UpdateActivityFilterForSession",
		zap.Int32("sessionID", int32(sessionID)),
	)

	return api.s.activity.UpdateFilterForSession(sessionID, filter)
}

func (api *API) ResetActivityFilterSession(id activity.SessionID) error {
	logutils.ZapLogger().Debug("wallet.api.ResetActivityFilterSession",
		zap.Int32("id", int32(id)),
	)

	return api.s.activity.ResetFilterSession(id)
}

func (api *API) GetMoreForActivityFilterSession(id activity.SessionID) error {
	logutils.ZapLogger().Debug("wallet.api.GetMoreForActivityFilterSession",
		zap.Int32("id", int32(id)),
	)

	return api.s.activity.GetMoreForFilterSession(id)
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
func (api *API) HashMessageEIP191(ctx context.Context, message types2.HexBytes) types2.Hash {
	logutils.ZapLogger().Debug("wallet.api.HashMessageEIP191", zap.Int("len(data)", len(message)))
	safeMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), string(message))
	return crypto.Keccak256Hash([]byte(safeMsg))
}

// SignTypedDataV4 dApps use it to execute "eth_signTypedData_v4" requests
// the formatted typed data will be prefixed with \x19\x01 based on the EIP-712
// @deprecated
func (api *API) SignTypedDataV4(typedJson string, address string, password string) (types2.HexBytes, error) {
	logutils.ZapLogger().Debug("wallet.api.SignTypedDataV4",
		zap.Int("len(typedJson)", len(typedJson)),
		zap.String("address", address),
		zap.Int("len(password)", len(password)),
	)

	account, err := api.s.gethManager.GetVerifiedWalletAccount(types2.HexToAddress(address), password)
	if err != nil {
		return types2.HexBytes{}, err
	}
	var typed signercore.TypedData
	err = json.Unmarshal([]byte(typedJson), &typed)
	if err != nil {
		return types2.HexBytes{}, err
	}

	// This is not used down the line but required by the typeddata.SignTypedDataV4 function call
	chain := new(big.Int).SetUint64(api.s.config.NetworkID)
	sig, err := typeddata.SignTypedDataV4(typed, account.PrivateKey(), chain)
	if err != nil {
		return types2.HexBytes{}, err
	}
	return types2.HexBytes(sig), err
}

// SafeSignTypedDataForDApps is used to execute requests for "eth_signTypedData"
// if legacy is true else "eth_signTypedData_v4"
// the formatted typed data won't be prefixed in case of legacy calls, as the
// old dApps implementation expects
// the chain is validate for both cases
func (api *API) SafeSignTypedDataForDApps(typedJson string, address string, password string, chainID uint64, legacy bool) (types2.HexBytes, error) {
	logutils.ZapLogger().Debug("wallet.api.SafeSignTypedDataForDApps",
		zap.Int("len(typedJson)", len(typedJson)),
		zap.String("address", address),
		zap.Int("len(password)", len(password)),
		zap.Uint64("chainID", chainID),
		zap.Bool("legacy", legacy),
	)

	account, err := api.s.gethManager.GetVerifiedWalletAccount(types2.HexToAddress(address), password)
	if err != nil {
		return types2.HexBytes{}, err
	}

	return walletconnect.SafeSignTypedDataForDApps(typedJson, account.PrivateKey(), chainID, legacy)
}

func (api *API) RestartWalletReloadTimer(ctx context.Context) error {
	api.s.multistandardBalanceController.TriggerFullFetch()
	return nil
}

func (api *API) IsChecksumValidForAddress(address string) (bool, error) {
	logutils.ZapLogger().Debug("wallet.api.isChecksumValidForAddress", zap.String("address", address))
	return abi_spec.CheckAddressChecksum(address)
}

// GetLeaderboardData returns cryptocurrency data with updated price information
func (api *API) GetLeaderboardData(ctx context.Context) ([]leaderboard.Cryptocurrency, error) {
	logutils.ZapLogger().Debug("call to GetLeaderboardData")
	if api.s.leaderboardService == nil {
		return nil, errors.New("leaderboard service not initialized")
	}
	return api.s.leaderboardService.GetCombinedData(), nil
}

func (api *API) FetchMarketTokenPageAsync(ctx context.Context, page, pageSize, sortOrder int, currency string) error {
	logutils.ZapLogger().Debug("call to GetMarketTokenPageAsync", zap.Int("page", page), zap.Int("pageSize", pageSize), zap.Int("sortOrder", sortOrder), zap.String("currency", currency))
	api.s.leaderboardService.FetchLeaderboardPageAsync(page, pageSize, sortOrder, currency)
	return nil
}

func (api *API) UnsubscribeFromLeaderboard() error {
	logutils.ZapLogger().Debug("call to UnsubscribeFromLeaderboard")
	return api.s.leaderboardService.UnsubscribeFromLeaderboard()
}

// GetFollowingAddresses fetches the list of addresses that the given user is following via EFP
func (api *API) GetFollowingAddresses(ctx context.Context, userAddress common.Address, search string, limit, offset int) ([]efp.FollowingAddress, error) {
	logutils.ZapLogger().Debug("call to GetFollowingAddresses",
		zap.String("userAddress", userAddress.Hex()),
		zap.String("search", search),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if api.s.followingManager == nil {
		return nil, errors.New("following manager not initialized")
	}

	return api.s.followingManager.FetchFollowingAddresses(ctx, userAddress, search, limit, offset)
}

// GetFollowingStats fetches the stats (following count) for a user
func (api *API) GetFollowingStats(ctx context.Context, userAddress common.Address) (int, error) {
	logutils.ZapLogger().Debug("call to GetFollowingStats",
		zap.String("userAddress", userAddress.Hex()))

	if api.s.followingManager == nil {
		return 0, errors.New("following manager not initialized")
	}

	return api.s.followingManager.FetchFollowingStats(ctx, userAddress)
}
