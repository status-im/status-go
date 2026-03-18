package messaging

import (
	"database/sql"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

type config struct {
	logger                 *zap.Logger
	tracer                 trace.Tracer
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig
	metricsEnabled         bool
	persistence            Persistence
	missingDepsObserver    MissingDependenciesObserver
}

func newConfig(options ...Options) *config {
	config := &config{
		logger: zap.NewNop(),
		tracer: trace.NewNoopTracer(),
		envelopesMonitorConfig: &transport.EnvelopesMonitorConfig{
			IsMailserver: func(wakutypes.EnodeID) bool {
				return false
			},
		},
		metricsEnabled: false,
	}

	for _, option := range options {
		option(config)
	}

	return config
}

type Options func(*config)

// MissingDependenciesObserver allows tests to observe SDS missing dependency
// events before the core triggers background fetching.
type MissingDependenciesObserver func(messageID string, missingDeps []string, channelID string)

func WithLogger(logger *zap.Logger) Options {
	return func(c *config) {
		c.logger = logger
		c.envelopesMonitorConfig.Logger = logger
	}
}

func WithTracer(tracer trace.Tracer) Options {
	return func(c *config) {
		c.tracer = tracer
	}
}

func WithEnvelopeEventsConfig(econf *types2.EnvelopeEventsConfig) Options {
	return func(c *config) {
		if econf != nil {
			c.envelopesMonitorConfig.EnvelopeEventsHandler = econf.EnvelopeEventsHandler
			c.envelopesMonitorConfig.AwaitOnlyMailServerConfirmations = econf.MailServerConfirmations
		}
	}
}

func WithMetrics(enabled bool) Options {
	return func(c *config) {
		c.metricsEnabled = enabled
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

func WithMissingDependenciesObserver(observer MissingDependenciesObserver) Options {
	return func(c *config) {
		c.missingDepsObserver = observer
	}
}
