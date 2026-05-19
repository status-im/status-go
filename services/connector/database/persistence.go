package persistence

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/status-im/status-go/internal/crypto/types"
)

// EphemeralClientIDSuffix marks connector sessions that must not persist across runs.
const EphemeralClientIDSuffix = "#ephemeral"

const deleteEphemeralDAppsQuery = "DELETE FROM connector_dapps WHERE client_id LIKE '%' || ?"

// IsEphemeralClientID reports whether clientID uses the ephemeral suffix convention.
func IsEphemeralClientID(id string) bool {
	return strings.HasSuffix(id, EphemeralClientIDSuffix)
}

// DeleteEphemeralDApps removes connector_dapps rows whose clientID ends with
// EphemeralClientIDSuffix (e.g. "status-desktop/dapp-browser#ephemeral").
// Linked connector_permissions rows are removed via FK ON DELETE CASCADE.
func DeleteEphemeralDApps(db *sql.DB) error {
	_, err := db.Exec(deleteEphemeralDAppsQuery, EphemeralClientIDSuffix)
	return err
}

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
const deletePermissionQuery = "DELETE FROM connector_permissions WHERE url = ? AND client_id = ? AND parent_capability = ?"

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

// DeletePermission removes a single EIP-2255 capability row for the dApp.
func DeletePermission(db *sql.DB, url string, clientID string, parentCapability string) error {
	normalizedURL := NormalizeURL(url)
	_, err := db.Exec(deletePermissionQuery, normalizedURL, clientID, parentCapability)
	return err
}

// PermissionExists reports whether a parent_capability row exists for the origin.
func PermissionExists(db *sql.DB, url string, clientID string, parentCapability string) (bool, error) {
	normalizedURL := NormalizeURL(url)
	var count int
	err := db.QueryRow(selectPermissionExistsQuery, normalizedURL, clientID, parentCapability).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

const (
	// WCClientID is the client ID used for WalletConnect sessions
	WCClientID = "walletconnect"

	upsertWCSessionQuery = `INSERT INTO connector_wc_sessions 
		(topic, session_json, expiry, pairing_topic, dapp_url, client_id, created_timestamp, sym_key) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) 
		ON CONFLICT(topic) DO UPDATE SET 
			session_json = excluded.session_json, 
			expiry = excluded.expiry, 
			pairing_topic = excluded.pairing_topic, 
			dapp_url = excluded.dapp_url, 
			sym_key = excluded.sym_key`

	selectWCSessionQuery = `SELECT topic, session_json, expiry, pairing_topic, dapp_url, client_id, sym_key 
		FROM connector_wc_sessions 
		WHERE topic = ?`

	deleteWCSessionQuery = `DELETE FROM connector_wc_sessions 
		WHERE topic = ?`

	selectWCSessionsQuery = `SELECT topic, session_json, expiry, pairing_topic, dapp_url, client_id, sym_key 
		FROM connector_wc_sessions 
		WHERE expiry >= ? 
		ORDER BY created_timestamp DESC`

	selectWCSessionsByDAppURLQuery = `SELECT topic, session_json, expiry, pairing_topic, dapp_url, client_id, sym_key 
		FROM connector_wc_sessions 
		WHERE dapp_url = ?`
)

type WCSession struct {
	Topic        string `json:"topic"`
	SessionJSON  string `json:"sessionJson"`
	Expiry       int64  `json:"expiry"`
	PairingTopic string `json:"pairingTopic"`
	DAppURL      string `json:"dappUrl"`
	ClientID     string `json:"clientId"`
	SymKey       string `json:"symKey"`
}

func UpsertWCSession(db *sql.DB, topic, sessionJSON string, expiry int64, pairingTopic, dappURL, symKey string, createdAt int64) error {
	normalizedURL := NormalizeURL(dappURL)
	_, err := db.Exec(upsertWCSessionQuery, topic, sessionJSON, expiry, pairingTopic, normalizedURL, WCClientID, createdAt, symKey)
	return err
}

func SelectWCSession(db *sql.DB, topic string) (*WCSession, error) {
	s := &WCSession{Topic: topic}
	err := db.QueryRow(selectWCSessionQuery, topic).Scan(&s.Topic, &s.SessionJSON, &s.Expiry, &s.PairingTopic, &s.DAppURL, &s.ClientID, &s.SymKey)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func DeleteWCSession(db *sql.DB, topic string) error {
	_, err := db.Exec(deleteWCSessionQuery, topic)
	return err
}

func SelectActiveWCSessions(db *sql.DB, validAtTimestamp int64) ([]WCSession, error) {
	rows, err := db.Query(selectWCSessionsQuery, validAtTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []WCSession
	for rows.Next() {
		var s WCSession
		err = rows.Scan(&s.Topic, &s.SessionJSON, &s.Expiry, &s.PairingTopic, &s.DAppURL, &s.ClientID, &s.SymKey)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func SelectWCSessionsByDAppURL(db *sql.DB, dappURL string) ([]WCSession, error) {
	normalizedURL := NormalizeURL(dappURL)
	rows, err := db.Query(selectWCSessionsByDAppURLQuery, normalizedURL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []WCSession
	for rows.Next() {
		var s WCSession
		err = rows.Scan(&s.Topic, &s.SessionJSON, &s.Expiry, &s.PairingTopic, &s.DAppURL, &s.ClientID, &s.SymKey)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}
