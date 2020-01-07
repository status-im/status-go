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
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"go.uber.org/zap"

	mapset "github.com/deckarep/golang-set"
	"golang.org/x/crypto/pbkdf2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// TimeSyncError error for clock skew errors.
type TimeSyncError error

type settings struct {
	MaxMsgSize               uint32  // Maximal message length allowed by the waku node
	EnableConfirmations      bool    // Enable sending message confirmations
	MinPow                   float64 // Minimal PoW required by the waku node
	MinPowTolerance          float64 // Minimal PoW tolerated by the waku node for a limited time
	BloomFilter              []byte  // Bloom filter for topics of interest for this node
	BloomFilterTolerance     []byte  // Bloom filter tolerated by the waku node for a limited time
	LightClient              bool    // Light client mode enabled does not forward messages
	RestrictLightClientsConn bool    // Restrict connection between two light clients
	SyncAllowance            int     // Maximum time in seconds allowed to process the waku-related messages
}

// Waku represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Waku struct {
	protocol p2p.Protocol // Protocol description and parameters
	filters  *Filters     // Message filters installed with Subscribe function

	privateKeys map[string]*ecdsa.PrivateKey // Private key storage
	symKeys     map[string][]byte            // Symmetric key storage
	keyMu       sync.RWMutex                 // Mutex associated with key stores

	envelopes   map[common.Hash]*Envelope // Pool of envelopes currently tracked by this node
	expirations map[uint32]mapset.Set     // Message expiration pool
	poolMu      sync.RWMutex              // Mutex to sync the message and expiration pools

	peers  map[*Peer]struct{} // Set of currently active peers
	peerMu sync.RWMutex       // Mutex to sync the active peer set

	msgQueue    chan *Envelope   // Message queue for normal waku messages
	p2pMsgQueue chan interface{} // Message queue for peer-to-peer messages (not to be forwarded any further) and history delivery confirmations.
	quit        chan struct{}    // Channel used for graceful exit

	settings   settings     // Holds configuration settings that can be dynamically changed
	settingsMu sync.RWMutex // Mutex to sync the settings access

	mailServer MailServer

	rateLimiter *PeerRateLimiter

	envelopeFeed event.Feed

	timeSource func() time.Time // source of time for waku

	logger *zap.Logger
}

