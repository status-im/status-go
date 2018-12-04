package mailservers

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/whisper/whisperv6"
)

// NewLastUsedConnectionMonitor returns pointer to the instance of LastUsedConnectionMonitor.
func NewLastUsedConnectionMonitor(ps *PeerStore, cache *Cache, whisper EnvelopeEventSubscbriber) *LastUsedConnectionMonitor {
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

	whisper EnvelopeEventSubscbriber

	quit chan struct{}
	wg   sync.WaitGroup
}

// Start spins a separate goroutine to watch connections.
func (mon *LastUsedConnectionMonitor) Start() {
	mon.quit = make(chan struct{})
	mon.wg.Add(1)
	go func() {
		events := make(chan whisperv6.EnvelopeEvent, whisperEventsBuffer)
		sub := mon.whisper.SubscribeEnvelopeEvents(events)
		for {
			select {
			case <-mon.quit:
				sub.Unsubscribe()
				mon.wg.Done()
				return
			case err := <-sub.Err():
				log.Error("retry after error suscribing to whisper events", "error", err)
				sub = mon.whisper.SubscribeEnvelopeEvents(events)
			case ev := <-events:
				node := mon.ps.Get(ev.Peer)
				if node == nil {
					continue
				}
				if ev.Event == whisperv6.EventMailServerRequestCompleted {
					err := mon.updateRecord(ev.Peer)
					if err != nil {
						log.Error("unable to store that server was used", "peer", ev.Peer, "error", err)
					}
				}
			}
		}
	}()
}

func (mon *LastUsedConnectionMonitor) updateRecord(nodeID enode.ID) error {
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
