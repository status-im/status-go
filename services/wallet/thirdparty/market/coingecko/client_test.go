package coingecko

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

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

func TestGettingTokens(t *testing.T) {
	expectedTokenIDTokenMap := make(map[string]GeckoToken)
	for _, token := range coinsList {
		expectedTokenIDTokenMap[token.ID] = token
	}

	expectedKeyIDMap := map[string]string{
		"1-0X72E4F9F808C49A2A61DE9C5896298920DC4EEEA9": "harrypotterobamasonic10in",
		"1-0X744D70FDBE2BA4CF95131626614A1763DF805B9E": "status",
		"1-0X78BA134C3ACE18E69837B01703D07F0DB6FB0A60": "sentinel-bot-ai",
		"1-0X9B06F3C5DE42D4623D7A2BD940EC735103C68A76": "volta-club",
		"1-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48": "usd-coin",
		"10-": "bridged-usdt",
		"10-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48":       "usd-coin",
		"11155111-0X72E4F9F808C49A2A61DE9C5896298920DC4EEEA9": "harrypotterobamasonic10in",
		"11155111-0X744D70FDBE2BA4CF95131626614A1763DF805B9E": "status",
		"11155111-0X78BA134C3ACE18E69837B01703D07F0DB6FB0A60": "sentinel-bot-ai",
		"11155111-0X9B06F3C5DE42D4623D7A2BD940EC735103C68A76": "volta-club",
		"11155111-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48": "usd-coin",
		"11155420-": "bridged-usdt",
		"11155420-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48": "usd-coin",
		"42161-0X9B06F3C5DE42D4623D7A2BD940EC735103C68A76":    "volta-club",
		"42161-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48":    "usd-coin",
		"421614-0X9B06F3C5DE42D4623D7A2BD940EC735103C68A76":   "volta-club",
		"421614-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48":   "usd-coin",
		"8453-0X72E4F9F808C49A2A61DE9C5896298920DC4EEEA9":     "harrypotterobamasonic10in",
		"8453-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48":     "usd-coin",
		"84532-0X72E4F9F808C49A2A61DE9C5896298920DC4EEEA9":    "harrypotterobamasonic10in",
		"84532-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48":    "usd-coin",
	}

	srv, stop := setupTest(t, responseCoinsListData)
	defer stop()

	geckoClient := &Client{
		httpClient:      thirdparty.NewHTTPClient(),
		baseURL:         srv.URL,
		tokenIDTokenMap: make(map[string]GeckoToken),
		tokenKeyIDMap:   make(map[string]string),
	}

	tokenIDTokenMap, err := geckoClient.getTokenIDTokenMap()
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedTokenIDTokenMap, tokenIDTokenMap))

	tokenKeyIDMap, err := geckoClient.getTokenKeyIDMap()
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedKeyIDMap, tokenKeyIDMap))
}

func TestFetchPrices(t *testing.T) {
	var (
		ethTokenKeyChain1    = utils.MakeTokenKey(1, ethAddress)
		ethTokenKeyChain2    = utils.MakeTokenKey(11155111, ethAddress)
		statusTokenKeyChain1 = utils.MakeTokenKey(1, "0x78ba134c3ace18e69837b01703d07f0db6fb0a60")
		statusTokenKeyChain2 = utils.MakeTokenKey(10, "0x78ba134c3ace18e69837b01703d07f0db6fb0a60")
	)
	tokenKeys := []string{
		ethTokenKeyChain1,
		ethTokenKeyChain2,
		statusTokenKeyChain1,
		statusTokenKeyChain2,
		"UNSUPPORTED",
		"TOKENS",
	}

	var expectedPrices = make(map[string]map[string]float64)
	expectedPrices[ethTokenKeyChain1] = make(map[string]float64)
	expectedPrices[ethTokenKeyChain1]["USD"] = 3181.32
	expectedPrices[ethTokenKeyChain2] = expectedPrices[ethTokenKeyChain1]
	expectedPrices[statusTokenKeyChain1] = make(map[string]float64)
	expectedPrices[statusTokenKeyChain1]["USD"] = 0.02391704
	expectedPrices[statusTokenKeyChain2] = expectedPrices[statusTokenKeyChain1]
	expectedPrices["UNSUPPORTED"] = make(map[string]float64)
	expectedPrices["UNSUPPORTED"]["USD"] = 0
	expectedPrices["TOKENS"] = make(map[string]float64)
	expectedPrices["TOKENS"]["USD"] = 0

	mux := http.NewServeMux()

	// Register handlers for different URL paths
	mux.HandleFunc("/coins/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := "[{\"id\":\"ethereum\",\"symbol\":\"eth\",\"name\":\"Ethereum\",\"platforms\":{\"ethereum\":\"0x5e21d1ee5cf0077b314c381720273ae82378d613\"}},{\"id\":\"status\",\"symbol\":\"snt\",\"name\":\"Status\",\"platforms\":{\"ethereum\":\"0x78ba134c3ace18e69837b01703d07f0db6fb0a60\",\"optimistic-ethereum\":\"0x78ba134c3ace18e69837b01703d07f0db6fb0a60\"}}]"
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
		httpClient:      thirdparty.NewHTTPClient(),
		baseURL:         srv.URL,
		tokenIDTokenMap: make(map[string]GeckoToken),
		tokenKeyIDMap:   make(map[string]string),
	}

	prices, err := geckoClient.FetchPrices(tokenKeys, []string{"USD"})
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedPrices, prices))
}

