package transfer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	ReactorNotStarted string = "reactor not started"

	NonArchivalNodeBlockChunkSize = 100
	DefaultNodeBlockChunkSize     = 100000
)

var errAlreadyRunning = errors.New("already running")

type FetchStrategyType int32

const (
	OnDemandFetchStrategyType FetchStrategyType = iota
	SequentialFetchStrategyType
)

// HeaderReader interface for reading headers using block number or hash.
type HeaderReader interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

// BalanceReader interface for reading balance at a specifeid address.
type BalanceReader interface {
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	FullTransactionByBlockNumberAndIndex(ctx context.Context, blockNumber *big.Int, index uint) (*chain.FullTransaction, error)
}

type HistoryFetcher interface {
	start() error
	stop()
	kind() FetchStrategyType

	getTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int,
		limit int64, fetchMore bool) ([]Transfer, error)
}

type OnDemandFetchStrategy struct {
	db                 *Database
	blockDAO           *BlockDAO
	feed               *event.Feed
	mu                 sync.Mutex
	group              *async.Group
	balanceCache       *balanceCache
	transactionManager *TransactionManager
	tokenManager       *token.Manager
	chainClients       map[uint64]*chain.ClientWithFallback
	accounts           []common.Address
}

func (s *OnDemandFetchStrategy) newControlCommand(chainClient *chain.ClientWithFallback, accounts []common.Address) *controlCommand {
	signer := types.NewLondonSigner(chainClient.ToBigInt())
	ctl := &controlCommand{
		db:          s.db,
		chainClient: chainClient,
		accounts:    accounts,
		blockDAO:    s.blockDAO,
		eth: &ETHDownloader{
			chainClient: chainClient,
			accounts:    accounts,
			signer:      signer,
			db:          s.db,
		},
		erc20:              NewERC20TransfersDownloader(chainClient, accounts, signer),
		feed:               s.feed,
		errorsCount:        0,
		transactionManager: s.transactionManager,
		tokenManager:       s.tokenManager,
	}

	return ctl
}

func (s *OnDemandFetchStrategy) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.group != nil {
		return errAlreadyRunning
	}
	s.group = async.NewGroup(context.Background())

	for _, chainClient := range s.chainClients {
		ctl := s.newControlCommand(chainClient, s.accounts)
		s.group.Add(ctl.Command())
	}

	return nil
}

// Stop stops reactor loop and waits till it exits.
func (s *OnDemandFetchStrategy) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.group == nil {
		return
	}
	s.group.Stop()
	s.group.Wait()
	s.group = nil
}

func (s *OnDemandFetchStrategy) kind() FetchStrategyType {
	return OnDemandFetchStrategyType
}

