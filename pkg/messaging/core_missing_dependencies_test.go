package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	mvdsnode "github.com/status-im/mvds/node"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	reliabilitypb "github.com/status-im/status-go/pkg/messaging/layers/reliability/protobuf"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/waku"
)

type fakeMissingDependencyFetcher struct {
	mu        sync.Mutex
	processed map[string]bool
	seen      map[string]bool
	fetches   [][]string
	// fetchErrs[i] is returned by the i-th fetch; fetches beyond it succeed.
	fetchErrs []error
	// onFetch, if set, runs on every fetch (e.g. to mark hashes as processed).
	onFetch  func(attempt int)
	cacheErr error
}

func (f *fakeMissingDependencyFetcher) AlreadyProcessed(hashes []string) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	hits := make(map[string]bool)
	for _, h := range hashes {
		if f.processed[h] {
			hits[h] = true
		}
	}
	return hits, f.cacheErr
}

type fakeSDSDependencyReconciler struct {
	received     map[dependencyKey]bool
	marked       map[string][]string
	checkErr     error
	markErr      error
	markAttempts int
	onMark       func()
	lookup       func(channelID, messageID string) (bool, error)
}

func (r *fakeSDSDependencyReconciler) IsSDSMessageReceived(channelID, messageID string) (bool, error) {
	if r.lookup != nil {
		return r.lookup(channelID, messageID)
	}
	return r.received[dependencyKey{channelID: channelID, messageID: messageID}], r.checkErr
}

func TestMissingDepsFetcher_ExpiredReceiptUsesNormalRecovery(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer(nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, SQLiteMigrate(db, 0))
	store := reliability.NewSQLiteRetrievalHintPersistence(db)
	require.NoError(t, store.SaveSDSMessageReceived("channel", "expired"))
	_, err = db.Exec("UPDATE sds_received_messages SET received_at = ?",
		time.Now().Add(-types.StoreMessageRetention).Unix())
	require.NoError(t, err)
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)
	r := &fakeSDSDependencyReconciler{lookup: store.IsSDSMessageReceived}
	m.reconciler = r
	m.enqueueDependencies("channel", []reliability.MissingDependency{
		{MessageID: "expired", EnvelopeHashes: []string{"missing-envelope"}},
	})
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.Empty(t, r.marked)
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.True(t, step(t, m))
	require.Equal(t, [][]string{{"missing-envelope"}}, f.fetches)
}

func (r *fakeSDSDependencyReconciler) MarkDependenciesMet(ids []string, channelID string) error {
	r.markAttempts++
	if r.markErr != nil {
		return r.markErr
	}
	if r.marked == nil {
		r.marked = make(map[string][]string)
	}
	r.marked[channelID] = append(r.marked[channelID], ids...)
	if r.onMark != nil {
		r.onMark()
	}
	return nil
}

func TestMissingDepsFetcher_WorkerReconcilesWithoutHashFetches(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m := newMissingDepsFetcher(zap.NewNop(), f, newRequestedHashes(time.Hour, 100))
	m.batchDelay = time.Millisecond
	marked := make(chan struct{}, 1)
	r := &fakeSDSDependencyReconciler{
		received: map[dependencyKey]bool{{channelID: "channel", messageID: "local"}: true},
		onMark:   func() { marked <- struct{}{} },
	}
	m.reconciler = r
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer panics.LogOnPanic()
		defer wg.Done()
		m.run(ctx)
	}()
	defer func() {
		cancel()
		wg.Wait()
	}()
	m.enqueueDependencies("channel", []reliability.MissingDependency{{MessageID: "local"}})
	select {
	case <-marked:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not reconcile a dependency with no fetchable hashes")
	}
	cancel()
	wg.Wait()
	require.Empty(t, f.fetches)
	require.Empty(t, m.dependencies)
	require.Equal(t, []string{"local"}, r.marked["channel"])
}

