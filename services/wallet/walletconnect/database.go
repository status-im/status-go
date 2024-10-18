package walletconnect

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
)

type DBSession struct {
	Topic            Topic  `json:"topic"`
	Disconnected     bool   `json:"disconnected"`
	SessionJSON      string `json:"sessionJson"`
	Expiry           int64  `json:"expiry"`
	CreatedTimestamp int64  `json:"createdTimestamp"`
	PairingTopic     Topic  `json:"pairingTopic"`
	TestChains       bool   `json:"testChains"`
	DBDApp
}

type DBDApp struct {
	URL     string `json:"url"`
	Name    string `json:"name"`
	IconURL string `json:"iconUrl"`
}

func UpsertSession(db *sql.DB, data DBSession) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %v", err)
	}
	defer func() {
		if err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				logutils.ZapLogger().Error("error rolling back transaction", zap.NamedError("rollErr", rollErr), zap.Error(err))
			}
		}
	}()

	upsertDappStmt := `INSERT INTO wallet_connect_dapps (url, name, icon_url) VALUES (?, ?, ?)
                   ON CONFLICT(url) DO UPDATE SET name = excluded.name, icon_url = excluded.icon_url`
	_, err = tx.Exec(upsertDappStmt, data.URL, data.Name, data.IconURL)
	if err != nil {
		return fmt.Errorf("upsert wallet_connect_dapps: %v", err)
	}

	upsertSessionStmt := `INSERT INTO wallet_connect_sessions (
			topic,
			disconnected,
			session_json,
			expiry,
			created_timestamp,
			pairing_topic,
			test_chains,
			dapp_url
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(topic) DO UPDATE SET
			disconnected = excluded.disconnected,
			session_json = excluded.session_json,
			expiry = excluded.expiry,
			created_timestamp = excluded.created_timestamp,
			pairing_topic = excluded.pairing_topic,
			test_chains = excluded.test_chains,
			dapp_url = excluded.dapp_url;`
	_, err = tx.Exec(upsertSessionStmt, data.Topic, data.Disconnected, data.SessionJSON, data.Expiry, data.CreatedTimestamp, data.PairingTopic, data.TestChains, data.URL)
	if err != nil {
		return fmt.Errorf("insert session: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %v", err)
	}

	return nil
}

func DeleteSession(db *sql.DB, topic Topic) error {
	_, err := db.Exec("DELETE FROM wallet_connect_sessions WHERE topic = ?", topic)
	return err
}

func DisconnectSession(db *sql.DB, topic Topic) error {
	res, err := db.Exec("UPDATE wallet_connect_sessions SET disconnected = 1 WHERE topic = ?", topic)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("topic %s not found to update state", topic)
	}

	return nil
}

// GetSessionByTopic returns sql.ErrNoRows if no session is found.
func GetSessionByTopic(db *sql.DB, topic Topic) (*DBSession, error) {
	query := selectAndJoinQueryStr + " WHERE sessions.topic = ?"

	row := db.QueryRow(query, topic)
	return scanSession(singleRow{row})
}

// GetSessionsByPairingTopic returns sql.ErrNoRows if no session is found.
func GetSessionsByPairingTopic(db *sql.DB, pairingTopic Topic) ([]DBSession, error) {
	query := selectAndJoinQueryStr + " WHERE sessions.pairing_topic = ?"

	rows, err := db.Query(query, pairingTopic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSessions(rows)
}

type Scanner interface {
	Scan(dest ...interface{}) error
}

type singleRow struct {
	*sql.Row
}

func (r singleRow) Scan(dest ...interface{}) error {
	return r.Row.Scan(dest...)
}

const selectAndJoinQueryStr = `
	SELECT
		sessions.topic, sessions.disconnected, sessions.session_json, sessions.expiry, sessions.created_timestamp,
		sessions.pairing_topic, sessions.test_chains, sessions.dapp_url, dapps.name, dapps.icon_url
	FROM
		wallet_connect_sessions sessions
	JOIN
		wallet_connect_dapps dapps ON sessions.dapp_url = dapps.url`

// scanSession scans a single session from the given scanner following selectAndJoinQueryStr.
func scanSession(scanner Scanner) (*DBSession, error) {
	var session DBSession

	err := scanner.Scan(
		&session.Topic,
		&session.Disconnected,
		&session.SessionJSON,
		&session.Expiry,
		&session.CreatedTimestamp,
		&session.PairingTopic,
		&session.TestChains,
		&session.URL,
		&session.Name,
		&session.IconURL,
	)

	if err != nil {
		return nil, err
	}

	return &session, nil
}

// scanSessions returns sql.ErrNoRows if nothing is scanned.
func scanSessions(rows *sql.Rows) ([]DBSession, error) {
	var sessions []DBSession

	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// GetActiveSessions returns all active sessions (not disconnected and not expired) that have an expiry timestamp newer or equal to the given timestamp.
func GetActiveSessions(db *sql.DB, validAtTimestamp int64) ([]DBSession, error) {
	querySQL := selectAndJoinQueryStr + `
		WHERE
			sessions.disconnected = 0 AND
			sessions.expiry >= ?
		ORDER BY
			sessions.expiry DESC`

	rows, err := db.Query(querySQL, validAtTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// GetSessions returns all sessions in the ascending order of creation time
func GetSessions(db *sql.DB) ([]DBSession, error) {
	querySQL := selectAndJoinQueryStr + `
		ORDER BY
			sessions.created_timestamp DESC`

	rows, err := db.Query(querySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// GetActiveDapps returns all dapps in the order of last first time connected (first session creation time)
func GetActiveDapps(db *sql.DB, validAtTimestamp int64, testChains bool) ([]DBDApp, error) {
	query := `SELECT dapps.url, dapps.name, dapps.icon_url, MIN(sessions.created_timestamp) as dapp_creation_time
		FROM
			wallet_connect_dapps dapps
		JOIN
			wallet_connect_sessions sessions ON dapps.url = sessions.dapp_url
		WHERE sessions.disconnected = 0 AND sessions.expiry >= ? AND sessions.test_chains = ?
		GROUP BY dapps.url
		ORDER BY dapp_creation_time DESC;`

	rows, err := db.Query(query, validAtTimestamp, testChains)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dapps []DBDApp

	for rows.Next() {
		var dapp DBDApp
		var creationTime sql.NullInt64
		if err := rows.Scan(&dapp.URL, &dapp.Name, &dapp.IconURL, &creationTime); err != nil {
			return nil, err
		}
		dapps = append(dapps, dapp)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dapps, nil
}
