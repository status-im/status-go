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

var numberOfBlocksCheckedPerIteration = 40
var blocksDelayThreshhold = 40 * time.Second

type ethHistoricalCommand struct {
	db           *Database
	eth          TransferDownloader
	address      common.Address
	client       reactorClient
	balanceCache *balanceCache
	feed         *event.Feed
	foundHeaders []*DBHeader
	noLimit      bool

	from, to, resultingFrom *big.Int
}

func (c *ethHistoricalCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *ethHistoricalCommand) Run(ctx context.Context) (err error) {
	start := time.Now()
	totalRequests, cacheHits := c.balanceCache.getStats(c.address)
	log.Info("balance cache before checking range", "total", totalRequests, "cached", totalRequests-cacheHits)
	from, headers, err := findBlocksWithEthTransfers(ctx, c.client, c.balanceCache, c.eth, c.address, c.from, c.to, c.noLimit)

	if err != nil {
		return err
	}

	c.foundHeaders = headers
	c.resultingFrom = from

	log.Info("eth historical downloader finished successfully", "address", c.address, "from", from, "to", c.to, "total blocks", len(headers), "time", time.Since(start))
	totalRequests, cacheHits = c.balanceCache.getStats(c.address)
	log.Info("balance cache after checking range", "total", totalRequests, "cached", totalRequests-cacheHits)

	//err = c.db.ProcessBlocks(c.address, from, c.to, headers, ethTransfer)
	if err != nil {
		log.Error("failed to save found blocks with transfers", "error", err)
		return err
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

	iterator     *IterativeDownloader
	to           *big.Int
	from         *big.Int
	foundHeaders []*DBHeader
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
			c.db, c.client, c.address,
			c.erc20, erc20BatchSize, c.to, c.from)
		if err != nil {
			log.Error("failed to setup historical downloader for erc20")
			return err
		}
	}
	for !c.iterator.Finished() {
		headers, _, _, err := c.iterator.Next(ctx)
		if err != nil {
			log.Error("failed to get next batch", "error", err)
			return err
		}
		c.foundHeaders = append(c.foundHeaders, headers...)

		/*err = c.db.ProcessBlocks(c.address, from, to, headers, erc20Transfer)
		if err != nil {
			c.iterator.Revert()
			log.Error("failed to save downloaded erc20 blocks with transfers", "error", err)
			return err
		}*/
	}
	log.Info("wallet historical downloader for erc20 transfers finished", "in", time.Since(start))
	return nil
}

