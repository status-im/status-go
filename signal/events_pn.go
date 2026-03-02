package signal

import "fmt"

const (
	notificationEvent = "local-notifications"
)

// SendLocalNotifications sends event with a local notification.
func SendLocalNotifications(event interface{}) {
	fmt.Printf("[signal] SendLocalNotifications: invoking send(%q, ...)\n", notificationEvent)
	send(notificationEvent, event)
}
