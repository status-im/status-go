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
)

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
	log.Debug("findBlocksCommand checkRange", "startBlock", startBlock, "newFromBlock", newFromBlock.Number, "toBlockNumber", to, "noLimit", c.noLimit)

	// There could be incoming ERC20 transfers which don't change the balance
	// and nonce of ETH account, so we keep looking for them
	erc20Headers, err := c.fastIndexErc20(parent, newFromBlock.Number, to)
	if err != nil {
		log.Error("findBlocksCommand checkRange fastIndexErc20", "err", err)
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

	log.Debug("end findBlocksCommand checkRange", "c.startBlock", c.startBlockNumber, "newFromBlock", newFromBlock.Number,
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

	log.Debug("fast index started", "accounts", c.account, "from", fromBlock.Number, "to", toBlockNumber)

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
		log.Debug("fast indexer finished", "in", time.Since(start), "startBlock", command.startBlock, "resultingFrom", resultingFrom.Number, "headers", len(headers))
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
		erc20:        NewERC20TransfersDownloader(c.chainClient, []common.Address{c.account}, types.NewLondonSigner(c.chainClient.ToBigInt())),
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
		log.Debug("fast indexer Erc20 finished", "in", time.Since(start), "headers", len(headers))
		return headers, nil
	}
}

// TODO Think on how to reuse loadTransfersCommand, as it shares many members and some methods
// but does not need to return the transfers but only save them to DB, as there can be too many of them
// and the logic of `loadTransfersLoop` is different from `loadTransfers“
type loadAllTransfersCommand struct {
	accounts           []common.Address
	db                 *Database
	blockDAO           *BlockDAO
	chainClient        *chain.ClientWithFallback
	blocksByAddress    map[common.Address][]*big.Int
	transactionManager *TransactionManager
	tokenManager       *token.Manager
	blocksLimit        int
	feed               *event.Feed
}

func (c *loadAllTransfersCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *loadAllTransfersCommand) Run(parent context.Context) error {
	start := time.Now()
	group := async.NewGroup(parent)

	for _, address := range c.accounts {
		transfers := &transfersCommand{
			db:          c.db,
			blockDAO:    c.blockDAO,
			chainClient: c.chainClient,
			address:     address,
			eth: &ETHDownloader{
				chainClient: c.chainClient,
				accounts:    []common.Address{address},
				signer:      types.NewLondonSigner(c.chainClient.ToBigInt()),
				db:          c.db,
			},
			blockNums:          c.blocksByAddress[address],
			blocksLimit:        c.blocksLimit,
			transactionManager: c.transactionManager,
			tokenManager:       c.tokenManager,
			feed:               c.feed,
		}
		group.Add(transfers.Command())
	}

	select {
	case <-parent.Done():
		log.Info("loadTransfers transfersCommand error", "chain", c.chainClient.ChainID, "error", parent.Err())
		return parent.Err()
	case <-group.WaitAsync():
		log.Debug("loadTransfers finished for account", "in", time.Since(start), "chain", c.chainClient.ChainID, "limit", c.blocksLimit)
	}

	return nil
}

func newLoadBlocksAndTransfersCommand(accounts []common.Address, db *Database,
	blockDAO *BlockDAO, chainClient *chain.ClientWithFallback, feed *event.Feed,
	transactionManager *TransactionManager, tokenManager *token.Manager) *loadBlocksAndTransfersCommand {

	return &loadBlocksAndTransfersCommand{
		accounts:           accounts,
		db:                 db,
		blockRangeDAO:      &BlockRangeSequentialDAO{db.client},
		blockDAO:           blockDAO,
		chainClient:        chainClient,
		feed:               feed,
		errorsCount:        0,
		transactionManager: transactionManager,
		tokenManager:       tokenManager,
		transfersLoaded:    make(map[common.Address]bool),
	}
}

type loadBlocksAndTransfersCommand struct {
	accounts      []common.Address
	db            *Database
	blockRangeDAO *BlockRangeSequentialDAO
	blockDAO      *BlockDAO
	chainClient   *chain.ClientWithFallback
	feed          *event.Feed
	balanceCache  *balanceCache
	errorsCount   int
	// nonArchivalRPCNode bool // TODO Make use of it
	transactionManager *TransactionManager
	tokenManager       *token.Manager

	// Not to be set by the caller
	transfersLoaded map[common.Address]bool // For event RecentHistoryReady to be sent only once per account during app lifetime
}

