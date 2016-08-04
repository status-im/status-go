// Copyright 2015 The go-ethereum Authors
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

// Package downloader contains the manual full chain synchronisation.
package downloader

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/rcrowley/go-metrics"
)

var (
	MaxHashFetch    = 512 // Amount of hashes to be fetched per retrieval request
	MaxBlockFetch   = 128 // Amount of blocks to be fetched per retrieval request
	MaxHeaderFetch  = 192 // Amount of block headers to be fetched per retrieval request
	MaxSkeletonSize = 128 // Number of header fetches to need for a skeleton assembly
	MaxBodyFetch    = 128 // Amount of block bodies to be fetched per retrieval request
	MaxReceiptFetch = 256 // Amount of transaction receipts to allow fetching per request
	MaxStateFetch   = 384 // Amount of node state values to allow fetching per request

	MaxForkAncestry = 3 * params.EpochDuration.Uint64() // Maximum chain reorganisation

	hashTTL        = 3 * time.Second     // [eth/61] Time it takes for a hash request to time out
	blockTargetRTT = 3 * time.Second / 2 // [eth/61] Target time for completing a block retrieval request
	blockTTL       = 3 * blockTargetRTT  // [eth/61] Maximum time allowance before a block request is considered expired

	rttMinEstimate   = 2 * time.Second  // Minimum round-trip time to target for download requests
	rttMaxEstimate   = 20 * time.Second // Maximum rount-trip time to target for download requests
	rttMinConfidence = 0.1              // Worse confidence factor in our estimated RTT value
	ttlScaling       = 3                // Constant scaling factor for RTT -> TTL conversion
	ttlLimit         = time.Minute      // Maximum TTL allowance to prevent reaching crazy timeouts

	qosTuningPeers   = 5    // Number of peers to tune based on (best peers)
	qosConfidenceCap = 10   // Number of peers above which not to modify RTT confidence
	qosTuningImpact  = 0.25 // Impact that a new tuning target has on the previous value

	maxQueuedHashes   = 32 * 1024 // [eth/61] Maximum number of hashes to queue for import (DOS protection)
	maxQueuedHeaders  = 32 * 1024 // [eth/62] Maximum number of headers to queue for import (DOS protection)
	maxHeadersProcess = 2048      // Number of header download results to import at once into the chain
	maxResultsProcess = 2048      // Number of content download results to import at once into the chain

	fsHeaderCheckFrequency = 100  // Verification frequency of the downloaded headers during fast sync
	fsHeaderSafetyNet      = 2048 // Number of headers to discard in case a chain violation is detected
	fsHeaderForceVerify    = 24   // Number of headers to verify before and after the pivot to accept it
	fsPivotInterval        = 512  // Number of headers out of which to randomize the pivot point
	fsMinFullBlocks        = 1024 // Number of blocks to retrieve fully even in fast sync
	fsCriticalTrials       = 10   // Number of times to retry in the cricical section before bailing
)

var (
	errBusy                    = errors.New("busy")
	errUnknownPeer             = errors.New("peer is unknown or unhealthy")
	errBadPeer                 = errors.New("action from bad peer ignored")
	errStallingPeer            = errors.New("peer is stalling")
	errNoPeers                 = errors.New("no peers to keep download active")
	errTimeout                 = errors.New("timeout")
	errEmptyHashSet            = errors.New("empty hash set by peer")
	errEmptyHeaderSet          = errors.New("empty header set by peer")
	errPeersUnavailable        = errors.New("no peers available or all tried for download")
	errAlreadyInPool           = errors.New("hash already in pool")
	errInvalidAncestor         = errors.New("retrieved ancestor is invalid")
	errInvalidChain            = errors.New("retrieved hash chain is invalid")
	errInvalidBlock            = errors.New("retrieved block is invalid")
	errInvalidBody             = errors.New("retrieved block body is invalid")
	errInvalidReceipt          = errors.New("retrieved receipt is invalid")
	errCancelHashFetch         = errors.New("hash download canceled (requested)")
	errCancelBlockFetch        = errors.New("block download canceled (requested)")
	errCancelHeaderFetch       = errors.New("block header download canceled (requested)")
	errCancelBodyFetch         = errors.New("block body download canceled (requested)")
	errCancelReceiptFetch      = errors.New("receipt download canceled (requested)")
	errCancelStateFetch        = errors.New("state data download canceled (requested)")
	errCancelHeaderProcessing  = errors.New("header processing canceled (requested)")
	errCancelContentProcessing = errors.New("content processing canceled (requested)")
	errNoSyncActive            = errors.New("no sync active")
)

type Downloader struct {
	mode SyncMode       // Synchronisation mode defining the strategy used (per sync cycle)
	mux  *event.TypeMux // Event multiplexer to announce sync operation events

	queue *queue   // Scheduler for selecting the hashes to download
	peers *peerSet // Set of active peers from which download can proceed

	fsPivotLock  *types.Header // Pivot header on critical section entry (cannot change between retries)
	fsPivotFails int           // Number of fast sync failures in the critical section

	rttEstimate   uint64 // Round trip time to target for download requests
	rttConfidence uint64 // Confidence in the estimated RTT (unit: millionths to allow atomic ops)

	// Statistics
	syncStatsChainOrigin uint64       // Origin block number where syncing started at
	syncStatsChainHeight uint64       // Highest block number known when syncing started
	syncStatsStateDone   uint64       // Number of state trie entries already pulled
	syncStatsLock        sync.RWMutex // Lock protecting the sync stats fields

	// Callbacks
	hasHeader        headerCheckFn            // Checks if a header is present in the chain
	hasBlockAndState blockAndStateCheckFn     // Checks if a block and associated state is present in the chain
	getHeader        headerRetrievalFn        // Retrieves a header from the chain
	getBlock         blockRetrievalFn         // Retrieves a block from the chain
	headHeader       headHeaderRetrievalFn    // Retrieves the head header from the chain
	headBlock        headBlockRetrievalFn     // Retrieves the head block from the chain
	headFastBlock    headFastBlockRetrievalFn // Retrieves the head fast-sync block from the chain
	commitHeadBlock  headBlockCommitterFn     // Commits a manually assembled block as the chain head
	getTd            tdRetrievalFn            // Retrieves the TD of a block from the chain
	insertHeaders    headerChainInsertFn      // Injects a batch of headers into the chain
	insertBlocks     blockChainInsertFn       // Injects a batch of blocks into the chain
	insertReceipts   receiptChainInsertFn     // Injects a batch of blocks and their receipts into the chain
	rollback         chainRollbackFn          // Removes a batch of recently added chain links
	dropPeer         peerDropFn               // Drops a peer for misbehaving

	// Status
	synchroniseMock func(id string, hash common.Hash) error // Replacement for synchronise during testing
	synchronising   int32
	notified        int32

	// Channels
	newPeerCh     chan *peer
	hashCh        chan dataPack        // [eth/61] Channel receiving inbound hashes
	blockCh       chan dataPack        // [eth/61] Channel receiving inbound blocks
	headerCh      chan dataPack        // [eth/62] Channel receiving inbound block headers
	bodyCh        chan dataPack        // [eth/62] Channel receiving inbound block bodies
	receiptCh     chan dataPack        // [eth/63] Channel receiving inbound receipts
	stateCh       chan dataPack        // [eth/63] Channel receiving inbound node state data
	blockWakeCh   chan bool            // [eth/61] Channel to signal the block fetcher of new tasks
	bodyWakeCh    chan bool            // [eth/62] Channel to signal the block body fetcher of new tasks
	receiptWakeCh chan bool            // [eth/63] Channel to signal the receipt fetcher of new tasks
	stateWakeCh   chan bool            // [eth/63] Channel to signal the state fetcher of new tasks
	headerProcCh  chan []*types.Header // [eth/62] Channel to feed the header processor new tasks

	cancelCh   chan struct{} // Channel to cancel mid-flight syncs
	cancelLock sync.RWMutex  // Lock to protect the cancel channel in delivers

	quitCh   chan struct{} // Quit channel to signal termination
	quitLock sync.RWMutex  // Lock to prevent double closes

	// Testing hooks
	syncInitHook     func(uint64, uint64)  // Method to call upon initiating a new sync run
	bodyFetchHook    func([]*types.Header) // Method to call upon starting a block body fetch
	receiptFetchHook func([]*types.Header) // Method to call upon starting a receipt fetch
	chainInsertHook  func([]*fetchResult)  // Method to call upon inserting a chain of blocks (possibly in multiple invocations)
}

// New creates a new downloader to fetch hashes and blocks from remote peers.
func New(mode SyncMode, stateDb ethdb.Database, mux *event.TypeMux, hasHeader headerCheckFn, hasBlockAndState blockAndStateCheckFn,
	getHeader headerRetrievalFn, getBlock blockRetrievalFn, headHeader headHeaderRetrievalFn, headBlock headBlockRetrievalFn,
	headFastBlock headFastBlockRetrievalFn, commitHeadBlock headBlockCommitterFn, getTd tdRetrievalFn, insertHeaders headerChainInsertFn,
	insertBlocks blockChainInsertFn, insertReceipts receiptChainInsertFn, rollback chainRollbackFn, dropPeer peerDropFn) *Downloader {

	dl := &Downloader{
		mode:             mode,
		mux:              mux,
		queue:            newQueue(stateDb),
		peers:            newPeerSet(),
		rttEstimate:      uint64(rttMaxEstimate),
		rttConfidence:    uint64(1000000),
		hasHeader:        hasHeader,
		hasBlockAndState: hasBlockAndState,
		getHeader:        getHeader,
		getBlock:         getBlock,
		headHeader:       headHeader,
		headBlock:        headBlock,
		headFastBlock:    headFastBlock,
		commitHeadBlock:  commitHeadBlock,
		getTd:            getTd,
		insertHeaders:    insertHeaders,
		insertBlocks:     insertBlocks,
		insertReceipts:   insertReceipts,
		rollback:         rollback,
		dropPeer:         dropPeer,
		newPeerCh:        make(chan *peer, 1),
		hashCh:           make(chan dataPack, 1),
		blockCh:          make(chan dataPack, 1),
		headerCh:         make(chan dataPack, 1),
		bodyCh:           make(chan dataPack, 1),
		receiptCh:        make(chan dataPack, 1),
		stateCh:          make(chan dataPack, 1),
		blockWakeCh:      make(chan bool, 1),
		bodyWakeCh:       make(chan bool, 1),
		receiptWakeCh:    make(chan bool, 1),
		stateWakeCh:      make(chan bool, 1),
		headerProcCh:     make(chan []*types.Header, 1),
		quitCh:           make(chan struct{}),
	}
	go dl.qosTuner()
	return dl
}

