package requests

import (
	"errors"
)

// Deprecated: profile chats are deprecated
var ErrCreateProfileChatInvalidID = errors.New("create-public-chat: invalid id")

// Deprecated: profile chats are deprecated
type CreateProfileChat struct {
	ID string `json:"id"`
}

// Deprecated: profile chats are deprecated
func (c *CreateProfileChat) Validate() error {
	return errors.New("profile chats are deprecated")
	if len(c.ID) == 0 {
		return ErrCreateProfileChatInvalidID
	}

	return nil
}
