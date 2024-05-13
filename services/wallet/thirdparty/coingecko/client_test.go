package coingecko

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
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
		"ETH": []GeckoToken{{
			ID:     "ethereum",
			Symbol: "eth",
			Name:   "Ethereum",
		}},
		"SNT": []GeckoToken{{
			ID:     "status",
			Symbol: "snt",
			Name:   "Status",
		}},
	}
	response, _ := json.Marshal(expected)

	srv, stop := setupTest(t, response)
	defer stop()

	geckoClient := &Client{
		client:    srv.Client(),
		tokens:    make(map[string][]GeckoToken),
		tokensURL: srv.URL,
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
		client:    srv.Client(),
		tokens:    make(map[string][]GeckoToken),
		tokensURL: srv.URL,
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
		client:    srv.Client(),
		tokens:    make(map[string][]GeckoToken),
		tokensURL: srv.URL,
	}

	_, err := geckoClient.getTokens()
	require.Error(t, err)
}
