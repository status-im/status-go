package wallet

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/params"
)

// pow block on main chain is mined once per ~14 seconds
// but for tests we are using clique chain with immediate block signer
// hence we can use different polling periods for methods that depend on mining time.
func pollingPeriodByChain(chain *big.Int) time.Duration {
	switch chain.Int64() {
	case int64(params.MainNetworkID):
		return 10 * time.Second
	case int64(params.RopstenNetworkID):
		return 4 * time.Second
	default:
		return 500 * time.Millisecond
	}
}

func reorgSafetyDepth(chain *big.Int) *big.Int {
	switch chain.Int64() {
	case int64(params.MainNetworkID):
		return big.NewInt(5)
	case int64(params.RopstenNetworkID):
		return big.NewInt(15)
	default:
		return big.NewInt(15)
	}
}

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
}

type reactorClient interface {
	HeaderReader
	BalanceReader
}

// NewReactor creates instance of the Reactor.
func NewReactor(db *Database, feed *event.Feed, client *ethclient.Client, chain *big.Int) *Reactor {
	return &Reactor{
		db:     db,
		client: client,
		feed:   feed,
		chain:  chain,
	}
}

// Reactor listens to new blocks and stores transfers into the database.
type Reactor struct {
	client *ethclient.Client
	db     *Database
	feed   *event.Feed
	chain  *big.Int

	mu    sync.Mutex
	group *Group
}

func (r *Reactor) newControlCommand(accounts []common.Address) *controlCommand {
	signer := types.NewEIP155Signer(r.chain)
	ctl := &controlCommand{
		db:       r.db,
		chain:    r.chain,
		client:   r.client,
		accounts: accounts,
		eth: &ETHTransferDownloader{
			client:   r.client,
			accounts: accounts,
			signer:   signer,
			db:       r.db,
		},
		erc20:       NewERC20TransfersDownloader(r.client, accounts, signer),
		feed:        r.feed,
		safetyDepth: reorgSafetyDepth(r.chain),
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
