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
	"github.com/status-im/status-go/services/wallet/balance"
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
	log.Debug("start findNewBlocksCommand", "account", c.account, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit)

	headNum, err := getHeadBlockNumber(parent, c.chainClient)
	if err != nil {
		// c.error = err
		return err // Might need to retry a couple of times
	}

	blockRange, err := loadBlockRangeInfo(c.chainClient.NetworkID(), c.account, c.blockRangeDAO)
	if err != nil {
		log.Error("findBlocksCommand loadBlockRangeInfo", "error", err)
		// c.error = err
		return err // Will keep spinning forever nomatter what
	}

	if blockRange != nil {
		c.fromBlockNumber = blockRange.LastKnown

		log.Debug("Launching new blocks command", "chainID", c.chainClient.NetworkID(), "account", c.account,
			"from", c.fromBlockNumber, "headNum", headNum)

		// In case interval between checks is set smaller than block mining time,
		// we might need to wait for the next block to be mined
		if c.fromBlockNumber.Cmp(headNum) >= 0 {
			return
		}

		c.toBlockNumber = headNum

		_ = c.findBlocksCommand.Run(parent)
	}

	return nil
}

// TODO NewFindBlocksCommand
type findBlocksCommand struct {
	account                   common.Address
	db                        *Database
	blockRangeDAO             *BlockRangeSequentialDAO
	chainClient               chain.ClientInterface
	balanceCacher             balance.Cacher
	feed                      *event.Feed
	noLimit                   bool
	transactionManager        *TransactionManager
	tokenManager              *token.Manager
	fromBlockNumber           *big.Int
	toBlockNumber             *big.Int
	blocksLoadedCh            chan<- []*DBHeader
	defaultNodeBlockChunkSize int

	// Not to be set by the caller
	resFromBlock           *Block
	startBlockNumber       *big.Int
	reachedETHHistoryStart bool
	error                  error
}

func (c *findBlocksCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *findBlocksCommand) ERC20ScanByBalance(parent context.Context, fromBlock, toBlock *big.Int, token common.Address) ([]*DBHeader, error) {
	var err error
	batchSize := getErc20BatchSize(c.chainClient.NetworkID())
	ranges := [][]*big.Int{{fromBlock, toBlock}}
	foundHeaders := []*DBHeader{}
	cache := map[int64]*big.Int{}
	for {
		nextRanges := [][]*big.Int{}
		for _, blockRange := range ranges {
			from, to := blockRange[0], blockRange[1]
			fromBalance, ok := cache[from.Int64()]
			if !ok {
				fromBalance, err = c.tokenManager.GetTokenBalanceAt(parent, c.chainClient, c.account, token, from)
				if err != nil {
					return nil, err
				}

				if fromBalance == nil {
					fromBalance = big.NewInt(0)
				}
				cache[from.Int64()] = fromBalance
			}

			toBalance, ok := cache[to.Int64()]
			if !ok {
				toBalance, err = c.tokenManager.GetTokenBalanceAt(parent, c.chainClient, c.account, token, to)
				if err != nil {
					return nil, err
				}
				if toBalance == nil {
					toBalance = big.NewInt(0)
				}
				cache[to.Int64()] = toBalance
			}

			if fromBalance.Cmp(toBalance) != 0 {
				diff := new(big.Int).Sub(to, from)
				if diff.Cmp(batchSize) <= 0 {
					headers, err := c.fastIndexErc20(parent, from, to)
					if err != nil {
						return nil, err
					}
					foundHeaders = append(foundHeaders, headers...)

					continue
				}

				halfOfDiff := new(big.Int).Div(diff, big.NewInt(2))
				mid := new(big.Int).Add(from, halfOfDiff)

				nextRanges = append(nextRanges, []*big.Int{from, mid})
				nextRanges = append(nextRanges, []*big.Int{mid, to})
			}
		}

		if len(nextRanges) == 0 {
			break
		}

		ranges = nextRanges
	}

	return foundHeaders, nil
}

