package requests

import (
	"errors"
	"strings"

	"github.com/status-im/status-go/internal/protocol/common"
)

var (
	ErrEditThreadInvalidRequest = errors.New("edit-thread: invalid request")
	ErrEditThreadInvalidChatID  = errors.New("edit-thread: invalid chat id")
	ErrEditThreadInvalidID      = errors.New("edit-thread: invalid thread id")
	ErrEditThreadInvalidName    = errors.New("edit-thread: invalid name")
	ErrEditThreadNameTooLong    = errors.New("edit-thread: name exceeds 50 characters")
)

type EditThread struct {
	ChatID   string  `json:"chatId"`
	ThreadID string  `json:"threadId"`
	Name     *string `json:"name,omitempty"`
}

func (e *EditThread) Validate() error {
	if e == nil || e.Name == nil {
		return ErrEditThreadInvalidRequest
	}
	if strings.TrimSpace(e.ChatID) == "" {
		return ErrEditThreadInvalidChatID
	}
	if strings.TrimSpace(e.ThreadID) == "" {
		return ErrEditThreadInvalidID
	}
	name, overlong := common.NormalizeThreadName(*e.Name, false)
	if name == "" {
		return ErrEditThreadInvalidName
	}
	if overlong {
		return ErrEditThreadNameTooLong
	}
	return nil
}
