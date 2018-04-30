package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
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
		Timestamp:   time.Now().Unix() * 1000,
	}
}

// ToPayload  converts current struct to a valid payload
func (m *Msg) ToPayload() string {
	message := fmt.Sprintf(messagePayloadFormat,
		m.Text,
		m.Timestamp*100,
		m.Timestamp)

	return rawrChatMessage(message)
}

// MessageFromPayload : TODO ...
func MessageFromPayload(payload string) (*Msg, error) {
	message, err := unrawrChatMessage(payload)
	if err != nil {
		return nil, err
	}
	var x []interface{}
	json.Unmarshal([]byte(message), &x)
	if len(x) < 1 {
		return nil, errors.New("unsupported message type")
	}
	if x[0].(string) != "~#c4" {
		return nil, errors.New("unsupported message type")
	}
	properties := x[1].([]interface{})

	return &Msg{
		From:      "TODO : someone",
		Text:      properties[0].(string),
		Timestamp: int64(properties[3].(float64)),
		Raw:       string(message),
	}, nil
}
