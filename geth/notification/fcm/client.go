package fcm

import "github.com/NaySoftware/go-fcm"

// FirebaseClient is a copy of "go-fcm" client methods.
type FirebaseClient interface {
	NewFcmRegIdsMsg(tokens []string, body interface{}) *fcm.FcmClient
	Send() (*fcm.FcmResponseStatus, error)
	SetNotificationPayload(payload *fcm.NotificationPayload) *fcm.FcmClient
}