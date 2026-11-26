package protocol

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum"

	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/contracts"
	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
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

func (s *MessengerCuratedRefreshSuite) TearDownTest() {
	releaseTestMessenger(s.T(), s.m)
}

func (s *MessengerCuratedRefreshSuite) newCommunityID() string {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	return types.EncodeHex(crypto.CompressPubkey(&key.PublicKey))
}

// countStoreNodeBatches counts the store node requests the messenger issues.
func (s *MessengerCuratedRefreshSuite) countStoreNodeBatches() *int32 {
	var batches int32
	s.m.storeNodeRequestsManager.onPerformingBatch = func(messagingtypes.StoreNodeBatch) {
		atomic.AddInt32(&batches, 1)
	}
	return &batches
}

func (s *MessengerCuratedRefreshSuite) TestCuratedCommunitiesReadIssuesNoStoreNodeRequest() {
	batches := s.countStoreNodeBatches()
	unknownCommunityID := s.newCommunityID()

	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities:         []string{unknownCommunityID},
		ContractFeaturedCommunities: []string{unknownCommunityID},
	}))

	response, err := s.m.CuratedCommunities()
	s.Require().NoError(err)
	s.Require().Equal([]string{unknownCommunityID}, response.ContractCommunities)
	s.Require().Equal([]string{unknownCommunityID}, response.UnknownCommunities)
	s.Require().Equal([]string{unknownCommunityID}, response.ContractFeaturedCommunities)
	s.Require().Empty(response.Descriptions)

	// Whatever the read started in the background has finished by now.
	s.m.shutdownWaitGroup.Wait()
	s.Require().Zero(atomic.LoadInt32(batches), "reading the curated communities must not query store nodes")
}

// storeCommunity stores a community the way one received from the network
// is stored and returns its ID.
func (s *MessengerCuratedRefreshSuite) storeCommunity() string {
	community := s.newReceivedCommunity()
	s.Require().NoError(s.m.communitiesManager.GetPersistence().SaveCommunity(community))
	return community.IDString()
}

// newReceivedCommunity builds a community as one received from the network.
func (s *MessengerCuratedRefreshSuite) newReceivedCommunity() *communities.Community {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	description := &protobuf.CommunityDescription{Clock: 1}
	controlNode, err := communities.New(communities.Config{
		MemberIdentity:       s.m.identity,
		PrivateKey:           key,
		ControlNode:          &key.PublicKey,
		ControlDevice:        true,
		ID:                   &key.PublicKey,
		CommunityDescription: description,
	}, s.m.getTimesource(), nil, nil)
	s.Require().NoError(err)
	wrappedDescription, err := controlNode.ToProtocolMessageBytes()
	s.Require().NoError(err)

	community, err := communities.New(communities.Config{
		MemberIdentity:                      s.m.identity,
		ControlNode:                         &key.PublicKey,
		ID:                                  &key.PublicKey,
		CommunityDescription:                description,
		CommunityDescriptionProtocolMessage: wrappedDescription,
	}, s.m.getTimesource(), nil, nil)
	s.Require().NoError(err)
	return community
}

type curatedRefreshEvent struct {
	name            string
	changed         bool
	directoryFailed bool
	resolved        int
	notFound        int
	failed          int
	cancelled       bool
}

// curatedRefreshSignals records the curated refresh signals in emission order.
type curatedRefreshSignals struct {
	MessengerSignalsHandlerMock

	mu     sync.Mutex
	events []curatedRefreshEvent
	// runIDs holds the run ID of each event.
	runIDs   []uint64
	finished chan struct{}
	// onStarted runs when the started signal is emitted.
	onStarted func()
}

func (s *MessengerCuratedRefreshSuite) recordSignals() *curatedRefreshSignals {
	signals := &curatedRefreshSignals{finished: make(chan struct{}, 10)}
	s.m.config.messengerSignalsHandler = signals
	return signals
}

func (r *curatedRefreshSignals) record(runID uint64, event curatedRefreshEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	r.runIDs = append(r.runIDs, runID)
}

func (r *curatedRefreshSignals) CuratedCommunitiesRefreshStarted(runID uint64, changed, directoryFailed bool) {
	r.record(runID, curatedRefreshEvent{name: "started", changed: changed, directoryFailed: directoryFailed})
	if r.onStarted != nil {
		r.onStarted()
	}
}

func (r *curatedRefreshSignals) CuratedCommunitiesRefreshFinished(runID uint64, resolved, notFound, failed int, cancelled bool) {
	r.record(runID, curatedRefreshEvent{name: "finished", resolved: resolved, notFound: notFound, failed: failed, cancelled: cancelled})
	r.finished <- struct{}{}
}

func (r *curatedRefreshSignals) recorded() []curatedRefreshEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]curatedRefreshEvent{}, r.events...)
}

func (r *curatedRefreshSignals) recordedRunIDs() []uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]uint64{}, r.runIDs...)
}

func (s *MessengerCuratedRefreshSuite) refresh(directory curatedDirectoryReader, resolve curatedCommunityResolver) error {
	_, err := s.m.refreshCuratedCommunities(directory, resolve)
	return err
}

func (s *MessengerCuratedRefreshSuite) refreshFromContract() error {
	_, err := s.m.RefreshCuratedCommunities()
	return err
}

func (s *MessengerCuratedRefreshSuite) waitFinished(signals *curatedRefreshSignals) {
	select {
	case <-signals.finished:
	case <-time.After(5 * time.Second):
		s.FailNow("refresh did not finish in time")
	}
}

