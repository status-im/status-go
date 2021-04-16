package appmetrics

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/status-im/status-go/appdatabase"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "appmetrics-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "appmetrics-tests")
	require.NoError(t, err)

	return NewDB(db), func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
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

func generateMetrics(num int) []AppMetric {
	var appMetrics []AppMetric
	for i := 0; i < num; i++ {
		am := AppMetric{
			Event:      NavigateTo,
			Value:      json.RawMessage(`{"view_id": "some-view-id", "params": {"screen": "login"}}`),
			OS:         "android",
			AppVersion: "1.11",
		}
		if i < num/2 {
			am.Processed = true
		}
		appMetrics = append(appMetrics, am)
	}

	return appMetrics
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
			err := db.SaveAppMetrics(generateMetrics(metricsPerSession), "rand-omse-ssid-" + string(ii))
			require.NoError(t, err)

			uam, err = db.GetUnprocessed()
			require.NoError(t, err)
			require.Len(t, uam, unprocessedMetricsPerSession*ii+(i*numberOfSessions*unprocessedMetricsPerSession))
		}
	}

	// Test metrics are grouped by session_id
	lastSessionId := ""
	sessionCount := 0
	for _, m := range uam {
		if lastSessionId != m.SessionID {
			lastSessionId = m.SessionID
			sessionCount++
		}
	}
	require.Equal(t, numberOfSessions, sessionCount)
}

func TestDatabase_SetProcessedMetrics(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	// Add sample data to the DB
	err := db.SaveAppMetrics(generateMetrics(20), "rand-omse-ssid")
	require.NoError(t, err)

	// Get only the unprocessed metrics
	uam, err := db.GetUnprocessed()
	require.NoError(t, err)

	// Extract the ids from the metrics IDs
	ids := GetAppMetricsIDs(uam)

	// Add some more metrics to the DB
	err = db.SaveAppMetrics(generateMetrics(20), "rand-omse-ssid-2")
	require.NoError(t, err)

	// Set metrics as processed with the given ids
	err = db.SetToProcessedByIDs(ids)
	require.NoError(t, err)

	// Check we have the expected number of unprocessed metrics in the db
	uam, err = db.GetUnprocessed()
	require.NoError(t, err)
	require.Len(t, uam, 10)
}