// Progress retrieves the synchronisation boundaries, specifically the origin
// block where synchronisation started at (may have failed/suspended); the block
// or header sync is currently at; and the latest known block which the sync targets.
//
// In addition, during the state download phase of fast synchronisation the number
// of processed and the total number of known states are also returned. Otherwise
// these are zero.
func (d *Downloader) Progress() (uint64, uint64, uint64, uint64, uint64) {
	// Fetch the pending state count outside of the lock to prevent unforeseen deadlocks
	pendingStates := uint64(d.queue.PendingNodeData())

	// Lock the current stats and return the progress
	d.syncStatsLock.RLock()
	defer d.syncStatsLock.RUnlock()

	current := uint64(0)
	switch d.mode {
	case FullSync:
		current = d.headBlock().NumberU64()
	case FastSync:
		current = d.headFastBlock().NumberU64()
	case LightSync:
		current = d.headHeader().Number.Uint64()
	}
	return d.syncStatsChainOrigin, current, d.syncStatsChainHeight, d.syncStatsStateDone, d.syncStatsStateDone + pendingStates
}

// Synchronising returns whether the downloader is currently retrieving blocks.
func (d *Downloader) Synchronising() bool {
	return atomic.LoadInt32(&d.synchronising) > 0
}

// RegisterPeer injects a new download peer into the set of block source to be
// used for fetching hashes and blocks from.
func (d *Downloader) RegisterPeer(id string, version int, head common.Hash,
	getRelHashes relativeHashFetcherFn, getAbsHashes absoluteHashFetcherFn, getBlocks blockFetcherFn, // eth/61 callbacks, remove when upgrading
	getRelHeaders relativeHeaderFetcherFn, getAbsHeaders absoluteHeaderFetcherFn, getBlockBodies blockBodyFetcherFn,
	getReceipts receiptFetcherFn, getNodeData stateFetcherFn) error {

	glog.V(logger.Detail).Infoln("Registering peer", id)
	if err := d.peers.Register(newPeer(id, version, head, getRelHashes, getAbsHashes, getBlocks, getRelHeaders, getAbsHeaders, getBlockBodies, getReceipts, getNodeData)); err != nil {
		glog.V(logger.Error).Infoln("Register failed:", err)
		return err
	}
	d.qosReduceConfidence()

	return nil
}

// UnregisterPeer remove a peer from the known list, preventing any action from
// the specified peer. An effort is also made to return any pending fetches into
// the queue.
func (d *Downloader) UnregisterPeer(id string) error {
	glog.V(logger.Detail).Infoln("Unregistering peer", id)
	if err := d.peers.Unregister(id); err != nil {
		glog.V(logger.Error).Infoln("Unregister failed:", err)
		return err
	}
	d.queue.Revoke(id)
	return nil
}

// Synchronise tries to sync up our local block chain with a remote peer, both
// adding various sanity checks as well as wrapping it with various log entries.
func (d *Downloader) Synchronise(id string, head common.Hash, td *big.Int, mode SyncMode) error {
	glog.V(logger.Detail).Infof("Attempting synchronisation: %v, head [%x…], TD %v", id, head[:4], td)

	err := d.synchronise(id, head, td, mode)
	switch err {
	case nil:
		glog.V(logger.Detail).Infof("Synchronisation completed")

	case errBusy:
		glog.V(logger.Detail).Infof("Synchronisation already in progress")

	case errTimeout, errBadPeer, errStallingPeer, errEmptyHashSet, errEmptyHeaderSet, errPeersUnavailable, errInvalidAncestor, errInvalidChain:
		glog.V(logger.Debug).Infof("Removing peer %v: %v", id, err)
		d.dropPeer(id)

	default:
		glog.V(logger.Warn).Infof("Synchronisation failed: %v", err)
	}
	return err
}

// synchronise will select the peer and use it for synchronising. If an empty string is given
// it will use the best peer possible and synchronize if it's TD is higher than our own. If any of the
// checks fail an error will be returned. This method is synchronous
func (d *Downloader) synchronise(id string, hash common.Hash, td *big.Int, mode SyncMode) error {
	// Mock out the synchronisation if testing
	if d.synchroniseMock != nil {
		return d.synchroniseMock(id, hash)
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.synchronising, 0, 1) {
		return errBusy
	}
	defer atomic.StoreInt32(&d.synchronising, 0)

	// Post a user notification of the sync (only once per session)
	if atomic.CompareAndSwapInt32(&d.notified, 0, 1) {
		glog.V(logger.Info).Infoln("Block synchronisation started")
	}
	// Reset the queue, peer set and wake channels to clean any internal leftover state
	d.queue.Reset()
	d.peers.Reset()

	for _, ch := range []chan bool{d.blockWakeCh, d.bodyWakeCh, d.receiptWakeCh, d.stateWakeCh} {
		select {
		case <-ch:
		default:
		}
	}
	for _, ch := range []chan dataPack{d.hashCh, d.blockCh, d.headerCh, d.bodyCh, d.receiptCh, d.stateCh} {
		for empty := false; !empty; {
			select {
			case <-ch:
			default:
				empty = true
			}
		}
	}
	for empty := false; !empty; {
		select {
		case <-d.headerProcCh:
		default:
			empty = true
		}
	}
	// Create cancel channel for aborting mid-flight
	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelLock.Unlock()

	defer d.cancel() // No matter what, we can't leave the cancel channel open

	// Set the requested sync mode, unless it's forbidden
	d.mode = mode
	if d.mode == FastSync && d.fsPivotFails >= fsCriticalTrials {
		d.mode = FullSync
	}
	// Retrieve the origin peer and initiate the downloading process
	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	return d.syncWithPeer(p, hash, td)
}

// syncWithPeer starts a block synchronization based on the hash chain from the
// specified peer and head hash.
func (d *Downloader) syncWithPeer(p *peer, hash common.Hash, td *big.Int) (err error) {
	d.mux.Post(StartEvent{})
	defer func() {
		// reset on error
		if err != nil {
			d.mux.Post(FailedEvent{err})
		} else {
			d.mux.Post(DoneEvent{})
		}
	}()

	glog.V(logger.Debug).Infof("Synchronising with the network using: %s [eth/%d]", p.id, p.version)
	defer func(start time.Time) {
		glog.V(logger.Debug).Infof("Synchronisation terminated after %v", time.Since(start))
	}(time.Now())

	switch {
	case p.version == 61:
		// Look up the sync boundaries: the common ancestor and the target block
		latest, err := d.fetchHeight61(p)
		if err != nil {
			return err
		}
		origin, err := d.findAncestor61(p, latest)
		if err != nil {
			return err
		}
		d.syncStatsLock.Lock()
		if d.syncStatsChainHeight <= origin || d.syncStatsChainOrigin > origin {
			d.syncStatsChainOrigin = origin
		}
		d.syncStatsChainHeight = latest
		d.syncStatsLock.Unlock()

		// Initiate the sync using a concurrent hash and block retrieval algorithm
		d.queue.Prepare(origin+1, d.mode, 0, nil)
		if d.syncInitHook != nil {
			d.syncInitHook(origin, latest)
		}
		return d.spawnSync(origin+1,
			func() error { return d.fetchHashes61(p, td, origin+1) },
			func() error { return d.fetchBlocks61(origin + 1) },
		)

	case p.version >= 62:
		// Look up the sync boundaries: the common ancestor and the target block
		latest, err := d.fetchHeight(p)
		if err != nil {
			return err
		}
		height := latest.Number.Uint64()

		origin, err := d.findAncestor(p, height)
		if err != nil {
			return err
		}
		d.syncStatsLock.Lock()
		if d.syncStatsChainHeight <= origin || d.syncStatsChainOrigin > origin {
			d.syncStatsChainOrigin = origin
		}
		d.syncStatsChainHeight = height
		d.syncStatsLock.Unlock()

		// Initiate the sync using a concurrent header and content retrieval algorithm
		pivot := uint64(0)
		switch d.mode {
		case LightSync:
			pivot = height
		case FastSync:
			// Calculate the new fast/slow sync pivot point
			if d.fsPivotLock == nil {
				pivotOffset, err := rand.Int(rand.Reader, big.NewInt(int64(fsPivotInterval)))
				if err != nil {
					panic(fmt.Sprintf("Failed to access crypto random source: %v", err))
				}
				if height > uint64(fsMinFullBlocks)+pivotOffset.Uint64() {
					pivot = height - uint64(fsMinFullBlocks) - pivotOffset.Uint64()
				}
			} else {
				// Pivot point locked in, use this and do not pick a new one!
				pivot = d.fsPivotLock.Number.Uint64()
			}
			// If the point is below the origin, move origin back to ensure state download
			if pivot < origin {
				if pivot > 0 {
					origin = pivot - 1
				} else {
					origin = 0
				}
			}
			glog.V(logger.Debug).Infof("Fast syncing until pivot block #%d", pivot)
		}
		d.queue.Prepare(origin+1, d.mode, pivot, latest)
		if d.syncInitHook != nil {
			d.syncInitHook(origin, height)
		}
		return d.spawnSync(origin+1,
			func() error { return d.fetchHeaders(p, origin+1) },    // Headers are always retrieved
			func() error { return d.processHeaders(origin+1, td) }, // Headers are always retrieved
			func() error { return d.fetchBodies(origin + 1) },      // Bodies are retrieved during normal and fast sync
			func() error { return d.fetchReceipts(origin + 1) },    // Receipts are retrieved during fast sync
			func() error { return d.fetchNodeData() },              // Node state data is retrieved during fast sync
		)

	default:
		// Something very wrong, stop right here
		glog.V(logger.Error).Infof("Unsupported eth protocol: %d", p.version)
		return errBadPeer
	}
}

