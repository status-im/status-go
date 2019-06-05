package wallet

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/services/wallet/migrations"
	"github.com/status-im/status-go/sqlite"
)

// DBHeader fields from header that are stored in database.
type DBHeader struct {
	Number *big.Int
	Hash   common.Hash
}

func toDBHeader(header *types.Header) *DBHeader {
	return &DBHeader{
		Hash:   header.Hash(),
		Number: header.Number,
	}
}

// SyncOption is used to specify that application processed transfers for that block.
type SyncOption uint

const (
	errNoRows = "sql: no rows in result set"

	// sync options
	ethSync   SyncOption = 1
	erc20Sync SyncOption = 2
)

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path string) (*Database, error) {
	db, err := sqlite.OpenDB(path)
	if err != nil {
		return nil, err
	}
	err = migrations.Migrate(db)
	if err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
}

// SQLBigInt type for storing uint256 in the databse.
// FIXME(dshulyak) SQL big int is max 64 bits. Maybe store as bytes in big endian and hope
// that lexographical sorting will work.
type SQLBigInt big.Int

// Scan implements interface.
func (i *SQLBigInt) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("not an integer")
	}
	(*big.Int)(i).SetInt64(val)
	return nil
}

// Value implements interface.
func (i *SQLBigInt) Value() (driver.Value, error) {
	if !(*big.Int)(i).IsInt64() {
		return nil, errors.New("not at int64")
	}
	return (*big.Int)(i).Int64(), nil
}

// JSONBlob type for marshaling/unmarshaling inner type to json.
type JSONBlob struct {
	data interface{}
}

// Scan implements interface.
func (blob *JSONBlob) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("not a byte slice")
	}
	if len(bytes) == 0 {
		return nil
	}
	err := json.Unmarshal(bytes, blob.data)
	return err
}

// Value implements interface.
func (blob *JSONBlob) Value() (driver.Value, error) {
	return json.Marshal(blob.data)
}

// Database sql wrapper for operations with wallet objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

// ProcessTranfers atomically adds/removes blocks and adds new tranfers.
func (db Database) ProcessTranfers(transfers []Transfer, added, removed []*DBHeader, option SyncOption) (err error) {
	// TODO(dshulyak) split this method
	var (
		tx             *sql.Tx
		insert         *sql.Stmt
		blocks         *sql.Stmt
		delete         *sql.Stmt
		accountsUpdate *sql.Stmt
		accountsInsert *sql.Stmt
		rst            sql.Result
		affected       int64
	)
	tx, err = db.db.Begin()
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

	insert, err = tx.Prepare("INSERT OR IGNORE INTO transfers(hash, blk_hash, address, tx, receipt, type) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	delete, err = tx.Prepare("DELETE FROM blocks WHERE hash = ?")
	if err != nil {
		return err
	}
	blocks, err = tx.Prepare("INSERT OR IGNORE INTO blocks(hash, number) VALUES (?, ?)")
	if err != nil {
		return err
	}
	accountsUpdate, err = tx.Prepare("UPDATE accounts_to_blocks SET sync=sync|? WHERE address=? AND blk_number=?")
	if err != nil {
		return err
	}
	accountsInsert, err = tx.Prepare("INSERT OR IGNORE INTO accounts_to_blocks(address,blk_number,sync) VALUES(?,?,?)")
	if err != nil {
		return err
	}
	for _, header := range removed {
		_, err = delete.Exec(header.Hash)
		if err != nil {
			return err
		}
	}
	for _, header := range added {
		_, err = blocks.Exec(header.Hash, (*SQLBigInt)(header.Number))
		if err != nil {
			return err
		}
	}

	accountsChanges := map[common.Address]map[string]*big.Int{}
	for _, t := range transfers {
		_, err = insert.Exec(t.Transaction.Hash(), t.BlockHash, t.Address, &JSONBlob{t.Transaction}, &JSONBlob{t.Receipt}, t.Type)
		if err != nil {
			return err
		}
		_, exist := accountsChanges[t.Address]
		if !exist {
			accountsChanges[t.Address] = map[string]*big.Int{}
		}
		accountsChanges[t.Address][t.BlockNumber.String()] = t.BlockNumber
	}
	for address, changedBlocks := range accountsChanges {
		for _, blkNumber := range changedBlocks {
			rst, err = accountsUpdate.Exec(option, address, (*SQLBigInt)(blkNumber))
			if err != nil {
				return err
			}
			affected, err = rst.RowsAffected()
			if err != nil {
				return err
			}
			if affected > 0 {
				continue
			}
			_, err = accountsInsert.Exec(address, (*SQLBigInt)(blkNumber), option)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// GetTransfersByAddress loads transfers for a given address between two blocks.
func (db *Database) GetTransfersByAddress(address common.Address, start, end *big.Int) (rst []Transfer, err error) {
	// TODO(dshulyak) DRY
	query := "SELECT type, blocks.hash, blocks.number, address, tx, receipt FROM transfers JOIN blocks ON blk_hash = blocks.hash WHERE address == ? AND blocks.number >= ?"
	var (
		rows *sql.Rows
	)
	if end != nil {
		query += " AND blocks.number <= ?"
		rows, err = db.db.Query(query, address, (*SQLBigInt)(start), (*SQLBigInt)(end))
	} else {
		rows, err = db.db.Query(query, address, (*SQLBigInt)(start))
	}
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		transfer := Transfer{
			BlockNumber: &big.Int{},
			Transaction: &types.Transaction{},
			Receipt:     &types.Receipt{},
		}
		err = rows.Scan(
			&transfer.Type, &transfer.BlockHash, &transfer.Address, (*SQLBigInt)(transfer.BlockNumber),
			&JSONBlob{transfer.Transaction}, &JSONBlob{transfer.Receipt})
		if err != nil {
			return nil, err
		}
		rst = append(rst, transfer)
	}
	return
}

// GetTransfers load transfers transfer betweeen two blocks.
func (db *Database) GetTransfers(start, end *big.Int) (rst []Transfer, err error) {
	query := "SELECT type, blocks.hash, blocks.number, address, tx, receipt FROM transfers JOIN blocks ON blk_hash = blocks.hash WHERE blocks.number >= ?"
	var (
		rows *sql.Rows
	)
	if end != nil {
		query += " AND blocks.number <= ?"
		rows, err = db.db.Query(query, (*SQLBigInt)(start), (*SQLBigInt)(end))
	} else {
		rows, err = db.db.Query(query, (*SQLBigInt)(start))
	}
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		transfer := Transfer{
			BlockNumber: &big.Int{},
			Transaction: &types.Transaction{},
			Receipt:     &types.Receipt{},
		}
		err = rows.Scan(
			&transfer.Type, &transfer.BlockHash, (*SQLBigInt)(transfer.BlockNumber), &transfer.Address,
			&JSONBlob{transfer.Transaction}, &JSONBlob{transfer.Receipt})
		if err != nil {
			return nil, err
		}
		rst = append(rst, transfer)
	}
	return
}

// SaveHeader stores a single header.
func (db *Database) SaveHeader(header *types.Header) error {
	_, err := db.db.Exec("INSERT INTO blocks(number, hash) VALUES (?, ?)", (*SQLBigInt)(header.Number), header.Hash())
	return err
}

// SaveHeaders stores a list of headers atomically.
func (db *Database) SaveHeaders(headers []*types.Header) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)
	tx, err = db.db.Begin()
	if err != nil {
		return
	}
	insert, err = tx.Prepare("INSERT INTO blocks(number, hash) VALUES (?,?)")
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()

	for _, h := range headers {
		_, err = insert.Exec((*SQLBigInt)(h.Number), h.Hash())
		if err != nil {
			return
		}
	}
	return
}

