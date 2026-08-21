package walletdb

import (
	"database/sql"

	sqlite "github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/db/walletdb/migrations"
)

var walletCustomSteps = []*sqlite.PostStep{}

func doMigration(db *sql.DB) error {
	return migrations.Migrate(db, walletCustomSteps)
}

func MigrateDB(db *sql.DB) error {
	return doMigration(db)
}
