package transport

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

func TestTrackedEnvelopeHashes_ReturnsWireHashesOfSentEnvelopes(t *testing.T) {
	monitor := NewEnvelopesMonitor(nil, EnvelopesMonitorConfig{
		IsMailserver: func(wakutypes.EnodeID) bool { return false },
		Logger:       zap.NewNop(),
	})
	tr := &Transport{envelopesMonitor: monitor}

	id := []byte("message-id")
	requestIDs := []wakutypes.RequestID{"req-1", "req-2", "req-3"}
	hashes := []types.Hash{{0x01}, {0x02}, {0x03}}
	require.NoError(t, monitor.Add([][]byte{id}, requestIDs, []*wakutypes.NewMessage{{}, {}, {}}))

	_, err := tr.TrackedEnvelopeHashes(id)
	require.Error(t, err)

	for i := range requestIDs {
		monitor.handleEvent(wakutypes.EnvelopeEvent{Event: wakutypes.EventEnvelopeSent, RequestID: requestIDs[i], Hash: hashes[i]})
	}

	got, err := tr.TrackedEnvelopeHashes(id)
	require.NoError(t, err)
	require.Equal(t, []string{hashes[0].String(), hashes[1].String(), hashes[2].String()}, got)
}

func TestTrackedEnvelopeHashes_NoTrackedHash(t *testing.T) {
	monitor := NewEnvelopesMonitor(nil, EnvelopesMonitorConfig{
		IsMailserver: func(wakutypes.EnodeID) bool { return false },
		Logger:       zap.NewNop(),
	})
	tr := &Transport{envelopesMonitor: monitor}

	_, err := tr.TrackedEnvelopeHashes([]byte("unknown"))
	require.Error(t, err)
}

func TestTrackedEnvelopeHashes_NoMonitor(t *testing.T) {
	tr := &Transport{}

	_, err := tr.TrackedEnvelopeHashes([]byte("message-id"))
	require.Error(t, err)
}

func TestWaitForEnvelopeHashes(t *testing.T) {
	monitor := NewEnvelopesMonitor(nil, EnvelopesMonitorConfig{
		IsMailserver: func(wakutypes.EnodeID) bool { return false },
		Logger:       zap.NewNop(),
	})
	tr := &Transport{envelopesMonitor: monitor}
	id := []byte("message-id")

	require.NoError(t, tr.WaitForEnvelopeHashes(context.Background(), id), "an untracked message has nothing to wait for")

	requestIDs := []wakutypes.RequestID{"req-1", "req-2"}
	require.NoError(t, monitor.Add([][]byte{id}, requestIDs, []*wakutypes.NewMessage{{}, {}}))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, tr.WaitForEnvelopeHashes(ctx, id), context.DeadlineExceeded)

	done := make(chan error, 1)
	go func() { done <- tr.WaitForEnvelopeHashes(context.Background(), id) }()
	monitor.handleEvent(wakutypes.EnvelopeEvent{Event: wakutypes.EventEnvelopeSent, RequestID: requestIDs[0], Hash: types.Hash{0x01}})
	select {
	case <-done:
		t.Fatal("returned before every envelope had a wire hash")
	case <-time.After(50 * time.Millisecond):
	}
	monitor.handleEvent(wakutypes.EnvelopeEvent{Event: wakutypes.EventEnvelopeSent, RequestID: requestIDs[1], Hash: types.Hash{0x02}})
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("did not return once every envelope had a wire hash")
	}
}

func TestWaitForEnvelopeHashesEndsWhenTheSendExpires(t *testing.T) {
	monitor := NewEnvelopesMonitor(nil, EnvelopesMonitorConfig{
		IsMailserver: func(wakutypes.EnodeID) bool { return false },
		Logger:       zap.NewNop(),
	})
	tr := &Transport{envelopesMonitor: monitor}
	id := []byte("message-id")
	require.NoError(t, monitor.Add([][]byte{id}, []wakutypes.RequestID{"req-1"}, []*wakutypes.NewMessage{{}}))

	done := make(chan error, 1)
	go func() { done <- tr.WaitForEnvelopeHashes(context.Background(), id) }()
	monitor.handleEvent(wakutypes.EnvelopeEvent{Event: wakutypes.EventEnvelopeExpired, RequestID: "req-1"})
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("did not return once the send expired")
	}
}
