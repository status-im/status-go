package api

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"
)

const (
	MainnetChainID         uint64 = 1
	SepoliaChainID         uint64 = 11155111
	OptimismChainID        uint64 = 10
	OptimismSepoliaChainID uint64 = 11155420
	ArbitrumChainID        uint64 = 42161
	ArbitrumSepoliaChainID uint64 = 421614
	sntSymbol                     = "SNT"
	sttSymbol                     = "STT"
)

var ganacheTokenAddress = common.HexToAddress("0x8571Ddc46b10d31EF963aF49b6C7799Ea7eff818")

func mainnet(stageName string) params.Network {
	return params.Network{
		ChainID:                MainnetChainID,
		ChainName:              "Mainnet",
		DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/ethereum/mainnet/", stageName),
		DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/ethereum/mainnet/", stageName),
		DefaultFallbackURL2:    fmt.Sprintf("https://%s.api.status.im/grove/ethereum/mainnet/", stageName),
		RPCURL:                 "https://mainnet.infura.io/v3/",
		FallbackURL:            "https://eth-archival.rpc.grove.city/v1/",
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
	return params.Network{
		ChainID:                SepoliaChainID,
		ChainName:              "Mainnet",
		DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/ethereum/sepolia/", stageName),
		DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/ethereum/sepolia/", stageName),
		DefaultFallbackURL2:    fmt.Sprintf("https://%s.api.status.im/grove/ethereum/sepolia/", stageName),
		RPCURL:                 "https://sepolia.infura.io/v3/",
		FallbackURL:            "https://sepolia-archival.rpc.grove.city/v1/",
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
	return params.Network{
		ChainID:                OptimismChainID,
		ChainName:              "Optimism",
		DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/optimism/mainnet/", stageName),
		DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/optimism/mainnet/", stageName),
		DefaultFallbackURL2:    fmt.Sprintf("https://%s.api.status.im/grove/optimism/mainnet/", stageName),
		RPCURL:                 "https://optimism-mainnet.infura.io/v3/",
		FallbackURL:            "https://optimism-archival.rpc.grove.city/v1/",
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
	return params.Network{
		ChainID:                OptimismSepoliaChainID,
		ChainName:              "Optimism",
		DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/optimism/sepolia/", stageName),
		DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/optimism/sepolia/", stageName),
		DefaultFallbackURL2:    fmt.Sprintf("https://%s.api.status.im/grove/optimism/sepolia/", stageName),
		RPCURL:                 "https://optimism-sepolia.infura.io/v3/",
		FallbackURL:            "https://optimism-sepolia-archival.rpc.grove.city/v1/",
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
	return params.Network{
		ChainID:                ArbitrumChainID,
		ChainName:              "Arbitrum",
		DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/arbitrum/mainnet/", stageName),
		DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/arbitrum/mainnet/", stageName),
		DefaultFallbackURL2:    fmt.Sprintf("https://%s.api.status.im/grove/arbitrum/mainnet/", stageName),
		RPCURL:                 "https://arbitrum-mainnet.infura.io/v3/",
		FallbackURL:            "https://arbitrum-one.rpc.grove.city/v1/",
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
	return params.Network{
		ChainID:                ArbitrumSepoliaChainID,
		ChainName:              "Arbitrum",
		DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/arbitrum/sepolia/", stageName),
		DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/arbitrum/sepolia/", stageName),
		DefaultFallbackURL2:    fmt.Sprintf("https://%s.api.status.im/grove/arbitrum/sepolia/", stageName),
		RPCURL:                 "https://arbitrum-sepolia.infura.io/v3/",
		FallbackURL:            "https://arbitrum-sepolia-archival.rpc.grove.city/v1/",
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

func defaultNetworks(stageName string) []params.Network {
	return []params.Network{
		mainnet(stageName),
		sepolia(stageName),
		optimism(stageName),
		optimismSepolia(stageName),
		arbitrum(stageName),
		arbitrumSepolia(stageName),
	}
}

var mainnetGanacheTokenOverrides = params.TokenOverride{
	Symbol:  sntSymbol,
	Address: ganacheTokenAddress,
}

func setRPCs(networks []params.Network, request *requests.WalletSecretsConfig) []params.Network {

	var networksWithRPC []params.Network

	const (
		infura = "infura.io/"
		grove  = "grove.city/"
	)

	appendToken := func(url string) string {
		if strings.Contains(url, infura) && request.InfuraToken != "" {
			return url + request.InfuraToken
		} else if strings.Contains(url, grove) && request.PoktToken != "" {
			return url + request.PoktToken
		}
		return url
	}

	for _, n := range networks {
		n.DefaultRPCURL = appendToken(n.DefaultRPCURL)
		n.DefaultFallbackURL = appendToken(n.DefaultFallbackURL)
		n.DefaultFallbackURL2 = appendToken(n.DefaultFallbackURL2)
		n.RPCURL = appendToken(n.RPCURL)
		n.FallbackURL = appendToken(n.FallbackURL)

		if request.GanacheURL != "" {
			n.RPCURL = request.GanacheURL
			n.FallbackURL = request.GanacheURL
			if n.ChainID == MainnetChainID {
				n.TokenOverrides = []params.TokenOverride{
					mainnetGanacheTokenOverrides,
				}
			}
		}

		networksWithRPC = append(networksWithRPC, n)
	}

	return networksWithRPC
}

func BuildDefaultNetworks(walletSecretsConfig *requests.WalletSecretsConfig) []params.Network {
	return setRPCs(defaultNetworks(walletSecretsConfig.StatusProxyStageName), walletSecretsConfig)
}
