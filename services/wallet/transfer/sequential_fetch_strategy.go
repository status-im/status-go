package transfer

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
)

type SequentialFetchStrategy struct {
	db                 *Database
	blockDAO           *BlockDAO
	feed               *event.Feed
	mu                 sync.Mutex
	group              *async.Group
	transactionManager *TransactionManager
	chainClients       map[uint64]*chain.ClientWithFallback
	accounts           []common.Address
}

func (s *SequentialFetchStrategy) newCommand(chainClient *chain.ClientWithFallback,
	// accounts []common.Address) *loadAllTransfersCommand {
	accounts []common.Address) async.Commander {

	signer := types.NewLondonSigner(chainClient.ToBigInt())
	// ctl := &loadAllTransfersCommand{
	ctl := &controlCommand{ // TODO Will be replaced by loadAllTranfersCommand in upcoming commit
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
	}
	return ctl
}

func (s *SequentialFetchStrategy) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.group != nil {
		return errAlreadyRunning
	}
	s.group = async.NewGroup(context.Background())

	for _, chainClient := range s.chainClients {
		ctl := s.newCommand(chainClient, s.accounts)
		s.group.Add(ctl.Command())
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
	limit int64, fetchMore bool) ([]Transfer, error) {

	// TODO: implement - load from database
	return []Transfer{}, nil
}
