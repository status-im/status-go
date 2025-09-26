package multicall

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
)

// Call for Multicall3 function "getEthBalance(address)"
func BuildNativeBalanceCall(accountAddress common.Address, multicall3Address common.Address) multicall3.IMulticall3Call {
	abi, err := multicall3.Multicall3MetaData.GetAbi()
	if err != nil {
		panic(err)
	}

	callData, err := abi.Pack("getEthBalance", accountAddress)
	if err != nil {
		panic(err)
	}

	call := multicall3.IMulticall3Call{
		Target:   multicall3Address,
		CallData: callData,
	}

	return call
}

func ProcessNativeBalanceResult(result multicall3.IMulticall3Result) (*big.Int, error) {
	if result.Success {
		return new(big.Int).SetBytes(result.ReturnData), nil
	}
	return nil, errors.New(string(result.ReturnData))
}
