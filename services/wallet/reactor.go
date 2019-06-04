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
	"github.com/ethereum/go-ethereum/log"
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

	wg     sync.WaitGroup
	ctx    context.Context
	cancel func()
}

// Start runs reactor loop in background.
func (r *Reactor) Start() error {
	if r.ctx != nil {
		return errors.New("already running")
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	r.wg.Add(1)
	go func() {
		log.Info("wallet loop for new transfers started", "address", r.address)
		r.newTransfersLoop()
		r.wg.Done()
	}()
	r.wg.Add(1)
	go func() {
		log.Info("wallet loop for erc20 historical transfers started", "address", r.address)
		r.erc20HistoricalLoop()
		r.wg.Done()
	}()
	r.wg.Add(1)
	go func() {
		log.Info("wallet loop for eth historical transfers started", "address", r.address)
		r.ethHistoricalLoop()
		r.wg.Done()
	}()
	return nil
}

// Stop stops reactor loop and waits till it exits.
func (r *Reactor) Stop() {
	if r.ctx == nil {
		return
	}
	r.cancel()
	r.wg.Wait()
	r.cancel = nil
	r.ctx = nil
}

func (r *Reactor) erc20HistoricalLoop() {
	// TODO(dshulyak) waiting makes sense only in case of error
	// otherwise contiue processing in batches without waiting.
	var (
		ticker   = time.NewTicker(5 * time.Second)
		iterator *IterativeDownloader
		err      error
	)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if iterator == nil {
				iterator, err = SetupIterativeDownloader(r.db, r.client, erc20Sync, r.erc20, erc20BatchSize)
				if err != nil {
					log.Error("failed to setup historical downloader for erc20")
					continue
				}
			}
			for !iterator.Finished() {
				transfers, err := iterator.Next(r.ctx)
				if err != nil {
					log.Error("failed to get next batch", "error", err)
					break
				}
				headers := headersFromTransfers(transfers)
				headers = append(headers, iterator.Header())
				err = r.db.ProcessTranfers(transfers, headers, nil, erc20Sync)
				if err != nil {
					iterator.Revert()
					log.Error("failed to save downloaded erc20 transfers", "error", err)
					break
				}
				if len(transfers) > 0 {
					r.feed.Send(Event{
						Type:        EventNewHistory,
						BlockNumber: iterator.Header().Number,
					})
				}
			}
			if iterator.Finished() {
				log.Info("wallet historical downloader for erc20 transfers finished")
				return
			}
		}
	}
}

func (r *Reactor) ethHistoricalLoop() {
	var (
		ticker   = time.NewTicker(5 * time.Second)
		previous *DBHeader
		err      error
	)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if previous == nil {
				previous, err = r.db.GetEarliestSynced(ethSync)
				if err != nil {
					continue
				}
				if previous == nil {
					previous, err = lastKnownHeader(r.db, r.client)
					if err != nil {
						continue
					}
				}
			}
			ctx, cancel := context.WithTimeout(r.ctx, 10*time.Minute)
			defer cancel()
			concurrent := NewConcurrentDownloader(ctx)
			start := time.Now()
			downloadEthConcurrently(concurrent, r.client, r.eth, r.address, zero, previous.Number)
			concurrent.Wait()
			if concurrent.Error() != nil {
				log.Error("failed to dowloader transfers using concurrent downloader", "error", err)
				break
			}
			transfers := concurrent.Transfers()
			log.Info("eth historical downloader finished succesfully", "total transfers", len(transfers), "time", time.Since(start))
			// TODO(dshulyak) insert 0 block number with transfers
			err = r.db.ProcessTranfers(transfers, headersFromTransfers(transfers), nil, ethSync)
			if err != nil {
				log.Error("failed to save downloaded erc20 transfers", "error", err)
				break
			}
			if len(transfers) > 0 {
				// we download all or nothing
				r.feed.Send(Event{
					Type:        EventNewHistory,
					BlockNumber: zero,
				})
			}
			log.Debug("eth transfer persisted. loop is closed")
			return
		}
	}
}

