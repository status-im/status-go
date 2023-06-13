package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	// FilterResponse json is sent as a message in the EventActivityFilteringDone event
	EventActivityFilteringDone walletevent.EventType = "wallet-activity-filtering-done"
)

type Service struct {
	db           *sql.DB
	tokenManager *token.Manager
	eventFeed    *event.Feed

	context     context.Context
	cancelFn    context.CancelFunc
	wg          sync.WaitGroup
	cancelMutex sync.Mutex
}

func NewService(db *sql.DB, tokenManager *token.Manager, eventFeed *event.Feed) *Service {
	return &Service{
		db:           db,
		tokenManager: tokenManager,
		eventFeed:    eventFeed,
	}
}

type ErrorCode = int

const (
	ErrorCodeSuccess ErrorCode = iota + 1
	ErrorCodeFilterCanceled
	ErrorCodeFilterFailed
)

type FilterResponse struct {
	Activities []Entry `json:"activities"`
	Offset     int     `json:"offset"`
	// Used to indicate that there might be more entries that were not returned
	// based on a simple heuristic
	HasMore   bool      `json:"hasMore"`
	ErrorCode ErrorCode `json:"errorCode"`
}

// FilterActivityAsync allows only one filter task to run at a time
// and it cancels the current one if a new one is started
// All calls will trigger an EventActivityFilteringDone event with the result of the filtering
func (s *Service) FilterActivityAsync(ctx context.Context, addresses []common.Address, chainIDs []w_common.ChainID, filter Filter, offset int, limit int) error {
	s.cancelMutex.Lock()
	defer s.cancelMutex.Unlock()

	// If a previous task is running, cancel it and wait to finish
	if s.cancelFn != nil {
		s.cancelFn()
		s.cancelFn = nil
		s.wg.Wait()
	}

	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	s.context, s.cancelFn = context.WithCancel(context.Background())
	thisCancelFn := s.cancelFn

	s.wg.Add(1)

	go func() {
		activities, err := getActivityEntries(s.context, s.getDeps(), addresses, chainIDs, filter, offset, limit)
		// Don't lock s.cancelMutex, it might have been locked already
		s.wg.Done()

		// Release context resources
		thisCancelFn()

		res := FilterResponse{
			ErrorCode: ErrorCodeFilterFailed,
		}

		if errors.Is(err, context.Canceled) {
			res.ErrorCode = ErrorCodeFilterCanceled
		} else if err == nil {
			res.Activities = activities
			res.Offset = offset
			res.HasMore = len(activities) == limit
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(res)
	}()

	return nil
}

func (s *Service) Stop() {
	s.cancelMutex.Lock()
	defer s.cancelMutex.Unlock()

	// If a previous task is running, cancel it and wait to finish
	if s.cancelFn != nil {
		s.cancelFn()
		s.wg.Wait()
		s.cancelFn = nil
	}
}

func (s *Service) getDeps() FilterDependencies {
	return FilterDependencies{
		db: s.db,
		tokenSymbol: func(t Token) string {
			info := s.tokenManager.LookupTokenIdentity(uint64(t.ChainID), t.Address, t.TokenType == Native)
			if info == nil {
				return ""
			}
			return info.Symbol
		},
		tokenFromSymbol: func(chainID *w_common.ChainID, symbol string) *Token {
			var cID *uint64
			if chainID != nil {
				cID = new(uint64)
				*cID = uint64(*chainID)
			}
			t, detectedNative := s.tokenManager.LookupToken(cID, symbol)
			if t == nil {
				return nil
			}
			tokenType := Native
			if !detectedNative {
				tokenType = Erc20
			}
			return &Token{
				TokenType: tokenType,
				ChainID:   w_common.ChainID(t.ChainID),
				Address:   t.Address,
			}
		},
	}
}

func (s *Service) sendResponseEvent(response FilterResponse) {
	payload, err := json.Marshal(response)
	if err != nil {
		log.Error("Error marshaling response: %v", err)
	}

	log.Debug("wallet.api.FilterActivityAsync RESPONSE", "activities.len", len(response.Activities), "offset", response.Offset, "hasMore", response.HasMore, "error", response.ErrorCode)

	s.eventFeed.Send(walletevent.Event{
		Type:    EventActivityFilteringDone,
		Message: string(payload),
	})
}
