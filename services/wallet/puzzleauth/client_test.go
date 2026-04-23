package puzzleauth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	netUrl "net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPClient(t *testing.T) {
	c := NewHTTPClient("https://test.nft.status.im")
	require.NotNil(t, c)
	require.Equal(t, 60*time.Second, c.Timeout)
	require.NotNil(t, c.Transport)
}

func TestNewTransport_SameOriginSharesAuthService(t *testing.T) {
	origin := "https://test.eth-rpc.status.im"
	a := NewTransport(origin, http.DefaultTransport)
	b := NewTransport(origin, http.DefaultTransport)
	require.Same(t, a.authService, b.authService, "all transports for one puzzle-auth host must share JWT cache")

	other := NewTransport("https://other.status.im", http.DefaultTransport)
	require.NotSame(t, a.authService, other.authService)
}

func TestTransport_Do_Success(t *testing.T) {
	var resourceReq int32
	server := newPuzzleAuthServer(t)
	defer server.Close()

	client := NewHTTPClient(server.URL)
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

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, int32(1), atomic.LoadInt32(&resourceReq))
	resp.Body.Close()
}

// Go-ethereum HTTP client sets both Body and GetBody; without buffering the request body, each
// retry would reuse an exhausted io.Reader and send an empty body (e.g. Cloudflare 400).
func TestTransport_RoundTrip_401Retry_SendsGetBodyOnEachAttempt(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1}`)
	var calls int32
	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, payload, b, "body must be identical on every round-trip (including retries)")
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}

	server := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
	defer server.Close()

	ctx := context.Background()
	rdr := bytes.NewReader(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/resource", rdr)
	require.NoError(t, err)
	req.ContentLength = int64(len(payload))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(payload)), nil }

	transport := NewTransport(server.URL, http.DefaultTransport)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "unauthorized + successful retry")
}

func TestTransport_RoundTrip_DoesNotMutateCallerRequestHeaders(t *testing.T) {
	var resourceReq int32
	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&resourceReq, 1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}

	server := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
	defer server.Close()

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)
	require.Equal(t, "", req.Header.Get("Authorization"))

	transport := NewTransport(server.URL, http.DefaultTransport)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	require.GreaterOrEqual(t, atomic.LoadInt32(&resourceReq), int32(2))
	require.Equal(t, "", req.Header.Get("Authorization"), "original request must not get Authorization; only cloned requests use Bearer")
}

func TestTransport_Do_AuthRetry(t *testing.T) {
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

	client := NewHTTPClient(server.URL)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, int32(2), atomic.LoadInt32(&resourceReq))
	require.Equal(t, int32(1), atomic.LoadInt32(&puzzleReq))
	require.Equal(t, int32(1), atomic.LoadInt32(&solveReq))
	require.True(t, tokenProvided)
	resp.Body.Close()
}

func TestTransport_Do_RetryStatusCodes(t *testing.T) {
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

			client := NewHTTPClient(server.URL)
			ctx := context.Background()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.GreaterOrEqual(t, atomic.LoadInt32(&resourceReq), int32(2))
			require.Equal(t, int32(1), atomic.LoadInt32(&requestsWithToken))
			resp.Body.Close()
		})
	}
}

func TestTransport_Do_MaxRetriesExceeded(t *testing.T) {
	var attempts int32

	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}

	server := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAuthRotating))
	require.Nil(t, resp)

	numAttempts := atomic.LoadInt32(&attempts)
	require.Equal(t, int32(3), numAttempts, fmt.Sprintf("Expected 3 attempts, got %d", numAttempts))
}

func TestTransport_Do_AuthFailure(t *testing.T) {
	var resourceReq int32

	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&resourceReq, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}

	server := newPuzzleAuthServer(t,
		withResourceHandler(resourceHandler),
		withPuzzleError(500))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "failed to get auth token")
	require.GreaterOrEqual(t, atomic.LoadInt32(&resourceReq), int32(1))
}

func TestTransport_Do_NetworkError(t *testing.T) {
	client := NewHTTPClient("http://localhost:1")
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:1/resource", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestTransport_DoGet(t *testing.T) {
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

			client := NewHTTPClient(server.URL)
			ctx := context.Background()

			u := server.URL + "/resource"
			if len(tt.params) > 0 {
				u = u + "?" + tt.params.Encode()
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(body))
		})
	}
}

func TestTransport_DoGet_NonOK(t *testing.T) {
	resourceHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}

	server := newPuzzleAuthServer(t, withResourceHandler(resourceHandler))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, "not found", string(body))
}

func TestTransport_Do_ContextCancelled(t *testing.T) {
	client := NewHTTPClient("https://test.nft.status.im")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:1", nil)
	_, err := client.Do(req)
	require.Error(t, err)
}
