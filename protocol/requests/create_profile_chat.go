package requests

import (
	"errors"
)

var ErrCreateProfileChatInvalidID = errors.New("create-public-chat: invalid id")

type CreateProfileChat struct {
	ID string
}

func (j *CreateProfileChat) Validate() error {
	if len(j.ID) == 0 {
		return ErrCreateProfileChatInvalidID
	}

	return nil
}
