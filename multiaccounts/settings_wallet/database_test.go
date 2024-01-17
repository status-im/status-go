package walletsettings

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
)

var (
	config = params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}
	networks    = json.RawMessage("{}")
	settingsObj = settings.Settings{
		Networks: &networks,
	}
)

func setupTestDB(t *testing.T) (*WalletSettings, func()) {
	db, stop, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "settings-wallet-tests-")
	require.NoError(t, err)
	settDb, err := settings.MakeNewDB(db)
	require.NoError(t, err)
	err = settDb.CreateSettings(settingsObj, config)
	require.NoError(t, err)
	walletSettings := NewWalletSettings(db)
	return walletSettings, func() { require.NoError(t, stop()) }
}

func TestSetClockOfLastTokenPreferencesChange(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	clock, err := walletSettings.GetClockOfLastTokenPreferencesChange()
	require.NoError(t, err)
	require.Equal(t, uint64(0), clock)

	err = walletSettings.setClockOfLastTokenPreferencesChange(nil, 123)
	require.Error(t, err)

	tx, err := walletSettings.db.Begin()
	require.NoError(t, err)
	err = walletSettings.setClockOfLastTokenPreferencesChange(tx, 123)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	clock, err = walletSettings.GetClockOfLastTokenPreferencesChange()
	require.NoError(t, err)
	require.Equal(t, uint64(123), clock)
}

func TestGetTokenPreferencesEmpty(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	preferences, err := walletSettings.GetTokenPreferences(true)
	require.NoError(t, err)
	require.Equal(t, 0, len(preferences))
	preferences, err = walletSettings.GetTokenPreferences(false)
	require.NoError(t, err)
	require.Equal(t, 0, len(preferences))
}

func TestUpdateTokenPreferencesEmpty(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	err := walletSettings.UpdateTokenPreferences([]TokenPreferences{}, false, false, 0)
	require.Error(t, err)
	err = walletSettings.UpdateTokenPreferences([]TokenPreferences{}, false, true, 0)
	require.Error(t, err)
}

func TestUpdateTokenPreferencesTestnet(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	err := walletSettings.UpdateTokenPreferences([]TokenPreferences{
		{"SNT", 0, -1, false, ""},
	}, false, true, 0)
	require.NoError(t, err)

	// Mainnet is not affected by testnet preferences
	preferences, err := walletSettings.GetTokenPreferences(false)
	require.NoError(t, err)
	require.Equal(t, 0, len(preferences))

	preferences, err = walletSettings.GetTokenPreferences(true)
	require.NoError(t, err)
	require.Equal(t, 1, len(preferences))
	require.Equal(t, "SNT", preferences[0].Key)
	require.Equal(t, 0, preferences[0].Position)
	require.Equal(t, -1, preferences[0].GroupPosition)
	require.Equal(t, false, preferences[0].Visible)
	require.Equal(t, "", preferences[0].CommunityID)

	// Inserting into testnet doesn't affect mainnet
	err = walletSettings.UpdateTokenPreferences([]TokenPreferences{
		{"ABC0", 0, -1, true, ""},
		{"ABC1", 1, -1, true, ""},
	}, false, false, 0)
	require.NoError(t, err)
	err = walletSettings.UpdateTokenPreferences([]TokenPreferences{
		{"ABC0", 1, -1, true, ""},
		{"ABC1", 0, -1, true, ""},
	}, false, true, 0)
	require.NoError(t, err)

	// Having same symbols on mainnet and testnet is allowed
	preferences, err = walletSettings.GetTokenPreferences(true)
	require.NoError(t, err)
	require.Equal(t, 2, len(preferences))
	require.Equal(t, "ABC0", preferences[0].Key)
	require.Equal(t, 1, preferences[0].Position)
	require.Equal(t, -1, preferences[0].GroupPosition)
	require.Equal(t, true, preferences[0].Visible)
	require.Equal(t, "", preferences[0].CommunityID)
	require.Equal(t, "ABC1", preferences[1].Key)
	require.Equal(t, 0, preferences[1].Position)
	require.Equal(t, -1, preferences[1].GroupPosition)
	require.Equal(t, true, preferences[1].Visible)
	require.Equal(t, "", preferences[1].CommunityID)

	preferences, err = walletSettings.GetTokenPreferences(false)
	require.NoError(t, err)
	require.Equal(t, 2, len(preferences))
	require.Equal(t, "ABC0", preferences[0].Key)
	require.Equal(t, 0, preferences[0].Position)
	require.Equal(t, -1, preferences[0].GroupPosition)
	require.Equal(t, true, preferences[0].Visible)
	require.Equal(t, "", preferences[0].CommunityID)
	require.Equal(t, "ABC1", preferences[1].Key)
	require.Equal(t, 1, preferences[1].Position)
	require.Equal(t, -1, preferences[1].GroupPosition)
	require.Equal(t, true, preferences[1].Visible)
	require.Equal(t, "", preferences[1].CommunityID)
}