func (s *MessengerCuratedRefreshSuite) receive(from <-chan string, what string) string {
	select {
	case id := <-from:
		return id
	case <-time.After(5 * time.Second):
		s.FailNow("timed out: " + what)
		return ""
	}
}

// waitUnlessAborted lets a blocked resolver go when its test has ended.
func waitUnlessAborted(blocked, aborted <-chan struct{}) {
	select {
	case <-blocked:
	case <-aborted:
	}
}

// waitIdle waits for the messenger's background work, the refresh included.
func (s *MessengerCuratedRefreshSuite) waitIdle() {
	idle := make(chan struct{})
	go func() {
		defer panics.LogOnPanic()
		s.m.shutdownWaitGroup.Wait()
		close(idle)
	}()
	select {
	case <-idle:
	case <-time.After(5 * time.Second):
		s.FailNow("refresh did not end in time")
	}
}

func curatedDirectory(featured string, ids ...string) curatedDirectoryReader {
	return func(context.Context) (*communities.CuratedCommunities, error) {
		return &communities.CuratedCommunities{
			ContractCommunities:         ids,
			ContractFeaturedCommunities: []string{featured},
		}, nil
	}
}

// Only the entries the refresh has to fetch are counted: the ones already
// stored are read with CuratedCommunities.
func (s *MessengerCuratedRefreshSuite) TestRefreshStoresDirectoryAndCountsFetchedEntries() {
	signals := s.recordSignals()
	known := s.storeCommunity()
	resolvable, unresolvable, failing := s.newCommunityID(), s.newCommunityID(), s.newCommunityID()

	var mu sync.Mutex
	var attempted []string
	var directoryAtFirstAttempt []string
	resolve := func(ctx context.Context, id string) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		if len(attempted) == 0 {
			response, err := s.m.CuratedCommunities()
			if err == nil {
				directoryAtFirstAttempt = response.ContractCommunities
			}
		}
		attempted = append(attempted, id)
		if id == failing {
			return false, errors.New("store node unreachable")
		}
		return id == resolvable, nil
	}

	s.Require().NoError(s.refresh(curatedDirectory(known, known, resolvable, unresolvable, failing), resolve))
	s.waitFinished(signals)

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", changed: true},
		{name: "finished", resolved: 1, notFound: 1, failed: 1},
	}, signals.recorded())
	s.Require().ElementsMatch([]string{resolvable, unresolvable, failing}, attempted, "only entries with no stored description are resolved")
	s.Require().Equal([]string{known, resolvable, unresolvable, failing}, directoryAtFirstAttempt, "the directory is stored before any entry is resolved")

	response, err := s.m.CuratedCommunities()
	s.Require().NoError(err)
	s.Require().Equal([]string{known, resolvable, unresolvable, failing}, response.ContractCommunities)
	s.Require().Equal([]string{known}, response.ContractFeaturedCommunities)
	s.Require().Contains(response.Descriptions, known)
	s.Require().Equal([]string{resolvable, unresolvable, failing}, response.UnknownCommunities)
}

// The app reads the directory when the refresh starts: the list read from
// the contract is stored by then.
func (s *MessengerCuratedRefreshSuite) TestRefreshStartsOnceDirectoryIsStored() {
	signals := s.recordSignals()
	first, second := s.newCommunityID(), s.newCommunityID()

	var atStart *communities.KnownCommunitiesResponse
	signals.onStarted = func() {
		atStart, _ = s.m.CuratedCommunities()
	}
	resolve := func(ctx context.Context, id string) (bool, error) {
		return false, nil
	}

	s.Require().NoError(s.refresh(curatedDirectory(first, first, second), resolve))
	s.waitFinished(signals)

	s.Require().NotNil(atStart)
	s.Require().Equal([]string{first, second}, atStart.ContractCommunities)
	s.Require().Equal([]string{first}, atStart.ContractFeaturedCommunities)
	s.Require().Equal([]string{first, second}, atStart.UnknownCommunities)
	s.Require().Equal("started", signals.recorded()[0].name)
}

// An entry that store nodes never answer for holds one fetch, not the
// refresh: the other entries are fetched next to it.
func (s *MessengerCuratedRefreshSuite) TestRefreshFetchesTwoEntriesAtATime() {
	signals := s.recordSignals()
	ids := make([]string, 7)
	for i := range ids {
		ids[i] = s.newCommunityID()
	}
	stuck := ids[0]

	var current, maxSeen int32
	entered := make(chan string, len(ids))
	returned := make(chan string, len(ids))
	gate := make(chan struct{})
	unblock := make(chan struct{})
	abort := make(chan struct{})
	defer close(abort)
	resolve := func(ctx context.Context, id string) (bool, error) {
		n := atomic.AddInt32(&current, 1)
		for {
			seen := atomic.LoadInt32(&maxSeen)
			if n <= seen || atomic.CompareAndSwapInt32(&maxSeen, seen, n) {
				break
			}
		}
		entered <- id
		waitUnlessAborted(gate, abort)
		if id == stuck {
			waitUnlessAborted(unblock, abort)
		}
		atomic.AddInt32(&current, -1)
		returned <- id
		return false, nil
	}

	s.Require().NoError(s.refresh(curatedDirectory("", ids...), resolve))
	for i := 0; i < 2; i++ {
		s.receive(entered, "two entries are fetched at once")
	}
	select {
	case id := <-entered:
		s.FailNow("a third entry is fetched while two are in flight", id)
	case <-time.After(100 * time.Millisecond):
	}

	close(gate)
	for i := 0; i < len(ids)-1; i++ {
		s.Require().NotEqual(stuck, s.receive(returned, "the other entries are fetched while one is stuck"))
	}
	s.Require().Len(signals.recorded(), 1, "the refresh has not finished while an entry is in flight")

	close(unblock)
	s.waitFinished(signals)

	s.Require().Equal(int32(2), atomic.LoadInt32(&maxSeen))
	events := signals.recorded()
	s.Require().Equal(curatedRefreshEvent{name: "finished", notFound: 7}, events[len(events)-1])
}

