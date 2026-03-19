package connector

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	state := setupTests(t)

	assert.NotNil(t, state.service)
}

func TestService_Start(t *testing.T) {
	state := setupTests(t)
	state.service.config.WSEnabled = true
	state.service.config.WSHost = "127.0.0.1"
	state.service.config.WSPort = 0

	err := state.service.Start()
	assert.NoError(t, err)
	assert.NotNil(t, state.service.wsServer)
	t.Cleanup(func() {
		assert.NoError(t, state.service.Stop())
	})
}

func TestService_Start_WSDisabled_DoesNotCreateWSServer(t *testing.T) {
	state := setupTests(t)
	state.service.config.WSEnabled = false

	err := state.service.Start()
	assert.NoError(t, err)
	assert.Nil(t, state.service.wsServer)
}

func TestService_Stop(t *testing.T) {
	state := setupTests(t)

	err := state.service.Stop()
	assert.NoError(t, err)
}

func TestService_APIs(t *testing.T) {
	state := setupTests(t)

	apis := state.api.s.APIs()

	assert.Len(t, apis, 1)
	assert.Equal(t, "connector", apis[0].Namespace)
	assert.Equal(t, "0.1.0", apis[0].Version)
	assert.NotNil(t, apis[0].Service)
}

func TestService_PauseResumeBackground(t *testing.T) {
	state := setupTests(t)

	// Pause/resume only affects the WS listener path; use a pre-assigned port so we can dial it.
	host, port := freeLocalTCPPort(t)
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	state.service.config.WSEnabled = true
	state.service.config.WSHost = host
	state.service.config.WSPort = port

	t.Cleanup(func() { _ = state.service.Stop() })

	err := state.service.Start()
	require.NoError(t, err)
	require.NotNil(t, state.service.wsServer)
	waitUntilTCPAccepts(t, addr)

	err = state.service.PauseBackground()
	require.NoError(t, err)
	require.True(t, state.service.paused)
	waitUntilTCPRefused(t, addr)

	err = state.service.ResumeForeground()
	require.NoError(t, err)
	require.False(t, state.service.paused)
	waitUntilTCPAccepts(t, addr)

	err = state.service.Stop()
	require.NoError(t, err)
}

// freeLocalTCPPort reserves an ephemeral TCP port on loopback for the test process.
// There is a small race where another process could bind between Close and Start; acceptable for tests.
func freeLocalTCPPort(t *testing.T) (host string, port int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	tcpAddr := ln.Addr().(*net.TCPAddr)
	require.NoError(t, ln.Close())
	return tcpAddr.IP.String(), tcpAddr.Port
}

func waitUntilTCPAccepts(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		c, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = c.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond, "expected TCP accept on %s", addr)
}

func waitUntilTCPRefused(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		c, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if c != nil {
			_ = c.Close()
		}
		return err != nil
	}, 5*time.Second, 10*time.Millisecond, "expected no listener on %s after pause", addr)
}
