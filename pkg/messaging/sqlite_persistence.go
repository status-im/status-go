package messaging

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"

	"github.com/status-im/status-go/internal/db/sqlite"
	common2 "github.com/status-im/status-go/pkg/messaging/common"
	messagesendermigrations "github.com/status-im/status-go/pkg/messaging/common/migrations"
	encryption2 "github.com/status-im/status-go/pkg/messaging/layers/encryption"
	encryptionmigrations "github.com/status-im/status-go/pkg/messaging/layers/encryption/migrations"
	segmentation2 "github.com/status-im/status-go/pkg/messaging/layers/segmentation"
	segmentationmigrations "github.com/status-im/status-go/pkg/messaging/layers/segmentation/migrations"
	transport2 "github.com/status-im/status-go/pkg/messaging/layers/transport"
	transportmigrations "github.com/status-im/status-go/pkg/messaging/layers/transport/migrations"
	wakumigrations "github.com/status-im/status-go/pkg/messaging/waku/migrations"
)

type migrationsMetadata struct {
	*bindata.AssetSource
	MigrationTableName string
}

var migrations = []migrationsMetadata{
	{
		AssetSource: &bindata.AssetSource{
			Names:     wakumigrations.AssetNames(),
			AssetFunc: wakumigrations.Asset,
		},
		MigrationTableName: "status_schema_migrations_waku",
	},
	{
		AssetSource: &bindata.AssetSource{
			Names:     transportmigrations.AssetNames(),
			AssetFunc: transportmigrations.Asset,
		},
		MigrationTableName: "status_schema_migrations_transport",
	},
	{
		AssetSource: &bindata.AssetSource{
			Names:     segmentationmigrations.AssetNames(),
			AssetFunc: segmentationmigrations.Asset,
		},
		MigrationTableName: "status_schema_migrations_segmentation",
	},
	{
		AssetSource: &bindata.AssetSource{
			Names:     encryptionmigrations.AssetNames(),
			AssetFunc: encryptionmigrations.Asset,
		},
		MigrationTableName: "status_schema_migrations_encryption",
	},
	{
		AssetSource: &bindata.AssetSource{
			Names:     messagesendermigrations.AssetNames(),
			AssetFunc: messagesendermigrations.Asset,
		},
		MigrationTableName: "status_schema_migrations_message_sender",
	},
}

// SQLiteMigrate applies necessary migrations to the SQLite database schema.
func SQLiteMigrate(database *sql.DB, maxVersion uint) error {
	if maxVersion > 0 {
		err := createMigrationTables(database, maxVersion)
		if err != nil {
			return errors.Wrap(err, "failed to update migration tables")
		}
	}

	err := mvdsmigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply mvds migrations")
	}

	for _, m := range migrations {
		err := sqlite.Migrate(database, m.AssetSource, sqlite.MigrateOptions{MigrationTableName: m.MigrationTableName})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to apply %s migrations", m.MigrationTableName))
		}
	}

	return nil
}

// Migration tables were transitioned from a single shared table in the client (status_protocol_go)
// to dedicated pre-component tables. To maintain migration consistency and prevent reapplication
// of migrations already executed in the client, it is essential to initialize any newly created
// migration tables with the latest version. This ensures that components introduced into the client
// do not re-run migrations that have previously been applied.
func createMigrationTables(database *sql.DB, maxVersion uint) error {
	for _, m := range migrations {
		err := sqlite.UpdateMigrationTableVersion(database, m.MigrationTableName, m.Names, maxVersion)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to update migration table %s", m.MigrationTableName))
		}
	}

	return nil
}

type sqlitePersistence struct {
	db *sql.DB
}

var _ Persistence = (*sqlitePersistence)(nil)

func newSQLitePersistence(db *sql.DB) Persistence {
	return &sqlitePersistence{db: db}
}

type sqliteTransportPersistence struct {
	db *sql.DB
}

var _ transport2.Persistence = (*sqliteTransportPersistence)(nil)

func (p *sqliteTransportPersistence) KeysStorage() transport2.KeysPersistence {
	return transport2.NewSQLiteKeysPersistence(p.db)
}

func (p *sqliteTransportPersistence) ProcessedMessageIDsCacheStorage() transport2.ProcessedMessageIDsCachePersistence {
	return transport2.NewSQLiteProcessedMessageIDsCachePersistence(p.db)
}

func (p *sqlitePersistence) TransportStorage() transport2.Persistence {
	return &sqliteTransportPersistence{db: p.db}
}

func (p *sqlitePersistence) SegmentationStorage() segmentation2.Persistence {
	return segmentation2.NewSQLitePersistence(p.db)
}

func (p *sqlitePersistence) MVDSStorage() mvdsnode.Persistence {
	return mvdsnode.NewSQLitePersistence(p.db)
}

func (p *sqlitePersistence) EncryptionStorage() encryption2.Persistence {
	return encryption2.NewSQLitePersistence(p.db)
}

func (p *sqlitePersistence) MessageConfirmationStorage() common2.MessageConfirmationPersistence {
	return common2.NewSQLiteMessageConfirmationPersistence(p.db)
}

func (p *sqlitePersistence) HashRatchetStorage() common2.HashRatchetPersistence {
	return common2.NewSQLiteHashRatchetPersistence(p.db)
}
