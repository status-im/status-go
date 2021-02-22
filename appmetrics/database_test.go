package appmetrics

import (
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
	db, stop := setupTestDB(t)
	defer stop()

	// we need backticks (``) for value because it is expected by gojsonschema
	// it considers text inside tics to be stringified json
	appMetrics := []AppMetric{
		{Event: TestEvent1, Value: `"str"`, OS: "android", AppVersion: "1.11"},
		{Event: TestEvent2, Value: `"str"`, OS: "ios", AppVersion: "1.10"},
	}

	err := db.SaveAppMetrics(appMetrics)
	require.NoError(t, err)

	res, err := db.GetAppMetrics(10, 0)
	require.NoError(t, err)
	require.Equal(t, appMetrics, res)
}
