package mailservers

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
)

// NewLastUsedConnectionMonitor returns pointer to the instance of LastUsedConnectionMonitor.
func NewLastUsedConnectionMonitor(ps *PeerStore, cache *Cache, whisper EnvelopeEventSubscriber) *LastUsedConnectionMonitor {
	return &LastUsedConnectionMonitor{
		ps:      ps,
		cache:   cache,
		whisper: whisper,
	}
}

// LastUsedConnectionMonitor watches relevant events and reflects it in cache.
type LastUsedConnectionMonitor struct {
	ps    *PeerStore
	cache *Cache

	whisper EnvelopeEventSubscriber

	quit chan struct{}
	wg   sync.WaitGroup
}

// Start spins a separate goroutine to watch connections.
func (mon *LastUsedConnectionMonitor) Start() {
	mon.quit = make(chan struct{})
	mon.wg.Add(1)
	go func() {
		events := make(chan whispertypes.EnvelopeEvent, whisperEventsBuffer)
		sub := mon.whisper.SubscribeEnvelopeEvents(events)
		defer sub.Unsubscribe()
		defer mon.wg.Done()
		for {
			select {
			case <-mon.quit:
				return
			case err := <-sub.Err():
				log.Error("retry after error suscribing to whisper events", "error", err)
				return
			case ev := <-events:
				node := mon.ps.Get(ev.Peer)
				if node == nil {
					continue
				}
				if ev.Event == whispertypes.EventMailServerRequestCompleted {
					err := mon.updateRecord(ev.Peer)
					if err != nil {
						log.Error("unable to update storage", "peer", ev.Peer, "error", err)
					}
				}
			}
		}
	}()
}

func (mon *LastUsedConnectionMonitor) updateRecord(nodeID whispertypes.EnodeID) error {
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
