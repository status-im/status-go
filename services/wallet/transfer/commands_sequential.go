package transfer

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"
)

type findNewBlocksCommand struct {
	*findBlocksCommand
}

func (c *findNewBlocksCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: 13 * time.Second, // TODO - make it configurable based on chain block mining time
		Runable:  c.Run,
	}.Run
}

func (c *findNewBlocksCommand) Run(parent context.Context) (err error) {
	log.Debug("start findNewBlocksCommand", "account", c.account, "chain", c.chainClient.ChainID, "noLimit", c.noLimit)

	headNum, err := getHeadBlockNumber(parent, c.chainClient)
	if err != nil {
		// c.error = err
		return err // Might need to retry a couple of times
	}

	blockRange, err := loadBlockRangeInfo(c.chainClient.ChainID, c.account, c.blockRangeDAO)
	if err != nil {
		log.Error("findBlocksCommand loadBlockRangeInfo", "error", err)
		// c.error = err
		return err // Will keep spinning forever nomatter what
	}

	if blockRange != nil {
		c.fromBlockNumber = new(big.Int).Add(blockRange.LastKnown, big.NewInt(1))

		log.Debug("Launching new blocks command", "chainID", c.chainClient.ChainID, "account", c.account,
			"from", c.fromBlockNumber, "headNum", headNum)

		// In case interval between checks is set smaller than block mining time,
		// we might need to wait for the next block to be mined
		if c.fromBlockNumber.Cmp(headNum) > 0 {
			return
		}

		c.toBlockNumber = headNum

		_ = c.findBlocksCommand.Run(parent)
	}

	return nil
}

// TODO NewFindBlocksCommand
type findBlocksCommand struct {
	account            common.Address
	db                 *Database
	blockRangeDAO      *BlockRangeSequentialDAO
	chainClient        *chain.ClientWithFallback
	balanceCache       *balanceCache
	feed               *event.Feed
	noLimit            bool
	transactionManager *TransactionManager
	fromBlockNumber    *big.Int
	toBlockNumber      *big.Int
	blocksLoadedCh     chan<- []*DBHeader

	// Not to be set by the caller
	resFromBlock     *Block
	startBlockNumber *big.Int
	error            error
}

func (c *findBlocksCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *findBlocksCommand) Run(parent context.Context) (err error) {
	log.Debug("start findBlocksCommand", "account", c.account, "chain", c.chainClient.ChainID, "noLimit", c.noLimit)

	rangeSize := big.NewInt(DefaultNodeBlockChunkSize)

	from, to := new(big.Int).Set(c.fromBlockNumber), new(big.Int).Set(c.toBlockNumber)

	// Limit the range size to DefaultNodeBlockChunkSize
	if new(big.Int).Sub(to, from).Cmp(rangeSize) > 0 {
		from.Sub(to, rangeSize)
	}

	for {
		headers, _ := c.checkRange(parent, from, to)
		if c.error != nil {
			log.Error("findBlocksCommand checkRange", "error", c.error, "account", c.account,
				"chain", c.chainClient.ChainID, "from", from, "to", to)
			break
		}

		if len(headers) > 0 {
			log.Debug("findBlocksCommand saving headers", "len", len(headers), "lastBlockNumber", to,
				"balance", c.balanceCache.ReadCachedBalance(c.account, to),
				"nonce", c.balanceCache.ReadCachedNonce(c.account, to))

			err = c.db.SaveBlocks(c.chainClient.ChainID, c.account, headers)
			if err != nil {
				c.error = err
				// return err
				break
			}

			c.blocksFound(headers)
		}

		err = c.upsertBlockRange(&BlockRange{c.startBlockNumber, c.resFromBlock.Number, to})
		if err != nil {
			break
		}

		from, to = nextRange(c.resFromBlock.Number, c.fromBlockNumber)

		if to.Cmp(c.fromBlockNumber) <= 0 || (c.startBlockNumber != nil &&
			c.startBlockNumber.Cmp(big.NewInt(0)) > 0 && to.Cmp(c.startBlockNumber) <= 0) {
			log.Debug("Checked all ranges, stop execution", "startBlock", c.startBlockNumber, "from", from, "to", to)
			break
		}
	}

	log.Debug("end findBlocksCommand", "account", c.account, "chain", c.chainClient.ChainID, "noLimit", c.noLimit)

	return nil
}

