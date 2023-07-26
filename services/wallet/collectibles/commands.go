package collectibles

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/async"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	fetchLimit                     = 50 // Limit number of collectibles we fetch per provider call
	accountOwnershipUpdateInterval = 30 * time.Minute
)

// Fetches owned collectibles for all chainIDs and wallet addresses
type refreshOwnedCollectiblesCommand struct {
	manager        *Manager
	ownershipDB    *OwnershipDB
	accountsDB     *accounts.Database
	walletFeed     *event.Feed
	networkManager *network.Manager
}

func newRefreshOwnedCollectiblesCommand(manager *Manager, ownershipDB *OwnershipDB, accountsDB *accounts.Database, walletFeed *event.Feed, networkManager *network.Manager) *refreshOwnedCollectiblesCommand {
	return &refreshOwnedCollectiblesCommand{
		manager:        manager,
		ownershipDB:    ownershipDB,
		accountsDB:     accountsDB,
		walletFeed:     walletFeed,
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
	return c.updateOwnershipForAllAccounts(ctx)
}

func (c *refreshOwnedCollectiblesCommand) updateOwnershipForAllAccounts(ctx context.Context) error {
	networks, err := c.networkManager.Get(false)
	if err != nil {
		return err
	}

	addresses, err := c.accountsDB.GetWalletAddresses()
	if err != nil {
		return err
	}

	areTestNetworksEnabled, err := c.accountsDB.GetTestNetworksEnabled()
	if err != nil {
		return err
	}

	start := time.Now()
	group := async.NewGroup(ctx)

	log.Debug("refreshOwnedCollectiblesCommand started")

	for _, network := range networks {
		if network.IsTest != areTestNetworksEnabled {
			continue
		}
		for _, address := range addresses {
			command := newLoadOwnedCollectiblesCommand(c.manager, c.ownershipDB, c.walletFeed, walletCommon.ChainID(network.ChainID), common.Address(address))
			group.Add(command.Command())
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-group.WaitAsync():
	}

	log.Debug("refreshOwnedCollectiblesCommand finished", "in", time.Since(start))

	return nil
}

// Fetches owned collectibles for a ChainID+OwnerAddress combination in chunks
// and updates the ownershipDB when all chunks are loaded
type loadOwnedCollectiblesCommand struct {
	chainID     walletCommon.ChainID
	account     common.Address
	manager     *Manager
	ownershipDB *OwnershipDB
	walletFeed  *event.Feed

	// Not to be set by the caller
	partialOwnership []thirdparty.CollectibleUniqueID
	err              error
}

func newLoadOwnedCollectiblesCommand(manager *Manager, ownershipDB *OwnershipDB, walletFeed *event.Feed, chainID walletCommon.ChainID, account common.Address) *loadOwnedCollectiblesCommand {
	return &loadOwnedCollectiblesCommand{
		manager:     manager,
		ownershipDB: ownershipDB,
		walletFeed:  walletFeed,
		chainID:     chainID,
		account:     account,
	}
}

func (c *loadOwnedCollectiblesCommand) Command() async.Command {
	return c.Run
}

func (c *loadOwnedCollectiblesCommand) triggerEvent(eventType walletevent.EventType, chainID walletCommon.ChainID, account common.Address, message string) {
	c.walletFeed.Send(walletevent.Event{
		Type:    eventType,
		ChainID: uint64(chainID),
		Accounts: []common.Address{
			account,
		},
		Message: message,
	})
}

func (c *loadOwnedCollectiblesCommand) Run(parent context.Context) (err error) {
	log.Debug("start loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account)

	pageNr := 0
	cursor := FetchFromStartCursor

	c.triggerEvent(EventCollectiblesOwnershipUpdateStarted, c.chainID, c.account, "")
	// Fetch collectibles in chunks
	for {
		if shouldCancel(parent) {
			c.err = errors.New("context cancelled")
			break
		}

		partialOwnership, err := c.manager.FetchCollectibleOwnershipByOwner(c.chainID, c.account, cursor, fetchLimit)

		if err != nil {
			log.Error("failed loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "page", pageNr, "error", err)
			c.err = err
			break
		}

		log.Debug("partial loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "page", pageNr, "found", len(partialOwnership.Collectibles), "collectibles")

		c.partialOwnership = append(c.partialOwnership, partialOwnership.Collectibles...)

		pageNr++
		cursor = partialOwnership.NextCursor

		if cursor == FetchFromStartCursor {
			err = c.ownershipDB.Update(c.chainID, c.account, c.partialOwnership)
			if err != nil {
				log.Error("failed updating ownershipDB in loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "error", err)
				c.err = err
			}
			break
		}
	}

	if c.err != nil {
		c.triggerEvent(EventCollectiblesOwnershipUpdateFinishedWithError, c.chainID, c.account, c.err.Error())
	} else {
		c.triggerEvent(EventCollectiblesOwnershipUpdateFinished, c.chainID, c.account, "")
	}

	log.Debug("end loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account)
	return nil
}

// shouldCancel returns true if the context has been cancelled and task should be aborted
func shouldCancel(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}
