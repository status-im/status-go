package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// ContentTypeTextPlain means that the message contains plain text.
	ContentTypeTextPlain = "text/plain"
)

// Message types.
const (
	MessageTypePublicGroup  = "public-group-user-message"
	MessageTypePrivate      = "user-message"
	MessageTypePrivateGroup = "group-user-message"
)

var (
	// ErrInvalidDecodedValue means that the decoded message is of wrong type.
	// This might mean that the status message serialization tag changed.
	ErrInvalidDecodedValue = errors.New("invalid decoded value type")
)

// Content contains the chat ID and the actual text of a message.
type Content struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// TimestampInMs is a timestamp in milliseconds.
type TimestampInMs int64

// Time returns a time.Time instance.
func (t TimestampInMs) Time() time.Time {
	ts := int64(t)
	seconds := ts / 1000
	return time.Unix(seconds, (ts%1000)*int64(time.Millisecond))
}

// TimestampInMsFromTime returns a TimestampInMs from a time.Time instance.
func TimestampInMsFromTime(t time.Time) TimestampInMs {
	return TimestampInMs(t.UnixNano() / int64(time.Millisecond))
}

// Flags define various boolean properties of a message.
type Flags uint64

func (f *Flags) Set(val Flags)     { *f = *f | val }
func (f *Flags) Clear(val Flags)   { *f = *f &^ val }
func (f *Flags) Toggle(val Flags)  { *f = *f ^ val }
func (f Flags) Has(val Flags) bool { return f&val != 0 }

// A list of Message flags. By default, a message is unread.
const (
	MessageRead Flags = 1 << iota
)

// Message contains all message details.
type Message struct {
	Text      string        `json:"text"` // TODO: why is this duplicated?
	ContentT  string        `json:"content_type"`
	MessageT  string        `json:"message_type"`
	Clock     int64         `json:"clock"` // lamport timestamp; see CalcMessageClock for more details
	Timestamp TimestampInMs `json:"timestamp"`
	Content   Content       `json:"content"`

	// not protocol defined fields
	ID        []byte           `json:"-"`
	SigPubKey *ecdsa.PublicKey `json:"-"`
	Flags     Flags            `json:"-"`
}

func (m *Message) MarshalJSON() ([]byte, error) {
	type MessageAlias Message
	item := struct {
		*MessageAlias
		ID string `json:"id"`
	}{
		MessageAlias: (*MessageAlias)(m),
		ID:           fmt.Sprintf("%#x", m.ID),
	}

	return json.Marshal(item)
}

func (m Message) Unread() bool {
	return !m.Flags.Has(MessageRead)
}

// createTextMessage creates a Message.
func createTextMessage(data []byte, lastClock int64, chatID, messageType string) Message {
	text := strings.TrimSpace(string(data))
	ts := TimestampInMsFromTime(time.Now())
	clock := CalcMessageClock(lastClock, ts)

	return Message{
		Text:      text,
		ContentT:  ContentTypeTextPlain,
		MessageT:  messageType,
		Clock:     clock,
		Timestamp: ts,
		Content:   Content{ChatID: chatID, Text: text},
	}
}

// CreatePublicTextMessage creates a public text Message.
func CreatePublicTextMessage(data []byte, lastClock int64, chatID string) Message {
	return createTextMessage(data, lastClock, chatID, MessageTypePublicGroup)
}

// CreatePrivateTextMessage creates a public text Message.
func CreatePrivateTextMessage(data []byte, lastClock int64, chatID string) Message {
	return createTextMessage(data, lastClock, chatID, MessageTypePrivate)
}

// DecodeMessage decodes a raw payload to Message struct.
func DecodeMessage(data []byte) (message Message, err error) {
	buf := bytes.NewBuffer(data)
	decoder := NewMessageDecoder(buf)
	value, err := decoder.Decode()
	if err != nil {
		return
	}

	message, ok := value.(Message)
	if !ok {
		return message, ErrInvalidDecodedValue
	}
	return
}

// EncodeMessage encodes a Message using Transit serialization.
func EncodeMessage(value Message) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewMessageEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
