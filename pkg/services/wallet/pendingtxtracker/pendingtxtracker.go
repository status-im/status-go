package pendingtxtracker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/logutils"

	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	"github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/responses"
	"github.com/status-im/status-go/pkg/services/wallet/routeexecution/storage"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

const (
	// EventPendingTransactionUpdate is emitted when a pending transaction is updated (added or deleted). Carries PendingTxUpdatePayload in message
	EventPendingTransactionUpdate walletevent.EventType = "pending-transaction-update"
	// EventPendingTransactionStatusChanged carries StatusChangedPayload in message
	EventPendingTransactionStatusChanged walletevent.EventType = "pending-transaction-status-changed"

	PendingCheckInterval = 10 * time.Second

	GetTransactionReceiptRPCName = "eth_getTransactionReceipt"
)

var (
	ErrStillPending = errors.New("transaction is still pending")
)

type AutoDeleteType = bool

const (
	AutoDelete AutoDeleteType = true
	Keep       AutoDeleteType = false
)

type TxIdentity struct {
	ChainID common.ChainID `json:"chainId"`
	Hash    eth.Hash       `json:"hash"`
}

type TxDetails struct {
	SendDetails      *responses.SendDetails             `json:"sendDetails"`
	SentTransactions []*responses.RouterSentTransaction `json:"sentTransactions"`
}

type PendingTxUpdatePayload struct {
	TxIdentity
	TxDetails
	Deleted bool `json:"deleted"`
}

type StatusChangedPayload struct {
	TxIdentity
	TxDetails
	Status ac.TxStatus `json:"status"`
}

// PendingTxTracker implements StatusService in common/status_node_service.go
type PendingTxTracker struct {
	db                    *sql.DB
	routeExecutionStorage *storage.DB
	trackedTxDB           *DB
	txStatusFetcher       TxStatusFetcher

	eventFeed *event.Feed

	taskRunner *ConditionalRepeater
	logger     *zap.Logger
}

func NewPendingTxTracker(db *sql.DB, txStatusFetcher TxStatusFetcher, eventFeed *event.Feed, checkInterval time.Duration) *PendingTxTracker {
	tm := &PendingTxTracker{
		db:                    db,
		routeExecutionStorage: storage.NewDB(db),
		trackedTxDB:           NewDB(db),
		txStatusFetcher:       txStatusFetcher,
		eventFeed:             eventFeed,
		logger:                logutils.ZapLogger().Named("PendingTxTracker"),
	}
	tm.taskRunner = NewConditionalRepeater(checkInterval, func(ctx context.Context) bool {
		return tm.fetchAndUpdateDB(ctx)
	})
	return tm
}

func (tm *PendingTxTracker) fetchAndUpdateDB(ctx context.Context) bool {
	res := WorkNotDone

	txs, err := tm.GetAllPending()
	if err != nil {
		tm.logger.Error("Failed to get pending transactions", zap.Error(err))
		return WorkDone
	}
	tm.logger.Debug("Checking for PT status", zap.Int("count", len(txs)))

	txsMap := make(map[common.ChainID][]eth.Hash)
	for _, tx := range txs {
		chainID := tx.ChainID
		txsMap[chainID] = append(txsMap[chainID], tx.Hash)
	}

	doneCount := 0
	// Batch request for each chain
	for chainID, txs := range txsMap {
		tm.logger.Debug("Processing PTs", zap.Stringer("chainID", chainID), zap.Int("count", len(txs)))
		batchRes, err := tm.txStatusFetcher.FetchTxStatus(ctx, chainID, txs)
		if err != nil {
			tm.logger.Error("Failed to batch fetch pending transactions status for", zap.Stringer("chainID", chainID), zap.Error(err))
			continue
		}
		if len(batchRes) == 0 {
			tm.logger.Debug("No change to PTs status", zap.Stringer("chainID", chainID))
			continue
		}
		tm.logger.Debug("PTs done", zap.Stringer("chainID", chainID), zap.Int("count", len(batchRes)))
		doneCount += len(batchRes)

		updateRes, err := tm.updateDBStatus(ctx, chainID, batchRes)
		if err != nil {
			tm.logger.Error("Failed to update pending transactions status for", zap.Stringer("chainID", chainID), zap.Error(err))
			continue
		}

		tm.logger.Debug("Emit notifications for PTs", zap.Stringer("chainID", chainID), zap.Int("count", len(updateRes)))
		tm.emitNotifications(chainID, updateRes)
	}

	if len(txs) == doneCount {
		res = WorkDone
	}

	tm.logger.Debug("Done PTs iteration", zap.Int("count", doneCount), zap.Bool("completed", res))

	return res
}

