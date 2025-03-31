package currency

import (
	"context"
	"database/sql"
	"time"

	"github.com/ethereum/go-ethereum/event"
	gocommon "github.com/status-im/status-go/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
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
}

func NewService(db *sql.DB, walletFeed *event.Feed, tokenManager *token.Manager, marketManager *market.Manager) *Service {
	return &Service{
		currency:     NewCurrency(marketManager),
		db:           NewCurrencyDB(db),
		tokenManager: tokenManager,
		walletFeed:   walletFeed,
	}
}

func (s *Service) Start(ctx context.Context) {
	// Update all fiat currency formats in cache
	fiatFormats, err := s.getAllFiatCurrencyFormats()

	if err == nil {
		_ = s.db.UpdateCachedFormats(fiatFormats)
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

func (s *Service) GetCachedCurrencyFormats() (Formats, error) {
	return s.db.GetCachedFormats()
}

func (s *Service) FetchAllCurrencyFormats() (Formats, error) {
	// Only token prices can change, so we fetch those
	tokenFormats, err := s.fetchAllTokenCurrencyFormats()

	if err != nil {
		return nil, err
	}

	err = s.db.UpdateCachedFormats(tokenFormats)

	if err != nil {
		return nil, err
	}

	return s.GetCachedCurrencyFormats()
}

func (s *Service) getAllFiatCurrencyFormats() (Formats, error) {
	return GetFiatCurrencyFormats(GetAllFiatCurrencySymbols())
}

func (s *Service) fetchAllTokenCurrencyFormats() (Formats, error) {
	tokens, err := s.tokenManager.GetAllTokens()
	if err != nil {
		return nil, err
	}

	// Doesn't need to be unique list, cause provider handles that
	tokenKeys := []string{
		tokenTypes.GetEthTokenKeyForChain(walletCommon.EthereumMainnet), // Add ETH
	}
	for _, t := range tokens {
		tokenKeys = append(tokenKeys, t.TokenKey())
	}

	tokenFormats, err := s.currency.FetchTokenCurrencyFormats(tokenKeys)
	if err != nil {
		return nil, err
	}
	gweiSymbol := "Gwei"
	tokenFormats[gweiSymbol] = Format{
		TokenKey:            gweiSymbol,
		DisplayDecimals:     9,
		StripTrailingZeroes: true,
	}
	return tokenFormats, err
}
