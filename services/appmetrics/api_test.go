package appmetrics

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/appmetrics"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*appmetrics.Database, func()) {
	tmpfile, err := ioutil.TempFile("", "appmetrics-service")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "appmetrics-tests")
	require.NoError(t, err)
	return appmetrics.NewDB(db), func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestValidateAppMetrics(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()
	api := NewAPI(db)

	validMetrics := []appmetrics.AppMetric{appmetrics.AppMetric{
		Event:      "navigation/navigate-to",
		Value:      json.RawMessage(`{"view_id": "some-view-oid", "params": {"screen": "allowed-screen-name"}}`),
		AppVersion: "1.12",
		OS:         "android"}}

	invalidMetrics := []appmetrics.AppMetric{appmetrics.AppMetric{
		Event:      "navigation/navigate-to",
		Value:      json.RawMessage("{}"),
		AppVersion: "1.12",
		OS:         "android"}}

	err := api.ValidateAppMetrics(context.Background(), validMetrics)
	require.NoError(t, err)

	err = api.ValidateAppMetrics(context.Background(), invalidMetrics)
	require.Error(t, err)
}
