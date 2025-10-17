package pendingtxtracker

//go:generate go tool mockgen -package=mock_pendingtxtracker -source=txstatusfetcher.go -destination=mock/txstatusfetcher.go

import (
	"context"
	"encoding/json"
	"time"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"

	"github.com/status-im/status-go/rpc"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/common"
)

type TxStatusRes struct {
	Status ac.TxStatus
	Hash   eth.Hash
}

type TxStatusFetcher interface {
	FetchTxStatus(ctx context.Context, chainID common.ChainID, hashes []eth.Hash) ([]TxStatusRes, error)
}

type BatchTxStatusFetcher struct {
	ethClientGetter rpc.EthClientGetter
	logger          *zap.Logger
}

func NewBatchTxStatusFetcher(ethClientGetter rpc.EthClientGetter, logger *zap.Logger) *BatchTxStatusFetcher {
	return &BatchTxStatusFetcher{
		ethClientGetter: ethClientGetter,
		logger:          logger,
	}
}

// fetchBatchTxStatus returns not pending transactions (confirmed or errored)
// it excludes the still pending or errored request from the result
func (f *BatchTxStatusFetcher) FetchTxStatus(ctx context.Context, chainID common.ChainID, hashes []eth.Hash) ([]TxStatusRes, error) {
	chainClient, err := f.ethClientGetter.EthClient(uint64(chainID))
	if err != nil {
		f.logger.Error("Failed to get chain client", zap.Error(err))
		return nil, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	batch := make([]ethrpc.BatchElem, 0, len(hashes))
	for _, hash := range hashes {
		batch = append(batch, ethrpc.BatchElem{
			Method: GetTransactionReceiptRPCName,
			Args:   []interface{}{hash},
			Result: new(nullableReceipt),
		})
	}

	err = chainClient.BatchCallContext(reqCtx, batch)
	if err != nil {
		f.logger.Error("Transactions request fail", zap.Error(err))
		return nil, err
	}

	res := make([]TxStatusRes, 0, len(batch))
	for i, b := range batch {
		err := b.Error
		if err != nil {
			f.logger.Error("Failed to get transaction", zap.Stringer("hash", hashes[i]), zap.Error(err))
			continue
		}

		if b.Result == nil {
			f.logger.Error("Transaction not found", zap.Stringer("hash", hashes[i]))
			continue
		}

		receiptWrapper, ok := b.Result.(*nullableReceipt)
		if !ok {
			f.logger.Error("Failed to cast transaction receipt", zap.Stringer("hash", hashes[i]))
			continue
		}

		if receiptWrapper == nil || receiptWrapper.Receipt == nil {
			// the transaction is not available yet
			continue
		}

		receipt := receiptWrapper.Receipt
		isPending := receipt != nil && receipt.BlockNumber == nil
		if !isPending {
			var status ac.TxStatus
			if receipt.Status == types.ReceiptStatusSuccessful {
				status = ac.Success
			} else {
				status = ac.Failed
			}
			res = append(res, TxStatusRes{
				Hash:   hashes[i],
				Status: status,
			})
		}
	}
	return res, nil
}

type nullableReceipt struct {
	*types.Receipt
}

func (nr *nullableReceipt) UnmarshalJSON(data []byte) error {
	transactionNotAvailable := (string(data) == "null")
	if transactionNotAvailable {
		return nil
	}
	return json.Unmarshal(data, &nr.Receipt)
}