// updateDBStatus returns entries that were updated only
func (tm *PendingTxTracker) updateDBStatus(ctx context.Context, chainID common.ChainID, statuses []TxStatusRes) ([]TxStatusRes, error) {
	for _, br := range statuses {
		err := tm.trackedTxDB.UpdateTxStatus(
			TxIdentity{
				ChainID: chainID,
				Hash:    br.Hash,
			},
			br.Status,
		)
		if err != nil {
			tm.logger.Error("Failed to update trackedTx status", zap.Stringer("hash", br.Hash), zap.Error(err))
			continue
		}
	}

	res := make([]TxStatusRes, 0, len(statuses))
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	updateStmt, err := tx.PrepareContext(ctx, `UPDATE pending_transactions SET status = ? WHERE network_id = ? AND hash = ?`)
	if err != nil {
		rollErr := tx.Rollback()
		if rollErr != nil {
			err = fmt.Errorf("failed to rollback transaction due to: %w", err)
		}
		return nil, fmt.Errorf("failed to prepare update statement: %w", err)
	}

	checkAutoDelStmt, err := tx.PrepareContext(ctx, `SELECT auto_delete FROM pending_transactions WHERE network_id = ? AND hash = ?`)
	if err != nil {
		rollErr := tx.Rollback()
		if rollErr != nil {
			err = fmt.Errorf("failed to rollback transaction: %w", err)
		}
		return nil, fmt.Errorf("failed to prepare auto delete statement: %w", err)
	}

	notifyFunctions := make([]func(), 0, len(statuses))
	for _, br := range statuses {
		row := checkAutoDelStmt.QueryRowContext(ctx, chainID, br.Hash)
		var autoDel bool
		err = row.Scan(&autoDel)
		if err != nil {
			if err == sql.ErrNoRows {
				tm.logger.Warn("Missing entry while checking for auto_delete", zap.Stringer("hash", br.Hash))
			} else {
				tm.logger.Error("Failed to retrieve auto_delete for pending transaction", zap.Stringer("hash", br.Hash), zap.Error(err))
			}
			continue
		}

		if autoDel {
			notifyFn, err := tm.deleteBySQLTx(tx, chainID, br.Hash)
			if err != nil && err != ErrStillPending {
				tm.logger.Error("Failed to delete pending transaction", zap.Stringer("hash", br.Hash), zap.Error(err))
				continue
			}
			notifyFunctions = append(notifyFunctions, notifyFn)
		} else {
			// If the entry was not deleted, update the status
			txStatus := br.Status

			res, err := updateStmt.ExecContext(ctx, txStatus, chainID, br.Hash)
			if err != nil {
				tm.logger.Error("Failed to update pending transaction status", zap.Stringer("hash", br.Hash), zap.Error(err))
				continue
			}
			affected, err := res.RowsAffected()
			if err != nil {
				tm.logger.Error("Failed to get updated rows", zap.Stringer("hash", br.Hash), zap.Error(err))
				continue
			}

			if affected == 0 {
				tm.logger.Warn("Missing entry to update for", zap.Stringer("hash", br.Hash))
				continue
			}
		}

		res = append(res, br)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	for _, fn := range notifyFunctions {
		fn()
	}

	return res, nil
}

func (tm *PendingTxTracker) updateTxDetails(txDetails *TxDetails, chainID uint64, txHash cryptotypes.Hash) {
	if txDetails == nil {
		txDetails = &TxDetails{}
	}
	txDetails.SendDetails = &responses.SendDetails{}
	routeData, err := tm.routeExecutionStorage.GetRouteDataByHash(chainID, txHash)
	if err != nil {
		tm.logger.Warn("Missing tx data ", zap.Stringer("hash", txHash), zap.Error(err))
	}
	if routeData != nil {
		if routeData.RouteInputParams != nil {
			txDetails.SendDetails.UpdateFields(*routeData.RouteInputParams, routeData.RouteInputParams.FromChainID, routeData.RouteInputParams.ToChainID)
		}

		for _, pd := range routeData.PathsData {
			if pd.IsApprovalPlaced() && pd.ApprovalTxData.SentHash == txHash {
				txDetails.SentTransactions = append(txDetails.SentTransactions, responses.NewRouterSentTransaction(
					pd.ApprovalTxData.TxArgs,
					pd.ApprovalTxData.SentHash,
					true))
			}
			if pd.IsTxPlaced() && pd.TxData.SentHash == txHash {
				txDetails.SentTransactions = append(txDetails.SentTransactions, responses.NewRouterSentTransaction(
					pd.TxData.TxArgs,
					pd.TxData.SentHash,
					false))
			}
		}
	}
}

func (tm *PendingTxTracker) emitNotifications(chainID common.ChainID, changes []TxStatusRes) {
	if tm.eventFeed != nil {
		for _, change := range changes {
			payload := StatusChangedPayload{
				TxIdentity: TxIdentity{
					ChainID: chainID,
					Hash:    change.Hash,
				},
				Status: change.Status,
			}

			tm.updateTxDetails(&payload.TxDetails, chainID.ToUint(), cryptotypes.Hash(change.Hash))

			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				tm.logger.Error("Failed to marshal pending transaction status", zap.Stringer("hash", change.Hash), zap.Error(err))
				continue
			}
			tm.eventFeed.Send(walletevent.Event{
				Type:    EventPendingTransactionStatusChanged,
				ChainID: uint64(chainID),
				Message: string(jsonPayload),
			})
		}
	}
}