// New creates a Waku client ready to communicate through the Ethereum P2P network.
func New(cfg *Config, logger *zap.Logger) *Waku {
	if cfg == nil {
		c := DefaultConfig
		cfg = &c
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	waku := &Waku{
		privateKeys: make(map[string]*ecdsa.PrivateKey),
		symKeys:     make(map[string][]byte),
		envelopes:   make(map[common.Hash]*Envelope),
		expirations: make(map[uint32]mapset.Set),
		peers:       make(map[*Peer]struct{}),
		msgQueue:    make(chan *Envelope, messageQueueLimit),
		p2pMsgQueue: make(chan interface{}, messageQueueLimit),
		quit:        make(chan struct{}),
		timeSource:  time.Now,
		logger:      logger,
	}

	waku.settings = settings{
		MaxMsgSize:               cfg.MaxMessageSize,
		MinPow:                   cfg.MinimumAcceptedPoW,
		MinPowTolerance:          cfg.MinimumAcceptedPoW,
		EnableConfirmations:      cfg.EnableConfirmations,
		LightClient:              cfg.LightClient,
		RestrictLightClientsConn: cfg.RestrictLightClientsConn,
		SyncAllowance:            DefaultSyncAllowance,
	}

	if cfg.FullNode {
		waku.settings.BloomFilter = MakeFullNodeBloom()
		waku.settings.BloomFilterTolerance = MakeFullNodeBloom()
	}

	waku.filters = NewFilters(waku)

	// p2p waku sub-protocol handler
	waku.protocol = p2p.Protocol{
		Name:    ProtocolName,
		Version: uint(ProtocolVersion),
		Length:  NumberOfMessageCodes,
		Run:     waku.HandlePeer,
		NodeInfo: func() interface{} {
			return map[string]interface{}{
				"version":        ProtocolVersionStr,
				"maxMessageSize": waku.MaxMessageSize(),
				"minimumPoW":     waku.MinPow(),
			}
		},
	}

	return waku
}

// Version returns the waku sub-protocol version number.
func (w *Waku) Version() uint {
	return w.protocol.Version
}

// MinPow returns the PoW value required by this node.
func (w *Waku) MinPow() float64 {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.MinPow
}

// SetMinimumPoW sets the minimal PoW required by this node
func (w *Waku) SetMinimumPoW(val float64, tolerate bool) error {
	if val < 0.0 {
		return fmt.Errorf("invalid PoW: %f", val)
	}

	w.settingsMu.Lock()
	w.settings.MinPow = val
	w.settingsMu.Unlock()

	w.notifyPeersAboutPowRequirementChange(val)

	if tolerate {
		go func() {
			// allow some time before all the peers have processed the notification
			time.Sleep(time.Duration(w.settings.SyncAllowance) * time.Second)
			w.settingsMu.Lock()
			w.settings.MinPowTolerance = val
			w.settingsMu.Unlock()
		}()
	}

	return nil
}

// MinPowTolerance returns the value of minimum PoW which is tolerated for a limited
// time after PoW was changed. If sufficient time have elapsed or no change of PoW
// have ever occurred, the return value will be the same as return value of MinPow().
func (w *Waku) MinPowTolerance() float64 {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.MinPowTolerance
}

// BloomFilter returns the aggregated bloom filter for all the topics of interest.
// The nodes are required to send only messages that match the advertised bloom filter.
// If a message does not match the bloom, it will tantamount to spam, and the peer will
// be disconnected.
func (w *Waku) BloomFilter() []byte {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.BloomFilter
}

// BloomFilterTolerance returns the bloom filter which is tolerated for a limited
// time after new bloom was advertised to the peers. If sufficient time have elapsed
// or no change of bloom filter have ever occurred, the return value will be the same
// as return value of BloomFilter().
func (w *Waku) BloomFilterTolerance() []byte {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.BloomFilterTolerance
}

// SetBloomFilter sets the new bloom filter
func (w *Waku) SetBloomFilter(bloom []byte) error {
	if len(bloom) != BloomFilterSize {
		return fmt.Errorf("invalid bloom filter size: %d", len(bloom))
	}

	b := make([]byte, BloomFilterSize)
	copy(b, bloom)

	w.settingsMu.Lock()
	w.settings.BloomFilter = b
	w.settingsMu.Unlock()
	w.notifyPeersAboutBloomFilterChange(b)

	go func() {
		// allow some time before all the peers have processed the notification
		time.Sleep(time.Duration(w.settings.SyncAllowance) * time.Second)
		w.settingsMu.Lock()
		w.settings.BloomFilterTolerance = b
		w.settingsMu.Unlock()
	}()

	return nil
}

// MaxMessageSize returns the maximum accepted message size.
func (w *Waku) MaxMessageSize() uint32 {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.MaxMsgSize
}

// SetMaxMessageSize sets the maximal message size allowed by this node
func (w *Waku) SetMaxMessageSize(size uint32) error {
	if size > MaxMessageSize {
		return fmt.Errorf("message size too large [%d>%d]", size, MaxMessageSize)
	}
	w.settingsMu.Lock()
	w.settings.MaxMsgSize = size
	w.settingsMu.Unlock()
	return nil
}

// LightClientMode indicates is this node is light client (does not forward any messages)
func (w *Waku) LightClientMode() bool {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.LightClient
}

// SetLightClientMode makes node light client (does not forward any messages)
func (w *Waku) SetLightClientMode(v bool) {
	w.settingsMu.Lock()
	w.settings.LightClient = v
	w.settingsMu.Unlock()
}

// LightClientModeConnectionRestricted indicates that connection to light client in light client mode not allowed
func (w *Waku) LightClientModeConnectionRestricted() bool {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.RestrictLightClientsConn
}

// RateLimiting returns RateLimits information.
func (w *Waku) RateLimits() RateLimits {
	if w.rateLimiter == nil {
		return RateLimits{}
	}
	return RateLimits{
		IPLimits:     uint64(w.rateLimiter.limitPerSecIP),
		PeerIDLimits: uint64(w.rateLimiter.limitPerSecPeerID),
	}
}

// ConfirmationsEnabled returns true if message confirmations are enabled.
func (w *Waku) ConfirmationsEnabled() bool {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.EnableConfirmations
}

// CurrentTime returns current time.
func (w *Waku) CurrentTime() time.Time {
	return w.timeSource()
}

// SetTimeSource assigns a particular source of time to a waku object.
func (w *Waku) SetTimeSource(timesource func() time.Time) {
	w.timeSource = timesource
}

// APIs returns the RPC descriptors the Waku implementation offers
func (w *Waku) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   NewPublicWakuAPI(w),
			Public:    true,
		},
	}
}

// Protocols returns the waku sub-protocols ran by this particular client.
func (w *Waku) Protocols() []p2p.Protocol {
	return []p2p.Protocol{w.protocol}
}

// RegisterMailServer registers MailServer interface.
// MailServer will process all the incoming messages with p2pRequestCode.
func (w *Waku) RegisterMailServer(server MailServer) {
	w.mailServer = server
}

// SetRateLimiter registers a rate limiter.
func (w *Waku) RegisterRateLimiter(r *PeerRateLimiter) {
	w.rateLimiter = r
}

// SubscribeEnvelopeEvents subscribes to envelopes feed.
// In order to prevent blocking waku producers events must be amply buffered.
func (w *Waku) SubscribeEnvelopeEvents(events chan<- EnvelopeEvent) event.Subscription {
	return w.envelopeFeed.Subscribe(events)
}

func (w *Waku) notifyPeersAboutPowRequirementChange(pow float64) {
	arr := w.getPeers()
	for _, p := range arr {
		err := p.notifyAboutPowRequirementChange(pow)
		if err != nil {
			// allow one retry
			err = p.notifyAboutPowRequirementChange(pow)
		}
		if err != nil {
			w.logger.Warn("failed to notify peer about new pow requirement", zap.Binary("peer", p.ID()), zap.Error(err))
		}
	}
}

func (w *Waku) notifyPeersAboutBloomFilterChange(bloom []byte) {
	arr := w.getPeers()
	for _, p := range arr {
		err := p.notifyAboutBloomFilterChange(bloom)
		if err != nil {
			// allow one retry
			err = p.notifyAboutBloomFilterChange(bloom)
		}
		if err != nil {
			w.logger.Warn("failed to notify peer about new pow requirement", zap.Binary("peer", p.ID()), zap.Error(err))
		}
	}
}

func (w *Waku) getPeers() []*Peer {
	arr := make([]*Peer, len(w.peers))
	i := 0
	w.peerMu.Lock()
	for p := range w.peers {
		arr[i] = p
		i++
	}
	w.peerMu.Unlock()
	return arr
}

