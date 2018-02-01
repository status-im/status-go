package notification

// Notification represents messaging provider for notifications.
type Notification struct {
	client Client
}

// NewNotification is a new messaging client constructor.
func NewNotification(client Client) Constructor {
	return func() Notifier {
		return &Notification{client}
	}
}

// Send send to the tokens list.
func (n *Notification) Send(body string, payload *Payload, deviceIDs ...string) error {
	data := map[string]string{
		"msg": body,
	}

	if payload.Title == "" {
		payload.Title = "Status - new message"
	}
	if payload.Body == "" {
		payload.Body = "ping"
	}

	n.client.AddDevices(deviceIDs, data)
	_, err := n.client.Send(payload)

	return err
}
