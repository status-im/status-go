package notification

import (
	"github.com/NaySoftware/go-fcm"
	"github.com/status-im/status-go/geth/notification/message"
	"github.com/status-im/status-go/geth/common"
)

// FCMProvider represents messaging provider for notifications.
type FCMProvider struct {
	common.FirebaseClient
}

// NewFCMProvider Firebase Cloud Messaging client constructor.
func NewFCMProvider(fcmClient common.FirebaseClient) *FCMProvider {
	return &FCMProvider{fcmClient}
}

// SetMessage to send for given ids.
func (p *FCMProvider) SetMessage(ids []string, body interface{}) {
	p.NewFcmRegIdsMsg(ids, body)
}

// SetPayload sets payload message information.
func (p *FCMProvider) SetPayload(payload *message.Payload) {
	fcmPayload := p.toFCMPayload(payload)
	p.SetNotificationPayload(fcmPayload)
}

// Send message.
func (p *FCMProvider) Send() error {
	_, err := p.FirebaseClient.Send()
	return err
}

func (p *FCMProvider) toFCMPayload(payload *message.Payload) *fcm.NotificationPayload {
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
