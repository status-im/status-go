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
