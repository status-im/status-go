package reliability

import (
	"crypto/ecdsa"
	"errors"
	"testing"
	"time"

	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"
	mvdsstate "github.com/status-im/mvds/state"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/testutils"
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
	return NewReliability(newTestPersistence(t), key, zap.NewNop())
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
	r := NewReliability(persistence, key, zap.NewNop())
	t.Cleanup(r.Stop)

	require.NoError(t, r.Start(noopDispatch))
	require.True(t, r.Started())

	// The recreate fails: SetPaused must report the error and must NOT leave the
	// atomic pointer aimed at the stopped old node.
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
// atomic.Pointer swap and the mutex.
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
