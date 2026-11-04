package datasync

import (
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	mvdsnode "github.com/status-im/mvds/node"
	"github.com/status-im/mvds/state"

	"github.com/status-im/status-go/pkg/messaging/diagnostics"
)

type tracePersistence struct {
	mvdsnode.Persistence
	epochDuration time.Duration
}

type traceSyncState struct {
	state.SyncState
	epochDuration time.Duration
}

var scheduleOperations atomic.Uint64

// WithTracePersistence observes the actual MVDS state store without changing its results.
func WithTracePersistence(persistence mvdsnode.Persistence, epochDuration time.Duration) mvdsnode.Persistence {
	if !diagnostics.Enabled() {
		return persistence
	}
	return &tracePersistence{Persistence: persistence, epochDuration: epochDuration}
}

func (p *tracePersistence) StateStore() state.SyncState {
	return &traceSyncState{SyncState: p.Persistence.StateStore(), epochDuration: p.epochDuration}
}

func (s *traceSyncState) stateEvent(stage string, value state.State) diagnostics.Event {
	stateType := "unknown"
	switch value.Type {
	case state.MESSAGE:
		stateType = "message"
	case state.OFFER:
		stateType = "offer"
	case state.REQUEST:
		stateType = "request"
	}
	return diagnostics.Event{Metadata: diagnostics.Metadata{
		Reliability: "mvds", Peer: diagnostics.Hexes([][]byte{value.PeerID[:]})[0],
		MVDSMessageIDs: diagnostics.Hexes([][]byte{value.MessageID[:]}),
	}, Stage: stage, StateType: stateType, SendCount: &value.SendCount,
		SendEpoch: &value.SendEpoch, EpochDurationNS: int64(s.epochDuration)}
}

func (s *traceSyncState) Add(value state.State) error {
	err := s.SyncState.Add(value)
	event := s.stateEvent("mvds_state_added", value)
	if err != nil {
		event.Stage = "mvds_state_add_failed"
		event.Error = err.Error()
	} else {
		event.OperationCompletedAtNS = time.Now().UnixNano()
	}
	diagnostics.Emit(event)
	return err
}

// Observe the callbacks and original return value without changing the selected
// states, callback results, transaction boundaries, or MVDS dispatch ordering.
func (s *traceSyncState) Map(epoch int64, process func(state.State) state.State) error {
	return s.observeSchedule("mvds_state_scheduled", "epoch_send", nil, &epoch, func(observe func(state.State) state.State) error {
		return s.SyncState.Map(epoch, observe)
	}, process)
}

func (s *traceSyncState) MapWithPeerId(peer state.PeerID, process func(state.State) state.State) error {
	// In the pinned MVDS version this operation is called only by resetPeerEpoch.
	// The reset callback has no current-epoch argument; do not invent one from time.
	return s.observeSchedule("mvds_peer_epoch_reset", "peer_online", &peer, nil, func(observe func(state.State) state.State) error {
		return s.SyncState.MapWithPeerId(peer, observe)
	}, process)
}

func (s *traceSyncState) observeSchedule(stage, reason string, peer *state.PeerID, epoch *int64,
	apply func(func(state.State) state.State) error, process func(state.State) state.State,
) error {
	operationID := strconv.FormatUint(scheduleOperations.Add(1), 10)
	var events []diagnostics.Event
	err := apply(func(previous state.State) state.State {
		next := process(previous)
		event := s.stateEvent(stage, next)
		event.TimestampNS = time.Now().UnixNano()
		event.PreviousSendCount = &previous.SendCount
		event.PreviousSendEpoch = &previous.SendEpoch
		event.CurrentEpoch = epoch
		event.ScheduleOperationID = operationID
		event.Reason = reason
		events = append(events, event)
		return next
	})
	completedAt := time.Now().UnixNano()
	count := len(events)
	summary := diagnostics.Event{Stage: "mvds_schedule_operation_completed", Reason: reason,
		ScheduleOperationID: operationID, CurrentEpoch: epoch, StateCount: &count,
		TimestampNS: completedAt, EpochDurationNS: int64(s.epochDuration)}
	summary.Reliability = "mvds"
	if peer != nil {
		summary.Peer = diagnostics.Hexes([][]byte{peer[:]})[0]
	}
	if err != nil {
		// A callback is only a proposed update. Never emit applied state changes
		// from a failed operation, including rolled-back SQLite transactions.
		summary.Stage = "mvds_schedule_operation_failed"
		summary.Error = err.Error()
		diagnostics.Emit(summary)
		return err
	}
	for _, event := range events {
		event.OperationCompletedAtNS = completedAt
		diagnostics.Emit(event)
	}
	if count > 0 || peer != nil {
		summary.OperationCompletedAtNS = completedAt
		diagnostics.Emit(summary)
	}
	return nil
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
