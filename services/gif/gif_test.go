package gif

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/t/helpers"
)

func setupSQLTestDb(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "local-notifications-tests-")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func setupTestDB(t *testing.T, db *sql.DB) (*accounts.Database, func()) {
	acc, err := accounts.NewDB(db)
	require.NoError(t, err)
	return acc, func() {
		require.NoError(t, db.Close())
	}
}

func TestSetTenorAPIKey(t *testing.T) {
	appDB, appStop := setupSQLTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	gifAPI := NewGifAPI(db)

	require.NoError(t, gifAPI.SetTenorAPIKey("DU7DWZ27STB2"))
	require.Equal(t, "DU7DWZ27STB2", tenorAPIKey)
}

func TestGetContentWithRetry(t *testing.T) {
	appDB, appStop := setupSQLTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	gifAPI := NewGifAPI(db)

	require.NoError(t, gifAPI.SetTenorAPIKey(""))
	require.Equal(t, "", tenorAPIKey)

	gifs, err := gifAPI.GetContentWithRetry("trending?")
	require.Error(t, err)
	require.Equal(t, "", gifs)

	require.NoError(t, gifAPI.SetTenorAPIKey("DU7DWZ27STB2"))
	require.Equal(t, "DU7DWZ27STB2", tenorAPIKey)

	gifs, err = gifAPI.GetContentWithRetry("trending?")
	require.NoError(t, err)
	require.NotEqual(t, "", gifs)
}

func TestFavoriteGifs(t *testing.T) {
	appDB, appStop := setupSQLTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	gifAPI := NewGifAPI(db)

	require.NoError(t, gifAPI.SetTenorAPIKey("DU7DWZ27STB2"))
	require.Equal(t, "DU7DWZ27STB2", tenorAPIKey)

	recent := map[string]interface{}{
		"id":         "23833142",
		"title":      "",
		"url":        "https://media.tenor.com/images/b845ae14f43883e5cd6283e705f09efb/tenor.gif",
		"tinyUrl":    "https://media.tenor.com/images/2067bdc0375f9606dfb9fb4d2bfaafde/tenor.gif",
		"height":     498,
		"isFavorite": true,
	}

	newRecents := map[string]interface{}{
		"items": recent,
	}
	inputJSON := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "gif_setTenorAPIKey",
		"params":  newRecents,
	}
	like, _ := json.Marshal(inputJSON)

	source := (json.RawMessage)(like)

	require.NoError(t, gifAPI.UpdateFavoriteGifs(source))
}

func TestRecentGifs(t *testing.T) {
	appDB, appStop := setupSQLTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	gifAPI := NewGifAPI(db)

	require.NoError(t, gifAPI.SetTenorAPIKey("DU7DWZ27STB2"))
	require.Equal(t, "DU7DWZ27STB2", tenorAPIKey)

	recent := map[string]interface{}{
		"id":         "23833142",
		"title":      "",
		"url":        "https://media.tenor.com/images/b845ae14f43883e5cd6283e705f09efb/tenor.gif",
		"tinyUrl":    "https://media.tenor.com/images/2067bdc0375f9606dfb9fb4d2bfaafde/tenor.gif",
		"height":     498,
		"isFavorite": true,
	}

	newRecents := map[string]interface{}{
		"items": recent,
	}
	inputJSON := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "gif_setTenorAPIKey",
		"params":  newRecents,
	}
	like, _ := json.Marshal(inputJSON)

	source := (json.RawMessage)(like)

	require.NoError(t, gifAPI.UpdateRecentGifs(source))
}
