package wallet

import (
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
)

// TODO(status-im/status-app#23177): remove with the test proxy fallback.

// fallbackMarketProxyConfigs returns the config of the market proxy of every
// fallback stage. Their clients go after the other market data providers.
func fallbackMarketProxyConfigs(config params.MarketDataProxyConfig) []params.MarketDataProxyConfig {
	var configs []params.MarketDataProxyConfig
	for _, stage := range params.FallbackProxyStages(config.StageName, config.UrlOverride.Reveal()) {
		config.StageName = stage
		config.User, config.Password = security.SensitiveString{}, security.SensitiveString{}
		config.UsePuzzleAuth = true
		configs = append(configs, config)
	}
	return configs
}

// fallbackNftProxyConfigs returns the config of the NFT proxy of every fallback
// stage. Their clients go after the other collectible providers.
func fallbackNftProxyConfigs(config params.NftProxyConfig) []params.NftProxyConfig {
	var configs []params.NftProxyConfig
	for _, stage := range params.FallbackProxyStages(config.StageName, config.UrlOverride.Reveal()) {
		config.StageName = stage
		config.User, config.Password = security.SensitiveString{}, security.SensitiveString{}
		config.UsePuzzleAuth = true
		configs = append(configs, config)
	}
	return configs
}
