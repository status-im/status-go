package transfer

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/bigint"
)

// DBHeader fields from header that are stored in database.
type DBHeader struct {
	Number         *big.Int
	Hash           common.Hash
	Timestamp      uint64
	Erc20Transfers []*Transfer
	Network        uint64
	Address        common.Address
	// Head is true if the block was a head at the time it was pulled from chain.
	Head bool
	// Loaded is true if trasfers from this block has been already fetched
	Loaded bool
}

func toDBHeader(header *types.Header) *DBHeader {
	return &DBHeader{
		Hash:      header.Hash(),
		Number:    header.Number,
		Timestamp: header.Time,
		Loaded:    false,
	}
}

// SyncOption is used to specify that application processed transfers for that block.
type SyncOption uint

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

func NewDB(client *sql.DB) *Database {
	return &Database{client: client}
}

// Database sql wrapper for operations with wallet objects.
type Database struct {
	client *sql.DB
}

// Close closes database.
func (db *Database) Close() error {
	return db.client.Close()
}

func (db *Database) ProcessBlocks(chainID uint64, account common.Address, from *big.Int, to *Block, headers []*DBHeader) (err error) {
	var (
		tx *sql.Tx
	)

	tx, err = db.client.Begin()
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

	err = insertBlocksWithTransactions(chainID, tx, account, headers)
	if err != nil {
		return
	}

	err = upsertRange(chainID, tx, account, from, to)
	if err != nil {
		return
	}

	return
}

// TODO remove as not used
func (db *Database) SaveBlocks(chainID uint64, account common.Address, headers []*DBHeader) (err error) {
	var (
		tx *sql.Tx
	)
	tx, err = db.client.Begin()
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

	err = insertBlocksWithTransactions(chainID, tx, account, headers)
	if err != nil {
		return
	}

	return
}

// ProcessTransfers atomically adds/removes blocks and adds new transfers.
func (db *Database) ProcessTransfers(chainID uint64, transfers []Transfer, removed []*DBHeader) (err error) {
	var (
		tx *sql.Tx
	)
	tx, err = db.client.Begin()
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

	err = updateOrInsertTransfers(chainID, tx, transfers)
	if err != nil {
		return
	}
	return
}

// SaveTransfersMarkBlocksLoaded
func (db *Database) SaveTransfersMarkBlocksLoaded(chainID uint64, address common.Address, transfers []Transfer, blocks []*big.Int) (err error) {
	err = db.SaveTransfers(chainID, address, transfers)
	if err != nil {
		return
	}

	var tx *sql.Tx
	tx, err = db.client.Begin()
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
	err = markBlocksAsLoaded(chainID, tx, address, blocks)
	if err != nil {
		return
	}

	return
}

// SaveTransfers
func (db *Database) SaveTransfers(chainID uint64, address common.Address, transfers []Transfer) (err error) {
	var tx *sql.Tx
	tx, err = db.client.Begin()
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

	err = updateOrInsertTransfers(chainID, tx, transfers)
	if err != nil {
		return
	}

	return
}

