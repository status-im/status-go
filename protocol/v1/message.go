package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	protocol "github.com/status-im/status-go/protocol/types"
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

// Message is a chat message sent by an user.
type Message struct {
	Text      string        `json:"text"` // TODO: why is this duplicated?
	ContentT  string        `json:"content_type"`
	MessageT  string        `json:"message_type"`
	Clock     int64         `json:"clock"` // lamport timestamp; see CalcMessageClock for more details
	Timestamp TimestampInMs `json:"timestamp"`
	Content   Content       `json:"content"`

	Flags     Flags            `json:"-"`
	ID        []byte           `json:"-"`
	SigPubKey *ecdsa.PublicKey `json:"-"`
	ChatID    string           `json:"-"` // reference to Chat.ID; not connected to Content.ChatID which is set by sender
}

func (m *Message) MarshalJSON() ([]byte, error) {
	type MessageAlias Message
	item := struct {
		*MessageAlias
		ID string `json:"id"`
	}{
		MessageAlias: (*MessageAlias)(m),
		ID:           "0x" + hex.EncodeToString(m.ID),
	}

	return json.Marshal(item)
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
	m := createTextMessage(data, lastClock, chatID, MessageTypePublicGroup)
	return m
}

// CreatePrivateTextMessage creates a one-to-one message.
func CreatePrivateTextMessage(data []byte, lastClock int64, chatID string) Message {
	return createTextMessage(data, lastClock, chatID, MessageTypePrivate)
}

// CreatePrivateGroupTextMessage creates a group message.
func CreatePrivateGroupTextMessage(data []byte, lastClock int64, chatID string) Message {
	return createTextMessage(data, lastClock, chatID, MessageTypePrivateGroup)
}

func decodeTransitMessage(originalPayload []byte) (interface{}, error) {
	payload := make([]byte, len(originalPayload))
	copy(payload, originalPayload)
	// This modifies the payload
	buf := bytes.NewBuffer(payload)

	decoder := NewMessageDecoder(buf)
	value, err := decoder.Decode()
	if err != nil {
		return nil, err
	}
	return value, nil
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

// MessageID calculates the messageID from author's compressed public key
// and not encrypted but encoded payload.
func MessageID(author *ecdsa.PublicKey, data []byte) protocol.HexBytes {
	keyBytes := crypto.FromECDSAPub(author)
	return crypto.Keccak256(append(keyBytes, data...))
}

// WrapMessageV1 wraps a payload into a protobuf message and signs it if an identity is provided
func WrapMessageV1(payload []byte, identity *ecdsa.PrivateKey) ([]byte, error) {
	var signature []byte
	if identity != nil {
		var err error
		signature, err = crypto.Sign(crypto.Keccak256(payload), identity)
		if err != nil {
			return nil, err
		}
	}

	message := &StatusProtocolMessage{
		Signature: signature,
		Payload:   payload,
	}
	return proto.Marshal(message)
}
