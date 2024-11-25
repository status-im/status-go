package activity

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/routeexecution"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"

	"go.uber.org/zap"
)

type Version string

const (
	V1 Version = "v1"
	V2 Version = "v2"
)

type TransactionID struct {
	ChainID common.ChainID
	Hash    eth.Hash
}

func (t TransactionID) key() string {
	return strconv.FormatUint(uint64(t.ChainID), 10) + t.Hash.Hex()
}

type EntryUpdate struct {
	Pos   int    `json:"pos"`
	Entry *Entry `json:"entry"`
}

// SessionUpdate payload for EventActivitySessionUpdated
type SessionUpdate struct {
	HasNewOnTop *bool           `json:"hasNewOnTop,omitempty"`
	New         []*EntryUpdate  `json:"new,omitempty"`
	Removed     []EntryIdentity `json:"removed,omitempty"`
}

type fullFilterParams struct {
	sessionID SessionID
	version   Version
	addresses []eth.Address
	chainIDs  []common.ChainID
	filter    Filter
}

func (s *Service) getActivityEntries(ctx context.Context, f fullFilterParams, offset int, count int) ([]Entry, error) {
	allAddresses := s.areAllAddresses(f.addresses)
	if f.version == V1 {
		return getActivityEntries(ctx, s.getDeps(), f.addresses, allAddresses, f.chainIDs, f.filter, offset, count)
	}
	return getActivityEntriesV2(ctx, s.getDeps(), f.addresses, allAddresses, f.chainIDs, f.filter, offset, count)
}

func (s *Service) internalFilter(f fullFilterParams, offset int, count int, processResults func(entries []Entry) (offsetOverride int)) {
	s.scheduler.Enqueue(int32(f.sessionID), filterTask, func(ctx context.Context) (interface{}, error) {
		return s.getActivityEntries(ctx, f, offset, count)
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := FilterResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			activities := result.([]Entry)
			res.Activities = activities
			res.HasMore = len(activities) == count
			res.ErrorCode = ErrorCodeSuccess

			res.Offset = processResults(activities)
		}

		int32SessionID := int32(f.sessionID)
		sendResponseEvent(s.eventFeed, &int32SessionID, EventActivityFilteringDone, res, err)

		s.getActivityDetailsAsync(int32SessionID, res.Activities)
	})
}

// mirrorIdentities for update use
func mirrorIdentities(entries []Entry) []EntryIdentity {
	model := make([]EntryIdentity, 0, len(entries))
	for _, a := range entries {
		model = append(model, EntryIdentity{
			payloadType: a.payloadType,
			transaction: a.transaction,
			id:          a.id,
		})
	}
	return model
}

func (s *Service) internalFilterForSession(session *Session, firstPageCount int) {
	s.internalFilter(
		session.getFullFilterParams(),
		0,
		firstPageCount,
		func(entries []Entry) (offset int) {
			session.model = mirrorIdentities(entries)

			return 0
		},
	)
}

func (s *Service) StartFilterSession(addresses []eth.Address, chainIDs []common.ChainID, filter Filter, firstPageCount int, version Version) SessionID {
	sessionID := s.nextSessionID()

	session := &Session{
		id:      sessionID,
		version: version,

		addresses: addresses,
		chainIDs:  chainIDs,
		filter:    filter,

		model: make([]EntryIdentity, 0, firstPageCount),

		mu: &sync.RWMutex{},
	}

	s.addSession(session)

	session.mu.Lock()
	defer session.mu.Unlock()
	s.internalFilterForSession(session, firstPageCount)

	return sessionID
}

