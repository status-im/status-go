package leaderboard

import (
	"testing"
	"time"

	"github.com/status-im/status-go/internal/security"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
)

func TestServiceConfigValidate(t *testing.T) {
	{
		// Zero intervals
		config := NewLeaderbordConfig(params.MarketDataProxyConfig{
			Url:                     "https://example.com",
			User:                    security.NewSensitiveString("user"),
			Password:                security.NewSensitiveString("pass"),
			FullDataRefreshInterval: 0,
			PriceRefreshInterval:    0,
		})

		require.Equal(t, defaultFullDataInterval, config.FullDataInterval)
		require.Equal(t, defaultPriceUpdateInterval, config.PriceUpdateInterval)
		require.Equal(t, "https://example.com", config.ProxyURL)
		require.True(t, config.User.Equal("user"))
		require.True(t, config.Password.Equal("pass"))
		require.Equal(t, true, config.AllowGzip)
		require.Equal(t, true, config.AllowETag)
	}

	{
		// Negative intervals
		config := NewLeaderbordConfig(params.MarketDataProxyConfig{
			Url:                     "https://example.com",
			User:                    security.NewSensitiveString("user"),
			Password:                security.NewSensitiveString("pass"),
			FullDataRefreshInterval: -5,
			PriceRefreshInterval:    -5,
		})

		require.Equal(t, defaultFullDataInterval, config.FullDataInterval)
		require.Equal(t, defaultPriceUpdateInterval, config.PriceUpdateInterval)
		require.Equal(t, "https://example.com", config.ProxyURL)
		require.True(t, config.User.Equal("user"))
		require.True(t, config.Password.Equal("pass"))
		require.Equal(t, true, config.AllowGzip)
		require.Equal(t, true, config.AllowETag)
	}

	{
		// Custom intervals
		config := NewLeaderbordConfig(params.MarketDataProxyConfig{
			Url:                     "https://example.com",
			User:                    security.NewSensitiveString("user"),
			Password:                security.NewSensitiveString("pass"),
			FullDataRefreshInterval: 50,
			PriceRefreshInterval:    65,
		})

		require.Equal(t, 50*time.Second, config.FullDataInterval)
		require.Equal(t, 65*time.Second, config.PriceUpdateInterval)
		require.Equal(t, "https://example.com", config.ProxyURL)
		require.True(t, config.User.Equal("user"))
		require.True(t, config.Password.Equal("pass"))
		require.Equal(t, true, config.AllowGzip)
		require.Equal(t, true, config.AllowETag)
	}
}
