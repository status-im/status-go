package wakuv2

import (
	"github.com/ethereum/go-ethereum/event"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv2/common"
)

// NewWakuV2EnvelopeEventWrapper returns a wakutypes.EnvelopeEvent object that mimics Geth's EnvelopeEvent
func NewWakuV2EnvelopeEventWrapper(envelopeEvent *common.EnvelopeEvent) *types.EnvelopeEvent {
	if envelopeEvent == nil {
		panic("envelopeEvent should not be nil")
	}

	wrappedData := envelopeEvent.Data
	switch data := envelopeEvent.Data.(type) {
	case []common.EnvelopeError:
		wrappedData := make([]types.EnvelopeError, len(data))
		for index := range data {
			wrappedData[index] = *NewWakuV2EnvelopeErrorWrapper(&data[index])
		}
	}
	return &types.EnvelopeEvent{
		Event: types.EventType(envelopeEvent.Event),
		Hash:  ethtypes.Hash(envelopeEvent.Hash),
		Batch: ethtypes.Hash(envelopeEvent.Batch),
		Peer:  ethtypes.EnodeID(envelopeEvent.Peer),
		Data:  wrappedData,
	}
}

// NewWakuEnvelopeErrorWrapper returns a wakutypes.EnvelopeError object that mimics Geth's EnvelopeError
func NewWakuV2EnvelopeErrorWrapper(envelopeError *common.EnvelopeError) *types.EnvelopeError {
	if envelopeError == nil {
		panic("envelopeError should not be nil")
	}

	return &types.EnvelopeError{
		Hash:        ethtypes.Hash(envelopeError.Hash),
		Code:        mapGethErrorCode(envelopeError.Code),
		Description: envelopeError.Description,
	}
}

func mapGethErrorCode(code uint) uint {
	const (
		EnvelopeTimeNotSynced uint = iota + 1
		EnvelopeOtherError
	)
	switch code {
	case EnvelopeTimeNotSynced:
		return types.EnvelopeTimeNotSynced
	case EnvelopeOtherError:
		return types.EnvelopeOtherError
	}
	return types.EnvelopeOtherError
}

type gethSubscriptionWrapper struct {
	subscription event.Subscription
}

// NewGethSubscriptionWrapper returns an object that wraps Geth's Subscription in a types interface
func NewGethSubscriptionWrapper(subscription event.Subscription) types.Subscription {
	if subscription == nil {
		panic("subscription cannot be nil")
	}

	return &gethSubscriptionWrapper{
		subscription: subscription,
	}
}

func (w *gethSubscriptionWrapper) Err() <-chan error {
	return w.subscription.Err()
}

func (w *gethSubscriptionWrapper) Unsubscribe() {
	w.subscription.Unsubscribe()
}