type newBlocksTransfersCommand struct {
	db                   *Database
	accounts             []common.Address
	chain                *big.Int
	erc20                *ERC20TransfersDownloader
	eth                  *ETHTransferDownloader
	client               reactorClient
	feed                 *event.Feed
	lastFetchedBlockTime time.Time

	initialFrom, from, to *DBHeader
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

func (c *newBlocksTransfersCommand) getAllTransfers(parent context.Context, from, to uint64) (map[common.Address][]Transfer, error) {
	transfersByAddress := map[common.Address][]Transfer{}
	if to-from > reorgSafetyDepth(c.chain).Uint64() {
		fromByAddress := map[common.Address]*big.Int{}
		toByAddress := map[common.Address]*big.Int{}
		for _, account := range c.accounts {
			fromByAddress[account] = new(big.Int).SetUint64(from)
			toByAddress[account] = new(big.Int).SetUint64(to)
		}

		balanceCache := newBalanceCache()
		blocksCommand := &findAndCheckBlockRangeCommand{
			accounts:      c.accounts,
			db:            c.db,
			chain:         c.chain,
			client:        c.eth.client,
			balanceCache:  balanceCache,
			feed:          c.feed,
			fromByAddress: fromByAddress,
			toByAddress:   toByAddress,
			noLimit:       true,
		}

		if err := blocksCommand.Command()(parent); err != nil {
			return nil, err
		}

		for address, headers := range blocksCommand.foundHeaders {
			blocks := make([]*big.Int, len(headers))
			for i, header := range headers {
				blocks[i] = header.Number
			}
			txCommand := &loadTransfersCommand{
				accounts:        []common.Address{address},
				db:              c.db,
				chain:           c.chain,
				client:          c.erc20.client,
				blocksByAddress: map[common.Address][]*big.Int{address: blocks},
			}

			err := txCommand.Command()(parent)
			if err != nil {
				return nil, err
			}

			transfersByAddress[address] = txCommand.foundTransfersByAddress[address]
		}
	} else {
		all := []Transfer{}
		newHeadersByAddress := map[common.Address][]*DBHeader{}
		for n := from; n <= to; n++ {
			ctx, cancel := context.WithTimeout(parent, 10*time.Second)
			header, err := c.client.HeaderByNumber(ctx, big.NewInt(int64(n)))
			cancel()
			if err != nil {
				return nil, err
			}
			dbHeader := toDBHeader(header)
			log.Info("reactor get transfers", "block", dbHeader.Hash, "number", dbHeader.Number)
			transfers, err := c.getTransfers(parent, dbHeader)
			if err != nil {
				log.Error("failed to get transfers", "header", dbHeader.Hash, "error", err)
				return nil, err
			}
			if len(transfers) > 0 {
				for _, transfer := range transfers {
					headers, ok := newHeadersByAddress[transfer.Address]
					if !ok {
						headers = []*DBHeader{}
					}

					transfers, ok := transfersByAddress[transfer.Address]
					if !ok {
						transfers = []Transfer{}
					}
					transfersByAddress[transfer.Address] = append(transfers, transfer)
					newHeadersByAddress[transfer.Address] = append(headers, dbHeader)
				}
			}
			all = append(all, transfers...)
		}

		err := c.saveHeaders(parent, newHeadersByAddress)
		if err != nil {
			return nil, err
		}

		err = c.db.ProcessTranfers(all, nil)
		if err != nil {
			log.Error("failed to persist transfers", "error", err)
			return nil, err
		}
	}

	return transfersByAddress, nil
}

func (c *newBlocksTransfersCommand) saveHeaders(parent context.Context, newHeadersByAddress map[common.Address][]*DBHeader) (err error) {
	for _, address := range c.accounts {
		headers, ok := newHeadersByAddress[address]
		if ok {
			err = c.db.SaveBlocks(address, headers)
			if err != nil {
				log.Error("failed to persist blocks", "error", err)
				return err
			}
		}
	}

	return nil
}

func (c *newBlocksTransfersCommand) checkDelay(parent context.Context, nextHeader *types.Header) (*types.Header, error) {
	if time.Since(c.lastFetchedBlockTime) > blocksDelayThreshhold {
		log.Info("There was a delay before loading next block", "time since previous successful fetching", time.Since(c.lastFetchedBlockTime))
		ctx, cancel := context.WithTimeout(parent, 5*time.Second)
		latestHeader, err := c.client.HeaderByNumber(ctx, nil)
		cancel()
		if err != nil {
			log.Warn("failed to get latest block", "number", nextHeader.Number, "error", err)
			return nil, err
		}
		diff := new(big.Int).Sub(latestHeader.Number, nextHeader.Number)
		if diff.Cmp(reorgSafetyDepth(c.chain)) >= 0 {
			num := new(big.Int).Sub(latestHeader.Number, reorgSafetyDepth(c.chain))
			ctx, cancel := context.WithTimeout(parent, 5*time.Second)
			nextHeader, err = c.client.HeaderByNumber(ctx, num)
			cancel()
			if err != nil {
				log.Warn("failed to get next block", "number", num, "error", err)
				return nil, err
			}
		}
	}

	return nextHeader, nil
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
	}
	num := new(big.Int).Add(c.from.Number, one)
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	nextHeader, err := c.client.HeaderByNumber(ctx, num)
	cancel()
	if err != nil {
		log.Warn("failed to get next block", "number", num, "error", err)
		return err
	}
	log.Info("reactor received new block", "header", num)

	nextHeader, err = c.checkDelay(parent, nextHeader)
	if err != nil {
		return err
	}

	ctx, cancel = context.WithTimeout(parent, 10*time.Second)
	latestHeader, removed, latestValidSavedBlock, reorgSpotted, err := c.onNewBlock(ctx, c.from, nextHeader)
	cancel()
	if err != nil {
		log.Error("failed to process new header", "header", nextHeader, "error", err)
		return err
	}

	err = c.db.ProcessTranfers(nil, removed)
	if err != nil {
		return err
	}

	latestHeader.Loaded = true

	fromN := nextHeader.Number.Uint64()

	if reorgSpotted {
		if latestValidSavedBlock != nil {
			fromN = latestValidSavedBlock.Number.Uint64()
		}
		if c.initialFrom != nil {
			fromN = c.initialFrom.Number.Uint64()
		}
	}
	toN := latestHeader.Number.Uint64()
	all, err := c.getAllTransfers(parent, fromN, toN)
	if err != nil {
		return err
	}

	c.from = toDBHeader(nextHeader)
	c.lastFetchedBlockTime = time.Now()
	if len(removed) != 0 {
		lth := len(removed)
		c.feed.Send(Event{
			Type:        EventReorg,
			BlockNumber: removed[lth-1].Number,
			Accounts:    uniqueAccountsFromHeaders(removed),
		})
	}
	log.Info("before sending new block event", "latest", latestHeader != nil, "removed", len(removed), "len", len(uniqueAccountsFromTransfers(all)))

	c.feed.Send(Event{
		Type:                      EventNewBlock,
		BlockNumber:               latestHeader.Number,
		Accounts:                  uniqueAccountsFromTransfers(all),
		NewTransactionsPerAccount: transfersPerAccount(all),
	})

	return nil
}

