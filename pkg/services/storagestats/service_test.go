package storagestats

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/signal"
)

func TestServiceRefusesWithoutAnAccount(t *testing.T) {
	s := New(nil, nil, zap.NewNop())
	_, err := s.StartCollect()
	require.ErrorIs(t, err, ErrNoAccount)
}

func TestServiceRefusesASecondCollectionWhileOneRuns(t *testing.T) {
	s := New(setupAppDB(t), nil, zap.NewNop())

	// Hold the guard the way a running collection does, and prove a second
	// start is rejected rather than queued: two concurrent full-table walks
	// would only make both slower.
	s.mu.Lock()
	s.requestID = "in-flight"
	s.mu.Unlock()

	_, err := s.StartCollect()
	require.ErrorIs(t, err, ErrAlreadyRunning)
}

// TestServiceEmitsProgressThenResult exercises the whole delivery path: the
// start call must return immediately and everything else must arrive as
// signals.
func TestServiceEmitsProgressThenResult(t *testing.T) {
	appDB := setupAppDB(t)
	walletDB := setupWalletDB(t)
	seedAccount(t, appDB, walletDB)

	type envelope struct {
		Type  string          `json:"type"`
		Event json.RawMessage `json:"event"`
	}

	// The profile arrives as raw JSON, which is what a client actually forwards
	// to its UI and clipboard.
	done := make(chan resultEnvelope, 1)
	progress := make(chan signal.StorageStatsProgressEvent, 1024)

	signal.SetHandler(func(data []byte) {
		var e envelope
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		switch e.Type {
		case signal.EventStorageStatsProgress:
			var event signal.StorageStatsProgressEvent
			if err := json.Unmarshal(e.Event, &event); err == nil {
				select {
				case progress <- event:
				default:
				}
			}
		case signal.EventStorageStatsResult:
			var event resultEnvelope
			if err := json.Unmarshal(e.Event, &event); err == nil {
				done <- event
			}
		}
	})
	t.Cleanup(signal.ResetHandler)

	s := New(appDB, walletDB, zap.NewNop())
	requestID, err := s.StartCollect()
	require.NoError(t, err)
	require.NotEmpty(t, requestID)

	var result resultEnvelope
	select {
	case result = <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("no storage-stats.result signal arrived")
	}

	require.Empty(t, result.Error)
	require.Equal(t, requestID, result.RequestID)

	// The signal has to carry a profile the client can actually render.
	var profile Profile
	require.NoError(t, json.Unmarshal(result.Data, &profile))
	require.Equal(t, ProfileVersion, profile.ProfileVersion)
	require.Equal(t, int64(200), profile.Messaging.MessagesTotal)

	// Every progress event belongs to this request and carries the same total.
	require.NotEmpty(t, progress)
	close(progress)
	var last signal.StorageStatsProgressEvent
	for event := range progress {
		require.Equal(t, requestID, event.RequestID)
		require.Positive(t, event.Total)
		if last.Total != 0 {
			require.Equal(t, last.Total, event.Total)
			require.Greater(t, event.Step, last.Step)
		}
		last = event
	}
	require.Equal(t, last.Total, last.Step, "the last progress event must complete the walk")

	// The guard has to be released, or the section could never be used twice.
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Empty(t, s.requestID)
}

// captureResults routes storage-stats.result signals into a channel.
func captureResults(t *testing.T) <-chan resultEnvelope {
	t.Helper()

	results := make(chan resultEnvelope, 8)
	signal.SetHandler(func(data []byte) {
		var e struct {
			Type  string          `json:"type"`
			Event json.RawMessage `json:"event"`
		}
		if err := json.Unmarshal(data, &e); err != nil || e.Type != signal.EventStorageStatsResult {
			return
		}
		var event resultEnvelope
		if err := json.Unmarshal(e.Event, &event); err == nil {
			select {
			case results <- event:
			default:
			}
		}
	})
	t.Cleanup(signal.ResetHandler)
	return results
}

type resultEnvelope struct {
	RequestID string          `json:"requestId"`
	Error     string          `json:"error"`
	Data      json.RawMessage `json:"data"`
}

// Every request gets exactly one result, cancellation included: a client that
// clears its in-progress state only on this signal must never be left hanging.
func TestServiceEmitsACancelledResult(t *testing.T) {
	appDB := setupAppDB(t)
	seedAccount(t, appDB, setupWalletDB(t))
	results := captureResults(t)

	s := New(appDB, nil, zap.NewNop())
	requestID, err := s.StartCollect()
	require.NoError(t, err)
	require.NoError(t, s.Stop())

	select {
	case result := <-results:
		require.Equal(t, requestID, result.RequestID)
		require.Equal(t, ErrCancelled.Error(), result.Error)
		require.Empty(t, result.Data)
	case <-time.After(60 * time.Second):
		t.Fatal("a cancelled collection produced no result signal")
	}

	// And the guard is free again, so the next Collect is accepted.
	s.mu.Lock()
	requestIDAfter := s.requestID
	s.mu.Unlock()
	require.Empty(t, requestIDAfter)
}

// Stop must cancel a collection that StartCollect has already returned from.
// The cancel func has to be published together with the guard, or logout races
// the walk and closes sqlcipher under a live COUNT(*).
func TestServiceStopAlwaysCancelsAStartedCollection(t *testing.T) {
	appDB := setupAppDB(t)
	seedAccount(t, appDB, setupWalletDB(t))
	results := captureResults(t)

	for attempt := 0; attempt < 20; attempt++ {
		requestID, err := func() (string, error) {
			s := New(appDB, nil, zap.NewNop())
			id, err := s.StartCollect()
			if err != nil {
				return "", err
			}
			// Stop immediately: the walk may not even have been scheduled yet.
			return id, s.Stop()
		}()
		require.NoError(t, err)

		select {
		case result := <-results:
			require.Equal(t, requestID, result.RequestID)
			require.Equal(t, ErrCancelled.Error(), result.Error,
				"Stop after StartCollect must always cancel, attempt %d", attempt)
		case <-time.After(60 * time.Second):
			t.Fatalf("attempt %d: Stop did not cancel the collection", attempt)
		}
	}
}
