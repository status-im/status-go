//go:build gowaku_no_rln
// +build gowaku_no_rln

package leaderboard

import (
	"encoding/json"
	"errors"
	"os"
)

// ServiceConfig defines the configuration for the market data service
type ServiceConfig struct {
	// API connection settings
	ProxyURL string `json:"proxy_url"`
	Login    string `json:"login"`
	Password string `json:"password"`

	// Refresh intervals (in seconds)
	FullDataInterval    int `json:"full_data_interval"`
	PriceUpdateInterval int `json:"price_update_interval"`

	// Feature flags
	AllowGzip bool `json:"allow_gzip"`
	AllowETag bool `json:"allow_etag"`
}

// LoadConfig loads the service configuration from a JSON file
func LoadConfig(path string) (*ServiceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ServiceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Validate the config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *ServiceConfig) Validate() error {
	if c.ProxyURL == "" {
		return errors.New("proxy URL is required")
	}
	if c.Login == "" {
		return errors.New("login is required")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}

	// Set default refresh intervals if not provided
	if c.FullDataInterval <= 0 {
		c.FullDataInterval = 10 // Default full data refresh interval in seconds
	}

	if c.PriceUpdateInterval <= 0 {
		c.PriceUpdateInterval = 1 // Default price update interval in seconds
	}

	// Set default feature flags if not explicitly set
	if !c.AllowGzip {
		c.AllowGzip = true // Enable gzip by default
	}

	if !c.AllowETag {
		c.AllowETag = true // Enable ETag by default
	}

	return nil
}

// SaveConfig saves the configuration to a JSON file
func SaveConfig(config *ServiceConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
