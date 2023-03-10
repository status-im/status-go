package collectibles

import (
	"database/sql"
	"strings"
)

type Database struct {
	db *sql.DB
}

type TokenOwner struct {
	Address string
	Amount  int
}

func NewCollectiblesDatabase(db *sql.DB) *Database {
	return &Database{db: db}
}

func (db *Database) GetAmount(chainID uint64, contractAddress string, owner string) (int, error) {
	const selectQuery = `SELECT amount FROM community_token_owners WHERE chain_id=? AND address=? AND owner=? LIMIT 1`
	rows, err := db.db.Query(selectQuery, chainID, contractAddress, owner)
	if err != nil {
		return -1, err
	}
	defer rows.Close()
	if rows.Next() {
		var amount int
		err := rows.Scan(&amount)
		if err != nil {
			return -1, err
		}
		return amount, nil
	}
	return 0, nil
}

func (db *Database) setAmount(chainID uint64, contractAddress string, owner string, amount int) error {
	const sqlQuery = `INSERT OR REPLACE INTO community_token_owners(chain_id, address, owner, amount) VALUES (?, ?, ?, ?)`
	_, err := db.db.Exec(sqlQuery, chainID, contractAddress, owner, amount)
	return err
}

func (db *Database) AddTokenOwners(chainID uint64, contractAddress string, owners []string) error {
	for _, v := range owners {
		lowerVal := strings.ToLower(v)
		amount, err := db.GetAmount(chainID, contractAddress, lowerVal)
		if err != nil {
			return err
		}
		err = db.setAmount(chainID, contractAddress, lowerVal, amount+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *Database) GetTokenOwners(chainID uint64, contractAddress string) ([]TokenOwner, error) {
	const selectQuery = `SELECT owner, amount FROM community_token_owners WHERE chain_id=? AND address=?`
	rows, err := db.db.Query(selectQuery, chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var owners []TokenOwner
	for rows.Next() {
		var owner TokenOwner
		err := rows.Scan(&owner.Address, &owner.Amount)
		if err != nil {
			return nil, err
		}
		owners = append(owners, owner)
	}
	return owners, nil
}
