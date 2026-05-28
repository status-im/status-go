package coingecko

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/common"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/builder"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/pkg/security"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"

	"github.com/stretchr/testify/require"
)

func TestCoingeckoAPIKeyInQueryProOverDemo(t *testing.T) {
	var sawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	client := NewClientWithParams(Params{
		URL:                 srv.URL,
		CoingeckoAPIKey:     security.NewSensitiveString("pro-secret"),
		CoingeckoDemoAPIKey: security.NewSensitiveString("demo-secret"),
	})
	_, err := client.fetchTokens(context.Background())
	require.NoError(t, err)
	require.Contains(t, sawQuery, "x_cg_pro_api_key=pro-secret")
	require.NotContains(t, sawQuery, "demo-secret")
}

func TestCoingeckoAPIKeyInQueryDemoWhenNoPro(t *testing.T) {
	var sawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	client := NewClientWithParams(Params{
		URL:                 srv.URL,
		CoingeckoDemoAPIKey: security.NewSensitiveString("demo-only"),
	})
	_, err := client.fetchTokens(context.Background())
	require.NoError(t, err)
	require.Contains(t, sawQuery, "x_cg_demo_api_key=demo-only")
}

func setupTest(t *testing.T, response []byte) (*httptest.Server, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write(response)
		if err != nil {
			return
		}
	}))

	return srv, func() {
		srv.Close()
	}
}

func TestGetTokensSuccess(t *testing.T) {
	expected := []GeckoToken{
		{
			ID:     "usdc-coin",
			Symbol: "usdc",
			Name:   "USDC",
			Platforms: map[string]string{
				"ethereum":            "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"unichain":            "0x078d782b760474a361dda0af3839290b0ef57ad6",
				"optimistic-ethereum": "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
				"arbitrum-one":        "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			},
		},
		{
			ID:     "status",
			Symbol: "snt",
			Name:   "Status",
			Platforms: map[string]string{
				"ethereum": "0x744d70fdbe2ba4cf95131626614a1763df805b9e",
				"energi":   "0x6bb14afedc740dce4904b7a65807fe3b967f4c94",
			},
		},
	}

	expectedMap := map[string]GeckoToken{
		"1-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48":     expected[0],
		"10-0x0b2c639c533813f4aa9d7837caf62653d097ff85":    expected[0],
		"42161-0xaf88d065e77c8cc2239327c5edb3a432268e5831": expected[0],
		"1-0x744d70fdbe2ba4cf95131626614a1763df805b9e":     expected[1],
	}

	for _, chainID := range walletcommon.AllChainIDsAsUint64() {
		token := tokentypes.Token{Token: &types.Token{ChainID: chainID}}

		if chainID == common.BSCMainnet || chainID == common.BSCTestnet {
			expectedMap[token.Key()] = GeckoToken{
				ID:     nativeBNBTokenID,
				Name:   builder.BinanceSmartChainNativeName,
				Symbol: builder.BinanceSmartChainNativeSymbol,
			}
			continue
		}

		expectedMap[token.Key()] = GeckoToken{
			ID:     nativeEthTokenID,
			Name:   builder.EthereumNativeName,
			Symbol: builder.EthereumNativeSymbol,
		}
	}

	response, _ := json.Marshal(expected)

	srv, stop := setupTest(t, response)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	tokenMap, err := geckoClient.getCoingeckoTokensByTokenKey()
	require.NoError(t, err)

	for tokenKey, token := range expectedMap {
		require.Equal(t, token, tokenMap[strings.ToLower(tokenKey)])
	}
}

func TestGetTokensFailure(t *testing.T) {
	resp := []byte{}
	srv, stop := setupTest(t, resp)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	tokenMap, err := geckoClient.getCoingeckoTokensByTokenKey()
	require.Error(t, err)
	require.Nil(t, tokenMap)
}

func TestFetchPrices(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/coins/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := `[
{"id":"usd-coin","symbol":"usdc","name":"USDC","platforms":{"ethereum":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}},
{"id":"status","symbol":"snt","name":"Status","platforms":{"ethereum":"0x744d70fdbe2ba4cf95131626614a1763df805b9e"}}
]`
		_, _ = w.Write([]byte(response))
	})

	mux.HandleFunc("/simple/price", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := `{"usd-coin":{"usd":1.001},"status":{"usd":0.02391704}}`
		_, _ = w.Write([]byte(response))
	})

	srv := httptest.NewServer(mux)

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	tokens := []*tokentypes.Token{
		{
			Token: &types.Token{
				Name:    "USDC",
				Symbol:  "USDC",
				ChainID: common.EthereumMainnet,
				Address: gethcommon.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
			},
		},
		{
			Token: &types.Token{
				Name:    "Status",
				Symbol:  "SNT",
				ChainID: common.EthereumMainnet,
				Address: gethcommon.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"),
			},
		},
		{
			Token: &types.Token{
				Name:    "Dai",
				Symbol:  "DAI",
				ChainID: common.EthereumMainnet,
				Address: gethcommon.HexToAddress("0x6b175474e89094c44da98b954eedeac495271d0f"),
			},
		},
	}
	prices, err := geckoClient.FetchPrices(tokens, []string{"USD"})
	require.NoError(t, err)
	require.NotNil(t, prices)
	require.Len(t, prices, len(tokens))

	require.Equal(t, prices[tokens[0].Key()], map[string]float64{"USD": 1.001})
	require.Equal(t, prices[tokens[1].Key()], map[string]float64{"USD": 0.02391704})
	require.Equal(t, prices[tokens[2].Key()], map[string]float64{"USD": 0})
}

