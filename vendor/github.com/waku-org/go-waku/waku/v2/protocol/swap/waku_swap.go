package swap

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/protocol"
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

	log *zap.Logger

	Accounting      map[string]int
	accountingMutex sync.RWMutex
}

func NewWakuSwap(log *zap.Logger, opts ...SwapOption) *WakuSwap {
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
	logger := s.log.With(zap.String("peer", peerId))
	if s.Accounting[peerId] <= s.params.disconnectThreshold {
		logger.Warn("disconnect threshold reached", zap.Int("value", s.Accounting[peerId]))
	}

	if s.Accounting[peerId] >= s.params.paymentThreshold {
		logger.Warn("payment threshold reached", zap.Int("value", s.Accounting[peerId]))
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

func (s *WakuSwap) Start(ctx context.Context) error {
	return nil
}

func (s *WakuSwap) Stop() {
}

func (s *WakuSwap) IsStarted() bool {
	return false
}
