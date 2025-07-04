package transfer

import (
	"context"
	"database/sql"
	"slices"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	statusaccounts "github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/services/wallet/blockchainstate"
)

type Controller struct {
	db                 *Database
	accountsDB         *statusaccounts.Database
	rpcClient          *rpc.Client
	blockDAO           *BlockDAO
	blockRangesSeqDAO  *BlockRangeSequentialDAO
	reactor            *Reactor
	accountPublisher   *pubsub.Publisher
	transactionManager *TransactionManager
	blockChainState    *blockchainstate.BlockChainState
	stopCh             chan struct{}
}

func NewTransferController(db *sql.DB, accountsDB *statusaccounts.Database, rpcClient *rpc.Client, accountPublisher *pubsub.Publisher,
	transactionManager *TransactionManager, blockChainState *blockchainstate.BlockChainState) *Controller {

	blockDAO := &BlockDAO{db}
	return &Controller{
		db:                 NewDB(db),
		accountsDB:         accountsDB,
		blockDAO:           blockDAO,
		blockRangesSeqDAO:  &BlockRangeSequentialDAO{db},
		rpcClient:          rpcClient,
		accountPublisher:   accountPublisher,
		transactionManager: transactionManager,
		blockChainState:    blockChainState,
	}
}

func (c *Controller) Start(ctx context.Context) {
	c.stopCh = make(chan struct{})
	go func() {
		defer gocommon.LogOnPanic()
		_ = c.cleanupAccountsLeftovers()
	}()
	c.startAccountWatcher()
	c.startNetworksWatcher()
}

func (c *Controller) Stop() {
	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
	}

	if c.reactor != nil {
		c.reactor.stop()
	}
}

func (c *Controller) startAccountWatcher() {
	if c.accountPublisher == nil {
		return
	}

	chAdded, unsubAddedFn := pubsub.Subscribe[accountsevent.AccountsAddedEvent](c.accountPublisher, 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsubAddedFn()
		for {
			select {
			case <-c.stopCh:
				return
			case _, ok := <-chAdded:
				if !ok {
					return
				}
				c.restartReactor()
			}
		}
	}()

	chRemoved, unsubRemovedFn := pubsub.Subscribe[accountsevent.AccountsRemovedEvent](c.accountPublisher, 10)
	go func() {
		defer gocommon.LogOnPanic()
		defer unsubRemovedFn()
		for {
			select {
			case <-c.stopCh:
				return
			case _, ok := <-chRemoved:
				if !ok {
					return
				}
				c.restartReactor()
			}
		}
	}()
}

func (c *Controller) startNetworksWatcher() {
	if c.rpcClient != nil && c.rpcClient.NetworkManager != nil {
		networksPublisher := c.rpcClient.NetworkManager.GetPublisher()
		if networksPublisher == nil {
			return
		}

		ch, unsubFn := pubsub.Subscribe[network.EventActiveNetworksChanged](networksPublisher, 10)
		go func() {
			defer gocommon.LogOnPanic()
			defer unsubFn()
			for {
				select {
				case <-c.stopCh:
					return
				case _, ok := <-ch:
					if !ok {
						return
					}
					c.restartReactor()
				}
			}
		}()
	}
}

func (c *Controller) restartReactor() {
	if c.reactor == nil {
		logutils.ZapLogger().Warn("reactor is not initialized")
		return
	}

	currentEthAddresses, err := c.accountsDB.GetWalletAddresses()

	if err != nil {
		logutils.ZapLogger().Error("failed getting wallet addresses", zap.Error(err))
		return
	}

	currentAddresses := make([]common.Address, 0, len(currentEthAddresses))
	for _, ethAddress := range currentEthAddresses {
		currentAddresses = append(currentAddresses, common.Address(ethAddress))
	}

	logutils.ZapLogger().Debug("list of accounts was changed from a previous version. reactor will be restarted", zap.Stringers("new", currentAddresses))

	currentNetworks, err := c.rpcClient.NetworkManager.Get(false)
	if err != nil {
		logutils.ZapLogger().Error("failed getting active networks", zap.Error(err))
		return
	}

	chainIDs := make([]uint64, 0, len(currentNetworks))
	for _, network := range currentNetworks {
		chainIDs = append(chainIDs, network.ChainID)
	}

	chainClients, err := c.rpcClient.EthClients(chainIDs)
	if err != nil {
		return
	}

	err = c.reactor.restart(chainClients, currentAddresses)
	if err != nil {
		logutils.ZapLogger().Error("failed to restart reactor with new accounts", zap.Error(err))
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
