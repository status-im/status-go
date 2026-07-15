package protocol

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/waku-org/go-waku/waku/v2/api/history"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/contacts"

	"github.com/status-im/status-go/protocol/communities"
)

const (
	storeNodeAvailableTimeout = 30 * time.Second
)

// StoreNodeRequestStats is used in tests
type StoreNodeRequestStats struct {
	FetchedEnvelopesCount int
	FetchedPagesCount     int
}

type storeNodeRequestID struct {
	RequestType storeNodeRequestType `json:"requestType"`
	DataID      string               `json:"dataID"`
}

func (r *storeNodeRequestID) getCommunityID() string {
	switch r.RequestType {
	case storeNodeCommunityRequest:
		return r.DataID
	default:
		return ""
	}
}

type StoreNodeRequestManager struct {
	messenger *Messenger
	logger    *zap.Logger

	// activeRequests contain all ongoing store node requests.
	// Map is indexed with `DataID`.
	// Request might be duplicated in the map in case of contentType collisions.
	activeRequests map[storeNodeRequestID]*storeNodeRequest

	// activeRequestsLock should be locked each time activeRequests is being accessed or changed.
	activeRequestsLock sync.RWMutex

	onPerformingBatch func(types2.StoreNodeBatch)
}

func NewStoreNodeRequestManager(m *Messenger) *StoreNodeRequestManager {
	return &StoreNodeRequestManager{
		messenger:          m,
		logger:             m.logger.Named("StoreNodeRequestManager"),
		activeRequests:     map[storeNodeRequestID]*storeNodeRequest{},
		activeRequestsLock: sync.RWMutex{},
		onPerformingBatch:  nil,
	}
}

// FetchCommunity makes a single request to store node for a given community id.
// When a community is successfully fetched, a `CommunityFound` event will be emitted. If `waitForResponse == true`,
// the function will also wait for the store node response and return the fetched community.
// Automatically waits for an available store node.
// When a `nil` community and `nil` error is returned, that means the community wasn't found at the store node.
func (m *StoreNodeRequestManager) FetchCommunity(ctx context.Context, communityID string, opts []StoreNodeRequestOption) (*communities.Community, StoreNodeRequestStats, error) {
	cfg := buildCommunityStoreNodeRequestConfig(opts)

	m.logger.Info("requesting community from store node",
		zap.String("community", communityID),
		zap.Any("config", cfg))

	fetch := func() (*communities.Community, StoreNodeRequestStats, error) {
		sub, err := m.subscribeToRequest(ctx, storeNodeCommunityRequest, communityID, types2.DefaultShard(), cfg)
		if err != nil {
			return nil, StoreNodeRequestStats{}, fmt.Errorf("failed to create a shard info request: %w", err)
		}
		res := <-sub
		if res.community != nil {
			return res.community, res.stats, res.err
		}
		return nil, StoreNodeRequestStats{}, nil
	}

	if !cfg.WaitForResponse {
		go func() {
			defer gocommon.LogOnPanic()
			_, _, err := fetch()
			if err != nil {
				m.logger.Error("failed to fetch community", zap.Error(err))
			}
		}()
		return nil, StoreNodeRequestStats{}, nil
	}

	return fetch()
}

// FetchContact - similar to FetchCommunity
// If a `nil` contact and a `nil` error are returned, it means that the contact wasn't found at the store node.
func (m *StoreNodeRequestManager) FetchContact(ctx context.Context, contactID string, opts []StoreNodeRequestOption) (*contacts.Contact, StoreNodeRequestStats, error) {

	cfg := buildStoreNodeRequestConfig(opts)

	m.logger.Info("requesting contact from store node",
		zap.Any("contactID", contactID),
		zap.Any("config", cfg))

	channel, err := m.subscribeToRequest(ctx, storeNodeContactRequest, contactID, nil, cfg)
	if err != nil {
		return nil, StoreNodeRequestStats{}, fmt.Errorf("failed to create a request for community: %w", err)
	}

	if !cfg.WaitForResponse {
		return nil, StoreNodeRequestStats{}, nil
	}

	result := <-channel
	return result.contact, result.stats, result.err
}

