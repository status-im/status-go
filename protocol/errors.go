package protocol

import (
	"github.com/pkg/errors"
)

var (
	ErrChatIDEmpty     = errors.New("chat ID is empty")
	ErrChatNotFound    = errors.New("can't find chat")
	ErrNotImplemented  = errors.New("not implemented")
	ErrContactNotFound = errors.New("contact not found")
)
