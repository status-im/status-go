package rpcfilters

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterLiveness(t *testing.T) {
	api := &PublicAPI{
		filters:              make(map[rpc.ID]filter),
		filterLivenessLoop:   10 * time.Millisecond,
		filterLivenessPeriod: 15 * time.Millisecond,
		client:               func() ContextCaller { return &callTracker{} },
	}
	id, err := api.NewFilter(filters.FilterCriteria{})
	require.NoError(t, err)
	quit := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		api.timeoutLoop(quit)
		wg.Done()
	}()
	tick := time.Tick(10 * time.Millisecond)
	after := time.After(100 * time.Millisecond)
	func() {
		for {
			select {
			case <-after:
				assert.FailNow(t, "filter wasn't removed")
				close(quit)
				return
			case <-tick:
				api.filtersMu.Lock()
				_, exist := api.filters[id]
				api.filtersMu.Unlock()
				if !exist {
					close(quit)
					return
				}
			}
		}
	}()
	wg.Wait()
}

func TestGetFilterChangesResetsTimer(t *testing.T) {
	api := &PublicAPI{
		filters:              make(map[rpc.ID]filter),
		filterLivenessLoop:   10 * time.Millisecond,
		filterLivenessPeriod: 15 * time.Millisecond,
		client:               func() ContextCaller { return &callTracker{} },
	}
	id, err := api.NewFilter(filters.FilterCriteria{})
	require.NoError(t, err)

	api.filtersMu.Lock()
	f := api.filters[id]
	require.True(t, f.deadline().Stop())
	fake := make(chan time.Time, 1)
	fake <- time.Time{}
	f.deadline().C = fake
	api.filtersMu.Unlock()

	require.False(t, f.deadline().Stop())
	// GetFilterChanges will Reset deadline
	_, err = api.GetFilterChanges(id)
	require.NoError(t, err)
	require.True(t, f.deadline().Stop())
}

func TestGetFilterLogs(t *testing.T) {
	t.Skip("Skipping due to flakiness: https://github.com/status-im/status-go/issues/1281")

	tracker := new(callTracker)
	api := &PublicAPI{
		filters: make(map[rpc.ID]filter),
		client:  func() ContextCaller { return tracker },
	}
	block := big.NewInt(10)
	id, err := api.NewFilter(filters.FilterCriteria{
		FromBlock: block,
	})
	require.NoError(t, err)
	logs, err := api.GetFilterLogs(context.TODO(), id)
	require.NoError(t, err)
	require.Empty(t, logs)
	require.Len(t, tracker.criterias, 1)
	rst, err := hexutil.DecodeBig(tracker.criterias[0]["fromBlock"].(string))
	require.NoError(t, err)
	require.Equal(t, block, rst)
}