func TestAggregatePrices(t *testing.T) {
	prices := []thirdparty.HistoricalPrice{
		{Timestamp: 100, Value: 10.0},
		{Timestamp: 200, Value: 20.0},
		{Timestamp: 300, Value: 30.0},
		{Timestamp: 400, Value: 40.0},
		{Timestamp: 500, Value: 50.0},
		{Timestamp: 600, Value: 60.0},
	}

	t.Run("aggregate=1 is a no-op", func(t *testing.T) {
		result := aggregatePrices(prices, 1)
		require.Equal(t, prices, result)
	})

	t.Run("empty slice returns empty slice", func(t *testing.T) {
		result := aggregatePrices([]thirdparty.HistoricalPrice{}, 3)
		require.Empty(t, result)
	})

	t.Run("aggregate=2 groups pairs and averages values", func(t *testing.T) {
		result := aggregatePrices(prices, 2)
		require.Len(t, result, 3)
		// group [0,1]: avg=(10+20)/2=15, timestamp=200
		require.Equal(t, int64(200), result[0].Timestamp)
		require.InDelta(t, 15.0, result[0].Value, 1e-10)
		// group [2,3]: avg=(30+40)/2=35, timestamp=400
		require.Equal(t, int64(400), result[1].Timestamp)
		require.InDelta(t, 35.0, result[1].Value, 1e-10)
		// group [4,5]: avg=(50+60)/2=55, timestamp=600
		require.Equal(t, int64(600), result[2].Timestamp)
		require.InDelta(t, 55.0, result[2].Value, 1e-10)
	})

	t.Run("aggregate=4 with remainder group", func(t *testing.T) {
		result := aggregatePrices(prices, 4)
		require.Len(t, result, 2)
		// group [0..3]: avg=(10+20+30+40)/4=25, timestamp=400
		require.Equal(t, int64(400), result[0].Timestamp)
		require.InDelta(t, 25.0, result[0].Value, 1e-10)
		// remainder [4,5]: avg=(50+60)/2=55, timestamp=600
		require.Equal(t, int64(600), result[1].Timestamp)
		require.InDelta(t, 55.0, result[1].Value, 1e-10)
	})

	t.Run("aggregate >= len returns single group", func(t *testing.T) {
		result := aggregatePrices(prices, 10)
		require.Len(t, result, 1)
		require.Equal(t, int64(600), result[0].Timestamp)
		require.InDelta(t, (10.0+20+30+40+50+60)/6, result[0].Value, 1e-10)
	})
}

