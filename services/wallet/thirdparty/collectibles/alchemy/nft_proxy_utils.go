package alchemy

import (
	"fmt"
	"strings"

	"github.com/status-im/status-go/params/networkdefaults"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
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
	host := GetNftProxyHost(customURL, stageName)

	chainName, networkName := networkdefaults.GetProxyChainAndNetworkName(uint64(chainID))
	if networkName == "" || chainName == "" {
		return "", thirdparty.ErrChainIDNotSupported
	}

	return fmt.Sprintf("%s/%s/%s/nft/v3", host, chainName, networkName), nil
}
