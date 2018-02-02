package notification

// Client is a generic-purpose messaging interface client
type Client interface {
	AddDevices(deviceIDs []string, body interface{})
	Send(payload *Payload) (*Response, error)
}
