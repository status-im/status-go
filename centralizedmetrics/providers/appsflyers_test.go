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

func TestAppsflyerMetricProcessor(t *testing.T) {
	// Create a test server to mock Appsflyer API
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request method and URL
		require.Equal(t, http.MethodPost, r.Method)

		expectedPath := "/inappevent/testAppID"
		require.Equal(t, expectedPath, r.URL.Path)

		// Check headers
		require.Equal(t, "application/json", r.Header.Get("accept"))
		require.Equal(t, "testSecret", r.Header.Get("authentication"))
		require.Equal(t, "application/json", r.Header.Get("content-type"))

		// Check request body
		var metric appsflyerMetric
		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		require.Equal(t, "user123", metric.AppsflyerID)
		require.Equal(t, "purchase", metric.EventName)
		require.Equal(t, map[string]interface{}{"price": 10.0}, metric.EventValue)
		require.Equal(t, "2024-07-02 15:14:54.765", metric.EventTime)

		// Respond with 200 OK
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	// Initialize the AppsflyerMetricProcessor with the test server URL
	processor := NewAppsflyerMetricProcessor("testAppID", "testSecret", testServer.URL, tt.MustCreateTestLogger())

	// Example metrics
	metrics := []common.Metric{
		{UserID: "user123", EventName: "purchase", EventValue: map[string]interface{}{"price": 10.0}, Timestamp: 1719933294765},
	}

	// Process metrics
	require.NoError(t, processor.Process(metrics))
}
