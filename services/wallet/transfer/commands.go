package transfer

import (
	"context"
	"database/sql"
	"math/big"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/logutils"
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

	// Internal events emitted when different kinds of transfers are detected
	EventInternalETHTransferDetected     walletevent.EventType = walletevent.InternalEventTypePrefix + "eth-transfer-detected"
	EventInternalERC20TransferDetected   walletevent.EventType = walletevent.InternalEventTypePrefix + "erc20-transfer-detected"
	EventInternalERC721TransferDetected  walletevent.EventType = walletevent.InternalEventTypePrefix + "erc721-transfer-detected"
	EventInternalERC1155TransferDetected walletevent.EventType = walletevent.InternalEventTypePrefix + "erc1155-transfer-detected"

	numberOfBlocksCheckedPerIteration = 40
	noBlockLimit                      = 0
)

var (
	// This will work only for binance testnet as mainnet doesn't support
	// archival request.
	binanceChainErc20BatchSize    = big.NewInt(5000)
	sepoliaErc20BatchSize         = big.NewInt(100000)
	sepoliaErc20ArbitrumBatchSize = big.NewInt(10000)
	sepoliaErc20OptimismBatchSize = big.NewInt(10000)
	erc20BatchSize                = big.NewInt(100000)

	transfersRetryInterval = 5 * time.Second
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
	logutils.ZapLogger().Debug("eth historical downloader start",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringer("address", c.address),
		zap.Stringer("from", c.from.Number),
		zap.Stringer("to", c.to),
		zap.Bool("noLimit", c.noLimit),
	)

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
		logutils.ZapLogger().Error("failed to find blocks with transfers",
			zap.Uint64("chainID", c.chainClient.NetworkID()),
			zap.Stringer("address", c.address),
			zap.Stringer("from", c.from.Number),
			zap.Stringer("to", c.to),
			zap.Error(err),
		)
		return nil
	}

	c.foundHeaders = headers
	c.resultingFrom = from
	c.startBlock = startBlock

	logutils.ZapLogger().Debug("eth historical downloader finished successfully",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringer("address", c.address),
		zap.Stringer("from", from),
		zap.Stringer("to", c.to),
		zap.Int("totalBlocks", len(headers)),
		zap.Duration("time", time.Since(start)),
	)

	return nil
}

type erc20HistoricalCommand struct {
	erc20       BatchDownloader
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
	switch chainID {
	case w_common.EthereumSepolia:
		return sepoliaErc20BatchSize
	case w_common.OptimismSepolia:
		return sepoliaErc20OptimismBatchSize
	case w_common.ArbitrumSepolia:
		return sepoliaErc20ArbitrumBatchSize
	case w_common.BinanceChainID:
		return binanceChainErc20BatchSize
	case w_common.BinanceTestChainID:
		return binanceChainErc20BatchSize
	default:
		return erc20BatchSize
	}
}

func (c *erc20HistoricalCommand) Run(ctx context.Context) (err error) {
	logutils.ZapLogger().Debug("wallet historical downloader for erc20 transfers start",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringer("from", c.from),
		zap.Stringer("to", c.to),
	)

	start := time.Now()
	if c.iterator == nil {
		c.iterator, err = SetupIterativeDownloader(
			c.chainClient,
			c.erc20, getErc20BatchSize(c.chainClient.NetworkID()), c.to, c.from)
		if err != nil {
			logutils.ZapLogger().Error("failed to setup historical downloader for erc20")
			return err
		}
	}
	for !c.iterator.Finished() {
		headers, _, _, err := c.iterator.Next(ctx)
		if err != nil {
			logutils.ZapLogger().Error("failed to get next batch",
				zap.Uint64("chainID", c.chainClient.NetworkID()),
				zap.Error(err),
			) // TODO: stop inifinite command in case of an error that we can't fix like missing trie node
			return err
		}
		c.foundHeaders = append(c.foundHeaders, headers...)
	}
	logutils.ZapLogger().Debug("wallet historical downloader for erc20 transfers finished",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringer("from", c.from),
		zap.Stringer("to", c.to),
		zap.Duration("time", time.Since(start)),
		zap.Int("headers", len(c.foundHeaders)),
	)
	return nil
}

