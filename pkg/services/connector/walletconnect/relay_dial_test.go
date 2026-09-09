package walletconnect

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// newRejectingRelay serves status and body instead of upgrading.
func newRejectingRelay(t *testing.T, status int, body string) *RelayClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	r, err := NewRelayClient("test")
	require.NoError(t, err)
	r.url = strings.Replace(srv.URL, "http://", "ws://", 1)
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestRelayClient_DialError_ReportsHTTPStatus(t *testing.T) {
	r := newRejectingRelay(t, http.StatusForbidden, `{"error":"Project not found"}`)

	err := r.Connect()
	require.Error(t, err)
	require.ErrorContains(t, err, "403")
	require.ErrorContains(t, err, "Project not found")
}

func TestRelayClient_DialError_ClassifiesRetryability(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"clock skew", 401, `{"error":"JWT validation error: JWT Token is not yet valid: ..."}`, "clock_skew, retryable=false"},
		{"expired token", 401, `{"error":"JWT validation error: JWT Token is expired: Some(1)"}`, "clock_skew, retryable=false"},
		{"missing token", 401, `{"error":"JWT is missing"}`, "auth, retryable=false"},
		{"unknown project", 403, `{"error":"Project not found"}`, "project, retryable=false"},
		{"rate limited", 429, `{"error":"Too many requests"}`, "rate_limited, retryable=true"},
		{"relay down", 503, `{"error":"unavailable"}`, "server, retryable=true"},
		{"proxy in the way", 502, "<html><title>502 Bad Gateway</title></html>", "proxy, retryable=true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := newRejectingRelay(t, tc.status, tc.body).Connect()
			require.Error(t, err)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestRelayClient_DialError_NoResponseIsRetryable(t *testing.T) {
	r, err := NewRelayClient("test")
	require.NoError(t, err)
	r.url = "ws://127.0.0.1:1" // nothing listens here
	t.Cleanup(func() { _ = r.Close() })

	err = r.Connect()
	require.Error(t, err)
	require.ErrorContains(t, err, "network, retryable=true")
}
