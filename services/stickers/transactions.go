package stickers

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/stickers"
)

func (api *API) StickerMarketAddress(ctx context.Context, chainID uint64) (common.Address, error) {
	return stickers.StickerMarketContractAddress(chainID)
}