// The counts and the not found record are the same in whatever order the
// fetches end.
func (s *MessengerCuratedRefreshSuite) TestRefreshCountsDoNotDependOnCompletionOrder() {
	signals := s.recordSignals()
	known := s.storeCommunity()
	resolvable := []string{s.newCommunityID(), s.newCommunityID()}
	unresolvable := []string{s.newCommunityID(), s.newCommunityID()}
	failing := []string{s.newCommunityID(), s.newCommunityID()}
	fetched := []string{resolvable[0], unresolvable[0], failing[0], failing[1], unresolvable[1], resolvable[1]}
	directory := curatedDirectory("", append([]string{known, "0xnot-hex"}, fetched...)...)

	var mu sync.Mutex
	var attempted []string
	release := make(map[string]chan struct{}, len(fetched))
	for _, id := range fetched {
		release[id] = make(chan struct{})
	}
	returned := make(chan string, 2*len(fetched))
	abort := make(chan struct{})
	defer close(abort)
	resolve := func(ctx context.Context, id string) (bool, error) {
		mu.Lock()
		attempted = append(attempted, id)
		blocked := release[id]
		mu.Unlock()
		waitUnlessAborted(blocked, abort)
		defer func() { returned <- id }()
		switch id {
		case resolvable[0], resolvable[1]:
			return true, nil
		case failing[0], failing[1]:
			return false, errors.New("store node unreachable")
		}
		return false, nil
	}

	s.Require().NoError(s.refresh(directory, resolve))
	// The later of the two entries in flight ends first.
	for _, i := range []int{1, 0, 3, 2, 5, 4} {
		close(release[fetched[i]])
		s.Require().Equal(fetched[i], s.receive(returned, "the released fetch ends"))
	}
	s.waitFinished(signals)
	s.waitIdle()

	s.Require().ElementsMatch(fetched, attempted)

	// The entries recorded as not found are not fetched again. The resolver
	// stores nothing, so the resolved ones are.
	mu.Lock()
	attempted = nil
	mu.Unlock()
	s.Require().NoError(s.refresh(directory, resolve))
	s.waitFinished(signals)

	s.Require().ElementsMatch(append(resolvable, failing...), attempted)
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", changed: true},
		{name: "finished", resolved: 2, notFound: 3, failed: 2},
		{name: "started"},
		{name: "finished", resolved: 2, failed: 2},
	}, signals.recorded())
}

// A refresh requested while one is in flight joins it.
func (s *MessengerCuratedRefreshSuite) TestRefreshIsSingleFlight() {
	signals := s.recordSignals()
	ids := []string{s.newCommunityID(), s.newCommunityID(), s.newCommunityID()}

	release := make(chan struct{})
	var attempts int32
	resolve := func(ctx context.Context, id string) (bool, error) {
		atomic.AddInt32(&attempts, 1)
		<-release
		return false, nil
	}

	runID, err := s.m.refreshCuratedCommunities(curatedDirectory("", ids...), resolve)
	s.Require().NoError(err)
	joinedRunID, err := s.m.refreshCuratedCommunities(curatedDirectory("", ids...), resolve)
	s.Require().NoError(err)
	close(release)
	s.waitFinished(signals)
	s.m.shutdownWaitGroup.Wait()

	s.Require().NotZero(runID)
	s.Require().Equal(runID, joinedRunID, "a refresh that joins a run returns the ID of that run")
	s.Require().Equal([]uint64{runID, runID}, signals.recordedRunIDs())
	s.Require().Equal(int32(3), atomic.LoadInt32(&attempts), "every entry is attempted once")
	started := 0
	for _, event := range signals.recorded() {
		if event.name == "started" {
			started++
		}
	}
	s.Require().Equal(1, started)
}

// A stop ends the fetches in flight and starts no other: the entries that
// were being fetched and the ones not reached are not counted.
func (s *MessengerCuratedRefreshSuite) TestStopCancelsRefresh() {
	signals := s.recordSignals()
	ids := make([]string, 7)
	for i := range ids {
		ids[i] = s.newCommunityID()
	}

	entered := make(chan string, len(ids))
	windDown := make(chan struct{})
	abort := make(chan struct{})
	defer close(abort)
	resolve := func(ctx context.Context, id string) (bool, error) {
		entered <- id
		if id == ids[0] {
			return false, nil
		}
		waitUnlessAborted(ctx.Done(), abort)
		waitUnlessAborted(windDown, abort)
		return false, nil
	}

	s.Require().NoError(s.refresh(curatedDirectory("", ids...), resolve))
	// The first entry has ended and two others are in flight.
	for i := 0; i < 3; i++ {
		s.receive(entered, "the entries in flight are attempted")
	}
	s.m.StopCuratedCommunitiesRefresh()
	select {
	case <-signals.finished:
		s.FailNow("the refresh finished before its fetches have returned")
	case <-time.After(100 * time.Millisecond):
	}
	close(windDown)
	s.waitFinished(signals)

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", changed: true},
		{name: "finished", notFound: 1, cancelled: true},
	}, signals.recorded())
	s.Require().Len(entered, 0, "no entry is attempted after the stop")
}

