package rpcfilters

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pborman/uuid"
)

type filter struct {
	hashes []common.Hash
	mu     sync.Mutex
	done   chan struct{}
}

// AddHash adds a hash to the filter
func (f *filter) AddHash(hash common.Hash) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hashes = append(f.hashes, hash)
}

// PopHashes returns all the hashes stored in the filter and clears the filter contents
func (f *filter) PopHashes() []common.Hash {
	f.mu.Lock()
	defer f.mu.Unlock()
	hashes := f.hashes
	f.hashes = nil
	return returnHashes(hashes)
}

func newFilter() *filter {
	return &filter{
		done: make(chan struct{}),
	}
}

// PublicAPI represents filter API that is exported to `eth` namespace
type PublicAPI struct {
	filters                        map[rpc.ID]*filter
	filtersMu                      sync.Mutex
	latestBlockChangedEvent        *latestBlockChangedEvent
	transactionSentToUpstreamEvent *transactionSentToUpstreamEvent
	blockProvider                  *blockProviderRPC
}

// NewPublicAPI returns a reference to the PublicAPI object
func NewPublicAPI(latestBlockChangedEvent *latestBlockChangedEvent,
	transactionSentToUpstreamEvent *transactionSentToUpstreamEvent,
	blockProvider *blockProviderRPC) *PublicAPI {
	return &PublicAPI{
		filters:                        make(map[rpc.ID]*filter),
		latestBlockChangedEvent:        latestBlockChangedEvent,
		transactionSentToUpstreamEvent: transactionSentToUpstreamEvent,
		blockProvider:                  blockProvider,
	}
}

// NewBlockFilter is an implemenation of `eth_newBlockFilter` API
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (api *PublicAPI) NewBlockFilter() rpc.ID {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	f := newFilter()
	id := rpc.ID(uuid.New())

	api.filters[id] = f

	go func() {
		id, s := api.latestBlockChangedEvent.Subscribe()
		defer api.latestBlockChangedEvent.Unsubscribe(id)

		for {
			select {
			case hash := <-s:
				f.AddHash(hash)
			case <-f.done:
				return
			}
		}

	}()

	return id
}

// NewPendingTransactionFilter is an implementation of `eth_newPendingTransactionFilter` API
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newpendingtransactionfilter
func (api *PublicAPI) NewPendingTransactionFilter() rpc.ID {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	f := newFilter()
	id := rpc.ID(uuid.New())

	api.filters[id] = f

	go func() {
		id, s := api.transactionSentToUpstreamEvent.Subscribe()
		defer api.transactionSentToUpstreamEvent.Unsubscribe(id)

		for {
			select {
			case hash := <-s:
				f.AddHash(hash)
			case <-f.done:
				return
			}
		}

	}()

	return id

}

// NewFilter creates a new filter and returns the filter id. It can be
// used to retrieve logs when the state changes. This method cannot be
// used to fetch logs that are already stored in the state.
//
// Default criteria for the from and to block are "latest".
// Using "latest" as block number will return logs for mined blocks.
// Using "pending" as block number returns logs for not yet mined (pending) blocks.
// In case logs are removed (chain reorg) previously returned logs are returned
// again but with the removed property set to true.
//
// In case "fromBlock" > "toBlock" an error is returned.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (api *PublicAPI) NewFilter(crit FilterCriteria) (rpc.ID, error) {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	f := newFilter()
	id := rpc.ID(uuid.New())

	api.filters[id] = f

	logFilter, err := NewLogFilter(api.blockProvider, &crit)

	go func() {
		id, s := logFilter.Subscribe()
		defer logFilter.Unsubscribe(id)

		for {
			select {
			case hash := <-s:
				f.AddHash(hash)
			case <-f.done:
				return
			}
		}

	}()

	logFilter.Log()

	return id, err

}

// UninstallFilter is an implemenation of `eth_uninstallFilter` API
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_uninstallfilter
func (api *PublicAPI) UninstallFilter(id rpc.ID) bool {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	if found {
		delete(api.filters, id)
	}
	api.filtersMu.Unlock()

	if found {
		close(f.done)
	}

	return found
}

// GetFilterChanges returns the hashes for the filter with the given id since
// last time it was called. This can be used for polling.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (api *PublicAPI) GetFilterChanges(id rpc.ID) ([]common.Hash, error) {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	if f, found := api.filters[id]; found {
		return f.PopHashes(), nil
	}

	return []common.Hash{}, errors.New("filter not found")
}

// returnHashes is a helper that will return an empty hash array case the given hash array is nil,
// otherwise the given hashes array is returned.
func returnHashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}
