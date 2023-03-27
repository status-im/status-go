package filterv2

import (
	"sync"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type SubscriptionDetails struct {
	sync.RWMutex

	id     string
	mapRef *SubscriptionsMap
	closed bool
	once   sync.Once

	peerID        peer.ID
	pubsubTopic   string
	contentTopics map[string]struct{}
	C             chan *protocol.Envelope
}

type SubscriptionSet map[string]*SubscriptionDetails

type PeerSubscription struct {
	peerID                peer.ID
	subscriptionsPerTopic map[string]SubscriptionSet
}

type SubscriptionsMap struct {
	sync.RWMutex
	items map[peer.ID]*PeerSubscription
}

func NewSubscriptionMap() *SubscriptionsMap {
	return &SubscriptionsMap{
		items: make(map[peer.ID]*PeerSubscription),
	}
}

func (sub *SubscriptionsMap) NewSubscription(peerID peer.ID, topic string, contentTopics []string) *SubscriptionDetails {
	sub.Lock()
	defer sub.Unlock()

	peerSubscription, ok := sub.items[peerID]
	if !ok {
		peerSubscription = &PeerSubscription{
			peerID:                peerID,
			subscriptionsPerTopic: make(map[string]SubscriptionSet),
		}
		sub.items[peerID] = peerSubscription
	}

	_, ok = peerSubscription.subscriptionsPerTopic[topic]
	if !ok {
		peerSubscription.subscriptionsPerTopic[topic] = make(SubscriptionSet)
	}

	details := &SubscriptionDetails{
		id:            uuid.NewString(),
		mapRef:        sub,
		peerID:        peerID,
		pubsubTopic:   topic,
		C:             make(chan *protocol.Envelope),
		contentTopics: make(map[string]struct{}),
	}

	for _, ct := range contentTopics {
		details.contentTopics[ct] = struct{}{}
	}

	sub.items[peerID].subscriptionsPerTopic[topic][details.id] = details

	return details
}

func (sub *SubscriptionsMap) Has(peerID peer.ID, topic string, contentTopics []string) bool {
	// Check if peer exits
	peerSubscription, ok := sub.items[peerID]
	if !ok {
		return false
	}

	// Check if pubsub topic exists
	subscriptions, ok := peerSubscription.subscriptionsPerTopic[topic]
	if !ok {
		return false
	}

	// Check if the content topic exists within the list of subscriptions for this peer
	for _, ct := range contentTopics {
		found := false
		for _, subscription := range subscriptions {
			_, exists := subscription.contentTopics[ct]
			if exists {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (sub *SubscriptionsMap) Delete(subscription *SubscriptionDetails) error {
	sub.Lock()
	defer sub.Unlock()

	peerSubscription, ok := sub.items[subscription.peerID]
	if !ok {
		return ErrNotFound
	}

	delete(peerSubscription.subscriptionsPerTopic[subscription.pubsubTopic], subscription.id)

	return nil
}

func (s *SubscriptionDetails) Add(contentTopics ...string) {
	s.Lock()
	defer s.Unlock()

	for _, ct := range contentTopics {
		s.contentTopics[ct] = struct{}{}
	}
}

func (s *SubscriptionDetails) Remove(contentTopics ...string) {
	s.Lock()
	defer s.Unlock()

	for _, ct := range contentTopics {
		delete(s.contentTopics, ct)
	}
}

func (s *SubscriptionDetails) closeC() {
	s.once.Do(func() {
		s.Lock()
		defer s.Unlock()

		s.closed = true
		close(s.C)
	})
}

func (s *SubscriptionDetails) Close() error {
	s.closeC()
	return s.mapRef.Delete(s)
}

func (s *SubscriptionDetails) Clone() *SubscriptionDetails {
	s.RLock()
	defer s.RUnlock()

	result := &SubscriptionDetails{
		id:            uuid.NewString(),
		mapRef:        s.mapRef,
		closed:        false,
		peerID:        s.peerID,
		pubsubTopic:   s.pubsubTopic,
		contentTopics: make(map[string]struct{}),
		C:             make(chan *protocol.Envelope),
	}

	for k := range s.contentTopics {
		result.contentTopics[k] = struct{}{}
	}

	return result
}

func (sub *SubscriptionsMap) clear() {
	for _, peerSubscription := range sub.items {
		for _, subscriptionSet := range peerSubscription.subscriptionsPerTopic {
			for _, subscription := range subscriptionSet {
				subscription.closeC()
			}
		}
	}

	sub.items = make(map[peer.ID]*PeerSubscription)
}

func (sub *SubscriptionsMap) Clear() {
	sub.Lock()
	defer sub.Unlock()
	sub.clear()
}

func (sub *SubscriptionsMap) Notify(peerID peer.ID, envelope *protocol.Envelope) {
	sub.RLock()
	defer sub.RUnlock()

	subscriptions, ok := sub.items[peerID].subscriptionsPerTopic[envelope.PubsubTopic()]
	if ok {
		iterateSubscriptionSet(subscriptions, envelope)
	}
}

func iterateSubscriptionSet(subscriptions SubscriptionSet, envelope *protocol.Envelope) {
	for _, subscription := range subscriptions {
		func(subscription *SubscriptionDetails) {
			subscription.RLock()
			defer subscription.RUnlock()

			_, ok := subscription.contentTopics[envelope.Message().ContentTopic]
			if !ok && len(subscription.contentTopics) != 0 { // TODO: confirm if no content topics are allowed
				return
			}

			if !subscription.closed {
				// TODO: consider pushing or dropping if subscription is not available
				subscription.C <- envelope
			}
		}(subscription)
	}
}
