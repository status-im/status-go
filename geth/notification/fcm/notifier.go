package fcm

import (
	"github.com/NaySoftware/go-fcm"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/notification"
)

// Notifier represents messaging provider for notifications.
type Notifier struct {
	client firebaseClient
}

// NewNotifier Firebase Cloud Messaging client constructor.
func NewNotifier(key string) common.NotificationConstructor {
	return func() common.Notifier {
		return &Notifier{fcm.NewFcmClient(key)}
	}
}

// Send send to the tokens list.
func (n *Notifier) Send(body interface{}, tokens ...string) error {
	n.setPayload(&notification.Payload{
		Title: "Status - new message",
		Body:  "ping",
	})

	n.setMessage(body, tokens...)
	_, err := n.client.Send()

	return err
}

// SetMessage to send for given the tokens list.
func (n *Notifier) setMessage(body interface{}, tokens ...string) {
	n.client.NewFcmRegIdsMsg(tokens, body)
}

// SetPayload sets payload message information.
func (n *Notifier) setPayload(payload *notification.Payload) {
	fcmPayload := n.toFCMPayload(payload)
	n.client.SetNotificationPayload(fcmPayload)
}

func (n *Notifier) toFCMPayload(payload *notification.Payload) *fcm.NotificationPayload {
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
