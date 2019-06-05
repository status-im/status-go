package wallet

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

type Command interface {
	Run(context.Context)
}

type FiniteCommand struct {
	Interval time.Duration
	Runable  func(context.Context) error
}

func (c FiniteCommand) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := c.Runable(ctx)
			if err == nil {
				return
			}
		}
	}
}

type InfiniteCommand struct {
	Interval time.Duration
	Runable  func(context.Context) error
}

func (c InfiniteCommand) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = c.Runable(ctx)
		}
	}
}

func NewGroup() *Group {
	ctx, cancel := context.WithCancel(context.Background())
	return &Group{
		ctx:    ctx,
		cancel: cancel,
	}
}

type Group struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
}

func (g *Group) Add(cmd Command) {
	g.wg.Add(1)
	go func() {
		cmd.Run(g.ctx)
		g.wg.Done()
	}()
}

func (g *Group) Stop() {
	g.cancel()
	g.wg.Wait()
}

type ethHistoricalCommand struct {
	db      *Database
	eth     TransferDownloader
	address common.Address
	client  reactorClient
	feed    *event.Feed

	previous *DBHeader
}

func (c *ethHistoricalCommand) Command() FiniteCommand {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}
}

func (c *ethHistoricalCommand) Run(ctx context.Context) (err error) {
	if c.previous == nil {
		c.previous, err = c.db.GetEarliestSynced(c.address, ethSync)
		if err != nil {
			return err
		}
		if c.previous == nil {
			c.previous, err = lastKnownHeader(c.db, c.client)
			if err != nil {
				return err
			}
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	concurrent := NewConcurrentDownloader(ctx)
	start := time.Now()
	downloadEthConcurrently(concurrent, c.client, c.eth, c.address, zero, c.previous.Number)
	concurrent.Wait()
	if concurrent.Error() != nil {
		log.Error("failed to dowloader transfers using concurrent downloader", "error", err)
		return concurrent.Error()
	}
	transfers := concurrent.Transfers()
	log.Info("eth historical downloader finished succesfully", "total transfers", len(transfers), "time", time.Since(start))
	// TODO(dshulyak) insert 0 block number with transfers
	err = c.db.ProcessTranfers(transfers, headersFromTransfers(transfers), nil, ethSync)
	if err != nil {
		log.Error("failed to save downloaded erc20 transfers", "error", err)
		return err
	}
	if len(transfers) > 0 {
		// we download all or nothing
		c.feed.Send(Event{
			Type:        EventNewHistory,
			BlockNumber: zero,
			Accounts:    []common.Address{c.address},
		})
	}
	log.Debug("eth transfers were persisted. command is closed")
	return nil
}

type erc20HistoricalCommand struct {
	db      *Database
	erc20   BatchDownloader
	address common.Address
	client  reactorClient
	feed    *event.Feed

	iterator *IterativeDownloader
}

func (c *erc20HistoricalCommand) Command() FiniteCommand {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}
}

func (c *erc20HistoricalCommand) Run(ctx context.Context) (err error) {
	if c.iterator == nil {
		c.iterator, err = SetupIterativeDownloader(c.db, c.client, c.address, erc20Sync, c.erc20, erc20BatchSize)
		if err != nil {
			log.Error("failed to setup historical downloader for erc20")
			return err
		}
	}
	for !c.iterator.Finished() {
		transfers, err := c.iterator.Next(ctx)
		if err != nil {
			log.Error("failed to get next batch", "error", err)
			break
		}
		headers := headersFromTransfers(transfers)
		headers = append(headers, c.iterator.Header())
		err = c.db.ProcessTranfers(transfers, headers, nil, erc20Sync)
		if err != nil {
			c.iterator.Revert()
			log.Error("failed to save downloaded erc20 transfers", "error", err)
			return err
		}
		if len(transfers) > 0 {
			c.feed.Send(Event{
				Type:        EventNewHistory,
				BlockNumber: c.iterator.Header().Number,
				Accounts:    []common.Address{c.address},
			})
		}
	}
	log.Info("wallet historical downloader for erc20 transfers finished")
	return nil
}

type newBlocksTransfersCommand struct {
	db     *Database
	chain  *big.Int
	erc20  *ERC20TransfersDownloader
	eth    *ETHTransferDownloader
	client reactorClient
	feed   *event.Feed

	previous *DBHeader
}

func (c *newBlocksTransfersCommand) Command() InfiniteCommand {
	return InfiniteCommand{
		Interval: pollingPeriodByChain(c.chain),
		Runable:  c.Run,
	}
}

func (c *newBlocksTransfersCommand) Run(parent context.Context) (err error) {
	if c.previous == nil {
		c.previous, err = lastKnownHeader(c.db, c.client)
		if err != nil {
			log.Error("failed to get last known header", "error", err)
			return err
		}
	}
	num := new(big.Int).Add(c.previous.Number, one)
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	latest, err := c.client.HeaderByNumber(ctx, num)
	cancel()
	if err != nil {
		log.Warn("failed to get latest block", "number", num, "error", err)
		return err
	}
	log.Debug("reactor received new block", "header", latest.Hash())
	ctx, cancel = context.WithTimeout(parent, 10*time.Second)
	added, removed, err := c.onNewBlock(ctx, c.previous, latest)
	cancel()
	if err != nil {
		log.Error("failed to process new header", "header", latest, "error", err)
		return err
	}
	// for each added block get tranfers from downloaders
	all := []Transfer{}
	for i := range added {
		log.Debug("reactor get transfers", "block", added[i].Hash, "number", added[i].Number)
		transfers, err := c.getTransfers(parent, added[i])
		if err != nil {
			log.Error("failed to get transfers", "header", added[i].Hash, "error", err)
			continue
		}
		log.Debug("reactor adding transfers", "block", added[i].Hash, "number", added[i].Number, "len", len(transfers))
		all = append(all, transfers...)
	}
	err = c.db.ProcessTranfers(all, added, removed, erc20Sync|ethSync)
	if err != nil {
		log.Error("failed to persist transfers", "error", err)
		return err
	}
	c.previous = toDBHeader(latest)
	if len(added) == 1 && len(removed) == 0 {
		c.feed.Send(Event{
			Type:        EventNewBlock,
			BlockNumber: added[0].Number,
			Accounts:    uniqueAccountsFromTransfers(all),
		})
	}
	if len(removed) != 0 {
		lth := len(removed)
		c.feed.Send(Event{
			Type:        EventReorg,
			BlockNumber: removed[lth-1].Number,
			Accounts:    uniqueAccountsFromTransfers(all),
		})
	}
	return nil
}

func (c *newBlocksTransfersCommand) onNewBlock(ctx context.Context, previous *DBHeader, latest *types.Header) (added, removed []*DBHeader, err error) {
	if previous == nil {
		// first node in the cache
		return []*DBHeader{toDBHeader(latest)}, nil, nil
	}
	if previous.Hash == latest.ParentHash {
		// parent matching previous node in the cache. on the same chain.
		return []*DBHeader{toDBHeader(latest)}, nil, nil
	}
	exists, err := c.db.HeaderExists(latest.Hash())
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
		latest, err = c.client.HeaderByHash(ctx, latest.ParentHash)
		if err != nil {
			return nil, nil, err
		}
		previous, err = c.db.GetHeaderByNumber(new(big.Int).Sub(latest.Number, one))
		if err != nil {
			return nil, nil, err
		}
	}
	added = append(added, toDBHeader(latest))
	return added, removed, nil
}

func (c *newBlocksTransfersCommand) getTransfers(parent context.Context, header *DBHeader) ([]Transfer, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	ethT, err := c.eth.GetTransfers(ctx, header)
	cancel()
	if err != nil {
		return nil, err
	}
	ctx, cancel = context.WithTimeout(parent, 5*time.Second)
	erc20T, err := c.erc20.GetTransfers(ctx, header)
	cancel()
	if err != nil {
		return nil, err
	}
	return append(ethT, erc20T...), nil
}

func uniqueAccountsFromTransfers(transfers []Transfer) []common.Address {
	accounts := []common.Address{}
	unique := map[common.Address]struct{}{}
	for i := range transfers {
		_, exist := unique[transfers[i].Address]
		if exist {
			continue
		}
		unique[transfers[i].Address] = struct{}{}
		accounts = append(accounts, transfers[i].Address)
	}
	return accounts
}
