package migrations

import (
	"database/sql"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/internal/db/sqlite"
)

// Migrate applies migrations.
// see Migrate in vendor/status-go/sqlite/migrate.go
func Migrate(db *sql.DB, customSteps []*sqlite.PostStep) error {
	options := sqlite.MigrateOptions{
		CustomSteps: customSteps,
	}
	return sqlite.Migrate(db, bindata.Resource(
		AssetNames(),
		func(name string) ([]byte, error) {
			return Asset(name)
		},
	), options)
}
