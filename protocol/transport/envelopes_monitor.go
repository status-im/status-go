package transport

import (
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
)

type EnvelopesMonitorConfig struct {
	EnvelopeEventsHandler          EnvelopeEventsHandler
	MaxAttempts                    int
	MailserverConfirmationsEnabled bool
	IsMailserver                   func(types.EnodeID) bool
	Logger                         *zap.Logger
}

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent([][]byte)
	EnvelopeExpired([][]byte, error)
	MailServerRequestCompleted(types.Hash, types.Hash, []byte, error)
	MailServerRequestExpired(types.Hash)
}
