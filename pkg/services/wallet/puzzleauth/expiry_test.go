package puzzleauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// bearerServer issues distinct tokens that live for lifetime and answers 401
// to resource requests without a live bearer. With a quota, a token is also
// rejected past its quota-th request.
type bearerServer struct {
	*httptest.Server
	lifetime     time.Duration
	quota        int
	solves       atomic.Int32
	unauthorized atomic.Int32
	served       atomic.Int32

	mu     sync.Mutex
	tokens map[string]time.Time
	usage  map[string]int
}

func newBearerServer(t *testing.T, lifetime time.Duration) *bearerServer {
	return newQuotaServer(t, lifetime, 0)
}

func newQuotaServer(t *testing.T, lifetime time.Duration, quota int) *bearerServer {
	s := &bearerServer{lifetime: lifetime, quota: quota, tokens: map[string]time.Time{}, usage: map[string]int{}}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/puzzle":
			_ = json.NewEncoder(w).Encode(testPuzzle())
		case "/auth/solve":
			n := s.solves.Add(1)
			token := fmt.Sprintf("token-%d", n)
			expiresAt := time.Now().Add(s.lifetime)
			s.mu.Lock()
			s.tokens[token] = expiresAt
			s.mu.Unlock()
			_ = json.NewEncoder(w).Encode(TokenResponse{Token: token, ExpiresAt: expiresAt.Format(time.RFC3339Nano), RequestLimit: s.quota})
		case "/resource":
			if !s.authorized(r) {
				s.unauthorized.Add(1)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			s.served.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *bearerServer) authorized(r *http.Request) bool {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) {
		return false
	}
	token := h[len(prefix):]
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.tokens[token]
	if !ok {
		return false
	}
	s.usage[token]++
	return time.Now().Before(expiresAt) && (s.quota == 0 || s.usage[token] <= s.quota)
}

func (s *bearerServer) do(client *http.Client) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, s.URL+"/resource", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func (s *bearerServer) get(t *testing.T, client *http.Client) {
	require.NoError(t, s.do(client))
}

func (s *bearerServer) burst(t *testing.T, client *http.Client, n int) {
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- s.do(client)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}

func TestTransport_FirstRequestCarriesBearer(t *testing.T) {
	server := newBearerServer(t, time.Hour)
	client := &http.Client{Transport: &Transport{base: http.DefaultTransport, authService: NewService(server.URL, nil), maxRetries: 2}}

	server.burst(t, client, 20)

	require.Zero(t, server.unauthorized.Load(), "no request goes out without a bearer")
	require.Equal(t, int32(1), server.solves.Load())
	require.Equal(t, int32(20), server.served.Load())
}

func TestTransport_ExpiredTokenRefreshedBeforeSending(t *testing.T) {
	const lifetime = 2 * time.Second
	server := newBearerServer(t, lifetime)
	client := &http.Client{Transport: &Transport{base: http.DefaultTransport, authService: NewService(server.URL, nil), maxRetries: 2}}

	server.get(t, client)
	time.Sleep(lifetime)
	server.burst(t, client, 20)

	require.Zero(t, server.unauthorized.Load(), "an expired token is replaced before the request is sent")
	require.Equal(t, int32(2), server.solves.Load(), "one solve per token lifetime")
}

func TestTransport_TokenReusedForMostOfItsLifetime(t *testing.T) {
	const lifetime = 2 * time.Second
	server := newBearerServer(t, lifetime)
	client := &http.Client{Transport: &Transport{base: http.DefaultTransport, authService: NewService(server.URL, nil), maxRetries: 2}}

	deadline := time.Now().Add(lifetime * 6 / 10)
	for time.Now().Before(deadline) {
		server.get(t, client)
		time.Sleep(50 * time.Millisecond)
	}

	require.Zero(t, server.unauthorized.Load())
	require.Equal(t, int32(1), server.solves.Load(), "a token is not refreshed in the first 60% of its lifetime")
}

func TestTransport_TokenRenewedBeforeQuotaRunsOut(t *testing.T) {
	const quota, n = 10, 45
	server := newQuotaServer(t, time.Hour, quota)
	client := &http.Client{Transport: &Transport{base: http.DefaultTransport, authService: NewService(server.URL, nil), maxRetries: 2}}

	server.burst(t, client, n)

	require.Zero(t, server.unauthorized.Load(), "no request is sent with a token past its quota")
	require.Equal(t, int32((n+quota-1)/quota), server.solves.Load(), "one solve per quota of requests")
	require.Equal(t, int32(n), server.served.Load())
}