// subscribeToRequest checks if a request for given community/contact is already in progress, creates and installs
// a new one if not found, and returns a subscription to the result of the found/started request.
// The subscription can then be used to get the result of the request, this could be either a community/contact or an error.
func (m *StoreNodeRequestManager) subscribeToRequest(ctx context.Context, requestType storeNodeRequestType, dataID string, shard *types2.Shard, cfg StoreNodeRequestConfig) (storeNodeResponseSubscription, error) {
	// It's important to unlock only after getting the subscription channel.
	// We also lock `activeRequestsLock` during finalizing the requests. This ensures that the subscription
	// created in this function will get the result even if the requests proceeds faster than this function ends.
	m.activeRequestsLock.Lock()
	defer m.activeRequestsLock.Unlock()

	requestID := storeNodeRequestID{
		RequestType: requestType,
		DataID:      dataID,
	}

	request, requestFound := m.activeRequests[requestID]

	if !requestFound {
		// Create corresponding filter
		var err error
		var filter *types2.ChatFilter
		filterCreated := false

		filter, filterCreated, err = m.getFilter(requestType, dataID, shard)
		if err != nil {
			if filterCreated {
				m.forgetFilter(filter)
			}
			return nil, fmt.Errorf("failed to create community filter: %w", err)
		}

		request = m.newStoreNodeRequest(ctx)
		request.config = cfg
		request.pubsubTopic = filter.PubsubTopic()
		request.requestID = requestID
		request.contentTopic = filter.ContentTopic()
		if filterCreated {
			request.filterToForget = filter
		}

		m.activeRequests[requestID] = request
		request.start()
	}

	return request.subscribe(), nil
}

// newStoreNodeRequest creates a new storeNodeRequest struct
func (m *StoreNodeRequestManager) newStoreNodeRequest(ctx context.Context) *storeNodeRequest {
	return &storeNodeRequest{
		manager:       m,
		ctx:           ctx,
		subscriptions: make([]storeNodeResponseSubscription, 0),
	}
}

// getFilter checks if a filter for a given community is already created and creates one of not found.
// Returns the found/created filter, a flag if the filter was created by the function and an error.
func (m *StoreNodeRequestManager) getFilter(requestType storeNodeRequestType, dataID string, shard *types2.Shard) (*types2.ChatFilter, bool, error) {
	// First check if such filter already exists.
	filter := m.messenger.messaging.ChatFilterByChatID(dataID)
	if filter != nil {
		//we don't remember filter id associated with community because it was already installed
		return filter, false, nil
	}

	switch requestType {
	case storeNodeCommunityRequest:
		// If filter wasn't installed we create it and
		// remember for uninstalling after response is received
		filters, err := m.messenger.messaging.InitPublicChats(types2.ChatsToInitialize{{
			ChatID:      dataID,
			PubsubTopic: shard.PubsubTopic(),
		}})

		if err != nil {
			m.logger.Error("failed to install filter for community", zap.Error(err))
			return nil, false, err
		}

		if len(filters) != 1 {
			m.logger.Error("Unexpected number of filters created")
			return nil, true, fmt.Errorf("unexepcted number of filters created")
		}

		filter = filters[0]
	case storeNodeContactRequest:
		publicKeyBytes, err := types.DecodeHex(dataID)
		if err != nil {
			return nil, false, fmt.Errorf("failed to decode contact id: %w", err)
		}

		publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
		if err != nil {
			return nil, false, fmt.Errorf("failed to unmarshal public key: %w", err)
		}

		filter, err = m.messenger.messaging.JoinPrivateChat(publicKey)
		if err != nil {
			return nil, false, fmt.Errorf("failed to install filter for contact: %w", err)
		}

	default:
		return nil, false, fmt.Errorf("invalid store node request type: %d", requestType)
	}

	err := m.messenger.messaging.UpdateFilterEphemerality(filter.ChatID(), true)
	if err != nil {
		return nil, false, fmt.Errorf("failed to update filter: %w", err)
	}

	return filter, true, nil
}

