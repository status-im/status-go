package collectibles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc/network"

	"github.com/status-im/status-go/services/wallet/async"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// These events are used to notify the UI of state changes
const (
	EventCollectiblesOwnershipUpdateStarted           walletevent.EventType = "wallet-collectibles-ownership-update-started"
	EventCollectiblesOwnershipUpdatePartial           walletevent.EventType = "wallet-collectibles-ownership-update-partial"
	EventCollectiblesOwnershipUpdateFinished          walletevent.EventType = "wallet-collectibles-ownership-update-finished"
	EventCollectiblesOwnershipUpdateFinishedWithError walletevent.EventType = "wallet-collectibles-ownership-update-finished-with-error"
	EventCommunityCollectiblesReceived                walletevent.EventType = "wallet-collectibles-community-collectibles-received"

	EventOwnedCollectiblesFilteringDone walletevent.EventType = "wallet-owned-collectibles-filtering-done"
	EventGetCollectiblesDetailsDone     walletevent.EventType = "wallet-get-collectibles-details-done"
)

var (
	filterOwnedCollectiblesTask = async.TaskType{
		ID:     1,
		Policy: async.ReplacementPolicyCancelOld,
	}
	getCollectiblesDataTask = async.TaskType{
		ID:     2,
		Policy: async.ReplacementPolicyCancelOld,
	}
)

type Service struct {
	manager     *Manager
	controller  *Controller
	ownershipDB *OwnershipDB
	walletFeed  *event.Feed
	scheduler   *async.MultiClientScheduler
}

func NewService(
	db *sql.DB,
	walletFeed *event.Feed,
	accountsDB *accounts.Database,
	accountsFeed *event.Feed,
	settingsFeed *event.Feed,
	networkManager *network.Manager,
	manager *Manager) *Service {
	return &Service{
		manager:     manager,
		controller:  NewController(db, walletFeed, accountsDB, accountsFeed, settingsFeed, networkManager, manager),
		ownershipDB: NewOwnershipDB(db),
		walletFeed:  walletFeed,
		scheduler:   async.NewMultiClientScheduler(),
	}
}

type ErrorCode = int

const (
	ErrorCodeSuccess ErrorCode = iota + 1
	ErrorCodeTaskCanceled
	ErrorCodeFailed
)

type OwnershipStatus struct {
	State     OwnershipState `json:"state"`
	Timestamp int64          `json:"timestamp"`
}

type OwnershipStatusPerChainID = map[walletCommon.ChainID]OwnershipStatus
type OwnershipStatusPerAddressAndChainID = map[common.Address]OwnershipStatusPerChainID

type FilterOwnedCollectiblesResponse struct {
	Collectibles []CollectibleHeader `json:"collectibles"`
	Offset       int                 `json:"offset"`
	// Used to indicate that there might be more collectibles that were not returned
	// based on a simple heuristic
	HasMore         bool                                `json:"hasMore"`
	OwnershipStatus OwnershipStatusPerAddressAndChainID `json:"ownershipStatus"`
	ErrorCode       ErrorCode                           `json:"errorCode"`
}

type GetCollectiblesDetailsResponse struct {
	Collectibles []CollectibleDetails `json:"collectibles"`
	ErrorCode    ErrorCode            `json:"errorCode"`
}

type filterOwnedCollectiblesTaskReturnType struct {
	headers         []CollectibleHeader
	hasMore         bool
	ownershipStatus OwnershipStatusPerAddressAndChainID
}

// FilterOwnedCollectiblesResponse allows only one filter task to run at a time
// and it cancels the current one if a new one is started
// All calls will trigger an EventOwnedCollectiblesFilteringDone event with the result of the filtering
func (s *Service) FilterOwnedCollectiblesAsync(requestID int32, chainIDs []walletCommon.ChainID, addresses []common.Address, offset int, limit int) {
	s.scheduler.Enqueue(requestID, filterOwnedCollectiblesTask, func(ctx context.Context) (interface{}, error) {
		collectibles, hasMore, err := s.GetOwnedCollectibles(chainIDs, addresses, offset, limit)
		if err != nil {
			return nil, err
		}
		data, err := s.manager.FetchAssetsByCollectibleUniqueID(collectibles)
		if err != nil {
			return nil, err
		}
		ownershipStatus, err := s.GetOwnershipStatus(chainIDs, addresses)
		if err != nil {
			return nil, err
		}
		headers, err := s.fullCollectiblesDataToHeaders(data)

		return filterOwnedCollectiblesTaskReturnType{
			headers:         headers,
			hasMore:         hasMore,
			ownershipStatus: ownershipStatus,
		}, err
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := FilterOwnedCollectiblesResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			fnRet := result.(filterOwnedCollectiblesTaskReturnType)
			res.Collectibles = fnRet.headers
			res.Offset = offset
			res.HasMore = fnRet.hasMore
			res.OwnershipStatus = fnRet.ownershipStatus
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(&requestID, EventOwnedCollectiblesFilteringDone, res, err)
	})
}