func TestMissingDepsFetcher_ReconcilesWholeDependencies(t *testing.T) {
	f := &fakeMissingDependencyFetcher{processed: map[string]bool{
		"local": true, "segment-1": true, "segment-2": true, "partial-1": true,
	}}
	m, clock := newTestMissingDepsFetcher(f)
	r := &fakeSDSDependencyReconciler{}
	m.reconciler = r
	queued, dropped := m.enqueueDependencies("channel", []reliability.MissingDependency{
		{MessageID: "local-id", EnvelopeHashes: []string{"local"}},
		{MessageID: "segmented-id", EnvelopeHashes: []string{"segment-1", "segment-2"}},
		{MessageID: "partial-id", EnvelopeHashes: []string{"partial-1", "partial-2"}},
		{MessageID: "missing-id", EnvelopeHashes: []string{"missing"}},
	})
	require.Equal(t, 4, queued)
	require.Zero(t, dropped)
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.ElementsMatch(t, []string{"local-id", "segmented-id"}, r.marked["channel"])
	require.Empty(t, f.fetches)

	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.True(t, step(t, m))
	require.Len(t, f.fetches, 1)
	require.ElementsMatch(t, []string{"partial-2", "missing"}, f.fetches[0])
	require.Len(t, m.dependencies, 2)
}

func TestMissingDepsFetcher_ReceivedEvidenceIsChannelScoped(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)
	r := &fakeSDSDependencyReconciler{received: map[dependencyKey]bool{
		{channelID: "a", messageID: "same-id"}: true,
	}}
	m.reconciler = r
	dep := []reliability.MissingDependency{{MessageID: "same-id", EnvelopeHashes: []string{"one", "two"}}}
	m.enqueueDependencies("a", dep)
	m.enqueueDependencies("b", dep)
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.Equal(t, []string{"same-id"}, r.marked["a"])
	require.Empty(t, r.marked["b"])
	require.Empty(t, f.fetches)
	require.Len(t, m.dependencies, 1)
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.True(t, step(t, m))
	require.ElementsMatch(t, []string{"one", "two"}, f.fetches[0])
}

func TestMissingDepsFetcher_SeenIsNotReceived(t *testing.T) {
	f := &fakeMissingDependencyFetcher{seen: map[string]bool{"seen": true}}
	m, clock := newTestMissingDepsFetcher(f)
	r := &fakeSDSDependencyReconciler{}
	m.reconciler = r
	m.enqueueDependencies("channel", []reliability.MissingDependency{
		{MessageID: "seen-id", EnvelopeHashes: []string{"seen"}},
		{MessageID: "no-hint"},
	})
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.Empty(t, r.marked)
	require.Empty(t, f.fetches)
	require.Zero(t, m.pendingCount())
	require.Len(t, m.dependencies, 2, "keep checking envelopes still in processing")
}

func TestMissingDepsFetcher_RechecksBeforeFetching(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)
	r := &fakeSDSDependencyReconciler{received: make(map[dependencyKey]bool)}
	m.reconciler = r
	m.enqueueDependencies("channel", []reliability.MissingDependency{
		{MessageID: "segmented", EnvelopeHashes: []string{"one", "two"}},
	})
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.Equal(t, 2, m.pendingCount())
	// Complete reconstruction need not mark every constituent envelope processed.
	r.received[dependencyKey{channelID: "channel", messageID: "segmented"}] = true
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.False(t, step(t, m))
	require.Empty(t, f.fetches)
	require.Equal(t, []string{"segmented"}, r.marked["channel"])
}

