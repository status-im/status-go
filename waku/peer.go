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

package waku

import (
	"bytes"
	"fmt"
	"math"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

// Peer represents a waku protocol peer connection.
type Peer struct {
	host   *Waku
	peer   *p2p.Peer
	ws     p2p.MsgReadWriter
	logger *zap.Logger

	trusted        bool
	powRequirement float64
	// bloomMu is to allow thread safe access to
	// the bloom filter
	bloomMu     sync.Mutex
	bloomFilter []byte
	// topicInterestMu is to allow thread safe access to
	// the map of topic interests
	topicInterestMu sync.Mutex
	topicInterest   map[TopicType]bool
	// fullNode is used to indicate that the node will be accepting any
	// envelope. The opposite is an "empty node" , which is when
	// a bloom filter is all 0s or topic interest is an empty map (not nil).
	// In that case no envelope is accepted.
	fullNode             bool
	confirmationsEnabled bool
	rateLimitsMu         sync.Mutex
	rateLimits           RateLimits

	known mapset.Set // Messages already known by the peer to avoid wasting bandwidth

	quit chan struct{}
}

// newPeer creates a new waku peer object, but does not run the handshake itself.
func newPeer(host *Waku, remote *p2p.Peer, rw p2p.MsgReadWriter, logger *zap.Logger) *Peer {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Peer{
		host:           host,
		peer:           remote,
		ws:             rw,
		logger:         logger,
		trusted:        false,
		powRequirement: 0.0,
		known:          mapset.NewSet(),
		quit:           make(chan struct{}),
		bloomFilter:    MakeFullNodeBloom(),
		fullNode:       true,
	}
}

// start initiates the peer updater, periodically broadcasting the waku packets
// into the network.
func (p *Peer) start() {
	go p.update()
	p.logger.Debug("starting peer", zap.Binary("peerID", p.ID()))
}

// stop terminates the peer updater, stopping message forwarding to it.
func (p *Peer) stop() {
	close(p.quit)
	p.logger.Debug("stopping peer", zap.Binary("peerID", p.ID()))
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (p *Peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	opts := p.host.toStatusOptions()
	go func() {
		errc <- p2p.SendItems(p.ws, statusCode, ProtocolVersion, opts)
	}()

	// Fetch the remote status packet and verify protocol match
	packet, err := p.ws.ReadMsg()
	if err != nil {
		return err
	}
	if packet.Code != statusCode {
		return fmt.Errorf("p [%x] sent packet %x before status packet", p.ID(), packet.Code)
	}

	var (
		peerProtocolVersion uint64
		peerOptions         statusOptions
	)
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	if _, err := s.List(); err != nil {
		return fmt.Errorf("p [%x]: failed to decode status packet: %v", p.ID(), err)
	}
	// Validate protocol version.
	if err := s.Decode(&peerProtocolVersion); err != nil {
		return fmt.Errorf("p [%x]: failed to decode peer protocol version: %v", p.ID(), err)
	}
	if peerProtocolVersion != ProtocolVersion {
		return fmt.Errorf("p [%x]: protocol version mismatch %d != %d", p.ID(), peerProtocolVersion, ProtocolVersion)
	}
	// Decode and validate other status packet options.
	if err := s.Decode(&peerOptions); err != nil {
		return fmt.Errorf("p [%x]: failed to decode status options: %v", p.ID(), err)
	}
	if err := s.ListEnd(); err != nil {
		return fmt.Errorf("p [%x]: failed to decode status packet: %v", p.ID(), err)
	}
	if err := p.setOptions(peerOptions.WithDefaults()); err != nil {
		return fmt.Errorf("p [%x]: failed to set options: %v", p.ID(), err)
	}
	if err := <-errc; err != nil {
		return fmt.Errorf("p [%x] failed to send status packet: %v", p.ID(), err)
	}
	return nil
}

func (p *Peer) setOptions(peerOptions statusOptions) error {

	p.logger.Debug("settings options", zap.Binary("peerID", p.ID()), zap.Any("Options", peerOptions))

	if err := peerOptions.Validate(); err != nil {
		return fmt.Errorf("p [%x]: sent invalid options: %v", p.ID(), err)
	}
	// Validate and save peer's PoW.
	pow := peerOptions.PoWRequirementF()
	if pow != nil {
		if math.IsInf(*pow, 0) || math.IsNaN(*pow) || *pow < 0.0 {
			return fmt.Errorf("p [%x]: sent bad status message: invalid pow", p.ID())
		}
		p.powRequirement = *pow
	}

	if peerOptions.TopicInterest != nil {
		p.setTopicInterest(peerOptions.TopicInterest)
	} else if peerOptions.BloomFilter != nil {
		// Validate and save peer's bloom filters.
		bloom := peerOptions.BloomFilter
		bloomSize := len(bloom)
		if bloomSize != 0 && bloomSize != BloomFilterSize {
			return fmt.Errorf("p [%x] sent bad status message: wrong bloom filter size %d", p.ID(), bloomSize)
		}
		p.setBloomFilter(bloom)
	}

	if peerOptions.LightNodeEnabled != nil {
		// Validate and save other peer's options.
		if *peerOptions.LightNodeEnabled && p.host.LightClientMode() && p.host.LightClientModeConnectionRestricted() {
			return fmt.Errorf("p [%x] is useless: two light client communication restricted", p.ID())
		}
	}
	if peerOptions.ConfirmationsEnabled != nil {
		p.confirmationsEnabled = *peerOptions.ConfirmationsEnabled
	}
	if peerOptions.RateLimits != nil {
		p.setRateLimits(*peerOptions.RateLimits)
	}

	return nil
}

// update executes periodic operations on the peer, including message transmission
// and expiration.
func (p *Peer) update() {
	// Start the tickers for the updates
	expire := time.NewTicker(expirationCycle)
	transmit := time.NewTicker(transmissionCycle)

	// Loop and transmit until termination is requested
	for {
		select {
		case <-expire.C:
			p.expire()

		case <-transmit.C:
			if err := p.broadcast(); err != nil {
				p.logger.Debug("broadcasting failed", zap.Binary("peer", p.ID()), zap.Error(err))
				return
			}

		case <-p.quit:
			return
		}
	}
}

// mark marks an envelope known to the peer so that it won't be sent back.
func (p *Peer) mark(envelope *Envelope) {
	p.known.Add(envelope.Hash())
}

// marked checks if an envelope is already known to the remote peer.
func (p *Peer) marked(envelope *Envelope) bool {
	return p.known.Contains(envelope.Hash())
}

// expire iterates over all the known envelopes in the host and removes all
// expired (unknown) ones from the known list.
func (p *Peer) expire() {
	unmark := make(map[common.Hash]struct{})
	p.known.Each(func(v interface{}) bool {
		if !p.host.isEnvelopeCached(v.(common.Hash)) {
			unmark[v.(common.Hash)] = struct{}{}
		}
		return true
	})
	// Dump all known but no longer cached
	for hash := range unmark {
		p.known.Remove(hash)
	}
}

// broadcast iterates over the collection of envelopes and transmits yet unknown
// ones over the network.
func (p *Peer) broadcast() error {
	envelopes := p.host.Envelopes()
	bundle := make([]*Envelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		if !p.marked(envelope) && envelope.PoW() >= p.powRequirement && p.topicOrBloomMatch(envelope) {
			bundle = append(bundle, envelope)
		}
	}

	if len(bundle) == 0 {
		return nil
	}

	batchHash, err := sendBundle(p.ws, bundle)
	if err != nil {
		p.logger.Debug("failed to deliver envelopes", zap.Binary("peer", p.ID()), zap.Error(err))
		return err
	}

	// mark envelopes only if they were successfully sent
	for _, e := range bundle {
		p.mark(e)
		event := EnvelopeEvent{
			Event: EventEnvelopeSent,
			Hash:  e.Hash(),
			Peer:  p.peer.ID(),
		}
		if p.confirmationsEnabled {
			event.Batch = batchHash
		}
		p.host.envelopeFeed.Send(event)
	}
	p.logger.Debug("broadcasted bundles successfully", zap.Binary("peer", p.ID()), zap.Int("count", len(bundle)))
	return nil
}

// ID returns a peer's id
func (p *Peer) ID() []byte {
	id := p.peer.ID()
	return id[:]
}

func (p *Peer) notifyAboutPowRequirementChange(pow float64) error {
	i := math.Float64bits(pow)
	return p2p.Send(p.ws, statusUpdateCode, statusOptions{PoWRequirement: &i})
}

func (p *Peer) notifyAboutBloomFilterChange(bloom []byte) error {
	return p2p.Send(p.ws, statusUpdateCode, statusOptions{BloomFilter: bloom})
}

func (p *Peer) notifyAboutTopicInterestChange(topics []TopicType) error {
	return p2p.Send(p.ws, statusUpdateCode, statusOptions{TopicInterest: topics})
}

func (p *Peer) bloomMatch(env *Envelope) bool {
	p.bloomMu.Lock()
	defer p.bloomMu.Unlock()
	return p.fullNode || BloomFilterMatch(p.bloomFilter, env.Bloom())
}

func (p *Peer) topicInterestMatch(env *Envelope) bool {
	p.topicInterestMu.Lock()
	defer p.topicInterestMu.Unlock()

	if p.topicInterest == nil {
		return false
	}

	return p.fullNode || p.topicInterest[env.Topic]
}

// topicOrBloomMatch matches against topic-interest if topic interest
// is not nil. Otherwise it will match against the bloom-filter.
// If the bloom-filter is nil, or full, the node is considered a full-node
// and any envelope will be accepted. An empty topic-interest (but not nil)
// signals that we are not interested in any envelope.
func (p *Peer) topicOrBloomMatch(env *Envelope) bool {
	p.topicInterestMu.Lock()
	topicInterestMode := p.topicInterest != nil
	p.topicInterestMu.Unlock()

	if topicInterestMode {
		return p.topicInterestMatch(env)
	}
	return p.bloomMatch(env)
}

func (p *Peer) setBloomFilter(bloom []byte) {
	p.bloomMu.Lock()
	defer p.bloomMu.Unlock()
	p.bloomFilter = bloom
	p.fullNode = isFullNode(bloom)
	if p.fullNode && p.bloomFilter == nil {
		p.bloomFilter = MakeFullNodeBloom()
	}
	p.topicInterest = nil
}

func (p *Peer) setTopicInterest(topicInterest []TopicType) {
	p.topicInterestMu.Lock()
	defer p.topicInterestMu.Unlock()
	if topicInterest == nil {
		p.topicInterest = nil
		return
	}
	p.topicInterest = make(map[TopicType]bool)
	for _, topic := range topicInterest {
		p.topicInterest[topic] = true
	}
	p.bloomFilter = nil
}

func (p *Peer) setRateLimits(r RateLimits) {
	p.rateLimitsMu.Lock()
	p.rateLimits = r
	p.rateLimitsMu.Unlock()
}

func MakeFullNodeBloom() []byte {
	bloom := make([]byte, BloomFilterSize)
	for i := 0; i < BloomFilterSize; i++ {
		bloom[i] = 0xFF
	}
	return bloom
}

func sendBundle(rw p2p.MsgWriter, bundle []*Envelope) (rst common.Hash, err error) {
	data, err := rlp.EncodeToBytes(bundle)
	if err != nil {
		return
	}
	err = rw.WriteMsg(p2p.Msg{
		Code:    messagesCode,
		Size:    uint32(len(data)),
		Payload: bytes.NewBuffer(data),
	})
	if err != nil {
		return
	}
	return crypto.Keccak256Hash(data), nil
}
