package paraswap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const transactionsURL = "https://apiv5.paraswap.io/transactions/%d"

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
	destTokenAddress common.Address, destTokenDecimals uint, destAmountWei *big.Int,
	addressFrom common.Address, addressTo common.Address, priceRoute json.RawMessage) (Transaction, error) {

	params := map[string]interface{}{}
	params["srcToken"] = srcTokenAddress.Hex()
	params["srcDecimals"] = srcTokenDecimals
	params["srcAmount"] = srcAmountWei.String()
	params["destToken"] = destTokenAddress.Hex()
	params["destDecimals"] = destTokenDecimals
	// params["destAmount"] = destAmountWei.String()
	params["userAddress"] = addressFrom.Hex()
	// params["receiver"] = addressTo.Hex() // at this point paraswap doesn't allow swap and transfer transaction
	params["slippage"] = "500"
	params["priceRoute"] = priceRoute

	url := fmt.Sprintf(transactionsURL, c.chainID)
	response, err := c.httpClient.doPostRequest(ctx, url, params)
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
