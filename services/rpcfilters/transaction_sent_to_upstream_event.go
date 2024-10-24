package rpcfilters

import (
	"errors"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
)

type PendingTxInfo struct {
	Hash    common.Hash
	Type    string
	From    common.Address
	ChainID uint64
}

// transactionSentToUpstreamEvent represents an event that one can subscribe to
type transactionSentToUpstreamEvent struct {
	sxMu     sync.Mutex
	sx       map[int]chan *PendingTxInfo
	listener chan *PendingTxInfo
	quit     chan struct{}
}

func newTransactionSentToUpstreamEvent() *transactionSentToUpstreamEvent {
	return &transactionSentToUpstreamEvent{
		sx:       make(map[int]chan *PendingTxInfo),
		listener: make(chan *PendingTxInfo),
	}
}

func (e *transactionSentToUpstreamEvent) Start() error {
	if e.quit != nil {
		return errors.New("latest transaction sent to upstream event is already started")
	}

	e.quit = make(chan struct{})

	go func() {
		defer gocommon.LogOnPanic()
		for {
			select {
			case transactionInfo := <-e.listener:
				if e.numberOfSubscriptions() == 0 {
					continue
				}
				e.processTransactionSentToUpstream(transactionInfo)
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

func (e *transactionSentToUpstreamEvent) processTransactionSentToUpstream(transactionInfo *PendingTxInfo) {

	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	for id, channel := range e.sx {
		select {
		case channel <- transactionInfo:
		default:
			logutils.ZapLogger().Error("dropping messages because the channel is full", zap.Any("transactionInfo", transactionInfo), zap.Int("id", id))
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

func (e *transactionSentToUpstreamEvent) Subscribe() (int, interface{}) {
	e.sxMu.Lock()
	defer e.sxMu.Unlock()

	channel := make(chan *PendingTxInfo, 512)
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
func (e *transactionSentToUpstreamEvent) Trigger(transactionInfo *PendingTxInfo) {
	e.listener <- transactionInfo
}
