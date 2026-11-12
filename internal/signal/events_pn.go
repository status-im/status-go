package signal

const (
	notificationEvent = "local-notifications"
)

// SendLocalNotifications sends event with a local notification.
func SendLocalNotifications(event interface{}) {
	CurrentSession().SendLocalNotifications(event)
}

// SendLocalNotifications sends in the session s.
func (s Session) SendLocalNotifications(event interface{}) {
	s.send(notificationEvent, event)
}
