package migrations

import (
	"database/sql"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/sqlite"
)

const (
	uuid      = "7676fd0c"
	TableName = "encryption_" + uuid + "_schema_migrations"
)

func Migrate(database *sql.DB) error {
	resources := bindata.Resource(
		AssetNames(),
		func(name string) ([]byte, error) {
			return Asset(name)
		},
	)

	options := sqlite.MigrateOptions{
		MigrationTableName: TableName,
	}

	return sqlite.Migrate(database, resources, options)
}