// spawnSync runs d.process and all given fetcher functions to completion in
// separate goroutines, returning the first error that appears.
func (d *Downloader) spawnSync(origin uint64, fetchers ...func() error) error {
	var wg sync.WaitGroup
	errc := make(chan error, len(fetchers)+1)
	wg.Add(len(fetchers) + 1)
	go func() { defer wg.Done(); errc <- d.processContent() }()
	for _, fn := range fetchers {
		fn := fn
		go func() { defer wg.Done(); errc <- fn() }()
	}
	// Wait for the first error, then terminate the others.
	var err error
	for i := 0; i < len(fetchers)+1; i++ {
		if i == len(fetchers) {
			// Close the queue when all fetchers have exited.
			// This will cause the block processor to end when
			// it has processed the queue.
			d.queue.Close()
		}
		if err = <-errc; err != nil {
			break
		}
	}
	d.queue.Close()
	d.cancel()
	wg.Wait()
	return err
}

// cancel cancels all of the operations and resets the queue. It returns true
// if the cancel operation was completed.
func (d *Downloader) cancel() {
	// Close the current cancel channel
	d.cancelLock.Lock()
	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
			// Channel was already closed
		default:
			close(d.cancelCh)
		}
	}
	d.cancelLock.Unlock()
}

// Terminate interrupts the downloader, canceling all pending operations.
// The downloader cannot be reused after calling Terminate.
func (d *Downloader) Terminate() {
	// Close the termination channel (make sure double close is allowed)
	d.quitLock.Lock()
	select {
	case <-d.quitCh:
	default:
		close(d.quitCh)
	}
	d.quitLock.Unlock()

	// Cancel any pending download requests
	d.cancel()
}

// fetchHeight61 retrieves the head block of the remote peer to aid in estimating
// the total time a pending synchronisation would take.
func (d *Downloader) fetchHeight61(p *peer) (uint64, error) {
	glog.V(logger.Debug).Infof("%v: retrieving remote chain height", p)

	// Request the advertised remote head block and wait for the response
	go p.getBlocks([]common.Hash{p.head})

	timeout := time.After(hashTTL)
	for {
		select {
		case <-d.cancelCh:
			return 0, errCancelBlockFetch

		case packet := <-d.blockCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received blocks from incorrect peer(%s)", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			blocks := packet.(*blockPack).blocks
			if len(blocks) != 1 {
				glog.V(logger.Debug).Infof("%v: invalid number of head blocks: %d != 1", p, len(blocks))
				return 0, errBadPeer
			}
			return blocks[0].NumberU64(), nil

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head block timeout", p)
			return 0, errTimeout

		case <-d.hashCh:
			// Out of bounds hashes received, ignore them

		case <-d.headerCh:
		case <-d.bodyCh:
		case <-d.stateCh:
		case <-d.receiptCh:
			// Ignore eth/{62,63} packets because this is eth/61.
			// These can arrive as a late delivery from a previous sync.
		}
	}
}

// findAncestor61 tries to locate the common ancestor block of the local chain and
// a remote peers blockchain. In the general case when our node was in sync and
// on the correct chain, checking the top N blocks should already get us a match.
// In the rare scenario when we ended up on a long reorganisation (i.e. none of
// the head blocks match), we do a binary search to find the common ancestor.
func (d *Downloader) findAncestor61(p *peer, height uint64) (uint64, error) {
	glog.V(logger.Debug).Infof("%v: looking for common ancestor", p)

	// Figure out the valid ancestor range to prevent rewrite attacks
	floor, ceil := int64(-1), d.headBlock().NumberU64()
	if ceil >= MaxForkAncestry {
		floor = int64(ceil - MaxForkAncestry)
	}
	// Request the topmost blocks to short circuit binary ancestor lookup
	head := ceil
	if head > height {
		head = height
	}
	from := int64(head) - int64(MaxHashFetch) + 1
	if from < 0 {
		from = 0
	}
	go p.getAbsHashes(uint64(from), MaxHashFetch)

	// Wait for the remote response to the head fetch
	number, hash := uint64(0), common.Hash{}
	timeout := time.After(hashTTL)

	for finished := false; !finished; {
		select {
		case <-d.cancelCh:
			return 0, errCancelHashFetch

		case packet := <-d.hashCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			hashes := packet.(*hashPack).hashes
			if len(hashes) == 0 {
				glog.V(logger.Debug).Infof("%v: empty head hash set", p)
				return 0, errEmptyHashSet
			}
			// Check if a common ancestor was found
			finished = true
			for i := len(hashes) - 1; i >= 0; i-- {
				// Skip any headers that underflow/overflow our requested set
				header := d.getHeader(hashes[i])
				if header == nil || header.Number.Int64() < from || header.Number.Uint64() > head {
					continue
				}
				// Otherwise check if we already know the header or not
				if d.hasBlockAndState(hashes[i]) {
					number, hash = header.Number.Uint64(), header.Hash()
					break
				}
			}

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head hash timeout", p)
			return 0, errTimeout

		case <-d.blockCh:
			// Out of bounds blocks received, ignore them

		case <-d.headerCh:
		case <-d.bodyCh:
		case <-d.stateCh:
		case <-d.receiptCh:
			// Ignore eth/{62,63} packets because this is eth/61.
			// These can arrive as a late delivery from a previous sync.
		}
	}
	// If the head fetch already found an ancestor, return
	if !common.EmptyHash(hash) {
		if int64(number) <= floor {
			glog.V(logger.Warn).Infof("%v: potential rewrite attack: #%d [%x…] <= #%d limit", p, number, hash[:4], floor)
			return 0, errInvalidAncestor
		}
		glog.V(logger.Debug).Infof("%v: common ancestor: #%d [%x…]", p, number, hash[:4])
		return number, nil
	}
	// Ancestor not found, we need to binary search over our chain
	start, end := uint64(0), head
	if floor > 0 {
		start = uint64(floor)
	}
	for start+1 < end {
		// Split our chain interval in two, and request the hash to cross check
		check := (start + end) / 2

		timeout := time.After(hashTTL)
		go p.getAbsHashes(uint64(check), 1)

		// Wait until a reply arrives to this request
		for arrived := false; !arrived; {
			select {
			case <-d.cancelCh:
				return 0, errCancelHashFetch

			case packet := <-d.hashCh:
				// Discard anything not from the origin peer
				if packet.PeerId() != p.id {
					glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", packet.PeerId())
					break
				}
				// Make sure the peer actually gave something valid
				hashes := packet.(*hashPack).hashes
				if len(hashes) != 1 {
					glog.V(logger.Debug).Infof("%v: invalid search hash set (%d)", p, len(hashes))
					return 0, errBadPeer
				}
				arrived = true

				// Modify the search interval based on the response
				if !d.hasBlockAndState(hashes[0]) {
					end = check
					break
				}
				block := d.getBlock(hashes[0]) // this doesn't check state, hence the above explicit check
				if block.NumberU64() != check {
					glog.V(logger.Debug).Infof("%v: non requested hash #%d [%x…], instead of #%d", p, block.NumberU64(), block.Hash().Bytes()[:4], check)
					return 0, errBadPeer
				}
				start = check

			case <-timeout:
				glog.V(logger.Debug).Infof("%v: search hash timeout", p)
				return 0, errTimeout

			case <-d.blockCh:
				// Out of bounds blocks received, ignore them

			case <-d.headerCh:
			case <-d.bodyCh:
			case <-d.stateCh:
			case <-d.receiptCh:
				// Ignore eth/{62,63} packets because this is eth/61.
				// These can arrive as a late delivery from a previous sync.
			}
		}
	}
	// Ensure valid ancestry and return
	if int64(start) <= floor {
		glog.V(logger.Warn).Infof("%v: potential rewrite attack: #%d [%x…] <= #%d limit", p, start, hash[:4], floor)
		return 0, errInvalidAncestor
	}
	glog.V(logger.Debug).Infof("%v: common ancestor: #%d [%x…]", p, start, hash[:4])
	return start, nil
}

