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

package discv5

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	errInvalidEvent = errors.New("invalid in current state")
	errNoQuery      = errors.New("no pending query")
	errWrongAddress = errors.New("unknown sender address")
)

const (
	autoRefreshInterval = 1 * time.Hour
	seedCount           = 30
	seedMaxAge          = 5 * 24 * time.Hour
)

const testTopic = "foo"

const (
	printDebugLogs   = false
	printTestImgLogs = false
)

func debugLog(s string) {
	if printDebugLogs {
		fmt.Println(s)
	}
}

// BootNodes are the enode URLs of the P2P bootstrap nodes for the experimental RLPx v5 "Topic Discovery" network
// warning: local bootnodes for testing!!!
var BootNodes = []*Node{
	MustParseNode("enode://fcb7ff7a1437465711900bebc8831b6814d9f3176f3745e8549808af897ede4b51f8e32b52caf1a1c4aee90dc14f2aaeebf8b099f5d19245fd00ab160f59b9c8@127.0.0.1:30001"),
	MustParseNode("enode://9ec7e836e3eff84c5f51b169d8bd26a90e9c0a1c5a5b01125722fbcc7808803451fdde5c31e0fbf32fa730e11bef3a4ec35c751bd896a3db601155c1313d48a2@127.0.0.1:30002"),
	MustParseNode("enode://1949788c417a52653b438a33a2b08a70dc495b24c83d0ae533be13b1c7da0af0159bd0b50823fee4a1bc8f24f81e23435fe167df972256968d881f98f1eeb5de@127.0.0.1:30003"),
	MustParseNode("enode://2a09e932145093688ce0fe885256dfac432402a5546c5b467afe988e205bbaf1a5793f4b625638ba5ba214549965aaacf4881201a82f92bfbaf37e40e7fe7132@127.0.0.1:30004"),
	MustParseNode("enode://fc3d5a1696c48e02a45a78a6e94554f4292265de3c7461dd2b27ec664b09a23613c6f6be9fe7ab18f4788390abd592c272d549c583af6e92b5b8254f335d71bc@127.0.0.1:30005"),
}

// Network manages the table and all protocol interaction.
type Network struct {
	db   *nodeDB // database of known nodes
	conn transport

	closed           chan struct{}          // closed when loop is done
	closeReq         chan struct{}          // 'request to close'
	refreshReq       chan []*Node           // lookups ask for refresh on this channel
	refreshResp      chan (<-chan struct{}) // ...and get the channel to block on from this one
	read             chan ingressPacket     // ingress packets arrive here
	timeout          chan timeoutEvent
	queryReq         chan *findnodeQuery // lookups submit findnode queries on this channel
	tableOpReq       chan func()
	tableOpResp      chan struct{}
	topicRegisterReq chan topicRegisterReq

	// State of the main loop.
	tab           *Table
	topictab      *topicTable
	ticketStore   *ticketStore
	nursery       []*Node
	nodes         map[NodeID]*Node // tracks active nodes with state != known
	timeoutTimers map[timeoutEvent]*time.Timer

	// Revalidation queues.
	// Nodes put on these queues will be pinged eventually.
	slowRevalidateQueue []*Node
	fastRevalidateQueue []*Node

	// Buffers for state transition.
	sendBuf []*ingressPacket
}

// transport is implemented by the UDP transport.
// it is an interface so we can test without opening lots of UDP
// sockets and without generating a private key.
type transport interface {
	sendPing(remote *Node, remoteAddr *net.UDPAddr, topics []Topic) (hash []byte)
	sendNeighbours(remote *Node, nodes []*Node)
	sendFindnodeHash(remote *Node, target common.Hash)
	sendTopicRegister(remote *Node, topics []Topic, topicIdx int, pong []byte)
	sendTopicNodes(remote *Node, queryHash common.Hash, nodes []*Node)

	send(remote *Node, ptype nodeEvent, p interface{})

	localAddr() *net.UDPAddr
	Close()
}

type findnodeQuery struct {
	remote   *Node
	target   common.Hash
	reply    chan<- []*Node
	nresults int // counter for received nodes
}

type topicRegisterReq struct {
	add   bool
	topic Topic
}

type timeoutEvent struct {
	ev   nodeEvent
	node *Node
}

