package fcm

import (
	gofcm "github.com/NaySoftware/go-fcm"
)

// FirebaseClient is a copy of "go-fcm" client methods.
type FirebaseClient interface {
	NewFcmRegIdsMsg(tokens []string, body interface{}) *gofcm.FcmClient
	Send() (*gofcm.FcmResponseStatus, error)
}
