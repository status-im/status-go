package rpcfilters

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
)

var transactionHashes = []types.Hash{types.HexToHash("0xAA"), types.HexToHash("0xBB"), types.HexToHash("0xCC")}

func TestTransactionSentToUpstreamEventMultipleSubscribe(t *testing.T) {
	event := newTransactionSentToUpstreamEvent()
	require.NoError(t, event.Start())
	defer event.Stop()

	var subscriptionChannels []chan types.Hash
	for i := 0; i < 3; i++ {
		id, channel := event.Subscribe()
		// test id assignment
		require.Equal(t, i, id)
		// test numberOfSubscriptions
		require.Equal(t, event.numberOfSubscriptions(), i+1)
		subscriptionChannels = append(subscriptionChannels, channel)
	}

	var wg sync.WaitGroup

	wg.Add(9)
	go func() {
		for _, channel := range subscriptionChannels {
			ch := channel
			go func() {
				for _, expectedHash := range transactionHashes {
					select {
					case receivedHash := <-ch:
						require.Equal(t, expectedHash, receivedHash)
					case <-time.After(1 * time.Second):
						assert.Fail(t, "timeout")
					}
					wg.Done()
				}
			}()
		}
	}()

	for _, hashToTrigger := range transactionHashes {
		event.Trigger(hashToTrigger)
	}
	wg.Wait()
}
