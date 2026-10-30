package requests

import (
	"errors"
	"strings"
)

var ErrDeleteThreadInvalidRequest = errors.New("delete-thread: chat ID and thread ID are required")

type DeleteThread struct {
	ChatID   string `json:"chatId"`
	ThreadID string `json:"threadId"`
}

func (r *DeleteThread) Validate() error {
	if r == nil || strings.TrimSpace(r.ChatID) == "" || strings.TrimSpace(r.ThreadID) == "" {
		return ErrDeleteThreadInvalidRequest
	}
	return nil
}
