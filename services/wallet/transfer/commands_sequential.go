package transfer

import (
	"context"
	"math/big"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	allBlocksLoaded = "all blocks loaded"
)

// TODO NewFindBlocksCommand
type findBlocksCommand struct {
	account            common.Address
	db                 *Database
	blockDAO           *BlockRangeSequentialDAO
	chainClient        *chain.ClientWithFallback
	balanceCache       *balanceCache
	feed               *event.Feed
	noLimit            bool
	error              error
	resFromBlock       *Block
	startBlockNumber   *big.Int
	transactionManager *TransactionManager
}

func (c *findBlocksCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *findBlocksCommand) Run(parent context.Context) (err error) {
	log.Info("start findBlocksCommand", "account", c.account, "chain", c.chainClient.ChainID, "noLimit", c.noLimit)

	rangeSize := big.NewInt(DefaultNodeBlockChunkSize)

	to, err := c.loadFirstKnownBlockNumber()
	log.Info("findBlocksCommand", "firstKnownBlockNumber", to, "error", err)

	if err != nil {
		if err.Error() != allBlocksLoaded {
			c.error = err
		}

		return
	}

	var head *types.Header = nil

	if to == nil {
		ctx, cancel := context.WithTimeout(parent, 3*time.Second)
		head, err = c.chainClient.HeaderByNumber(ctx, nil)
		cancel()

		if err != nil {
			c.error = err
			log.Error("findBlocksCommand failed to get head block", "error", err)
			return nil
		}

		log.Info("current head is", "chain", c.chainClient.ChainID, "block number", head.Number)

		to = new(big.Int).Set(head.Number) // deep copy
	} else {
		to.Sub(to, big.NewInt(1))
	}

	var from = big.NewInt(0)
	if to.Cmp(rangeSize) > 0 {
		from.Sub(to, rangeSize)
	}

	for {
		headers, _ := c.checkRange(parent, from, to)
		if c.error != nil {
			log.Error("findBlocksCommand checkRange", "error", c.error)
			break
		}

		// 'to' is set to 'head' if 'last' block not found in DB
		if head != nil && to.Cmp(head.Number) == 0 {
			log.Info("update blockrange", "head", head.Number, "to", to, "chain", c.chainClient.ChainID, "account", c.account)

			err = c.blockDAO.upsertRange(c.chainClient.ChainID, c.account, c.startBlockNumber,
				c.resFromBlock.Number, to)
			if err != nil {
				c.error = err
				log.Error("findBlocksCommand upsertRange", "error", err)
				break
			}
		}

		log.Info("findBlocksCommand.Run()", "headers len", len(headers), "resFromBlock", c.resFromBlock.Number)
		err = c.blockDAO.updateFirstBlock(c.chainClient.ChainID, c.account, c.resFromBlock.Number)
		if err != nil {
			c.error = err
			log.Error("findBlocksCommand failed to update first block", "error", err)
			break
		}

		if c.startBlockNumber.Cmp(big.NewInt(0)) > 0 {
			err = c.blockDAO.updateStartBlock(c.chainClient.ChainID, c.account, c.startBlockNumber)
			if err != nil {
				c.error = err
				log.Error("findBlocksCommand failed to update start block", "error", err)
				break
			}
		}

		// Assign new range
		to.Sub(from, big.NewInt(1)) // it won't hit the cache, but we wont load the transfers twice
		if to.Cmp(rangeSize) > 0 {
			from.Sub(to, rangeSize)
		} else {
			from = big.NewInt(0)
		}

		if to.Cmp(big.NewInt(0)) <= 0 || (c.startBlockNumber != nil &&
			c.startBlockNumber.Cmp(big.NewInt(0)) > 0 && to.Cmp(c.startBlockNumber) <= 0) {
			log.Info("Start block has been found, stop execution", "startBlock", c.startBlockNumber, "to", to)
			break
		}
	}

	log.Info("end findBlocksCommand", "account", c.account, "chain", c.chainClient.ChainID, "noLimit", c.noLimit)

	return nil
}

func (c *findBlocksCommand) checkRange(parent context.Context, from *big.Int, to *big.Int) (
	foundHeaders []*DBHeader, err error) {

	fromBlock := &Block{Number: from}

	newFromBlock, ethHeaders, startBlock, err := c.fastIndex(parent, c.balanceCache, fromBlock, to)
	log.Info("findBlocksCommand checkRange", "startBlock", startBlock, "newFromBlock", newFromBlock.Number, "toBlockNumber", to, "noLimit", c.noLimit)
	if err != nil {
		log.Info("findBlocksCommand checkRange fastIndex", "err", err)
		c.error = err
		// return err // In case c.noLimit is true, hystrix "max concurrency" may be reached and we will not be able to index ETH transfers
		return nil, nil
	}

	// TODO There should be transfers when either when we have found headers
	// or when newFromBlock is different from fromBlock, but if I check for
	// ERC20 transfers only when there are ETH transfers, I will miss ERC20 transfers

	// if len(ethHeaders) > 0 || newFromBlock.Number.Cmp(fromBlock.Number) != 0 { // there is transaction history for this account

	erc20Headers, err := c.fastIndexErc20(parent, newFromBlock.Number, to)
	if err != nil {
		log.Info("findBlocksCommand checkRange fastIndexErc20", "err", err)
		c.error = err
		// return err
		return nil, nil
	}

	allHeaders := append(ethHeaders, erc20Headers...)

	if len(allHeaders) > 0 {
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

		foundHeaders = uniqHeaders

		log.Info("saving headers", "len", len(uniqHeaders), "lastBlockNumber", to,
			"balance", c.balanceCache.ReadCachedBalance(c.account, to),
			"nonce", c.balanceCache.ReadCachedNonce(c.account, to))

		err = c.db.SaveBlocks(c.chainClient.ChainID, c.account, uniqHeaders)
		if err != nil {
			c.error = err
			// return err
			return nil, nil
		}

		sort.SliceStable(foundHeaders, func(i, j int) bool {
			return foundHeaders[i].Number.Cmp(foundHeaders[j].Number) == 1
		})
	}
	// }

	c.resFromBlock = newFromBlock
	c.startBlockNumber = startBlock

	log.Info("end findBlocksCommand checkRange", "c.startBlock", c.startBlockNumber, "newFromBlock", newFromBlock.Number,
		"toBlockNumber", to, "c.resFromBlock", c.resFromBlock.Number)

	return
}

