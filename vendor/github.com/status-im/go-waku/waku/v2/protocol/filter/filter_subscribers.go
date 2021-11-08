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

func (self *Subscribers) Append(s Subscriber) int {
	self.Lock()
	defer self.Unlock()

	self.subscribers = append(self.subscribers, s)
	return len(self.subscribers)
}

func (self *Subscribers) Items() <-chan Subscriber {
	c := make(chan Subscriber)

	f := func() {
		self.RLock()
		defer self.RUnlock()
		for _, value := range self.subscribers {
			c <- value
		}
		close(c)
	}
	go f()

	return c
}

func (self *Subscribers) Length() int {
	self.RLock()
	defer self.RUnlock()

	return len(self.subscribers)
}

func (self *Subscribers) RemoveContentFilters(peerID peer.ID, contentFilters []*pb.FilterRequest_ContentFilter) {
	var peerIdsToRemove []peer.ID

	for _, subscriber := range self.subscribers {
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
		for i, s := range self.subscribers {
			if s.peer == peerId {
				l := len(self.subscribers) - 1
				self.subscribers[l], self.subscribers[i] = self.subscribers[i], self.subscribers[l]
				self.subscribers = self.subscribers[:l]
				break
			}
		}
	}
}