func TestMissingDepsFetcher_RetriesMarkFailureWithoutFetching(t *testing.T) {
	f := &fakeMissingDependencyFetcher{processed: map[string]bool{"local": true}}
	m, clock := newTestMissingDepsFetcher(f)
	r := &fakeSDSDependencyReconciler{markErr: errors.New("injected mark failure")}
	m.reconciler = r
	m.enqueueDependencies("channel", []reliability.MissingDependency{
		{MessageID: "local-id", EnvelopeHashes: []string{"local"}},
	})
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.Equal(t, 1, r.markAttempts)
	require.Len(t, m.dependencies, 1)
	require.Empty(t, f.fetches)
	r.markErr = nil
	clock.advance(missingDepsRetryDelays[0])
	m.reconcile()
	require.Equal(t, []string{"local-id"}, r.marked["channel"])
	require.Empty(t, m.dependencies)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_DoesNotTrustFailedAvailabilityChecks(t *testing.T) {
	f := &fakeMissingDependencyFetcher{
		processed: map[string]bool{"hash": true}, cacheErr: errors.New("cache failure"),
	}
	m, clock := newTestMissingDepsFetcher(f)
	r := &fakeSDSDependencyReconciler{
		received: map[dependencyKey]bool{{channelID: "channel", messageID: "id"}: true},
		checkErr: errors.New("received-message lookup failure"),
	}
	m.reconciler = r
	m.enqueueDependencies("channel", []reliability.MissingDependency{
		{MessageID: "id", EnvelopeHashes: []string{"hash"}},
	})
	clock.advance(missingDepsBatchDelay)
	m.reconcile()
	require.Empty(t, r.marked)
	require.Equal(t, 1, m.pendingCount())
}

func TestMissingDepsFetcher_DependencyQueueBoundedAndDeduplicated(t *testing.T) {
	m, _ := newTestMissingDepsFetcher(&fakeMissingDependencyFetcher{})
	m.maxQueue = 1
	dep := []reliability.MissingDependency{{MessageID: "id"}}
	queued, dropped := m.enqueueDependencies("channel", dep)
	require.Equal(t, 1, queued)
	require.Zero(t, dropped)
	queued, dropped = m.enqueueDependencies("channel", dep)
	require.Zero(t, queued)
	require.Zero(t, dropped)
	queued, dropped = m.enqueueDependencies("another-channel", dep)
	require.Zero(t, queued)
	require.Equal(t, 1, dropped)
	require.Len(t, m.dependencies, 1)
}

func TestMissingDepsFetcher_FreshSDSManagerWithLocalHistory(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(fmt.Sprintf("mixed=%t", mixed), func(t *testing.T) {
			db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer(nil))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			require.NoError(t, SQLiteMigrate(db, 0))
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			newReliability := func(logger *zap.Logger) *reliability.Reliability {
				r, err := reliability.NewReliability(
					mvdsnode.NewSQLitePersistence(db), key,
					func(string, []reliability.MissingDependency, string) error { return nil }, logger,
				)
				require.NoError(t, err)
				r.SetSDSRetrievalHintPersistence(reliability.NewSQLiteRetrievalHintPersistence(db))
				t.Cleanup(r.Close)
				return r
			}
			const channelID = "existing-status-history"
			sender := newReliability(zap.NewNop())
			sender.SetRetrievalHintProvider(func(id string) []byte {
				hint, err := proto.Marshal(&reliabilitypb.RetrievalHint{EnvelopeHashes: [][]byte{[]byte(id)}})
				require.NoError(t, err)
				return hint
			})
			localWrapped, localID, err := sender.WrapPayloadForSDS([]byte("already received"), channelID)
			require.NoError(t, err)
			previous := newReliability(zap.NewNop())
			_, err = previous.UnwrapPayloadFromSDS(localWrapped)
			require.NoError(t, err)
			require.NoError(t, previous.RecordSDSMessageReceived(channelID, localID))
			previous.Close()

			var missingWrapped, missingID []byte
			if mixed {
				missingWrapped, missingID, err = sender.WrapPayloadForSDS([]byte("not received"), channelID)
				require.NoError(t, err)
			}
			waitingWrapped, waitingID, err := sender.WrapPayloadForSDS([]byte("waiting message"), channelID)
			require.NoError(t, err)
			logger, logs := observer.New(zap.DebugLevel)
			receiver := newReliability(zap.New(logger))
			f := &fakeMissingDependencyFetcher{}
			m, clock := newTestMissingDepsFetcher(f)
			m.reconciler = receiver
			callback := make(chan []reliability.MissingDependency, 4)
			receiver.SetMissingDependenciesHandler(func(_ string, deps []reliability.MissingDependency, channelID string) error {
				m.enqueueDependencies(channelID, deps)
				callback <- deps
				return nil
			})
			_, err = receiver.UnwrapPayloadFromSDS(waitingWrapped)
			require.NoError(t, err)
			select {
			case deps := <-callback:
				require.Contains(t, deps, reliability.MissingDependency{
					MessageID: cryptotypes.EncodeHex(localID), EnvelopeHashes: []string{cryptotypes.EncodeHex(localID)},
				})
			case <-time.After(5 * time.Second):
				t.Fatal("SDS did not report the locally available dependency")
			}
			isReady := func() bool {
				for _, entry := range logs.FilterMessage("message ready").All() {
					fields := entry.ContextMap()
					if fields["messageId"] == cryptotypes.EncodeHex(waitingID) && fields["channelId"] == channelID {
						return true
					}
				}
				return false
			}
			require.False(t, isReady(), "payload bytes alone do not prove SDS readiness")
			clock.advance(missingDepsBatchDelay)
			m.reconcile()
			if mixed {
				require.False(t, isReady(), "the genuinely missing dependency must still block SDS")
				f.onFetch = func(int) {
					_, err := receiver.UnwrapPayloadFromSDS(missingWrapped)
					require.NoError(t, err)
				}
				clock.advance(missingDepsBatchDelay)
				m.reconcile()
				require.True(t, step(t, m))
				require.Equal(t, [][]string{{cryptotypes.EncodeHex(missingID)}}, f.fetches)
			} else {
				require.Empty(t, f.fetches)
				require.Zero(t, m.pendingCount())
			}
			require.Eventually(t, func() bool {
				select {
				case <-callback:
					clock.advance(missingDepsBatchDelay)
					m.reconcile()
				default:
				}
				return isReady()
			}, 5*time.Second, time.Millisecond)
		})
	}
}