// The app tells the signals of its refresh from those of an earlier run by
// the run ID.
func (s *MessengerCuratedRefreshSuite) TestEachRunHasItsOwnRunID() {
	signals := s.recordSignals()
	resolve := func(ctx context.Context, id string) (bool, error) {
		return false, nil
	}

	firstRunID, err := s.m.refreshCuratedCommunities(curatedDirectory("", s.newCommunityID()), resolve)
	s.Require().NoError(err)
	s.waitFinished(signals)
	s.waitIdle()
	secondRunID, err := s.m.refreshCuratedCommunities(curatedDirectory("", s.newCommunityID()), resolve)
	s.Require().NoError(err)
	s.waitFinished(signals)

	s.Require().NotZero(firstRunID)
	s.Require().Greater(secondRunID, firstRunID)
	s.Require().Equal([]uint64{firstRunID, firstRunID, secondRunID, secondRunID}, signals.recordedRunIDs())
}

// The started signal carries the ID of its own run, also when another
// messenger of the process has started a run since.
func (s *MessengerCuratedRefreshSuite) TestStartedCarriesItsOwnRunID() {
	signals := s.recordSignals()
	other, err := newTestMessenger(s.T(), s.messagingEnv, testMessengerConfig{})
	s.Require().NoError(err)
	defer releaseTestMessenger(s.T(), other)

	resolve := func(ctx context.Context, id string) (bool, error) {
		return false, nil
	}
	directoryRead := make(chan struct{})
	abort := make(chan struct{})
	defer close(abort)
	blockedDirectory := func(ctx context.Context) (*communities.CuratedCommunities, error) {
		waitUnlessAborted(directoryRead, abort)
		return &communities.CuratedCommunities{}, nil
	}

	runID, err := s.m.refreshCuratedCommunities(blockedDirectory, resolve)
	s.Require().NoError(err)
	otherRunID, err := other.refreshCuratedCommunities(curatedDirectory(""), resolve)
	s.Require().NoError(err)
	other.shutdownWaitGroup.Wait()
	close(directoryRead)
	s.waitFinished(signals)

	s.Require().Greater(otherRunID, runID)
	s.Require().Equal([]uint64{runID, runID}, signals.recordedRunIDs())
}

// A refresh requested right after a stop must not be lost to the run that is
// still winding down.
func (s *MessengerCuratedRefreshSuite) TestRefreshAfterStopStartsNewRun() {
	signals := s.recordSignals()
	id := s.newCommunityID()

	entered := make(chan struct{}, 1)
	windDown := make(chan struct{})
	blocked := func(ctx context.Context, id string) (bool, error) {
		entered <- struct{}{}
		<-ctx.Done()
		<-windDown
		return false, nil
	}
	resolved := func(ctx context.Context, id string) (bool, error) {
		return false, nil
	}

	stoppedRunID, err := s.m.refreshCuratedCommunities(curatedDirectory("", id), blocked)
	s.Require().NoError(err)
	<-entered
	s.m.StopCuratedCommunitiesRefresh()
	runID, err := s.m.refreshCuratedCommunities(curatedDirectory("", id), resolved)
	s.Require().NoError(err)
	s.Require().Len(signals.recorded(), 1, "the stopped run has not finished yet")
	close(windDown)
	s.waitFinished(signals)
	s.waitFinished(signals)

	s.Require().NotZero(stoppedRunID)
	s.Require().Greater(runID, stoppedRunID)
	s.Require().Equal([]uint64{stoppedRunID, stoppedRunID, runID, runID}, signals.recordedRunIDs(),
		"the late signals of the stopped run carry its own ID")

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", changed: true},
		{name: "finished", cancelled: true},
		{name: "started"},
		{name: "finished", notFound: 1},
	}, signals.recorded())
}

func (s *MessengerCuratedRefreshSuite) TestRefreshStopsWithMessenger() {
	signals := s.recordSignals()

	entered := make(chan struct{}, 1)
	resolve := func(ctx context.Context, id string) (bool, error) {
		entered <- struct{}{}
		<-ctx.Done()
		return false, nil
	}

	s.Require().NoError(s.refresh(curatedDirectory("", s.newCommunityID()), resolve))
	<-entered
	s.m.cancel()
	s.m.shutdownWaitGroup.Wait()

	events := signals.recorded()
	s.Require().Equal(curatedRefreshEvent{name: "finished", cancelled: true}, events[len(events)-1])
}

// blockedEthClient never answers a contract call: the call ends with its
// context, or when released.
type blockedEthClient struct {
	ethclient.EthClientInterface

	calling chan struct{}
	release chan struct{}
}

func (c *blockedEthClient) EthClient(uint64) (ethclient.EthClientInterface, error) {
	return c, nil
}

func (c *blockedEthClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	select {
	case c.calling <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.release:
		return nil, errors.New("contract call released by the test")
	}
}

