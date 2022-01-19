package settings

import (
	"errors"
)

var (
	ErrInvalidValueType = errors.New("invalid value type")
)

type SyncHandler func(interface{}) error

func SyncCurrency(value interface{}) error {
	if _, ok := value.(string); !ok {
		return ErrInvalidValueType
	}

	return nil
}