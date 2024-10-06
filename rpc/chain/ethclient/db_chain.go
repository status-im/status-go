package ethclient

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type EthClientChainStorageReader interface {
	GetBlockJSONByNumber(blockNumber *big.Int, withTransactionDetails bool) (json.RawMessage, error)
	GetBlockJSONByHash(blockHash common.Hash, withTransactionDetails bool) (json.RawMessage, error)
	GetBlockUncleJSONByHashAndIndex(blockHash common.Hash, index uint64) (json.RawMessage, error)
	GetTransactionJSONByHash(transactionHash common.Hash) (json.RawMessage, error)
	GetTransactionReceiptJSONByHash(transactionHash common.Hash) (json.RawMessage, error)
}

type EthClientChainStorageWriter interface {
	PutBlockJSON(blkJSON json.RawMessage, transactionDetailsFlag bool) error
	PutBlockUnclesJSON(blockHash common.Hash, unclesJSON []json.RawMessage) error
	PutTransactionsJSON(transactionsJSON []json.RawMessage) error
	PutTransactionReceiptsJSON(receiptsJSON []json.RawMessage) error
}

type EthClientChainStorage interface {
	EthClientChainStorageReader
	EthClientChainStorageWriter
}

type DBChain struct {
	s       EthClientStorage
	chainID uint64
}

func NewDBChain(s EthClientStorage, chainID uint64) *DBChain {
	return &DBChain{
		s:       s,
		chainID: chainID,
	}
}

func (b *DBChain) GetBlockJSONByNumber(blockNumber *big.Int, withTransactionDetails bool) (json.RawMessage, error) {
	return b.s.GetBlockJSONByNumber(b.chainID, blockNumber, withTransactionDetails)
}

func (b *DBChain) GetBlockJSONByHash(blockHash common.Hash, withTransactionDetails bool) (json.RawMessage, error) {
	return b.s.GetBlockJSONByHash(b.chainID, blockHash, withTransactionDetails)
}

func (b *DBChain) GetBlockUncleJSONByHashAndIndex(blockHash common.Hash, index uint64) (json.RawMessage, error) {
	return b.s.GetBlockUncleJSONByHashAndIndex(b.chainID, blockHash, index)
}

func (b *DBChain) GetTransactionJSONByHash(transactionHash common.Hash) (json.RawMessage, error) {
	return b.s.GetTransactionJSONByHash(b.chainID, transactionHash)
}

func (b *DBChain) GetTransactionReceiptJSONByHash(transactionHash common.Hash) (json.RawMessage, error) {
	return b.s.GetTransactionReceiptJSONByHash(b.chainID, transactionHash)
}

func (b *DBChain) PutBlockJSON(blkJSON json.RawMessage, transactionDetailsFlag bool) error {
	return b.s.PutBlockJSON(b.chainID, blkJSON, transactionDetailsFlag)
}

func (b *DBChain) PutBlockUnclesJSON(blockHash common.Hash, unclesJSON []json.RawMessage) error {
	return b.s.PutBlockUnclesJSON(b.chainID, blockHash, unclesJSON)
}

func (b *DBChain) PutTransactionsJSON(transactionsJSON []json.RawMessage) error {
	return b.s.PutTransactionsJSON(b.chainID, transactionsJSON)
}

func (b *DBChain) PutTransactionReceiptsJSON(receiptsJSON []json.RawMessage) error {
	return b.s.PutTransactionReceiptsJSON(b.chainID, receiptsJSON)
}