func (s *MessengerCuratedRefreshSuite) TestStopDoesNotWaitForDirectoryContract() {
	signals := s.recordSignals()
	client := &blockedEthClient{calling: make(chan struct{}, 1), release: make(chan struct{})}
	defer close(client.release)
	s.m.contractMaker = contracts.NewContractMaker(client)

	s.Require().NoError(s.refreshFromContract())
	select {
	case <-client.calling:
	case <-time.After(5 * time.Second):
		s.FailNow("the directory contract was not called")
	}
	s.m.StopCuratedCommunitiesRefresh()
	s.waitIdle()

	s.Require().Empty(signals.recorded(), "a run stopped before it has its directory emits no signal")
}

// A directory that arrives after the messenger began to shut down is neither
// stored nor resolved.
func (s *MessengerCuratedRefreshSuite) TestShutdownDropsDirectoryReadInFlight() {
	signals := s.recordSignals()
	id := s.newCommunityID()

	reading := make(chan struct{})
	directory := func(ctx context.Context) (*communities.CuratedCommunities, error) {
		close(reading)
		<-ctx.Done()
		return &communities.CuratedCommunities{ContractCommunities: []string{id}}, nil
	}
	resolve := func(ctx context.Context, id string) (bool, error) {
		s.Fail("no entry is resolved after the shutdown")
		return false, nil
	}

	s.Require().NoError(s.refresh(directory, resolve))
	<-reading
	s.m.cancel()
	s.waitIdle()

	s.Require().Empty(signals.recorded())
	response, err := s.m.CuratedCommunities()
	s.Require().NoError(err)
	s.Require().Empty(response.ContractCommunities)
}

func (s *MessengerCuratedRefreshSuite) TestRefreshDoesNothingOnceMessengerShutsDown() {
	signals := s.recordSignals()
	s.m.cancel()

	directoryRead := false
	directory := func(context.Context) (*communities.CuratedCommunities, error) {
		directoryRead = true
		return &communities.CuratedCommunities{}, nil
	}
	resolve := func(ctx context.Context, id string) (bool, error) {
		return false, nil
	}

	runID, err := s.m.refreshCuratedCommunities(directory, resolve)
	s.Require().NoError(err)
	s.m.shutdownWaitGroup.Wait()

	s.Require().Zero(runID, "no run was started or joined")
	s.Require().False(directoryRead)
	s.Require().Empty(signals.recorded())
}

// playStoreNodePages answers every store node request with pages, which gets
// the request's per-page check.
func (s *MessengerCuratedRefreshSuite) playStoreNodePages(pages func(pageCheck func(int) (bool, uint64))) {
	s.messagingEnv.SetProcessMailserverBatchHook(func(ctx context.Context, batch wakutypes.MailserverBatch, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error {
		pages(shouldProcessNextPage)
		return nil
	})
	s.T().Cleanup(func() {
		s.messagingEnv.SetProcessMailserverBatchHook(nil)
	})
}

// With no contract at hand the refresh resolves the directory stored earlier.
func (s *MessengerCuratedRefreshSuite) TestRefreshFetchesUnknownEntryFromStoreNodes() {
	signals := s.recordSignals()
	batches := s.countStoreNodeBatches()
	unknown := s.newCommunityID()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknown},
	}))
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {})

	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	s.Require().Equal(int32(1), atomic.LoadInt32(batches), "one attempt per entry that is not found")
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", notFound: 1},
	}, signals.recorded())
}

// An entry that store nodes do not have is not fetched again for a while:
// later refreshes neither request it nor count it.
func (s *MessengerCuratedRefreshSuite) TestSecondRefreshDoesNotFetchNotFoundEntryAgain() {
	signals := s.recordSignals()
	batches := s.countStoreNodeBatches()
	unknown := s.newCommunityID()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknown},
	}))
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {})

	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)
	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	s.Require().Equal(int32(1), atomic.LoadInt32(batches), "one store node request per not found entry and messenger")
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", notFound: 1},
		{name: "started", directoryFailed: true},
		{name: "finished"},
	}, signals.recorded())
}

// An entry whose store node request failed has not had its attempt.
func (s *MessengerCuratedRefreshSuite) TestRefreshFetchesEntryAgainAfterFailedStoreNodeRequest() {
	signals := s.recordSignals()
	batches := s.countStoreNodeBatches()
	unknown := s.newCommunityID()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknown},
	}))
	s.messagingEnv.SetProcessMailserverBatchHook(func(ctx context.Context, batch wakutypes.MailserverBatch, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error {
		return errors.New("store node unreachable")
	})
	s.T().Cleanup(func() {
		s.messagingEnv.SetProcessMailserverBatchHook(nil)
	})

	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)
	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	s.Require().Equal(int32(2), atomic.LoadInt32(batches))
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", failed: 1},
		{name: "started", directoryFailed: true},
		{name: "finished", failed: 1},
	}, signals.recorded())
}

// An entry whose fetch was interrupted by a stop has not had its attempt.
func (s *MessengerCuratedRefreshSuite) TestRefreshFetchesEntryInterruptedByStopAgain() {
	signals := s.recordSignals()
	batches := s.countStoreNodeBatches()
	unknown := s.newCommunityID()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknown},
	}))

	requested := make(chan struct{}, 2)
	storeNodeAnswer := make(chan struct{})
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {
		requested <- struct{}{}
		<-storeNodeAnswer
	})

	s.Require().NoError(s.refreshFromContract())
	<-requested
	s.m.StopCuratedCommunitiesRefresh()
	s.waitFinished(signals)
	// Let the interrupted store node request end, so that the next refresh
	// issues its own.
	close(storeNodeAnswer)
	s.m.shutdownWaitGroup.Wait()

	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	s.Require().Equal(int32(2), atomic.LoadInt32(batches))
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", cancelled: true},
		{name: "started", directoryFailed: true},
		{name: "finished", notFound: 1},
	}, signals.recorded())
}

