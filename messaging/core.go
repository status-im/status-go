package messaging

import (
	"crypto/ecdsa"

	"go.uber.org/zap"

	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type Core struct {
	waku                   wakutypes.Waku
	transport              *transport.Transport
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig
	logger                 *zap.Logger
}

func NewCore(waku wakutypes.Waku, identity *ecdsa.PrivateKey, persistence types.Persistence, options ...Options) (*Core, error) {
	core := &Core{
		waku: waku,
		envelopesMonitorConfig: &transport.EnvelopesMonitorConfig{
			IsMailserver: func(ethtypes.EnodeID) bool { return false },
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
		&adapters.KeysPersistence{P: persistence},
		&adapters.ProcessedMessageIDsCache{P: persistence},
		core.envelopesMonitorConfig,
		core.logger,
	)
	if err != nil {
		return nil, err
	}

	return core, nil
}

func (m *Core) API() *API {
	return NewAPI(m.waku, m.transport)
}

type Options func(*Core)

func WithLogger(logger *zap.Logger) Options {
	return func(c *Core) {
		c.logger = logger
	}
}

func WithEnvelopeEventsConfig(config *types.EnvelopeEventsConfig) Options {
	return func(c *Core) {
		if config != nil {
			c.envelopesMonitorConfig.EnvelopeEventsHandler = config.EnvelopeEventsHandler
			c.envelopesMonitorConfig.MaxAttempts = config.MaxMessageDeliveryAttempts
			c.envelopesMonitorConfig.AwaitOnlyMailServerConfirmations = config.MailServerConfirmations
		}
	}
}
