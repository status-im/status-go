package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

const multiTransactionColumns = "id, from_network_id, from_tx_hash, from_address, from_asset, from_amount, to_network_id, to_tx_hash, to_address, to_asset, to_amount, type, cross_tx_id, timestamp"
const selectMultiTransactionColumns = "id, COALESCE(from_network_id, 0), from_tx_hash, from_address, from_asset, from_amount, COALESCE(to_network_id, 0), to_tx_hash, to_address, to_asset, to_amount, type, cross_tx_id, timestamp"

var pendingTxTimeout time.Duration = 10 * time.Minute
var ErrWatchPendingTxTimeout = errors.New("timeout watching for pending transaction")
var ErrPendingTxNotExists = errors.New("pending transaction does not exist")

func (tm *TransactionManager) InsertMultiTransaction(multiTransaction *MultiTransaction) (wallet_common.MultiTransactionIDType, error) {
	return multiTransaction.ID, tm.storage.CreateMultiTransaction(multiTransaction)
}

func (tm *TransactionManager) UpdateMultiTransaction(multiTransaction *MultiTransaction) error {
	return tm.storage.UpdateMultiTransaction(multiTransaction)
}

func (tm *TransactionManager) CreateMultiTransactionFromCommand(command *MultiTransactionCommand,
	data []*pathprocessor.MultipathProcessorTxArgs) (*MultiTransaction, error) {

	multiTransaction := multiTransactionFromCommand(command)

	// Extract network from args
	switch multiTransaction.Type {
	case MultiTransactionSend, MultiTransactionApprove, MultiTransactionSwap:
		if multiTransaction.FromNetworkID == wallet_common.UnknownChainID && len(data) == 1 {
			multiTransaction.FromNetworkID = data[0].ChainID
		}
	case MultiTransactionBridge:
		if len(data) == 1 && data[0].HopTx != nil {
			if multiTransaction.FromNetworkID == wallet_common.UnknownChainID {
				multiTransaction.FromNetworkID = data[0].HopTx.ChainID
			}
			if multiTransaction.ToNetworkID == wallet_common.UnknownChainID {
				multiTransaction.ToNetworkID = data[0].HopTx.ChainIDTo
			}
		}
	default:
		return nil, fmt.Errorf("unsupported multi transaction type: %v", multiTransaction.Type)
	}

	return multiTransaction, nil
}

func (tm *TransactionManager) SendTransactionForSigningToKeycard(ctx context.Context, multiTransaction *MultiTransaction, data []*pathprocessor.MultipathProcessorTxArgs, pathProcessors map[string]pathprocessor.PathProcessor) error {
	acc, err := tm.accountsDB.GetAccountByAddress(types.Address(multiTransaction.FromAddress))
	if err != nil {
		return err
	}

	kp, err := tm.accountsDB.GetKeypairByKeyUID(acc.KeyUID)
	if err != nil {
		return err
	}

	if !kp.MigratedToKeycard() {
		return fmt.Errorf("account being used is not migrated to a keycard, password is required")
	}

	tm.multiTransactionForKeycardSigning = multiTransaction
	tm.multipathTransactionsData = data
	hashes, err := tm.buildTransactions(pathProcessors)
	if err != nil {
		return err
	}

	signal.SendWalletEvent(signal.SignTransactions, hashes)

	return nil
}

func (tm *TransactionManager) SendTransactions(ctx context.Context, multiTransaction *MultiTransaction, data []*pathprocessor.MultipathProcessorTxArgs, pathProcessors map[string]pathprocessor.PathProcessor, account *account.SelectedExtKey) (*MultiTransactionCommandResult, error) {
	updateDataFromMultiTx(data, multiTransaction)
	hashes, err := sendTransactions(data, pathProcessors, account)
	if err != nil {
		return nil, err
	}

	return &MultiTransactionCommandResult{
		ID:     int64(multiTransaction.ID),
		Hashes: hashes,
	}, nil
}

func (tm *TransactionManager) ProceedWithTransactionsSignatures(ctx context.Context, signatures map[string]SignatureDetails) (*MultiTransactionCommandResult, error) {
	if err := addSignaturesToTransactions(tm.transactionsForKeycardSigning, signatures); err != nil {
		return nil, err
	}

	// send transactions
	hashes := make(map[uint64][]types.Hash)
	for _, desc := range tm.transactionsForKeycardSigning {
		txWithSignature, err := tm.transactor.AddSignatureToTransaction(desc.chainID, desc.builtTx, desc.signature)
		if err != nil {
			return nil, err
		}

		hash, err := tm.transactor.SendTransactionWithSignature(desc.from, tm.multiTransactionForKeycardSigning.FromAsset, tm.multiTransactionForKeycardSigning.ID, txWithSignature)
		if err != nil {
			return nil, err // TODO: One of transfers within transaction could have been sent. Need to notify user about it
		}
		hashes[desc.chainID] = append(hashes[desc.chainID], hash)
	}

	_, err := tm.InsertMultiTransaction(tm.multiTransactionForKeycardSigning)
	if err != nil {
		logutils.ZapLogger().Error("failed to insert multi transaction", zap.Error(err))
	}

	return &MultiTransactionCommandResult{
		ID:     int64(tm.multiTransactionForKeycardSigning.ID),
		Hashes: hashes,
	}, nil
}

func (tm *TransactionManager) GetMultiTransactions(ctx context.Context, ids []wallet_common.MultiTransactionIDType) ([]*MultiTransaction, error) {
	return tm.storage.ReadMultiTransactions(&MultiTxDetails{IDs: ids})
}

func (tm *TransactionManager) GetBridgeOriginMultiTransaction(ctx context.Context, toChainID uint64, crossTxID string) (*MultiTransaction, error) {
	details := NewMultiTxDetails()
	details.ToChainID = toChainID
	details.CrossTxID = crossTxID

	multiTxs, err := tm.storage.ReadMultiTransactions(details)
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
	details := NewMultiTxDetails()
	details.ToChainID = toChainID
	details.CrossTxID = crossTxID

	multiTxs, err := tm.storage.ReadMultiTransactions(details)
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

func (tm *TransactionManager) WatchTransaction(ctx context.Context, chainID uint64, transactionHash common.Hash) error {
	// Workaround to keep the blocking call until the clients use the PendingTxTracker APIs
	eventChan := make(chan walletevent.Event, 2)
	sub := tm.eventFeed.Subscribe(eventChan)
	defer sub.Unsubscribe()

	status, err := tm.pendingTracker.Watch(ctx, wallet_common.ChainID(chainID), transactionHash)
	if err == nil && *status != transactions.Pending {
		logutils.ZapLogger().Error("transaction is not pending", zap.String("status", *status))
		return nil
	}

	for {
		select {
		case we := <-eventChan:
			if transactions.EventPendingTransactionStatusChanged == we.Type {
				var p transactions.StatusChangedPayload
				err = json.Unmarshal([]byte(we.Message), &p)
				if err != nil {
					return err
				}
				if p.ChainID == wallet_common.ChainID(chainID) && p.Hash == transactionHash {
					signal.SendWalletEvent(signal.TransactionStatusChanged, p)
					return nil
				}
			}
		case <-time.After(pendingTxTimeout):
			return ErrWatchPendingTxTimeout
		}
	}
}
