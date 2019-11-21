package shhext

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
)

const (
	// defaultRequestsDelay will be used in RequestsRegistry if no other was provided.
	defaultRequestsDelay = 3 * time.Second
)

type requestMeta struct {
	timestamp time.Time
	lastUID   protocol.Hash
}

// NewRequestsRegistry creates instance of the RequestsRegistry and returns pointer to it.
func NewRequestsRegistry(delay time.Duration) *RequestsRegistry {
	r := &RequestsRegistry{
		delay: delay,
	}
	r.Clear()
	return r
}

// RequestsRegistry keeps map for all requests with timestamp when they were made.
type RequestsRegistry struct {
	mu           sync.Mutex
	delay        time.Duration
	uidToTopics  map[protocol.Hash]protocol.Hash
	byTopicsHash map[protocol.Hash]requestMeta
}

// Register request with given topics. If request with same topics was made in less then configured delay then error
// will be returned.
func (r *RequestsRegistry) Register(uid protocol.Hash, topics []whispertypes.TopicType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	topicsHash := topicsToHash(topics)
	if meta, exist := r.byTopicsHash[topicsHash]; exist {
		if time.Since(meta.timestamp) < r.delay {
			return fmt.Errorf("another request with the same topics was sent less than %s ago. Please wait for a bit longer, or set `force` to true in request parameters", r.delay)
		}
	}
	newMeta := requestMeta{
		timestamp: time.Now(),
		lastUID:   uid,
	}
	r.uidToTopics[uid] = topicsHash
	r.byTopicsHash[topicsHash] = newMeta
	return nil
}

// Has returns true if given uid is stored in registry.
func (r *RequestsRegistry) Has(uid protocol.Hash) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exist := r.uidToTopics[uid]
	return exist
}

// Unregister removes request with given UID from registry.
func (r *RequestsRegistry) Unregister(uid protocol.Hash) {
	r.mu.Lock()
	defer r.mu.Unlock()
	topicsHash, exist := r.uidToTopics[uid]
	if !exist {
		return
	}
	delete(r.uidToTopics, uid)
	meta := r.byTopicsHash[topicsHash]
	// remove topicsHash only if we are trying to unregister last request with this topic.
	if meta.lastUID == uid {
		delete(r.byTopicsHash, topicsHash)
	}
}

// Clear recreates all structures used for caching requests.
func (r *RequestsRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uidToTopics = map[protocol.Hash]protocol.Hash{}
	r.byTopicsHash = map[protocol.Hash]requestMeta{}
}

// topicsToHash returns non-cryptographic hash of the topics.
func topicsToHash(topics []whispertypes.TopicType) protocol.Hash {
	hash := fnv.New32()
	for i := range topics {
		_, _ = hash.Write(topics[i][:]) // never returns error per documentation
	}
	return protocol.BytesToHash(hash.Sum(nil))
}
