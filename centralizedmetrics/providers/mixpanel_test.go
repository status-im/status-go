package providers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/centralizedmetrics/common"
	"github.com/status-im/status-go/protocol/tt"
)

func TestMixpanelMetricProcessor(t *testing.T) {
	var t1 int64 = 1719933294765
	var t2 int64 = 1719933294766

	// Create a test server to mock Mixpanel API
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request method and URL
		require.Equal(t, http.MethodPost, r.Method)

		expectedPath := "/track"
		require.Equal(t, expectedPath, r.URL.Path)
		queryParams := r.URL.Query()
		require.Equal(t, "1", queryParams.Get("strict"))
		require.Equal(t, "testAppID", queryParams.Get("project_id"))

		// Check headers
		require.Equal(t, "application/json", r.Header.Get("accept"))
		require.Equal(t, "application/json", r.Header.Get("content-type"))

		// Check request body
		var metrics []mixpanelMetric
		err := json.NewDecoder(r.Body).Decode(&metrics)
		require.NoError(t, err)

		require.Len(t, metrics, 2)
		metric1 := metrics[0]
		require.Equal(t, "user123", metric1.Properties.UserID)
		require.Equal(t, "testSecret", metric1.Properties.Token)
		require.Equal(t, "purchase", metric1.Event)
		require.Equal(t, t1, metric1.Properties.Time)
		require.Equal(t, map[string]interface{}{"price": 10.0}, metric1.Properties.AdditionalProperties)
		metric2 := metrics[1]
		require.Equal(t, "user123", metric2.Properties.UserID)
		require.Equal(t, "testSecret", metric2.Properties.Token)
		require.Equal(t, "purchase", metric2.Event)
		require.Equal(t, t2, metric2.Properties.Time)
		require.Equal(t, map[string]interface{}{"price": 11.0}, metric2.Properties.AdditionalProperties)

		// Respond with 200 OK
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	// Initialize the MixpanelMetricProcessor with the test server URL
	processor := NewMixpanelMetricProcessor("testAppID", "testSecret", testServer.URL, tt.MustCreateTestLogger())

	// Example metrics
	metrics := []common.Metric{
		{UserID: "user123", EventName: "purchase", EventValue: map[string]interface{}{"price": 10.0}, Timestamp: t1},
		{UserID: "user123", EventName: "purchase", EventValue: map[string]interface{}{"price": 11.0}, Timestamp: t2},
	}

	// Process metrics
	require.NoError(t, processor.Process(metrics))
}
