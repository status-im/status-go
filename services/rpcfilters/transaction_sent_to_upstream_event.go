package rpcfilters

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/eth-node/types"
)

// transactionSentToUpstreamEvent represents an event that one can subscribe to
type transactionSentToUpstreamEvent struct {
	sxMu     sync.Mutex
	sx       map[int]chan types.Hash
	listener chan types.Hash
	quit     chan struct{}
}

func newTransactionSentToUpstreamEvent() *transactionSentToUpstreamEvent {
	return &transactionSentToUpstreamEvent{
		sx:       make(map[int]chan types.Hash),
		listener: make(chan types.Hash),
	}
}

func (e *transactionSentToUpstreamEvent) Start() error {
	if e.quit != nil {
		return errors.New("latest transaction sent to upstream event is already started")
	}

	e.quit = make(chan struct{})

	go func() {
		for {
			select {
			case transactionHash := <-e.listener:
				if e.numberOfSubscriptions() == 0 {
					continue
				}
				e.processTransactionSentToUpstream(transactionHash)
			case <-e.quit:
				return
			}
		}
	}()

	return nil
}

func (e *transactionSentToUpstreamEvent) numberOfSubscriptions() int {
	e.sxMu.Lock()
	defer e.sxMu.Unlock()
	return len(e.sx)
}

func (e *transactionSentToUpstreamEvent) processTransactionSentToUpstream(transactionHash types.Hash) {

	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	for id, channel := range e.sx {
		select {
		case channel <- transactionHash:
		default:
			log.Error("dropping messages %s for subscriotion %d because the channel is full", transactionHash, id)
		}
	}
}

func (e *transactionSentToUpstreamEvent) Stop() {
	if e.quit == nil {
		return
	}

	select {
	case <-e.quit:
		return
	default:
		close(e.quit)
	}

	e.quit = nil
}

func (e *transactionSentToUpstreamEvent) Subscribe() (int, chan types.Hash) {
	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	channel := make(chan types.Hash, 512)
	id := len(e.sx)
	e.sx[id] = channel
	return id, channel
}

func (e *transactionSentToUpstreamEvent) Unsubscribe(id int) {
	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	delete(e.sx, id)
}

// Trigger gets called in order to trigger the event
func (e *transactionSentToUpstreamEvent) Trigger(transactionHash types.Hash) {
	e.listener <- transactionHash
}