// getPeer retrieves peer by ID
func (w *Waku) getPeer(peerID []byte) (*Peer, error) {
	w.peerMu.Lock()
	defer w.peerMu.Unlock()
	for p := range w.peers {
		id := p.peer.ID()
		if bytes.Equal(peerID, id[:]) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("could not find peer with ID: %x", peerID)
}

// AllowP2PMessagesFromPeer marks specific peer trusted,
// which will allow it to send historic (expired) messages.
func (w *Waku) AllowP2PMessagesFromPeer(peerID []byte) error {
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	p.trusted = true
	return nil
}

// RequestHistoricMessages sends a message with p2pRequestCode to a specific peer,
// which is known to implement MailServer interface, and is supposed to process this
// request and respond with a number of peer-to-peer messages (possibly expired),
// which are not supposed to be forwarded any further.
// The waku protocol is agnostic of the format and contents of envelope.
func (w *Waku) RequestHistoricMessages(peerID []byte, envelope *Envelope) error {
	return w.RequestHistoricMessagesWithTimeout(peerID, envelope, 0)
}

// RequestHistoricMessagesWithTimeout acts as RequestHistoricMessages but requires to pass a timeout.
// It sends an event EventMailServerRequestExpired after the timeout.
func (w *Waku) RequestHistoricMessagesWithTimeout(peerID []byte, envelope *Envelope, timeout time.Duration) error {
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	p.trusted = true

	w.envelopeFeed.Send(EnvelopeEvent{
		Peer:  p.peer.ID(),
		Topic: envelope.Topic,
		Hash:  envelope.Hash(),
		Event: EventMailServerRequestSent,
	})

	err = p2p.Send(p.ws, p2pRequestCode, envelope)
	if timeout != 0 {
		go w.expireRequestHistoricMessages(p.peer.ID(), envelope.Hash(), timeout)
	}
	return err
}

func (w *Waku) SendMessagesRequest(peerID []byte, request MessagesRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	p.trusted = true
	if err := p2p.Send(p.ws, p2pRequestCode, request); err != nil {
		return err
	}
	w.envelopeFeed.Send(EnvelopeEvent{
		Peer:  p.peer.ID(),
		Hash:  common.BytesToHash(request.ID),
		Event: EventMailServerRequestSent,
	})
	return nil
}

func (w *Waku) expireRequestHistoricMessages(peer enode.ID, hash common.Hash, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-w.quit:
		return
	case <-timer.C:
		w.envelopeFeed.Send(EnvelopeEvent{
			Peer:  peer,
			Hash:  hash,
			Event: EventMailServerRequestExpired,
		})
	}
}

func (w *Waku) SendHistoricMessageResponse(peerID []byte, payload []byte) error {
	size, r, err := rlp.EncodeToReader(payload)
	if err != nil {
		return err
	}
	peer, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	return peer.ws.WriteMsg(p2p.Msg{Code: p2pRequestCompleteCode, Size: uint32(size), Payload: r})
}

// SendP2PMessage sends a peer-to-peer message to a specific peer.
// It sends one or more envelopes in a single batch.
func (w *Waku) SendP2PMessages(peerID []byte, envelopes ...*Envelope) error {
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	return p2p.Send(p.ws, p2pMessageCode, envelopes)
}

// SendP2PDirect sends a peer-to-peer message to a specific peer.
// It sends one or more envelopes in a single batch.
func (w *Waku) SendP2PDirect(peerID []byte, envelopes ...*Envelope) error {
	peer, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	return p2p.Send(peer.ws, p2pMessageCode, envelopes)
}

// SendRawP2PDirect sends a peer-to-peer message to a specific peer.
// It sends one or more envelopes in a single batch.
func (w *Waku) SendRawP2PDirect(peerID []byte, envelopes ...rlp.RawValue) error {
	peer, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	return p2p.Send(peer.ws, p2pMessageCode, envelopes)
}

// NewKeyPair generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption. Returns ID of the new key pair.
func (w *Waku) NewKeyPair() (string, error) {
	key, err := crypto.GenerateKey()
	if err != nil || !validatePrivateKey(key) {
		key, err = crypto.GenerateKey() // retry once
	}
	if err != nil {
		return "", err
	}
	if !validatePrivateKey(key) {
		return "", fmt.Errorf("failed to generate valid key")
	}

	id, err := toDeterministicID(hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)), keyIDSize)
	if err != nil {
		return "", err
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.privateKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	w.privateKeys[id] = key
	return id, nil
}

// DeleteKeyPair deletes the specified key if it exists.
func (w *Waku) DeleteKeyPair(key string) bool {
	deterministicID, err := toDeterministicID(key, keyIDSize)
	if err != nil {
		return false
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.privateKeys[deterministicID] != nil {
		delete(w.privateKeys, deterministicID)
		return true
	}
	return false
}

// AddKeyPair imports a asymmetric private key and returns it identifier.
func (w *Waku) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	id, err := makeDeterministicID(hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)), keyIDSize)
	if err != nil {
		return "", err
	}
	if w.HasKeyPair(id) {
		return id, nil // no need to re-inject
	}

	w.keyMu.Lock()
	w.privateKeys[id] = key
	w.keyMu.Unlock()

	return id, nil
}

// SelectKeyPair adds cryptographic identity, and makes sure
// that it is the only private key known to the node.
func (w *Waku) SelectKeyPair(key *ecdsa.PrivateKey) error {
	id, err := makeDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIDSize)
	if err != nil {
		return err
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	w.privateKeys = make(map[string]*ecdsa.PrivateKey) // reset key store
	w.privateKeys[id] = key

	return nil
}

