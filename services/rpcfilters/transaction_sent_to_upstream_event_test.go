package rpcfilters

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var transactionHashes = []common.Hash{common.HexToHash("0xAA"), common.HexToHash("0xBB"), common.HexToHash("0xCC")}

func TestTransactionSentToUpstreamEventSubscribe(t *testing.T) {
	event := newTransactionSentToUpstreamEvent()
	assert.NoError(t, event.Start())
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
	assert.NoError(t, event.Start())
	defer event.Stop()

	subscriptionChannels := [](chan common.Hash){}
	for i := 0; i < 3; i++ {
		id, channel := event.Subscribe()
		// test id assignment
		assert.Equal(t, i, id)
		// test numberOfSubscriptions
		assert.Equal(t, event.numberOfSubscriptions(), i+1)
		subscriptionChannels = append(subscriptionChannels, channel)
	}

	done := make(chan struct{})
	go func() {
		for i, channel := range subscriptionChannels {
			for _, expectedHash := range transactionHashes {
				select {
				case receivedHash := <-channel:
					assert.Equal(t, expectedHash, receivedHash)
				case <-time.After(1 * time.Second):
					assert.Fail(t, "timeout")
				}
			}
			if i == len(subscriptionChannels)-1 {
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
