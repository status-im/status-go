package statusproto

import (
	"errors"
	"io"
	"reflect"

	"github.com/russolsen/transit"
)

// NewMessageEncoder returns a new Transit encoder
// that can encode Message values.
// More about Transit: https://github.com/cognitect/transit-format
func NewMessageEncoder(w io.Writer) *transit.Encoder {
	encoder := transit.NewEncoder(w, false)
	encoder.AddHandler(messageType, defaultMessageValueEncoder)
	encoder.AddHandler(pairMessageType, defaultMessageValueEncoder)
	return encoder
}

var (
	messageType                = reflect.TypeOf(Message{})
	pairMessageType            = reflect.TypeOf(PairMessage{})
	defaultMessageValueEncoder = &messageValueEncoder{}
)

type messageValueEncoder struct{}

func (messageValueEncoder) IsStringable(reflect.Value) bool {
	return false
}

func (messageValueEncoder) Encode(e transit.Encoder, value reflect.Value, asString bool) error {
	switch message := value.Interface().(type) {
	case Message:
		taggedValue := transit.TaggedValue{
			Tag: messageTag,
			Value: []interface{}{
				message.Text,
				message.ContentT,
				transit.Keyword(message.MessageT),
				message.Clock,
				message.Timestamp,
				map[interface{}]interface{}{
					transit.Keyword("chat-id"): message.Content.ChatID,
					transit.Keyword("text"):    message.Content.Text,
				},
			},
		}
		return e.EncodeInterface(taggedValue, false)
	case PairMessage:
		taggedValue := transit.TaggedValue{
			Tag: pairMessageTag,
			Value: []interface{}{
				message.InstallationID,
				message.DeviceType,
				message.Name,
				message.FCMToken,
			},
		}
		return e.EncodeInterface(taggedValue, false)
	}

	return errors.New("unknown message type to encode")
}