func (c *newBlocksTransfersCommand) onNewBlock(ctx context.Context, from *DBHeader, latest *types.Header) (lastestHeader *DBHeader, removed []*DBHeader, lastSavedValidHeader *DBHeader, reorgSpotted bool, err error) {
	if from.Hash == latest.ParentHash {
		// parent matching from node in the cache. on the same chain.
		return toHead(latest), nil, nil, false, nil
	}

	lastSavedBlock, err := c.db.GetLastSavedBlock()
	if err != nil {
		return nil, nil, nil, false, err
	}

	if lastSavedBlock == nil {
		return toHead(latest), nil, nil, true, nil
	}

	header, err := c.client.HeaderByNumber(ctx, lastSavedBlock.Number)
	if err != nil {
		return nil, nil, nil, false, err
	}

	if header.Hash() == lastSavedBlock.Hash {
		return toHead(latest), nil, lastSavedBlock, true, nil
	}

	log.Debug("wallet reactor spotted reorg", "last header in db", from.Hash, "new parent", latest.ParentHash)
	for lastSavedBlock != nil {
		removed = append(removed, lastSavedBlock)
		lastSavedBlock, err = c.db.GetLastSavedBlockBefore(lastSavedBlock.Number)

		if err != nil {
			return nil, nil, nil, false, err
		}

		if lastSavedBlock == nil {
			continue
		}

		header, err := c.client.HeaderByNumber(ctx, lastSavedBlock.Number)
		if err != nil {
			return nil, nil, nil, false, err
		}

		// the last saved block is still valid
		if header.Hash() == lastSavedBlock.Hash {
			return toHead(latest), nil, lastSavedBlock, true, nil
		}
	}

	return toHead(latest), removed, lastSavedBlock, true, nil
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
func (c *findAndCheckBlockRangeCommand) fastIndex(ctx context.Context, bCache *balanceCache, fromByAddress map[common.Address]*big.Int, toByAddress map[common.Address]*big.Int) (map[common.Address]*big.Int, map[common.Address][]*DBHeader, error) {
	start := time.Now()
	group := NewGroup(ctx)

	commands := make([]*ethHistoricalCommand, len(c.accounts))
	for i, address := range c.accounts {
		eth := &ethHistoricalCommand{
			db:           c.db,
			client:       c.client,
			balanceCache: bCache,
			address:      address,
			eth: &ETHTransferDownloader{
				client:   c.client,
				accounts: []common.Address{address},
				signer:   types.NewEIP155Signer(c.chain),
				db:       c.db,
			},
			feed:    c.feed,
			from:    fromByAddress[address],
			to:      toByAddress[address],
			noLimit: c.noLimit,
		}
		commands[i] = eth
		group.Add(eth.Command())
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-group.WaitAsync():
		resultingFromByAddress := map[common.Address]*big.Int{}
		headers := map[common.Address][]*DBHeader{}
		for _, command := range commands {
			resultingFromByAddress[command.address] = command.resultingFrom
			headers[command.address] = command.foundHeaders
		}
		log.Info("fast indexer finished", "in", time.Since(start))
		return resultingFromByAddress, headers, nil
	}
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findAndCheckBlockRangeCommand) fastIndexErc20(ctx context.Context, fromByAddress map[common.Address]*big.Int, toByAddress map[common.Address]*big.Int) (map[common.Address][]*DBHeader, error) {
	start := time.Now()
	group := NewGroup(ctx)

	commands := make([]*erc20HistoricalCommand, len(c.accounts))
	for i, address := range c.accounts {
		erc20 := &erc20HistoricalCommand{
			db:           c.db,
			erc20:        NewERC20TransfersDownloader(c.client, []common.Address{address}, types.NewEIP155Signer(c.chain)),
			client:       c.client,
			feed:         c.feed,
			address:      address,
			from:         fromByAddress[address],
			to:           toByAddress[address],
			foundHeaders: []*DBHeader{},
		}
		commands[i] = erc20
		group.Add(erc20.Command())
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-group.WaitAsync():
		headres := map[common.Address][]*DBHeader{}
		for _, command := range commands {
			headres[command.address] = command.foundHeaders
		}
		log.Info("fast indexer Erc20 finished", "in", time.Since(start))
		return headres, nil
	}
}

func getTransfersByBlocks(ctx context.Context, db *Database, downloader *ETHTransferDownloader, address common.Address, blocks []*big.Int) ([]Transfer, error) {
	allTransfers := []Transfer{}

	for _, block := range blocks {
		transfers, err := downloader.GetTransfersByNumber(ctx, block)
		if err != nil {
			return nil, err
		}
		log.Debug("loadTransfers", "block", block, "new transfers", len(transfers))
		if len(transfers) > 0 {
			allTransfers = append(allTransfers, transfers...)
		}
	}

	return allTransfers, nil
}

func loadTransfers(ctx context.Context, accounts []common.Address, db *Database, client *ethclient.Client, chain *big.Int, limit int, blocksByAddress map[common.Address][]*big.Int) (map[common.Address][]Transfer, error) {
	start := time.Now()
	group := NewGroup(ctx)

	commands := []*transfersCommand{}
	for _, address := range accounts {
		blocks, ok := blocksByAddress[address]

		if !ok {
			blocks, _ = db.GetBlocksByAddress(address, numberOfBlocksCheckedPerIteration)
		}

		for _, block := range blocks {
			erc20 := &transfersCommand{
				db:      db,
				client:  client,
				address: address,
				eth: &ETHTransferDownloader{
					client:   client,
					accounts: []common.Address{address},
					signer:   types.NewEIP155Signer(chain),
					db:       db,
				},
				block: block,
			}
			commands = append(commands, erc20)
			group.Add(erc20.Command())
		}
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-group.WaitAsync():
		transfersByAddress := map[common.Address][]Transfer{}
		for _, command := range commands {
			if len(command.fetchedTransfers) == 0 {
				continue
			}

			transfers, ok := transfersByAddress[command.address]
			if !ok {
				transfers = []Transfer{}
			}

			for _, transfer := range command.fetchedTransfers {
				transfersByAddress[command.address] = append(transfers, transfer)
			}
		}
		log.Info("loadTransfers finished", "in", time.Since(start))
		return transfersByAddress, nil
	}
}

func (c *controlCommand) LoadTransfers(ctx context.Context, downloader *ETHTransferDownloader, limit int) (map[common.Address][]Transfer, error) {
	return loadTransfers(ctx, c.accounts, c.db, c.client, c.chain, limit, make(map[common.Address][]*big.Int))
}

/*
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
	log.Info("spawn reorg verifier", "from", last.Number, "to", header.Number)
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
*/
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
	log.Info("start control command")
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	head, err := c.client.HeaderByNumber(ctx, nil)
	cancel()
	if err != nil {
		return err
	}

	c.feed.Send(Event{
		Type:     EventFetchingRecentHistory,
		Accounts: c.accounts,
	})

	log.Info("current head is", "block number", head.Number)
	lastKnownEthBlocks, accountsWithoutHistory, err := c.db.GetLastKnownBlockByAddresses(c.accounts)
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
	if err != nil {
		return err
	}

	downloader := &ETHTransferDownloader{
		client:   c.client,
		accounts: c.accounts,
		signer:   types.NewEIP155Signer(c.chain),
		db:       c.db,
	}
	_, err = c.LoadTransfers(parent, downloader, 40)
	if err != nil {
		return err
	}

	c.feed.Send(Event{
		Type:        EventRecentHistoryReady,
		Accounts:    c.accounts,
		BlockNumber: head.Number,
	})

	log.Info("watching new blocks", "start from", head.Number)
	cmd := &newBlocksTransfersCommand{
		db:                   c.db,
		chain:                c.chain,
		client:               c.client,
		accounts:             c.accounts,
		eth:                  c.eth,
		erc20:                c.erc20,
		feed:                 c.feed,
		initialFrom:          toDBHeader(head),
		from:                 toDBHeader(head),
		lastFetchedBlockTime: time.Now(),
	}

	err = cmd.Command()(parent)
	if err != nil {
		log.Warn("error on running newBlocksTransfersCommand", "err", err)
		return err
	}

	log.Info("end control command")
	return err
}

