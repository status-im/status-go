package leaderboard

import (
	"fmt"
	"strings"
)

const (
	MarketProxyHostSuffix = "market.status.im"
)

// GetMarketProxyHost creates market proxy base URL based on stage name
func GetMarketProxyHost(customUrl, stageName string) string {
	if customUrl != "" {
		return strings.TrimRight(customUrl, "/")
	}
	if stageName == "" {
		stageName = "test"
	}
	return fmt.Sprintf("https://%s.%s", stageName, MarketProxyHostSuffix)
}

// GetMarketProxyUrl creates market proxy URL based on stage name with /v1 suffix
func GetMarketProxyUrl(customUrl, stageName string) string {
	baseUrl := GetMarketProxyHost(customUrl, stageName)
	return baseUrl + "/v1"
}
