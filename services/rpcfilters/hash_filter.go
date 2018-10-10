package rpcfilters

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type hashFilter struct {
	hashes []common.Hash
	mu     sync.Mutex
	done   chan struct{}
	timer  *time.Timer
}

// add adds a hash to the hashFilter
func (f *hashFilter) add(data interface{}) {
	hash := data.(common.Hash)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hashes = append(f.hashes, hash)
}

// pop returns all the hashes stored in the hashFilter and clears the hashFilter contents
func (f *hashFilter) pop() interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	hashes := f.hashes
	f.hashes = nil
	return returnHashes(hashes)
}

func (f *hashFilter) stop() {
	close(f.done)
}

func (f *hashFilter) deadline() *time.Timer {
	return f.timer
}

func newHashFilter() *hashFilter {
	return &hashFilter{
		done: make(chan struct{}),
	}
}
