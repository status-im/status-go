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
		return 2 * time.Second
	default:
		return 500 * time.Millisecond
	}
}

var (
	reorgSafetyDepth = big.NewInt(15)
	erc20BatchSize   = big.NewInt(50000)
)

// HeaderReader interface for reading headers using block number or hash.
type HeaderReader interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

// BalanceReader interface for reading balance at a specifeid address.
type BalanceReader interface {
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

type reactorClient interface {
	HeaderReader
	BalanceReader
}

// NewReactor creates instance of the Reactor.
func NewReactor(db *Database, feed *event.Feed, client *ethclient.Client, accounts []common.Address, chain *big.Int) *Reactor {
	return &Reactor{
		db:       db,
		client:   client,
		feed:     feed,
		accounts: accounts,
		chain:    chain,
	}
}

// Reactor listens to new blocks and stores transfers into the database.
type Reactor struct {
	client   *ethclient.Client
	db       *Database
	feed     *event.Feed
	accounts []common.Address
	chain    *big.Int

	mu    sync.Mutex
	group *Group
}

// Start runs reactor loop in background.
func (r *Reactor) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.group != nil {
		return errors.New("already running")
	}
	r.group = NewGroup()
	// TODO(dshulyak) to support adding accounts in runtime implement keyed group
	// and export private api to start downloaders from accounts
	// private api should have access only to reactor
	for _, address := range r.accounts {
		erc20 := &erc20HistoricalCommand{
			db:          r.db,
			erc20:       NewERC20TransfersDownloader(r.client, []common.Address{address}),
			client:      r.client,
			feed:        r.feed,
			safetyDepth: reorgSafetyDepth,
			address:     address,
		}
		r.group.Add(erc20.Command())
		eth := &ethHistoricalCommand{
			db:      r.db,
			client:  r.client,
			address: address,
			eth: &ETHTransferDownloader{
				client:   r.client,
				accounts: []common.Address{address},
				signer:   types.NewEIP155Signer(r.chain),
			},
			feed:        r.feed,
			safetyDepth: reorgSafetyDepth,
		}
		r.group.Add(eth.Command())
	}
	newBlocks := &newBlocksTransfersCommand{
		db:       r.db,
		chain:    r.chain,
		client:   r.client,
		accounts: r.accounts,
		eth: &ETHTransferDownloader{
			client:   r.client,
			accounts: r.accounts,
			signer:   types.NewEIP155Signer(r.chain),
		},
		erc20:       NewERC20TransfersDownloader(r.client, r.accounts),
		feed:        r.feed,
		safetyDepth: reorgSafetyDepth,
	}
	r.group.Add(newBlocks.Command())
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
