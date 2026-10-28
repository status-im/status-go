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

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// clockAuthServer issues tokens that expire lifetime after the fake clock's now.
// Solves wait while blocked and fail while failing.
type clockAuthServer struct {
	*httptest.Server
	clock    *fakeClock
	lifetime time.Duration
	solves   atomic.Int32
	failing  atomic.Bool

	mu      sync.Mutex
	blocked chan struct{}
}

func newClockAuthServer(t *testing.T, lifetime time.Duration) *clockAuthServer {
	s := &clockAuthServer{
		clock:    &fakeClock{now: time.Date(2026, 10, 27, 11, 55, 28, 0, time.UTC)},
		lifetime: lifetime,
	}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/puzzle":
			_ = json.NewEncoder(w).Encode(testPuzzle())
		case "/auth/solve":
			s.mu.Lock()
			blocked := s.blocked
			s.mu.Unlock()
			if blocked != nil {
				select {
				case <-blocked:
				case <-r.Context().Done():
					return
				}
			}
			if s.failing.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			n := s.solves.Add(1)
			_ = json.NewEncoder(w).Encode(TokenResponse{
				Token:     fmt.Sprintf("token-%d", n),
				ExpiresAt: s.clock.Now().Add(s.lifetime).Format(time.RFC3339),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *clockAuthServer) blockSolves(t *testing.T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocked = make(chan struct{})
	t.Cleanup(func() { close(s.blocked) })
}

func (s *clockAuthServer) service() *Service {
	svc := NewService(s.URL, nil)
	svc.now = s.clock.Now
	return svc
}

// waitForRefresh returns once no refresh is running.
func waitForRefresh(svc *Service) {
	svc.mu.RLock()
	call := svc.refreshing
	svc.mu.RUnlock()
	if call != nil {
		<-call.done
	}
}

func ensureToken(svc *Service) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return svc.EnsureToken(ctx)
}

func TestService_EnsureToken_RequestAtExpiryDoesNotWaitForSolve(t *testing.T) {
	server := newClockAuthServer(t, 10*time.Minute)
	svc := server.service()

	token, err := ensureToken(svc)
	require.NoError(t, err)
	require.Equal(t, "token-1", token)

	// Balance ticks every 2 minutes; the one before expiry still gets the current token.
	server.clock.Advance(8 * time.Minute)
	token, err = ensureToken(svc)
	require.NoError(t, err)
	require.Equal(t, "token-1", token)
	waitForRefresh(svc)

	server.blockSolves(t)
	server.clock.Advance(2 * time.Minute)
	token, err = ensureToken(svc)
	require.NoError(t, err, "a request at expiry waited for the puzzle solve")
	require.Equal(t, "token-2", token)
	require.Equal(t, int32(2), server.solves.Load())
}

func TestService_EnsureToken_NoRefreshEarlyInLifetime(t *testing.T) {
	server := newClockAuthServer(t, 10*time.Minute)
	svc := server.service()

	for range 3 {
		token, err := ensureToken(svc)
		require.NoError(t, err)
		require.Equal(t, "token-1", token)
		waitForRefresh(svc)
		server.clock.Advance(2 * time.Minute)
	}

	require.Equal(t, int32(1), server.solves.Load(), "a token is not replaced while it has most of its lifetime left")
}

func TestService_EnsureToken_FailedBackgroundRefreshBacksOff(t *testing.T) {
	server := newClockAuthServer(t, 10*time.Minute)
	svc := server.service()
	_, err := ensureToken(svc)
	require.NoError(t, err)

	server.failing.Store(true)
	server.clock.Advance(7 * time.Minute)
	token, err := ensureToken(svc)
	require.NoError(t, err)
	require.Equal(t, "token-1", token, "a failed background refresh keeps the current token")
	waitForRefresh(svc)

	server.failing.Store(false)
	server.clock.Advance(10 * time.Second)
	_, err = ensureToken(svc)
	require.NoError(t, err)
	waitForRefresh(svc)
	require.Equal(t, int32(1), server.solves.Load(), "no new puzzle right after a failed one")

	server.clock.Advance(backgroundRetryDelay)
	_, err = ensureToken(svc)
	require.NoError(t, err)
	waitForRefresh(svc)
	require.Equal(t, int32(2), server.solves.Load())
}
