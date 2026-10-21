package activityfetcher

import (
	"slices"
	"sync"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/balance/refresh"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc1155"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc721"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/services/wallet/transferdetector"
)

// causeWindow is how long the causes of one account and chain are gathered
// into one activity fetch.
const causeWindow = 10 * time.Second

// causes gathers why activity of an account and chain has to be fetched and
// hands the gathered fetcher IDs to fetch once causeWindow after the first
// cause. A fetch that failed is asked again at the next balance read of its
// account and chain.
type causes struct {
	fetch func([]fetcherID)

	mu      sync.Mutex
	pending []fetcherID
	timer   *time.Timer
	failed  map[fetcherID]struct{}
}

func newCauses(fetch func([]fetcherID)) *causes {
	return &causes{fetch: fetch, failed: make(map[fetcherID]struct{})}
}

func (c *causes) add(ids ...fetcherID) {
	if len(ids) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		if !slices.Contains(c.pending, id) {
			c.pending = append(c.pending, id)
		}
	}
	if c.timer == nil {
		c.timer = time.AfterFunc(causeWindow, c.flush)
	}
}

func (c *causes) flush() {
	defer panics.LogOnPanic()
	c.mu.Lock()
	ids := c.pending
	c.pending = nil
	c.timer = nil
	c.mu.Unlock()
	if len(ids) > 0 {
		c.fetch(ids)
	}
}

func (c *causes) fetchFailed(id fetcherID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failed[id] = struct{}{}
}

func (c *causes) fetchSucceeded(id fetcherID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.failed, id)
}

// balanceRead asks again a failed fetch of the account and chain whose balance
// was just read.
func (c *causes) balanceRead(id fetcherID) {
	c.mu.Lock()
	_, failed := c.failed[id]
	delete(c.failed, id)
	c.mu.Unlock()
	if failed {
		c.add(id)
	}
}

func (c *causes) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.pending = nil
}

// transferCauses are the accounts the logs found by transfer detection show a
// transfer of: known tokens, native transfer logs and every collectible.
// Transfers of other tokens are spam.
func transferCauses(ev transferdetector.EventTransferDetectionFinished, known func(gethcommon.Address) bool) []fetcherID {
	var ids []fetcherID
	addAccount := func(account gethcommon.Address) {
		id := fetcherID{account: account, chainID: ev.ChainID}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	for _, transfer := range refresh.TransfersFromLogs(ev.ChainID, ev.Accounts, ev.Events, known) {
		addAccount(transfer.Pair.Account)
	}
	addCollectible := func(from, to gethcommon.Address) {
		for _, account := range []gethcommon.Address{from, to} {
			if slices.Contains(ev.Accounts, account) {
				addAccount(account)
			}
		}
	}
	for _, event := range ev.Events {
		switch t := event.Unpacked.(type) {
		case erc721.Erc721Transfer:
			if event.EventKey == eventlog.ERC721Transfer {
				addCollectible(t.From, t.To)
			}
		case erc1155.Erc1155TransferSingle:
			addCollectible(t.From, t.To)
		case erc1155.Erc1155TransferBatch:
			addCollectible(t.From, t.To)
		}
	}
	return ids
}

// gapCauses are the accounts of a range transfer detection failed to look at.
func gapCauses(ev transferdetector.EventTransferDetectionError) []fetcherID {
	ids := make([]fetcherID, 0, len(ev.Accounts))
	for _, account := range ev.Accounts {
		ids = append(ids, fetcherID{account: account, chainID: ev.ChainID})
	}
	return ids
}
