package datasync

import (
	"testing"
	"time"

	mvdsnode "github.com/status-im/mvds/node"
	"github.com/status-im/mvds/state"
	"github.com/stretchr/testify/require"
)

type peerOnlineTestPersistence struct {
	mvdsnode.Persistence
	store state.SyncState
}

func (p peerOnlineTestPersistence) StateStore() state.SyncState { return p.store }

type peerOnlineTestSyncState struct {
	state.SyncState
	states []state.State
}

func (s *peerOnlineTestSyncState) MapWithPeerId(peer state.PeerID, process func(state.State) state.State) error {
	for i, value := range s.states {
		if value.PeerID == peer {
			s.states[i] = process(value)
		}
	}
	return nil
}

func newPeerOnlineTestStore(states ...state.State) *peerOnlineTestSyncState {
	return &peerOnlineTestSyncState{states: states}
}

func applyPeerOnlineReset(t *testing.T, value state.State, epoch int64, available bool, resetTo int64) state.State {
	t.Helper()
	store := newPeerOnlineTestStore(value)
	persistence := WithPeerOnlineResetPolicy(
		peerOnlineTestPersistence{store: store},
		time.Second,
		func() (int64, bool) { return epoch, available },
	)
	err := persistence.StateStore().MapWithPeerId(value.PeerID, func(s state.State) state.State {
		s.SendEpoch = resetTo
		return s
	})
	require.NoError(t, err)
	require.Len(t, store.states, 1)
	return store.states[0]
}

func TestPeerOnlineResetPolicy(t *testing.T) {
	var peer state.PeerID

	t.Run("does not change never-sent state", func(t *testing.T) {
		value := state.State{PeerID: peer, SendEpoch: 120}
		got := applyPeerOnlineReset(t, value, 100, true, 101)
		require.Equal(t, value, got)
	})

	t.Run("allows overdue retry to move forward but preserves ACK grace", func(t *testing.T) {
		value := state.State{PeerID: peer, SendCount: 2, SendEpoch: 200}
		got := applyPeerOnlineReset(t, value, 100, true, 102)
		require.EqualValues(t, 105, got.SendEpoch)
		require.Less(t, got.SendEpoch, value.SendEpoch)
	})

	t.Run("does not postpone an existing deadline inside grace", func(t *testing.T) {
		value := state.State{PeerID: peer, SendCount: 2, SendEpoch: 103}
		got := applyPeerOnlineReset(t, value, 100, true, 100)
		require.EqualValues(t, value.SendEpoch, got.SendEpoch)
	})

	t.Run("does not postpone a future retry", func(t *testing.T) {
		value := state.State{PeerID: peer, SendCount: 2, SendEpoch: 150}
		got := applyPeerOnlineReset(t, value, 100, true, 180)
		require.EqualValues(t, value.SendEpoch, got.SendEpoch)
	})

	t.Run("does not change state when epoch is unavailable", func(t *testing.T) {
		value := state.State{PeerID: peer, SendCount: 2, SendEpoch: 200}
		got := applyPeerOnlineReset(t, value, 100, false, 101)
		require.Equal(t, value, got)
	})
}
