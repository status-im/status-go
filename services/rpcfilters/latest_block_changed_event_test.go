package rpcfilters

import (
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type latestBlockProviderTest struct {
	BlockFunc func() (blockInfo, error)
}

func (p latestBlockProviderTest) GetLatestBlock() (blockInfo, error) {
	return p.BlockFunc()
}

func TestEventSubscribe(t *testing.T) {
	counter := 0

	hashes := []common.Hash{common.HexToHash("0xAA"), common.HexToHash("0xBB"), common.HexToHash("0xCC")}

	f := func() (blockInfo, error) {
		counter++
		number := big.NewInt(int64(counter))
		if counter > len(hashes) {
			counter = len(hashes)
		}
		return blockInfo{hashes[counter-1], hexutil.Bytes(number.Bytes())}, nil
	}

	testEventSubscribe(t, f, hashes)
}

func TestZeroSubsciptionsOptimization(t *testing.T) {
	counter := int64(0)
	hash := common.HexToHash("0xFF")

	f := func() (blockInfo, error) {
		atomic.AddInt64(&counter, 1)
		number := big.NewInt(1)
		return blockInfo{hash, hexutil.Bytes(number.Bytes())}, nil
	}

	event := newLatestBlockChangedEvent(latestBlockProviderTest{f})
	event.tickerPeriod = time.Millisecond

	assert.NoError(t, event.Start())
	defer event.Stop()

	// let the ticker to call ~10 times
	time.Sleep(10 * time.Millisecond)

	// check that our provider function wasn't called when there are no subscribers to it
	assert.Equal(t, int64(0), atomic.LoadInt64(&counter))

	// subscribing an event, checking that it works
	id, channelInterface := event.Subscribe()
	channel, ok := channelInterface.(chan common.Hash)
	assert.True(t, ok)

	timeout := time.After(1 * time.Second)
	select {
	case receivedHash := <-channel:
		assert.Equal(t, hash, receivedHash)
	case <-timeout:
		assert.Fail(t, "timeout")
	}

	event.Unsubscribe(id)

	// check that our function was called multiple times
	counterValue := atomic.LoadInt64(&counter)
	assert.True(t, counterValue > 0)

	// let the ticker to call ~10 times
	time.Sleep(10 * time.Millisecond)

	// check that our provider function wasn't called when there are no subscribers to it
	assert.Equal(t, counterValue, atomic.LoadInt64(&counter))
}

func TestMultipleSubscribe(t *testing.T) {
	hash := common.HexToHash("0xFF")

	f := func() (blockInfo, error) {
		number := big.NewInt(1)
		return blockInfo{hash, hexutil.Bytes(number.Bytes())}, nil
	}

	event := newLatestBlockChangedEvent(latestBlockProviderTest{f})
	event.tickerPeriod = time.Millisecond

	wg := sync.WaitGroup{}

	testFunc := func() {
		testEvent(t, event, []common.Hash{hash})
		wg.Done()
	}

	numberOfSubscriptions := 3

	wg.Add(numberOfSubscriptions)
	for i := 0; i < numberOfSubscriptions; i++ {
		go testFunc()
	}

	assert.NoError(t, event.Start())
	defer event.Stop()

	wg.Wait()

	assert.Equal(t, 0, len(event.sx))
}

func testEventSubscribe(t *testing.T, f func() (blockInfo, error), expectedHashes []common.Hash) {
	event := newLatestBlockChangedEvent(latestBlockProviderTest{f})
	event.tickerPeriod = time.Millisecond

	assert.NoError(t, event.Start())
	defer event.Stop()

	testEvent(t, event, expectedHashes)
}

func testEvent(t *testing.T, event *latestBlockChangedEvent, expectedHashes []common.Hash) {
	id, channelInterface := event.Subscribe()
	channel, ok := channelInterface.(chan common.Hash)
	assert.True(t, ok)

	timeout := time.After(1 * time.Second)

	for _, hash := range expectedHashes {
		select {
		case receivedHash := <-channel:
			assert.Equal(t, hash, receivedHash)
		case <-timeout:
			assert.Fail(t, "timeout")
		}
	}

	event.Unsubscribe(id)

}

func TestEventReceivedBlocksOutOfOrders(t *testing.T) {
	// We are sending blocks out of order (simulating load balancing on RPC
	// nodes). Note that hashes are the same.
	// We should still receive them in order and not have the event
	// fired for out-of-order events.
	expectedHashes := []common.Hash{common.HexToHash("0xAA"), common.HexToHash("0xBB"), common.HexToHash("0xCC")}
	sentHashes := []common.Hash{common.HexToHash("0xAA"), common.HexToHash("0xBB"), common.HexToHash("0xAA"), common.HexToHash("0xCC")}
	sentBlockNumbers := []int64{1, 2, 1, 3}

	counter := 0
	f := func() (blockInfo, error) {
		counter++
		if counter > len(sentHashes) {
			counter = len(sentHashes)
		}
		number := big.NewInt(sentBlockNumbers[counter-1])
		return blockInfo{sentHashes[counter-1], hexutil.Bytes(number.Bytes())}, nil
	}

	testEventSubscribe(t, f, expectedHashes)
}

func TestEventDivergedChain(t *testing.T) {
	// We are sending blocks out of order (simulating chain diverges).
	// Note that every hash is unique. We should still receive them all.
	hashes := []common.Hash{common.HexToHash("0xC11"), common.HexToHash("0xC12"), common.HexToHash("0xC21"), common.HexToHash("0xC22"), common.HexToHash("0xC23")}
	blockNumbers := []int64{1, 2, 1, 2, 3}

	counter := 0
	f := func() (blockInfo, error) {
		counter++
		if counter > len(hashes) {
			counter = len(hashes)
		}
		number := big.NewInt(blockNumbers[counter-1])
		return blockInfo{hashes[counter-1], hexutil.Bytes(number.Bytes())}, nil
	}

	testEventSubscribe(t, f, hashes)
}
