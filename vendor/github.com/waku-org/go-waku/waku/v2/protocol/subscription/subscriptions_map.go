package subscription

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

type SubscriptionDetails struct {
	sync.RWMutex

	ID     string
	mapRef *SubscriptionsMap
	Closed bool
	once   sync.Once

	PeerID        peer.ID
	ContentFilter protocol.ContentFilter
	C             chan *protocol.Envelope
}

// Map of SubscriptionDetails.ID to subscriptions
type SubscriptionSet map[string]*SubscriptionDetails

type PeerSubscription struct {
	PeerID             peer.ID
	SubsPerPubsubTopic map[string]SubscriptionSet
}

type SubscriptionsMap struct {
	sync.RWMutex
	logger *zap.Logger
	Items  map[peer.ID]*PeerSubscription
}

var ErrNotFound = errors.New("not found")

func NewSubscriptionMap(logger *zap.Logger) *SubscriptionsMap {
	return &SubscriptionsMap{
		logger: logger.Named("subscription-map"),
		Items:  make(map[peer.ID]*PeerSubscription),
	}
}

func (sub *SubscriptionsMap) NewSubscription(peerID peer.ID, cf protocol.ContentFilter) *SubscriptionDetails {
	sub.Lock()
	defer sub.Unlock()

	peerSubscription, ok := sub.Items[peerID]
	if !ok {
		peerSubscription = &PeerSubscription{
			PeerID:             peerID,
			SubsPerPubsubTopic: make(map[string]SubscriptionSet),
		}
		sub.Items[peerID] = peerSubscription
	}

	_, ok = peerSubscription.SubsPerPubsubTopic[cf.PubsubTopic]
	if !ok {
		peerSubscription.SubsPerPubsubTopic[cf.PubsubTopic] = make(SubscriptionSet)
	}

	details := &SubscriptionDetails{
		ID:            uuid.NewString(),
		mapRef:        sub,
		PeerID:        peerID,
		C:             make(chan *protocol.Envelope, 1024),
		ContentFilter: protocol.ContentFilter{PubsubTopic: cf.PubsubTopic, ContentTopics: maps.Clone(cf.ContentTopics)},
	}

	sub.Items[peerID].SubsPerPubsubTopic[cf.PubsubTopic][details.ID] = details

	return details
}

func (sub *SubscriptionsMap) IsSubscribedTo(peerID peer.ID) bool {
	sub.RLock()
	defer sub.RUnlock()

	_, ok := sub.Items[peerID]
	return ok
}

// Check if we have subscriptions for all (pubsubTopic, contentTopics[i]) pairs provided
func (sub *SubscriptionsMap) Has(peerID peer.ID, cf protocol.ContentFilter) bool {
	sub.RLock()
	defer sub.RUnlock()

	// Check if peer exits
	peerSubscription, ok := sub.Items[peerID]
	if !ok {
		return false
	}
	//TODO: Handle pubsubTopic as null
	// Check if pubsub topic exists
	subscriptions, ok := peerSubscription.SubsPerPubsubTopic[cf.PubsubTopic]
	if !ok {
		return false
	}

	// Check if the content topic exists within the list of subscriptions for this peer
	for _, ct := range cf.ContentTopicsList() {
		found := false
		for _, subscription := range subscriptions {
			_, exists := subscription.ContentFilter.ContentTopics[ct]
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

	peerSubscription, ok := sub.Items[subscription.PeerID]
	if !ok {
		return ErrNotFound
	}

	delete(peerSubscription.SubsPerPubsubTopic[subscription.ContentFilter.PubsubTopic], subscription.ID)

	return nil
}

func (s *SubscriptionDetails) Add(contentTopics ...string) {
	s.Lock()
	defer s.Unlock()

	for _, ct := range contentTopics {
		s.ContentFilter.ContentTopics[ct] = struct{}{}
	}
}

func (s *SubscriptionDetails) Remove(contentTopics ...string) {
	s.Lock()
	defer s.Unlock()

	for _, ct := range contentTopics {
		delete(s.ContentFilter.ContentTopics, ct)
	}
}

func (s *SubscriptionDetails) CloseC() {
	s.once.Do(func() {
		s.Lock()
		defer s.Unlock()

		s.Closed = true
		close(s.C)
	})
}

func (s *SubscriptionDetails) Close() error {
	s.CloseC()
	return s.mapRef.Delete(s)
}

func (s *SubscriptionDetails) Clone() *SubscriptionDetails {
	s.RLock()
	defer s.RUnlock()

	result := &SubscriptionDetails{
		ID:            uuid.NewString(),
		mapRef:        s.mapRef,
		Closed:        false,
		PeerID:        s.PeerID,
		ContentFilter: protocol.ContentFilter{PubsubTopic: s.ContentFilter.PubsubTopic, ContentTopics: maps.Clone(s.ContentFilter.ContentTopics)},
		C:             make(chan *protocol.Envelope),
	}

	return result
}

func (sub *SubscriptionsMap) clear() {
	for _, peerSubscription := range sub.Items {
		for _, subscriptionSet := range peerSubscription.SubsPerPubsubTopic {
			for _, subscription := range subscriptionSet {
				subscription.CloseC()
			}
		}
	}

	sub.Items = make(map[peer.ID]*PeerSubscription)
}

func (sub *SubscriptionsMap) Clear() {
	sub.Lock()
	defer sub.Unlock()
	sub.clear()
}

func (sub *SubscriptionsMap) Notify(peerID peer.ID, envelope *protocol.Envelope) {
	sub.RLock()
	defer sub.RUnlock()

	subscriptions, ok := sub.Items[peerID].SubsPerPubsubTopic[envelope.PubsubTopic()]
	if ok {
		iterateSubscriptionSet(sub.logger, subscriptions, envelope)
	}
}

func iterateSubscriptionSet(logger *zap.Logger, subscriptions SubscriptionSet, envelope *protocol.Envelope) {
	for _, subscription := range subscriptions {
		func(subscription *SubscriptionDetails) {
			subscription.RLock()
			defer subscription.RUnlock()

			_, ok := subscription.ContentFilter.ContentTopics[envelope.Message().ContentTopic]
			if !ok { // only send the msg to subscriptions that have matching contentTopic
				return
			}

			if !subscription.Closed {
				select {
				case subscription.C <- envelope:
				default:
					logger.Warn("can't deliver message to subscription. subscriber too slow")
				}
			}
		}(subscription)
	}
}

func (s *SubscriptionDetails) MarshalJSON() ([]byte, error) {
	type resultType struct {
		PeerID        string   `json:"peerID"`
		PubsubTopic   string   `json:"pubsubTopic"`
		ContentTopics []string `json:"contentTopics"`
	}

	result := resultType{
		PeerID:      s.PeerID.Pretty(),
		PubsubTopic: s.ContentFilter.PubsubTopic,
	}

	for c := range s.ContentFilter.ContentTopics {
		result.ContentTopics = append(result.ContentTopics, c)
	}

	return json.Marshal(result)
}