func newNetwork(conn transport, ourPubkey ecdsa.PublicKey, natm nat.Interface, dbPath string) (*Network, error) {
	ourID := PubkeyID(&ourPubkey)

	var db *nodeDB
	if dbPath != "<no database>" {
		var err error
		if db, err = newNodeDB(dbPath, Version, ourID); err != nil {
			return nil, err
		}
	}

	tab := newTable(ourID, conn.localAddr())
	net := &Network{
		db:               db,
		conn:             conn,
		tab:              tab,
		topictab:         newTopicTable(db, tab.self),
		ticketStore:      newTicketStore(),
		refreshReq:       make(chan []*Node),
		refreshResp:      make(chan (<-chan struct{})),
		closed:           make(chan struct{}),
		closeReq:         make(chan struct{}),
		read:             make(chan ingressPacket, 100),
		timeout:          make(chan timeoutEvent),
		timeoutTimers:    make(map[timeoutEvent]*time.Timer),
		tableOpReq:       make(chan func()),
		tableOpResp:      make(chan struct{}),
		queryReq:         make(chan *findnodeQuery),
		topicRegisterReq: make(chan topicRegisterReq),
		nodes:            make(map[NodeID]*Node),
	}
	go net.loop()
	return net, nil
}

// Close terminates the network listener and flushes the node database.
func (net *Network) Close() {
	net.conn.Close()
	select {
	case <-net.closed:
	case net.closeReq <- struct{}{}:
		<-net.closed
	}
}

// Self returns the local node.
// The returned node should not be modified by the caller.
func (net *Network) Self() *Node {
	return net.tab.self
}

// ReadRandomNodes fills the given slice with random nodes from the
// table. It will not write the same node more than once. The nodes in
// the slice are copies and can be modified by the caller.
func (net *Network) ReadRandomNodes(buf []*Node) (n int) {
	net.reqTableOp(func() { n = net.tab.readRandomNodes(buf) })
	return n
}

// SetFallbackNodes sets the initial points of contact. These nodes
// are used to connect to the network if the table is empty and there
// are no known nodes in the database.
func (net *Network) SetFallbackNodes(nodes []*Node) error {
	nursery := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		if err := n.validateComplete(); err != nil {
			return fmt.Errorf("bad bootstrap/fallback node %q (%v)", n, err)
		}
		// Recompute cpy.sha because the node might not have been
		// created by NewNode or ParseNode.
		cpy := *n
		cpy.sha = crypto.Keccak256Hash(n.ID[:])
		nursery = append(nursery, &cpy)
	}
	net.reqRefresh(nursery)
	return nil
}

// Resolve searches for a specific node with the given ID.
// It returns nil if the node could not be found.
func (net *Network) Resolve(targetID NodeID) *Node {
	result := net.lookup(crypto.Keccak256Hash(targetID[:]), true)
	for _, n := range result {
		if n.ID == targetID {
			return n
		}
	}
	return nil
}

// Lookup performs a network search for nodes close
// to the given target. It approaches the target by querying
// nodes that are closer to it on each iteration.
// The given target does not need to be an actual node
// identifier.
//
// The local node may be included in the result.
func (net *Network) Lookup(targetID NodeID) []*Node {
	return net.lookup(crypto.Keccak256Hash(targetID[:]), false)
}

func (net *Network) lookup(target common.Hash, stopOnMatch bool) []*Node {
	var (
		asked          = make(map[NodeID]bool)
		seen           = make(map[NodeID]bool)
		reply          = make(chan []*Node, alpha)
		result         = nodesByDistance{target: target}
		pendingQueries = 0
	)
	// Get initial answers from the local node.
	result.push(net.tab.self, bucketSize)
	for {
		// Ask the α closest nodes that we haven't asked yet.
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.ID] {
				asked[n.ID] = true
				pendingQueries++
				net.reqQueryFindnode(n, target, reply)
			}
		}
		if pendingQueries == 0 {
			// We have asked all closest nodes, stop the search.
			break
		}
		// Wait for the next reply.
		for _, n := range <-reply {
			if n != nil && !seen[n.ID] {
				seen[n.ID] = true
				result.push(n, bucketSize)
				if stopOnMatch && n.sha == target {
					return result.entries
				}
			}
		}
		pendingQueries--
	}
	return result.entries
}

