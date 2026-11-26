package protocol

import (
	"context"
	"errors"
	"maps"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/pkg/services/wallet/common"
)

func (m *Messenger) getCuratedCommunitiesFromContract(ctx context.Context) (*communities.CuratedCommunities, error) {
	if m.contractMaker == nil {
		return nil, errors.New("contract maker not initialized")
	}

	testNetworksEnabled, err := m.settings.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	chainID := common.OptimismMainnet
	if testNetworksEnabled {
		chainID = common.OptimismSepolia
	}

	directory, err := m.contractMaker.NewDirectory(chainID)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}

	contractCommunities, err := directory.GetCommunities(callOpts)
	if err != nil {
		return nil, err
	}
	var contractCommunityIDs []string
	for _, c := range contractCommunities {
		contractCommunityIDs = append(contractCommunityIDs, types.HexBytes(c).String())
	}

	featuredContractCommunities, err := directory.GetFeaturedCommunities(callOpts)
	if err != nil {
		return nil, err
	}
	var contractFeaturedCommunityIDs []string
	for _, c := range featuredContractCommunities {
		contractFeaturedCommunityIDs = append(contractFeaturedCommunityIDs, types.HexBytes(c).String())
	}

	return &communities.CuratedCommunities{
		ContractCommunities:         contractCommunityIDs,
		ContractFeaturedCommunities: contractFeaturedCommunityIDs,
	}, nil
}

// CuratedCommunities returns the stored curated directory with the
// descriptions stored locally. It never queries the contract or store nodes;
// RefreshCuratedCommunities does.
func (m *Messenger) CuratedCommunities() (*communities.KnownCommunitiesResponse, error) {
	curatedCommunities, err := m.communitiesManager.GetCuratedCommunities()
	if err != nil {
		return nil, err
	}

	response, err := m.communitiesManager.GetStoredDescriptionForCommunities(curatedCommunities.ContractCommunities)
	if err != nil {
		return nil, err
	}
	response.ContractFeaturedCommunities = curatedCommunities.ContractFeaturedCommunities

	return response, nil
}

var errStoreNodesNotAllowed = errors.New("store nodes may not be queried on this connection")

// curatedDirectoryReader reads the curated directory from its contract.
type curatedDirectoryReader func(ctx context.Context) (*communities.CuratedCommunities, error)

// curatedCommunityResolver resolves one curated directory entry against store
// nodes and reports whether its description is now stored locally. The error
// tells that the attempt does not show whether store nodes have the entry.
type curatedCommunityResolver func(ctx context.Context, communityID string) (stored bool, err error)

// maxConcurrentCuratedFetches bounds the entries a refresh fetches at once.
const maxConcurrentCuratedFetches = 2

// curatedDirectoryReadTimeout bounds the read of the directory contract.
var curatedDirectoryReadTimeout = 30 * time.Second

// curatedNotFoundRetryAfter is how long an entry that store nodes did not
// have is skipped before it is fetched again.
const curatedNotFoundRetryAfter = time.Hour

// lastCuratedRefreshRunID numbers the runs of this process from 1.
var lastCuratedRefreshRunID atomic.Uint64

// curatedCommunitiesRefresh keeps a single curated refresh in flight.
type curatedCommunitiesRefresh struct {
	mu sync.Mutex
	// runID identifies the latest run.
	runID uint64
	// cancel is set while a run is in flight.
	cancel  context.CancelFunc
	stopped bool
	// done is closed when the latest run has emitted its finished signal.
	done chan struct{}
	// notFound holds when each entry that store nodes were asked for and did
	// not have was marked. It is skipped for curatedNotFoundRetryAfter.
	notFound map[string]time.Time
	// now is time.Now unless a test sets it.
	now func() time.Time
}

func (r *curatedCommunitiesRefresh) clock() time.Time {
	if r.now != nil {
		return r.now()
	}
	return time.Now()
}

func (r *curatedCommunitiesRefresh) isNotFound(communityID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	markedAt, notFound := r.notFound[communityID]
	if !notFound {
		return false
	}
	if r.clock().Sub(markedAt) >= curatedNotFoundRetryAfter {
		delete(r.notFound, communityID)
		return false
	}
	return true
}

func (r *curatedCommunitiesRefresh) markNotFound(communityID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.notFound == nil {
		r.notFound = make(map[string]time.Time)
	}
	r.notFound[communityID] = r.clock()
}

// begin starts a run, unless one is in flight and was not stopped: runID is
// then the ID of that run. Once parent is cancelled it starts and joins
// nothing and runID is 0. A started run is added to wg under the same lock
// as shutdown takes. The new run must wait for previous before it emits
// anything.
func (r *curatedCommunitiesRefresh) begin(parent context.Context, wg *sync.WaitGroup) (ctx context.Context, previous <-chan struct{}, done chan struct{}, runID uint64, started bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if parent.Err() != nil {
		return nil, nil, nil, 0, false
	}
	if r.cancel != nil && !r.stopped {
		return nil, nil, nil, r.runID, false
	}

	previous = r.done
	ctx, r.cancel = context.WithCancel(parent)
	r.stopped = false
	r.done = make(chan struct{})
	r.runID = lastCuratedRefreshRunID.Add(1)
	wg.Add(1)
	return ctx, previous, r.done, r.runID, true
}

