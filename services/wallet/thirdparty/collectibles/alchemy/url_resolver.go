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

func (r *proxyURLResolver) GetNFTBaseURL(chainID walletCommon.ChainID) (string, error) {
	return GetNftProxyBaseURL(r.customURL, r.stageName, chainID)
}

func (r *proxyURLResolver) IsChainSupported(chainID walletCommon.ChainID) bool {
	_, err := GetNftProxyBaseURL(r.customURL, r.stageName, chainID)
	return err == nil
}

func getBaseURL(chainID walletCommon.ChainID) (string, error) {
	switch uint64(chainID) {
	case walletCommon.EthereumMainnet:
		return "https://eth-mainnet.g.alchemy.com", nil
	case walletCommon.EthereumSepolia:
		return "https://eth-sepolia.g.alchemy.com", nil
	case walletCommon.OptimismMainnet:
		return "https://opt-mainnet.g.alchemy.com", nil
	case walletCommon.OptimismSepolia:
		return "https://opt-sepolia.g.alchemy.com", nil
	case walletCommon.ArbitrumMainnet:
		return "https://arb-mainnet.g.alchemy.com", nil
	case walletCommon.ArbitrumSepolia:
		return "https://arb-sepolia.g.alchemy.com", nil
	case walletCommon.BaseMainnet:
		return "https://base-mainnet.g.alchemy.com", nil
	case walletCommon.BaseSepolia:
		return "https://base-sepolia.g.alchemy.com", nil
	case walletCommon.LineaMainnet:
		return "https://linea-mainnet.g.alchemy.com", nil
	case walletCommon.LineaSepolia:
		return "https://linea-sepolia.g.alchemy.com", nil
	case walletCommon.UnichainMainnet:
		return "https://unichain-mainnet.g.alchemy.com", nil
	case walletCommon.UnichainSepolia:
		return "https://unichain-sepolia.g.alchemy.com", nil
	case walletCommon.InkMainnet:
		return "https://ink-mainnet.g.alchemy.com", nil
	case walletCommon.InkSepolia:
		return "https://ink-sepolia.g.alchemy.com", nil
	case walletCommon.AbstractMainnet:
		return "https://abstract-mainnet.g.alchemy.com", nil
	case walletCommon.AbstractTestnet:
		return "https://abstract-sepolia.g.alchemy.com", nil
	case walletCommon.ZkSyncMainnet:
		return "https://zksync-mainnet.g.alchemy.com", nil
	case walletCommon.ZkSyncSepolia:
		return "https://zksync-sepolia.g.alchemy.com", nil
	case walletCommon.SoneiumMainnet:
		return "https://soneium-mainnet.g.alchemy.com", nil
	case walletCommon.SoneiumMinato:
		return "https://soneium-minato.g.alchemy.com", nil
	case walletCommon.ScrollMainnet:
		return "https://scroll-mainnet.g.alchemy.com", nil
	case walletCommon.ScrollSepolia:
		return "https://scroll-sepolia.g.alchemy.com", nil
	case walletCommon.BlastMainnet:
		return "https://blast-mainnet.g.alchemy.com", nil
	case walletCommon.BlastSepolia:
		return "https://blast-sepolia.g.alchemy.com", nil
	}

	return "", thirdparty.ErrChainIDNotSupported
}

func getAPIKeySubpath(apiKey security.SensitiveString) security.SensitiveString {
	if apiKey.Empty() {
		return security.NewSensitiveString("demo")
	}
	return apiKey
}
