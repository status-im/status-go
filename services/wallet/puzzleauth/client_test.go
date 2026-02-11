package puzzleauth

import (
	"context"
	"fmt"
	"net/http"
	netUrl "net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name            string
		httpClient      *http.Client
		expectedTimeout time.Duration
	}{
		{
			name:            "default HTTP client",
			httpClient:      nil,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:            "custom HTTP client",
			httpClient:      &http.Client{Timeout: 10 * time.Second},
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("https://test.nft.status.im", tt.httpClient)
			require.NotNil(t, client)
			require.NotNil(t, client.httpClient)
			require.NotNil(t, client.authService)
			require.Equal(t, 2, client.maxRetries)
			require.Equal(t, tt.expectedTimeout, client.httpClient.Timeout)
		})
	}
}

func TestClient_DoRequest_Success(t *testing.T) {
	var resourceReq int32
	server := newPuzzleAuthServer(t)
	defer server.Close()

	client := NewClient(server.URL, nil)
	ctx := context.Background()

	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&resourceReq, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)

	server2 := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
	defer server2.Close()
	parsedURL, err := netUrl.Parse(server2.URL)
	require.NoError(t, err)
	req.URL.Host = parsedURL.Host
	req.URL.Scheme = parsedURL.Scheme

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, int32(1), atomic.LoadInt32(&resourceReq))
	resp.Body.Close()
}

func TestClient_DoRequest_AuthRetry(t *testing.T) {
	var resourceReq, puzzleReq, solveReq int32
	var tokenProvided bool

	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&resourceReq, 1)
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
		} else {
			tokenProvided = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		}
	}

	server := newPuzzleAuthServer(t,
		withResourceHandler(resourceHandler),
		withCounters(&puzzleReq, &solveReq))
	defer server.Close()

	client := NewClient(server.URL, nil)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, int32(2), atomic.LoadInt32(&resourceReq))
	require.Equal(t, int32(1), atomic.LoadInt32(&puzzleReq))
	require.Equal(t, int32(1), atomic.LoadInt32(&solveReq))
	require.True(t, tokenProvided)
	resp.Body.Close()
}

func TestClient_DoRequest_RetryStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"403 retry", http.StatusForbidden},
		{"429 retry", http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resourceReq, requestsWithToken int32

			resourceHandler := func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&resourceReq, 1)
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					atomic.AddInt32(&requestsWithToken, 1)
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(tt.statusCode)
				}
			}

			server := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
			defer server.Close()

			client := NewClient(server.URL, nil)
			ctx := context.Background()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
			require.NoError(t, err)

			resp, err := client.DoRequest(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.GreaterOrEqual(t, atomic.LoadInt32(&resourceReq), int32(2))
			require.Equal(t, int32(1), atomic.LoadInt32(&requestsWithToken))
			resp.Body.Close()
		})
	}
}

func TestClient_DoRequest_MaxRetriesExceeded(t *testing.T) {
	var attempts int32

	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}

	server := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
	defer server.Close()

	client := NewClient(server.URL, nil)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	numAttempts := atomic.LoadInt32(&attempts)
	require.Equal(t, int32(3), numAttempts, fmt.Sprintf("Expected 3 attempts, got %d", numAttempts))
}

func TestClient_DoRequest_AuthFailure(t *testing.T) {
	var resourceReq int32

	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&resourceReq, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}

	server := newPuzzleAuthServer(t,
		withResourceHandler(resourceHandler),
		withPuzzleError(500))
	defer server.Close()

	client := NewClient(server.URL, nil)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "failed to get auth token")
	require.GreaterOrEqual(t, atomic.LoadInt32(&resourceReq), int32(1))
}

func TestClient_DoRequest_NetworkError(t *testing.T) {
	client := NewClient("http://localhost:1", nil)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:1/resource", nil)
	require.NoError(t, err)

	resp, err := client.DoRequest(req)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestClient_DoGetRequest(t *testing.T) {
	tests := []struct {
		name     string
		params   netUrl.Values
		expected string
	}{
		{
			name: "with params",
			params: netUrl.Values{
				"param1": []string{"value1"},
				"param2": []string{"value2"},
			},
			expected: `{"result": "success"}`,
		},
		{
			name:     "no params",
			params:   nil,
			expected: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceHandler := func(w http.ResponseWriter, r *http.Request) {
				if tt.params != nil {
					require.Equal(t, "value1", r.URL.Query().Get("param1"))
					require.Equal(t, "value2", r.URL.Query().Get("param2"))
					_, _ = w.Write([]byte(`{"result": "success"}`))
				} else {
					require.Empty(t, r.URL.RawQuery)
					_, _ = w.Write([]byte("success"))
				}
			}

			server := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
			defer server.Close()

			client := NewClient(server.URL, nil)
			ctx := context.Background()

			body, err := client.DoGetRequest(ctx, server.URL+"/resource", tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(body))
		})
	}
}

func TestClient_DoGetRequest_NonOK(t *testing.T) {
	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}

	server := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
	defer server.Close()

	client := NewClient(server.URL, nil)
	ctx := context.Background()

	body, err := client.DoGetRequest(ctx, server.URL+"/resource", nil)
	require.Error(t, err)
	require.Nil(t, body)
	require.Contains(t, err.Error(), "request failed with status 404")
}

func TestClient_GetAuthService(t *testing.T) {
	client := NewClient("https://test.nft.status.im", nil)

	service := client.GetAuthService()
	require.NotNil(t, service)
	require.Equal(t, "https://test.nft.status.im", service.origin)
}

func TestClient_DoRequest_ContextCancelled(t *testing.T) {
	client := NewClient("https://test.nft.status.im", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:1", nil)
	_, err := client.DoRequest(req)
	require.Error(t, err)
}
