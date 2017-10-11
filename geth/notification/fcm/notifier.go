package fcm

import (
	"github.com/NaySoftware/go-fcm"
	"github.com/status-im/status-go/geth/notification"
)

// Notifier represents messaging provider for notifications.
type Notifier struct {
	firebaseClient
}

// NewNotifier Firebase Cloud Messaging client constructor.
func NewNotifier(key string) *Notifier {
	return &Notifier{fcm.NewFcmClient(key)}
}

// Notify preparation and send to the tokens list.
func (p *Notifier) Notify(body interface{}, tokens ...string) error {
	p.setPayload(&notification.Payload{
		Title: "Status - new message",
		Body:  "ping",
	})

	p.setMessage(body, tokens...)
	_, err := p.firebaseClient.Send()

	return err
}

// SetMessage to send for given the tokens list.
func (p *Notifier) setMessage(body interface{}, tokens ...string) {
	p.NewFcmRegIdsMsg(tokens, body)
}

// SetPayload sets payload message information.
func (p *Notifier) setPayload(payload *notification.Payload) {
	fcmPayload := p.toFCMPayload(payload)
	p.SetNotificationPayload(fcmPayload)
}

func (p *Notifier) toFCMPayload(payload *notification.Payload) *fcm.NotificationPayload {
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