func (net *Network) RegisterTopic(topic Topic, stop <-chan struct{}) {
	select {
	case net.topicRegisterReq <- topicRegisterReq{true, topic}:
	case <-net.closed:
		return
	}
	select {
	case <-net.closed:
	case <-stop:
		select {
		case net.topicRegisterReq <- topicRegisterReq{false, topic}:
		case <-net.closed:
		}
	}
}

func (net *Network) reqRefresh(nursery []*Node) <-chan struct{} {
	select {
	case net.refreshReq <- nursery:
		return <-net.refreshResp
	case <-net.closed:
		return net.closed
	}
}

func (net *Network) reqQueryFindnode(n *Node, target common.Hash, reply chan []*Node) bool {
	q := &findnodeQuery{remote: n, target: target, reply: reply}
	select {
	case net.queryReq <- q:
		return true
	case <-net.closed:
		return false
	}
}

func (net *Network) reqReadPacket(pkt ingressPacket) {
	select {
	case net.read <- pkt:
	case <-net.closed:
	}
}

func (net *Network) reqTableOp(f func()) (called bool) {
	select {
	case net.tableOpReq <- f:
		<-net.tableOpResp
		return true
	case <-net.closed:
		return false
	}
}

// TODO: external address handling.

func (net *Network) loop() {
	var (
		refreshTimer = time.NewTicker(autoRefreshInterval)
		refreshDone  chan struct{} // closed when the 'refresh' lookup has ended
	)

	// Tracking the next ticket to register.
	var (
		nextTicket        *ticketRef
		nextRegisterTimer *time.Timer
		nextRegisterTime  <-chan time.Time
	)
	defer func() {
		if nextRegisterTimer != nil {
			nextRegisterTimer.Stop()
		}
	}()
	resetNextTicket := func() {
		t, timeout := net.ticketStore.nextFilteredTicket()
		if t != nextTicket {
			nextTicket = t
			if nextRegisterTimer != nil {
				nextRegisterTimer.Stop()
				nextRegisterTime = nil
			}
			if t != nil {
				nextRegisterTimer = time.NewTimer(timeout)
				nextRegisterTime = nextRegisterTimer.C
			}
		}
	}

	// Tracking registration lookups.
	var (
		topicRegisterLookupTarget lookupInfo
		topicRegisterLookupDone   chan []*Node
		topicRegisterLookupTick   = time.NewTimer(0)
	)
	<-topicRegisterLookupTick.C

	statsDump := time.NewTicker(10 * time.Second)

loop:
	for {
		resetNextTicket()

		select {
		case <-net.closeReq:
			debugLog("<-net.closeReq")
			break loop

		// Ingress packet handling.
		case pkt := <-net.read:
			debugLog("<-net.read")
			n := net.internNode(&pkt)
			prestate := n.state
			status := "ok"
			if err := net.handle(n, pkt.ev, &pkt); err != nil {
				status = err.Error()
			}
			if glog.V(logger.Detail) {
				glog.Infof("<<< (%d) %v from %x@%v: %v -> %v (%v)",
					net.tab.count, pkt.ev, pkt.remoteID[:8], pkt.remoteAddr, prestate, n.state, status)
			}
			// TODO: persist state if n.state goes >= known, delete if it goes <= known

		// State transition timeouts.
		case timeout := <-net.timeout:
			debugLog("<-net.timeout")
			if net.timeoutTimers[timeout] == nil {
				// Stale timer (was aborted).
				continue
			}
			delete(net.timeoutTimers, timeout)
			prestate := timeout.node.state
			status := "ok"
			if err := net.handle(timeout.node, timeout.ev, nil); err != nil {
				status = err.Error()
			}
			if glog.V(logger.Detail) {
				glog.Infof("--- (%d) %v for %x@%v: %v -> %v (%v)",
					net.tab.count, timeout.ev, timeout.node.ID[:8], timeout.node.addr(), prestate, timeout.node.state, status)
			}

		// Querying.
		case q := <-net.queryReq:
			debugLog("<-net.queryReq")
			if !q.start(net) {
				q.remote.deferQuery(q)
			}

		// Interacting with the table.
		case f := <-net.tableOpReq:
			debugLog("<-net.tableOpReq")
			f()
			net.tableOpResp <- struct{}{}

		// Topic registration stuff.
		case req := <-net.topicRegisterReq:
			debugLog("<-net.topicRegisterReq")
			if !req.add {
				net.ticketStore.removeRegisterTopic(req.topic)
				continue
			}
			net.ticketStore.addTopic(req.topic, true)
			// If we're currently waiting idle (nothing to look up), give the ticket store a
			// chance to start it sooner. This should speed up convergence of the radius
			// determination for new topics.
			// if topicRegisterLookupDone == nil {
			if topicRegisterLookupTarget.target == (common.Hash{}) {
				debugLog("topicRegisterLookupTarget == null")
				if topicRegisterLookupTick.Stop() {
					<-topicRegisterLookupTick.C
				}
				target, delay := net.ticketStore.nextRegisterLookup()
				topicRegisterLookupTarget = target
				topicRegisterLookupTick.Reset(delay)
			}

		case nodes := <-topicRegisterLookupDone:
			debugLog("<-topicRegisterLookupDone")
			net.ticketStore.registerLookupDone(topicRegisterLookupTarget, nodes, func(n *Node) []byte {
				net.ping(n, n.addr())
				return n.pingEcho
			})
			target, delay := net.ticketStore.nextRegisterLookup()
			topicRegisterLookupTarget = target
			topicRegisterLookupTick.Reset(delay)
			topicRegisterLookupDone = nil

		case <-topicRegisterLookupTick.C:
			debugLog("<-topicRegisterLookupTick")
			if (topicRegisterLookupTarget.target == common.Hash{}) {
				target, delay := net.ticketStore.nextRegisterLookup()
				topicRegisterLookupTarget = target
				topicRegisterLookupTick.Reset(delay)
				topicRegisterLookupDone = nil
			} else {
				topicRegisterLookupDone = make(chan []*Node)
				target := topicRegisterLookupTarget.target
				go func() { topicRegisterLookupDone <- net.lookup(target, false) }()
			}

		case <-nextRegisterTime:
			debugLog("<-nextRegisterTime")
			net.ticketStore.ticketRegistered(*nextTicket)
			net.conn.sendTopicRegister(nextTicket.t.node, nextTicket.t.topics, nextTicket.idx, nextTicket.t.pong)

		case <-statsDump.C:
			debugLog("<-statsDump.C")
			/*r, ok := net.ticketStore.radius[testTopic]
			if !ok {
				fmt.Printf("(%x) no radius @ %v\n", net.tab.self.ID[:8], time.Now())
			} else {
				topics := len(net.ticketStore.tickets)
				tickets := len(net.ticketStore.nodes)
				rad := r.radius / (maxRadius/10000+1)
				fmt.Printf("(%x) topics:%d radius:%d tickets:%d @ %v\n", net.tab.self.ID[:8], topics, rad, tickets, time.Now())
			}*/

			tm := monotonicTime()
			for topic, r := range net.ticketStore.radius {
				if r.converged {
					rad := r.radius / (maxRadius/1000000 + 1)
					if printTestImgLogs {
						fmt.Printf("*R %d %v %016x %v\n", tm/1000000, topic, net.tab.self.sha[:8], rad)
						fmt.Printf("*MR %d %v %016x %v\n", tm/1000000, topic, net.tab.self.sha[:8], net.ticketStore.minRadius/(maxRadius/1000000+1))
					}
				}
			}
			for topic, t := range net.topictab.topics {
				wp := t.wcl.nextWaitPeriod(tm)
				if printTestImgLogs {
					fmt.Printf("*W %d %v %016x %d\n", tm/1000000, topic, net.tab.self.sha[:8], wp/1000000)
				}
			}

		// Periodic / lookup-initiated bucket refresh.
		case <-refreshTimer.C:
			debugLog("<-refreshTimer.C")
			// TODO: ideally we would start the refresh timer after
			// fallback nodes have been set for the first time.
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				net.refresh(refreshDone)
			}
		case newNursery := <-net.refreshReq:
			debugLog("<-net.refreshReq")
			if newNursery != nil {
				net.nursery = newNursery
			}
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				net.refresh(refreshDone)
			}
			net.refreshResp <- refreshDone
		case <-refreshDone:
			debugLog("<-net.refreshDone")
			refreshDone = nil
		}
		debugLog("3")
	}
	debugLog("loop stopped")

	glog.V(logger.Debug).Infof("shutting down")
	if net.conn != nil {
		net.conn.Close()
	}
	if refreshDone != nil {
		// TODO: wait for pending refresh.
		//<-refreshResults
	}
	// Cancel all pending timeouts.
	for _, timer := range net.timeoutTimers {
		timer.Stop()
	}
	if net.db != nil {
		net.db.close()
	}
	close(net.closed)
}

