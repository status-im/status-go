package activity

import (
	"context"
	"encoding/json"
	"errors"

	"golang.org/x/exp/slices"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"
)

type EntryIdentity struct {
	payloadType PayloadType
	transaction *transfer.TransactionIdentity
	id          transfer.MultiTransactionIDType
}

type SessionID int32

type Session struct {
	id SessionID

	// Filter info
	//
	addresses    []eth.Address
	allAddresses bool
	chainIDs     []common.ChainID
	filter       Filter

	// model is a mirror of the data model presentation has (EventActivityFilteringDone)
	model []EntryIdentity
}

// SessionUpdate payload for EventActivitySessionUpdated
type SessionUpdate struct {
	// TODO #12120: add index for each entry, now all new are first entries
	NewEntries []Entry         `json:"newEntries,omitempty"`
	Removed    []EntryIdentity `json:"removed,omitempty"`
	Updated    []Entry         `json:"updated,omitempty"`
}

type fullFilterParams struct {
	sessionID    SessionID
	addresses    []eth.Address
	allAddresses bool
	chainIDs     []common.ChainID
	filter       Filter
}

func (s *Service) internalFilter(f fullFilterParams, offset int, count int, processResults func(entries []Entry)) {
	s.scheduler.Enqueue(int32(f.sessionID), filterTask, func(ctx context.Context) (interface{}, error) {
		activities, err := getActivityEntries(ctx, s.getDeps(), f.addresses, f.allAddresses, f.chainIDs, f.filter, offset, count)
		return activities, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := FilterResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			activities := result.([]Entry)
			res.Activities = activities
			res.Offset = 0
			res.HasMore = len(activities) == count
			res.ErrorCode = ErrorCodeSuccess

			processResults(activities)
		}

		int32SessionID := int32(f.sessionID)
		sendResponseEvent(s.eventFeed, &int32SessionID, EventActivityFilteringDone, res, err)

		s.getActivityDetailsAsync(int32SessionID, res.Activities)
	})
}

func (s *Service) StartFilterSession(addresses []eth.Address, allAddresses bool, chainIDs []common.ChainID, filter Filter, firstPageCount int) SessionID {
	sessionID := s.nextSessionID()

	// TODO #12120: sort rest of the filters
	// TODO #12120: prettyfy this
	slices.SortFunc(addresses, func(a eth.Address, b eth.Address) bool {
		return a.Hex() < b.Hex()
	})
	slices.Sort(chainIDs)
	slices.SortFunc(filter.CounterpartyAddresses, func(a eth.Address, b eth.Address) bool {
		return a.Hex() < b.Hex()
	})

	s.sessionsRWMutex.Lock()
	subscribeToEvents := len(s.sessions) == 0
	s.sessions[sessionID] = &Session{
		id: sessionID,

		addresses:    addresses,
		allAddresses: allAddresses,
		chainIDs:     chainIDs,
		filter:       filter,

		model: make([]EntryIdentity, 0, firstPageCount),
	}
	if subscribeToEvents {
		s.subscribeToEvents()
	}
	s.sessionsRWMutex.Unlock()

	s.internalFilter(fullFilterParams{
		sessionID:    sessionID,
		addresses:    addresses,
		allAddresses: allAddresses,
		chainIDs:     chainIDs,
		filter:       filter,
	}, 0, firstPageCount, func(entries []Entry) {
		// Mirror identities for update use
		s.sessionsRWMutex.Lock()
		defer s.sessionsRWMutex.Unlock()
		session, ok := s.sessions[sessionID]
		if ok {
			session.model = make([]EntryIdentity, 0, len(entries))
			for _, a := range entries {
				session.model = append(session.model, EntryIdentity{
					payloadType: a.payloadType,
					transaction: a.transaction,
					id:          a.id,
				})
			}
		}
	})

	return sessionID
}

// TODO #12120: extend the session based API
//func (s *Service) GetMoreForFilterSession(count int) {}

// subscribeToEvents should be called with sessionsRWMutex locked for writing
func (s *Service) subscribeToEvents() {
	s.ch = make(chan walletevent.Event, 100)
	s.subscriptions = s.eventFeed.Subscribe(s.ch)
	go s.processEvents()
}

func (s *Service) processEvents() {
	for event := range s.ch {
		if event.Type == transactions.EventPendingTransactionUpdate {
			var p transactions.PendingTxUpdatePayload
			err := json.Unmarshal([]byte(event.Message), &p)
			if err != nil {
				log.Error("Error unmarshalling PendingTxUpdatePayload", "error", err)
				continue
			}

			for id := range s.sessions {
				s.sessionsRWMutex.RLock()
				pTx, pass := s.checkFilterForPending(s.sessions[id], p.TxIdentity)
				if pass {
					s.sessionsRWMutex.RUnlock()
					s.sessionsRWMutex.Lock()
					addOnTop(s.sessions[id], p.TxIdentity)
					s.sessionsRWMutex.Unlock()
					// TODO #12120: can't send events from an event handler
					go notify(s.eventFeed, id, *pTx)
				} else {
					s.sessionsRWMutex.RUnlock()
				}
			}
		}
	}
}

