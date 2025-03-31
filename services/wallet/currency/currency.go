package currency

import (
	"errors"
	"math"
	"strings"

	iso4217 "github.com/ladydascalie/currency"

	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/token"
)

const decimalsCalculationCurrency = "USD"

const lowerTokenResolutionInUsd = 0.1
const higherTokenResolutionInUsd = 0.01

type Format struct {
	TokenKey            string `json:"tokenKey"`
	DisplayDecimals     uint   `json:"displayDecimals"`
	StripTrailingZeroes bool   `json:"stripTrailingZeroes"`
}

type Formats = map[string]Format // [tokenKey]Format

type Currency struct {
	marketManager *market.Manager
}

func NewCurrency(marketManager *market.Manager) *Currency {
	return &Currency{
		marketManager: marketManager,
	}
}

func IsCurrencyFiat(symbol string) bool {
	return iso4217.Valid(strings.ToUpper(symbol))
}

func GetAllFiatCurrencySymbols() []string {
	return iso4217.ValidCodes
}

func calculateFiatDisplayDecimals(symbol string) (uint, error) {
	currency, err := iso4217.Get(strings.ToUpper(symbol))

	if err != nil {
		return 0, err
	}

	return uint(currency.MinorUnits()), nil
}

func calculateFiatCurrencyFormat(symbol string) (*Format, error) {
	displayDecimals, err := calculateFiatDisplayDecimals(symbol)

	if err != nil {
		return nil, err
	}

	format := &Format{
		TokenKey:            symbol,
		DisplayDecimals:     displayDecimals,
		StripTrailingZeroes: false,
	}

	return format, nil
}

func calculateTokenDisplayDecimals(price float64) uint {
	var displayDecimals float64 = 0.0

	if price > 0 {
		lowerDecimalsBound := math.Max(0.0, math.Log10(price)-math.Log10(lowerTokenResolutionInUsd))
		upperDecimalsBound := math.Max(0.0, math.Log10(price)-math.Log10(higherTokenResolutionInUsd))

		// Use as few decimals as needed to ensure lower precision
		displayDecimals = math.Ceil(lowerDecimalsBound)
		if displayDecimals+1.0 <= upperDecimalsBound {
			// If allowed by upper bound, ensure resolution changes as soon as currency hits multiple of 10
			displayDecimals += 1.0
		}
	}

	return uint(displayDecimals)
}

func (cm *Currency) calculateTokenCurrencyFormat(tokenKey string, price float64) (*Format, error) {
	pegSymbol := token.GetTokenPegSymbol(tokenKey)

	if pegSymbol != "" {
		var currencyFormat, err = calculateFiatCurrencyFormat(pegSymbol)
		if err != nil {
			return nil, err
		}
		currencyFormat.TokenKey = tokenKey
		return currencyFormat, nil
	}

	currencyFormat := &Format{
		TokenKey:            tokenKey,
		DisplayDecimals:     calculateTokenDisplayDecimals(price),
		StripTrailingZeroes: true,
	}
	return currencyFormat, nil
}

func GetFiatCurrencyFormats(symbols []string) (Formats, error) {
	formats := make(Formats)

	for _, symbol := range symbols {
		format, err := calculateFiatCurrencyFormat(symbol)

		if err != nil {
			return nil, err
		}

		formats[symbol] = *format
	}

	return formats, nil
}

func (cm *Currency) FetchTokenCurrencyFormats(tokenKeys []string) (Formats, error) {
	formats := make(Formats)

	// Get latest cached price, fetch only if not available
	prices, err := cm.marketManager.GetOrFetchPrices(tokenKeys, []string{decimalsCalculationCurrency}, math.MaxInt64)
	if err != nil {
		return nil, err
	}

	for _, tokenKey := range tokenKeys {
		priceData, ok := prices[tokenKey][decimalsCalculationCurrency]

		if !ok {
			return nil, errors.New("Could not get price for: " + tokenKey)
		}

		format, err := cm.calculateTokenCurrencyFormat(tokenKey, priceData.Price)

		if err != nil {
			return nil, err
		}

		formats[tokenKey] = *format
	}

	return formats, nil
}