// Everything below runs on the Network.loop goroutine
// and can modify Node, Table and Network at any time without locking.

func (net *Network) refresh(done chan<- struct{}) {
	var seeds []*Node
	if net.db != nil {
		seeds = net.db.querySeeds(seedCount, seedMaxAge)
	}
	if len(seeds) == 0 {
		seeds = net.nursery
	}
	if len(seeds) == 0 {
		glog.V(logger.Detail).Info("no seed nodes found")
		close(done)
		return
	}
	for _, n := range seeds {
		if glog.V(logger.Debug) {
			var age string
			if net.db != nil {
				age = time.Since(net.db.lastPong(n.ID)).String()
			} else {
				age = "unknown"
			}
			glog.Infof("seed node (age %s): %v", age, n)
		}
		n = net.internNodeFromDB(n)
		if n.state == unknown {
			net.transition(n, verifyinit)
		}
		// Force-add the seed node so Lookup does something.
		// It will be deleted again if verification fails.
		net.tab.add(n)
	}
	// Start self lookup to fill up the buckets.
	go func() {
		net.Lookup(net.tab.self.ID)
		close(done)
	}()
}

// Node Interning.

func (net *Network) internNode(pkt *ingressPacket) *Node {
	if n := net.nodes[pkt.remoteID]; n != nil {
		return n
	}
	n := NewNode(pkt.remoteID, pkt.remoteAddr.IP, uint16(pkt.remoteAddr.Port), uint16(pkt.remoteAddr.Port))
	n.state = unknown
	net.nodes[pkt.remoteID] = n
	return n
}

