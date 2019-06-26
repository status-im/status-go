package sharedsecret

import (
	"database/sql"
	"strings"
)

type PersistenceService interface {
	// Add adds a shared secret, associated with an identity and an installationID
	Add(identity []byte, secret []byte, installationID string) error
	// Get returns a shared secret associated with multiple installationIDs
	Get(identity []byte, installationIDs []string) (*Response, error)
	// All returns an array of shared secrets, each one of them represented
	// as a byte array
	All() ([][][]byte, error)
}

type Response struct {
	secret          []byte
	installationIDs map[string]bool
}

type SQLLitePersistence struct {
	db *sql.DB
}

func NewSQLLitePersistence(db *sql.DB) *SQLLitePersistence {
	return &SQLLitePersistence{db: db}
}

func (s *SQLLitePersistence) Add(identity []byte, secret []byte, installationID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	insertSecretStmt, err := tx.Prepare("INSERT INTO secrets(identity, secret) VALUES (?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer insertSecretStmt.Close()

	_, err = insertSecretStmt.Exec(identity, secret)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	insertInstallationIDStmt, err := tx.Prepare("INSERT INTO secret_installation_ids(id, identity_id) VALUES (?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer insertInstallationIDStmt.Close()

	_, err = insertInstallationIDStmt.Exec(installationID, identity)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLLitePersistence) Get(identity []byte, installationIDs []string) (*Response, error) {
	response := &Response{
		installationIDs: make(map[string]bool),
	}
	args := make([]interface{}, len(installationIDs)+1)
	args[0] = identity
	for i, installationID := range installationIDs {
		args[i+1] = installationID
	}

	/* #nosec */
	query := `SELECT secret, id
	          FROM secrets t
		  JOIN
		  secret_installation_ids tid
		  ON t.identity = tid.identity_id
		  WHERE
		  t.identity = ?
		  AND
		  tid.id IN (?` + strings.Repeat(",?", len(installationIDs)-1) + `)`

	rows, err := s.db.Query(query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	for rows.Next() {
		var installationID string
		var secret []byte
		err = rows.Scan(&secret, &installationID)
		if err != nil {
			return nil, err
		}

		response.secret = secret
		response.installationIDs[installationID] = true
	}

	return response, nil
}

func (s *SQLLitePersistence) All() ([][][]byte, error) {
	query := `SELECT identity, secret
	          FROM secrets`

	var secrets [][][]byte

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var secret []byte
		var identity []byte
		err = rows.Scan(&identity, &secret)
		if err != nil {
			return nil, err
		}

		secrets = append(secrets, [][]byte{identity, secret})
	}

	return secrets, nil
}
