package connector

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/pkg/services/connector/database"
)

func ephemeralBrowserClientID() string {
	return "status-desktop/dapp-browser" + persistence.EphemeralClientIDSuffix
}

func TestService_StartPurgesStaleEphemeralDAppsOnce(t *testing.T) {
	state := setupTests(t)

	require.NoError(t, persistence.UpsertDApp(state.walletDb, &persistence.DApp{
		URL: "https://stale.com", Name: "Stale", IconURL: "",
		ClientID:      ephemeralBrowserClientID(),
		SharedAccount: types.Address{}, ChainID: 0x1,
	}))

	require.NoError(t, state.service.Start())
	require.NoError(t, state.service.Stop())

	got, err := persistence.SelectDApp(state.walletDb, "https://stale.com", ephemeralBrowserClientID())
	require.NoError(t, err)
	require.Nil(t, got, "stale ephemeral row must be purged on first Start")
}

func TestService_StopPurgesEphemeralDApps(t *testing.T) {
	state := setupTests(t)

	require.NoError(t, state.service.Start())

	// Insert ephemeral row after Start (simulates a fresh incognito session).
	require.NoError(t, persistence.UpsertDApp(state.walletDb, &persistence.DApp{
		URL: "https://incognito-dapp2.com", Name: "Incognito DApp 2", IconURL: "",
		ClientID:      ephemeralBrowserClientID(),
		SharedAccount: types.Address{}, ChainID: 0x1,
	}))

	require.NoError(t, state.service.Stop())

	got, err := persistence.SelectDApp(state.walletDb, "https://incognito-dapp2.com", ephemeralBrowserClientID())
	require.NoError(t, err)
	require.Nil(t, got, "ephemeral dApp must be removed on service Stop")
}

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

func TestService_APIPointerStableAcrossPauseResume(t *testing.T) {
	state := setupTests(t)
	state.service.config.WSEnabled = false

	require.NoError(t, state.service.Start())
	apiBefore := state.service.APIs()[0].Service
	require.NoError(t, state.service.Pause())
	require.NoError(t, state.service.Resume())
	apiAfter := state.service.APIs()[0].Service
	require.Same(t, apiBefore, apiAfter)
	require.NoError(t, state.service.Stop())
}

func TestService_ProviderReturnsFreshClientAfterPauseResume(t *testing.T) {
	state := setupTests(t)
	state.service.config.WSEnabled = false

	require.NoError(t, state.service.Start())
	c1 := state.service.GetClient()
	require.NotNil(t, c1)
	require.NoError(t, state.service.Pause())
	require.Nil(t, state.service.GetClient())
	require.NoError(t, state.service.Resume())
	c2 := state.service.GetClient()
	require.NotNil(t, c2)
	require.NotSame(t, c1, c2)
	require.NoError(t, state.service.Stop())
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

	err = state.service.Pause()
	require.NoError(t, err)
	require.True(t, state.service.paused)
	waitUntilTCPRefused(t, addr)

	err = state.service.Resume()
	require.NoError(t, err)
	require.False(t, state.service.paused)
	waitUntilTCPAccepts(t, addr)

	err = state.service.Stop()
	require.NoError(t, err)
}

func TestService_PauseResumePreservesEphemeralDApps(t *testing.T) {
	state := setupTests(t)
	state.service.config.WSEnabled = false

	t.Cleanup(func() { _ = state.service.Stop() })

	require.NoError(t, state.service.Start())

	require.NoError(t, persistence.UpsertDApp(state.walletDb, &persistence.DApp{
		URL: "https://pause-incognito.com", Name: "Pause Test", IconURL: "",
		ClientID:      ephemeralBrowserClientID(),
		SharedAccount: types.Address{}, ChainID: 0x1,
	}))

	require.NoError(t, state.service.Pause())

	got, err := persistence.SelectDApp(state.walletDb, "https://pause-incognito.com", ephemeralBrowserClientID())
	require.NoError(t, err)
	require.NotNil(t, got, "ephemeral dApp must survive Pause")

	require.NoError(t, state.service.Resume())

	got, err = persistence.SelectDApp(state.walletDb, "https://pause-incognito.com", ephemeralBrowserClientID())
	require.NoError(t, err)
	require.NotNil(t, got, "ephemeral dApp must survive Resume")
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
