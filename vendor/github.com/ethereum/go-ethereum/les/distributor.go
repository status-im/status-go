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

// Package light implements on-demand retrieval capable state and chain objects
// for the Ethereum Light Client.
package les

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

// ErrNoPeers is returned if no peers capable of serving a queued request are
// available for a certain amount of time (noPeersTimeout)
var (
	ErrNoPeers     = errors.New("no suitable peers available")
	noPeersTimeout = time.Second * 5
)

// requestDistributor implements a mechanism that distributes requests to
// suitable peers, obeying flow control rules and prioritizing them in creation
// order (even when a resend is necessary).
type requestDistributor struct {
	queueFirst, queueLast   *distReq
	lastReqOrder            uint64
	stopChn                 chan struct{}
	loopChn                 chan bool
	loopRunning, loopWakeup bool
	lock                    sync.Mutex

	getAllPeers func() map[distPeer]struct{}
}

// distPeer is an LES server peer interface for the request distributor.
// waitBefore returns either the necessary waiting time before sending a request
// with the given upper estimated cost or the estimated remaining relative buffer
// value after sending such a request (in which case the request can be sent
// immediately). At least one of these values is always zero.
type distPeer interface {
	waitBefore(uint64) (time.Duration, float64)
	canSendOrdered() bool
	orderedSend(f func())
}

// distReq is the request abstraction used by the distributor. It is based on
// three callback functions:
// - getCost returns the upper estimate of the cost of sending the request to a given peer
// - canSend tells if the server peer is suitable to serve the request
// - request prepares sending the request to the given peer and returns a function that
// does the actual sending. Request order should be preserved but the callback itself should not
// block until it is sent because other peers might still be able to receive requests while
// one of them is blocking. Instead, the returned function is put in the peer's orderedSend queue.
type distReq struct {
	getCost func(distPeer) uint64
	canSend func(distPeer) bool
	request func(distPeer) func()

	reqOrder uint64
	// only for queued requests
	noPeersTime          mclock.AbsTime
	queuePrev, queueNext *distReq
	sentChn              chan distPeer
}

// newRequestDistributor creates a new request distributor
func newRequestDistributor(getAllPeers func() map[distPeer]struct{}, stopChn chan struct{}) *requestDistributor {
	r := &requestDistributor{
		loopChn:     make(chan bool, 2),
		stopChn:     stopChn,
		getAllPeers: getAllPeers,
	}
	go r.loop()
	return r
}

// newDistReq creates a new request instance that can be queued for distribution
func newDistReq(getCost func(distPeer) uint64, canSend func(distPeer) bool, request func(distPeer) func()) *distReq {
	return &distReq{
		getCost: getCost,
		canSend: canSend,
		request: request,
	}
}

// distMaxWait is the maximum waiting time after which further necessary waiting
// times are recalculated based on new feedback from the servers
const distMaxWait = time.Millisecond * 10

// main event loop
func (d *requestDistributor) loop() {
mainLoop:
	for {
		select {
		case <-d.stopChn:
			d.lock.Lock()
			req := d.queueFirst
			for req != nil {
				close(req.sentChn)
				req = req.queueNext
			}
			d.lock.Unlock()
			return
		case wakeup := <-d.loopChn:
			d.lock.Lock()
			if wakeup {
				d.loopWakeup = false
				if d.loopRunning {
					d.lock.Unlock()
					continue mainLoop
				}
			}
			d.loopRunning = false
		loop:
			for {
				peer, req, wait := d.nextRequest()
				if req != nil && wait == 0 {
					chn := req.sentChn
					d.remove(req)
					send := req.request(peer)
					if send != nil {
						peer.orderedSend(send)
					}
					chn <- peer
					close(chn)
				} else {
					if wait == 0 {
						if d.queueFirst == nil || d.loopWakeup {
							break loop
						}
						d.loopWakeup = true
						wait = retryPeers
					} else {
						d.loopRunning = true
						if wait > distMaxWait {
							wait = distMaxWait
						}
					}
					go func() {
						time.Sleep(wait)
						d.loopChn <- d.loopWakeup
					}()
					break loop
				}
			}
			d.lock.Unlock()
		}
	}
}

