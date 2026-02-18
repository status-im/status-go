package backend

import (
	"fmt"
	"strings"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/wallet/common"
)

const (
	// Host suffixes for providers
	SmartProxyHostSuffix = "eth-rpc.status.im"
	ProxyHostSuffix      = "api.status.im"
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

// New eth-rpc-proxy endpoint with smart proxy URL tragetting a specific provider
func smartProxyUrlWithProvider(proxyHost, chainName, networkName, provider string) security.SensitiveString {
	return security.NewSensitiveStringPrintf("%s/%s/%s/%s", proxyHost, chainName, networkName, provider)
}

func mainnet(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.EthereumMainnet
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://mainnet.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://eth.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.EthereumSepolia,
		IsActive:               true,
		IsDeactivatable:        false,
	}
}

func sepolia(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.EthereumSepolia
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://sepolia.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://eth-sepolia-testnet.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.EthereumMainnet,
		IsActive:               true,
		IsDeactivatable:        false,
	}
}

func optimism(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.OptimismMainnet
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://optimism-mainnet.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://optimism.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.OptimismSepolia,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func optimismSepolia(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.OptimismSepolia
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://optimism-sepolia.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://optimism-sepolia-testnet.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.OptimismMainnet,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func arbitrum(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.ArbitrumMainnet
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://arbitrum-mainnet.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://arbitrum-one.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.ArbitrumSepolia,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func arbitrumSepolia(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.ArbitrumSepolia
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://arbitrum-sepolia.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://arbitrum-sepolia-testnet.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Arbitrum Sepolia",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia.arbiscan.io/",
		IconURL:                "network/Network=Arbitrum-test",
		ChainColor:             "#51D0F0",
		ShortName:              "arb1",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         common.ArbitrumMainnet,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func base(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.BaseMainnet
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://base-mainnet.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://base.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.BaseSepolia,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func baseSepolia(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.BaseSepolia
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://base-sepolia.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://base-testnet.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.BaseMainnet,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func statusNetworkSepolia(proxyHost string, enableRpcProviders bool) params.Network {
	const chainID = common.StatusNetworkSepolia
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectStatus, security.NewSensitiveString("https://public.sepolia.rpc.status.network"), false),
		}
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

func bnbSmartChain(proxyHost string, enableRpcProviders bool) params.Network {
	const chainID = common.BSCMainnet
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://bsc-mainnet.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://bsc.rpc.grove.city/v1/"), false),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.BSCTestnet,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

func linea(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.LineaMainnet
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://linea-mainnet.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://linea.rpc.grove.city/v1/"), false), // rpc.grove.city is not working at all, maybe we should remove it as an option from all networks
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Linea",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://lineascan.build/",
		IconURL:                "network/Network=Linea",
		ChainColor:             "#121212",
		ShortName:              "linea",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         common.LineaSepolia,
		IsActive:               false,
		IsDeactivatable:        true,
	}
}

func lineaSepolia(proxyHost, stageName string, enableRpcProviders bool) params.Network {
	const chainID = common.LineaSepolia
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Proxy providers
			*params.NewProxyProvider(chainID, common.ProxyInfura, proxyUrl(stageName, common.Infura, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://linea-sepolia.infura.io/v3/"), true),
			*params.NewDirectProvider(chainID, common.DirectGrove, security.NewSensitiveString("https://linea-sepolia-testnet.rpc.grove.city/v1/"), false), // rpc.grove.city is not working at all, maybe we should remove it as an option from all networks
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
	}

	return params.Network{
		ChainID:                chainID,
		ChainName:              "Linea Sepolia",
		RpcProviders:           rpcProviders,
		BlockExplorerURL:       "https://sepolia.lineascan.build/",
		IconURL:                "network/Network=Linea-test",
		ChainColor:             "#121212",
		ShortName:              "linea",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  2,
		Enabled:                false,
		RelatedChainID:         common.LineaMainnet,
		IsActive:               false,
		IsDeactivatable:        true,
	}
}

func bnbSmartChainTestnet(proxyHost string, enableRpcProviders bool) params.Network {
	const chainID = common.BSCTestnet
	chainName, networkName, _ := networkhelper.ChainIDToChainAndNetwork(chainID)

	var rpcProviders []params.RpcProvider = []params.RpcProvider{}
	if enableRpcProviders {
		rpcProviders = []params.RpcProvider{
			// Smart proxy provider
			*params.NewEthRpcProxyProvider(chainID, common.StatusSmartProxy, smartProxyUrl(proxyHost, chainName, networkName), false),
			// Direct providers
			*params.NewDirectProvider(chainID, common.DirectInfura, security.NewSensitiveString("https://bsc-testnet.infura.io/v3/"), true),
			// Smart Proxy specific providers
			*params.NewEthRpcProxyProvider(chainID, common.SmartProxyAlchemy, smartProxyUrlWithProvider(proxyHost, chainName, networkName, common.Alchemy), false),
		}
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
		RelatedChainID:         common.BSCMainnet,
		IsActive:               false,
		IsDeactivatable:        true,
	}
}

func defaultNetworks(proxyHost, stageName string, thirdpartyServicesEnabled bool) []params.Network {
	return []params.Network{
		mainnet(proxyHost, stageName, thirdpartyServicesEnabled),
		sepolia(proxyHost, stageName, thirdpartyServicesEnabled),
		optimism(proxyHost, stageName, thirdpartyServicesEnabled),
		optimismSepolia(proxyHost, stageName, thirdpartyServicesEnabled),
		arbitrum(proxyHost, stageName, thirdpartyServicesEnabled),
		arbitrumSepolia(proxyHost, stageName, thirdpartyServicesEnabled),
		base(proxyHost, stageName, thirdpartyServicesEnabled),
		baseSepolia(proxyHost, stageName, thirdpartyServicesEnabled),
		linea(proxyHost, stageName, thirdpartyServicesEnabled),
		lineaSepolia(proxyHost, stageName, thirdpartyServicesEnabled),
		statusNetworkSepolia(proxyHost, thirdpartyServicesEnabled),
		bnbSmartChain(proxyHost, thirdpartyServicesEnabled),
		bnbSmartChainTestnet(proxyHost, thirdpartyServicesEnabled),
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
	hasOldProxyCredentials := !walletConfig.StatusProxyUser.Empty() && !walletConfig.StatusProxyPassword.Empty()
	networks = networkhelper.OverrideBasicAuth(
		networks,
		params.EmbeddedProxyProviderType,
		hasOldProxyCredentials,
		walletConfig.StatusProxyUser,
		walletConfig.StatusProxyPassword)

	return networks
}

func BuildDefaultNetworks(walletSecretsConfig *requests.WalletSecretsConfig, thirdpartyServicesEnabled bool) []params.Network {
	proxyHost := getProxyHost(walletSecretsConfig.EthRpcProxyUrl.Reveal(), walletSecretsConfig.StatusProxyStageName)
	networks := setRPCs(defaultNetworks(proxyHost, walletSecretsConfig.StatusProxyStageName, thirdpartyServicesEnabled), walletSecretsConfig)
	return networkhelper.WithCommunitiesSupported(networks)
}
