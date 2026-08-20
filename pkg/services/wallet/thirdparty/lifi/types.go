package lifi

//go:generate go tool mockgen -package=mock_lifi -source=types.go -destination=mock/types.go

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

var NativeTokenAddress = walletcommon.ZeroAddress()

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

type ClientInterface interface {
	SetChainID(chainID uint64)
	FetchQuote(ctx context.Context, params QuoteParams) (Quote, error)
	FetchTokensList(ctx context.Context) ([]Token, error)
}
