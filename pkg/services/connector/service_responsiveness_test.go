package connector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/pausable/pausabletest"
	persistence "github.com/status-im/status-go/pkg/services/connector/database"
	"github.com/status-im/status-go/pkg/services/connector/walletconnect/relaytest"
)

// setupServiceWithSession returns a service that talks to relay and has one
// active WalletConnect session, so Resume has a relay to reconnect to.
func setupServiceWithSession(t *testing.T, relay *relaytest.Server) *Service {
	t.Helper()
	state := setupTests(t)
	s := state.service
	s.config.WSEnabled = false
	s.config.RelayURL = relay.URL()

	require.NoError(t, persistence.UpsertDApp(state.walletDb, &persistence.DApp{
		URL:      "https://dapp.example",
		Name:     "dapp",
		ClientID: persistence.WCClientID,
	}))
	now := time.Now().Unix()
	symKey := "0f0e0d0c0b0a09080706050403020100000102030405060708090a0b0c0d0e0f"
	require.NoError(t, persistence.UpsertWCSession(state.walletDb, "session-topic", "{}", now+3600,
		"pairing-topic", "https://dapp.example", symKey, now))

	t.Cleanup(func() { _ = s.Stop() })
	return s
}

func requireLifecycleCall(t *testing.T, op string, f func() error) {
	t.Helper()
	pausabletest.RequireWithin(t, "connector", op, pausabletest.Budget, f)
}

func TestService_LifecycleIsResponsiveWhileRelayIsUnreachable(t *testing.T) {
	for name, mode := range map[string]relaytest.Mode{
		"blackhole": relaytest.Blackhole,
		"reject":    relaytest.Reject,
	} {
		t.Run(name, func(t *testing.T) {
			relay := relaytest.New(t, mode)
			s := setupServiceWithSession(t, relay)

			requireLifecycleCall(t, "Start", s.Start)
			for i := 0; i < 2; i++ {
				requireLifecycleCall(t, "Pause", s.Pause)
				requireLifecycleCall(t, "Resume", s.Resume)
			}
			requireLifecycleCall(t, "Stop", s.Stop)
		})
	}
}

func TestService_PauseIsResponsiveWhileRelayReconnects(t *testing.T) {
	relay := relaytest.New(t, relaytest.Healthy)
	s := setupServiceWithSession(t, relay)

	require.NoError(t, s.Start())
	require.NoError(t, s.Pause())
	requireLifecycleCall(t, "Resume", s.Resume)
	require.Eventually(t, func() bool { return relay.Subscribes() >= 1 }, 3*time.Second, 10*time.Millisecond,
		"session was not re-subscribed after resume")

	relay.SetMode(relaytest.Blackhole)
	relay.DropAll()
	require.Eventually(t, func() bool { return relay.Held() >= 1 }, 5*time.Second, 10*time.Millisecond,
		"relay client did not start reconnecting")

	requireLifecycleCall(t, "Pause", s.Pause)
}
