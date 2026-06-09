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
	"maps"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Filter represents a Waku message filter. Since the move of the reception
// pipeline into the transport (status-im/status-go#7464), the waku-side filter
// is only a wire-subscription record: which content topics on which pubsub
// topic the node asks the network for. Decoding, matching and buffering all
// happen in the transport now.
type Filter struct {
	Src           *ecdsa.PublicKey  // Sender of the message
	KeyAsym       *ecdsa.PrivateKey // Private Key of recipient
	KeySym        []byte            // Key associated with the Topic
	PubsubTopic   string            // Pubsub topic used to filter messages with
	ContentTopics TopicSet          // ContentTopics to filter messages with
	SymKeyHash    common.Hash       // The Keccak256Hash of the symmetric key, needed for optimization
	id            string            // unique identifier
}

// Filters represents a collection of filters
type Filters struct {
	// Map of random ID to Filter
	watchers map[string]*Filter

	logger *zap.Logger

	sync.RWMutex
}

// NewFilters returns a newly created filter collection
func NewFilters(logger *zap.Logger) *Filters {
	return &Filters{
		watchers: make(map[string]*Filter),
		logger:   logger,
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

	fs.Lock()
	defer fs.Unlock()

	if fs.watchers[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	if watcher.expectsSymmetricEncryption() {
		watcher.SymKeyHash = crypto.Keccak256Hash(watcher.KeySym)
	}

	watcher.id = id

	fs.watchers[id] = watcher

	fs.logger.Debug("filters install", zap.String("id", id))
	return id, err
}

// Uninstall will remove a filter whose id has been specified from
// the filter collection
func (fs *Filters) Uninstall(id string) bool {
	fs.Lock()
	defer fs.Unlock()
	watcher := fs.watchers[id]
	if watcher != nil {
		delete(fs.watchers, id)

		fs.logger.Debug("filters uninstall", zap.String("id", id))
		return true
	}
	return false
}

// Get returns a filter from the collection with a specific ID
func (fs *Filters) Get(id string) *Filter {
	fs.RLock()
	defer fs.RUnlock()
	return fs.watchers[id]
}

func (fs *Filters) All() []*Filter {
	fs.RLock()
	defer fs.RUnlock()
	var filters []*Filter
	for _, f := range fs.watchers {
		filters = append(filters, f)
	}
	return filters
}

func (fs *Filters) GetFilters() map[string]*Filter {
	fs.RLock()
	defer fs.RUnlock()
	return maps.Clone(fs.watchers)
}

func (f *Filter) expectsSymmetricEncryption() bool {
	return f.KeySym != nil
}

func (f *Filter) ID() string {
	return f.id
}
