package walletconnect

import (
	"database/sql"
	"errors"
)

type Pairing struct {
	Topic       PairingTopic `json:"topic"`
	Expiry      int64        `json:"expiry"`
	Active      bool         `json:"active"`
	AppName     string       `json:"appName"`
	URL         string       `json:"url"`
	Description string       `json:"description"`
	Icon        string       `json:"icon"`
	Verified    Verified     `json:"verified"`
}

func InsertPairing(db *sql.DB, pairing Pairing) error {
	insertSQL := `INSERT INTO wallet_connect_pairings (topic, expiry_timestamp, active, app_name, url, description, icon, verified_is_scam, verified_origin, verified_verify_url, verified_validation) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(insertSQL, pairing.Topic, pairing.Expiry, pairing.Active, pairing.AppName, pairing.URL, pairing.Description, pairing.Icon, pairing.Verified.IsScam, pairing.Verified.Origin, pairing.Verified.VerifyURL, pairing.Verified.Validation)
	return err
}

func ChangePairingState(db *sql.DB, topic PairingTopic, active bool) error {
	stmt, err := db.Prepare("UPDATE wallet_connect_pairings SET active = ? WHERE topic = ?")
	if err != nil {
		return err
	}

	res, err := stmt.Exec(active, topic)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("unable to locate pairing entry for DB state change")
	}

	return nil
}

func GetPairingByTopic(db *sql.DB, topic PairingTopic) (*Pairing, error) {
	querySQL := `SELECT topic, expiry_timestamp, active, app_name, url, description, icon, verified_is_scam, verified_origin, verified_verify_url, verified_validation FROM wallet_connect_pairings WHERE topic = ?`

	row := db.QueryRow(querySQL, topic)

	var pairing Pairing
	err := row.Scan(&pairing.Topic, &pairing.Expiry, &pairing.Active, &pairing.AppName, &pairing.URL, &pairing.Description, &pairing.Icon, &pairing.Verified.IsScam, &pairing.Verified.Origin, &pairing.Verified.VerifyURL, &pairing.Verified.Validation)
	if err != nil {
		return nil, err
	}

	return &pairing, nil
}

// GetActivePairings returns all active pairings (active and not expired) that have an expiry timestamp newer or equal to the given timestamp.
func GetActivePairings(db *sql.DB, expiryNotOlderThanTimestamp int64) ([]Pairing, error) {
	querySQL := `SELECT topic, expiry_timestamp, active, app_name, url, description, icon, verified_is_scam, verified_origin, verified_verify_url, verified_validation FROM wallet_connect_pairings WHERE active != 0 AND expiry_timestamp >= ? ORDER BY expiry_timestamp DESC`

	rows, err := db.Query(querySQL, expiryNotOlderThanTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairings := make([]Pairing, 0, 2)
	for rows.Next() {
		var pairing Pairing
		err := rows.Scan(&pairing.Topic, &pairing.Expiry, &pairing.Active, &pairing.AppName, &pairing.URL, &pairing.Description, &pairing.Icon, &pairing.Verified.IsScam, &pairing.Verified.Origin, &pairing.Verified.VerifyURL, &pairing.Verified.Validation)
		if err != nil {
			return nil, err
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		pairings = append(pairings, pairing)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pairings, nil
}

func HasActivePairings(db *sql.DB, expiryNotOlderThanTimestamp int64) (bool, error) {
	querySQL := `SELECT EXISTS(SELECT 1 FROM wallet_connect_pairings WHERE active != 0 AND expiry_timestamp >= ?)`

	row := db.QueryRow(querySQL, expiryNotOlderThanTimestamp)

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
