package mailservers

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
)

// NewLastUsedConnectionMonitor returns pointer to the instance of LastUsedConnectionMonitor.
func NewLastUsedConnectionMonitor(ps *PeerStore, cache *Cache, eventSub EnvelopeEventSubscriber) *LastUsedConnectionMonitor {
	return &LastUsedConnectionMonitor{
		ps:       ps,
		cache:    cache,
		eventSub: eventSub,
	}
}

// LastUsedConnectionMonitor watches relevant events and reflects it in cache.
type LastUsedConnectionMonitor struct {
	ps    *PeerStore
	cache *Cache

	eventSub EnvelopeEventSubscriber

	quit chan struct{}
	wg   sync.WaitGroup
}

// Start spins a separate goroutine to watch connections.
func (mon *LastUsedConnectionMonitor) Start() {
	mon.quit = make(chan struct{})
	mon.wg.Add(1)
	go func() {
		defer common.LogOnPanic()
		events := make(chan types.EnvelopeEvent, whisperEventsBuffer)
		sub := mon.eventSub.SubscribeEnvelopeEvents(events)
		defer sub.Unsubscribe()
		defer mon.wg.Done()
		for {
			select {
			case <-mon.quit:
				return
			case err := <-sub.Err():
				logutils.ZapLogger().Error("retry after error suscribing to eventSub events", zap.Error(err))
				return
			case ev := <-events:
				node := mon.ps.Get(ev.Peer)
				if node == nil {
					continue
				}
				if ev.Event == types.EventMailServerRequestCompleted {
					err := mon.updateRecord(ev.Peer)
					if err != nil {
						logutils.ZapLogger().Error("unable to update storage", zap.Stringer("peer", ev.Peer), zap.Error(err))
					}
				}
			}
		}
	}()
}

func (mon *LastUsedConnectionMonitor) updateRecord(nodeID types.EnodeID) error {
	node := mon.ps.Get(nodeID)
	if node == nil {
		return nil
	}
	return mon.cache.UpdateRecord(PeerRecord{node: node, LastUsed: time.Now()})
}

// Stop closes channel to signal a quit and waits until all goroutines are stoppped.
func (mon *LastUsedConnectionMonitor) Stop() {
	if mon.quit == nil {
		return
	}
	select {
	case <-mon.quit:
		return
	default:
	}
	close(mon.quit)
	mon.wg.Wait()
	mon.quit = nil
}
