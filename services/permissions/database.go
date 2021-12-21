package permissions

import (
	"database/sql"
)

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

type DappPermissions struct {
	Name        string   `json:"dapp"`
	Permissions []string `json:"permissions,omitempty"`
}

func (db *Database) AddPermissions(perms DappPermissions) (err error) {
	tx, err := db.db.Begin()
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

	dInsert, err := tx.Prepare("INSERT OR REPLACE INTO dapps(name) VALUES(?)")
	if err != nil {
		return
	}
	_, err = dInsert.Exec(perms.Name)
	dInsert.Close()
	if err != nil {
		return
	}

	if len(perms.Permissions) == 0 {
		return
	}
	pInsert, err := tx.Prepare("INSERT INTO permissions(dapp_name, permission) VALUES(?, ?)")
	if err != nil {
		return
	}
	defer pInsert.Close()
	for _, perm := range perms.Permissions {
		_, err = pInsert.Exec(perms.Name, perm)
		if err != nil {
			return
		}
	}
	return
}

func (db *Database) GetPermissions() (rst []DappPermissions, err error) {
	tx, err := db.db.Begin()
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
	dRows, err := tx.Query("SELECT name FROM dapps")
	if err != nil {
		return
	}
	defer dRows.Close()
	dapps := map[string]*DappPermissions{}
	for dRows.Next() {
		perms := DappPermissions{}
		err = dRows.Scan(&perms.Name)
		if err != nil {
			return nil, err
		}
		dapps[perms.Name] = &perms
	}

	pRows, err := tx.Query("SELECT dapp_name, permission from permissions")
	if err != nil {
		return
	}
	defer pRows.Close()
	var (
		name       string
		permission string
	)
	for pRows.Next() {
		err = pRows.Scan(&name, &permission)
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

func (db *Database) GetPermissionsByDappName(dappName string) (rst *DappPermissions, err error) {
	tx, err := db.db.Begin()
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

	rst = &DappPermissions{
		Name: dappName,
	}

	pRows, err := tx.Query("SELECT permission from permissions WHERE dapp_name = ?", dappName)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer pRows.Close()

	var permission string
	for pRows.Next() {
		err = pRows.Scan(&permission)
		if err != nil {
			return
		}
		rst.Permissions = append(rst.Permissions, permission)
	}

	return rst, nil
}

func (db *Database) DeletePermission(name string) error {
	_, err := db.db.Exec("DELETE FROM dapps WHERE name = ?", name)
	return err
}

func (db *Database) DeleteDappPermission(dappName, permission string) error {
	_, err := db.db.Exec("DELETE FROM permissions WHERE dapp_name = ? AND permission = ?", dappName, permission)
	return err
}

func (db *Database) HasPermission(dappName string, permission string) (bool, error) {
	var count uint64
	err := db.db.QueryRow(`SELECT COUNT(1) FROM permissions WHERE dapp_name = ? AND permission = ?`, dappName, permission).Scan(&count)
	return count > 0, err
}
