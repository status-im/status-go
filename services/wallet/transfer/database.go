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
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/sqlite"
)

// DBHeader fields from header that are stored in database.
type DBHeader struct {
	Number                *big.Int
	Hash                  common.Hash
	Timestamp             uint64
	PreloadedTransactions []*PreloadedTransaction
	Network               uint64
	Address               common.Address
	// Head is true if the block was a head at the time it was pulled from chain.
	Head bool
	// Loaded is true if transfers from this block have been already fetched
	Loaded bool
}

func toDBHeader(header *types.Header, blockHash common.Hash) *DBHeader {
	return &DBHeader{
		Hash:      blockHash,
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

func saveTransfersMarkBlocksLoaded(tx *sql.Tx, chainID uint64, address common.Address, transfers []Transfer, blocks []*big.Int) (err error) {
	err = updateOrInsertTransfers(chainID, tx, transfers)
	if err != nil {
		return
	}

	err = markBlocksAsLoaded(chainID, tx, address, blocks)
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
	return query.TransferScan(rows)
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
	return query.TransferScan(rows)
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
	return query.TransferScan(rows)
}

// GetTransfers load transfers transfer between two blocks.
func (db *Database) GetTransfers(chainID uint64, start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterNetwork(chainID).FilterStart(start).FilterEnd(end).FilterLoaded(1)
	rows, err := db.client.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.TransferScan(rows)
}

func (db *Database) GetTransfersForIdentities(ctx context.Context, identities []TransactionIdentity) (rst []Transfer, err error) {
	query := newTransfersQuery()
	for _, identity := range identities {
		subQuery := newSubQuery()
		subQuery = subQuery.FilterNetwork(uint64(identity.ChainID)).FilterTransactionHash(identity.Hash).FilterAddress(identity.Address)
		query.addSubQuery(subQuery, OrSeparator)
	}
	rows, err := db.client.QueryContext(ctx, query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.TransferScan(rows)
}

func (db *Database) GetTransactionsToLoad(chainID uint64, address common.Address, blockNumber *big.Int) (rst []PreloadedTransaction, err error) {
	query := newTransfersQuery().
		FilterNetwork(chainID).
		FilterAddress(address).
		FilterBlockNumber(blockNumber).
		FilterLoaded(0)

	rows, err := db.client.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.PreloadedTransactionScan(rows)
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

// Only used by status-mobile
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

	blockDB := blockDBFields{
		chainID:     chainID,
		account:     account,
		blockNumber: blockNumber,
		blockHash:   blockHash,
	}
	return insertBlockDBFields(tx, blockDB)
}

type blockDBFields struct {
	chainID     uint64
	account     common.Address
	blockNumber *big.Int
	blockHash   common.Hash
}

func insertBlockDBFields(creator statementCreator, block blockDBFields) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO blocks(network_id, address, blk_number, blk_hash, loaded) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = insert.Exec(block.chainID, block.account, (*bigint.SQLBigInt)(block.blockNumber), block.blockHash, true)
	return err
}

func insertBlocksWithTransactions(chainID uint64, creator statementCreator, account common.Address, headers []*DBHeader) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO blocks(network_id, address, blk_number, blk_hash, loaded) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	updateTx, err := creator.Prepare(`UPDATE transfers
	SET log = ?, log_index = ?
	WHERE network_id = ? AND address = ? AND hash = ?`)
	if err != nil {
		return err
	}

	insertTx, err := creator.Prepare(`INSERT OR IGNORE
	INTO transfers (network_id, address, sender, hash, blk_number, blk_hash, type, timestamp, log, loaded, log_index)
	VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, 0, ?)`)
	if err != nil {
		return err
	}

	for _, header := range headers {
		_, err = insert.Exec(chainID, account, (*bigint.SQLBigInt)(header.Number), header.Hash, header.Loaded)
		if err != nil {
			return err
		}
		for _, transaction := range header.PreloadedTransactions {
			var logIndex *uint
			if transaction.Log != nil {
				logIndex = new(uint)
				*logIndex = transaction.Log.Index
			}
			res, err := updateTx.Exec(&JSONBlob{transaction.Log}, logIndex, chainID, account, transaction.ID)
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

			_, err = insertTx.Exec(chainID, account, account, transaction.ID, (*bigint.SQLBigInt)(header.Number), header.Hash, w_common.Erc20Transfer, &JSONBlob{transaction.Log}, logIndex)
			if err != nil {
				log.Error("error saving Erc20transfer", "err", err)
				return err
			}
		}
	}
	return nil
}

func updateOrInsertTransfers(chainID uint64, creator statementCreator, transfers []Transfer) error {
	txsDBFields := make([]transferDBFields, 0, len(transfers))
	for _, t := range transfers {
		var receiptType *uint8
		var txHash, blockHash *common.Hash
		var receiptStatus, cumulativeGasUsed, gasUsed *uint64
		var contractAddress *common.Address
		var transactionIndex, logIndex *uint

		if t.Receipt != nil {
			receiptType = &t.Receipt.Type
			receiptStatus = &t.Receipt.Status
			txHash = &t.Receipt.TxHash
			if t.Log != nil {
				logIndex = new(uint)
				*logIndex = t.Log.Index
			}
			blockHash = &t.Receipt.BlockHash
			cumulativeGasUsed = &t.Receipt.CumulativeGasUsed
			contractAddress = &t.Receipt.ContractAddress
			gasUsed = &t.Receipt.GasUsed
			transactionIndex = &t.Receipt.TransactionIndex
		}

		var txProtected *bool
		var txGas, txNonce, txSize *uint64
		var txGasPrice, txGasTipCap, txGasFeeCap *big.Int
		var txType *uint8
		var txValue *big.Int
		var tokenAddress *common.Address
		var tokenID *big.Int
		var txFrom *common.Address
		var txTo *common.Address
		if t.Transaction != nil {
			if t.Log != nil {
				_, tokenAddress, tokenID, txValue, txFrom, txTo = w_common.ExtractTokenIdentity(t.Type, t.Log, t.Transaction)
			} else {
				txValue = new(big.Int).Set(t.Transaction.Value())
				txFrom = &t.From
				txTo = t.Transaction.To()
			}

			txType = new(uint8)
			*txType = t.Transaction.Type()
			txProtected = new(bool)
			*txProtected = t.Transaction.Protected()
			txGas = new(uint64)
			*txGas = t.Transaction.Gas()
			txGasPrice = t.Transaction.GasPrice()
			txGasTipCap = t.Transaction.GasTipCap()
			txGasFeeCap = t.Transaction.GasFeeCap()
			txNonce = new(uint64)
			*txNonce = t.Transaction.Nonce()
			txSize = new(uint64)
			*txSize = t.Transaction.Size()
		}

		dbFields := transferDBFields{
			chainID:            chainID,
			id:                 t.ID,
			blockHash:          t.BlockHash,
			blockNumber:        t.BlockNumber,
			timestamp:          t.Timestamp,
			address:            t.Address,
			transaction:        t.Transaction,
			sender:             t.From,
			receipt:            t.Receipt,
			log:                t.Log,
			transferType:       t.Type,
			baseGasFees:        t.BaseGasFees,
			multiTransactionID: t.MultiTransactionID,
			receiptStatus:      receiptStatus,
			receiptType:        receiptType,
			txHash:             txHash,
			logIndex:           logIndex,
			receiptBlockHash:   blockHash,
			cumulativeGasUsed:  cumulativeGasUsed,
			contractAddress:    contractAddress,
			gasUsed:            gasUsed,
			transactionIndex:   transactionIndex,
			txType:             txType,
			txProtected:        txProtected,
			txGas:              txGas,
			txGasPrice:         txGasPrice,
			txGasTipCap:        txGasTipCap,
			txGasFeeCap:        txGasFeeCap,
			txValue:            txValue,
			txNonce:            txNonce,
			txSize:             txSize,
			tokenAddress:       tokenAddress,
			tokenID:            tokenID,
			txFrom:             txFrom,
			txTo:               txTo,
		}
		txsDBFields = append(txsDBFields, dbFields)
	}

	return updateOrInsertTransfersDBFields(creator, txsDBFields)
}

type transferDBFields struct {
	chainID            uint64
	id                 common.Hash
	blockHash          common.Hash
	blockNumber        *big.Int
	timestamp          uint64
	address            common.Address
	transaction        *types.Transaction
	sender             common.Address
	receipt            *types.Receipt
	log                *types.Log
	transferType       w_common.Type
	baseGasFees        string
	multiTransactionID MultiTransactionIDType
	receiptStatus      *uint64
	receiptType        *uint8
	txHash             *common.Hash
	logIndex           *uint
	receiptBlockHash   *common.Hash
	cumulativeGasUsed  *uint64
	contractAddress    *common.Address
	gasUsed            *uint64
	transactionIndex   *uint
	txType             *uint8
	txProtected        *bool
	txGas              *uint64
	txGasPrice         *big.Int
	txGasTipCap        *big.Int
	txGasFeeCap        *big.Int
	txValue            *big.Int
	txNonce            *uint64
	txSize             *uint64
	tokenAddress       *common.Address
	tokenID            *big.Int
	txFrom             *common.Address
	txTo               *common.Address
}

func updateOrInsertTransfersDBFields(creator statementCreator, transfers []transferDBFields) error {
	insert, err := creator.Prepare(`INSERT OR REPLACE INTO transfers
        (network_id, hash, blk_hash, blk_number, timestamp, address, tx, sender, receipt, log, type, loaded, base_gas_fee, multi_transaction_id,
		status, receipt_type, tx_hash, log_index, block_hash, cumulative_gas_used, contract_address, gas_used, tx_index,
		tx_type, protected, gas_limit, gas_price_clamped64, gas_tip_cap_clamped64, gas_fee_cap_clamped64, amount_padded128hex, account_nonce, size, token_address, token_id, tx_from_address, tx_to_address)
	VALUES
        (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, t := range transfers {
		txGasPrice := sqlite.BigIntToClampedInt64(t.txGasPrice)
		txGasTipCap := sqlite.BigIntToClampedInt64(t.txGasTipCap)
		txGasFeeCap := sqlite.BigIntToClampedInt64(t.txGasFeeCap)
		txValue := sqlite.BigIntToPadded128BitsStr(t.txValue)

		_, err = insert.Exec(t.chainID, t.id, t.blockHash, (*bigint.SQLBigInt)(t.blockNumber), t.timestamp, t.address, &JSONBlob{t.transaction}, t.sender, &JSONBlob{t.receipt}, &JSONBlob{t.log}, t.transferType, t.baseGasFees, t.multiTransactionID,
			t.receiptStatus, t.receiptType, t.txHash, t.logIndex, t.receiptBlockHash, t.cumulativeGasUsed, t.contractAddress, t.gasUsed, t.transactionIndex,
			t.txType, t.txProtected, t.txGas, txGasPrice, txGasTipCap, txGasFeeCap, txValue, t.txNonce, t.txSize, t.tokenAddress, (*bigint.SQLBigIntBytes)(t.tokenID), t.txFrom, t.txTo)
		if err != nil {
			log.Error("can't save transfer", "b-hash", t.blockHash, "b-n", t.blockNumber, "a", t.address, "h", t.id)
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
