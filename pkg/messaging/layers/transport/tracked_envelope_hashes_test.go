package transport

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

func TestTrackedEnvelopeHashes_ReturnsAllHashes(t *testing.T) {
	monitor := NewEnvelopesMonitor(nil, EnvelopesMonitorConfig{
		IsMailserver: func(wakutypes.EnodeID) bool { return false },
		Logger:       zap.NewNop(),
	})
	tr := &Transport{envelopesMonitor: monitor}

	id := []byte("message-id")
	hashes := []types.Hash{{0x01}, {0x02}, {0x03}}
	err := monitor.Add([][]byte{id}, hashes, []*wakutypes.NewMessage{{}, {}, {}})
	require.NoError(t, err)

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
