package messaging

import (
	"database/sql"

	"github.com/pkg/errors"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"

	"github.com/status-im/status-go/messaging/internal"
	"github.com/status-im/status-go/messaging/layers/encryption"
	encryptionmigrations "github.com/status-im/status-go/messaging/layers/encryption/migrations"
	"github.com/status-im/status-go/messaging/layers/segmentation"
	segmentationmigrations "github.com/status-im/status-go/messaging/layers/segmentation/migrations"
	"github.com/status-im/status-go/messaging/layers/transport"
	transportmigrations "github.com/status-im/status-go/messaging/layers/transport/migrations"
	"github.com/status-im/status-go/messaging/types"
	waku "github.com/status-im/status-go/messaging/waku"
	wakumigrations "github.com/status-im/status-go/messaging/waku/migrations"
	"github.com/status-im/status-go/sqlite"
)

/*
 Reference implementation of the Persistence interface using SQLite.
*/

// SQLiteMigrate applies necessary migrations to the SQLite database schema.
func SQLiteMigrate(database *sql.DB, version uint) error {
	if version > 0 {
		err := createMigrationTables(database, version)
		if err != nil {
			return errors.Wrap(err, "failed to create migration tables")
		}
	}

	err := mvdsmigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply mvds migrations")
	}

	err = wakumigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply waku migrations")
	}

	err = transportmigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply transport migrations")
	}

	err = segmentationmigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply segmentation migrations")
	}

	err = encryptionmigrations.Migrate(database)
	if err != nil {
		return errors.Wrap(err, "failed to apply encryption migrations")
	}

	return nil
}

// Migration tables were transitioned from a single shared table in the client (status_protocol_go)
// to dedicated pre-component tables. To maintain migration consistency and prevent reapplication
// of migrations already executed in the client, it is essential to initialize any newly created
// migration tables with the latest version. This ensures that components introduced into the client
// do not re-run migrations that have previously been applied.
func createMigrationTables(database *sql.DB, version uint) error {
	createMigrationTable := func(tableName string, assetNames []string) error {
		latestApplied := sqlite.LatestMigrationUpToVersion(assetNames, version)
		if latestApplied > 0 {
			return sqlite.CreateMigrationTable(database, tableName, latestApplied)
		}
		return nil
	}

	err := createMigrationTable(wakumigrations.TableName, wakumigrations.AssetNames())
	if err != nil {
		return err
	}

	err = createMigrationTable(transportmigrations.TableName, transportmigrations.AssetNames())
	if err != nil {
		return err
	}

	err = createMigrationTable(segmentationmigrations.TableName, segmentationmigrations.AssetNames())
	if err != nil {
		return err
	}

	return createMigrationTable(encryptionmigrations.TableName, encryptionmigrations.AssetNames())
}

// Returns internal implementation of WakuPersistence.
// Clients should not use this instance directly.
func NewWakuSQLitePersistence(db *sql.DB) types.WakuPersistence {
	return &internal.SQLiteWakuPersistence{
		SQLiteProtectedTopicsPersistence: waku.NewSQLiteProtectedTopicsPersistence(db),
	}
}

// Returns internal implementation of TransportPersistence.
// Clients should not use this instance directly.
func NewTransportSQLitePersistence(db *sql.DB) types.TransportPersistence {
	return &internal.SQLiteTransportPersistence{
		SQLiteKeysPersistence:                     transport.NewSQLiteKeysPersistence(db),
		SQLiteProcessedMessageIDsCachePersistence: transport.NewSQLiteProcessedMessageIDsCachePersistence(db),
	}
}

// Returns internal implementation of SegmentationPersistence.
// Clients should not use this instance directly.
func NewSegmentationSQLitePersistence(db *sql.DB) types.SegmentationPersistence {
	return &internal.SQLiteSegmentationPersistence{
		SQLitePersistence: segmentation.NewSQLitePersistence(db),
	}
}

// Returns internal implementation of EncryptionPersistence.
// Clients should not use this instance directly.
func NewEncryptionSQLitePersistence(db *sql.DB) types.EncryptionPersistence {
	return &internal.SQLiteEncryptionPersistence{
		SQLitePersistence: encryption.NewSQLitePersistence(db),
	}
}
