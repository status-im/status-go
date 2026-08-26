package coingecko

import (
	"context"
	"net/http"
	"net/http/httptest"
	netUrl "net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoinsMarketsCurrency(t *testing.T) {
	proxy := NewClientWithParams(Params{IsMarketProxy: true})
	direct := NewClientWithParams(Params{})

	t.Run("proxy converts anything but USD", func(t *testing.T) {
		// status-desktop sends the code uppercased.
		vsCurrency, convert := proxy.coinsMarketsCurrency("EUR")
		require.Equal(t, baseCurrency, vsCurrency)
		require.Equal(t, "eur", convert)
	})

	t.Run("proxy needs no conversion for USD", func(t *testing.T) {
		vsCurrency, convert := proxy.coinsMarketsCurrency("USD")
		require.Equal(t, "USD", vsCurrency)
		require.Empty(t, convert)
	})

	t.Run("direct CoinGecko keeps vs_currency", func(t *testing.T) {
		vsCurrency, convert := direct.coinsMarketsCurrency("EUR")
		require.Equal(t, "EUR", vsCurrency)
		require.Empty(t, convert)
	})
}

func TestSimplePriceCurrencies(t *testing.T) {
	proxy := NewClientWithParams(Params{IsMarketProxy: true})
	direct := NewClientWithParams(Params{})

	t.Run("the requested currency is asked for as convert_currency", func(t *testing.T) {
		// Which currencies the proxy holds outright is its own business; the
		// client asks for what it wants and lets the proxy pick the source.
		vsCurrencies, convert := proxy.simplePriceCurrencies([]string{"JPY"})
		require.Equal(t, []string{baseCurrency}, vsCurrencies)
		require.Equal(t, "jpy", convert)

		vsCurrencies, convert = proxy.simplePriceCurrencies([]string{"EUR"})
		require.Equal(t, []string{baseCurrency}, vsCurrencies)
		require.Equal(t, "eur", convert)
	})

	t.Run("the base currency needs no conversion", func(t *testing.T) {
		vsCurrencies, convert := proxy.simplePriceCurrencies([]string{"USD"})
		require.Equal(t, []string{baseCurrency}, vsCurrencies)
		require.Empty(t, convert)
	})

	t.Run("the converted currency is never listed in vs_currencies", func(t *testing.T) {
		vsCurrencies, convert := proxy.simplePriceCurrencies([]string{"USD", "JPY"})
		require.Equal(t, "jpy", convert)
		require.NotContains(t, vsCurrencies, convert)
		require.Equal(t, []string{baseCurrency}, vsCurrencies)
	})

	t.Run("further currencies are asked for as vs_currencies", func(t *testing.T) {
		// convert_currency names one currency, so only the first is certain;
		// the rest are served if the proxy happens to hold them.
		vsCurrencies, convert := proxy.simplePriceCurrencies([]string{"JPY", "PLN"})
		require.Equal(t, "jpy", convert)
		require.Equal(t, []string{"pln"}, vsCurrencies)
	})

	t.Run("duplicates are collapsed", func(t *testing.T) {
		vsCurrencies, convert := proxy.simplePriceCurrencies([]string{"JPY", "jpy", "USD"})
		require.Equal(t, "jpy", convert)
		require.Equal(t, []string{baseCurrency}, vsCurrencies)
	})

	t.Run("an empty request still names a currency", func(t *testing.T) {
		// vs_currencies is a required parameter.
		vsCurrencies, convert := proxy.simplePriceCurrencies(nil)
		require.Equal(t, []string{baseCurrency}, vsCurrencies)
		require.Empty(t, convert)
	})

	t.Run("direct CoinGecko keeps vs_currencies", func(t *testing.T) {
		vsCurrencies, convert := direct.simplePriceCurrencies([]string{"JPY"})
		require.Equal(t, []string{"JPY"}, vsCurrencies)
		require.Empty(t, convert)
	})
}

func TestRequestsCarryConvertCurrency(t *testing.T) {
	var query netUrl.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/simple/price" {
			_, _ = w.Write([]byte(`{"bitcoin":{"usd":1,"jpy":150}}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClientWithParams(Params{URL: server.URL, IsMarketProxy: true})

	_, err := client.FetchCoinsMarkets(context.Background(), []string{"bitcoin"}, "JPY")
	require.NoError(t, err)
	require.Equal(t, baseCurrency, query.Get("vs_currency"))
	require.Equal(t, "jpy", query.Get(convertCurrencyParam))

	prices, err := client.FetchSimplePrice(context.Background(), []string{"bitcoin"}, []string{"JPY"})
	require.NoError(t, err)
	require.Equal(t, baseCurrency, query.Get("vs_currencies"))
	require.Equal(t, "jpy", query.Get(convertCurrencyParam))
	require.Equal(t, 150.0, prices["bitcoin"]["jpy"])
}
