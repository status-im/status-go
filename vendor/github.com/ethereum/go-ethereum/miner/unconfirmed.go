// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"container/ring"
	"sync"

	"github.com/teslapatrick/go-ethereum/common"
	"github.com/teslapatrick/go-ethereum/core/types"
	"github.com/teslapatrick/go-ethereum/logger"
	"github.com/teslapatrick/go-ethereum/logger/glog"
)

// headerRetriever is used by the unconfirmed block set to verify whether a previously
// mined block is part of the canonical chain or not.
type headerRetriever interface {
	// GetHeaderByNumber retrieves the canonical header associated with a block number.
	GetHeaderByNumber(number uint64) *types.Header
}

// unconfirmedBlock is a small collection of metadata about a locally mined block
// that is placed into a unconfirmed set for canonical chain inclusion tracking.
type unconfirmedBlock struct {
	index uint64
	hash  common.Hash
}

// unconfirmedBlocks implements a data structure to maintain locally mined blocks
// have have not yet reached enough maturity to guarantee chain inclusion. It is
// used by the miner to provide logs to the user when a previously mined block
// has a high enough guarantee to not be reorged out of te canonical chain.
type unconfirmedBlocks struct {
	chain  headerRetriever // Blockchain to verify canonical status through
	depth  uint            // Depth after which to discard previous blocks
	blocks *ring.Ring      // Block infos to allow canonical chain cross checks
	lock   sync.RWMutex    // Protects the fields from concurrent access
}

// newUnconfirmedBlocks returns new data structure to track currently unconfirmed blocks.
func newUnconfirmedBlocks(chain headerRetriever, depth uint) *unconfirmedBlocks {
	return &unconfirmedBlocks{
		chain: chain,
		depth: depth,
	}
}

// Insert adds a new block to the set of unconfirmed ones.
func (set *unconfirmedBlocks) Insert(index uint64, hash common.Hash) {
	// If a new block was mined locally, shift out any old enough blocks
	set.Shift(index)

	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &unconfirmedBlock{
		index: index,
		hash:  hash,
	}
	// Set as the initial ring or append to the end
	set.lock.Lock()
	defer set.lock.Unlock()

	if set.blocks == nil {
		set.blocks = item
	} else {
		set.blocks.Move(-1).Link(item)
	}
	// Display a log for the user to notify of a new mined block unconfirmed
	glog.V(logger.Info).Infof("🔨  mined potential block #%d [%x…], waiting for %d blocks to confirm", index, hash.Bytes()[:4], set.depth)
}

// Shift drops all unconfirmed blocks from the set which exceed the unconfirmed sets depth
// allowance, checking them against the canonical chain for inclusion or staleness
// report.
func (set *unconfirmedBlocks) Shift(height uint64) {
	set.lock.Lock()
	defer set.lock.Unlock()

	for set.blocks != nil {
		// Retrieve the next unconfirmed block and abort if too fresh
		next := set.blocks.Value.(*unconfirmedBlock)
		if next.index+uint64(set.depth) > height {
			break
		}
		// Block seems to exceed depth allowance, check for canonical status
		header := set.chain.GetHeaderByNumber(next.index)
		switch {
		case header == nil:
			glog.V(logger.Warn).Infof("failed to retrieve header of mined block #%d [%x…]", next.index, next.hash.Bytes()[:4])
		case header.Hash() == next.hash:
			glog.V(logger.Info).Infof("🔗  mined block #%d [%x…] reached canonical chain", next.index, next.hash.Bytes()[:4])
		default:
			glog.V(logger.Info).Infof("⑂ mined block #%d [%x…] became a side fork", next.index, next.hash.Bytes()[:4])
		}
		// Drop the block out of the ring
		if set.blocks.Value == set.blocks.Next().Value {
			set.blocks = nil
		} else {
			set.blocks = set.blocks.Move(-1)
			set.blocks.Unlink(1)
			set.blocks = set.blocks.Move(1)
		}
	}
}