func (c *loadBlocksAndTransfersCommand) Run(parent context.Context) error {
	log.Debug("start load all transfers command", "chain", c.chainClient.ChainID)

	ctx := parent

	if c.balanceCache == nil {
		c.balanceCache = newBalanceCache() // TODO - need to keep balanceCache in memory??? What about sharing it with other packages?
	}

	group := async.NewGroup(ctx)

	headNum, err := getHeadBlockNumber(parent, c.chainClient)
	if err != nil {
		// c.error = err
		return err // Might need to retry a couple of times
	}

	for _, address := range c.accounts {
		blockRange, err := loadBlockRangeInfo(c.chainClient.ChainID, address, c.blockRangeDAO)
		if err != nil {
			log.Error("findBlocksCommand loadBlockRangeInfo", "error", err)
			// c.error = err
			return err // Will keep spinning forever nomatter what
		}

		allHistoryLoaded := areAllHistoryBlocksLoaded(blockRange)
		toHistoryBlockNum := getToHistoryBlockNumber(headNum, blockRange, allHistoryLoaded)

		if !allHistoryLoaded {
			c.fetchHistoryBlocks(ctx, group, address, big.NewInt(0), toHistoryBlockNum)
		} else {
			if !c.transfersLoaded[address] {
				transfersLoaded, err := c.areAllTransfersLoadedForAddress(address)
				if err != nil {
					return err
				}

				if transfersLoaded {
					c.transfersLoaded[address] = true
					c.notifyHistoryReady(address)
				}
			}
		}

		// If no block ranges are stored, all blocks will be fetched by fetchHistoryBlocks method
		if blockRange != nil {
			c.fetchNewBlocks(ctx, group, address, blockRange, headNum)
		}
	}

	c.fetchTransfers(ctx, group)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Debug("end load all transfers command", "chain", c.chainClient.ChainID)
		return nil
	}
}

func (c *loadBlocksAndTransfersCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: 13 * time.Second, // Slightly more that block mining time
		Runable:  c.Run,
	}.Run
}

func (c *loadBlocksAndTransfersCommand) fetchHistoryBlocks(ctx context.Context, group *async.Group,
	address common.Address, from *big.Int, to *big.Int) {

	log.Debug("Launching history command", "account", address, "from", from, "to", to)

	fbc := &findBlocksCommand{
		account:            address,
		db:                 c.db,
		blockRangeDAO:      c.blockRangeDAO,
		chainClient:        c.chainClient,
		balanceCache:       c.balanceCache,
		feed:               c.feed,
		noLimit:            false,
		fromBlockNumber:    from,
		toBlockNumber:      to,
		transactionManager: c.transactionManager,
	}
	group.Add(fbc.Command())
}

func (c *loadBlocksAndTransfersCommand) fetchNewBlocks(ctx context.Context, group *async.Group,
	address common.Address, blockRange *BlockRange, headNum *big.Int) {

	fromBlockNumber := new(big.Int).Add(blockRange.LastKnown, big.NewInt(1))

	log.Debug("Launching new blocks command", "chainID", c.chainClient.ChainID, "account", address, "from", fromBlockNumber, "headNum", headNum)

	// In case interval between checks is set smaller than block mining time,
	// we might need to wait for the next block to be mined
	if fromBlockNumber.Cmp(headNum) > 0 {
		return
	}

	newBlocksCmd := &findBlocksCommand{
		account:            address,
		db:                 c.db,
		blockRangeDAO:      c.blockRangeDAO,
		chainClient:        c.chainClient,
		balanceCache:       c.balanceCache,
		feed:               c.feed,
		noLimit:            false,
		fromBlockNumber:    fromBlockNumber,
		toBlockNumber:      headNum,
		transactionManager: c.transactionManager,
	}
	group.Add(newBlocksCmd.Command())
}

func (c *loadBlocksAndTransfersCommand) fetchTransfers(ctx context.Context, group *async.Group) {
	txCommand := &loadAllTransfersCommand{
		accounts:           c.accounts,
		db:                 c.db,
		blockDAO:           c.blockDAO,
		chainClient:        c.chainClient,
		transactionManager: c.transactionManager,
		blocksLimit:        noBlockLimit, // load transfers from all `unloaded` blocks
		feed:               c.feed,
	}

	group.Add(txCommand.Command())
}

func (c *loadBlocksAndTransfersCommand) notifyHistoryReady(address common.Address) {
	if c.feed != nil {
		c.feed.Send(walletevent.Event{
			Type:     EventRecentHistoryReady,
			Accounts: []common.Address{address},
		})
	}
}

func (c *loadBlocksAndTransfersCommand) areAllTransfersLoadedForAddress(address common.Address) (bool, error) {
	allBlocksLoaded, err := areAllHistoryBlocksLoadedForAddress(c.blockRangeDAO, c.chainClient.ChainID, address)
	if err != nil {
		log.Error("loadBlockAndTransfersCommand allHistoryBlocksLoaded", "error", err)
		return false, err
	}

	if allBlocksLoaded {
		firstHeader, err := c.blockDAO.GetFirstSavedBlock(c.chainClient.ChainID, address)
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
