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
	db           *Database
	eth          TransferDownloader
	address      common.Address
	client       reactorClient
	balanceCache *balanceCache
	feed         *event.Feed

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
	downloadEthConcurrently(concurrent, c.client, c.balanceCache, c.eth, c.address, c.from, c.to)
	select {
	case <-concurrent.WaitAsync():
	case <-ctx.Done():
		log.Error("eth downloader is stuck")
		return errors.New("eth downloader is stuck")
	}
	if concurrent.Error() != nil {
		log.Error("failed to dowload transfers using concurrent downloader", "error", concurrent.Error())
		return concurrent.Error()
	}
	transfers := concurrent.Get()
	blocks := concurrent.GetBlocks()
	log.Info("eth historical downloader finished successfully", "address", c.address, "total transfers", len(transfers), "total blocks", len(blocks), "blocks", blocks, "time", time.Since(start))
	//err = c.db.ProcessTranfers(transfers, []common.Address{c.address}, headersFromTransfers(transfers), nil, ethSync)
	err = c.db.ProcessBlocks(c.address, c.from, c.to, blocks, ethTransfer)
	if err != nil {
		log.Error("failed to save downloaded eth transfers", "error", err)
		return err
	}
	/*
		if len(transfers) > 0 {
			// we download all or nothing
			c.feed.Send(Event{
				Type:        EventNewHistory,
				BlockNumber: c.from,
				Accounts:    []common.Address{c.address},
				ERC20:       false,
			})
		}
	*/
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
	to       *big.Int
	from     *big.Int
}

