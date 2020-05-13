package protocol

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/status-im/markdown"

	"github.com/status-im/status-go/protocol/protobuf"
)

// QuotedMessage contains the original text of the message replied to
type QuotedMessage struct {
	// From is a public key of the author of the message.
	From string `json:"from"`
	Text string `json:"text"`
}

type CommandState int

const (
	CommandStateRequestAddressForTransaction CommandState = iota + 1
	CommandStateRequestAddressForTransactionDeclined
	CommandStateRequestAddressForTransactionAccepted
	CommandStateRequestTransaction
	CommandStateRequestTransactionDeclined
	CommandStateTransactionPending
	CommandStateTransactionSent
)

type CommandParameters struct {
	// ID is the ID of the initial message
	ID string `json:"id"`
	// From is the address we are sending the command from
	From string `json:"from"`
	// Address is the address sent with the command
	Address string `json:"address"`
	// Contract is the contract address for ERC20 tokens
	Contract string `json:"contract"`
	// Value is the value as a string sent
	Value string `json:"value"`
	// TransactionHash is the hash of the transaction
	TransactionHash string `json:"transactionHash"`
	// CommandState is the state of the command
	CommandState CommandState `json:"commandState"`
	// The Signature of the pk-bytes+transaction-hash from the wallet
	// address originating
	Signature []byte `json:"signature"`
}

func (c *CommandParameters) IsTokenTransfer() bool {
	return len(c.Contract) != 0
}

const (
	OutgoingStatusSending = "sending"
	OutgoingStatusSent    = "sent"
)

// Message represents a message record in the database,
// more specifically in user_messages table.
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

	Seen           bool   `json:"seen"`
	OutgoingStatus string `json:"outgoingStatus,omitempty"`

	QuotedMessage *QuotedMessage `json:"quotedMessage"`

	// CommandParameters is the parameters sent with the message
	CommandParameters *CommandParameters `json:"commandParameters"`

	// Computed fields
	// RTL is whether this is a right-to-left message (arabic/hebrew script etc)
	RTL bool `json:"rtl"`
	// ParsedText is the parsed markdown for displaying
	ParsedText []byte `json:"parsedText"`
	// LineCount is the count of newlines in the message
	LineCount int `json:"lineCount"`
	// Base64Image is the converted base64 image
	Base64Image string `json:"image,omitempty"`
	// ImagePath is the path of the image to be sent
	ImagePath string `json:"imagePath,omitempty"`

	// Replace indicates that this is a replacement of a message
	// that has been updated
	Replace   string           `json:"replace,omitempty"`
	SigPubKey *ecdsa.PublicKey `json:"-"`
}

// RawMessage represent a sent or received message, kept for being able
// to re-send/propagate
type RawMessage struct {
	ID                  string
	LocalChatID         string
	LastSent            uint64
	SendCount           int
	Sent                bool
	ResendAutomatically bool
	MessageType         protobuf.ApplicationMetadataMessage_Type
	Payload             []byte
	Recipients          []*ecdsa.PublicKey
}

func (m *Message) MarshalJSON() ([]byte, error) {
	type StickerAlias struct {
		Hash string `json:"hash"`
		Pack int32  `json:"pack"`
	}
	item := struct {
		ID                string                           `json:"id"`
		WhisperTimestamp  uint64                           `json:"whisperTimestamp"`
		From              string                           `json:"from"`
		Alias             string                           `json:"alias"`
		Identicon         string                           `json:"identicon"`
		Seen              bool                             `json:"seen"`
		OutgoingStatus    string                           `json:"outgoingStatus,omitempty"`
		QuotedMessage     *QuotedMessage                   `json:"quotedMessage"`
		RTL               bool                             `json:"rtl"`
		ParsedText        json.RawMessage                  `json:"parsedText"`
		LineCount         int                              `json:"lineCount"`
		Text              string                           `json:"text"`
		ChatID            string                           `json:"chatId"`
		LocalChatID       string                           `json:"localChatId"`
		Clock             uint64                           `json:"clock"`
		Replace           string                           `json:"replace"`
		ResponseTo        string                           `json:"responseTo"`
		EnsName           string                           `json:"ensName"`
		Image             string                           `json:"image,omitempty"`
		Sticker           *StickerAlias                    `json:"sticker"`
		CommandParameters *CommandParameters               `json:"commandParameters"`
		Timestamp         uint64                           `json:"timestamp"`
		ContentType       protobuf.ChatMessage_ContentType `json:"contentType"`
		MessageType       protobuf.ChatMessage_MessageType `json:"messageType"`
	}{
		ID:                m.ID,
		WhisperTimestamp:  m.WhisperTimestamp,
		From:              m.From,
		Alias:             m.Alias,
		Identicon:         m.Identicon,
		Seen:              m.Seen,
		OutgoingStatus:    m.OutgoingStatus,
		QuotedMessage:     m.QuotedMessage,
		RTL:               m.RTL,
		ParsedText:        m.ParsedText,
		LineCount:         m.LineCount,
		Text:              m.Text,
		Replace:           m.Replace,
		ChatID:            m.ChatId,
		LocalChatID:       m.LocalChatID,
		Clock:             m.Clock,
		ResponseTo:        m.ResponseTo,
		EnsName:           m.EnsName,
		Image:             m.Base64Image,
		Timestamp:         m.Timestamp,
		ContentType:       m.ContentType,
		MessageType:       m.MessageType,
		CommandParameters: m.CommandParameters,
	}

	if sticker := m.GetSticker(); sticker != nil {
		item.Sticker = &StickerAlias{
			Pack: sticker.Pack,
			Hash: sticker.Hash,
		}
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

// parseImage check the message contains an image, and if so
// it creates the a base64 encoded version of it.
func (m *Message) parseImage() error {
	if m.ContentType != protobuf.ChatMessage_IMAGE {
		return nil
	}
	image := m.GetImage()
	if image == nil {
		return errors.New("image empty")
	}

	payload := image.Payload

	e64 := base64.StdEncoding

	maxEncLen := e64.EncodedLen(len(payload))
	encBuf := make([]byte, maxEncLen)

	e64.Encode(encBuf, payload)
	var mime string

	switch image.Type {
	case protobuf.ImageMessage_PNG:
		mime = "png"
	case protobuf.ImageMessage_JPEG:
		mime = "jpeg"
	case protobuf.ImageMessage_WEBP:
		mime = "webp"
	case protobuf.ImageMessage_GIF:
		mime = "gif"
	default:
		return errors.New("image format not supported")
	}

	m.Base64Image = fmt.Sprintf("data:image/%s;base64,%s", mime, encBuf)

	return nil
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
	return m.parseImage()
}
