package common

import (
	"sync"

	"github.com/status-im/status-go/pkg/pubsub"
)

// ServiceState represents the lifecycle state of a pausable service.
type ServiceState string

const (
	ServiceStateRunning ServiceState = "running"
	ServiceStatePaused  ServiceState = "paused"
	ServiceStateStopped ServiceState = "stopped"
)

// Pausable is implemented by services that support granular pause/resume control.
// The name returned by PausableName is the stable identifier used in the API.
type Pausable interface {
	PausableName() string
	Pause() error
	Resume() error
	PausableState() ServiceState
}

// Subscription is the read side of a pause-state subscription from a PauseBroadcaster.
type Subscription interface {
	// C returns a channel that receives true when paused, false when resumed.
	// The current pause state is always sent immediately on subscribe.
	C() <-chan bool
	Unsubscribe()
}

type lifecycleSubscription struct {
	publisher *pubsub.TypePublisher[bool]
	ch        chan bool
}

func (s *lifecycleSubscription) C() <-chan bool { return s.ch }
func (s *lifecycleSubscription) Unsubscribe()   { s.publisher.Unsubscribe(s.ch) }

// PauseBroadcaster provides common pause/resume state tracking for services implementing Pausable.
// Embed this struct and call the Mark* methods to avoid duplicating state logic.
// Calling MarkPaused/MarkResumed also broadcasts to any active Subscribe() subscribers.
type PauseBroadcaster struct {
	mu   sync.RWMutex
	once sync.Once
	pub  *pubsub.TypePublisher[bool]

	state ServiceState
}

func (m *PauseBroadcaster) initPub() {
	m.once.Do(func() {
		m.pub = pubsub.NewTypePublisher[bool]()
	})
}

func (m *PauseBroadcaster) PausableState() ServiceState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state == "" {
		return ServiceStateStopped
	}
	return m.state
}

func (m *PauseBroadcaster) MarkStarted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = ServiceStateRunning
}

func (m *PauseBroadcaster) MarkStopped() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = ServiceStateStopped
}

func (m *PauseBroadcaster) MarkPaused() {
	m.initPub()
	m.mu.Lock()
	m.state = ServiceStatePaused
	m.mu.Unlock()
	m.pub.Publish(true)
}

func (m *PauseBroadcaster) MarkResumed() {
	m.initPub()
	m.mu.Lock()
	m.state = ServiceStateRunning
	m.mu.Unlock()
	m.pub.Publish(false)
}

func (m *PauseBroadcaster) IsPaused() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state == ServiceStatePaused
}

// Subscribe returns a Subscription that receives pause-state transitions.
// The current pause state is sent immediately so the caller does not need a
// separate IsPaused() check before entering the select loop.
func (m *PauseBroadcaster) Subscribe() Subscription {
	m.initPub()
	// Buffer must fit initial snapshot plus one transition before the consumer reads; see
	// common.TestPauseBroadcaster_Subscribe_* and pkg/pubsub.TestTypePublisher_publishDroppedWhenPerSubscriberBufferFull.
	ch := m.pub.Subscribe(2)
	// Deliver the current state immediately.
	select {
	case ch <- m.IsPaused():
	default:
	}
	return &lifecycleSubscription{publisher: m.pub, ch: ch}
}