// DeleteKeyPairs removes all cryptographic identities known to the node
func (w *Waku) DeleteKeyPairs() error {
	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	w.privateKeys = make(map[string]*ecdsa.PrivateKey)

	return nil
}

// HasKeyPair checks if the waku node is configured with the private key
// of the specified public pair.
func (w *Waku) HasKeyPair(id string) bool {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return false
	}

	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	return w.privateKeys[deterministicID] != nil
}

// GetPrivateKey retrieves the private key of the specified identity.
func (w *Waku) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return nil, err
	}

	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	key := w.privateKeys[deterministicID]
	if key == nil {
		return nil, fmt.Errorf("invalid id")
	}
	return key, nil
}

// GenerateSymKey generates a random symmetric key and stores it under id,
// which is then returned. Will be used in the future for session key exchange.
func (w *Waku) GenerateSymKey() (string, error) {
	key, err := generateSecureRandomData(aesKeyLength)
	if err != nil {
		return "", err
	} else if !validateDataIntegrity(key, aesKeyLength) {
		return "", fmt.Errorf("error in GenerateSymKey: crypto/rand failed to generate random data")
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.symKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	w.symKeys[id] = key
	return id, nil
}

// AddSymKey stores the key with a given id.
func (w *Waku) AddSymKey(id string, key []byte) (string, error) {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return "", err
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.symKeys[deterministicID] != nil {
		return "", fmt.Errorf("key already exists: %v", id)
	}
	w.symKeys[deterministicID] = key
	return deterministicID, nil
}

// AddSymKeyDirect stores the key, and returns its id.
func (w *Waku) AddSymKeyDirect(key []byte) (string, error) {
	if len(key) != aesKeyLength {
		return "", fmt.Errorf("wrong key size: %d", len(key))
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.symKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	w.symKeys[id] = key
	return id, nil
}

// AddSymKeyFromPassword generates the key from password, stores it, and returns its id.
func (w *Waku) AddSymKeyFromPassword(password string) (string, error) {
	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}
	if w.HasSymKey(id) {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	// kdf should run no less than 0.1 seconds on an average computer,
	// because it's an once in a session experience
	derived := pbkdf2.Key([]byte(password), nil, 65356, aesKeyLength, sha256.New)
	if err != nil {
		return "", err
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	// double check is necessary, because deriveKeyMaterial() is very slow
	if w.symKeys[id] != nil {
		return "", fmt.Errorf("critical error: failed to generate unique ID")
	}
	w.symKeys[id] = derived
	return id, nil
}

// HasSymKey returns true if there is a key associated with the given id.
// Otherwise returns false.
func (w *Waku) HasSymKey(id string) bool {
	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	return w.symKeys[id] != nil
}

// DeleteSymKey deletes the key associated with the name string if it exists.
func (w *Waku) DeleteSymKey(id string) bool {
	w.keyMu.Lock()
	defer w.keyMu.Unlock()
	if w.symKeys[id] != nil {
		delete(w.symKeys, id)
		return true
	}
	return false
}

// GetSymKey returns the symmetric key associated with the given id.
func (w *Waku) GetSymKey(id string) ([]byte, error) {
	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	if w.symKeys[id] != nil {
		return w.symKeys[id], nil
	}
	return nil, fmt.Errorf("non-existent key ID")
}

// Subscribe installs a new message handler used for filtering, decrypting
// and subsequent storing of incoming messages.
func (w *Waku) Subscribe(f *Filter) (string, error) {
	s, err := w.filters.Install(f)
	if err == nil {
		w.updateBloomFilter(f)
	}
	return s, err
}

// updateBloomFilter recalculates the new value of bloom filter,
// and informs the peers if necessary.
func (w *Waku) updateBloomFilter(f *Filter) {
	aggregate := make([]byte, BloomFilterSize)
	for _, t := range f.Topics {
		top := BytesToTopic(t)
		b := TopicToBloom(top)
		aggregate = addBloom(aggregate, b)
	}

	if !BloomFilterMatch(w.BloomFilter(), aggregate) {
		// existing bloom filter must be updated
		aggregate = addBloom(w.BloomFilter(), aggregate)
		w.SetBloomFilter(aggregate)
	}
}

// GetFilter returns the filter by id.
func (w *Waku) GetFilter(id string) *Filter {
	return w.filters.Get(id)
}

// Unsubscribe removes an installed message handler.
func (w *Waku) Unsubscribe(id string) error {
	ok := w.filters.Uninstall(id)
	if !ok {
		return fmt.Errorf("Unsubscribe: Invalid ID")
	}
	return nil
}

// Send injects a message into the waku send queue, to be distributed in the
// network in the coming cycles.
func (w *Waku) Send(envelope *Envelope) error {
	ok, err := w.add(envelope, false)
	if err == nil && !ok {
		return fmt.Errorf("failed to add envelope")
	}
	return err
}

// Start implements node.Service, starting the background data propagation thread
// of the Waku protocol.
func (w *Waku) Start(*p2p.Server) error {
	go w.update()

	numCPU := runtime.NumCPU()
	for i := 0; i < numCPU; i++ {
		go w.processQueue()
	}
	go w.processP2P()

	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Waku protocol.
func (w *Waku) Stop() error {
	close(w.quit)
	return nil
}

// HandlePeer is called by the underlying P2P layer when the waku sub-protocol
// connection is negotiated.
func (w *Waku) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Create the new peer and start tracking it
	wakuPeer := newPeer(w, peer, rw, w.logger.Named("waku/peer"))

	w.peerMu.Lock()
	w.peers[wakuPeer] = struct{}{}
	w.peerMu.Unlock()

	defer func() {
		w.peerMu.Lock()
		delete(w.peers, wakuPeer)
		w.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := wakuPeer.handshake(); err != nil {
		return err
	}
	wakuPeer.start()
	defer wakuPeer.stop()

	if w.rateLimiter != nil {
		return w.rateLimiter.decorate(wakuPeer, rw, w.runMessageLoop)
	}
	return w.runMessageLoop(wakuPeer, rw)
}

// sendConfirmation sends messageResponseCode and batchAcknowledgedCode messages.
func (w *Waku) sendConfirmation(rw p2p.MsgReadWriter, data []byte, envelopeErrors []EnvelopeError) (err error) {
	batchHash := crypto.Keccak256Hash(data)
	err = p2p.Send(rw, messageResponseCode, NewMessagesResponse(batchHash, envelopeErrors))
	err = p2p.Send(rw, batchAcknowledgedCode, batchHash) // DEPRECATED
	return
}

// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (w *Waku) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	logger := w.logger.Named("runMessageLoop")
	peerID := p.peer.ID()

	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			logger.Info("failed to read a message", zap.Binary("peer", peerID[:]), zap.Error(err))
			return err
		}

		if packet.Size > w.MaxMessageSize() {
			logger.Warn("oversize message received", zap.Binary("peer", peerID[:]), zap.Uint32("size", packet.Size))
			return errors.New("oversize message received")
		}

		switch packet.Code {
		case messagesCode:
			if err := w.handleMessagesCode(p, rw, packet, logger); err != nil {
				logger.Warn("failed to handle messagesCode message, peer will be disconnected", zap.Binary("peer", peerID[:]), zap.Error(err))
				return err
			}
		case messageResponseCode:
			if err := w.handleMessageResponseCode(p, packet, logger); err != nil {
				logger.Warn("failed to handle messageResponseCode message, peer will be disconnected", zap.Binary("peer", peerID[:]), zap.Error(err))
				return err
			}
		case batchAcknowledgedCode:
			if err := w.handleBatchAcknowledgeCode(p, packet, logger); err != nil {

			}
		case powRequirementCode:
			if err := w.handlePowRequirementCode(p, packet, logger); err != nil {
				logger.Warn("failed to handle powRequirementCode message, peer will be disconnected", zap.Binary("peer", peerID[:]), zap.Error(err))
				return err
			}
		case bloomFilterExCode:
			if err := w.handleBloomFilterExCode(p, packet, logger); err != nil {
				logger.Warn("failed to decode bloom filter exchange message, peer will be disconnected", zap.Binary("peer", peerID[:]), zap.Error(err))
				return err
			}
		case p2pMessageCode:
			if err := w.handleP2PMessageCode(p, packet, logger); err != nil {
				logger.Warn("failed to decode direct message, peer will be disconnected", zap.Binary("peer", peerID[:]), zap.Error(err))
				return err
			}
		case p2pRequestCode:
			if err := w.handleP2PRequestCode(p, packet, logger); err != nil {
				logger.Warn("failed to decode p2p request message, peer will be disconnected", zap.Binary("peer", peerID[:]), zap.Error(err))
				return err
			}
		case p2pRequestCompleteCode:
			if err := w.handleP2PRequestCompleteCode(p, packet, logger); err != nil {
				logger.Warn("failed to decode p2p request complete message, peer will be disconnected", zap.Binary("peer", peerID[:]), zap.Error(err))
				return err
			}
		default:
			// New message types might be implemented in the future versions of Waku.
			// For forward compatibility, just ignore.
			logger.Debug("ignored packet with message code", zap.Uint64("code", packet.Code))
		}

		_ = packet.Discard()
	}
}

func (w *Waku) handleMessagesCode(p *Peer, rw p2p.MsgReadWriter, packet p2p.Msg, logger *zap.Logger) error {
	peerID := p.peer.ID()

	// decode the contained envelopes
	data, err := ioutil.ReadAll(packet.Payload)
	if err != nil {
		envelopesRejectedCounter.WithLabelValues("failed_read").Inc()
		return fmt.Errorf("failed to read packet payload: %w", err)
	}

	var envelopes []*Envelope
	if err := rlp.DecodeBytes(data, &envelopes); err != nil {
		envelopesRejectedCounter.WithLabelValues("invalid_data").Inc()
		return fmt.Errorf("invalid payload: %w", err)
	}

	envelopeErrors := make([]EnvelopeError, 0)
	trouble := false
	for _, env := range envelopes {
		cached, err := w.add(env, w.LightClientMode())
		if err != nil {
			_, isTimeSyncError := err.(TimeSyncError)
			if !isTimeSyncError {
				trouble = true
				logger.Info("invalid envelope received", zap.Binary("peer", peerID[:]), zap.Error(err))
			}
			envelopeErrors = append(envelopeErrors, ErrorToEnvelopeError(env.Hash(), err))
		} else if cached {
			p.mark(env)
		}

		w.envelopeFeed.Send(EnvelopeEvent{
			Event: EventEnvelopeReceived,
			Topic: env.Topic,
			Hash:  env.Hash(),
			Peer:  p.peer.ID(),
		})
		envelopesValidatedCounter.Inc()
	}

	if w.ConfirmationsEnabled() {
		go w.sendConfirmation(rw, data, envelopeErrors)
	}

	if trouble {
		return errors.New("received invalid envelope")
	}
	return nil
}

func (w *Waku) handlePowRequirementCode(p *Peer, packet p2p.Msg, logger *zap.Logger) error {
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	i, err := s.Uint()
	if err != nil {
		envelopesRejectedCounter.WithLabelValues("invalid_pow_req").Inc()
		return fmt.Errorf("invalid powRequirementCode message: %w", err)
	}
	f := math.Float64frombits(i)
	if math.IsInf(f, 0) || math.IsNaN(f) || f < 0.0 {
		envelopesRejectedCounter.WithLabelValues("invalid_pow_req").Inc()
		return errors.New("invalid value in powRequirementCode message")
	}
	p.powRequirement = f
	return nil
}

func (w *Waku) handleBloomFilterExCode(p *Peer, packet p2p.Msg, logger *zap.Logger) error {
	var bloom []byte
	err := packet.Decode(&bloom)
	if err == nil && len(bloom) != BloomFilterSize {
		err = fmt.Errorf("wrong bloom filter size %d", len(bloom))
	}
	if err != nil {
		envelopesRejectedCounter.WithLabelValues("invalid_bloom").Inc()
		return errors.New("invalid bloom filter exchange message")
	}

	p.setBloomFilter(bloom)
	return nil
}

func (w *Waku) handleP2PMessageCode(p *Peer, packet p2p.Msg, logger *zap.Logger) error {
	// peer-to-peer message, sent directly to peer bypassing PoW checks, etc.
	// this message is not supposed to be forwarded to other peers, and
	// therefore might not satisfy the PoW, expiry and other requirements.
	// these messages are only accepted from the trusted peer.
	if !p.trusted {
		return nil
	}

	var (
		envelopes []*Envelope
		err       error
	)

	if err = packet.Decode(&envelopes); err != nil {
		return fmt.Errorf("invalid direct message payload: %w", err)
	}

	for _, envelope := range envelopes {
		w.postP2P(envelope)
	}
	return nil
}

func (w *Waku) handleP2PRequestCode(p *Peer, packet p2p.Msg, logger *zap.Logger) error {
	peerID := p.peer.ID()

	// Must be processed if mail server is implemented. Otherwise ignore.
	if w.mailServer == nil {
		return nil
	}

	// Read all data as we will try to decode it possibly twice.
	data, err := ioutil.ReadAll(packet.Payload)
	if err != nil {
		return fmt.Errorf("invalid p2p request messages: %w", err)
	}
	r := bytes.NewReader(data)
	packet.Payload = r

	var requestDeprecated Envelope
	errDepReq := packet.Decode(&requestDeprecated)
	if errDepReq == nil {
		w.mailServer.DeliverMail(p.ID(), &requestDeprecated)
		return nil
	} else {
		logger.Info("failed to decode p2p request message (deprecated)", zap.Binary("peer", peerID[:]), zap.Error(errDepReq))
	}

	// As we failed to decode the request, let's set the offset
	// to the beginning and try decode it again.
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("invalid p2p request message: %w", err)
	}

	var request MessagesRequest
	errReq := packet.Decode(&request)
	if errReq == nil {
		w.mailServer.Deliver(p.ID(), request)
		return nil
	} else {
		logger.Info("failed to decode p2p request message", zap.Binary("peer", peerID[:]), zap.Error(errDepReq))
	}

	return errors.New("invalid p2p request message")
}

