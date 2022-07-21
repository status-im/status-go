package walletconnect

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

type Session struct {
	PeerId           string   `json:"peer-id"`
	ConnectorInfo    string   `json:"connector-info"`
}

func (db *Database) InsertWalletConnectSession(session Session) (Session, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return session,err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	sessionInsertPreparedStatement, err := tx.Prepare("INSERT OR REPLACE INTO wallet_connect_sessions(peer_id, connector_info) VALUES(?, ?)")
	if err != nil {
		return session,err
	}
	_, err = sessionInsertPreparedStatement.Exec(session.PeerId, session.ConnectorInfo)
	sessionInsertPreparedStatement.Close()
	if err != nil {
		return session,err
	}

	return session,err
}

func (db *Database) GetWalletConnectSession() (Session, error) {
	tx, err := db.db.Begin()

	seshObject := Session{
		PeerId:        "",
		ConnectorInfo: "",
	}
	if err != nil {
		return seshObject,err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	rows, err := tx.Query("SELECT * FROM wallet_connect_sessions")

	if err != nil {
		return seshObject,err
	}

	defer rows.Close()

	for rows.Next() {
		var PeerId string
		var ConnectorInfo string

		errPeerId := rows.Scan(&PeerId)
		errConnectorInfo := rows.Scan(&ConnectorInfo)

		if errPeerId != nil {
			return seshObject, errPeerId
		}

		if errConnectorInfo != nil {
			return seshObject, errConnectorInfo
		}

		seshObject.PeerId = PeerId
		seshObject.ConnectorInfo = ConnectorInfo
	}

	return seshObject,err
}