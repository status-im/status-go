package relay

//go:generate go tool mockgen -package=mock_relay -source=types.go -destination=mock/types.go

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// AppFeeRecipient collects the app fee added to every quote (claimable in USDC on Relay's side).
var AppFeeRecipient = common.HexToAddress("0xa99B907F267C956F0e92359b145e3ba87CE107A4")

// Step kinds and ids returned in a quote's `steps` array.
const (
	StepKindTransaction = "transaction"
	StepKindSignature   = "signature"

	StepIDApprove = "approve"
)

type QuoteParams struct {
	FromChainID        uint64
	ToChainID          uint64
	FromToken          common.Address
	ToToken            common.Address
	FromAddress        common.Address
	ToAddress          common.Address
	AmountIn           *big.Int
	SlippagePercentage float32
}

// ClientInterface is the subset of the Relay API used by the router.
type ClientInterface interface {
	SetChainID(chainID uint64)
	FetchQuote(ctx context.Context, params QuoteParams) (Quote, error)
	FetchChains(ctx context.Context) ([]Chain, error)
}