func (tm *PendingTxTracker) Start() error {
	tm.taskRunner.RunUntilDone()
	return nil
}

// APIs returns a list of new APIs.
func (tm *PendingTxTracker) APIs() []ethrpc.API {
	return []ethrpc.API{
		{
			Namespace: "pending",
			Version:   "0.1.0",
			Service:   tm,
			Public:    true,
		},
	}
}

func (tm *PendingTxTracker) Stop() error {
	tm.taskRunner.Stop()
	return nil
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
	SetSignerPublicKey        PendingTrxType = "SetSignerPublicKey"
	WalletConnectTransfer     PendingTrxType = "WalletConnectTransfer"
)

type PendingTransaction struct {
	Hash           eth.Hash       `json:"hash"`
	Timestamp      uint64         `json:"timestamp"`
	Value          bigint.BigInt  `json:"value"`
	From           eth.Address    `json:"from"`
	To             eth.Address    `json:"to"`
	Data           string         `json:"data"`
	Symbol         string         `json:"symbol"`
	GasPrice       bigint.BigInt  `json:"gasPrice"`
	GasLimit       bigint.BigInt  `json:"gasLimit"`
	Type           PendingTrxType `json:"type"`
	AdditionalData string         `json:"additionalData"`
	ChainID        common.ChainID `json:"network_id"`
	Nonce          uint64         `json:"nonce"`

	// nil will insert the default value (Pending) in DB
	Status *ac.TxStatus `json:"status,omitempty"`
	// nil will insert the default value (true) in DB
	AutoDelete *bool `json:"autoDelete,omitempty"`
}

func (ptx *PendingTransaction) Identity() TxIdentity {
	return TxIdentity{
		ChainID: ptx.ChainID,
		Hash:    ptx.Hash,
	}
}

const selectFromPending = `SELECT hash, timestamp, value, from_address, to_address, data,
								symbol, gas_price, gas_limit, type, additional_data,
								network_id, status, auto_delete, nonce
							FROM pending_transactions
							`

func rowsToTransactions(rows *sql.Rows) (transactions []*PendingTransaction, err error) {
	for rows.Next() {
		transaction := &PendingTransaction{
			Value:    bigint.BigInt{Int: new(big.Int)},
			GasPrice: bigint.BigInt{Int: new(big.Int)},
			GasLimit: bigint.BigInt{Int: new(big.Int)},
		}

		transaction.Status = new(ac.TxStatus)
		transaction.AutoDelete = new(bool)
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
			transaction.Status,
			transaction.AutoDelete,
			&transaction.Nonce,
		)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, transaction)
	}
	return transactions, nil
}