func newSNTServer(t *testing.T, pricesJSON string) (*httptest.Server, *tokentypes.Token) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/coins/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"status","symbol":"snt","name":"Status","platforms":{"ethereum":"0x744d70fdbe2ba4cf95131626614a1763df805b9e"}}]`))
	})
	mux.HandleFunc("/coins/status/market_chart", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"prices":` + pricesJSON + `}`))
	})
	token := &tokentypes.Token{Token: &types.Token{
		Name:    "Status",
		Symbol:  "SNT",
		ChainID: common.EthereumMainnet,
		Address: gethcommon.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"),
	}}
	return httptest.NewServer(mux), token
}

func TestFetchHistoricalDailyPrices(t *testing.T) {
	tests := []struct {
		name      string
		prices    string
		aggregate int
		want      []thirdparty.HistoricalPrice
	}{
		{
			name:      "timestamps converted from ms to seconds",
			prices:    `[[1737889461000,1.0],[1737893061000,2.0],[1737896661000,3.0]]`,
			aggregate: 1,
			want: []thirdparty.HistoricalPrice{
				{Timestamp: 1737889461, Value: 1.0},
				{Timestamp: 1737893061, Value: 2.0},
				{Timestamp: 1737896661, Value: 3.0},
			},
		},
		{
			name:      "aggregate=3 averages groups",
			prices:    `[[1000,10.0],[2000,20.0],[3000,30.0],[4000,40.0],[5000,50.0],[6000,60.0]]`,
			aggregate: 3,
			want: []thirdparty.HistoricalPrice{
				{Timestamp: 3, Value: 20.0},
				{Timestamp: 6, Value: 50.0},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, token := newSNTServer(t, tc.prices)
			defer srv.Close()
			client := &Client{httpClient: thirdparty.NewHTTPClient(), baseURL: srv.URL}
			result, err := client.FetchHistoricalDailyPrices(token, "usd", len(tc.want)*tc.aggregate, false, tc.aggregate)
			require.NoError(t, err)
			require.Equal(t, tc.want, result)
		})
	}
}

