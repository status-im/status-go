package protocol

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/transport"
)

const (
	storeNodeAvailableTimeout = 5 * time.Second
)

// FetchCommunityStats is used in tests
type FetchCommunityStats struct {
	FetchedEnvelopesCount int
	FetchedPagesCount     int
}

type StoreNodeRequestManager struct {
	messenger *Messenger
	logger    *zap.Logger

	// activeRequests contain all ongoing store node requests.
	// Map is indexed with `CommunityID`.
	// Request might be duplicated in the map if the request is for multiple communities.
	activeRequests map[string]*storeNodeRequest

	// activeRequestsLock should be locked each time activeRequests is being accessed or changed.
	activeRequestsLock sync.RWMutex

	onPerformingBatch func(MailserverBatch)
}

func NewCommunityRequestsManager(m *Messenger) *StoreNodeRequestManager {
	return &StoreNodeRequestManager{
		messenger:          m,
		logger:             m.logger.Named("StoreNodeRequestManager"),
		activeRequests:     map[string]*storeNodeRequest{},
		activeRequestsLock: sync.RWMutex{},
		onPerformingBatch:  nil,
	}
}

// FetchCommunity makes a single request to store node for a given community id/shard pair.
// When a community is successfully fetched, a `CommunityFound` event will be emitted. If `waitForResponse == true`,
// the function will also wait for the store node response and return the fetched community.
// Automatically waits for an available store node.
// When a `nil` community and `nil` error is returned, that means the community wasn't found at the store node.
func (m *StoreNodeRequestManager) FetchCommunity(community communities.CommunityShard, waitForResponse bool) (*communities.Community, FetchCommunityStats, error) {
	m.logger.Info("requesting community from store node",
		zap.Any("community", community),
		zap.Bool("waitForResponse", waitForResponse))

	channel, err := m.subscribeToCommunityRequest(community)
	if err != nil {
		return nil, FetchCommunityStats{}, fmt.Errorf("failed to create a request for community: %w", err)
	}

	if !waitForResponse {
		return nil, FetchCommunityStats{}, nil
	}

	result := <-*channel
	return result.community, result.stats, result.err
}

// FetchCommunities makes a FetchCommunity for each element in given `communities` list.
// For each successfully fetched community, a `CommunityFound` event will be emitted. Ability to subscribe
// to results is not provided, because it's not needed and would complicate the code. `FetchCommunity` can
// be called directly if such functionality is needed.
//
// This function intentionally doesn't fetch multiple content topics in a single store node request. For now
// FetchCommunities is only used for regular (once in 2 minutes) fetching of curated communities. If one of
// those content topics is spammed with to many envelopes, then on each iteration we will have to fetch all
// of this spam first to get the envelopes in other content topics. To avoid this we keep independent requests
// for each content topic.
func (m *StoreNodeRequestManager) FetchCommunities(communities []communities.CommunityShard) error {
	m.logger.Info("requesting communities from store node", zap.Any("communities", communities))

	var outErr error

	for _, community := range communities {
		_, _, err := m.FetchCommunity(community, false)
		if err != nil {
			outErr = fmt.Errorf("%sfailed to create a request for community %s: %w", outErr, community.CommunityID, err)
		}
	}

	return outErr
}

// subscribeToCommunityRequest checks if a request for given community is already in progress, creates and installs
// a new one if not found, and returns a subscription to the result of the found/started request.
// The subscription can then be used to get the result of the request, this could be either a community or an error.
func (m *StoreNodeRequestManager) subscribeToCommunityRequest(community communities.CommunityShard) (*communitySubscriptionChannel, error) {
	// It's important to unlock only after getting the subscription channel.
	// We also lock `activeRequestsLock` during finalizing the requests. This ensures that the subscription
	// created in this function will get the result even if the requests proceeds faster than this function ends.
	m.activeRequestsLock.Lock()
	defer m.activeRequestsLock.Unlock()

	request, requestFound := m.activeRequests[community.CommunityID]

	if !requestFound {
		// Create corresponding filter
		filter, filterCreated, err := m.getFilter(community)
		if err != nil {
			return nil, fmt.Errorf("failed to create community filter: %w", err)
		}

		request = m.newStoreNodeRequest()
		request.pubsubTopic = filter.PubsubTopic
		request.communityID = community.CommunityID
		request.contentTopic = filter.ContentTopic
		if filterCreated {
			request.filterToForget = filter
		}

		m.activeRequests[community.CommunityID] = request
		request.start()
	}

	return request.subscribe(), nil
}

// newStoreNodeRequest creates a new storeNodeRequest struct
func (m *StoreNodeRequestManager) newStoreNodeRequest() *storeNodeRequest {
	return &storeNodeRequest{
		manager:       m,
		subscriptions: make([]*communitySubscriptionChannel, 0),
	}
}

// getFilter checks if a filter for a given community is already created and creates one of not found.
// Returns the found/created filter, a flag if the filter was created by the function and an error.
func (m *StoreNodeRequestManager) getFilter(c communities.CommunityShard) (*transport.Filter, bool, error) {
	// First check if such filter already exists.
	filter := m.messenger.transport.FilterByChatID(c.CommunityID)
	if filter != nil {
		//we don't remember filter id associated with community because it was already installed
		return filter, false, nil
	}

	// If filter wasn't installed we create it and
	// remember for uninstalling after response is received
	filters, err := m.messenger.transport.InitPublicFilters([]transport.FiltersToInitialize{{
		ChatID:      c.CommunityID,
		PubsubTopic: c.Shard.PubsubTopic(),
	}})

	if err != nil {
		m.logger.Error("can't install filter for community", zap.Error(err))
		return nil, true, err
	}

	if len(filters) != 1 {
		m.logger.Error("Unexpected number of filters created")
		return nil, true, fmt.Errorf("unexepcted number of filters created")
	}

	return filters[0], true, nil
}

