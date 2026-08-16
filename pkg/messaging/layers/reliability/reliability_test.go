package reliability

import (
	"crypto/ecdsa"
	"errors"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"
	mvdsstate "github.com/status-im/mvds/state"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability/protobuf"
)

func noopDispatch(*ecdsa.PublicKey, []byte, [][]byte) error { return nil }

func newTestPersistence(t *testing.T) mvdsnode.Persistence {
	t.Helper()
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer(nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, mvdsmigrations.Migrate(db))
	return mvdsnode.NewSQLitePersistence(db)
}

func newTestReliability(t *testing.T) *Reliability {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	r, err := NewReliability(newTestPersistence(t), key, func(string, []string, string) error { return nil }, zap.NewNop())
	require.NoError(t, err)
	return r
}

func TestHandleMissingDependenciesDecodesAllHashes(t *testing.T) {
	r := newTestReliability(t)

	var captured []string
	r.SetMissingDependenciesHandler(func(_ string, missingDeps []string, _ string) error {
		captured = append(captured, missingDeps...)
		return nil
	})

	depOneHint, err := proto.Marshal(&protobuf.RetrievalHint{
		EnvelopeHashes: [][]byte{[]byte("0x01"), []byte("0x02")},
	})
	require.NoError(t, err)
	depTwoHint, err := proto.Marshal(&protobuf.RetrievalHint{
		EnvelopeHashes: [][]byte{[]byte("0x03")},
	})
	require.NoError(t, err)

	r.handleMissingDependencies(
		sds.MessageID("message-id"),
		[]sds.HistoryEntry{
			{MessageID: sds.MessageID("dep-1"), RetrievalHint: depOneHint},
			{MessageID: sds.MessageID("dep-2"), RetrievalHint: depTwoHint},
		},
		"channel-id",
	)

	require.Equal(t, []string{"0x01", "0x02", "0x03"}, captured)
}

func TestHandleSDSMessageSentForwardsMessageID(t *testing.T) {
	r := newTestReliability(t)

	var received string
	r.SetMessageSentHandler(func(messageID string) {
		received = messageID
	})

	r.handleSDSMessageSent(sds.MessageID("0x1234"), "community-chat")
	require.Equal(t, "0x1234", received)
}

func TestHandleMissingDependenciesSkipsInvalidHint(t *testing.T) {
	r := newTestReliability(t)

	called := false
	r.SetMissingDependenciesHandler(func(_ string, _ []string, _ string) error {
		called = true
		return nil
	})

	// Non-protobuf bytes must not reach the handler.
	r.handleMissingDependencies(
		sds.MessageID("message-id"),
		[]sds.HistoryEntry{
			{MessageID: sds.MessageID("dep-1"), RetrievalHint: []byte{0xff, 0xff, 0xff, 0xff}},
		},
		"channel-id",
	)

	require.False(t, called)
}

func TestNewReliabilityRequiresMissingDepsHandler(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	r, err := NewReliability(newTestPersistence(t), key, nil, zap.NewNop())
	require.Error(t, err)
	require.Nil(t, r)
}

func TestNewReliabilityFailsWhenSDSManagerInitFails(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	r, err := newReliabilityWithSDSFactory(
		newTestPersistence(t),
		key,
		func(string, []string, string) error { return nil },
		zap.NewNop(),
		func(*zap.Logger) (*sds.ReliabilityManager, error) {
			return nil, errors.New("boom")
		},
	)
	require.Error(t, err)
	require.Nil(t, r)
	require.ErrorContains(t, err, "failed to create ReliabilityManager")
}

func TestReliabilityStartFailsWhenSDSRebuildFails(t *testing.T) {
	r := newTestReliability(t)
	require.NoError(t, r.Start(noopDispatch))
	r.Close()
	require.Nil(t, r.sdsManager)

	r.sdsManagerFactory = func(*zap.Logger) (*sds.ReliabilityManager, error) {
		return nil, errors.New("boom")
	}

	err := r.Start(noopDispatch)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to create ReliabilityManager")
	require.False(t, r.Started())
}

