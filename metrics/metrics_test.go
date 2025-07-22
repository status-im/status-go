package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/metrics"
)

func createTestServer(t *testing.T) *Server {
	server := NewMetricsServer(":8080", nil)
	require.NotNil(t, server)
	return server
}

func TestNewMetricsServer(t *testing.T) {
	server := NewMetricsServer(":8080", nil)
	require.NotNil(t, server)
	require.Equal(t, ":8080", server.server.Addr)
}

func TestNewMetricsServer_WithRegistry(t *testing.T) {
	registry := metrics.NewRegistry()
	server := NewMetricsServer(":8080", registry)
	require.NotNil(t, server)

	// Check that geth handler was registered
	_, exists := server.handlers.Load("geth")
	require.True(t, exists)
}

func TestServer_RegisterHandler(t *testing.T) {
	server := createTestServer(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("test"))
		require.NoError(t, err)
	})

	server.RegisterHandler("test", handler)

	_, exists := server.handlers.Load("test")
	require.True(t, exists)
}

func TestServer_RegisterHandler_Nil(t *testing.T) {
	server := createTestServer(t)

	// Register nil handler - should be ignored
	server.RegisterHandler("nil-test", nil)

	_, exists := server.handlers.Load("nil-test")
	require.False(t, exists)
}

func TestServer_UnregisterHandler(t *testing.T) {
	server := createTestServer(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	server.RegisterHandler("test", handler)

	// Verify it exists
	_, exists := server.handlers.Load("test")
	require.True(t, exists)

	// Unregister
	server.UnregisterHandler("test")

	// Verify it's gone
	_, exists = server.handlers.Load("test")
	require.False(t, exists)
}

func TestServer_MetricsHandler(t *testing.T) {
	server := createTestServer(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("test-metrics"))
		require.NoError(t, err)
	})
	server.RegisterHandler("test", handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	metricsHandler := server.metricsHandler()
	metricsHandler.ServeHTTP(w, req)

	require.Contains(t, w.Body.String(), "test-metrics")
}

func TestHealthHandler(t *testing.T) {
	handler := healthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "OK", w.Body.String())
}

func TestServer_Stop(t *testing.T) {
	server := createTestServer(t)

	err := server.Stop()
	require.NoError(t, err)
}

func TestServer_Stop_NilServer(t *testing.T) {
	server := &Server{server: nil}

	err := server.Stop()
	require.NoError(t, err)
}