// forgetFilter uninstalls the given filter
func (m *StoreNodeRequestManager) forgetFilter(filter *transport.Filter) {
	err := m.messenger.transport.RemoveFilters([]*transport.Filter{filter})
	if err != nil {
		m.logger.Warn("failed to remove filter", zap.Error(err))
	}
}

// storeNodeRequest represents a single store node batch request.
// For a valid storeNodeRequest to be performed, the user must set all the struct fields and call start method.
type storeNodeRequest struct {
	// request parameters
	pubsubTopic  string
	contentTopic types.TopicType

	// request corresponding metadata to be used in finalize
	filterToForget *transport.Filter
	communityID    string

	// internal fields
	manager       *StoreNodeRequestManager
	subscriptions []*communitySubscriptionChannel
	result        fetchCommunityResult
}

// fetchCommunityResult contains result of a single storeNodeRequest
// If any error occurs during the request, err field will be set.
// If a community was successfully fetched, community field will contain the fetched information.
// If a community wasn't found in store node, then a community will be set to `nil`.
// stats will contain information about the performed request that might be useful for testing.
type fetchCommunityResult struct {
	err       error
	community *communities.Community
	stats     FetchCommunityStats
}

type communitySubscriptionChannel = chan fetchCommunityResult

func (r *storeNodeRequest) subscribe() *communitySubscriptionChannel {
	channel := make(communitySubscriptionChannel, 100)
	r.subscriptions = append(r.subscriptions, &channel)
	return &channel
}

func (r *storeNodeRequest) finalize() {
	r.manager.activeRequestsLock.Lock()
	defer r.manager.activeRequestsLock.Unlock()

	r.manager.logger.Debug("request finished",
		zap.String("communityID", r.communityID),
		zap.Bool("communityFound", r.result.community != nil),
		zap.Error(r.result.err))

	// Send the result to subscribers
	// It's important that this is done with `activeRequestsLock` locked.
	for _, s := range r.subscriptions {
		*s <- r.result
	}

	if r.result.community != nil {
		r.manager.messenger.passStoredCommunityInfoToSignalHandler(r.result.community)
	}

	delete(r.manager.activeRequests, r.communityID)

	if r.filterToForget != nil {
		r.manager.forgetFilter(r.filterToForget)
	}
}

func (r *storeNodeRequest) shouldFetchNextPage(envelopesCount int) (bool, uint32) {
	logger := r.manager.logger.With(
		zap.String("communityID", r.communityID),
		zap.Int("envelopesCount", envelopesCount))

	r.result.stats.FetchedEnvelopesCount += envelopesCount
	r.result.stats.FetchedPagesCount++

	// Force all received envelopes to be processed
	r.manager.messenger.ProcessAllMessages()

	// Try to get community from database
	community, err := r.manager.messenger.communitiesManager.GetByIDString(r.communityID)

	if err != nil {
		logger.Error("failed to read from database",
			zap.String("communityID", r.communityID),
			zap.Error(err))
		r.result = fetchCommunityResult{
			community: nil,
			err:       fmt.Errorf("failed to read from database: %w", err),
		}
		return false, 0 // failed to read from database, no sense to continue the procedure
	}

	if community == nil {
		// community not found in the database, request next page
		logger.Debug("community still not fetched")
		return true, defaultStoreNodeRequestPageSize
	}

	logger.Debug("community found",
		zap.String("displayName", community.Name()))

	r.result.community = community

	return false, 0
}

func (r *storeNodeRequest) routine() {
	// Return a nil community and no error when request was
	// performed successfully, but no community found.
	r.result = fetchCommunityResult{
		community: nil,
		err:       nil,
	}

	defer func() {
		r.finalize()
	}()

	if !r.manager.messenger.waitForAvailableStoreNode(storeNodeAvailableTimeout) {
		r.result.community = nil
		r.result.err = fmt.Errorf("store node is not available")
		return
	}

	to := uint32(math.Ceil(float64(r.manager.messenger.GetCurrentTimeInMillis()) / 1000))
	from := to - oneMonthInSeconds

	_, err := r.manager.messenger.performMailserverRequest(func() (*MessengerResponse, error) {
		batch := MailserverBatch{
			From:        from,
			To:          to,
			PubsubTopic: r.pubsubTopic,
			Topics:      []types.TopicType{r.contentTopic},
		}
		r.manager.logger.Info("perform store node request", zap.Any("batch", batch))
		if r.manager.onPerformingBatch != nil {
			r.manager.onPerformingBatch(batch)
		}

		return nil, r.manager.messenger.processMailserverBatchWithOptions(batch, 1, r.shouldFetchNextPage, true)
	})

	r.result.err = err
}

func (r *storeNodeRequest) start() {
	r.manager.logger.Debug("starting new community request",
		zap.Any("communities", r.communityID),
		zap.String("pubsubTopic", r.pubsubTopic),
		zap.Any("contentTopic", r.contentTopic),
	)

	go r.routine()
}
