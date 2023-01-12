package wallet

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
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/bridge"
	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/transactions"
)

type TransactionManager struct {
	db          *sql.DB
	gethManager *account.GethManager
	transactor  *transactions.Transactor
	config      *params.NodeConfig
	accountsDB  *accounts.Database
}

type MultiTransactionType uint8

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
	Type        MultiTransactionType `json:"type"`
}

type MultiTransactionResult struct {
	ID     int64                   `json:"id"`
	Hashes map[uint64][]types.Hash `json:"hashes"`
}

type PendingTrxType string

const (
	RegisterENS           PendingTrxType = "RegisterENS"
	ReleaseENS            PendingTrxType = "ReleaseENS"
	SetPubKey             PendingTrxType = "SetPubKey"
	BuyStickerPack        PendingTrxType = "BuyStickerPack"
	WalletTransfer        PendingTrxType = "WalletTransfer"
	CollectibleDeployment PendingTrxType = "CollectibleDeployment"
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

func (tm *TransactionManager) getAllPendings(chainIDs []uint64) ([]*PendingTransaction, error) {
	if len(chainIDs) == 0 {
		return nil, errors.New("at least 1 chainID is required")
	}

	inVector := strings.Repeat("?, ", len(chainIDs)-1) + "?"
	var parameters []interface{}
	for _, c := range chainIDs {
		parameters = append(parameters, c)
	}

	rows, err := tm.db.Query(fmt.Sprintf(`SELECT hash, timestamp, value, from_address, to_address, data,
                                         symbol, gas_price, gas_limit, type, additional_data,
										 network_id
                                  FROM pending_transactions
                                  WHERE network_id in (%s)`, inVector), parameters...)

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

func (tm *TransactionManager) getPendingByAddress(chainIDs []uint64, address common.Address) ([]*PendingTransaction, error) {
	if len(chainIDs) == 0 {
		return nil, errors.New("at least 1 chainID is required")
	}

	inVector := strings.Repeat("?, ", len(chainIDs)-1) + "?"
	var parameters []interface{}
	for _, c := range chainIDs {
		parameters = append(parameters, c)
	}

	parameters = append(parameters, address)

	rows, err := tm.db.Query(fmt.Sprintf(`SELECT hash, timestamp, value, from_address, to_address, data,
                                         symbol, gas_price, gas_limit, type, additional_data,
										 network_id
                                  FROM pending_transactions
                                  WHERE network_id in (%s) AND from_address = ?`, inVector), parameters...)
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

func (tm *TransactionManager) createMultiTransaction(ctx context.Context, multiTransaction *MultiTransaction, data []*bridge.TransactionBridge, bridges map[string]bridge.Bridge, password string) (*MultiTransactionResult, error) {
	selectedAccount, err := tm.getVerifiedWalletAccount(multiTransaction.FromAddress.Hex(), password)
	if err != nil {
		return nil, err
	}

	insert, err := tm.db.Prepare(`INSERT OR REPLACE INTO multi_transactions
                                      (from_address, from_asset, from_amount, to_address, to_asset, type, timestamp)
                                      VALUES
                                      (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	result, err := insert.Exec(
		multiTransaction.FromAddress,
		multiTransaction.FromAsset,
		multiTransaction.FromAmount.String(),
		multiTransaction.ToAddress,
		multiTransaction.ToAsset,
		multiTransaction.Type,
		time.Now().Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer insert.Close()
	multiTransactionID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	hashes := make(map[uint64][]types.Hash)
	for _, tx := range data {
		hash, err := bridges[tx.BridgeName].Send(tx, selectedAccount)
		if err != nil {
			return nil, err
		}
		err = tm.addPending(PendingTransaction{
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
		})
		if err != nil {
			return nil, err
		}
		hashes[tx.ChainID] = append(hashes[tx.ChainID], hash)
	}

	return &MultiTransactionResult{
		ID:     multiTransactionID,
		Hashes: hashes,
	}, nil
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
