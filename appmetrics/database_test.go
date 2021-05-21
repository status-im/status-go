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
	require.NotNil(t, res[0].CreatedAt)
	require.Equal(t, count, 1)
}