func (c *controlCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func uniqueAccountsFromTransfers(allTransfers map[common.Address][]Transfer) []common.Address {
	accounts := []common.Address{}
	unique := map[common.Address]struct{}{}
	for address, transfers := range allTransfers {
		if len(transfers) == 0 {
			continue
		}

		_, exist := unique[address]
		if exist {
			continue
		}
		unique[address] = struct{}{}
		accounts = append(accounts, address)
	}
	return accounts
}

func transfersPerAccount(allTransfers map[common.Address][]Transfer) map[common.Address]int {
	res := map[common.Address]int{}
	for address, transfers := range allTransfers {
		res[address] = len(transfers)
	}

	return res
}

func uniqueAccountsFromHeaders(headers []*DBHeader) []common.Address {
	accounts := []common.Address{}
	unique := map[common.Address]struct{}{}
	for i := range headers {
		_, exist := unique[headers[i].Address]
		if exist {
			continue
		}
		unique[headers[i].Address] = struct{}{}
		accounts = append(accounts, headers[i].Address)
	}
	return accounts
}

type transfersCommand struct {
	db               *Database
	eth              *ETHTransferDownloader
	block            *big.Int
	address          common.Address
	client           reactorClient
	fetchedTransfers []Transfer
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

	err = c.db.SaveTranfers(c.address, allTransfers, []*big.Int{c.block})
	if err != nil {
		log.Error("SaveTranfers error", "error", err)
		return err
	}

	c.fetchedTransfers = allTransfers
	log.Debug("transfers loaded", "address", c.address, "len", len(allTransfers))
	return nil
}

type loadTransfersCommand struct {
	accounts                []common.Address
	db                      *Database
	chain                   *big.Int
	client                  *ethclient.Client
	blocksByAddress         map[common.Address][]*big.Int
	foundTransfersByAddress map[common.Address][]Transfer
}

func (c *loadTransfersCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *loadTransfersCommand) LoadTransfers(ctx context.Context, downloader *ETHTransferDownloader, limit int, blocksByAddress map[common.Address][]*big.Int) (map[common.Address][]Transfer, error) {
	return loadTransfers(ctx, c.accounts, c.db, c.client, c.chain, limit, blocksByAddress)
}

func (c *loadTransfersCommand) Run(parent context.Context) (err error) {
	downloader := &ETHTransferDownloader{
		client:   c.client,
		accounts: c.accounts,
		signer:   types.NewEIP155Signer(c.chain),
		db:       c.db,
	}
	transfersByAddress, err := c.LoadTransfers(parent, downloader, 40, c.blocksByAddress)
	if err != nil {
		return err
	}
	c.foundTransfersByAddress = transfersByAddress

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
	foundHeaders  map[common.Address][]*DBHeader
	noLimit       bool
}

func (c *findAndCheckBlockRangeCommand) Command() Command {
	return FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *findAndCheckBlockRangeCommand) Run(parent context.Context) (err error) {
	log.Debug("start findAndCHeckBlockRangeCommand")
	newFromByAddress, ethHeadersByAddress, err := c.fastIndex(parent, c.balanceCache, c.fromByAddress, c.toByAddress)
	if err != nil {
		return err
	}
	if c.noLimit {
		newFromByAddress = map[common.Address]*big.Int{}
		for _, address := range c.accounts {
			newFromByAddress[address] = c.fromByAddress[address]
		}
	}
	erc20HeadersByAddress, err := c.fastIndexErc20(parent, newFromByAddress, c.toByAddress)
	if err != nil {
		return err
	}

	foundHeaders := map[common.Address][]*DBHeader{}
	for _, address := range c.accounts {
		ethHeaders := ethHeadersByAddress[address]
		erc20Headers := erc20HeadersByAddress[address]

		allHeaders := append(ethHeaders, erc20Headers...)
		foundHeaders[address] = allHeaders

		log.Debug("saving headers", "len", len(allHeaders), "address")
		err = c.db.ProcessBlocks(address, newFromByAddress[address], c.toByAddress[address], allHeaders)
		if err != nil {
			return err
		}
	}

	c.foundHeaders = foundHeaders

	return
}
