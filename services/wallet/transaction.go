package wallet

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/chain"
)

type TransactionManager struct {
	db *sql.DB
}

type PendingTrxType string

const (
	RegisterENS    PendingTrxType = "RegisterENS"
	ReleaseENS     PendingTrxType = "ReleaseENS"
	SetPubKey      PendingTrxType = "SetPubKey"
	BuyStickerPack PendingTrxType = "BuyStickerPack"
	WalletTransfer PendingTrxType = "WalletTransfer"
)

type PendingTransaction struct {
	Hash           common.Hash    `json:"hash"`
	Timestamp      uint64         `json:"timestamp"`
	Value          bigint.BigInt  `json:"value"`
	From           common.Address `json:"from"`
	To             common.Address `json:"to"`
	Data           string         `json:"data"`
	Symbol         string         `json:"symbol"`
	GasPrice       bigint.BigInt  `json:"gasPrice"`
	GasLimit       bigint.BigInt  `json:"gasLimit"`
	Type           PendingTrxType `json:"type"`
	AdditionalData string         `json:"additionalData"`
	ChainID        uint64         `json:"network_id"`
}

func (tm *TransactionManager) getAllPendings(chainID uint64) ([]*PendingTransaction, error) {
	rows, err := tm.db.Query(`SELECT hash, timestamp, value, from_address, to_address, data,
                                         symbol, gas_price, gas_limit, type, additional_data,
										 network_id
                                  FROM pending_transactions
                                  WHERE network_id = ?`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*PendingTransaction
	for rows.Next() {
		transaction := &PendingTransaction{
			Value:    bigint.BigInt{Int: new(big.Int)},
			GasPrice: bigint.BigInt{Int: new(big.Int)},
			GasLimit: bigint.BigInt{Int: new(big.Int)},
		}
		err := rows.Scan(&transaction.Hash,
			&transaction.Timestamp,
			(*bigint.SQLBigIntBytes)(transaction.Value.Int),
			&transaction.From,
			&transaction.To,
			&transaction.Data,
			&transaction.Symbol,
			(*bigint.SQLBigIntBytes)(transaction.GasPrice.Int),
			(*bigint.SQLBigIntBytes)(transaction.GasLimit.Int),
			&transaction.Type,
			&transaction.AdditionalData,
			&transaction.ChainID,
		)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (tm *TransactionManager) getPendingByAddress(chainID uint64, address common.Address) ([]*PendingTransaction, error) {
	rows, err := tm.db.Query(`SELECT hash, timestamp, value, from_address, to_address, data,
                                         symbol, gas_price, gas_limit, type, additional_data,
										 network_id
                                  FROM pending_transactions
                                  WHERE network_id = ? AND from_address = ?`, chainID, address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*PendingTransaction
	for rows.Next() {
		transaction := &PendingTransaction{
			Value:    bigint.BigInt{Int: new(big.Int)},
			GasPrice: bigint.BigInt{Int: new(big.Int)},
			GasLimit: bigint.BigInt{Int: new(big.Int)},
		}
		err := rows.Scan(&transaction.Hash,
			&transaction.Timestamp,
			(*bigint.SQLBigIntBytes)(transaction.Value.Int),
			&transaction.From,
			&transaction.To,
			&transaction.Data,
			&transaction.Symbol,
			(*bigint.SQLBigIntBytes)(transaction.GasPrice.Int),
			(*bigint.SQLBigIntBytes)(transaction.GasLimit.Int),
			&transaction.Type,
			&transaction.AdditionalData,
			&transaction.ChainID,
		)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (tm *TransactionManager) addPending(transaction PendingTransaction) error {
	insert, err := tm.db.Prepare(`INSERT OR REPLACE INTO pending_transactions
                                      (network_id, hash, timestamp, value, from_address, to_address,
                                       data, symbol, gas_price, gas_limit, type, additional_data)
                                      VALUES
                                      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	_, err = insert.Exec(
		transaction.ChainID,
		transaction.Hash,
		transaction.Timestamp,
		(*bigint.SQLBigIntBytes)(transaction.Value.Int),
		transaction.From,
		transaction.To,
		transaction.Data,
		transaction.Symbol,
		(*bigint.SQLBigIntBytes)(transaction.GasPrice.Int),
		(*bigint.SQLBigIntBytes)(transaction.GasLimit.Int),
		transaction.Type,
		transaction.AdditionalData,
	)
	return err
}

func (tm *TransactionManager) deletePending(chainID uint64, hash common.Hash) error {
	_, err := tm.db.Exec(`DELETE FROM pending_transactions WHERE network_id = ? AND hash = ?`, chainID, hash)
	return err
}

func (tm *TransactionManager) watch(ctx context.Context, transactionHash common.Hash, client *chain.Client) error {
	watchTxCommand := &watchTransactionCommand{
		hash:   transactionHash,
		client: client,
	}

	commandContext, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	return watchTxCommand.Command()(commandContext)
}

type watchTransactionCommand struct {
	client *chain.Client
	hash   common.Hash
}

func (c *watchTransactionCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 10 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *watchTransactionCommand) Run(ctx context.Context) error {
	requestContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, isPending, err := c.client.TransactionByHash(requestContext, c.hash)

	if err != nil {
		log.Error("Watching transaction error", "error", err)
		return err
	}

	if isPending {
		return errors.New("transaction is pending")
	}

	return nil
}
