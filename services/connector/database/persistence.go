package persistence

import (
	"database/sql"

	"github.com/status-im/status-go/eth-node/types"
)

const upsertDAppQuery = "INSERT INTO connector_dapps (url, name, icon_url, shared_account, chain_id) VALUES (?, ?, ?, ?, ?) ON CONFLICT(url) DO UPDATE SET name = excluded.name, icon_url = excluded.icon_url, shared_account = excluded.shared_account, chain_id = excluded.chain_id"
const selectDAppByUrlQuery = "SELECT name, icon_url, shared_account, chain_id FROM connector_dapps WHERE url = ?"
const selectDAppsQuery = "SELECT url, name, icon_url, shared_account, chain_id FROM connector_dapps"
const deleteDAppQuery = "DELETE FROM connector_dapps WHERE url = ?"

type DApp struct {
	URL           string        `json:"url"`
	Name          string        `json:"name"`
	IconURL       string        `json:"iconUrl"`
	SharedAccount types.Address `json:"sharedAccount"`
	ChainID       uint64        `json:"chainId"`
}

func UpsertDApp(db *sql.DB, dApp *DApp) error {
	_, err := db.Exec(upsertDAppQuery, dApp.URL, dApp.Name, dApp.IconURL, dApp.SharedAccount, dApp.ChainID)
	return err
}

func SelectDAppByUrl(db *sql.DB, url string) (*DApp, error) {
	dApp := &DApp{
		URL: url,
	}
	err := db.QueryRow(selectDAppByUrlQuery, url).Scan(&dApp.Name, &dApp.IconURL, &dApp.SharedAccount, &dApp.ChainID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return dApp, err
}

func SelectAllDApps(db *sql.DB) ([]DApp, error) {
	rows, err := db.Query(selectDAppsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dApps []DApp
	for rows.Next() {
		dApp := DApp{}
		err = rows.Scan(&dApp.URL, &dApp.Name, &dApp.IconURL, &dApp.SharedAccount, &dApp.ChainID)
		if err != nil {
			return nil, err
		}
		dApps = append(dApps, dApp)
	}
	return dApps, nil
}

func DeleteDApp(db *sql.DB, url string) error {
	_, err := db.Exec(deleteDAppQuery, url)
	return err
}
