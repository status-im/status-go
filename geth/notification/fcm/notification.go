package fcm

import (
	"fmt"

	"github.com/NaySoftware/go-fcm"
	"github.com/status-im/status-go/geth/common"
)

// Notification represents messaging provider for notifications.
type Notification struct {
	client firebaseClient
}

// NewNotification Firebase Cloud Messaging client constructor.
func NewNotification(key string) common.NotificationConstructor {
	return func() common.Notifier {
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
		payload.Title = "Status - new message"
	}
	if payload.Body == "" {
		payload.Body = "ping"
	}

	fmt.Println(payload.Title, payload.Body)

	n.client.NewFcmRegIdsMsg(tokens, data)
	n.client.SetNotificationPayload(&payload)
	_, err := n.client.Send()

	return err
}
