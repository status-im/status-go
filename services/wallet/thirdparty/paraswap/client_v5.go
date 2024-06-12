package paraswap

import "github.com/status-im/status-go/services/wallet/thirdparty"

type SwapSide string

const (
	SellSide = SwapSide("SELL")
	BuySide  = SwapSide("BUY")
)

type ClientV5 struct {
	httpClient *thirdparty.HTTPClient
	chainID    uint64
}

func NewClientV5(chainID uint64) *ClientV5 {
	return &ClientV5{
		httpClient: thirdparty.NewHTTPClient(),
		chainID:    chainID,
	}
}

func (c *ClientV5) SetChainID(chainID uint64) {
	c.chainID = chainID
}
