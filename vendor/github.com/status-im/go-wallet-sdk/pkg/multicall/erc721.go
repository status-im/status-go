package multicall

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc721"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
)

// Call for ERC721 function "balanceOf(owner)"
func BuildERC721BalanceCall(accountAddress common.Address, tokenAddress common.Address) multicall3.IMulticall3Call {
	abi, err := erc721.Erc721MetaData.GetAbi()
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

func ProcessERC721BalanceResult(result multicall3.IMulticall3Result) (*big.Int, error) {
	if result.Success {
		return new(big.Int).SetBytes(result.ReturnData), nil
	}
	return nil, errors.New(string(result.ReturnData))
}
