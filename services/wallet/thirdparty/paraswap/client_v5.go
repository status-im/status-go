package paraswap

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type SwapSide string

const (
	SellSide = SwapSide("SELL")
	BuySide  = SwapSide("BUY")
)

type ClientV5 struct {
	httpClient     *thirdparty.HTTPClient
	chainID        uint64
	partnerID      string
	partnerAddress common.Address
	partnerFeePcnt float64
}

func NewClientV5(
	chainID uint64,
	partnerID string,
	partnerAddress common.Address,
	partnerFeePcnt float64) *ClientV5 {
	return &ClientV5{
		httpClient:     thirdparty.NewHTTPClient(),
		chainID:        chainID,
		partnerID:      partnerID,
		partnerAddress: partnerAddress,
		partnerFeePcnt: partnerFeePcnt,
	}
}

func (c *ClientV5) SetChainID(chainID uint64) {
	c.chainID = chainID
}

func (c *ClientV5) SetPartnerAddress(partnerAddress common.Address) {
	c.partnerAddress = partnerAddress
}

func (c *ClientV5) SetPartnerFeePcnt(partnerFeePcnt float64) {
	c.partnerFeePcnt = partnerFeePcnt
}
