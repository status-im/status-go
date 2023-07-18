package transactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/rpcfilters"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	// PendingTransactionUpdate is emitted when a pending transaction is updated (added or deleted)
	EventPendingTransactionUpdate walletevent.EventType = "pending-transaction-update"
)

type TransactionManager struct {
	db             *sql.DB
	pendingTxEvent rpcfilters.ChainEvent
	eventFeed      *event.Feed
	quit           chan struct{}
}

func NewTransactionManager(db *sql.DB, pendingTxEvent rpcfilters.ChainEvent, eventFeed *event.Feed) *TransactionManager {
	return &TransactionManager{
		db:             db,
		eventFeed:      eventFeed,
		pendingTxEvent: pendingTxEvent,
	}
}

func (tm *TransactionManager) Start() error {
	if tm.quit != nil {
		return errors.New("latest transaction sent to upstream event is already started")
	}

	tm.quit = make(chan struct{})

	go func() {
		_, chi := tm.pendingTxEvent.Subscribe()
		ch, ok := chi.(chan *rpcfilters.PendingTxInfo)
		if !ok {
			panic("pendingTxEvent returned wront type of channel")
		}

		for {
			select {
			case tx := <-ch:
				log.Info("Pending transaction event received", tx)
				err := tm.AddPending(&PendingTransaction{
					Hash:      tx.Hash,
					Timestamp: uint64(time.Now().Unix()),
					From:      tx.From,
					ChainID:   tx.ChainID,
				})
				if err != nil {
					log.Error("Failed to add pending transaction", "error", err, "hash", tx.Hash,
						"chainID", tx.ChainID)
				}
			case <-tm.quit:
				return
			}
		}
	}()

	return tm.pendingTxEvent.Start()
}

func (tm *TransactionManager) Stop() {
	if tm.quit == nil {
		return
	}

	select {
	case <-tm.quit:
		return
	default:
		close(tm.quit)
	}

	tm.quit = nil
}

type PendingTrxType string

const (
	RegisterENS               PendingTrxType = "RegisterENS"
	ReleaseENS                PendingTrxType = "ReleaseENS"
	SetPubKey                 PendingTrxType = "SetPubKey"
	BuyStickerPack            PendingTrxType = "BuyStickerPack"
	WalletTransfer            PendingTrxType = "WalletTransfer"
	DeployCommunityToken      PendingTrxType = "DeployCommunityToken"
	AirdropCommunityToken     PendingTrxType = "AirdropCommunityToken"
	RemoteDestructCollectible PendingTrxType = "RemoteDestructCollectible"
	BurnCommunityToken        PendingTrxType = "BurnCommunityToken"
	DeployOwnerToken          PendingTrxType = "DeployOwnerToken"
)

type PendingTransaction struct {
	Hash               common.Hash    `json:"hash"`
	Timestamp          uint64         `json:"timestamp"`
	Value              bigint.BigInt  `json:"value"`
	From               common.Address `json:"from"`
	To                 common.Address `json:"to"`
	Data               string         `json:"data"`
	Symbol             string         `json:"symbol"`
	GasPrice           bigint.BigInt  `json:"gasPrice"`
	GasLimit           bigint.BigInt  `json:"gasLimit"`
	Type               PendingTrxType `json:"type"`
	AdditionalData     string         `json:"additionalData"`
	ChainID            uint64         `json:"network_id"`
	MultiTransactionID int64          `json:"multi_transaction_id"`
}

const selectFromPending = `SELECT hash, timestamp, value, from_address, to_address, data,
								symbol, gas_price, gas_limit, type, additional_data,
								network_id, COALESCE(multi_transaction_id, 0)
							FROM pending_transactions
							`

func rowsToTransactions(rows *sql.Rows) (transactions []*PendingTransaction, err error) {
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
			&transaction.MultiTransactionID,
		)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, transaction)
	}
	return transactions, nil
}

