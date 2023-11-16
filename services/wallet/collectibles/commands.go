package collectibles

import (
	"context"
	"errors"
	"sync/atomic"
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
	fetchLimit                          = 50 // Limit number of collectibles we fetch per provider call
	accountOwnershipUpdateInterval      = 60 * time.Minute
	accountOwnershipUpdateDelayInterval = 30 * time.Second
)

type OwnershipState = int

type OwnedCollectibles struct {
	chainID walletCommon.ChainID
	account common.Address
	ids     []thirdparty.CollectibleUniqueID
}

type OwnedCollectiblesCb func(OwnedCollectibles)

const (
	OwnershipStateIdle OwnershipState = iota + 1
	OwnershipStateDelayed
	OwnershipStateUpdating
	OwnershipStateError
)

type periodicRefreshOwnedCollectiblesCommand struct {
	chainID                walletCommon.ChainID
	account                common.Address
	manager                *Manager
	ownershipDB            *OwnershipDB
	walletFeed             *event.Feed
	receivedCollectiblesCb OwnedCollectiblesCb

	group *async.Group
	state atomic.Value
}

func newPeriodicRefreshOwnedCollectiblesCommand(
	manager *Manager,
	ownershipDB *OwnershipDB,
	walletFeed *event.Feed,
	chainID walletCommon.ChainID,
	account common.Address,
	receivedCollectiblesCb OwnedCollectiblesCb) *periodicRefreshOwnedCollectiblesCommand {
	ret := &periodicRefreshOwnedCollectiblesCommand{
		manager:                manager,
		ownershipDB:            ownershipDB,
		walletFeed:             walletFeed,
		chainID:                chainID,
		account:                account,
		receivedCollectiblesCb: receivedCollectiblesCb,
	}
	ret.state.Store(OwnershipStateIdle)
	return ret
}

func (c *periodicRefreshOwnedCollectiblesCommand) DelayedCommand() async.Command {
	return async.SingleShotCommand{
		Interval: accountOwnershipUpdateDelayInterval,
		Init: func(ctx context.Context) (err error) {
			c.state.Store(OwnershipStateDelayed)
			return nil
		},
		Runable: c.Command(),
	}.Run
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

func (c *periodicRefreshOwnedCollectiblesCommand) GetState() OwnershipState {
	return c.state.Load().(OwnershipState)
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

	receivedCollectiblesCh := make(chan OwnedCollectibles)
	command := newLoadOwnedCollectiblesCommand(c.manager, c.ownershipDB, c.walletFeed, c.chainID, c.account, receivedCollectiblesCh)

	c.state.Store(OwnershipStateUpdating)
	defer func() {
		if command.err != nil {
			c.state.Store(OwnershipStateError)
		} else {
			c.state.Store(OwnershipStateIdle)
		}
	}()

	c.group.Add(command.Command())

	select {
	case ownedCollectibles := <-receivedCollectiblesCh:
		if c.receivedCollectiblesCb != nil {
			c.receivedCollectiblesCb(ownedCollectibles)
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-c.group.WaitAsync():
		return nil
	}

	return nil
}

// Fetches owned collectibles for a ChainID+OwnerAddress combination in chunks
// and updates the ownershipDB when all chunks are loaded
type loadOwnedCollectiblesCommand struct {
	chainID                walletCommon.ChainID
	account                common.Address
	manager                *Manager
	ownershipDB            *OwnershipDB
	walletFeed             *event.Feed
	receivedCollectiblesCh chan<- OwnedCollectibles

	// Not to be set by the caller
	partialOwnership []thirdparty.CollectibleUniqueID
	err              error
}

func newLoadOwnedCollectiblesCommand(
	manager *Manager,
	ownershipDB *OwnershipDB,
	walletFeed *event.Feed,
	chainID walletCommon.ChainID,
	account common.Address,
	receivedCollectiblesCh chan<- OwnedCollectibles) *loadOwnedCollectiblesCommand {
	return &loadOwnedCollectiblesCommand{
		manager:                manager,
		ownershipDB:            ownershipDB,
		walletFeed:             walletFeed,
		chainID:                chainID,
		account:                account,
		receivedCollectiblesCh: receivedCollectiblesCh,
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
	providerID := thirdparty.FetchFromAnyProvider
	start := time.Now()

	c.triggerEvent(EventCollectiblesOwnershipUpdateStarted, c.chainID, c.account, "")

	lastFetchTimestamp, err := c.ownershipDB.GetOwnershipUpdateTimestamp(c.account, c.chainID)
	if err != nil {
		c.err = err
	} else {
		initialFetch := lastFetchTimestamp == InvalidTimestamp
		// Fetch collectibles in chunks
		for {
			if walletCommon.ShouldCancel(parent) {
				c.err = errors.New("context cancelled")
				break
			}

			pageStart := time.Now()
			log.Debug("start loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "page", pageNr)

			partialOwnership, err := c.manager.FetchCollectibleOwnershipByOwner(parent, c.chainID, c.account, cursor, fetchLimit, providerID)

			if err != nil {
				log.Error("failed loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "page", pageNr, "error", err)
				c.err = err
				break
			}

			log.Debug("partial loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "page", pageNr, "in", time.Since(pageStart), "found", len(partialOwnership.Items))

			c.partialOwnership = append(c.partialOwnership, partialOwnership.Items...)

			pageNr++
			cursor = partialOwnership.NextCursor
			providerID = partialOwnership.Provider

			finished := cursor == thirdparty.FetchFromStartCursor

			// Normally, update the DB once we've finished fetching
			// If this is the first fetch, make partial updates to the client to get a better UX
			if initialFetch || finished {
				receivedIDs, err := c.ownershipDB.GetIDsNotInDB(c.chainID, c.account, c.partialOwnership)
				if err != nil {
					log.Error("failed GetIDsNotInDB in processOwnedIDs", "chain", c.chainID, "account", c.account, "error", err)
					return err
				}

				err = c.ownershipDB.Update(c.chainID, c.account, c.partialOwnership, start.Unix())
				if err != nil {
					log.Error("failed updating ownershipDB in loadOwnedCollectiblesCommand", "chain", c.chainID, "account", c.account, "error", err)
					c.err = err
				}

				c.receivedCollectiblesCh <- OwnedCollectibles{
					chainID: c.chainID,
					account: c.account,
					ids:     receivedIDs,
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
