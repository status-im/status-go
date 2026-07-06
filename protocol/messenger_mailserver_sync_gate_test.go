package protocol

import (
	"errors"
	"testing"

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
