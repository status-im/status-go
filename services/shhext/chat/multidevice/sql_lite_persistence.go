package multidevice

import (
	"database/sql"
)

// SQLLitePersistence represents a persistence service tied to an SQLite database
type SQLLitePersistence struct {
	db *sql.DB
}

// NewSQLLitePersistence creates a new SQLLitePersistence instance, given a path and a key
func NewSQLLitePersistence(db *sql.DB) *SQLLitePersistence {
	return &SQLLitePersistence{db: db}
}

// GetActiveInstallations returns the active installations for a given identity
func (s *SQLLitePersistence) GetActiveInstallations(maxInstallations int, identity []byte) ([]*Installation, error) {
	stmt, err := s.db.Prepare(`SELECT installation_id, version
				   FROM installations
				   WHERE enabled = 1 AND identity = ?
				   ORDER BY timestamp DESC
				   LIMIT ?`)
	if err != nil {
		return nil, err
	}

	var installations []*Installation
	rows, err := stmt.Query(identity, maxInstallations)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var installationID string
		var version uint32
		err = rows.Scan(
			&installationID,
			&version,
		)
		if err != nil {
			return nil, err
		}
		installations = append(installations, &Installation{
			ID:      installationID,
			Version: version,
		})

	}

	return installations, nil

}

// AddInstallations adds the installations for a given identity, maintaining the enabled flag
func (s *SQLLitePersistence) AddInstallations(identity []byte, timestamp int64, installations []*Installation, defaultEnabled bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return nil
	}

	for _, installation := range installations {
		stmt, err := tx.Prepare(`SELECT enabled, version
					 FROM installations
					 WHERE identity = ? AND installation_id = ?
					 LIMIT 1`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		var oldEnabled bool
		// We don't override version once we saw one
		var oldVersion uint32
		latestVersion := installation.Version

		err = stmt.QueryRow(identity, installation.ID).Scan(&oldEnabled, &oldVersion)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if err == sql.ErrNoRows {
			stmt, err = tx.Prepare(`INSERT INTO installations(identity, installation_id, timestamp, enabled, version)
						VALUES (?, ?, ?, ?, ?)`)
			if err != nil {
				return err
			}
			defer stmt.Close()

			_, err = stmt.Exec(
				identity,
				installation.ID,
				timestamp,
				defaultEnabled,
				latestVersion,
			)
			if err != nil {
				return err
			}
		} else {
			// We update timestamp if present without changing enabled, only if this is a new bundle
			// and we set the version to the latest we ever saw
			if oldVersion > installation.Version {
				latestVersion = oldVersion
			}

			stmt, err = tx.Prepare(`UPDATE installations
					        SET timestamp = ?,  enabled = ?, version = ?
						WHERE identity = ?
						AND installation_id = ?
						AND timestamp < ?`)
			if err != nil {
				return err
			}
			defer stmt.Close()

			_, err = stmt.Exec(
				timestamp,
				oldEnabled,
				latestVersion,
				identity,
				installation.ID,
				timestamp,
			)
			if err != nil {
				return err
			}
		}

	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil

}

// EnableInstallation enables the installation
func (s *SQLLitePersistence) EnableInstallation(identity []byte, installationID string) error {
	stmt, err := s.db.Prepare(`UPDATE installations
				   SET enabled = 1
				   WHERE identity = ? AND installation_id = ?`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(identity, installationID)
	return err

}

// DisableInstallation disable the installation
func (s *SQLLitePersistence) DisableInstallation(identity []byte, installationID string) error {

	stmt, err := s.db.Prepare(`UPDATE installations
				   SET enabled = 0
				   WHERE identity = ? AND installation_id = ?`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(identity, installationID)
	return err
}
