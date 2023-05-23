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
			log.Error("findBlocksCommand checkRange", "error", c.error)
			break
		}

		log.Debug("findBlocksCommand saving headers", "len", len(headers), "lastBlockNumber", to,
			"balance", c.balanceCache.ReadCachedBalance(c.account, to),
			"nonce", c.balanceCache.ReadCachedNonce(c.account, to))

		err = c.db.SaveBlocks(c.chainClient.ChainID, c.account, headers)
		if err != nil {
			c.error = err
			// return err
			break
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
		log.Error("findBlocksCommand checkRange fastIndex", "err", err)
		c.error = err
		// return err // In case c.noLimit is true, hystrix "max concurrency" may be reached and we will not be able to index ETH transfers
		return nil, nil
	}
	log.Debug("findBlocksCommand checkRange", "startBlock", startBlock, "newFromBlock", newFromBlock.Number, "toBlockNumber", to, "noLimit", c.noLimit)

	// There should be transfers when either when we have found headers
	// or newFromBlock is different from fromBlock
	if len(ethHeaders) > 0 || newFromBlock.Number.Cmp(fromBlock.Number) != 0 {
		erc20Headers, err := c.fastIndexErc20(parent, newFromBlock.Number, to)
		if err != nil {
			log.Error("findBlocksCommand checkRange fastIndexErc20", "err", err)
			c.error = err
			// return err
			return nil, nil
		}

		allHeaders := append(ethHeaders, erc20Headers...)

		if len(allHeaders) > 0 {
			foundHeaders = uniqueHeaders(allHeaders)
		}
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

// Returns if all the blocks prior to first known block are loaded, not considering
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
		eth: &ETHDownloader{
			chainClient: c.chainClient,
			accounts:    []common.Address{c.account},
			signer:      types.NewLondonSigner(c.chainClient.ToBigInt()),
			db:          c.db,
		},
		feed:        c.feed,
		from:        fromBlock,
		to:          toBlockNumber,
		noLimit:     c.noLimit,
		threadLimit: SequentialThreadLimit,
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
	blocksLimit        int
}

func (c *loadAllTransfersCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *loadAllTransfersCommand) Run(parent context.Context) error {
	return loadTransfersLoop(parent, c.accounts, c.blockDAO, c.db, c.chainClient, c.blocksLimit, c.blocksByAddress, c.transactionManager)
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
}

func (c *loadBlocksAndTransfersCommand) Run(parent context.Context) error {
	log.Debug("start load all transfers command", "chain", c.chainClient.ChainID)

	ctx := parent

	if c.feed != nil {
		c.feed.Send(walletevent.Event{
			Type:     EventFetchingRecentHistory,
			Accounts: c.accounts,
		})
	}

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
			c.fetchHistoryBlocks(ctx, group, address, blockRange, toHistoryBlockNum, headNum)
		}

		// If no block ranges are stored, all blocks will be fetched by startFetchingHistoryBlocks method
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
	address common.Address, blockRange *BlockRange, toHistoryBlockNum *big.Int, headNum *big.Int) {

	log.Info("Launching history command")

	fbc := &findBlocksCommand{
		account:            address,
		db:                 c.db,
		blockRangeDAO:      c.blockRangeDAO,
		chainClient:        c.chainClient,
		balanceCache:       c.balanceCache,
		feed:               c.feed,
		noLimit:            false,
		fromBlockNumber:    big.NewInt(0), // Beginning of the chain history
		toBlockNumber:      toHistoryBlockNum,
		transactionManager: c.transactionManager,
	}
	group.Add(fbc.Command())
}

func (c *loadBlocksAndTransfersCommand) fetchNewBlocks(ctx context.Context, group *async.Group,
	address common.Address, blockRange *BlockRange, headNum *big.Int) {

	log.Info("Launching new blocks command")
	fromBlockNumber := new(big.Int).Add(blockRange.LastKnown, big.NewInt(1))

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
	}

	group.Add(txCommand.Command())
}

func loadTransfersLoop(ctx context.Context, accounts []common.Address, blockDAO *BlockDAO, db *Database,
	chainClient *chain.ClientWithFallback, blocksLimitPerAccount int, blocksByAddress map[common.Address][]*big.Int,
	transactionManager *TransactionManager) error {

	log.Debug("loadTransfers start", "accounts", accounts, "chain", chainClient.ChainID, "limit", blocksLimitPerAccount)

	start := time.Now()
	group := async.NewGroup(ctx)

	for _, address := range accounts {
		// Take blocks from cache if available and disrespect the limit
		// If no blocks are available in cache, take blocks from DB respecting the limit
		// If no limit is set, take all blocks from DB
		blocks, ok := blocksByAddress[address]

		commands := []*transfersCommand{}
		for {
			if !ok {
				blocks, _ = blockDAO.GetBlocksByAddress(chainClient.ChainID, address, numberOfBlocksCheckedPerIteration)
			}

			for _, block := range blocks {
				transfers := &transfersCommand{
					db:          db,
					chainClient: chainClient,
					address:     address,
					eth: &ETHDownloader{
						chainClient: chainClient,
						accounts:    []common.Address{address},
						signer:      types.NewLondonSigner(chainClient.ToBigInt()),
						db:          db,
					},
					blockNum:           block,
					transactionManager: transactionManager,
				}
				commands = append(commands, transfers)
				group.Add(transfers.Command())
			}

			// We need to wait until the retrieved blocks are processed, otherwise
			// they will be retrieved again in the next iteration
			// It blocks transfer loading for single account at a time
			select {
			case <-ctx.Done():
				log.Info("loadTransfers transfersCommand error", "chain", chainClient.ChainID, "address", address, "error", ctx.Err())
				continue
				// return nil, ctx.Err()
			case <-group.WaitAsync():
				// TODO Remove when done debugging
				transfers := []Transfer{}
				for _, command := range commands {
					if len(command.fetchedTransfers) == 0 {
						continue
					}

					transfers = append(transfers, command.fetchedTransfers...)
				}
				log.Debug("loadTransfers finished for account", "address", address, "in", time.Since(start), "chain", chainClient.ChainID, "transfers", len(transfers), "limit", blocksLimitPerAccount)
			}

			if ok || len(blocks) == 0 ||
				(blocksLimitPerAccount > noBlockLimit && len(blocks) >= blocksLimitPerAccount) {
				log.Debug("loadTransfers breaking loop on block limits reached or 0 blocks", "chain", chainClient.ChainID, "address", address, "limit", blocksLimitPerAccount, "blocks", len(blocks))
				break
			}
		}
	}

	return nil
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

func uniqueHeaders(allHeaders []*DBHeader) []*DBHeader {
	uniqHeadersByHash := map[common.Hash]*DBHeader{}
	for _, header := range allHeaders {
		uniqHeader, ok := uniqHeadersByHash[header.Hash]
		if ok {
			if len(header.Erc20Transfers) > 0 {
				uniqHeader.Erc20Transfers = append(uniqHeader.Erc20Transfers, header.Erc20Transfers...)
			}
			uniqHeadersByHash[header.Hash] = uniqHeader
		} else {
			uniqHeadersByHash[header.Hash] = header
		}
	}

	uniqHeaders := []*DBHeader{}
	for _, header := range uniqHeadersByHash {
		uniqHeaders = append(uniqHeaders, header)
	}

	return uniqHeaders
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
