package rpcfilters

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

var transactionInfos = []*PendingTxInfo{
	{
		Hash:    common.HexToHash("0xAA"),
		Type:    "RegisterENS",
		From:    common.Address{1},
		ChainID: 0,
	},
	{
		Hash:    common.HexToHash("0xBB"),
		Type:    "WalletTransfer",
		ChainID: 1,
	},
	{
		Hash:    common.HexToHash("0xCC"),
		Type:    "SetPubKey",
		From:    common.Address{3},
		ChainID: 2,
	},
}

func TestTransactionSentToUpstreamEventMultipleSubscribe(t *testing.T) {
	event := newTransactionSentToUpstreamEvent()
	require.NoError(t, event.Start())
	defer event.Stop()

	var subscriptionChannels []chan *PendingTxInfo
	for i := 0; i < 3; i++ {
		id, channelInterface := event.Subscribe()
		channel, ok := channelInterface.(chan *PendingTxInfo)
		require.True(t, ok)
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
				for _, expectedTxInfo := range transactionInfos {
					select {
					case receivedTxInfo := <-ch:
						require.True(t, reflect.DeepEqual(expectedTxInfo, receivedTxInfo))
					case <-time.After(1 * time.Second):
						assert.Fail(t, "timeout")
					}
					wg.Done()
				}
			}()
		}
	}()

	for _, txInfo := range transactionInfos {
		event.Trigger(txInfo)
	}
	wg.Wait()
}
