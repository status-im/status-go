package permissions

import (
	"database/sql"

	"github.com/status-im/status-go/services/permissions/migrations"
	"github.com/status-im/status-go/sqlite"
)

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path, password string) (*Database, error) {
	db, err := sqlite.OpenDB(path, password)
	if err != nil {
		return nil, err
	}
	err = migrations.Migrate(db)
	if err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
}

type DappPermissions struct {
	Name        string   `json:"dapp"`
	Permissions []string `json:"permissions,omitempty"`
}

func (db *Database) AddPermissions(perms DappPermissions) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)
	tx, err = db.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()
	insert, err = tx.Prepare("INSERT OR REPLACE INTO dapps(name) VALUES(?)")
	if err != nil {
		return
	}
	_, err = insert.Exec(perms.Name)
	insert.Close()
	if err != nil {
		return
	}
	if len(perms.Permissions) == 0 {
		return
	}
	insert, err = tx.Prepare("INSERT INTO permissions(dapp_name, permission) VALUES(?, ?)")
	if err != nil {
		return
	}
	defer insert.Close()
	for _, perm := range perms.Permissions {
		_, err = insert.Exec(perms.Name, perm)
		if err != nil {
			return
		}
	}
	return
}

func (db *Database) GetPermissions() (rst []DappPermissions, err error) {
	var (
		tx   *sql.Tx
		rows *sql.Rows
	)
	tx, err = db.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()
	// FULL and RIGHT joins are not supported
	rows, err = tx.Query("SELECT name FROM dapps")
	if err != nil {
		return
	}
	dapps := map[string]*DappPermissions{}
	for rows.Next() {
		perms := DappPermissions{}
		err = rows.Scan(&perms.Name)
		if err != nil {
			return nil, err
		}
		dapps[perms.Name] = &perms
	}
	rows, err = tx.Query("SELECT dapp_name, permission from permissions")
	if err != nil {
		return
	}
	var (
		name       string
		permission string
	)
	for rows.Next() {
		err = rows.Scan(&name, &permission)
		if err != nil {
			return
		}
		dapps[name].Permissions = append(dapps[name].Permissions, permission)
	}
	rst = make([]DappPermissions, 0, len(dapps))
	for key := range dapps {
		rst = append(rst, *dapps[key])
	}
	return rst, nil
}

func (db *Database) DeletePermission(name string) error {
	_, err := db.db.Exec("DELETE FROM dapps WHERE name = ?", name)
	return err
}