// fetchHashes61 keeps retrieving hashes from the requested number, until no more
// are returned, potentially throttling on the way.
func (d *Downloader) fetchHashes61(p *peer, td *big.Int, from uint64) error {
	glog.V(logger.Debug).Infof("%v: downloading hashes from #%d", p, from)

	// Create a timeout timer, and the associated hash fetcher
	request := time.Now()       // time of the last fetch request
	timeout := time.NewTimer(0) // timer to dump a non-responsive active peer
	<-timeout.C                 // timeout channel should be initially empty
	defer timeout.Stop()

	getHashes := func(from uint64) {
		glog.V(logger.Detail).Infof("%v: fetching %d hashes from #%d", p, MaxHashFetch, from)

		request = time.Now()
		timeout.Reset(hashTTL)
		go p.getAbsHashes(from, MaxHashFetch)
	}
	// Start pulling hashes, until all are exhausted
	getHashes(from)
	gotHashes := false

	for {
		select {
		case <-d.cancelCh:
			return errCancelHashFetch

		case packet := <-d.hashCh:
			// Make sure the active peer is giving us the hashes
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", packet.PeerId())
				break
			}
			hashReqTimer.UpdateSince(request)
			timeout.Stop()

			// If no more hashes are inbound, notify the block fetcher and return
			if packet.Items() == 0 {
				glog.V(logger.Debug).Infof("%v: no available hashes", p)

				select {
				case d.blockWakeCh <- false:
				case <-d.cancelCh:
				}
				// If no hashes were retrieved at all, the peer violated it's TD promise that it had a
				// better chain compared to ours. The only exception is if it's promised blocks were
				// already imported by other means (e.g. fetcher):
				//
				// R <remote peer>, L <local node>: Both at block 10
				// R: Mine block 11, and propagate it to L
				// L: Queue block 11 for import
				// L: Notice that R's head and TD increased compared to ours, start sync
				// L: Import of block 11 finishes
				// L: Sync begins, and finds common ancestor at 11
				// L: Request new hashes up from 11 (R's TD was higher, it must have something)
				// R: Nothing to give
				if !gotHashes && td.Cmp(d.getTd(d.headBlock().Hash())) > 0 {
					return errStallingPeer
				}
				return nil
			}
			gotHashes = true
			hashes := packet.(*hashPack).hashes

			// Otherwise insert all the new hashes, aborting in case of junk
			glog.V(logger.Detail).Infof("%v: scheduling %d hashes from #%d", p, len(hashes), from)

			inserts := d.queue.Schedule61(hashes, true)
			if len(inserts) != len(hashes) {
				glog.V(logger.Debug).Infof("%v: stale hashes", p)
				return errBadPeer
			}
			// Notify the block fetcher of new hashes, but stop if queue is full
			if d.queue.PendingBlocks() < maxQueuedHashes {
				// We still have hashes to fetch, send continuation wake signal (potential)
				select {
				case d.blockWakeCh <- true:
				default:
				}
			} else {
				// Hash limit reached, send a termination wake signal (enforced)
				select {
				case d.blockWakeCh <- false:
				case <-d.cancelCh:
				}
				return nil
			}
			// Queue not yet full, fetch the next batch
			from += uint64(len(hashes))
			getHashes(from)

		case <-timeout.C:
			glog.V(logger.Debug).Infof("%v: hash request timed out", p)
			hashTimeoutMeter.Mark(1)
			return errTimeout

		case <-d.headerCh:
		case <-d.bodyCh:
		case <-d.stateCh:
		case <-d.receiptCh:
			// Ignore eth/{62,63} packets because this is eth/61.
			// These can arrive as a late delivery from a previous sync.
		}
	}
}

// fetchBlocks61 iteratively downloads the scheduled hashes, taking any available
// peers, reserving a chunk of blocks for each, waiting for delivery and also
// periodically checking for timeouts.
func (d *Downloader) fetchBlocks61(from uint64) error {
	glog.V(logger.Debug).Infof("Downloading blocks from #%d", from)
	defer glog.V(logger.Debug).Infof("Block download terminated")

	// Create a timeout timer for scheduling expiration tasks
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	update := make(chan struct{}, 1)

	// Fetch blocks until the hash fetcher's done
	finished := false
	for {
		select {
		case <-d.cancelCh:
			return errCancelBlockFetch

		case packet := <-d.blockCh:
			// If the peer was previously banned and failed to deliver it's pack
			// in a reasonable time frame, ignore it's message.
			if peer := d.peers.Peer(packet.PeerId()); peer != nil {
				blocks := packet.(*blockPack).blocks

				// Deliver the received chunk of blocks and check chain validity
				accepted, err := d.queue.DeliverBlocks(peer.id, blocks)
				if err == errInvalidChain {
					return err
				}
				// Unless a peer delivered something completely else than requested (usually
				// caused by a timed out request which came through in the end), set it to
				// idle. If the delivery's stale, the peer should have already been idled.
				if err != errStaleDelivery {
					peer.SetBlocksIdle(accepted)
				}
				// Issue a log to the user to see what's going on
				switch {
				case err == nil && len(blocks) == 0:
					glog.V(logger.Detail).Infof("%s: no blocks delivered", peer)
				case err == nil:
					glog.V(logger.Detail).Infof("%s: delivered %d blocks", peer, len(blocks))
				default:
					glog.V(logger.Detail).Infof("%s: delivery failed: %v", peer, err)
				}
			}
			// Blocks arrived, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case cont := <-d.blockWakeCh:
			// The hash fetcher sent a continuation flag, check if it's done
			if !cont {
				finished = true
			}
			// Hashes arrive, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-ticker.C:
			// Sanity check update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-update:
			// Short circuit if we lost all our peers
			if d.peers.Len() == 0 {
				return errNoPeers
			}
			// Check for block request timeouts and demote the responsible peers
			for pid, fails := range d.queue.ExpireBlocks(blockTTL) {
				if peer := d.peers.Peer(pid); peer != nil {
					if fails > 1 {
						glog.V(logger.Detail).Infof("%s: block delivery timeout", peer)
						peer.SetBlocksIdle(0)
					} else {
						glog.V(logger.Debug).Infof("%s: stalling block delivery, dropping", peer)
						d.dropPeer(pid)
					}
				}
			}
			// If there's nothing more to fetch, wait or terminate
			if d.queue.PendingBlocks() == 0 {
				if !d.queue.InFlightBlocks() && finished {
					glog.V(logger.Debug).Infof("Block fetching completed")
					return nil
				}
				break
			}
			// Send a download request to all idle peers, until throttled
			throttled := false
			idles, total := d.peers.BlockIdlePeers()

			for _, peer := range idles {
				// Short circuit if throttling activated
				if d.queue.ShouldThrottleBlocks() {
					throttled = true
					break
				}
				// Reserve a chunk of hashes for a peer. A nil can mean either that
				// no more hashes are available, or that the peer is known not to
				// have them.
				request := d.queue.ReserveBlocks(peer, peer.BlockCapacity(blockTargetRTT))
				if request == nil {
					continue
				}
				if glog.V(logger.Detail) {
					glog.Infof("%s: requesting %d blocks", peer, len(request.Hashes))
				}
				// Fetch the chunk and make sure any errors return the hashes to the queue
				if err := peer.Fetch61(request); err != nil {
					// Although we could try and make an attempt to fix this, this error really
					// means that we've double allocated a fetch task to a peer. If that is the
					// case, the internal state of the downloader and the queue is very wrong so
					// better hard crash and note the error instead of silently accumulating into
					// a much bigger issue.
					panic(fmt.Sprintf("%v: fetch assignment failed", peer))
				}
			}
			// Make sure that we have peers available for fetching. If all peers have been tried
			// and all failed throw an error
			if !throttled && !d.queue.InFlightBlocks() && len(idles) == total {
				return errPeersUnavailable
			}

		case <-d.headerCh:
		case <-d.bodyCh:
		case <-d.stateCh:
		case <-d.receiptCh:
			// Ignore eth/{62,63} packets because this is eth/61.
			// These can arrive as a late delivery from a previous sync.
		}
	}
}

// fetchHeight retrieves the head header of the remote peer to aid in estimating
// the total time a pending synchronisation would take.
func (d *Downloader) fetchHeight(p *peer) (*types.Header, error) {
	glog.V(logger.Debug).Infof("%v: retrieving remote chain height", p)

	// Request the advertised remote head block and wait for the response
	go p.getRelHeaders(p.head, 1, 0, false)

	timeout := time.After(d.requestTTL())
	for {
		select {
		case <-d.cancelCh:
			return nil, errCancelBlockFetch

		case packet := <-d.headerCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received headers from incorrect peer(%s)", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			headers := packet.(*headerPack).headers
			if len(headers) != 1 {
				glog.V(logger.Debug).Infof("%v: invalid number of head headers: %d != 1", p, len(headers))
				return nil, errBadPeer
			}
			return headers[0], nil

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head header timeout", p)
			return nil, errTimeout

		case <-d.bodyCh:
		case <-d.stateCh:
		case <-d.receiptCh:
			// Out of bounds delivery, ignore

		case <-d.hashCh:
		case <-d.blockCh:
			// Ignore eth/61 packets because this is eth/62+.
			// These can arrive as a late delivery from a previous sync.
		}
	}
}

