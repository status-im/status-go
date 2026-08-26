package coingecko

import "strings"

const (
	// baseCurrency is the currency /simple/price is asked for when the caller
	// wants something else: `vs_currencies` is a required parameter and must
	// not name the currency handed to convert_currency.
	baseCurrency = "usd"

	// convertCurrencyParam asks the market proxy for values in another
	// currency. The proxy decides where they come from - provider values when
	// it has them, a conversion otherwise. It is a proxy extension;
	// api.coingecko.com does not know it.
	convertCurrencyParam = "convert_currency"
)

func normalizeCurrency(currency string) string {
	return strings.ToLower(strings.TrimSpace(currency))
}

// coinsMarketsCurrency decides how to ask for a currency on /coins/markets.
// The proxy normalizes vs_currency to usd for cache consistency, so the only
// parameter that has any effect there is convert_currency.
func (c *Client) coinsMarketsCurrency(currency string) (vsCurrency string, convertCurrency string) {
	normalized := normalizeCurrency(currency)
	if !c.isMarketProxy || normalized == "" || normalized == baseCurrency {
		return currency, ""
	}
	return baseCurrency, normalized
}

// simplePriceCurrencies decides how to ask for currencies on /simple/price.
//
// convert_currency names a single currency, so the first one asked for is
// answered for certain; any others are passed through as vs_currencies and come
// back only for those the proxy already holds.
//
// The converted currency is never repeated in vs_currencies: one currency
// cannot be requested both ways.
func (c *Client) simplePriceCurrencies(currencies []string) (vsCurrencies []string, convertCurrency string) {
	if !c.isMarketProxy {
		return currencies, ""
	}

	seen := make(map[string]struct{}, len(currencies))
	var normalized []string
	for _, currency := range currencies {
		currency = normalizeCurrency(currency)
		if currency == "" {
			continue
		}
		if _, duplicate := seen[currency]; duplicate {
			continue
		}
		seen[currency] = struct{}{}
		normalized = append(normalized, currency)
	}

	for i, currency := range normalized {
		if currency == baseCurrency {
			continue
		}
		convertCurrency = currency
		vsCurrencies = append(append([]string{}, normalized[:i]...), normalized[i+1:]...)
		break
	}

	if convertCurrency == "" {
		// Nothing to convert: the caller asked for the base currency, or for
		// nothing at all - vs_currencies is required, so fill in the base.
		if len(normalized) == 0 {
			return []string{baseCurrency}, ""
		}
		return normalized, ""
	}

	if len(vsCurrencies) == 0 {
		vsCurrencies = []string{baseCurrency}
	}

	return vsCurrencies, convertCurrency
}
