package gas

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/ethclient"
)

// getLineaGasPriceForAccount calculates gas price suggestions for Linea-based chains
// Results might change depending on the account placing the transaction (i.e. gasless chains like Status Network),
// so the address always needs to be provided.
func suggestLineaGasPriceForAccount(ctx context.Context, ethClient EthClient, address gethcommon.Address) (*GasPrice, error) {
	// Dummy transaction
	toAddress := gethcommon.Address{}
	callMsg := &ethereum.CallMsg{
		From:  address,
		To:    &toAddress,
		Value: big.NewInt(0),
	}

	var estimateGasResult *ethclient.LineaEstimateGasResult
	estimateGasResult, err := estimateLineaTxGas(ctx, ethClient, callMsg)
	if err != nil {
		return nil, err
	}

	return suggestLineaGasPrice(estimateGasResult)
}

func suggestLineaGasPrice(estimateGasResult *ethclient.LineaEstimateGasResult) (*GasPrice, error) {
	// Base fee stabilizes to a fixed value, use it as is
	// Add 15% buffer to the priority fee
	priorityFeePerGas := new(big.Int).Mul(estimateGasResult.PriorityFeePerGas, big.NewInt(115))
	priorityFeePerGas.Div(priorityFeePerGas, big.NewInt(100))
	ret := &GasPrice{
		BaseFeePerGas:           estimateGasResult.BaseFeePerGas,
		LowPriorityFeePerGas:    priorityFeePerGas,
		MediumPriorityFeePerGas: priorityFeePerGas,
		HighPriorityFeePerGas:   priorityFeePerGas,
	}

	return ret, nil
}
