package transfer

import (
	"context"
	"database/sql"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	// EventNewTransfers emitted when new block was added to the same canonical chan.
	EventNewTransfers walletevent.EventType = "new-transfers"
	// EventFetchingRecentHistory emitted when fetching of lastest tx history is started
	EventFetchingRecentHistory walletevent.EventType = "recent-history-fetching"
	// EventRecentHistoryReady emitted when fetching of lastest tx history is started
	EventRecentHistoryReady walletevent.EventType = "recent-history-ready"
	// EventFetchingHistoryError emitted when fetching of tx history failed
	EventFetchingHistoryError walletevent.EventType = "fetching-history-error"
	// EventNonArchivalNodeDetected emitted when a connection to a non archival node is detected
	EventNonArchivalNodeDetected walletevent.EventType = "non-archival-node-detected"
)

var (
	// This will work only for binance testnet as mainnet doesn't support
	// archival request.
	binanceChainMaxInitialRange       = big.NewInt(500000)
	binanceChainErc20BatchSize        = big.NewInt(5000)
	goerliErc20BatchSize              = big.NewInt(100000)
	goerliErc20ArbitrumBatchSize      = big.NewInt(100000)
	erc20BatchSize                    = big.NewInt(500000)
	binancChainID                     = uint64(56)
	goerliChainID                     = uint64(5)
	goerliArbitrumChainID             = uint64(421613)
	binanceTestChainID                = uint64(97)
	numberOfBlocksCheckedPerIteration = 40
)

type ethHistoricalCommand struct {
	eth          Downloader
	address      common.Address
	chainClient  *chain.ClientWithFallback
	balanceCache *balanceCache
	feed         *event.Feed
	foundHeaders []*DBHeader
	error        error
	noLimit      bool

	from              *Block
	to, resultingFrom *big.Int
}

func (c *ethHistoricalCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *ethHistoricalCommand) Run(ctx context.Context) (err error) {
	log.Info("eth historical downloader start", "address", c.address, "from", c.from.Number, "to", c.to, "noLimit", c.noLimit)

	start := time.Now()
	if c.from.Number != nil && c.from.Balance != nil {
		c.balanceCache.addBalanceToCache(c.address, c.from.Number, c.from.Balance)
	}
	if c.from.Number != nil && c.from.Nonce != nil {
		c.balanceCache.addNonceToCache(c.address, c.from.Number, c.from.Nonce)
	}
	from, headers, err := findBlocksWithEthTransfers(ctx, c.chainClient, c.balanceCache, c.eth, c.address, c.from.Number, c.to, c.noLimit)

	if err != nil {
		c.error = err
		return nil
	}

	c.foundHeaders = headers
	c.resultingFrom = from

	log.Info("eth historical downloader finished successfully", "address", c.address, "from", from, "to", c.to, "total blocks", len(headers), "time", time.Since(start))

	return nil
}

type erc20HistoricalCommand struct {
	erc20       BatchDownloader
	address     common.Address
	chainClient *chain.ClientWithFallback
	feed        *event.Feed

	iterator     *IterativeDownloader
	to           *big.Int
	from         *big.Int
	foundHeaders []*DBHeader
}

