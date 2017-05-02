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

package light

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/hashicorp/golang-lru"
)

var (
	bodyCacheLimit  = 256
	blockCacheLimit = 256
)

// LightChain represents a canonical chain that by default only handles block
// headers, downloading block bodies and receipts on demand through an ODR
// interface. It only does header validation during chain insertion.
type LightChain struct {
	hc           *core.HeaderChain
	chainDb      ethdb.Database
	odr          OdrBackend
	eventMux     *event.TypeMux
	genesisBlock *types.Block

	mu      sync.RWMutex
	chainmu sync.RWMutex
	procmu  sync.RWMutex

	bodyCache    *lru.Cache // Cache for the most recent block bodies
	bodyRLPCache *lru.Cache // Cache for the most recent block bodies in RLP encoded format
	blockCache   *lru.Cache // Cache for the most recent entire blocks

	quit    chan struct{}
	running int32 // running must be called automically
	// procInterrupt must be atomically called
	procInterrupt int32 // interrupt signaler for block processing
	wg            sync.WaitGroup

	engine consensus.Engine
}

// NewLightChain returns a fully initialised light chain using information
// available in the database. It initialises the default Ethereum header
// validator.
func NewLightChain(odr OdrBackend, config *params.ChainConfig, engine consensus.Engine, mux *event.TypeMux) (*LightChain, error) {
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)

	bc := &LightChain{
		chainDb:      odr.Database(),
		odr:          odr,
		eventMux:     mux,
		quit:         make(chan struct{}),
		bodyCache:    bodyCache,
		bodyRLPCache: bodyRLPCache,
		blockCache:   blockCache,
		engine:       engine,
	}
	var err error
	bc.hc, err = core.NewHeaderChain(odr.Database(), config, bc.engine, bc.getProcInterrupt)
	if err != nil {
		return nil, err
	}
	bc.genesisBlock, _ = bc.GetBlockByNumber(NoOdr, 0)
	if bc.genesisBlock == nil {
		return nil, core.ErrNoGenesis
	}
	if bc.genesisBlock.Hash() == params.MainNetGenesisHash {
		// add trusted CHT
		WriteTrustedCht(bc.chainDb, TrustedCht{Number: 805, Root: common.HexToHash("85e4286fe0a730390245c49de8476977afdae0eb5530b277f62a52b12313d50f")})
		log.Info("Added trusted CHT for mainnet")
	}
	if bc.genesisBlock.Hash() == params.TestNetGenesisHash {
		// add trusted CHT
		WriteTrustedCht(bc.chainDb, TrustedCht{Number: 206, Root: common.HexToHash("0xea68c59db0887e81aaba06c888053576228b364234cbefc6e8cc28044b82725f")})
		log.Info("Added trusted CHT for testnet")
	}

	if err := bc.loadLastState(); err != nil {
		return nil, err
	}
	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for hash := range core.BadHashes {
		if header := bc.GetHeaderByHash(hash); header != nil {
			log.Error("Found bad hash, rewinding chain", "number", header.Number, "hash", header.ParentHash)
			bc.SetHead(header.Number.Uint64() - 1)
			log.Error("Chain rewind was successful, resuming normal operation")
		}
	}
	return bc, nil
}

func (self *LightChain) getProcInterrupt() bool {
	return atomic.LoadInt32(&self.procInterrupt) == 1
}

// Odr returns the ODR backend of the chain
func (self *LightChain) Odr() OdrBackend {
	return self.odr
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (self *LightChain) loadLastState() error {
	if head := core.GetHeadHeaderHash(self.chainDb); head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		self.Reset()
	} else {
		if header := self.GetHeaderByHash(head); header != nil {
			self.hc.SetCurrentHeader(header)
		}
	}

	// Issue a status log and return
	header := self.hc.CurrentHeader()
	headerTd := self.GetTd(header.Hash(), header.Number.Uint64())
	log.Info("Loaded most recent local header", "number", header.Number, "hash", header.Hash(), "td", headerTd)

	return nil
}

// SetHead rewinds the local chain to a new head. Everything above the new
// head will be deleted and the new one set.
func (bc *LightChain) SetHead(head uint64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.hc.SetHead(head, nil)
	bc.loadLastState()
}

// GasLimit returns the gas limit of the current HEAD block.
func (self *LightChain) GasLimit() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.hc.CurrentHeader().GasLimit
}

// LastBlockHash return the hash of the HEAD block.
func (self *LightChain) LastBlockHash() common.Hash {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.hc.CurrentHeader().Hash()
}