func (c *findBlocksCommand) blocksFound(headers []*DBHeader) {
	c.blocksLoadedCh <- headers
}

func (c *findBlocksCommand) upsertBlockRange(blockRange *BlockRange) error {
	log.Debug("upsert block range", "Start", blockRange.Start, "FirstKnown", blockRange.FirstKnown, "LastKnown", blockRange.LastKnown,
		"chain", c.chainClient.ChainID, "account", c.account)

	err := c.blockRangeDAO.upsertRange(c.chainClient.ChainID, c.account, blockRange)
	if err != nil {
		c.error = err
		log.Error("findBlocksCommand upsertRange", "error", err)
		return err
	}

	return nil
}

func (c *findBlocksCommand) checkRange(parent context.Context, from *big.Int, to *big.Int) (
	foundHeaders []*DBHeader, err error) {

	fromBlock := &Block{Number: from}

	newFromBlock, ethHeaders, startBlock, err := c.fastIndex(parent, c.balanceCache, fromBlock, to)
	if err != nil {
		log.Error("findBlocksCommand checkRange fastIndex", "err", err, "account", c.account,
			"chain", c.chainClient.ChainID)
		c.error = err
		// return err // In case c.noLimit is true, hystrix "max concurrency" may be reached and we will not be able to index ETH transfers
		return nil, nil
	}
	log.Debug("findBlocksCommand checkRange", "chainID", c.chainClient.ChainID, "account", c.account,
		"startBlock", startBlock, "newFromBlock", newFromBlock.Number, "toBlockNumber", to, "noLimit", c.noLimit)

	// There could be incoming ERC20 transfers which don't change the balance
	// and nonce of ETH account, so we keep looking for them
	erc20Headers, err := c.fastIndexErc20(parent, newFromBlock.Number, to)
	if err != nil {
		log.Error("findBlocksCommand checkRange fastIndexErc20", "err", err, "account", c.account, "chain", c.chainClient.ChainID)
		c.error = err
		// return err
		return nil, nil
	}

	allHeaders := append(ethHeaders, erc20Headers...)

	if len(allHeaders) > 0 {
		foundHeaders = uniqueHeaderPerBlockHash(allHeaders)
	}

	c.resFromBlock = newFromBlock
	c.startBlockNumber = startBlock

	log.Debug("end findBlocksCommand checkRange", "chainID", c.chainClient.ChainID, "account", c.account,
		"c.startBlock", c.startBlockNumber, "newFromBlock", newFromBlock.Number,
		"toBlockNumber", to, "c.resFromBlock", c.resFromBlock.Number)

	return
}

func loadBlockRangeInfo(chainID uint64, account common.Address, blockDAO *BlockRangeSequentialDAO) (
	*BlockRange, error) {

	blockRange, err := blockDAO.getBlockRange(chainID, account)
	if err != nil {
		log.Error("failed to load block ranges from database", "chain", chainID, "account", account,
			"error", err)
		return nil, err
	}

	return blockRange, nil
}

// Returns if all blocks are loaded, which means that start block (beginning of account history)
// has been found and all block headers saved to the DB
func areAllHistoryBlocksLoaded(blockInfo *BlockRange) bool {
	if blockInfo == nil {
		return false
	}

	if blockInfo.FirstKnown != nil && blockInfo.Start != nil &&
		blockInfo.Start.Cmp(blockInfo.FirstKnown) >= 0 {

		return true
	}

	return false
}