func (c *erc20HistoricalCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func getErc20BatchSize(chainID uint64) *big.Int {
	if isBinanceChain(chainID) {
		return binanceChainErc20BatchSize
	}

	if chainID == goerliChainID {
		return goerliErc20BatchSize
	}

	if chainID == goerliArbitrumChainID {
		return goerliErc20ArbitrumBatchSize
	}

	return erc20BatchSize
}

func (c *erc20HistoricalCommand) Run(ctx context.Context) (err error) {
	log.Info("wallet historical downloader for erc20 transfers start", "address", c.address,
		"from", c.from, "to", c.to)

	start := time.Now()
	if c.iterator == nil {
		c.iterator, err = SetupIterativeDownloader(
			c.chainClient, c.address,
			c.erc20, getErc20BatchSize(c.chainClient.ChainID), c.to, c.from)
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
	}
	log.Info("wallet historical downloader for erc20 transfers finished", "address", c.address,
		"from", c.from, "to", c.to, "time", time.Since(start))
	return nil
}

// controlCommand implements following procedure (following parts are executed sequeantially):
// - verifies that the last header that was synced is still in the canonical chain
// - runs fast indexing for each account separately
// - starts listening to new blocks and watches for reorgs
type controlCommand struct {
	accounts           []common.Address
	db                 *Database
	blockDAO           *BlockDAO
	eth                *ETHDownloader
	erc20              *ERC20TransfersDownloader
	chainClient        *chain.ClientWithFallback
	feed               *event.Feed
	errorsCount        int
	nonArchivalRPCNode bool
	transactionManager *TransactionManager
}

func (c *controlCommand) LoadTransfers(ctx context.Context, limit int) (map[common.Address][]Transfer, error) {
	return loadTransfers(ctx, c.accounts, c.blockDAO, c.db, c.chainClient, limit, make(map[common.Address][]*big.Int), c.transactionManager)
}

func (c *controlCommand) Run(parent context.Context) error {
	log.Info("start control command")
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	head, err := c.chainClient.HeaderByNumber(ctx, nil)
	cancel()
	if err != nil {
		if c.NewError(err) {
			return nil
		}
		return err
	}

	if c.feed != nil {
		c.feed.Send(walletevent.Event{
			Type:     EventFetchingRecentHistory,
			Accounts: c.accounts,
		})
	}

	log.Info("current head is", "block number", head.Number)

	// Get last known block for each account
	lastKnownEthBlocks, accountsWithoutHistory, err := c.blockDAO.GetLastKnownBlockByAddresses(c.chainClient.ChainID, c.accounts)
	if err != nil {
		log.Error("failed to load last head from database", "error", err)
		if c.NewError(err) {
			return nil
		}
		return err
	}

	// For accounts without history, find the block where 20 < headNonce - nonce < 25 (blocks have between 20-25 transactions)
	fromMap := map[common.Address]*big.Int{}

	if !c.nonArchivalRPCNode {
		fromMap, err = findFirstRanges(parent, accountsWithoutHistory, head.Number, c.chainClient)
		if err != nil {
			if c.NewError(err) {
				return nil
			}
			return err
		}
	}

	// Set "fromByAddress" from the information we have
	target := head.Number
	fromByAddress := map[common.Address]*Block{}
	toByAddress := map[common.Address]*big.Int{}

	for _, address := range c.accounts {
		from, ok := lastKnownEthBlocks[address]
		if !ok {
			from = &Block{Number: fromMap[address]}
		}
		if c.nonArchivalRPCNode {
			from = &Block{Number: big.NewInt(0).Sub(target, big.NewInt(100))}
		}

		fromByAddress[address] = from
		toByAddress[address] = target
	}

	bCache := newBalanceCache()
	cmnd := &findAndCheckBlockRangeCommand{
		accounts:      c.accounts,
		db:            c.db,
		chainClient:   c.chainClient,
		balanceCache:  bCache,
		feed:          c.feed,
		fromByAddress: fromByAddress,
		toByAddress:   toByAddress,
	}

	err = cmnd.Command()(parent)
	if err != nil {
		if c.NewError(err) {
			return nil
		}
		return err
	}

	if cmnd.error != nil {
		if c.NewError(cmnd.error) {
			return nil
		}
		return cmnd.error
	}

	_, err = c.LoadTransfers(parent, 40)
	if err != nil {
		if c.NewError(err) {
			return nil
		}
		return err
	}

	if c.feed != nil {
		events := map[common.Address]walletevent.Event{}
		for _, address := range c.accounts {
			event := walletevent.Event{
				Type:     EventNewTransfers,
				Accounts: []common.Address{address},
			}
			for _, header := range cmnd.foundHeaders[address] {
				if event.BlockNumber == nil || header.Number.Cmp(event.BlockNumber) == 1 {
					event.BlockNumber = header.Number
				}
			}
			if event.BlockNumber != nil {
				events[address] = event
			}
		}

		for _, event := range events {
			c.feed.Send(event)
		}

		c.feed.Send(walletevent.Event{
			Type:        EventRecentHistoryReady,
			Accounts:    c.accounts,
			BlockNumber: target,
		})
	}

	log.Info("end control command")
	return err
}

func nonArchivalNodeError(err error) bool {
	return strings.Contains(err.Error(), "missing trie node") ||
		strings.Contains(err.Error(), "project ID does not have access to archive state")
}

func (c *controlCommand) NewError(err error) bool {
	c.errorsCount++
	log.Error("controlCommand error", "error", err, "counter", c.errorsCount)
	if nonArchivalNodeError(err) {
		log.Info("Non archival node detected")
		c.nonArchivalRPCNode = true
		c.feed.Send(walletevent.Event{
			Type: EventNonArchivalNodeDetected,
		})
	}
	if c.errorsCount >= 3 {
		c.feed.Send(walletevent.Event{
			Type:    EventFetchingHistoryError,
			Message: err.Error(),
		})
		return true
	}
	return false
}

func (c *controlCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

type transfersCommand struct {
	db                 *Database
	eth                *ETHDownloader
	block              *big.Int
	address            common.Address
	chainClient        *chain.ClientWithFallback
	fetchedTransfers   []Transfer
	transactionManager *TransactionManager
}

func (c *transfersCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *transfersCommand) Run(ctx context.Context) (err error) {
	startTs := time.Now()

	allTransfers, err := getTransfersByBlocks(ctx, c.db, c.eth, []*big.Int{c.block})
	if err != nil {
		log.Info("getTransfersByBlocks error", "error", err)
		return err
	}

	// Update MultiTransactionID from pending entry
	for index := range allTransfers {
		transfer := &allTransfers[index]
		if transfer.MultiTransactionID == NoMultiTransactionID {
			entry, err := c.transactionManager.GetPendingEntry(c.chainClient.ChainID, transfer.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Info("Pending transaction not found for", "chainID", c.chainClient.ChainID, "transferID", transfer.ID)
				} else {
					return err
				}
			} else {
				transfer.MultiTransactionID = entry.MultiTransactionID
				if transfer.Receipt != nil && transfer.Receipt.Status == types.ReceiptStatusSuccessful {
					// TODO: Nim logic was deleting pending previously, should we notify UI about it?
					err := c.transactionManager.DeletePending(c.chainClient.ChainID, transfer.ID)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	if len(allTransfers) > 0 {
		err = c.db.SaveTransfersMarkBlocksLoaded(c.chainClient.ChainID, c.address, allTransfers, []*big.Int{c.block})
		if err != nil {
			log.Error("SaveTransfers error", "error", err)
			return err
		}
	}

	c.fetchedTransfers = allTransfers
	log.Debug("transfers loaded", "address", c.address, "len", len(allTransfers), "in", time.Since(startTs))
	return nil
}

type loadTransfersCommand struct {
	accounts                []common.Address
	db                      *Database
	blockDAO                *BlockDAO
	chainClient             *chain.ClientWithFallback
	blocksByAddress         map[common.Address][]*big.Int
	foundTransfersByAddress map[common.Address][]Transfer
	transactionManager      *TransactionManager
}

func (c *loadTransfersCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *loadTransfersCommand) LoadTransfers(ctx context.Context, limit int, blocksByAddress map[common.Address][]*big.Int, transactionManager *TransactionManager) (map[common.Address][]Transfer, error) {
	return loadTransfers(ctx, c.accounts, c.blockDAO, c.db, c.chainClient, limit, blocksByAddress, c.transactionManager)
}

func (c *loadTransfersCommand) Run(parent context.Context) (err error) {
	transfersByAddress, err := c.LoadTransfers(parent, 40, c.blocksByAddress, c.transactionManager)
	if err != nil {
		return err
	}
	c.foundTransfersByAddress = transfersByAddress

	return
}

type findAndCheckBlockRangeCommand struct {
	accounts      []common.Address
	db            *Database
	chainClient   *chain.ClientWithFallback
	balanceCache  *balanceCache
	feed          *event.Feed
	fromByAddress map[common.Address]*Block
	toByAddress   map[common.Address]*big.Int
	foundHeaders  map[common.Address][]*DBHeader
	noLimit       bool
	error         error
}

func (c *findAndCheckBlockRangeCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *findAndCheckBlockRangeCommand) Run(parent context.Context) (err error) {
	log.Debug("start findAndCHeckBlockRangeCommand")

	newFromByAddress, ethHeadersByAddress, err := c.fastIndex(parent, c.balanceCache, c.fromByAddress, c.toByAddress)
	if err != nil {
		c.error = err
		// return err // In case c.noLimit is true, hystrix "max concurrency" may be reached and we will not be able to index ETH transfers. But if we return error, we will get stuck in inifinite loop.
		return nil
	}
	if c.noLimit {
		newFromByAddress = map[common.Address]*big.Int{}
		for _, address := range c.accounts {
			newFromByAddress[address] = c.fromByAddress[address].Number
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

		foundHeaders[address] = uniqHeaders

		lastBlockNumber := c.toByAddress[address]
		log.Debug("saving headers", "len", len(uniqHeaders), "lastBlockNumber", lastBlockNumber, "balance", c.balanceCache.ReadCachedBalance(address, lastBlockNumber), "nonce", c.balanceCache.ReadCachedNonce(address, lastBlockNumber))
		to := &Block{
			Number:  lastBlockNumber,
			Balance: c.balanceCache.ReadCachedBalance(address, lastBlockNumber),
			Nonce:   c.balanceCache.ReadCachedNonce(address, lastBlockNumber),
		}
		err = c.db.ProcessBlocks(c.chainClient.ChainID, address, newFromByAddress[address], to, uniqHeaders)
		if err != nil {
			return err
		}
	}

	c.foundHeaders = foundHeaders

	log.Debug("end findAndCheckBlockRangeCommand")
	return
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findAndCheckBlockRangeCommand) fastIndex(ctx context.Context, bCache *balanceCache,
	fromByAddress map[common.Address]*Block, toByAddress map[common.Address]*big.Int) (map[common.Address]*big.Int,
	map[common.Address][]*DBHeader, error) {

	log.Info("fast indexer started")

	start := time.Now()
	group := async.NewGroup(ctx)

	commands := make([]*ethHistoricalCommand, len(c.accounts))
	for i, address := range c.accounts {
		eth := &ethHistoricalCommand{
			chainClient:  c.chainClient,
			balanceCache: bCache,
			address:      address,
			eth: &ETHDownloader{
				chainClient: c.chainClient,
				accounts:    []common.Address{address},
				signer:      types.NewLondonSigner(c.chainClient.ToBigInt()),
				db:          c.db,
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
			if command.error != nil {
				return nil, nil, command.error
			}
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
	log.Info("fast indexer Erc20 started")

	start := time.Now()
	group := async.NewGroup(ctx)

	commands := make([]*erc20HistoricalCommand, len(c.accounts))
	for i, address := range c.accounts {
		erc20 := &erc20HistoricalCommand{
			erc20:        NewERC20TransfersDownloader(c.chainClient, []common.Address{address}, types.NewLondonSigner(c.chainClient.ToBigInt())),
			chainClient:  c.chainClient,
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

func loadTransfers(ctx context.Context, accounts []common.Address, blockDAO *BlockDAO, db *Database,
	chainClient *chain.ClientWithFallback, limit int, blocksByAddress map[common.Address][]*big.Int,
	transactionManager *TransactionManager) (map[common.Address][]Transfer, error) {

	log.Info("loadTransfers start", "accounts", accounts, "limit", limit)

	start := time.Now()
	group := async.NewGroup(ctx)

	commands := []*transfersCommand{}
	for _, address := range accounts {
		blocks, ok := blocksByAddress[address]

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
				block:              block,
				transactionManager: transactionManager,
			}
			commands = append(commands, transfers)
			group.Add(transfers.Command())
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

func isBinanceChain(chainID uint64) bool {
	return chainID == binancChainID || chainID == binanceTestChainID
}

func getLowestFrom(chainID uint64, to *big.Int) *big.Int {
	from := big.NewInt(0)
	if isBinanceChain(chainID) && big.NewInt(0).Sub(to, from).Cmp(binanceChainMaxInitialRange) == 1 {
		from = big.NewInt(0).Sub(to, binanceChainMaxInitialRange)
	}

	return from
}

// Finds the latest range up to initialTo where the number of transactions is between 20 and 25
func findFirstRange(c context.Context, account common.Address, initialTo *big.Int, client *chain.ClientWithFallback) (*big.Int, error) {
	log.Info("findFirstRange", "account", account, "initialTo", initialTo, "client", client)

	from := getLowestFrom(client.ChainID, initialTo)
	to := initialTo
	goal := uint64(20)

	if from.Cmp(to) == 0 {
		return to, nil
	}

	firstNonce, err := client.NonceAt(c, account, to) // this is the latest nonce actually
	log.Info("find range with 20 <= len(tx) <= 25", "account", account, "firstNonce", firstNonce, "from", from, "to", to)

	if err != nil {
		return nil, err
	}

	if firstNonce <= goal {
		return from, nil
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
			// from = from - (to - from) / 2
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

// Finds the latest ranges up to initialTo where the number of transactions is between 20 and 25
func findFirstRanges(c context.Context, accounts []common.Address, initialTo *big.Int, client *chain.ClientWithFallback) (map[common.Address]*big.Int, error) {
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

func getTransfersByBlocks(ctx context.Context, db *Database, downloader *ETHDownloader, blocks []*big.Int) ([]Transfer, error) {
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
