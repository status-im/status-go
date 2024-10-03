package ethclient

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type CachedEthClient struct {
	RPSLimitedEthClientInterface
	storage EthClientChainStorage
}

func NewCachedEthClient(client RPSLimitedEthClientInterface, storage EthClientChainStorage) *CachedEthClient {
	return &CachedEthClient{
		RPSLimitedEthClientInterface: client,
		storage:                      storage,
	}
}

func (c *CachedEthClient) CopyWithName(name string) RPSLimitedEthClientInterface {
	return NewCachedEthClient(c.RPSLimitedEthClientInterface.CopyWithName(name), c.storage)
}

func (c *CachedEthClient) processBlockJSON(blockJSON json.RawMessage, transactionDetailsFlag bool) (*parsedBlockJSON, error) {
	if len(blockJSON) == 0 {
		return nil, ethereum.NotFound
	}

	block, err := parseBlockJSON(blockJSON, transactionDetailsFlag)
	if err != nil {
		return nil, err
	}

	if err := c.storage.PutBlockJSON(blockJSON, transactionDetailsFlag); err != nil {
		return nil, err
	}

	return block, nil
}

func (c *CachedEthClient) getOrFetchBlockByNumber(ctx context.Context, number *big.Int, transactionDetailsFlag bool) (*parsedBlockJSON, error) {
	var block *parsedBlockJSON

	blockJSON, err := c.storage.GetBlockJSONByNumber(number, transactionDetailsFlag)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if len(blockJSON) > 0 {
		block, err = parseBlockJSON(blockJSON, transactionDetailsFlag)
		if err != nil {
			return nil, err
		}
		return block, nil
	}

	// Not available in storage, we fetch it
	err = c.RPSLimitedEthClientInterface.CallContext(ctx, &blockJSON, "eth_getBlockByNumber", toBlockNumArg(number), transactionDetailsFlag)
	if err != nil {
		return nil, err
	}
	return c.processBlockJSON(blockJSON, transactionDetailsFlag)
}

func (c *CachedEthClient) getOrFetchBlockByHash(ctx context.Context, hash common.Hash, transactionDetailsFlag bool) (*parsedBlockJSON, error) {
	var block *parsedBlockJSON

	blockJSON, err := c.storage.GetBlockJSONByHash(hash, transactionDetailsFlag)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if len(blockJSON) > 0 {
		block, err = parseBlockJSON(blockJSON, transactionDetailsFlag)
		if err != nil {
			return nil, err
		}
		return block, nil
	}

	// Not available in storage
	err = c.RPSLimitedEthClientInterface.CallContext(ctx, &blockJSON, "eth_getBlockByHash", hash, transactionDetailsFlag)
	if err != nil {
		return nil, err
	}
	return c.processBlockJSON(blockJSON, transactionDetailsFlag)
}

func (c *CachedEthClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	block, err := c.getOrFetchBlockByHash(ctx, hash, false)
	return block.header, err
}

func (c *CachedEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	block, err := c.getOrFetchBlockByNumber(ctx, number, false)
	return block.header, err
}

func (c *CachedEthClient) getOrFetchUncles(ctx context.Context, blockHash common.Hash, uncleHashes []common.Hash) ([]*types.Header, error) {
	var err error
	uncles := make([]*types.Header, len(uncleHashes))
	unclesJSON := make([]json.RawMessage, len(uncleHashes))

	cacheValid := true
	for i := range uncleHashes {
		unclesJSON[i], err = c.storage.GetBlockUncleJSONByHashAndIndex(blockHash, uint64(i))
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if len(unclesJSON[i]) == 0 {
			cacheValid = false
			break
		}
	}

	if !cacheValid {
		reqs := make([]rpc.BatchElem, len(uncleHashes))
		for i := range reqs {
			reqs[i] = rpc.BatchElem{
				Method: "eth_getUncleByBlockHashAndIndex",
				Args:   []interface{}{blockHash, hexutil.EncodeUint64(uint64(i))},
				Result: &unclesJSON[i],
			}
		}
		if err := c.BatchCallContext(ctx, reqs); err != nil {
			return nil, err
		}
		for i := range reqs {
			if reqs[i].Error != nil {
				return nil, reqs[i].Error
			}
		}
	}

	for i := range unclesJSON {
		if err := json.Unmarshal(unclesJSON[i], &uncles[i]); err != nil {
			return nil, err
		}
		if uncles[i] == nil {
			return nil, fmt.Errorf("got null header for uncle %d of block %x", i, blockHash[:])
		}
	}

	if !cacheValid {
		if err := c.storage.PutBlockUnclesJSON(blockHash, unclesJSON); err != nil {
			return nil, err
		}
	}

	return uncles, nil
}

