package gif

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
)

func setupSQLTestDb(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "local-notifications-tests-")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func setupTestDB(t *testing.T, db *sql.DB) (*accounts.Database, func()) {
	acc, err := accounts.NewDB(db)
	require.NoError(t, err)
	config := params.NodeConfig{
		NetworkID: 10,
	}
	networks := json.RawMessage("{}")
	settingsObj := settings.Settings{
		Networks: &networks,
	}

	err = acc.CreateSettings(settingsObj, config)
	require.NoError(t, err)

	return acc, func() {
		require.NoError(t, db.Close())
	}
}

func TestSetKlipyAPIKey(t *testing.T) {
	appDB, appStop := setupSQLTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	gifAPI := NewGifAPI(db)

	require.NoError(t, gifAPI.SetKlipyAPIKey("test-key"))
	require.Equal(t, "test-key", klipyAPIKey)
}

func TestGetContentWithRetry(t *testing.T) {
	appDB, appStop := setupSQLTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	gifAPI := NewGifAPI(db)

	const responseBody = `{"result":true,"data":{"data":[],"current_page":1,"per_page":50,"has_next":false}}`
	var lastPath, lastQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		lastQuery = r.URL.RawQuery
		if strings.Contains(r.URL.Path, "//") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()

	originalBaseURL := baseURL
	baseURL = server.URL + "/api/v1/"
	defer func() { baseURL = originalBaseURL }()

	// No api key set
	require.NoError(t, gifAPI.SetKlipyAPIKey(""))
	require.Equal(t, "", klipyAPIKey)

	gifs, err := gifAPI.GetContentWithRetry("trending?")
	require.Error(t, err)
	require.Equal(t, "", gifs)

	// Valid api key set
	require.NoError(t, gifAPI.SetKlipyAPIKey("test-key"))
	require.Equal(t, "test-key", klipyAPIKey)

	gifs, err = gifAPI.GetContentWithRetry("trending?")
	require.NoError(t, err)
	require.Equal(t, responseBody, gifs)

	// Verify the KLIPY URL format
	require.Equal(t, "/api/v1/test-key/gifs/trending", lastPath)
	require.Contains(t, lastQuery, "per_page=50")
	require.Contains(t, lastQuery, "content_filter=low")
	require.Contains(t, lastQuery, "customer_id=status-app")

	// A search query is properly forwarded
	gifs, err = gifAPI.GetContentWithRetry("search?q=cat&")
	require.NoError(t, err)
	require.Equal(t, responseBody, gifs)
	require.Equal(t, "/api/v1/test-key/gifs/search", lastPath)
	require.Contains(t, lastQuery, "q=cat")
}

func TestFavoriteGifs(t *testing.T) {
	appDB, appStop := setupSQLTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	gifAPI := NewGifAPI(db)

	require.NoError(t, gifAPI.SetKlipyAPIKey("test-key"))
	require.Equal(t, "test-key", klipyAPIKey)

	favorite := Gif{
		ID:         "23833142",
		Title:      "",
		URL:        "https://static.klipy.com/ii/935d7ab9d8c6202580a668421940ec81/14/af/8GCrVAB7.gif",
		TinyURL:    "https://static.klipy.com/ii/935d7ab9d8c6202580a668421940ec81/14/af/y6iepZM7.gif",
		Height:     498,
		IsFavorite: true,
	}

	source, err := json.Marshal(Container{Items: []Gif{favorite}})
	require.NoError(t, err)

	require.NoError(t, gifAPI.UpdateFavoriteGifs(source))

	storedFavorites, err := gifAPI.GetFavoriteGifs()
	require.NoError(t, err)
	require.Equal(t, []Gif{favorite}, storedFavorites)
}

func TestRecentGifs(t *testing.T) {
	appDB, appStop := setupSQLTestDb(t)
	defer appStop()

	db, stop := setupTestDB(t, appDB)
	defer stop()

	gifAPI := NewGifAPI(db)

	require.NoError(t, gifAPI.SetKlipyAPIKey("test-key"))
	require.Equal(t, "test-key", klipyAPIKey)

	recent := Gif{
		ID:         "23833142",
		Title:      "",
		URL:        "https://static.klipy.com/ii/935d7ab9d8c6202580a668421940ec81/14/af/8GCrVAB7.gif",
		TinyURL:    "https://static.klipy.com/ii/935d7ab9d8c6202580a668421940ec81/14/af/y6iepZM7.gif",
		Height:     498,
		IsFavorite: true,
	}

	source, err := json.Marshal(Container{Items: []Gif{recent}})
	require.NoError(t, err)

	require.NoError(t, gifAPI.UpdateRecentGifs(source))

	storedRecents, err := gifAPI.GetRecentGifs()
	require.NoError(t, err)
	require.Equal(t, []Gif{recent}, storedRecents)
}
