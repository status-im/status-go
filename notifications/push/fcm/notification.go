package fcm

import (
	"encoding/json"
	"fmt"

	"github.com/NaySoftware/go-fcm"
)

// Notifier manages Push Notifications.
type Notifier interface {
	Send(dataPayloadJSON string, tokens ...string) error
}

// NotificationConstructor returns constructor of configured instance Notifier interface.
type NotificationConstructor func() Notifier

// Notification represents messaging provider for notifications.
type Notification struct {
	client FirebaseClient
}

// NewNotification Firebase Cloud Messaging client constructor.
func NewNotification(key string) NotificationConstructor {
	return func() Notifier {
		client := fcm.NewFcmClient(key).
			SetDelayWhileIdle(true).
			SetContentAvailable(true).
			SetPriority(fcm.Priority_HIGH). // Message needs to be marked as high-priority so that background task in an Android's recipient device can be invoked (https://github.com/invertase/react-native-firebase/blob/d13f0af53f1c8f20db8bc8d4b6f8c6d210e108b9/android/src/main/java/io/invertase/firebase/messaging/RNFirebaseMessagingService.java#L56)
			SetTimeToLive(fcm.MAX_TTL)

		return &Notification{client}
	}
}

// Send sends a push notification to the tokens list.
func (n *Notification) Send(dataPayloadJSON string, tokens ...string) error {
	var dataPayload map[string]string
	err := json.Unmarshal([]byte(dataPayloadJSON), &dataPayload)
	if err != nil {
		return err
	}

	n.client.NewFcmRegIdsMsg(tokens, dataPayload)
	resp, err := n.client.Send()
	if err != nil {
		return err
	}

	if resp != nil && !resp.Ok {
		return fmt.Errorf("FCM error sending message, code=%d err=%s", resp.StatusCode, resp.Err)
	}

	return nil
}
