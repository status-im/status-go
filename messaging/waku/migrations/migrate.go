package migrations

import (
	"database/sql"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/sqlite"
)

const (
	uuid      = "27b55b8c"
	TableName = "waku_" + uuid + "_schema_migrations"
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
