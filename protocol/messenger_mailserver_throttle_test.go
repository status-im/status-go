package protocol

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithHistoricSyncInFlightResetsFlagOnError(t *testing.T) {
	m := &Messenger{logger: zap.NewNop()}

	_, err := m.withHistoricSyncInFlight(time.Now(), func() (*MessengerResponse, error) {
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
	resp, err := m.withHistoricSyncInFlight(time.Now(), func() (*MessengerResponse, error) {
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

func TestWithHistoricSyncInFlightSkipsWhenThrottled(t *testing.T) {
	now := time.Now()
	m := &Messenger{
		logger:                    zap.NewNop(),
		lastHistoricSyncRequestAt: now.Add(-(historicSyncMinInterval / 2)),
	}

	called := false
	resp, err := m.withHistoricSyncInFlight(now, func() (*MessengerResponse, error) {
		called = true
		return &MessengerResponse{}, nil
	})
	require.NoError(t, err)
	require.Nil(t, resp)
	require.False(t, called)

	m.historicSyncMu.Lock()
	require.False(t, m.historicSyncInFlight)
	require.Equal(t, now.Add(-(historicSyncMinInterval / 2)), m.lastHistoricSyncRequestAt)
	m.historicSyncMu.Unlock()
}

func TestWithHistoricSyncInFlightRunsAndUpdatesTimestamp(t *testing.T) {
	now := time.Now()
	old := now.Add(-2 * historicSyncMinInterval)
	m := &Messenger{logger: zap.NewNop(), lastHistoricSyncRequestAt: old}

	called := false
	resp, err := m.withHistoricSyncInFlight(now, func() (*MessengerResponse, error) {
		called = true
		return &MessengerResponse{}, nil
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, called)

	m.historicSyncMu.Lock()
	require.False(t, m.historicSyncInFlight)
	require.Equal(t, now, m.lastHistoricSyncRequestAt)
	m.historicSyncMu.Unlock()
}
