package puzzleauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// stallServer holds /auth/puzzle until release is closed, then answers with puzzleStatus.
type stallServer struct {
	*httptest.Server
	release      chan struct{}
	puzzleStatus int
	releaseOnce  sync.Once
	puzzles      atomic.Int32
	solves       atomic.Int32
}

func newStallServer(t *testing.T, puzzleStatus int) *stallServer {
	s := &stallServer{release: make(chan struct{}), puzzleStatus: puzzleStatus}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/puzzle":
			s.puzzles.Add(1)
			select {
			case <-s.release:
			case <-r.Context().Done():
				return
			}
			if s.puzzleStatus != http.StatusOK {
				w.WriteHeader(s.puzzleStatus)
				return
			}
			_ = json.NewEncoder(w).Encode(testPuzzle())
		case "/auth/solve":
			s.solves.Add(1)
			_ = json.NewEncoder(w).Encode(testTokenResponse(""))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	t.Cleanup(s.unblock)
	return s
}

func (s *stallServer) unblock() { s.releaseOnce.Do(func() { close(s.release) }) }

func TestService_EnsureToken_StalledAuthServerFailsFast(t *testing.T) {
	server := newStallServer(t, http.StatusOK)
	svc := NewService(server.URL, nil)
	svc.authTimeout = 100 * time.Millisecond

	// The circuit breaker gives a provider call 20 s; the puzzle request must give up long before.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	_, err := svc.EnsureToken(ctx)

	require.Error(t, err)
	require.Less(t, time.Since(start), time.Second, "a stalled /auth/puzzle held the request until the caller's deadline")
}

func TestService_EnsureToken_WaitersShareFailedAttempt(t *testing.T) {
	server := newStallServer(t, http.StatusInternalServerError)
	svc := NewService(server.URL, nil)

	const waiters = 10
	errs := make(chan error, waiters)
	for range waiters {
		go func() {
			_, err := svc.EnsureToken(context.Background())
			errs <- err
		}()
	}
	require.Eventually(t, func() bool { return server.puzzles.Load() == 1 }, time.Second, time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	server.unblock()

	for range waiters {
		require.Error(t, <-errs)
	}
	require.Equal(t, int32(1), server.puzzles.Load(), "requests waiting for a failed attempt each started a new one")
}

func TestService_EnsureToken_CancelledCallerDoesNotAbortSharedAttempt(t *testing.T) {
	server := newStallServer(t, http.StatusOK)
	svc := NewService(server.URL, nil)

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderErr := make(chan error, 1)
	go func() {
		_, err := svc.EnsureToken(leaderCtx)
		leaderErr <- err
	}()
	require.Eventually(t, func() bool { return server.puzzles.Load() == 1 }, time.Second, time.Millisecond)

	waiter := make(chan string, 1)
	go func() {
		token, err := svc.EnsureToken(context.Background())
		require.NoError(t, err)
		waiter <- token
	}()
	time.Sleep(50 * time.Millisecond)

	// The circuit breaker gave up on the first caller.
	cancelLeader()
	require.ErrorIs(t, <-leaderErr, context.Canceled)
	server.unblock()

	require.Equal(t, "test-jwt-token", <-waiter)
	require.Equal(t, int32(1), server.puzzles.Load(), "the attempt in flight died with its caller and the next waiter started over")
	require.Equal(t, int32(1), server.solves.Load())
}
