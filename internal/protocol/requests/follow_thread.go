package requests

import (
	"errors"
	"strings"
)

var ErrFollowThreadInvalidRequest = errors.New("follow-thread: chat ID and thread ID are required")

type FollowThread struct {
	ChatID   string `json:"chatId"`
	ThreadID string `json:"threadId"`
}

func (r *FollowThread) Validate() error {
	if r == nil {
		return ErrFollowThreadInvalidRequest
	}
	return validateThreadFollowRequest(r.ChatID, r.ThreadID)
}

type UnfollowThread struct {
	ChatID   string `json:"chatId"`
	ThreadID string `json:"threadId"`
}

func (r *UnfollowThread) Validate() error {
	if r == nil {
		return ErrFollowThreadInvalidRequest
	}
	return validateThreadFollowRequest(r.ChatID, r.ThreadID)
}

func validateThreadFollowRequest(chatID, threadID string) error {
	if strings.TrimSpace(chatID) == "" || strings.TrimSpace(threadID) == "" {
		return ErrFollowThreadInvalidRequest
	}
	return nil
}
