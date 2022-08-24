//go:build !gowaku_run_migrations
// +build !gowaku_run_migrations

package migrations

import (
	"database/sql"
)

// Skip migration code
func Migrate(db *sql.DB) error {
	return nil
}
