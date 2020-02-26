package bridge

import (
	"sync"
	"unsafe"

	"go.uber.org/zap"

	"github.com/status-im/status-go/waku"
	"github.com/status-im/status-go/whisper/v6"
)

type Bridge struct {
	whisper *whisper.Whisper
	waku    *waku.Waku
	logger  *zap.Logger

	cancel chan struct{}
	wg     sync.WaitGroup

	whisperIn  chan *whisper.Envelope
	whisperOut chan *whisper.Envelope
	wakuIn     chan *waku.Envelope
	wakuOut    chan *waku.Envelope
}

func New(shh *whisper.Whisper, w *waku.Waku, logger *zap.Logger) *Bridge {
	return &Bridge{
		whisper:    shh,
		waku:       w,
		logger:     logger,
		whisperOut: make(chan *whisper.Envelope),
		whisperIn:  make(chan *whisper.Envelope),
		wakuIn:     make(chan *waku.Envelope),
		wakuOut:    make(chan *waku.Envelope),
	}
}

type bridgeWhisper struct {
	*Bridge
}

func (b *bridgeWhisper) Pipe() (<-chan *whisper.Envelope, chan<- *whisper.Envelope) {
	return b.whisperOut, b.whisperIn
}

type bridgeWaku struct {
	*Bridge
}

func (b *bridgeWaku) Pipe() (<-chan *waku.Envelope, chan<- *waku.Envelope) {
	return b.wakuOut, b.wakuIn
}

func (b *Bridge) Start() {
	b.cancel = make(chan struct{})

	b.waku.RegisterBridge(&bridgeWaku{Bridge: b})
	b.whisper.RegisterBridge(&bridgeWhisper{Bridge: b})

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-b.cancel:
				return
			case env := <-b.wakuIn:
				shhEnvelope := (*whisper.Envelope)(unsafe.Pointer(env)) // nolint: gosec
				b.logger.Info("received whisper envelope from waku", zap.Any("envelope", shhEnvelope))
				b.whisperOut <- shhEnvelope
			}
		}
	}()

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-b.cancel:
				return
			case env := <-b.whisperIn:
				wakuEnvelope := (*waku.Envelope)(unsafe.Pointer(env)) // nolint: gosec
				b.logger.Info("received waku envelope from whisper", zap.Any("envelope", wakuEnvelope))
				b.wakuOut <- wakuEnvelope
			}
		}
	}()
}

func (b *Bridge) Cancel() {
	close(b.cancel)
	b.wg.Wait()
}
