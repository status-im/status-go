package gas

import (
	"context"

	"github.com/ethereum/go-ethereum"

	"github.com/status-im/go-wallet-sdk/pkg/ethclient"
)

func estimateLineaTxGas(ctx context.Context, ethClient EthClient, callMsg *ethereum.CallMsg) (*ethclient.LineaEstimateGasResult, error) {
	return ethClient.LineaEstimateGas(ctx, *callMsg)
}