// Status returns status information about the current chain such as the HEAD Td,
// the HEAD hash and the hash of the genesis block.
func (self *LightChain) Status() (td *big.Int, currentBlock common.Hash, genesisBlock common.Hash) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	header := self.hc.CurrentHeader()
	hash := header.Hash()
	return self.GetTd(hash, header.Number.Uint64()), hash, self.genesisBlock.Hash()
}

// State returns a new mutable state based on the current HEAD block.
func (self *LightChain) State() *LightState {
	return NewLightState(StateTrieID(self.hc.CurrentHeader()), self.odr)
}

// Reset purges the entire blockchain, restoring it to its genesis state.
func (bc *LightChain) Reset() {
	bc.ResetWithGenesisBlock(bc.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (bc *LightChain) ResetWithGenesisBlock(genesis *types.Block) {
	// Dump the entire block chain and purge the caches
	bc.SetHead(0)

	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	if err := core.WriteTd(bc.chainDb, genesis.Hash(), genesis.NumberU64(), genesis.Difficulty()); err != nil {
		log.Crit("Failed to write genesis block TD", "err", err)
	}
	if err := core.WriteBlock(bc.chainDb, genesis); err != nil {
		log.Crit("Failed to write genesis block", "err", err)
	}
	bc.genesisBlock = genesis
	bc.hc.SetGenesis(bc.genesisBlock.Header())
	bc.hc.SetCurrentHeader(bc.genesisBlock.Header())
}

// Accessors

// Engine retrieves the light chain's consensus engine.
func (bc *LightChain) Engine() consensus.Engine { return bc.engine }

// Genesis returns the genesis block
func (bc *LightChain) Genesis() *types.Block {
	return bc.genesisBlock
}

// GetBody retrieves a block body (transactions and uncles) from the database
// or ODR service by hash, caching it if found.
func (self *LightChain) GetBody(ctx context.Context, hash common.Hash) (*types.Body, error) {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body, nil
	}
	body, err := GetBody(ctx, self.odr, hash, self.hc.GetBlockNumber(hash))
	if err != nil {
		return nil, err
	}
	// Cache the found body for next time and return
	self.bodyCache.Add(hash, body)
	return body, nil
}

// GetBodyRLP retrieves a block body in RLP encoding from the database or
// ODR service by hash, caching it if found.
func (self *LightChain) GetBodyRLP(ctx context.Context, hash common.Hash) (rlp.RawValue, error) {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyRLPCache.Get(hash); ok {
		return cached.(rlp.RawValue), nil
	}
	body, err := GetBodyRLP(ctx, self.odr, hash, self.hc.GetBlockNumber(hash))
	if err != nil {
		return nil, err
	}
	// Cache the found body for next time and return
	self.bodyRLPCache.Add(hash, body)
	return body, nil
}

// HasBlock checks if a block is fully present in the database or not, caching
// it if present.
func (bc *LightChain) HasBlock(hash common.Hash) bool {
	blk, _ := bc.GetBlockByHash(NoOdr, hash)
	return blk != nil
}

// GetBlock retrieves a block from the database or ODR service by hash and number,
// caching it if found.
func (self *LightChain) GetBlock(ctx context.Context, hash common.Hash, number uint64) (*types.Block, error) {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := self.blockCache.Get(hash); ok {
		return block.(*types.Block), nil
	}
	block, err := GetBlock(ctx, self.odr, hash, number)
	if err != nil {
		return nil, err
	}
	// Cache the found block for next time and return
	self.blockCache.Add(block.Hash(), block)
	return block, nil
}

// GetBlockByHash retrieves a block from the database or ODR service by hash,
// caching it if found.
func (self *LightChain) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return self.GetBlock(ctx, hash, self.hc.GetBlockNumber(hash))
}

// GetBlockByNumber retrieves a block from the database or ODR service by
// number, caching it (associated with its hash) if found.
func (self *LightChain) GetBlockByNumber(ctx context.Context, number uint64) (*types.Block, error) {
	hash, err := GetCanonicalHash(ctx, self.odr, number)
	if hash == (common.Hash{}) || err != nil {
		return nil, err
	}
	return self.GetBlock(ctx, hash, number)
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (bc *LightChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	close(bc.quit)
	atomic.StoreInt32(&bc.procInterrupt, 1)

	bc.wg.Wait()
	log.Info("Blockchain manager stopped")
}

// Rollback is designed to remove a chain of links from the database that aren't
// certain enough to be valid.
func (self *LightChain) Rollback(chain []common.Hash) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		if head := self.hc.CurrentHeader(); head.Hash() == hash {
			self.hc.SetCurrentHeader(self.GetHeader(head.ParentHash, head.Number.Uint64()-1))
		}
	}
}

