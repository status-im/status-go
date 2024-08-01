package publish

import (
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

// PublishFn represents a function that will publish a message.
type PublishFn = func(envelope *protocol.Envelope, logger *zap.Logger) error
