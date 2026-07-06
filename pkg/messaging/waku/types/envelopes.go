package types

import (
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// Envelope represents a clear-text data packet to transmit through the Whisper
// network. Its contents may or may not be encrypted and signed.
type Envelope interface {
	Wrapped

	Hash() cryptotypes.Hash // cached hash of the envelope to avoid rehashing every time
	Bloom() []byte
	PoW() float64
	Expiry() uint32
	TTL() uint32
	Topic() TopicType
	Size() int
}

// EventType identifies envelope events emitted by the waku backend.
//
// These events are internal to the messaging stack: they are consumed by the
// transport layer only and never leave the process. Client-facing signals are
// a separate contract owned by the signal package.
type EventType string

const (
	// EventEnvelopeSent fires when an envelope is confirmed as sent into the network.
	EventEnvelopeSent EventType = "envelope.sent"
	// EventEnvelopeExpired fires when an envelope failed to be sent and won't be retried.
	EventEnvelopeExpired EventType = "envelope.expired"
	// EventEnvelopeAvailable fires when a received envelope is available for the
	// transport to decode and route. Data carries a *ReceivedMessage.
	EventEnvelopeAvailable EventType = "envelope.available"
)

// EnvelopeEvent used for envelopes events.
type EnvelopeEvent struct {
	Event EventType
	Hash  cryptotypes.Hash
	Data  interface{}
}

// Subscription represents a stream of events. The carrier of the events is typically a
// channel, but isn't part of the interface.
//
// Subscriptions can fail while established. Failures are reported through an error
// channel. It receives a value if there is an issue with the subscription (e.g. the
// network connection delivering the events has been closed). Only one value will ever be
// sent.
//
// The error channel is closed when the subscription ends successfully (i.e. when the
// source of events is closed). It is also closed when Unsubscribe is called.
//
// The Unsubscribe method cancels the sending of events. You must call Unsubscribe in all
// cases to ensure that resources related to the subscription are released. It can be
// called any number of times.
type Subscription interface {
	Err() <-chan error // returns the error channel
	Unsubscribe()      // cancels sending of events, closing the error channel
}
