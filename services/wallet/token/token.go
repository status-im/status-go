package token

//go:generate go tool mockgen -source=token.go -destination=mock/token/tokenmanager.go

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"go.uber.org/zap"

	"golang.org/x/exp/maps"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/autofetcher"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/fetcher"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/manager"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/parsers"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/contracts"
	"github.com/status-im/status-go/internal/contracts/snt"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	settings2 "github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/communitytokens/communitytokensdatabase"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/community"
	defaulttokenlists "github.com/status-im/status-go/services/wallet/token/local-token-lists/default-lists"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/tokenhistoricalownership"
	"github.com/status-im/status-go/signal"
)

const (
	remoteListOfTokenListsID = "status-list-of-token-lists" // #nosec G101
	// #nosec G101
	remoteListOfTokenLists = "https://prod.market.status.im/static/lists.json"

	communityTokenListName   = "Community tokens"
	communityTokenListSource = "local"
)

type CommunityTokenImageBuilder interface {
	MakeCommunityTokenImagesURL(communityID string, chainID uint64, symbol string) string
}

type HistoricallyOwnedTokensProvider interface {
	GetOwnedTokenKeys(ownerAddress common.Address) ([]string, error)
	GetPublisher() *pubsub.Publisher
}

type ManagerInterface interface {
	GetTokenByChainAddress(chainID uint64, address common.Address) (*tokentypes.Token, error)
	GetTokensByChains(chainIDs []uint64) ([]*tokentypes.Token, error)
	GetTokensByKeys(tokenKeys []string) ([]*tokentypes.Token, error)
	FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address common.Address) (*tokentypes.Token, error)
}

// Manager is used for accessing token store. It changes the token store based on overridden tokens
type Manager struct {
	walletDB                        *sql.DB
	settings                        *settings2.Database
	ethClientGetter                 rpc.EthClientGetter
	ContractMaker                   *contracts.ContractMaker
	networkManager                  network.ManagerInterface
	communityTokensDB               *communitytokensdatabase.Database
	communityManager                community.CommunityManagerInterface
	communityTokenImageBuilder      CommunityTokenImageBuilder
	accountsDB                      *accounts.Database
	accountsPublisher               *pubsub.Publisher
	historicallyOwnedTokensProvider HistoricallyOwnedTokensProvider

	tokensManager manager.Manager

	stopCh   chan struct{}
	notifyCh chan struct{}

	publisher *pubsub.Publisher
}

func NewTokenManager(
	walletDB *sql.DB,
	ethClientGetter rpc.EthClientGetter,
	networkManager network.ManagerInterface,
	appDB *sql.DB,
	accountsPublisher *pubsub.Publisher,
	accountsDB *accounts.Database,
	communityManager *community.Manager,
	communityTokensDB *communitytokensdatabase.Database,
	communityTokenImageBuilder CommunityTokenImageBuilder,
	autoRefreshInterval time.Duration,
	autoRefreshCheckInterval time.Duration,
	publisher *pubsub.Publisher,
) (*Manager, error) {
	maker := contracts.NewContractMaker(ethClientGetter)

	settings, err := settings2.MakeNewDB(appDB)
	if err != nil {
		return nil, err
	}

	lastTokensUpdate, err := settings.LastTokensUpdate()
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		walletDB:                   walletDB,
		settings:                   settings,
		ethClientGetter:            ethClientGetter,
		ContractMaker:              maker,
		networkManager:             networkManager,
		communityManager:           communityManager,
		communityTokensDB:          communityTokensDB,
		communityTokenImageBuilder: communityTokenImageBuilder,
		accountsPublisher:          accountsPublisher,
		accountsDB:                 accountsDB,
		publisher:                  publisher,
	}

	tokensManager, err := setUpTokenListsManager(manager, walletDB, lastTokensUpdate, autoRefreshInterval, autoRefreshCheckInterval)
	if err != nil {
		logutils.ZapLogger().Error("Failed to create token lists manager", zap.Error(err))
		return nil, err
	}

	manager.tokensManager = tokensManager

	return manager, nil

}