func (tm *TransactionManager) GetAllPending(chainIDs []uint64) ([]*PendingTransaction, error) {
	log.Info("Getting all pending transactions", "chainIDs", chainIDs)

	if len(chainIDs) == 0 {
		return nil, errors.New("at least 1 chainID is required")
	}

	inVector := strings.Repeat("?, ", len(chainIDs)-1) + "?"
	var parameters []interface{}
	for _, c := range chainIDs {
		parameters = append(parameters, c)
	}

	rows, err := tm.db.Query(fmt.Sprintf(selectFromPending+"WHERE network_id in (%s)", inVector), parameters...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToTransactions(rows)
}

func (tm *TransactionManager) GetPendingByAddress(chainIDs []uint64, address common.Address) ([]*PendingTransaction, error) {
	log.Info("Getting pending transaction by address", "chainIDs", chainIDs, "address", address)

	if len(chainIDs) == 0 {
		return nil, errors.New("at least 1 chainID is required")
	}

	inVector := strings.Repeat("?, ", len(chainIDs)-1) + "?"
	var parameters []interface{}
	for _, c := range chainIDs {
		parameters = append(parameters, c)
	}

	parameters = append(parameters, address)

	rows, err := tm.db.Query(fmt.Sprintf(selectFromPending+"WHERE network_id in (%s) AND from_address = ?", inVector), parameters...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToTransactions(rows)
}

// GetPendingEntry returns sql.ErrNoRows if no pending transaction is found for the given identity
// TODO: consider using address also in case we expect to have also for the receiver
func (tm *TransactionManager) GetPendingEntry(chainID uint64, hash common.Hash) (*PendingTransaction, error) {
	log.Info("Getting pending transaction", "chainID", chainID, "hash", hash)

	row := tm.db.QueryRow(`SELECT timestamp, value, from_address, to_address, data,
								symbol, gas_price, gas_limit, type, additional_data,
								network_id, COALESCE(multi_transaction_id, 0)
							FROM pending_transactions
							WHERE network_id = ? AND hash = ?`, chainID, hash)
	transaction := &PendingTransaction{
		Hash:     hash,
		Value:    bigint.BigInt{Int: new(big.Int)},
		GasPrice: bigint.BigInt{Int: new(big.Int)},
		GasLimit: bigint.BigInt{Int: new(big.Int)},
		ChainID:  chainID,
	}
	err := row.Scan(
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
		&transaction.MultiTransactionID,
	)
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (tm *TransactionManager) AddPending(transaction *PendingTransaction) error {
	insert, err := tm.db.Prepare(`INSERT OR REPLACE INTO pending_transactions
                                      (network_id, hash, timestamp, value, from_address, to_address,
                                       data, symbol, gas_price, gas_limit, type, additional_data, multi_transaction_id)
                                      VALUES
                                      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
		transaction.MultiTransactionID,
	)
	// Notify listeners of new pending transaction (used in activity history)
	if err == nil {
		tm.notifyPendingTransactionListeners(transaction.ChainID, []common.Address{transaction.From, transaction.To}, transaction.Timestamp)
	}
	return err
}

func (tm *TransactionManager) notifyPendingTransactionListeners(chainID uint64, addresses []common.Address, timestamp uint64) {
	if tm.eventFeed != nil {
		tm.eventFeed.Send(walletevent.Event{
			Type:     EventPendingTransactionUpdate,
			ChainID:  chainID,
			Accounts: addresses,
			At:       int64(timestamp),
		})
	}
}

func (tm *TransactionManager) deletePending(chainID uint64, hash common.Hash) error {
	tx, err := tm.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRow(`SELECT from_address, to_address, timestamp FROM pending_transactions WHERE network_id = ? AND hash = ?`, chainID, hash)
	var from, to common.Address
	var timestamp uint64
	err = row.Scan(&from, &to, &timestamp)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM pending_transactions WHERE network_id = ? AND hash = ?`, chainID, hash)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err == nil {
		tm.notifyPendingTransactionListeners(chainID, []common.Address{from, to}, timestamp)
	}
	return err
}

func (tm *TransactionManager) Watch(ctx context.Context, transactionHash common.Hash, client *chain.ClientWithFallback) error {
	log.Info("Watching transaction", "chainID", client.ChainID, "hash", transactionHash)

	watchTxCommand := &watchTransactionCommand{
		hash:   transactionHash,
		client: client,
	}

	commandContext, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	err := watchTxCommand.Command()(commandContext)
	if err != nil {
		log.Error("watchTxCommand error", "error", err, "chainID", client.ChainID, "hash", transactionHash)
		return err
	}

	return tm.deletePending(client.ChainID, transactionHash)
}

type watchTransactionCommand struct {
	client *chain.ClientWithFallback
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
