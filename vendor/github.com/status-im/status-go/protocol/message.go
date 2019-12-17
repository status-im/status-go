package protocol

import (
	"crypto/ecdsa"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gomarkdown/markdown"
	"github.com/status-im/status-go/protocol/protobuf"
)

// QuotedMessage contains the original text of the message replied to
type QuotedMessage struct {
	// From is a public key of the author of the message.
	From string `json:"from"`
	Text string `json:"text"`
}

const (
	OutgoingStatusSending = "sending"
	OutgoingStatusSent    = "sent"
)

// Message represents a message record in the database,
// more specifically in user_messages_legacy table.
type Message struct {
	protobuf.ChatMessage

	// ID calculated as keccak256(compressedAuthorPubKey, data) where data is unencrypted payload.
	ID string `json:"id"`
	// WhisperTimestamp is a timestamp of a Whisper envelope.
	WhisperTimestamp uint64 `json:"whisperTimestamp"`
	// From is a public key of the author of the message.
	From string `json:"from"`
	// Random 3 words name
	Alias string `json:"alias"`
	// Identicon of the author
	Identicon string `json:"identicon"`
	// The chat id to be stored locally
	LocalChatID string `json:"localChatId"`

	RetryCount     int    `json:"retryCount"`
	Seen           bool   `json:"seen"`
	OutgoingStatus string `json:"outgoingStatus,omitempty"`

	QuotedMessage *QuotedMessage `json:"quotedMessage"`

	// Computed fields
	RTL        bool   `json:"rtl"`
	ParsedText []byte `json:"parsedText"`
	LineCount  int    `json:"lineCount"`

	SigPubKey *ecdsa.PublicKey `json:"-"`
	// RawPayload is the marshaled payload, used for resending the message
	RawPayload []byte `json:"-"`
}

func (m *Message) MarshalJSON() ([]byte, error) {
	type MessageAlias Message
	item := struct {
		ID               string                           `json:"id"`
		WhisperTimestamp uint64                           `json:"whisperTimestamp"`
		From             string                           `json:"from"`
		Alias            string                           `json:"alias"`
		Identicon        string                           `json:"identicon"`
		RetryCount       int                              `json:"retryCount"`
		Seen             bool                             `json:"seen"`
		OutgoingStatus   string                           `json:"outgoingStatus,omitempty"`
		QuotedMessage    *QuotedMessage                   `json:"quotedMessage"`
		RTL              bool                             `json:"rtl"`
		ParsedText       json.RawMessage                  `json:"parsedText"`
		LineCount        int                              `json:"lineCount"`
		Text             string                           `json:"text"`
		ChatId           string                           `json:"chatId"`
		LocalChatID      string                           `json:"localChatId"`
		Clock            uint64                           `json:"clock"`
		ResponseTo       string                           `json:"responseTo"`
		EnsName          string                           `json:"ensName"`
		Sticker          *protobuf.StickerMessage         `json:"sticker"`
		Timestamp        uint64                           `json:"timestamp"`
		ContentType      protobuf.ChatMessage_ContentType `json:"contentType"`
		MessageType      protobuf.ChatMessage_MessageType `json:"messageType"`
	}{
		ID:               m.ID,
		WhisperTimestamp: m.WhisperTimestamp,
		From:             m.From,
		Alias:            m.Alias,
		Identicon:        m.Identicon,
		RetryCount:       m.RetryCount,
		Seen:             m.Seen,
		OutgoingStatus:   m.OutgoingStatus,
		QuotedMessage:    m.QuotedMessage,
		RTL:              m.RTL,
		ParsedText:       m.ParsedText,
		LineCount:        m.LineCount,
		Text:             m.Text,
		ChatId:           m.ChatId,
		LocalChatID:      m.LocalChatID,
		Clock:            m.Clock,
		ResponseTo:       m.ResponseTo,
		EnsName:          m.EnsName,
		Timestamp:        m.Timestamp,
		ContentType:      m.ContentType,
		MessageType:      m.MessageType,
		Sticker:          m.GetSticker(),
	}

	return json.Marshal(item)
}

func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := struct {
		*Alias
		ResponseTo  string                           `json:"responseTo"`
		EnsName     string                           `json:"ensName"`
		ChatID      string                           `json:"chatId"`
		Sticker     *protobuf.StickerMessage         `json:"sticker"`
		ContentType protobuf.ChatMessage_ContentType `json:"contentType"`
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ContentType == protobuf.ChatMessage_STICKER {
		m.Payload = &protobuf.ChatMessage_Sticker{Sticker: aux.Sticker}
	}
	m.ResponseTo = aux.ResponseTo
	m.EnsName = aux.EnsName
	m.ChatId = aux.ChatID
	m.ContentType = aux.ContentType
	return nil
}

// Check if the first character is Hebrew or Arabic or the RTL character
func isRTL(s string) bool {
	first, _ := utf8.DecodeRuneInString(s)
	return unicode.Is(unicode.Hebrew, first) ||
		unicode.Is(unicode.Arabic, first) ||
		// RTL character
		first == '\u200f'
}

// PrepareContent return the parsed content of the message, the line-count and whether
// is a right-to-left message
func (m *Message) PrepareContent() error {
	parsedText := markdown.Parse([]byte(m.Text), nil)
	jsonParsedText, err := json.Marshal(parsedText)
	if err != nil {
		return err
	}
	m.ParsedText = jsonParsedText
	m.LineCount = strings.Count(m.Text, "\n")
	m.RTL = isRTL(m.Text)
	return nil
}