func areAllHistoryBlocksLoadedForAddress(blockRangeDAO *BlockRangeSequentialDAO, chainID uint64,
	address common.Address) (bool, error) {

	blockRange, err := blockRangeDAO.getBlockRange(chainID, address)
	if err != nil {
		log.Error("findBlocksCommand getBlockRange", "error", err)
		return false, err
	}

	return areAllHistoryBlocksLoaded(blockRange), nil
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findBlocksCommand) fastIndex(ctx context.Context, bCache *balanceCache,
	fromBlock *Block, toBlockNumber *big.Int) (resultingFrom *Block, headers []*DBHeader,
	startBlock *big.Int, err error) {

	log.Debug("fast index started", "chainID", c.chainClient.ChainID, "account", c.account,
		"from", fromBlock.Number, "to", toBlockNumber)

	start := time.Now()
	group := async.NewGroup(ctx)

	command := &ethHistoricalCommand{
		chainClient:  c.chainClient,
		balanceCache: bCache,
		address:      c.account,
		feed:         c.feed,
		from:         fromBlock,
		to:           toBlockNumber,
		noLimit:      c.noLimit,
		threadLimit:  SequentialThreadLimit,
	}
	group.Add(command.Command())

	select {
	case <-ctx.Done():
		err = ctx.Err()
		log.Info("fast indexer ctx Done", "error", err)
		return
	case <-group.WaitAsync():
		if command.error != nil {
			err = command.error
			return
		}
		resultingFrom = &Block{Number: command.resultingFrom}
		headers = command.foundHeaders
		startBlock = command.startBlock
		log.Debug("fast indexer finished", "chainID", c.chainClient.ChainID, "account", c.account, "in", time.Since(start),
			"startBlock", command.startBlock, "resultingFrom", resultingFrom.Number, "headers", len(headers))
		return
	}
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findBlocksCommand) fastIndexErc20(ctx context.Context, fromBlockNumber *big.Int,
	toBlockNumber *big.Int) ([]*DBHeader, error) {

	start := time.Now()
	group := async.NewGroup(ctx)

	erc20 := &erc20HistoricalCommand{
		erc20:        NewERC20TransfersDownloader(c.chainClient, []common.Address{c.account}, types.LatestSignerForChainID(c.chainClient.ToBigInt())),
		chainClient:  c.chainClient,
		feed:         c.feed,
		address:      c.account,
		from:         fromBlockNumber,
		to:           toBlockNumber,
		foundHeaders: []*DBHeader{},
	}
	group.Add(erc20.Command())

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-group.WaitAsync():
		headers := erc20.foundHeaders
		log.Debug("fast indexer Erc20 finished", "chainID", c.chainClient.ChainID, "account", c.account,
			"in", time.Since(start), "headers", len(headers))
		return headers, nil
	}
}

func loadTransfersLoop(ctx context.Context, account common.Address, blockDAO *BlockDAO, db *Database,
	chainClient *chain.ClientWithFallback, transactionManager *TransactionManager, pendingTxManager *transactions.TransactionManager,
	tokenManager *token.Manager, feed *event.Feed, blocksLoadedCh <-chan []*DBHeader) {

	log.Debug("loadTransfersLoop start", "chain", chainClient.ChainID, "account", account)

	for {
		select {
		case <-ctx.Done():
			log.Info("loadTransfersLoop error", "chain", chainClient.ChainID, "account", account, "error", ctx.Err())
			return
		case dbHeaders := <-blocksLoadedCh:
			log.Debug("loadTransfersOnDemand transfers received", "chain", chainClient.ChainID, "account", account, "headers", len(dbHeaders))

			blockNums := make([]*big.Int, len(dbHeaders))
			for i, dbHeader := range dbHeaders {
				blockNums[i] = dbHeader.Number
			}

			blocksByAddress := map[common.Address][]*big.Int{account: blockNums}
			go func() {
				_ = loadTransfers(ctx, []common.Address{account}, blockDAO, db, chainClient, noBlockLimit,
					blocksByAddress, transactionManager, pendingTxManager, tokenManager, feed)
			}()
		}
	}
}

func newLoadBlocksAndTransfersCommand(account common.Address, db *Database,
	blockDAO *BlockDAO, chainClient *chain.ClientWithFallback, feed *event.Feed,
	transactionManager *TransactionManager, pendingTxManager *transactions.TransactionManager,
	tokenManager *token.Manager) *loadBlocksAndTransfersCommand {

	return &loadBlocksAndTransfersCommand{
		account:            account,
		db:                 db,
		blockRangeDAO:      &BlockRangeSequentialDAO{db.client},
		blockDAO:           blockDAO,
		chainClient:        chainClient,
		feed:               feed,
		errorsCount:        0,
		transactionManager: transactionManager,
		pendingTxManager:   pendingTxManager,
		tokenManager:       tokenManager,
		blocksLoadedCh:     make(chan []*DBHeader, 100),
	}
}