func TestReliabilityPauseResume(t *testing.T) {
	r := newTestReliability(t)
	require.NoError(t, r.Start(noopDispatch))
	t.Cleanup(r.Stop)

	require.True(t, r.Started())
	require.Less(t, r.currentTick(), time.Second, "running node should tick frequently")

	// Pause: the mvds node is recreated with the long (pausedDuration) interval
	// so its outbound loop idles. The node stays started.
	require.NoError(t, r.SetPaused(true))
	require.True(t, r.Started())
	require.Equal(t, pausedDuration, r.currentTick())

	// Idempotent: a second SetPaused(true) does not recreate the node again.
	require.NoError(t, r.SetPaused(true))
	require.Equal(t, pausedDuration, r.currentTick())

	// Resume: recreated with the normal interval.
	require.NoError(t, r.SetPaused(false))
	require.True(t, r.Started())
	require.Less(t, r.currentTick(), time.Second)

	r.Stop()
	require.False(t, r.Started())
}

func TestReliabilitySetPausedBeforeStart(t *testing.T) {
	r := newTestReliability(t)

	// Before Start: SetPaused only records the state; there is no node yet.
	require.NoError(t, r.SetPaused(true))
	require.False(t, r.Started())

	// Start while paused → the node is built with the paused interval.
	require.NoError(t, r.Start(noopDispatch))
	t.Cleanup(r.Stop)
	require.True(t, r.Started())
	require.Equal(t, pausedDuration, r.currentTick())

	// Resume.
	require.NoError(t, r.SetPaused(false))
	require.Less(t, r.currentTick(), time.Second)
}

// Going offline (Stop) must not discard SDS state: its bloom filter and causal
// history are what detect messages missed while offline. Pausing must not
// either, since the app is merely backgrounded.
func TestReliabilityStopAndPauseKeepSDS(t *testing.T) {
	r := newTestReliability(t)
	require.NoError(t, r.Start(noopDispatch))
	t.Cleanup(r.Close)

	payload := []byte("hello")
	wrapped, _, err := r.WrapPayloadForSDS(payload, "channel-1")
	require.NoError(t, err)
	require.NotEqual(t, payload, wrapped)

	require.NoError(t, r.SetPaused(true))
	require.NotNil(t, r.sdsManager, "pausing must keep SDS alive")
	require.NoError(t, r.SetPaused(false))

	// Offline → online.
	r.Stop()
	require.False(t, r.Started())
	require.NotNil(t, r.sdsManager, "going offline must keep SDS alive")
	require.NoError(t, r.Start(noopDispatch))

	unwrapped, err := r.UnwrapPayloadFromSDS(wrapped)
	require.NoError(t, err)
	require.Equal(t, payload, unwrapped)
}

// Close releases the SDS manager, so a later Start must rebuild it rather than
// leaving every incoming wrapped message undecodable.
func TestReliabilityCloseStartRestoresSDS(t *testing.T) {
	r := newTestReliability(t)
	require.NoError(t, r.Start(noopDispatch))
	t.Cleanup(r.Close)

	r.Close()
	require.Nil(t, r.sdsManager)

	// With no manager, unwrapping must report a failure rather than silently
	// returning the still-wrapped payload.
	_, err := r.UnwrapPayloadFromSDS([]byte("wrapped"))
	require.ErrorIs(t, err, ErrSDSManagerUnavailable)

	require.NoError(t, r.Start(noopDispatch))
	require.NotNil(t, r.sdsManager)

	payload := []byte("hello")
	wrapped, _, err := r.WrapPayloadForSDS(payload, "channel-1")
	require.NoError(t, err)

	unwrapped, err := r.UnwrapPayloadFromSDS(wrapped)
	require.NoError(t, err)
	require.Equal(t, payload, unwrapped)
}

