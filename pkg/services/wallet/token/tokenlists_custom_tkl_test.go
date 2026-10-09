package token

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func TestTKLCustomSQLAndCommunity(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	facade, err := newTKLReadManager(m, []uint64{1}, time.Time{})
	require.NoError(t, err)
	defer func() { _ = facade.Stop() }()
	m.tokensManager = facade
	row := tokentypes.Token{Token: &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}}
	require.NoError(t, m.UpsertCustom(row)) // config write before Start
	require.NoError(t, facade.Start(context.Background(), false, nil))
	row.Symbol = "UPDATED"
	require.NoError(t, m.UpsertCustom(row))
	found, ok := facade.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok)
	require.Equal(t, "UPDATED", found.Symbol)
	_, err = m.walletDB.Exec(`CREATE TRIGGER reject_custom BEFORE INSERT ON tokens BEGIN SELECT RAISE(FAIL, 'storage failure'); END`)
	require.NoError(t, err)
	row.Symbol = "FAILED"
	require.Error(t, m.UpsertCustom(row))
	found, ok = facade.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok)
	require.Equal(t, "UPDATED", found.Symbol)
	_, err = m.walletDB.Exec("DROP TRIGGER reject_custom")
	require.NoError(t, err)
	row.CommunityData = &tokentypes.CommunityData{ID: "community"}
	require.NoError(t, m.UpsertCustom(row))
	_, ok = facade.GetTokenByChainAddress(1, row.Address)
	require.False(t, ok, "community rows must not remain in ordinary catalogue")
	communities, err := m.GetCustoms(true)
	require.NoError(t, err)
	require.Len(t, communities, 1)
	require.NoError(t, m.DeleteCustom(1, row.Address))
	communities, err = m.GetCustoms(true)
	require.NoError(t, err)
	require.Empty(t, communities)
}

func TestTKLIneligibleCustomPersistsWithoutStaleCatalogueEntry(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	facade, err := newTKLReadManager(m, []uint64{1}, time.Time{})
	require.NoError(t, err)
	defer func() { _ = facade.Stop() }()
	m.tokensManager = facade
	require.NoError(t, facade.Start(context.Background(), false, nil))
	row := tokentypes.Token{Token: &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 24}}
	require.NoError(t, m.UpsertCustom(row))
	_, ok := facade.GetTokenByChainAddress(1, row.Address)
	require.False(t, ok)
	stored, err := m.GetCustoms(false)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	require.Equal(t, uint(24), stored[0].Decimals)
	row.Decimals = 18
	require.NoError(t, m.UpsertCustom(row))
	_, ok = facade.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok)
	_, err = m.walletDB.Exec(`CREATE TRIGGER reject_custom BEFORE INSERT ON tokens BEGIN SELECT RAISE(FAIL, 'storage failure'); END`)
	require.NoError(t, err)
	row.Decimals = 24
	require.Error(t, m.UpsertCustom(row))
	found, ok := facade.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok)
	require.Equal(t, uint(18), found.Decimals)
	_, err = m.walletDB.Exec("DROP TRIGGER reject_custom")
	require.NoError(t, err)
	require.NoError(t, m.UpsertCustom(row))
	_, ok = facade.GetTokenByChainAddress(1, row.Address)
	require.False(t, ok)
	stored, err = m.GetCustoms(false)
	require.NoError(t, err)
	require.Equal(t, uint(24), stored[0].Decimals)
}

func TestConfigCustomMetadataMatchesPersistence(t *testing.T) {
	{
		m, _, cleanup := setupTestTokenManager(t)
		row := tokentypes.Token{Token: &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18, LogoURI: "https://example.org/icon.png", CrossChainID: "custom-id"}}
		require.NoError(t, m.UpsertCustom(row))
		require.NoError(t, m.Start(context.Background()))
		found, ok := m.tokensManager.GetTokenByChainAddress(1, row.Address)
		require.True(t, ok)
		require.Empty(t, found.LogoURI)
		require.Empty(t, found.CrossChainID)
		cleanup()
	}
}

func TestNativeCatalogueNetworkGate(t *testing.T) {
	m, _, cleanup := setupTestTokenManager(t)
	defer cleanup()
	selected := m.tokensManager
	require.IsType(t, &tklmanager.Manager{}, selected)
	require.NotNil(t, m.CataloguePausable())
	networks := json.RawMessage(`{}`)
	require.NoError(t, m.settings.CreateSettings(settings.Settings{Networks: &networks, ThirdpartyServicesEnabled: false}, params.NodeConfig{}))
	require.NoError(t, m.Start(context.Background()))
	defer m.Stop()
	require.NotEmpty(t, selected.UniqueTokens())
	// The fixture has third-party services disabled. Manual refresh must obey
	// that gate too, independently of the auto-refresh preference.
	require.ErrorIs(t, selected.TriggerRefresh(context.Background()), tkl.Aborted)
}

func TestTKLExistingWalletRegressions(t *testing.T) {
	t.Run("AccountCleanup", func(t *testing.T) { testRemoveTokenBalanceOnEventAccountRemoved(t) })
	t.Run("TokenLists", func(t *testing.T) { testTokenListsValidity(t) })
	t.Run("TokensOfInterest", func(t *testing.T) { testTokensOfInterest(t) })
}

func TestTKLDiscoveredCommunityLeavesCatalogue(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	facade, err := newTKLReadManager(m, []uint64{1}, time.Time{})
	require.NoError(t, err)
	defer func() { _ = facade.Stop() }()
	m.tokensManager = facade
	row := tokentypes.Token{Token: &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}}
	require.NoError(t, m.UpsertCustom(row))
	require.NoError(t, facade.Start(context.Background(), false, nil))
	require.NoError(t, m.setDiscoveredCommunityID(context.Background(), row.Token, "community"))
	_, ok := facade.GetTokenByChainAddress(1, row.Address)
	require.False(t, ok)
	communities, err := m.GetCustoms(true)
	require.NoError(t, err)
	require.Len(t, communities, 1)
}

func TestTKLStopCancelsCustomSQL(t *testing.T) {
	testTKLStopCancelsCustomSQL(t, true)
}

func TestTKLStopCancelsPrestartCustomBootstrap(t *testing.T) {
	testTKLStopCancelsCustomSQL(t, false)
}

func testTKLStopCancelsCustomSQL(t *testing.T, start bool) {
	m, _, cleanup := setupTestTokenManager(t)
	defer cleanup()
	selected := m.tokensManager
	if start {
		require.NoError(t, m.Start(context.Background()))
	}
	defer m.Stop()
	m.walletDB.SetMaxOpenConns(1)
	held, err := m.walletDB.Conn(context.Background())
	require.NoError(t, err)
	defer held.Close()
	before := m.walletDB.Stats().WaitCount
	done := make(chan error, 1)
	go func() {
		done <- m.UpsertCustom(tokentypes.Token{Token: &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}})
	}()
	require.Eventually(t, func() bool { return m.walletDB.Stats().WaitCount > before }, time.Second, time.Millisecond)
	stopped := make(chan struct{})
	go func() { m.Stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop blocked behind custom SQL")
	}
	require.ErrorIs(t, <-done, context.Canceled)
	require.Empty(t, selected.UniqueTokens())
}
