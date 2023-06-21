package transfer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/bridge"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/transactions"
)

type MultiTransactionIDType int64

const (
	NoMultiTransactionID = MultiTransactionIDType(0)
)

type TransactionManager struct {
	db             *sql.DB
	gethManager    *account.GethManager
	transactor     *transactions.Transactor
	config         *params.NodeConfig
	accountsDB     *accounts.Database
	pendingManager *transactions.TransactionManager
}

func NewTransactionManager(db *sql.DB, gethManager *account.GethManager, transactor *transactions.Transactor,
	config *params.NodeConfig, accountsDB *accounts.Database,
	pendingTxManager *transactions.TransactionManager) *TransactionManager {

	return &TransactionManager{
		db:             db,
		gethManager:    gethManager,
		transactor:     transactor,
		config:         config,
		accountsDB:     accountsDB,
		pendingManager: pendingTxManager,
	}
}

var (
	emptyHash = common.Hash{}
)

type MultiTransactionType uint8

const (
	MultiTransactionSend = iota
	MultiTransactionSwap
	MultiTransactionBridge
)

type MultiTransaction struct {
	ID            uint                 `json:"id"`
	Timestamp     uint64               `json:"timestamp"`
	FromNetworkID uint64               `json:"fromNetworkID"`
	ToNetworkID   uint64               `json:"toNetworkID"`
	FromTxHash    common.Hash          `json:"fromTxHash"`
	ToTxHash      common.Hash          `json:"toTxHash"`
	FromAddress   common.Address       `json:"fromAddress"`
	ToAddress     common.Address       `json:"toAddress"`
	FromAsset     string               `json:"fromAsset"`
	ToAsset       string               `json:"toAsset"`
	FromAmount    *hexutil.Big         `json:"fromAmount"`
	ToAmount      *hexutil.Big         `json:"toAmount"`
	Type          MultiTransactionType `json:"type"`
	CrossTxID     string
}

type MultiTransactionCommand struct {
	FromAddress common.Address       `json:"fromAddress"`
	ToAddress   common.Address       `json:"toAddress"`
	FromAsset   string               `json:"fromAsset"`
	ToAsset     string               `json:"toAsset"`
	FromAmount  *hexutil.Big         `json:"fromAmount"`
	Type        MultiTransactionType `json:"type"`
}

type MultiTransactionCommandResult struct {
	ID     int64                   `json:"id"`
	Hashes map[uint64][]types.Hash `json:"hashes"`
}

type TransactionIdentity struct {
	ChainID wallet_common.ChainID `json:"chainId"`
	Hash    common.Hash           `json:"hash"`
	Address common.Address        `json:"address"`
}

const multiTransactionColumns = "from_network_id, from_tx_hash, from_address, from_asset, from_amount, to_network_id, to_tx_hash, to_address, to_asset, to_amount, type, cross_tx_id, timestamp"

