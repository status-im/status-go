package db

import (
	"github.com/status-im/status-go/params"
)

// Deprecated: fillDeprecatedURLs populates the `original_rpc_url`, `original_fallback_url`, `rpc_url`,
// `fallback_url`, `defaultRpcUrl`, `defaultFallbackURL`, and `defaultFallbackURL2` fields.
// Keep for backwrad compatibility until it's fully integrated
func FillDeprecatedURLs(network *params.Network, providers []params.RpcProvider) {
	var embeddedDirect []params.RpcProvider
	var embeddedProxy []params.RpcProvider
	var userProviders []params.RpcProvider

	// Categorize providers
	for _, provider := range providers {
		switch provider.Type {
		case params.EmbeddedDirectProviderType:
			embeddedDirect = append(embeddedDirect, provider)
		case params.EmbeddedProxyProviderType:
			embeddedProxy = append(embeddedProxy, provider)
		case params.UserProviderType:
			userProviders = append(userProviders, provider)
		}
	}

	// Set original_*_url fields based on EmbeddedDirectProviderType providers
	if len(embeddedDirect) > 0 {
		network.OriginalRPCURL = embeddedDirect[0].URL
		if len(embeddedDirect) > 1 {
			network.OriginalFallbackURL = embeddedDirect[1].URL
		}
	}

	// Set rpc_url and fallback_url based on User providers or EmbeddedDirectProviderType if no User providers exist
	if len(userProviders) > 0 {
		network.RPCURL = userProviders[0].URL
		if len(userProviders) > 1 {
			network.FallbackURL = userProviders[1].URL
		}
	} else {
		// Default to EmbeddedDirectProviderType providers if no User providers exist
		network.RPCURL = network.OriginalRPCURL
		network.FallbackURL = network.OriginalFallbackURL
	}

	// Set default_*_url fields based on EmbeddedProxyProviderType providers
	if len(embeddedProxy) > 0 {
		network.DefaultRPCURL = embeddedProxy[0].URL
		if len(embeddedProxy) > 1 {
			network.DefaultFallbackURL = embeddedProxy[1].URL
		}
		if len(embeddedProxy) > 2 {
			network.DefaultFallbackURL2 = embeddedProxy[2].URL
		}
	}
}
