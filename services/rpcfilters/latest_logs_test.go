package rpcfilters

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type callTracker struct {
	mu        sync.Mutex
	calls     int
	reply     [][]types.Log
	criterias []map[string]interface{}
}

func (c *callTracker) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if len(args) != 1 {
		return errors.New("unexpected length of args")
	}
	crit := args[0].(map[string]interface{})
	c.criterias = append(c.criterias, crit)
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

func runLogsFetcherTest(t *testing.T, f *logsFilter) *callTracker {
	c := callTracker{reply: [][]types.Log{
		make([]types.Log, 2),
	}}
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
				if num >= 2 {
					f.stop()
					return
				}
			}
		}
	}()
	wg.Wait()
	require.Len(t, c.criterias, 2)
	return &c
}

func TestLogsFetcherAdjusted(t *testing.T) {
	f := &logsFilter{
		ctx: context.TODO(),
		crit: ethereum.FilterQuery{
			FromBlock: big.NewInt(10),
		},
		done: make(chan struct{}),
	}
	c := runLogsFetcherTest(t, f)
	require.Equal(t, hexutil.EncodeBig(big.NewInt(10)), c.criterias[0]["fromBlock"])
	require.Equal(t, c.criterias[1]["fromBlock"], "latest")
}

func TestLogsFetcherCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &logsFilter{
		ctx: ctx,
		crit: ethereum.FilterQuery{
			FromBlock: big.NewInt(10),
		},
		done: make(chan struct{}),
	}
	cancel()
	c := runLogsFetcherTest(t, f)
	require.Equal(t, hexutil.EncodeBig(big.NewInt(10)), c.criterias[0]["fromBlock"])
	require.Equal(t, hexutil.EncodeBig(big.NewInt(10)), c.criterias[1]["fromBlock"])
}

func TestAdjustFromBlock(t *testing.T) {
	type testCase struct {
		description string
		initial     ethereum.FilterQuery
		result      ethereum.FilterQuery
	}

	for _, tc := range []testCase{
		{
			"ToBlockHigherThenLatest",
			ethereum.FilterQuery{ToBlock: big.NewInt(10)},
			ethereum.FilterQuery{ToBlock: big.NewInt(10)},
		},
		{
			"FromBlockIsPending",
			ethereum.FilterQuery{FromBlock: big.NewInt(-2)},
			ethereum.FilterQuery{FromBlock: big.NewInt(-2)},
		},
		{
			"FromBlockIsOlderThenLatest",
			ethereum.FilterQuery{FromBlock: big.NewInt(10)},
			ethereum.FilterQuery{FromBlock: big.NewInt(-1)},
		},
		{
			"NotInterestedInLatestBlocks",
			ethereum.FilterQuery{FromBlock: big.NewInt(10), ToBlock: big.NewInt(15)},
			ethereum.FilterQuery{FromBlock: big.NewInt(10), ToBlock: big.NewInt(15)},
		},
	} {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			adjustFromBlock(&tc.initial)
			require.Equal(t, tc.result, tc.initial)
		})
	}
}
