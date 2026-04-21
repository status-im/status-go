package server

import (
	"net/netip"
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