func (tm *Manager) SetHistoricallyOwnedTokensProvider(provider HistoricallyOwnedTokensProvider) {
	tm.historicallyOwnedTokensProvider = provider
}

func setUpTokenListsManager(mng *Manager, walletDB *sql.DB, lastUpdate time.Time, autoRefreshInterval time.Duration,
	autoRefreshCheckInterval time.Duration) (manager.Manager, error) {

	wsdkFetcher := fetcher.New(fetcher.DefaultConfig())

	contentStore := NewContentStore(walletDB)

	customTokenStore := NewCustomTokenStore(mng)

	config := &manager.Config{
		AutoFetcherConfig: &autofetcher.ConfigRemoteListOfTokenLists{
			Config: autofetcher.Config{
				LastUpdate:               lastUpdate,
				AutoRefreshInterval:      autoRefreshInterval,
				AutoRefreshCheckInterval: autoRefreshCheckInterval,
			},
			RemoteListOfTokenListsFetchDetails: types.ListDetails{
				ID:        remoteListOfTokenListsID,
				SourceURL: remoteListOfTokenLists,
				Schema:    fetcher.ListOfTokenListsSchema,
			},
			RemoteListOfTokenListsParser: &parsers.StatusListOfTokenListsParser{},
		},

		MainListID: walletcommon.StatusTokenListID,

		InitialLists: map[string][]byte{
			walletcommon.StatusTokenListID:            defaulttokenlists.StatusTokenList.JsonData,
			walletcommon.UniswapTokenListID:           defaulttokenlists.UniswapTokenList.JsonData,
			walletcommon.CoingeckoEthereumTokenListID: defaulttokenlists.CoingeckoEthereumTokenList.JsonData,
			walletcommon.CoingeckoOptimismTokenListID: defaulttokenlists.CoingeckoOptimismTokenList.JsonData,
			walletcommon.CoingeckoArbitrumTokenListID: defaulttokenlists.CoingeckoArbitrumTokenList.JsonData,
			walletcommon.CoingeckoBSCTokenListID:      defaulttokenlists.CoingeckoBscTokenList.JsonData,
			walletcommon.CoingeckoBaseTokenListID:     defaulttokenlists.CoingeckoBaseTokenList.JsonData,
			walletcommon.CoingeckoLineaTokenListID:    defaulttokenlists.CoingeckoLineaTokenList.JsonData,
		},
		CustomParsers: map[string]parsers.TokenListParser{
			walletcommon.StatusTokenListID: &parsers.StatusTokenListParser{},
		},

		Chains: walletcommon.AllChainIDsAsUint64(),

		SkippedTokenKeys: walletcommon.SkippedTokenKeys(),
	}

	for key, data := range config.InitialLists {
		if len(data) == 0 {
			delete(config.InitialLists, key)
		}
	}

	return manager.New(config, wsdkFetcher, contentStore, customTokenStore)
}

func (tm *Manager) Start(ctx context.Context) error {
	tm.stopCh = make(chan struct{})

	tm.notifyCh = make(chan struct{})
	return tm.startTokenListsNotifier(ctx)
}

