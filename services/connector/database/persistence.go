package persistence

import (
	"database/sql"

	"github.com/status-im/status-go/crypto/types"
)

const upsertDAppQuery = "INSERT INTO connector_dapps (url, name, icon_url, client_id, shared_account, chain_id) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(url, client_id) DO UPDATE SET name = excluded.name, icon_url = excluded.icon_url, shared_account = excluded.shared_account, chain_id = excluded.chain_id"
const selectDAppByUrlAndClientIDQuery = "SELECT name, icon_url, shared_account, chain_id FROM connector_dapps WHERE url = ? AND client_id = ?"
const selectDAppsQuery = "SELECT url, name, icon_url, client_id, shared_account, chain_id FROM connector_dapps"
const deleteDAppQuery = "DELETE FROM connector_dapps WHERE url = ? AND client_id = ?"

type DApp struct {
	URL           string        `json:"url"`
	Name          string        `json:"name"`
	IconURL       string        `json:"iconUrl"`
	ClientID      string        `json:"clientId"`
	SharedAccount types.Address `json:"sharedAccount"`
	ChainID       uint64        `json:"chainId"`
}

func UpsertDApp(db *sql.DB, dApp *DApp) error {
	_, err := db.Exec(upsertDAppQuery, dApp.URL, dApp.Name, dApp.IconURL, dApp.ClientID, dApp.SharedAccount, dApp.ChainID)
	return err
}

func SelectDAppByUrlAndClientID(db *sql.DB, url string, clientID string) (*DApp, error) {
	// clientID can be empty for backward compatibility with browser extension
	dApp := &DApp{
		URL:      url,
		ClientID: clientID,
	}
	err := db.QueryRow(selectDAppByUrlAndClientIDQuery, url, clientID).Scan(&dApp.Name, &dApp.IconURL, &dApp.SharedAccount, &dApp.ChainID)
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
		err = rows.Scan(&dApp.URL, &dApp.Name, &dApp.IconURL, &dApp.ClientID, &dApp.SharedAccount, &dApp.ChainID)
		if err != nil {
			return nil, err
		}
		dApps = append(dApps, dApp)
	}
	return dApps, nil
}

func DeleteDApp(db *sql.DB, url string, clientID string) error {
	_, err := db.Exec(deleteDAppQuery, url, clientID)
	return err
}
