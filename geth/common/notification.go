package common

// Notifier manages Push Notifications.
type Notifier interface {
	Send(body interface{}, tokens ...string) error
}

// NotificationConstructor returns constructor of configured instance Notifier interface.
type NotificationConstructor func() Notifier
