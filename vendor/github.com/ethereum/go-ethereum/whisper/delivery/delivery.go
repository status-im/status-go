package delivery

import (
	"sync"

	"github.com/ethereum/go-ethereum/common/message"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

// MessageDeliveryState holds the current delivery state of a envelope.
// TODO(influx6): Consider adding a error object or string to provided context.
// I believe an error object might be more suitable for status that denote failure
// rejection.
type MessageDeliveryState struct {
	Status   message.Status
	Envelope whisper.Envelope
}

// DeliverySubscriber defines a function type for subscrubers.
type DeliverySubscriber func(MessageDeliveryState)

// DeliveryNotification defines a notification implementation for listening to message status
// events.
type DeliveryNotification struct {
	sml  sync.RWMutex
	subs []DeliverySubscriber
}

// Send delivers envelope with status to all subscribers.
func (d *DeliveryNotification) Send(env *whisper.Envelope, status message.Status) {
	d.sml.RLock()
	defer d.sml.RUnlock()

	var mstatus MessageDeliveryState
	mstatus.Status = status
	mstatus.Envelope = *env

	for _, item := range d.subs {
		item(mstatus)
	}
}

// Unsubscribe removes subscriber into delivery subscription list.
func (d *DeliveryNotification) Unsubscribe(ind int) {
	d.sml.Lock()
	defer d.sml.Unlock()

	if ind > -1 && ind < len(d.subs) {
		d.subs = append(d.subs[:ind], d.subs[ind+1:]...)
	}
}

// FilterUntil filters all messages with a Delivery status below giving status but
// delivers all messages above or equal to provided status.
func (d *DeliveryNotification) FilterUntil(status message.Status, sub DeliverySubscriber) int {
	return d.Subscribe(func(m MessageDeliveryState) {
		if m.Status >= status {
			return
		}

		sub(m)
	})
}

// Filter filters out messages status events who status does not match provided.
func (d *DeliveryNotification) Filter(status message.Status, sub DeliverySubscriber) int {
	return d.Subscribe(func(m MessageDeliveryState) {
		if m.Status != status {
			return
		}

		sub(m)
	})
}

// Subscribe adds subscriber into delivery subscription list.
// It returns the index of subscription.
func (d *DeliveryNotification) Subscribe(sub DeliverySubscriber) int {
	d.sml.Lock()
	defer d.sml.Unlock()

	d.subs = append(d.subs, sub)
	return len(d.subs)
}
