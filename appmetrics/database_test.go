package appmetrics

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "appmetrics-tests")
	require.NoError(t, err)
	return NewDB(db), func() { require.NoError(t, cleanup()) }
}

func TestSaveAppMetrics(t *testing.T) {
	sessionID := "rand-omse-ssid"
	db, stop := setupTestDB(t)
	defer stop()

	// we need backticks (``) for value because it is expected by gojsonschema
	// it considers text inside tics to be stringified json
	appMetrics := []AppMetric{
		{Event: NavigateTo, Value: json.RawMessage(`{"view_id": "some-view-id", "params": {"screen": "login"}}`), OS: "android", AppVersion: "1.11"},
	}

	err := db.SaveAppMetrics(appMetrics, sessionID)
	require.NoError(t, err)

	appMetricsPage, err := db.GetAppMetrics(10, 0)
	res := appMetricsPage.AppMetrics
	count := appMetricsPage.TotalCount
	require.NoError(t, err)
	require.Equal(t, appMetrics[0].Event, res[0].Event)
	require.Equal(t, appMetrics[0].Value, res[0].Value)
	require.Equal(t, appMetrics[0].OS, res[0].OS)
	require.Equal(t, appMetrics[0].AppVersion, res[0].AppVersion)
	require.False(t, res[0].Processed)
	require.NotNil(t, res[0].CreatedAt)
	require.Equal(t, count, 1)
}

func TestDatabase_GetUnprocessedMetrics(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	var uam []AppMetric
	metricsPerSession := 10
	unprocessedMetricsPerSession := 5
	numberOfSessions := 3
	numberOfSessionSaves := 5

	for i := 0; i < numberOfSessionSaves; i++ {
		for ii := 1; ii < numberOfSessions+1; ii++ {
			err := db.SaveAppMetrics(GenerateMetrics(metricsPerSession), "rand-omse-ssid-"+fmt.Sprint(ii))
			require.NoError(t, err)

			uam, err = db.GetUnprocessed()
			require.NoError(t, err)
			require.Len(t, uam, unprocessedMetricsPerSession*ii+(i*numberOfSessions*unprocessedMetricsPerSession))
		}
	}

	// Test metrics are grouped by session_id
	lastSessionID := ""
	sessionCount := 0
	for _, m := range uam {
		if lastSessionID != m.SessionID {
			lastSessionID = m.SessionID
			sessionCount++
		}
	}
	require.Equal(t, numberOfSessions, sessionCount)
}

func TestDatabase_SetProcessedMetrics(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	// Add sample data to the DB
	err := db.SaveAppMetrics(GenerateMetrics(20), "rand-omse-ssid")
	require.NoError(t, err)

	// Get only the unprocessed metrics
	uam, err := db.GetUnprocessed()
	require.NoError(t, err)

	// Extract the ids from the metrics IDs
	ids := GetAppMetricsIDs(uam)

	// Add some more metrics to the DB
	err = db.SaveAppMetrics(GenerateMetrics(20), "rand-omse-ssid-2")
	require.NoError(t, err)

	// Set metrics as processed with the given ids
	err = db.SetToProcessedByIDs(ids)
	require.NoError(t, err)

	// Check we have the expected number of unprocessed metrics in the db
	uam, err = db.GetUnprocessed()
	require.NoError(t, err)
	require.Len(t, uam, 10)
}

func TestDatabase_GetUnprocessedGroupedBySession(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	// Add sample data to the DB
	err := db.SaveAppMetrics(GenerateMetrics(20), "rand-omse-ssid")
	require.NoError(t, err)

	// Add some more metrics to the DB
	err = db.SaveAppMetrics(GenerateMetrics(20), "rand-omse-ssid-2")
	require.NoError(t, err)

	// Check we have the expected number of unprocessed metrics in the db
	uam, err := db.GetUnprocessedGroupedBySession()
	require.NoError(t, err)

	// Check we have 2 groups / sessions
	require.Len(t, uam, 2)
	require.Len(t, uam["rand-omse-ssid"], 10)
	require.Len(t, uam["rand-omse-ssid-2"], 10)
}

func TestDatabase_DeleteOlderThan(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	threeHoursAgo := time.Now().Add(time.Hour * -3) // go is annoying sometimes
	oneHourHence := time.Now().Add(time.Hour)

	// Add sample data to the DB
	err := db.SaveAppMetrics(GenerateMetrics(20), "rand-omse-ssid")
	require.NoError(t, err)

	// Delete all messages older than 3 hours old
	err = db.DeleteOlderThan(&threeHoursAgo)
	require.NoError(t, err)

	// Get all metrics from DB, none should be deleted
	ams, err := db.GetAppMetrics(100, 0)
	require.NoError(t, err)
	require.Len(t, ams.AppMetrics, 20)

	// Delete all messages older than 1 hours in the future
	err = db.DeleteOlderThan(&oneHourHence)
	require.NoError(t, err)

	// Get all metrics from DB, all should be deleted
	ams, err = db.GetAppMetrics(100, 0)
	require.NoError(t, err)
	require.Len(t, ams.AppMetrics, 0)
}
