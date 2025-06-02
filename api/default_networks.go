package api

import (
	"fmt"
	"strings"

	"github.com/status-im/status-go/api/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/protocol/requests"
)

const (
	// Host suffixes for providers
	SmartProxyHostSuffix = "eth-rpc.status.im"
	ProxyHostSuffix      = "api.status.im"
)

// ProviderID represents the internal ID of a blockchain provider
type ProviderID = string

// Provider IDs
const (
	StatusSmartProxy = "status-smart-proxy"
	ProxyNodefleet   = "proxy-nodefleet"
	ProxyInfura      = "proxy-infura"
	ProxyGrove       = "proxy-grove"
	Nodefleet        = "nodefleet"
	Infura           = "infura"
	Grove            = "grove"
	DirectInfura     = "direct-infura"
	DirectGrove      = "direct-grove"
	DirectStatus     = "direct-status"
)

// Direct proxy endpoint (1 endpoint per chain/network)
func proxyUrl(stageName, provider, chainName, networkName string) security.SensitiveString {
	return security.NewSensitiveStringPrintf("https://%s.%s/%s/%s/%s/", stageName, ProxyHostSuffix, provider, chainName, networkName)
}

// New eth-rpc-proxy endpoint (provider agnostic)
func getProxyHost(customUrl, stageName string) string {
	if customUrl != "" {
		return strings.TrimRight(customUrl, "/")
	}
	return fmt.Sprintf("https://%s.%s", stageName, SmartProxyHostSuffix)
}

// New eth-rpc-proxy endpoint with smart proxy URL
func smartProxyUrl(proxyHost, chainName, networkName string) security.SensitiveString {
	return security.NewSensitiveStringPrintf("%s/%s/%s/", proxyHost, chainName, networkName)
}

