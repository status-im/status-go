package rpcfilters

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type callTracker struct {
	mu       sync.Mutex
	calls    int
	reply    [][]types.Log
	criteria []map[string]interface{}
}

func (c *callTracker) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if len(args) != 1 {
		return errors.New("unexpected length of args")
	}
	crit := args[0].(map[string]interface{})
	c.criteria = append(c.criteria, crit)
	select {
	case <-ctx.Done():
		return errors.New("context canceled")
	default:
	}
	if c.calls <= len(c.reply) {
		rst := result.(*[]types.Log)
		*rst = c.reply[c.calls-1]
	}
	return nil
}

func runLogsFetcherTest(t *testing.T, f *logsFilter, replies [][]types.Log, queries int) *callTracker {
	c := callTracker{reply: replies}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		pollLogs(&c, f, time.Second, 100*time.Millisecond)
		wg.Done()
	}()
	tick := time.Tick(10 * time.Millisecond)
	after := time.After(time.Second)
	func() {
		for {
			select {
			case <-after:
				f.stop()
				assert.FailNow(t, "failed waiting for requests")
				return
			case <-tick:
				c.mu.Lock()
				num := c.calls
				c.mu.Unlock()
				if num >= queries {
					f.stop()
					return
				}
			}
		}
	}()
	wg.Wait()
	require.Len(t, c.criteria, queries)
	return &c
}

func TestLogsFetcherAdjusted(t *testing.T) {
	f := &logsFilter{
		ctx: context.TODO(),
		crit: ethereum.FilterQuery{
			FromBlock: big.NewInt(10),
		},
		done:      make(chan struct{}),
		logsCache: newCache(defaultCacheSize),
	}
	logs := []types.Log{
		{BlockNumber: 11}, {BlockNumber: 12},
	}
	c := runLogsFetcherTest(t, f, [][]types.Log{logs}, 2)
	require.Equal(t, hexutil.EncodeBig(big.NewInt(10)), c.criteria[0]["fromBlock"])
	require.Equal(t, c.criteria[1]["fromBlock"], "latest")
}

func TestAdjustedDueToReorg(t *testing.T) {
	f := &logsFilter{
		ctx: context.TODO(),
		crit: ethereum.FilterQuery{
			FromBlock: big.NewInt(10),
		},
		done:      make(chan struct{}),
		logsCache: newCache(defaultCacheSize),
	}
	logs := []types.Log{
		{BlockNumber: 11, BlockHash: common.Hash{1}}, {BlockNumber: 12, BlockHash: common.Hash{2}},
	}
	reorg := []types.Log{
		{BlockNumber: 12, BlockHash: common.Hash{2, 2}},
	}
	c := runLogsFetcherTest(t, f, [][]types.Log{logs, reorg}, 3)
	require.Equal(t, hexutil.EncodeBig(big.NewInt(10)), c.criteria[0]["fromBlock"])
	require.Equal(t, "latest", c.criteria[1]["fromBlock"])
	require.Equal(t, hexutil.EncodeBig(big.NewInt(11)), c.criteria[2]["fromBlock"])
}

func TestLogsFetcherCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &logsFilter{
		ctx: ctx,
		crit: ethereum.FilterQuery{
			FromBlock: big.NewInt(10),
		},
		done:      make(chan struct{}),
		logsCache: newCache(defaultCacheSize),
	}
	cancel()
	c := runLogsFetcherTest(t, f, [][]types.Log{make([]types.Log, 2)}, 2)
	require.Equal(t, hexutil.EncodeBig(big.NewInt(10)), c.criteria[0]["fromBlock"])
	require.Equal(t, hexutil.EncodeBig(big.NewInt(10)), c.criteria[1]["fromBlock"])
}
