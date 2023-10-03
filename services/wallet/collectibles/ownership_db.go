package collectibles

import (
	"database/sql"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/jmoiron/sqlx"

	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/sqlite"
)

const InvalidTimestamp = int64(-1)

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

const ownershipTimestampColumns = "owner_address, chain_id, timestamp"
const selectOwnershipTimestampColumns = "timestamp"

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

func updateAddressOwnershipTimestamp(creator sqlite.StatementCreator, ownerAddress common.Address, chainID w_common.ChainID, timestamp int64) error {
	updateTimestamp, err := creator.Prepare(fmt.Sprintf(`INSERT OR REPLACE INTO collectibles_ownership_update_timestamps (%s) 
																				VALUES (?, ?, ?)`, ownershipTimestampColumns))
	if err != nil {
		return err
	}

	_, err = updateTimestamp.Exec(ownerAddress, chainID, timestamp)

	return err
}

func (o *OwnershipDB) Update(chainID w_common.ChainID, ownerAddress common.Address, collectibles []thirdparty.CollectibleUniqueID, timestamp int64) (err error) {
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

	// Update timestamp
	err = updateAddressOwnershipTimestamp(tx, ownerAddress, chainID, timestamp)

	return
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

	return thirdparty.RowsToCollectibles(rows)
}

func (o *OwnershipDB) GetOwnedCollectible(chainID w_common.ChainID, ownerAddresses common.Address, contractAddress common.Address, tokenID *big.Int) (*thirdparty.CollectibleUniqueID, error) {
	query := fmt.Sprintf(`SELECT %s
		FROM collectibles_ownership_cache
		WHERE chain_id = ? AND owner_address = ? AND contract_address = ? AND token_id = ?`, selectOwnershipColumns)

	stmt, err := o.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(chainID, ownerAddresses, contractAddress, (*bigint.SQLBigIntBytes)(tokenID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids, err := thirdparty.RowsToCollectibles(rows)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return &ids[0], nil
}

func (o *OwnershipDB) GetOwnershipUpdateTimestamp(owner common.Address, chainID walletCommon.ChainID) (int64, error) {
	query := fmt.Sprintf(`SELECT %s
		FROM collectibles_ownership_update_timestamps
		WHERE owner_address = ? AND chain_id = ?`, selectOwnershipTimestampColumns)

	stmt, err := o.db.Prepare(query)
	if err != nil {
		return InvalidTimestamp, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(owner, chainID)

	var timestamp int64

	err = row.Scan(&timestamp)

	if err == sql.ErrNoRows {
		return InvalidTimestamp, nil
	} else if err != nil {
		return InvalidTimestamp, err
	}

	return timestamp, nil
}

func (o *OwnershipDB) GetLatestOwnershipUpdateTimestamp(chainID walletCommon.ChainID) (int64, error) {
	query := `SELECT MAX(timestamp)
		FROM collectibles_ownership_update_timestamps
		WHERE chain_id = ?`

	stmt, err := o.db.Prepare(query)
	if err != nil {
		return InvalidTimestamp, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(chainID)

	var timestamp sql.NullInt64

	err = row.Scan(&timestamp)

	if err != nil {
		return InvalidTimestamp, err
	}
	if timestamp.Valid {
		return timestamp.Int64, nil
	}

	return InvalidTimestamp, nil
}