func (f *fakeMissingDependencyFetcher) EnvelopesSeen(hashes []string) map[string]bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	hits := make(map[string]bool)
	for _, h := range hashes {
		if f.seen[h] {
			hits[h] = true
		}
	}
	return hits
}

func (f *fakeMissingDependencyFetcher) markSeen(h string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.seen == nil {
		f.seen = make(map[string]bool)
	}
	f.seen[h] = true
}

func (f *fakeMissingDependencyFetcher) FetchMessagesByHashes(_ context.Context, hashes []string) error {
	f.mu.Lock()
	attempt := len(f.fetches)
	f.fetches = append(f.fetches, append([]string(nil), hashes...))
	onFetch := f.onFetch
	f.mu.Unlock()
	if onFetch != nil {
		onFetch(attempt)
	}
	if attempt < len(f.fetchErrs) {
		return f.fetchErrs[attempt]
	}
	return nil
}

func (f *fakeMissingDependencyFetcher) markProcessed(h string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.processed == nil {
		f.processed = make(map[string]bool)
	}
	f.processed[h] = true
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestMissingDepsFetcher(f missingDependencyFetcher) (*missingDepsFetcher, *fakeClock) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	requested := newRequestedHashes(time.Hour, 100)
	requested.now = clock.Now
	fetcher := newMissingDepsFetcher(zap.NewNop(), f, requested)
	fetcher.now = clock.Now
	return fetcher, clock
}

// step runs one worker iteration at the current fake time and reports whether
// a batch was taken.
func step(t *testing.T, m *missingDepsFetcher) bool {
	t.Helper()
	batch, _ := m.takeBatch(m.now())
	if len(batch) == 0 {
		return false
	}
	m.fetchBatch(context.Background(), batch)
	return true
}

