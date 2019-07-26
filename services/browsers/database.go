package browsers

import (
	"database/sql"

	"github.com/status-im/status-go/services/browsers/migrations"
	"github.com/status-im/status-go/sqlite"
)

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path, password string) (*Database, error) {
	db, err := sqlite.OpenDB(path, password)
	if err != nil {
		return nil, err
	}
	err = migrations.Migrate(db)
	if err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
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
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)
	tx, err = db.db.Begin()
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
	insert, err = tx.Prepare("INSERT OR REPLACE INTO browsers(id, name, timestamp, dapp, historyIndex) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		return
	}
	_, err = insert.Exec(browser.ID, browser.Name, browser.Timestamp, browser.Dapp, browser.HistoryIndex)
	insert.Close()
	if err != nil {
		return
	}
	if len(browser.History) == 0 {
		return
	}
	insert, err = tx.Prepare("INSERT INTO browsers_history(browser_id, history) VALUES(?, ?)")
	if err != nil {
		return
	}
	defer insert.Close()
	for _, history := range browser.History {
		_, err = insert.Exec(browser.ID, history)
		if err != nil {
			return
		}
	}
	return
}

func (db *Database) GetBrowsers() (rst []*Browser, err error) {
	var (
		tx   *sql.Tx
		rows *sql.Rows
	)
	tx, err = db.db.Begin()
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
	rows, err = tx.Query("SELECT id, name, timestamp, dapp, historyIndex FROM browsers ORDER BY timestamp DESC")
	if err != nil {
		return
	}
	browsers := map[string]*Browser{}
	for rows.Next() {
		browser := Browser{}
		err = rows.Scan(&browser.ID, &browser.Name, &browser.Timestamp, &browser.Dapp, &browser.HistoryIndex)
		if err != nil {
			return nil, err
		}
		browsers[browser.ID] = &browser
		rst = append(rst, &browser)
	}
	rows, err = tx.Query("SELECT browser_id, history from browsers_history")
	if err != nil {
		return
	}
	var (
		id      string
		history string
	)
	for rows.Next() {
		err = rows.Scan(&id, &history)
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