func mainnet(proxyHost, stageName string) params.Network {
	const chainID = common.MainnetChainID
	const chainName = "ethereum"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Proxy providers
		*params.NewProxyProvider(chainID, ProxyNodefleet, proxyUrl(stageName, Nodefleet, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyInfura, proxyUrl(stageName, Infura, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyGrove, proxyUrl(stageName, Grove, chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://mainnet.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://eth.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Ethereum",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://etherscan.io/",
		IconURL:                "network/Network=Ethereum",
		ChainColor:             "#627EEA",
		ShortName:              "eth",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  1,
		Enabled:                true,
		RelatedChainID:         common.SepoliaChainID,
		IsActive:               true,
		IsDeactivatable:        false,
	}
}

func sepolia(proxyHost, stageName string) params.Network {
	const chainID = common.SepoliaChainID
	const chainName = "ethereum"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Proxy providers
		*params.NewProxyProvider(chainID, ProxyNodefleet, proxyUrl(stageName, Nodefleet, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyInfura, proxyUrl(stageName, Infura, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyGrove, proxyUrl(stageName, Grove, chainName, networkName), true),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://sepolia.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://eth-sepolia-testnet.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Sepolia",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia.etherscan.io/",
		IconURL:                "network/Network=Ethereum-test",
		ChainColor:             "#627EEA",
		ShortName:              "eth",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  1,
		Enabled:                true,
		RelatedChainID:         common.MainnetChainID,
		IsActive:               true,
		IsDeactivatable:        false,
	}
}

func optimism(proxyHost, stageName string) params.Network {
	const chainID = common.OptimismChainID
	const chainName = "optimism"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Proxy providers
		*params.NewProxyProvider(chainID, ProxyNodefleet, proxyUrl(stageName, Nodefleet, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyInfura, proxyUrl(stageName, Infura, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyGrove, proxyUrl(stageName, Grove, chainName, networkName), true),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://optimism-mainnet.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://optimism.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Optimism",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://optimistic.etherscan.io",
		IconURL:                "network/Network=Optimism",
		ChainColor:             "#E90101",
		ShortName:              "oeth",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  2,
		Enabled:                true,
		RelatedChainID:         common.OptimismSepoliaChainID,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func optimismSepolia(proxyHost, stageName string) params.Network {
	const chainID = common.OptimismSepoliaChainID
	const chainName = "optimism"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Proxy providers
		*params.NewProxyProvider(chainID, ProxyNodefleet, proxyUrl(stageName, Nodefleet, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyInfura, proxyUrl(stageName, Infura, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyGrove, proxyUrl(stageName, Grove, chainName, networkName), true),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://optimism-sepolia.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://optimism-sepolia-testnet.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Optimism Sepolia",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia-optimism.etherscan.io/",
		IconURL:                "network/Network=Optimism-test",
		ChainColor:             "#E90101",
		ShortName:              "oeth",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         common.OptimismChainID,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func arbitrum(proxyHost, stageName string) params.Network {
	const chainID = common.ArbitrumChainID
	const chainName = "arbitrum"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Proxy providers
		*params.NewProxyProvider(chainID, ProxyNodefleet, proxyUrl(stageName, Nodefleet, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyInfura, proxyUrl(stageName, Infura, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyGrove, proxyUrl(stageName, Grove, chainName, networkName), true),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://arbitrum-mainnet.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://arbitrum-one.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Arbitrum",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://arbiscan.io/",
		IconURL:                "network/Network=Arbitrum",
		ChainColor:             "#51D0F0",
		ShortName:              "arb1",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  2,
		Enabled:                true,
		RelatedChainID:         common.ArbitrumSepoliaChainID,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func arbitrumSepolia(proxyHost, stageName string) params.Network {
	const chainID = common.ArbitrumSepoliaChainID
	const chainName = "arbitrum"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Proxy providers
		*params.NewProxyProvider(chainID, ProxyNodefleet, proxyUrl(stageName, Nodefleet, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyInfura, proxyUrl(stageName, Infura, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyGrove, proxyUrl(stageName, Grove, chainName, networkName), true),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://arbitrum-sepolia.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://arbitrum-sepolia-testnet.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Arbitrum Sepolia",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia-explorer.arbitrum.io/",
		IconURL:                "network/Network=Arbitrum-test",
		ChainColor:             "#51D0F0",
		ShortName:              "arb1",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         common.ArbitrumChainID,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func base(proxyHost, stageName string) params.Network {
	const chainID = common.BaseChainID
	const chainName = "base"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Proxy providers
		*params.NewProxyProvider(chainID, ProxyNodefleet, proxyUrl(stageName, Nodefleet, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyInfura, proxyUrl(stageName, Infura, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyGrove, proxyUrl(stageName, Grove, chainName, networkName), true),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://base-mainnet.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://base.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Base",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://basescan.org",
		IconURL:                "network/Network=Base",
		ChainColor:             "#0052FF",
		ShortName:              "base",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  2,
		Enabled:                true,
		RelatedChainID:         common.BaseSepoliaChainID,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func baseSepolia(proxyHost, stageName string) params.Network {
	const chainID = common.BaseSepoliaChainID
	const chainName = "base"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Proxy providers
		*params.NewProxyProvider(chainID, ProxyNodefleet, proxyUrl(stageName, Nodefleet, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyInfura, proxyUrl(stageName, Infura, chainName, networkName), false),
		*params.NewProxyProvider(chainID, ProxyGrove, proxyUrl(stageName, Grove, chainName, networkName), true),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://base-sepolia.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://base-testnet.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Base Sepolia",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia.basescan.org/",
		IconURL:                "network/Network=Base-test",
		ChainColor:             "#0052FF",
		ShortName:              "base",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         common.BaseChainID,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func statusNetworkSepolia(proxyHost string) params.Network {
	const chainID = common.StatusNetworkSepoliaChainID
	const chainName = "status"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectStatus, security.NewSensitiveString("https://public.sepolia.rpc.status.network"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Status Network Sepolia",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepoliascan.status.network/",
		IconURL:                "network/Network=Status-test",
		ChainColor:             "#7140FD",
		ShortName:              "status",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                true,
		IsActive:               true,
		IsDeactivatable:        true,
		// TODO: Update related chain ID
		// RelatedChainID:  1,
	}
}

func bnbSmartChain(proxyHost string) params.Network {
	const chainID = common.BNBSmartChainID
	const chainName = "bsc"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://bsc-mainnet.infura.io/v3/"), true),
		*params.NewDirectProvider(chainID, DirectGrove, security.NewSensitiveString("https://bsc.rpc.grove.city/v1/"), false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Binance",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://bscscan.com/",
		IconURL:                "network/Network=bsc",
		ChainColor:             "#f7bb0f",
		ShortName:              "bsc",
		NativeCurrencyName:     "BNB",
		NativeCurrencySymbol:   "BNB",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  1,
		Enabled:                true,
		RelatedChainID:         common.BNBSmartChainTestnetChainID,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func bnbSmartChainTestnet(proxyHost string) params.Network {
	const chainID = common.BNBSmartChainTestnetChainID
	const chainName = "bsc"
	const networkName = "testnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewEthRpcProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, security.NewSensitiveString("https://bsc-testnet.infura.io/v3/"), true),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Binance Testnet",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://testnet.bscscan.com/",
		IconURL:                "network/Network=bsc-test",
		ChainColor:             "#f7bb0f",
		ShortName:              "bsc",
		NativeCurrencyName:     "BNB",
		NativeCurrencySymbol:   "BNB",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  1,
		Enabled:                true,
		RelatedChainID:         common.BNBSmartChainID,
		IsActive:               false,
		IsDeactivatable:        true,
	}
}

func defaultNetworks(proxyHost, stageName string) []params.Network {
	return []params.Network{
		mainnet(proxyHost, stageName),
		sepolia(proxyHost, stageName),
		optimism(proxyHost, stageName),
		optimismSepolia(proxyHost, stageName),
		arbitrum(proxyHost, stageName),
		arbitrumSepolia(proxyHost, stageName),
		base(proxyHost, stageName),
		baseSepolia(proxyHost, stageName),
		statusNetworkSepolia(proxyHost),
		bnbSmartChain(proxyHost),
		bnbSmartChainTestnet(proxyHost),
	}
}

func setRPCs(networks []params.Network, walletConfig *requests.WalletSecretsConfig) []params.Network {
	authTokens := map[string]security.SensitiveString{
		"infura.io":  walletConfig.InfuraToken,
		"grove.city": walletConfig.PoktToken,
	}
	networks = networkhelper.OverrideDirectProvidersAuth(networks, authTokens)

	// Apply auth for new smart proxy
	hasSmartProxyCredentials := !walletConfig.EthRpcProxyUser.Empty() && !walletConfig.EthRpcProxyPassword.Empty()
	networks = networkhelper.OverrideBasicAuth(
		networks,
		params.EmbeddedEthRpcProxyProviderType,
		hasSmartProxyCredentials,
		walletConfig.EthRpcProxyUser,
		walletConfig.EthRpcProxyPassword)

	// Apply auth for old proxy
	hasOldProxyCredentials := !walletConfig.StatusProxyBlockchainUser.Empty() && !walletConfig.StatusProxyBlockchainPassword.Empty()
	networks = networkhelper.OverrideBasicAuth(
		networks,
		params.EmbeddedProxyProviderType,
		hasOldProxyCredentials,
		walletConfig.StatusProxyBlockchainUser,
		walletConfig.StatusProxyBlockchainPassword)

	return networks
}

func BuildDefaultNetworks(walletSecretsConfig *requests.WalletSecretsConfig) []params.Network {
	proxyHost := getProxyHost(walletSecretsConfig.EthRpcProxyUrl.Reveal(), walletSecretsConfig.StatusProxyStageName)
	return setRPCs(defaultNetworks(proxyHost, walletSecretsConfig.StatusProxyStageName), walletSecretsConfig)
}
