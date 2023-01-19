package ens

import (
	"database/sql"
)

type Database struct {
	db *sql.DB
}

type UsernameDetails struct {
	Username string `json:"username"`
	ChainID  uint64 `json:"chainId"`
}

func NewEnsDatabase(db *sql.DB) *Database {
	return &Database{db: db}
}

func (db *Database) GetEnsUsernames() (result []*UsernameDetails, err error) {

	const sqlQuery = `SELECT username, chain_id
					  FROM ens_usernames`

	rows, err := db.db.Query(sqlQuery)

	if err != nil {
		return result, err
	}

	defer rows.Close()

	for rows.Next() {
		var ensUsername UsernameDetails
		err = rows.Scan(&ensUsername.Username, &ensUsername.ChainID)
		if err != nil {
			return nil, err
		}
		result = append(result, &ensUsername)
	}

	return result, nil
}

func (db *Database) AddEnsUsername(details UsernameDetails) error {
	const sqlQuery = `INSERT OR REPLACE INTO ens_usernames(username, chain_id)
					  VALUES (?, ?)`
	_, err := db.db.Exec(sqlQuery, details.Username, details.ChainID)
	return err
}

func (db *Database) RemoveEnsUsername(Username string, ChainID uint64) error {
	const sqlQuery = `DELETE FROM ens_usernames
					  WHERE username = (?) AND chain_id = ?`
	_, err := db.db.Exec(sqlQuery, Username, ChainID)
	return err
}
