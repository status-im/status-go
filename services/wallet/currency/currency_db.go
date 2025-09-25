package currency

import (
	"context"
	"database/sql"
)

type DB struct {
	db *sql.DB
}

func NewCurrencyDB(sqlDb *sql.DB) *DB {
	return &DB{
		db: sqlDb,
	}
}

func getCachedFormatsFromDBRows(rows *sql.Rows) (FormatPerKey, error) {
	formats := make(FormatPerKey)

	for rows.Next() {
		var format Format
		if err := rows.Scan(&format.Key, &format.Symbol, &format.DisplayDecimals, &format.StripTrailingZeroes); err != nil {
			return nil, err
		}

		formats[format.Key] = format
	}

	return formats, nil
}

func (cdb *DB) GetCachedFormats() (FormatPerKey, error) {
	rows, err := cdb.db.Query("SELECT key, symbol, display_decimals, strip_trailing_zeroes FROM currency_format_cache")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getCachedFormatsFromDBRows(rows)
}

func (cdb *DB) UpdateCachedFormats(formats FormatPerKey) error {
	tx, err := cdb.db.BeginTx(context.Background(), &sql.TxOptions{})
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

	insert, err := tx.Prepare(`INSERT OR REPLACE INTO currency_format_cache
				(key, symbol, display_decimals, strip_trailing_zeroes)
				VALUES
				(?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insert.Close()

	for _, format := range formats {
		_, err = insert.Exec(format.Key, format.Symbol, format.DisplayDecimals, format.StripTrailingZeroes)
		if err != nil {
			return err
		}
	}
	return nil
}