func (tm *Manager) startTokenListsNotifier(ctx context.Context) error {
	thirdpartyServicesEnabled, err := tm.settings.ThirdpartyServicesEnabled()
	if err != nil {
		logutils.ZapLogger().Error("failed to get if thirdparty services are enabled", zap.Error(err))
		return err
	}
	autoRefreshEnabled, err := tm.settings.AutoRefreshTokensEnabled()
	if err != nil {
		logutils.ZapLogger().Error("failed to get auto refresh tokens enabled", zap.Error(err))
		return err
	}

	autoRefresh := thirdpartyServicesEnabled && autoRefreshEnabled

	err = tm.tokensManager.Start(ctx, autoRefresh, tm.notifyCh)
	if err != nil {
		logutils.ZapLogger().Error("failed to start token lists notifier", zap.Error(err))
		return err
	}

	go func() {
		defer gocommon.LogOnPanic()
		for {
			select {
			case <-tm.stopCh:
				err := tm.tokensManager.Stop()
				if err != nil {
					logutils.ZapLogger().Error("failed to stop token lists notifier", zap.Error(err))
				}
				return
			case <-tm.notifyCh:
				err := tm.setLastTokenListsRefreshTime(time.Now().UTC())
				if err != nil {
					logutils.ZapLogger().Error("failed to set last tokens update", zap.Error(err))
				}
				if tm.publisher != nil {
					pubsub.Publish(tm.publisher, tokentypes.EventTokenListUpdated{})
				}
				signal.SendWalletEvent(signal.TokenListsUpdated, nil)
			}
		}
	}()

	go tm.watchHistoricallyOwnedTokensEvents()

	return nil
}

func (tm *Manager) Stop() {
	if tm.stopCh != nil {
		close(tm.stopCh)
		tm.stopCh = nil
	}
}

func (tm *Manager) GetTokenByChainAddress(chainID uint64, address common.Address) (*tokentypes.Token, error) {
	if token, ok := tm.tokensManager.GetTokenByChainAddress(chainID, address); ok {
		return &tokentypes.Token{Token: token}, nil
	}

	// search in the custom tokens
	communityToken, err := tm.getCustomByChainAddress(true, chainID, address)
	if err != nil {
		return nil, err
	}
	return communityToken, nil
}