type transfersCommand struct {
	db               *Database
	blockDAO         *BlockDAO
	eth              *ETHDownloader
	blockNums        []*big.Int
	address          common.Address
	chainClient      chain.ClientInterface
	blocksLimit      int
	pendingTxManager *transactions.PendingTxTracker
	tokenManager     *token.Manager
	feed             *event.Feed

	// result
	fetchedTransfers []Transfer
}

func (c *transfersCommand) Runner(interval ...time.Duration) async.Runner {
	intvl := transfersRetryInterval
	if len(interval) > 0 {
		intvl = interval[0]
	}
	return async.FiniteCommandWithErrorCounter{
		FiniteCommand: async.FiniteCommand{
			Interval: intvl,
			Runable:  c.Run,
		},
		ErrorCounter: async.NewErrorCounter(5, "transfersCommand"),
	}
}

func (c *transfersCommand) Command(interval ...time.Duration) async.Command {
	return c.Runner(interval...).Run
}

func (c *transfersCommand) Run(ctx context.Context) (err error) {
	// Take blocks from cache if available and disrespect the limit
	// If no blocks are available in cache, take blocks from DB respecting the limit
	// If no limit is set, take all blocks from DB
	logutils.ZapLogger().Debug("start transfersCommand",
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Stringer("address", c.address),
		zap.Stringers("blockNums", c.blockNums),
	)
	startTs := time.Now()

	for {
		blocks := c.blockNums
		if blocks == nil {
			blocks, _ = c.blockDAO.GetBlocksToLoadByAddress(c.chainClient.NetworkID(), c.address, numberOfBlocksCheckedPerIteration)
		}

		for _, blockNum := range blocks {
			logutils.ZapLogger().Debug("transfersCommand block start",
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Stringer("address", c.address),
				zap.Stringer("blockNum", blockNum),
			)

			allTransfers, err := c.eth.GetTransfersByNumber(ctx, blockNum)
			if err != nil {
				logutils.ZapLogger().Error("getTransfersByBlocks error", zap.Error(err))
				return err
			}

			c.processUnknownErc20CommunityTransactions(ctx, allTransfers)

			if len(allTransfers) > 0 {
				// First, try to match to any pre-existing pending/multi-transaction
				err := c.saveAndConfirmPending(allTransfers, blockNum)
				if err != nil {
					logutils.ZapLogger().Error("saveAndConfirmPending error", zap.Error(err))
					return err
				}
			} else {
				// If no transfers found, that is suspecting, because downloader returned this block as containing transfers
				logutils.ZapLogger().Error("no transfers found in block",
					zap.Uint64("chain", c.chainClient.NetworkID()),
					zap.Stringer("address", c.address),
					zap.Stringer("block", blockNum),
				)

				err = markBlocksAsLoaded(c.chainClient.NetworkID(), c.db.client, c.address, []*big.Int{blockNum})
				if err != nil {
					logutils.ZapLogger().Error("Mark blocks loaded error", zap.Error(err))
					return err
				}
			}

			c.fetchedTransfers = append(c.fetchedTransfers, allTransfers...)

			c.notifyOfNewTransfers(blockNum, allTransfers)
			c.notifyOfLatestTransfers(allTransfers, w_common.EthTransfer)
			c.notifyOfLatestTransfers(allTransfers, w_common.Erc20Transfer)
			c.notifyOfLatestTransfers(allTransfers, w_common.Erc721Transfer)
			c.notifyOfLatestTransfers(allTransfers, w_common.Erc1155Transfer)

			logutils.ZapLogger().Debug("transfersCommand block end",
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Stringer("address", c.address),
				zap.Stringer("blockNum", blockNum),
				zap.Int("transfersLen", len(allTransfers)),
				zap.Int("fetchedTransfersLen", len(c.fetchedTransfers)),
			)
		}

		if c.blockNums != nil || len(blocks) == 0 ||
			(c.blocksLimit > noBlockLimit && len(blocks) >= c.blocksLimit) {
			logutils.ZapLogger().Debug("loadTransfers breaking loop on block limits reached or 0 blocks",
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Stringer("address", c.address),
				zap.Int("limit", c.blocksLimit),
				zap.Int("blocks", len(blocks)),
			)
			break
		}
	}

	logutils.ZapLogger().Debug("end transfersCommand",
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Stringer("address", c.address),
		zap.Int("blocks.len", len(c.blockNums)),
		zap.Int("transfers.len", len(c.fetchedTransfers)),
		zap.Duration("in", time.Since(startTs)),
	)

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
	notifyFunctions := c.confirmPendingTransactions(tx, allTransfers)
	defer func() {
		if resErr == nil {
			commitErr := tx.Commit()
			if commitErr != nil {
				logutils.ZapLogger().Error("failed to commit", zap.Error(commitErr))
			}
			for _, notify := range notifyFunctions {
				notify()
			}
		} else {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				logutils.ZapLogger().Error("failed to rollback", zap.Error(rollbackErr))
			}
		}
	}()

	resErr = saveTransfersMarkBlocksLoaded(tx, c.chainClient.NetworkID(), c.address, allTransfers, []*big.Int{blockNum})
	if resErr != nil {
		logutils.ZapLogger().Error("SaveTransfers error", zap.Error(resErr))
	}

	return resErr
}

