package messenger

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

// supportedMessage : check if the message type is supported
func supportedMessage(msgType string) bool {
	_, ok := map[string]string{
		"~#c1": "newContactKeyPrefix",
		"~#c2": "contactRequestPrefix",
		"~#c3": "confirmedContactRequestPrefix",
		"~#c4": "standardMessagePrefix",
		"~#c5": "seenPrefix",
		"~#c6": "contactUpdatePrefix",
	}[msgType]

	return ok
}

// Msg is a structure used by Subscribers and Publish().
type Msg struct {
	From        string `json:"from"`
	Text        string `json:"text"`
	ChannelName string `json:"channel"`
	Timestamp   int64  `json:"ts"`
	Raw         string `json:"-"`
	Type        string `json:"-"`
}

// NewMsg : Creates a new Msg with a generated UUID
func NewMsg(from, text, channel string) *Msg {
	return &Msg{
		From:        from,
		Text:        text,
		ChannelName: channel,
		Timestamp:   time.Now().Unix(),
	}
}

// ID : get the message id
func (m *Msg) ID() string {
	return fmt.Sprintf("%X", sha3.Sum256([]byte(m.Raw)))
}

// ToPayload  converts current struct to a valid payload
func (m *Msg) ToPayload() string {
	message := fmt.Sprintf(messagePayloadMsg,
		m.Text,
		m.Timestamp*100,
		m.Timestamp)
	println(message)

	return rawrChatMessage(message)
}

func rawrChatMessage(raw string) string {
	bytes := []byte(raw)

	return fmt.Sprintf("0x%s", hex.EncodeToString(bytes))
}

func unrawrChatMessage(message string) ([]byte, error) {
	return hex.DecodeString(message[2:])
}