func (c *findBlocksCommand) checkERC20Tail(parent context.Context) ([]*DBHeader, error) {
	log.Debug("checkERC20Tail", "account", c.account, "to block", c.startBlockNumber, "from", c.resFromBlock.Number)
	tokens, err := c.tokenManager.GetTokens(c.chainClient.NetworkID())
	if err != nil {
		return nil, err
	}
	addresses := make([]common.Address, len(tokens))
	for i, token := range tokens {
		addresses[i] = token.Address
	}

	from := new(big.Int).Sub(c.resFromBlock.Number, big.NewInt(1))

	clients := make(map[uint64]chain.ClientInterface, 1)
	clients[c.chainClient.NetworkID()] = c.chainClient
	atBlocks := make(map[uint64]*big.Int, 1)
	atBlocks[c.chainClient.NetworkID()] = from
	balances, err := c.tokenManager.GetBalancesAtByChain(parent, clients, []common.Address{c.account}, addresses, atBlocks)
	if err != nil {
		return nil, err
	}

	headers := []*DBHeader{}
	for token, balance := range balances[c.chainClient.NetworkID()][c.account] {
		bigintBalance := big.NewInt(balance.ToInt().Int64())
		if bigintBalance.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		result, err := c.ERC20ScanByBalance(parent, big.NewInt(0), from, token)
		if err != nil {
			return nil, err
		}

		headers = append(headers, result...)
	}

	return headers, nil
}

func (c *findBlocksCommand) Run(parent context.Context) (err error) {
	log.Debug("start findBlocksCommand", "account", c.account, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit, "from", c.fromBlockNumber, "to", c.toBlockNumber)

	rangeSize := big.NewInt(int64(c.defaultNodeBlockChunkSize))

	from, to := new(big.Int).Set(c.fromBlockNumber), new(big.Int).Set(c.toBlockNumber)

	// Limit the range size to DefaultNodeBlockChunkSize
	if new(big.Int).Sub(to, from).Cmp(rangeSize) > 0 {
		from.Sub(to, rangeSize)
	}

	for {
		var headers []*DBHeader
		if c.reachedETHHistoryStart {
			if c.fromBlockNumber.Cmp(zero) == 0 && c.startBlockNumber != nil && c.startBlockNumber.Cmp(zero) == 1 {
				headers, err = c.checkERC20Tail(parent)
				if err != nil {
					c.error = err
				}
			}
		} else {
			headers, _ = c.checkRange(parent, from, to)
		}

		if c.error != nil {
			log.Error("findBlocksCommand checkRange", "error", c.error, "account", c.account,
				"chain", c.chainClient.NetworkID(), "from", from, "to", to)
			break
		}

		if len(headers) > 0 {
			log.Debug("findBlocksCommand saving headers", "len", len(headers), "lastBlockNumber", to,
				"balance", c.balanceCacher.Cache().GetBalance(c.account, c.chainClient.NetworkID(), to),
				"nonce", c.balanceCacher.Cache().GetNonce(c.account, c.chainClient.NetworkID(), to))

			err = c.db.SaveBlocks(c.chainClient.NetworkID(), c.account, headers)
			if err != nil {
				c.error = err
				// return err
				break
			}

			c.blocksFound(headers)
		}

		if c.reachedETHHistoryStart {
			break
		}

		err = c.upsertBlockRange(&BlockRange{c.startBlockNumber, c.resFromBlock.Number, to})
		if err != nil {
			break
		}

		if from.Cmp(to) == 0 {
			break
		}

		nextFrom, nextTo := nextRange(c.defaultNodeBlockChunkSize, c.resFromBlock.Number, c.fromBlockNumber)

		if nextFrom.Cmp(from) == 0 && nextTo.Cmp(to) == 0 {
			break
		}

		from = nextFrom
		to = nextTo

		if to.Cmp(c.fromBlockNumber) <= 0 || (c.startBlockNumber != nil &&
			c.startBlockNumber.Cmp(big.NewInt(0)) > 0 && to.Cmp(c.startBlockNumber) <= 0) {
			log.Debug("Checked all ranges, stop execution", "startBlock", c.startBlockNumber, "from", from, "to", to)
			c.reachedETHHistoryStart = true
		}
	}

	log.Debug("end findBlocksCommand", "account", c.account, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit)

	return nil
}

func (c *findBlocksCommand) blocksFound(headers []*DBHeader) {
	c.blocksLoadedCh <- headers
}

