package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	eth "github.com/ethereum/go-ethereum/common"
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
	NewEntries []Entry         `json:"newEntries,omitempty"`
	Removed    []EntryIdentity `json:"removed,omitempty"`
	Updated    []Entry         `json:"updated,omitempty"`
}

type fullFilter struct {
	sessionID    SessionID
	addresses    []eth.Address
	allAddresses bool
	chainIDs     []common.ChainID
	filter       Filter
}

func (s *Service) internalFilter(f fullFilter, offset int, count int, processResults func(entries []Entry)) {
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

	s.internalFilter(fullFilter{
		sessionID:    sessionID,
		addresses:    addresses,
		allAddresses: allAddresses,
		chainIDs:     chainIDs,
		filter:       filter,
	}, 0, firstPageCount, func(entries []Entry) {
		// Mirror identities for update use
		s.sessionsRWMutex.Lock()
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
		s.sessionsRWMutex.Unlock()
	})

	return sessionID
}

// TODO: #12120: extend the session based API
//func (s *Service) GetMoreForFilterSession(count int) {}

// subscribeToEvents should be called with sessionsRWMutex locked for writing
func (s *Service) subscribeToEvents() {
	s.ch = make(chan walletevent.Event)
	s.subscriptions = s.eventFeed.Subscribe(s.ch)
	go s.processEvents()
}

func (s *Service) processEvents() {
	for {
		select {
		case event := <-s.ch:
			if event.Type == transactions.EventPendingTransactionUpdate {
				// TODO:
				fmt.Println("@dd transactions.EventPendingTransactionUpdate", event)

				var p transactions.PendingTxUpdatePayload
				err := json.Unmarshal([]byte(event.Message), &p)
				if err != nil {
					log.Error("Error unmarshalling PendingTxUpdatePayload", "error", err)
					continue
				}

				s.sessionsRWMutex.RLock()
				for id, _ := range s.sessions {
					if checkFilter(s.sessions[id], p.TxIdentity) {
						tx := addOnTop(s.sessions[id], p.TxIdentity)
						notify(s.eventFeed, s.sessions[id], tx)
					}
				}
				s.sessionsRWMutex.RUnlock()
			}
		}
	}
}

// checkFilter should be called with sessionsRWMutex locked for reading
func checkFilter(session *Session, id transactions.TxIdentity) bool {
	// TODO #12120: check filter only
	return true
}

// addOnTop should be called with sessionsRWMutex locked for writing
func addOnTop(session *Session, id transactions.TxIdentity) transactions.PendingTransaction {
	// TODO #12120: add identity to session model
	return transactions.PendingTransaction{}
}

func notify(eventFeed *event.Feed, session *Session, tx transactions.PendingTransaction) {
	// TODO #12120: notify client

	payload := SessionUpdate{
		NewEntries: []Entry{
			{
				payloadType: PendingTransactionPT,
				transaction: &transfer.TransactionIdentity{
					ChainID: tx.ChainID,
					Hash:    tx.Hash,
				},
				id: transfer.NoMultiTransactionID,
				// TODO: transfer tx details
			},
		},
	}

	sendResponseEvent(eventFeed, (*int32)(&session.id), EventActivitySessionUpdated, payload, nil)
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
