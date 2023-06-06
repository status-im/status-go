package api

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"
)

const (
	mainnetChainID        uint64 = 1
	goerliChainID         uint64 = 5
	optimismChainID       uint64 = 10
	optimismGoerliChainID uint64 = 420
	arbitrumChainID       uint64 = 42161
	arbitrumGoerliChainID uint64 = 421613
	sntSymbol                    = "SNT"
	sttSymbol                    = "STT"
)

var ganacheTokenAddress = common.HexToAddress("0x8571Ddc46b10d31EF963aF49b6C7799Ea7eff818")

var mainnet = params.Network{
	ChainID:                mainnetChainID,
	ChainName:              "Ethereum Mainnet",
	RPCURL:                 "https://eth-archival.gateway.pokt.network/v1/lb/",
	FallbackURL:            "https://mainnet.infura.io/v3/",
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
}

var goerli = params.Network{
	ChainID:                goerliChainID,
	ChainName:              "Ethereum Mainnet",
	RPCURL:                 "https://goerli-archival.gateway.pokt.network/v1/lb/",
	FallbackURL:            "https://goerli.infura.io/v3/",
	BlockExplorerURL:       "https://goerli.etherscan.io/",
	IconURL:                "network/Network=Testnet",
	ChainColor:             "#939BA1",
	ShortName:              "goEth",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 true,
	Layer:                  1,
	Enabled:                true,
}

var optimism = params.Network{
	ChainID:                optimismChainID,
	ChainName:              "Optimism",
	RPCURL:                 "https://optimism-mainnet.gateway.pokt.network/v1/lb/",
	FallbackURL:            "https://optimism-mainnet.infura.io/v3/",
	BlockExplorerURL:       "https://optimistic.etherscan.io",
	IconURL:                "network/Network=Optimism",
	ChainColor:             "#E90101",
	ShortName:              "opt",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 false,
	Layer:                  2,
	Enabled:                true,
}

var optimismGoerli = params.Network{
	ChainID:                optimismGoerliChainID,
	ChainName:              "Optimism Goerli Testnet",
	RPCURL:                 "https://optimism-goerli.infura.io/v3/",
	FallbackURL:            "",
	BlockExplorerURL:       "https://goerli-optimism.etherscan.io/",
	IconURL:                "network/Network=Testnet",
	ChainColor:             "#939BA1",
	ShortName:              "goOpt",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 true,
	Layer:                  2,
	Enabled:                false,
}
var arbitrum = params.Network{
	ChainID:                arbitrumChainID,
	ChainName:              "Arbitrum",
	RPCURL:                 "https://arbitrum-one.gateway.pokt.network/v1/lb/",
	FallbackURL:            "https://arbitrum-mainnet.infura.io/v3/",
	BlockExplorerURL:       "https://arbiscan.io/",
	IconURL:                "network/Network=Arbitrum",
	ChainColor:             "#51D0F0",
	ShortName:              "arb",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 false,
	Layer:                  2,
	Enabled:                true,
}

var arbitrumGoerli = params.Network{
	ChainID:                arbitrumGoerliChainID,
	ChainName:              "Arbitrum Goerli",
	RPCURL:                 "https://arbitrum-goerli.infura.io/v3/",
	FallbackURL:            "",
	BlockExplorerURL:       "https://goerli.arbiscan.io/",
	IconURL:                "network/Network=Testnet",
	ChainColor:             "#939BA1",
	ShortName:              "goArb",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 true,
	Layer:                  2,
	Enabled:                false,
}

var defaultNetworks = []params.Network{
	mainnet,
	goerli,
	optimism,
	optimismGoerli,
	arbitrum,
	arbitrumGoerli,
}

var mainnetGanacheTokenOverrides = params.TokenOverride{
	Symbol:  sntSymbol,
	Address: ganacheTokenAddress,
}

var goerliGanacheTokenOverrides = params.TokenOverride{
	Symbol:  sttSymbol,
	Address: ganacheTokenAddress,
}

func setRPCs(networks []params.Network, request *requests.WalletSecretsConfig) []params.Network {

	var networksWithRPC []params.Network

	for _, n := range networks {

		if request.InfuraToken != "" {
			if strings.Contains(n.RPCURL, "infura") {
				n.RPCURL += request.InfuraToken
			}
			if strings.Contains(n.FallbackURL, "infura") {
				n.FallbackURL += request.InfuraToken
			}
		}

		if request.PoktToken != "" {
			if strings.Contains(n.RPCURL, "pokt") {
				n.RPCURL += request.PoktToken
			}
			if strings.Contains(n.FallbackURL, "pokt") {
				n.FallbackURL += request.PoktToken
			}

		}

		if request.GanacheURL != "" {
			n.RPCURL = request.GanacheURL
			n.FallbackURL = request.GanacheURL
			if n.ChainID == mainnetChainID {
				n.TokenOverrides = []params.TokenOverride{
					mainnetGanacheTokenOverrides,
				}
			} else if n.ChainID == goerliChainID {
				n.TokenOverrides = []params.TokenOverride{
					goerliGanacheTokenOverrides,
				}
			}
		}

		networksWithRPC = append(networksWithRPC, n)
	}

	return networksWithRPC
}

func buildDefaultNetworks(request *requests.CreateAccount) []params.Network {
	return setRPCs(defaultNetworks, &request.WalletSecretsConfig)
}