func (w *Waku) handleP2PRequestCompleteCode(p *Peer, packet p2p.Msg, logger *zap.Logger) error {
	if !p.trusted {
		return nil
	}

	var payload []byte
	if err := packet.Decode(&payload); err != nil {
		return fmt.Errorf("invalid p2p request complete message: %w", err)
	}

	event, err := CreateMailServerEvent(p.peer.ID(), payload)
	if err != nil {
		return fmt.Errorf("invalid p2p request complete payload: %w", err)
	}

	w.postP2P(*event)
	return nil
}

func (w *Waku) handleMessageResponseCode(p *Peer, packet p2p.Msg, logger *zap.Logger) error {
	var resp MultiVersionResponse
	if err := packet.Decode(&resp); err != nil {
		envelopesRejectedCounter.WithLabelValues("failed_read").Inc()
		return fmt.Errorf("invalid response message: %w", err)
	}
	if resp.Version != 1 {
		logger.Info("received unsupported version of MultiVersionResponse for messageResponseCode packet", zap.Uint("version", resp.Version))
		return nil
	}

	response, err := resp.DecodeResponse1()
	if err != nil {
		envelopesRejectedCounter.WithLabelValues("invalid_data").Inc()
		return fmt.Errorf("failed to decode response message: %w", err)
	}

	w.envelopeFeed.Send(EnvelopeEvent{
		Batch: response.Hash,
		Event: EventBatchAcknowledged,
		Peer:  p.peer.ID(),
		Data:  response.Errors,
	})

	return nil
}

