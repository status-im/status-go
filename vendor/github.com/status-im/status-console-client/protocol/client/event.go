package client

import (
	"github.com/status-im/status-console-client/protocol/v1"
)

type EventType int

//go:generate stringer -type=EventType

// A list of available events sent from the client.
const (
	EventTypeInit      EventType = iota + 1
	EventTypeRearrange           // messages were rearranged
	EventTypeMessage             // a new message was appended
	EventTypeError
)

// Event is used to workaround event.Feed type checking.
// Every event.Feed instance will remember first type that was used either in Send or Subscribe.
// After that value of every object will be matched against that type.
// For example if we subscribed first with interface{} - feed.etype will be changed to interface{}
// and then when client.messageFeed is posted to event.Feed it will get value and match it against interface{}.
// Feed type checking is either not accurate or it was designed to prevent subscribing with various interfaces.
type Event struct {
	Interface interface{}
}

type EventWithContact interface {
	GetContact() Contact
}

type EventWithType interface {
	GetType() EventType
}

type EventWithError interface {
	GetError() error
}

type EventWithMessage interface {
	GetMessage() *protocol.Message
}

type baseEvent struct {
	Contact Contact   `json:"contact"`
	Type    EventType `json:"type"`
}

func (e baseEvent) GetContact() Contact { return e.Contact }
func (e baseEvent) GetType() EventType  { return e.Type }

type messageEvent struct {
	baseEvent
	Message *protocol.Message `json:"message"`
}

func (e messageEvent) GetMessage() *protocol.Message { return e.Message }
