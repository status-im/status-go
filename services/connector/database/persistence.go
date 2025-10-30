package persistence

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/status-im/status-go/crypto/types"
)

func NormalizeURL(url string) string {
	return strings.TrimRight(url, "/")
}

const upsertDAppQuery = "INSERT INTO connector_dapps (url, name, icon_url, client_id, shared_account, chain_id) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(url, client_id) DO UPDATE SET name = excluded.name, icon_url = excluded.icon_url, shared_account = excluded.shared_account, chain_id = excluded.chain_id"
const selectDAppQuery = "SELECT name, icon_url, shared_account, chain_id FROM connector_dapps WHERE url = ? AND client_id = ?"
const selectDAppsQuery = "SELECT url, name, icon_url, client_id, shared_account, chain_id FROM connector_dapps"
const deleteDAppQuery = "DELETE FROM connector_dapps WHERE url = ? AND client_id = ?"

// Permissions queries
const insertPermissionQuery = "INSERT INTO connector_permissions (url, client_id, parent_capability, caveats, created_at) VALUES (?, ?, ?, ?, ?)"
const selectPermissionsQuery = "SELECT parent_capability, caveats FROM connector_permissions WHERE url = ? AND client_id = ?"
const selectPermissionExistsQuery = "SELECT COUNT(*) FROM connector_permissions WHERE url = ? AND client_id = ? AND parent_capability = ?"
const deletePermissionsQuery = "DELETE FROM connector_permissions WHERE url = ? AND client_id = ?"

type DApp struct {
	URL           string        `json:"url"`
	Name          string        `json:"name"`
	IconURL       string        `json:"iconUrl"`
	ClientID      string        `json:"clientId"`
	SharedAccount types.Address `json:"sharedAccount"`
	ChainID       uint64        `json:"chainId"`
}

// EIP-2255
// TODO: Add caveat validation and enforcement mechanism
// TODO: Implement standard caveat types (requiredMethods, expiry, etc.)
type Caveat struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type Permission struct {
	Invoker          string   `json:"invoker"`
	ParentCapability string   `json:"parentCapability"`
	Caveats          []Caveat `json:"caveats"`
}

func UpsertDApp(db *sql.DB, dApp *DApp) error {
	normalizedURL := NormalizeURL(dApp.URL)
	_, err := db.Exec(upsertDAppQuery, normalizedURL, dApp.Name, dApp.IconURL, dApp.ClientID, dApp.SharedAccount, dApp.ChainID)
	return err
}

func SelectDApp(db *sql.DB, url string, clientID string) (*DApp, error) {
	// clientID can be empty for backward compatibility with browser extension
	normalizedURL := NormalizeURL(url)
	dApp := &DApp{
		URL:      normalizedURL,
		ClientID: clientID,
	}
	err := db.QueryRow(selectDAppQuery, normalizedURL, clientID).Scan(&dApp.Name, &dApp.IconURL, &dApp.SharedAccount, &dApp.ChainID)
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
	normalizedURL := NormalizeURL(url)
	_, err := db.Exec(deleteDAppQuery, normalizedURL, clientID)
	return err
}

func InsertPermission(db *sql.DB, url string, clientID string, parentCapability string, caveats []Caveat, createdAt int64) error {
	normalizedURL := NormalizeURL(url)

	var count int
	err := db.QueryRow(selectPermissionExistsQuery, normalizedURL, clientID, parentCapability).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	caveatsJSON, err := json.Marshal(caveats)
	if err != nil {
		return err
	}

	_, err = db.Exec(insertPermissionQuery, normalizedURL, clientID, parentCapability, string(caveatsJSON), createdAt)
	return err
}

func SelectPermissions(db *sql.DB, url string, clientID string) ([]Permission, error) {
	normalizedURL := NormalizeURL(url)

	rows, err := db.Query(selectPermissionsQuery, normalizedURL, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var parentCapability string
		var caveatsJSON string

		err = rows.Scan(&parentCapability, &caveatsJSON)
		if err != nil {
			return nil, err
		}

		var caveats []Caveat
		if err := json.Unmarshal([]byte(caveatsJSON), &caveats); err != nil {
			return nil, err
		}

		permission := Permission{
			Invoker:          normalizedURL,
			ParentCapability: parentCapability,
			Caveats:          caveats,
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

func DeletePermissions(db *sql.DB, url string, clientID string) error {
	normalizedURL := NormalizeURL(url)
	_, err := db.Exec(deletePermissionsQuery, normalizedURL, clientID)
	return err
}
