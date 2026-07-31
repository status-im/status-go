package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetBindAddrPortUsesCachedPortForEphemeralConfig(t *testing.T) {
	s := NewServer(zap.NewNop(), &Config{
		AddrPort: netip.MustParseAddrPort("127.0.0.1:0"),
	})
	s.cachedPort = 45678

	require.Equal(t, netip.MustParseAddrPort("127.0.0.1:45678"), s.getBindAddrPort())
}

func TestServerToBackgroundToForegroundReusesEphemeralPort(t *testing.T) {
	s := NewServer(zap.NewNop(), &Config{
		AddrPort: netip.MustParseAddrPort("127.0.0.1:0"),
	})

	require.NoError(t, s.Start())
	require.Eventually(t, func() bool {
		return s.GetPort() != 0
	}, time.Second, 10*time.Millisecond)

	firstPort := s.GetPort()
	require.NotZero(t, firstPort)

	s.ToBackground()
	require.Eventually(t, func() bool {
		return !s.IsRunning()
	}, time.Second, 10*time.Millisecond)

	s.ToForeground()
	require.Eventually(t, func() bool {
		return s.IsRunning() && s.GetPort() != 0
	}, time.Second, 10*time.Millisecond)

	require.Equal(t, firstPort, s.GetPort())
	require.NoError(t, s.Stop())
}

func TestShouldForceCloseOnShutdownError(t *testing.T) {
	require.False(t, shouldForceCloseOnShutdownError(nil))
	require.True(t, shouldForceCloseOnShutdownError(context.DeadlineExceeded))
	require.True(t, shouldForceCloseOnShutdownError(context.Canceled))
	require.True(t, shouldForceCloseOnShutdownError(fmt.Errorf("wrapped: %w", context.DeadlineExceeded)))
	require.False(t, shouldForceCloseOnShutdownError(errors.New("boom")))
}

func TestServerToBackgroundReturnsPromptlyAndStopsAcceptingConnections(t *testing.T) {
	s := NewServer(zap.NewNop(), &Config{
		AddrPort: netip.MustParseAddrPort("127.0.0.1:0"),
	})

	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	var releaseOnce sync.Once
	s.SetHandlers(HandlerPatternMap{
		"/block": func(w http.ResponseWriter, r *http.Request) {
			startedOnce.Do(func() { close(started) })

			select {
			case <-release:
			case <-r.Context().Done():
			}

			_, _ = io.WriteString(w, "ok")
		},
	})

	require.NoError(t, s.Start())
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		_ = s.Stop()
	})

	require.Eventually(t, func() bool {
		return s.GetListeningAddrPort() != ""
	}, time.Second, 10*time.Millisecond)

	addr := s.GetListeningAddrPort()
	reqErr := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/block")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		reqErr <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocking request to start")
	}

	start := time.Now()
	s.ToBackground()
	elapsed := time.Since(start)

	require.LessOrEqual(t, elapsed, backgroundShutdownTimeout+250*time.Millisecond)

	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err == nil {
		_ = conn.Close()
	}
	require.Error(t, err)

	releaseOnce.Do(func() { close(release) })

	select {
	case <-reqErr:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for blocked request to finish")
	}
}

// Simulates the iOS suspension failure: the OS kills the listening socket
// without the accept loop returning (running stays true), so a later Start()
// must detect the dead listener and rebind instead of early-returning.
func TestStartRebindsWhenListenerDiesSilently(t *testing.T) {
	s := NewServer(zap.NewNop(), &Config{
		AddrPort: netip.MustParseAddrPort("127.0.0.1:0"),
	})

	require.NoError(t, s.Start())
	require.Eventually(t, func() bool {
		return s.GetPort() != 0 && s.listenerAlive()
	}, time.Second, 10*time.Millisecond)
	firstPort := s.GetPort()

	// Kill the listener out from under the server; Serve returns an error and
	// the serve goroutine exits, but on iOS running can remain true — force
	// that state to model it.
	require.NoError(t, s.listener.Close())
	s.serveWg.Wait()
	s.running.Store(true)
	require.False(t, s.listenerAlive())

	require.NoError(t, s.Start())
	require.Eventually(t, func() bool {
		return s.IsRunning() && s.listenerAlive()
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, firstPort, s.GetPort())
	require.NoError(t, s.Stop())
}

// Regression: a handler that never returns (e.g. the media server blocked on a
// stuck IPFS download) used to wedge Stop forever, and with it the whole logout
// path that owns the backend mutex.
func TestServerStopReturnsDespiteStuckHandler(t *testing.T) {
	s := NewServer(zap.NewNop(), &Config{
		AddrPort: netip.MustParseAddrPort("127.0.0.1:0"),
	})

	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	var releaseOnce sync.Once
	s.SetHandlers(HandlerPatternMap{
		// Deliberately ignores r.Context(): Shutdown does not cancel in-flight
		// requests, so only a bounded Stop can break this.
		"/stuck": func(w http.ResponseWriter, r *http.Request) {
			startedOnce.Do(func() { close(started) })
			<-release
		},
	})

	require.NoError(t, s.Start())
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })

	require.Eventually(t, func() bool {
		return s.GetListeningAddrPort() != ""
	}, time.Second, 10*time.Millisecond)

	go func() {
		resp, err := http.Get("http://" + s.GetListeningAddrPort() + "/stuck")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stuck request to start")
	}

	stopped := make(chan error, 1)
	go func() { stopped <- s.Stop() }()

	select {
	case <-stopped:
	case <-time.After(teardownShutdownTimeout + 2*time.Second):
		t.Fatal("Stop did not return while a handler was stuck")
	}

	require.False(t, s.IsRunning())
}
