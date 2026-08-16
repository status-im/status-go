package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	walletcommon "github.com/status-im/status-go/services/wallet/common"
)

const (
	coinsListURL = "%s/coins/list"

	nativeEthTokenID = "ethereum"
	nativeBNBTokenID = "binancecoin"
)

type GeckoToken struct {
	ID        string            `json:"id"`
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
	Platforms map[string]string `json:"platforms"`
}

func (gt *GeckoToken) keys() []string {
	keys := make([]string, 0)

	for platform, contractAddress := range gt.Platforms {
		if len(contractAddress) != walletcommon.HexAddressLength {
			continue
		}
		chainID, ok := walletcommon.CoinGeckoPlatformChainMapper[platform]
		if !ok {
			continue
		}
		key := types.TokenKey(chainID, gethcommon.HexToAddress(contractAddress))
		keys = append(keys, key)
	}
	return keys
}

func (c *Client) fetchTokens(ctx context.Context) ([]GeckoToken, error) {

	params := netUrl.Values{}
	params.Add("include_platform", "true")
	url := fmt.Sprintf(coinsListURL, c.baseURL)
	response, err := c.doGetRequestWithOptionalAuth(ctx, url, params)
	if err != nil {
		return nil, err
	}

	var tokens []GeckoToken
	err = json.Unmarshal(response, &tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
