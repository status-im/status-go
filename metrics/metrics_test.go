package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/stretchr/testify/require"
)

func createTestServer(t *testing.T) *Server {
	server := NewMetricsServer(":8080", nil)
	require.NotNil(t, server)
	require.NotNil(t, server.handlers)
	return server
}

func TestNewMetricsServer(t *testing.T) {
	server := NewMetricsServer(":8080", nil)
	require.NotNil(t, server)
	require.Equal(t, ":8080", server.server.Addr)
	require.NotNil(t, server.handlers)
}

func TestNewMetricsServer_WithRegistry(t *testing.T) {
	registry := metrics.NewRegistry()
	server := NewMetricsServer(":8080", registry)
	require.NotNil(t, server)
	require.Contains(t, server.handlers, "geth")
}

func TestServer_RegisterHandler(t *testing.T) {
	server := createTestServer(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})

	server.RegisterHandler("test", handler)

	server.handlersMutex.RLock()
	_, exists := server.handlers["test"]
	server.handlersMutex.RUnlock()

	require.True(t, exists)
}

func TestServer_UnregisterHandler(t *testing.T) {
	server := createTestServer(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	server.RegisterHandler("test", handler)

	// Verify it exists
	server.handlersMutex.RLock()
	_, exists := server.handlers["test"]
	server.handlersMutex.RUnlock()
	require.True(t, exists)

	// Unregister
	server.UnregisterHandler("test")

	// Verify it's gone
	server.handlersMutex.RLock()
	_, exists = server.handlers["test"]
	server.handlersMutex.RUnlock()
	require.False(t, exists)
}

func TestServer_MetricsHandler(t *testing.T) {
	server := createTestServer(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test-metrics"))
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
