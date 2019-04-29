package incentivisation

import (
	"bytes"
	"io"
	"reflect"
	"time"

	"github.com/russolsen/transit"
)

type StatusMessageContent struct {
	ChatID string
	Text   string
}

type StatusMessage struct {
	Text      string
	ContentT  string
	MessageT  string
	Clock     int64
	Timestamp int64
	Content   StatusMessageContent
}

// CreateTextStatusMessage creates a StatusMessage.
func CreateTextStatusMessage(text string, chatID string) StatusMessage {
	ts := time.Now().Unix() * 1000

	return StatusMessage{
		Text:      text,
		ContentT:  "text/plain",
		MessageT:  "public-group-user-message",
		Clock:     ts * 100,
		Timestamp: ts,
		Content:   StatusMessageContent{ChatID: chatID, Text: text},
	}
}

func EncodeMessage(content string, chatID string) ([]byte, error) {
	value := CreateTextStatusMessage(content, chatID)
	var buf bytes.Buffer
	encoder := NewMessageEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewMessageEncoder returns a new Transit encoder
// that can encode StatusMessage values.
// More about Transit: https://github.com/cognitect/transit-format
func NewMessageEncoder(w io.Writer) *transit.Encoder {
	encoder := transit.NewEncoder(w, false)
	encoder.AddHandler(statusMessageType, defaultStatusMessageValueEncoder)
	return encoder
}

var (
	statusMessageType                = reflect.TypeOf(StatusMessage{})
	defaultStatusMessageValueEncoder = &statusMessageValueEncoder{}
)

type statusMessageValueEncoder struct{}

func (statusMessageValueEncoder) IsStringable(reflect.Value) bool {
	return false
}

func (statusMessageValueEncoder) Encode(e transit.Encoder, value reflect.Value, asString bool) error {
	message := value.Interface().(StatusMessage)
	taggedValue := transit.TaggedValue{
		Tag: "c4",
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
