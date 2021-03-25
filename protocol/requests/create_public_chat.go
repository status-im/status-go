package requests

import (
	"errors"
)

var ErrCreatePublicChatInvalidID = errors.New("create-public-chat: invalid id")

type CreatePublicChat struct {
	ID string
}

func (j *CreatePublicChat) Validate() error {
	if len(j.ID) == 0 {
		return ErrCreatePublicChatInvalidID
	}

	return nil
}
