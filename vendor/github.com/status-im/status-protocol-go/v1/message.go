package statusproto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	datasyncprotobuf "github.com/vacp2p/mvds/protobuf"
	"log"
	"strings"
	"time"
)

//go:generate protoc --go_out=. ./message.proto

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

// StatusMessage is any Status Protocol message.
type StatusMessage struct {
	Message           interface{}
	RawTransitPayload []byte           `json:"-"`
	ID                []byte           `json:"-"`
	SigPubKey         *ecdsa.PublicKey `json:"-"`
}

func (m *StatusMessage) MarshalJSON() ([]byte, error) {
	type MessageAlias StatusMessage
	item := struct {
		*MessageAlias
		ID string `json:"id"`
	}{
		MessageAlias: (*MessageAlias)(m),
		ID:           "0x" + hex.EncodeToString(m.ID),
	}

	return json.Marshal(item)
}

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
	ChatID    string           `json:"-"`
	Public    bool             `json:"-"`
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

func unwrapMessage(data []byte) (*StatusProtocolMessage, error) {
	var message StatusProtocolMessage
	err := proto.Unmarshal(data, &message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func decodeTransitMessage(data []byte) (interface{}, error) {
	buf := bytes.NewBuffer(data)
	decoder := NewMessageDecoder(buf)
	value, err := decoder.Decode()
	if err != nil {
		return nil, err
	}
	return value, nil
}

func UnwrapDatasync(payload []byte) (datasyncPayload datasyncprotobuf.Payload, err error) {
	err = proto.Unmarshal(payload, &datasyncPayload)
	return
}

// DecodeMessage decodes a raw payload to StatusMessage struct.
func DecodeMessage(transportPublicKey *ecdsa.PublicKey, data []byte) (message StatusMessage, err error) {
	transitMessage := data

	// Getting a signature from transport message should happen only if
	// the signature was not defined in the payload itself.
	message.SigPubKey = transportPublicKey

	statusProtocolMessage, err := unwrapMessage(data)
	if err == nil {
		// Wrapped message, extract transit and signature
		transitMessage = statusProtocolMessage.Payload
		if statusProtocolMessage.Signature != nil {
			recoveredKey, err := crypto.SigToPub(
				crypto.Keccak256(transitMessage),
				statusProtocolMessage.Signature,
			)
			if err != nil {
				return message, err
			}

			message.SigPubKey = recoveredKey
		}
	}

	// Copy payload to message
	message.RawTransitPayload = make([]byte, len(transitMessage))
	copy(message.RawTransitPayload, transitMessage)

	message.ID = MessageID(message.SigPubKey, transitMessage)
	// This modifies transitMessage
	value, err := decodeTransitMessage(transitMessage)
	if err != nil {
		log.Printf("[message::DecodeMessage] could not decode message: %#x", message.ID)
		return message, err
	}
	message.Message = value

	return message, nil
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

// MessageID calculates the messageID, by appending the sha3-256 to the compress pubkey
func MessageID(author *ecdsa.PublicKey, data []byte) []byte {
	keyBytes := crypto.CompressPubkey(author)
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
