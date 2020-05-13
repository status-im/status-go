package browsers

import (
	"database/sql"
)

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

type Browser struct {
	ID           string   `json:"browser-id"`
	Name         string   `json:"name"`
	Timestamp    uint64   `json:"timestamp"`
	Dapp         bool     `json:"dapp?"`
	HistoryIndex int      `json:"history-index"`
	History      []string `json:"history,omitempty"`
}

func (db *Database) InsertBrowser(browser Browser) (err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	bInsert, err := tx.Prepare("INSERT OR REPLACE INTO browsers(id, name, timestamp, dapp, historyIndex) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		return
	}
	_, err = bInsert.Exec(browser.ID, browser.Name, browser.Timestamp, browser.Dapp, browser.HistoryIndex)
	bInsert.Close()
	if err != nil {
		return
	}

	if len(browser.History) == 0 {
		return
	}
	bhInsert, err := tx.Prepare("INSERT INTO browsers_history(browser_id, history) VALUES(?, ?)")
	if err != nil {
		return
	}
	defer bhInsert.Close()
	for _, history := range browser.History {
		_, err = bhInsert.Exec(browser.ID, history)
		if err != nil {
			return
		}
	}

	return
}

func (db *Database) GetBrowsers() (rst []*Browser, err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	// FULL and RIGHT joins are not supported
	bRows, err := tx.Query("SELECT id, name, timestamp, dapp, historyIndex FROM browsers ORDER BY timestamp DESC")
	if err != nil {
		return
	}
	defer bRows.Close()
	browsers := map[string]*Browser{}
	for bRows.Next() {
		browser := Browser{}
		err = bRows.Scan(&browser.ID, &browser.Name, &browser.Timestamp, &browser.Dapp, &browser.HistoryIndex)
		if err != nil {
			return nil, err
		}
		browsers[browser.ID] = &browser
		rst = append(rst, &browser)
	}

	bhRows, err := tx.Query("SELECT browser_id, history from browsers_history")
	if err != nil {
		return
	}
	defer bhRows.Close()
	var (
		id      string
		history string
	)
	for bhRows.Next() {
		err = bhRows.Scan(&id, &history)
		if err != nil {
			return
		}
		browsers[id].History = append(browsers[id].History, history)
	}

	return rst, nil
}

func (db *Database) DeleteBrowser(id string) error {
	_, err := db.db.Exec("DELETE from browsers WHERE id = ?", id)
	return err
}