func (c *erc20HistoricalCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *erc20HistoricalCommand) Run(ctx context.Context) (err error) {
	start := time.Now()
	if c.iterator == nil {
		c.iterator, err = SetupIterativeDownloader(
			c.db, c.client, c.address, erc20Sync,
			c.erc20, erc20BatchSize, c.to, c.from)
		if err != nil {
			log.Error("failed to setup historical downloader for erc20")
			return err
		}
	}
	for !c.iterator.Finished() {
		blocks, from, to, err := c.iterator.Next(ctx)
		if err != nil {
			log.Error("failed to get next batch", "error", err)
			return err
		}
		err = c.db.ProcessBlocks(c.address, from, to, blocks, erc20Transfer)
		if err != nil {
			c.iterator.Revert()
			log.Error("failed to save downloaded erc20 blocks with transfers", "error", err)
			return err
		}
	}
	log.Info("wallet historical downloader for erc20 transfers finished", "in", time.Since(start))
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
	if len(added) == 0 && len(removed) == 0 {
		log.Debug("new block already in the database", "block", latest.Number)
		return nil
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

// controlCommand implements following procedure (following parts are executed sequeantially):
// - verifies that the last header that was synced is still in the canonical chain
// - runs fast indexing for each account separately
// - starts listening to new blocks and watches for reorgs
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

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findAndCheckBlockRangeCommand) fastIndex(ctx context.Context, bCache *balanceCache, fromByAddress map[common.Address]*big.Int, toByAddress map[common.Address]*big.Int) error {
	start := time.Now()
	group := NewGroup(ctx)

	for _, address := range c.accounts {
		eth := &ethHistoricalCommand{
			db:           c.db,
			client:       c.client,
			balanceCache: bCache,
			address:      address,
			eth: &ETHTransferDownloader{
				client:   c.client,
				accounts: []common.Address{address},
				signer:   types.NewEIP155Signer(c.chain),
			},
			feed: c.feed,
			from: fromByAddress[address],
			to:   toByAddress[address],
		}
		group.Add(eth.Command())
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Info("fast indexer finished", "in", time.Since(start))
		return nil
	}
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findAndCheckBlockRangeCommand) fastIndexErc20(ctx context.Context, fromByAddress map[common.Address]*big.Int, toByAddress map[common.Address]*big.Int) error {
	start := time.Now()
	group := NewGroup(ctx)

	for _, address := range c.accounts {
		erc20 := &erc20HistoricalCommand{
			db:      c.db,
			erc20:   NewERC20TransfersDownloader(c.client, []common.Address{address}, types.NewEIP155Signer(c.chain)),
			client:  c.client,
			feed:    c.feed,
			address: address,
			from:    fromByAddress[address],
			to:      toByAddress[address],
		}
		group.Add(erc20.Command())
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Info("fast indexer Erc20 finished", "in", time.Since(start))
		return nil
	}
}

func getTransfersByBlocks(ctx context.Context, db *Database, downloader *ETHTransferDownloader, address common.Address, blocks []*big.Int) ([]Transfer, error) {
	allTransfers := []Transfer{}

	for _, block := range blocks {
		transfers, err := downloader.GetTransfersByNumber(ctx, block)
		if err != nil {
			return nil, err
		}
		log.Info("loadTransfers", "block", block, "new transfers", len(transfers))
		if len(transfers) == 0 {
			db.RemoveBlockWithTransfer(address, block)
		} else {
			allTransfers = append(allTransfers, transfers...)
		}
	}

	return allTransfers, nil
}

func loadTransfers(ctx context.Context, accounts []common.Address, db *Database, client *ethclient.Client, chain *big.Int, limit int) error {
	start := time.Now()
	group := NewGroup(ctx)

	for _, address := range accounts {
		blocks, _ := db.GetBlocksByAddress(address, 40)
		for _, block := range blocks {
			erc20 := &transfersCommand{
				db:      db,
				client:  client,
				address: address,
				eth: &ETHTransferDownloader{
					client:   client,
					accounts: []common.Address{address},
					signer:   types.NewEIP155Signer(chain),
				},
				block: block,
			}
			group.Add(erc20.Command())
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Info("loadTransfers finished", "in", time.Since(start))
		return nil
	}
}

func (c *controlCommand) LoadTransfers(ctx context.Context, downloader *ETHTransferDownloader, limit int) error {
	return loadTransfers(ctx, c.accounts, c.db, c.client, c.chain, limit)
}

// verifyLastSynced verifies that last header that was added to the database is still in the canonical chain.
// it is done by downloading configured number of parents for the last header in the db.
func (c *controlCommand) verifyLastSynced(parent context.Context, last *DBHeader, head *types.Header) error {
	log.Debug("verifying that previous header is still in canonical chan", "from", last.Number, "chain head", head.Number)
	if new(big.Int).Sub(head.Number, last.Number).Cmp(c.safetyDepth) <= 0 {
		log.Debug("no need to verify. last block is close enough to chain head")
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	header, err := c.client.HeaderByNumber(ctx, new(big.Int).Add(last.Number, c.safetyDepth))
	cancel()
	if err != nil {
		return err
	}
	log.Debug("spawn reorg verifier", "from", last.Number, "to", header.Number)
	// TODO(dshulyak) make a standalone command that
	// doesn't manage transfers and has an upper limit
	cmd := &newBlocksTransfersCommand{
		db:     c.db,
		chain:  c.chain,
		client: c.client,
		eth:    c.eth,
		erc20:  c.erc20,
		feed:   c.feed,

		from: last,
		to:   toDBHeader(header),
	}
	return cmd.Command()(parent)
}

func findFirstRange(c context.Context, account common.Address, initialTo *big.Int, client *ethclient.Client) (*big.Int, error) {
	from := big.NewInt(0)
	to := initialTo
	goal := uint64(20)

	firstNonce, err := client.NonceAt(c, account, to)
	log.Info("find range with 20 <= len(tx) <= 25", "account", account, "firstNonce", firstNonce, "from", from, "to", to)

	if err != nil {
		return nil, err
	}

	if firstNonce <= goal {
		return zero, nil
	}

	nonceDiff := firstNonce
	iterations := 0
	for iterations < 50 {
		iterations = iterations + 1

		if nonceDiff > goal {
			// from = (from + to) / 2
			from = from.Add(from, to)
			from = from.Div(from, big.NewInt(2))
		} else {
			// from = from - (from + to) / 2
			// to = from
			diff := big.NewInt(0).Sub(to, from)
			diff.Div(diff, big.NewInt(2))
			to = big.NewInt(from.Int64())
			from.Sub(from, diff)
		}
		fromNonce, err := client.NonceAt(c, account, from)
		if err != nil {
			return nil, err
		}
		nonceDiff = firstNonce - fromNonce

		log.Info("next nonce", "from", from, "n", fromNonce, "diff", firstNonce-fromNonce)

		if goal <= nonceDiff && nonceDiff <= (goal+5) {
			log.Info("range found", "account", account, "from", from, "to", to)
			return from, nil
		}
	}

	log.Info("range found", "account", account, "from", from, "to", to)

	return from, nil
}

func findFirstRanges(c context.Context, accounts []common.Address, initialTo *big.Int, client *ethclient.Client) (map[common.Address]*big.Int, error) {
	res := map[common.Address]*big.Int{}

	for _, address := range accounts {
		from, err := findFirstRange(c, address, initialTo, client)
		if err != nil {
			return nil, err
		}

		res[address] = from
	}

	return res, nil
}

func (c *controlCommand) Run(parent context.Context) error {
	log.Debug("start control command")
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	head, err := c.client.HeaderByNumber(ctx, nil)
	cancel()
	if err != nil {
		return err
	}

	c.feed.Send(Event{
		Type: EventFetchingRecentHistory,
	})

	log.Debug("current head is", "block number", head.Number)
	lastKnownEthBlocks, accountsWithoutHistory, err := c.db.GetLastKnownBlockByAddresses(c.accounts, ethTransfer)
	if err != nil {
		log.Error("failed to load last head from database", "error", err)
		return err
	}

	fromMap, err := findFirstRanges(parent, accountsWithoutHistory, head.Number, c.client)
	if err != nil {
		return err
	}

	target := new(big.Int).Sub(head.Number, c.safetyDepth)
	if target.Cmp(zero) <= 0 {
		target = zero
	}
	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	head, err = c.client.HeaderByNumber(ctx, target)
	cancel()
	if err != nil {
		return err
	}

	fromByAddress := map[common.Address]*big.Int{}
	toByAddress := map[common.Address]*big.Int{}

	for _, address := range c.accounts {
		from, ok := lastKnownEthBlocks[address]
		if !ok {
			from = fromMap[address]
		}

		fromByAddress[address] = from
		toByAddress[address] = head.Number
	}

	bCache := newBalanceCache()
	cmnd := &findAndCheckBlockRangeCommand{
		accounts:      c.accounts,
		db:            c.db,
		chain:         c.chain,
		client:        c.client,
		balanceCache:  bCache,
		feed:          c.feed,
		fromByAddress: fromByAddress,
		toByAddress:   toByAddress,
	}

	err = cmnd.Command()(parent)

	downloader := &ETHTransferDownloader{
		client:   c.client,
		accounts: c.accounts,
		signer:   types.NewEIP155Signer(c.chain),
	}
	err = c.LoadTransfers(parent, downloader, 40)
	if err != nil {
		return err
	}

	c.feed.Send(Event{
		Type: EventRecentHistoryReady,
	})

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
			Hash:      transfers[i].BlockHash,
			Number:    transfers[i].BlockNumber,
			Timestamp: transfers[i].Timestamp,
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

type transfersCommand struct {
	db             *Database
	eth            *ETHTransferDownloader
	block          *big.Int
	address        common.Address
	client         reactorClient
	feed           *event.Feed
	iterator       *IterativeDownloader
	controlCommand *controlCommand
}

func (c *transfersCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *transfersCommand) Run(ctx context.Context) (err error) {
	allTransfers, err := getTransfersByBlocks(ctx, c.db, c.eth, c.address, []*big.Int{c.block})
	if err != nil {
		log.Info("getTransfersByBlocks error", "error", err)
		return err
	}

	err = c.db.SaveTranfers(c.address, allTransfers)
	if err != nil {
		log.Info("SaveTranfers error", "error", err)
		return err
	}

	log.Info("transfers loaded", "address", c.address, "len", len(allTransfers))
	return nil
}

type loadTransfersCommand struct {
	accounts []common.Address
	db       *Database
	chain    *big.Int
	client   *ethclient.Client
}

func (c *loadTransfersCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *loadTransfersCommand) LoadTransfers(ctx context.Context, downloader *ETHTransferDownloader, limit int) error {
	return loadTransfers(ctx, c.accounts, c.db, c.client, c.chain, limit)
}

func (c *loadTransfersCommand) Run(parent context.Context) (err error) {
	log.Debug("start loadTransfersCommand")

	downloader := &ETHTransferDownloader{
		client:   c.client,
		accounts: c.accounts,
		signer:   types.NewEIP155Signer(c.chain),
	}
	err = c.LoadTransfers(parent, downloader, 40)
	if err != nil {
		return err
	}

	return
}

type findAndCheckBlockRangeCommand struct {
	accounts      []common.Address
	db            *Database
	chain         *big.Int
	client        *ethclient.Client
	balanceCache  *balanceCache
	feed          *event.Feed
	fromByAddress map[common.Address]*big.Int
	toByAddress   map[common.Address]*big.Int
}

func (c *findAndCheckBlockRangeCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *findAndCheckBlockRangeCommand) Run(parent context.Context) (err error) {
	log.Debug("start findAndCHeckBlockRangeCommand")
	err = c.fastIndex(parent, c.balanceCache, c.fromByAddress, c.toByAddress)
	if err != nil {
		return err
	}
	err = c.fastIndexErc20(parent, c.fromByAddress, c.toByAddress)
	if err != nil {
		return err
	}

	return
}
