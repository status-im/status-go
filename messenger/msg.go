package messenger

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

// Msg is a structure used by Subscribers and Publish().
type Msg struct {
	ID          string `json:"id"`
	From        string `json:"from"`
	Text        string `json:"text"`
	ChannelName string `json:"channel"`
	Timestamp   int64  `json:"ts"`
	Raw         string `json:"-"`
}

// NewMsg : Creates a new Msg with a generated UUID
func NewMsg(from, text, channel string) *Msg {
	return &Msg{
		ID:          newUUID(),
		From:        from,
		Text:        text,
		ChannelName: channel,
		Timestamp:   time.Now().Unix(),
	}
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

// newUUID generates a random UUID according to RFC 4122
func newUUID() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		panic(err)
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func rawrChatMessage(raw string) string {
	bytes := []byte(raw)

	return fmt.Sprintf("0x%s", hex.EncodeToString(bytes))
}

func unrawrChatMessage(message string) ([]byte, error) {
	return hex.DecodeString(message[2:])
}
