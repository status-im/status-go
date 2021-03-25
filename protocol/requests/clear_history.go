package requests

import (
	"errors"
)

var ErrClearHistoryInvalidID = errors.New("clear-history: invalid id")

type ClearHistory struct {
	ID string
}

func (j *ClearHistory) Validate() error {
	if len(j.ID) == 0 {
		return ErrClearHistoryInvalidID
	}

	return nil
}
