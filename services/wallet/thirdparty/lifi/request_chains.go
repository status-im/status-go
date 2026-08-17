package lifi

import (
	"context"
	"encoding/json"
	netUrl "net/url"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

// ChainInfo carries the per-chain contract addresses LI.FI publishes on /chains. They are
// routing data, not a source of truth: callers check them against the deployments pinned
// in the permit2 package before letting a swap involve them.
type ChainInfo struct {
	ID           uint64
	Permit2      common.Address
	Permit2Proxy common.Address
}

type chainsResponse struct {
	Chains []struct {
		ID           uint64 `json:"id"`
		Permit2      string `json:"permit2"`
		Permit2Proxy string `json:"permit2Proxy"`
	} `json:"chains"`
}

// FetchChains reads the full EVM chain list from LI.FI.
func (c *Client) FetchChains(ctx context.Context) ([]ChainInfo, error) {
	params := netUrl.Values{}
	params.Add("chainTypes", "EVM")

	options := []thirdparty.RequestOption{}
	if c.apiKey != "" {
		options = append(options, thirdparty.WithHeader("x-lifi-api-key", c.apiKey))
	}

	response, err := c.httpClient.DoGetRequest(ctx, baseURL+"/chains", params, options...)
	if err != nil {
		return nil, err
	}

	var parsed chainsResponse
	if err := json.Unmarshal(response, &parsed); err != nil {
		return nil, err
	}

	chains := make([]ChainInfo, 0, len(parsed.Chains))
	for _, ch := range parsed.Chains {
		chains = append(chains, ChainInfo{
			ID:           ch.ID,
			Permit2:      parseAddress(ch.Permit2),
			Permit2Proxy: parseAddress(ch.Permit2Proxy),
		})
	}
	return chains, nil
}

// parseAddress turns an optional address field into a common.Address, yielding the zero
// address for the empty/absent case.
func parseAddress(value string) common.Address {
	if !common.IsHexAddress(value) {
		return common.Address{}
	}
	return common.HexToAddress(value)
}

// GetChainInfo returns the cached metadata for a single chain, fetching the chain list on
// first use. A failed fetch isn't cached, so a transient API outage degrades to the regular
// approve-then-swap flow rather than disabling permits for the whole session.
func (c *Client) GetChainInfo(ctx context.Context, chainID uint64) (*ChainInfo, error) {
	c.chainInfoMu.RLock()
	cached, ok := c.chainInfo[chainID]
	loaded := c.chainInfoLoaded
	c.chainInfoMu.RUnlock()

	if ok {
		return &cached, nil
	}
	if loaded {
		// The list was fetched and this chain simply is not on it.
		return nil, ErrChainNotSupported
	}

	chains, err := c.FetchChains(ctx)
	if err != nil {
		return nil, err
	}

	c.chainInfoMu.Lock()
	c.chainInfo = make(map[uint64]ChainInfo, len(chains))
	for _, ch := range chains {
		c.chainInfo[ch.ID] = ch
	}
	c.chainInfoLoaded = true
	cached, ok = c.chainInfo[chainID]
	c.chainInfoMu.Unlock()

	if !ok {
		return nil, ErrChainNotSupported
	}
	return &cached, nil
}
