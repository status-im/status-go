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
	Symbol              string `json:"symbol"`
	DisplayDecimals     uint   `json:"displayDecimals"`
	StripTrailingZeroes bool   `json:"stripTrailingZeroes"`
}

type FormatPerSymbol = map[string]Format

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
		Symbol:              symbol,
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

func (cm *Currency) calculateTokenCurrencyFormat(symbol string, price float64) (*Format, error) {
	currencyFormat := &Format{
		Symbol:              symbol,
		DisplayDecimals:     calculateTokenDisplayDecimals(price),
		StripTrailingZeroes: true,
	}
	return currencyFormat, nil
}

func GetFiatCurrencyFormats(symbols []string) (FormatPerSymbol, error) {
	formats := make(FormatPerSymbol)

	for _, symbol := range symbols {
		format, err := calculateFiatCurrencyFormat(symbol)

		if err != nil {
			return nil, err
		}

		formats[symbol] = *format
	}

	return formats, nil
}

func (cm *Currency) FetchTokenCurrencyFormats(tokens []*tokentypes.Token) (FormatPerSymbol, error) {
	formats := make(FormatPerSymbol)

	peggedTokens := make(map[string]*tokentypes.Token, 0)
	nonPeggedTokens := make(map[string]*tokentypes.Token, 0)

	for _, token := range tokens {
		if token.PegSymbol != "" {
			peggedTokens[token.Symbol] = token
		} else {
			nonPeggedTokens[token.Symbol] = token
		}
	}

	for _, token := range peggedTokens {
		var currencyFormat, err = calculateFiatCurrencyFormat(token.PegSymbol)
		if err != nil {
			cm.logger.Error("Failed to calculate fiat currency format for pegged token", zap.Error(err))
			continue
		}
		currencyFormat.Symbol = token.Symbol
		formats[token.Symbol] = *currencyFormat
	}

	if len(nonPeggedTokens) > 0 {
		symbols := make([]string, 0, len(nonPeggedTokens))
		for symbol := range nonPeggedTokens {
			symbols = append(symbols, symbol)
		}

		// Get latest cached price, fetch only if not available
		prices, err := cm.marketManager.GetOrFetchPrices(symbols, []string{decimalsCalculationCurrency}, math.MaxInt64)
		if err != nil {
			return nil, err
		}

		for _, symbol := range symbols {
			priceData, ok := prices[symbol][decimalsCalculationCurrency]

			if !ok {
				cm.logger.Error("Could not get price for: " + symbol)
				continue
			}

			format, err := cm.calculateTokenCurrencyFormat(symbol, priceData.Price)
			if err != nil {
				return nil, err
			}

			formats[symbol] = *format
		}
	}

	return formats, nil
}