// UpdateFilterForSession is to be called for updating the filter of a specific session
// After calling this method to set a filter all the incoming changes will be reported with
// Entry.isNew = true when filter is reset to empty
func (s *Service) UpdateFilterForSession(id SessionID, filter Filter, firstPageCount int) error {
	session := s.getSession(id)
	if session == nil {
		return ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	prevFilterEmpty := session.filter.IsEmpty()
	newFilerEmpty := filter.IsEmpty()

	session.new = nil

	session.filter = filter

	if prevFilterEmpty && !newFilerEmpty {
		// Session is moving from empty to non-empty filter
		// Take a snapshot of the current model
		session.noFilterModel = entryIdsToMap(session.model)

		session.model = make([]EntryIdentity, 0, firstPageCount)

		// In this case there is nothing to flag so we request the first page
		s.internalFilterForSession(session, firstPageCount)
	} else if !prevFilterEmpty && newFilerEmpty {
		// Session is moving from non-empty to empty filter
		// In this case we need to flag all the new entries that are not in the noFilterModel
		s.internalFilter(
			session.getFullFilterParams(),
			0,
			firstPageCount,
			func(entries []Entry) (offset int) {
				// Mark new entries
				for i, a := range entries {
					_, found := session.noFilterModel[a.getIdentity().key()]
					entries[i].isNew = !found
				}

				// Mirror identities for update use
				session.model = mirrorIdentities(entries)
				session.noFilterModel = nil
				return 0
			},
		)
	} else {
		// Else act as a normal filter update
		s.internalFilterForSession(session, firstPageCount)
	}

	return nil
}

// ResetFilterSession is to be called when SessionUpdate.HasNewOnTop == true to
// update client with the latest state including new on top entries
func (s *Service) ResetFilterSession(id SessionID, firstPageCount int) error {
	session := s.getSession(id)
	if session == nil {
		return ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	s.internalFilter(
		session.getFullFilterParams(),
		0,
		firstPageCount,
		func(entries []Entry) (offset int) {
			// Mark new entries
			newMap := entryIdsToMap(session.new)
			for i, a := range entries {
				_, isNew := newMap[a.getIdentity().key()]
				entries[i].isNew = isNew
			}
			session.new = nil

			if session.noFilterModel != nil {
				// Add reported new entries to mark them as seen
				for _, a := range newMap {
					session.noFilterModel[a.key()] = a
				}
			}

			// Mirror client identities for checking updates
			session.model = mirrorIdentities(entries)

			return 0
		},
	)
	return nil
}

func (s *Service) GetMoreForFilterSession(id SessionID, pageCount int) error {
	session := s.getSession(id)
	if session == nil {
		return ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	prevModelLen := len(session.model)
	s.internalFilter(
		session.getFullFilterParams(),
		prevModelLen+len(session.new),
		pageCount,
		func(entries []Entry) (offset int) {
			// Mirror client identities for checking updates
			for _, a := range entries {
				session.model = append(session.model, EntryIdentity{
					payloadType: a.payloadType,
					transaction: a.transaction,
					id:          a.id,
				})
			}

			// Overwrite the offset to account for new entries
			return prevModelLen
		},
	)
	return nil
}

// subscribeToEvents should be called with sessionsRWMutex locked for writing
func (s *Service) subscribeToEvents() {
	s.ch = make(chan walletevent.Event, 100)
	s.subscriptions = s.eventFeed.Subscribe(s.ch)
	ctx, cancel := context.WithCancel(context.Background())
	s.subscriptionsCancelFn = cancel
	go s.processEvents(ctx)
}

// processEvents runs only if more than one session is active
func (s *Service) processEvents(ctx context.Context) {
	defer gocommon.LogOnPanic()
	eventCount := 0
	changedTxs := make([]TransactionID, 0)
	newTxs := false

	var debounceTimer *time.Timer
	debouncerCh := make(chan struct{})
	debounceProcessChangesFn := func() {
		if debounceTimer == nil {
			debounceTimer = time.AfterFunc(s.debounceDuration, func() {
				debouncerCh <- struct{}{}
			})
		}
	}

	for {
		select {
		case event := <-s.ch:
			switch event.Type {
			case transactions.EventPendingTransactionUpdate:
				eventCount++
				var payload transactions.PendingTxUpdatePayload
				if err := json.Unmarshal([]byte(event.Message), &payload); err != nil {
					logutils.ZapLogger().Error("Error unmarshalling PendingTxUpdatePayload", zap.Error(err))
					continue
				}
				changedTxs = append(changedTxs, TransactionID{
					ChainID: payload.ChainID,
					Hash:    payload.Hash,
				})
				debounceProcessChangesFn()
			case transactions.EventPendingTransactionStatusChanged:
				eventCount++
				var payload transactions.StatusChangedPayload
				if err := json.Unmarshal([]byte(event.Message), &payload); err != nil {
					logutils.ZapLogger().Error("Error unmarshalling StatusChangedPayload", zap.Error(err))
					continue
				}
				changedTxs = append(changedTxs, TransactionID{
					ChainID: payload.ChainID,
					Hash:    payload.Hash,
				})
				debounceProcessChangesFn()
			case transfer.EventNewTransfers:
				eventCount++
				// No updates here, these are detected with their final state, just trigger
				// the detection of new entries
				newTxs = true
				debounceProcessChangesFn()
			case routeexecution.EventRouteExecutionTransactionSent:
				sentTxs, ok := event.EventParams.(*responses.RouterSentTransactions)
				if ok && sentTxs != nil {
					for _, tx := range sentTxs.SentTransactions {
						changedTxs = append(changedTxs, TransactionID{
							ChainID: common.ChainID(tx.FromChain),
							Hash:    eth.Hash(tx.Hash),
						})
					}
				}
				debounceProcessChangesFn()
			}
		case <-debouncerCh:
			if eventCount > 0 || newTxs || len(changedTxs) > 0 {
				s.processChanges(eventCount, changedTxs)
				eventCount = 0
				newTxs = false
				changedTxs = nil
				debounceTimer = nil
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) processChangesForSession(session *Session, eventCount int, changedTxs []TransactionID) {
	session.mu.Lock()
	defer session.mu.Unlock()

	f := session.getFullFilterParams()
	limit := NoLimit
	if session.version == V1 {
		limit = len(session.model) + eventCount
	}
	activities, err := s.getActivityEntries(context.Background(), f, 0, limit)
	if err != nil {
		logutils.ZapLogger().Error("Error getting activity entries", zap.Error(err))
		return
	}

	if session.version != V1 {
		s.processEntryDataUpdates(session.id, activities, changedTxs)
	}

	allData := append(session.new, session.model...)
	new, _ /*removed*/ := findUpdates(allData, activities)

	lastProcessed := -1
	onTop := true
	var mixed []*EntryUpdate
	for i, idRes := range new {
		// Detect on top
		if onTop {
			// mixedIdentityResult.newPos includes session.new, therefore compensate for it
			if ((idRes.newPos - len(session.new)) - lastProcessed) > 1 {
				// From now on the events are not on top and continuous but mixed between existing entries
				onTop = false
				mixed = make([]*EntryUpdate, 0, len(new)-i)
			}
			lastProcessed = idRes.newPos
		}

		if onTop {
			if session.new == nil {
				session.new = make([]EntryIdentity, 0, len(new))
			}
			session.new = append(session.new, idRes.id)
		} else {
			modelPos := idRes.newPos - len(session.new)
			entry := activities[idRes.newPos]
			entry.isNew = true
			mixed = append(mixed, &EntryUpdate{
				Pos:   modelPos,
				Entry: &entry,
			})
			// Insert in session model at modelPos index
			session.model = append(session.model[:modelPos], append([]EntryIdentity{{payloadType: entry.payloadType, transaction: entry.transaction, id: entry.id}}, session.model[modelPos:]...)...)
		}
	}

	if len(session.new) > 0 || len(mixed) > 0 {
		go notify(s.eventFeed, session.id, len(session.new) > 0, mixed)
	}
}

func (s *Service) processChanges(eventCount int, changedTxs []TransactionID) {
	sessions := s.getAllSessions()
	for _, session := range sessions {
		s.processChangesForSession(session, eventCount, changedTxs)
	}
}

func (s *Service) processEntryDataUpdates(sessionID SessionID, entries []Entry, changedTxs []TransactionID) {
	updateData := make([]*EntryData, 0, len(changedTxs))

	entriesMap := make(map[string]Entry, len(entries))
	for _, e := range entries {
		if e.payloadType == MultiTransactionPT {
			if e.id != common.NoMultiTransactionID {
				for _, tx := range e.transactions {
					id := TransactionID{
						ChainID: tx.ChainID,
						Hash:    tx.Hash,
					}
					entriesMap[id.key()] = e
				}
			}
		} else if e.transaction != nil {
			id := TransactionID{
				ChainID: e.transaction.ChainID,
				Hash:    e.transaction.Hash,
			}
			entriesMap[id.key()] = e
		}
	}

	for _, tx := range changedTxs {
		e, found := entriesMap[tx.key()]
		if !found {
			continue
		}

		data := &EntryData{
			ActivityStatus: &e.activityStatus,
		}
		if e.payloadType == MultiTransactionPT {
			data.ID = common.NewAndSet(e.id)
		} else {
			data.Transaction = e.transaction
		}
		data.PayloadType = e.payloadType

		updateData = append(updateData, data)
	}

	if len(updateData) > 0 {
		requestID := int32(sessionID)
		sendResponseEvent(s.eventFeed, &requestID, EventActivityFilteringUpdate, updateData, nil)
	}
}

func notify(eventFeed *event.Feed, id SessionID, hasNewOnTop bool, mixed []*EntryUpdate) {
	defer gocommon.LogOnPanic()
	payload := SessionUpdate{
		New: mixed,
	}

	if hasNewOnTop {
		payload.HasNewOnTop = &hasNewOnTop
	}

	sendResponseEvent(eventFeed, (*int32)(&id), EventActivitySessionUpdated, payload, nil)
}

// unsubscribeFromEvents should be called with sessionsRWMutex locked for writing
func (s *Service) unsubscribeFromEvents() {
	s.subscriptionsCancelFn()
	s.subscriptionsCancelFn = nil
	s.subscriptions.Unsubscribe()
	close(s.ch)
	s.ch = nil
	s.subscriptions = nil
}

func (s *Service) StopFilterSession(id SessionID) {
	s.removeSesssion(id)

	// Cancel any pending or ongoing task
	s.scheduler.Enqueue(int32(id), filterTask, func(ctx context.Context) (interface{}, error) {
		return nil, nil
	}, func(result interface{}, taskType async.TaskType, err error) {})
}

func (s *Service) getActivityDetailsAsync(requestID int32, entries []Entry) {
	if len(entries) == 0 {
		return
	}

	ctx := context.Background()

	go func() {
		defer gocommon.LogOnPanic()
		activityData, err := s.getActivityDetails(ctx, entries)
		if len(activityData) != 0 {
			sendResponseEvent(s.eventFeed, &requestID, EventActivityFilteringUpdate, activityData, err)
		}
	}()
}

type mixedIdentityResult struct {
	newPos int
	id     EntryIdentity
}

func entryIdsToMap(ids []EntryIdentity) map[string]EntryIdentity {
	idsMap := make(map[string]EntryIdentity, len(ids))
	for _, id := range ids {
		idsMap[id.key()] = id
	}
	return idsMap
}

func entriesToMap(entries []Entry) map[string]Entry {
	entryMap := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		updatedIdentity := entry.getIdentity()
		entryMap[updatedIdentity.key()] = entry
	}
	return entryMap
}

// FindUpdates returns changes in updated entries compared to the identities
//
// expects identities and entries to be sorted by timestamp
//
// the returned newer are entries that are newer than the first identity
// the returned mixed are entries that are older than the first identity (sorted by timestamp)
// the returned removed are identities that are not present in the updated entries (sorted by timestamp)
//
// implementation assumes the order of each identity doesn't change from old state (identities) and new state (updated); we have either add or removed.
func findUpdates(identities []EntryIdentity, updated []Entry) (new []mixedIdentityResult, removed []EntryIdentity) {
	if len(updated) == 0 {
		return
	}

	idsMap := entryIdsToMap(identities)
	updatedMap := entriesToMap(updated)

	for newIndex, entry := range updated {
		id := entry.getIdentity()
		if _, found := idsMap[id.key()]; !found {
			new = append(new, mixedIdentityResult{
				newPos: newIndex,
				id:     id,
			})
		}

		if len(identities) > 0 && entry.getIdentity().same(identities[len(identities)-1]) {
			break
		}
	}

	// Account for new entries
	for i := 0; i < len(identities); i++ {
		id := identities[i]
		if _, found := updatedMap[id.key()]; !found {
			removed = append(removed, id)
		}
	}
	return
}

func (s *Service) addSession(session *Session) {
	s.sessionsRWMutex.Lock()
	defer s.sessionsRWMutex.Unlock()
	subscribeToEvents := len(s.sessions) == 0

	s.sessions[session.id] = session

	if subscribeToEvents {
		s.subscribeToEvents()
	}
}

func (s *Service) removeSesssion(id SessionID) {
	s.sessionsRWMutex.Lock()
	defer s.sessionsRWMutex.Unlock()
	delete(s.sessions, id)
	if len(s.sessions) == 0 {
		s.unsubscribeFromEvents()
	}
}

func (s *Service) getSession(id SessionID) *Session {
	s.sessionsRWMutex.RLock()
	defer s.sessionsRWMutex.RUnlock()
	return s.sessions[id]
}

func (s *Service) getAllSessions() []*Session {
	s.sessionsRWMutex.RLock()
	defer s.sessionsRWMutex.RUnlock()
	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}