func (tm *Manager) GetTokensByKeys(tokenKeys []string) ([]*tokentypes.Token, error) {
	wsdkTokens, err := tm.tokensManager.GetTokensByKeys(tokenKeys)
	if err != nil {
		return nil, err
	}

	tokens := make([]*tokentypes.Token, 0)
	for _, token := range wsdkTokens {
		tokens = append(tokens, &tokentypes.Token{Token: token})
	}

	communityTokens, err := tm.GetCustoms(true)
	if err != nil {
		return nil, err
	}
	for _, token := range communityTokens {
		if !slices.Contains(tokenKeys, token.Key()) {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (tm *Manager) GetTokenByKey(tokenKey string) (*tokentypes.Token, error) {
	if tokenKey == "" {
		return nil, errors.New("token key is empty")
	}
	chainID, address, ok := types.ChainAndAddressFromTokenKey(tokenKey)
	if !ok {
		return nil, errors.New("token key is invalid")
	}
	return tm.GetTokenByChainAddress(chainID, address)
}

func (tm *Manager) GetNativeTokenForChain(chainID uint64) (*tokentypes.Token, error) {
	return tm.GetTokenByChainAddress(chainID, common.Address{})
}

func (tm *Manager) GetTokensByChain(chainID uint64) ([]*tokentypes.Token, error) {
	wsdkTokens := tm.tokensManager.GetTokensByChain(chainID)
	tokens := make([]*tokentypes.Token, 0)
	for _, token := range wsdkTokens {
		tokens = append(tokens, &tokentypes.Token{Token: token})
	}

	communityTokens, err := tm.GetCustoms(true)
	if err != nil {
		return nil, err
	}
	for _, token := range communityTokens {
		if token.ChainID != chainID {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (tm *Manager) GetTokensByChains(chainIDs []uint64) ([]*tokentypes.Token, error) {
	allWsdkTokens := tm.tokensManager.UniqueTokens()
	tokens := make([]*tokentypes.Token, 0)
	for _, token := range allWsdkTokens {
		if slices.Contains(chainIDs, token.ChainID) {
			tokens = append(tokens, &tokentypes.Token{Token: token})
		}
	}

	communityTokens, err := tm.GetCustoms(true)
	if err != nil {
		return nil, err
	}
	for _, token := range communityTokens {
		if !slices.Contains(chainIDs, token.ChainID) {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (tm *Manager) GetTokensForActiveNetworksMode() ([]*tokentypes.Token, error) {
	testnetMode, err := tm.settings.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	chainIDs := make([]uint64, 0)
	for _, chainID := range walletcommon.AllChainIDs() {
		if chainID.IsMainnet() == testnetMode {
			continue
		}
		chainIDs = append(chainIDs, uint64(chainID))
	}

	return tm.GetTokensByChains(chainIDs)
}

// getTokenKeysForTokensOfInterestForActiveNetworksMode returns the token keys for the tokens of interest for the active networks mode (testnet or mainnet)
// On top of the used tokens keys, it adds all tokens that share the same cross chain id (because of grouping) and all mandatory tokens.
func (tm *Manager) getTokenKeysForTokensOfInterestForActiveNetworksMode() ([]string, error) {
	activeNetworks, err := tm.networkManager.GetActiveNetworks()
	if err != nil {
		return nil, err
	}

	activeChainIDs := make([]uint64, 0)
	for _, network := range activeNetworks {
		activeChainIDs = append(activeChainIDs, network.ChainID)
	}

	accounts, err := tm.accountsDB.GetAllAccounts()
	if err != nil {
		return nil, err
	}

	usedTokensKeys := make(map[string]struct{})

	if tm.historicallyOwnedTokensProvider != nil {
		for _, account := range accounts {
			userTokenKeys, err := tm.historicallyOwnedTokensProvider.GetOwnedTokenKeys(common.Address(account.Address[:]))
			if err != nil {
				return nil, err
			}
			for _, tokenKey := range userTokenKeys {
				usedTokensKeys[tokenKey] = struct{}{}
			}
		}

		tokens, err := tm.GetTokensByKeys(maps.Keys(usedTokensKeys))
		if err != nil {
			return nil, err
		}

		// Because of grouping it's important to add all tokens that share the same cross chain id to the used tokens keys
		for _, token := range tokens {
			if token.CrossChainID == "" {
				continue
			}
			usedTokensKeys[token.Key()] = struct{}{}
		}
	}

	// It's also important to add all mandatory tokens to the used tokens keys
	for _, tokenKey := range walletcommon.MandatoryTokens() {
		if _, ok := usedTokensKeys[tokenKey]; ok {
			continue
		}
		chainID, _, ok := types.ChainAndAddressFromTokenKey(tokenKey)
		if !ok {
			continue
		}
		if slices.Contains(activeChainIDs, chainID) {
			continue
		}
		usedTokensKeys[tokenKey] = struct{}{}
	}

	return maps.Keys(usedTokensKeys), nil
}

func (tm *Manager) GetTokensOfInterestForActiveNetworksMode() ([]*tokentypes.Token, error) {
	tokensKeys, err := tm.getTokenKeysForTokensOfInterestForActiveNetworksMode()
	if err != nil {
		return nil, err
	}
	return tm.GetTokensByKeys(tokensKeys)
}

// GetTokensForFetchingMarketData returns all unique tokens for fetching market data from Coingecko (doesn't affect CryptoCompare cause it maps tokens differently, by symbol)
// Special handling for test tokens, for fetching market data from Coingecko, cause their API doesn't support test tokens
// Corresponding mainnet tokens are needed to fetch market data for test tokens.
// Returns list of test tokens and list of mainnet tokens that have a cross chain id set.
func (tm *Manager) GetTokensForFetchingMarketData() ([]*tokentypes.Token, error) {
	testnetMode, err := tm.settings.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	// If not in testnet mode, use the regular tokens
	if !testnetMode {
		return tm.GetTokensOfInterestForActiveNetworksMode()
	}

	// Test tokens handling...
	// Use all test tokens and add to the list only mainnet tokens that have a cross chain id set
	tokensKeys, err := tm.getTokenKeysForTokensOfInterestForActiveNetworksMode()
	if err != nil {
		return nil, err
	}
	return tm.GetTokensByKeysForFetchingMarketData(tokensKeys)
}

// GetTokensByKeysForFetchingMarketData returns all unique tokens for fetching market data from Coingecko (doesn't affect CryptoCompare cause it maps tokens differently, by symbol)
// Special handling for test tokens, for fetching market data from Coingecko, cause their API doesn't support test tokens
// Corresponding mainnet tokens are needed to fetch market data for test tokens.
// Returns list of test tokens that match the given token keys and corresponding mainnet tokens for those test tokens that have a cross chain id set.
func (tm *Manager) GetTokensByKeysForFetchingMarketData(tokenKeys []string) ([]*tokentypes.Token, error) {
	testnetMode, err := tm.settings.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	// If not in testnet mode, use the regular tokens
	if !testnetMode {
		return tm.GetTokensByKeys(tokenKeys)
	}

	// Test tokens handling...
	// Use corresponding mainnet tokens for the test tokens that contains the same cross chain id
	mainnetTokenKeysByCrossChainIDs := make(map[string][]string, 0) // keeps token keys of all mainnet tokens by cross chain id
	allWsdkTokens := tm.tokensManager.UniqueTokens()
	tokens := make([]*tokentypes.Token, 0)
	for _, token := range allWsdkTokens {
		if token.CrossChainID != "" && walletcommon.ChainID(token.ChainID).IsMainnet() {
			mainnetTokenKeysByCrossChainIDs[token.CrossChainID] = append(mainnetTokenKeysByCrossChainIDs[token.CrossChainID], token.Key())
		}
		if !slices.Contains(tokenKeys, token.Key()) {
			continue
		}
		tokens = append(tokens, &tokentypes.Token{Token: token})
	}

	mainnetTokenKeys := make([]string, 0) // keeps token keys of mainnet tokens that have the same cross chain id as the test tokens
	for _, token := range tokens {
		crossChainID := token.CrossChainID
		if crossChainID == "" {
			continue
		}
		// Special handling for status test token STT, cause even it's the same token belongs to different group and has different symbol SNT/STT.
		if crossChainID == walletcommon.StatusTestTokenCrossChainID {
			crossChainID = walletcommon.StatusMainnetTokenCrossChainID
		}
		if _, ok := mainnetTokenKeysByCrossChainIDs[crossChainID]; !ok {
			continue
		}
		mainnetTokenKeys = append(mainnetTokenKeys, mainnetTokenKeysByCrossChainIDs[crossChainID]...)
	}

	mainnetTokens, err := tm.GetTokensByKeys(mainnetTokenKeys)
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, mainnetTokens...)

	return tokens, nil
}

func (tm *Manager) GetAllTokens() ([]*tokentypes.Token, error) {
	wsdkTokens := tm.tokensManager.UniqueTokens()
	tokens := make([]*tokentypes.Token, 0)
	for _, token := range wsdkTokens {
		tokens = append(tokens, &tokentypes.Token{Token: token})
	}

	communityTokens, err := tm.GetCustoms(true)
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, communityTokens...)
	return tokens, nil
}

func (tm *Manager) GetTokenList(id string) (*tokentypes.TokenList, error) {
	if id == walletcommon.CommunityTokenListID {
		communityTokens, err := tm.GetCustoms(true)
		if err != nil {
			return nil, err
		}
		return &tokentypes.TokenList{
			TokenList: &types.TokenList{
				ID:     walletcommon.CommunityTokenListID,
				Name:   communityTokenListName,
				Source: communityTokenListSource,
			},
			Tokens: communityTokens,
		}, nil
	}

	wsdkTokenList, ok := tm.tokensManager.TokenList(id)
	if ok {
		tokenList := &tokentypes.TokenList{TokenList: wsdkTokenList}
		tokenList.Tokens = make([]*tokentypes.Token, 0)
		for _, token := range wsdkTokenList.Tokens {
			tokenList.Tokens = append(tokenList.Tokens, &tokentypes.Token{Token: token})
		}
		return tokenList, nil
	}

	return nil, fmt.Errorf("token list %s not found", id)
}

func (tm *Manager) GetAllTokenLists() ([]*tokentypes.TokenList, error) {
	wsdkTokenLists := tm.tokensManager.TokenLists()
	tokenLists := make([]*tokentypes.TokenList, 0)
	for _, wsdkTokenList := range wsdkTokenLists {
		tokenList := &tokentypes.TokenList{TokenList: wsdkTokenList}
		tokenList.Tokens = make([]*tokentypes.Token, 0)
		for _, token := range wsdkTokenList.Tokens {
			tokenList.Tokens = append(tokenList.Tokens, &tokentypes.Token{Token: token})
		}
		tokenLists = append(tokenLists, tokenList)
	}

	communityTokens, err := tm.GetCustoms(true)
	if err != nil {
		return nil, err
	}

	tokenList := &tokentypes.TokenList{TokenList: &types.TokenList{
		ID:     walletcommon.CommunityTokenListID,
		Name:   communityTokenListName,
		Source: communityTokenListSource,
	}}

	tokenList.Tokens = make([]*tokentypes.Token, 0)
	tokenList.Tokens = append(tokenList.Tokens, communityTokens...)

	tokenLists = append(tokenLists, tokenList)

	return tokenLists, nil
}

func (tm *Manager) setLastTokenListsRefreshTime(time time.Time) error {
	return tm.settings.SaveSettingField(settings2.LastTokensUpdate, time)
}

func (tm *Manager) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address common.Address) (*tokentypes.Token, error) {
	token, err := tm.GetTokenByChainAddress(chainID, address)
	if err == nil && token != nil {
		if token.CommunityData == nil && !token.CustomToken {
			return token, nil
		}

		tm.discoverTokenCommunityID(ctx, token, address)
		return token, nil
	}

	// Not found among known tokens, discover it and insert it into the `tokens` table
	token, err = tm.DiscoverToken(ctx, chainID, address)
	if err != nil {
		return nil, err
	}

	err = tm.UpsertCustom(*token)
	if err != nil {
		return nil, err
	}

	tm.discoverTokenCommunityID(ctx, token, address)
	return token, nil
}

func (tm *Manager) FindSNT(chainID uint64) (*tokentypes.Token, error) {
	address, err := snt.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	token, err := tm.GetTokenByChainAddress(chainID, address)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (tm *Manager) DiscoverToken(ctx context.Context, chainID uint64, address common.Address) (*tokentypes.Token, error) {
	caller, err := tm.ContractMaker.NewERC20(chainID, address)
	if err != nil {
		return nil, err
	}

	name, err := caller.Name(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	symbol, err := caller.Symbol(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	decimal, err := caller.Decimals(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	return &tokentypes.Token{
		Token: &types.Token{
			Address:  address,
			Name:     name,
			Symbol:   symbol,
			Decimals: uint(decimal),
			ChainID:  chainID,
		},
	}, nil
}

func (tm *Manager) GetPublisher() *pubsub.Publisher {
	return tm.publisher
}

func (tm *Manager) watchHistoricallyOwnedTokensEvents() {
	defer gocommon.LogOnPanic()

	if tm.historicallyOwnedTokensProvider == nil {
		logutils.ZapLogger().Warn("historicallyOwnedTokensProvider is nil, token historical ownership events will not be monitored")
		return
	}

	ch, unsub := pubsub.Subscribe[tokenhistoricalownership.EventOwnershipChanged](tm.historicallyOwnedTokensProvider.GetPublisher(), 10)
	defer unsub()

	for {
		select {
		case <-tm.stopCh:
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			// Signal token list change
			tm.notifyCh <- struct{}{}
		}
	}
}
