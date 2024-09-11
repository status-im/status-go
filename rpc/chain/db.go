package chain

import (
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type DB struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *DB {
	return &DB{db: db}
}

func (b *DB) GetBlockByNumber(chainID uint64, blockNumber *big.Int) (*types.Block, error) {
	row := b.db.QueryRow("SELECT block_json FROM blockchain_data_blocks WHERE chain_id = ? AND block_number = ?", chainID, blockNumber)
	var block types.Block
	err := row.Scan(&block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (b *DB) GetBlockByHash(chainID uint64, blockHash common.Hash) (*types.Block, error) {
	row := b.db.QueryRow("SELECT block_json FROM blockchain_data_blocks WHERE chain_id = ? AND block_hash = ?", chainID, blockHash)
	var block types.Block
	err := row.Scan(&block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (b *DB) GetBlockHeaderByNumber(chainID uint64, blockNumber *big.Int) (*types.Header, error) {
	row := b.db.QueryRow("SELECT block_json FROM blockchain_data_blocks WHERE chain_id = ? AND block_number = ?", chainID, blockNumber)
	var blockHeader types.Header
	err := row.Scan(&blockHeader)
	if err != nil {
		return nil, err
	}
	return &blockHeader, nil
}

func (b *DB) GetBlockHeaderByHash(chainID uint64, blockHash common.Hash) (*types.Header, error) {
	row := b.db.QueryRow("SELECT block_json FROM blockchain_data_blocks WHERE chain_id = ? AND block_hash = ?", chainID, blockHash)
	var blockHeader types.Header
	err := row.Scan(&blockHeader)
	if err != nil {
		return nil, err
	}
	return &blockHeader, nil
}

func (b *DB) PutBlock(chainID uint64, block *types.Block) error {
	_, err := b.db.Exec("INSERT INTO blockchain_data_blocks (chain_id, block_number, block_hash, block_header_json, block_json) VALUES (?, ?, ?, ?, ?)", chainID, block.Number(), block.Hash(), block.Header(), block)
	if err != nil {
		return err
	}
	return b.PutTransactions(chainID, block.Transactions())
}

func (b *DB) PutBlockHeader(chainID uint64, blockHeader *types.Header) error {
	_, err := b.db.Exec("INSERT INTO blockchain_data_blocks (chain_id, block_number, block_hash, block_header_json) VALUES (?, ?, ?, ?)", chainID, blockHeader.Number, blockHeader.Hash(), blockHeader)
	if err != nil {
		return err
	}
	return nil
}

func (t *DB) GetTransactionsByBlockHash(chainID uint64, blockHash common.Hash) (types.Transactions, error) {
	rows, err := t.db.Query("SELECT transaction_json FROM blockchain_data_transactions WHERE chain_id = ? AND block_hash = ?", chainID, blockHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions types.Transactions
	for rows.Next() {
		var transaction types.Transaction
		err := rows.Scan(&transaction)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, nil
}

func (t *DB) GetTransactionsByBlockNumber(chainID uint64, blockNumber *big.Int) (types.Transactions, error) {
	rows, err := t.db.Query("SELECT transaction_json FROM blockchain_data_transactions WHERE chain_id = ? AND block_number = ?", chainID, blockNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions types.Transactions
	for rows.Next() {
		var transaction types.Transaction
		err := rows.Scan(&transaction)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, nil
}

func (t *DB) GetTransactionByHash(chainID uint64, transactionHash common.Hash) (*types.Transaction, error) {
	row := t.db.QueryRow("SELECT transaction_json FROM blockchain_data_transactions WHERE chain_id = ? AND transaction_hash = ?", chainID, transactionHash)
	var transaction types.Transaction
	err := row.Scan(&transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (t *DB) PutTransactions(chainID uint64, transactions types.Transactions) error {
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO blockchain_data_transactions (chain_id, transaction_hash, transaction_json) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, transaction := range transactions {
		_, err = stmt.Exec(chainID, transaction.Hash(), transaction)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (t *DB) GetTransactionReceipt(chainID uint64, transactionHash common.Hash) (*types.Receipt, error) {
	row := t.db.QueryRow("SELECT receipt_json FROM blockchain_data_transactions_receipts WHERE chain_id = ? AND transaction_hash = ?", chainID, transactionHash)
	var receipt types.Receipt
	err := row.Scan(&receipt)
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

func (t *DB) PutTransactionReceipt(chainID uint64, receipt *types.Receipt) error {
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO blockchain_data_transactions_receipts (chain_id, transaction_hash, receipt_json) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(chainID, receipt.TxHash, receipt)
	if err != nil {
		return err
	}

	return tx.Commit()
}
