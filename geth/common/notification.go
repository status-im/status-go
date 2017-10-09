package common

import "github.com/NaySoftware/go-fcm"

// Notification manages Push Notifications and send messages
type Notification interface {
	Notify(token string) string
	Send() error
}

// Messaging manages send/notification messaging clients
type Messaging interface {
	NewFcmRegIdsMsg(list []string, body interface{}) *fcm.FcmClient
	Send() (*fcm.FcmResponseStatus, error)
	SetNotificationPayload(payload *fcm.NotificationPayload) *fcm.FcmClient
}
