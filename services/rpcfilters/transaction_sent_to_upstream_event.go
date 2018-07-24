package rpcfilters

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// transactionSentToUpstreamEvent represents an event that one can subscribe to
type transactionSentToUpstreamEvent struct {
	sxMu     sync.Mutex
	sx       map[int]chan common.Hash
	listener chan common.Hash
	quit     chan struct{}
}

func newTransactionSentToUpstreamEvent() *transactionSentToUpstreamEvent {
	return &transactionSentToUpstreamEvent{
		sx: make(map[int]chan common.Hash),
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

func (e *transactionSentToUpstreamEvent) processTransactionSentToUpstream(transactionHash common.Hash) {

	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	for _, channel := range e.sx {
		channel <- transactionHash
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
}

func (e *transactionSentToUpstreamEvent) Subscribe() (int, chan common.Hash) {
	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	channel := make(chan common.Hash)
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
func (e *transactionSentToUpstreamEvent) Trigger(transactionHash common.Hash) {
	e.listener <- transactionHash
}