// selectPeerItem represents a peer to be selected for a request by weightedRandomSelect
type selectPeerItem struct {
	peer   distPeer
	req    *distReq
	weight int64
}

// Weight implements wrsItem interface
func (sp selectPeerItem) Weight() int64 {
	return sp.weight
}

// nextRequest returns the next possible request from any peer, along with the
// associated peer and necessary waiting time
func (d *requestDistributor) nextRequest() (distPeer, *distReq, time.Duration) {
	peers := d.getAllPeers()

	req := d.queueFirst
	var (
		bestPeer distPeer
		bestReq  *distReq
		bestWait time.Duration
		sel      *weightedRandomSelect
	)

	now := mclock.Now()

	for (len(peers) > 0 || req == d.queueFirst) && req != nil {
		canSend := false
		for peer, _ := range peers {
			if peer.canSendOrdered() && req.canSend(peer) {
				canSend = true
				cost := req.getCost(peer)
				wait, bufRemain := peer.waitBefore(cost)
				if wait == 0 {
					if sel == nil {
						sel = newWeightedRandomSelect()
					}
					sel.update(selectPeerItem{peer: peer, req: req, weight: int64(bufRemain*1000000) + 1})
				} else {
					if bestReq == nil || wait < bestWait {
						bestPeer = peer
						bestReq = req
						bestWait = wait
					}
				}
				delete(peers, peer)
			}
		}
		next := req.queueNext
		if !canSend && req.noPeersTime == 0 {
			req.noPeersTime = now
		}
		if req == d.queueFirst && !canSend && time.Duration(now-req.noPeersTime) > noPeersTimeout {
			close(req.sentChn)
			d.remove(req)
		}
		req = next
	}

	if sel != nil {
		c := sel.choose().(selectPeerItem)
		return c.peer, c.req, 0
	}
	if bestReq == nil {
	} else {
	}
	return bestPeer, bestReq, bestWait
}

// queue adds a request to the distribution queue, returns a channel where the
// receiving peer is sent once the request has been sent (request callback returned).
// If the request is cancelled or timed out without suitable peers, the channel is
// closed without sending any peer references to it.
func (d *requestDistributor) queue(r *distReq) chan distPeer {
	d.lock.Lock()
	defer d.lock.Unlock()

	if r.reqOrder == 0 {
		d.lastReqOrder++
		r.reqOrder = d.lastReqOrder
	}

	if d.queueLast == nil {
		d.queueFirst = r
		d.queueLast = r
	} else {
		if r.reqOrder > d.queueLast.reqOrder {
			d.queueLast.queueNext = r
			r.queuePrev = d.queueLast
			d.queueLast = r
		} else {
			before := d.queueFirst
			for before.reqOrder < r.reqOrder {
				before = before.queueNext
			}
			r.queueNext = before
			r.queuePrev = before.queuePrev
			r.queueNext.queuePrev = r
			if r.queuePrev == nil {
				d.queueFirst = r
			} else {
				r.queuePrev.queueNext = r
			}
		}
	}

	if !d.loopRunning {
		d.loopRunning = true
		d.loopChn <- false
	}

	r.sentChn = make(chan distPeer, 1)
	return r.sentChn
}

// cancel removes a request from the queue if it has not been sent yet (returns
// false if it has been sent already). It is guaranteed that the callback functions
// will not be called after cancel returns.
func (d *requestDistributor) cancel(r *distReq) bool {
	d.lock.Lock()
	defer d.lock.Unlock()

	if r.sentChn == nil {
		return false
	}

	close(r.sentChn)
	d.remove(r)
	return true
}

// remove removes a request from the queue
func (d *requestDistributor) remove(r *distReq) {
	r.sentChn = nil
	if r.queueNext == nil {
		d.queueLast = r.queuePrev
	} else {
		r.queueNext.queuePrev = r.queuePrev
	}
	if r.queuePrev == nil {
		d.queueFirst = r.queueNext
	} else {
		r.queuePrev.queueNext = r.queueNext
	}
	r.queueNext = nil
	r.queuePrev = nil
}
