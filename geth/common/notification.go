package common

import (
	"github.com/NaySoftware/go-fcm"
	"github.com/status-im/status-go/geth/notification/message"
)

// Notification manages Push Notifications and send messages.
type Notification interface {
	Notify(token string, msg *message.Message) (string, error)
}

// MessagingProvider manages send/notification messaging clients.
type MessagingProvider interface {
	SetMessage(ids []string, body interface{})
	SetPayload(payload *message.Payload)
	Send() error
}

// FirebaseClient is a copy of "go-fcm" client methods.
type FirebaseClient interface {
	NewFcmRegIdsMsg(list []string, body interface{}) *fcm.FcmClient
	Send() (*fcm.FcmResponseStatus, error)
	SetNotificationPayload(payload *fcm.NotificationPayload) *fcm.FcmClient
}
