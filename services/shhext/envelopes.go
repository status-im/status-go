package shhext

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/shhext/mailservers"
	whisper "github.com/status-im/whisper/whisperv6"
)

type deliveryState struct {
	state    EnvelopeState
	expired  int
	envelope *whisper.Envelope
}

// EnvelopesMonitor monitors state of the envelopes delivery.
type EnvelopesMonitor struct {
	config params.MessageResendConfig

	w                      *whisper.Whisper
	handler                EnvelopeEventsHandler
	mailServerConfirmation bool

	mu      sync.Mutex
	cache   map[common.Hash]*deliveryState
	batches map[common.Hash]map[common.Hash]struct{}

	mailPeers *mailservers.PeerStore

	wg   sync.WaitGroup
	quit chan struct{}
}

// Add hash to a tracker.
func (mon *EnvelopesMonitor) Add(envelope *whisper.Envelope) {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	mon.cache[envelope.Hash()] = &deliveryState{
		state:    EnvelopePosted,
		envelope: envelope,
	}
}

func (mon *EnvelopesMonitor) GetState(hash common.Hash) EnvelopeState {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	state, exist := mon.cache[hash]
	if !exist {
		return NotRegistered
	}
	return state.state
}

// Start processing events.
func (mon *EnvelopesMonitor) Start() {
	mon.quit = make(chan struct{})
	mon.wg.Add(1)
	go func() {
		mon.monitor()
		mon.wg.Done()
	}()
}

// Stop process events.
func (mon *EnvelopesMonitor) Stop() {
	close(mon.quit)
	mon.wg.Wait()
}

func (mon *EnvelopesMonitor) monitor() {
	events := make(chan whisper.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	sub := mon.w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	for {
		select {
		case <-mon.quit:
			return
		case event := <-events:
			mon.process(event)
		}
	}
}

func (mon *EnvelopesMonitor) process(event whisper.EnvelopeEvent) {
	handlers := map[whisper.EventType]func(whisper.EnvelopeEvent){
		whisper.EventEnvelopeSent:      mon.handleEventEnvelopeSent,
		whisper.EventEnvelopeExpired:   mon.handleEventEnvelopeExpired,
		whisper.EventBatchAcknowledged: mon.handleAcknowledgedBatch,
	}

	if handler, ok := handlers[event.Event]; ok {
		handler(event)
	}
}

func (mon *EnvelopesMonitor) handleEventEnvelopeSent(event whisper.EnvelopeEvent) {
	if mon.mailServerConfirmation {
		if !mon.isMailserver(event.Peer) {
			return
		}
	}

	mon.mu.Lock()
	defer mon.mu.Unlock()

	state, ok := mon.cache[event.Hash]
	// if we didn't send a message using extension - skip it
	// if message was already confirmed - skip it
	if !ok || state.state == EnvelopeSent {
		return
	}
	log.Debug("envelope is sent", "hash", event.Hash, "peer", event.Peer)
	if event.Batch != (common.Hash{}) {
		log.Debug("envelope sent. waiting for a confirmation", "batch", event.Batch, "peer", event.Peer)
		if _, ok := mon.batches[event.Batch]; !ok {
			mon.batches[event.Batch] = map[common.Hash]struct{}{}
		}
		mon.batches[event.Batch][event.Hash] = struct{}{}
	} else {
		log.Debug("envelope sent without confirmation", "hash", event.Hash, "peer", event.Peer)
		mon.cache[event.Hash].state = EnvelopeSent
		if mon.handler != nil {
			mon.handler.EnvelopeSent(event.Hash)
		}
	}
	// FIXME with config
	time.AfterFunc(mon.config.BaseTimeout, func() {
		mon.checkResendEnvelope(event.Hash)
	})
}

func (mon *EnvelopesMonitor) checkResendEnvelope(hash common.Hash) {
	mon.mu.Lock()
	defer mon.mu.Unlock()
	state, ok := mon.cache[hash]
	if !ok || state.state == EnvelopeSent {
		return
	}
	state.expired++
	if state.expired == mon.config.MaxRetries {
		log.Debug("maximum number of retries reached", "hash", hash)
	}
	log.Debug("envelope was sent but delivery expired", "hash", hash, "retry count", state.expired)
	err := mon.w.Send(state.envelope)
	if err != nil {
		log.Error("failed to resend envelope. envelope will eventually expire", "hash", hash)
		return
	}
	time.AfterFunc(mon.config.BaseTimeout+mon.config.StepTimeout*time.Duration(state.expired), func() {
		mon.checkResendEnvelope(hash)
	})

}

func (mon *EnvelopesMonitor) isMailserver(peer enode.ID) bool {
	return mon.mailPeers.Exist(peer)
}

func (mon *EnvelopesMonitor) handleAcknowledgedBatch(event whisper.EnvelopeEvent) {
	if mon.mailServerConfirmation {
		if !mon.isMailserver(event.Peer) {
			return
		}
	}

	mon.mu.Lock()
	defer mon.mu.Unlock()
	envelopes, ok := mon.batches[event.Batch]
	if !ok {
		log.Debug("batch is not found", "batch", event.Batch)
	}
	log.Debug("received a confirmation", "batch", event.Batch, "peer", event.Peer)
	for hash := range envelopes {
		state, ok := mon.cache[hash]
		if !ok || state.state == EnvelopeSent {
			continue
		}
		mon.cache[hash].state = EnvelopeSent
		if mon.handler != nil {
			mon.handler.EnvelopeSent(hash)
		}
	}
	delete(mon.batches, event.Batch)
}

func (mon *EnvelopesMonitor) handleEventEnvelopeExpired(event whisper.EnvelopeEvent) {
	mon.mu.Lock()
	defer mon.mu.Unlock()

	if state, ok := mon.cache[event.Hash]; ok {
		if state.state != EnvelopeSent {
			log.Debug("failed delivery. envelope removed from a queue", "hash", event.Hash, "retry count", state.expired)
			if mon.handler != nil {
				mon.handler.EnvelopeExpired(event.Hash)
			}
		}
		delete(mon.cache, event.Hash)
	}
}
