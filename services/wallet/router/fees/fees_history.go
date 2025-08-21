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

func (fh *FeeHistory) isEIP1559Compatible(chainID uint64) bool {
	// Since the Status Network is gasless chain, but EIP-1559 compatible, we should not rely on checking the BaseFeePerGas, that's why we have this special case.
	eip1559Enabled, err := walletCommon.IsPartiallyOrFullyGaslessChainEIP1559Compatible(chainID)
	if err == nil {
		return eip1559Enabled
	}

	if len(fh.BaseFeePerGas) == 0 {
		return false
	}

	for _, fee := range fh.BaseFeePerGas {
		if fee != "0x0" {
			return true
		}
	}

	return false
}

func (f *FeeManager) getFeeHistory(ctx context.Context, chainID uint64, newestBlock string, rewardPercentiles []int) (*FeeHistory, error) {
	blockCount := uint64(10) // use the last 10 blocks for L1 chains
	if chainID != walletCommon.EthereumMainnet &&
		chainID != walletCommon.EthereumSepolia &&
		chainID != walletCommon.AnvilMainnet {
		blockCount = 50 // use the last 50 blocks for L2 chains
	}

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