func (w *Waku) handleBatchAcknowledgeCode(p *Peer, packet p2p.Msg, logger *zap.Logger) error {
	var batchHash common.Hash
	if err := packet.Decode(&batchHash); err != nil {
		return fmt.Errorf("invalid batch ack message: %w", err)
	}
	w.envelopeFeed.Send(EnvelopeEvent{
		Batch: batchHash,
		Event: EventBatchAcknowledged,
		Peer:  p.peer.ID(),
	})
	return nil
}

// add inserts a new envelope into the message pool to be distributed within the
// waku network. It also inserts the envelope into the expiration pool at the
// appropriate time-stamp. In case of error, connection should be dropped.
// param isP2P indicates whether the message is peer-to-peer (should not be forwarded).
func (w *Waku) add(envelope *Envelope, isP2P bool) (bool, error) {
	now := uint32(w.timeSource().Unix())
	sent := envelope.Expiry - envelope.TTL

	envelopesReceivedCounter.Inc()
	if sent > now {
		if sent-DefaultSyncAllowance > now {
			envelopesCacheFailedCounter.WithLabelValues("in_future").Inc()
			log.Warn("envelope created in the future", "hash", envelope.Hash())
			return false, TimeSyncError(errors.New("envelope from future"))
		}
		// recalculate PoW, adjusted for the time difference, plus one second for latency
		envelope.calculatePoW(sent - now + 1)
	}

	if envelope.Expiry < now {
		if envelope.Expiry+DefaultSyncAllowance*2 < now {
			envelopesCacheFailedCounter.WithLabelValues("very_old").Inc()
			log.Warn("very old envelope", "hash", envelope.Hash())
			return false, TimeSyncError(errors.New("very old envelope"))
		}
		log.Debug("expired envelope dropped", "hash", envelope.Hash().Hex())
		envelopesCacheFailedCounter.WithLabelValues("expired").Inc()
		return false, nil // drop envelope without error
	}

	if uint32(envelope.size()) > w.MaxMessageSize() {
		envelopesCacheFailedCounter.WithLabelValues("oversized").Inc()
		return false, fmt.Errorf("huge messages are not allowed [%x]", envelope.Hash())
	}

	if envelope.PoW() < w.MinPow() {
		// maybe the value was recently changed, and the peers did not adjust yet.
		// in this case the previous value is retrieved by MinPowTolerance()
		// for a short period of peer synchronization.
		if envelope.PoW() < w.MinPowTolerance() {
			envelopesCacheFailedCounter.WithLabelValues("low_pow").Inc()
			return false, fmt.Errorf("envelope with low PoW received: PoW=%f, hash=[%v]", envelope.PoW(), envelope.Hash().Hex())
		}
	}

	if !BloomFilterMatch(w.BloomFilter(), envelope.Bloom()) {
		// maybe the value was recently changed, and the peers did not adjust yet.
		// in this case the previous value is retrieved by BloomFilterTolerance()
		// for a short period of peer synchronization.
		if !BloomFilterMatch(w.BloomFilterTolerance(), envelope.Bloom()) {
			envelopesCacheFailedCounter.WithLabelValues("no_bloom_match").Inc()
			return false, fmt.Errorf("envelope does not match bloom filter, hash=[%v], bloom: \n%x \n%x \n%x",
				envelope.Hash().Hex(), w.BloomFilter(), envelope.Bloom(), envelope.Topic)
		}
	}

	hash := envelope.Hash()

	w.poolMu.Lock()
	_, alreadyCached := w.envelopes[hash]
	if !alreadyCached {
		w.envelopes[hash] = envelope
		if w.expirations[envelope.Expiry] == nil {
			w.expirations[envelope.Expiry] = mapset.NewThreadUnsafeSet()
		}
		if !w.expirations[envelope.Expiry].Contains(hash) {
			w.expirations[envelope.Expiry].Add(hash)
		}
	}
	w.poolMu.Unlock()

	if alreadyCached {
		log.Trace("w envelope already cached", "hash", envelope.Hash().Hex())
		envelopesCachedCounter.WithLabelValues("hit").Inc()
	} else {
		log.Trace("cached w envelope", "hash", envelope.Hash().Hex())
		envelopesCachedCounter.WithLabelValues("miss").Inc()
		envelopesSizeMeter.Observe(float64(envelope.size()))
		w.postEvent(envelope, isP2P) // notify the local node about the new message
		if w.mailServer != nil {
			w.mailServer.Archive(envelope)
			w.envelopeFeed.Send(EnvelopeEvent{
				Topic: envelope.Topic,
				Hash:  envelope.Hash(),
				Event: EventMailServerEnvelopeArchived,
			})
		}
	}
	return true, nil
}