func (c *CachedEthClient) getBlock(ctx context.Context, block *parsedBlockJSON) (*types.Block, error) {
	// Quick-verify transaction and uncle lists. This mostly helps with debugging the server.
	if block.header.UncleHash == types.EmptyUncleHash && len(block.body.UncleHashes) > 0 {
		return nil, fmt.Errorf("server returned non-empty uncle list but block header indicates no uncles")
	}
	if block.header.UncleHash != types.EmptyUncleHash && len(block.body.UncleHashes) == 0 {
		return nil, fmt.Errorf("server returned empty uncle list but block header indicates uncles")
	}
	if block.header.TxHash == types.EmptyRootHash && len(block.txs) > 0 {
		return nil, fmt.Errorf("server returned non-empty transaction list but block header indicates no transactions")
	}
	if block.header.TxHash != types.EmptyRootHash && len(block.txs) == 0 {
		return nil, fmt.Errorf("server returned empty transaction list but block header indicates transactions")
	}

	// Load uncles because they are not included in the block response.
	uncles, err := c.getOrFetchUncles(ctx, block.body.Hash, block.body.UncleHashes)
	if err != nil {
		return nil, err
	}

	// Fill the sender cache of transactions in the block.
	txs := make([]*types.Transaction, len(block.txs))
	for i, tx := range block.txs {
		if tx.From != nil {
			setSenderFromServer(tx.tx, *tx.From, block.body.Hash)
		}
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(block.header).WithBody(txs, uncles), nil
}

func (c *CachedEthClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, err := c.getOrFetchBlockByHash(ctx, hash, true)
	if err != nil {
		return nil, err
	}

	return c.getBlock(ctx, block)
}

func (c *CachedEthClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	block, err := c.getOrFetchBlockByNumber(ctx, number, true)
	if err != nil {
		return nil, err
	}

	return c.getBlock(ctx, block)
}

func (c *CachedEthClient) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	var txJSON json.RawMessage
	var err error

	cacheValid := true
	txJSON, err = c.storage.GetTransactionJSONByHash(hash)
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}
	if len(txJSON) == 0 {
		cacheValid = false
		err = c.CallContext(ctx, &txJSON, "eth_getTransactionByHash", hash)
		if err != nil {
			return nil, false, err
		}
	}

	var rpcTx *rpcTransaction
	if err := json.Unmarshal(txJSON, &rpcTx); err != nil {
		return nil, false, err
	} else if rpcTx == nil {
		return nil, false, ethereum.NotFound
	} else if _, r, _ := rpcTx.tx.RawSignatureValues(); r == nil {
		return nil, false, fmt.Errorf("server returned transaction without signature")
	}
	if rpcTx.From != nil && rpcTx.BlockHash != nil {
		setSenderFromServer(rpcTx.tx, *rpcTx.From, *rpcTx.BlockHash)
	}

	if !cacheValid {
		if err := c.storage.PutTransactionsJSON([]json.RawMessage{txJSON}); err != nil {
			return nil, false, err
		}
	}

	return rpcTx.tx, rpcTx.BlockNumber == nil, nil
}

func (c *CachedEthClient) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	var r *types.Receipt

	cacheValid := true
	receiptJSON, err := c.storage.GetTransactionReceiptJSONByHash(hash)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if len(receiptJSON) == 0 {
		cacheValid = false
		err = c.CallContext(ctx, &receiptJSON, "eth_getTransactionReceipt", hash)
		if err != nil {
			return nil, err
		}
	}

	if err := json.Unmarshal(receiptJSON, &r); err != nil {
		return nil, err
	} else if r == nil {
		return nil, ethereum.NotFound
	}

	if !cacheValid {
		if err := c.storage.PutTransactionReceiptsJSON([]json.RawMessage{receiptJSON}); err != nil {
			return nil, err
		}
	}

	return r, nil
}
