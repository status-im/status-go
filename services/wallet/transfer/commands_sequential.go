package transfer

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	nodetypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"
)

var trLoopCnt int = 0
var fetchNewBlocksCnt int = 0

func verifyOnce(cnt *int, msg string) {
	if *cnt > 0 {
		panic("verifyOnce, function: " + msg)
	}
	*cnt++
}

type findNewBlocksCommand struct {
	*findBlocksCommand
}

func (c *findNewBlocksCommand) Command() async.Command {
	return async.InfiniteCommand{
		// TODO - make it configurable based on chain block mining time
		// NOTE(rasom): ^ it is unclear why each block has to be checked,
		// that is rather undesirable, as it causes a lot of RPC requests
		Interval: 2 * time.Minute,
		Runable:  c.Run,
	}.Run
}

func (c *findNewBlocksCommand) Run(parent context.Context) (err error) {
	headNum, err := getHeadBlockNumber(parent, c.chainClient)
	if err != nil {
		log.Error("findNewBlocksCommand getHeadBlockNumber", "error", err, "chain", c.chainClient.NetworkID())
		return err
	}

	// In case this is the first check, skip it, history fetching will do it
	if c.fromBlockNumber.Cmp(headNum) >= 0 {
		return nil
	}

	c.findAndSaveEthBlocks(parent, c.fromBlockNumber, headNum)
	c.findAndSaveTokenBlocks(parent, c.fromBlockNumber, headNum)

	c.fromBlockNumber = headNum

	return nil
}

func (c *findNewBlocksCommand) findAndSaveEthBlocks(parent context.Context, fromNum, headNum *big.Int) {
	// Check ETH transfers for each account independently
	mnemonicWasNotShown, err := c.accountsDB.GetMnemonicWasNotShown()
	if err != nil {
		c.error = err
		return
	}

	for _, account := range c.accounts {
		if mnemonicCheckEnabled && mnemonicWasNotShown {
			acc, err := c.accountsDB.GetAccountByAddress(nodetypes.Address(account))
			if err != nil {
				c.error = err
				return
			}
			if acc.AddressWasNotShown {
				log.Info("skip findNewBlocksCommand, mnemonic has not been shown and the address has not been shared yet", "address", account)
				continue
			}
		}

		log.Debug("start findNewBlocksCommand", "account", account, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit, "from", fromNum, "to", headNum)

		headers, startBlockNum, _ := c.findBlocksWithEthTransfers(parent, account, fromNum, headNum)
		if len(headers) > 0 {
			log.Debug("findNewBlocksCommand saving headers", "len", len(headers), "lastBlockNumber", headNum,
				"balance", c.balanceCacher.Cache().GetBalance(account, c.chainClient.NetworkID(), headNum),
				"nonce", c.balanceCacher.Cache().GetNonce(account, c.chainClient.NetworkID(), headNum))

			err := c.db.SaveBlocks(c.chainClient.NetworkID(), headers)
			if err != nil {
				c.error = err
				break
			}

			c.blocksFound(headers)
		}

		err := c.markEthBlockRangeChecked(account, &BlockRange{startBlockNum, fromNum, headNum})
		if err != nil {
			c.error = err
			break
		}

		log.Debug("end findNewBlocksCommand", "account", account, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit, "from", fromNum, "to", headNum)
	}
}

func (c *findNewBlocksCommand) findAndSaveTokenBlocks(parent context.Context, fromNum, headNum *big.Int) {
	// Check token transfers for all accounts.
	// Each account's last checked block can be different, so we can get duplicated headers,
	// so we need to deduplicate them
	const incomingOnly = false
	erc20Headers, err := c.fastIndexErc20(parent, fromNum, headNum, incomingOnly)
	if err != nil {
		log.Error("findNewBlocksCommand fastIndexErc20", "err", err, "account", c.accounts, "chain", c.chainClient.NetworkID())
		c.error = err
		return
	}

	if len(erc20Headers) > 0 {
		log.Debug("findNewBlocksCommand saving headers", "len", len(erc20Headers), "from", fromNum, "to", headNum)

		// get not loaded headers from DB for all accs and blocks
		preLoadedTransactions, err := c.db.GetTransactionsToLoad(c.chainClient.NetworkID(), common.Address{}, nil)
		if err != nil {
			c.error = err
			return
		}

		tokenBlocksFiltered := filterNewPreloadedTransactions(erc20Headers, preLoadedTransactions)

		err = c.db.SaveBlocks(c.chainClient.NetworkID(), tokenBlocksFiltered)
		if err != nil {
			c.error = err
			return
		}

		c.blocksFound(tokenBlocksFiltered)
	}

	err = c.markTokenBlockRangeChecked(c.accounts, fromNum, headNum)
	if err != nil {
		c.error = err
		return
	}
}

