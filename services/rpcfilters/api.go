package rpcfilters

import (
	"context"
	"errors"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pborman/uuid"
)

const (
	filterLivenessPeriod = 5 * time.Minute
	logsPeriod           = 10 * time.Second
	logsQueryTimeout     = 10 * time.Second
)

type filter interface {
	add(interface{})
	pop() interface{}
	stop()
	deadline() *time.Timer
}

// PublicAPI represents filter API that is exported to `eth` namespace
type PublicAPI struct {
	filtersMu sync.Mutex
	filters   map[rpc.ID]filter

	filterLivenessLoop   time.Duration
	filterLivenessPeriod time.Duration

	client func() ContextCaller

	latestBlockChangedEvent        *latestBlockChangedEvent
	transactionSentToUpstreamEvent *transactionSentToUpstreamEvent
}

// NewPublicAPI returns a reference to the PublicAPI object
func NewPublicAPI(s *Service) *PublicAPI {
	api := &PublicAPI{
		filters:                        make(map[rpc.ID]filter),
		latestBlockChangedEvent:        s.latestBlockChangedEvent,
		transactionSentToUpstreamEvent: s.transactionSentToUpstreamEvent,
		client:               func() ContextCaller { return s.rpc.RPCClient() },
		filterLivenessLoop:   filterLivenessPeriod,
		filterLivenessPeriod: filterLivenessPeriod + 10*time.Second,
	}
	go func() {
		api.timeoutLoop(s.quit)
	}()
	return api
}

func (api *PublicAPI) timeoutLoop(quit chan struct{}) {
	for {
		select {
		case <-quit:
			return
		default:
		}
		time.Sleep(api.filterLivenessLoop)
		api.filtersMu.Lock()
		for id, f := range api.filters {
			deadline := f.deadline()
			if deadline == nil {
				continue
			}
			select {
			case <-deadline.C:
				delete(api.filters, id)
				f.stop()
			default:
				continue
			}
		}
		api.filtersMu.Unlock()
	}
}

func (api *PublicAPI) NewFilter(crit filters.FilterCriteria) (rpc.ID, error) {
	id := rpc.ID(uuid.New())
	ctx, cancel := context.WithCancel(context.Background())
	f := &logsFilter{
		id:     id,
		crit:   ethereum.FilterQuery(crit),
		done:   make(chan struct{}),
		timer:  time.NewTimer(api.filterLivenessPeriod),
		ctx:    ctx,
		cancel: cancel,
	}
	api.filtersMu.Lock()
	api.filters[id] = f
	api.filtersMu.Unlock()

	go func() {
		pollLogs(api.client(), f, logsQueryTimeout, logsPeriod)
	}()
	return id, nil
}

// NewBlockFilter is an implemenation of `eth_newBlockFilter` API
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (api *PublicAPI) NewBlockFilter() rpc.ID {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	f := newHashFilter()
	id := rpc.ID(uuid.New())

	api.filters[id] = f

	go func() {
		id, s := api.latestBlockChangedEvent.Subscribe()
		defer api.latestBlockChangedEvent.Unsubscribe(id)

		for {
			select {
			case hash := <-s:
				f.add(hash)
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

	f := newHashFilter()
	id := rpc.ID(uuid.New())

	api.filters[id] = f

	go func() {
		id, s := api.transactionSentToUpstreamEvent.Subscribe()
		defer api.transactionSentToUpstreamEvent.Unsubscribe(id)

		for {
			select {
			case hash := <-s:
				f.add(hash)
			case <-f.done:
				return
			}
		}

	}()

	return id

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
		f.stop()
	}

	return found
}

// GetFilterChanges returns the hashes for the filter with the given id since
// last time it was called. This can be used for polling.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (api *PublicAPI) GetFilterChanges(id rpc.ID) (interface{}, error) {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	if f, found := api.filters[id]; found {
		deadline := f.deadline()
		if deadline != nil {
			if !deadline.Stop() {
				// timer expired but filter is not yet removed in timeout loop
				// receive timer value and reset timer
				<-deadline.C
			}
			deadline.Reset(api.filterLivenessPeriod)
		}
		return f.pop(), nil
	}
	return []interface{}{}, errors.New("filter not found")
}

// returnHashes is a helper that will return an empty hash array case the given hash array is nil,
// otherwise the given hashes array is returned.
func returnHashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}
