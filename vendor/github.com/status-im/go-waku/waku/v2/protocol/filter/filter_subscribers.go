package filter

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

type Subscriber struct {
	peer      peer.ID
	requestId string
	filter    pb.FilterRequest // @TODO MAKE THIS A SEQUENCE AGAIN?
}

type Subscribers struct {
	sync.RWMutex
	subscribers []Subscriber
	timeout     time.Duration
	failedPeers map[peer.ID]time.Time
}

func NewSubscribers(timeout time.Duration) *Subscribers {
	return &Subscribers{
		timeout:     timeout,
		failedPeers: make(map[peer.ID]time.Time),
	}
}

func (sub *Subscribers) Append(s Subscriber) int {
	sub.Lock()
	defer sub.Unlock()

	sub.subscribers = append(sub.subscribers, s)
	return len(sub.subscribers)
}

func (sub *Subscribers) Items() <-chan Subscriber {
	c := make(chan Subscriber)

	f := func() {
		sub.RLock()
		defer sub.RUnlock()
		for _, value := range sub.subscribers {
			c <- value
		}
		close(c)
	}
	go f()

	return c
}

func (sub *Subscribers) Length() int {
	sub.RLock()
	defer sub.RUnlock()

	return len(sub.subscribers)
}

func (sub *Subscribers) FlagAsSuccess(peerID peer.ID) {
	sub.Lock()
	defer sub.Unlock()

	_, ok := sub.failedPeers[peerID]
	if ok {
		delete(sub.failedPeers, peerID)
	}
}

func (sub *Subscribers) FlagAsFailure(peerID peer.ID) {
	sub.Lock()
	defer sub.Unlock()

	lastFailure, ok := sub.failedPeers[peerID]
	if ok {
		elapsedTime := time.Since(lastFailure)
		if elapsedTime > sub.timeout {
			var tmpSubs []Subscriber
			for _, s := range sub.subscribers {
				if s.peer != peerID {
					tmpSubs = append(tmpSubs, s)
				}
			}
			sub.subscribers = tmpSubs

			delete(sub.failedPeers, peerID)
		}
	} else {
		sub.failedPeers[peerID] = time.Now()
	}
}

func (sub *Subscribers) RemoveContentFilters(peerID peer.ID, contentFilters []*pb.FilterRequest_ContentFilter) {
	sub.Lock()
	defer sub.Unlock()

	var peerIdsToRemove []peer.ID

	for _, subscriber := range sub.subscribers {
		if subscriber.peer != peerID {
			continue
		}

		// make sure we delete the content filter
		// if no more topics are left
		for i, contentFilter := range contentFilters {
			subCfs := subscriber.filter.ContentFilters
			for _, cf := range subCfs {
				if cf.ContentTopic == contentFilter.ContentTopic {
					l := len(subCfs) - 1
					subCfs[l], subCfs[i] = subCfs[i], subCfs[l]
					subscriber.filter.ContentFilters = subCfs[:l]
				}
			}
		}

		if len(subscriber.filter.ContentFilters) == 0 {
			peerIdsToRemove = append(peerIdsToRemove, subscriber.peer)
		}
	}

	// make sure we delete the subscriber
	// if no more content filters left
	for _, peerId := range peerIdsToRemove {
		for i, s := range sub.subscribers {
			if s.peer == peerId {
				l := len(sub.subscribers) - 1
				sub.subscribers[l], sub.subscribers[i] = sub.subscribers[i], sub.subscribers[l]
				sub.subscribers = sub.subscribers[:l]
				break
			}
		}
	}
}
