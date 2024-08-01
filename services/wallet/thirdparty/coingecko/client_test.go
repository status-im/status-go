package coingecko

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type TestTokenPlatform struct {
	Ethereum string `json:"ethereum"`
	Arb      string `json:"arb"`
}

type TestGeckoToken struct {
	ID        string            `json:"id"`
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
	Platforms TestTokenPlatform `json:"platforms"`
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
			ID:     "ethereum",
			Symbol: "eth",
			Name:   "Ethereum",
		},
		{
			ID:     "status",
			Symbol: "snt",
			Name:   "Status",
		},
	}

	expectedMap := map[string][]GeckoToken{
		"ETH": {{
			ID:     "ethereum",
			Symbol: "eth",
			Name:   "Ethereum",
		}},
		"SNT": {{
			ID:     "status",
			Symbol: "snt",
			Name:   "Status",
		}},
	}
	response, _ := json.Marshal(expected)

	srv, stop := setupTest(t, response)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		tokens:     make(map[string][]GeckoToken),
		tokensURL:  srv.URL,
	}

	tokenMap, err := geckoClient.getTokens()
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedMap, tokenMap))
}

func TestGetTokensEthPlatform(t *testing.T) {
	tokenList := []TestGeckoToken{
		{
			ID:     "ethereum",
			Symbol: "eth-test",
			Name:   "Ethereum",
			Platforms: TestTokenPlatform{
				Ethereum: "0x123",
			},
		},
		{
			ID:     "usdt-bridge-test",
			Symbol: "usdt-test",
			Name:   "USDT Bridge Test",
			Platforms: TestTokenPlatform{
				Arb: "0x123",
			},
		},
		{
			ID:     "tether",
			Symbol: "usdt-test",
			Name:   "Tether",
			Platforms: TestTokenPlatform{
				Arb:      "0x1234",
				Ethereum: "0x12345",
			},
		},
		{
			ID:     "AirDao",
			Symbol: "amb-test",
			Name:   "Amber",
			Platforms: TestTokenPlatform{
				Arb: "0x123455",
			},
		},
	}

	expectedMap := map[string][]GeckoToken{
		"ETH-TEST": {{
			ID:          "ethereum",
			Symbol:      "eth-test",
			Name:        "Ethereum",
			EthPlatform: true,
		}},
		"USDT-TEST": {
			{
				ID:          "usdt-bridge-test",
				Symbol:      "usdt-test",
				Name:        "USDT Bridge Test",
				EthPlatform: false,
			},
			{
				ID:          "tether",
				Symbol:      "usdt-test",
				Name:        "Tether",
				EthPlatform: true,
			},
		},
		"AMB-TEST": {{
			ID:          "AirDao",
			Symbol:      "amb-test",
			Name:        "Amber",
			EthPlatform: false,
		}},
	}

	response, _ := json.Marshal(tokenList)

	srv, stop := setupTest(t, response)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		tokens:     make(map[string][]GeckoToken),
		tokensURL:  srv.URL,
	}

	tokenMap, err := geckoClient.getTokens()
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedMap, tokenMap))
}

func TestGetTokensFailure(t *testing.T) {
	resp := []byte{}
	srv, stop := setupTest(t, resp)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		tokens:     make(map[string][]GeckoToken),
		tokensURL:  srv.URL,
	}

	_, err := geckoClient.getTokens()
	require.Error(t, err)
}

func TestFetchPrices(t *testing.T) {
	mux := http.NewServeMux()

	// Register handlers for different URL paths
	mux.HandleFunc("/coins/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := "[{\"id\":\"ethereum\",\"symbol\":\"eth\",\"name\":\"Ethereum\",\"platforms\":{\"ethereum\":\"0x5e21d1ee5cf0077b314c381720273ae82378d613\"}},{\"id\":\"status\",\"symbol\":\"snt\",\"name\":\"Status\",\"platforms\":{\"ethereum\":\"0x78ba134c3ace18e69837b01703d07f0db6fb0a60\"}}]"
		_, _ = w.Write([]byte(response))
	})

	mux.HandleFunc("/simple/price", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := "{\"ethereum\":{\"usd\":3181.32},\"status\":{\"usd\":0.02391704}}"
		_, _ = w.Write([]byte(response))
	})

	srv := httptest.NewServer(mux)

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		tokens:     make(map[string][]GeckoToken),
		tokensURL:  srv.URL + "/coins/list",
	}

	symbols := []string{"ETH", "SNT", "UNSUPPORTED", "TOKENS"}
	prices, err := geckoClient.FetchPrices(symbols, []string{"USD"})
	require.NoError(t, err)
	require.Len(t, prices, 2)
}

