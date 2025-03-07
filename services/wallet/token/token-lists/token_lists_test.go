package tokenlists

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/helpers"
)

func setupTestAppDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "app-tests")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func initSettings(appDb *sql.DB, autoRefreshEnabled bool) (*settings.Database, error) {
	settingsDB, err := settings.MakeNewDB(appDb)
	if err != nil {
		return nil, err
	}

	var (
		config = params.NodeConfig{
			NetworkID: 10,
			DataDir:   "test",
		}
		networks    = json.RawMessage("{}")
		settingsObj = settings.Settings{
			Networks:                 &networks,
			AutoRefreshTokensEnabled: autoRefreshEnabled,
		}
	)

	err = settingsDB.CreateSettings(settingsObj, config)
	if err != nil {
		return nil, err
	}

	return settingsDB, nil
}

type listsUpdatedEnvelope struct {
	Type string `json:"type"`
}

func setupSignalHandler(t *testing.T) (chan struct{}, func()) {
	notifyCh := make(chan struct{})
	signalHandler := signal.MobileSignalHandler(func(data []byte) {
		var envelope signal.Envelope
		err := json.Unmarshal(data, &envelope)
		require.NoError(t, err)
		if envelope.Type == string(signal.TokenListsUpdated) {
			var response listsUpdatedEnvelope
			err := json.Unmarshal(data, &response)
			require.NoError(t, err)

			notifyCh <- struct{}{}
		}
	})
	signal.SetMobileSignalHandler(signalHandler)

	closeFn := func() {
		signal.ResetMobileSignalHandler()
		close(notifyCh)
	}

	return notifyCh, closeFn
}

func TestTokensLists(t *testing.T) {
	var tokensLists *TokenLists
	appDb, closeAppDb := setupTestAppDB(t)
	walletDb, closeWalletDb := fetcher.SetupTestWalletDB(t)
	server, closeServer := fetcher.GetTestServer()

	notifyCh, closeSignalHandler := setupSignalHandler(t)

	t.Cleanup(func() {
		closeSignalHandler()
		tokensLists.Stop()
		closeAppDb()
		closeWalletDb()
		closeServer()
	})

	// init settings with auto-refresh disabled
	settingsDB, err := initSettings(appDb, false)
	require.NoError(t, err)
	require.NotNil(t, settingsDB)

	tokensLists, err = NewTokenLists(appDb, walletDb)
	require.NoError(t, err)

	lastUpdate, err := tokensLists.LastTokensUpdate()
	require.NoError(t, err)
	require.True(t, lastUpdate.IsZero())

	// before starting the auto-refresh, we should have no token lists
	allTokens := tokensLists.GetUniqueTokens()
	require.NoError(t, err)
	require.Len(t, allTokens, 0)

	// start the auto-refresh process with a 5 second interval, and 1 second check interval
	const autoRefreshInterval = 3 * time.Second
	const autoRefreshCheckInterval = 1 * time.Second

	tokensLists.Start(server.URL, autoRefreshInterval, autoRefreshCheckInterval)

	// immediately after starting the server check if the initial token lists are loaded
	allTokensLists := tokensLists.GetTokensLists()
	require.Len(t, allTokensLists, 3)
	allTokens = tokensLists.GetUniqueTokens()
	allTokensCount := len(allTokens)
	require.True(t, allTokensCount > 0)

	// wait for the auto-refresh to try to fetch the token lists, while auto-refresh is disabled
	select {
	case <-notifyCh:
		t.FailNow()
		break
	case <-time.After(autoRefreshCheckInterval + autoRefreshInterval):
		lastUpdatedTime, err := tokensLists.LastTokensUpdate()
		require.NoError(t, err)
		require.True(t, lastUpdatedTime.Unix() == getTheLatestFetchTimeOfDefaultTokenLists().Unix())
		break
	}

	// the token lists should not be updated
	allTokens = tokensLists.GetUniqueTokens()
	require.True(t, allTokensCount == len(allTokens))

	// the list should not contain "special" testing purpose uniswap and aave tokens
	foundSpecialTokens := false
	for _, token := range allTokens {
		if token.Symbol == fetcher.AaveSpecialTokenSymbol || token.Symbol == fetcher.UniswapSpecialTokenSymbol {
			foundSpecialTokens = true
			break
		}
	}
	require.False(t, foundSpecialTokens)

	// enable auto-refresh
	err = settingsDB.SaveSettingField(settings.AutoRefreshTokensEnabled, true)
	require.NoError(t, err)

	// wait for the auto-refresh to try to fetch the token lists, while auto-refresh is enabled
	select {
	case <-notifyCh:
		lastUpdatedTime, err := tokensLists.LastTokensUpdate()
		require.NoError(t, err)
		require.False(t, lastUpdatedTime.IsZero())
		break
	case <-time.After(autoRefreshCheckInterval + autoRefreshInterval):
		t.FailNow()
		break
	}

	tokensList := tokensLists.GetTokensList("uniswap")
	require.NotNil(t, tokensList)
	require.Equal(t, fetcher.UniswapTokensListVersion, tokensList.GetVersion())

	tokensList = tokensLists.GetTokensList("aave")
	require.NotNil(t, tokensList)
	require.Equal(t, fetcher.AaveTokensListVersion, tokensList.GetVersion())

	// the token lists should be updated
	allTokens = tokensLists.GetUniqueTokens()
	require.True(t, allTokensCount != len(allTokens))

	lastUpdate, err = tokensLists.LastTokensUpdate()
	require.NoError(t, err)
	require.False(t, lastUpdate.IsZero())

	// the list should contain "special" testing purpose uniswap and aave tokens
	foundSpecialUniswapTokens := false
	foundSpecialAaveTokens := false
	for _, token := range allTokens {
		if foundSpecialUniswapTokens && foundSpecialAaveTokens {
			break
		}
		if token.Symbol == fetcher.UniswapSpecialTokenSymbol && token.Name == fetcher.UniswapSpecialTokenName {
			foundSpecialUniswapTokens = true
			continue
		}
		if token.Symbol == fetcher.AaveSpecialTokenSymbol && token.Name == fetcher.AaveSpecialTokenName {
			foundSpecialAaveTokens = true
			continue
		}
	}
	require.True(t, foundSpecialUniswapTokens)
	require.True(t, foundSpecialAaveTokens)
}