func TestMissingDepsFetcher_WaitsBatchDelay(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	require.False(t, step(t, m))

	clock.advance(missingDepsBatchDelay)
	require.True(t, step(t, m))
	require.Equal(t, [][]string{{"0x01"}}, f.fetches)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_BatchesCallbacksIntoOneQuery(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01", "0x02"})
	clock.advance(time.Second)
	m.enqueue([]string{"0x02", "0x03"})
	clock.advance(time.Second)

	require.True(t, step(t, m))
	require.False(t, step(t, m))
	require.Equal(t, [][]string{{"0x01", "0x02", "0x03"}}, f.fetches)
}

func TestMissingDepsFetcher_RespectsMaxBatch(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)
	m.maxBatch = 2

	m.enqueue([]string{"0x01", "0x02", "0x03"})
	clock.advance(missingDepsBatchDelay)

	require.True(t, step(t, m))
	require.True(t, step(t, m))
	require.False(t, step(t, m))
	require.Equal(t, [][]string{{"0x01", "0x02"}, {"0x03"}}, f.fetches)
}

func TestMissingDepsFetcher_SkipsRecentlyRequested(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01", "0x02"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	// 0x01/0x02 were not processed (e.g. failed processing or not in store), but
	// must not be requested again; only the new hash is fetched.
	queued, _ := m.enqueue([]string{"0x01", "0x02", "0x03"})
	require.Equal(t, 1, queued)
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	require.Equal(t, [][]string{{"0x01", "0x02"}, {"0x03"}}, f.fetches)
}

func TestMissingDepsFetcher_SkipsAlreadyProcessed(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	f.markProcessed("0x01")
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	require.Empty(t, f.fetches)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_SkipsSeenOnEnqueue(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	f.markSeen("0x01")
	m, clock := newTestMissingDepsFetcher(f)

	queued, _ := m.enqueue([]string{"0x01", "0x02"})
	require.Equal(t, 1, queued)
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	require.Equal(t, [][]string{{"0x02"}}, f.fetches)
	// Seen hashes were never requested from the store, so they are not blocked.
	require.Equal(t, []string{"0x01"}, m.requested.claim([]string{"0x01"}))
}

func TestMissingDepsFetcher_DropsHashesSeenWhileQueued(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01", "0x02"})
	// 0x01 arrives (e.g. through history sync) while waiting for the batch,
	// then fails processing so it is never marked as processed.
	f.markSeen("0x01")
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	require.Equal(t, [][]string{{"0x02"}}, f.fetches)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_NoQueryWhenAllSeen(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01", "0x02"})
	f.markSeen("0x01")
	f.markSeen("0x02")
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	require.Empty(t, f.fetches)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_RetriesWithBackoff(t *testing.T) {
	eof := errors.New("EOF")
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{eof}}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	clock.advance(missingDepsRetryDelays[0] - time.Millisecond)
	require.False(t, step(t, m))
	clock.advance(time.Millisecond)
	require.True(t, step(t, m))

	require.Equal(t, [][]string{{"0x01"}, {"0x01"}}, f.fetches)
	require.Zero(t, m.pendingCount())
}

func TestMissingDepsFetcher_TimeoutUsesLongerBackoff(t *testing.T) {
	timeout := fmt.Errorf("%w: %w", waku.ErrStorenodeTimeout, context.DeadlineExceeded)
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{timeout}}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	clock.advance(missingDepsRetryDelays[0])
	require.False(t, step(t, m))
	clock.advance(missingDepsTimeoutRetryDelays[0] - missingDepsRetryDelays[0])
	require.True(t, step(t, m))
	require.Len(t, f.fetches, 2)
}

func TestMissingDepsFetcher_RetrySkipsAlreadyReceived(t *testing.T) {
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{errors.New("EOF")}}
	// The first attempt delivers 0x01 before the connection drops.
	f.onFetch = func(attempt int) {
		if attempt == 0 {
			f.markProcessed("0x01")
		}
	}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01", "0x02"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)
	clock.advance(missingDepsRetryDelays[0])
	step(t, m)

	require.Equal(t, [][]string{{"0x01", "0x02"}, {"0x02"}}, f.fetches)
}

