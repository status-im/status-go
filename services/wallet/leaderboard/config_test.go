package leaderboard

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
)

func TestServiceConfigValidate(t *testing.T) {
	{
		// Zero intervals
		config := NewLeaderbordConfig(params.MarketDataProxyConfig{
			Url:                     "https://example.com",
			Login:                   "user",
			Password:                "pass",
			FullDataRefreshInterval: 0,
			PriceRefreshInterval:    0,
		})

		config.Validate()

		require.Equal(t, defaultFullDataInterval, config.FullDataInterval)
		require.Equal(t, defaultPriceUpdateInterval, config.PriceUpdateInterval)
		require.Equal(t, "https://example.com", config.ProxyURL)
		require.Equal(t, "user", config.Login)
		require.Equal(t, "pass", config.Password)
		require.Equal(t, true, config.AllowGzip)
		require.Equal(t, true, config.AllowETag)
	}

	{
		// Negative intervals
		config := NewLeaderbordConfig(params.MarketDataProxyConfig{
			Url:                     "https://example.com",
			Login:                   "user",
			Password:                "pass",
			FullDataRefreshInterval: -5,
			PriceRefreshInterval:    -5,
		})

		config.Validate()

		require.Equal(t, defaultFullDataInterval, config.FullDataInterval)
		require.Equal(t, defaultPriceUpdateInterval, config.PriceUpdateInterval)
		require.Equal(t, "https://example.com", config.ProxyURL)
		require.Equal(t, "user", config.Login)
		require.Equal(t, "pass", config.Password)
		require.Equal(t, true, config.AllowGzip)
		require.Equal(t, true, config.AllowETag)
	}

	{
		// Custom intervals
		config := NewLeaderbordConfig(params.MarketDataProxyConfig{
			Url:                     "https://example.com",
			Login:                   "user",
			Password:                "pass",
			FullDataRefreshInterval: 50,
			PriceRefreshInterval:    65,
		})

		config.Validate()

		require.Equal(t, 50, config.FullDataInterval)
		require.Equal(t, 65, config.PriceUpdateInterval)
		require.Equal(t, "https://example.com", config.ProxyURL)
		require.Equal(t, "user", config.Login)
		require.Equal(t, "pass", config.Password)
		require.Equal(t, true, config.AllowGzip)
		require.Equal(t, true, config.AllowETag)
	}
}
