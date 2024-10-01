package filter

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type PeerSet map[peer.ID]struct{}

type PubsubTopics map[string]protocol.ContentTopicSet // pubsubTopic => contentTopics

var errNotFound = errors.New("not found")

const cleanupInterval = time.Minute

type SubscribersMap struct {
	sync.RWMutex

	items       map[peer.ID]PubsubTopics
	interestMap map[string]PeerSet // key: sha256(pubsubTopic-contentTopic) => peers
	timeout     time.Duration
	lastSeen    map[peer.ID]time.Time
}

func NewSubscribersMap(timeout time.Duration) *SubscribersMap {
	return &SubscribersMap{
		items:       make(map[peer.ID]PubsubTopics),
		interestMap: make(map[string]PeerSet),
		timeout:     timeout,
		lastSeen:    make(map[peer.ID]time.Time),
	}
}

func (sub *SubscribersMap) Start(ctx context.Context) {
	go sub.cleanUp(ctx, cleanupInterval)
}

func (sub *SubscribersMap) Clear() {
	sub.Lock()
	defer sub.Unlock()

	sub.items = make(map[peer.ID]PubsubTopics)
	sub.interestMap = make(map[string]PeerSet)
	sub.lastSeen = make(map[peer.ID]time.Time)
}

func (sub *SubscribersMap) Set(peerID peer.ID, pubsubTopic string, contentTopics []string) {
	sub.Lock()
	defer sub.Unlock()

	sub.lastSeen[peerID] = time.Now()

	pubsubTopicMap, ok := sub.items[peerID]
	if !ok {
		pubsubTopicMap = make(PubsubTopics)
	}

	contentTopicsMap, ok := pubsubTopicMap[pubsubTopic]
	if !ok {
		contentTopicsMap = make(protocol.ContentTopicSet)
	}

	for _, c := range contentTopics {
		contentTopicsMap[c] = struct{}{}
	}

	pubsubTopicMap[pubsubTopic] = contentTopicsMap

	sub.items[peerID] = pubsubTopicMap

	for _, c := range contentTopics {
		c := c
		sub.addToInterestMap(peerID, pubsubTopic, c)
	}
}

func (sub *SubscribersMap) Get(peerID peer.ID) (PubsubTopics, bool) {
	sub.RLock()
	defer sub.RUnlock()

	value, ok := sub.items[peerID]

	return value, ok
}

func (sub *SubscribersMap) Has(peerID peer.ID) bool {
	sub.RLock()
	defer sub.RUnlock()

	_, ok := sub.items[peerID]

	return ok
}

func (sub *SubscribersMap) Delete(peerID peer.ID, pubsubTopic string, contentTopics []string) error {
	sub.Lock()
	defer sub.Unlock()

	pubsubTopicMap, ok := sub.items[peerID]
	if !ok {
		return errNotFound
	}

	contentTopicsMap, ok := pubsubTopicMap[pubsubTopic]
	if !ok {
		return errNotFound
	}

	// Updating first the lastSeen since this is a valid activity
	// (it will still get deleted if all content topics are removed)
	sub.lastSeen[peerID] = time.Now()

	// Removing content topics individually
	for _, c := range contentTopics {
		c := c
		delete(contentTopicsMap, c)
		sub.removeFromInterestMap(peerID, pubsubTopic, c)
	}

	pubsubTopicMap[pubsubTopic] = contentTopicsMap

	// No more content topics available. Removing content topic completely
	if len(contentTopicsMap) == 0 {
		delete(pubsubTopicMap, pubsubTopic)
	}

	sub.items[peerID] = pubsubTopicMap

	if len(sub.items[peerID]) == 0 {
		delete(sub.items, peerID)
		delete(sub.lastSeen, peerID)
	}

	return nil
}

func (sub *SubscribersMap) deleteAll(peerID peer.ID) error {
	pubsubTopicMap, ok := sub.items[peerID]
	if !ok {
		return errNotFound
	}

	for pubsubTopic, contentTopicsMap := range pubsubTopicMap {
		// Remove all content topics related to this pubsub topic
		for c := range contentTopicsMap {
			sub.removeFromInterestMap(peerID, pubsubTopic, c)
		}
	}

	delete(sub.items, peerID)
	delete(sub.lastSeen, peerID)

	return nil
}

func (sub *SubscribersMap) DeleteAll(peerID peer.ID) error {
	sub.Lock()
	defer sub.Unlock()

	return sub.deleteAll(peerID)
}

func (sub *SubscribersMap) RemoveAll() {
	sub.Lock()
	defer sub.Unlock()

	sub.items = make(map[peer.ID]PubsubTopics)
	sub.lastSeen = make(map[peer.ID]time.Time)
}

func (sub *SubscribersMap) Count() int {
	sub.RLock()
	defer sub.RUnlock()

	return len(sub.items)
}

func (sub *SubscribersMap) Items(pubsubTopic string, contentTopic string) <-chan peer.ID {
	c := make(chan peer.ID)

	key := getKey(pubsubTopic, contentTopic)

	f := func() {
		sub.RLock()
		defer sub.RUnlock()

		if peers, ok := sub.interestMap[key]; ok {
			for p := range peers {
				c <- p
			}
		}
		close(c)
	}
	go f()

	return c
}

func (sub *SubscribersMap) addToInterestMap(peerID peer.ID, pubsubTopic string, contentTopic string) {
	key := getKey(pubsubTopic, contentTopic)
	peerSet, ok := sub.interestMap[key]
	if !ok {
		peerSet = make(PeerSet)
	}
	peerSet[peerID] = struct{}{}
	sub.interestMap[key] = peerSet
}

func (sub *SubscribersMap) removeFromInterestMap(peerID peer.ID, pubsubTopic string, contentTopic string) {
	key := getKey(pubsubTopic, contentTopic)
	_, exists := sub.interestMap[key]
	if exists {
		delete(sub.interestMap[key], peerID)
	}
}

func getKey(pubsubTopic string, contentTopic string) string {
	pubsubTopicBytes := []byte(pubsubTopic)
	key := append(pubsubTopicBytes, []byte(contentTopic)...)
	return hex.EncodeToString(crypto.Keccak256(key))

}

func (sub *SubscribersMap) Refresh(peerID peer.ID) {
	sub.Lock()
	defer sub.Unlock()

	sub.lastSeen[peerID] = time.Now()
}

func (sub *SubscribersMap) cleanUp(ctx context.Context, cleanupInterval time.Duration) {
	t := time.NewTicker(cleanupInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			sub.Lock()
			for peerID, lastSeen := range sub.lastSeen {
				elapsedTime := time.Since(lastSeen)
				if elapsedTime < sub.timeout {
					_ = sub.deleteAll(peerID)
				}

			}
			sub.Unlock()
		}
	}
}