func TestMissingDepsFetcher_NewHashesJoinDueRetry(t *testing.T) {
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{errors.New("EOF")}}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)

	clock.advance(missingDepsRetryDelays[0])
	m.enqueue([]string{"0x02"})
	require.True(t, step(t, m))

	require.Equal(t, [][]string{{"0x01"}, {"0x01", "0x02"}}, f.fetches)
}

func TestMissingDepsFetcher_GivesUpThenAllowsLaterRequest(t *testing.T) {
	eof := errors.New("EOF")
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{eof, eof, eof, eof}}
	m, clock := newTestMissingDepsFetcher(f)

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	step(t, m)
	for _, d := range missingDepsRetryDelays {
		clock.advance(d)
		require.True(t, step(t, m))
	}
	require.Len(t, f.fetches, len(missingDepsRetryDelays)+1)
	require.Zero(t, m.pendingCount())

	// Exhausted hashes are not re-requested right away...
	queued, _ := m.enqueue([]string{"0x01"})
	require.Zero(t, queued)
	// ...but sooner than successfully requested ones.
	clock.advance(missingDepsExhaustedTTL)
	queued, _ = m.enqueue([]string{"0x01"})
	require.Equal(t, 1, queued)
}

func TestMissingDepsFetcher_DropsOverCapacity(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, clock := newTestMissingDepsFetcher(f)
	m.maxQueue = 2

	queued, dropped := m.enqueue([]string{"0x01", "0x02", "0x03"})
	require.Equal(t, 2, queued)
	require.Equal(t, 1, dropped)

	clock.advance(missingDepsBatchDelay)
	step(t, m)

	// The dropped hash was never requested, so it can be queued again.
	queued, _ = m.enqueue([]string{"0x03"})
	require.Equal(t, 1, queued)
}

func TestMissingDepsFetcher_RepeatedOverflowKeepsRequestedCacheBounded(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	m, _ := newTestMissingDepsFetcher(f)
	m.maxQueue = 2
	maxSize := m.requested.maxSize

	queued, _ := m.enqueue([]string{"0xaa", "0xbb"})
	require.Equal(t, 2, queued)

	// The queue is full, so every callback claims hashes and releases them on
	// drop, leaving stale records behind the live queued ones.
	for i := 0; i < 50*maxSize; i++ {
		_, dropped := m.enqueue([]string{fmt.Sprintf("0xdrop%x", i)})
		require.Equal(t, 1, dropped)
		require.LessOrEqual(t, len(m.requested.order), 2*maxSize)
	}

	require.Equal(t, 2, m.requested.len())
	require.Empty(t, m.requested.claim([]string{"0xaa", "0xbb"}))
}

func TestMissingDepsFetcher_CancelledFetchKeepsPending(t *testing.T) {
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{context.Canceled}}
	m, clock := newTestMissingDepsFetcher(f)
	ctx, cancel := context.WithCancel(context.Background())
	f.onFetch = func(int) { cancel() }

	m.enqueue([]string{"0x01"})
	clock.advance(missingDepsBatchDelay)
	batch, _ := m.takeBatch(m.now())
	m.fetchBatch(ctx, batch)

	require.Equal(t, 1, m.pendingCount())
}

