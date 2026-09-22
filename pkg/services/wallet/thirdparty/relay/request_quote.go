package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	"github.com/status-im/status-go/pkg/services/wallet/common"
)

// Amounts are decimal strings in Relay responses, hence bigint.BigInt rather than hexutil.Big.

type StepTxData struct {
	From                 string         `json:"from"`
	To                   string         `json:"to"`
	Data                 string         `json:"data"`
	Value                *bigint.BigInt `json:"value"`
	Gas                  *bigint.BigInt `json:"gas,omitempty"`
	MaxFeePerGas         *bigint.BigInt `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *bigint.BigInt `json:"maxPriorityFeePerGas"`
	ChainID              uint64         `json:"chainId"`
}

type StepCheck struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
}

type StepItem struct {
	Status string     `json:"status"`
	Data   StepTxData `json:"data"`
	Check  StepCheck  `json:"check"`
}

type Step struct {
	ID          string     `json:"id"`
	Action      string     `json:"action"`
	Description string     `json:"description"`
	Kind        string     `json:"kind"`
	RequestID   string     `json:"requestId"`
	Items       []StepItem `json:"items"`
}

type Currency struct {
	ChainID  uint64 `json:"chainId"`
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Decimals uint   `json:"decimals"`
}

type CurrencyAmount struct {
	Currency      Currency       `json:"currency"`
	Amount        *bigint.BigInt `json:"amount"`
	MinimumAmount *bigint.BigInt `json:"minimumAmount"`
	AmountUsd     string         `json:"amountUsd"`
}

type Fees struct {
	Gas            CurrencyAmount `json:"gas"`
	Relayer        CurrencyAmount `json:"relayer"`
	RelayerGas     CurrencyAmount `json:"relayerGas"`
	RelayerService CurrencyAmount `json:"relayerService"`
	App            CurrencyAmount `json:"app"`
}

// RouteLeg describes one side of the route; Router names the swap source used on that leg
// (a DEX aggregator such as "0x" or "kyberswap", or "relay" for Relay's own liquidity).
type RouteLeg struct {
	InputCurrency  CurrencyAmount `json:"inputCurrency"`
	OutputCurrency CurrencyAmount `json:"outputCurrency"`
	Router         string         `json:"router"`
}

type Route struct {
	Origin      RouteLeg `json:"origin"`
	Destination RouteLeg `json:"destination"`
}

type Details struct {
	Operation    string         `json:"operation"`
	Sender       string         `json:"sender"`
	Recipient    string         `json:"recipient"`
	TimeEstimate int            `json:"timeEstimate"` // seconds
	CurrencyIn   CurrencyAmount `json:"currencyIn"`
	CurrencyOut  CurrencyAmount `json:"currencyOut"`
	Rate         string         `json:"rate"`
	Route        Route          `json:"route"`
}

type Quote struct {
	RequestID string  `json:"requestId"`
	Steps     []Step  `json:"steps"`
	Fees      Fees    `json:"fees"`
	Details   Details `json:"details"`
}

// APIError is Relay's `{message, errorCode}` error payload.
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("relay: %s: %s", e.Code, e.Message)
}

type apiErrorResponse struct {
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

func parseAPIError(body []byte) *APIError {
	var resp apiErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil || resp.ErrorCode == "" {
		return nil
	}
	return &APIError{Code: resp.ErrorCode, Message: resp.Message}
}

func (c *Client) quoteRequestBody(p QuoteParams) map[string]interface{} {
	toChainID := p.ToChainID
	if toChainID == 0 {
		toChainID = p.FromChainID
	}

	body := map[string]interface{}{
		"user":                p.FromAddress.Hex(),
		"originChainId":       p.FromChainID,
		"destinationChainId":  toChainID,
		"originCurrency":      p.FromToken.Hex(),
		"destinationCurrency": p.ToToken.Hex(),
		"amount":              p.AmountIn.String(),
		"tradeType":           "EXACT_INPUT",
		"recipient":           p.ToAddress.Hex(),

		"useRouteRacing": true,
	}

	// percent -> basis points; omitted when unset so Relay applies its own auto slippage
	if p.SlippagePercentage > 0 {
		body["slippageTolerance"] = strconv.Itoa(int(math.Round(float64(p.SlippagePercentage) * 100)))
	}

	// Relay rejects a referrer that isn't backed by an API key.
	if c.apiKey != "" {
		body["referrer"] = c.referrer
	}

	if AppFeeRecipient != common.ZeroAddress() {
		body["appFees"] = []map[string]string{{
			"recipient": AppFeeRecipient.Hex(),
			"fee":       appFeeBps,
		}}
	}

	return body
}

func (c *Client) FetchQuote(ctx context.Context, p QuoteParams) (Quote, error) {
	response, err := c.httpClient.DoPostRequest(ctx, c.baseURL+"/quote/v2", c.quoteRequestBody(p), nil, c.requestOptions()...)
	if err != nil {
		return Quote{}, err
	}

	if apiErr := parseAPIError(response); apiErr != nil {
		return Quote{}, apiErr
	}

	var quote Quote
	if err := json.Unmarshal(response, &quote); err != nil {
		return Quote{}, err
	}
	return quote, nil
}