func (net *Network) internNodeFromDB(dbn *Node) *Node {
	if n := net.nodes[dbn.ID]; n != nil {
		return n
	}
	n := NewNode(dbn.ID, dbn.IP, dbn.UDP, dbn.TCP)
	n.state = unknown
	net.nodes[n.ID] = n
	return n
}

func (net *Network) internNodeFromNeighbours(rn rpcNode) (n *Node, err error) {
	if rn.ID == net.tab.self.ID {
		return nil, errors.New("is self")
	}
	n = net.nodes[rn.ID]
	if n == nil {
		// We haven't seen this node before.
		n, err = nodeFromRPC(rn)
		n.state = unknown
		if err == nil {
			net.nodes[n.ID] = n
		}
		return n, err
	}
	if !bytes.Equal(n.IP, rn.IP) || n.UDP != rn.UDP || n.TCP != rn.TCP {
		err = fmt.Errorf("metadata mismatch: got %v, want %v", rn, n)
	}
	return n, err
}

// nodeNetGuts is embedded in Node and contains fields.
type nodeNetGuts struct {
	// This is a cached copy of sha3(ID) which is used for node
	// distance calculations. This is part of Node in order to make it
	// possible to write tests that need a node at a certain distance.
	// In those tests, the content of sha will not actually correspond
	// with ID.
	sha common.Hash

	// State machine fields. Access to these fields
	// is restricted to the Network.loop goroutine.
	state             *nodeState
	pingEcho          []byte           // hash of last ping sent by us
	pingTopics        []Topic          // topic set sent by us in last ping
	deferredQueries   []*findnodeQuery // queries that can't be sent yet
	pendingNeighbours *findnodeQuery   // current query, waiting for reply
	queryTimeouts     int
}

func (n *nodeNetGuts) deferQuery(q *findnodeQuery) {
	n.deferredQueries = append(n.deferredQueries, q)
}

func (n *nodeNetGuts) startNextQuery(net *Network) {
	if len(n.deferredQueries) == 0 {
		return
	}
	nextq := n.deferredQueries[0]
	if nextq.start(net) {
		n.deferredQueries = append(n.deferredQueries[:0], n.deferredQueries[1:]...)
	}
}

