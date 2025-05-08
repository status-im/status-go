package messaging

import (
	"crypto/ecdsa"

	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/transport"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type Core struct {
	transport              *transport.Transport
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig
	logger                 *zap.Logger
}

func NewCore(waku wakutypes.Waku, identity *ecdsa.PrivateKey, persistence Persistence, options ...Options) (*Core, error) {
	core := &Core{
		envelopesMonitorConfig: &transport.EnvelopesMonitorConfig{
			IsMailserver: func(types.EnodeID) bool { return false },
		},
	}

	for _, option := range options {
		option(core)
	}

	if core.logger == nil {
		core.logger = zap.NewNop()
	}
	core.envelopesMonitorConfig.Logger = core.logger

	var err error
	core.transport, err = transport.NewTransport(
		waku,
		identity,
		&keysPersistenceAdapter{p: persistence},
		&processedMessageIDsCacheAdapter{p: persistence},
		core.envelopesMonitorConfig,
		core.logger,
	)
	if err != nil {
		return nil, err
	}

	return core, nil
}

func (m *Core) API() *API {
	return NewAPI(m.transport)
}

type Options func(*Core)

func WithLogger(logger *zap.Logger) Options {
	return func(c *Core) {
		c.logger = logger
	}
}

func WithEnvelopeEventsConfig(config *EnvelopeEventsConfig) Options {
	return func(c *Core) {
		if config != nil {
			c.envelopesMonitorConfig.EnvelopeEventsHandler = config.EnvelopeEventsHandler
			c.envelopesMonitorConfig.MaxAttempts = config.MaxMessageDeliveryAttempts
			c.envelopesMonitorConfig.AwaitOnlyMailServerConfirmations = config.MailServerConfirmations
		}
	}
}
