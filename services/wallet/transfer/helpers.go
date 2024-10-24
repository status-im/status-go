package transfer

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
)

func rowsToMultiTransactions(rows *sql.Rows) ([]*MultiTransaction, error) {
	var multiTransactions []*MultiTransaction
	for rows.Next() {
		multiTransaction := &MultiTransaction{}
		var fromAmountDB, toAmountDB sql.NullString
		var fromTxHash, toTxHash sql.RawBytes
		err := rows.Scan(
			&multiTransaction.ID,
			&multiTransaction.FromNetworkID,
			&fromTxHash,
			&multiTransaction.FromAddress,
			&multiTransaction.FromAsset,
			&fromAmountDB,
			&multiTransaction.ToNetworkID,
			&toTxHash,
			&multiTransaction.ToAddress,
			&multiTransaction.ToAsset,
			&toAmountDB,
			&multiTransaction.Type,
			&multiTransaction.CrossTxID,
			&multiTransaction.Timestamp,
		)
		if len(fromTxHash) > 0 {
			multiTransaction.FromTxHash = common.BytesToHash(fromTxHash)
		}
		if len(toTxHash) > 0 {
			multiTransaction.ToTxHash = common.BytesToHash(toTxHash)
		}
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

func addSignaturesToTransactions(transactions map[common.Hash]*TransactionDescription, signatures map[string]SignatureDetails) error {
	if len(transactions) == 0 {
		return errors.New("no transactions to proceed with")
	}
	if len(signatures) != len(transactions) {
		return errors.New("not all transactions have been signed")
	}

	// check if all transactions have been signed
	for hash, desc := range transactions {
		sigDetails, ok := signatures[hash.String()]
		if !ok {
			return fmt.Errorf("missing signature for transaction %s", hash)
		}

		rBytes, _ := hex.DecodeString(sigDetails.R)
		sBytes, _ := hex.DecodeString(sigDetails.S)
		vByte := byte(0)
		if sigDetails.V == "01" {
			vByte = 1
		}

		desc.signature = make([]byte, crypto.SignatureLength)
		copy(desc.signature[32-len(rBytes):32], rBytes)
		copy(desc.signature[64-len(rBytes):64], sBytes)
		desc.signature[64] = vByte
	}

	return nil
}

func multiTransactionFromCommand(command *MultiTransactionCommand) *MultiTransaction {
	toAmount := new(hexutil.Big)
	if command.ToAmount != nil {
		toAmount = command.ToAmount
	}
	multiTransaction := NewMultiTransaction(
		/* Timestamp:     */ uint64(time.Now().Unix()),
		/* FromNetworkID: */ 0,
		/* ToNetworkID:	  */ 0,
		/* FromTxHash:    */ common.Hash{},
		/* ToTxHash:      */ common.Hash{},
		/* FromAddress:   */ command.FromAddress,
		/* ToAddress:     */ command.ToAddress,
		/* FromAsset:     */ command.FromAsset,
		/* ToAsset:       */ command.ToAsset,
		/* FromAmount:    */ command.FromAmount,
		/* ToAmount:      */ toAmount,
		/* Type:		  */ command.Type,
		/* CrossTxID:	  */ "",
	)

	return multiTransaction
}

func updateDataFromMultiTx(data []*pathprocessor.MultipathProcessorTxArgs, multiTransaction *MultiTransaction) {
	for _, tx := range data {
		if tx.TransferTx != nil {
			tx.TransferTx.MultiTransactionID = multiTransaction.ID
			tx.TransferTx.Symbol = multiTransaction.FromAsset
		}
		if tx.HopTx != nil {
			tx.HopTx.MultiTransactionID = multiTransaction.ID
			tx.HopTx.Symbol = multiTransaction.FromAsset
		}
		if tx.CbridgeTx != nil {
			tx.CbridgeTx.MultiTransactionID = multiTransaction.ID
			tx.CbridgeTx.Symbol = multiTransaction.FromAsset
		}
		if tx.ERC721TransferTx != nil {
			tx.ERC721TransferTx.MultiTransactionID = multiTransaction.ID
			tx.ERC721TransferTx.Symbol = multiTransaction.FromAsset
		}
		if tx.ERC1155TransferTx != nil {
			tx.ERC1155TransferTx.MultiTransactionID = multiTransaction.ID
			tx.ERC1155TransferTx.Symbol = multiTransaction.FromAsset
		}
		if tx.SwapTx != nil {
			tx.SwapTx.MultiTransactionID = multiTransaction.ID
			tx.SwapTx.Symbol = multiTransaction.FromAsset
		}
	}
}

func sendTransactions(data []*pathprocessor.MultipathProcessorTxArgs, pathProcessors map[string]pathprocessor.PathProcessor, account *account.SelectedExtKey) (
	map[uint64][]types.Hash, error) {

	hashes := make(map[uint64][]types.Hash)
	usedNonces := make(map[uint64]int64)
	for _, tx := range data {

		lastUsedNonce := int64(-1)
		if nonce, ok := usedNonces[tx.ChainID]; ok {
			lastUsedNonce = nonce
		}

		hash, usedNonce, err := pathProcessors[tx.Name].Send(tx, lastUsedNonce, account)
		if err != nil {
			return nil, err // TODO: One of transfers within transaction could have been sent. Need to notify user about it
		}

		hashes[tx.ChainID] = append(hashes[tx.ChainID], hash)
		usedNonces[tx.ChainID] = int64(usedNonce)
	}
	return hashes, nil
}

func idFromTimestamp() wallet_common.MultiTransactionIDType {
	return wallet_common.MultiTransactionIDType(time.Now().UnixMilli())
}

var multiTransactionIDGenerator func() wallet_common.MultiTransactionIDType = idFromTimestamp

func (tm *TransactionManager) removeMultiTransactionByAddress(address common.Address) error {
	// We must not remove those transactions, where from_address and to_address are different and both are stored in accounts DB
	// and one of them is equal to the address, as we want to keep the records for the other address
	// That is why we don't use cascade delete here with references to transfers table, as we might have 2 records in multi_transactions
	// for the same transaction, one for each address

	details := NewMultiTxDetails()
	details.FromAddress = address
	mtxs, err := tm.storage.ReadMultiTransactions(details)

	ids := make([]wallet_common.MultiTransactionIDType, 0)
	for _, mtx := range mtxs {
		// Remove self transactions as well, leave only those where we have the counterparty in accounts DB
		if mtx.FromAddress != mtx.ToAddress {
			// If both addresses are stored in accounts DB, we don't remove the record
			var addressToCheck common.Address
			if mtx.FromAddress == address {
				addressToCheck = mtx.ToAddress
			} else {
				addressToCheck = mtx.FromAddress
			}
			counterpartyExists, err := tm.accountsDB.AddressExists(types.Address(addressToCheck))
			if err != nil {
				logutils.ZapLogger().Error("Failed to query accounts db for a given address",
					zap.Stringer("address", address),
					zap.Error(err),
				)
				continue
			}

			// Skip removal if counterparty is in accounts DB and removed address is not sender
			if counterpartyExists && address != mtx.FromAddress {
				continue
			}
		}

		ids = append(ids, mtx.ID)
	}

	if len(ids) > 0 {
		for _, id := range ids {
			err = tm.storage.DeleteMultiTransaction(id)
			if err != nil {
				logutils.ZapLogger().Error("Failed to delete multi transaction", zap.Int64("id", int64(id)), zap.Error(err))
			}
		}
	}

	return err
}
