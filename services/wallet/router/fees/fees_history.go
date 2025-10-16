package fees

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"

	gaspriceproxy "github.com/status-im/status-go/contracts/gas-price-proxy"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

type FeeHistory struct {
	BaseFeePerGas []string   `json:"baseFeePerGas"`
	GasUsedRatio  []float64  `json:"gasUsedRatio"`
	OldestBlock   string     `json:"oldestBlock"`
	Reward        [][]string `json:"reward,omitempty"`
}

func getFeeHistoryBlockCount(chainID uint64) uint64 {
	if chainID == walletCommon.EthereumMainnet || chainID == walletCommon.EthereumSepolia || chainID == walletCommon.AnvilMainnet {
		return 10 // use the last 10 blocks for L1 chains
	}
	return 50 // use the last 50 blocks for L2 chains
}

func (f *FeeManager) getFeeHistory(ctx context.Context, chainID uint64, blockCount uint64, newestBlock string, rewardPercentiles []int) (*FeeHistory, error) {
	feeHistory := &FeeHistory{}
	err := f.RPCClient.Call(feeHistory, chainID, "eth_feeHistory", blockCount, newestBlock, rewardPercentiles)
	return feeHistory, err
}

func (f *FeeManager) getGaslessParamsForAccount(ctx context.Context, chainID uint64, address ethCommon.Address) (baseFee *big.Int, priorityFee *big.Int, err error) {
	if !walletCommon.IsPartiallyOrFullyGaslessChain(chainID) {
		return nil, nil, nil
	}

	toAddress := walletCommon.ZeroAddress()
	msg := ethereum.CallMsg{
		From:  address,
		To:    &toAddress,
		Value: walletCommon.ZeroBigIntValue(),
	}

	var result struct {
		BaseFeePerGas     string `json:"baseFeePerGas"`
		PriorityFeePerGas string `json:"priorityFeePerGas"`
	}

	err = f.RPCClient.CallContext(ctx, &result, chainID, "linea_estimateGas", walletCommon.ToCallArg(msg))
	if err != nil {
		return
	}

	baseFee, err = hexStringToBigInt(result.BaseFeePerGas)
	if err != nil {
		return nil, nil, err
	}

	priorityFee, err = hexStringToBigInt(result.PriorityFeePerGas)
	if err != nil {
		return nil, nil, err
	}

	return
}

// GetL1Fee returns L1 fee for placing a transaction to L1 chain, appicable only for txs made from L2.
func (f *FeeManager) GetL1Fee(ctx context.Context, chainID uint64, input []byte) (uint64, error) {
	if chainID == walletCommon.EthereumMainnet || chainID == walletCommon.EthereumSepolia {
		return 0, nil
	}

	ethClient, err := f.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	contractAddress, err := gaspriceproxy.ContractAddress(chainID)
	if err != nil {
		return 0, err
	}

	contract, err := gaspriceproxy.NewGaspriceproxy(contractAddress, ethClient)
	if err != nil {
		return 0, err
	}

	callOpt := &bind.CallOpts{}

	result, err := contract.GetL1Fee(callOpt, input)
	if err != nil {
		return 0, err
	}

	return result.Uint64(), nil
}