// shutdown returns once no begin is adding a run to the wait group. Called
// after the parent context is cancelled, no run is added after it returns.
func (r *curatedCommunitiesRefresh) shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()
}

// release lets the next refresh start a run of its own.
func (r *curatedCommunitiesRefresh) release(done chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.done == done && r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
}

func (r *curatedCommunitiesRefresh) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
		r.stopped = true
	}
}

// RefreshCuratedCommunities reads the curated directory from its contract,
// stores it and resolves the entries with no stored description against store
// nodes, maxConcurrentCuratedFetches at a time. Signals: started, once the
// directory is stored (or, when the contract cannot be read or its directory
// stored, once the directory stored earlier is taken), which tells whether the
// directory differs from the one stored before and whether updating it failed;
// then finished, which counts the entries the refresh had to fetch as
// resolved, not found or failed. A refresh cancelled before it has its
// directory emits no signal. Entries already stored are read with
// CuratedCommunities. An entry that store nodes were asked for and did not
// have is not found: later refreshes skip it without counting it until
// curatedNotFoundRetryAfter has passed. An entry whose ID is malformed is also
// not found, and never fetched. An entry whose fetch returned an error, also because
// store nodes may not be asked, is failed: the next refresh fetches it again,
// as it does an entry whose fetch was interrupted, which is not counted. A
// refresh requested while one is in flight joins it. Every run has an ID,
// unique in this process and growing from 1, that its signals carry. The ID of
// the run that was started or joined is returned; 0 when the messenger is
// shutting down and nothing was started.
func (m *Messenger) RefreshCuratedCommunities() (uint64, error) {
	return m.refreshCuratedCommunities(m.getCuratedCommunitiesFromContract, m.resolveCuratedCommunity)
}

// StopCuratedCommunitiesRefresh cancels the refresh in flight, if any.
func (m *Messenger) StopCuratedCommunitiesRefresh() {
	m.curatedCommunitiesRefresh.stop()
}

func (m *Messenger) refreshCuratedCommunities(directory curatedDirectoryReader, resolve curatedCommunityResolver) (uint64, error) {
	ctx, previous, done, runID, started := m.curatedCommunitiesRefresh.begin(m.ctx, &m.shutdownWaitGroup)
	if !started {
		return runID, nil
	}

	go func() {
		defer panics.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
		defer close(done)
		defer m.curatedCommunitiesRefresh.release(done)

		if previous != nil {
			<-previous
		}

		communityIDs, changed, directoryFailed := m.updateCuratedDirectory(ctx, directory)
		// A run cancelled before it has its directory emits no signal.
		if ctx.Err() != nil {
			return
		}

		signals := m.config.messengerSignalsHandler
		if signals != nil {
			signals.CuratedCommunitiesRefreshStarted(runID, changed, directoryFailed)
		}

		resolved, notFound, failed, cancelled := m.resolveCuratedCommunities(ctx, communityIDs, resolve)

		m.curatedCommunitiesRefresh.release(done)
		if signals != nil {
			signals.CuratedCommunitiesRefreshFinished(runID, resolved, notFound, failed, cancelled)
		}
	}()

	return runID, nil
}

// updateCuratedDirectory stores the directory read from the contract and
// returns its entries, and whether it differs from the directory stored
// before. An entry listed more than once is kept once. When the contract
// cannot be read within curatedDirectoryReadTimeout or the directory cannot
// be stored it has failed and returns the entries stored earlier, unchanged,
// or none when they cannot be read either. Once ctx is cancelled it stores
// and returns nothing.
func (m *Messenger) updateCuratedDirectory(ctx context.Context, directory curatedDirectoryReader) (communityIDs []string, changed, failed bool) {
	readCtx, cancelRead := context.WithTimeout(ctx, curatedDirectoryReadTimeout)
	curatedCommunities, err := directory(readCtx)
	cancelRead()
	if ctx.Err() != nil {
		return nil, false, false
	}
	if err == nil {
		curatedCommunities = &communities.CuratedCommunities{
			ContractCommunities:         uniqueCommunityIDs(curatedCommunities.ContractCommunities),
			ContractFeaturedCommunities: uniqueCommunityIDs(curatedCommunities.ContractFeaturedCommunities),
		}
	}

	stored, storedErr := m.communitiesManager.GetCuratedCommunities()
	if storedErr != nil {
		m.logger.Error("failed to read stored curated communities", zap.Error(storedErr))
	}

	if err != nil {
		m.logger.Error("failed to read curated communities directory", zap.Error(err))
	} else if err = m.communitiesManager.SetCuratedCommunities(curatedCommunities); err != nil {
		m.logger.Error("failed to store curated communities", zap.Error(err))
	} else {
		return curatedCommunities.ContractCommunities, storedErr != nil || !sameCuratedDirectory(stored, curatedCommunities), false
	}

	if storedErr != nil {
		return nil, false, true
	}
	return stored.ContractCommunities, false, true
}

