package leaderboard

import (
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
)

// TODO(status-im/status-app#23177): remove with the test proxy fallback.

// fallbackProxyConfigs returns the config of the market proxy of every
// fallback stage. The fetcher asks them after the configured proxy.
func fallbackProxyConfigs(config ServiceConfig) []ServiceConfig {
	var configs []ServiceConfig
	for _, stage := range params.FallbackProxyStages(config.StageName, config.UrlOverride.Reveal()) {
		config.StageName = stage
		config.User, config.Password = security.SensitiveString{}, security.SensitiveString{}
		config.UsePuzzleAuth = true
		configs = append(configs, config)
	}
	return configs
}