func (w *Waku) postP2P(event interface{}) {
	w.p2pMsgQueue <- event
}

// postEvent queues the message for further processing.
func (w *Waku) postEvent(envelope *Envelope, isP2P bool) {
	if isP2P {
		w.postP2P(envelope)
	} else {
		w.msgQueue <- envelope
	}
}

// processQueue delivers the messages to the watchers during the lifetime of the waku node.
func (w *Waku) processQueue() {
	for {
		select {
		case <-w.quit:
			return
		case e := <-w.msgQueue:
			w.filters.NotifyWatchers(e, false)
			w.envelopeFeed.Send(EnvelopeEvent{
				Topic: e.Topic,
				Hash:  e.Hash(),
				Event: EventEnvelopeAvailable,
			})
		}
	}
}

func (w *Waku) processP2P() {
	for {
		select {
		case <-w.quit:
			return
		case e := <-w.p2pMsgQueue:
			switch event := e.(type) {
			case *Envelope:
				w.filters.NotifyWatchers(event, true)
				w.envelopeFeed.Send(EnvelopeEvent{
					Topic: event.Topic,
					Hash:  event.Hash(),
					Event: EventEnvelopeAvailable,
				})
			case EnvelopeEvent:
				w.envelopeFeed.Send(event)
			}
		}
	}
}

