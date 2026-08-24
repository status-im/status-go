package lifi

import (
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

const (
	baseURL = "https://li.quest/v1"

	Integrator = "status-app"

	feeFraction = "0.0062" // 0.62%
)

type Client struct {
	httpClient *thirdparty.HTTPClient
	chainID    uint64
	integrator string
	apiKey     string
}

func NewClient(chainID uint64, integrator string, apiKey string) *Client {
	return &Client{
		httpClient: thirdparty.NewHTTPClient(),
		chainID:    chainID,
		integrator: integrator,
		apiKey:     apiKey,
	}
}

func (c *Client) SetChainID(chainID uint64) {
	c.chainID = chainID
}
