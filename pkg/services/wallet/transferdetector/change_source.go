package transferdetector

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/balance/refresh"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/pubsub"

	"go.uber.org/zap"
)

// KnownTokensProvider gives the tokens whose transfers are kept: the token
// lists and custom tokens. Transfers of other tokens are spam.
type KnownTokensProvider interface {
	GetTokenContractAddresses(chainID uint64) ([]common.Address, error)
}

// ChangeSource tells the balance refresh what transfer detection finds:
// transfers of known tokens and native transfer logs, and the ranges it
// failed to look at as detection gaps.
type ChangeSource struct {
	controller *Controller
	tokens     KnownTokensProvider
	logger     *zap.Logger
}

var _ refresh.ChangeSource = (*ChangeSource)(nil)

func NewChangeSource(controller *Controller, tokens KnownTokensProvider, logger *zap.Logger) *ChangeSource {
	return &ChangeSource{controller: controller, tokens: tokens, logger: logger}
}

func (s *ChangeSource) Scan() {
	s.controller.TriggerFetch()
}

func (s *ChangeSource) Watch(emit func(refresh.Event)) (stop func()) {
	finishedCh, unsubFinished := pubsub.Subscribe[EventTransferDetectionFinished](s.controller.GetPublisher(), 100)
	failedCh, unsubFailed := pubsub.Subscribe[EventTransferDetectionError](s.controller.GetPublisher(), 100)
	done := make(chan struct{})
	go func() {
		defer panics.LogOnPanic()
		for {
			select {
			case <-done:
				return
			case ev, ok := <-finishedCh:
				if !ok {
					return
				}
				for _, transfer := range s.transfers(ev) {
					emit(transfer)
				}
			case ev, ok := <-failedCh:
				if !ok {
					return
				}
				emit(refresh.DetectionGap{Chain: ev.ChainID})
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			unsubFinished()
			unsubFailed()
		})
	}
}

func (s *ChangeSource) transfers(ev EventTransferDetectionFinished) []refresh.TransferDetected {
	if len(ev.Events) == 0 {
		return nil
	}
	tokens, err := s.tokens.GetTokenContractAddresses(ev.ChainID)
	if err != nil {
		s.logger.Error("failed to get the tokens transfers are kept for", zap.Uint64("chainID", ev.ChainID), zap.Error(err))
	}
	known := make(map[common.Address]struct{}, len(tokens))
	for _, token := range tokens {
		known[token] = struct{}{}
	}
	return refresh.TransfersFromLogs(ev.ChainID, ev.Accounts, ev.Events, func(token common.Address) bool {
		_, ok := known[token]
		return ok
	})
}