type loadBlocksAndTransfersCommand struct {
	account       common.Address
	db            *Database
	blockRangeDAO *BlockRangeSequentialDAO
	blockDAO      *BlockDAO
	chainClient   *chain.ClientWithFallback
	feed          *event.Feed
	balanceCache  *balanceCache
	errorsCount   int
	// nonArchivalRPCNode bool // TODO Make use of it
	transactionManager *TransactionManager
	pendingTxManager   *transactions.TransactionManager
	tokenManager       *token.Manager
	blocksLoadedCh     chan []*DBHeader

	// Not to be set by the caller
	transfersLoaded bool // For event RecentHistoryReady to be sent only once per account during app lifetime
}

func (c *loadBlocksAndTransfersCommand) Run(parent context.Context) error {
	log.Debug("start load all transfers command", "chain", c.chainClient.ChainID, "account", c.account)

	ctx := parent

	if c.balanceCache == nil {
		c.balanceCache = newBalanceCache() // TODO - need to keep balanceCache in memory??? What about sharing it with other packages?
	}

	group := async.NewGroup(ctx)

	err := c.fetchTransfersForLoadedBlocks(group)
	for err != nil {
		return err
	}

	c.startTransfersLoop(ctx)

	err = c.fetchHistoryBlocks(parent, group, c.blocksLoadedCh)
	for err != nil {
		group.Stop()
		group.Wait()
		return err
	}

	c.startFetchingNewBlocks(group, c.account, c.blocksLoadedCh)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Debug("end loadBlocksAndTransfers command", "chain", c.chainClient.ChainID, "account", c.account)
		return nil
	}
}

func (c *loadBlocksAndTransfersCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *loadBlocksAndTransfersCommand) startTransfersLoop(ctx context.Context) {
	go loadTransfersLoop(ctx, c.account, c.blockDAO, c.db, c.chainClient, c.transactionManager,
		c.pendingTxManager, c.tokenManager, c.feed, c.blocksLoadedCh)
}

func (c *loadBlocksAndTransfersCommand) fetchHistoryBlocks(ctx context.Context, group *async.Group, blocksLoadedCh chan []*DBHeader) error {

	log.Debug("fetchHistoryBlocks start", "chainID", c.chainClient.ChainID, "account", c.account)

	headNum, err := getHeadBlockNumber(ctx, c.chainClient)
	if err != nil {
		// c.error = err
		return err // Might need to retry a couple of times
	}

	blockRange, err := loadBlockRangeInfo(c.chainClient.ChainID, c.account, c.blockRangeDAO)
	if err != nil {
		log.Error("findBlocksCommand loadBlockRangeInfo", "error", err)
		// c.error = err
		return err // Will keep spinning forever nomatter what
	}

	allHistoryLoaded := areAllHistoryBlocksLoaded(blockRange)
	to := getToHistoryBlockNumber(headNum, blockRange, allHistoryLoaded)

	log.Debug("fetchHistoryBlocks", "chainID", c.chainClient.ChainID, "account", c.account, "to", to, "allHistoryLoaded", allHistoryLoaded)

	if !allHistoryLoaded {
		fbc := &findBlocksCommand{
			account:            c.account,
			db:                 c.db,
			blockRangeDAO:      c.blockRangeDAO,
			chainClient:        c.chainClient,
			balanceCache:       c.balanceCache,
			feed:               c.feed,
			noLimit:            false,
			fromBlockNumber:    big.NewInt(0),
			toBlockNumber:      to,
			transactionManager: c.transactionManager,
			blocksLoadedCh:     blocksLoadedCh,
		}
		group.Add(fbc.Command())
	} else {
		if !c.transfersLoaded {
			transfersLoaded, err := c.areAllTransfersLoaded()
			if err != nil {
				return err
			}

			if transfersLoaded {
				c.transfersLoaded = true
				c.notifyHistoryReady()
			}
		}
	}

	log.Debug("fetchHistoryBlocks end", "chainID", c.chainClient.ChainID, "account", c.account)

	return nil
}