func externalTransactionOrError(err error, mTID int64) bool {
	if err == sql.ErrNoRows {
		// External transaction downloaded, ignore it
		return true
	} else if err != nil {
		logutils.ZapLogger().Warn("GetOwnedMultiTransactionID", zap.Error(err))
		return true
	} else if mTID <= 0 {
		// Existing external transaction, ignore it
		return true
	}
	return false
}

func (c *transfersCommand) confirmPendingTransactions(tx *sql.Tx, allTransfers []Transfer) (notifyFunctions []func()) {
	notifyFunctions = make([]func(), 0)

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
				existingMTID, err := GetOwnedMultiTransactionID(tx, chainID, txHash, tr.Address)
				if externalTransactionOrError(err, existingMTID) {
					continue
				}
				mTID = w_common.NewAndSet(existingMTID)
			}
		} else if err != nil {
			logutils.ZapLogger().Warn("GetOwnedPendingStatus", zap.Error(err))
			continue
		}

		if mTID != nil {
			allTransfers[i].MultiTransactionID = w_common.MultiTransactionIDType(*mTID)
		}
		if txType != nil && *txType == transactions.WalletTransfer {
			notify, err := c.pendingTxManager.DeleteBySQLTx(tx, chainID, txHash)
			if err != nil && err != transactions.ErrStillPending {
				logutils.ZapLogger().Error("DeleteBySqlTx error", zap.Error(err))
			}
			notifyFunctions = append(notifyFunctions, notify)
		}
	}
	return notifyFunctions
}

func (c *transfersCommand) processUnknownErc20CommunityTransactions(ctx context.Context, allTransfers []Transfer) {
	for _, tx := range allTransfers {
		// To can be nil in case of erc20 contract creation
		if tx.Type == w_common.Erc20Transfer && tx.Transaction.To() != nil {
			// Find token in db or if this is a community token, find its metadata
			token := c.tokenManager.FindOrCreateTokenByAddress(ctx, tx.NetworkID, *tx.Transaction.To())
			if token != nil {
				isFirst := false
				if token.Verified || token.CommunityData != nil {
					isFirst, _ = c.tokenManager.MarkAsPreviouslyOwnedToken(token, tx.Address)
				}
				if token.CommunityData != nil {
					go c.tokenManager.SignalCommunityTokenReceived(tx.Address, tx.ID, tx.TokenValue, token, isFirst)
				}
			}
		}
	}
}

