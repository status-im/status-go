package requests

import (
	"errors"
)

var ErrSyncChatInvalidID = errors.New("sync-chat: invalid id")

type SyncChat struct {
	ID string `json:"id"`
}

func (c *SyncChat) Validate() error {
	if len(c.ID) == 0 {
		return ErrSyncChatInvalidID
	}

	return nil
}