func TestUpdateTokenPreferencesRollback(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	err := db.UpdateTokenPreferences([]TokenPreferences{
		{"SNT", 0, -1, false, ""},
	}, false, true, 0)
	require.NoError(t, err)

	// Duplicate is not allowed
	err = db.UpdateTokenPreferences([]TokenPreferences{
		{"ABC", 0, -1, false, ""},
		{"ABC", 0, -1, true, ""},
	}, false, true, 0)
	require.Error(t, err)

	// Rolled back to previous state
	preferences, err := db.GetTokenPreferences(true)
	require.NoError(t, err)
	require.Equal(t, 1, len(preferences))
	require.Equal(t, "SNT", preferences[0].Key)
	require.Equal(t, 0, preferences[0].Position)
	require.Equal(t, -1, preferences[0].GroupPosition)
	require.Equal(t, false, preferences[0].Visible)
	require.Equal(t, "", preferences[0].CommunityID)
}

func TestTokenPrefrencesGroupByCommunity(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	communityID01 := "0x000001"
	communityID02 := "0x000002"

	err := walletSettings.UpdateTokenPreferences([]TokenPreferences{
		{"SNT", 0, -1, true, ""},
		{"ETH", 1, -1, true, ""},
		{"T01", 0, -1, true, communityID01},
		{"T02", 1, -1, true, communityID01},
	}, false, true, 0)
	require.NoError(t, err)

	preferences, err := walletSettings.GetTokenPreferences(true)
	require.NoError(t, err)
	require.Equal(t, 4, len(preferences))
	require.Equal(t, "SNT", preferences[0].Key)
	require.Equal(t, 0, preferences[0].Position)
	require.Equal(t, -1, preferences[0].GroupPosition)
	require.Equal(t, true, preferences[0].Visible)
	require.Equal(t, "", preferences[0].CommunityID)
	require.Equal(t, "ETH", preferences[1].Key)
	require.Equal(t, 1, preferences[1].Position)
	require.Equal(t, -1, preferences[1].GroupPosition)
	require.Equal(t, true, preferences[1].Visible)
	require.Equal(t, "", preferences[1].CommunityID)
	require.Equal(t, "T01", preferences[2].Key)
	require.Equal(t, 0, preferences[2].Position)
	require.Equal(t, -1, preferences[2].GroupPosition)
	require.Equal(t, true, preferences[2].Visible)
	require.Equal(t, communityID01, preferences[2].CommunityID)
	require.Equal(t, "T02", preferences[3].Key)
	require.Equal(t, 1, preferences[3].Position)
	require.Equal(t, -1, preferences[3].GroupPosition)
	require.Equal(t, true, preferences[3].Visible)
	require.Equal(t, communityID01, preferences[3].CommunityID)

	err = walletSettings.UpdateTokenPreferences([]TokenPreferences{
		{"SNT", 0, -1, true, ""},
		{"ETH", 1, -1, true, ""},
		{"T01", 0, 1, true, communityID01},
		{"T02", 1, 0, true, communityID01},
		{"T03", 0, 0, true, communityID02},
	}, false, true, 0)
	require.NoError(t, err)

	preferences, err = walletSettings.GetTokenPreferences(true)
	require.NoError(t, err)
	require.Equal(t, 5, len(preferences))
	require.Equal(t, "SNT", preferences[0].Key)
	require.Equal(t, 0, preferences[0].Position)
	require.Equal(t, -1, preferences[0].GroupPosition)
	require.Equal(t, true, preferences[0].Visible)
	require.Equal(t, "", preferences[0].CommunityID)
	require.Equal(t, "ETH", preferences[1].Key)
	require.Equal(t, 1, preferences[1].Position)
	require.Equal(t, -1, preferences[1].GroupPosition)
	require.Equal(t, true, preferences[1].Visible)
	require.Equal(t, "", preferences[1].CommunityID)
	require.Equal(t, "T01", preferences[2].Key)
	require.Equal(t, 0, preferences[2].Position)
	require.Equal(t, 1, preferences[2].GroupPosition)
	require.Equal(t, true, preferences[2].Visible)
	require.Equal(t, communityID01, preferences[2].CommunityID)
	require.Equal(t, "T02", preferences[3].Key)
	require.Equal(t, 1, preferences[3].Position)
	require.Equal(t, 0, preferences[3].GroupPosition)
	require.Equal(t, true, preferences[3].Visible)
	require.Equal(t, communityID01, preferences[3].CommunityID)
	require.Equal(t, "T03", preferences[4].Key)
	require.Equal(t, 0, preferences[4].Position)
	require.Equal(t, 0, preferences[4].GroupPosition)
	require.Equal(t, true, preferences[4].Visible)
	require.Equal(t, communityID02, preferences[4].CommunityID)

	// Insert not full group positioning (one group has -1 group position)
	err = walletSettings.UpdateTokenPreferences([]TokenPreferences{
		{"SNT", 0, -1, true, ""},
		{"ETH", 1, -1, true, ""},
		{"T01", 0, 1, true, communityID01},
		{"T02", 1, 0, true, communityID01},
		{"T03", 0, -1, true, communityID02},
	}, true, true, 0)
	require.Error(t, err)

	// Group by community is disabled so there's no check for proper grouping
	err = walletSettings.UpdateTokenPreferences([]TokenPreferences{
		{"SNT", 0, -1, true, ""},
		{"ETH", 1, -1, true, ""},
		{"T01", 0, 1, true, communityID01},
		{"T02", 1, 0, true, communityID01},
		{"T03", 0, -1, true, communityID02},
	}, false, true, 0)
	require.NoError(t, err)

	// Insert not full group positioning with invsibile item set
	err = walletSettings.UpdateTokenPreferences([]TokenPreferences{
		{"SNT", 0, -1, true, ""},
		{"ETH", 1, -1, true, ""},
		{"T01", 0, 1, true, communityID01},
		{"T02", 1, 0, true, communityID01},
		{"T03", 0, -1, false, communityID02},
	}, true, true, 0)
	require.NoError(t, err)
}