// update loops until the lifetime of the waku node, updating its internal
// state by expiring stale messages from the pool.
func (w *Waku) update() {
	// Start a ticker to check for expirations
	expire := time.NewTicker(expirationCycle)

	// Repeat updates until termination is requested
	for {
		select {
		case <-expire.C:
			w.expire()

		case <-w.quit:
			return
		}
	}
}

// expire iterates over all the expiration timestamps, removing all stale
// messages from the pools.
func (w *Waku) expire() {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	now := uint32(w.timeSource().Unix())
	for expiry, hashSet := range w.expirations {
		if expiry < now {
			// Dump all expired messages and remove timestamp
			hashSet.Each(func(v interface{}) bool {
				delete(w.envelopes, v.(common.Hash))
				envelopesCachedCounter.WithLabelValues("clear").Inc()
				w.envelopeFeed.Send(EnvelopeEvent{
					Hash:  v.(common.Hash),
					Event: EventEnvelopeExpired,
				})
				return false
			})
			w.expirations[expiry].Clear()
			delete(w.expirations, expiry)
		}
	}
}

// Envelopes retrieves all the messages currently pooled by the node.
func (w *Waku) Envelopes() []*Envelope {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	all := make([]*Envelope, 0, len(w.envelopes))
	for _, envelope := range w.envelopes {
		all = append(all, envelope)
	}
	return all
}

// GetEnvelope retrieves an envelope from the message queue by its hash.
// It returns nil if the envelope can not be found.
func (w *Waku) GetEnvelope(hash common.Hash) *Envelope {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()
	return w.envelopes[hash]
}

// isEnvelopeCached checks if envelope with specific hash has already been received and cached.
func (w *Waku) isEnvelopeCached(hash common.Hash) bool {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	_, exist := w.envelopes[hash]
	return exist
}

// ValidatePublicKey checks the format of the given public key.
func ValidatePublicKey(k *ecdsa.PublicKey) bool {
	return k != nil && k.X != nil && k.Y != nil && k.X.Sign() != 0 && k.Y.Sign() != 0
}

// validatePrivateKey checks the format of the given private key.
func validatePrivateKey(k *ecdsa.PrivateKey) bool {
	if k == nil || k.D == nil || k.D.Sign() == 0 {
		return false
	}
	return ValidatePublicKey(&k.PublicKey)
}

// validateDataIntegrity returns false if the data have the wrong or contains all zeros,
// which is the simplest and the most common bug.
func validateDataIntegrity(k []byte, expectedSize int) bool {
	if len(k) != expectedSize {
		return false
	}
	if expectedSize > 3 && containsOnlyZeros(k) {
		return false
	}
	return true
}

// containsOnlyZeros checks if the data contain only zeros.
func containsOnlyZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// bytesToUintLittleEndian converts the slice to 64-bit unsigned integer.
func bytesToUintLittleEndian(b []byte) (res uint64) {
	mul := uint64(1)
	for i := 0; i < len(b); i++ {
		res += uint64(b[i]) * mul
		mul *= 256
	}
	return res
}

// BytesToUintBigEndian converts the slice to 64-bit unsigned integer.
func BytesToUintBigEndian(b []byte) (res uint64) {
	for i := 0; i < len(b); i++ {
		res *= 256
		res += uint64(b[i])
	}
	return res
}

// GenerateRandomID generates a random string, which is then returned to be used as a key id
func GenerateRandomID() (id string, err error) {
	buf, err := generateSecureRandomData(keyIDSize)
	if err != nil {
		return "", err
	}
	if !validateDataIntegrity(buf, keyIDSize) {
		return "", fmt.Errorf("error in generateRandomID: crypto/rand failed to generate random data")
	}
	id = common.Bytes2Hex(buf)
	return id, err
}

// makeDeterministicID generates a deterministic ID, based on a given input
func makeDeterministicID(input string, keyLen int) (id string, err error) {
	buf := pbkdf2.Key([]byte(input), nil, 4096, keyLen, sha256.New)
	if !validateDataIntegrity(buf, keyIDSize) {
		return "", fmt.Errorf("error in GenerateDeterministicID: failed to generate key")
	}
	id = common.Bytes2Hex(buf)
	return id, err
}

// toDeterministicID reviews incoming id, and transforms it to format
// expected internally be private key store. Originally, public keys
// were used as keys, now random keys are being used. And in order to
// make it easier to consume, we now allow both random IDs and public
// keys to be passed.
func toDeterministicID(id string, expectedLen int) (string, error) {
	if len(id) != (expectedLen * 2) { // we received hex key, so number of chars in id is doubled
		var err error
		id, err = makeDeterministicID(id, expectedLen)
		if err != nil {
			return "", err
		}
	}

	return id, nil
}

func isFullNode(bloom []byte) bool {
	if bloom == nil {
		return true
	}
	for _, b := range bloom {
		if b != 255 {
			return false
		}
	}
	return true
}

func BloomFilterMatch(filter, sample []byte) bool {
	if filter == nil {
		return true
	}

	for i := 0; i < BloomFilterSize; i++ {
		f := filter[i]
		s := sample[i]
		if (f | s) != f {
			return false
		}
	}

	return true
}

func addBloom(a, b []byte) []byte {
	c := make([]byte, BloomFilterSize)
	for i := 0; i < BloomFilterSize; i++ {
		c[i] = a[i] | b[i]
	}
	return c
}
