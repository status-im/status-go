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
	ID                  string `json:"id"` // represents a grouped token key or currency code
	DisplayDecimals     uint   `json:"displayDecimals"`
	StripTrailingZeroes bool   `json:"stripTrailingZeroes"`
}

type Formats = map[string]Format // [id]Format

type Currency struct {
	marketManager *market.Manager
}

func NewCurrency(marketManager *market.Manager) *Currency {
	return &Currency{
		marketManager: marketManager,
	}
}

func IsCurrencyFiat(currencyCode string) bool {
	return iso4217.Valid(strings.ToUpper(currencyCode))
}

func GetAllFiatCurrencySymbols() []string {
	return iso4217.ValidCodes
}

func calculateFiatDisplayDecimals(id string) (uint, error) {
	currency, err := iso4217.Get(strings.ToUpper(id))

	if err != nil {
		return 0, err
	}

	return uint(currency.MinorUnits()), nil
}

func calculateFiatCurrencyFormat(id string) (*Format, error) {
	displayDecimals, err := calculateFiatDisplayDecimals(id)

	if err != nil {
		return nil, err
	}

	format := &Format{
		ID:                  id,
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

func (cm *Currency) calculateTokenCurrencyFormat(id string, price float64) (*Format, error) {
	pegSymbol := token.GetTokenPegSymbol(id)

	if pegSymbol != "" {
		var currencyFormat, err = calculateFiatCurrencyFormat(pegSymbol)
		if err != nil {
			return nil, err
		}
		currencyFormat.ID = id
		return currencyFormat, nil
	}

	currencyFormat := &Format{
		ID:                  id,
		DisplayDecimals:     calculateTokenDisplayDecimals(price),
		StripTrailingZeroes: true,
	}
	return currencyFormat, nil
}

func GetFiatCurrencyFormats(ids []string) (Formats, error) {
	formats := make(Formats)

	for _, id := range ids {
		format, err := calculateFiatCurrencyFormat(id)

		if err != nil {
			return nil, err
		}

		formats[id] = *format
	}

	return formats, nil
}

func (cm *Currency) FetchTokenCurrencyFormats(groupedTokensKeys []string) (Formats, error) {
	formats := make(Formats)

	// Get latest cached price, fetch only if not available
	prices, err := cm.marketManager.GetOrFetchPrices(groupedTokensKeys, []string{decimalsCalculationCurrency}, math.MaxInt64)
	if err != nil {
		return nil, err
	}

	for _, gtKey := range groupedTokensKeys {
		priceData, ok := prices[gtKey][decimalsCalculationCurrency]
		if !ok {
			return nil, errors.New("Could not get price for: " + gtKey)
		}

		format, err := cm.calculateTokenCurrencyFormat(gtKey, priceData.Price)
		if err != nil {
			return nil, err
		}

		formats[gtKey] = *format
	}

	return formats, nil
}
