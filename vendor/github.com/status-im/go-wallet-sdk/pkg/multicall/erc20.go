package multicall

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc20"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
)

// Call for ERC20 function "balanceOf(owner)"
func BuildERC20BalanceCall(accountAddress common.Address, tokenAddress common.Address) multicall3.IMulticall3Call {
	abi, err := erc20.Erc20MetaData.GetAbi()
	if err != nil {
		panic(err)
	}

	callData, err := abi.Pack("balanceOf", accountAddress)
	if err != nil {
		panic(err)
	}

	call := multicall3.IMulticall3Call{
		Target:   tokenAddress,
		CallData: callData,
	}

	return call
}

func ProcessERC20BalanceResult(result multicall3.IMulticall3Result) (*big.Int, error) {
	if result.Success {
		return new(big.Int).SetBytes(result.ReturnData), nil
	}
	return nil, errors.New(string(result.ReturnData))
}
