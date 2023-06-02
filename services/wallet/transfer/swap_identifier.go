package transfer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	uniswapv2 "github.com/status-im/status-go/contracts/uniswapV2"
	uniswapv3 "github.com/status-im/status-go/contracts/uniswapV3"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/token"
)

func fetchUniswapV2PairInfo(ctx context.Context, client *chain.ClientWithFallback, pairAddress common.Address) (*common.Address, *common.Address, error) {
	caller, err := uniswapv2.NewUniswapv2Caller(pairAddress, client)
	if err != nil {
		return nil, nil, err
	}

	token0Address, err := caller.Token0(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}

	token1Address, err := caller.Token1(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}

	return &token0Address, &token1Address, nil
}

func fetchUniswapV3PoolInfo(ctx context.Context, client *chain.ClientWithFallback, poolAddress common.Address) (*common.Address, *common.Address, error) {
	caller, err := uniswapv3.NewUniswapv3Caller(poolAddress, client)
	if err != nil {
		return nil, nil, err
	}

	token0Address, err := caller.Token0(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}

	token1Address, err := caller.Token1(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}

	return &token0Address, &token1Address, nil
}

func identifyUniswapV2Asset(tokenManager *token.Manager, chainID uint64, amount0 *big.Int, contractAddress0 common.Address, amount1 *big.Int, contractAddress1 common.Address) (token *token.Token, amount *big.Int, err error) {
	// Either amount0 or amount1 should be 0
	if amount1.Sign() == 0 && amount0.Sign() != 0 {
		token = tokenManager.FindTokenByAddress(chainID, contractAddress0)
		if token == nil {
			err = fmt.Errorf("couldn't find symbol for token0 %v", contractAddress0)
			return
		}
		amount = amount0
	} else if amount0.Sign() == 0 && amount1.Sign() != 0 {
		token = tokenManager.FindTokenByAddress(chainID, contractAddress1)
		if token == nil {
			err = fmt.Errorf("couldn't find symbol for token1 %v", contractAddress1)
			return
		}
		amount = amount1
	} else {
		err = fmt.Errorf("couldn't identify token %v %v %v %v", contractAddress0, amount0, contractAddress1, amount1)
		return
	}

	return
}

func fetchUniswapV2Info(ctx context.Context, client *chain.ClientWithFallback, tokenManager *token.Manager, log *types.Log) (fromAsset string, fromAmount *hexutil.Big, toAsset string, toAmount *hexutil.Big, err error) {
	pairAddress, _, _, amount0In, amount1In, amount0Out, amount1Out, err := parseUniswapV2Log(log)
	if err != nil {
		return
	}

	token0ContractAddress, token1ContractAddress, err := fetchUniswapV2PairInfo(ctx, client, pairAddress)
	if err != nil {
		return
	}

	fromToken, fromAmountInt, err := identifyUniswapV2Asset(tokenManager, client.ChainID, amount0In, *token0ContractAddress, amount1In, *token1ContractAddress)
	if err != nil {
		// "Soft" error, allow to continue with unknown asset
		fromAsset = ""
		fromAmount = (*hexutil.Big)(big.NewInt(0))
	} else {
		fromAsset = fromToken.Symbol
		fromAmount = (*hexutil.Big)(fromAmountInt)
	}

	toToken, toAmountInt, err := identifyUniswapV2Asset(tokenManager, client.ChainID, amount0Out, *token0ContractAddress, amount1Out, *token1ContractAddress)
	if err != nil {
		// "Soft" error, allow to continue with unknown asset
		toAsset = ""
		toAmount = (*hexutil.Big)(big.NewInt(0))
	} else {
		toAsset = toToken.Symbol
		toAmount = (*hexutil.Big)(toAmountInt)
	}

	err = nil
	return
}

