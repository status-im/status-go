package collectibles

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/circuitbreaker"
	"github.com/status-im/status-go/internal/healthmanager/provider_errors"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/services/wallet/collectibles/ownership"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const EventCollectiblesConnectionStatusChanged walletevent.EventType = "wallet-collectible-status-changed"

// Reset connection status to trigger notifications
// on the next status update
func (o *Manager) ResetConnectionStatus() {
	o.statuses.Range(func(key, value interface{}) bool {
		value.(*connection.Status).ResetStateValue()
		return true
	})
}

func (o *Manager) recordChainOutcome(chainID walletCommon.ChainID, err error) {
	if o.statuses == nil {
		return
	}
	if err == nil {
		o.setChainConnected(chainID, true)
		return
	}
	if isCollectiblesIgnorableError(err) || errors.Is(err, ErrNoProvidersAvailableForChainID) {
		return
	}
	o.setChainConnected(chainID, false)
}

func (o *Manager) applyCallStatuses(chainID walletCommon.ChainID, statuses []circuitbreaker.FunctorCallStatus) {
	if len(statuses) == 0 {
		return
	}
	var firstErr error
	allIgnored := true
	for _, s := range statuses {
		if s.Err != nil && isCollectiblesIgnorableError(s.Err) {
			continue
		}
		allIgnored = false
		if s.Err == nil {
			o.recordChainOutcome(chainID, nil)
			return
		}
		if firstErr == nil {
			firstErr = s.Err
		}
	}
	if allIgnored {
		return
	}
	o.recordChainOutcome(chainID, firstErr)
}

func (o *Manager) setChainConnected(chainID walletCommon.ChainID, connected bool) {
	if o.statuses == nil {
		return
	}
	key := chainID.String()
	if v, ok := o.statuses.Load(key); ok {
		v.(*connection.Status).SetIsConnected(connected)
		return
	}
	st := connection.NewStatus()
	st.SetIsConnected(connected)
	if actual, loaded := o.statuses.LoadOrStore(key, st); loaded {
		actual.(*connection.Status).SetIsConnected(connected)
	} else {
		o.updateStatusNotifier()
	}
}

func isCollectiblesIgnorableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	if provider_errors.IsNonCriticalRpcError(err) {
		return true
	}
	if provider_errors.IsNonCriticalProviderError(err) {
		return true
	}
	return false
}

func logProviderSearchErr(method, providerID string, chainID walletCommon.ChainID, err error) {
	logutils.ZapLogger().Error("collectibles search request failed",
		zap.String("method", method),
		zap.String("provider", providerID),
		zap.Stringer("chainID", chainID),
		zap.Error(err),
	)
}

func (o *Manager) updateStatusNotifier() {
	o.statusNotifier = createStatusNotifier(o.statuses, o.feed)
}

func initStatuses(ownershipDB ownership.OwnershipStorage) *sync.Map {
	statuses := &sync.Map{}
	for _, chainID := range walletCommon.AllChainIDs() {
		status := connection.NewStatus()
		state := status.GetState()
		latestUpdateTimestamp, err := ownershipDB.GetLatestOwnershipUpdateTimestamp(chainID)
		if err == nil {
			state.LastSuccessAt = latestUpdateTimestamp
			status.SetState(state)
		}
		statuses.Store(chainID.String(), status)
	}

	return statuses
}

func createStatusNotifier(statuses *sync.Map, feed *event.Feed) *connection.StatusNotifier {
	return connection.NewStatusNotifier(
		statuses,
		EventCollectiblesConnectionStatusChanged,
		feed,
	)
}
