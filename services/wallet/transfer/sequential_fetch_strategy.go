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

func NewSequentialFetchStrategy(db *Database, blockDAO *BlockDAO, feed *event.Feed,
	transactionManager *TransactionManager, pendingTxManager *transactions.PendingTxTracker,
	tokenManager *token.Manager,
	chainClients map[uint64]*chain.ClientWithFallback,
	accounts []common.Address,
	balanceCacher balance.Cacher,
) *SequentialFetchStrategy {

	return &SequentialFetchStrategy{
		db:                 db,
		blockDAO:           blockDAO,
		feed:               feed,
		transactionManager: transactionManager,
		pendingTxManager:   pendingTxManager,
		tokenManager:       tokenManager,
		chainClients:       chainClients,
		accounts:           accounts,
		balanceCacher:      balanceCacher,
	}
}

type SequentialFetchStrategy struct {
	db                 *Database
	blockDAO           *BlockDAO
	feed               *event.Feed
	mu                 sync.Mutex
	group              *async.Group
	transactionManager *TransactionManager
	pendingTxManager   *transactions.PendingTxTracker
	tokenManager       *token.Manager
	chainClients       map[uint64]*chain.ClientWithFallback
	accounts           []common.Address
	balanceCacher      balance.Cacher
}

func (s *SequentialFetchStrategy) newCommand(chainClient *chain.ClientWithFallback,
	account common.Address) async.Commander {

	return newLoadBlocksAndTransfersCommand(account, s.db, s.blockDAO, chainClient, s.feed,
		s.transactionManager, s.pendingTxManager, s.tokenManager, s.balanceCacher)
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

// TODO: remove fetchMore parameter from here and interface, it is used by OnDemandFetchStrategy only
func (s *SequentialFetchStrategy) getTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int,
	limit int64, fetchMore bool) ([]Transfer, error) {

	log.Info("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address, "fetchMore", fetchMore,
		"chainID", chainID, "toBlock", toBlock, "limit", limit)

	rst, err := s.db.GetTransfersByAddress(chainID, address, toBlock, limit)
	if err != nil {
		log.Error("[WalletAPI:: GetTransfersByAddress] can't fetch transfers", "err", err)
		return nil, err
	}

	return rst, nil
}
