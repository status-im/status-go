package alchemy

import (
	"fmt"

	"github.com/status-im/status-go/pkg/security"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

// URLResolver constructs base URLs for Alchemy Client
type URLResolver interface {
	// For direct Alchemy: "https://eth-mainnet.g.alchemy.com/nft/v3/{apiKey}"
	// For proxy: "https://test.nft.status.im/ethereum/mainnet/nft/v3"
	GetNFTBaseURL(chainID walletCommon.ChainID) (string, error)

	// IsChainSupported returns true if the resolver can construct a URL for this chain.
	IsChainSupported(chainID walletCommon.ChainID) bool
}

type directURLResolver struct {
	apiKey security.SensitiveString
}

func (r *directURLResolver) GetNFTBaseURL(chainID walletCommon.ChainID) (string, error) {
	baseURL, err := getBaseURL(chainID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/nft/v3/%s", baseURL, getAPIKeySubpath(r.apiKey).Reveal()), nil
}

func (r *directURLResolver) IsChainSupported(chainID walletCommon.ChainID) bool {
	_, err := getBaseURL(chainID)
	return err == nil
}

type proxyURLResolver struct {
	customURL string
	stageName string
}

// alchemyNFTDirectHosts is the Alchemy NFT API base URL per chain for direct HTTP calls.
var alchemyNFTDirectHosts = map[uint64]string{
	walletCommon.EthereumMainnet:  "https://eth-mainnet.g.alchemy.com",
	walletCommon.EthereumSepolia:  "https://eth-sepolia.g.alchemy.com",
	walletCommon.OptimismMainnet:  "https://opt-mainnet.g.alchemy.com",
	walletCommon.OptimismSepolia:  "https://opt-sepolia.g.alchemy.com",
	walletCommon.ArbitrumMainnet:  "https://arb-mainnet.g.alchemy.com",
	walletCommon.ArbitrumSepolia:  "https://arb-sepolia.g.alchemy.com",
	walletCommon.BaseMainnet:      "https://base-mainnet.g.alchemy.com",
	walletCommon.BaseSepolia:      "https://base-sepolia.g.alchemy.com",
	walletCommon.LineaMainnet:     "https://linea-mainnet.g.alchemy.com",
	walletCommon.LineaSepolia:     "https://linea-sepolia.g.alchemy.com",
	walletCommon.UnichainMainnet:  "https://unichain-mainnet.g.alchemy.com",
	walletCommon.UnichainSepolia:  "https://unichain-sepolia.g.alchemy.com",
	walletCommon.AbstractMainnet:  "https://abstract-mainnet.g.alchemy.com",
	walletCommon.AbstractTestnet:  "https://abstract-sepolia.g.alchemy.com",
	walletCommon.ZkSyncMainnet:    "https://zksync-mainnet.g.alchemy.com",
	walletCommon.ZkSyncSepolia:    "https://zksync-sepolia.g.alchemy.com",
	walletCommon.SoneiumMainnet:   "https://soneium-mainnet.g.alchemy.com",
	walletCommon.SoneiumMinato:    "https://soneium-minato.g.alchemy.com",
	walletCommon.ScrollMainnet:    "https://scroll-mainnet.g.alchemy.com",
	walletCommon.ScrollSepolia:    "https://scroll-sepolia.g.alchemy.com",
	walletCommon.BlastMainnet:     "https://blast-mainnet.g.alchemy.com",
	walletCommon.BlastSepolia:     "https://blast-sepolia.g.alchemy.com",
	walletCommon.RobinhoodMainnet: "https://robinhood-mainnet.g.alchemy.com",
	walletCommon.RobinhoodTestnet: "https://robinhood-testnet.g.alchemy.com",
}

func (r *proxyURLResolver) GetNFTBaseURL(chainID walletCommon.ChainID) (string, error) {
	return GetNftProxyBaseURL(r.customURL, r.stageName, chainID)
}

func (r *proxyURLResolver) IsChainSupported(chainID walletCommon.ChainID) bool {
	_, err := GetNftProxyBaseURL(r.customURL, r.stageName, chainID)
	return err == nil
}

func getBaseURL(chainID walletCommon.ChainID) (string, error) {
	id := uint64(chainID)
	host, ok := alchemyNFTDirectHosts[id]
	if !ok {
		return "", thirdparty.ErrChainIDNotSupported
	}
	return host, nil
}

func getAPIKeySubpath(apiKey security.SensitiveString) security.SensitiveString {
	if apiKey.Empty() {
		return security.NewSensitiveString("demo")
	}
	return apiKey
}