// findAncestor tries to locate the common ancestor link of the local chain and
// a remote peers blockchain. In the general case when our node was in sync and
// on the correct chain, checking the top N links should already get us a match.
// In the rare scenario when we ended up on a long reorganisation (i.e. none of
// the head links match), we do a binary search to find the common ancestor.
func (d *Downloader) findAncestor(p *peer, height uint64) (uint64, error) {
	glog.V(logger.Debug).Infof("%v: looking for common ancestor", p)

	// Figure out the valid ancestor range to prevent rewrite attacks
	floor, ceil := int64(-1), d.headHeader().Number.Uint64()
	if d.mode == FullSync {
		ceil = d.headBlock().NumberU64()
	} else if d.mode == FastSync {
		ceil = d.headFastBlock().NumberU64()
	}
	if ceil >= MaxForkAncestry {
		floor = int64(ceil - MaxForkAncestry)
	}
	// Request the topmost blocks to short circuit binary ancestor lookup
	head := ceil
	if head > height {
		head = height
	}
	from := int64(head) - int64(MaxHeaderFetch) + 1
	if from < 0 {
		from = 0
	}
	go p.getAbsHeaders(uint64(from), MaxHeaderFetch, 0, false)

	// Wait for the remote response to the head fetch
	number, hash := uint64(0), common.Hash{}
	timeout := time.After(d.requestTTL())

	for finished := false; !finished; {
		select {
		case <-d.cancelCh:
			return 0, errCancelHashFetch

		case packet := <-d.headerCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received headers from incorrect peer(%s)", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			headers := packet.(*headerPack).headers
			if len(headers) == 0 {
				glog.V(logger.Warn).Infof("%v: empty head header set", p)
				return 0, errEmptyHeaderSet
			}
			// Make sure the peer's reply conforms to the request
			for i := 0; i < len(headers); i++ {
				if number := headers[i].Number.Int64(); number != from+int64(i) {
					glog.V(logger.Warn).Infof("%v: head header set (item %d) broke chain ordering: requested %d, got %d", p, i, from+int64(i), number)
					return 0, errInvalidChain
				}
				if i > 0 && headers[i-1].Hash() != headers[i].ParentHash {
					glog.V(logger.Warn).Infof("%v: head header set (item %d) broke chain ancestry: expected [%x], got [%x]", p, i, headers[i-1].Hash().Bytes()[:4], headers[i].ParentHash[:4])
					return 0, errInvalidChain
				}
			}
			// Check if a common ancestor was found
			finished = true
			for i := len(headers) - 1; i >= 0; i-- {
				// Skip any headers that underflow/overflow our requested set
				if headers[i].Number.Int64() < from || headers[i].Number.Uint64() > head {
					continue
				}
				// Otherwise check if we already know the header or not
				if (d.mode == FullSync && d.hasBlockAndState(headers[i].Hash())) || (d.mode != FullSync && d.hasHeader(headers[i].Hash())) {
					number, hash = headers[i].Number.Uint64(), headers[i].Hash()
					break
				}
			}

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head header timeout", p)
			return 0, errTimeout

		case <-d.bodyCh:
		case <-d.stateCh:
		case <-d.receiptCh:
			// Out of bounds delivery, ignore

		case <-d.hashCh:
		case <-d.blockCh:
			// Ignore eth/61 packets because this is eth/62+.
			// These can arrive as a late delivery from a previous sync.
		}
	}
	// If the head fetch already found an ancestor, return
	if !common.EmptyHash(hash) {
		if int64(number) <= floor {
			glog.V(logger.Warn).Infof("%v: potential rewrite attack: #%d [%x…] <= #%d limit", p, number, hash[:4], floor)
			return 0, errInvalidAncestor
		}
		glog.V(logger.Debug).Infof("%v: common ancestor: #%d [%x…]", p, number, hash[:4])
		return number, nil
	}
	// Ancestor not found, we need to binary search over our chain
	start, end := uint64(0), head
	if floor > 0 {
		start = uint64(floor)
	}
	for start+1 < end {
		// Split our chain interval in two, and request the hash to cross check
		check := (start + end) / 2

		timeout := time.After(d.requestTTL())
		go p.getAbsHeaders(uint64(check), 1, 0, false)

		// Wait until a reply arrives to this request
		for arrived := false; !arrived; {
			select {
			case <-d.cancelCh:
				return 0, errCancelHashFetch

			case packer := <-d.headerCh:
				// Discard anything not from the origin peer
				if packer.PeerId() != p.id {
					glog.V(logger.Debug).Infof("Received headers from incorrect peer(%s)", packer.PeerId())
					break
				}
				// Make sure the peer actually gave something valid
				headers := packer.(*headerPack).headers
				if len(headers) != 1 {
					glog.V(logger.Debug).Infof("%v: invalid search header set (%d)", p, len(headers))
					return 0, errBadPeer
				}
				arrived = true

				// Modify the search interval based on the response
				if (d.mode == FullSync && !d.hasBlockAndState(headers[0].Hash())) || (d.mode != FullSync && !d.hasHeader(headers[0].Hash())) {
					end = check
					break
				}
				header := d.getHeader(headers[0].Hash()) // Independent of sync mode, header surely exists
				if header.Number.Uint64() != check {
					glog.V(logger.Debug).Infof("%v: non requested header #%d [%x…], instead of #%d", p, header.Number, header.Hash().Bytes()[:4], check)
					return 0, errBadPeer
				}
				start = check

			case <-timeout:
				glog.V(logger.Debug).Infof("%v: search header timeout", p)
				return 0, errTimeout

			case <-d.bodyCh:
			case <-d.stateCh:
			case <-d.receiptCh:
				// Out of bounds delivery, ignore

			case <-d.hashCh:
			case <-d.blockCh:
				// Ignore eth/61 packets because this is eth/62+.
				// These can arrive as a late delivery from a previous sync.
			}
		}
	}
	// Ensure valid ancestry and return
	if int64(start) <= floor {
		glog.V(logger.Warn).Infof("%v: potential rewrite attack: #%d [%x…] <= #%d limit", p, start, hash[:4], floor)
		return 0, errInvalidAncestor
	}
	glog.V(logger.Debug).Infof("%v: common ancestor: #%d [%x…]", p, start, hash[:4])
	return start, nil
}

// fetchHeaders keeps retrieving headers concurrently from the number
// requested, until no more are returned, potentially throttling on the way. To
// facilitate concurrency but still protect against malicious nodes sending bad
// headers, we construct a header chain skeleton using the "origin" peer we are
// syncing with, and fill in the missing headers using anyone else. Headers from
// other peers are only accepted if they map cleanly to the skeleton. If no one
// can fill in the skeleton - not even the origin peer - it's assumed invalid and
// the origin is dropped.
func (d *Downloader) fetchHeaders(p *peer, from uint64) error {
	glog.V(logger.Debug).Infof("%v: directing header downloads from #%d", p, from)
	defer glog.V(logger.Debug).Infof("%v: header download terminated", p)

	// Create a timeout timer, and the associated header fetcher
	skeleton := true            // Skeleton assembly phase or finishing up
	request := time.Now()       // time of the last skeleton fetch request
	timeout := time.NewTimer(0) // timer to dump a non-responsive active peer
	<-timeout.C                 // timeout channel should be initially empty
	defer timeout.Stop()

	getHeaders := func(from uint64) {
		request = time.Now()
		timeout.Reset(d.requestTTL())

		if skeleton {
			glog.V(logger.Detail).Infof("%v: fetching %d skeleton headers from #%d", p, MaxHeaderFetch, from)
			go p.getAbsHeaders(from+uint64(MaxHeaderFetch)-1, MaxSkeletonSize, MaxHeaderFetch-1, false)
		} else {
			glog.V(logger.Detail).Infof("%v: fetching %d full headers from #%d", p, MaxHeaderFetch, from)
			go p.getAbsHeaders(from, MaxHeaderFetch, 0, false)
		}
	}
	// Start pulling the header chain skeleton until all is done
	getHeaders(from)

	for {
		select {
		case <-d.cancelCh:
			return errCancelHeaderFetch

		case packet := <-d.headerCh:
			// Make sure the active peer is giving us the skeleton headers
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received skeleton headers from incorrect peer (%s)", packet.PeerId())
				break
			}
			headerReqTimer.UpdateSince(request)
			timeout.Stop()

			// If the skeleton's finished, pull any remaining head headers directly from the origin
			if packet.Items() == 0 && skeleton {
				skeleton = false
				getHeaders(from)
				continue
			}
			// If no more headers are inbound, notify the content fetchers and return
			if packet.Items() == 0 {
				glog.V(logger.Debug).Infof("%v: no available headers", p)
				select {
				case d.headerProcCh <- nil:
					return nil
				case <-d.cancelCh:
					return errCancelHeaderFetch
				}
			}
			headers := packet.(*headerPack).headers

			// If we received a skeleton batch, resolve internals concurrently
			if skeleton {
				filled, proced, err := d.fillHeaderSkeleton(from, headers)
				if err != nil {
					glog.V(logger.Debug).Infof("%v: skeleton chain invalid: %v", p, err)
					return errInvalidChain
				}
				headers = filled[proced:]
				from += uint64(proced)
			}
			// Insert all the new headers and fetch the next batch
			if len(headers) > 0 {
				glog.V(logger.Detail).Infof("%v: schedule %d headers from #%d", p, len(headers), from)
				select {
				case d.headerProcCh <- headers:
				case <-d.cancelCh:
					return errCancelHeaderFetch
				}
				from += uint64(len(headers))
			}
			getHeaders(from)

		case <-timeout.C:
			// Header retrieval timed out, consider the peer bad and drop
			glog.V(logger.Debug).Infof("%v: header request timed out", p)
			headerTimeoutMeter.Mark(1)
			d.dropPeer(p.id)

			// Finish the sync gracefully instead of dumping the gathered data though
			for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh, d.stateWakeCh} {
				select {
				case ch <- false:
				case <-d.cancelCh:
				}
			}
			select {
			case d.headerProcCh <- nil:
			case <-d.cancelCh:
			}
			return errBadPeer

		case <-d.hashCh:
		case <-d.blockCh:
			// Ignore eth/61 packets because this is eth/62+.
			// These can arrive as a late delivery from a previous sync.
		}
	}
}

