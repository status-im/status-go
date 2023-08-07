package collectibles

import (
	"database/sql"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/jmoiron/sqlx"

	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/sqlite"
)

type OwnershipDB struct {
	db *sql.DB
}

func NewOwnershipDB(sqlDb *sql.DB) *OwnershipDB {
	return &OwnershipDB{
		db: sqlDb,
	}
}

const ownershipColumns = "chain_id, contract_address, token_id, owner_address"
const selectOwnershipColumns = "chain_id, contract_address, token_id"

func removeAddressOwnership(creator sqlite.StatementCreator, chainID w_common.ChainID, ownerAddress common.Address) error {
	deleteOwnership, err := creator.Prepare("DELETE FROM collectibles_ownership_cache WHERE chain_id = ? AND owner_address = ?")
	if err != nil {
		return err
	}

	_, err = deleteOwnership.Exec(chainID, ownerAddress)
	if err != nil {
		return err
	}

	return nil
}

func insertAddressOwnership(creator sqlite.StatementCreator, ownerAddress common.Address, collectibles []thirdparty.CollectibleUniqueID) error {
	insertOwnership, err := creator.Prepare(fmt.Sprintf(`INSERT INTO collectibles_ownership_cache (%s) 
																				VALUES (?, ?, ?, ?)`, ownershipColumns))
	if err != nil {
		return err
	}

	for _, c := range collectibles {
		_, err = insertOwnership.Exec(c.ContractID.ChainID, c.ContractID.Address, (*bigint.SQLBigIntBytes)(c.TokenID.Int), ownerAddress)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *OwnershipDB) Update(chainID w_common.ChainID, ownerAddress common.Address, collectibles []thirdparty.CollectibleUniqueID) (err error) {
	var (
		tx *sql.Tx
	)
	tx, err = o.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	// Remove previous ownership data
	err = removeAddressOwnership(tx, chainID, ownerAddress)
	if err != nil {
		return err
	}

	// Insert new ownership data
	err = insertAddressOwnership(tx, ownerAddress, collectibles)
	if err != nil {
		return err
	}

	return
}

func rowsToCollectibles(rows *sql.Rows) ([]thirdparty.CollectibleUniqueID, error) {
	var ids []thirdparty.CollectibleUniqueID
	for rows.Next() {
		id := thirdparty.CollectibleUniqueID{
			TokenID: &bigint.BigInt{Int: big.NewInt(0)},
		}
		err := rows.Scan(
			&id.ContractID.ChainID,
			&id.ContractID.Address,
			(*bigint.SQLBigIntBytes)(id.TokenID.Int),
		)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func (o *OwnershipDB) GetOwnedCollectibles(chainIDs []w_common.ChainID, ownerAddresses []common.Address, offset int, limit int) ([]thirdparty.CollectibleUniqueID, error) {
	query, args, err := sqlx.In(fmt.Sprintf(`SELECT %s
		FROM collectibles_ownership_cache
		WHERE chain_id IN (?) AND owner_address IN (?)
		LIMIT ? OFFSET ?`, selectOwnershipColumns), chainIDs, ownerAddresses, limit, offset)
	if err != nil {
		return nil, err
	}

	stmt, err := o.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToCollectibles(rows)
}