func rowsToMultiTransactions(rows *sql.Rows) ([]*MultiTransaction, error) {
	var multiTransactions []*MultiTransaction
	for rows.Next() {
		multiTransaction := &MultiTransaction{}
		var fromAmountDB, toAmountDB sql.NullString
		err := rows.Scan(
			&multiTransaction.ID,
			&multiTransaction.FromNetworkID,
			&multiTransaction.FromTxHash,
			&multiTransaction.FromAddress,
			&multiTransaction.FromAsset,
			&fromAmountDB,
			&multiTransaction.ToNetworkID,
			&multiTransaction.ToTxHash,
			&multiTransaction.ToAddress,
			&multiTransaction.ToAsset,
			&toAmountDB,
			&multiTransaction.Type,
			&multiTransaction.CrossTxID,
			&multiTransaction.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if fromAmountDB.Valid {
			multiTransaction.FromAmount = new(hexutil.Big)
			if _, ok := (*big.Int)(multiTransaction.FromAmount).SetString(fromAmountDB.String, 0); !ok {
				return nil, errors.New("failed to convert fromAmountDB.String to big.Int: " + fromAmountDB.String)
			}
		}

		if toAmountDB.Valid {
			multiTransaction.ToAmount = new(hexutil.Big)
			if _, ok := (*big.Int)(multiTransaction.ToAmount).SetString(toAmountDB.String, 0); !ok {
				return nil, errors.New("failed to convert fromAmountDB.String to big.Int: " + toAmountDB.String)
			}
		}

		multiTransactions = append(multiTransactions, multiTransaction)
	}

	return multiTransactions, nil
}

func insertMultiTransaction(db *sql.DB, multiTransaction *MultiTransaction) (MultiTransactionIDType, error) {
	insert, err := db.Prepare(fmt.Sprintf(`INSERT INTO multi_transactions (%s)
											VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, multiTransactionColumns))
	if err != nil {
		return NoMultiTransactionID, err
	}

	timestamp := time.Now().Unix()
	result, err := insert.Exec(
		multiTransaction.FromNetworkID,
		multiTransaction.FromTxHash,
		multiTransaction.FromAddress,
		multiTransaction.FromAsset,
		multiTransaction.FromAmount.String(),
		multiTransaction.ToNetworkID,
		multiTransaction.ToTxHash,
		multiTransaction.ToAddress,
		multiTransaction.ToAsset,
		multiTransaction.ToAmount.String(),
		multiTransaction.Type,
		multiTransaction.CrossTxID,
		timestamp,
	)
	if err != nil {
		return NoMultiTransactionID, err
	}
	defer insert.Close()
	multiTransactionID, err := result.LastInsertId()

	multiTransaction.Timestamp = uint64(timestamp)
	multiTransaction.ID = uint(multiTransactionID)

	return MultiTransactionIDType(multiTransactionID), err
}

func (tm *TransactionManager) InsertMultiTransaction(multiTransaction *MultiTransaction) (MultiTransactionIDType, error) {
	return insertMultiTransaction(tm.db, multiTransaction)
}

func updateMultiTransaction(db *sql.DB, multiTransaction *MultiTransaction) error {
	if MultiTransactionIDType(multiTransaction.ID) == NoMultiTransactionID {
		return fmt.Errorf("no multitransaction ID")
	}

	update, err := db.Prepare(fmt.Sprintf(`REPLACE INTO multi_transactions (rowid, %s)
	VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, multiTransactionColumns))

	if err != nil {
		return err
	}
	_, err = update.Exec(
		multiTransaction.ID,
		multiTransaction.FromNetworkID,
		multiTransaction.FromTxHash,
		multiTransaction.FromAddress,
		multiTransaction.FromAsset,
		multiTransaction.FromAmount.String(),
		multiTransaction.ToNetworkID,
		multiTransaction.ToTxHash,
		multiTransaction.ToAddress,
		multiTransaction.ToAsset,
		multiTransaction.ToAmount.String(),
		multiTransaction.Type,
		multiTransaction.CrossTxID,
		time.Now().Unix(),
	)
	if err != nil {
		return err
	}
	return update.Close()
}

func (tm *TransactionManager) UpdateMultiTransaction(multiTransaction *MultiTransaction) error {
	return updateMultiTransaction(tm.db, multiTransaction)
}

func (tm *TransactionManager) CreateMultiTransactionFromCommand(ctx context.Context, command *MultiTransactionCommand,
	data []*bridge.TransactionBridge, bridges map[string]bridge.Bridge, password string) (*MultiTransactionCommandResult, error) {

	multiTransaction := multiTransactionFromCommand(command)

	multiTransactionID, err := insertMultiTransaction(tm.db, multiTransaction)
	if err != nil {
		return nil, err
	}

	multiTransaction.ID = uint(multiTransactionID)
	hashes, err := tm.sendTransactions(multiTransaction, data, bridges, password)
	if err != nil {
		return nil, err
	}

	err = tm.storePendingTransactions(multiTransaction, hashes, data)
	if err != nil {
		return nil, err
	}

	return &MultiTransactionCommandResult{
		ID:     int64(multiTransactionID),
		Hashes: hashes,
	}, nil
}

func (tm *TransactionManager) storePendingTransactions(multiTransaction *MultiTransaction,
	hashes map[uint64][]types.Hash, data []*bridge.TransactionBridge) error {

	txs := createPendingTransactions(hashes, data, multiTransaction)
	for _, tx := range txs {
		err := tm.pendingManager.AddPending(tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func createPendingTransactions(hashes map[uint64][]types.Hash, data []*bridge.TransactionBridge,
	multiTransaction *MultiTransaction) []*transactions.PendingTransaction {

	txs := make([]*transactions.PendingTransaction, 0)
	for _, tx := range data {
		for _, hash := range hashes[tx.ChainID] {
			pendingTransaction := &transactions.PendingTransaction{
				Hash:               common.Hash(hash),
				Timestamp:          uint64(time.Now().Unix()),
				Value:              bigint.BigInt{Int: multiTransaction.FromAmount.ToInt()},
				From:               common.Address(tx.From()),
				To:                 common.Address(tx.To()),
				Data:               tx.Data().String(),
				Type:               transactions.WalletTransfer,
				ChainID:            tx.ChainID,
				MultiTransactionID: int64(multiTransaction.ID),
				Symbol:             multiTransaction.FromAsset,
			}
			txs = append(txs, pendingTransaction)
		}
	}
	return txs
}

func multiTransactionFromCommand(command *MultiTransactionCommand) *MultiTransaction {

	log.Info("Creating multi transaction", "command", command)

	multiTransaction := &MultiTransaction{
		FromAddress: command.FromAddress,
		ToAddress:   command.ToAddress,
		FromAsset:   command.FromAsset,
		ToAsset:     command.ToAsset,
		FromAmount:  command.FromAmount,
		ToAmount:    new(hexutil.Big),
		Type:        command.Type,
	}

	return multiTransaction
}

func (tm *TransactionManager) sendTransactions(multiTransaction *MultiTransaction,
	data []*bridge.TransactionBridge, bridges map[string]bridge.Bridge, password string) (
	map[uint64][]types.Hash, error) {

	log.Info("Making transactions", "multiTransaction", multiTransaction)

	selectedAccount, err := tm.getVerifiedWalletAccount(multiTransaction.FromAddress.Hex(), password)
	if err != nil {
		return nil, err
	}

	hashes := make(map[uint64][]types.Hash)
	for _, tx := range data {
		hash, err := bridges[tx.BridgeName].Send(tx, selectedAccount)
		if err != nil {
			return nil, err
		}
		hashes[tx.ChainID] = append(hashes[tx.ChainID], hash)
	}
	return hashes, nil
}

func (tm *TransactionManager) GetMultiTransactions(ctx context.Context, ids []MultiTransactionIDType) ([]*MultiTransaction, error) {
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, v := range ids {
		placeholders[i] = "?"
		args[i] = v
	}

	stmt, err := tm.db.Prepare(fmt.Sprintf(`SELECT rowid, %s
											FROM multi_transactions
											WHERE rowid in (%s)`,
		multiTransactionColumns,
		strings.Join(placeholders, ",")))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToMultiTransactions(rows)
}

func (tm *TransactionManager) getBridgeMultiTransactions(ctx context.Context, toChainID uint64, crossTxID string) ([]*MultiTransaction, error) {
	stmt, err := tm.db.Prepare(fmt.Sprintf(`SELECT rowid, %s
											FROM multi_transactions
											WHERE type=? AND to_network_id=? AND cross_tx_id=?`,
		multiTransactionColumns))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(MultiTransactionBridge, toChainID, crossTxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToMultiTransactions(rows)
}

func (tm *TransactionManager) GetBridgeOriginMultiTransaction(ctx context.Context, toChainID uint64, crossTxID string) (*MultiTransaction, error) {
	multiTxs, err := tm.getBridgeMultiTransactions(ctx, toChainID, crossTxID)
	if err != nil {
		return nil, err
	}

	for _, multiTx := range multiTxs {
		// Origin MultiTxs will have a missing "ToTxHash"
		if multiTx.ToTxHash == emptyHash {
			return multiTx, nil
		}
	}

	return nil, nil
}

func (tm *TransactionManager) GetBridgeDestinationMultiTransaction(ctx context.Context, toChainID uint64, crossTxID string) (*MultiTransaction, error) {
	multiTxs, err := tm.getBridgeMultiTransactions(ctx, toChainID, crossTxID)
	if err != nil {
		return nil, err
	}

	for _, multiTx := range multiTxs {
		// Destination MultiTxs will have a missing "FromTxHash"
		if multiTx.FromTxHash == emptyHash {
			return multiTx, nil
		}
	}

	return nil, nil
}

func (tm *TransactionManager) getVerifiedWalletAccount(address, password string) (*account.SelectedExtKey, error) {
	exists, err := tm.accountsDB.AddressExists(types.HexToAddress(address))
	if err != nil {
		log.Error("failed to query db for a given address", "address", address, "error", err)
		return nil, err
	}

	if !exists {
		log.Error("failed to get a selected account", "err", transactions.ErrInvalidTxSender)
		return nil, transactions.ErrAccountDoesntExist
	}

	key, err := tm.gethManager.VerifyAccountPassword(tm.config.KeyStoreDir, address, password)
	if err != nil {
		log.Error("failed to verify account", "account", address, "error", err)
		return nil, err
	}

	return &account.SelectedExtKey{
		Address:    key.Address,
		AccountKey: key,
	}, nil
}
