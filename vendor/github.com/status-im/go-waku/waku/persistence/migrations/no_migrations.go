//go:build gowaku_skip_migrations
// +build gowaku_skip_migrations

package migrations

import (
	"database/sql"
)

// Skip migration code
func Migrate(db *sql.DB) error {
	return nil
}
