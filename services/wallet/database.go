package wallet

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// DBHeader fields from header that are stored in database.
type DBHeader struct {
	Number    *big.Int
	Hash      common.Hash
	Timestamp uint64
	// Head is true if the block was a head at the time it was pulled from chain.
	Head bool
}

func toDBHeader(header *types.Header) *DBHeader {
	return &DBHeader{
		Hash:      header.Hash(),
		Number:    header.Number,
		Timestamp: header.Time,
	}
}

func toHead(header *types.Header) *DBHeader {
	dbheader := toDBHeader(header)
	dbheader.Head = true
	return dbheader
}

// SyncOption is used to specify that application processed transfers for that block.
type SyncOption uint

const (
	// sync options
	ethSync   SyncOption = 1
	erc20Sync SyncOption = 2
)

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
		return nil, errors.New("not an int64")
	}
	return (*big.Int)(i).Int64(), nil
}

// JSONBlob type for marshaling/unmarshaling inner type to json.
type JSONBlob struct {
	data interface{}
}

// Scan implements interface.
func (blob *JSONBlob) Scan(value interface{}) error {
	if value == nil || reflect.ValueOf(blob.data).IsNil() {
		return nil
	}
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
	if blob.data == nil || reflect.ValueOf(blob.data).IsNil() {
		return nil, nil
	}
	return json.Marshal(blob.data)
}

func NewDB(db *sql.DB, network uint64) *Database {
	return &Database{db: db, network: network}
}

// Database sql wrapper for operations with wallet objects.
type Database struct {
	db      *sql.DB
	network uint64
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
	err = insertHeaders(tx, db.network, added)
	if err != nil {
		return
	}
	err = insertTransfers(tx, db.network, transfers)
	if err != nil {
		return
	}
	err = updateAccounts(tx, db.network, accounts, added, option)
	return
}

// GetTransfersByAddress loads transfers for a given address between two blocks.
func (db *Database) GetTransfersByAddress(address common.Address, start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterNetwork(db.network).FilterAddress(address).FilterStart(start).FilterEnd(end)
	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetTransfers load transfers transfer betweeen two blocks.
func (db *Database) GetTransfers(start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterNetwork(db.network).FilterStart(start).FilterEnd(end)
	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
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
	insert, err = tx.Prepare("INSERT INTO blocks(network_id, number, hash, timestamp) VALUES (?, ?, ?, ?)")
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
		_, err = insert.Exec(db.network, (*SQLBigInt)(h.Number), h.Hash(), h.Time)
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
	insert, err = tx.Prepare("INSERT INTO accounts_to_blocks(network_id, address, blk_number, sync) VALUES (?, ?,?,?)")
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
	_, err = insert.Exec(db.network, address, (*SQLBigInt)(header.Number), option)
	if err != nil {
		return
	}
	return err
}

// HeaderExists checks if header with hash exists in db.
func (db *Database) HeaderExists(hash common.Hash) (bool, error) {
	var val sql.NullBool
	err := db.db.QueryRow("SELECT EXISTS (SELECT hash FROM blocks WHERE hash = ? AND network_id = ?)", hash, db.network).Scan(&val)
	if err != nil {
		return false, err
	}
	return val.Bool, nil
}

// GetHeaderByNumber selects header using block number.
func (db *Database) GetHeaderByNumber(number *big.Int) (header *DBHeader, err error) {
	header = &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
	err = db.db.QueryRow("SELECT hash,number FROM blocks WHERE number = ? AND network_id = ?", (*SQLBigInt)(number), db.network).Scan(&header.Hash, (*SQLBigInt)(header.Number))
	if err == nil {
		return header, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

func (db *Database) GetLastHead() (header *DBHeader, err error) {
	header = &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
	err = db.db.QueryRow("SELECT hash,number FROM blocks WHERE network_id = $1 AND head = 1 AND number = (SELECT MAX(number) FROM blocks WHERE network_id = $1)", db.network).Scan(&header.Hash, (*SQLBigInt)(header.Number))
	if err == nil {
		return header, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// GetLatestSynced downloads last synced block with a given option.
func (db *Database) GetLatestSynced(address common.Address, option SyncOption) (header *DBHeader, err error) {
	header = &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
	err = db.db.QueryRow(`
SELECT blocks.hash, blk_number FROM accounts_to_blocks JOIN blocks ON blk_number = blocks.number WHERE blocks.network_id = $1 AND address = $2 AND blk_number
= (SELECT MAX(blk_number) FROM accounts_to_blocks WHERE network_id = $1 AND address = $2 AND sync & $3 = $3)`, db.network, address, option).Scan(&header.Hash, (*SQLBigInt)(header.Number))
	if err == nil {
		return header, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

type Token struct {
	Address common.Address `json:"address"`
	Name    string         `json:"name"`
	Symbol  string         `json:"symbol"`
	Color   string         `json:"color"`

	// Decimals defines how divisible the token is. For example, 0 would be
	// not divisible, whereas 18 would allow very small amounts of the token
	// to be traded.
	Decimals uint `json:"decimals"`
}

func (db *Database) GetCustomTokens() ([]*Token, error) {
	rows, err := db.db.Query(`SELECT address, name, symbol, decimals, color FROM tokens WHERE network_id = ?`, db.network)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []*Token
	for rows.Next() {
		token := &Token{}
		err = rows.Scan(&token.Address, &token.Name, &token.Symbol, &token.Decimals, &token.Color)

		if err != nil {
			return nil, err
		}

		rst = append(rst, token)
	}

	return rst, nil
}

func (db *Database) AddCustomToken(token Token) (err error) {
	var insert *sql.Stmt
	insert, err = db.db.Prepare("INSERT OR REPLACE INTO TOKENS (network_id, address, name, symbol, decimals, color) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return
	}
	_, err = insert.Exec(db.network, token.Address, token.Name, token.Symbol, token.Decimals, token.Color)
	return
}

func (db *Database) DeleteCustomToken(address common.Address) error {
	_, err := db.db.Exec(`DELETE FROM TOKENS WHERE address = ?`, address)
	return err
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

func insertHeaders(creator statementCreator, network uint64, headers []*DBHeader) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO blocks(network_id, hash, number, timestamp, head) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	for _, h := range headers {
		_, err = insert.Exec(network, h.Hash, (*SQLBigInt)(h.Number), h.Timestamp, h.Head)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertTransfers(creator statementCreator, network uint64, transfers []Transfer) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO transfers(network_id, hash, blk_hash, address, tx, sender, receipt, log, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	for _, t := range transfers {
		_, err = insert.Exec(network, t.ID, t.BlockHash, t.Address, &JSONBlob{t.Transaction}, t.From, &JSONBlob{t.Receipt}, &JSONBlob{t.Log}, t.Type)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateAccounts(creator statementCreator, network uint64, accounts []common.Address, headers []*DBHeader, option SyncOption) error {
	update, err := creator.Prepare("UPDATE accounts_to_blocks SET sync=sync|? WHERE address=? AND blk_number=? AND network_id=?")
	if err != nil {
		return err
	}
	insert, err := creator.Prepare("INSERT OR IGNORE INTO accounts_to_blocks(network_id,address,blk_number,sync) VALUES(?,?,?,?)")
	if err != nil {
		return err
	}
	for _, acc := range accounts {
		for _, h := range headers {
			rst, err := update.Exec(option, acc, (*SQLBigInt)(h.Number), network)
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
			_, err = insert.Exec(network, acc, (*SQLBigInt)(h.Number), option)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