func (s *OnDemandFetchStrategy) getTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int,
	limit int64, fetchMore bool) ([]Transfer, error) {

	log.Info("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address, "fetchMore", fetchMore,
		"chainID", chainID, "toBlock", toBlock, "limit", limit)

	rst, err := s.db.GetTransfersByAddress(chainID, address, toBlock, limit)
	if err != nil {
		log.Error("[WalletAPI:: GetTransfersByAddress] can't fetch transfers", "err", err)
		return nil, err
	}

	transfersCount := int64(len(rst))

	if fetchMore && limit > transfersCount {

		block, err := s.blockDAO.GetFirstKnownBlock(chainID, address)
		if err != nil {
			return nil, err
		}

		// if zero block was already checked there is nothing to find more
		if block == nil || big.NewInt(0).Cmp(block) == 0 {
			log.Info("[WalletAPI:: GetTransfersByAddress] ZERO block is found for", "address", address, "chaindID", chainID)
			return rst, nil
		}

		chainClient, err := getChainClientByID(s.chainClients, chainID)
		if err != nil {
			return nil, err
		}

		from, err := findFirstRange(ctx, address, block, chainClient)
		if err != nil {
			if nonArchivalNodeError(err) {
				if s.feed != nil {
					s.feed.Send(walletevent.Event{
						Type: EventNonArchivalNodeDetected,
					})
				}
				if block.Cmp(big.NewInt(NonArchivalNodeBlockChunkSize)) >= 0 {
					from = big.NewInt(0).Sub(block, big.NewInt(NonArchivalNodeBlockChunkSize))
				} else {
					from = big.NewInt(0)
				}
			} else {
				log.Error("first range error", "error", err)
				return nil, err
			}
		}
		fromByAddress := map[common.Address]*Block{address: {
			Number: from,
		}}
		toByAddress := map[common.Address]*big.Int{address: block}

		if s.balanceCache == nil {
			s.balanceCache = newBalanceCache()
		}
		blocksCommand := &findAndCheckBlockRangeCommand{
			accounts:      []common.Address{address},
			db:            s.db,
			chainClient:   chainClient,
			balanceCache:  s.balanceCache,
			feed:          s.feed,
			fromByAddress: fromByAddress,
			toByAddress:   toByAddress,
		}

		if err = blocksCommand.Command()(ctx); err != nil {
			return nil, err
		}

		blocks, err := s.blockDAO.GetBlocksToLoadByAddress(chainID, address, numberOfBlocksCheckedPerIteration)
		if err != nil {
			return nil, err
		}

		log.Info("checking blocks again", "blocks", len(blocks))
		if len(blocks) > 0 {
			txCommand := &loadTransfersCommand{
				accounts:           []common.Address{address},
				db:                 s.db,
				blockDAO:           s.blockDAO,
				chainClient:        chainClient,
				transactionManager: s.transactionManager,
				blocksLimit:        numberOfBlocksCheckedPerIteration,
				tokenManager:       s.tokenManager,
			}

			err = txCommand.Command()(ctx)
			if err != nil {
				return nil, err
			}

			rst, err = s.db.GetTransfersByAddress(chainID, address, toBlock, limit)
			if err != nil {
				return nil, err
			}
		}
	}

	return rst, nil
}

// Reactor listens to new blocks and stores transfers into the database.
type Reactor struct {
	db                 *Database
	blockDAO           *BlockDAO
	feed               *event.Feed
	transactionManager *TransactionManager
	tokenManager       *token.Manager
	strategy           HistoryFetcher
}

func NewReactor(db *Database, blockDAO *BlockDAO, feed *event.Feed, tm *TransactionManager, tokenManager *token.Manager) *Reactor {
	return &Reactor{
		db:                 db,
		blockDAO:           blockDAO,
		feed:               feed,
		transactionManager: tm,
		tokenManager:       tokenManager,
	}
}

// Start runs reactor loop in background.
func (r *Reactor) start(chainClients map[uint64]*chain.ClientWithFallback, accounts []common.Address,
	loadAllTransfers bool) error {

	r.strategy = r.createFetchStrategy(chainClients, accounts, loadAllTransfers)
	return r.strategy.start()
}

// Stop stops reactor loop and waits till it exits.
func (r *Reactor) stop() {
	if r.strategy != nil {
		r.strategy.stop()
	}
}

func (r *Reactor) restart(chainClients map[uint64]*chain.ClientWithFallback, accounts []common.Address,
	loadAllTransfers bool) error {

	r.stop()
	return r.start(chainClients, accounts, loadAllTransfers)
}

func (r *Reactor) createFetchStrategy(chainClients map[uint64]*chain.ClientWithFallback,
	accounts []common.Address, loadAllTransfers bool) HistoryFetcher {

	if loadAllTransfers {
		return NewSequentialFetchStrategy(
			r.db,
			r.blockDAO,
			r.feed,
			r.transactionManager,
			r.tokenManager,
			chainClients,
			accounts,
		)
	}

	return &OnDemandFetchStrategy{
		db:                 r.db,
		feed:               r.feed,
		blockDAO:           r.blockDAO,
		transactionManager: r.transactionManager,
		tokenManager:       r.tokenManager,
		chainClients:       chainClients,
		accounts:           accounts,
	}
}

func (r *Reactor) getTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int,
	limit int64, fetchMore bool) ([]Transfer, error) {

	if r.strategy != nil {
		return r.strategy.getTransfersByAddress(ctx, chainID, address, toBlock, limit, fetchMore)
	}

	return nil, errors.New(ReactorNotStarted)
}

func getChainClientByID(clients map[uint64]*chain.ClientWithFallback, id uint64) (*chain.ClientWithFallback, error) {
	for _, client := range clients {
		if client.ChainID == id {
			return client, nil
		}
	}

	return nil, fmt.Errorf("chain client not found with id=%d", id)
}
