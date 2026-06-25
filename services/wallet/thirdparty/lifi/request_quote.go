package lifi

import (
	"context"
	"encoding/json"
	netUrl "net/url"
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type TransactionRequest struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	GasPrice string `json:"gasPrice"`
	GasLimit string `json:"gasLimit"`
	ChainID  uint64 `json:"chainId"`
}

type Estimate struct {
	FromAmount      *bigint.BigInt `json:"fromAmount"`
	ToAmount        *bigint.BigInt `json:"toAmount"`
	ToAmountMin     *bigint.BigInt `json:"toAmountMin"`
	ApprovalAddress common.Address `json:"approvalAddress"`
}

type Quote struct {
	ID                 string             `json:"id"`
	Type               string             `json:"type"`
	Tool               string             `json:"tool"`
	Estimate           Estimate           `json:"estimate"`
	TransactionRequest TransactionRequest `json:"transactionRequest"`
}

func (c *Client) FetchQuote(ctx context.Context, p QuoteParams) (Quote, error) {
	params := netUrl.Values{}
	params.Add("fromChain", strconv.FormatUint(p.ChainID, 10))
	params.Add("toChain", strconv.FormatUint(p.ChainID, 10))
	params.Add("fromToken", p.FromToken.Hex())
	params.Add("toToken", p.ToToken.Hex())
	params.Add("fromAddress", p.FromAddress.Hex())
	params.Add("toAddress", p.ToAddress.Hex())
	params.Add("fromAmount", p.AmountIn.String())
	params.Add("slippage", strconv.FormatFloat(float64(p.SlippagePercentage)/100.0, 'f', -1, 64))
	params.Add("integrator", c.integrator)

	options := []thirdparty.RequestOption{}
	if c.apiKey != "" {
		options = append(options, thirdparty.WithHeader("x-lifi-api-key", c.apiKey))
	}

	response, err := c.httpClient.DoGetRequest(ctx, baseURL+"/quote", params, options...)
	if err != nil {
		return Quote{}, err
	}

	var quote Quote
	if err := json.Unmarshal(response, &quote); err != nil {
		return Quote{}, err
	}
	return quote, nil
}
