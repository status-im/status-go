package datasync

import (
	"time"

	mvdsnode "github.com/status-im/mvds/node"
	"github.com/status-im/mvds/state"
)

// Give data and ACKs time to cross both receiver processing windows before a
// reconnect notification can bring a retry forward. Normal due retries keep
// their existing deadlines.
const peerOnlineACKGrace = 5 * time.Second

type peerOnlinePersistence struct {
	mvdsnode.Persistence
	epochDuration time.Duration
	currentEpoch  func() (int64, bool)
}

type peerOnlineSyncState struct {
	state.SyncState
	epochDuration time.Duration
	currentEpoch  func() (int64, bool)
}

// WithPeerOnlineResetPolicy protects pending sends independently of diagnostics.
// The epoch source becomes available after MVDS node construction, before Start.
func WithPeerOnlineResetPolicy(persistence mvdsnode.Persistence, epochDuration time.Duration, currentEpoch func() (int64, bool)) mvdsnode.Persistence {
	return &peerOnlinePersistence{Persistence: persistence, epochDuration: epochDuration, currentEpoch: currentEpoch}
}

func (p *peerOnlinePersistence) StateStore() state.SyncState {
	return &peerOnlineSyncState{SyncState: p.Persistence.StateStore(), epochDuration: p.epochDuration, currentEpoch: p.currentEpoch}
}

func (s *peerOnlineSyncState) MapWithPeerId(peer state.PeerID, process func(state.State) state.State) error {
	// In the pinned MVDS version this operation is used only by resetPeerEpoch.
	return s.SyncState.MapWithPeerId(peer, func(previous state.State) state.State {
		// A presence update must not change a never-sent message's first deadline.
		if previous.SendCount == 0 {
			return previous
		}
		if s.epochDuration <= 0 || s.currentEpoch == nil {
			return previous
		}
		epoch, available := s.currentEpoch()
		if !available {
			return previous
		}
		next := process(previous)
		graceCycles := int64((peerOnlineACKGrace-1)/s.epochDuration) + 1
		minimumEpoch := epoch + graceCycles
		if next.SendEpoch < minimumEpoch {
			next.SendEpoch = minimumEpoch
		}
		// Recovery can accelerate an old retry, but must never postpone an
		// existing deadline, including one already due within the grace window.
		if next.SendEpoch > previous.SendEpoch {
			next.SendEpoch = previous.SendEpoch
		}
		return next
	})
}
