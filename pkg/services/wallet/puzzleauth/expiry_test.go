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
// to resource requests without a live bearer.
type bearerServer struct {
	*httptest.Server
	lifetime     time.Duration
	solves       atomic.Int32
	unauthorized atomic.Int32
	served       atomic.Int32

	mu     sync.Mutex
	tokens map[string]time.Time
}

func newBearerServer(t *testing.T, lifetime time.Duration) *bearerServer {
	s := &bearerServer{lifetime: lifetime, tokens: map[string]time.Time{}}
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
			_ = json.NewEncoder(w).Encode(TokenResponse{Token: token, ExpiresAt: expiresAt.Format(time.RFC3339Nano)})
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
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.tokens[h[len(prefix):]]
	return ok && time.Now().Before(expiresAt)
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

	deadline := time.Now().Add(lifetime * 8 / 10)
	for time.Now().Before(deadline) {
		server.get(t, client)
		time.Sleep(50 * time.Millisecond)
	}

	require.Zero(t, server.unauthorized.Load())
	require.Equal(t, int32(1), server.solves.Load(), "a token is not refreshed while it has most of its lifetime left")
}
