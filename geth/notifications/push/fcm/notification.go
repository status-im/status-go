package fcm

import (
	"fmt"

	"github.com/NaySoftware/go-fcm"
)

// Notifier manages Push Notifications.
type Notifier interface {
	Send(body string, payload fcm.NotificationPayload, tokens ...string) error
}

// NotificationConstructor returns constructor of configured instance Notifier interface.
type NotificationConstructor func() Notifier

// Notification represents messaging provider for notifications.
type Notification struct {
	client firebaseClient
}

// NewNotification Firebase Cloud Messaging client constructor.
func NewNotification(key string) NotificationConstructor {
	return func() Notifier {
		client := fcm.NewFcmClient(key).
			SetDelayWhileIdle(true).
			SetContentAvailable(true).
			SetTimeToLive(fcm.MAX_TTL)

		return &Notification{client}
	}
}

// Send send to the tokens list.
func (n *Notification) Send(body string, payload fcm.NotificationPayload, tokens ...string) error {
	data := map[string]string{
		"msg": body,
	}

	if payload.Title == "" {
		payload.Title = "Status"
	}
	if payload.Body == "" {
		payload.Body = "You have a new message"
	}

	fmt.Println(payload.Title, payload.Body)

	n.client.NewFcmRegIdsMsg(tokens, data)
	n.client.SetNotificationPayload(&payload)
	_, err := n.client.Send()

	return err
}
