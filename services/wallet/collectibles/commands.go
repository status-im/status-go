package collectibles

import (
	"context"
	"database/sql"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	statustypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/async"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	accountOwnershipUpdateInterval = 30 * time.Minute
)

type refreshOwnedCollectiblesCommand struct {
	manager        *Manager
	db             *sql.DB
	eventFeed      *event.Feed
	networkManager *network.Manager
}

func newRefreshOwnedCollectiblesCommand(manager *Manager, db *sql.DB, eventFeed *event.Feed, networkManager *network.Manager) *refreshOwnedCollectiblesCommand {
	return &refreshOwnedCollectiblesCommand{
		manager:        manager,
		db:             db,
		eventFeed:      eventFeed,
		networkManager: networkManager,
	}
}

func (c *refreshOwnedCollectiblesCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: accountOwnershipUpdateInterval,
		Runable:  c.Run,
	}.Run
}

func (c *refreshOwnedCollectiblesCommand) Run(ctx context.Context) (err error) {
	err = c.updateOwnershipForAllAccounts(ctx)
	if ctx.Err() != nil {
		c.triggerEvent(EventCollectiblesOwnershipUpdateFinished, statustypes.Address{}, "Service cancelled")
		return ctx.Err()
	}
	if err != nil {
		c.triggerEvent(EventCollectiblesOwnershipUpdateFinishedWithError, statustypes.Address{}, err.Error())
	}
	return err
}

func (c *refreshOwnedCollectiblesCommand) triggerEvent(eventType walletevent.EventType, account statustypes.Address, message string) {
	c.eventFeed.Send(walletevent.Event{
		Type: eventType,
		Accounts: []common.Address{
			common.Address(account),
		},
		Message: message,
	})
}

func (c *refreshOwnedCollectiblesCommand) updateOwnershipForAllAccounts(ctx context.Context) error {
	accountsDB, err := accounts.NewDB(c.db)
	if err != nil {
		return err
	}

	addresses, err := accountsDB.GetWalletAddresses()
	if err != nil {
		return err
	}

	for _, address := range addresses {
		_ = c.updateOwnershipForAccount(ctx, address)
	}
	return nil
}

func (c *refreshOwnedCollectiblesCommand) updateOwnershipForAccount(ctx context.Context, address statustypes.Address) error {
	networks, err := c.networkManager.Get(false)
	if err != nil {
		return err
	}

	c.triggerEvent(EventCollectiblesOwnershipUpdateStarted, address, "")
	for _, network := range networks {
		err := c.manager.UpdateOwnedCollectibles(walletCommon.ChainID(network.ChainID), common.Address(address))
		if err != nil {
			log.Warn("Error updating collectibles ownership", "chainID", network.ChainID, "address", address.String(), "err", err)
		}
	}
	c.triggerEvent(EventCollectiblesOwnershipUpdateFinished, address, "")

	return nil
}
