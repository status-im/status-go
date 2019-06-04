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
func NewReactor(db *Database, feed *event.Feed, client *ethclient.Client, address common.Address, chain *big.Int) *Reactor {
	reactor := &Reactor{
		db:      db,
		client:  client,
		feed:    feed,
		address: address,
		chain:   chain,
	}
	reactor.erc20 = NewERC20TransfersDownloader(client, address)
	reactor.eth = &ETHTransferDownloader{
		client:  client,
		address: address,
		signer:  types.NewEIP155Signer(chain),
	}
	return reactor
}

// Reactor listens to new blocks and stores transfers into the database.
type Reactor struct {
	// FIXME(dshulyak) references same object. rework this part
	client  reactorClient
	db      *Database
	feed    *event.Feed
	address common.Address
	chain   *big.Int

	eth   *ETHTransferDownloader
	erc20 *ERC20TransfersDownloader

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
	erc20 := &erc20HistoricalCommand{
		db:     r.db,
		erc20:  r.erc20,
		client: r.client,
		feed:   r.feed,
	}
	r.group.Add(erc20.Command())
	eth := &ethHistoricalCommand{
		db:      r.db,
		client:  r.client,
		address: r.address,
		eth:     r.eth,
		feed:    r.feed,
	}
	r.group.Add(eth.Command())
	newBlocks := &newBlocksTransfersCommand{
		db:     r.db,
		chain:  r.chain,
		client: r.client,
		erc20:  r.erc20,
		eth:    r.eth,
		feed:   r.feed,
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
	r.group = nil
}

func headersFromTransfers(transfers []Transfer) []*DBHeader {
	byHash := map[common.Hash]struct{}{}
	rst := []*DBHeader{}
	for i := range transfers {
		_, exists := byHash[transfers[i].BlockHash]
		if exists {
			continue
		}
		rst = append(rst, &DBHeader{
			Hash:   transfers[i].BlockHash,
			Number: transfers[i].BlockNumber,
		})
	}
	return rst
}

// lastKnownHeader selects last stored header in database.
// If not found it will get head of the chain and in this case last known header will be atleast
// `reorgSafetyDepth` blocks away from chain head.
// `reorgSafetyDepth` is used for two purposes:
// 1. to minimize chances that historical downloader and new blocks downloader will find different transfers
// due to hitting different replicas (infura load balancer). new blocks downloader will eventually resolve conflicts
// by going back parent by parent but it will require more time.
// 2. as we don't store whole chain of blocks, but only blocks with transfers we won't be always able to find parent
// when reorg occurs if we won't start syncing headers atleast 15 blocks away from canonical chain head
func lastKnownHeader(db *Database, client HeaderReader) (*DBHeader, error) {
	known, err := db.LastHeader()
	if err != nil {
		return nil, err
	}
	if known == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		latest, err := client.HeaderByNumber(ctx, nil)
		cancel()
		if err != nil {
			return nil, err
		}
		if latest.Number.Cmp(reorgSafetyDepth) >= 0 {
			num := new(big.Int).Sub(latest.Number, reorgSafetyDepth)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			latest, err = client.HeaderByNumber(ctx, num)
			cancel()
			if err != nil {
				return nil, err
			}
		}
		known = toDBHeader(latest)
	}
	return known, nil
}
