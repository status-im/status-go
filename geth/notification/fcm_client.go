package notification

import "github.com/NaySoftware/go-fcm"

const (
	//todo(jeka): should be removed
	fcmServerKey = "AAAAxwa-r08:APA91bFtMIToDVKGAmVCm76iEXtA4dn9MPvLdYKIZqAlNpLJbd12EgdBI9DSDSXKdqvIAgLodepmRhGVaWvhxnXJzVpE6MoIRuKedDV3kfHSVBhWFqsyoLTwXY4xeufL9Sdzb581U-lx"
)

// NewFCMClient Firebase Cloud Messaging client constructor
func NewFCMClient() *fcm.FcmClient {
	return fcm.NewFcmClient(fcmServerKey).SetNotificationPayload(getNotificationPayload())
}

// only for feature testing
func getNotificationPayload() *fcm.NotificationPayload {
	return &fcm.NotificationPayload{
		Title: "Status - new message",
		Body:  "ping",
	}
}
