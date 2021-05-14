package requests

import (
	"errors"
)

var ErrCreateProfileChatInvalidID = errors.New("create-public-chat: invalid id")

type CreateProfileChat struct {
	ID string `json:"id"`
}

func (c *CreateProfileChat) Validate() error {
	if len(c.ID) == 0 {
		return ErrCreateProfileChatInvalidID
	}

	return nil
}
