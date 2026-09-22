package relay_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/relay"
)

const sampleQuoteResponse = `{
	"requestId": "0x92b99e6e1ee1deeb9531b5ad7f87091b3d71254b3176de9e8b5f6c6d0bd3a331",
	"steps": [
		{
			"id": "approve",
			"action": "Approve USDC",
			"description": "Approve the router to spend USDC",
			"kind": "transaction",
			"requestId": "0x92b99e6e1ee1deeb9531b5ad7f87091b3d71254b3176de9e8b5f6c6d0bd3a331",
			"items": [
				{
					"status": "incomplete",
					"data": {
						"from": "0x0CccD55A5Ac261Ea29136831eeaA93bfE07f5Db6",
						"to": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
						"data": "0x095ea7b3",
						"value": "0",
						"maxFeePerGas": "12205661344",
						"maxPriorityFeePerGas": "2037863396",
						"chainId": 8453
					}
				}
			]
		},
		{
			"id": "swap",
			"action": "Confirm transaction in your wallet",
			"description": "Depositing funds to the relayer to execute the swap",
			"kind": "transaction",
			"requestId": "0x92b99e6e1ee1deeb9531b5ad7f87091b3d71254b3176de9e8b5f6c6d0bd3a331",
			"items": [
				{
					"status": "incomplete",
					"data": {
						"from": "0x0CccD55A5Ac261Ea29136831eeaA93bfE07f5Db6",
						"to": "0xaaaaaaae92cc1ceef79a038017889fdd26d23d4d",
						"data": "0x00fad611",
						"value": "1000000000000000000",
						"gas": 210000,
						"maxFeePerGas": "12205661344",
						"maxPriorityFeePerGas": "2037863396",
						"chainId": 8453
					},
					"check": {
						"endpoint": "/intents/status?requestId=0x92b99e6e1ee1deeb9531b5ad7f87091b3d71254b3176de9e8b5f6c6d0bd3a331",
						"method": "GET"
					}
				}
			]
		}
	],
	"fees": {
		"gas": {"currency": {"chainId": 8453, "address": "0x0000000000000000000000000000000000000000", "symbol": "ETH", "decimals": 18}, "amount": "20000000000000", "amountUsd": "0.05"},
		"relayer": {"currency": {"chainId": 8453, "address": "0x0000000000000000000000000000000000000000", "symbol": "ETH", "decimals": 18}, "amount": "30000000000000", "amountUsd": "0.07"},
		"app": {"currency": {"chainId": 8453, "address": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", "symbol": "USDC", "decimals": 6}, "amount": "6200", "amountUsd": "0.0062"}
	},
	"details": {
		"operation": "swap",
		"sender": "0x0CccD55A5Ac261Ea29136831eeaA93bfE07f5Db6",
		"recipient": "0x0CccD55A5Ac261Ea29136831eeaA93bfE07f5Db6",
		"timeEstimate": 15,
		"currencyIn": {"currency": {"chainId": 8453, "address": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", "symbol": "USDC", "decimals": 6}, "amount": "1000000", "amountUsd": "1.00"},
		"currencyOut": {"currency": {"chainId": 8453, "address": "0x0000000000000000000000000000000000000000", "symbol": "ETH", "decimals": 18}, "amount": "300000000000000", "minimumAmount": "298500000000000", "amountUsd": "0.99"},
		"rate": "0.0003",
		"route": {
			"origin": {"inputCurrency": {"amount": "1000000"}, "outputCurrency": {"amount": "300000000000000"}, "router": "0x"},
			"destination": {"inputCurrency": {"amount": "300000000000000"}, "outputCurrency": {"amount": "300000000000000"}, "router": "kyberswap"}
		}
	}
}`

func TestDecodeQuote(t *testing.T) {
	var quote relay.Quote
	require.NoError(t, json.Unmarshal([]byte(sampleQuoteResponse), &quote))

	require.Equal(t, "0x92b99e6e1ee1deeb9531b5ad7f87091b3d71254b3176de9e8b5f6c6d0bd3a331", quote.RequestID)
	require.Len(t, quote.Steps, 2)

	approve := quote.Steps[0]
	require.Equal(t, relay.StepIDApprove, approve.ID)
	require.Equal(t, relay.StepKindTransaction, approve.Kind)
	require.Len(t, approve.Items, 1)
	require.Equal(t, "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", approve.Items[0].Data.To)
	require.Equal(t, "0", approve.Items[0].Data.Value.String())
	require.Nil(t, approve.Items[0].Data.Gas)

	swap := quote.Steps[1]
	require.Equal(t, "swap", swap.ID)
	data := swap.Items[0].Data
	require.Equal(t, "0xaaaaaaae92cc1ceef79a038017889fdd26d23d4d", data.To)
	require.Equal(t, "0x00fad611", data.Data)
	require.Equal(t, "1000000000000000000", data.Value.String())
	require.Equal(t, "210000", data.Gas.String())
	require.Equal(t, "12205661344", data.MaxFeePerGas.String())
	require.Equal(t, uint64(8453), data.ChainID)
	require.Equal(t, "GET", swap.Items[0].Check.Method)

	require.Equal(t, "20000000000000", quote.Fees.Gas.Amount.String())
	require.Equal(t, "6200", quote.Fees.App.Amount.String())
	require.Equal(t, uint64(8453), quote.Fees.App.Currency.ChainID)

	require.Equal(t, "swap", quote.Details.Operation)
	require.Equal(t, 15, quote.Details.TimeEstimate)
	require.Equal(t, "1000000", quote.Details.CurrencyIn.Amount.String())
	require.Equal(t, "300000000000000", quote.Details.CurrencyOut.Amount.String())
	require.Equal(t, "298500000000000", quote.Details.CurrencyOut.MinimumAmount.String())
	require.Equal(t, "0x", quote.Details.Route.Origin.Router)
	require.Equal(t, "kyberswap", quote.Details.Route.Destination.Router)
}