// checkFilterForPending should be called with sessionsRWMutex locked for reading
func (s *Service) checkFilterForPending(session *Session, id transactions.TxIdentity) (tr *transactions.PendingTransaction, pass bool) {
	allChains := len(session.chainIDs) == 0
	if !allChains {
		_, found := slices.BinarySearch(session.chainIDs, id.ChainID)
		if !found {
			return nil, false
		}
	}

	tr, err := s.pendingTracker.GetPendingEntry(id.ChainID, id.Hash)
	if err != nil {
		log.Error("Error getting pending entry", "error", err)
		return nil, false
	}

	if !session.allAddresses {
		_, found := slices.BinarySearchFunc(session.addresses, tr.From, func(a eth.Address, b eth.Address) int {
			// TODO #12120: optimize this
			if a.Hex() < b.Hex() {
				return -1
			}
			if a.Hex() > b.Hex() {
				return 1
			}
			return 0
		})
		if !found {
			return nil, false
		}
	}

	fl := session.filter
	if fl.Period.StartTimestamp != NoLimitTimestampForPeriod || fl.Period.EndTimestamp != NoLimitTimestampForPeriod {
		ts := int64(tr.Timestamp)
		if ts < fl.Period.StartTimestamp || ts > fl.Period.EndTimestamp {
			return nil, false
		}
	}

	// TODO #12120 check filter
	// Types                 []Type        `json:"types"`
	// Statuses              []Status      `json:"statuses"`
	// CounterpartyAddresses []eth.Address `json:"counterpartyAddresses"`

	// // Tokens
	// Assets                []Token `json:"assets"`
	// Collectibles          []Token `json:"collectibles"`
	// FilterOutAssets       bool    `json:"filterOutAssets"`
	// FilterOutCollectibles bool    `json:"filterOutCollectibles"`

	return tr, true
}

// addOnTop should be called with sessionsRWMutex locked for writing
func addOnTop(session *Session, id transactions.TxIdentity) {
	session.model = append([]EntryIdentity{{
		payloadType: PendingTransactionPT,
		transaction: &transfer.TransactionIdentity{
			ChainID: id.ChainID,
			Hash:    id.Hash,
		},
	}}, session.model...)
}

func notify(eventFeed *event.Feed, id SessionID, tx transactions.PendingTransaction) {
	payload := SessionUpdate{
		NewEntries: []Entry{
			{
				payloadType: PendingTransactionPT,
				transaction: &transfer.TransactionIdentity{
					ChainID: tx.ChainID,
					Hash:    tx.Hash,
					Address: tx.From,
				},
				id:              transfer.NoMultiTransactionID,
				timestamp:       int64(tx.Timestamp),
				activityType:    SendAT,
				activityStatus:  PendingAS,
				amountOut:       (*hexutil.Big)(tx.Value.Int),
				amountIn:        nil,
				tokenOut:        nil,
				tokenIn:         nil,
				symbolOut:       &tx.Symbol,
				symbolIn:        nil,
				sender:          &tx.From,
				recipient:       &tx.To,
				chainIDOut:      &tx.ChainID,
				chainIDIn:       nil,
				transferType:    nil,
				contractAddress: nil,
			},
		},
	}

	sendResponseEvent(eventFeed, (*int32)(&id), EventActivitySessionUpdated, payload, nil)
}

// unsubscribeFromEvents should be called with sessionsRWMutex locked for writing
func (s *Service) unsubscribeFromEvents() {
	s.subscriptions.Unsubscribe()
	s.subscriptions = nil
}

func (s *Service) StopFilterSession(id SessionID) {
	s.sessionsRWMutex.Lock()
	delete(s.sessions, id)
	if len(s.sessions) == 0 {
		s.unsubscribeFromEvents()
	}
	s.sessionsRWMutex.Unlock()

	// Cancel any pending or ongoing task
	s.scheduler.Enqueue(int32(id), filterTask, func(ctx context.Context) (interface{}, error) {
		return nil, nil
	}, func(result interface{}, taskType async.TaskType, err error) {
		// Ignore result
	})
}

func (s *Service) getActivityDetailsAsync(requestID int32, entries []Entry) {
	if len(entries) == 0 {
		return
	}

	ctx := context.Background()

	go func() {
		activityData, err := s.getActivityDetails(ctx, entries)
		if len(activityData) != 0 {
			sendResponseEvent(s.eventFeed, &requestID, EventActivityFilteringUpdate, activityData, err)
		}
	}()
}
