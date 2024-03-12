package requests

import (
	"errors"
	"time"
)

var ErrInvalidMuteChatParams = errors.New("mute-chat: invalid params")

type MutingVariation int

type MuteChat struct {
	ChatID          string
	MutedType       MutingVariation
	CustomTimestamp int64 // used if MutedType is Custom
}

func (a *MuteChat) Validate() error {
	if len(a.ChatID) == 0 {
		return ErrInvalidMuteChatParams
	}

	if a.MutedType < 0 {
		return ErrInvalidMuteChatParams
	}

	if a.MutedType == 8 && a.CustomTimestamp <= 0 && time.Now().Unix() > a.CustomTimestamp {
		return ErrInvalidMuteChatParams
	}

	return nil
}
