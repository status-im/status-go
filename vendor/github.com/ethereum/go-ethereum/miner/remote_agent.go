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

package miner

import (
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type hashrate struct {
	ping time.Time
	rate uint64
}

type RemoteAgent struct {
	mu sync.Mutex

	quitCh   chan struct{}
	workCh   chan *Work
	returnCh chan<- *Result

	chain       consensus.ChainReader
	engine      consensus.Engine
	currentWork *Work
	work        map[common.Hash]*Work

	hashrateMu sync.RWMutex
	hashrate   map[common.Hash]hashrate

	running int32 // running indicates whether the agent is active. Call atomically
}

func NewRemoteAgent(chain consensus.ChainReader, engine consensus.Engine) *RemoteAgent {
	return &RemoteAgent{
		chain:    chain,
		engine:   engine,
		work:     make(map[common.Hash]*Work),
		hashrate: make(map[common.Hash]hashrate),
	}
}

func (a *RemoteAgent) SubmitHashrate(id common.Hash, rate uint64) {
	a.hashrateMu.Lock()
	defer a.hashrateMu.Unlock()

	a.hashrate[id] = hashrate{time.Now(), rate}
}

func (a *RemoteAgent) Work() chan<- *Work {
	return a.workCh
}

func (a *RemoteAgent) SetReturnCh(returnCh chan<- *Result) {
	a.returnCh = returnCh
}

func (a *RemoteAgent) Start() {
	if !atomic.CompareAndSwapInt32(&a.running, 0, 1) {
		return
	}
	a.quitCh = make(chan struct{})
	a.workCh = make(chan *Work, 1)
	go a.loop(a.workCh, a.quitCh)
}

func (a *RemoteAgent) Stop() {
	if !atomic.CompareAndSwapInt32(&a.running, 1, 0) {
		return
	}
	close(a.quitCh)
	close(a.workCh)
}

// GetHashRate returns the accumulated hashrate of all identifier combined
func (a *RemoteAgent) GetHashRate() (tot int64) {
	a.hashrateMu.RLock()
	defer a.hashrateMu.RUnlock()

	// this could overflow
	for _, hashrate := range a.hashrate {
		tot += int64(hashrate.rate)
	}
	return
}

func (a *RemoteAgent) GetWork() ([3]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var res [3]string

	if a.currentWork != nil {
		block := a.currentWork.Block

		res[0] = block.HashNoNonce().Hex()
		seedHash := ethash.SeedHash(block.NumberU64())
		res[1] = common.BytesToHash(seedHash).Hex()
		// Calculate the "target" to be returned to the external miner
		n := big.NewInt(1)
		n.Lsh(n, 255)
		n.Div(n, block.Difficulty())
		n.Lsh(n, 1)
		res[2] = common.BytesToHash(n.Bytes()).Hex()

		a.work[block.HashNoNonce()] = a.currentWork
		return res, nil
	}
	return res, errors.New("No work available yet, don't panic.")
}

// SubmitWork tries to inject a pow solution into the remote agent, returning
// whether the solution was accepted or not (not can be both a bad pow as well as
// any other error, like no work pending).
func (a *RemoteAgent) SubmitWork(nonce types.BlockNonce, mixDigest, hash common.Hash) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Make sure the work submitted is present
	work := a.work[hash]
	if work == nil {
		log.Info("Work submitted but none pending", "hash", hash)
		return false
	}
	// Make sure the Engine solutions is indeed valid
	result := work.Block.Header()
	result.Nonce = nonce
	result.MixDigest = mixDigest

	if err := a.engine.VerifySeal(a.chain, result); err != nil {
		log.Warn("Invalid proof-of-work submitted", "hash", hash, "err", err)
		return false
	}
	block := work.Block.WithSeal(result)

	// Solutions seems to be valid, return to the miner and notify acceptance
	a.returnCh <- &Result{work, block}
	delete(a.work, hash)

	return true
}

// loop monitors mining events on the work and quit channels, updating the internal
// state of the remote miner until a termination is requested.
//
// Note, the reason the work and quit channels are passed as parameters is because
// RemoteAgent.Start() constantly recreates these channels, so the loop code cannot
// assume data stability in these member fields.
func (a *RemoteAgent) loop(workCh chan *Work, quitCh chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quitCh:
			return
		case work := <-workCh:
			a.mu.Lock()
			a.currentWork = work
			a.mu.Unlock()
		case <-ticker.C:
			// cleanup
			a.mu.Lock()
			for hash, work := range a.work {
				if time.Since(work.createdAt) > 7*(12*time.Second) {
					delete(a.work, hash)
				}
			}
			a.mu.Unlock()

			a.hashrateMu.Lock()
			for id, hashrate := range a.hashrate {
				if time.Since(hashrate.ping) > 10*time.Second {
					delete(a.hashrate, id)
				}
			}
			a.hashrateMu.Unlock()
		}
	}
}
