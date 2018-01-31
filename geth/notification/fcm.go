package notification

import (
	"github.com/NaySoftware/go-fcm"
)

// FirebaseClient notification client
type FirebaseClient struct {
	FcmClient *fcm.FcmClient
}

// NewRegIdsMsg is a Firebase implementation of notification.Client interface method
func (fC FirebaseClient) NewRegIdsMsg(tokens []string, body interface{}) Client {
	fC.FcmClient = fC.FcmClient.NewFcmRegIdsMsg(tokens, body)
	return fC
}

// Send is a Firebase implementation of notification.Client interface method:
func (fC FirebaseClient) Send() (*Response, error) {
	resp, err := fC.FcmClient.Send()
	if err != nil {
		return nil, err
	}
	return FromFCMResponseStatus(resp), nil
}

// SetNotificationPayload is a Firebase implementation of notification.Client interface method:
func (fC FirebaseClient) SetNotificationPayload(payload *Payload) Client {
	fC.FcmClient = fC.FcmClient.SetNotificationPayload(payload.ToFCMNotificationPayload())
	return fC
}
