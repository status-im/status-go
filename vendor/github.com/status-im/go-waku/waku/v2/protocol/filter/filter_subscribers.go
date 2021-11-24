package filter

import (
	"sync"

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
}

func NewSubscribers() *Subscribers {
	return &Subscribers{}
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

func (sub *Subscribers) RemoveContentFilters(peerID peer.ID, contentFilters []*pb.FilterRequest_ContentFilter) {
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
