package leaderboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMarketProxyHost(t *testing.T) {
	tests := []struct {
		name        string
		customUrl   string
		stageName   string
		expectedUrl string
	}{
		{
			name:        "Empty custom URL with test stage",
			customUrl:   "",
			stageName:   "test",
			expectedUrl: "https://test.market.status.im",
		},
		{
			name:        "Empty custom URL with prod stage - should still use test",
			customUrl:   "",
			stageName:   "prod",
			expectedUrl: "https://test.market.status.im",
		},
		{
			name:        "Empty custom URL with random stage - should still use test",
			customUrl:   "",
			stageName:   "staging",
			expectedUrl: "https://test.market.status.im",
		},
		{
			name:        "Custom URL provided - should use custom URL",
			customUrl:   "https://custom-market.example.com",
			stageName:   "test",
			expectedUrl: "https://custom-market.example.com",
		},
		{
			name:        "Custom URL with trailing slash - should trim",
			customUrl:   "https://custom-market.example.com/",
			stageName:   "test",
			expectedUrl: "https://custom-market.example.com",
		},
		{
			name:        "Custom URL with multiple trailing slashes - should trim all",
			customUrl:   "https://custom-market.example.com///",
			stageName:   "test",
			expectedUrl: "https://custom-market.example.com",
		},
		{
			name:        "Custom localhost URL",
			customUrl:   "http://localhost:8080",
			stageName:   "dev",
			expectedUrl: "http://localhost:8080",
		},
		{
			name:        "Empty stage name with empty custom URL",
			customUrl:   "",
			stageName:   "",
			expectedUrl: "https://test.market.status.im",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMarketProxyHost(tt.customUrl, tt.stageName)
			require.Equal(t, tt.expectedUrl, result)
		})
	}
}
