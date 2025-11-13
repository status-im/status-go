package backend

import (
	"go.uber.org/zap"
)

type config struct {
	logger *zap.Logger

	// Media service options
	mediaServiceEnabled      bool
	mediaServerAddress       string
	mediaServerAdvertizeHost string
	mediaServerAdvertizePort int

	// Centralized metrics
	mixpanelAppID string
	mixpanelToken string
	sentryDSN     string

	// Override log level
	LogLevel string `json:"logLevel"`

	//
	LogsDir string `json:"logsDir"`

	//
	APILoggingEnabled bool `json:"apiLoggingEnabled"`

	// Add
	MetricsEnabled bool   `json:"metricsEnabled"`
	MetricsAddress string `json:"metricsAddress"`

	// WakuFleetsConfigFilePath specifies the file path for configuring fleets supported by the app.
	// File structure must be as params.FleetsMap.
	// When successfully loaded, overrides all hard-coded fleets with file contents.
	WakuFleetsConfigFilePath string `json:"wakuFleetsConfigFilePath"`

	// PushNotificationsConfigFilePath specifies the file path for configuring push notifications servers.
	// File structure must be as params.PushNotificationsFleet.
	PushFleetsConfigFilePath string `json:"pushFleetsConfigFilePath"`
}

type Option func(*config)

func WithLogger(logger *zap.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

func WithMediaServer(address string, enableTLS *bool, advertizeHost string, advertizePort int) Option {
	return func(c *config) {
		c.mediaServiceEnabled = true
		c.mediaServerAddress = address
		c.mediaServerAdvertizeHost = advertizeHost
		c.mediaServerAdvertizePort = advertizePort
	}
}

func WithCentralizedMetrics(mixpanelAppID string, mixpanelToken string) Option {
	return func(c *config) {
		c.mixpanelAppID = mixpanelAppID
		c.mixpanelToken = mixpanelToken
	}
}

func WithSentry(sentryDSN string) Option {
	return func(c *config) {
		c.sentryDSN = sentryDSN
	}
}

func WithWakuFleets(wakuFleetsConfigFilePath string, pushFleetsConfigFilePath string) Option {
	return func(c *config) {
		c.WakuFleetsConfigFilePath = wakuFleetsConfigFilePath
		c.PushFleetsConfigFilePath = pushFleetsConfigFilePath
	}
}

func WithMetrics(address string) Option {
	return func(c *config) {
		c.MetricsEnabled = true
		c.MetricsAddress = address
	}
}

func WithLogLevel(logLevel string) Option {
	return func(c *config) {
		c.LogLevel = logLevel
	}
}

func WithLogsDir(logsDir string) Option {
	return func(c *config) {
		c.LogsDir = logsDir
	}
}

func WithAPILogging() Option {
	return func(c *config) {
		c.APILoggingEnabled = true
	}
}