func (c *findNewBlocksCommand) markTokenBlockRangeChecked(accounts []common.Address, from, to *big.Int) error {
	log.Debug("markTokenBlockRangeChecked", "chain", c.chainClient.NetworkID(), "from", from.Uint64(), "to", to.Uint64())

	for _, account := range accounts {
		err := c.blockRangeDAO.updateTokenRange(c.chainClient.NetworkID(), account, &BlockRange{LastKnown: to})
		if err != nil {
			c.error = err
			log.Error("findNewBlocksCommand upsertTokenRange", "error", err)
			return err
		}
	}

	return nil
}

func filterNewPreloadedTransactions(erc20Headers []*DBHeader, preLoadedTransfers []*PreloadedTransaction) []*DBHeader {
	var uniqueErc20Headers []*DBHeader
	for _, header := range erc20Headers {
		loaded := false
		for _, transfer := range preLoadedTransfers {
			if header.PreloadedTransactions[0].ID == transfer.ID {
				loaded = true
				break
			}
		}

		if !loaded {
			uniqueErc20Headers = append(uniqueErc20Headers, header)
		}
	}

	return uniqueErc20Headers
}

func (c *findNewBlocksCommand) findBlocksWithEthTransfers(parent context.Context, account common.Address, fromOrig, toOrig *big.Int) (headers []*DBHeader, startBlockNum *big.Int, err error) {
	log.Debug("start findNewBlocksCommand::findBlocksWithEthTransfers", "account", account, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit, "from", c.fromBlockNumber, "to", c.toBlockNumber)

	rangeSize := big.NewInt(int64(c.defaultNodeBlockChunkSize))

	from, to := new(big.Int).Set(fromOrig), new(big.Int).Set(toOrig)

	// Limit the range size to DefaultNodeBlockChunkSize
	if new(big.Int).Sub(to, from).Cmp(rangeSize) > 0 {
		from.Sub(to, rangeSize)
	}

	for {
		if from.Cmp(to) == 0 {
			log.Debug("findNewBlocksCommand empty range", "from", from, "to", to)
			break
		}

		fromBlock := &Block{Number: from}

		var newFromBlock *Block
		var ethHeaders []*DBHeader
		newFromBlock, ethHeaders, startBlockNum, err = c.fastIndex(parent, account, c.balanceCacher, fromBlock, to)
		if err != nil {
			log.Error("findNewBlocksCommand checkRange fastIndex", "err", err, "account", account,
				"chain", c.chainClient.NetworkID())
			c.error = err
			// return err // In case c.noLimit is true, hystrix "max concurrency" may be reached and we will not be able to index ETH transfers
			return nil, nil, nil
		}
		log.Debug("findNewBlocksCommand checkRange", "chainID", c.chainClient.NetworkID(), "account", account,
			"startBlock", startBlockNum, "newFromBlock", newFromBlock.Number, "toBlockNumber", to, "noLimit", c.noLimit)

		headers = append(headers, ethHeaders...)

		if startBlockNum != nil && startBlockNum.Cmp(from) >= 0 {
			log.Debug("Checked all ranges, stop execution", "startBlock", startBlockNum, "from", from, "to", to)
			break
		}

		nextFrom, nextTo := nextRange(c.defaultNodeBlockChunkSize, newFromBlock.Number, fromOrig)

		if nextFrom.Cmp(from) == 0 && nextTo.Cmp(to) == 0 {
			log.Debug("findNewBlocksCommand empty next range", "from", from, "to", to)
			break
		}

		from = nextFrom
		to = nextTo
	}

	log.Debug("end findNewBlocksCommand::findBlocksWithEthTransfers", "account", account, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit)

	return headers, startBlockNum, nil
}

