package messaging

import (
	"database/sql"

	"go.uber.org/zap"

	"github.com/libp2p/go-libp2p/core/peer"

	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
)

type config struct {
	logger                          *zap.Logger
	envelopesMonitorConfig          *transport.EnvelopesMonitorConfig
	metricsEnabled                  bool
	onHistoricMessagesRequestFailed func([]byte, peer.AddrInfo, error)
	onPeerStats                     func(types.ConnStatus)
	persistence                     Persistence
}

func newConfig(options ...Options) *config {
	config := &config{
		logger: zap.NewNop(),
		envelopesMonitorConfig: &transport.EnvelopesMonitorConfig{
			IsMailserver: func(ethtypes.EnodeID) bool {
				return false
			},
		},
		metricsEnabled:                  false,
		onHistoricMessagesRequestFailed: func([]byte, peer.AddrInfo, error) {},
		onPeerStats:                     func(types.ConnStatus) {},
	}

	for _, option := range options {
		option(config)
	}

	return config
}

type Options func(*config)

func WithLogger(logger *zap.Logger) Options {
	return func(c *config) {
		c.logger = logger
		c.envelopesMonitorConfig.Logger = logger
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

func WithMetrics(enabled bool) Options {
	return func(c *config) {
		c.metricsEnabled = enabled
	}
}

func WithHistoricMessagesRequestFailedHandler(onHistoricMessagesRequestFailed func([]byte, peer.AddrInfo, error)) Options {
	return func(c *config) {
		c.onHistoricMessagesRequestFailed = onHistoricMessagesRequestFailed
	}
}

func WithPeerStatsHandler(onPeerStats func(types.ConnStatus)) Options {
	return func(c *config) {
		c.onPeerStats = onPeerStats
	}
}

// WithSQLitePersistence sets up the messaging persistence using internal SQLite implementation.
// Migrations must be applied beforehand. See SQLiteMigrate.
func WithSQLitePersistence(db *sql.DB) Options {
	return func(c *config) {
		c.persistence = newSQLitePersistence(db)
	}
}

// WithPersistence sets up the messaging persistence using the provided implementation.
func WithPersistence(persistence Persistence) Options {
	return func(c *config) {
		c.persistence = persistence
	}
}