// postChainEvents iterates over the events generated by a chain insertion and
// posts them into the event mux.
func (self *LightChain) postChainEvents(events []interface{}) {
	for _, event := range events {
		if event, ok := event.(core.ChainEvent); ok {
			if self.LastBlockHash() == event.Hash {
				self.eventMux.Post(core.ChainHeadEvent{Block: event.Block})
			}
		}
		// Fire the insertion events individually too
		self.eventMux.Post(event)
	}
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verfy nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
//
// In the case of a light chain, InsertHeaderChain also creates and posts light
// chain events when necessary.
func (self *LightChain) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	start := time.Now()
	if i, err := self.hc.ValidateHeaderChain(chain, checkFreq); err != nil {
		return i, err
	}

	// Make sure only one thread manipulates the chain at once
	self.chainmu.Lock()
	defer func() {
		self.chainmu.Unlock()
		time.Sleep(time.Millisecond * 10) // ugly hack; do not hog chain lock in case syncing is CPU-limited by validation
	}()

	self.wg.Add(1)
	defer self.wg.Done()

	var events []interface{}
	whFunc := func(header *types.Header) error {
		self.mu.Lock()
		defer self.mu.Unlock()

		status, err := self.hc.WriteHeader(header)

		switch status {
		case core.CanonStatTy:
			log.Debug("Inserted new header", "number", header.Number, "hash", header.Hash())
			events = append(events, core.ChainEvent{Block: types.NewBlockWithHeader(header), Hash: header.Hash()})

		case core.SideStatTy:
			log.Debug("Inserted forked header", "number", header.Number, "hash", header.Hash())
			events = append(events, core.ChainSideEvent{Block: types.NewBlockWithHeader(header)})

		case core.SplitStatTy:
			events = append(events, core.ChainSplitEvent{Block: types.NewBlockWithHeader(header)})
		}
		return err
	}
	i, err := self.hc.InsertHeaderChain(chain, whFunc, start)
	go self.postChainEvents(events)
	return i, err
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (self *LightChain) CurrentHeader() *types.Header {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.hc.CurrentHeader()
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash and number, caching it if found.
func (self *LightChain) GetTd(hash common.Hash, number uint64) *big.Int {
	return self.hc.GetTd(hash, number)
}

// GetTdByHash retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (self *LightChain) GetTdByHash(hash common.Hash) *big.Int {
	return self.hc.GetTdByHash(hash)
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (self *LightChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	return self.hc.GetHeader(hash, number)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (self *LightChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return self.hc.GetHeaderByHash(hash)
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *LightChain) HasHeader(hash common.Hash) bool {
	return bc.hc.HasHeader(hash)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (self *LightChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	return self.hc.GetBlockHashesFromHash(hash, max)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (self *LightChain) GetHeaderByNumber(number uint64) *types.Header {
	return self.hc.GetHeaderByNumber(number)
}

// GetHeaderByNumberOdr retrieves a block header from the database or network
// by number, caching it (associated with its hash) if found.
func (self *LightChain) GetHeaderByNumberOdr(ctx context.Context, number uint64) (*types.Header, error) {
	if header := self.hc.GetHeaderByNumber(number); header != nil {
		return header, nil
	}
	return GetHeaderByNumber(ctx, self.odr, number)
}

func (self *LightChain) SyncCht(ctx context.Context) bool {
	headNum := self.CurrentHeader().Number.Uint64()
	cht := GetTrustedCht(self.chainDb)
	if headNum+1 < cht.Number*ChtFrequency {
		num := cht.Number*ChtFrequency - 1
		header, err := GetHeaderByNumber(ctx, self.odr, num)
		if header != nil && err == nil {
			self.mu.Lock()
			if self.hc.CurrentHeader().Number.Uint64() < header.Number.Uint64() {
				self.hc.SetCurrentHeader(header)
			}
			self.mu.Unlock()
			return true
		}
	}
	return false
}

// LockChain locks the chain mutex for reading so that multiple canonical hashes can be
// retrieved while it is guaranteed that they belong to the same version of the chain
func (self *LightChain) LockChain() {
	self.chainmu.RLock()
}

// UnlockChain unlocks the chain mutex
func (self *LightChain) UnlockChain() {
	self.chainmu.RUnlock()
}
