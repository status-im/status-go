package api

import (
	"fmt"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/protocol/requests"
)

const (
	MainnetChainID         uint64 = 1
	SepoliaChainID         uint64 = 11155111
	OptimismChainID        uint64 = 10
	OptimismSepoliaChainID uint64 = 11155420
	ArbitrumChainID        uint64 = 42161
	ArbitrumSepoliaChainID uint64 = 421614
	BaseChainID            uint64 = 8453
	BaseSepoliaChainID     uint64 = 84532
	sntSymbol                     = "SNT"
	sttSymbol                     = "STT"
)

// ProviderID represents the internal ID of a blockchain provider
type ProviderID = string

// Provider IDs
const (
	StatusSmartProxy = "status-smart-proxy"
	DirectInfura     = "direct-infura"
	DirectGrove      = "direct-grove"
)

func smartProxyUrl(chainName, networkName string) string {
	return fmt.Sprintf("https://test.eth-rpc.status.im/%s/%s/", chainName, networkName)
}

func mainnet(stageName string) params.Network {
	const chainID = MainnetChainID
	const chainName = "ethereum"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, "https://mainnet.infura.io/v3/", true),
		*params.NewDirectProvider(chainID, DirectGrove, "https://eth.rpc.grove.city/v1/", false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Mainnet",
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
		RelatedChainID:         SepoliaChainID,
	}
}

func sepolia(stageName string) params.Network {
	const chainID = SepoliaChainID
	const chainName = "ethereum"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, "https://sepolia.infura.io/v3/", true),
		*params.NewDirectProvider(chainID, DirectGrove, "https://eth-sepolia-testnet.rpc.grove.city/v1/", false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Mainnet",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia.etherscan.io/",
		IconURL:                "network/Network=Ethereum",
		ChainColor:             "#627EEA",
		ShortName:              "eth",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  1,
		Enabled:                true,
		RelatedChainID:         MainnetChainID,
	}
}

func optimism(stageName string) params.Network {
	const chainID = OptimismChainID
	const chainName = "optimism"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, "https://optimism-mainnet.infura.io/v3/", true),
		*params.NewDirectProvider(chainID, DirectGrove, "https://optimism.rpc.grove.city/v1/", false),
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
		RelatedChainID:         OptimismSepoliaChainID,
	}
}

func optimismSepolia(stageName string) params.Network {
	const chainID = OptimismSepoliaChainID
	const chainName = "optimism"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, "https://optimism-sepolia.infura.io/v3/", true),
		*params.NewDirectProvider(chainID, DirectGrove, "https://optimism-sepolia-testnet.rpc.grove.city/v1/", false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Optimism",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia-optimism.etherscan.io/",
		IconURL:                "network/Network=Optimism",
		ChainColor:             "#E90101",
		ShortName:              "oeth",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         OptimismChainID,
	}
}

func arbitrum(stageName string) params.Network {
	const chainID = ArbitrumChainID
	const chainName = "arbitrum"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, "https://arbitrum-mainnet.infura.io/v3/", true),
		*params.NewDirectProvider(chainID, DirectGrove, "https://arbitrum-one.rpc.grove.city/v1/", false),
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
		RelatedChainID:         ArbitrumSepoliaChainID,
	}
}

func arbitrumSepolia(stageName string) params.Network {
	const chainID = ArbitrumSepoliaChainID
	const chainName = "arbitrum"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, "https://arbitrum-sepolia.infura.io/v3/", true),
		*params.NewDirectProvider(chainID, DirectGrove, "https://arbitrum-sepolia-testnet.rpc.grove.city/v1/", false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Arbitrum",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia-explorer.arbitrum.io/",
		IconURL:                "network/Network=Arbitrum",
		ChainColor:             "#51D0F0",
		ShortName:              "arb1",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         ArbitrumChainID,
	}
}

func base(stageName string) params.Network {
	const chainID = BaseChainID
	const chainName = "base"
	const networkName = "mainnet"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, "https://base-mainnet.infura.io/v3/", true),
		*params.NewDirectProvider(chainID, DirectGrove, "https://base.rpc.grove.city/v1/", false),
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
		RelatedChainID:         BaseSepoliaChainID,
	}
}

func baseSepolia(stageName string) params.Network {
	const chainID = BaseSepoliaChainID
	const chainName = "base"
	const networkName = "sepolia"

	rpcProviders := []params.RpcProvider{
		// Smart proxy provider
		*params.NewProxyProvider(chainID, StatusSmartProxy, smartProxyUrl(chainName, networkName), false),
		// Direct providers
		*params.NewDirectProvider(chainID, DirectInfura, "https://base-sepolia.infura.io/v3/", true),
		*params.NewDirectProvider(chainID, DirectGrove, "https://base-testnet.rpc.grove.city/v1/", false),
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Base",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia.basescan.org/",
		IconURL:                "network/Network=Base",
		ChainColor:             "#0052FF",
		ShortName:              "base",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         BaseChainID,
	}
}

func defaultNetworks(stageName string) []params.Network {
	return []params.Network{
		mainnet(stageName),
		sepolia(stageName),
		optimism(stageName),
		optimismSepolia(stageName),
		arbitrum(stageName),
		arbitrumSepolia(stageName),
		base(stageName),
		baseSepolia(stageName),
	}
}

func setRPCs(networks []params.Network, request *requests.WalletSecretsConfig) []params.Network {
	authTokens := map[string]string{
		"infura.io":  request.InfuraToken,
		"grove.city": request.PoktToken,
	}
	return networkhelper.OverrideDirectProvidersAuth(networks, authTokens)
}

func BuildDefaultNetworks(walletSecretsConfig *requests.WalletSecretsConfig) []params.Network {
	return setRPCs(defaultNetworks(walletSecretsConfig.StatusProxyStageName), walletSecretsConfig)
}
