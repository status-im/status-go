package newsfeed

import (
	"database/sql"
	"errors"
	"time"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/services/newsfeed/migrations"
	"github.com/status-im/status-go/sqlite"
)

func SQLiteMigrate(db *sql.DB) error {
	assets := &bindata.AssetSource{
		Names:     migrations.AssetNames(),
		AssetFunc: migrations.Asset,
	}
	migrationsTableName := "status_schema_migrations_service_newsfeed"

	err := sqlite.UpdateMigrationTableVersion(db, migrationsTableName, assets.Names, 0)
	if err != nil {
		return err
	}

	err = sqlite.Migrate(db, assets, sqlite.MigrateOptions{MigrationTableName: migrationsTableName})
	if err != nil {
		return err
	}

	return nil
}

type SQLitePersistence struct {
	db *sql.DB
}

func NewSQLitePersistence(db *sql.DB) *SQLitePersistence {
	return &SQLitePersistence{db: db}
}

func (p *SQLitePersistence) NewsFeedEnabled() (bool, error) {
	var enabled bool
	err := p.db.QueryRow("SELECT enabled FROM newsfeed_settings WHERE id = 1").Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func (p *SQLitePersistence) SaveNewsFeedEnabled(value bool) error {
	const query = "UPDATE newsfeed_settings SET enabled = ? WHERE id = 1"
	result, err := p.db.Exec(query, value, value)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	return err
}

func (p *SQLitePersistence) NewsRSSEnabled() (bool, error) {
	var enabled bool
	err := p.db.QueryRow("SELECT rss_enabled FROM newsfeed_settings WHERE id = 1").Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func (p *SQLitePersistence) SaveNewsRSSEnabled(value bool) error {
	const query = "UPDATE newsfeed_settings SET rss_enabled = ? WHERE id = 1"
	result, err := p.db.Exec(query, value, value)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	return err
}

func (p *SQLitePersistence) NewsFeedLastFetchedTimestamp() (time.Time, error) {
	var timestamp time.Time
	err := p.db.QueryRow("SELECT last_fetched_timestamp FROM newsfeed_settings WHERE id = 1").Scan(&timestamp)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return timestamp, nil
}

func (p *SQLitePersistence) SaveNewsFeedLastFetchedTimestamp(t time.Time) error {
	const query = "UPDATE newsfeed_settings SET last_fetched_timestamp = ? WHERE id = 1"
	result, err := p.db.Exec(query, t, t)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	return err
}