// GetTransfersInRange loads transfers for a given address between two blocks.
func (db *Database) GetTransfersInRange(chainID uint64, address common.Address, start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterNetwork(chainID).FilterAddress(address).FilterStart(start).FilterEnd(end).FilterLoaded(1)
	rows, err := db.client.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetTransfersByAddress loads transfers for a given address between two blocks.
func (db *Database) GetTransfersByAddress(chainID uint64, address common.Address, toBlock *big.Int, limit int64) (rst []Transfer, err error) {
	query := newTransfersQuery().
		FilterNetwork(chainID).
		FilterAddress(address).
		FilterEnd(toBlock).
		FilterLoaded(1).
		Limit(limit)

	rows, err := db.client.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetTransfersByAddressAndBlock loads transfers for a given address and block.
func (db *Database) GetTransfersByAddressAndBlock(chainID uint64, address common.Address, block *big.Int, limit int64) (rst []Transfer, err error) {
	query := newTransfersQuery().
		FilterNetwork(chainID).
		FilterAddress(address).
		FilterBlockNumber(block).
		FilterLoaded(1).
		Limit(limit)

	rows, err := db.client.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetTransfers load transfers transfer between two blocks.
func (db *Database) GetTransfers(chainID uint64, start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterNetwork(chainID).FilterStart(start).FilterEnd(end).FilterLoaded(1)
	rows, err := db.client.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

func (db *Database) GetTransfersForIdentities(ctx context.Context, identities []TransactionIdentity) (rst []Transfer, err error) {
	query := newTransfersQuery()
	for _, identity := range identities {
		subQuery := newSubQuery()
		// TODO optimization: consider using tuples in sqlite and IN operator
		subQuery = subQuery.FilterNetwork(identity.ChainID).FilterTransactionHash(identity.Hash).FilterAddress(identity.Address)
		query.addSubQuery(subQuery, OrSeparator)
	}
	rows, err := db.client.QueryContext(ctx, query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

func (db *Database) GetPreloadedTransactions(chainID uint64, address common.Address, blockHash common.Hash) (rst []Transfer, err error) {
	query := newTransfersQuery().
		FilterNetwork(chainID).
		FilterAddress(address).
		FilterBlockHash(blockHash).
		FilterLoaded(0)

	rows, err := db.client.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

func (db *Database) GetTransactionsLog(chainID uint64, address common.Address, transactionHash common.Hash) (*types.Log, error) {
	l := &types.Log{}
	err := db.client.QueryRow("SELECT log FROM transfers WHERE network_id = ? AND address = ? AND hash = ?",
		chainID, address, transactionHash).
		Scan(&JSONBlob{l})
	if err == nil {
		return l, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// saveHeaders stores a list of headers atomically.
func (db *Database) saveHeaders(chainID uint64, headers []*types.Header, address common.Address) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)
	tx, err = db.client.Begin()
	if err != nil {
		return
	}
	insert, err = tx.Prepare("INSERT INTO blocks(network_id, blk_number, blk_hash, address) VALUES (?, ?, ?, ?)")
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
		_, err = insert.Exec(chainID, (*bigint.SQLBigInt)(h.Number), h.Hash(), address)
		if err != nil {
			return
		}
	}
	return
}

// getHeaderByNumber selects header using block number.
func (db *Database) getHeaderByNumber(chainID uint64, number *big.Int) (header *DBHeader, err error) {
	header = &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
	err = db.client.QueryRow("SELECT blk_hash, blk_number FROM blocks WHERE blk_number = ? AND network_id = ?", (*bigint.SQLBigInt)(number), chainID).Scan(&header.Hash, (*bigint.SQLBigInt)(header.Number))
	if err == nil {
		return header, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// statementCreator allows to pass transaction or database to use in consumer.
type statementCreator interface {
	Prepare(query string) (*sql.Stmt, error)
}

func deleteHeaders(creator statementCreator, headers []*DBHeader) error {
	delete, err := creator.Prepare("DELETE FROM blocks WHERE blk_hash = ?")
	if err != nil {
		return err
	}
	deleteTransfers, err := creator.Prepare("DELETE FROM transfers WHERE blk_hash = ?")
	if err != nil {
		return err
	}
	for _, h := range headers {
		_, err = delete.Exec(h.Hash)
		if err != nil {
			return err
		}

		_, err = deleteTransfers.Exec(h.Hash)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *Database) InsertBlock(chainID uint64, account common.Address, blockNumber *big.Int, blockHash common.Hash) error {
	var (
		tx *sql.Tx
	)
	tx, err := db.client.Begin()
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

	insert, err := tx.Prepare("INSERT OR IGNORE INTO blocks(network_id, address, blk_number, blk_hash, loaded) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = insert.Exec(chainID, account, (*bigint.SQLBigInt)(blockNumber), blockHash, true)
	return err
}

func insertBlocksWithTransactions(chainID uint64, creator statementCreator, account common.Address, headers []*DBHeader) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO blocks(network_id, address, blk_number, blk_hash, loaded) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	updateTx, err := creator.Prepare(`UPDATE transfers
	SET log = ?
	WHERE network_id = ? AND address = ? AND hash = ?`)
	if err != nil {
		return err
	}

	insertTx, err := creator.Prepare(`INSERT OR IGNORE
	INTO transfers (network_id, address, sender, hash, blk_number, blk_hash, type, timestamp, log, loaded, multi_transaction_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`)
	if err != nil {
		return err
	}

	for _, header := range headers {
		_, err = insert.Exec(chainID, account, (*bigint.SQLBigInt)(header.Number), header.Hash, header.Loaded)
		if err != nil {
			return err
		}
		if len(header.Erc20Transfers) > 0 {
			for _, transfer := range header.Erc20Transfers {
				res, err := updateTx.Exec(&JSONBlob{transfer.Log}, chainID, account, transfer.ID)
				if err != nil {
					return err
				}
				affected, err := res.RowsAffected()
				if err != nil {
					return err
				}
				if affected > 0 {
					continue
				}

				_, err = insertTx.Exec(chainID, account, account, transfer.ID, (*bigint.SQLBigInt)(header.Number), header.Hash, erc20Transfer, transfer.Timestamp, &JSONBlob{transfer.Log}, transfer.MultiTransactionID)
				if err != nil {
					log.Error("error saving erc20transfer", "err", err)
					return err
				}
			}
		}
	}
	return nil
}

func updateOrInsertTransfers(chainID uint64, creator statementCreator, transfers []Transfer) error {
	update, err := creator.Prepare(`UPDATE transfers
        SET tx = ?, sender = ?, receipt = ?, timestamp = ?, loaded = 1, base_gas_fee = ?
	WHERE address =?  AND hash = ?`)
	if err != nil {
		return err
	}

	insert, err := creator.Prepare(`INSERT OR IGNORE INTO transfers
        (network_id, hash, blk_hash, blk_number, timestamp, address, tx, sender, receipt, log, type, loaded, base_gas_fee, multi_transaction_id)
	VALUES
        (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`)
	if err != nil {
		return err
	}
	for _, t := range transfers {
		res, err := update.Exec(&JSONBlob{t.Transaction}, t.From, &JSONBlob{t.Receipt}, t.Timestamp, t.BaseGasFees, t.Address, t.ID)

		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected > 0 {
			continue
		}

		_, err = insert.Exec(chainID, t.ID, t.BlockHash, (*bigint.SQLBigInt)(t.BlockNumber), t.Timestamp, t.Address, &JSONBlob{t.Transaction}, t.From, &JSONBlob{t.Receipt}, &JSONBlob{t.Log}, t.Type, t.BaseGasFees, t.MultiTransactionID)
		if err != nil {
			log.Error("can't save transfer", "b-hash", t.BlockHash, "b-n", t.BlockNumber, "a", t.Address, "h", t.ID)
			return err
		}
	}
	return nil
}

// markBlocksAsLoaded(tx, address, chainID, blocks)
func markBlocksAsLoaded(chainID uint64, creator statementCreator, address common.Address, blocks []*big.Int) error {
	update, err := creator.Prepare("UPDATE blocks SET loaded=? WHERE address=? AND blk_number=? AND network_id=?")
	if err != nil {
		return err
	}

	for _, block := range blocks {
		_, err := update.Exec(true, address, (*bigint.SQLBigInt)(block), chainID)
		if err != nil {
			return err
		}
	}
	return nil
}
