package wakuv2

import (
	"github.com/ethereum/go-ethereum/event"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	common "github.com/status-im/status-go/pkg/messaging/waku/common"
	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// NewWakuV2EnvelopeEventWrapper converts an internal waku envelope event into
// the neutral types.EnvelopeEvent consumed by the transport layer.
func NewWakuV2EnvelopeEventWrapper(envelopeEvent *common.EnvelopeEvent) *types.EnvelopeEvent {
	if envelopeEvent == nil {
		panic("envelopeEvent should not be nil")
	}

	return &types.EnvelopeEvent{
		Event: types.EventType(envelopeEvent.Event),
		Hash:  cryptotypes.Hash(envelopeEvent.Hash),
		Data:  envelopeEvent.Data,
	}
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
