package message

// Message with data and payload
type Message struct {
	Body    interface{}
	Payload *Payload
}
