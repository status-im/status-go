package gas

//go:generate mockgen -destination=mock/ethclient.go . EthClient

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"

	"github.com/status-im/go-wallet-sdk/pkg/ethclient"
)

type EthClient interface {
	FeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	LineaEstimateGas(ctx context.Context, msg ethereum.CallMsg) (*ethclient.LineaEstimateGasResult, error)
}