func identifyUniswapV3Assets(tokenManager *token.Manager, chainID uint64, amount0 *big.Int, contractAddress0 common.Address, amount1 *big.Int, contractAddress1 common.Address) (fromToken *token.Token, fromAmount *big.Int, toToken *token.Token, toAmount *big.Int, err error) {
	token0 := tokenManager.FindTokenByAddress(chainID, contractAddress0)
	if token0 == nil {
		err = fmt.Errorf("couldn't find symbol for token0 %v", contractAddress0)
		return
	}

	token1 := tokenManager.FindTokenByAddress(chainID, contractAddress1)
	if token1 == nil {
		err = fmt.Errorf("couldn't find symbol for token1 %v", contractAddress1)
		return
	}

	// amount0 and amount1 are the balance deltas of the pool
	// The positive amount is how much the sender spent
	// The negative amount is how much the recipent got
	if amount0.Sign() > 0 && amount1.Sign() < 0 {
		fromToken = token0
		fromAmount = amount0
		toToken = token1
		toAmount = new(big.Int).Neg(amount1)
	} else if amount0.Sign() < 0 && amount1.Sign() > 0 {
		fromToken = token1
		fromAmount = amount1
		toToken = token0
		toAmount = new(big.Int).Neg(amount0)
	} else {
		err = fmt.Errorf("couldn't identify tokens %v %v %v %v", contractAddress0, amount0, contractAddress1, amount1)
		return
	}

	return
}

func fetchUniswapV3Info(ctx context.Context, client *chain.ClientWithFallback, tokenManager *token.Manager, log *types.Log) (fromAsset string, fromAmount *hexutil.Big, toAsset string, toAmount *hexutil.Big, err error) {
	poolAddress, _, _, amount0, amount1, err := parseUniswapV3Log(log)
	if err != nil {
		return
	}

	token0ContractAddress, token1ContractAddress, err := fetchUniswapV3PoolInfo(ctx, client, poolAddress)
	if err != nil {
		return
	}

	fromToken, fromAmountInt, toToken, toAmountInt, err := identifyUniswapV3Assets(tokenManager, client.ChainID, amount0, *token0ContractAddress, amount1, *token1ContractAddress)
	if err != nil {
		// "Soft" error, allow to continue with unknown asset
		err = nil
		fromAsset = ""
		fromAmount = (*hexutil.Big)(big.NewInt(0))
		toAsset = ""
		toAmount = (*hexutil.Big)(big.NewInt(0))
	} else {
		fromAsset = fromToken.Symbol
		fromAmount = (*hexutil.Big)(fromAmountInt)
		toAsset = toToken.Symbol
		toAmount = (*hexutil.Big)(toAmountInt)
	}

	return
}

func fetchUniswapInfo(ctx context.Context, client *chain.ClientWithFallback, tokenManager *token.Manager, log *types.Log, logType EventType) (fromAsset string, fromAmount *hexutil.Big, toAsset string, toAmount *hexutil.Big, err error) {
	switch logType {
	case uniswapV2SwapEventType:
		return fetchUniswapV2Info(ctx, client, tokenManager, log)
	case uniswapV3SwapEventType:
		return fetchUniswapV3Info(ctx, client, tokenManager, log)
	}
	err = fmt.Errorf("wrong log type %s", logType)
	return
}

// Build a Swap multitransaction from a list containing one or several uniswapV2/uniswapV3 subTxs
// We only care about the first and last swap to identify the input/output token and amounts
func buildUniswapSwapMultitransaction(ctx context.Context, client *chain.ClientWithFallback, tokenManager *token.Manager, transfer *Transfer) (*MultiTransaction, error) {
	multiTransaction := MultiTransaction{
		Type:        MultiTransactionSwap,
		FromAddress: transfer.Address,
		ToAddress:   transfer.Address,
	}

	var firstSwapLog, lastSwapLog *types.Log
	var firstSwapLogType, lastSwapLogType EventType

	for _, ethlog := range transfer.Receipt.Logs {
		logType := GetEventType(ethlog)
		switch logType {
		case uniswapV2SwapEventType, uniswapV3SwapEventType:
			if firstSwapLog == nil {
				firstSwapLog = ethlog
				firstSwapLogType = logType
			}
			lastSwapLog = ethlog
			lastSwapLogType = logType
		}
	}

	var err error

	multiTransaction.FromAsset, multiTransaction.FromAmount, multiTransaction.ToAsset, multiTransaction.ToAmount, err = fetchUniswapInfo(ctx, client, tokenManager, firstSwapLog, firstSwapLogType)
	if err != nil {
		return nil, err
	}

	if firstSwapLog != lastSwapLog {
		_, _, multiTransaction.ToAsset, multiTransaction.ToAmount, err = fetchUniswapInfo(ctx, client, tokenManager, lastSwapLog, lastSwapLogType)
		if err != nil {
			return nil, err
		}
	}

	return &multiTransaction, nil
}
