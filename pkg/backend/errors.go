package backend

import (
	"errors"
)

// errors
var (
	ErrNodeRunning   = errors.New("node is already running")
	ErrNoRunningNode = errors.New("there is no running node")
)
