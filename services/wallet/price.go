package wallet

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type PricesPerTokenAndCurrency = map[string]map[string]float64

type PriceManager struct {
	db            *sql.DB
	cryptoCompare *thirdparty.CryptoCompare
}

func NewPriceManager(db *sql.DB, cryptoCompare *thirdparty.CryptoCompare) *PriceManager {
	return &PriceManager{db: db, cryptoCompare: cryptoCompare}
}

func (pm *PriceManager) FetchPrices(symbols []string, currencies []string) (PricesPerTokenAndCurrency, error) {
	result, err := pm.cryptoCompare.FetchPrices(symbols, currencies)
	if err != nil {
		return nil, err
	}
	if err = pm.updatePriceCache(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (pm *PriceManager) updatePriceCache(prices PricesPerTokenAndCurrency) error {
	tx, err := pm.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	insert, err := tx.Prepare(`INSERT OR REPLACE INTO price_cache 
		(token,	currency, price)
		VALUES
		(?, ?, ?)`)
	if err != nil {
		return err
	}

	for token, pricesPerCurrency := range prices {
		for currency, price := range pricesPerCurrency {
			_, err = insert.Exec(token, currency, fmt.Sprintf("%f", price))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getCachedPricesFromDBRows(rows *sql.Rows) (PricesPerTokenAndCurrency, error) {
	prices := make(PricesPerTokenAndCurrency)

	for rows.Next() {
		var token string
		var currency string
		var price float64
		if err := rows.Scan(&token, &currency, &price); err != nil {
			return nil, err
		}

		_, present := prices[token]
		if !present {
			prices[token] = map[string]float64{}
		}

		prices[token][currency] = price
	}

	return prices, nil
}

func (pm *PriceManager) GetCachedPrices() (PricesPerTokenAndCurrency, error) {
	rows, err := pm.db.Query("SELECT token, currency, price FROM price_cache")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getCachedPricesFromDBRows(rows)
}