func (db *Database) SaveSyncedHeader(address common.Address, header *types.Header, option SyncOption) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)
	tx, err = db.db.Begin()
	if err != nil {
		return
	}
	insert, err = tx.Prepare("INSERT INTO accounts_to_blocks(address, blk_number, sync) VALUES (?,?,?)")
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()
	_, err = insert.Exec(address, (*SQLBigInt)(header.Number), option)
	if err != nil {
		return
	}
	return err
}

// LastHeader selects last header by block number.
func (db *Database) LastHeader() (header *DBHeader, err error) {
	rows, err := db.db.Query("SELECT hash,number FROM blocks WHERE number = (SELECT MAX(number) FROM blocks)")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		header = &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
		err = rows.Scan(&header.Hash, (*SQLBigInt)(header.Number))
		if err != nil {
			return nil, err
		}
		if header != nil {
			return header, nil
		}
	}
	return nil, nil
}

// HeaderExists checks if header with hash exists in db.
func (db *Database) HeaderExists(hash common.Hash) (bool, error) {
	var val sql.NullBool
	err := db.db.QueryRow("SELECT EXISTS (SELECT hash FROM blocks WHERE hash = ?)", hash).Scan(&val)
	if err != nil {
		return false, err
	}
	return val.Bool, nil
}

// GetHeaderByNumber selects header using block number.
func (db *Database) GetHeaderByNumber(number *big.Int) (header *DBHeader, err error) {
	header = &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
	err = db.db.QueryRow("SELECT hash,number FROM blocks WHERE number = ?", (*SQLBigInt)(number)).Scan(&header.Hash, (*SQLBigInt)(header.Number))
	if err == nil {
		return header, nil
	}
	if err.Error() == errNoRows {
		return nil, nil
	}
	return nil, err
}

func (db *Database) GetEarliestSynced(address common.Address, option SyncOption) (header *DBHeader, err error) {
	rows, err := db.db.Query(`
SELECT blocks.hash, blk_number FROM accounts_to_blocks JOIN blocks ON blk_number = blocks.number WHERE address = $1 AND blk_number
= (SELECT MIN(blk_number) FROM accounts_to_blocks WHERE address = $1 AND sync & $2 = $2)`, address, option)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		header = &DBHeader{Number: new(big.Int)}
		err = rows.Scan(&header.Hash, (*SQLBigInt)(header.Number))
		if err != nil {
			return nil, err
		}
		if header != nil {
			return header, nil
		}
	}
	return nil, nil
}
