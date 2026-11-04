package datasync

import (
	"errors"

	mvdsnode "github.com/status-im/mvds/node"
	"github.com/status-im/mvds/state"

	"github.com/status-im/status-go/pkg/messaging/diagnostics"
)

type tracePersistence struct{ mvdsnode.Persistence }
type traceSyncState struct{ state.SyncState }

// WithTracePersistence observes the actual MVDS state store without changing its results.
func WithTracePersistence(persistence mvdsnode.Persistence) mvdsnode.Persistence {
	if !diagnostics.Enabled() {
		return persistence
	}
	return &tracePersistence{Persistence: persistence}
}

func (p *tracePersistence) StateStore() state.SyncState {
	return &traceSyncState{SyncState: p.Persistence.StateStore()}
}

func (s *traceSyncState) Remove(id state.MessageID, peer state.PeerID) error {
	err := s.SyncState.Remove(id, peer)
	event := diagnostics.Event{Metadata: diagnostics.Metadata{
		Reliability: "mvds", Peer: diagnostics.Hexes([][]byte{peer[:]})[0],
		MVDSMessageIDs: diagnostics.Hexes([][]byte{id[:]}),
	}, Stage: "mvds_state_removed"}
	if errors.Is(err, state.ErrStateNotFound) {
		event.Stage = "mvds_state_not_found"
	} else if err != nil {
		event.Stage = "mvds_state_remove_failed"
	}
	if err != nil {
		event.Error = err.Error()
	}
	diagnostics.Emit(event)
	return err
}
