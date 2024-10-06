package ethclient

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"

	sq "github.com/Masterminds/squirrel"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/sqlite"
)

var ErrNotFound = errors.New("not found")

type EthClientStorageReader interface {
	GetBlockJSONByNumber(chainID uint64, blockNumber *big.Int, withTransactionDetails bool) (json.RawMessage, error)
	GetBlockJSONByHash(chainID uint64, blockHash common.Hash, withTransactionDetails bool) (json.RawMessage, error)
	GetBlockUncleJSONByHashAndIndex(chainID uint64, blockHash common.Hash, index uint64) (json.RawMessage, error)
	GetTransactionJSONByHash(chainID uint64, transactionHash common.Hash) (json.RawMessage, error)
	GetTransactionReceiptJSONByHash(chainID uint64, transactionHash common.Hash) (json.RawMessage, error)
}

type EthClientStorageWriter interface {
	PutBlockJSON(chainID uint64, blkJSON json.RawMessage, transactionDetailsFlag bool) error
	PutBlockUnclesJSON(chainID uint64, blockHash common.Hash, unclesJSON []json.RawMessage) error
	PutTransactionsJSON(chainID uint64, transactionsJSON []json.RawMessage) error
	PutTransactionReceiptsJSON(chainID uint64, receiptsJSON []json.RawMessage) error
}

type EthClientStorage interface {
	EthClientStorageReader
	EthClientStorageWriter
}

type DB struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *DB {
	return &DB{db: db}
}

func (b *DB) GetBlockJSONByNumber(chainID uint64, blockNumber *big.Int, withTransactionDetails bool) (json.RawMessage, error) {
	q := sq.Select("block_json").
		From("blockchain_data_blocks").
		Where(sq.Eq{"chain_id": chainID, "block_number": (*bigint.SQLBigInt)(blockNumber), "with_transaction_details": withTransactionDetails})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	blockJSON := json.RawMessage{}

	err = b.db.QueryRow(query, args...).Scan(&blockJSON)
	if err != nil {
		return nil, err
	}

	return blockJSON, nil
}

func (b *DB) GetBlockJSONByHash(chainID uint64, blockHash common.Hash, withTransactionDetails bool) (json.RawMessage, error) {
	q := sq.Select("block_json").
		From("blockchain_data_blocks").
		Where(sq.Eq{"chain_id": chainID, "block_hash": blockHash, "with_transaction_details": withTransactionDetails})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	blockJSON := json.RawMessage{}

	err = b.db.QueryRow(query, args...).Scan(&blockJSON)
	if err != nil {
		return nil, err
	}

	return blockJSON, nil
}

func (b *DB) GetBlockUncleJSONByHashAndIndex(chainID uint64, blockHash common.Hash, index uint64) (json.RawMessage, error) {
	q := sq.Select("block_uncle_json").
		From("blockchain_data_block_uncles").
		Where(sq.Eq{"chain_id": chainID, "block_hash": blockHash, "uncle_index": index})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	uncleJSON := json.RawMessage{}

	err = b.db.QueryRow(query, args...).Scan(&uncleJSON)
	if err != nil {
		return nil, err
	}

	return uncleJSON, nil
}

func (b *DB) GetTransactionJSONByHash(chainID uint64, transactionHash common.Hash) (json.RawMessage, error) {
	q := sq.Select("transaction_json").
		From("blockchain_data_transactions").
		Where(sq.Eq{"chain_id": chainID, "transaction_hash": transactionHash})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	transactionJSON := json.RawMessage{}

	err = b.db.QueryRow(query, args...).Scan(&transactionJSON)
	if err != nil {
		return nil, err
	}

	return transactionJSON, nil
}

func (b *DB) GetTransactionReceiptJSONByHash(chainID uint64, transactionHash common.Hash) (json.RawMessage, error) {
	q := sq.Select("receipt_json").
		From("blockchain_data_receipts").
		Where(sq.Eq{"chain_id": chainID, "transaction_hash": transactionHash})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	receiptJSON := json.RawMessage{}

	err = b.db.QueryRow(query, args...).Scan(&receiptJSON)
	if err != nil {
		return nil, err
	}

	return receiptJSON, nil
}