func (c *findBlocksCommand) upsertBlockRange(blockRange *BlockRange) error {
	log.Debug("upsert block range", "Start", blockRange.Start, "FirstKnown", blockRange.FirstKnown, "LastKnown", blockRange.LastKnown,
		"chain", c.chainClient.NetworkID(), "account", c.account)

	err := c.blockRangeDAO.upsertRange(c.chainClient.NetworkID(), c.account, blockRange)
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

	newFromBlock, ethHeaders, startBlock, err := c.fastIndex(parent, c.balanceCacher, fromBlock, to)
	if err != nil {
		log.Error("findBlocksCommand checkRange fastIndex", "err", err, "account", c.account,
			"chain", c.chainClient.NetworkID())
		c.error = err
		// return err // In case c.noLimit is true, hystrix "max concurrency" may be reached and we will not be able to index ETH transfers
		return nil, nil
	}
	log.Debug("findBlocksCommand checkRange", "chainID", c.chainClient.NetworkID(), "account", c.account,
		"startBlock", startBlock, "newFromBlock", newFromBlock.Number, "toBlockNumber", to, "noLimit", c.noLimit)

	// There could be incoming ERC20 transfers which don't change the balance
	// and nonce of ETH account, so we keep looking for them
	erc20Headers, err := c.fastIndexErc20(parent, newFromBlock.Number, to)
	if err != nil {
		log.Error("findBlocksCommand checkRange fastIndexErc20", "err", err, "account", c.account, "chain", c.chainClient.NetworkID())
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

	log.Debug("end findBlocksCommand checkRange", "chainID", c.chainClient.NetworkID(), "account", c.account,
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
func (c *findBlocksCommand) fastIndex(ctx context.Context, bCacher balance.Cacher,
	fromBlock *Block, toBlockNumber *big.Int) (resultingFrom *Block, headers []*DBHeader,
	startBlock *big.Int, err error) {

	log.Debug("fast index started", "chainID", c.chainClient.NetworkID(), "account", c.account,
		"from", fromBlock.Number, "to", toBlockNumber)

	start := time.Now()
	group := async.NewGroup(ctx)

	command := &ethHistoricalCommand{
		chainClient:   c.chainClient,
		balanceCacher: bCacher,
		address:       c.account,
		feed:          c.feed,
		from:          fromBlock,
		to:            toBlockNumber,
		noLimit:       c.noLimit,
		threadLimit:   SequentialThreadLimit,
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
		log.Debug("fast indexer finished", "chainID", c.chainClient.NetworkID(), "account", c.account, "in", time.Since(start),
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
		log.Debug("fast indexer Erc20 finished", "chainID", c.chainClient.NetworkID(), "account", c.account,
			"in", time.Since(start), "headers", len(headers))
		return headers, nil
	}
}

func loadTransfersLoop(ctx context.Context, account common.Address, blockDAO *BlockDAO, db *Database,
	chainClient chain.ClientInterface, transactionManager *TransactionManager, pendingTxManager *transactions.PendingTxTracker,
	tokenManager *token.Manager, feed *event.Feed, blocksLoadedCh <-chan []*DBHeader) {

	log.Debug("loadTransfersLoop start", "chain", chainClient.NetworkID(), "account", account)

	for {
		select {
		case <-ctx.Done():
			log.Info("loadTransfersLoop error", "chain", chainClient.NetworkID(), "account", account, "error", ctx.Err())
			return
		case dbHeaders := <-blocksLoadedCh:
			log.Debug("loadTransfersOnDemand transfers received", "chain", chainClient.NetworkID(), "account", account, "headers", len(dbHeaders))

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
	blockDAO *BlockDAO, chainClient chain.ClientInterface, feed *event.Feed,
	transactionManager *TransactionManager, pendingTxManager *transactions.PendingTxTracker,
	tokenManager *token.Manager, balanceCacher balance.Cacher) *loadBlocksAndTransfersCommand {

	return &loadBlocksAndTransfersCommand{
		account:            account,
		db:                 db,
		blockRangeDAO:      &BlockRangeSequentialDAO{db.client},
		blockDAO:           blockDAO,
		chainClient:        chainClient,
		feed:               feed,
		balanceCacher:      balanceCacher,
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
	chainClient   chain.ClientInterface
	feed          *event.Feed
	balanceCacher balance.Cacher
	errorsCount   int
	// nonArchivalRPCNode bool // TODO Make use of it
	transactionManager *TransactionManager
	pendingTxManager   *transactions.PendingTxTracker
	tokenManager       *token.Manager
	blocksLoadedCh     chan []*DBHeader

	// Not to be set by the caller
	transfersLoaded bool // For event RecentHistoryReady to be sent only once per account during app lifetime
}

func (c *loadBlocksAndTransfersCommand) Run(parent context.Context) error {
	log.Debug("start load all transfers command", "chain", c.chainClient.NetworkID(), "account", c.account)

	ctx := parent
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
		log.Debug("end loadBlocksAndTransfers command", "chain", c.chainClient.NetworkID(), "account", c.account)
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

	log.Debug("fetchHistoryBlocks start", "chainID", c.chainClient.NetworkID(), "account", c.account)

	headNum, err := getHeadBlockNumber(ctx, c.chainClient)
	if err != nil {
		// c.error = err
		return err // Might need to retry a couple of times
	}

	blockRange, err := loadBlockRangeInfo(c.chainClient.NetworkID(), c.account, c.blockRangeDAO)
	if err != nil {
		log.Error("findBlocksCommand loadBlockRangeInfo", "error", err)
		// c.error = err
		return err // Will keep spinning forever nomatter what
	}

	/// first
	allHistoryLoaded := areAllHistoryBlocksLoaded(blockRange)
	to := getToHistoryBlockNumber(headNum, blockRange, allHistoryLoaded)

	log.Debug("fetchHistoryBlocks", "chainID", c.chainClient.NetworkID(), "account", c.account, "to", to, "allHistoryLoaded", allHistoryLoaded)

	if !allHistoryLoaded {
		fbc := &findBlocksCommand{
			account:                   c.account,
			db:                        c.db,
			blockRangeDAO:             c.blockRangeDAO,
			chainClient:               c.chainClient,
			balanceCacher:             c.balanceCacher,
			feed:                      c.feed,
			noLimit:                   false,
			fromBlockNumber:           big.NewInt(0),
			toBlockNumber:             to,
			transactionManager:        c.transactionManager,
			tokenManager:              c.tokenManager,
			blocksLoadedCh:            blocksLoadedCh,
			defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
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

	log.Debug("fetchHistoryBlocks end", "chainID", c.chainClient.NetworkID(), "account", c.account)

	return nil
}

func (c *loadBlocksAndTransfersCommand) startFetchingNewBlocks(group *async.Group, address common.Address, blocksLoadedCh chan<- []*DBHeader) {

	log.Debug("startFetchingNewBlocks", "chainID", c.chainClient.NetworkID(), "account", address)

	newBlocksCmd := &findNewBlocksCommand{
		findBlocksCommand: &findBlocksCommand{
			account:            address,
			db:                 c.db,
			blockRangeDAO:      c.blockRangeDAO,
			chainClient:        c.chainClient,
			balanceCacher:      c.balanceCacher,
			feed:               c.feed,
			noLimit:            false,
			transactionManager: c.transactionManager,
			tokenManager:       c.tokenManager,
			blocksLoadedCh:     blocksLoadedCh,
		},
	}
	group.Add(newBlocksCmd.Command())
}

func (c *loadBlocksAndTransfersCommand) fetchTransfersForLoadedBlocks(group *async.Group) error {

	log.Debug("fetchTransfers start", "chainID", c.chainClient.NetworkID(), "account", c.account)

	blocks, err := c.blockDAO.GetBlocksToLoadByAddress(c.chainClient.NetworkID(), c.account, numberOfBlocksCheckedPerIteration)
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
			ChainID:  c.chainClient.NetworkID(),
		})
	}
}

func (c *loadBlocksAndTransfersCommand) areAllTransfersLoaded() (bool, error) {
	allBlocksLoaded, err := areAllHistoryBlocksLoadedForAddress(c.blockRangeDAO, c.chainClient.NetworkID(), c.account)
	if err != nil {
		log.Error("loadBlockAndTransfersCommand allHistoryBlocksLoaded", "error", err)
		return false, err
	}

	if allBlocksLoaded {
		firstHeader, err := c.blockDAO.GetFirstSavedBlock(c.chainClient.NetworkID(), c.account)
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
func getHeadBlockNumber(parent context.Context, chainClient chain.ClientInterface) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	head, err := chainClient.HeaderByNumber(ctx, nil)
	cancel()
	if err != nil {
		return nil, err
	}

	return head.Number, err
}

func nextRange(maxRangeSize int, prevFrom, zeroBlockNumber *big.Int) (*big.Int, *big.Int) {
	log.Debug("next range start", "from", prevFrom, "zeroBlockNumber", zeroBlockNumber)

	rangeSize := big.NewInt(int64(maxRangeSize))

	to := big.NewInt(0).Set(prevFrom)
	from := big.NewInt(0).Sub(to, rangeSize)
	if from.Cmp(zeroBlockNumber) < 0 {
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
