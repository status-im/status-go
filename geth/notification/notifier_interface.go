package notification

// Notifier manages Push Notifications.
type Notifier interface {
	Send(body string, payload Payload, tokens ...string) error
}

// Constructor returns constructor of configured instance Notifier interface.
type Constructor func() Notifier