// TODO NewFindBlocksCommand
type findBlocksCommand struct {
	accounts                  []common.Address
	db                        *Database
	accountsDB                *accounts.Database
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

type ERC20BlockRange struct {
	from *big.Int
	to   *big.Int
}

func (c *findBlocksCommand) ERC20ScanByBalance(parent context.Context, account common.Address, fromBlock, toBlock *big.Int, token common.Address) ([]ERC20BlockRange, error) {
	var err error
	batchSize := getErc20BatchSize(c.chainClient.NetworkID())
	ranges := [][]*big.Int{{fromBlock, toBlock}}
	foundRanges := []ERC20BlockRange{}
	cache := map[int64]*big.Int{}
	for {
		nextRanges := [][]*big.Int{}
		for _, blockRange := range ranges {
			from, to := blockRange[0], blockRange[1]
			fromBalance, ok := cache[from.Int64()]
			if !ok {
				fromBalance, err = c.tokenManager.GetTokenBalanceAt(parent, c.chainClient, account, token, from)
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
				toBalance, err = c.tokenManager.GetTokenBalanceAt(parent, c.chainClient, account, token, to)
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
					foundRanges = append(foundRanges, ERC20BlockRange{from, to})
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

	return foundRanges, nil
}

func (c *findBlocksCommand) checkERC20Tail(parent context.Context, account common.Address) ([]*DBHeader, error) {
	log.Debug("checkERC20Tail", "account", account, "to block", c.startBlockNumber, "from", c.resFromBlock.Number)
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
	balances, err := c.tokenManager.GetBalancesAtByChain(parent, clients, []common.Address{account}, addresses, atBlocks)
	if err != nil {
		return nil, err
	}

	foundRanges := []ERC20BlockRange{}
	for token, balance := range balances[c.chainClient.NetworkID()][account] {
		bigintBalance := big.NewInt(balance.ToInt().Int64())
		if bigintBalance.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		result, err := c.ERC20ScanByBalance(parent, account, big.NewInt(0), from, token)
		if err != nil {
			return nil, err
		}

		foundRanges = append(foundRanges, result...)
	}

	uniqRanges := []ERC20BlockRange{}
	rangesMap := map[string]bool{}
	for _, rangeItem := range foundRanges {
		key := rangeItem.from.String() + "-" + rangeItem.to.String()
		if _, ok := rangesMap[key]; !ok {
			rangesMap[key] = true
			uniqRanges = append(uniqRanges, rangeItem)
		}
	}

	foundHeaders := []*DBHeader{}
	for _, rangeItem := range uniqRanges {
		headers, err := c.fastIndexErc20(parent, rangeItem.from, rangeItem.to, true)
		if err != nil {
			return nil, err
		}
		foundHeaders = append(foundHeaders, headers...)
	}

	return foundHeaders, nil
}

var mnemonicCheckEnabled = false

func (c *findBlocksCommand) Run(parent context.Context) (err error) {
	log.Debug("start findBlocksCommand", "accounts", c.accounts, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit, "from", c.fromBlockNumber, "to", c.toBlockNumber)

	account := c.accounts[0] // For now this command supports only 1 account
	mnemonicWasNotShown, err := c.accountsDB.GetMnemonicWasNotShown()
	if err != nil {
		c.error = err
		return err
	}

	if mnemonicCheckEnabled && mnemonicWasNotShown {
		account, err := c.accountsDB.GetAccountByAddress(nodetypes.BytesToAddress(account.Bytes()))
		if err != nil {
			c.error = err
			return err
		}
		if account.AddressWasNotShown {
			log.Info("skip findBlocksCommand, mnemonic has not been shown and the address has not been shared yet", "address", account)
			return nil
		}
	}

	rangeSize := big.NewInt(int64(c.defaultNodeBlockChunkSize))
	from, to := new(big.Int).Set(c.fromBlockNumber), new(big.Int).Set(c.toBlockNumber)

	// Limit the range size to DefaultNodeBlockChunkSize
	if new(big.Int).Sub(to, from).Cmp(rangeSize) > 0 {
		from.Sub(to, rangeSize)
	}

	for {
		if from.Cmp(to) == 0 {
			log.Debug("findBlocksCommand empty range", "from", from, "to", to)
			break
		}

		var headers []*DBHeader
		if c.reachedETHHistoryStart {
			if c.fromBlockNumber.Cmp(zero) == 0 && c.startBlockNumber != nil && c.startBlockNumber.Cmp(zero) == 1 {
				headers, err = c.checkERC20Tail(parent, account)
				if err != nil {
					c.error = err
				}
			}
		} else {
			headers, _ = c.checkRange(parent, from, to)
		}

		if c.error != nil {
			log.Error("findBlocksCommand checkRange", "error", c.error, "account", account,
				"chain", c.chainClient.NetworkID(), "from", from, "to", to)
			break
		}

		if len(headers) > 0 {
			log.Debug("findBlocksCommand saving headers", "len", len(headers), "lastBlockNumber", to,
				"balance", c.balanceCacher.Cache().GetBalance(account, c.chainClient.NetworkID(), to),
				"nonce", c.balanceCacher.Cache().GetNonce(account, c.chainClient.NetworkID(), to))

			err = c.db.SaveBlocks(c.chainClient.NetworkID(), headers)
			if err != nil {
				c.error = err
				// return err
				break
			}

			c.blocksFound(headers)
		}

		if c.reachedETHHistoryStart {
			log.Debug("findBlocksCommand reached first ETH transfer and checked erc20 tail", "chain", c.chainClient.NetworkID(), "account", account)
			break
		}

		err = c.markEthBlockRangeChecked(account, &BlockRange{c.startBlockNumber, c.resFromBlock.Number, to})
		if err != nil {
			break
		}

		// if we have found first ETH block and we have not reached the start of ETH history yet
		if c.startBlockNumber != nil && c.fromBlockNumber.Cmp(from) == -1 {
			log.Debug("ERC20 tail should be checked", "initial from", c.fromBlockNumber, "actual from", from, "first ETH block", c.startBlockNumber)
			c.reachedETHHistoryStart = true
			continue
		}

		if c.startBlockNumber != nil && c.startBlockNumber.Cmp(from) >= 0 {
			log.Debug("Checked all ranges, stop execution", "startBlock", c.startBlockNumber, "from", from, "to", to)
			break
		}

		nextFrom, nextTo := nextRange(c.defaultNodeBlockChunkSize, c.resFromBlock.Number, c.fromBlockNumber)

		if nextFrom.Cmp(from) == 0 && nextTo.Cmp(to) == 0 {
			log.Debug("findBlocksCommand empty next range", "from", from, "to", to)
			break
		}

		from = nextFrom
		to = nextTo
	}

	log.Debug("end findBlocksCommand", "account", account, "chain", c.chainClient.NetworkID(), "noLimit", c.noLimit)

	return nil
}

func (c *findBlocksCommand) blocksFound(headers []*DBHeader) {
	c.blocksLoadedCh <- headers
}

func (c *findBlocksCommand) markEthBlockRangeChecked(account common.Address, blockRange *BlockRange) error {
	log.Debug("upsert block range", "Start", blockRange.Start, "FirstKnown", blockRange.FirstKnown, "LastKnown", blockRange.LastKnown,
		"chain", c.chainClient.NetworkID(), "account", account)

	err := c.blockRangeDAO.upsertEthRange(c.chainClient.NetworkID(), account, blockRange)
	if err != nil {
		c.error = err
		log.Error("findBlocksCommand upsertRange", "error", err)
		return err
	}

	return nil
}

func (c *findBlocksCommand) checkRange(parent context.Context, from *big.Int, to *big.Int) (
	foundHeaders []*DBHeader, err error) {

	account := c.accounts[0]
	fromBlock := &Block{Number: from}

	newFromBlock, ethHeaders, startBlock, err := c.fastIndex(parent, account, c.balanceCacher, fromBlock, to)
	if err != nil {
		log.Error("findBlocksCommand checkRange fastIndex", "err", err, "account", account,
			"chain", c.chainClient.NetworkID())
		c.error = err
		// return err // In case c.noLimit is true, hystrix "max concurrency" may be reached and we will not be able to index ETH transfers
		return nil, nil
	}
	log.Debug("findBlocksCommand checkRange", "chainID", c.chainClient.NetworkID(), "account", account,
		"startBlock", startBlock, "newFromBlock", newFromBlock.Number, "toBlockNumber", to, "noLimit", c.noLimit)

	// There could be incoming ERC20 transfers which don't change the balance
	// and nonce of ETH account, so we keep looking for them
	erc20Headers, err := c.fastIndexErc20(parent, newFromBlock.Number, to, false)
	if err != nil {
		log.Error("findBlocksCommand checkRange fastIndexErc20", "err", err, "account", account, "chain", c.chainClient.NetworkID())
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

	log.Debug("end findBlocksCommand checkRange", "chainID", c.chainClient.NetworkID(), "account", account,
		"c.startBlock", c.startBlockNumber, "newFromBlock", newFromBlock.Number,
		"toBlockNumber", to, "c.resFromBlock", c.resFromBlock.Number)

	return
}

func loadBlockRangeInfo(chainID uint64, account common.Address, blockDAO *BlockRangeSequentialDAO) (
	*ethTokensBlockRanges, error) {

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
	if blockInfo != nil && blockInfo.FirstKnown != nil && blockInfo.Start != nil &&
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

	return areAllHistoryBlocksLoaded(blockRange.eth) && areAllHistoryBlocksLoaded(blockRange.tokens), nil
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findBlocksCommand) fastIndex(ctx context.Context, account common.Address, bCacher balance.Cacher,
	fromBlock *Block, toBlockNumber *big.Int) (resultingFrom *Block, headers []*DBHeader,
	startBlock *big.Int, err error) {

	log.Debug("fast index started", "chainID", c.chainClient.NetworkID(), "account", account,
		"from", fromBlock.Number, "to", toBlockNumber)

	start := time.Now()
	group := async.NewGroup(ctx)

	command := &ethHistoricalCommand{
		chainClient:   c.chainClient,
		balanceCacher: bCacher,
		address:       account,
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
		log.Debug("fast indexer ctx Done", "error", err)
		return
	case <-group.WaitAsync():
		if command.error != nil {
			err = command.error
			return
		}
		resultingFrom = &Block{Number: command.resultingFrom}
		headers = command.foundHeaders
		startBlock = command.startBlock
		log.Debug("fast indexer finished", "chainID", c.chainClient.NetworkID(), "account", account, "in", time.Since(start),
			"startBlock", command.startBlock, "resultingFrom", resultingFrom.Number, "headers", len(headers))
		return
	}
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findBlocksCommand) fastIndexErc20(ctx context.Context, fromBlockNumber *big.Int,
	toBlockNumber *big.Int, incomingOnly bool) ([]*DBHeader, error) {

	start := time.Now()
	group := async.NewGroup(ctx)

	erc20 := &erc20HistoricalCommand{
		erc20:        NewERC20TransfersDownloader(c.chainClient, c.accounts, types.LatestSignerForChainID(c.chainClient.ToBigInt()), incomingOnly),
		chainClient:  c.chainClient,
		feed:         c.feed,
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
		log.Debug("fast indexer Erc20 finished", "chainID", c.chainClient.NetworkID(),
			"in", time.Since(start), "headers", len(headers))
		return headers, nil
	}
}

func loadTransfersLoop(ctx context.Context, blockDAO *BlockDAO, db *Database,
	chainClient chain.ClientInterface, transactionManager *TransactionManager, pendingTxManager *transactions.PendingTxTracker,
	tokenManager *token.Manager, feed *event.Feed, blocksLoadedCh <-chan []*DBHeader) {

	log.Debug("loadTransfersLoop start", "chain", chainClient.NetworkID())

	verifyOnce(&trLoopCnt, "loadTransfersLoop")

	for {
		select {
		case <-ctx.Done():
			log.Info("loadTransfersLoop done", "chain", chainClient.NetworkID(), "error", ctx.Err())
			return
		case dbHeaders := <-blocksLoadedCh:
			log.Debug("loadTransfersOnDemand transfers received", "chain", chainClient.NetworkID(), "headers", len(dbHeaders))

			blocksByAddress := map[common.Address][]*big.Int{}
			// iterate over headers and group them by address
			for _, dbHeader := range dbHeaders {
				blocksByAddress[dbHeader.Address] = append(blocksByAddress[dbHeader.Address], dbHeader.Number)
			}

			go func() {
				_ = loadTransfers(ctx, blockDAO, db, chainClient, noBlockLimit,
					blocksByAddress, transactionManager, pendingTxManager, tokenManager, feed)
			}()
		}
	}
}

func newLoadBlocksAndTransfersCommand(accounts []common.Address, db *Database, accountsDB *accounts.Database,
	blockDAO *BlockDAO, blockRangesSeqDAO *BlockRangeSequentialDAO, chainClient chain.ClientInterface, feed *event.Feed,
	transactionManager *TransactionManager, pendingTxManager *transactions.PendingTxTracker,
	tokenManager *token.Manager, balanceCacher balance.Cacher, omitHistory bool) *loadBlocksAndTransfersCommand {

	return &loadBlocksAndTransfersCommand{
		accounts:           accounts,
		db:                 db,
		blockRangeDAO:      blockRangesSeqDAO,
		accountsDB:         accountsDB,
		blockDAO:           blockDAO,
		chainClient:        chainClient,
		feed:               feed,
		balanceCacher:      balanceCacher,
		transactionManager: transactionManager,
		pendingTxManager:   pendingTxManager,
		tokenManager:       tokenManager,
		blocksLoadedCh:     make(chan []*DBHeader, 100),
		omitHistory:        omitHistory,
	}
}

type loadBlocksAndTransfersCommand struct {
	accounts      []common.Address
	db            *Database
	accountsDB    *accounts.Database
	blockRangeDAO *BlockRangeSequentialDAO
	blockDAO      *BlockDAO
	chainClient   chain.ClientInterface
	feed          *event.Feed
	balanceCacher balance.Cacher
	// nonArchivalRPCNode bool // TODO Make use of it
	transactionManager *TransactionManager
	pendingTxManager   *transactions.PendingTxTracker
	tokenManager       *token.Manager
	blocksLoadedCh     chan []*DBHeader
	omitHistory        bool

	// Not to be set by the caller
	transfersLoaded map[common.Address]bool // For event RecentHistoryReady to be sent only once per account during app lifetime
	started         bool
}

// func (c *loadBlocksAndTransfersCommand) Run(ctx context.Context) error {
func (c *loadBlocksAndTransfersCommand) Run(ctx context.Context) error {
	log.Debug("start load all transfers command", "chain", c.chainClient.NetworkID(), "accounts", c.accounts)

	// Finite processes (to be restarted on error, but stopped on success):
	// fetching transfers for loaded blocks
	// fetching history blocks

	// Infinite processes (to be restarted on error):
	// fetching new blocks
	// fetching transfers for new blocks

	fromNum := big.NewInt(0)
	headNum, err := getHeadBlockNumber(ctx, c.chainClient)
	if err != nil {
		return err
	}

	group := async.NewAtomicGroup(ctx)
	defer func() {
		group.Stop()
		group.Wait()
	}()

	// It will start loadTransfersCommand which will run until success when all transfers from DB are loaded
	err = c.startFetchingTransfersForLoadedBlocks(group)
	if err != nil {
		log.Error("loadBlocksAndTransfersCommand fetchTransfersForLoadedBlocks", "error", err)
		return err
	}

	if !c.started {
		c.started = true
		c.startTransfersLoop(ctx)
		c.startFetchingNewBlocks(ctx, c.accounts, headNum, c.blocksLoadedCh)
	}

	// It will start findBlocksCommands which will run until success when all blocks are loaded
	err = c.fetchHistoryBlocks(group, c.accounts, fromNum, headNum, c.blocksLoadedCh)
	if err != nil {
		log.Error("loadBlocksAndTransfersCommand fetchHistoryBlocks", "error", err)
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Debug("end loadBlocksAndTransfers command", "chain", c.chainClient.NetworkID(), "accounts", c.accounts)
		return nil
	}
	return nil
}

func (c *loadBlocksAndTransfersCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

// Start transfers loop to load transfers for new blocks
func (c *loadBlocksAndTransfersCommand) startTransfersLoop(ctx context.Context) {
	go loadTransfersLoop(ctx, c.blockDAO, c.db, c.chainClient, c.transactionManager,
		c.pendingTxManager, c.tokenManager, c.feed, c.blocksLoadedCh)
}

func (c *loadBlocksAndTransfersCommand) fetchHistoryBlocks(group *async.AtomicGroup, accounts []common.Address, fromNum, toNum *big.Int, blocksLoadedCh chan []*DBHeader) (err error) {
	for _, account := range accounts {
		err = c.fetchHistoryBlocksForAccount(group, account, fromNum, toNum, c.blocksLoadedCh)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *loadBlocksAndTransfersCommand) fetchHistoryBlocksForAccount(group *async.AtomicGroup, account common.Address, fromNum, toNum *big.Int, blocksLoadedCh chan []*DBHeader) error {

	log.Debug("fetchHistoryBlocks start", "chainID", c.chainClient.NetworkID(), "account", account, "omit", c.omitHistory)

	if c.omitHistory {
		blockRange := &ethTokensBlockRanges{eth: &BlockRange{nil, big.NewInt(0), toNum}, tokens: &BlockRange{nil, big.NewInt(0), toNum}}
		err := c.blockRangeDAO.upsertRange(c.chainClient.NetworkID(), account, blockRange)
		log.Error("fetchHistoryBlocks upsertRange", "error", err)
		return err
	}

	blockRange, err := loadBlockRangeInfo(c.chainClient.NetworkID(), account, c.blockRangeDAO)
	if err != nil {
		log.Error("fetchHistoryBlocks loadBlockRangeInfo", "error", err)
		// c.error = err
		return err // Will keep spinning forever nomatter what
	}

	ranges := [][]*big.Int{}

	// There are 2 history intervals:
	// 1) from 0 to FirstKnown
	// 2) from LastKnown to `toNum`` (head)
	// If we blockRange is nil, we need to load all blocks from `fromNum` to `toNum`
	// As current implementation checks ETH first then tokens, tokens ranges maybe behind ETH ranges in
	// cases when block searching was interrupted, so we use tokens ranges
	if blockRange != nil && blockRange.tokens != nil {
		if blockRange.tokens.LastKnown != nil && toNum.Cmp(blockRange.tokens.LastKnown) > 0 {
			ranges = append(ranges, []*big.Int{blockRange.tokens.LastKnown, toNum})
		}

		if blockRange.tokens.FirstKnown != nil {
			if fromNum.Cmp(blockRange.tokens.FirstKnown) < 0 {
				ranges = append(ranges, []*big.Int{fromNum, blockRange.tokens.FirstKnown})
			} else {
				if !c.transfersLoaded[account] {
					transfersLoaded, err := c.areAllTransfersLoaded(account)
					if err != nil {
						return err
					}

					if transfersLoaded {
						if c.transfersLoaded == nil {
							c.transfersLoaded = make(map[common.Address]bool)
						}
						c.transfersLoaded[account] = true
						c.notifyHistoryReady(account)
					}
				}
			}
		}
	} else {
		ranges = append(ranges, []*big.Int{fromNum, toNum})
	}

	for _, rangeItem := range ranges {
		fbc := &findBlocksCommand{
			accounts:                  []common.Address{account},
			db:                        c.db,
			accountsDB:                c.accountsDB,
			blockRangeDAO:             c.blockRangeDAO,
			chainClient:               c.chainClient,
			balanceCacher:             c.balanceCacher,
			feed:                      c.feed,
			noLimit:                   false,
			fromBlockNumber:           rangeItem[0],
			toBlockNumber:             rangeItem[1],
			transactionManager:        c.transactionManager,
			tokenManager:              c.tokenManager,
			blocksLoadedCh:            blocksLoadedCh,
			defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
		}
		group.Add(fbc.Command())
	}

	log.Debug("fetchHistoryBlocks end", "chainID", c.chainClient.NetworkID(), "account", account)

	return nil
}

func (c *loadBlocksAndTransfersCommand) startFetchingNewBlocks(ctx context.Context, addresses []common.Address, fromNum *big.Int, blocksLoadedCh chan<- []*DBHeader) {
	log.Debug("startFetchingNewBlocks start", "chainID", c.chainClient.NetworkID(), "accounts", addresses)

	verifyOnce(&fetchNewBlocksCnt, "startFetchingNewBlocks")

	go func() {
		newBlocksCmd := &findNewBlocksCommand{
			findBlocksCommand: &findBlocksCommand{
				accounts:                  addresses,
				db:                        c.db,
				accountsDB:                c.accountsDB,
				blockRangeDAO:             c.blockRangeDAO,
				chainClient:               c.chainClient,
				balanceCacher:             c.balanceCacher,
				feed:                      c.feed,
				noLimit:                   false,
				fromBlockNumber:           fromNum,
				transactionManager:        c.transactionManager,
				tokenManager:              c.tokenManager,
				blocksLoadedCh:            blocksLoadedCh,
				defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
			},
		}
		group := async.NewGroup(ctx)
		group.Add(newBlocksCmd.Command())

		// No need to wait for the group since it is infinite
		<-ctx.Done()
	}()

	log.Debug("startFetchingNewBlocks end", "chainID", c.chainClient.NetworkID(), "accounts", addresses, "error", ctx.Err())
}

func (c *loadBlocksAndTransfersCommand) getBlocksToLoad() (map[common.Address][]*big.Int, error) {
	blocksMap := make(map[common.Address][]*big.Int)
	for _, account := range c.accounts {
		blocks, err := c.blockDAO.GetBlocksToLoadByAddress(c.chainClient.NetworkID(), account, numberOfBlocksCheckedPerIteration)
		if err != nil {
			log.Error("loadBlocksAndTransfersCommand GetBlocksToLoadByAddress", "error", err)
			return nil, err
		}

		if len(blocks) == 0 {
			log.Debug("fetchTransfers no blocks to load", "chainID", c.chainClient.NetworkID(), "account", account)
			continue
		}

		blocksMap[account] = blocks
	}

	if len(blocksMap) == 0 {
		log.Debug("fetchTransfers no blocks to load", "chainID", c.chainClient.NetworkID())
	}

	return blocksMap, nil
}

func (c *loadBlocksAndTransfersCommand) startFetchingTransfersForLoadedBlocks(group *async.AtomicGroup) error {

	log.Debug("fetchTransfers start", "chainID", c.chainClient.NetworkID(), "accounts", c.accounts)

	blocksMap, err := c.getBlocksToLoad()
	if err != nil {
		return err
	}

	go func() {
		txCommand := &loadTransfersCommand{
			accounts:           c.accounts,
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
		log.Debug("fetchTransfers end", "chainID", c.chainClient.NetworkID(), "accounts", c.accounts)
	}()

	return nil
}

func (c *loadBlocksAndTransfersCommand) notifyHistoryReady(account common.Address) {
	if c.feed != nil {
		c.feed.Send(walletevent.Event{
			Type:     EventRecentHistoryReady,
			Accounts: []common.Address{account},
			ChainID:  c.chainClient.NetworkID(),
		})
	}
}

func (c *loadBlocksAndTransfersCommand) areAllTransfersLoaded(account common.Address) (bool, error) {
	allBlocksLoaded, err := areAllHistoryBlocksLoadedForAddress(c.blockRangeDAO, c.chainClient.NetworkID(), account)
	if err != nil {
		log.Error("loadBlockAndTransfersCommand allHistoryBlocksLoaded", "error", err)
		return false, err
	}

	if allBlocksLoaded {
		headers, err := c.blockDAO.GetBlocksToLoadByAddress(c.chainClient.NetworkID(), account, 1)
		if err != nil {
			log.Error("loadBlocksAndTransfersCommand GetFirstSavedBlock", "error", err)
			return false, err
		}

		if len(headers) == 0 {
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
		log.Error("getHeadBlockNumber", "error", err)
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