// forgetFilter uninstalls the given filter
func (m *StoreNodeRequestManager) forgetFilter(filter *types2.ChatFilter) {
	err := m.messenger.messaging.RemoveFilters(types2.ChatFilters{filter})
	if err != nil {
		m.logger.Warn("failed to remove filter", zap.Error(err))
	}
}

type storeNodeRequestType int

const (
	storeNodeCommunityRequest storeNodeRequestType = iota
	storeNodeContactRequest
)

// storeNodeRequest represents a single store node batch request.
// For a valid storeNodeRequest to be performed, the user must set all the struct fields and call start method.
type storeNodeRequest struct {
	requestID storeNodeRequestID
	ctx       context.Context

	// request parameters
	pubsubTopic      string
	contentTopic     types2.ContentTopic
	minimumDataClock uint64
	config           StoreNodeRequestConfig

	// descriptionSeen latches true once a description for the requested
	// community has been processed in this request; pages are newest-first, so
	// later pages can only carry older descriptions.
	descriptionSeen bool

	// request corresponding metadata to be used in finalize
	filterToForget *types2.ChatFilter

	// internal fields
	manager       *StoreNodeRequestManager
	subscriptions []storeNodeResponseSubscription
	result        storeNodeRequestResult
}

// storeNodeRequestResult contains result of a single storeNodeRequest
// Further by using `data` we mean community/contact, depending on request type.
// If any error occurs during the request, err field will be set.
// If data was successfully fetched, data field will contain the fetched information.
// If data wasn't found in store node, then a data will be set to `nil`.
// stats will contain information about the performed request that might be useful for testing.
type storeNodeRequestResult struct {
	err   error
	stats StoreNodeRequestStats
	// One of data fields (community or contact) will be present depending on request type
	community *communities.Community
	contact   *contacts.Contact
}

type storeNodeResponseSubscription = chan storeNodeRequestResult

func (r *storeNodeRequest) subscribe() storeNodeResponseSubscription {
	channel := make(storeNodeResponseSubscription, 100)
	r.subscriptions = append(r.subscriptions, channel)
	return channel
}

func (r *storeNodeRequest) finalize() {
	r.manager.activeRequestsLock.Lock()
	defer r.manager.activeRequestsLock.Unlock()

	r.manager.logger.Info("request finished",
		zap.Any("requestID", r.requestID),
		zap.Bool("communityFound", r.result.community != nil),
		zap.Bool("contactFound", r.result.contact != nil),
		zap.Error(r.result.err))

	// Send the result to subscribers
	// It's important that this is done with `activeRequestsLock` locked.
	for _, s := range r.subscriptions {
		s <- r.result
		close(s)
	}

	if r.result.community != nil {
		r.manager.messenger.passStoredCommunityInfoToSignalHandler(r.result.community)
	}

	delete(r.manager.activeRequests, r.requestID)

	if r.filterToForget != nil {
		r.manager.forgetFilter(r.filterToForget)
	}
}

// shouldFetchNextPage wraps the natural per-page decision with the
// description-seen and page-cap gates. A gate-forced stop returns false, which
// drives the request through the same finalize() path as an ordinary "not
// found" result, so the UI's loading state clears normally.
func (r *storeNodeRequest) shouldFetchNextPage(envelopesCount int) (bool, uint64) {
	fetchNext, pageSize := r.shouldFetchNextPageUncapped(envelopesCount)

	fetchNext, descriptionGateTripped := r.config.gateNextPageByDescriptionSeen(fetchNext, r.descriptionSeen)
	if descriptionGateTripped {
		r.manager.logger.Info("community description seen and not newer; stopping pagination (older pages cannot contain newer descriptions)",
			zap.Any("requestID", r.requestID),
			zap.Int("fetchedPagesCount", r.result.stats.FetchedPagesCount),
			zap.Int("fetchedEnvelopesCount", r.result.stats.FetchedEnvelopesCount))
	}

	fetchNext, pageSize, capTripped := r.config.gateNextPageByCap(fetchNext, pageSize, r.result.stats.FetchedPagesCount)
	if capTripped {
		r.manager.logger.Warn("store node request reached page cap; stopping pagination to bound runaway fetch",
			zap.Any("requestID", r.requestID),
			zap.Int("fetchedPagesCount", r.result.stats.FetchedPagesCount),
			zap.Int("fetchedEnvelopesCount", r.result.stats.FetchedEnvelopesCount),
			zap.Int("maxPageCount", r.config.MaxPageCount))
	}

	return fetchNext, pageSize
}

