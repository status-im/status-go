package common

// Notifier manages Push Notifications.
type Notifier interface {
	Notify(body interface{}, tokens ...string) error
}
