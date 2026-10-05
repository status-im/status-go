package requests

import (
	"errors"
	"strings"

	"github.com/status-im/status-go/internal/protocol/common"
)

var ErrStartThreadFromNewMessageInvalidMessage = errors.New("start-thread-from-new-message: invalid message")
var ErrStartThreadFromNewMessageInvalidChatID = errors.New("start-thread-from-new-message: invalid chat id")
var ErrStartThreadFromNewMessageInvalidThreadID = errors.New("start-thread-from-new-message: thread id must be empty")
var ErrStartThreadFromNewMessageInvalidResponseTo = errors.New("start-thread-from-new-message: response to must be empty")

type StartThreadFromNewMessage struct {
	Message    *common.Message `json:"message"`
	ThreadName string          `json:"threadName,omitempty"`
}

func (c *StartThreadFromNewMessage) Validate() error {
	if c == nil || c.Message == nil || c.Message.ChatMessage == nil {
		return ErrStartThreadFromNewMessageInvalidMessage
	}

	if strings.TrimSpace(c.Message.ChatId) == "" {
		return ErrStartThreadFromNewMessageInvalidChatID
	}

	if c.Message.GetThreadId() != "" {
		return ErrStartThreadFromNewMessageInvalidThreadID
	}

	if c.Message.ResponseTo != "" {
		return ErrStartThreadFromNewMessageInvalidResponseTo
	}

	return nil
}
