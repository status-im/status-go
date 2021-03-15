package appmetrics

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/status-im/status-go/appmetrics"

	"github.com/stretchr/testify/require"
)

func TestSaveAppMetrics(t *testing.T) {
	db, close := setupTestDB(t)
	const numMetricsToWrite = 20
	defer close()
	service := NewService(db)
	api := NewAPI(service.db, service.metricsBufferedChan)

	require.NoError(t, service.Start(nil))

	validMetric := appmetrics.AppMetric{
		Event:      "go/test1",
		Value:      json.RawMessage(`"init-val"`),
		AppVersion: "1.12",
	}

	var err error
	for i := 0; i < numMetricsToWrite; i++ {
		err = api.SaveAppMetrics(context.Background(), []appmetrics.AppMetric{validMetric})
		require.NoError(t, err)
	}

	// Stop service to flush down pending metrics in chan
	require.NoError(t, service.Stop())

	// limit greater than numMetricsToWrite, offset 0
	writtenMetrics, _ := api.db.GetAppMetrics(numMetricsToWrite+10, 0)
	require.Equal(t, len(writtenMetrics), numMetricsToWrite)
}
