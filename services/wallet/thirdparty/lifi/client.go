package lifi

import (
	"errors"
	"sync"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const (
	baseURL = "https://li.quest/v1"

	Integrator = "status.app"
)

// ErrChainNotSupported is returned when LI.FI does not list the requested chain.
var ErrChainNotSupported = errors.New("chain not supported by LI.FI")

type Client struct {
	httpClient *thirdparty.HTTPClient
	chainID    uint64
	integrator string
	apiKey     string

	chainInfoMu     sync.RWMutex
	chainInfo       map[uint64]ChainInfo
	chainInfoLoaded bool
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
