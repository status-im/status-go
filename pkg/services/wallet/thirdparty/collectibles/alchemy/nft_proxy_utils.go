package alchemy

import (
	"fmt"
	"strings"

	"github.com/status-im/status-go/pkg/services/networks"
	walletcollectibles "github.com/status-im/status-go/pkg/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

const (
	NftProxyHostSuffix = "nft.status.im"
)

// GetNftProxyHost creates NFT proxy base URL based on stage name
func GetNftProxyHost(customURL, stageName string) string {
	if customURL != "" {
		return strings.TrimRight(customURL, "/")
	}
	if stageName == "" {
		stageName = "test"
	}
	return fmt.Sprintf("https://%s.%s", stageName, NftProxyHostSuffix)
}

// GetNftProxyBaseURL creates NFT proxy URL for a specific chain
func GetNftProxyBaseURL(customURL, stageName string, chainID walletCommon.ChainID) (string, error) {
	if walletcollectibles.IsUnsupportedCollectibleChain(uint64(chainID)) {
		return "", thirdparty.ErrChainIDNotSupported
	}

	chainName, networkName := networks.GetProxyChainAndNetworkName(uint64(chainID))
	if networkName == "" || chainName == "" {
		return "", thirdparty.ErrChainIDNotSupported
	}

	host := GetNftProxyHost(customURL, stageName)
	return fmt.Sprintf("%s/%s/%s/nft/v3", host, chainName, networkName), nil
}
