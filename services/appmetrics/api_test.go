package appmetrics

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/appmetrics"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*appmetrics.Database, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "appmetrics-service")
	require.NoError(t, err)
	return appmetrics.NewDB(db), func() { require.NoError(t, cleanup()) }
}

func TestValidateAppMetrics(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()
	api := NewAPI(db)

	validMetrics := []appmetrics.AppMetric{appmetrics.AppMetric{
		Event:      "navigate-to",
		Value:      json.RawMessage(`{"view_id": "some-view-id", "params": {"screen": "login"}}`),
		AppVersion: "1.12",
		OS:         "android"}}

	invalidMetrics := []appmetrics.AppMetric{appmetrics.AppMetric{
		Event:      "navigate-to",
		Value:      json.RawMessage("{}"),
		AppVersion: "1.12",
		OS:         "android"}}

	err := api.ValidateAppMetrics(context.Background(), validMetrics)
	require.NoError(t, err)

	err = api.ValidateAppMetrics(context.Background(), invalidMetrics)
	require.Error(t, err)
}
