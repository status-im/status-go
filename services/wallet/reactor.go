package wallet

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

var (
	erc20BatchSize    = big.NewInt(100000)
	errAlreadyRunning = errors.New("already running")
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
}

// NewReactor creates instance of the Reactor.
func NewReactor(db *Database, feed *event.Feed, client *chainClient, chainID uint64) *Reactor {
	return &Reactor{
		db:     db,
		client: client,
		feed:   feed,
		chain:  new(big.Int).SetUint64(chainID),
	}
}

// Reactor listens to new blocks and stores transfers into the database.
type Reactor struct {
	client *chainClient
	db     *Database
	feed   *event.Feed
	chain  *big.Int

	mu    sync.Mutex
	group *Group
}

func (r *Reactor) newControlCommand(accounts []common.Address) *controlCommand {
	signer := types.NewLondonSigner(r.chain)
	ctl := &controlCommand{
		db:       r.db,
		chain:    r.chain,
		client:   r.client,
		accounts: accounts,
		eth: &ETHTransferDownloader{
			chain:    r.chain,
			client:   r.client,
			accounts: accounts,
			signer:   signer,
			db:       r.db,
		},
		erc20:       NewERC20TransfersDownloader(r.client, accounts, signer),
		feed:        r.feed,
		errorsCount: 0,
	}

	return ctl
}

// Start runs reactor loop in background.
func (r *Reactor) Start(accounts []common.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.group != nil {
		return errAlreadyRunning
	}
	r.group = NewGroup(context.Background())
	ctl := r.newControlCommand(accounts)
	r.group.Add(ctl.Command())
	return nil
}

// Stop stops reactor loop and waits till it exits.
func (r *Reactor) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.group == nil {
		return
	}
	r.group.Stop()
	r.group.Wait()
	r.group = nil
}
