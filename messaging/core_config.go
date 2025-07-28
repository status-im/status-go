package messaging

import (
	"go.uber.org/zap"

	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
)

type config struct {
	logger                 *zap.Logger
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig
}

type Options func(*config)

func WithLogger(logger *zap.Logger) Options {
	return func(c *config) {
		c.logger = logger
	}
}

func WithEnvelopeEventsConfig(econf *types.EnvelopeEventsConfig) Options {
	return func(c *config) {
		if econf != nil {
			c.envelopesMonitorConfig.EnvelopeEventsHandler = econf.EnvelopeEventsHandler
			c.envelopesMonitorConfig.MaxAttempts = econf.MaxMessageDeliveryAttempts
			c.envelopesMonitorConfig.AwaitOnlyMailServerConfirmations = econf.MailServerConfirmations
		}
	}
}