// fillHeaderSkeleton concurrently retrieves headers from all our available peers
// and maps them to the provided skeleton header chain.
//
// Any partial results from the beginning of the skeleton is (if possible) forwarded
// immediately to the header processor to keep the rest of the pipeline full even
// in the case of header stalls.
//
// The method returs the entire filled skeleton and also the number of headers
// already forwarded for processing.
func (d *Downloader) fillHeaderSkeleton(from uint64, skeleton []*types.Header) ([]*types.Header, int, error) {
	glog.V(logger.Debug).Infof("Filling up skeleton from #%d", from)
	d.queue.ScheduleSkeleton(from, skeleton)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*headerPack)
			return d.queue.DeliverHeaders(pack.peerId, pack.headers, d.headerProcCh)
		}
		expire   = func() map[string]int { return d.queue.ExpireHeaders(d.requestTTL()) }
		throttle = func() bool { return false }
		reserve  = func(p *peer, count int) (*fetchRequest, bool, error) {
			return d.queue.ReserveHeaders(p, count), false, nil
		}
		fetch    = func(p *peer, req *fetchRequest) error { return p.FetchHeaders(req.From, MaxHeaderFetch) }
		capacity = func(p *peer) int { return p.HeaderCapacity(d.requestRTT()) }
		setIdle  = func(p *peer, accepted int) { p.SetHeadersIdle(accepted) }
	)
	err := d.fetchParts(errCancelHeaderFetch, d.headerCh, deliver, d.queue.headerContCh, expire,
		d.queue.PendingHeaders, d.queue.InFlightHeaders, throttle, reserve,
		nil, fetch, d.queue.CancelHeaders, capacity, d.peers.HeaderIdlePeers, setIdle, "Header")

	glog.V(logger.Debug).Infof("Skeleton fill terminated: %v", err)

	filled, proced := d.queue.RetrieveHeaders()
	return filled, proced, err
}

// fetchBodies iteratively downloads the scheduled block bodies, taking any
// available peers, reserving a chunk of blocks for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchBodies(from uint64) error {
	glog.V(logger.Debug).Infof("Downloading block bodies from #%d", from)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*bodyPack)
			return d.queue.DeliverBodies(pack.peerId, pack.transactions, pack.uncles)
		}
		expire   = func() map[string]int { return d.queue.ExpireBodies(d.requestTTL()) }
		fetch    = func(p *peer, req *fetchRequest) error { return p.FetchBodies(req) }
		capacity = func(p *peer) int { return p.BlockCapacity(d.requestRTT()) }
		setIdle  = func(p *peer, accepted int) { p.SetBodiesIdle(accepted) }
	)
	err := d.fetchParts(errCancelBodyFetch, d.bodyCh, deliver, d.bodyWakeCh, expire,
		d.queue.PendingBlocks, d.queue.InFlightBlocks, d.queue.ShouldThrottleBlocks, d.queue.ReserveBodies,
		d.bodyFetchHook, fetch, d.queue.CancelBodies, capacity, d.peers.BodyIdlePeers, setIdle, "Body")

	glog.V(logger.Debug).Infof("Block body download terminated: %v", err)
	return err
}

// fetchReceipts iteratively downloads the scheduled block receipts, taking any
// available peers, reserving a chunk of receipts for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchReceipts(from uint64) error {
	glog.V(logger.Debug).Infof("Downloading receipts from #%d", from)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*receiptPack)
			return d.queue.DeliverReceipts(pack.peerId, pack.receipts)
		}
		expire   = func() map[string]int { return d.queue.ExpireReceipts(d.requestTTL()) }
		fetch    = func(p *peer, req *fetchRequest) error { return p.FetchReceipts(req) }
		capacity = func(p *peer) int { return p.ReceiptCapacity(d.requestRTT()) }
		setIdle  = func(p *peer, accepted int) { p.SetReceiptsIdle(accepted) }
	)
	err := d.fetchParts(errCancelReceiptFetch, d.receiptCh, deliver, d.receiptWakeCh, expire,
		d.queue.PendingReceipts, d.queue.InFlightReceipts, d.queue.ShouldThrottleReceipts, d.queue.ReserveReceipts,
		d.receiptFetchHook, fetch, d.queue.CancelReceipts, capacity, d.peers.ReceiptIdlePeers, setIdle, "Receipt")

	glog.V(logger.Debug).Infof("Receipt download terminated: %v", err)
	return err
}

// fetchNodeData iteratively downloads the scheduled state trie nodes, taking any
// available peers, reserving a chunk of nodes for each, waiting for delivery and
// also periodically checking for timeouts.
func (d *Downloader) fetchNodeData() error {
	glog.V(logger.Debug).Infof("Downloading node state data")

	var (
		deliver = func(packet dataPack) (int, error) {
			start := time.Now()
			return d.queue.DeliverNodeData(packet.PeerId(), packet.(*statePack).states, func(err error, delivered int) {
				// If the peer returned old-requested data, forgive
				if err == trie.ErrNotRequested {
					glog.V(logger.Info).Infof("peer %s: replied to stale state request, forgiving", packet.PeerId())
					return
				}
				if err != nil {
					// If the node data processing failed, the root hash is very wrong, abort
					glog.V(logger.Error).Infof("peer %d: state processing failed: %v", packet.PeerId(), err)
					d.cancel()
					return
				}
				// Processing succeeded, notify state fetcher of continuation
				pending := d.queue.PendingNodeData()
				if pending > 0 {
					select {
					case d.stateWakeCh <- true:
					default:
					}
				}
				d.syncStatsLock.Lock()
				d.syncStatsStateDone += uint64(delivered)
				d.syncStatsLock.Unlock()

				// Log a message to the user and return
				if delivered > 0 {
					glog.V(logger.Info).Infof("imported %d state entries in %v: processed %d, pending at least %d", delivered, time.Since(start), d.syncStatsStateDone, pending)
				}
			})
		}
		expire   = func() map[string]int { return d.queue.ExpireNodeData(d.requestTTL()) }
		throttle = func() bool { return false }
		reserve  = func(p *peer, count int) (*fetchRequest, bool, error) {
			return d.queue.ReserveNodeData(p, count), false, nil
		}
		fetch    = func(p *peer, req *fetchRequest) error { return p.FetchNodeData(req) }
		capacity = func(p *peer) int { return p.NodeDataCapacity(d.requestRTT()) }
		setIdle  = func(p *peer, accepted int) { p.SetNodeDataIdle(accepted) }
	)
	err := d.fetchParts(errCancelStateFetch, d.stateCh, deliver, d.stateWakeCh, expire,
		d.queue.PendingNodeData, d.queue.InFlightNodeData, throttle, reserve, nil, fetch,
		d.queue.CancelNodeData, capacity, d.peers.NodeDataIdlePeers, setIdle, "State")

	glog.V(logger.Debug).Infof("Node state data download terminated: %v", err)
	return err
}