// newTransfersLoop looks for new transfers block by block
func (r *Reactor) newTransfersLoop() {
	var (
		ticker   = time.NewTicker(pollingPeriodByChain(r.chain))
		num      = new(big.Int)
		previous *DBHeader
		err      error
	)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if previous == nil {
				previous, err = lastKnownHeader(r.db, r.client)
				if err != nil {
					log.Error("failed to get last known header", "error", err)
					continue
				}
			}
			num = num.Add(previous.Number, one)
			ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
			latest, err := r.client.HeaderByNumber(ctx, num)
			cancel()
			if err != nil {
				log.Warn("failed to get latest block", "number", num, "error", err)
				continue
			}
			log.Debug("reactor received new block", "header", latest.Hash())
			ctx, cancel = context.WithTimeout(r.ctx, 10*time.Second)
			added, removed, err := r.onNewBlock(ctx, previous, latest)
			cancel()
			if err != nil {
				log.Error("failed to process new header", "header", latest, "error", err)
				continue
			}
			// for each added block get tranfers from downloaders
			all := []Transfer{}
			for i := range added {
				log.Debug("reactor get transfers", "block", added[i].Hash, "number", added[i].Number)
				transfers, err := r.getTransfers(added[i])
				if err != nil {
					log.Error("failed to get transfers", "header", added[i].Hash, "error", err)
					continue
				}
				log.Debug("reactor adding transfers", "block", added[i].Hash, "number", added[i].Number, "len", len(transfers))
				all = append(all, transfers...)
			}
			err = r.db.ProcessTranfers(all, added, removed, erc20Sync|ethSync)
			if err != nil {
				log.Error("failed to persist transfers", "error", err)
				continue
			}
			previous = toDBHeader(latest)
			if len(added) == 1 && len(removed) == 0 {
				r.feed.Send(Event{
					Type:        EventNewBlock,
					BlockNumber: added[0].Number,
				})
			}
			if len(removed) != 0 {
				lth := len(removed)
				r.feed.Send(Event{
					Type:        EventReorg,
					BlockNumber: removed[lth-1].Number,
				})
			}
		}
	}
}

// getTransfers fetches erc20 and eth transfers and returns single slice with them.
func (r *Reactor) getTransfers(header *DBHeader) ([]Transfer, error) {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	ethT, err := r.eth.GetTransfers(ctx, header)
	cancel()
	if err != nil {
		return nil, err
	}
	ctx, cancel = context.WithTimeout(r.ctx, 5*time.Second)
	erc20T, err := r.erc20.GetTransfers(ctx, header)
	cancel()
	if err != nil {
		return nil, err
	}
	return append(ethT, erc20T...), nil
}

// onNewBlock verifies if latest block extends current canonical chain view. In case if it doesn't it will find common
// parrent and replace all blocks after that parent.
func (r *Reactor) onNewBlock(ctx context.Context, previous *DBHeader, latest *types.Header) (added, removed []*DBHeader, err error) {
	if previous == nil {
		// first node in the cache
		return []*DBHeader{toDBHeader(latest)}, nil, nil
	}
	if previous.Hash == latest.ParentHash {
		// parent matching previous node in the cache. on the same chain.
		return []*DBHeader{toDBHeader(latest)}, nil, nil
	}
	exists, err := r.db.HeaderExists(latest.Hash())
	if err != nil {
		return nil, nil, err
	}
	if exists {
		return nil, nil, nil
	}
	log.Debug("wallet reactor spotted reorg", "last header in db", previous.Hash, "new parent", latest.ParentHash)
	for previous != nil && previous.Hash != latest.ParentHash {
		removed = append(removed, previous)
		added = append(added, toDBHeader(latest))
		latest, err = r.client.HeaderByHash(ctx, latest.ParentHash)
		if err != nil {
			return nil, nil, err
		}
		previous, err = r.db.GetHeaderByNumber(new(big.Int).Sub(latest.Number, one))
		if err != nil {
			return nil, nil, err
		}
	}
	added = append(added, toDBHeader(latest))
	return added, removed, nil
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
