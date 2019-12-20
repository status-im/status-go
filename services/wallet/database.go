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
	"github.com/ethereum/go-ethereum/log"
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

func (db Database) ProcessBlocks(account common.Address, from *big.Int, to *big.Int, blocks []*big.Int, transferType TransferType) (err error) {
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

	err = insertBlocksWithTransactions(tx, account, db.network, blocks)
	if err != nil {
		return
	}

	err = insertRange(tx, account, db.network, from, to, transferType)
	if err != nil {
		return
	}

	return
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

// SaveTranfers
func (db Database) SaveTranfers(address common.Address, transfers []Transfer) (err error) {
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

	err = insertTransfers(tx, db.network, transfers)
	if err != nil {
		return
	}

	err = markBlocksAsLoaded(tx, address, db.network, transfers)
	if err != nil {
		return
	}

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

// GetTransfersByAddressAndPage loads transfers for a given address between two blocks.
func (db *Database) GetTransfersByAddressAndPage(address common.Address, page, pageSize int64) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterNetwork(db.network).FilterAddress(address).Page(page, pageSize)
	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetTransfersByAddress loads transfers for a given address between two blocks.
func (db *Database) GetBlocksByAddress(address common.Address, limit int) (rst []*big.Int, err error) {
	query := `SELECT blk_number FROM blocks_with_transactions 
	WHERE address = ? AND network_id = ? AND loaded = 0
	ORDER BY blk_number DESC 
	LIMIT ?`
	rows, err := db.db.Query(query, address, db.network, limit)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*SQLBigInt)(block))
		if err != nil {
			return nil, err
		}
		rst = append(rst, block)
	}
	return rst, nil
}

func (db *Database) RemoveBlockWithTransfer(address common.Address, block *big.Int) error {
	query := `DELETE FROM blocks_with_transactions 
	WHERE address = ? 
	AND blk_number = ? 
	AND network_id = ?`

	_, err := db.db.Exec(query, address, (*SQLBigInt)(block), db.network)

	if err != nil {
		log.Info("block wasn't removed", "block", block, "error", err)
		return err
	}

	return nil
}

func (db *Database) GetLastBlockByAddress(address common.Address, limit int) (rst *big.Int, err error) {
	query := `SELECT * FROM 
	(SELECT blk_number FROM blocks_with_transactions WHERE address = ? AND network_id = ? ORDER BY blk_number DESC LIMIT ?)
	ORDER BY blk_number LIMIT 1`
	rows, err := db.db.Query(query, address, db.network, limit)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*SQLBigInt)(block))
		if err != nil {
			return nil, err
		}

		return block, nil
	}

	return nil, nil
}

func (db *Database) GetFirstKnownBlock(address common.Address, transferType TransferType) (rst *big.Int, err error) {
	query := `SELECT blk_from FROM blocks_ranges
	WHERE address = ?
	AND network_id = ?
	AND type = ?
	ORDER BY blk_from
	LIMIT 1`

	rows, err := db.db.Query(query, address, db.network, transferType)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*SQLBigInt)(block))
		if err != nil {
			return nil, err
		}

		return block, nil
	}

	return nil, nil
}

func (db *Database) GetLastKnownBlockByAddress(address common.Address, transferType TransferType) (rst *big.Int, err error) {
	query := `SELECT blk_to FROM blocks_ranges
	WHERE address = ?
	AND network_id = ?
	AND type = ?
	ORDER BY blk_to DESC
	LIMIT 1`

	rows, err := db.db.Query(query, address, db.network, transferType)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*SQLBigInt)(block))
		if err != nil {
			return nil, err
		}

		return block, nil
	}

	return nil, nil
}

func (db *Database) GetLastKnownBlockByAddresses(addresses []common.Address, transferType TransferType) (map[common.Address]*big.Int, []common.Address, error) {
	res := map[common.Address]*big.Int{}
	accountsWithoutHistory := []common.Address{}
	for _, address := range addresses {
		block, err := db.GetLastKnownBlockByAddress(address, transferType)
		if err != nil {
			log.Info("Can't get last block", "error", err)
			return nil, nil, err
		}

		if block != nil {
			res[address] = block
		} else {
			accountsWithoutHistory = append(accountsWithoutHistory, address)
		}
	}

	return res, accountsWithoutHistory, nil
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
	// indivisible, whereas 18 would allow very small amounts of the token
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
		err := rows.Scan(&token.Address, &token.Name, &token.Symbol, &token.Decimals, &token.Color)
		if err != nil {
			return nil, err
		}

		rst = append(rst, token)
	}

	return rst, nil
}

