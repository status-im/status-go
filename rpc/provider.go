package rpc

import (
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"

	"github.com/status-im/status-go/params"
)

const (
	ProviderMain                 = "main"
	ProviderFallback             = "fallback"
	ProviderStatusProxy          = "status-proxy"
	ProviderStatusProxyFallback  = ProviderStatusProxy + "-fallback"
	ProviderStatusProxyFallback2 = ProviderStatusProxy + "-fallback2"
)

type Provider struct {
	Key      string
	URL      string
	Auth     string
	Priority int
}

func (p Provider) authenticationNeeded() bool {
	return len(p.Auth) > 0
}

func getProviderPriorityByURL(url string) int {
	// Currently we have 5 providers and we want to use them in the following order:
	// 1. StatusProxy - Node Fleet
	// 2. StatusProxy - Infura
	// 3. Direct Infura
	// 4. StatusProxy - Grove
	// 5. Direct Grove
	if strings.Contains(url, "api.status.im/nodefleet/") || strings.Contains(url, "anvil") {
		return 0
	} else if strings.Contains(url, "api.status.im/infura/") {
		return 1
	} else if strings.Contains(url, "infura.io/") {
		return 2
	} else if strings.Contains(url, "api.status.im/grove/") {
		return 3
	}

	return 4
}

func getProviderConfig(providerConfigs []params.ProviderConfig, providerName string) (params.ProviderConfig, error) {
	for _, providerConfig := range providerConfigs {
		if providerConfig.Name == providerName {
			return providerConfig, nil
		}
	}
	return params.ProviderConfig{}, fmt.Errorf("provider config not found for provider: %s", providerName)
}

func createProvider(key, url, credentials string, providers *[]Provider) {
	priority := getProviderPriorityByURL(url)
	*providers = append(*providers, Provider{
		Key:      key,
		URL:      url,
		Auth:     credentials,
		Priority: priority,
	})
}

func (c *Client) prepareProviders(network *params.Network) []Provider {
	var providers []Provider

	// Retrieve the proxy provider configuration
	proxyProvider, err := getProviderConfig(c.providerConfigs, ProviderStatusProxy)
	if err != nil {
		c.logger.Warn("could not find provider config for status-proxy", zap.Error(err))
	}

	// Add main and fallback providers
	createProvider(ProviderMain, network.RPCURL, "", &providers)
	createProvider(ProviderFallback, network.FallbackURL, "", &providers)

	// If the proxy provider is enabled, add it and its fallback options
	if proxyProvider.Enabled {
		credentials := proxyProvider.User + ":" + proxyProvider.Password
		createProvider(ProviderStatusProxy, network.DefaultRPCURL, credentials, &providers)
		createProvider(ProviderStatusProxyFallback, network.DefaultFallbackURL, credentials, &providers)
		createProvider(ProviderStatusProxyFallback2, network.DefaultFallbackURL2, credentials, &providers)
	}

	// Sort providers by priority
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Priority < providers[j].Priority
	})

	return providers
}
