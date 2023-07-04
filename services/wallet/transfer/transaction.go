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
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/bridge"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/transactions"
)

type TransactionManager struct {
	db          *sql.DB
	gethManager *account.GethManager
	transactor  *transactions.Transactor
	config      *params.NodeConfig
	accountsDB  *accounts.Database
}

func NewTransactionManager(db *sql.DB, gethManager *account.GethManager, transactor *transactions.Transactor,
	config *params.NodeConfig, accountsDB *accounts.Database) *TransactionManager {
	return &TransactionManager{
		db:          db,
		gethManager: gethManager,
		transactor:  transactor,
		config:      config,
		accountsDB:  accountsDB,
	}
}

type MultiTransactionType uint8

// TODO: extend with know types
const (
	MultiTransactionSend = iota
	MultiTransactionSwap
	MultiTransactionBridge
)

type MultiTransaction struct {
	ID          uint                 `json:"id"`
	Timestamp   uint64               `json:"timestamp"`
	FromAddress common.Address       `json:"fromAddress"`
	ToAddress   common.Address       `json:"toAddress"`
	FromAsset   string               `json:"fromAsset"`
	ToAsset     string               `json:"toAsset"`
	FromAmount  *hexutil.Big         `json:"fromAmount"`
	ToAmount    *hexutil.Big         `json:"toAmount"`
	Type        MultiTransactionType `json:"type"`
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

type PendingTrxType string

const (
	RegisterENS                   PendingTrxType = "RegisterENS"
	ReleaseENS                    PendingTrxType = "ReleaseENS"
	SetPubKey                     PendingTrxType = "SetPubKey"
	BuyStickerPack                PendingTrxType = "BuyStickerPack"
	WalletTransfer                PendingTrxType = "WalletTransfer"
	CollectibleDeployment         PendingTrxType = "CollectibleDeployment"
	CollectibleAirdrop            PendingTrxType = "CollectibleAirdrop"
	CollectibleRemoteSelfDestruct PendingTrxType = "CollectibleRemoteSelfDestruct"
	CollectibleBurn               PendingTrxType = "CollectibleBurn"
)

type PendingTransaction struct {
	Hash               common.Hash            `json:"hash"`
	Timestamp          uint64                 `json:"timestamp"`
	Value              bigint.BigInt          `json:"value"`
	From               common.Address         `json:"from"`
	To                 common.Address         `json:"to"`
	Data               string                 `json:"data"`
	Symbol             string                 `json:"symbol"`
	GasPrice           bigint.BigInt          `json:"gasPrice"`
	GasLimit           bigint.BigInt          `json:"gasLimit"`
	Type               PendingTrxType         `json:"type"`
	AdditionalData     string                 `json:"additionalData"`
	ChainID            uint64                 `json:"network_id"`
	MultiTransactionID MultiTransactionIDType `json:"multi_transaction_id"`
}

type TransactionIdentity struct {
	ChainID wallet_common.ChainID `json:"chainId"`
	Hash    common.Hash           `json:"hash"`
	Address common.Address        `json:"address"`
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

func (tm *TransactionManager) AddPending(transaction PendingTransaction) error {
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
	return err
}

func (tm *TransactionManager) DeletePending(chainID uint64, hash common.Hash) error {
	_, err := tm.db.Exec(`DELETE FROM pending_transactions WHERE network_id = ? AND hash = ?`, chainID, hash)
	return err
}

func (tm *TransactionManager) Watch(ctx context.Context, transactionHash common.Hash, client *chain.ClientWithFallback) error {
	watchTxCommand := &watchTransactionCommand{
		hash:   transactionHash,
		client: client,
	}

	commandContext, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	return watchTxCommand.Command()(commandContext)
}

const multiTransactionColumns = "from_address, from_asset, from_amount, to_address, to_asset, to_amount, type, timestamp"

func insertMultiTransaction(db *sql.DB, multiTransaction *MultiTransaction) (MultiTransactionIDType, error) {
	insert, err := db.Prepare(fmt.Sprintf(`INSERT OR REPLACE INTO multi_transactions (%s)
											VALUES(?, ?, ?, ?, ?, ?, ?, ?)`, multiTransactionColumns))
	if err != nil {
		return 0, err
	}
	result, err := insert.Exec(
		multiTransaction.FromAddress,
		multiTransaction.FromAsset,
		multiTransaction.FromAmount.String(),
		multiTransaction.ToAddress,
		multiTransaction.ToAsset,
		multiTransaction.ToAmount.String(),
		multiTransaction.Type,
		time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	defer insert.Close()
	multiTransactionID, err := result.LastInsertId()
	return MultiTransactionIDType(multiTransactionID), err
}

func (tm *TransactionManager) InsertMultiTransaction(multiTransaction *MultiTransaction) (MultiTransactionIDType, error) {
	return insertMultiTransaction(tm.db, multiTransaction)
}

func (tm *TransactionManager) CreateMultiTransactionFromCommand(ctx context.Context, command *MultiTransactionCommand, data []*bridge.TransactionBridge, bridges map[string]bridge.Bridge, password string) (*MultiTransactionCommandResult, error) {
	multiTransaction := &MultiTransaction{
		FromAddress: command.FromAddress,
		ToAddress:   command.ToAddress,
		FromAsset:   command.FromAsset,
		ToAsset:     command.ToAsset,
		FromAmount:  command.FromAmount,
		ToAmount:    new(hexutil.Big),
		Type:        command.Type,
	}

	selectedAccount, err := tm.getVerifiedWalletAccount(multiTransaction.FromAddress.Hex(), password)
	if err != nil {
		return nil, err
	}

	multiTransactionID, err := insertMultiTransaction(tm.db, multiTransaction)
	if err != nil {
		return nil, err
	}

	hashes := make(map[uint64][]types.Hash)
	for _, tx := range data {
		hash, err := bridges[tx.BridgeName].Send(tx, selectedAccount)
		if err != nil {
			return nil, err
		}
		pendingTransaction := PendingTransaction{
			Hash:               common.Hash(hash),
			Timestamp:          uint64(time.Now().Unix()),
			Value:              bigint.BigInt{Int: multiTransaction.FromAmount.ToInt()},
			From:               common.Address(tx.From()),
			To:                 common.Address(tx.To()),
			Data:               tx.Data().String(),
			Type:               WalletTransfer,
			ChainID:            tx.ChainID,
			MultiTransactionID: multiTransactionID,
			Symbol:             multiTransaction.FromAsset,
		}
		err = tm.AddPending(pendingTransaction)
		if err != nil {
			return nil, err
		}
		hashes[tx.ChainID] = append(hashes[tx.ChainID], hash)
	}

	return &MultiTransactionCommandResult{
		ID:     int64(multiTransactionID),
		Hashes: hashes,
	}, nil
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

	var multiTransactions []*MultiTransaction
	for rows.Next() {
		multiTransaction := &MultiTransaction{}
		var fromAmountDB, toAmountDB sql.NullString
		err := rows.Scan(
			&multiTransaction.ID,
			&multiTransaction.FromAddress,
			&multiTransaction.FromAsset,
			&fromAmountDB,
			&multiTransaction.ToAddress,
			&multiTransaction.ToAsset,
			&toAmountDB,
			&multiTransaction.Type,
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