// uniqueCommunityIDs keeps the first of each ID, in order.
func uniqueCommunityIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// sameCuratedDirectory tells whether both directories list the same entries
// and feature the same ones, in whatever order.
func sameCuratedDirectory(a, b *communities.CuratedCommunities) bool {
	entries := func(directory *communities.CuratedCommunities) map[string]bool {
		featured := make(map[string]bool, len(directory.ContractCommunities))
		for _, communityID := range directory.ContractCommunities {
			featured[communityID] = false
		}
		for _, communityID := range directory.ContractFeaturedCommunities {
			if _, listed := featured[communityID]; listed {
				featured[communityID] = true
			}
		}
		return featured
	}
	return maps.Equal(entries(a), entries(b))
}

// resolveCuratedCommunities fetches the entries with no stored description,
// maxConcurrentCuratedFetches at a time, and counts each of them once.
// Resolved: the fetch stored it. Not found: store nodes were asked without
// error and did not have it, or its ID is malformed; it is neither fetched
// nor counted again for curatedNotFoundRetryAfter. Failed: the fetch, or the check of
// what is stored, returned an error; the next refresh fetches the entry
// again. A fetch interrupted by ctx is not counted, and no fetch starts once
// ctx is cancelled. Entries already stored are not counted:
// CuratedCommunities reads them. Interrupted tells whether ctx left an entry
// unhandled. It returns when every fetch has returned.
func (m *Messenger) resolveCuratedCommunities(ctx context.Context, communityIDs []string, resolve curatedCommunityResolver) (resolved, notFound, failed int, interrupted bool) {
	var (
		fetches sync.WaitGroup
		mu      sync.Mutex
		// skipped is set by the loop alone, interrupted under mu.
		skipped bool
	)
	slots := make(chan struct{}, maxConcurrentCuratedFetches)

	fetch := func(communityID string) {
		defer panics.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
		defer fetches.Done()
		defer func() { <-slots }()

		stored, err := resolve(ctx, communityID)

		mu.Lock()
		defer mu.Unlock()

		if ctx.Err() != nil {
			interrupted = true
			return
		}

		switch {
		case stored:
			resolved++
		case err != nil:
			failed++
			m.logger.Error("failed to fetch curated community",
				zap.String("communityID", communityID),
				zap.Error(err))
		default:
			notFound++
			m.curatedCommunitiesRefresh.markNotFound(communityID)
			m.logger.Warn("curated community not found", zap.String("communityID", communityID))
		}
	}

loop:
	for _, communityID := range communityIDs {
		if ctx.Err() != nil {
			skipped = true
			break
		}
		if m.curatedCommunitiesRefresh.isNotFound(communityID) {
			continue
		}
		id, err := types.DecodeHex(communityID)
		if err != nil {
			mu.Lock()
			notFound++
			mu.Unlock()
			m.curatedCommunitiesRefresh.markNotFound(communityID)
			m.logger.Warn("malformed curated community ID",
				zap.String("communityID", communityID),
				zap.Error(err))
			continue
		}
		stored, err := m.communitiesManager.CommunityExists(id)
		if err != nil {
			mu.Lock()
			failed++
			mu.Unlock()
			m.logger.Error("failed to check stored curated community",
				zap.String("communityID", communityID),
				zap.Error(err))
			continue
		}
		if stored {
			continue
		}

		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			skipped = true
			break loop
		}
		if ctx.Err() != nil {
			<-slots
			skipped = true
			break
		}

		fetches.Add(1)
		m.shutdownWaitGroup.Add(1)
		go fetch(communityID)
	}

	fetches.Wait()

	return resolved, notFound, failed, interrupted || skipped
}

func (m *Messenger) isCommunityStored(communityID string) (bool, error) {
	id, err := types.DecodeHex(communityID)
	if err != nil {
		return false, err
	}
	return m.communitiesManager.CommunityExists(id)
}

// resolveCuratedCommunity fetches the community from store nodes. It returns
// an error when store nodes may not be asked, as on a metered connection
// with sync on mobile network off. It returns as soon as ctx is cancelled;
// the store node request then ends on its own.
func (m *Messenger) resolveCuratedCommunity(ctx context.Context, communityID string) (bool, error) {
	canSync, err := m.canSyncWithStoreNodes()
	if err != nil {
		return false, err
	}
	if !canSync {
		return false, errStoreNodesNotAllowed
	}

	fetched := make(chan error, 1)

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer panics.LogOnPanic()
		defer m.shutdownWaitGroup.Done()

		cfg := buildStoreNodeRequestConfig([]StoreNodeRequestOption{
			WithRequireNewerCommunityDescription(false),
		})
		result, err := m.storeNodeRequestsManager.requestCommunity(ctx, communityID, cfg)
		if err == nil {
			err = result.err
		}
		fetched <- err
	}()

	select {
	case err := <-fetched:
		stored, storedErr := m.isCommunityStored(communityID)
		if err == nil {
			err = storedErr
		}
		return stored, err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
