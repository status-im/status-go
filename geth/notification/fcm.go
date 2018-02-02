package notification

import (
	"github.com/NaySoftware/go-fcm"
)

// FirebaseClient notification client
type FirebaseClient struct {
	FcmClient *fcm.FcmClient
}

// AddDevices is a Firebase implementation of notification.Client interface method
func (fC FirebaseClient) AddDevices(deviceIDs []string, body interface{}) {
	fC.FcmClient.NewFcmRegIdsMsg(deviceIDs, body)
}

// Send is a Firebase implementation of notification.Client interface method:
func (fC FirebaseClient) Send(payload *Payload) (*Response, error) {
	resp, err := fC.FcmClient.SetNotificationPayload(
		payload.ToFCMNotificationPayload()).Send()
	if err != nil {
		return nil, err
	}
	return FromFCMResponseStatus(resp), nil
}
