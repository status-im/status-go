package paraswap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

const transactionsURL = "https://api.paraswap.io/transactions/%d"

type Transaction struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	GasPrice string `json:"gasPrice"`
	Gas      string `json:"gas"`
	ChainID  uint64 `json:"chainId"`
	Error    string `json:"error"`
}

func (c *ClientV5) BuildTransaction(ctx context.Context, srcTokenAddress common.Address, srcTokenDecimals uint, srcAmountWei *big.Int,
	destTokenAddress common.Address, destTokenDecimals uint, destAmountWei *big.Int, slippageBasisPoints uint,
	addressFrom common.Address, addressTo common.Address, priceRoute json.RawMessage, side SwapSide) (Transaction, error) {

	params := map[string]interface{}{}
	params["srcToken"] = srcTokenAddress.Hex()
	params["srcDecimals"] = srcTokenDecimals
	params["destToken"] = destTokenAddress.Hex()
	params["destDecimals"] = destTokenDecimals
	params["userAddress"] = addressFrom.Hex()
	// params["receiver"] = addressTo.Hex() // at this point paraswap doesn't allow swap and transfer transaction
	params["priceRoute"] = priceRoute

	if slippageBasisPoints > 0 {
		params["slippage"] = slippageBasisPoints
		if side == SellSide {
			params["srcAmount"] = srcAmountWei.String()
		} else {
			params["destAmount"] = destAmountWei.String()
		}
	} else {
		params["srcAmount"] = srcAmountWei.String()
		params["destAmount"] = destAmountWei.String()
	}
	params["partner"] = c.partnerID
	if c.partnerAddress != walletCommon.ZeroAddress() && c.partnerFeePcnt > 0 {
		params["partnerAddress"] = c.partnerAddress.Hex()
		params["partnerFeeBps"] = uint(c.partnerFeePcnt * 100)
	}

	url := fmt.Sprintf(transactionsURL, c.chainID)
	response, err := c.httpClient.DoPostRequest(ctx, url, params, nil)
	if err != nil {
		return Transaction{}, err
	}

	tx, err := handleBuildTransactionResponse(response)
	if err != nil {
		return Transaction{}, err
	}

	return tx, nil
}

func handleBuildTransactionResponse(response []byte) (Transaction, error) {
	var transactionResponse Transaction
	err := json.Unmarshal(response, &transactionResponse)
	if err != nil {
		return Transaction{}, err
	}
	if transactionResponse.Error != "" {
		return Transaction{}, errors.New(transactionResponse.Error)
	}
	return transactionResponse, nil
}

// BuildTransactionWithRetry attempts to build a transaction with retry logic and eventually refresh the price route
func (c *ClientV5) BuildTransactionWithRetry(ctx context.Context, srcTokenAddress common.Address, srcTokenDecimals uint, srcAmountWei *big.Int,
	destTokenAddress common.Address, destTokenDecimals uint, destAmountWei *big.Int, slippageBasisPoints uint,
	addressFrom common.Address, addressTo common.Address, priceRoute json.RawMessage, side SwapSide) (Transaction, *Route, error) {

	const maxRetries = 3
	baseDelay := time.Second

	for i := 0; i < maxRetries; i++ {
		tx, err := c.BuildTransaction(ctx, srcTokenAddress, srcTokenDecimals, srcAmountWei,
			destTokenAddress, destTokenDecimals, destAmountWei, slippageBasisPoints,
			addressFrom, addressTo, priceRoute, side)
		if err == nil {
			return tx, nil, nil
		}

		if i == maxRetries-1 {
			time.Sleep(1 * time.Second)

			newRoute, err := c.FetchPriceRoute(ctx, srcTokenAddress, srcTokenDecimals,
				destTokenAddress, destTokenDecimals, srcAmountWei, addressFrom,
				addressTo, side)
			if err != nil {
				return Transaction{}, nil, fmt.Errorf("failed to fetch new price route: %v", err)
			}

			time.Sleep(1 * time.Second)

			tx, err = c.BuildTransaction(ctx, srcTokenAddress, srcTokenDecimals, srcAmountWei,
				destTokenAddress, destTokenDecimals, destAmountWei, slippageBasisPoints,
				addressFrom, addressTo, newRoute.RawPriceRoute, side)
			if err == nil {
				return tx, &newRoute, nil
			}
		}

		delay := baseDelay * time.Duration(math.Pow(2, float64(i)))
		select {
		case <-ctx.Done():
			return Transaction{}, nil, ctx.Err()
		case <-time.After(delay):
			continue
		}
	}

	return Transaction{}, nil, fmt.Errorf("failed to build transaction after %d retries", maxRetries)
}
