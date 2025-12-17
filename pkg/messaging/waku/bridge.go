package wakuv2

import (
	"github.com/ethereum/go-ethereum/event"

	cryptotypes "github.com/status-im/status-go/crypto/types"
	common2 "github.com/status-im/status-go/pkg/messaging/waku/common"
	types2 "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// NewWakuV2EnvelopeEventWrapper returns a types.EnvelopeEvent object that mimics Geth's EnvelopeEvent
func NewWakuV2EnvelopeEventWrapper(envelopeEvent *common2.EnvelopeEvent) *types2.EnvelopeEvent {
	if envelopeEvent == nil {
		panic("envelopeEvent should not be nil")
	}

	wrappedData := envelopeEvent.Data
	switch data := envelopeEvent.Data.(type) {
	case []common2.EnvelopeError:
		wrappedData := make([]types2.EnvelopeError, len(data))
		for index := range data {
			wrappedData[index] = *NewWakuV2EnvelopeErrorWrapper(&data[index])
		}
	}
	return &types2.EnvelopeEvent{
		Event: types2.EventType(envelopeEvent.Event),
		Hash:  cryptotypes.Hash(envelopeEvent.Hash),
		Batch: cryptotypes.Hash(envelopeEvent.Batch),
		Peer:  types2.EnodeID(envelopeEvent.Peer),
		Data:  wrappedData,
	}
}

// NewWakuEnvelopeErrorWrapper returns a types.EnvelopeError object that mimics Geth's EnvelopeError
func NewWakuV2EnvelopeErrorWrapper(envelopeError *common2.EnvelopeError) *types2.EnvelopeError {
	if envelopeError == nil {
		panic("envelopeError should not be nil")
	}

	return &types2.EnvelopeError{
		Hash:        cryptotypes.Hash(envelopeError.Hash),
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
		return types2.EnvelopeTimeNotSynced
	case EnvelopeOtherError:
		return types2.EnvelopeOtherError
	}
	return types2.EnvelopeOtherError
}

type gethSubscriptionWrapper struct {
	subscription event.Subscription
}

// NewGethSubscriptionWrapper returns an object that wraps Geth's Subscription in a types interface
func NewGethSubscriptionWrapper(subscription event.Subscription) types2.Subscription {
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