func TestFetchMarketValues(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/coins/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := `[
{"id":"usd-coin","symbol":"usdc","name":"USDC","platforms":{"ethereum":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}},
{"id":"status","symbol":"snt","name":"Status","platforms":{"ethereum":"0x744d70fdbe2ba4cf95131626614a1763df805b9e"}}
]`
		_, _ = w.Write([]byte(response))
	})

	mux.HandleFunc("/coins/markets", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := `[
	{
			"id": "usd-coin",
			"symbol": "usdc",
			"name": "USDC",
			"image": "https://coin-images.coingecko.com/coins/images/6319/large/usdc.png?1696506694",
			"current_price": 0.999812,
			"market_cap": 72388338754,
			"market_cap_rank": 7,
			"fully_diluted_valuation": 72386900172,
			"total_volume": 14111755948,
			"high_24h": 0.999835,
			"low_24h": 0.999619,
			"price_change_24h": 6.21125e-07,
			"price_change_percentage_24h": 6.0e-05,
			"market_cap_change_24h": 157976580,
			"market_cap_change_percentage_24h": 0.21871,
			"circulating_supply": 72402939072.45718,
			"total_supply": 72401500200.07101,
			"max_supply": null,
			"ath": 1.17,
			"ath_change_percentage": -14.74306,
			"ath_date": "2019-05-08T00:40:28.300Z",
			"atl": 0.877647,
			"atl_change_percentage": 13.91981,
			"atl_date": "2023-03-11T08:02:13.981Z",
			"roi": null,
			"last_updated": "2025-09-12T07:18:29.101Z",
			"price_change_percentage_1h_in_currency": 0.00874017695509919,
			"price_change_percentage_24h_in_currency": 6.212417017400316e-05
	},
	{
			"id": "status",
			"symbol": "snt",
			"name": "Status",
			"image": "https://coin-images.coingecko.com/coins/images/779/large/status.png?1696501931",
			"current_price": 0.02611135,
			"market_cap": 103479828,
			"market_cap_rank": 528,
			"fully_diluted_valuation": 177798177,
			"total_volume": 8813352,
			"high_24h": 0.02612809,
			"low_24h": 0.02565221,
			"price_change_24h": 3.819e-05,
			"price_change_percentage_24h": 0.14647,
			"market_cap_change_24h": 217827,
			"market_cap_change_percentage_24h": 0.21095,
			"circulating_supply": 3960483788.3096976,
			"total_supply": 6804870174.0,
			"max_supply": null,
			"ath": 0.684918,
			"ath_change_percentage": -96.18926,
			"ath_date": "2018-01-03T00:00:00.000Z",
			"atl": 0.00592935,
			"atl_change_percentage": 340.1917,
			"atl_date": "2020-03-13T02:10:36.877Z",
			"roi": null,
			"last_updated": "2025-09-12T07:18:18.390Z",
			"price_change_percentage_1h_in_currency": 0.05599335988029603,
			"price_change_percentage_24h_in_currency": 0.14647083029811012
	}
]`
		_, _ = w.Write([]byte(response))
	})

	srv := httptest.NewServer(mux)

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	tokens := []*tokentypes.Token{
		{
			Token: &types.Token{
				CrossChainID: "usd-coin",
				Name:         "USDC",
				Symbol:       "USDC",
				ChainID:      common.EthereumMainnet,
				Address:      gethcommon.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
			},
		},
		{
			Token: &types.Token{
				CrossChainID: "usd-coin",
				Name:         "USDC",
				Symbol:       "USDC",
				ChainID:      common.EthereumSepolia,
				Address:      gethcommon.HexToAddress("0x1c7d4b196cb0c7b01d743fbc6116a902379c7238"),
			},
		},
		{
			Token: &types.Token{
				CrossChainID: "status",
				Name:         "Status",
				Symbol:       "SNT",
				ChainID:      common.EthereumMainnet,
				Address:      gethcommon.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"),
			},
		},
		{
			Token: &types.Token{
				CrossChainID: "dai",
				Name:         "Dai",
				Symbol:       "DAI",
				ChainID:      common.EthereumMainnet,
				Address:      gethcommon.HexToAddress("0x6b175474e89094c44da98b954eedeac495271d0f"),
			},
		},
	}
	prices, err := geckoClient.FetchTokenMarketValues(tokens, "USD")
	require.NoError(t, err)
	require.Len(t, prices, len(tokens))

	for _, index := range []int{0, 1} {
		usdcPrice := prices[tokens[index].Key()]
		require.InDelta(t, 72388338754.0, usdcPrice.MKTCAP, 1e-6)
		require.InDelta(t, 0.999835, usdcPrice.HIGHDAY, 1e-10)
		require.InDelta(t, 0.999619, usdcPrice.LOWDAY, 1e-10)
		require.InDelta(t, 0.00874017695509919, usdcPrice.CHANGEPCTHOUR, 1e-10)
		require.InDelta(t, 6.0e-05, usdcPrice.CHANGEPCTDAY, 1e-5)
		require.InDelta(t, 6.0e-05, usdcPrice.CHANGEPCT24HOUR, 1e-5)
		require.InDelta(t, 6.21125e-07, usdcPrice.CHANGE24HOUR, 1e-10)
	}

	sntPrice := prices[tokens[2].Key()]
	require.InDelta(t, 103479828.0, sntPrice.MKTCAP, 1e-6)
	require.InDelta(t, 0.02612809, sntPrice.HIGHDAY, 1e-8)
	require.InDelta(t, 0.02565221, sntPrice.LOWDAY, 1e-8)
	require.InDelta(t, 0.05599335988029603, sntPrice.CHANGEPCTHOUR, 1e-8)
	require.InDelta(t, 0.14647083029811012, sntPrice.CHANGEPCTDAY, 1e-6)
	require.InDelta(t, 0.14647083029811012, sntPrice.CHANGEPCT24HOUR, 1e-6)
	require.InDelta(t, 3.819e-05, sntPrice.CHANGE24HOUR, 1e-8)

	require.Equal(t, prices[tokens[3].Key()], thirdparty.TokenMarketValues{})
}