func (b *DB) PutBlockJSON(chainID uint64, blkJSON json.RawMessage, transactionDetailsFlag bool) (err error) {
	var tx *sql.Tx
	tx, err = b.db.Begin()
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

	err = putBlockJSON(tx, chainID, blkJSON, transactionDetailsFlag)

	return
}
func (b *DB) PutBlockUnclesJSON(chainID uint64, blockHash common.Hash, unclesJSON []json.RawMessage) (err error) {
	var tx *sql.Tx
	tx, err = b.db.Begin()
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

	for index, uncleJSON := range unclesJSON {
		err = putBlockUncleJSON(tx, chainID, blockHash, uint64(index), uncleJSON)
		if err != nil {
			return
		}
	}

	return
}

func (b *DB) PutTransactionsJSON(chainID uint64, transactionsJSON []json.RawMessage) (err error) {
	var tx *sql.Tx
	tx, err = b.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	for _, transactionJSON := range transactionsJSON {
		err = putTransactionJSON(tx, chainID, transactionJSON)
		if err != nil {
			return
		}
	}

	return
}

func (b *DB) PutTransactionReceiptsJSON(chainID uint64, receiptsJSON []json.RawMessage) (err error) {
	var tx *sql.Tx
	tx, err = b.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	for _, receiptJSON := range receiptsJSON {
		err = putReceiptJSON(tx, chainID, receiptJSON)
		if err != nil {
			return
		}
	}
	return
}

func putBlockJSON(creator sqlite.StatementCreator, chainID uint64, blkJSON json.RawMessage, transactionDetailsFlag bool) error {
	var rpcBlock rpcBlock
	if err := json.Unmarshal(blkJSON, &rpcBlock); err != nil {
		return err
	}

	if rpcBlock.Number == nil {
		// Pending block, don't store
		return nil
	}

	q := sq.Replace("blockchain_data_blocks").
		SetMap(sq.Eq{"chain_id": chainID, "block_number": (*bigint.SQLBigInt)(rpcBlock.Number.ToInt()), "block_hash": rpcBlock.Hash, "with_transaction_details": transactionDetailsFlag,
			"block_json": blkJSON,
		})

	query, args, err := q.ToSql()
	if err != nil {
		return err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)

	return err
}

func putBlockUncleJSON(creator sqlite.StatementCreator, chainID uint64, blockHash common.Hash, index uint64, uncleJSON json.RawMessage) error {
	q := sq.Replace("blockchain_data_block_uncles").
		SetMap(sq.Eq{"chain_id": chainID, "block_hash": blockHash, "uncle_index": index,
			"block_uncle_json": uncleJSON,
		})

	query, args, err := q.ToSql()
	if err != nil {
		return err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)

	return err
}

func putTransactionJSON(creator sqlite.StatementCreator, chainID uint64, txJSON json.RawMessage) error {
	var rpcTransaction rpcTransaction
	if err := json.Unmarshal(txJSON, &rpcTransaction); err != nil {
		return err
	}

	if rpcTransaction.BlockNumber == nil {
		// Pending transaction, don't store
		return nil
	}

	q := sq.Replace("blockchain_data_transactions").
		SetMap(sq.Eq{"chain_id": chainID, "transaction_hash": rpcTransaction.tx.Hash(),
			"transaction_json": txJSON,
		})

	query, args, err := q.ToSql()
	if err != nil {
		return err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)

	return err
}

func putReceiptJSON(creator sqlite.StatementCreator, chainID uint64, receiptJSON json.RawMessage) error {
	var receipt types.Receipt
	if err := json.Unmarshal(receiptJSON, &receipt); err != nil {
		return err
	}

	q := sq.Replace("blockchain_data_receipts").
		SetMap(sq.Eq{"chain_id": chainID, "transaction_hash": receipt.TxHash,
			"receipt_json": receiptJSON,
		})

	query, args, err := q.ToSql()
	if err != nil {
		return err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)

	return err
}
