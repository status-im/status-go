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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"go.uber.org/zap"
)

// Peer represents a waku protocol peer connection.
type Peer struct {
	host   *Waku
	peer   *p2p.Peer
	ws     p2p.MsgReadWriter
	logger *zap.Logger

	trusted              bool
	powRequirement       float64
	bloomMu              sync.Mutex
	bloomFilter          []byte
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
	isLightNode := p.host.LightClientMode()
	isRestrictedLightNodeConnection := p.host.LightClientModeConnectionRestricted()
	go func() {
		pow := p.host.MinPow()
		powConverted := math.Float64bits(pow)
		bloom := p.host.BloomFilter()
		confirmationsEnabled := p.host.ConfirmationsEnabled()
		rateLimits := p.host.RateLimits()

		errc <- p2p.SendItems(p.ws, statusCode, ProtocolVersion, powConverted, bloom, isLightNode, confirmationsEnabled, rateLimits)
	}()

	// Fetch the remote status packet and verify protocol match
	packet, err := p.ws.ReadMsg()
	if err != nil {
		return err
	}
	if packet.Code != statusCode {
		return fmt.Errorf("p [%x] sent packet %x before status packet", p.ID(), packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	_, err = s.List()
	if err != nil {
		return fmt.Errorf("p [%x] sent bad status message: %v", p.ID(), err)
	}
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("p [%x] sent bad status message (unable to decode version): %v", p.ID(), err)
	}
	if peerVersion != ProtocolVersion {
		return fmt.Errorf("p [%x]: protocol version mismatch %d != %d", p.ID(), peerVersion, ProtocolVersion)
	}

	// only version is mandatory, subsequent parameters are optional
	powRaw, err := s.Uint()
	if err == nil {
		pow := math.Float64frombits(powRaw)
		if math.IsInf(pow, 0) || math.IsNaN(pow) || pow < 0.0 {
			return fmt.Errorf("p [%x] sent bad status message: invalid pow", p.ID())
		}
		p.powRequirement = pow

		var bloom []byte
		err = s.Decode(&bloom)
		if err == nil {
			sz := len(bloom)
			if sz != BloomFilterSize && sz != 0 {
				return fmt.Errorf("p [%x] sent bad status message: wrong bloom filter size %d", p.ID(), sz)
			}
			p.setBloomFilter(bloom)
		}
	}

	isRemotePeerLightNode, _ := s.Bool()
	if isRemotePeerLightNode && isLightNode && isRestrictedLightNodeConnection {
		return fmt.Errorf("p [%x] is useless: two light client communication restricted", p.ID())
	}
	confirmationsEnabled, err := s.Bool()
	if err != nil || !confirmationsEnabled {
		p.logger.Info("confirmations are disabled for peer", zap.Binary("peer", p.ID()))
	} else {
		p.confirmationsEnabled = confirmationsEnabled
	}

	var rateLimits RateLimits
	if err := s.Decode(&rateLimits); err != nil {
		p.logger.Info("rate limiting is disabled for peer", zap.Binary("peer", p.ID()))
	} else {
		p.setRateLimits(rateLimits)
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("p [%x] failed to send status packet: %v", p.ID(), err)
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
		if !p.marked(envelope) && envelope.PoW() >= p.powRequirement && p.bloomMatch(envelope) {
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
	return p2p.Send(p.ws, powRequirementCode, i)
}

func (p *Peer) notifyAboutBloomFilterChange(bloom []byte) error {
	return p2p.Send(p.ws, bloomFilterExCode, bloom)
}

func (p *Peer) bloomMatch(env *Envelope) bool {
	p.bloomMu.Lock()
	defer p.bloomMu.Unlock()
	return p.fullNode || BloomFilterMatch(p.bloomFilter, env.Bloom())
}

func (p *Peer) setBloomFilter(bloom []byte) {
	p.bloomMu.Lock()
	defer p.bloomMu.Unlock()
	p.bloomFilter = bloom
	p.fullNode = isFullNode(bloom)
	if p.fullNode && p.bloomFilter == nil {
		p.bloomFilter = MakeFullNodeBloom()
	}
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
