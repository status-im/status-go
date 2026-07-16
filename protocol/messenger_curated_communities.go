package protocol

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/services/wallet/common"
)

func (m *Messenger) getCuratedCommunitiesFromContract() (*communities.CuratedCommunities, error) {
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

	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}

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

// CuratedCommunities returns the curated directory ids together with any
// description already stored locally. It is a pure read: it never queries store
// nodes. Use RefreshCuratedCommunities to resolve missing descriptions.
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

// curatedCommunityResolver resolves a single curated directory entry against
// store nodes and reports whether a description is now stored locally.
type curatedCommunityResolver func(ctx context.Context, communityID string) bool

// curatedCommunitiesRefresh tracks the single in-flight curated refresh so a
// second refresh joins the running one instead of starting a new walk.
type curatedCommunitiesRefresh struct {
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// begin starts a new refresh derived from parent, unless one is already running.
func (r *curatedCommunitiesRefresh) begin(parent context.Context) (context.Context, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil, false
	}

	ctx, cancel := context.WithCancel(parent)
	r.running = true
	r.cancel = cancel
	return ctx, true
}

func (r *curatedCommunitiesRefresh) end() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.running = false
}

func (r *curatedCommunitiesRefresh) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
}

// RefreshCuratedCommunities reads the curated directory and resolves every entry
// (known and unknown) against store nodes. Progress is reported via signals. A
// refresh started while one is already running joins the in-progress run.
func (m *Messenger) RefreshCuratedCommunities() error {
	return m.refreshCuratedCommunities(m.getCuratedCommunitiesFromContract, m.fetchAndStoreCuratedCommunity)
}

// StopCuratedCommunitiesRefresh cancels the in-flight refresh, if any.
func (m *Messenger) StopCuratedCommunitiesRefresh() {
	m.curatedCommunitiesRefresh.stop()
}

func (m *Messenger) refreshCuratedCommunities(directory func() (*communities.CuratedCommunities, error), resolve curatedCommunityResolver) error {
	ctx, started := m.curatedCommunitiesRefresh.begin(m.ctx)
	if !started {
		return nil
	}

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
		defer m.curatedCommunitiesRefresh.end()

		m.config.messengerSignalsHandler.CuratedCommunitiesRefreshStarted()

		var resolved, unresolved int
		curatedCommunities, err := directory()
		if err != nil {
			m.logger.Error("failed to read curated communities directory", zap.Error(err))
		} else {
			if err := m.communitiesManager.SetCuratedCommunities(curatedCommunities); err != nil {
				m.logger.Error("failed to store curated communities", zap.Error(err))
			}
			resolved, unresolved = m.resolveCuratedCommunities(ctx, curatedCommunities.ContractCommunities, resolve)
		}

		m.config.messengerSignalsHandler.CuratedCommunitiesRefreshFinished(resolved, unresolved, ctx.Err() != nil)
	}()

	return nil
}

func (m *Messenger) resolveCuratedCommunities(ctx context.Context, communityIDs []string, resolve curatedCommunityResolver) (resolved, unresolved int) {
	const maxConcurrentResolves = 3

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	sem := make(chan struct{}, maxConcurrentResolves)

	for _, communityID := range communityIDs {
		select {
		case <-ctx.Done():
			wg.Wait()
			return resolved, unresolved
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(communityID string) {
			defer gocommon.LogOnPanic()
			defer wg.Done()
			defer func() { <-sem }()

			stored := resolve(ctx, communityID)
			m.config.messengerSignalsHandler.CuratedCommunityResolved(communityID, stored)

			mu.Lock()
			if stored {
				resolved++
			} else {
				unresolved++
			}
			mu.Unlock()
		}(communityID)
	}

	wg.Wait()
	return resolved, unresolved
}

// fetchAndStoreCuratedCommunity resolves a single curated entry through the
// bounded store node request manager and reports whether a description is now
// stored locally.
func (m *Messenger) fetchAndStoreCuratedCommunity(ctx context.Context, communityID string) bool {
	_, _, err := m.storeNodeRequestsManager.FetchCommunity(ctx, communityID, nil)
	if err != nil {
		m.logger.Error("failed to resolve curated community",
			zap.String("communityID", communityID),
			zap.Error(err))
	}

	response, err := m.communitiesManager.GetStoredDescriptionForCommunities([]string{communityID})
	if err != nil {
		m.logger.Error("failed to check stored curated community description",
			zap.String("communityID", communityID),
			zap.Error(err))
		return false
	}

	return len(response.Descriptions) > 0
}
