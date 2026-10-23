package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"

	gethcommon "github.com/ethereum/go-ethereum/common"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
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
	tokens, _, _, err := c.fetchTokensWithEtag(ctx, "")
	return tokens, err
}

// fetchTokensWithEtag sends If-None-Match when etag is set and reports a 304 as notModified.
func (c *Client) fetchTokensWithEtag(ctx context.Context, etag string) (tokens []GeckoToken, newEtag string, notModified bool, err error) {
	params := netUrl.Values{}
	params.Add("include_platform", "true")
	url := fmt.Sprintf(coinsListURL, c.baseURL)
	response, newEtag, err := c.httpClient.DoGetRequestWithEtag(ctx, url, c.mergeCoingeckoQueryKey(params), etag,
		thirdparty.WithCredentials(c.creds))
	if err != nil {
		return nil, "", false, err
	}
	if etag != "" && len(response) == 0 {
		return nil, newEtag, true, nil
	}

	err = json.Unmarshal(response, &tokens)
	if err != nil {
		return nil, "", false, err
	}

	return tokens, newEtag, false, nil
}
