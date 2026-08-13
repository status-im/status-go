package currency

import (
	"math"
	"strings"

	"go.uber.org/zap"

	iso4217 "github.com/ladydascalie/currency"

	"github.com/status-im/status-go/services/wallet/market"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

const decimalsCalculationCurrency = "USD"

const lowerTokenResolutionInUsd = 0.1
const higherTokenResolutionInUsd = 0.01

type Format struct {
	Key                 string `json:"key"` // fiat currency or token key
	Symbol              string `json:"symbol"`
	DisplayDecimals     uint   `json:"displayDecimals"`
	StripTrailingZeroes bool   `json:"stripTrailingZeroes"`
}

type FormatPerKey = map[string]Format

type Currency struct {
	marketManager *market.Manager
	logger        *zap.Logger
}

func NewCurrency(marketManager *market.Manager, logger *zap.Logger) *Currency {
	return &Currency{
		marketManager: marketManager,
		logger:        logger,
	}
}

func GetAllFiatCurrencySymbols() []string {
	return iso4217.ValidCodes
}

func calculateFiatDisplayDecimals(key string) (uint, error) {
	currency, err := iso4217.Get(strings.ToUpper(key))

	if err != nil {
		return 0, err
	}

	return uint(currency.MinorUnits()), nil
}

func calculateFiatCurrencyFormat(key string) (*Format, error) {
	displayDecimals, err := calculateFiatDisplayDecimals(key)

	if err != nil {
		return nil, err
	}

	format := &Format{
		Key:                 key,
		Symbol:              key,
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

func (cm *Currency) calculateTokenCurrencyFormat(key string, symbol string, price float64) (*Format, error) {
	currencyFormat := &Format{
		Key:                 key,
		Symbol:              symbol,
		DisplayDecimals:     calculateTokenDisplayDecimals(price),
		StripTrailingZeroes: true,
	}
	return currencyFormat, nil
}

func GetFiatCurrencyFormats(keys []string) (FormatPerKey, error) {
	formats := make(FormatPerKey)

	for _, key := range keys {
		format, err := calculateFiatCurrencyFormat(key)

		if err != nil {
			return nil, err
		}

		formats[key] = *format
	}

	return formats, nil
}

func (cm *Currency) FetchTokenCurrencyFormats(tokenKeysSymbolMap map[string]string) (FormatPerKey, error) {
	formats := make(FormatPerKey)

	peggedTokens := make([]string, 0)
	nonPeggedTokens := make([]string, 0)

	for tokenKey := range tokenKeysSymbolMap {
		pegTokenKey := tokentypes.GetTokenPegByTokenKey(tokenKey)
		if pegTokenKey != "" {
			peggedTokens = append(peggedTokens, tokenKey)
		} else {
			nonPeggedTokens = append(nonPeggedTokens, tokenKey)
		}
	}

	for _, tokenKey := range peggedTokens {
		pegTokenKey := tokentypes.GetTokenPegByTokenKey(tokenKey)
		var currencyFormat, err = calculateFiatCurrencyFormat(pegTokenKey)
		if err != nil {
			cm.logger.Error("Failed to calculate fiat currency format for pegged token", zap.Error(err))
			continue
		}
		currencyFormat.Key = tokenKey
		formats[currencyFormat.Key] = *currencyFormat
	}

	if len(nonPeggedTokens) > 0 {
		// Get latest cached price, fetch only if not available
		prices, err := cm.marketManager.GetOrFetchPrices(nonPeggedTokens, []string{decimalsCalculationCurrency}, math.MaxInt64)
		if err != nil {
			return nil, err
		}

		for _, tokenKey := range nonPeggedTokens {
			priceData, ok := prices[tokenKey][decimalsCalculationCurrency]

			if !ok {
				cm.logger.Error("Could not get price for: " + tokenKey)
				continue
			}

			format, err := cm.calculateTokenCurrencyFormat(tokenKey, tokenKeysSymbolMap[tokenKey], priceData.Price)
			if err != nil {
				return nil, err
			}

			formats[tokenKey] = *format
		}
	}

	return formats, nil
}