func baseQuoteParams() relay.QuoteParams {
	return relay.QuoteParams{
		FromChainID:        walletCommon.BaseMainnet,
		ToChainID:          walletCommon.OptimismMainnet,
		FromToken:          common.HexToAddress("0x0000000000000000000000000000000000000001"),
		ToToken:            common.HexToAddress("0x0000000000000000000000000000000000000002"),
		FromAddress:        common.HexToAddress("0x0000000000000000000000000000000000000003"),
		ToAddress:          common.HexToAddress("0x0000000000000000000000000000000000000004"),
		AmountIn:           big.NewInt(1000),
		SlippagePercentage: 0.5,
	}
}

// quoteBodySeenBy fetches a quote against a stub server and returns the JSON body the client sent.
func quoteBodySeenBy(t *testing.T, client func(baseURL string) *relay.Client, p relay.QuoteParams) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/quote/v2", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_, err := w.Write([]byte("{}"))
		require.NoError(t, err)
	}))
	defer srv.Close()

	_, err := client(srv.URL).FetchQuote(context.Background(), p)
	require.NoError(t, err)
	return body
}

func unkeyedClient(baseURL string) *relay.Client {
	return relay.NewClientWithBaseURL(baseURL, walletCommon.BaseMainnet, relay.ReferrerDev, "")
}

func TestQuoteRequestBody(t *testing.T) {
	body := quoteBodySeenBy(t, unkeyedClient, baseQuoteParams())

	require.Equal(t, "0x0000000000000000000000000000000000000003", body["user"])
	require.EqualValues(t, walletCommon.BaseMainnet, body["originChainId"])
	require.EqualValues(t, walletCommon.OptimismMainnet, body["destinationChainId"])
	require.Equal(t, "0x0000000000000000000000000000000000000001", body["originCurrency"])
	require.Equal(t, "0x0000000000000000000000000000000000000002", body["destinationCurrency"])
	require.Equal(t, "1000", body["amount"])
	require.Equal(t, "EXACT_INPUT", body["tradeType"])
	require.Equal(t, "0x0000000000000000000000000000000000000004", body["recipient"])
	require.Equal(t, "50", body["slippageTolerance"])
	require.Equal(t, true, body["useRouteRacing"])

	// Relay rejects a referrer without a key, so it's only sent alongside one.
	require.NotContains(t, body, "referrer")
	keyed := quoteBodySeenBy(t, func(baseURL string) *relay.Client {
		return relay.NewClientWithBaseURL(baseURL, walletCommon.BaseMainnet, relay.ReferrerProd, "key")
	}, baseQuoteParams())
	require.Equal(t, relay.ReferrerProd, keyed["referrer"])

	// app fee: 65 bps to the Status fee address
	appFees, ok := body["appFees"].([]interface{})
	require.True(t, ok)
	require.Len(t, appFees, 1)
	fee := appFees[0].(map[string]interface{})
	require.Equal(t, relay.AppFeeRecipient.Hex(), fee["recipient"])
	require.Equal(t, "65", fee["fee"])
}

func TestQuoteRequestBodyDefaults(t *testing.T) {
	p := baseQuoteParams()
	p.ToChainID = 0
	p.SlippagePercentage = 0
	body := quoteBodySeenBy(t, unkeyedClient, p)
	require.EqualValues(t, walletCommon.BaseMainnet, body["destinationChainId"])
	require.NotContains(t, body, "slippageTolerance") // let Relay auto-calculate

	orig := relay.AppFeeRecipient
	relay.AppFeeRecipient = common.Address{}
	defer func() { relay.AppFeeRecipient = orig }()
	require.NotContains(t, quoteBodySeenBy(t, unkeyedClient, baseQuoteParams()), "appFees")
}

func TestFetchQuoteAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"message":"No routes found","errorCode":"NO_SWAP_ROUTES_FOUND"}`))
		require.NoError(t, err)
	}))
	defer srv.Close()

	client := relay.NewClientWithBaseURL(srv.URL, walletCommon.BaseMainnet, relay.ReferrerDev, "")

	_, err := client.FetchQuote(context.Background(), baseQuoteParams())
	require.Error(t, err)
	var apiErr *relay.APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, "NO_SWAP_ROUTES_FOUND", apiErr.Code)
	require.Equal(t, "No routes found", apiErr.Message)
	require.Contains(t, err.Error(), "NO_SWAP_ROUTES_FOUND")
}

func TestFetchQuoteDecodesSteps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/quote/v2", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		_, err := w.Write([]byte(sampleQuoteResponse))
		require.NoError(t, err)
	}))
	defer srv.Close()

	client := relay.NewClientWithBaseURL(srv.URL, walletCommon.BaseMainnet, relay.ReferrerDev, "")

	quote, err := client.FetchQuote(context.Background(), baseQuoteParams())
	require.NoError(t, err)
	require.Len(t, quote.Steps, 2)
}
