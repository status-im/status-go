package coingecko

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
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

	expectedMap := map[string]GeckoToken{
		"ETH": {
			ID:     "ethereum",
			Symbol: "eth",
			Name:   "Ethereum",
		},
		"SNT": {
			ID:     "status",
			Symbol: "snt",
			Name:   "Status",
		},
	}
	response, _ := json.Marshal(expected)

	srv, stop := setupTest(t, response)
	defer stop()

	geckoClient := &Client{
		client:    srv.Client(),
		tokens:    make(map[string]GeckoToken),
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
		tokens:    make(map[string]GeckoToken),
		tokensURL: srv.URL,
	}

	_, err := geckoClient.getTokens()
	require.Error(t, err)
}
