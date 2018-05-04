package rpcfilters

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	defaultTickerPeriod = 3 * time.Second
)

// latestBlockChangedEvent represents an event that one can subscribe to
type latestBlockChangedEvent struct {
	sxMu sync.Mutex
	sx   map[int]chan common.Hash

	latestBlockMu       sync.Mutex
	previousLatestBlock blockInfo

	provider     latestBlockProvider
	quit         chan struct{}
	tickerPeriod time.Duration
}

func (e *latestBlockChangedEvent) Start() error {
	if e.quit != nil {
		return errors.New("latest block changed event is already started")
	}

	e.quit = make(chan struct{})

	go func() {
		ticker := time.NewTicker(e.tickerPeriod)
		for {
			select {
			case <-ticker.C:
				if e.numberOfSubscriptions() == 0 {
					continue
				}
				latestBlock, err := e.provider.GetLatestBlock()
				if err != nil {
					log.Error("error while receiving latest block", "error", err)
					continue
				}

				e.processLatestBlock(latestBlock)
			case <-e.quit:
				return
			}
		}
	}()

	return nil
}

func (e *latestBlockChangedEvent) numberOfSubscriptions() int {
	e.sxMu.Lock()
	defer e.sxMu.Unlock()
	return len(e.sx)
}

func (e *latestBlockChangedEvent) processLatestBlock(latestBlock blockInfo) {
	e.latestBlockMu.Lock()
	defer e.latestBlockMu.Unlock()

	// if we received the same or an older block than we already have, ignore it.
	if latestBlock.Number().Cmp(e.previousLatestBlock.Number()) <= 0 {
		return
	}

	e.previousLatestBlock = latestBlock

	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	for _, channel := range e.sx {
		channel <- e.previousLatestBlock.Hash
	}
}

func (e *latestBlockChangedEvent) Stop() {
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

func (e *latestBlockChangedEvent) Subscribe() (int, chan common.Hash) {
	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	channel := make(chan common.Hash)
	id := len(e.sx)
	e.sx[id] = channel
	return id, channel
}

func (e *latestBlockChangedEvent) Unsubscribe(id int) {
	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	delete(e.sx, id)
}

func newLatestBlockChangedEvent(provider latestBlockProvider) *latestBlockChangedEvent {
	return &latestBlockChangedEvent{
		sx:           make(map[int]chan common.Hash),
		provider:     provider,
		tickerPeriod: defaultTickerPeriod,
	}
}