func (s *Service) GetCollectiblesDetailsAsync(requestID int32, uniqueIDs []thirdparty.CollectibleUniqueID) {
	s.scheduler.Enqueue(requestID, getCollectiblesDataTask, func(ctx context.Context) (interface{}, error) {
		collectibles, err := s.manager.FetchAssetsByCollectibleUniqueID(uniqueIDs)
		if err != nil {
			return nil, err
		}
		return s.fullCollectiblesDataToDetails(collectibles)
	}, func(result interface{}, taskType async.TaskType, err error) {
		res := GetCollectiblesDetailsResponse{
			ErrorCode: ErrorCodeFailed,
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if err == nil {
			res.Collectibles = result.([]CollectibleDetails)
			res.ErrorCode = ErrorCodeSuccess
		}

		s.sendResponseEvent(&requestID, EventGetCollectiblesDetailsDone, res, err)
	})
}

func (s *Service) RefetchOwnedCollectibles() {
	s.controller.RefetchOwnedCollectibles()
}

func (s *Service) Start() {
	s.controller.Start()
}

func (s *Service) Stop() {
	s.controller.Stop()

	s.scheduler.Stop()
}

func (s *Service) sendResponseEvent(requestID *int32, eventType walletevent.EventType, payloadObj interface{}, resErr error) {
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		log.Error("Error marshaling response: %v; result error: %w", err, resErr)
	} else {
		err = resErr
	}

	log.Debug("wallet.api.collectibles.Service RESPONSE", "requestID", requestID, "eventType", eventType, "error", err, "payload.len", len(payload))

	event := walletevent.Event{
		Type:    eventType,
		Message: string(payload),
	}

	if requestID != nil {
		event.RequestID = new(int)
		*event.RequestID = int(*requestID)
	}

	s.walletFeed.Send(event)
}

func (s *Service) GetOwnedCollectibles(chainIDs []walletCommon.ChainID, owners []common.Address, offset int, limit int) ([]thirdparty.CollectibleUniqueID, bool, error) {
	// Request one more than limit, to check if DB has more available
	ids, err := s.ownershipDB.GetOwnedCollectibles(chainIDs, owners, offset, limit+1)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(ids) > limit
	if hasMore {
		ids = ids[:limit]
	}

	return ids, hasMore, nil
}

func (s *Service) GetOwnedCollectible(chainID walletCommon.ChainID, owner common.Address, contractAddress common.Address, tokenID *big.Int) (*thirdparty.CollectibleUniqueID, error) {
	return s.ownershipDB.GetOwnedCollectible(chainID, owner, contractAddress, tokenID)
}

func (s *Service) GetOwnershipStatus(chainIDs []walletCommon.ChainID, owners []common.Address) (OwnershipStatusPerAddressAndChainID, error) {
	ret := make(OwnershipStatusPerAddressAndChainID)
	for _, address := range owners {
		ret[address] = make(OwnershipStatusPerChainID)
		for _, chainID := range chainIDs {
			timestamp, err := s.ownershipDB.GetOwnershipUpdateTimestamp(address, chainID)
			if err != nil {
				return nil, err
			}
			ret[address][chainID] = OwnershipStatus{
				State:     s.controller.GetCommandState(chainID, address),
				Timestamp: timestamp,
			}
		}
	}

	return ret, nil
}

func (s *Service) fullCollectiblesDataToHeaders(data []thirdparty.FullCollectibleData) ([]CollectibleHeader, error) {
	res := make([]CollectibleHeader, 0, len(data))

	for _, c := range data {
		header := fullCollectibleDataToHeader(c)

		if c.CollectibleData.CommunityID != "" {
			communityInfo, err := s.manager.FetchCollectibleCommunityInfo(c.CollectibleData.CommunityID, c.CollectibleData.ID)
			if err != nil {
				return nil, err
			}

			communityHeader := communityInfoToHeader(*communityInfo)
			header.CommunityHeader = &communityHeader
		}

		res = append(res, header)
	}

	return res, nil
}

func (s *Service) fullCollectiblesDataToDetails(data []thirdparty.FullCollectibleData) ([]CollectibleDetails, error) {
	res := make([]CollectibleDetails, 0, len(data))

	for _, c := range data {
		details := fullCollectibleDataToDetails(c)

		if c.CollectibleData.CommunityID != "" {
			traits, err := s.manager.FetchCollectibleCommunityTraits(c.CollectibleData.CommunityID, c.CollectibleData.ID)
			if err != nil {
				return nil, err
			}
			details.Traits = traits

			communityInfo, err := s.manager.FetchCollectibleCommunityInfo(c.CollectibleData.CommunityID, c.CollectibleData.ID)
			if err != nil {
				return nil, err
			}

			details.CommunityInfo = communityInfo
		}

		res = append(res, details)
	}

	return res, nil
}
