package fcm

import (
	"github.com/NaySoftware/go-fcm"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/notification"
)

// Notification represents messaging provider for notifications.
type Notification struct {
	client firebaseClient
}

// NewNotification Firebase Cloud Messaging client constructor.
func NewNotification(key string) common.NotificationConstructor {
	return func() common.Notifier {
		return &Notification{fcm.NewFcmClient(key)}
	}
}

// Send send to the tokens list.
func (n *Notification) Send(body interface{}, tokens ...string) error {
	n.setPayload(&notification.Payload{
		Title: "Status - new message",
		Body:  "ping",
	})

	n.setMessage(body, tokens...)
	_, err := n.client.Send()

	return err
}

// SetMessage to send for given the tokens list.
func (n *Notification) setMessage(body interface{}, tokens ...string) {
	n.client.NewFcmRegIdsMsg(tokens, body)
}

// SetPayload sets payload message information.
func (n *Notification) setPayload(payload *notification.Payload) {
	fcmPayload := n.toFCMPayload(payload)
	n.client.SetNotificationPayload(fcmPayload)
}

func (n *Notification) toFCMPayload(payload *notification.Payload) *fcm.NotificationPayload {
	return &fcm.NotificationPayload{
		Title: payload.Title,
		Body:  payload.Body,
		Icon:  payload.Icon,
		Sound: payload.Sound,
		Badge: payload.Badge,
		Tag:   payload.Tag,
		Color: payload.Color,
	}
}
