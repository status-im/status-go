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

func TestOneIsOne(t *testing.T) {
	_, stop := setupTestDB(t)
	defer stop()

	require.Equal(t, 1, 1)
}

func TestSaveAppMetrics(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	appMetrics := []AppMetric{
		{Event: TestEvent1, Value: "1", OS: "android", AppVersion: "1.11"},
		{Event: TestEvent2, Value: "2", OS: "ios", AppVersion: "1.10"},
	}

	err := db.SaveAppMetrics(appMetrics)
	require.NoError(t, err)

	res, err := db.GetAppMetrics(10, 0)
	require.NoError(t, err)
	t.Log(res)
}
