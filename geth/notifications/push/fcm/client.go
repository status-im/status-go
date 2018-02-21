package fcm

import "github.com/NaySoftware/go-fcm"

// firebaseClient is a copy of "go-fcm" client methods.
type firebaseClient interface {
	NewFcmRegIdsMsg(tokens []string, body interface{}) *fcm.FcmClient
	Send() (*fcm.FcmResponseStatus, error)
	SetNotificationPayload(payload *fcm.NotificationPayload) *fcm.FcmClient
}
