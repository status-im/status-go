package transfer

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/rpc/chain"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/token"
)

// TODO: Find proper way to uniquely match Origin and Destination transactions (some sort of hash or uniqueID)
// Current approach is not failsafe (for example, if multiple identical bridge operations are triggered
// at the same time)
// Recipient + Relayer + Data should match in both Origin and Destination transactions
func getHopBridgeCrossTxID(recipient common.Address, relayer common.Address, logData []byte) string {
	return fmt.Sprintf("%s_%s_%s", recipient.String(), relayer.String(), hex.EncodeToString(logData))
}

func buildHopBridgeMultitransaction(ctx context.Context, client *chain.ClientWithFallback, transactionManager *TransactionManager, tokenManager *token.Manager, subTx *Transfer) (*MultiTransaction, error) {
	// Identify if it's from/to transaction
	switch w_common.GetEventType(subTx.Log) {
	case w_common.HopBridgeTransferSentToL2EventType:
		// L1-L2 Origin transaciton
		fromChainID := subTx.NetworkID
		fromTxHash := subTx.Receipt.TxHash

		toChainID, recipient, relayer, fromAmount, err := w_common.ParseHopBridgeTransferSentToL2Log(subTx.Log)
		if err != nil {
			return nil, err
		}
		crossTxID := getHopBridgeCrossTxID(recipient, relayer, subTx.Log.Data)

		// Try to find "destination" half of the multiTx
		multiTx, err := transactionManager.GetBridgeDestinationMultiTransaction(ctx, toChainID, crossTxID)
		if err != nil {
			return nil, err
		}

		if multiTx == nil {
			multiTx = &MultiTransaction{
				// Data from "origin" transaction
				FromNetworkID: fromChainID,
				FromTxHash:    fromTxHash,
				FromAddress:   subTx.From,
				FromAsset:     "ETH",
				FromAmount:    (*hexutil.Big)(fromAmount),
				ToNetworkID:   toChainID,
				ToAddress:     recipient,
				// To be replaced by "destination" transaction, need to be non-null
				ToAmount: (*hexutil.Big)(fromAmount),
				// Common data
				Type:      MultiTransactionBridge,
				CrossTxID: crossTxID,
			}

			_, err := transactionManager.InsertMultiTransaction(multiTx)
			if err != nil {
				return nil, err
			}

		} else {
			multiTx.FromNetworkID = fromChainID
			multiTx.FromTxHash = fromTxHash
			multiTx.FromAddress = subTx.From
			multiTx.FromAsset = "ETH"
			multiTx.FromAmount = (*hexutil.Big)(fromAmount)

			err := transactionManager.UpdateMultiTransaction(multiTx)
			if err != nil {
				return nil, err
			}
		}
		return multiTx, nil

	case w_common.HopBridgeTransferFromL1CompletedEventType:
		// L1-L2 Destination transaciton
		toChainID := subTx.NetworkID
		toTxHash := subTx.Receipt.TxHash

		recipient, relayer, toAmount, err := w_common.ParseHopBridgeTransferFromL1CompletedLog(subTx.Log)
		if err != nil {
			return nil, err
		}
		crossTxID := getHopBridgeCrossTxID(recipient, relayer, subTx.Log.Data)

		// Try to find "origin" half of the multiTx
		multiTx, err := transactionManager.GetBridgeOriginMultiTransaction(ctx, toChainID, crossTxID)
		if err != nil {
			return nil, err
		}

		if multiTx == nil {
			multiTx = &MultiTransaction{
				// To be replaced by "origin" transaction, need to be non-null
				FromAddress: recipient,
				FromAsset:   "ETH",
				FromAmount:  (*hexutil.Big)(toAmount),
				// Data from "destination" transaction
				ToNetworkID: toChainID,
				ToTxHash:    toTxHash,
				ToAddress:   recipient,
				ToAsset:     "ETH",
				ToAmount:    (*hexutil.Big)(toAmount),
				// Common data
				Type:      MultiTransactionBridge,
				CrossTxID: crossTxID,
			}

			_, err := transactionManager.InsertMultiTransaction(multiTx)
			if err != nil {
				return nil, err
			}
		} else {
			multiTx.ToTxHash = toTxHash
			multiTx.ToAsset = "ETH"
			multiTx.ToAmount = (*hexutil.Big)(toAmount)

			err := transactionManager.UpdateMultiTransaction(multiTx)
			if err != nil {
				return nil, err
			}
		}
		return multiTx, nil
	}
	return nil, nil
}