func TestUpdateTokenPreferencesSettings(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	preferences := []TokenPreferences{
		{"SNT", 0, -1, true, ""},
		{"ETH", 1, -1, true, ""},
		{"T01", 0, 1, true, "0x000001"},
		{"T02", 1, 0, true, "0x000001"},
	}

	err := walletSettings.UpdateTokenPreferences(preferences, true, true, 123)
	require.NoError(t, err)

	// Verify that the preferences are updated correctly
	prefs, err := walletSettings.GetTokenPreferences(true)
	require.NoError(t, err)
	require.Equal(t, len(preferences), len(prefs))
	for i := range preferences {
		require.Equal(t, preferences[i], prefs[i])
	}

	// Verify that the clock is updated
	clock, err := walletSettings.GetClockOfLastTokenPreferencesChange()
	require.NoError(t, err)
	require.True(t, clock > 0)
}

func TestSetClockOfLastCollectiblePreferencesChange(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	clock, err := walletSettings.GetClockOfLastCollectiblePreferencesChange()
	require.NoError(t, err)
	require.Equal(t, uint64(0), clock)

	err = walletSettings.setClockOfLastCollectiblePreferencesChange(nil, 123)
	require.Error(t, err)

	tx, err := walletSettings.db.Begin()
	require.NoError(t, err)
	err = walletSettings.setClockOfLastCollectiblePreferencesChange(tx, 123)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	clock, err = walletSettings.GetClockOfLastCollectiblePreferencesChange()
	require.NoError(t, err)
	require.Equal(t, uint64(123), clock)
}

func TestGetCollectiblePreferencesEmpty(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	preferences, err := walletSettings.GetCollectiblePreferences(true)
	require.NoError(t, err)
	require.Equal(t, 0, len(preferences))
	preferences, err = walletSettings.GetCollectiblePreferences(false)
	require.NoError(t, err)
	require.Equal(t, 0, len(preferences))
}

func TestUpdateCollectiblePreferencesEmpty(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	err := walletSettings.UpdateCollectiblePreferences([]CollectiblePreferences{}, false, false, false, 0)
	require.Error(t, err)
	err = walletSettings.UpdateCollectiblePreferences([]CollectiblePreferences{}, false, false, true, 0)
	require.Error(t, err)
}

