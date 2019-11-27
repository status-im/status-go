package shhext

import (
	"time"

	"github.com/status-im/status-go/eth-node/types"
)

const (
	// WhisperTimeAllowance is needed to ensure that we won't miss envelopes that were
	// delivered to mail server after we made a request.
	WhisperTimeAllowance = 20 * time.Second
)

// TopicRequest defines what user has to provide.
type TopicRequest struct {
	Topic    types.TopicType
	Duration time.Duration
}
