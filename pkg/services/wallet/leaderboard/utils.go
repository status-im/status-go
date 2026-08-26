package leaderboard

import (
	"fmt"
	"strings"
)

const (
	MarketProxyHostSuffix = "market.status.im"

	// DefaultCurrency is what the proxy serves when no conversion is requested.
	DefaultCurrency = "usd"
)

// normalizeCurrency brings a display currency code into the form the proxy
// expects (lowercase). Clients send it in whatever case they keep it in:
// status-desktop uppercases it, the settings DB stores it lowercase.
func normalizeCurrency(currency string) string {
	currency = strings.ToLower(strings.TrimSpace(currency))
	if currency == "" {
		return DefaultCurrency
	}
	return currency
}

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