func TestUpdateCollectiblePreferencesTestnet(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	err := walletSettings.UpdateCollectiblePreferences([]CollectiblePreferences{
		{CollectiblePreferencesTypeNonCommunityCollectible, "First", 0, false},
	}, false, false, true, 0)
	require.NoError(t, err)

	// Mainnet is not affected by testnet preferences
	preferences, err := walletSettings.GetCollectiblePreferences(false)
	require.NoError(t, err)
	require.Equal(t, 0, len(preferences))

	preferences, err = walletSettings.GetCollectiblePreferences(true)
	require.NoError(t, err)
	require.Equal(t, 1, len(preferences))
	require.Equal(t, CollectiblePreferencesTypeNonCommunityCollectible, preferences[0].Type)
	require.Equal(t, "First", preferences[0].Key)
	require.Equal(t, 0, preferences[0].Position)
	require.Equal(t, false, preferences[0].Visible)

	// Inserting into testnet doesn't affect mainnet
	err = walletSettings.UpdateCollectiblePreferences([]CollectiblePreferences{
		{CollectiblePreferencesTypeNonCommunityCollectible, "First", 0, true},
		{CollectiblePreferencesTypeNonCommunityCollectible, "Second", 1, true},
	}, false, false, false, 0)
	require.NoError(t, err)
	err = walletSettings.UpdateCollectiblePreferences([]CollectiblePreferences{
		{CollectiblePreferencesTypeCommunityCollectible, "First", 1, true},
		{CollectiblePreferencesTypeCommunityCollectible, "Second", 2, true},
	}, false, false, true, 0)
	require.NoError(t, err)

	// Having same symbols on mainnet and testnet is allowed
	preferences, err = walletSettings.GetCollectiblePreferences(true)
	require.NoError(t, err)
	require.Equal(t, 2, len(preferences))
	require.Equal(t, CollectiblePreferencesTypeCommunityCollectible, preferences[0].Type)
	require.Equal(t, "First", preferences[0].Key)
	require.Equal(t, 1, preferences[0].Position)
	require.Equal(t, true, preferences[0].Visible)
	require.Equal(t, CollectiblePreferencesTypeCommunityCollectible, preferences[1].Type)
	require.Equal(t, "Second", preferences[1].Key)
	require.Equal(t, 2, preferences[1].Position)
	require.Equal(t, true, preferences[1].Visible)

	preferences, err = walletSettings.GetCollectiblePreferences(false)
	require.NoError(t, err)
	require.Equal(t, 2, len(preferences))
	require.Equal(t, CollectiblePreferencesTypeNonCommunityCollectible, preferences[0].Type)
	require.Equal(t, "First", preferences[0].Key)
	require.Equal(t, 0, preferences[0].Position)
	require.Equal(t, true, preferences[0].Visible)
	require.Equal(t, CollectiblePreferencesTypeNonCommunityCollectible, preferences[1].Type)
	require.Equal(t, "Second", preferences[1].Key)
	require.Equal(t, 1, preferences[1].Position)
	require.Equal(t, true, preferences[1].Visible)
}

func TestUpdateCollectiblePreferencesRollback(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	err := db.UpdateCollectiblePreferences([]CollectiblePreferences{
		{CollectiblePreferencesTypeNonCommunityCollectible, "First", 0, false},
	}, false, false, true, 0)
	require.NoError(t, err)

	// Duplicate is not allowed
	err = db.UpdateCollectiblePreferences([]CollectiblePreferences{
		{CollectiblePreferencesTypeNonCommunityCollectible, "Second", -1, false},
		{CollectiblePreferencesTypeNonCommunityCollectible, "Third", -1, false},
	}, false, false, true, 0)
	require.Error(t, err)

	// Rolled back to previous state
	preferences, err := db.GetCollectiblePreferences(true)
	require.NoError(t, err)
	require.Equal(t, 1, len(preferences))
	require.Equal(t, CollectiblePreferencesTypeNonCommunityCollectible, preferences[0].Type)
	require.Equal(t, "First", preferences[0].Key)
	require.Equal(t, 0, preferences[0].Position)
	require.Equal(t, false, preferences[0].Visible)
}

func TestUpdateCollectiblePreferencesSettings(t *testing.T) {
	walletSettings, stop := setupTestDB(t)
	defer stop()

	preferences := []CollectiblePreferences{
		{CollectiblePreferencesTypeNonCommunityCollectible, "First", 0, false},
		{CollectiblePreferencesTypeCommunityCollectible, "Second", 1, true},
		{CollectiblePreferencesTypeCollection, "Third", 2, false},
		{CollectiblePreferencesTypeCommunity, "Fourth", 3, true},
	}

	err := walletSettings.UpdateCollectiblePreferences(preferences, true, true, true, 123)
	require.NoError(t, err)

	// Verify that the preferences are updated correctly
	prefs, err := walletSettings.GetCollectiblePreferences(true)
	require.NoError(t, err)
	require.Equal(t, len(preferences), len(prefs))
	for i := range preferences {
		require.Equal(t, preferences[i], prefs[i])
	}

	// Verify that the clock is updated
	clock, err := walletSettings.GetClockOfLastCollectiblePreferencesChange()
	require.NoError(t, err)
	require.True(t, clock > 0)
}