func (tm *PendingTxTracker) GetAllPending() ([]*PendingTransaction, error) {
	if tm.db == nil {
		return nil, errors.New("database is not initialized")
	}
	rows, err := tm.db.Query(selectFromPending+"WHERE status = ?", ac.Pending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToTransactions(rows)
}

// StoreAndTrackPendingTx store the details of a pending transaction and track it until it is mined
func (tm *PendingTxTracker) StoreAndTrackPendingTx(transaction *PendingTransaction) error {
	err := tm.addPendingAndNotify(transaction)
	if err != nil {
		return err
	}

	tm.taskRunner.RunUntilDone()

	return err
}

// StoreAndTrackPendingTx store the details of multiple pending transactions and track them until they are mined.
// It is more efficient to call this once instead of calling StoreAndTrackPendingTx for each transaction
func (tm *PendingTxTracker) StoreAndTrackPendingTxs(transactions []*PendingTransaction) error {
	for _, transaction := range transactions {
		err := tm.addPendingAndNotify(transaction)
		if err != nil {
			return err
		}
	}

	tm.taskRunner.RunUntilDone()

	return nil
}

func (tm *PendingTxTracker) addPending(transaction *PendingTransaction) error {
	err := tm.trackedTxDB.PutTx(TrackedTx{
		ID: TxIdentity{
			ChainID: transaction.ChainID,
			Hash:    transaction.Hash,
		},
		Status:    ac.Pending,
		Timestamp: transaction.Timestamp,
	})
	if err != nil {
		return err
	}

	var notifyFn func()
	tx, err := tm.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			if notifyFn != nil {
				notifyFn()
			}
			return
		}
		_ = tx.Rollback()
	}()

	exists := true
	var hash eth.Hash

	err = tx.QueryRow(`
		SELECT hash
		FROM
			pending_transactions
		WHERE
			network_id = ?
		AND
			from_address = ?
		AND
			nonce = ?
		`,
		transaction.ChainID,
		transaction.From,
		transaction.Nonce).
		Scan(&hash)
	if err != nil {
		if err == sql.ErrNoRows {
			exists = false
		} else {
			return err
		}
	}

	if exists {
		notifyFn, err = tm.deleteBySQLTx(tx, transaction.ChainID, hash)
		if err != nil && err != ErrStillPending {
			return err
		}
	}

	// TODO: maybe we should think of making (network_id, from_address, nonce) as primary key instead (network_id, hash) ????
	var insert *sql.Stmt
	insert, err = tx.Prepare(`INSERT OR REPLACE INTO pending_transactions
                                      (network_id, hash, timestamp, value, from_address, to_address,
                                       data, symbol, gas_price, gas_limit, type, additional_data, status,
																			 auto_delete, nonce)
                                      VALUES
                                      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insert.Close()

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
		transaction.Status,
		transaction.AutoDelete,
		transaction.Nonce,
	)
	return err
}

func (tm *PendingTxTracker) addPendingAndNotify(transaction *PendingTransaction) error {
	err := tm.addPending(transaction)

	// Notify listeners of new pending transaction (used in activity history)
	if err == nil {
		tm.notifyPendingTransactionListeners(PendingTxUpdatePayload{
			TxIdentity: TxIdentity{
				ChainID: transaction.ChainID,
				Hash:    transaction.Hash,
			},
			Deleted: false,
		}, []eth.Address{transaction.From, transaction.To}, transaction.Timestamp)
	}
	return err
}

func (tm *PendingTxTracker) notifyPendingTransactionListeners(payload PendingTxUpdatePayload, addresses []eth.Address, timestamp uint64) {
	tm.updateTxDetails(&payload.TxDetails, payload.ChainID.ToUint(), cryptotypes.Hash(payload.Hash))

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		tm.logger.Error("Failed to marshal PendingTxUpdatePayload", zap.Stringer("hash", payload.Hash), zap.Error(err))
		return
	}

	if tm.eventFeed != nil {
		tm.eventFeed.Send(walletevent.Event{
			Type:     EventPendingTransactionUpdate,
			ChainID:  uint64(payload.ChainID),
			Accounts: addresses,
			At:       int64(timestamp),
			Message:  string(jsonPayload),
		})
	}
}

// deleteBySQLTx returns ErrStillPending if the transaction is still pending
func (tm *PendingTxTracker) deleteBySQLTx(tx *sql.Tx, chainID common.ChainID, hash eth.Hash) (notify func(), err error) {
	row := tx.QueryRow(`SELECT from_address, to_address, timestamp, status FROM pending_transactions WHERE network_id = ? AND hash = ?`, chainID, hash)
	var from, to eth.Address
	var timestamp uint64
	var status ac.TxStatus
	err = row.Scan(&from, &to, &timestamp, &status)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`DELETE FROM pending_transactions WHERE network_id = ? AND hash = ?`, chainID, hash)
	if err != nil {
		return nil, err
	}

	if err == nil && status == ac.Pending {
		err = ErrStillPending
	}
	return func() {
		tm.notifyPendingTransactionListeners(PendingTxUpdatePayload{
			TxIdentity: TxIdentity{
				ChainID: chainID,
				Hash:    hash,
			},
			Deleted: true,
		}, []eth.Address{from, to}, timestamp)
	}, err
}

func (tm *PendingTxTracker) IsRunning() bool {
	return tm.taskRunner.IsRunning()
}
