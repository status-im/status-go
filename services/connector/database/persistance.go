package persistence

import (
	"database/sql"
)

const upsertDAppQuery = "INSERT INTO connector_dapps (url, name, icon_url, shared_account) VALUES (?, ?, ?, ?) ON CONFLICT(url) DO UPDATE SET name = excluded.name, icon_url = excluded.icon_url, shared_account = excluded.shared_account"
const selectDAppByUrlQuery = "SELECT name, icon_url, shared_account FROM connector_dapps WHERE url = ?"
const deleteDAppQuery = "DELETE FROM connector_dapps WHERE url = ?"

type DApp struct {
	URL           string `json:"url"`
	Name          string `json:"name"`
	IconURL       string `json:"iconUrl"`
	SharedAccount string `json:"sharedAccount"`
}

func UpsertDApp(db *sql.DB, dApp *DApp) error {
	_, err := db.Exec(upsertDAppQuery, dApp.URL, dApp.Name, dApp.IconURL, dApp.SharedAccount)
	return err
}

func SelectDAppByUrl(db *sql.DB, url string) (*DApp, error) {
	dApp := &DApp{
		URL: url,
	}
	err := db.QueryRow(selectDAppByUrlQuery, url).Scan(&dApp.Name, &dApp.IconURL, &dApp.SharedAccount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return dApp, err
}

func DeleteDApp(db *sql.DB, url string) error {
	_, err := db.Exec(deleteDAppQuery, url)
	return err
}