func (c *loadBlocksAndTransfersCommand) startFetchingNewBlocks(group *async.Group, address common.Address, blocksLoadedCh chan<- []*DBHeader) {

	log.Debug("startFetchingNewBlocks", "chainID", c.chainClient.ChainID, "account", address)

	newBlocksCmd := &findNewBlocksCommand{
		findBlocksCommand: &findBlocksCommand{
			account:            address,
			db:                 c.db,
			blockRangeDAO:      c.blockRangeDAO,
			chainClient:        c.chainClient,
			balanceCache:       c.balanceCache,
			feed:               c.feed,
			noLimit:            false,
			transactionManager: c.transactionManager,
			blocksLoadedCh:     blocksLoadedCh,
		},
	}
	group.Add(newBlocksCmd.Command())
}

func (c *loadBlocksAndTransfersCommand) fetchTransfersForLoadedBlocks(group *async.Group) error {

	log.Debug("fetchTransfers start", "chainID", c.chainClient.ChainID, "account", c.account)

	blocks, err := c.blockDAO.GetBlocksToLoadByAddress(c.chainClient.ChainID, c.account, numberOfBlocksCheckedPerIteration)
	if err != nil {
		log.Error("loadBlocksAndTransfersCommand GetBlocksToLoadByAddress", "error", err)
		return err
	}

	blocksMap := make(map[common.Address][]*big.Int)
	blocksMap[c.account] = blocks

	txCommand := &loadTransfersCommand{
		accounts:           []common.Address{c.account},
		db:                 c.db,
		blockDAO:           c.blockDAO,
		chainClient:        c.chainClient,
		transactionManager: c.transactionManager,
		pendingTxManager:   c.pendingTxManager,
		tokenManager:       c.tokenManager,
		blocksByAddress:    blocksMap,
		feed:               c.feed,
	}

	group.Add(txCommand.Command())

	return nil
}

func (c *loadBlocksAndTransfersCommand) notifyHistoryReady() {
	if c.feed != nil {
		c.feed.Send(walletevent.Event{
			Type:     EventRecentHistoryReady,
			Accounts: []common.Address{c.account},
		})
	}
}

func (c *loadBlocksAndTransfersCommand) areAllTransfersLoaded() (bool, error) {
	allBlocksLoaded, err := areAllHistoryBlocksLoadedForAddress(c.blockRangeDAO, c.chainClient.ChainID, c.account)
	if err != nil {
		log.Error("loadBlockAndTransfersCommand allHistoryBlocksLoaded", "error", err)
		return false, err
	}

	if allBlocksLoaded {
		firstHeader, err := c.blockDAO.GetFirstSavedBlock(c.chainClient.ChainID, c.account)
		if err != nil {
			log.Error("loadBlocksAndTransfersCommand GetFirstSavedBlock", "error", err)
			return false, err
		}

		// If first block is Loaded, we have fetched all the transfers
		if firstHeader != nil && firstHeader.Loaded {
			return true, nil
		}
	}

	return false, nil
}

// TODO - make it a common method for every service that wants head block number, that will cache the latest block
// and updates it on timeout
func getHeadBlockNumber(parent context.Context, chainClient *chain.ClientWithFallback) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	head, err := chainClient.HeaderByNumber(ctx, nil)
	cancel()
	if err != nil {
		return nil, err
	}

	return head.Number, err
}

func nextRange(from *big.Int, zeroBlockNumber *big.Int) (*big.Int, *big.Int) {
	log.Debug("next range start", "from", from, "zeroBlockNumber", zeroBlockNumber)

	rangeSize := big.NewInt(DefaultNodeBlockChunkSize)

	to := new(big.Int).Sub(from, big.NewInt(1)) // it won't hit the cache, but we wont load the transfers twice
	if to.Cmp(rangeSize) > 0 {
		from.Sub(to, rangeSize)
	} else {
		from = new(big.Int).Set(zeroBlockNumber)
	}

	log.Debug("next range end", "from", from, "to", to, "zeroBlockNumber", zeroBlockNumber)

	return from, to
}

func getToHistoryBlockNumber(headNum *big.Int, blockRange *BlockRange, allHistoryLoaded bool) *big.Int {
	var toBlockNum *big.Int
	if blockRange != nil {
		if !allHistoryLoaded {
			toBlockNum = blockRange.FirstKnown
		}
	} else {
		toBlockNum = headNum
	}

	return toBlockNum
}
