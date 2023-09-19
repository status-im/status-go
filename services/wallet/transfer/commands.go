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
	"github.com/status-im/status-go/services/wallet/balance"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"
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

	numberOfBlocksCheckedPerIteration = 40
	noBlockLimit                      = 0
)

var (
	// This will work only for binance testnet as mainnet doesn't support
	// archival request.
	binanceChainMaxInitialRange  = big.NewInt(500000)
	binanceChainErc20BatchSize   = big.NewInt(5000)
	goerliErc20BatchSize         = big.NewInt(100000)
	goerliErc20ArbitrumBatchSize = big.NewInt(10000)
	goerliErc20OptimismBatchSize = big.NewInt(10000)
	erc20BatchSize               = big.NewInt(500000)
	binancChainID                = uint64(56)
	goerliChainID                = uint64(5)
	goerliArbitrumChainID        = uint64(421613)
	goerliOptimismChainID        = uint64(420)
	binanceTestChainID           = uint64(97)
)

type ethHistoricalCommand struct {
	address       common.Address
	chainClient   chain.ClientInterface
	balanceCacher balance.Cacher
	feed          *event.Feed
	foundHeaders  []*DBHeader
	error         error
	noLimit       bool

	from                          *Block
	to, resultingFrom, startBlock *big.Int
	threadLimit                   uint32
}

type Transaction []*Transfer

func (c *ethHistoricalCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *ethHistoricalCommand) Run(ctx context.Context) (err error) {
	log.Info("eth historical downloader start", "chainID", c.chainClient.NetworkID(), "address", c.address,
		"from", c.from.Number, "to", c.to, "noLimit", c.noLimit)

	start := time.Now()
	if c.from.Number != nil && c.from.Balance != nil {
		c.balanceCacher.Cache().AddBalance(c.address, c.chainClient.NetworkID(), c.from.Number, c.from.Balance)
	}
	if c.from.Number != nil && c.from.Nonce != nil {
		c.balanceCacher.Cache().AddNonce(c.address, c.chainClient.NetworkID(), c.from.Number, c.from.Nonce)
	}
	from, headers, startBlock, err := findBlocksWithEthTransfers(ctx, c.chainClient,
		c.balanceCacher, c.address, c.from.Number, c.to, c.noLimit, c.threadLimit)

	if err != nil {
		c.error = err
		log.Error("failed to find blocks with transfers", "error", err, "chainID", c.chainClient.NetworkID(),
			"address", c.address, "from", c.from.Number, "to", c.to)
		return nil
	}

	c.foundHeaders = headers
	c.resultingFrom = from
	c.startBlock = startBlock

	log.Info("eth historical downloader finished successfully", "chain", c.chainClient.NetworkID(),
		"address", c.address, "from", from, "to", c.to, "total blocks", len(headers), "time", time.Since(start))

	return nil
}

type erc20HistoricalCommand struct {
	erc20       BatchDownloader
	address     common.Address
	chainClient chain.ClientInterface
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

	if chainID == goerliOptimismChainID {
		return goerliErc20OptimismBatchSize
	}

	if chainID == goerliArbitrumChainID {
		return goerliErc20ArbitrumBatchSize
	}

	return erc20BatchSize
}

