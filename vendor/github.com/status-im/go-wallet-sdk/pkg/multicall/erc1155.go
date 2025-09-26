package multicall

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc1155"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
)

// Call for ERC1155 function "balanceOf(account, id)"
func BuildERC1155BalanceCall(accountAddress common.Address, tokenAddress common.Address, tokenID *big.Int) multicall3.IMulticall3Call {
	abi, err := erc1155.Erc1155MetaData.GetAbi()
	if err != nil {
		panic(err)
	}

	callData, err := abi.Pack("balanceOf", accountAddress, tokenID)
	if err != nil {
		panic(err)
	}

	call := multicall3.IMulticall3Call{
		Target:   tokenAddress,
		CallData: callData,
	}

	return call
}

func ProcessERC1155BalanceResult(result multicall3.IMulticall3Result) (*big.Int, error) {
	if result.Success {
		return new(big.Int).SetBytes(result.ReturnData), nil
	}
	return nil, errors.New(string(result.ReturnData))
}