func (c *findBlocksCommand) loadFirstKnownBlockNumber() (*big.Int, error) {
	blockInfo, err := c.blockDAO.getBlockRange(c.chainClient.ChainID, c.account)
	if err != nil {
		log.Error("failed to load block ranges from database", "chain", c.chainClient.ChainID, "account", c.account, "error", err)
		return nil, err
	}

	if blockInfo != nil {
		log.Info("blockInfo for", "address", c.account, "chain", c.chainClient.ChainID, "Start",
			blockInfo.Start, "FirstKnown", blockInfo.FirstKnown, "LastKnown", blockInfo.LastKnown)

		// Check if we have fetched all blocks for this account
		if blockInfo.FirstKnown != nil && blockInfo.Start != nil && blockInfo.Start.Cmp(blockInfo.FirstKnown) >= 0 {
			log.Info("all blocks fetched", "chain", c.chainClient.ChainID, "account", c.account)
			return blockInfo.FirstKnown, errors.New(allBlocksLoaded)
		}

		return blockInfo.FirstKnown, nil
	}

	log.Info("no blockInfo for", "address", c.account, "chain", c.chainClient.ChainID)

	return nil, nil
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findBlocksCommand) fastIndex(ctx context.Context, bCache *balanceCache,
	fromBlock *Block, toBlockNumber *big.Int) (resultingFrom *Block, headers []*DBHeader,
	startBlock *big.Int, err error) {

	log.Info("fast index started", "accounts", c.account, "from", fromBlock.Number, "to", toBlockNumber)

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
		log.Info("fast indexer finished", "in", time.Since(start), "startBlock", command.startBlock, "resultingFrom", resultingFrom.Number, "headers", len(headers))
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
		log.Info("fast indexer Erc20 finished", "in", time.Since(start), "headers", len(headers))
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
	erc20         *ERC20TransfersDownloader
	chainClient   *chain.ClientWithFallback
	feed          *event.Feed
	balanceCache  *balanceCache
	errorsCount   int
	// nonArchivalRPCNode bool // TODO Make use of it
	transactionManager *TransactionManager
}

func (c *loadBlocksAndTransfersCommand) Run(parent context.Context) error {
	log.Info("start load all transfers command", "chain", c.chainClient.ChainID)

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

	for _, address := range c.accounts {
		log.Info("start findBlocks command", "chain", c.chainClient.ChainID)

		fbc := &findBlocksCommand{
			account:            address,
			db:                 c.db,
			blockDAO:           c.blockRangeDAO,
			chainClient:        c.chainClient,
			balanceCache:       c.balanceCache,
			feed:               c.feed,
			noLimit:            false,
			transactionManager: c.transactionManager,
		}
		group.Add(fbc.Command())
	}

	txCommand := &loadAllTransfersCommand{
		accounts:           c.accounts,
		db:                 c.db,
		blockDAO:           c.blockDAO,
		chainClient:        c.chainClient,
		transactionManager: c.transactionManager,
		blocksLimit:        noBlockLimit, // load transfers from all `unloaded` blocks
	}

	group.Add(txCommand.Command())

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Info("end load all transfers command", "chain", c.chainClient.ChainID)
		return nil
	}
}

func (c *loadBlocksAndTransfersCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func loadTransfersLoop(ctx context.Context, accounts []common.Address, blockDAO *BlockDAO, db *Database,
	chainClient *chain.ClientWithFallback, blocksLimitPerAccount int, blocksByAddress map[common.Address][]*big.Int,
	transactionManager *TransactionManager) error {

	log.Info("loadTransfers start", "accounts", accounts, "chain", chainClient.ChainID, "limit", blocksLimitPerAccount)

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
				log.Info("loadTransfers finished for account", "address", address, "in", time.Since(start), "chain", chainClient.ChainID, "transfers", len(transfers), "limit", blocksLimitPerAccount)
			}

			log.Info("loadTransfers after select", "chain", chainClient.ChainID, "address", address, "blocks.len", len(blocks))

			if ok || len(blocks) == 0 ||
				(blocksLimitPerAccount > noBlockLimit && len(blocks) >= blocksLimitPerAccount) {
				log.Info("loadTransfers breaking loop on block limits reached or 0 blocks", "chain", chainClient.ChainID, "address", address, "limit", blocksLimitPerAccount, "blocks", len(blocks))
				break
			}
		}
	}

	return nil
}