// Store node requests for different entries run next to each other.
func (s *MessengerCuratedRefreshSuite) TestRefreshRequestsTwoEntriesFromStoreNodesAtATime() {
	signals := s.recordSignals()
	batches := s.countStoreNodeBatches()
	ids := []string{s.newCommunityID(), s.newCommunityID(), s.newCommunityID(), s.newCommunityID()}

	requested := make(chan string, len(ids))
	storeNodeAnswer := make(chan struct{})
	abort := make(chan struct{})
	defer close(abort)
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {
		requested <- ""
		waitUnlessAborted(storeNodeAnswer, abort)
	})

	s.Require().NoError(s.refresh(curatedDirectory("", ids...), s.m.resolveCuratedCommunity))
	for i := 0; i < 2; i++ {
		s.receive(requested, "two store node requests are in flight")
	}
	select {
	case <-requested:
		s.FailNow("a third store node request is issued while two are in flight")
	case <-time.After(100 * time.Millisecond):
	}
	close(storeNodeAnswer)
	s.waitFinished(signals)

	s.Require().Equal(int32(4), atomic.LoadInt32(batches))
	events := signals.recorded()
	s.Require().Equal(curatedRefreshEvent{name: "finished", notFound: 4}, events[len(events)-1])
}

func (s *MessengerCuratedRefreshSuite) TestStopDoesNotWaitForStoreNodeRequest() {
	signals := s.recordSignals()
	unknown := s.newCommunityID()

	requested := make(chan struct{}, 1)
	storeNodeAnswer := make(chan struct{})
	defer s.m.shutdownWaitGroup.Wait()
	defer close(storeNodeAnswer)
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {
		requested <- struct{}{}
		<-storeNodeAnswer
	})

	s.Require().NoError(s.refresh(curatedDirectory("", unknown), s.m.resolveCuratedCommunity))
	<-requested
	s.m.StopCuratedCommunitiesRefresh()
	s.waitFinished(signals)

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", changed: true},
		{name: "finished", cancelled: true},
	}, signals.recorded())
}

func (s *MessengerCuratedRefreshSuite) TestRefreshKeepsStoredDirectoryWhenContractFails() {
	signals := s.recordSignals()
	known := s.storeCommunity()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities:         []string{known},
		ContractFeaturedCommunities: []string{known},
	}))

	failingDirectory := func(context.Context) (*communities.CuratedCommunities, error) {
		return nil, errors.New("contract unreachable")
	}
	s.Require().NoError(s.refresh(failingDirectory, s.m.resolveCuratedCommunity))
	s.waitFinished(signals)

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished"},
	}, signals.recorded())

	response, err := s.m.CuratedCommunities()
	s.Require().NoError(err)
	s.Require().Equal([]string{known}, response.ContractCommunities)
	s.Require().Equal([]string{known}, response.ContractFeaturedCommunities)
}

// On a metered connection with sync on mobile network off no store node is
// asked: the entry has not had its attempt.
func (s *MessengerCuratedRefreshSuite) TestRefreshFetchesEntryAgainWhenStoreNodesWereNotAsked() {
	signals := s.recordSignals()
	unknown := s.newCommunityID()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknown},
	}))
	var asked int32
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {
		atomic.AddInt32(&asked, 1)
	})
	s.Require().NoError(s.m.settings.SetSyncingOnMobileNetwork(false))

	s.m.ConnectionChanged(connection.State{Type: connection.NewType(connection.Cellular)})
	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)
	s.Require().Zero(atomic.LoadInt32(&asked), "store nodes are not asked on a metered connection")

	s.m.ConnectionChanged(connection.State{Type: connection.NewType(connection.Wifi)})
	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	s.Require().Equal(int32(1), atomic.LoadInt32(&asked))
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", failed: 1},
		{name: "started", directoryFailed: true},
		{name: "finished", notFound: 1},
	}, signals.recorded())
}

// An entry whose stored state cannot be read is not fetched and counts as
// failed, not as one that store nodes do not have.
func (s *MessengerCuratedRefreshSuite) TestRefreshSkipsEntryWhoseStoredStateIsUnreadable() {
	signals := s.recordSignals()
	batches := s.countStoreNodeBatches()
	unknown := s.newCommunityID()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknown},
	}))
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {})

	_, err := s.m.database.Exec("ALTER TABLE communities_communities RENAME TO communities_communities_gone")
	s.Require().NoError(err)
	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)
	s.Require().Zero(atomic.LoadInt32(batches))

	_, err = s.m.database.Exec("ALTER TABLE communities_communities_gone RENAME TO communities_communities")
	s.Require().NoError(err)
	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	s.Require().Equal(int32(1), atomic.LoadInt32(batches))
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", failed: 1},
		{name: "started", directoryFailed: true},
		{name: "finished", notFound: 1},
	}, signals.recorded())
}

// An entry recorded as not found is skipped before its stored state is read.
func (s *MessengerCuratedRefreshSuite) TestRefreshDoesNotCountNotFoundEntryWhoseStoredStateIsUnreadable() {
	signals := s.recordSignals()
	batches := s.countStoreNodeBatches()
	unknown := s.newCommunityID()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknown},
	}))
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {})

	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	_, err := s.m.database.Exec("ALTER TABLE communities_communities RENAME TO communities_communities_gone")
	s.Require().NoError(err)
	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)
	_, err = s.m.database.Exec("ALTER TABLE communities_communities_gone RENAME TO communities_communities")
	s.Require().NoError(err)

	s.Require().Equal(int32(1), atomic.LoadInt32(batches))
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", notFound: 1},
		{name: "started", directoryFailed: true},
		{name: "finished"},
	}, signals.recorded())
}

