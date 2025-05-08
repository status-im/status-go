package leaderboard

import (
	"github.com/status-im/status-go/params"
)

const (
	defaultFullDataInterval    = 10 // Default full data refresh interval in seconds
	defaultPriceUpdateInterval = 1  // Default price update interval in seconds
)

// ServiceConfig defines the configuration for the market data service
type ServiceConfig struct {
	// API connection settings
	ProxyURL string
	User     string
	Password string

	// Refresh intervals (in seconds)
	FullDataInterval    int
	PriceUpdateInterval int

	// Feature flags
	AllowGzip bool
	AllowETag bool
}

// Validate checks if the configuration is valid
func (c *ServiceConfig) Validate() {
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
		FullDataInterval:    config.FullDataRefreshInterval,
		PriceUpdateInterval: config.PriceRefreshInterval,
		AllowGzip:           true,
		AllowETag:           true,
	}

	// Validate the configuration
	serviceConfig.Validate()

	return serviceConfig
}
