package rpcfilters

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var transactionHashes = []common.Hash{common.HexToHash("0xAA"), common.HexToHash("0xBB"), common.HexToHash("0xCC")}

func TestTransactionSentToUpstreamEventSubscribe(t *testing.T) {
	event := newTransactionSentToUpstreamEvent()
	require.NoError(t, event.Start())
	defer event.Stop()

	_, channel := event.Subscribe()

	done := make(chan struct{})
	go func() {
		for i, expectedHash := range transactionHashes {
			select {
			case receivedHash := <-channel:
				assert.Equal(t, expectedHash, receivedHash)
			case <-time.After(1 * time.Second):
				assert.Fail(t, "timeout")
			}
			if i == len(transactionHashes)-1 {
				close(done)
			}
		}
	}()

	for _, hashToTrigger := range transactionHashes {
		// sleep in order to ensure those hashes come in order
		time.Sleep(10 * time.Millisecond)
		event.Trigger(hashToTrigger)
	}
	<-done
}

func TestTransactionSentToUpstreamEventMultipleSubscribe(t *testing.T) {
	event := newTransactionSentToUpstreamEvent()
	require.NoError(t, event.Start())
	defer event.Stop()

	var subscriptionChannels []chan common.Hash
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
