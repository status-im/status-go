package protocol

import (
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
	return encoder
}

var (
	messageType                = reflect.TypeOf(Message{})
	defaultMessageValueEncoder = &messageValueEncoder{}
)

type messageValueEncoder struct{}

func (messageValueEncoder) IsStringable(reflect.Value) bool {
	return false
}

func (messageValueEncoder) Encode(e transit.Encoder, value reflect.Value, asString bool) error {
	message := value.Interface().(Message)
	taggedValue := transit.TaggedValue{
		Tag: statusMessageTag,
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
}
