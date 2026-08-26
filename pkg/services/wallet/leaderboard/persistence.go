package leaderboard

import (
	"database/sql"
	"fmt"
	"sync"

	sq "github.com/Masterminds/squirrel"
)

type MarketDataPersistenceInterface interface {
	// UpsertCryptocurrencies stores the rows together with the currency their
	// values are expressed in.
	UpsertCryptocurrencies(data []Cryptocurrency, currency string) error
	// GetCryptocurrencies returns only the rows stored for the given currency;
	// rows left over from another currency are a cache miss, not data.
	GetCryptocurrencies(currency string) ([]Cryptocurrency, error)
	DeleteCryptocurrencies(ids []string) error
	// DeleteCryptocurrenciesNotIn drops every row that is not expressed in the
	// given currency, so a currency switch cannot leave stale values behind.
	DeleteCryptocurrenciesNotIn(currency string) error
}

type Persistence struct {
	db        *sql.DB
	dataMutex sync.RWMutex
}

func NewPersistance(db *sql.DB) *Persistence {
	return &Persistence{
		db: db,
	}
}

func (p *Persistence) UpsertCryptocurrencies(cryptos []Cryptocurrency, currency string) error {
	p.dataMutex.Lock()
	defer p.dataMutex.Unlock()
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	query := sq.Insert("market_data").
		Columns("id", "symbol", "current_price", "market_cap", "total_volume", "price_change_percentage_24h", "currency").
		Suffix(`ON CONFLICT (id) DO UPDATE SET
	        symbol = excluded.symbol,
	        current_price = excluded.current_price,
	        market_cap = excluded.market_cap,
	        total_volume = excluded.total_volume,
	        price_change_percentage_24h = excluded.price_change_percentage_24h,
	        currency = excluded.currency`).
		RunWith(tx)

	for _, crypto := range cryptos {
		query = query.Values(
			crypto.ID,
			crypto.Symbol,
			crypto.CurrentPrice,
			crypto.MarketCap,
			crypto.TotalVolume,
			crypto.PriceChangePercentage24h,
			normalizeCurrency(currency),
		)
	}

	_, err = query.Exec()

	if err != nil {
		return fmt.Errorf("failed to upsert records: %w", err)
	}

	return nil
}

func (p *Persistence) DeleteCryptocurrencies(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	p.dataMutex.Lock()
	defer p.dataMutex.Unlock()
	queryDelete := sq.Delete("market_data").
		Where(sq.Eq{"id": ids}).
		RunWith(p.db)
	_, err := queryDelete.Exec()
	if err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}
	return nil
}

func (p *Persistence) DeleteCryptocurrenciesNotIn(currency string) error {
	p.dataMutex.Lock()
	defer p.dataMutex.Unlock()
	queryDelete := sq.Delete("market_data").
		Where(sq.NotEq{"currency": normalizeCurrency(currency)}).
		RunWith(p.db)
	_, err := queryDelete.Exec()
	if err != nil {
		return fmt.Errorf("failed to delete records of other currencies: %w", err)
	}
	return nil
}

func (p *Persistence) GetCryptocurrencies(currency string) ([]Cryptocurrency, error) {
	p.dataMutex.Lock()
	defer p.dataMutex.Unlock()
	query := sq.Select("id", "symbol", "current_price", "market_cap", "total_volume", "price_change_percentage_24h").
		From("market_data").
		Where(sq.Eq{"currency": normalizeCurrency(currency)})

	rows, err := query.RunWith(p.db).Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	var cryptos []Cryptocurrency
	for rows.Next() {
		var crypto Cryptocurrency
		err := rows.Scan(
			&crypto.ID,
			&crypto.Symbol,
			&crypto.CurrentPrice,
			&crypto.MarketCap,
			&crypto.TotalVolume,
			&crypto.PriceChangePercentage24h,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		cryptos = append(cryptos, crypto)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return cryptos, nil
}