func (q *findnodeQuery) start(net *Network) bool {
	// Satisfy queries against the local node directly.
	if q.remote == net.tab.self {
		closest := net.tab.closest(crypto.Keccak256Hash(q.target[:]), bucketSize)
		q.reply <- closest.entries
		return true
	}
	if q.remote.state.canQuery && q.remote.pendingNeighbours == nil {
		net.conn.sendFindnodeHash(q.remote, q.target)
		net.timedEvent(respTimeout, q.remote, neighboursTimeout)
		q.remote.pendingNeighbours = q
		return true
	}
	// If the node is not known yet, it won't accept queries.
	// Initiate the transition to known.
	// The request will be sent later when the node reaches known state.
	if q.remote.state == unknown {
		net.transition(q.remote, verifyinit)
	}
	return false
}

// Node Events (the input to the state machine).

type nodeEvent uint

//go:generate stringer -type=nodeEvent

const (
	invalidEvent nodeEvent = iota // zero is reserved

	// Packet type events.
	// These correspond to packet types in the UDP protocol.
	pingPacket
	pongPacket
	findnodePacket
	neighborsPacket
	findnodeHashPacket
	topicRegisterPacket
	topicQueryPacket
	topicNodesPacket

	// Non-packet events.
	// Event values in this category are allocated outside
	// the packet type range (packet types are encoded as a single byte).
	pongTimeout nodeEvent = iota + 256
	pingTimeout
	neighboursTimeout
)

// Node State Machine.

type nodeState struct {
	name     string
	handle   func(*Network, *Node, nodeEvent, *ingressPacket) (next *nodeState, err error)
	enter    func(*Network, *Node)
	canQuery bool
}

func (s *nodeState) String() string {
	return s.name
}

var (
	unknown          *nodeState
	verifyinit       *nodeState
	verifywait       *nodeState
	remoteverifywait *nodeState
	known            *nodeState
	contested        *nodeState
	unresponsive     *nodeState
)

func init() {
	unknown = &nodeState{
		name: "unknown",
		enter: func(net *Network, n *Node) {
			net.tab.delete(n)
			n.pingEcho = nil
			// Abort active queries.
			for _, q := range n.deferredQueries {
				q.reply <- nil
			}
			n.deferredQueries = nil
			if n.pendingNeighbours != nil {
				n.pendingNeighbours.reply <- nil
				n.pendingNeighbours = nil
			}
			n.queryTimeouts = 0
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				net.ping(n, pkt.remoteAddr)
				return verifywait, nil
			default:
				return unknown, errInvalidEvent
			}
		},
	}

	verifyinit = &nodeState{
		name: "verifyinit",
		enter: func(net *Network, n *Node) {
			net.ping(n, n.addr())
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return verifywait, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return remoteverifywait, err
			case pongTimeout:
				return unknown, nil
			default:
				return verifyinit, errInvalidEvent
			}
		},
	}

	verifywait = &nodeState{
		name: "verifywait",
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			case pongTimeout:
				return unknown, nil
			default:
				return verifywait, errInvalidEvent
			}
		},
	}

	remoteverifywait = &nodeState{
		name: "remoteverifywait",
		enter: func(net *Network, n *Node) {
			net.timedEvent(respTimeout, n, pingTimeout)
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return remoteverifywait, nil
			case pingTimeout:
				return known, nil
			default:
				return remoteverifywait, errInvalidEvent
			}
		},
	}

	known = &nodeState{
		name:     "known",
		canQuery: true,
		enter: func(net *Network, n *Node) {
			n.queryTimeouts = 0
			n.startNextQuery(net)
			// Insert into the table and start revalidation of the last node
			// in the bucket if it is full.
			last := net.tab.add(n)
			if last != nil && last.state == known {
				// TODO: do this asynchronously
				net.transition(last, contested)
			}
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return known, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}

	contested = &nodeState{
		name:     "contested",
		canQuery: true,
		enter: func(net *Network, n *Node) {
			net.ping(n, n.addr())
		},
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pongPacket:
				// Node is still alive.
				err := net.handleKnownPong(n, pkt)
				return known, err
			case pongTimeout:
				net.tab.deleteReplace(n)
				return unresponsive, nil
			case pingPacket:
				net.handlePing(n, pkt)
				return contested, nil
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}

	unresponsive = &nodeState{
		name:     "unresponsive",
		canQuery: true,
		handle: func(net *Network, n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
			switch ev {
			case pingPacket:
				net.handlePing(n, pkt)
				return known, nil
			case pongPacket:
				err := net.handleKnownPong(n, pkt)
				return known, err
			default:
				return net.handleQueryEvent(n, ev, pkt)
			}
		},
	}
}

