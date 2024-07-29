package paraswap

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type ClientInterface interface {
	SetChainID(chainID uint64)
	SetPartnerAddress(partnerAddress common.Address)
	SetPartnerFeePcnt(partnerFeePcnt float64)
	BuildTransaction(ctx context.Context, srcTokenAddress common.Address, srcTokenDecimals uint, srcAmountWei *big.Int,
		destTokenAddress common.Address, destTokenDecimals uint, destAmountWei *big.Int, slippageBasisPoints uint,
		addressFrom common.Address, addressTo common.Address, priceRoute json.RawMessage, side SwapSide) (Transaction, error)
	FetchPriceRoute(ctx context.Context, srcTokenAddress common.Address, srcTokenDecimals uint,
		destTokenAddress common.Address, destTokenDecimals uint, amountWei *big.Int, addressFrom common.Address,
		addressTo common.Address, side SwapSide) (Route, error)
	FetchTokensList(ctx context.Context) ([]Token, error)
}
