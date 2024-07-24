package paraswap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

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
	if c.partnerAddress != walletCommon.ZeroAddress && c.partnerFeePcnt > 0 {
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
