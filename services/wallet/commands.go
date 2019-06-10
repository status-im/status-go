package wallet

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

type ethHistoricalCommand struct {
	db      *Database
	eth     TransferDownloader
	address common.Address
	client  reactorClient
	feed    *event.Feed

	from, to *big.Int
}

func (c *ethHistoricalCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *ethHistoricalCommand) Run(ctx context.Context) (err error) {
	if c.from == nil {
		from, err := c.db.GetLatestSynced(c.address, ethSync)
		if err != nil {
			return err
		}
		if from == nil {
			c.from = zero
		} else {
			c.from = from.Number
		}
		log.Debug("initialized downloader for eth historical transfers", "address", c.address, "starting at", c.from, "up to", c.to)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	concurrent := NewConcurrentDownloader(ctx)
	start := time.Now()
	downloadEthConcurrently(concurrent, c.client, c.eth, c.address, c.from, c.to)
	select {
	case <-concurrent.WaitAsync():
	case <-ctx.Done():
		log.Error("eth downloader is stuck")
		return errors.New("eth downloader is stuck")
	}
	if concurrent.Error() != nil {
		log.Error("failed to dowload transfers using concurrent downloader", "error", err)
		return concurrent.Error()
	}
	transfers := concurrent.Get()
	log.Info("eth historical downloader finished succesfully", "total transfers", len(transfers), "time", time.Since(start))
	// TODO(dshulyak) insert 0 block number with transfers
	err = c.db.ProcessTranfers(transfers, []common.Address{c.address}, headersFromTransfers(transfers), nil, ethSync)
	if err != nil {
		log.Error("failed to save downloaded erc20 transfers", "error", err)
		return err
	}
	if len(transfers) > 0 {
		// we download all or nothing
		c.feed.Send(Event{
			Type:        EventNewBlock,
			BlockNumber: c.from,
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
	to       *DBHeader
}

func (c *erc20HistoricalCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *erc20HistoricalCommand) Run(ctx context.Context) (err error) {
	if c.iterator == nil {
		c.iterator, err = SetupIterativeDownloader(
			c.db, c.client, c.address, erc20Sync,
			c.erc20, erc20BatchSize, c.to)
		if err != nil {
			log.Error("failed to setup historical downloader for erc20")
			return err
		}
	}
	for !c.iterator.Finished() {
		start := time.Now()
		transfers, err := c.iterator.Next(ctx)
		if err != nil {
			log.Error("failed to get next batch", "error", err)
			break
		}
		headers := headersFromTransfers(transfers)
		headers = append(headers, c.iterator.Header())
		err = c.db.ProcessTranfers(transfers, []common.Address{c.address}, headers, nil, erc20Sync)
		if err != nil {
			c.iterator.Revert()
			log.Error("failed to save downloaded erc20 transfers", "error", err)
			return err
		}
		if len(transfers) > 0 {
			log.Debug("erc20 downloader imported transfers", "len", len(transfers), "time", time.Since(start))
			c.feed.Send(Event{
				Type:        EventNewBlock,
				BlockNumber: c.iterator.Header().Number,
				Accounts:    []common.Address{c.address},
			})
		}
	}
	log.Info("wallet historical downloader for erc20 transfers finished")
	return nil
}

type newBlocksTransfersCommand struct {
	db       *Database
	accounts []common.Address
	chain    *big.Int
	erc20    *ERC20TransfersDownloader
	eth      *ETHTransferDownloader
	client   reactorClient
	feed     *event.Feed

	from, to *DBHeader
}

func (c *newBlocksTransfersCommand) Command() Command {
	// if both blocks are specified we will use this command to verify that lastly synced blocks are still
	// in canonical chain
	if c.to != nil && c.from != nil {
		return FiniteCommand{
			Interval: 5 * time.Second,
			Runable:  c.Verify,
		}.Run
	}
	return InfiniteCommand{
		Interval: pollingPeriodByChain(c.chain),
		Runable:  c.Run,
	}.Run
}

func (c *newBlocksTransfersCommand) Verify(parent context.Context) (err error) {
	if c.to == nil || c.from == nil {
		return errors.New("`from` and `to` blocks must be specified")
	}
	for c.from.Number.Cmp(c.to.Number) != 0 {
		err = c.Run(parent)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *newBlocksTransfersCommand) Run(parent context.Context) (err error) {
	if c.from == nil {
		ctx, cancel := context.WithTimeout(parent, 3*time.Second)
		from, err := c.client.HeaderByNumber(ctx, nil)
		cancel()
		if err != nil {
			log.Error("failed to get last known header", "error", err)
			return err
		}
		c.from = toDBHeader(from)
		log.Debug("initialized downloader for new blocks transfers", "starting at", c.from.Number)
	}
	num := new(big.Int).Add(c.from.Number, one)
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	latest, err := c.client.HeaderByNumber(ctx, num)
	cancel()
	if err != nil {
		log.Warn("failed to get latest block", "number", num, "error", err)
		return err
	}
	log.Debug("reactor received new block", "header", latest.Hash())
	ctx, cancel = context.WithTimeout(parent, 10*time.Second)
	added, removed, err := c.onNewBlock(ctx, c.from, latest)
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
	err = c.db.ProcessTranfers(all, c.accounts, added, removed, erc20Sync|ethSync)
	if err != nil {
		log.Error("failed to persist transfers", "error", err)
		return err
	}
	c.from = toDBHeader(latest)
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

func (c *newBlocksTransfersCommand) onNewBlock(ctx context.Context, from *DBHeader, latest *types.Header) (added, removed []*DBHeader, err error) {
	if from == nil {
		// first node in the cache
		return []*DBHeader{toHead(latest)}, nil, nil
	}
	if from.Hash == latest.ParentHash {
		// parent matching from node in the cache. on the same chain.
		return []*DBHeader{toHead(latest)}, nil, nil
	}
	exists, err := c.db.HeaderExists(latest.Hash())
	if err != nil {
		return nil, nil, err
	}
	if exists {
		return nil, nil, nil
	}
	log.Debug("wallet reactor spotted reorg", "last header in db", from.Hash, "new parent", latest.ParentHash)
	for from != nil && from.Hash != latest.ParentHash {
		removed = append(removed, from)
		added = append(added, toHead(latest))
		latest, err = c.client.HeaderByHash(ctx, latest.ParentHash)
		if err != nil {
			return nil, nil, err
		}
		from, err = c.db.GetHeaderByNumber(new(big.Int).Sub(latest.Number, one))
		if err != nil {
			return nil, nil, err
		}
	}
	added = append(added, toHead(latest))
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

type controlCommand struct {
	accounts    []common.Address
	db          *Database
	eth         *ETHTransferDownloader
	erc20       *ERC20TransfersDownloader
	chain       *big.Int
	client      *ethclient.Client
	feed        *event.Feed
	safetyDepth *big.Int
}

// run fast indexing for every accont managed by this command.
func (c *controlCommand) fastIndex(ctx context.Context, to *DBHeader) error {
	start := time.Now()
	group := NewGroup(ctx)
	for _, address := range c.accounts {
		erc20 := &erc20HistoricalCommand{
			db:      c.db,
			erc20:   NewERC20TransfersDownloader(c.client, []common.Address{address}),
			client:  c.client,
			feed:    c.feed,
			address: address,
			to:      to,
		}
		group.Add(erc20.Command())
		eth := &ethHistoricalCommand{
			db:      c.db,
			client:  c.client,
			address: address,
			eth: &ETHTransferDownloader{
				client:   c.client,
				accounts: []common.Address{address},
				signer:   types.NewEIP155Signer(c.chain),
			},
			feed: c.feed,
			to:   to.Number,
		}
		group.Add(eth.Command())
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Debug("fast indexer finished", "in", time.Since(start))
		return nil
	}
}

func (c *controlCommand) verifyLastSynced(parent context.Context, last *DBHeader, head *types.Header) error {
	log.Debug("verifying that previous header is still in canonical chan", "from", last.Number, "chain head", head.Number)
	if new(big.Int).Sub(head.Number, last.Number).Cmp(c.safetyDepth) <= 0 {
		log.Debug("no need to verify. last block is close enough to chain head")
		return nil
	}
	header, err := c.client.HeaderByNumber(parent, new(big.Int).Add(last.Number, c.safetyDepth))
	if err != nil {
		return err
	}
	log.Debug("spawn reorg verifier", "from", last.Number, "to", header.Number)
	cmd := &newBlocksTransfersCommand{
		db:       c.db,
		chain:    c.chain,
		client:   c.client,
		accounts: c.accounts,
		eth:      c.eth,
		erc20:    c.erc20,
		feed:     c.feed,

		from: last,
		to:   toDBHeader(header),
	}
	return cmd.Command()(parent)
}

func (c *controlCommand) Run(parent context.Context) error {
	log.Debug("start control command")
	head, err := c.client.HeaderByNumber(parent, nil)
	if err != nil {
		return err
	}
	log.Debug("current head is", "block number", head.Number)
	last, err := c.db.GetLastHead()
	if err != nil {
		log.Error("failed to load last head from database", "error", err)
		return err
	}
	if last != nil {
		err = c.verifyLastSynced(parent, last, head)
		if err != nil {
			log.Error("failed verification for last header in canonical chain", "error", err)
			return err
		}
	}
	target := new(big.Int).Sub(head.Number, c.safetyDepth)
	if target.Cmp(zero) <= 0 {
		target = zero
	}
	head, err = c.client.HeaderByNumber(parent, target)
	if err != nil {
		return err
	}
	log.Debug("run fast indexing for the transfers", "up to", head.Number)
	err = c.fastIndex(parent, toDBHeader(head))
	if err != nil {
		return err
	}
	log.Debug("watching new blocks", "start from", head.Number)
	cmd := &newBlocksTransfersCommand{
		db:       c.db,
		chain:    c.chain,
		client:   c.client,
		accounts: c.accounts,
		eth:      c.eth,
		erc20:    c.erc20,
		feed:     c.feed,
		from:     toDBHeader(head),
	}
	return cmd.Command()(parent)
}

func (c *controlCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
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