func (c *erc20HistoricalCommand) Run(ctx context.Context) (err error) {
	log.Info("wallet historical downloader for erc20 transfers start", "chainID", c.chainClient.NetworkID(), "address", c.address,
		"from", c.from, "to", c.to)

	start := time.Now()
	if c.iterator == nil {
		c.iterator, err = SetupIterativeDownloader(
			c.chainClient, c.address,
			c.erc20, getErc20BatchSize(c.chainClient.NetworkID()), c.to, c.from)
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
	log.Info("wallet historical downloader for erc20 transfers finished", "chainID", c.chainClient.NetworkID(), "address", c.address,
		"from", c.from, "to", c.to, "time", time.Since(start), "headers", len(c.foundHeaders))
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
	pendingTxManager   *transactions.PendingTxTracker
	tokenManager       *token.Manager
	balanceCacher      balance.Cacher
}

func (c *controlCommand) LoadTransfers(ctx context.Context, limit int) error {
	return loadTransfers(ctx, c.accounts, c.blockDAO, c.db, c.chainClient, limit, make(map[common.Address][]*big.Int),
		c.transactionManager, c.pendingTxManager, c.tokenManager, c.feed)
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
	lastKnownEthBlocks, accountsWithoutHistory, err := c.blockDAO.GetLastKnownBlockByAddresses(c.chainClient.NetworkID(), c.accounts)
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

	cmnd := &findAndCheckBlockRangeCommand{
		accounts:      c.accounts,
		db:            c.db,
		blockDAO:      c.blockDAO,
		chainClient:   c.chainClient,
		balanceCacher: c.balanceCacher,
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

	c.balanceCacher.Clear()
	err = c.LoadTransfers(parent, numberOfBlocksCheckedPerIteration)
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
				ChainID:  c.chainClient.NetworkID(),
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
	log.Error("controlCommand error", "chainID", c.chainClient.NetworkID(), "error", err, "counter", c.errorsCount)
	if nonArchivalNodeError(err) {
		log.Info("Non archival node detected", "chainID", c.chainClient.NetworkID())
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
	blockDAO           *BlockDAO
	eth                *ETHDownloader
	blockNums          []*big.Int
	address            common.Address
	chainClient        *chain.ClientWithFallback
	blocksLimit        int
	transactionManager *TransactionManager
	pendingTxManager   *transactions.PendingTxTracker
	tokenManager       *token.Manager
	feed               *event.Feed

	// result
	fetchedTransfers []Transfer
}

func (c *transfersCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *transfersCommand) Run(ctx context.Context) (err error) {
	// Take blocks from cache if available and disrespect the limit
	// If no blocks are available in cache, take blocks from DB respecting the limit
	// If no limit is set, take all blocks from DB
	log.Info("start transfersCommand", "chain", c.chainClient.NetworkID(), "address", c.address, "blockNums", c.blockNums)
	startTs := time.Now()

	for {
		blocks := c.blockNums
		if blocks == nil {
			blocks, _ = c.blockDAO.GetBlocksToLoadByAddress(c.chainClient.NetworkID(), c.address, numberOfBlocksCheckedPerIteration)
		}

		for _, blockNum := range blocks {
			log.Debug("transfersCommand block start", "chain", c.chainClient.NetworkID(), "address", c.address, "block", blockNum)

			allTransfers, err := c.eth.GetTransfersByNumber(ctx, blockNum)
			if err != nil {
				log.Error("getTransfersByBlocks error", "error", err)
				return err
			}

			c.processUnknownErc20CommunityTransactions(ctx, allTransfers)

			err = c.processMultiTransactions(ctx, allTransfers)
			if err != nil {
				log.Error("processMultiTransactions error", "error", err)
				return err
			}

			if len(allTransfers) > 0 {
				err := c.saveAndConfirmPending(allTransfers, blockNum)
				if err != nil {
					log.Error("saveAndConfirmPending error", "error", err)
					return err
				}
			} else {
				// If no transfers found, that is suspecting, because downloader returned this block as containing transfers
				log.Error("no transfers found in block", "chain", c.chainClient.NetworkID(), "address", c.address, "block", blockNum)

				err = markBlocksAsLoaded(c.chainClient.NetworkID(), c.db.client, c.address, []*big.Int{blockNum})
				if err != nil {
					log.Error("Mark blocks loaded error", "error", err)
					return err
				}
			}

			c.fetchedTransfers = append(c.fetchedTransfers, allTransfers...)

			c.notifyOfNewTransfers(allTransfers)

			log.Debug("transfersCommand block end", "chain", c.chainClient.NetworkID(), "address", c.address,
				"block", blockNum, "tranfers.len", len(allTransfers), "fetchedTransfers.len", len(c.fetchedTransfers))
		}

		if c.blockNums != nil || len(blocks) == 0 ||
			(c.blocksLimit > noBlockLimit && len(blocks) >= c.blocksLimit) {
			log.Debug("loadTransfers breaking loop on block limits reached or 0 blocks", "chain", c.chainClient.NetworkID(),
				"address", c.address, "limit", c.blocksLimit, "blocks", len(blocks))
			break
		}
	}

	log.Info("end transfersCommand", "chain", c.chainClient.NetworkID(), "address", c.address,
		"blocks.len", len(c.blockNums), "transfers.len", len(c.fetchedTransfers), "in", time.Since(startTs))

	return nil
}

// saveAndConfirmPending ensures only the transaction that has owner (Address) as a sender is matched to the
// corresponding multi-transaction (by multi-transaction ID). This way we ensure that if receiver is in the list
// of accounts filter will discard the proper one
func (c *transfersCommand) saveAndConfirmPending(allTransfers []Transfer, blockNum *big.Int) error {
	tx, resErr := c.db.client.Begin()
	if resErr != nil {
		return resErr
	}
	notifyFunctions := make([]func(), 0)
	defer func() {
		if resErr == nil {
			commitErr := tx.Commit()
			if commitErr != nil {
				log.Error("failed to commit", "error", commitErr)
			}
			for _, notify := range notifyFunctions {
				notify()
			}
		} else {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Error("failed to rollback", "error", rollbackErr)
			}
		}
	}()

	// Confirm all pending transactions that are included in this block
	for i, tr := range allTransfers {
		chainID := w_common.ChainID(tr.NetworkID)
		txHash := tr.Receipt.TxHash
		txType, mTID, err := transactions.GetOwnedPendingStatus(tx, chainID, txHash, tr.Address)
		if err == sql.ErrNoRows {
			if tr.MultiTransactionID > 0 {
				continue
			} else {
				// Outside transaction, already confirmed by another duplicate or not yet downloaded
				existingMTID, err := GetOwnedMultiTransactionID(tx, chainID, tr.ID, tr.Address)
				if err == sql.ErrNoRows || existingMTID == 0 {
					// Outside transaction, ignore it
					continue
				} else if err != nil {
					log.Warn("GetOwnedMultiTransactionID", "error", err)
					continue
				}
				mTID = w_common.NewAndSet(existingMTID)

			}
		} else if err != nil {
			log.Warn("GetOwnedPendingStatus", "error", err)
			continue
		}

		if mTID != nil {
			allTransfers[i].MultiTransactionID = MultiTransactionIDType(*mTID)
		}
		if txType != nil && *txType == transactions.WalletTransfer {
			notify, err := c.pendingTxManager.DeleteBySQLTx(tx, chainID, txHash)
			if err != nil && err != transactions.ErrStillPending {
				log.Error("DeleteBySqlTx error", "error", err)
			}
			notifyFunctions = append(notifyFunctions, notify)
		}
	}

	resErr = saveTransfersMarkBlocksLoaded(tx, c.chainClient.NetworkID(), c.address, allTransfers, []*big.Int{blockNum})
	if resErr != nil {
		log.Error("SaveTransfers error", "error", resErr)
	}

	return resErr
}

// Mark all subTxs of a given Tx with the same multiTxID
func setMultiTxID(tx Transaction, multiTxID MultiTransactionIDType) {
	for _, subTx := range tx {
		subTx.MultiTransactionID = multiTxID
	}
}

func (c *transfersCommand) checkAndProcessSwapMultiTx(ctx context.Context, tx Transaction) (bool, error) {
	for _, subTx := range tx {
		switch subTx.Type {
		// If the Tx contains any uniswapV2Swap/uniswapV3Swap subTx, generate a Swap multiTx
		case w_common.UniswapV2Swap, w_common.UniswapV3Swap:
			multiTransaction, err := buildUniswapSwapMultitransaction(ctx, c.chainClient, c.tokenManager, subTx)
			if err != nil {
				return false, err
			}

			if multiTransaction != nil {
				id, err := c.transactionManager.InsertMultiTransaction(multiTransaction)
				if err != nil {
					return false, err
				}
				setMultiTxID(tx, id)
				return true, nil
			}
		}
	}

	return false, nil
}

func (c *transfersCommand) checkAndProcessBridgeMultiTx(ctx context.Context, tx Transaction) (bool, error) {
	for _, subTx := range tx {
		switch subTx.Type {
		// If the Tx contains any hopBridge subTx, create/update Bridge multiTx
		case w_common.HopBridgeFrom, w_common.HopBridgeTo:
			multiTransaction, err := buildHopBridgeMultitransaction(ctx, c.chainClient, c.transactionManager, c.tokenManager, subTx)
			if err != nil {
				return false, err
			}

			if multiTransaction != nil {
				setMultiTxID(tx, MultiTransactionIDType(multiTransaction.ID))
				return true, nil
			}
		}
	}

	return false, nil
}

func (c *transfersCommand) processUnknownErc20CommunityTransactions(ctx context.Context, allTransfers []Transfer) {
	for _, tx := range allTransfers {
		if tx.Type == w_common.Erc20Transfer {
			// Find token in db or if this is a community token, find its metadata
			_ = c.tokenManager.FindOrCreateTokenByAddress(ctx, tx.NetworkID, *tx.Transaction.To())
		}
	}
}

func (c *transfersCommand) processMultiTransactions(ctx context.Context, allTransfers []Transfer) error {
	txByTxHash := subTransactionListToTransactionsByTxHash(allTransfers)

	// Detect / Generate multitransactions
	// Iterate over all detected transactions
	for _, tx := range txByTxHash {
		// Then check for a Swap transaction
		txProcessed, err := c.checkAndProcessSwapMultiTx(ctx, tx)
		if err != nil {
			return err
		}
		if txProcessed {
			continue
		}

		// Then check for a Bridge transaction
		_, err = c.checkAndProcessBridgeMultiTx(ctx, tx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *transfersCommand) notifyOfNewTransfers(transfers []Transfer) {
	if c.feed != nil {
		if len(transfers) > 0 {
			c.feed.Send(walletevent.Event{
				Type:     EventNewTransfers,
				Accounts: []common.Address{c.address},
				ChainID:  c.chainClient.NetworkID(),
			})
		}
	}
}

type loadTransfersCommand struct {
	accounts           []common.Address
	db                 *Database
	blockDAO           *BlockDAO
	chainClient        *chain.ClientWithFallback
	blocksByAddress    map[common.Address][]*big.Int
	transactionManager *TransactionManager
	pendingTxManager   *transactions.PendingTxTracker
	blocksLimit        int
	tokenManager       *token.Manager
	feed               *event.Feed
}

func (c *loadTransfersCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

func (c *loadTransfersCommand) LoadTransfers(ctx context.Context, limit int, blocksByAddress map[common.Address][]*big.Int) error {
	return loadTransfers(ctx, c.accounts, c.blockDAO, c.db, c.chainClient, limit, blocksByAddress,
		c.transactionManager, c.pendingTxManager, c.tokenManager, c.feed)
}

func (c *loadTransfersCommand) Run(parent context.Context) (err error) {
	err = c.LoadTransfers(parent, c.blocksLimit, c.blocksByAddress)
	return
}

type findAndCheckBlockRangeCommand struct {
	accounts      []common.Address
	db            *Database
	blockDAO      *BlockDAO
	chainClient   *chain.ClientWithFallback
	balanceCacher balance.Cacher
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

func (c *findAndCheckBlockRangeCommand) Run(parent context.Context) error {
	log.Debug("start findAndCHeckBlockRangeCommand")

	newFromByAddress, ethHeadersByAddress, err := c.fastIndex(parent, c.balanceCacher, c.fromByAddress, c.toByAddress)
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

		log.Debug("allHeaders found for account", "address", address, "allHeaders.len", len(allHeaders))

		// Ensure only 1 DBHeader per block hash.
		uniqHeaders := []*DBHeader{}
		if len(allHeaders) > 0 {
			uniqHeaders = uniqueHeaderPerBlockHash(allHeaders)
		}

		// Ensure only 1 PreloadedTransaction per transaction hash during block discovery.
		// Full list of SubTransactions will be obtained from the receipt logs
		// at a later stage.
		for _, header := range uniqHeaders {
			header.PreloadedTransactions = uniquePreloadedTransactionPerTxHash(header.PreloadedTransactions)
		}

		foundHeaders[address] = uniqHeaders

		lastBlockNumber := c.toByAddress[address]
		log.Debug("saving headers", "len", len(uniqHeaders), "lastBlockNumber", lastBlockNumber,
			"balance", c.balanceCacher.Cache().GetBalance(address, c.chainClient.NetworkID(), lastBlockNumber),
			"nonce", c.balanceCacher.Cache().GetNonce(address, c.chainClient.NetworkID(), lastBlockNumber))

		to := &Block{
			Number:  lastBlockNumber,
			Balance: c.balanceCacher.Cache().GetBalance(address, c.chainClient.NetworkID(), lastBlockNumber),
			Nonce:   c.balanceCacher.Cache().GetNonce(address, c.chainClient.NetworkID(), lastBlockNumber),
		}
		log.Debug("uniqHeaders found for account", "address", address, "uniqHeaders.len", len(uniqHeaders))
		err = c.db.ProcessBlocks(c.chainClient.NetworkID(), address, newFromByAddress[address], to, uniqHeaders)
		if err != nil {
			return err
		}
	}

	c.foundHeaders = foundHeaders

	log.Debug("end findAndCheckBlockRangeCommand")
	return nil
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findAndCheckBlockRangeCommand) fastIndex(ctx context.Context, bCacher balance.Cacher,
	fromByAddress map[common.Address]*Block, toByAddress map[common.Address]*big.Int) (map[common.Address]*big.Int,
	map[common.Address][]*DBHeader, error) {

	log.Info("fast indexer started")

	start := time.Now()
	group := async.NewGroup(ctx)

	commands := make([]*ethHistoricalCommand, len(c.accounts))
	for i, address := range c.accounts {
		eth := &ethHistoricalCommand{
			chainClient:   c.chainClient,
			balanceCacher: bCacher,
			address:       address,
			feed:          c.feed,
			from:          fromByAddress[address],
			to:            toByAddress[address],
			noLimit:       c.noLimit,
			threadLimit:   NoThreadLimit,
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
			erc20:        NewERC20TransfersDownloader(c.chainClient, []common.Address{address}, types.LatestSignerForChainID(c.chainClient.ToBigInt())),
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
		headers := map[common.Address][]*DBHeader{}
		for _, command := range commands {
			headers[command.address] = command.foundHeaders
		}
		log.Info("fast indexer Erc20 finished", "in", time.Since(start))
		return headers, nil
	}
}

func loadTransfers(ctx context.Context, accounts []common.Address, blockDAO *BlockDAO, db *Database,
	chainClient *chain.ClientWithFallback, blocksLimitPerAccount int, blocksByAddress map[common.Address][]*big.Int,
	transactionManager *TransactionManager, pendingTxManager *transactions.PendingTxTracker,
	tokenManager *token.Manager, feed *event.Feed) error {

	log.Info("loadTransfers start", "accounts", accounts, "chain", chainClient.ChainID, "limit", blocksLimitPerAccount)

	start := time.Now()
	group := async.NewGroup(ctx)

	for _, address := range accounts {
		transfers := &transfersCommand{
			db:          db,
			blockDAO:    blockDAO,
			chainClient: chainClient,
			address:     address,
			eth: &ETHDownloader{
				chainClient: chainClient,
				accounts:    []common.Address{address},
				signer:      types.LatestSignerForChainID(chainClient.ToBigInt()),
				db:          db,
			},
			blockNums:          blocksByAddress[address],
			transactionManager: transactionManager,
			pendingTxManager:   pendingTxManager,
			tokenManager:       tokenManager,
			feed:               feed,
		}
		group.Add(transfers.Command())
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
		log.Info("loadTransfers finished for account", "in", time.Since(start), "chain", chainClient.ChainID)
		return nil
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

// Ensure 1 DBHeader per Block Hash
func uniqueHeaderPerBlockHash(allHeaders []*DBHeader) []*DBHeader {
	uniqHeadersByHash := map[common.Hash]*DBHeader{}
	for _, header := range allHeaders {
		uniqHeader, ok := uniqHeadersByHash[header.Hash]
		if ok {
			if len(header.PreloadedTransactions) > 0 {
				uniqHeader.PreloadedTransactions = append(uniqHeader.PreloadedTransactions, header.PreloadedTransactions...)
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

// Ensure 1 PreloadedTransaction per Transaction Hash
func uniquePreloadedTransactionPerTxHash(allTransactions []*PreloadedTransaction) []*PreloadedTransaction {
	uniqTransactionsByTransactionHash := map[common.Hash]*PreloadedTransaction{}
	for _, transaction := range allTransactions {
		uniqTransactionsByTransactionHash[transaction.Log.TxHash] = transaction
	}
	uniqTransactions := []*PreloadedTransaction{}
	for _, transaction := range uniqTransactionsByTransactionHash {
		uniqTransactions = append(uniqTransactions, transaction)
	}

	return uniqTransactions
}

// Organize subTransactions by Transaction Hash
func subTransactionListToTransactionsByTxHash(subTransactions []Transfer) map[common.Hash]Transaction {
	rst := map[common.Hash]Transaction{}

	for index := range subTransactions {
		subTx := &subTransactions[index]
		txHash := subTx.Transaction.Hash()

		if _, ok := rst[txHash]; !ok {
			rst[txHash] = make([]*Transfer, 0)
		}
		rst[txHash] = append(rst[txHash], subTx)
	}

	return rst
}