// fetchParts iteratively downloads scheduled block parts, taking any available
// peers, reserving a chunk of fetch requests for each, waiting for delivery and
// also periodically checking for timeouts.
//
// As the scheduling/timeout logic mostly is the same for all downloaded data
// types, this method is used by each for data gathering and is instrumented with
// various callbacks to handle the slight differences between processing them.
//
// The instrumentation parameters:
//  - errCancel:   error type to return if the fetch operation is cancelled (mostly makes logging nicer)
//  - deliveryCh:  channel from which to retrieve downloaded data packets (merged from all concurrent peers)
//  - deliver:     processing callback to deliver data packets into type specific download queues (usually within `queue`)
//  - wakeCh:      notification channel for waking the fetcher when new tasks are available (or sync completed)
//  - expire:      task callback method to abort requests that took too long and return the faulty peers (traffic shaping)
//  - pending:     task callback for the number of requests still needing download (detect completion/non-completability)
//  - inFlight:    task callback for the number of in-progress requests (wait for all active downloads to finish)
//  - throttle:    task callback to check if the processing queue is full and activate throttling (bound memory use)
//  - reserve:     task callback to reserve new download tasks to a particular peer (also signals partial completions)
//  - fetchHook:   tester callback to notify of new tasks being initiated (allows testing the scheduling logic)
//  - fetch:       network callback to actually send a particular download request to a physical remote peer
//  - cancel:      task callback to abort an in-flight download request and allow rescheduling it (in case of lost peer)
//  - capacity:    network callback to retreive the estimated type-specific bandwidth capacity of a peer (traffic shaping)
//  - idle:        network callback to retrieve the currently (type specific) idle peers that can be assigned tasks
//  - setIdle:     network callback to set a peer back to idle and update its estimated capacity (traffic shaping)
//  - kind:        textual label of the type being downloaded to display in log mesages
func (d *Downloader) fetchParts(errCancel error, deliveryCh chan dataPack, deliver func(dataPack) (int, error), wakeCh chan bool,
	expire func() map[string]int, pending func() int, inFlight func() bool, throttle func() bool, reserve func(*peer, int) (*fetchRequest, bool, error),
	fetchHook func([]*types.Header), fetch func(*peer, *fetchRequest) error, cancel func(*fetchRequest), capacity func(*peer) int,
	idle func() ([]*peer, int), setIdle func(*peer, int), kind string) error {

	// Create a ticker to detect expired retrieval tasks
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	update := make(chan struct{}, 1)

	// Prepare the queue and fetch block parts until the block header fetcher's done
	finished := false
	for {
		select {
		case <-d.cancelCh:
			return errCancel

		case packet := <-deliveryCh:
			// If the peer was previously banned and failed to deliver it's pack
			// in a reasonable time frame, ignore it's message.
			if peer := d.peers.Peer(packet.PeerId()); peer != nil {
				// Deliver the received chunk of data and check chain validity
				accepted, err := deliver(packet)
				if err == errInvalidChain {
					return err
				}
				// Unless a peer delivered something completely else than requested (usually
				// caused by a timed out request which came through in the end), set it to
				// idle. If the delivery's stale, the peer should have already been idled.
				if err != errStaleDelivery {
					setIdle(peer, accepted)
				}
				// Issue a log to the user to see what's going on
				switch {
				case err == nil && packet.Items() == 0:
					glog.V(logger.Detail).Infof("%s: no %s delivered", peer, strings.ToLower(kind))
				case err == nil:
					glog.V(logger.Detail).Infof("%s: delivered %s %s(s)", peer, packet.Stats(), strings.ToLower(kind))
				default:
					glog.V(logger.Detail).Infof("%s: %s delivery failed: %v", peer, strings.ToLower(kind), err)
				}
			}
			// Blocks assembled, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case cont := <-wakeCh:
			// The header fetcher sent a continuation flag, check if it's done
			if !cont {
				finished = true
			}
			// Headers arrive, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-ticker.C:
			// Sanity check update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-update:
			// Short circuit if we lost all our peers
			if d.peers.Len() == 0 {
				return errNoPeers
			}
			// Check for fetch request timeouts and demote the responsible peers
			for pid, fails := range expire() {
				if peer := d.peers.Peer(pid); peer != nil {
					if fails > 1 {
						glog.V(logger.Detail).Infof("%s: %s delivery timeout", peer, strings.ToLower(kind))
						setIdle(peer, 0)
					} else {
						glog.V(logger.Debug).Infof("%s: stalling %s delivery, dropping", peer, strings.ToLower(kind))
						d.dropPeer(pid)
					}
				}
			}
			// If there's nothing more to fetch, wait or terminate
			if pending() == 0 {
				if !inFlight() && finished {
					glog.V(logger.Debug).Infof("%s fetching completed", kind)
					return nil
				}
				break
			}
			// Send a download request to all idle peers, until throttled
			progressed, throttled, running := false, false, inFlight()
			idles, total := idle()

			for _, peer := range idles {
				// Short circuit if throttling activated
				if throttle() {
					throttled = true
					break
				}
				// Reserve a chunk of fetches for a peer. A nil can mean either that
				// no more headers are available, or that the peer is known not to
				// have them.
				request, progress, err := reserve(peer, capacity(peer))
				if err != nil {
					return err
				}
				if progress {
					progressed = true
				}
				if request == nil {
					continue
				}
				if glog.V(logger.Detail) {
					if request.From > 0 {
						glog.Infof("%s: requesting %s(s) from #%d", peer, strings.ToLower(kind), request.From)
					} else if len(request.Headers) > 0 {
						glog.Infof("%s: requesting %d %s(s), first at #%d", peer, len(request.Headers), strings.ToLower(kind), request.Headers[0].Number)
					} else {
						glog.Infof("%s: requesting %d %s(s)", peer, len(request.Hashes), strings.ToLower(kind))
					}
				}
				// Fetch the chunk and make sure any errors return the hashes to the queue
				if fetchHook != nil {
					fetchHook(request.Headers)
				}
				if err := fetch(peer, request); err != nil {
					// Although we could try and make an attempt to fix this, this error really
					// means that we've double allocated a fetch task to a peer. If that is the
					// case, the internal state of the downloader and the queue is very wrong so
					// better hard crash and note the error instead of silently accumulating into
					// a much bigger issue.
					panic(fmt.Sprintf("%v: %s fetch assignment failed", peer, strings.ToLower(kind)))
				}
				running = true
			}
			// Make sure that we have peers available for fetching. If all peers have been tried
			// and all failed throw an error
			if !progressed && !throttled && !running && len(idles) == total && pending() > 0 {
				return errPeersUnavailable
			}

		case <-d.hashCh:
		case <-d.blockCh:
			// Ignore eth/61 packets because this is eth/62+.
			// These can arrive as a late delivery from a previous sync.
		}
	}
}

// processHeaders takes batches of retrieved headers from an input channel and
// keeps processing and scheduling them into the header chain and downloader's
// queue until the stream ends or a failure occurs.
func (d *Downloader) processHeaders(origin uint64, td *big.Int) error {
	// Calculate the pivoting point for switching from fast to slow sync
	pivot := d.queue.FastSyncPivot()

	// Keep a count of uncertain headers to roll back
	rollback := []*types.Header{}
	defer func() {
		if len(rollback) > 0 {
			// Flatten the headers and roll them back
			hashes := make([]common.Hash, len(rollback))
			for i, header := range rollback {
				hashes[i] = header.Hash()
			}
			lastHeader, lastFastBlock, lastBlock := d.headHeader().Number, common.Big0, common.Big0
			if d.headFastBlock != nil {
				lastFastBlock = d.headFastBlock().Number()
			}
			if d.headBlock != nil {
				lastBlock = d.headBlock().Number()
			}
			d.rollback(hashes)
			curFastBlock, curBlock := common.Big0, common.Big0
			if d.headFastBlock != nil {
				curFastBlock = d.headFastBlock().Number()
			}
			if d.headBlock != nil {
				curBlock = d.headBlock().Number()
			}
			glog.V(logger.Warn).Infof("Rolled back %d headers (LH: %d->%d, FB: %d->%d, LB: %d->%d)",
				len(hashes), lastHeader, d.headHeader().Number, lastFastBlock, curFastBlock, lastBlock, curBlock)

			// If we're already past the pivot point, this could be an attack, thread carefully
			if rollback[len(rollback)-1].Number.Uint64() > pivot {
				// If we didn't ever fail, lock in te pivot header (must! not! change!)
				if d.fsPivotFails == 0 {
					for _, header := range rollback {
						if header.Number.Uint64() == pivot {
							glog.V(logger.Warn).Infof("Fast-sync critical section failure, locked pivot to header #%d [%x…]", pivot, header.Hash().Bytes()[:4])
							d.fsPivotLock = header
						}
					}
				}
				d.fsPivotFails++
			}
		}
	}()

	// Wait for batches of headers to process
	gotHeaders := false

	for {
		select {
		case <-d.cancelCh:
			return errCancelHeaderProcessing

		case headers := <-d.headerProcCh:
			// Terminate header processing if we synced up
			if len(headers) == 0 {
				// Notify everyone that headers are fully processed
				for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh, d.stateWakeCh} {
					select {
					case ch <- false:
					case <-d.cancelCh:
					}
				}
				// If no headers were retrieved at all, the peer violated it's TD promise that it had a
				// better chain compared to ours. The only exception is if it's promised blocks were
				// already imported by other means (e.g. fecher):
				//
				// R <remote peer>, L <local node>: Both at block 10
				// R: Mine block 11, and propagate it to L
				// L: Queue block 11 for import
				// L: Notice that R's head and TD increased compared to ours, start sync
				// L: Import of block 11 finishes
				// L: Sync begins, and finds common ancestor at 11
				// L: Request new headers up from 11 (R's TD was higher, it must have something)
				// R: Nothing to give
				if d.mode != LightSync {
					if !gotHeaders && td.Cmp(d.getTd(d.headBlock().Hash())) > 0 {
						return errStallingPeer
					}
				}
				// If fast or light syncing, ensure promised headers are indeed delivered. This is
				// needed to detect scenarios where an attacker feeds a bad pivot and then bails out
				// of delivering the post-pivot blocks that would flag the invalid content.
				//
				// This check cannot be executed "as is" for full imports, since blocks may still be
				// queued for processing when the header download completes. However, as long as the
				// peer gave us something useful, we're already happy/progressed (above check).
				if d.mode == FastSync || d.mode == LightSync {
					if td.Cmp(d.getTd(d.headHeader().Hash())) > 0 {
						return errStallingPeer
					}
				}
				// Disable any rollback and return
				rollback = nil
				return nil
			}
			// Otherwise split the chunk of headers into batches and process them
			gotHeaders = true

			for len(headers) > 0 {
				// Terminate if something failed in between processing chunks
				select {
				case <-d.cancelCh:
					return errCancelHeaderProcessing
				default:
				}
				// Select the next chunk of headers to import
				limit := maxHeadersProcess
				if limit > len(headers) {
					limit = len(headers)
				}
				chunk := headers[:limit]

				// In case of header only syncing, validate the chunk immediately
				if d.mode == FastSync || d.mode == LightSync {
					// Collect the yet unknown headers to mark them as uncertain
					unknown := make([]*types.Header, 0, len(headers))
					for _, header := range chunk {
						if !d.hasHeader(header.Hash()) {
							unknown = append(unknown, header)
						}
					}
					// If we're importing pure headers, verify based on their recentness
					frequency := fsHeaderCheckFrequency
					if chunk[len(chunk)-1].Number.Uint64()+uint64(fsHeaderForceVerify) > pivot {
						frequency = 1
					}
					if n, err := d.insertHeaders(chunk, frequency); err != nil {
						// If some headers were inserted, add them too to the rollback list
						if n > 0 {
							rollback = append(rollback, chunk[:n]...)
						}
						glog.V(logger.Debug).Infof("invalid header #%d [%x…]: %v", chunk[n].Number, chunk[n].Hash().Bytes()[:4], err)
						return errInvalidChain
					}
					// All verifications passed, store newly found uncertain headers
					rollback = append(rollback, unknown...)
					if len(rollback) > fsHeaderSafetyNet {
						rollback = append(rollback[:0], rollback[len(rollback)-fsHeaderSafetyNet:]...)
					}
				}
				// If we're fast syncing and just pulled in the pivot, make sure it's the one locked in
				if d.mode == FastSync && d.fsPivotLock != nil && chunk[0].Number.Uint64() <= pivot && chunk[len(chunk)-1].Number.Uint64() >= pivot {
					if pivot := chunk[int(pivot-chunk[0].Number.Uint64())]; pivot.Hash() != d.fsPivotLock.Hash() {
						glog.V(logger.Warn).Infof("Pivot doesn't match locked in version: have #%v [%x…], want #%v [%x…]", pivot.Number, pivot.Hash().Bytes()[:4], d.fsPivotLock.Number, d.fsPivotLock.Hash().Bytes()[:4])
						return errInvalidChain
					}
				}
				// Unless we're doing light chains, schedule the headers for associated content retrieval
				if d.mode == FullSync || d.mode == FastSync {
					// If we've reached the allowed number of pending headers, stall a bit
					for d.queue.PendingBlocks() >= maxQueuedHeaders || d.queue.PendingReceipts() >= maxQueuedHeaders {
						select {
						case <-d.cancelCh:
							return errCancelHeaderProcessing
						case <-time.After(time.Second):
						}
					}
					// Otherwise insert the headers for content retrieval
					inserts := d.queue.Schedule(chunk, origin)
					if len(inserts) != len(chunk) {
						glog.V(logger.Debug).Infof("stale headers")
						return errBadPeer
					}
				}
				headers = headers[limit:]
				origin += uint64(limit)
			}
			// Signal the content downloaders of the availablility of new tasks
			for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh, d.stateWakeCh} {
				select {
				case ch <- true:
				default:
				}
			}
		}
	}
}

