package transfer

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
)

var errAlreadyRunning = errors.New("already running")

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
}

// Reactor listens to new blocks and stores transfers into the database.
type Reactor struct {
	db    *Database
	block *Block
	feed  *event.Feed
	mu    sync.Mutex
	group *async.Group
}

func (r *Reactor) newControlCommand(chainClient *chain.ClientWithFallback, accounts []common.Address) *controlCommand {
	signer := types.NewLondonSigner(chainClient.ToBigInt())
	ctl := &controlCommand{
		db:          r.db,
		chainClient: chainClient,
		accounts:    accounts,
		block:       r.block,
		eth: &ETHDownloader{
			chainClient: chainClient,
			accounts:    accounts,
			signer:      signer,
			db:          r.db,
		},
		erc20:       NewERC20TransfersDownloader(chainClient, accounts, signer),
		feed:        r.feed,
		errorsCount: 0,
	}

	return ctl
}

// Start runs reactor loop in background.
func (r *Reactor) start(chainClients []*chain.ClientWithFallback, accounts []common.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.group != nil {
		return errAlreadyRunning
	}
	r.group = async.NewGroup(context.Background())
	for _, chainClient := range chainClients {
		ctl := r.newControlCommand(chainClient, accounts)
		r.group.Add(ctl.Command())
	}
	return nil
}

// Stop stops reactor loop and waits till it exits.
func (r *Reactor) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.group == nil {
		return
	}
	r.group.Stop()
	r.group.Wait()
	r.group = nil
}

func (r *Reactor) restart(chainClients []*chain.ClientWithFallback, accounts []common.Address) error {
	r.stop()
	return r.start(chainClients, accounts)
}
