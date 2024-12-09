package transfer

import (
	"database/sql"

	"go.uber.org/zap"
	"golang.org/x/exp/slices" // since 1.21, this is in the standard library

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	statusaccounts "github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/services/wallet/blockchainstate"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type Controller struct {
	db                 *Database
	accountsDB         *statusaccounts.Database
	rpcClient          *rpc.Client
	blockDAO           *BlockDAO
	blockRangesSeqDAO  *BlockRangeSequentialDAO
	reactor            *Reactor
	accountFeed        *event.Feed
	TransferFeed       *event.Feed
	accWatcher         *accountsevent.Watcher
	transactionManager *TransactionManager
	pendingTxManager   *transactions.PendingTxTracker
	tokenManager       *token.Manager
	balanceCacher      balance.Cacher
	blockChainState    *blockchainstate.BlockChainState
}

func NewTransferController(db *sql.DB, accountsDB *statusaccounts.Database, rpcClient *rpc.Client, accountFeed *event.Feed, transferFeed *event.Feed,
	transactionManager *TransactionManager, pendingTxManager *transactions.PendingTxTracker, tokenManager *token.Manager,
	balanceCacher balance.Cacher, blockChainState *blockchainstate.BlockChainState) *Controller {

	blockDAO := &BlockDAO{db}
	return &Controller{
		db:                 NewDB(db),
		accountsDB:         accountsDB,
		blockDAO:           blockDAO,
		blockRangesSeqDAO:  &BlockRangeSequentialDAO{db},
		rpcClient:          rpcClient,
		accountFeed:        accountFeed,
		TransferFeed:       transferFeed,
		transactionManager: transactionManager,
		pendingTxManager:   pendingTxManager,
		tokenManager:       tokenManager,
		balanceCacher:      balanceCacher,
		blockChainState:    blockChainState,
	}
}

func (c *Controller) Start(ctx context.Context) {
	go func() {
		defer gocommon.LogOnPanic()
		_ = c.cleanupAccountsLeftovers()
	}()
}

func (c *Controller) Stop() {
	if c.reactor != nil {
		c.reactor.stop()
	}

	if c.accWatcher != nil {
		c.accWatcher.Stop()
		c.accWatcher = nil
	}
}

func (c *Controller) startAccountWatcher(chainIDs []uint64) {
	if c.accWatcher == nil {
		c.accWatcher = accountsevent.NewWatcher(c.accountsDB, c.accountFeed, func(changedAddresses []common.Address, eventType accountsevent.EventType, currentAddresses []common.Address) {
			c.onAccountsChanged(changedAddresses, eventType, currentAddresses, chainIDs)
		})
	}
	c.accWatcher.Start()
}

func (c *Controller) onAccountsChanged(changedAddresses []common.Address, eventType accountsevent.EventType, currentAddresses []common.Address, chainIDs []uint64) {
	if eventType == accountsevent.EventTypeRemoved {
		for _, address := range changedAddresses {
			c.cleanUpRemovedAccount(address)
		}
	}

	if c.reactor == nil {
		logutils.ZapLogger().Warn("reactor is not initialized")
		return
	}

	if eventType == accountsevent.EventTypeAdded || eventType == accountsevent.EventTypeRemoved {
		logutils.ZapLogger().Debug("list of accounts was changed from a previous version. reactor will be restarted", zap.Stringers("new", currentAddresses))

		chainClients, err := c.rpcClient.EthClients(chainIDs)
		if err != nil {
			return
		}

		err = c.reactor.restart(chainClients, currentAddresses)
		if err != nil {
			logutils.ZapLogger().Error("failed to restart reactor with new accounts", zap.Error(err))
		}
	}
}

func (c *Controller) cleanUpRemovedAccount(address common.Address) {
	// Transfers will be deleted by foreign key constraint by cascade
	err := deleteBlocks(c.db.client, address)
	if err != nil {
		logutils.ZapLogger().Error("Failed to delete blocks", zap.Error(err))
	}
	err = deleteAllRanges(c.db.client, address)
	if err != nil {
		logutils.ZapLogger().Error("Failed to delete old blocks ranges", zap.Error(err))
	}

	err = c.blockRangesSeqDAO.deleteRange(address)
	if err != nil {
		logutils.ZapLogger().Error("Failed to delete blocks ranges sequential", zap.Error(err))
	}

	err = c.transactionManager.removeMultiTransactionByAddress(address)
	if err != nil {
		logutils.ZapLogger().Error("Failed to delete multitransactions", zap.Error(err))
	}

	rpcLimitsStorage := rpclimiter.NewLimitsDBStorage(c.db.client)
	err = rpcLimitsStorage.Delete(accountLimiterTag(address))
	if err != nil {
		logutils.ZapLogger().Error("Failed to delete limits", zap.Error(err))
	}
}

func (c *Controller) cleanupAccountsLeftovers() error {
	// We clean up accounts that were deleted and soft removed
	accounts, err := c.accountsDB.GetWalletAddresses()
	if err != nil {
		logutils.ZapLogger().Error("Failed to get accounts", zap.Error(err))
		return err
	}

	existingAddresses := make([]common.Address, len(accounts))
	for i, account := range accounts {
		existingAddresses[i] = (common.Address)(account)
	}

	addressesInWalletDB, err := getAddresses(c.db.client)
	if err != nil {
		logutils.ZapLogger().Error("Failed to get addresses from wallet db", zap.Error(err))
		return err
	}

	missing := findMissingItems(addressesInWalletDB, existingAddresses)
	for _, address := range missing {
		c.cleanUpRemovedAccount(address)
	}

	return nil
}

// find items from one slice that are not in another
func findMissingItems(slice1 []common.Address, slice2 []common.Address) []common.Address {
	var missing []common.Address
	for _, item := range slice1 {
		if !slices.Contains(slice2, item) {
			missing = append(missing, item)
		}
	}
	return missing
}
