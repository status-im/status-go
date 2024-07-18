package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrencyList(t *testing.T) {
	currencies := GetCurrencies()
	require.NotNil(t, currencies)
	require.Equal(t, 83, len(currencies))

	popularCurrencies := make([]*Currency, 0)
	cryptoCurrencies := make([]*Currency, 0)
	otherFiatCurrencies := make([]*Currency, 0)
	for _, c := range currencies {
		if c.IsPopular {
			popularCurrencies = append(popularCurrencies, c)
		}
		if c.IsToken {
			cryptoCurrencies = append(cryptoCurrencies, c)
		}
		if !c.IsToken && !c.IsPopular {
			otherFiatCurrencies = append(otherFiatCurrencies, c)
		}
	}

	require.Equal(t, 5, len(popularCurrencies))
	require.Equal(t, 4, len(cryptoCurrencies))
	require.Equal(t, 74, len(otherFiatCurrencies))
}
