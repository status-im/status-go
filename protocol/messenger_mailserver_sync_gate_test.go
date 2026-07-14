package protocol

import (
	"errors"
	"testing"
	"time"

	"github.com/status-im/status-go/internal/connection"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithHistoricSyncInFlightResetsFlagOnError(t *testing.T) {
	m := &Messenger{logger: zap.NewNop()}

	_, err := m.withHistoricSyncInFlight(func() (*MessengerResponse, error) {
		return nil, errors.New("boom")
	})
	require.Error(t, err)

	m.historicSyncMu.Lock()
	require.False(t, m.historicSyncInFlight)
	m.historicSyncMu.Unlock()
}

func TestWithHistoricSyncInFlightSkipsWhenAlreadyInFlight(t *testing.T) {
	m := &Messenger{logger: zap.NewNop(), historicSyncInFlight: true}

	called := false
	resp, err := m.withHistoricSyncInFlight(func() (*MessengerResponse, error) {
		called = true
		return &MessengerResponse{}, nil
	})
	require.NoError(t, err)
	require.Nil(t, resp)
	require.False(t, called)

	m.historicSyncMu.Lock()
	require.True(t, m.historicSyncInFlight)
	m.historicSyncMu.Unlock()
}

func TestWithHistoricSyncInFlightRuns(t *testing.T) {
	m := &Messenger{logger: zap.NewNop()}

	called := false
	resp, err := m.withHistoricSyncInFlight(func() (*MessengerResponse, error) {
		called = true
		return &MessengerResponse{}, nil
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, called)

	m.historicSyncMu.Lock()
	require.False(t, m.historicSyncInFlight)
	m.historicSyncMu.Unlock()
}

func TestCapSyncPeriodForNetwork(t *testing.T) {
	t.Run("expensive network caps period", func(t *testing.T) {
		m := &Messenger{connectionState: connection.State{Type: connection.NewType(connection.Cellular)}}

		capped := m.capSyncPeriodForNetwork(uint32(30 * oneDayDuration / time.Second))

		require.Equal(t, uint32(expensiveNetworkMaxSyncDuration/time.Second), capped)
	})

	t.Run("expensive network keeps shorter period", func(t *testing.T) {
		m := &Messenger{connectionState: connection.State{Expensive: true}}

		original := uint32(12 * time.Hour / time.Second)
		capped := m.capSyncPeriodForNetwork(original)

		require.Equal(t, original, capped)
	})

	t.Run("non expensive keeps period", func(t *testing.T) {
		m := &Messenger{connectionState: connection.State{Type: connection.NewType(connection.Wifi)}}

		original := uint32(30 * oneDayDuration / time.Second)
		capped := m.capSyncPeriodForNetwork(original)

		require.Equal(t, original, capped)
	})
}

func TestSyncPeriodFromNow(t *testing.T) {
	t.Run("clamps at zero when sync period is larger than current time", func(t *testing.T) {
		from := syncPeriodFromNow(500, uint32(oneDayDuration/time.Second))
		require.Zero(t, from)
	})

	t.Run("subtracts sync period from current time", func(t *testing.T) {
		from := syncPeriodFromNow(90_000, 10)
		require.Equal(t, uint32(80), from)
	})
}
