// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package common

import (
	"crypto/ecdsa"
	"fmt"
	"sync"

	"github.com/waku-org/go-waku/waku/v2/protocol/relay"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// Filter represents a Waku message filter
type Filter struct {
	Src         *ecdsa.PublicKey  // Sender of the message
	KeyAsym     *ecdsa.PrivateKey // Private Key of recipient
	KeySym      []byte            // Key associated with the Topic
	PubsubTopic string            // Pubsub topic used to filter messages with
	Topics      [][]byte          // ContentTopics to filter messages with
	SymKeyHash  common.Hash       // The Keccak256Hash of the symmetric key, needed for optimization
	id          string            // unique identifier

	Messages MessageStore
}

type FilterSet = map[*Filter]struct{}
type ContentTopicToFilter = map[TopicType]FilterSet
type PubsubTopicToContentTopic = map[string]ContentTopicToFilter

// Filters represents a collection of filters
type Filters struct {
	watchers map[string]*Filter

	topicMatcher     PubsubTopicToContentTopic // map a topic to the filters that are interested in being notified when a message matches that topic
	allTopicsMatcher map[*Filter]struct{}      // list all the filters that will be notified of a new message, no matter what its topic is

	mutex sync.RWMutex
}

// NewFilters returns a newly created filter collection
func NewFilters() *Filters {
	return &Filters{
		watchers:         make(map[string]*Filter),
		topicMatcher:     make(PubsubTopicToContentTopic),
		allTopicsMatcher: make(map[*Filter]struct{}),
	}
}

// Install will add a new filter to the filter collection
func (fs *Filters) Install(watcher *Filter) (string, error) {
	if watcher.KeySym != nil && watcher.KeyAsym != nil {
		return "", fmt.Errorf("filters must choose between symmetric and asymmetric keys")
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", err
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.watchers[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	if watcher.expectsSymmetricEncryption() {
		watcher.SymKeyHash = crypto.Keccak256Hash(watcher.KeySym)
	}

	watcher.id = id
	fs.watchers[id] = watcher
	fs.addTopicMatcher(watcher)
	return id, err
}

// Uninstall will remove a filter whose id has been specified from
// the filter collection
func (fs *Filters) Uninstall(id string) bool {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	if fs.watchers[id] != nil {
		fs.removeFromTopicMatchers(fs.watchers[id])
		delete(fs.watchers, id)
		return true
	}
	return false
}

func (fs *Filters) AllTopics() []TopicType {
	var topics []TopicType
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	for _, topicsPerPubsubTopic := range fs.topicMatcher {
		for t := range topicsPerPubsubTopic {
			topics = append(topics, t)
		}
	}

	return topics
}

// addTopicMatcher adds a filter to the topic matchers.
// If the filter's Topics array is empty, it will be tried on every topic.
// Otherwise, it will be tried on the topics specified.
func (fs *Filters) addTopicMatcher(watcher *Filter) {
	if len(watcher.Topics) == 0 && (watcher.PubsubTopic == relay.DefaultWakuTopic || watcher.PubsubTopic == "") {
		fs.allTopicsMatcher[watcher] = struct{}{}
	} else {
		filtersPerContentTopic, ok := fs.topicMatcher[watcher.PubsubTopic]
		if !ok {
			filtersPerContentTopic = make(ContentTopicToFilter)
		}

		for _, t := range watcher.Topics {
			topic := BytesToTopic(t)
			if filtersPerContentTopic[topic] == nil {
				filtersPerContentTopic[topic] = make(FilterSet)
			}
			filtersPerContentTopic[topic][watcher] = struct{}{}
		}

		fs.topicMatcher[watcher.PubsubTopic] = filtersPerContentTopic
	}
}

// removeFromTopicMatchers removes a filter from the topic matchers
func (fs *Filters) removeFromTopicMatchers(watcher *Filter) {
	delete(fs.allTopicsMatcher, watcher)

	filtersPerContentTopic, ok := fs.topicMatcher[watcher.PubsubTopic]
	if !ok {
		return
	}

	for _, t := range watcher.Topics {
		topic := BytesToTopic(t)
		delete(filtersPerContentTopic[topic], watcher)
	}

	fs.topicMatcher[watcher.PubsubTopic] = filtersPerContentTopic
}

// GetWatchersByTopic returns a slice containing the filters that
// match a specific topic
func (fs *Filters) GetWatchersByTopic(pubsubTopic string, contentTopic TopicType) []*Filter {
	res := make([]*Filter, 0, len(fs.allTopicsMatcher))
	for watcher := range fs.allTopicsMatcher {
		res = append(res, watcher)
	}

	filtersPerContentTopic, ok := fs.topicMatcher[pubsubTopic]
	if !ok {
		return res
	}

	for watcher := range filtersPerContentTopic[contentTopic] {
		res = append(res, watcher)
	}
	return res
}

// Get returns a filter from the collection with a specific ID
func (fs *Filters) Get(id string) *Filter {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.watchers[id]
}

// NotifyWatchers notifies any filter that has declared interest
// for the envelope's topic.
func (fs *Filters) NotifyWatchers(recvMessage *ReceivedMessage) bool {
	var decodedMsg *ReceivedMessage

	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	var matched bool
	candidates := fs.GetWatchersByTopic(recvMessage.PubsubTopic, recvMessage.ContentTopic)

	if len(candidates) == 0 {
		log.Debug("no filters available for this topic", "message", recvMessage.Hash().Hex(), "pubsubTopic", recvMessage.PubsubTopic, "contentTopic", recvMessage.ContentTopic.String())
	}

	for _, watcher := range candidates {
		matched = true
		if decodedMsg == nil {
			decodedMsg = recvMessage.Open(watcher)
			if decodedMsg == nil {
				log.Debug("processing message: failed to open", "message", recvMessage.Hash().Hex(), "filter", watcher.id)
			}
		} else {
			matched = watcher.MatchMessage(decodedMsg)
		}

		if matched && decodedMsg != nil {
			log.Debug("processing message: decrypted", "hash", recvMessage.Hash().Hex())
			if watcher.Src == nil || IsPubKeyEqual(decodedMsg.Src, watcher.Src) {
				watcher.Trigger(decodedMsg)
			}
		}
	}
	return matched
}

func (f *Filter) expectsAsymmetricEncryption() bool {
	return f.KeyAsym != nil
}

func (f *Filter) expectsSymmetricEncryption() bool {
	return f.KeySym != nil
}

// Trigger adds a yet-unknown message to the filter's list of
// received messages.
func (f *Filter) Trigger(msg *ReceivedMessage) {
	err := f.Messages.Add(msg)
	if err != nil {
		log.Error("failed to add msg into the filters store", "hash", msg.Hash(), "error", err)
	}
}

// Retrieve will return the list of all received messages associated
// to a filter.
func (f *Filter) Retrieve() []*ReceivedMessage {
	msgs, err := f.Messages.Pop()
	if err != nil {
		log.Error("failed to retrieve messages from filter store", "error", err)
		return nil
	}
	return msgs
}

// MatchMessage checks if the filter matches an already decrypted
// message (i.e. a Message that has already been handled by
// MatchEnvelope when checked by a previous filter).
// Topics are not checked here, since this is done by topic matchers.
func (f *Filter) MatchMessage(msg *ReceivedMessage) bool {
	if f.expectsAsymmetricEncryption() && msg.isAsymmetricEncryption() {
		return IsPubKeyEqual(&f.KeyAsym.PublicKey, msg.Dst)
	} else if f.expectsSymmetricEncryption() && msg.isSymmetricEncryption() {
		return f.SymKeyHash == msg.SymKeyHash
	}
	return false
}
