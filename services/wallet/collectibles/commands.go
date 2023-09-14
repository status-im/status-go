package collectibles

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/async"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	fetchLimit                     = 50 // Limit number of collectibles we fetch per provider call
	accountOwnershipUpdateInterval = 30 * time.Minute
)

type periodicRefreshOwnedCollectiblesCommand struct {
	chainID     walletCommon.ChainID
	account     common.Address
	manager     *Manager
	ownershipDB *OwnershipDB
	walletFeed  *event.Feed

	group *async.Group
}

func newPeriodicRefreshOwnedCollectiblesCommand(manager *Manager, ownershipDB *OwnershipDB, walletFeed *event.Feed, chainID walletCommon.ChainID, account common.Address) *periodicRefreshOwnedCollectiblesCommand {
	return &periodicRefreshOwnedCollectiblesCommand{
		manager:     manager,
		ownershipDB: ownershipDB,
		walletFeed:  walletFeed,
		chainID:     chainID,
		account:     account,
	}
}

func (c *periodicRefreshOwnedCollectiblesCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: accountOwnershipUpdateInterval,
		Runable:  c.Run,
	}.Run
}

func (c *periodicRefreshOwnedCollectiblesCommand) Run(ctx context.Context) (err error) {
	return c.loadOwnedCollectibles(ctx)
}

func (c *periodicRefreshOwnedCollectiblesCommand) Stop() {
	if c.group != nil {
		c.group.Stop()
		c.group.Wait()
		c.group = nil
	}
}

func (c *periodicRefreshOwnedCollectiblesCommand) loadOwnedCollectibles(ctx context.Context) error {
	c.group = async.NewGroup(ctx)

	command := newLoadOwnedCollectiblesCommand(c.manager, c.ownershipDB, c.walletFeed, c.chainID, c.account)
	c.group.Add(command.Command())

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.group.WaitAsync():
	}

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
	cursor := thirdparty.FetchFromStartCursor
	start := time.Now()

	c.triggerEvent(EventCollectiblesOwnershipUpdateStarted, c.chainID, c.account, "")

	lastFetchTimestamp, err := c.ownershipDB.GetOwnershipUpdateTimestamp(c.account, c.chainID)
	if err != nil {
		c.err = err
	} else {
		initialFetch := lastFetchTimestamp == InvalidTimestamp
		// Fetch collectibles in chunks
		for {
			if shouldCancel(parent) {
				c.err = errors.New("context cancelled")
				break
			}

			pageStart := time.Now()
			log.Debug("start loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "page", pageNr)

			partialOwnership, err := c.manager.FetchCollectibleOwnershipByOwner(c.chainID, c.account, cursor, fetchLimit)

			if err != nil {
				log.Error("failed loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "page", pageNr, "error", err)
				c.err = err
				break
			}

			log.Debug("partial loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "page", pageNr, "in", time.Since(pageStart), "found", len(partialOwnership.Items), "collectibles")

			c.partialOwnership = append(c.partialOwnership, partialOwnership.Items...)

			pageNr++
			cursor = partialOwnership.NextCursor

			finished := cursor == thirdparty.FetchFromStartCursor

			// Normally, update the DB once we've finished fetching
			// If this is the first fetch, make partial updates to the client to get a better UX
			if initialFetch || finished {
				err = c.ownershipDB.Update(c.chainID, c.account, c.partialOwnership, start.Unix())
				if err != nil {
					log.Error("failed updating ownershipDB in loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "error", err)
					c.err = err
				}
			}

			if finished || c.err != nil {
				break
			} else if initialFetch {
				c.triggerEvent(EventCollectiblesOwnershipUpdatePartial, c.chainID, c.account, "")
			}
		}
	}

	if c.err != nil {
		c.triggerEvent(EventCollectiblesOwnershipUpdateFinishedWithError, c.chainID, c.account, c.err.Error())
	} else {
		c.triggerEvent(EventCollectiblesOwnershipUpdateFinished, c.chainID, c.account, "")
	}

	log.Debug("end loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "in", time.Since(start))
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
