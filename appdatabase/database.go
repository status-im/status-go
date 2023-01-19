package appdatabase

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/appdatabase/migrations"
	migrationsprevnodecfg "github.com/status-im/status-go/appdatabase/migrationsprevnodecfg"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/sqlite"
)

const nodeCfgMigrationDate = 1640111208

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path, password string, kdfIterationsNumber int) (*sql.DB, error) {
	db, err := sqlite.OpenDB(path, password, kdfIterationsNumber)
	if err != nil {
		return nil, err
	}

	// Check if the migration table exists
	row := db.QueryRow("SELECT exists(SELECT name FROM sqlite_master WHERE type='table' AND name='status_go_schema_migrations')")
	migrationTableExists := false
	err = row.Scan(&migrationTableExists)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var lastMigration uint64 = 0
	if migrationTableExists {
		row = db.QueryRow("SELECT version FROM status_go_schema_migrations")
		err = row.Scan(&lastMigration)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	if !migrationTableExists || (lastMigration > 0 && lastMigration < nodeCfgMigrationDate) {
		// If it's the first time migration's being run, or latest migration happened before migrating the nodecfg table
		err = migrationsprevnodecfg.Migrate(db)
		if err != nil {
			return nil, err
		}

		// NodeConfig migration cannot be done with SQL
		err = nodecfg.MigrateNodeConfig(db)
		if err != nil {
			return nil, err
		}
	}

	err = migrations.Migrate(db)
	if err != nil {
		return nil, err
	}

	// Migrate `settings.usernames` here, because current SQL implementation doesn't support `json_each`
	err = MigrateEnsUsernames(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// DecryptDatabase creates an unencrypted copy of the database and copies it
// over to the given directory
func DecryptDatabase(oldPath, newPath, password string, kdfIterationsNumber int) error {
	return sqlite.DecryptDB(oldPath, newPath, password, kdfIterationsNumber)
}

// EncryptDatabase creates an encrypted copy of the database and copies it to the
// user path
func EncryptDatabase(oldPath, newPath, password string, kdfIterationsNumber int) error {
	return sqlite.EncryptDB(oldPath, newPath, password, kdfIterationsNumber)
}

func ChangeDatabasePassword(path string, password string, kdfIterationsNumber int, newPassword string) error {
	return sqlite.ChangeEncryptionKey(path, password, kdfIterationsNumber, newPassword)
}

// GetDBFilename takes an instance of sql.DB and returns the filename of the "main" database
func GetDBFilename(db *sql.DB) (string, error) {
	if db == nil {
		logger := log.New()
		logger.Warn("GetDBFilename was passed a nil pointer sql.DB")
		return "", nil
	}

	var i, category, filename string
	rows, err := db.Query("PRAGMA database_list;")
	if err != nil {
		return "", err
	}

	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&i, &category, &filename)
		if err != nil {
			return "", err
		}

		// The "main" database is the one we care about
		if category == "main" {
			return filename, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	return "", errors.New("no main database found")
}

func MigrateEnsUsernames(db *sql.DB) error {

	// 1. Check if ens_usernames table already exist

	// row := db.QueryRow("SELECT exists(SELECT name FROM sqlite_master WHERE type='table' AND name='ens_usernames')")
	// tableExists := false
	// err := row.Scan(&tableExists)

	// if err != nil && err != sql.ErrNoRows {
	// 	return err
	// }

	// if tableExists {
	// 	return nil
	// }

	// -- 1. Create new ens_usernames table

	// _, err = db.Exec(`CREATE TABLE IF NOT EXISTS ens_usernames (
	// 	"username" TEXT NOT NULL,
	// 	"chain_id" UNSIGNED BIGINT DEFAULT 1);`)

	// if err != nil {
	// 	log.Error("Migrating ens usernames: failed to create table", "err", err.Error())
	// 	return err
	// }

	// -- 2. Move current `settings.usernames` to the new table
	/*
		INSERT INTO ens_usernames (username)
			SELECT json_each.value FROM settings, json_each(usernames);
	*/

	rows, err := db.Query(`SELECT usernames FROM settings`)

	if err != nil {
		log.Error("Migrating ens usernames: failed to query 'settings.usernames'", "err", err.Error())
		return err
	}

	defer rows.Close()

	var usernames []string

	for rows.Next() {
		var usernamesJSON sql.NullString
		err := rows.Scan(&usernamesJSON)

		if err != nil {
			return err
		}

		if !usernamesJSON.Valid {
			continue
		}

		var list []string
		err = json.Unmarshal([]byte(usernamesJSON.String), &list)
		if err != nil {
			return err
		}

		usernames = append(usernames, list...)
	}

	defaultChainID := 1

	for _, username := range usernames {

		var usernameAlreadyMigrated bool

		row := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM ens_usernames WHERE username=? AND chain_id=?)`, username, defaultChainID)
		err := row.Scan(&usernameAlreadyMigrated)

		if err != nil {
			return err
		}

		if usernameAlreadyMigrated {
			continue
		}

		_, err = db.Exec(`INSERT INTO ens_usernames (username, chain_id) VALUES (?, ?)`, username, defaultChainID)
		if err != nil {
			log.Error("Migrating ens usernames: failed to insert username into new database", "ensUsername", username, "err", err.Error())
		}
	}

	return nil
}
