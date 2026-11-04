package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
)

const minSupportedNWAKUversion = 8
const maxSupportedNWAKUversion = 8

func handleNWakuPreMigration(db *sql.DB) (bool, error) {
	// Check if there's an user version in the DB, and if migration table does not exist.
	// Rename existing table, and move data afterwards

	var nwakuDBVersion int
	err := db.QueryRow("PRAGMA user_version").Scan(&nwakuDBVersion)
	if err != nil {
		return false, fmt.Errorf("could not obtain sqlite user_version while attempting to migrate nwaku database: %w", err)
	}

	var gowakuDBVersion int
	err = db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&gowakuDBVersion)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("could not obtain schema_migrations data while attempting to migrate nwaku database: %w", err)
	}

	if nwakuDBVersion == 0 {
		// not a nwaku db
		return false, nil
	}

	if nwakuDBVersion < minSupportedNWAKUversion || nwakuDBVersion > maxSupportedNWAKUversion {
		err = fmt.Errorf("unsupported nwaku DB %d - Supported versions [%d,%d]", nwakuDBVersion, minSupportedNWAKUversion, maxSupportedNWAKUversion)
		return false, err
	}

	if gowakuDBVersion > 0 {
		// We have already migrated this database
		return false, nil
	}

	_, err = db.Exec("ALTER TABLE message RENAME TO message_nwaku")
	if err != nil {
		return false, fmt.Errorf("could not rename nwaku message table while attempting to migrate nwaku database: %w", err)
	}

	_, err = db.Exec("DROP INDEX i_ts;")
	if err != nil {
		return false, fmt.Errorf("could not drop indexes while attempting to migrate nwaku database: %w", err)
	}

	_, err = db.Exec("DROP INDEX i_query;")
	if err != nil {
		return false, fmt.Errorf("could not drop indexes while attempting to migrate nwaku database: %w", err)
	}

	return true, nil
}

func handleNWakuPostMigration(db *sql.DB) error {
	_, err := db.Exec("INSERT INTO message(pubsubTopic, contentTopic, payload, version, timestamp, id, messageHash, storedAt) SELECT pubsubTopic, contentTopic, payload, version, timestamp, id, messageHash, storedAt FROM message_nwaku")
	if err != nil {
		return fmt.Errorf("could not migrate nwaku messages: %w", err)
	}

	_, err = db.Exec("DROP TABLE message_nwaku")
	if err != nil {
		return fmt.Errorf("could not drop nwaku message table: %w", err)
	}

	return nil
}

func revertNWakuPreMigration(db *sql.DB) error {
	_, err := db.Exec("ALTER TABLE message_nwaku RENAME TO message")
	if err != nil {
		return fmt.Errorf("could not revert changes to nwaku db: %w", err)
	}

	return nil
}