func TestFetchMarketValues(t *testing.T) {
	var (
		ethTokenKeyChain1    = utils.MakeTokenKey(1, ethAddress)
		ethTokenKeyChain2    = utils.MakeTokenKey(11155111, ethAddress)
		statusTokenKeyChain1 = utils.MakeTokenKey(1, "0x78ba134c3ace18e69837b01703d07f0db6fb0a60")
		statusTokenKeyChain2 = utils.MakeTokenKey(10, "0x78ba134c3ace18e69837b01703d07f0db6fb0a60")
	)
	tokenKeys := []string{
		ethTokenKeyChain1,
		ethTokenKeyChain2,
		statusTokenKeyChain1,
		statusTokenKeyChain2,
		"UNSUPPORTED",
		"TOKENS",
	}

	var expectedMarketValues = make(map[string]thirdparty.TokenMarketValues)
	expectedMarketValues[ethTokenKeyChain1] = thirdparty.TokenMarketValues{
		MKTCAP:          3.82035912506e+11,
		HIGHDAY:         3325.57,
		LOWDAY:          3139.38,
		CHANGEPCTHOUR:   -0.14302683386053758,
		CHANGEPCTDAY:    -4.41377,
		CHANGEPCT24HOUR: -4.41377,
		CHANGE24HOUR:    -146.70781392198978,
	}
	expectedMarketValues[ethTokenKeyChain2] = expectedMarketValues[ethTokenKeyChain1]
	expectedMarketValues[statusTokenKeyChain1] = thirdparty.TokenMarketValues{
		MKTCAP:          9.4492012e+07,
		HIGHDAY:         0.02528227,
		LOWDAY:          0.02351923,
		CHANGEPCTHOUR:   -0.21239208982552796,
		CHANGEPCTDAY:    -4.69961,
		CHANGEPCT24HOUR: -4.69961,
		CHANGE24HOUR:    -0.001177587387552543,
	}
	expectedMarketValues[statusTokenKeyChain2] = expectedMarketValues[statusTokenKeyChain1]
	expectedMarketValues["UNSUPPORTED"] = thirdparty.TokenMarketValues{}
	expectedMarketValues["TOKENS"] = thirdparty.TokenMarketValues{}

	mux := http.NewServeMux()

	// Register handlers for different URL paths
	mux.HandleFunc("/coins/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := "[{\"id\":\"ethereum\",\"symbol\":\"eth\",\"name\":\"Ethereum\",\"platforms\":{\"ethereum\":\"0x5e21d1ee5cf0077b314c381720273ae82378d613\"}},{\"id\":\"status\",\"symbol\":\"snt\",\"name\":\"Status\",\"platforms\":{\"ethereum\":\"0x78ba134c3ace18e69837b01703d07f0db6fb0a60\",\"optimistic-ethereum\":\"0x78ba134c3ace18e69837b01703d07f0db6fb0a60\"}}]"
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
		httpClient:      thirdparty.NewHTTPClient(),
		baseURL:         srv.URL,
		tokenIDTokenMap: make(map[string]GeckoToken),
		tokenKeyIDMap:   make(map[string]string),
	}

	prices, err := geckoClient.FetchTokenMarketValues(tokenKeys, "USD")
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedMarketValues, prices))
}
