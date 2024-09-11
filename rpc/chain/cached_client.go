package chain

import (
	"context"
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type CachedClient struct {
	*ClientWithFallback
	db *DB
}

func NewCachedClient(ethClients []*EthClient, chainID uint64, db *sql.DB) *CachedClient {
	return &CachedClient{
		ClientWithFallback: NewClient(ethClients, chainID),
		db:                 NewDB(db),
	}
}

func (c *CachedClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	header, err := c.db.GetBlockHeaderByHash(c.NetworkID(), hash)
	if err == nil {
		return header, nil
	} else if err != sql.ErrNoRows {
		// Soft error, we can continue
		log.Error("Failed to get header from cache", "error", err)
	}

	header, err = c.ClientWithFallback.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	err = c.db.PutBlockHeader(c.NetworkID(), header)
	if err != nil {
		// Soft error, we can continue
		log.Error("Failed to put header into cache", "error", err)
	}

	return header, nil
}

func (c *CachedClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, err := c.db.GetBlockByHash(c.NetworkID(), hash)
	if err == nil {
		return block, nil
	} else if err != sql.ErrNoRows {
		// Soft error, we can continue
		log.Error("Failed to get block from cache", "error", err)
	}

	block, err = c.ClientWithFallback.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	if block != nil {
		err = c.db.PutBlock(c.NetworkID(), block)
		if err != nil {
			// Soft error, we can continue
			log.Error("Failed to put block into cache", "error", err)
		}
		err = c.db.PutTransactions(c.NetworkID(), block.Transactions())
		if err != nil {
			// Soft error, we can continue
			log.Error("Failed to put transactions into cache", "error", err)
		}
	}

	return block, nil
}

func (c *CachedClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	block, err := c.db.GetBlockByNumber(c.NetworkID(), number)
	if err == nil {
		return block, nil
	} else if err != sql.ErrNoRows {
		// Soft error, we can continue
		log.Error("Failed to get block from cache", "error", err)
	}

	block, err = c.ClientWithFallback.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	err = c.db.PutBlock(c.NetworkID(), block)
	if err != nil {
		// Soft error, we can continue
		log.Error("Failed to put block into cache", "error", err)
	}

	return block, nil
}

func (c *CachedClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	header, err := c.db.GetBlockHeaderByNumber(c.NetworkID(), number)
	if err == nil {
		return header, nil
	} else if err != sql.ErrNoRows {
		// Soft error, we can continue
		log.Error("Failed to get header from cache", "error", err)
	}

	header, err = c.ClientWithFallback.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	err = c.db.PutBlockHeader(c.NetworkID(), header)
	if err != nil {
		// Soft error, we can continue
		log.Error("Failed to put header into cache", "error", err)
	}

	return header, nil
}

func (c *CachedClient) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	transaction, err := c.db.GetTransactionByHash(c.NetworkID(), hash)
	if err == nil {
		return transaction, false, nil
	} else if err != sql.ErrNoRows {
		// Soft error, we can continue
		log.Error("Failed to get transaction from cache", "error", err)
	}

	transaction, pending, err := c.ClientWithFallback.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, pending, err
	}

	if !pending {
		err = c.db.PutTransactions(c.NetworkID(), types.Transactions{transaction})
		if err != nil {
			// Soft error, we can continue
			log.Error("Failed to put transaction into cache", "error", err)
		}
	}

	return transaction, pending, nil
}

func (c *CachedClient) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	receipt, err := c.db.GetTransactionReceipt(c.NetworkID(), hash)
	if err == nil {
		return receipt, nil
	} else if err != sql.ErrNoRows {
		// Soft error, we can continue
		log.Error("Failed to get transaction receipt from cache", "error", err)
	}

	receipt, err = c.ClientWithFallback.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	err = c.db.PutTransactionReceipt(c.NetworkID(), receipt)
	if err != nil {
		// Soft error, we can continue
		log.Error("Failed to put transaction receipt into cache", "error", err)
	}

	return receipt, nil
}