func (c *transfersCommand) notifyOfNewTransfers(blockNum *big.Int, transfers []Transfer) {
	if c.feed != nil {
		if len(transfers) > 0 {
			c.feed.Send(walletevent.Event{
				Type:        EventNewTransfers,
				Accounts:    []common.Address{c.address},
				ChainID:     c.chainClient.NetworkID(),
				BlockNumber: blockNum,
			})
		}
	}
}

func transferTypeToEventType(transferType w_common.Type) walletevent.EventType {
	switch transferType {
	case w_common.EthTransfer:
		return EventInternalETHTransferDetected
	case w_common.Erc20Transfer:
		return EventInternalERC20TransferDetected
	case w_common.Erc721Transfer:
		return EventInternalERC721TransferDetected
	case w_common.Erc1155Transfer:
		return EventInternalERC1155TransferDetected
	default:
		return ""
	}
}

func (c *transfersCommand) notifyOfLatestTransfers(transfers []Transfer, transferType w_common.Type) {
	if c.feed != nil {
		eventTransfers := make([]Transfer, 0, len(transfers))
		latestTransferTimestamp := uint64(0)
		for _, transfer := range transfers {
			if transfer.Type == transferType {
				eventTransfers = append(eventTransfers, transfer)
				if transfer.Timestamp > latestTransferTimestamp {
					latestTransferTimestamp = transfer.Timestamp
				}
			}
		}
		if len(eventTransfers) > 0 {
			c.feed.Send(walletevent.Event{
				Type:        transferTypeToEventType(transferType),
				Accounts:    []common.Address{c.address},
				ChainID:     c.chainClient.NetworkID(),
				At:          int64(latestTransferTimestamp),
				EventParams: eventTransfers,
			})
		}
	}
}

type loadTransfersCommand struct {
	accounts         []common.Address
	db               *Database
	blockDAO         *BlockDAO
	chainClient      chain.ClientInterface
	blocksByAddress  map[common.Address][]*big.Int
	pendingTxManager *transactions.PendingTxTracker
	blocksLimit      int
	tokenManager     *token.Manager
	feed             *event.Feed
}

func (c *loadTransfersCommand) Command() async.Command {
	return async.FiniteCommand{
		Interval: 5 * time.Second,
		Runable:  c.Run,
	}.Run
}

// This command always returns nil, even if there is an error in one of the commands.
// `transferCommand`s retry until maxError, but this command doesn't retry.
// In case some transfer is not loaded after max retries, it will be retried only after restart of the app.
// Currently there is no implementation to keep retrying until success. I think this should be implemented
// in `transferCommand` with exponential backoff instead of `loadTransfersCommand` (issue #4608).
func (c *loadTransfersCommand) Run(parent context.Context) (err error) {
	return loadTransfers(parent, c.blockDAO, c.db, c.chainClient, c.blocksLimit, c.blocksByAddress,
		c.pendingTxManager, c.tokenManager, c.feed)
}

func loadTransfers(ctx context.Context, blockDAO *BlockDAO, db *Database,
	chainClient chain.ClientInterface, blocksLimitPerAccount int, blocksByAddress map[common.Address][]*big.Int,
	pendingTxManager *transactions.PendingTxTracker, tokenManager *token.Manager, feed *event.Feed) error {

	logutils.ZapLogger().Debug("loadTransfers start",
		zap.Uint64("chain", chainClient.NetworkID()),
		zap.Int("limit", blocksLimitPerAccount),
	)

	start := time.Now()
	group := async.NewGroup(ctx)

	accounts := maps.Keys(blocksByAddress)
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
			blockNums:        blocksByAddress[address],
			pendingTxManager: pendingTxManager,
			tokenManager:     tokenManager,
			feed:             feed,
		}
		group.Add(transfers.Command())
	}

	select {
	case <-ctx.Done():
		logutils.ZapLogger().Debug("loadTransfers cancelled",
			zap.Uint64("chain", chainClient.NetworkID()),
			zap.Error(ctx.Err()),
		)
	case <-group.WaitAsync():
		logutils.ZapLogger().Debug("loadTransfers finished for account",
			zap.Duration("in", time.Since(start)),
			zap.Uint64("chain", chainClient.NetworkID()),
		)
	}
	return nil
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
