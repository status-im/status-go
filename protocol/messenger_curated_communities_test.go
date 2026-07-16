package protocol

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/pkg/messaging"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/communities"
)

func TestMessengerCuratedRefreshSuite(t *testing.T) {
	suite.Run(t, new(MessengerCuratedRefreshSuite))
}

type MessengerCuratedRefreshSuite struct {
	suite.Suite

	messagingEnv *messaging.TestMessagingEnvironment
	m            *Messenger
}

func (s *MessengerCuratedRefreshSuite) SetupTest() {
	var err error
	s.messagingEnv, err = messaging.NewTestMessagingEnvironment()
	s.Require().NoError(err)
	s.Require().NoError(s.messagingEnv.Setup(s.T()))

	s.m, err = newTestMessenger(s.T(), s.messagingEnv, testMessengerConfig{})
	s.Require().NoError(err)
}

// curatedRefreshSignals records the curated refresh signals in emission order.
type curatedRefreshSignals struct {
	MessengerSignalsHandler

	mu              sync.Mutex
	order           []string
	resolved        map[string]bool
	resolvedCount   int
	unresolvedCount int
	cancelled       bool
	finished        chan struct{}
}

func newCuratedRefreshSignals() *curatedRefreshSignals {
	return &curatedRefreshSignals{
		resolved: map[string]bool{},
		finished: make(chan struct{}, 1),
	}
}

func (r *curatedRefreshSignals) CuratedCommunitiesRefreshStarted() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, "started")
}

func (r *curatedRefreshSignals) CuratedCommunityResolved(communityID string, stored bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, "resolved")
	r.resolved[communityID] = stored
}

func (r *curatedRefreshSignals) CuratedCommunitiesRefreshFinished(resolved, unresolved int, cancelled bool) {
	r.mu.Lock()
	r.order = append(r.order, "finished")
	r.resolvedCount = resolved
	r.unresolvedCount = unresolved
	r.cancelled = cancelled
	r.mu.Unlock()
	r.finished <- struct{}{}
}

func (r *curatedRefreshSignals) count(event string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.order {
		if e == event {
			n++
		}
	}
	return n
}

func (s *MessengerCuratedRefreshSuite) waitFinished(mock *curatedRefreshSignals) {
	select {
	case <-mock.finished:
	case <-time.After(5 * time.Second):
		s.FailNow("refresh did not finish in time")
	}
}

func curatedDirectory(ids ...string) func() (*communities.CuratedCommunities, error) {
	return func() (*communities.CuratedCommunities, error) {
		return &communities.CuratedCommunities{ContractCommunities: ids}, nil
	}
}

// TestCuratedCommunitiesReadIsSideEffectFree verifies that reading the curated
// communities returns stored data only and never triggers a store node request.
func (s *MessengerCuratedRefreshSuite) TestCuratedCommunitiesReadIsSideEffectFree() {
	var batches int32
	s.m.storeNodeRequestsManager.onPerformingBatch = func(types2.StoreNodeBatch) {
		atomic.AddInt32(&batches, 1)
	}

	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities:         []string{unknownCommunityID},
		ContractFeaturedCommunities: []string{unknownCommunityID},
	}))

	response, err := s.m.CuratedCommunities()
	s.Require().NoError(err)
	s.Require().Contains(response.UnknownCommunities, unknownCommunityID)
	s.Require().Equal([]string{unknownCommunityID}, response.ContractFeaturedCommunities)

	s.Require().Equal(int32(0), atomic.LoadInt32(&batches), "a pure read must not issue store node requests")
}