func TestMissingDepsFetcher_RunSingleWorkerFetchesEachHashOnce(t *testing.T) {
	var inFlight, maxInFlight int
	var inFlightMu sync.Mutex
	f := &fakeMissingDependencyFetcher{}
	f.onFetch = func(int) {
		inFlightMu.Lock()
		inFlight++
		maxInFlight = max(maxInFlight, inFlight)
		inFlightMu.Unlock()
		time.Sleep(5 * time.Millisecond)
		inFlightMu.Lock()
		inFlight--
		inFlightMu.Unlock()
	}
	m := newMissingDepsFetcher(zap.NewNop(), f, newRequestedHashes(time.Hour, 100))
	m.batchDelay = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer panics.LogOnPanic()
		defer close(done)
		m.run(ctx)
	}()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer panics.LogOnPanic()
			defer wg.Done()
			m.enqueue([]string{"0x01", "0x02", "0x03"})
		}()
	}
	wg.Wait()

	require.Eventually(t, func() bool { return m.pendingCount() == 0 }, 5*time.Second, 5*time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run did not stop after context cancellation")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	counts := make(map[string]int)
	for _, fetch := range f.fetches {
		for _, h := range fetch {
			counts[h]++
		}
	}
	require.Equal(t, map[string]int{"0x01": 1, "0x02": 1, "0x03": 1}, counts)
	require.Len(t, f.fetches, 1)
	require.Equal(t, 1, maxInFlight)
}

func TestRequestedHashes_ExpiresAfterTTL(t *testing.T) {
	now := time.Unix(0, 0)
	r := newRequestedHashes(time.Minute, 100)
	r.now = func() time.Time { return now }

	require.Equal(t, []string{"0x01"}, r.claim([]string{"0x01"}))
	require.Empty(t, r.claim([]string{"0x01"}))

	now = now.Add(time.Minute)
	require.Equal(t, []string{"0x01"}, r.claim([]string{"0x01"}))
	require.Equal(t, 1, r.len())
}

func TestRequestedHashes_EvictsOldestWhenFull(t *testing.T) {
	now := time.Unix(0, 0)
	r := newRequestedHashes(time.Hour, 2)
	r.now = func() time.Time { return now }

	r.claim([]string{"0x01"})
	now = now.Add(time.Second)
	r.claim([]string{"0x02"})
	now = now.Add(time.Second)
	r.claim([]string{"0x03"})

	require.Equal(t, 2, r.len())
	require.Equal(t, []string{"0x01"}, r.claim([]string{"0x01"}))
	require.Empty(t, r.claim([]string{"0x03"}))
}

func TestRequestedHashes_NilClaimsEverything(t *testing.T) {
	var r *requestedHashes
	require.Equal(t, []string{"0x01", "0x02"}, r.claim([]string{"0x01", "0x02"}))
}

func TestRequestedHashes_ReleaseAndExpireIn(t *testing.T) {
	now := time.Unix(0, 0)
	r := newRequestedHashes(time.Hour, 100)
	r.now = func() time.Time { return now }

	r.claim([]string{"0x01", "0x02"})
	r.release([]string{"0x01"})
	require.Equal(t, []string{"0x01"}, r.claim([]string{"0x01"}))

	r.expireIn([]string{"0x02"}, time.Minute)
	require.Empty(t, r.claim([]string{"0x02"}))
	now = now.Add(time.Minute)
	require.Equal(t, []string{"0x02"}, r.claim([]string{"0x02"}))
}

func TestRequestedHashes_CompactsStaleRecordsBehindLiveHead(t *testing.T) {
	now := time.Unix(0, 0)
	r := newRequestedHashes(time.Hour, 10)
	r.now = func() time.Time { return now }

	r.claim([]string{"0xlive"})
	for i := 0; i < 1000; i++ {
		h := fmt.Sprintf("0xdrop%x", i)
		require.Equal(t, []string{h}, r.claim([]string{h}))
		r.release([]string{h})
		r.expireIn([]string{"0xlive"}, time.Hour)
		require.LessOrEqual(t, len(r.order), 2*r.maxSize)
	}

	require.Equal(t, 1, r.len())
	require.Empty(t, r.claim([]string{"0xlive"}))
}