func (r *storeNodeRequest) shouldFetchNextPageUncapped(envelopesCount int) (bool, uint64) {
	logger := r.manager.logger.With(
		zap.Any("requestID", r.requestID),
		zap.Int("envelopesCount", envelopesCount))

	r.result.stats.FetchedEnvelopesCount += envelopesCount
	r.result.stats.FetchedPagesCount++

	// Flush this page's envelopes into the DB before the GetByID check, keeping
	// the response so a community request can tell whether this page carried a
	// description for the target community. Skipped for empty pages —
	// ProcessAllMessages traverses the messenger's whole pending backlog and an
	// empty page has nothing to flush; the DB check below still runs.
	var response *MessengerResponse
	if shouldProcessStoreNodePage(envelopesCount) {
		response = r.manager.messenger.ProcessAllMessages()
	} else {
		logger.Debug("skipping ProcessAllMessages for empty store node page")
	}

	// Try to get community from database
	switch r.requestID.RequestType {
	case storeNodeCommunityRequest:
		communityID, err := types.DecodeHex(r.requestID.DataID)
		if err != nil {
			logger.Error("failed to decode community ID",
				zap.String("communityID", r.requestID.DataID),
				zap.Error(err))
			r.result = storeNodeRequestResult{
				community: nil,
				err:       fmt.Errorf("failed to decode community ID: %w", err),
			}
			return false, 0 // failed to decode community ID, no sense to continue the procedure
		}

		// Store-node history is paged newest-first, so the first page that carries
		// a description for this community carries the newest available one. Latch
		// that here; once set, gateNextPageByDescriptionSeen stops paging because
		// older pages cannot contain a newer description (issue #21470-hf).
		if !r.descriptionSeen && communityDescriptionSeen(response, communityID) {
			r.descriptionSeen = true
		}

		// check if community is waiting for a verification and do a verification manually
		_, err = r.manager.messenger.communitiesManager.ValidateCommunityByID(communityID)
		if err != nil {
			logger.Error("failed to validate community by ID",
				zap.String("communityID", r.requestID.DataID),
				zap.Error(err))
			r.result = storeNodeRequestResult{
				community: nil,
				err:       fmt.Errorf("failed to validate community by ID: %w", err),
			}
			return false, 0 // failed to validate community, no sense to continue the procedure
		}

		community, err := r.manager.messenger.communitiesManager.GetByID(communityID)

		if err != nil && err != communities.ErrOrgNotFound {
			logger.Error("failed to read community from database",
				zap.String("communityID", r.requestID.DataID),
				zap.Error(err))
			r.result = storeNodeRequestResult{
				community: nil,
				err:       fmt.Errorf("failed to read community from database: %w", err),
			}
			return false, 0 // failed to read from database, no sense to continue the procedure
		}

		if community == nil {
			// The community is absent from the community table, but it may already
			// be in hand and merely withheld pending on-chain owner validation. In
			// that state the description has been fetched and only verification is
			// blocking persistence; verification retries out-of-band on the
			// owner-verification loop. Fetching more store-node pages cannot make a
			// failed verification succeed — it only floods the network and starves
			// the very RPC calls whose success would end the request (issue
			// #21470-hf). So once the description is queued for validation, stop.
			queuedClock, qErr := r.manager.messenger.communitiesManager.HighestQueuedValidationClock(communityID)
			if qErr != nil {
				logger.Warn("failed to read community validation queue; continuing to page",
					zap.String("communityID", r.requestID.DataID),
					zap.Error(qErr))
			} else if gateNextPageByValidationQueue(false, queuedClock, r.minimumDataClock) {
				logger.Info("community description queued for owner validation; stopping pagination (verification retries out-of-band)",
					zap.Uint64("queuedClock", queuedClock),
					zap.Uint64("minimumDataClock", r.minimumDataClock))
				return false, 0
			}

			// community not found in the database, request next page
			logger.Debug("community still not fetched")
			return true, r.config.FurtherPageSize
		}

		// We check here if the community was fetched actually fetched and updated, because it
		// could be that the community was already in the database when we started the fetching.
		//
		// Would be perfect if we could track that the community was in these particular envelopes,
		// but I don't think that's possible right now. We check if clock was updated instead.

		if community.Clock() <= r.minimumDataClock {
			logger.Debug("local community description is not newer than existing",
				zap.Any("existingClock", community.Clock()),
				zap.Any("minimumDataClock", r.minimumDataClock),
			)
			return true, r.config.FurtherPageSize
		}

		logger.Debug("community found",
			zap.String("displayName", community.Name()))

		r.result.community = community

	case storeNodeContactRequest:
		contact := r.manager.messenger.GetContactByID(r.requestID.DataID)

		if contact == nil {
			// contact not found in the database, request next page
			logger.Debug("contact still not fetched")
			return true, r.config.FurtherPageSize
		}

		logger.Debug("contact found",
			zap.String("displayName", contact.DisplayName))

		r.result.contact = contact
	}

	return !r.config.StopWhenDataFound, r.config.FurtherPageSize
}

