package transfer

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

const (
	ReactorNotStarted string = "reactor not started"

	NonArchivalNodeBlockChunkSize = 100
	DefaultNodeBlockChunkSize     = 100000
)

var errAlreadyRunning = errors.New("already running")

type FetchStrategyType int32

const (
	SequentialFetchStrategyType FetchStrategyType = iota
)

// HeaderReader interface for reading headers using block number or hash.
type HeaderReader interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

type HistoryFetcher interface {
	start() error
	stop()
	kind() FetchStrategyType

	getTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int,
		limit int64) ([]Transfer, error)
}

// Reactor listens to new blocks and stores transfers into the database.
type Reactor struct {
	db                 *Database
	blockDAO           *BlockDAO
	blockRangesSeqDAO  *BlockRangeSequentialDAO
	feed               *event.Feed
	transactionManager *TransactionManager
	pendingTxManager   *transactions.PendingTxTracker
	tokenManager       *token.Manager
	strategy           HistoryFetcher
	balanceCacher      balance.Cacher
	omitHistory        bool
}

func NewReactor(db *Database, blockDAO *BlockDAO, blockRangesSeqDAO *BlockRangeSequentialDAO, feed *event.Feed, tm *TransactionManager,
	pendingTxManager *transactions.PendingTxTracker, tokenManager *token.Manager,
	balanceCacher balance.Cacher, omitHistory bool) *Reactor {
	return &Reactor{
		db:                 db,
		blockDAO:           blockDAO,
		blockRangesSeqDAO:  blockRangesSeqDAO,
		feed:               feed,
		transactionManager: tm,
		pendingTxManager:   pendingTxManager,
		tokenManager:       tokenManager,
		balanceCacher:      balanceCacher,
		omitHistory:        omitHistory,
	}
}

// Start runs reactor loop in background.
func (r *Reactor) start(chainClients map[uint64]chain.ClientInterface, accounts []common.Address) error {

	r.strategy = r.createFetchStrategy(chainClients, accounts)
	return r.strategy.start()
}

// Stop stops reactor loop and waits till it exits.
func (r *Reactor) stop() {
	if r.strategy != nil {
		r.strategy.stop()
	}
}

func (r *Reactor) restart(chainClients map[uint64]chain.ClientInterface, accounts []common.Address) error {

	r.stop()
	return r.start(chainClients, accounts)
}

func (r *Reactor) createFetchStrategy(chainClients map[uint64]chain.ClientInterface,
	accounts []common.Address) HistoryFetcher {

	return NewSequentialFetchStrategy(
		r.db,
		r.blockDAO,
		r.blockRangesSeqDAO,
		r.feed,
		r.transactionManager,
		r.pendingTxManager,
		r.tokenManager,
		chainClients,
		accounts,
		r.balanceCacher,
		r.omitHistory,
	)
}

func (r *Reactor) getTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int,
	limit int64) ([]Transfer, error) {

	if r.strategy != nil {
		return r.strategy.getTransfersByAddress(ctx, chainID, address, toBlock, limit)
	}

	return nil, errors.New(ReactorNotStarted)
}
