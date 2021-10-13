package requests

import (
	"errors"
)

var ErrMessageContextInvalidChatID = errors.New("message-context: invalid chat id")
var ErrMessageContextInvalidMessageID = errors.New("message-context: invalid message id")
var ErrMessageContextInvalidLimit = errors.New("message-context: invalid limit")

type MessageContext struct {
	ChatID    string `json:"chatId"`
	MessageID string `json:"messageId"`
	Limit     int    `json:"limit"`
}

func (m *MessageContext) Validate() error {
	if len(m.ChatID) == 0 {
		return ErrMessageContextInvalidChatID
	}

	if len(m.MessageID) == 0 {
		return ErrMessageContextInvalidMessageID
	}

	if m.Limit < 1 {
		return ErrMessageContextInvalidLimit
	}

	return nil
}
