package notification

// Client is a generic-purpose messaging interface client
type Client interface {
	NewRegIdsMsg(tokens []string, body interface{}) Client
	Send() (*Response, error)
	SetNotificationPayload(payload *Payload) Client
}
