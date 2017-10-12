package common

// Notifier manages Push Notifications.
type Notifier interface {
	Notify(body interface{}, tokens ...string) error
}

// NotifierConstructor returns constructor of configured instance Notifier interface.
type NotifierConstructor func() Notifier
