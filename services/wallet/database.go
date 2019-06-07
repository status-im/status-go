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
	// sync options
	ethSync   SyncOption = 1
	erc20Sync SyncOption = 2
)

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
func (db Database) ProcessTranfers(transfers []Transfer, accounts []common.Address, added, removed []*DBHeader, option SyncOption) (err error) {
	var (
		tx *sql.Tx
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
	err = deleteHeaders(tx, removed)
	if err != nil {
		return
	}
	err = insertHeaders(tx, added)
	if err != nil {
		return
	}
	err = insertTransfers(tx, transfers)
	if err != nil {
		return
	}
	err = updateAccounts(tx, accounts, added, option)
	return
}

// GetTransfersByAddress loads transfers for a given address between two blocks.
func (db *Database) GetTransfersByAddress(address common.Address, start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterAddress(address).FilterStart(start).FilterEnd(end)
	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetTransfers load transfers transfer betweeen two blocks.
func (db *Database) GetTransfers(start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterStart(start).FilterEnd(end)
	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
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

func (db *Database) LastHeaders(limit *big.Int) ([]*DBHeader, error) {
	rows, err := db.db.Query("SELECT hash,number FROM blocks WHERE number ORDER BY number DESC LIMIT ?", (*SQLBigInt)(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	headers := []*DBHeader{}
	for rows.Next() {
		header := &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
		err = rows.Scan(&header.Hash, (*SQLBigInt)(header.Number))
		if err != nil {
			return nil, err
		}
		headers = append(headers, header)
	}
	return headers, nil
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
	if err == sql.ErrNoRows {
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

// statementCreator allows to pass transaction or database to use in consumer.
type statementCreator interface {
	Prepare(query string) (*sql.Stmt, error)
}

func deleteHeaders(creator statementCreator, headers []*DBHeader) error {
	delete, err := creator.Prepare("DELETE FROM blocks WHERE hash = ?")
	if err != nil {
		return err
	}
	for _, h := range headers {
		_, err = delete.Exec(h.Hash)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertHeaders(creator statementCreator, headers []*DBHeader) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO blocks(hash, number) VALUES (?, ?)")
	if err != nil {
		return err
	}
	for _, h := range headers {
		_, err = insert.Exec(h.Hash, (*SQLBigInt)(h.Number))
		if err != nil {
			return err
		}
	}
	return nil
}

func insertTransfers(creator statementCreator, transfers []Transfer) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO transfers(hash, blk_hash, address, tx, receipt, type) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	for _, t := range transfers {
		_, err = insert.Exec(t.ID, t.BlockHash, t.Address, &JSONBlob{t.Transaction}, &JSONBlob{t.Receipt}, t.Type)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateAccounts(creator statementCreator, accounts []common.Address, headers []*DBHeader, option SyncOption) error {
	update, err := creator.Prepare("UPDATE accounts_to_blocks SET sync=sync|? WHERE address=? AND blk_number=?")
	if err != nil {
		return err
	}
	insert, err := creator.Prepare("INSERT OR IGNORE INTO accounts_to_blocks(address,blk_number,sync) VALUES(?,?,?)")
	if err != nil {
		return err
	}
	for _, acc := range accounts {
		for _, h := range headers {
			rst, err := update.Exec(option, acc, (*SQLBigInt)(h.Number))
			if err != nil {
				return err
			}
			affected, err := rst.RowsAffected()
			if err != nil {
				return err
			}
			if affected > 0 {
				continue
			}
			_, err = insert.Exec(acc, (*SQLBigInt)(h.Number), option)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
