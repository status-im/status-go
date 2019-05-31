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
	default:
		return 500 * time.Millisecond
	}
}

// HeaderReader interface for reading headers using block number or hash.
type HeaderReader interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
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
	client  HeaderReader
	db      *Database
	feed    *event.Feed
	address common.Address
	chain   *big.Int

	eth   *ETHTransferDownloader
	erc20 *ERC20TransfersDownloader

	wg   sync.WaitGroup
	quit chan struct{}
}

// Start runs reactor loop in background.
func (r *Reactor) Start() error {
	if r.quit != nil {
		return errors.New("already running")
	}
	r.quit = make(chan struct{})
	r.wg.Add(1)
	go func() {
		log.Info("wallet reactor started", "address", r.address.String())
		r.loop()
		r.wg.Done()
	}()
	return nil
}

// Stop stops reactor loop and waits till it exits.
func (r *Reactor) Stop() {
	if r.quit == nil {
		return
	}
	close(r.quit)
	r.wg.Wait()
	r.quit = nil
}

func (r *Reactor) loop() {
	var (
		ticker = time.NewTicker(pollingPeriodByChain(r.chain))
		latest *types.Header
		err    error
	)
	defer ticker.Stop()
	for {
		select {
		case <-r.quit:
			return
		case <-ticker.C:
			var num *big.Int
			if latest == nil {
				latest, err = r.db.LastHeader()
				if err != nil {
					log.Error("failed to read last header from database", "error", err)
					continue
				}
			}
			if latest != nil {
				num = new(big.Int).Add(latest.Number, one)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			header, err := r.client.HeaderByNumber(ctx, num)
			cancel()
			if err != nil {
				log.Error("failed to get latest block", "number", latest, "error", err)
				continue
			}
			log.Debug("reactor received new block", "header", header.Hash())
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			added, removed, err := r.onNewBlock(ctx, latest, header)
			cancel()
			if err != nil {
				log.Error("failed to process new header", "header", header, "error", err)
				continue
			}
			// for each added block get tranfers from downloaders
			all := []Transfer{}
			for i := range added {
				log.Debug("reactor get transfers", "block", added[i].Hash(), "number", added[i].Number)
				transfers, err := r.getTransfers(added[i])
				if err != nil {
					log.Error("failed to get transfers", "header", header, "error", err)
					continue
				}
				log.Debug("reactor adding transfers", "block", added[i].Hash(), "number", added[i].Number, "len", len(transfers))
				all = append(all, transfers...)
			}
			err = r.db.ProcessTranfers(all, added, removed)
			if err != nil {
				log.Error("failed to persist transfers", "error", err)
				continue
			}
			latest = header

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
func (r *Reactor) getTransfers(header *types.Header) ([]Transfer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ethT, err := r.eth.GetTransfers(ctx, header)
	cancel()
	if err != nil {
		return nil, err
	}
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	erc20T, err := r.erc20.GetTransfers(ctx, header)
	cancel()
	if err != nil {
		return nil, err
	}
	return append(ethT, erc20T...), nil
}

// onNewBlock verifies if latest block extends current canonical chain view. In case if it doesn't it will find common
// parrent and replace all blocks after that parent.
func (r *Reactor) onNewBlock(ctx context.Context, previous, latest *types.Header) (added, removed []*types.Header, err error) {
	if previous == nil {
		// first node in the cache
		return []*types.Header{latest}, nil, nil
	}
	if previous.Hash() == latest.ParentHash {
		// parent matching previous node in the cache. on the same chain.
		return []*types.Header{latest}, nil, nil
	}
	exists, err := r.db.HeaderExists(latest.Hash())
	if err != nil {
		return nil, nil, err
	}
	if exists {
		return nil, nil, nil
	}
	// reorg
	log.Debug("wallet reactor spotted reorg", "last header in db", previous.Hash(), "new parent", latest.ParentHash)
	for previous != nil && previous.Hash() != latest.ParentHash {
		removed = append(removed, previous)
		added = append(added, latest)
		latest, err = r.client.HeaderByHash(ctx, latest.ParentHash)
		if err != nil {
			return nil, nil, err
		}
		previous, err = r.db.GetHeaderByNumber(new(big.Int).Sub(latest.Number, one))
		if err != nil {
			return nil, nil, err
		}
	}
	added = append(added, latest)
	return added, removed, nil
}