func TestFetchMarketValues(t *testing.T) {
	mux := http.NewServeMux()

	// Register handlers for different URL paths
	mux.HandleFunc("/coins/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := "[{\"id\":\"ethereum\",\"symbol\":\"eth\",\"name\":\"Ethereum\",\"platforms\":{\"ethereum\":\"0x5e21d1ee5cf0077b314c381720273ae82378d613\"}},{\"id\":\"status\",\"symbol\":\"snt\",\"name\":\"Status\",\"platforms\":{\"ethereum\":\"0x78ba134c3ace18e69837b01703d07f0db6fb0a60\"}}]"
		_, _ = w.Write([]byte(response))
	})

	mux.HandleFunc("/coins/markets", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := "[{\"id\":\"ethereum\",\"symbol\":\"eth\",\"name\":\"Ethereum\",\"image\":\"https://coin-images.coingecko.com/coins/images/279/large/ethereum.png?1696501628\",\"current_price\":3177.16,\"market_cap\":382035912506,\"market_cap_rank\":2,\"fully_diluted_valuation\":382035912506,\"total_volume\":18958367285,\"high_24h\":3325.57,\"low_24h\":3139.38,\"price_change_24h\":-146.70781392198978,\"price_change_percentage_24h\":-4.41377,\"market_cap_change_24h\":-17315836985.42914,\"market_cap_change_percentage_24h\":-4.33599,\"circulating_supply\":120251313.934882,\"total_supply\":120251313.934882,\"max_supply\":null,\"ath\":4878.26,\"ath_change_percentage\":-34.74074,\"ath_date\":\"2021-11-10T14:24:19.604Z\",\"atl\":0.432979,\"atl_change_percentage\":735159.10684,\"atl_date\":\"2015-10-20T00:00:00.000Z\",\"roi\":{\"times\":64.75457822761112,\"currency\":\"btc\",\"percentage\":6475.457822761112},\"last_updated\":\"2024-08-01T14:17:02.604Z\",\"price_change_percentage_1h_in_currency\":-0.14302683386053758,\"price_change_percentage_24h_in_currency\":-4.413773698570276},{\"id\":\"status\",\"symbol\":\"snt\",\"name\":\"Status\",\"image\":\"https://coin-images.coingecko.com/coins/images/779/large/status.png?1696501931\",\"current_price\":0.02387956,\"market_cap\":94492012,\"market_cap_rank\":420,\"fully_diluted_valuation\":162355386,\"total_volume\":3315607,\"high_24h\":0.02528227,\"low_24h\":0.02351923,\"price_change_24h\":-0.001177587387552543,\"price_change_percentage_24h\":-4.69961,\"market_cap_change_24h\":-5410268.579258412,\"market_cap_change_percentage_24h\":-5.41556,\"circulating_supply\":3960483788.3096976,\"total_supply\":6804870174.0,\"max_supply\":null,\"ath\":0.684918,\"ath_change_percentage\":-96.50467,\"ath_date\":\"2018-01-03T00:00:00.000Z\",\"atl\":0.00592935,\"atl_change_percentage\":303.75704,\"atl_date\":\"2020-03-13T02:10:36.877Z\",\"roi\":null,\"last_updated\":\"2024-08-01T14:16:20.805Z\",\"price_change_percentage_1h_in_currency\":-0.21239208982552796,\"price_change_percentage_24h_in_currency\":-4.699606730698922}]"
		_, _ = w.Write([]byte(response))
	})

	srv := httptest.NewServer(mux)

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		tokens:     make(map[string][]GeckoToken),
		tokensURL:  srv.URL + "/coins/list",
	}

	symbols := []string{"ETH", "SNT", "UNSUPPORTED", "TOKENS"}
	prices, err := geckoClient.FetchTokenMarketValues(symbols, "USD")
	require.NoError(t, err)
	require.Len(t, prices, 2)
}
