package wallet

import (
	"context"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/rpc"
)

type SuggestedFees struct {
	GasPrice             *big.Float `json:"gasPrice"`
	BaseFee              *big.Float `json:"baseFee"`
	MaxPriorityFeePerGas *big.Float `json:"maxPriorityFeePerGas"`
	MaxFeePerGasLow      *big.Float `json:"maxFeePerGasLow"`
	MaxFeePerGasMedium   *big.Float `json:"maxFeePerGasMedium"`
	MaxFeePerGasHigh     *big.Float `json:"maxFeePerGasHigh"`
	EIP1559Enabled       bool       `json:"eip1559Enabled"`
}

type FeeHistory struct {
	BaseFeePerGas []string `json:"baseFeePerGas"`
}

type FeeManager struct {
	RPCClient *rpc.Client
}

func weiToGwei(val *big.Int) *big.Float {
	result := new(big.Float)
	result.SetInt(val)

	unit := new(big.Int)
	unit.SetInt64(params.GWei)

	return result.Quo(result, new(big.Float).SetInt(unit))
}

func (f *FeeManager) suggestedFees(ctx context.Context, chainID uint64) (*SuggestedFees, error) {
	backend, err := f.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	gasPrice, err := backend.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	block, err := backend.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	maxPriorityFeePerGas, err := backend.SuggestGasTipCap(ctx)
	if err != nil {
		return &SuggestedFees{
			GasPrice:             weiToGwei(gasPrice),
			BaseFee:              big.NewFloat(0),
			MaxPriorityFeePerGas: big.NewFloat(0),
			MaxFeePerGasLow:      big.NewFloat(0),
			MaxFeePerGasMedium:   big.NewFloat(0),
			MaxFeePerGasHigh:     big.NewFloat(0),
			EIP1559Enabled:       false,
		}, nil
	}

	config := params.MainnetChainConfig
	baseFee := misc.CalcBaseFee(config, block.Header())

	var feeHistory FeeHistory

	err = f.RPCClient.Call(&feeHistory, chainID, "eth_feeHistory", 101, "latest", nil)
	if err != nil {
		return nil, err
	}

	fees := []*big.Int{}
	for _, fee := range feeHistory.BaseFeePerGas {
		i := new(big.Int)
		i.SetString(strings.Replace(fee, "0x", "", 1), 16)
		fees = append(fees, i)
	}
	sort.Slice(fees, func(i, j int) bool { return fees[i].Cmp(fees[j]) < 0 })

	perc10 := fees[int64(0.1*float64(len(fees)))-1]
	perc20 := fees[int64(0.2*float64(len(fees)))-1]

	var maxFeePerGasMedium *big.Int
	if baseFee.Cmp(perc20) >= 0 {
		maxFeePerGasMedium = baseFee
	} else {
		maxFeePerGasMedium = perc20
	}

	if maxPriorityFeePerGas.Cmp(maxFeePerGasMedium) > 0 {
		maxFeePerGasMedium = maxPriorityFeePerGas
	}

	maxFeePerGasHigh := new(big.Int).Mul(maxPriorityFeePerGas, big.NewInt(2))
	twoTimesBaseFee := new(big.Int).Mul(baseFee, big.NewInt(2))
	if twoTimesBaseFee.Cmp(maxFeePerGasHigh) > 0 {
		maxFeePerGasHigh = twoTimesBaseFee
	}

	return &SuggestedFees{
		GasPrice:             weiToGwei(gasPrice),
		BaseFee:              weiToGwei(baseFee),
		MaxPriorityFeePerGas: weiToGwei(maxPriorityFeePerGas),
		MaxFeePerGasLow:      weiToGwei(perc10),
		MaxFeePerGasMedium:   weiToGwei(maxFeePerGasMedium),
		MaxFeePerGasHigh:     weiToGwei(maxFeePerGasHigh),
		EIP1559Enabled:       true,
	}, nil
}