func (r *storeNodeRequest) routine() {
	defer gocommon.LogOnPanic()

	r.manager.logger.Info("starting store node request",
		zap.Any("requestID", r.requestID),
		zap.String("pubsubTopic", r.pubsubTopic),
		zap.Any("contentTopic", r.contentTopic),
	)

	// Return a nil community and no error when request was
	// performed successfully, but no community/contact found.
	r.result = storeNodeRequestResult{
		err:       nil,
		community: nil,
		contact:   nil,
	}

	defer func() {
		r.finalize()
	}()

	communityID := r.requestID.getCommunityID()

	ctx, cancel := context.WithTimeout(r.ctx, storeNodeAvailableTimeout)
	defer cancel()
	if !r.manager.messenger.messaging.WaitForAvailableStoreNode(ctx) {
		r.result.err = fmt.Errorf("store node is not available")
		return
	}

	// Check if community already exists locally and get Clock.
	if r.requestID.RequestType == storeNodeCommunityRequest {
		localCommunity, _ := r.manager.messenger.communitiesManager.GetByIDString(communityID)
		if localCommunity != nil {
			r.minimumDataClock = localCommunity.Clock()
		}
	}

	// Start store node request
	from, to := r.manager.messenger.calculateMailserverTimeBounds(oneMonthDuration)

	storeNode := r.manager.messenger.messaging.GetActiveStorenode()
	_, err := r.manager.messenger.performStorenodeTask(func() (*MessengerResponse, error) {
		batch := types2.StoreNodeBatch{
			From:        from,
			To:          to,
			PubsubTopic: r.pubsubTopic,
			Topics:      []types2.ContentTopic{r.contentTopic},
		}
		r.manager.logger.Info("perform store node request", zap.Any("batch", batch))
		if r.manager.onPerformingBatch != nil {
			r.manager.onPerformingBatch(batch)
		}

		return nil, r.manager.messenger.processMailserverBatchWithOptions(storeNode, batch, r.config.InitialPageSize, r.shouldFetchNextPage, true)
	}, history.WithPeerID(storeNode.ID))

	r.result.err = err
}

func (r *storeNodeRequest) start() {
	go r.routine()
}

// communityDescriptionSeen reports whether the processed-messages response
// carries a description for the given community. A processed community
// description surfaces the community in the MessengerResponse via
// handleCommunityResponse -> AddCommunity — including the equal-clock
// re-processing that drives issue #21470-hf (UpdateCommunityDescription only
// rejects strictly-older clocks, so a re-fetched identical description still
// flows through and is added). Its presence is therefore the request-scoped
// "a description for this community was processed this page" signal used by
// gateNextPageByDescriptionSeen.
func communityDescriptionSeen(response *MessengerResponse, communityID []byte) bool {
	if response == nil {
		return false
	}
	for _, c := range response.Communities() {
		if c != nil && bytes.Equal(c.ID(), communityID) {
			return true
		}
	}
	return false
}
