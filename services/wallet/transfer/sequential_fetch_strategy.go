package transfer

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"
)

func NewSequentialFetchStrategy(db *Database, blockDAO *BlockDAO, blockRangesSeqDAO *BlockRangeSequentialDAO, feed *event.Feed,
	transactionManager *TransactionManager, pendingTxManager *transactions.PendingTxTracker,
	tokenManager *token.Manager,
	chainClients map[uint64]chain.ClientInterface,
	accounts []common.Address,
	balanceCacher balance.Cacher,
	omitHistory bool,
) *SequentialFetchStrategy {

	return &SequentialFetchStrategy{
		db:                 db,
		blockDAO:           blockDAO,
		blockRangesSeqDAO:  blockRangesSeqDAO,
		feed:               feed,
		transactionManager: transactionManager,
		pendingTxManager:   pendingTxManager,
		tokenManager:       tokenManager,
		chainClients:       chainClients,
		accounts:           accounts,
		balanceCacher:      balanceCacher,
		omitHistory:        omitHistory,
	}
}

type SequentialFetchStrategy struct {
	db                 *Database
	blockDAO           *BlockDAO
	blockRangesSeqDAO  *BlockRangeSequentialDAO
	feed               *event.Feed
	mu                 sync.Mutex
	group              *async.Group
	transactionManager *TransactionManager
	pendingTxManager   *transactions.PendingTxTracker
	tokenManager       *token.Manager
	chainClients       map[uint64]chain.ClientInterface
	accounts           []common.Address
	balanceCacher      balance.Cacher
	omitHistory        bool
}

func (s *SequentialFetchStrategy) newCommand(chainClient chain.ClientInterface,
	account common.Address) async.Commander {

	return newLoadBlocksAndTransfersCommand(account, s.db, s.blockDAO, s.blockRangesSeqDAO, chainClient, s.feed,
		s.transactionManager, s.pendingTxManager, s.tokenManager, s.balanceCacher, s.omitHistory)
}

func (s *SequentialFetchStrategy) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.group != nil {
		return errAlreadyRunning
	}
	s.group = async.NewGroup(context.Background())

	if s.feed != nil {
		s.feed.Send(walletevent.Event{
			Type:     EventFetchingRecentHistory,
			Accounts: s.accounts,
		})
	}

	for _, chainClient := range s.chainClients {
		for _, address := range s.accounts {
			ctl := s.newCommand(chainClient, address)
			s.group.Add(ctl.Command())
		}
	}

	return nil
}

// Stop stops reactor loop and waits till it exits.
func (s *SequentialFetchStrategy) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.group == nil {
		return
	}
	s.group.Stop()
	s.group.Wait()
	s.group = nil
}

func (s *SequentialFetchStrategy) kind() FetchStrategyType {
	return SequentialFetchStrategyType
}

func (s *SequentialFetchStrategy) getTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int,
	limit int64) ([]Transfer, error) {

	log.Debug("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address,
		"chainID", chainID, "toBlock", toBlock, "limit", limit)

	rst, err := s.db.GetTransfersByAddress(chainID, address, toBlock, limit)
	if err != nil {
		log.Error("[WalletAPI:: GetTransfersByAddress] can't fetch transfers", "err", err)
		return nil, err
	}

	return rst, nil
}
