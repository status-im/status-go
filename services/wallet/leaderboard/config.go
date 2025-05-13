package leaderboard

import (
	"time"

	"github.com/status-im/status-go/params"
)

const (
	defaultFullDataInterval    = 10 * time.Second
	defaultPriceUpdateInterval = 1 * time.Second
)

// ServiceConfig defines the configuration for the market data service
type ServiceConfig struct {
	// API connection settings
	ProxyURL string
	User     string
	Password string

	// Refresh intervals (in seconds)
	FullDataInterval    time.Duration
	PriceUpdateInterval time.Duration

	// Feature flags
	AllowGzip bool
	AllowETag bool
}

// Validate checks if the configuration is valid
func (c *ServiceConfig) setDefaults() {
	// Set default refresh intervals if not provided
	if c.FullDataInterval <= 0 {
		c.FullDataInterval = defaultFullDataInterval
	}

	if c.PriceUpdateInterval <= 0 {
		c.PriceUpdateInterval = defaultPriceUpdateInterval
	}
}

func NewLeaderbordConfig(config params.MarketDataProxyConfig) ServiceConfig {
	// Create a new ServiceConfig instance with default values
	serviceConfig := ServiceConfig{
		ProxyURL:            config.Url,
		User:                config.User,
		Password:            config.Password,
		FullDataInterval:    time.Duration(config.FullDataRefreshInterval) * time.Second,
		PriceUpdateInterval: time.Duration(config.PriceRefreshInterval) * time.Second,
		AllowGzip:           true,
		AllowETag:           true,
	}

	// Validate the configuration
	serviceConfig.setDefaults()

	return serviceConfig
}
