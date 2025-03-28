package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"
	"strings"

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

const coinsListURL = "%s/coins/list"

type GeckoToken struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	Platforms Platforms `json:"platforms"`
}

type Platforms struct { // When we add a new chain we should update it here
	Ethereum string `json:"ethereum"`
	Optimism string `json:"optimistic-ethereum"`
	Arbitrum string `json:"arbitrum-one"`
	Base     string `json:"base"`
}

func (gt GeckoToken) getKeyForChain(chainID uint64) (string, error) {
	var tokenAddress string
	switch chainID {
	case common.EthereumMainnet,
		common.EthereumSepolia,
		common.AnvilMainnet:
		tokenAddress = gt.Platforms.Ethereum
	case common.OptimismMainnet,
		common.OptimismSepolia:
		tokenAddress = gt.Platforms.Optimism
	case common.ArbitrumMainnet,
		common.ArbitrumSepolia:
		tokenAddress = gt.Platforms.Arbitrum
	case common.BaseMainnet,
		common.BaseSepolia:
		tokenAddress = gt.Platforms.Base
	default:
		return "", fmt.Errorf("chain %d is not supported", chainID)
	}

	if tokenAddress == "" {
		return "", fmt.Errorf("token address for chain %d is not set", chainID)
	}

	return utils.MakeTokenKey(chainID, strings.ToUpper(gt.Platforms.Ethereum)), nil
}

func (c *Client) FetchTokens(ctx context.Context) ([]GeckoToken, error) {

	params := netUrl.Values{}
	params.Add("include_platform", "true")
	url := fmt.Sprintf(coinsListURL, c.baseURL)
	response, err := c.httpClient.DoGetRequest(ctx, url, params)
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
