package shhext

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	whisper "github.com/status-im/whisper/whisperv6"
)

const (
	// defaultRequestsDelay will be used in RequestsRegistry if no other was provided.
	defaultRequestsDelay = 3 * time.Second
)

type requestMeta struct {
	timestamp time.Time
	lastUID   common.Hash
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
	uidToTopics  map[common.Hash]common.Hash
	byTopicsHash map[common.Hash]requestMeta
}

// Register request with given topics. If request with same topics was made in less then configured delay then error
// will be returned.
func (r *RequestsRegistry) Register(uid common.Hash, topics []whisper.TopicType) error {
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

// Unregister removes request with given UID from registry.
func (r *RequestsRegistry) Unregister(uid common.Hash) {
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
	r.uidToTopics = map[common.Hash]common.Hash{}
	r.byTopicsHash = map[common.Hash]requestMeta{}
}

// topicsToHash returns non-cryptographic hash of the topics.
func topicsToHash(topics []whisper.TopicType) common.Hash {
	hash := fnv.New32()
	for i := range topics {
		_, _ = hash.Write(topics[i][:]) // never returns error per documentation
	}
	return common.BytesToHash(hash.Sum(nil))
}
