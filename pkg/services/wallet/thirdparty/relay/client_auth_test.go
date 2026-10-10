package relay

import (
	"bytes"
	"encoding/json"
	"math/big"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestKeyedSepoliaQuoteOmitsAPIKeyAndReferrer(t *testing.T) {
	client := NewClient(walletcommon.EthereumMainnet, ReferrerDev, "mainnet-key")
	params := QuoteParams{
		FromChainID: walletcommon.EthereumSepolia,
		ToChainID:   walletcommon.BaseSepolia,
		FromToken:   common.HexToAddress("0x0000000000000000000000000000000000000001"),
		ToToken:     common.HexToAddress("0x0000000000000000000000000000000000000002"),
		FromAddress: common.HexToAddress("0x0000000000000000000000000000000000000003"),
		ToAddress:   common.HexToAddress("0x0000000000000000000000000000000000000004"),
		AmountIn:    big.NewInt(1000),
	}

	mainnet := quoteRequest(t, client, params)
	require.Equal(t, MainnetBaseURL+"/quote/v2", mainnet.URL.String())
	require.Equal(t, "mainnet-key", mainnet.Header.Get(apiKeyHeader))
	require.Equal(t, ReferrerDev, decodeQuoteBody(t, mainnet)["referrer"])

	client.SetChainID(walletcommon.EthereumSepolia)
	require.Equal(t, TestnetBaseURL, client.BaseURL())

	testnet := quoteRequest(t, client, params)
	require.Equal(t, TestnetBaseURL+"/quote/v2", testnet.URL.String())
	require.Empty(t, testnet.Header.Get(apiKeyHeader))
	body := decodeQuoteBody(t, testnet)
	require.NotContains(t, body, "referrer")
	require.Contains(t, body, "appFees")
}

func quoteRequest(t *testing.T, client *Client, params QuoteParams) *http.Request {
	t.Helper()
	payload, err := json.Marshal(client.quoteRequestBody(params))
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, client.BaseURL()+"/quote/v2", bytes.NewReader(payload))
	require.NoError(t, err)
	for _, opt := range client.requestOptions() {
		opt(req, nil)
	}
	return req
}

func decodeQuoteBody(t *testing.T, req *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
	return body
}