func (db *Database) AddCustomToken(token Token) error {
	insert, err := db.db.Prepare("INSERT OR REPLACE INTO TOKENS (network_id, address, name, symbol, decimals, color) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(db.network, token.Address, token.Name, token.Symbol, token.Decimals, token.Color)
	return err
}

func (db *Database) DeleteCustomToken(address common.Address) error {
	_, err := db.db.Exec(`DELETE FROM TOKENS WHERE address = ?`, address)
	return err
}

func (db *Database) getBlocksStats(address common.Address) (map[int64]int64, error) {
	query := `SELECT loaded, COUNT(blk_number) 
	FROM blocks_with_transactions 
	WHERE network_id = ?
	AND address = ?
	GROUP BY loaded`

	rows, err := db.db.Query(query, db.network, address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := map[int64]int64{}
	for rows.Next() {
		var loaded, count int64
		err = rows.Scan(&loaded, &count)
		if err != nil {
			return nil, err
		}

		res[loaded] = count
	}

	return res, nil
}

func (db *Database) getTransfersStats(address common.Address) int64 {
	query := `SELECT COUNT(blk_number) 
	FROM transfers 
	WHERE network_id = ?
	AND address = ?`

	rows, err := db.db.Query(query, db.network, address)
	if err != nil {
		return 0
	}
	defer rows.Close()

	res := int64(0)
	if rows.Next() {
		err = rows.Scan(&res)
		if err != nil {
			return 0
		}
	}

	return res
}

func (db *Database) GetTransfersStats(address common.Address) (map[int64]int64, int64) {
	blockStats, _ := db.getBlocksStats(address)
	transfersCount := db.getTransfersStats(address)

	return blockStats, transfersCount
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

func insertBlocksWithTransactions(creator statementCreator, account common.Address, network uint64, blocks []*big.Int) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO blocks_with_transactions(network_id, address, blk_number, loaded) VALUES (?, ?, ?, 0)")
	if err != nil {
		return err
	}
	for _, block := range blocks {
		_, err = insert.Exec(network, account, (*SQLBigInt)(block))
		if err != nil {
			return err
		}
	}
	return nil
}

func insertRange(creator statementCreator, account common.Address, network uint64, from *big.Int, to *big.Int, transferType TransferType) (err error) {
	insert, err := creator.Prepare("INSERT INTO blocks_ranges (network_id, address, blk_from, blk_to, type) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = insert.Exec(network, account, (*SQLBigInt)(from), (*SQLBigInt)(to), transferType)
	if err != nil {
		return err
	}

	return
}

func insertTransfers(creator statementCreator, network uint64, transfers []Transfer) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO transfers(network_id, hash, blk_hash, blk_number, timestamp, address, tx, sender, receipt, log, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	for _, t := range transfers {
		_, err = insert.Exec(network, t.ID, t.BlockHash, (*SQLBigInt)(t.BlockNumber), t.Timestamp, t.Address, &JSONBlob{t.Transaction}, t.From, &JSONBlob{t.Receipt}, &JSONBlob{t.Log}, t.Type)
		if err != nil {
			return err
		}
	}
	return nil
}

//markBlocksAsLoaded(tx, address, db.network, blocks)
func markBlocksAsLoaded(creator statementCreator, address common.Address, network uint64, transfers []Transfer) error {
	update, err := creator.Prepare("UPDATE blocks_with_transactions SET loaded=? WHERE address=? AND blk_number=? AND network_id=?")
	if err != nil {
		return err
	}

	blocks := []*big.Int{}
	uniqBlocksMap := map[*big.Int]struct{}{}
	for _, t := range transfers {
		if _, ok := uniqBlocksMap[t.BlockNumber]; !ok {
			uniqBlocksMap[t.BlockNumber] = struct{}{}
			blocks = append(blocks, t.BlockNumber)
		}
	}

	for _, block := range blocks {
		_, err := update.Exec(true, address, (*SQLBigInt)(block), network)
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