// handle processes packets sent by n and events related to n.
func (net *Network) handle(n *Node, ev nodeEvent, pkt *ingressPacket) error {
	if pkt != nil {
		if err := net.checkPacket(n, ev, pkt); err != nil {
			return err
		}
		// Start the background expiration goroutine after the first
		// successful communication. Subsequent calls have no effect if it
		// is already running. We do this here instead of somewhere else
		// so that the search for seed nodes also considers older nodes
		// that would otherwise be removed by the expirer.
		if net.db != nil {
			net.db.ensureExpirer()
		}
	}
	if n.state == nil {
		n.state = unknown //???
	}
	next, err := n.state.handle(net, n, ev, pkt)
	net.transition(n, next)
	return err
}

func (net *Network) checkPacket(n *Node, ev nodeEvent, pkt *ingressPacket) error {
	// Replay prevention checks.
	switch ev {
	case pingPacket, findnodePacket, neighborsPacket:
		// TODO: check date is > last date seen
		// TODO: check ping version
	case pongPacket:
		if !bytes.Equal(pkt.data.(*pong).ReplyTok, n.pingEcho) {
			// fmt.Println("pong reply token mismatch")
			return fmt.Errorf("pong reply token mismatch")
		}
		n.pingEcho = nil
	}
	// Address validation.
	// TODO: Ideally we would do the following:
	//  - reject all packets with wrong address except ping.
	//  - for ping with new address, transition to verifywait but keep the
	//    previous node (with old address) around. if the new one reaches known,
	//    swap it out.
	return nil
}

func (net *Network) transition(n *Node, next *nodeState) {
	if n.state != next {
		n.state = next
		if next.enter != nil {
			next.enter(net, n)
		}
	}

	// TODO: persist/unpersist node
}

func (net *Network) timedEvent(d time.Duration, n *Node, ev nodeEvent) {
	timeout := timeoutEvent{ev, n}
	net.timeoutTimers[timeout] = time.AfterFunc(d, func() {
		select {
		case net.timeout <- timeout:
		case <-net.closed:
		}
	})
}

func (net *Network) abortTimedEvent(n *Node, ev nodeEvent) {
	timer := net.timeoutTimers[timeoutEvent{ev, n}]
	if timer != nil {
		timer.Stop()
		delete(net.timeoutTimers, timeoutEvent{ev, n})
	}
}

func (net *Network) ping(n *Node, addr *net.UDPAddr) {
	debugLog(fmt.Sprintf("ping(node = %x)", n.ID[:8]))
	n.pingTopics = net.ticketStore.regTopicSet()
	n.pingEcho = net.conn.sendPing(n, addr, n.pingTopics)
	net.timedEvent(respTimeout, n, pongTimeout)
}

