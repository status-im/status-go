package swap

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/protocol"
	"go.uber.org/zap"
)

const (
	SoftMode int = 0
	MockMode int = 1
	HardMode int = 2
)

const WakuSwapID_v200 = protocol.ID("/vac/waku/swap/2.0.0-beta1")

type WakuSwap struct {
	params *SwapParameters

	log *zap.SugaredLogger

	Accounting      map[string]int
	accountingMutex sync.RWMutex
}

func NewWakuSwap(log *zap.SugaredLogger, opts ...SwapOption) *WakuSwap {
	params := &SwapParameters{}

	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	return &WakuSwap{
		params:     params,
		log:        log.Named("swap"),
		Accounting: make(map[string]int),
	}
}

func (s *WakuSwap) sendCheque(peerId string) {
	s.log.Debug("not yet implemented")
}

func (s *WakuSwap) applyPolicy(peerId string) {
	if s.Accounting[peerId] <= s.params.disconnectThreshold {
		s.log.Warnf("Disconnect threshhold has been reached for %s at %d", peerId, s.Accounting[peerId])
	}

	if s.Accounting[peerId] >= s.params.paymentThreshold {
		s.log.Warnf("Disconnect threshhold has been reached for %s at %d", peerId, s.Accounting[peerId])
		if s.params.mode != HardMode {
			s.sendCheque(peerId)
		}
	}
}

func (s *WakuSwap) Credit(peerId string, n int) {
	s.accountingMutex.Lock()
	defer s.accountingMutex.Unlock()

	s.Accounting[peerId] -= n
	s.applyPolicy(peerId)
}

func (s *WakuSwap) Debit(peerId string, n int) {
	s.accountingMutex.Lock()
	defer s.accountingMutex.Unlock()

	s.Accounting[peerId] += n
	s.applyPolicy(peerId)
}
