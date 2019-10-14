package gethbridge

import (
	"github.com/ethereum/go-ethereum/event"
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
)

type gethSubscriptionWrapper struct {
	subscription event.Subscription
}

// NewGethSubscriptionWrapper returns an object that wraps Geth's Subscription in a whispertypes interface
func NewGethSubscriptionWrapper(subscription event.Subscription) whispertypes.Subscription {
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