func (net *Network) handlePing(n *Node, pkt *ingressPacket) {
	ping := pkt.data.(*ping)
	n.TCP = ping.From.TCP
	t := net.topictab.getTicket(n, ping.Topics)

	pong := &pong{
		To:         makeEndpoint(n.addr(), 0), // TODO: maybe use known TCP port from DB
		ReplyTok:   pkt.hash,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	ticketToPong(t, pong)
	net.conn.send(n, pongPacket, pong)
}

func (net *Network) handleKnownPong(n *Node, pkt *ingressPacket) error {
	debugLog(fmt.Sprintf("handleKnownPong(node = %x)", n.ID[:8]))
	net.abortTimedEvent(n, pongTimeout)
	now := monotonicTime()
	ticket, err := pongToTicket(now, n.pingTopics, n, pkt)
	if err == nil {
		// fmt.Printf("(%x) ticket: %+v\n", net.tab.self.ID[:8], pkt.data)
		net.ticketStore.addTicket(now, pkt.data.(*pong).ReplyTok, ticket)
	} else {
		debugLog(fmt.Sprintf(" error: %v", err))
	}

	n.pingEcho = nil
	n.pingTopics = nil
	return err
}

func (net *Network) handleQueryEvent(n *Node, ev nodeEvent, pkt *ingressPacket) (*nodeState, error) {
	switch ev {
	case findnodePacket:
		target := crypto.Keccak256Hash(pkt.data.(*findnode).Target[:])
		results := net.tab.closest(target, bucketSize).entries
		net.conn.sendNeighbours(n, results)
		return n.state, nil
	case neighborsPacket:
		err := net.handleNeighboursPacket(n, pkt.data.(*neighbors))
		return n.state, err
	case neighboursTimeout:
		if n.pendingNeighbours != nil {
			n.pendingNeighbours.reply <- nil
			n.pendingNeighbours = nil
		}
		n.queryTimeouts++
		if n.queryTimeouts > maxFindnodeFailures && n.state == known {
			return contested, errors.New("too many timeouts")
		}
		return n.state, nil

	// v5

	case findnodeHashPacket:
		results := net.tab.closest(pkt.data.(*findnodeHash).Target, bucketSize).entries
		net.conn.sendNeighbours(n, results)
		return n.state, nil
	case topicRegisterPacket:
		//fmt.Println("got topicRegisterPacket")
		regdata := pkt.data.(*topicRegister)
		pong, err := net.checkTopicRegister(regdata)
		if err != nil {
			//fmt.Println(err)
			return n.state, fmt.Errorf("bad waiting ticket: %v", err)
		}
		net.topictab.useTicket(n, pong.TicketSerial, regdata.Topics, regdata.Idx, pong.Expiration, pong.WaitPeriods)
		return n.state, nil
	case topicQueryPacket:
		results := net.topictab.getEntries(pkt.data.(*topicQuery).Topic)
		if len(results) > 10 {
			results = results[:10]
		}
		var hash common.Hash
		copy(hash[:], pkt.hash)
		net.conn.sendTopicNodes(n, hash, results)
		return n.state, nil
	case topicNodesPacket:
		// if n.pendingTopicNodes != nil {
		// 	n.pendingNeighbours.reply <- nil
		// 	n.pendingNeighbours = nil
		// }
		// n.queryTimeouts++
		// if n.queryTimeouts > maxFindnodeFailures && n.state == known {
		// 	return contested, errors.New("too many timeouts")
		// }
		//
		// if n.pendingTopicNodes != nil {
		//
		// }
		return n.state, nil

	default:
		return n.state, errInvalidEvent
	}
}

func (net *Network) checkTopicRegister(data *topicRegister) (*pong, error) {
	var pongpkt ingressPacket
	//fmt.Println("got", data.Topics, data.Pong)
	if err := decodePacket(data.Pong, &pongpkt); err != nil {
		return nil, err
	}
	if pongpkt.ev != pongPacket {
		return nil, errors.New("is not pong packet")
	}
	if pongpkt.remoteID != net.tab.self.ID {
		return nil, errors.New("not signed by us")
	}
	// check that we previously authorised all topics
	// that the other side is trying to register.
	if rlpHash(data.Topics) != pongpkt.data.(*pong).TopicHash {
		return nil, errors.New("topic hash mismatch")
	}
	if data.Idx < 0 || data.Idx >= len(data.Topics) {
		return nil, errors.New("topic index out of range")
	}
	return pongpkt.data.(*pong), nil
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func (net *Network) handleNeighboursPacket(n *Node, req *neighbors) error {
	if n.pendingNeighbours == nil {
		return errNoQuery
	}
	net.abortTimedEvent(n, neighboursTimeout)

	nodes := make([]*Node, len(req.Nodes))
	for i, rn := range req.Nodes {
		nn, err := net.internNodeFromNeighbours(rn)
		if err != nil {
			glog.V(logger.Debug).Infof("invalid neighbour from %x: %v", n.ID[:8], err)
			continue
		}
		nodes[i] = nn
		// Start validation of query results immediately.
		// This fills the table quickly.
		// TODO: generates way too many packets, maybe do it via queue.
		if nn.state == unknown {
			net.transition(nn, verifyinit)
		}
	}
	// TODO: don't ignore second packet
	n.pendingNeighbours.reply <- nodes
	n.pendingNeighbours = nil
	// Now that this query is done, start the next one.
	n.startNextQuery(net)
	return nil
}