// processContent takes fetch results from the queue and tries to import them
// into the chain. The type of import operation will depend on the result contents.
func (d *Downloader) processContent() error {
	pivot := d.queue.FastSyncPivot()
	for {
		results := d.queue.WaitResults()
		if len(results) == 0 {
			return nil // queue empty
		}
		if d.chainInsertHook != nil {
			d.chainInsertHook(results)
		}
		// Actually import the blocks
		if glog.V(logger.Debug) {
			first, last := results[0].Header, results[len(results)-1].Header
			glog.Infof("Inserting chain with %d items (#%d [%x…] - #%d [%x…])", len(results), first.Number, first.Hash().Bytes()[:4], last.Number, last.Hash().Bytes()[:4])
		}
		for len(results) != 0 {
			// Check for any termination requests
			select {
			case <-d.quitCh:
				return errCancelContentProcessing
			default:
			}
			// Retrieve the a batch of results to import
			var (
				blocks   = make([]*types.Block, 0, maxResultsProcess)
				receipts = make([]types.Receipts, 0, maxResultsProcess)
			)
			items := int(math.Min(float64(len(results)), float64(maxResultsProcess)))
			for _, result := range results[:items] {
				switch {
				case d.mode == FullSync:
					blocks = append(blocks, types.NewBlockWithHeader(result.Header).WithBody(result.Transactions, result.Uncles))
				case d.mode == FastSync:
					blocks = append(blocks, types.NewBlockWithHeader(result.Header).WithBody(result.Transactions, result.Uncles))
					if result.Header.Number.Uint64() <= pivot {
						receipts = append(receipts, result.Receipts)
					}
				}
			}
			// Try to process the results, aborting if there's an error
			var (
				err   error
				index int
			)
			switch {
			case len(receipts) > 0:
				index, err = d.insertReceipts(blocks, receipts)
				if err == nil && blocks[len(blocks)-1].NumberU64() == pivot {
					glog.V(logger.Debug).Infof("Committing block #%d [%x…] as the new head", blocks[len(blocks)-1].Number(), blocks[len(blocks)-1].Hash().Bytes()[:4])
					index, err = len(blocks)-1, d.commitHeadBlock(blocks[len(blocks)-1].Hash())
				}
			default:
				index, err = d.insertBlocks(blocks)
			}
			if err != nil {
				glog.V(logger.Debug).Infof("Result #%d [%x…] processing failed: %v", results[index].Header.Number, results[index].Header.Hash().Bytes()[:4], err)
				return errInvalidChain
			}
			// Shift the results to the next batch
			results = results[items:]
		}
	}
}

// DeliverHashes injects a new batch of hashes received from a remote node into
// the download schedule. This is usually invoked through the BlockHashesMsg by
// the protocol handler.
func (d *Downloader) DeliverHashes(id string, hashes []common.Hash) (err error) {
	return d.deliver(id, d.hashCh, &hashPack{id, hashes}, hashInMeter, hashDropMeter)
}

// DeliverBlocks injects a new batch of blocks received from a remote node.
// This is usually invoked through the BlocksMsg by the protocol handler.
func (d *Downloader) DeliverBlocks(id string, blocks []*types.Block) (err error) {
	return d.deliver(id, d.blockCh, &blockPack{id, blocks}, blockInMeter, blockDropMeter)
}

// DeliverHeaders injects a new batch of block headers received from a remote
// node into the download schedule.
func (d *Downloader) DeliverHeaders(id string, headers []*types.Header) (err error) {
	return d.deliver(id, d.headerCh, &headerPack{id, headers}, headerInMeter, headerDropMeter)
}

// DeliverBodies injects a new batch of block bodies received from a remote node.
func (d *Downloader) DeliverBodies(id string, transactions [][]*types.Transaction, uncles [][]*types.Header) (err error) {
	return d.deliver(id, d.bodyCh, &bodyPack{id, transactions, uncles}, bodyInMeter, bodyDropMeter)
}

// DeliverReceipts injects a new batch of receipts received from a remote node.
func (d *Downloader) DeliverReceipts(id string, receipts [][]*types.Receipt) (err error) {
	return d.deliver(id, d.receiptCh, &receiptPack{id, receipts}, receiptInMeter, receiptDropMeter)
}

// DeliverNodeData injects a new batch of node state data received from a remote node.
func (d *Downloader) DeliverNodeData(id string, data [][]byte) (err error) {
	return d.deliver(id, d.stateCh, &statePack{id, data}, stateInMeter, stateDropMeter)
}

// deliver injects a new batch of data received from a remote node.
func (d *Downloader) deliver(id string, destCh chan dataPack, packet dataPack, inMeter, dropMeter metrics.Meter) (err error) {
	// Update the delivery metrics for both good and failed deliveries
	inMeter.Mark(int64(packet.Items()))
	defer func() {
		if err != nil {
			dropMeter.Mark(int64(packet.Items()))
		}
	}()
	// Deliver or abort if the sync is canceled while queuing
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()
	if cancel == nil {
		return errNoSyncActive
	}
	select {
	case destCh <- packet:
		return nil
	case <-cancel:
		return errNoSyncActive
	}
}

// qosTuner is the quality of service tuning loop that occasionally gathers the
// peer latency statistics and updates the estimated request round trip time.
func (d *Downloader) qosTuner() {
	for {
		// Retrieve the current median RTT and integrate into the previoust target RTT
		rtt := time.Duration(float64(1-qosTuningImpact)*float64(atomic.LoadUint64(&d.rttEstimate)) + qosTuningImpact*float64(d.peers.medianRTT()))
		atomic.StoreUint64(&d.rttEstimate, uint64(rtt))

		// A new RTT cycle passed, increase our confidence in the estimated RTT
		conf := atomic.LoadUint64(&d.rttConfidence)
		conf = conf + (1000000-conf)/2
		atomic.StoreUint64(&d.rttConfidence, conf)

		// Log the new QoS values and sleep until the next RTT
		glog.V(logger.Debug).Infof("Quality of service: rtt %v, conf %.3f, ttl %v", rtt, float64(conf)/1000000.0, d.requestTTL())
		select {
		case <-d.quitCh:
			return
		case <-time.After(rtt):
		}
	}
}

// qosReduceConfidence is meant to be called when a new peer joins the downloader's
// peer set, needing to reduce the confidence we have in out QoS estimates.
func (d *Downloader) qosReduceConfidence() {
	// If we have a single peer, confidence is always 1
	peers := uint64(d.peers.Len())
	if peers == 1 {
		atomic.StoreUint64(&d.rttConfidence, 1000000)
		return
	}
	// If we have a ton of peers, don't drop confidence)
	if peers >= uint64(qosConfidenceCap) {
		return
	}
	// Otherwise drop the confidence factor
	conf := atomic.LoadUint64(&d.rttConfidence) * (peers - 1) / peers
	if float64(conf)/1000000 < rttMinConfidence {
		conf = uint64(rttMinConfidence * 1000000)
	}
	atomic.StoreUint64(&d.rttConfidence, conf)

	rtt := time.Duration(atomic.LoadUint64(&d.rttEstimate))
	glog.V(logger.Debug).Infof("Quality of service: rtt %v, conf %.3f, ttl %v", rtt, float64(conf)/1000000.0, d.requestTTL())
}

// requestRTT returns the current target round trip time for a download request
// to complete in.
//
// Note, the returned RTT is .9 of the actually estimated RTT. The reason is that
// the downloader tries to adapt queries to the RTT, so multiple RTT values can
// be adapted to, but smaller ones are preffered (stabler download stream).
func (d *Downloader) requestRTT() time.Duration {
	return time.Duration(atomic.LoadUint64(&d.rttEstimate)) * 9 / 10
}

// requestTTL returns the current timeout allowance for a single download request
// to finish under.
func (d *Downloader) requestTTL() time.Duration {
	var (
		rtt  = time.Duration(atomic.LoadUint64(&d.rttEstimate))
		conf = float64(atomic.LoadUint64(&d.rttConfidence)) / 1000000.0
	)
	ttl := time.Duration(ttlScaling) * time.Duration(float64(rtt)/conf)
	if ttl > ttlLimit {
		ttl = ttlLimit
	}
	return ttl
}
