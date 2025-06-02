package currency

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/token"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	EventCurrencyTickUpdateFormat walletevent.EventType = "wallet-currency-tick-update-format"

	currencyFormatUpdateInterval = 1 * time.Hour
)

type Service struct {
	currency *Currency
	db       *DB

	tokenManager *token.Manager
	walletFeed   *event.Feed

	logger *zap.Logger
}

func NewService(db *sql.DB, walletFeed *event.Feed, tokenManager *token.Manager, marketManager *market.Manager) *Service {
	logger := logutils.ZapLogger().Named("Currency")
	return &Service{
		currency:     NewCurrency(marketManager, logger),
		db:           NewCurrencyDB(db),
		tokenManager: tokenManager,
		walletFeed:   walletFeed,
		logger:       logger,
	}
}

func (s *Service) Start(ctx context.Context) {
	// Update all fiat currency formats in cache
	fiatFormats, err := s.getAllFiatCurrencyFormats()
	if err == nil {
		_ = s.db.UpdateCachedFormats(fiatFormats)
	}

	fixedTokenFormats, err := s.getAllFixedTokenCurrencyFormats()
	if err == nil {
		_ = s.db.UpdateCachedFormats(fixedTokenFormats)
	}

	go func() {
		defer gocommon.LogOnPanic()
		ticker := time.NewTicker(currencyFormatUpdateInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.walletFeed.Send(walletevent.Event{
					Type: EventCurrencyTickUpdateFormat,
				})
			}
		}
	}()
}

func (s *Service) GetCachedCurrencyFormats() (FormatPerSymbol, error) {
	return s.db.GetCachedFormats()
}

func (s *Service) FetchAllCurrencyFormats() (FormatPerSymbol, error) {
	// Only token prices can change, so we fetch those
	tokenFormats, err := s.fetchAllTokenCurrencyFormats()

	if err != nil {
		s.logger.Error("Failed to fetch all token currency formats", zap.Error(err))
		return nil, err
	}

	err = s.db.UpdateCachedFormats(tokenFormats)

	if err != nil {
		s.logger.Error("Failed to update cached currency formats", zap.Error(err))
		return nil, err
	}

	return s.GetCachedCurrencyFormats()
}

func (s *Service) getAllFiatCurrencyFormats() (FormatPerSymbol, error) {
	return GetFiatCurrencyFormats(GetAllFiatCurrencySymbols())
}

func (s *Service) getAllFixedTokenCurrencyFormats() (FormatPerSymbol, error) {
	tokens, err := s.tokenManager.GetAllTokens()
	if err != nil {
		return nil, err
	}

	peggedTokens := make([]*tokentypes.Token, 0, len(tokens))
	for _, token := range tokens {
		if token.PegSymbol != "" {
			peggedTokens = append(peggedTokens, token)
		}
	}

	tokenFormats, err := s.currency.FetchTokenCurrencyFormats(peggedTokens)
	if err != nil {
		return nil, err
	}

	const gweiSymbol = "Gwei"
	tokenFormats[gweiSymbol] = Format{
		Symbol:              gweiSymbol,
		DisplayDecimals:     9,
		StripTrailingZeroes: true,
	}

	return tokenFormats, nil
}

func (s *Service) fetchAllTokenCurrencyFormats() (FormatPerSymbol, error) {
	tokens, err := s.tokenManager.GetAllTokens()
	if err != nil {
		return nil, err
	}

	tokenFormats, err := s.currency.FetchTokenCurrencyFormats(tokens)
	if err != nil {
		return nil, err
	}

	return tokenFormats, err
}
