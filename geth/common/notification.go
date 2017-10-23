package common

import "github.com/NaySoftware/go-fcm"

// Notifier manages Push Notifications.
type Notifier interface {
	Send(body string, payload fcm.NotificationPayload, tokens ...string) error
}

// NotificationConstructor returns constructor of configured instance Notifier interface.
type NotificationConstructor func() Notifier