// TestResolveRespectsParallelismAndCounts verifies bounded parallelism of 3
// across entries, a per-community resolved signal for every entry, and correct
// resolved/unresolved counts.
func (s *MessengerCuratedRefreshSuite) TestResolveRespectsParallelismAndCounts() {
	mock := newCuratedRefreshSignals()
	s.m.config.messengerSignalsHandler = mock

	ids := []string{"0x01", "0x02", "0x03", "0x04", "0x05", "0x06"}
	stored := map[string]bool{"0x01": true, "0x03": true, "0x05": true, "0x06": true}

	var (
		mu      sync.Mutex
		current int
		maxSeen int
	)
	resolve := func(ctx context.Context, id string) bool {
		mu.Lock()
		current++
		if current > maxSeen {
			maxSeen = current
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		current--
		mu.Unlock()
		return stored[id]
	}

	resolved, unresolved := s.m.resolveCuratedCommunities(context.Background(), ids, resolve)

	mu.Lock()
	s.Require().LessOrEqual(maxSeen, 3, "must not resolve more than 3 entries concurrently")
	mu.Unlock()

	s.Require().Equal(4, resolved)
	s.Require().Equal(2, unresolved)
	s.Require().Equal(len(ids), mock.count("resolved"), "one resolved signal per entry")
	for _, id := range ids {
		got, ok := mock.resolved[id]
		s.Require().True(ok, "missing resolved signal for %s", id)
		s.Require().Equal(stored[id], got)
	}
}

// TestRefreshIsSingleFlight verifies that a second refresh started while one is
// in progress joins the running one: it starts no new run and no entry is
// fetched twice.
func (s *MessengerCuratedRefreshSuite) TestRefreshIsSingleFlight() {
	mock := newCuratedRefreshSignals()
	s.m.config.messengerSignalsHandler = mock

	ids := []string{"0x01", "0x02", "0x03"}
	release := make(chan struct{})
	calls := &sync.Map{}
	resolve := func(ctx context.Context, id string) bool {
		c, _ := calls.LoadOrStore(id, new(int32))
		atomic.AddInt32(c.(*int32), 1)
		<-release
		return true
	}

	s.Require().NoError(s.m.refreshCuratedCommunities(curatedDirectory(ids...), resolve))
	// Second refresh while the first is still running must join it.
	s.Require().NoError(s.m.refreshCuratedCommunities(curatedDirectory(ids...), resolve))

	close(release)
	s.waitFinished(mock)

	s.Require().Equal(1, mock.count("started"), "only one run must start")
	s.Require().Equal(3, mock.resolvedCount)
	calls.Range(func(key, value any) bool {
		s.Require().Equal(int32(1), atomic.LoadInt32(value.(*int32)), "entry %v must be fetched once", key)
		return true
	})
}

// TestStopCancelsRefresh verifies that stopping cancels the in-flight refresh:
// no further entries are resolved and the finished signal reports cancelled.
func (s *MessengerCuratedRefreshSuite) TestStopCancelsRefresh() {
	mock := newCuratedRefreshSignals()
	s.m.config.messengerSignalsHandler = mock

	ids := []string{"0x01", "0x02", "0x03", "0x04", "0x05", "0x06"}
	entered := make(chan struct{}, len(ids))
	release := make(chan struct{})
	resolve := func(ctx context.Context, id string) bool {
		entered <- struct{}{}
		<-release
		return ctx.Err() == nil
	}

	s.Require().NoError(s.m.refreshCuratedCommunities(curatedDirectory(ids...), resolve))

	// Wait until the first bounded batch is in flight, then cancel.
	for i := 0; i < 3; i++ {
		<-entered
	}
	s.m.StopCuratedCommunitiesRefresh()
	close(release)

	s.waitFinished(mock)

	s.Require().True(mock.cancelled)
	s.Require().Less(mock.resolvedCount+mock.unresolvedCount, len(ids), "cancel must stop launching new entries")
}

// TestRefreshEmitsSignalsInOrder verifies the started -> resolved -> finished
// ordering and the finished counts for a successful refresh.
func (s *MessengerCuratedRefreshSuite) TestRefreshEmitsSignalsInOrder() {
	mock := newCuratedRefreshSignals()
	s.m.config.messengerSignalsHandler = mock

	ids := []string{"0x01", "0x02"}
	resolve := func(ctx context.Context, id string) bool {
		return id == "0x01"
	}

	s.Require().NoError(s.m.refreshCuratedCommunities(curatedDirectory(ids...), resolve))
	s.waitFinished(mock)

	mock.mu.Lock()
	order := append([]string{}, mock.order...)
	mock.mu.Unlock()

	s.Require().Equal("started", order[0])
	s.Require().Equal("finished", order[len(order)-1])
	s.Require().Equal(2, mock.count("resolved"))
	s.Require().Equal(1, mock.resolvedCount)
	s.Require().Equal(1, mock.unresolvedCount)
	s.Require().False(mock.cancelled)
	s.Require().True(mock.resolved["0x01"])
	s.Require().False(mock.resolved["0x02"])
}

// TestStartIssuesNoCuratedBatch pins that a started messenger never resolves the
// curated directory on its own: with a curated directory containing an unknown
// community, no store node batch is issued unless a refresh is requested.
func (s *MessengerCuratedRefreshSuite) TestStartIssuesNoCuratedBatch() {
	var batches int32
	s.m.storeNodeRequestsManager.onPerformingBatch = func(types2.StoreNodeBatch) {
		atomic.AddInt32(&batches, 1)
	}

	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknownCommunityID},
	}))

	s.Require().NoError(s.m.messaging.Start())
	s.T().Cleanup(func() { s.Require().NoError(s.m.messaging.Stop()) })
	_, err := s.m.Start()
	s.Require().NoError(err)
	s.T().Cleanup(func() { s.Require().NoError(s.m.Shutdown()) })

	time.Sleep(200 * time.Millisecond)

	s.Require().Equal(int32(0), atomic.LoadInt32(&batches), "start must not issue curated store node requests")
}