// A payload that was never SDS-wrapped is passed through untouched: wrapping is
// not mandatory, so this must not be reported as an error. Erroring here would
// drop every unwrapped message instead of just delivering it as-is.
func TestUnwrapPayloadFromSDSPassesThroughUnwrappedPayload(t *testing.T) {
	r := newTestReliability(t)
	require.NoError(t, r.Start(noopDispatch))
	t.Cleanup(r.Close)

	payloads := map[string][]byte{
		"text":   []byte("not an sds frame"),
		"binary": {0x0a, 0x2a, 0x08, 0x01, 0x12, 0xff, 0x00, 0xde, 0xad, 0xbe, 0xef},
		"empty":  {},
	}

	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			unwrapped, err := r.UnwrapPayloadFromSDS(payload)
			require.NoError(t, err)
			require.Equal(t, payload, unwrapped)
		})
	}
}

// failingEpochStore makes mvdsnode.NewPersistentNode fail (it reads the epoch
// during construction).
type failingEpochStore struct{}

func (failingEpochStore) Get(mvdsstate.PeerID) (int64, error) {
	return 0, errors.New("epoch read failed")
}
func (failingEpochStore) Set(mvdsstate.PeerID, int64) error { return nil }

// epochFailsOnNthBuild wraps a real Persistence but hands out a failing
// EpochStore on the n-th call (1-indexed) — used to make the recreate inside
// SetPaused fail while the initial Start succeeds.
type epochFailsOnNthBuild struct {
	mvdsnode.Persistence
	n     int
	calls int
}

func (p *epochFailsOnNthBuild) EpochStore() mvdsnode.EpochPersistence {
	p.calls++
	if p.calls == p.n {
		return failingEpochStore{}
	}
	return p.Persistence.EpochStore()
}

func TestReliabilityBuildNodeFailureLeavesNoZombie(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	// Fail the 2nd NewPersistentNode (the recreate in SetPaused); the 1st (Start)
	// succeeds.
	persistence := &epochFailsOnNthBuild{Persistence: newTestPersistence(t), n: 2}
	r, err := NewReliability(persistence, key, func(string, []string, string) error { return nil }, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(r.Stop)

	require.NoError(t, r.Start(noopDispatch))
	require.True(t, r.Started())

	// The recreate fails: SetPaused must report the error and must NOT leave the
	// reliability instance pointing at the stopped old node.
	err = r.SetPaused(true)
	require.Error(t, err)
	require.False(t, r.Started(), "a failed rebuild must not leave a zombie node")

	// A subsequent Start (e.g. on a connection change) recovers — the 3rd
	// NewPersistentNode succeeds.
	require.NoError(t, r.Start(noopDispatch))
	require.True(t, r.Started())
}

// TestReliabilityConcurrent stresses the lifecycle ops against concurrent
// readers (Started / currentTick); run the package with -race to exercise the
// lifecycle locking around concurrent state checks.
func TestReliabilityConcurrent(t *testing.T) {
	r := newTestReliability(t)
	require.NoError(t, r.Start(noopDispatch))
	t.Cleanup(r.Stop)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			_ = r.SetPaused(i%2 == 0)
		}
	}()
	for i := 0; i < 300; i++ {
		_ = r.Started()
		_ = r.currentTick()
	}
	<-done
	require.True(t, r.Started())
}

func TestReliabilityCloseWaitsForInFlightSDSOps(t *testing.T) {
	r := newTestReliability(t)
	require.NoError(t, r.Start(noopDispatch))

	lockHeld := make(chan struct{})
	releaseLock := make(chan struct{})
	go func() {
		r.sdsOpsMu.RLock()
		close(lockHeld)
		<-releaseLock
		r.sdsOpsMu.RUnlock()
	}()

	<-lockHeld

	closed := make(chan struct{})
	go func() {
		r.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close returned while SDS operation lock was still held")
	case <-time.After(50 * time.Millisecond):
		// expected: Close is blocked waiting for in-flight SDS operation
	}

	close(releaseLock)

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after in-flight SDS operation completed")
	}

	require.Nil(t, r.sdsManager)
}
