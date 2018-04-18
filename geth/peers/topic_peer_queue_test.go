package peers

import (
	"container/heap"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/mclock"
)

func TestPeerPriorityQueueSorting(t *testing.T) {
	count := 5
	discTimes := make([]mclock.AbsTime, count)

	// generate a slice of monotonic times
	for i := 0; i < count; i++ {
		discTimes[i] = mclock.Now()
	}

	// shuffle discTimes
	for i := range discTimes {
		j := rand.Intn(i + 1)
		discTimes[i], discTimes[j] = discTimes[j], discTimes[i]
	}

	// make a priority queue
	q := make(peerPriorityQueue, count)
	for i := 0; i < count; i++ {
		q[i] = &peerInfoItem{
			peerInfo: &peerInfo{
				discoveredTime: discTimes[i],
			},
		}
	}
	heap.Init(&q)

	// verify that the slice is sorted ascending by `discoveredTime`
	var item *peerInfoItem
	for q.Len() > 0 {
		newItem := heap.Pop(&q).(*peerInfoItem)
		if item != nil {
			require.True(t, item.discoveredTime < newItem.discoveredTime)
		}
		item = newItem
	}
}

func TestPeerPriorityQueueIndexUpdating(t *testing.T) {
	q := make(peerPriorityQueue, 0)
	heap.Init(&q)

	item1 := &peerInfoItem{
		index: -1,
		peerInfo: &peerInfo{
			discoveredTime: mclock.Now(),
		},
	}
	item2 := &peerInfoItem{
		index: -1,
		peerInfo: &peerInfo{
			discoveredTime: mclock.Now(),
		},
	}

	// insert older item first
	heap.Push(&q, item2)
	require.Equal(t, item2.index, 0)
	heap.Push(&q, item1)
	require.Equal(t, item1.index, 0)
	require.Equal(t, item2.index, 1)

	// poping should reset index
	popedItem := heap.Pop(&q)
	require.Equal(t, item1, popedItem)
	require.Equal(t, item1.index, -1)
	require.Equal(t, item2.index, 0)
}
