package statusproto

import (
	"database/sql/driver"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

type hexutilSQL hexutil.Bytes

func (h hexutilSQL) Value() (driver.Value, error) {
	return []byte(h), nil
}

func (h *hexutilSQL) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if b, ok := value.([]byte); ok {
		*h = hexutilSQL(b)
		return nil
	}
	return errors.New("failed to scan hexutilSQL")
}

// Message represents a message record in the database,
// more specifically in user_messages_legacy table.
// Encoding and decoding of byte blobs should be performed
// using hexutil package.
type Message struct {
	// ID calculated as keccak256(compressedAuthorPubKey, data) where data is unencrypted payload.
	ID string `json:"id"`
	// RawPayloadHash is a Whisper envelope hash.
	RawPayloadHash string `json:"rawPayloadHash"`
	// WhisperTimestamp is a timestamp of a Whisper envelope.
	WhisperTimestamp int64 `json:"whisperTimestamp"`
	// From is a public key of the author of the message.
	From hexutilSQL `json:"from"`
	// To is a public key of the recipient unless it's a public message then it's empty.
	To hexutilSQL `json:"to,omitempty"`
	// BEGIN: fields from protocol.Message.
	Content       string `json:"content"`
	ContentType   string `json:"contentType"`
	Timestamp     int64  `json:"timestamp"`
	ChatID        string `json:"chatId"`
	MessageType   string `json:"messageType,omitempty"`
	MessageStatus string `json:"messageStatus,omitempty"`
	ClockValue    int64  `json:"clockValue"`
	// END
	Username       string `json:"username,omitempty"`
	RetryCount     int    `json:"retryCount"`
	Show           bool   `json:"show"` // default true
	Seen           bool   `json:"seen"`
	OutgoingStatus string `json:"outgoingStatus,omitempty"`
}
