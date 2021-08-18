package requests

import (
	"errors"
)

type OrderingDirection int

const (
	OrderingDirectionDesc OrderingDirection = iota
	OrderingDirectionAsc
)

var ErrMessagesByChatIDInvalidChatID = errors.New("messages-by-chat-id: invalid chat id")
var ErrMessagesByChatIDInvalidLimit = errors.New("messages-by-chat-id: invalid limit")

type MessagesByChatID struct {
	ChatID    string            `json:"chatId"`
	Cursor    string            `json:"cursor"`
	Limit     int               `json:"limit"`
	Direction OrderingDirection `json:"direction"`
}

func (m *MessagesByChatID) Validate() error {
	if len(m.ChatID) == 0 {
		return ErrMessagesByChatIDInvalidChatID
	}

	if m.Limit < 1 {
		return ErrMessagesByChatIDInvalidLimit
	}

	return nil
}