// An entry whose ID is malformed can never be fetched: it is not found, once.
func (s *MessengerCuratedRefreshSuite) TestRefreshCountsMalformedEntryAsNotFoundOnce() {
	signals := s.recordSignals()
	malformed := "0xnot-a-community-id"
	resolve := func(ctx context.Context, id string) (bool, error) {
		s.Fail("a malformed entry is not fetched")
		return false, nil
	}

	for range 2 {
		s.Require().NoError(s.refresh(curatedDirectory("", malformed), resolve))
		s.waitFinished(signals)
		s.waitIdle()
	}

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", changed: true},
		{name: "finished", notFound: 1},
		{name: "started"},
		{name: "finished"},
	}, signals.recorded())
}

// A fetch that store nodes answered, after which what is stored cannot be
// read, has failed: the entry is not recorded as not found.
func (s *MessengerCuratedRefreshSuite) TestRefreshFetchesEntryAgainWhenStoredStateIsUnreadableAfterFetch() {
	signals := s.recordSignals()
	batches := s.countStoreNodeBatches()
	unknown := s.newCommunityID()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{unknown},
	}))
	var answered int32
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {
		if atomic.AddInt32(&answered, 1) == 1 {
			_, err := s.m.database.Exec("ALTER TABLE communities_communities RENAME TO communities_communities_gone")
			s.NoError(err)
		}
	})

	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)
	_, err := s.m.database.Exec("ALTER TABLE communities_communities_gone RENAME TO communities_communities")
	s.Require().NoError(err)
	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	s.Require().Equal(int32(2), atomic.LoadInt32(batches))
	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", failed: 1},
		{name: "started", directoryFailed: true},
		{name: "finished", notFound: 1},
	}, signals.recorded())
}

// The app fills the card of a fetched entry from the community found signal.
func (s *MessengerCuratedRefreshSuite) TestRefreshReportsFetchedCommunityAsFound() {
	signals := s.recordSignals()
	signals.communityFoundChan = make(chan *communities.Community, 4)
	community := s.newReceivedCommunity()
	id := community.IDString()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{id},
	}))
	// The store node page brings the community.
	s.playStoreNodePages(func(pageCheck func(int) (bool, uint64)) {
		s.NoError(s.m.communitiesManager.GetPersistence().SaveCommunity(community))
		next, _ := pageCheck(1)
		s.False(next)
	})

	s.Require().NoError(s.refreshFromContract())
	s.waitFinished(signals)

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished", resolved: 1},
	}, signals.recorded())
	select {
	case found := <-signals.communityFoundChan:
		s.Require().Equal(id, found.IDString())
	case <-time.After(5 * time.Second):
		s.FailNow("the fetched community was not reported as found")
	}
}

// A directory that cannot be stored is not the one the app reads on started:
// the refresh goes on with the directory stored earlier.
func (s *MessengerCuratedRefreshSuite) TestRefreshKeepsStoredDirectoryWhenStoringFails() {
	signals := s.recordSignals()
	known := s.storeCommunity()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities:         []string{known},
		ContractFeaturedCommunities: []string{known},
	}))

	s.failCuratedDirectoryStore()
	unknown := s.newCommunityID()
	resolve := func(ctx context.Context, id string) (bool, error) {
		s.Fail("an entry of a directory that was not stored is not resolved")
		return false, nil
	}
	s.Require().NoError(s.refresh(curatedDirectory("", unknown), resolve))
	s.waitFinished(signals)

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished"},
	}, signals.recorded())

	response, err := s.m.CuratedCommunities()
	s.Require().NoError(err)
	s.Require().Equal([]string{known}, response.ContractCommunities)
	s.Require().Equal([]string{known}, response.ContractFeaturedCommunities)
}

// failCuratedDirectoryStore makes every later SetCuratedCommunities fail.
func (s *MessengerCuratedRefreshSuite) failCuratedDirectoryStore() {
	_, err := s.m.database.Exec(`CREATE TRIGGER fail_curated_store BEFORE INSERT ON curated_communities
		BEGIN SELECT RAISE(ABORT, 'store refused by the test'); END`)
	s.Require().NoError(err)
}

// A contract that lists an entry twice still has its directory stored, the
// entry kept once and fetched once.
func (s *MessengerCuratedRefreshSuite) TestRefreshStoresEntryListedTwiceOnce() {
	signals := s.recordSignals()
	first, second := s.newCommunityID(), s.newCommunityID()

	var mu sync.Mutex
	var attempted []string
	resolve := func(ctx context.Context, id string) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		attempted = append(attempted, id)
		return false, nil
	}
	directory := func(context.Context) (*communities.CuratedCommunities, error) {
		return &communities.CuratedCommunities{
			ContractCommunities:         []string{first, second, first, second},
			ContractFeaturedCommunities: []string{second, second},
		}, nil
	}
	s.Require().NoError(s.refresh(directory, resolve))
	s.waitFinished(signals)

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", changed: true},
		{name: "finished", notFound: 2},
	}, signals.recorded())
	s.Require().ElementsMatch([]string{first, second}, attempted)

	response, err := s.m.CuratedCommunities()
	s.Require().NoError(err)
	s.Require().ElementsMatch([]string{first, second}, response.ContractCommunities)
	s.Require().Equal([]string{second}, response.ContractFeaturedCommunities)
}

// A contract call that never ends does not hold the refresh: it goes on with
// the directory stored earlier.
func (s *MessengerCuratedRefreshSuite) TestRefreshGivesUpOnDirectoryReadAfterTimeout() {
	signals := s.recordSignals()
	known := s.storeCommunity()
	s.Require().NoError(s.m.communitiesManager.SetCuratedCommunities(&communities.CuratedCommunities{
		ContractCommunities: []string{known},
	}))

	defer func(timeout time.Duration) { curatedDirectoryReadTimeout = timeout }(curatedDirectoryReadTimeout)
	curatedDirectoryReadTimeout = 50 * time.Millisecond
	hung := func(ctx context.Context) (*communities.CuratedCommunities, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	resolve := func(ctx context.Context, id string) (bool, error) {
		return false, nil
	}
	s.Require().NoError(s.refresh(hung, resolve))
	s.waitFinished(signals)

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished"},
	}, signals.recorded())
}

// An entry that store nodes did not have is fetched again once
// curatedNotFoundRetryAfter has passed.
func (s *MessengerCuratedRefreshSuite) TestRefreshFetchesNotFoundEntryAgainLater() {
	signals := s.recordSignals()
	unknown := s.newCommunityID()
	now := time.Now()
	s.m.curatedCommunitiesRefresh.now = func() time.Time { return now }

	var attempts int32
	resolve := func(ctx context.Context, id string) (bool, error) {
		atomic.AddInt32(&attempts, 1)
		return false, nil
	}
	for _, elapsed := range []time.Duration{0, curatedNotFoundRetryAfter - time.Second, curatedNotFoundRetryAfter} {
		now = now.Add(elapsed)
		s.Require().NoError(s.refresh(curatedDirectory("", unknown), resolve))
		s.waitFinished(signals)
		s.waitIdle()
	}

	s.Require().Equal(int32(2), atomic.LoadInt32(&attempts))
	events := signals.recorded()
	s.Require().Equal(curatedRefreshEvent{name: "finished", notFound: 1}, events[1])
	s.Require().Equal(curatedRefreshEvent{name: "finished"}, events[3])
	s.Require().Equal(curatedRefreshEvent{name: "finished", notFound: 1}, events[5])
}

// The app reads the directory again when the refresh tells that it changed.
func (s *MessengerCuratedRefreshSuite) TestRefreshTellsWhetherDirectoryChanged() {
	signals := s.recordSignals()
	first, second := s.newCommunityID(), s.newCommunityID()
	resolve := func(ctx context.Context, id string) (bool, error) {
		return false, nil
	}
	failingDirectory := func(context.Context) (*communities.CuratedCommunities, error) {
		return nil, errors.New("contract unreachable")
	}

	var changed, directoryFailed []bool
	for _, directory := range []curatedDirectoryReader{
		curatedDirectory(first, first),
		curatedDirectory(first, first),
		curatedDirectory(first, first, second),
		curatedDirectory(second, first, second),
		curatedDirectory(second, second, first),
		failingDirectory,
		curatedDirectory(second, second),
		// Listed twice: stored once.
		curatedDirectory(second, second, second),
		curatedDirectory(first, first),
	} {
		if len(signals.recorded()) == 16 {
			s.failCuratedDirectoryStore()
		}
		s.Require().NoError(s.refresh(directory, resolve))
		s.waitFinished(signals)
		s.waitIdle()
	}
	for _, event := range signals.recorded() {
		if event.name == "started" {
			changed = append(changed, event.changed)
			directoryFailed = append(directoryFailed, event.directoryFailed)
		}
	}

	s.Require().Equal([]bool{
		true,  // the first directory
		false, // the same directory
		true,  // an entry more
		true,  // another featured entry
		false, // the same entries in another order
		false, // the contract cannot be read
		true,  // an entry less
		false, // the same entry, listed twice
		false, // the directory cannot be stored
	}, changed)
	s.Require().Equal([]bool{false, false, false, false, false, true, false, false, true}, directoryFailed)
}

// With neither the contract nor the stored directory at hand the refresh
// still reports its failure.
func (s *MessengerCuratedRefreshSuite) TestRefreshReportsFailureWhenNoDirectoryCanBeRead() {
	signals := s.recordSignals()
	_, err := s.m.database.Exec("ALTER TABLE curated_communities RENAME TO curated_communities_gone")
	s.Require().NoError(err)
	defer func() {
		_, err := s.m.database.Exec("ALTER TABLE curated_communities_gone RENAME TO curated_communities")
		s.Require().NoError(err)
	}()

	failingDirectory := func(context.Context) (*communities.CuratedCommunities, error) {
		return nil, errors.New("contract unreachable")
	}
	resolve := func(ctx context.Context, id string) (bool, error) {
		s.Fail("there is no entry to resolve")
		return false, nil
	}
	for _, directory := range []curatedDirectoryReader{failingDirectory, curatedDirectory("", s.newCommunityID())} {
		s.Require().NoError(s.refresh(directory, resolve))
		s.waitFinished(signals)
		s.waitIdle()
	}

	s.Require().Equal([]curatedRefreshEvent{
		{name: "started", directoryFailed: true},
		{name: "finished"},
		{name: "started", directoryFailed: true},
		{name: "finished"},
	}, signals.recorded())
}
