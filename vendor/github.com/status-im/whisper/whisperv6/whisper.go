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

package whisperv6

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

	mapset "github.com/deckarep/golang-set"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/sync/syncmap"
)

// TimeSyncError error for clock skew errors.
type TimeSyncError error

// Statistics holds several message-related counter for analytics
// purposes.
type Statistics struct {
	messagesCleared      int
	memoryCleared        int
	memoryUsed           int
	cycles               int
	totalMessagesCleared int
}

const (
	maxMsgSizeIdx                            = iota // Maximal message length allowed by the whisper node
	overflowIdx                                     // Indicator of message queue overflow
	minPowIdx                                       // Minimal PoW required by the whisper node
	minPowToleranceIdx                              // Minimal PoW tolerated by the whisper node for a limited time
	bloomFilterIdx                                  // Bloom filter for topics of interest for this node
	bloomFilterToleranceIdx                         // Bloom filter tolerated by the whisper node for a limited time
	lightClientModeIdx                              // Light client mode. (does not forward any messages)
	restrictConnectionBetweenLightClientsIdx        // Restrict connection between two light clients
)

// MailServerResponse is the response payload sent by the mailserver
type MailServerResponse struct {
	LastEnvelopeHash common.Hash
	Cursor           []byte
	Error            error
}

// Whisper represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Whisper struct {
	protocol p2p.Protocol // Protocol description and parameters
	filters  *Filters     // Message filters installed with Subscribe function

	privateKeys map[string]*ecdsa.PrivateKey // Private key storage
	symKeys     map[string][]byte            // Symmetric key storage
	keyMu       sync.RWMutex                 // Mutex associated with key storages

	poolMu      sync.RWMutex              // Mutex to sync the message and expiration pools
	envelopes   map[common.Hash]*Envelope // Pool of envelopes currently tracked by this node
	expirations map[uint32]mapset.Set     // Message expiration pool

	peerMu sync.RWMutex       // Mutex to sync the active peer set
	peers  map[*Peer]struct{} // Set of currently active peers

	messageQueue chan *Envelope // Message queue for normal whisper messages
	p2pMsgQueue  chan *Envelope // Message queue for peer-to-peer messages (not to be forwarded any further)
	quit         chan struct{}  // Channel used for graceful exit

	settings syncmap.Map // holds configuration settings that can be dynamically changed

	disableConfirmations bool // do not reply with confirmations

	syncAllowance int // maximum time in seconds allowed to process the whisper-related messages

	statsMu sync.Mutex // guard stats
	stats   Statistics // Statistics of whisper node

	mailServer MailServer // MailServer interface

	envelopeFeed event.Feed

	timeSource func() time.Time // source of time for whisper
}

// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func New(cfg *Config) *Whisper {
	if cfg == nil {
		cfg = &DefaultConfig
	}

	whisper := &Whisper{
		privateKeys:          make(map[string]*ecdsa.PrivateKey),
		symKeys:              make(map[string][]byte),
		envelopes:            make(map[common.Hash]*Envelope),
		expirations:          make(map[uint32]mapset.Set),
		peers:                make(map[*Peer]struct{}),
		messageQueue:         make(chan *Envelope, messageQueueLimit),
		p2pMsgQueue:          make(chan *Envelope, messageQueueLimit),
		quit:                 make(chan struct{}),
		syncAllowance:        DefaultSyncAllowance,
		timeSource:           time.Now,
		disableConfirmations: cfg.DisableConfirmations,
	}

	whisper.filters = NewFilters(whisper)

	whisper.settings.Store(minPowIdx, cfg.MinimumAcceptedPOW)
	whisper.settings.Store(maxMsgSizeIdx, cfg.MaxMessageSize)
	whisper.settings.Store(overflowIdx, false)
	whisper.settings.Store(restrictConnectionBetweenLightClientsIdx, cfg.RestrictConnectionBetweenLightClients)

	// p2p whisper sub protocol handler
	whisper.protocol = p2p.Protocol{
		Name:    ProtocolName,
		Version: uint(ProtocolVersion),
		Length:  NumberOfMessageCodes,
		Run:     whisper.HandlePeer,
		NodeInfo: func() interface{} {
			return map[string]interface{}{
				"version":        ProtocolVersionStr,
				"maxMessageSize": whisper.MaxMessageSize(),
				"minimumPoW":     whisper.MinPow(),
			}
		},
	}

	return whisper
}

// SetTimeSource assigns a particular source of time to a whisper object.
func (whisper *Whisper) SetTimeSource(timesource func() time.Time) {
	whisper.timeSource = timesource
}

// SubscribeEnvelopeEvents subscribes to envelopes feed.
// In order to prevent blocking whisper producers events must be amply buffered.
func (whisper *Whisper) SubscribeEnvelopeEvents(events chan<- EnvelopeEvent) event.Subscription {
	return whisper.envelopeFeed.Subscribe(events)
}

// MinPow returns the PoW value required by this node.
func (whisper *Whisper) MinPow() float64 {
	val, exist := whisper.settings.Load(minPowIdx)
	if !exist || val == nil {
		return DefaultMinimumPoW
	}
	v, ok := val.(float64)
	if !ok {
		log.Error("Error loading minPowIdx, using default")
		return DefaultMinimumPoW
	}
	return v
}

// MinPowTolerance returns the value of minimum PoW which is tolerated for a limited
// time after PoW was changed. If sufficient time have elapsed or no change of PoW
// have ever occurred, the return value will be the same as return value of MinPow().
func (whisper *Whisper) MinPowTolerance() float64 {
	val, exist := whisper.settings.Load(minPowToleranceIdx)
	if !exist || val == nil {
		return DefaultMinimumPoW
	}
	return val.(float64)
}

// BloomFilter returns the aggregated bloom filter for all the topics of interest.
// The nodes are required to send only messages that match the advertised bloom filter.
// If a message does not match the bloom, it will tantamount to spam, and the peer will
// be disconnected.
func (whisper *Whisper) BloomFilter() []byte {
	val, exist := whisper.settings.Load(bloomFilterIdx)
	if !exist || val == nil {
		return nil
	}
	return val.([]byte)
}

// BloomFilterTolerance returns the bloom filter which is tolerated for a limited
// time after new bloom was advertised to the peers. If sufficient time have elapsed
// or no change of bloom filter have ever occurred, the return value will be the same
// as return value of BloomFilter().
func (whisper *Whisper) BloomFilterTolerance() []byte {
	val, exist := whisper.settings.Load(bloomFilterToleranceIdx)
	if !exist || val == nil {
		return nil
	}
	return val.([]byte)
}

// MaxMessageSize returns the maximum accepted message size.
func (whisper *Whisper) MaxMessageSize() uint32 {
	val, _ := whisper.settings.Load(maxMsgSizeIdx)
	return val.(uint32)
}

// Overflow returns an indication if the message queue is full.
func (whisper *Whisper) Overflow() bool {
	val, _ := whisper.settings.Load(overflowIdx)
	return val.(bool)
}

// APIs returns the RPC descriptors the Whisper implementation offers
func (whisper *Whisper) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   NewPublicWhisperAPI(whisper),
			Public:    true,
		},
	}
}

// GetCurrentTime returns current time.
func (whisper *Whisper) GetCurrentTime() time.Time {
	return whisper.timeSource()
}

// RegisterServer registers MailServer interface.
// MailServer will process all the incoming messages with p2pRequestCode.
func (whisper *Whisper) RegisterServer(server MailServer) {
	whisper.mailServer = server
}

// Protocols returns the whisper sub-protocols ran by this particular client.
func (whisper *Whisper) Protocols() []p2p.Protocol {
	return []p2p.Protocol{whisper.protocol}
}

// Version returns the whisper sub-protocols version number.
func (whisper *Whisper) Version() uint {
	return whisper.protocol.Version
}

// SetMaxMessageSize sets the maximal message size allowed by this node
func (whisper *Whisper) SetMaxMessageSize(size uint32) error {
	if size > MaxMessageSize {
		return fmt.Errorf("message size too large [%d>%d]", size, MaxMessageSize)
	}
	whisper.settings.Store(maxMsgSizeIdx, size)
	return nil
}

// SetBloomFilter sets the new bloom filter
func (whisper *Whisper) SetBloomFilter(bloom []byte) error {
	if len(bloom) != BloomFilterSize {
		return fmt.Errorf("invalid bloom filter size: %d", len(bloom))
	}

	b := make([]byte, BloomFilterSize)
	copy(b, bloom)

	whisper.settings.Store(bloomFilterIdx, b)
	whisper.notifyPeersAboutBloomFilterChange(b)

	go func() {
		// allow some time before all the peers have processed the notification
		time.Sleep(time.Duration(whisper.syncAllowance) * time.Second)
		whisper.settings.Store(bloomFilterToleranceIdx, b)
	}()

	return nil
}

// SetMinimumPoW sets the minimal PoW required by this node
func (whisper *Whisper) SetMinimumPoW(val float64) error {
	if val < 0.0 {
		return fmt.Errorf("invalid PoW: %f", val)
	}

	whisper.settings.Store(minPowIdx, val)
	whisper.notifyPeersAboutPowRequirementChange(val)

	go func() {
		// allow some time before all the peers have processed the notification
		time.Sleep(time.Duration(whisper.syncAllowance) * time.Second)
		whisper.settings.Store(minPowToleranceIdx, val)
	}()

	return nil
}

// SetMinimumPowTest sets the minimal PoW in test environment
func (whisper *Whisper) SetMinimumPowTest(val float64) {
	whisper.settings.Store(minPowIdx, val)
	whisper.notifyPeersAboutPowRequirementChange(val)
	whisper.settings.Store(minPowToleranceIdx, val)
}

//SetLightClientMode makes node light client (does not forward any messages)
func (whisper *Whisper) SetLightClientMode(v bool) {
	whisper.settings.Store(lightClientModeIdx, v)
}

//LightClientMode indicates is this node is light client (does not forward any messages)
func (whisper *Whisper) LightClientMode() bool {
	val, exist := whisper.settings.Load(lightClientModeIdx)
	if !exist || val == nil {
		return false
	}
	v, ok := val.(bool)
	return v && ok
}

//LightClientModeConnectionRestricted indicates that connection to light client in light client mode not allowed
func (whisper *Whisper) LightClientModeConnectionRestricted() bool {
	val, exist := whisper.settings.Load(restrictConnectionBetweenLightClientsIdx)
	if !exist || val == nil {
		return false
	}
	v, ok := val.(bool)
	return v && ok
}

func (whisper *Whisper) notifyPeersAboutPowRequirementChange(pow float64) {
	arr := whisper.getPeers()
	for _, p := range arr {
		err := p.notifyAboutPowRequirementChange(pow)
		if err != nil {
			// allow one retry
			err = p.notifyAboutPowRequirementChange(pow)
		}
		if err != nil {
			log.Warn("failed to notify peer about new pow requirement", "peer", p.ID(), "error", err)
		}
	}
}

func (whisper *Whisper) notifyPeersAboutBloomFilterChange(bloom []byte) {
	arr := whisper.getPeers()
	for _, p := range arr {
		err := p.notifyAboutBloomFilterChange(bloom)
		if err != nil {
			// allow one retry
			err = p.notifyAboutBloomFilterChange(bloom)
		}
		if err != nil {
			log.Warn("failed to notify peer about new bloom filter", "peer", p.ID(), "error", err)
		}
	}
}

func (whisper *Whisper) getPeers() []*Peer {
	arr := make([]*Peer, len(whisper.peers))
	i := 0
	whisper.peerMu.Lock()
	for p := range whisper.peers {
		arr[i] = p
		i++
	}
	whisper.peerMu.Unlock()
	return arr
}

// getPeer retrieves peer by ID
func (whisper *Whisper) getPeer(peerID []byte) (*Peer, error) {
	whisper.peerMu.Lock()
	defer whisper.peerMu.Unlock()
	for p := range whisper.peers {
		id := p.peer.ID()
		if bytes.Equal(peerID, id[:]) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("Could not find peer with ID: %x", peerID)
}

// AllowP2PMessagesFromPeer marks specific peer trusted,
// which will allow it to send historic (expired) messages.
func (whisper *Whisper) AllowP2PMessagesFromPeer(peerID []byte) error {
	p, err := whisper.getPeer(peerID)
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
// The whisper protocol is agnostic of the format and contents of envelope.
func (whisper *Whisper) RequestHistoricMessages(peerID []byte, envelope *Envelope) error {
	return whisper.RequestHistoricMessagesWithTimeout(peerID, envelope, 0)
}

func (whisper *Whisper) RequestHistoricMessagesWithTimeout(peerID []byte, envelope *Envelope, timeout time.Duration) error {
	p, err := whisper.getPeer(peerID)
	if err != nil {
		return err
	}
	whisper.envelopeFeed.Send(EnvelopeEvent{
		Peer:  p.peer.ID(),
		Hash:  envelope.Hash(),
		Event: EventMailServerRequestSent,
	})
	p.trusted = true
	err = p2p.Send(p.ws, p2pRequestCode, envelope)
	if timeout != 0 {
		go whisper.expireRequestHistoricMessages(p.peer.ID(), envelope.Hash(), timeout)
	}
	return err
}

func (whisper *Whisper) expireRequestHistoricMessages(peer enode.ID, hash common.Hash, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-whisper.quit:
		return
	case <-timer.C:
		whisper.envelopeFeed.Send(EnvelopeEvent{
			Peer:  peer,
			Hash:  hash,
			Event: EventMailServerRequestExpired,
		})
	}
}

func (whisper *Whisper) SendHistoricMessageResponse(peer *Peer, payload []byte) error {
	size, r, err := rlp.EncodeToReader(payload)
	if err != nil {
		return err
	}

	return peer.ws.WriteMsg(p2p.Msg{Code: p2pRequestCompleteCode, Size: uint32(size), Payload: r})
}

// SyncMessages can be sent between two Mail Servers and syncs envelopes between them.
func (whisper *Whisper) SyncMessages(peerID []byte, req SyncMailRequest) error {
	if whisper.mailServer == nil {
		return errors.New("can not sync messages if Mail Server is not configured")
	}

	p, err := whisper.getPeer(peerID)
	if err != nil {
		return err
	}

	if err := req.Validate(); err != nil {
		return err
	}

	return p2p.Send(p.ws, p2pSyncRequestCode, req)
}

// SendSyncResponse sends a response to a Mail Server with a slice of envelopes.
func (whisper *Whisper) SendSyncResponse(p *Peer, data SyncResponse) error {
	return p2p.Send(p.ws, p2pSyncResponseCode, data)
}

// SendP2PMessage sends a peer-to-peer message to a specific peer.
func (whisper *Whisper) SendP2PMessage(peerID []byte, envelopes ...*Envelope) error {
	p, err := whisper.getPeer(peerID)
	if err != nil {
		return err
	}
	return whisper.SendP2PDirect(p, envelopes...)
}

// SendP2PDirect sends a peer-to-peer message to a specific peer.
// If only a single envelope is given, data is sent as a single object
// rather than a slice. This is important to keep this method backward compatible
// as it used to send only single envelopes.
func (whisper *Whisper) SendP2PDirect(peer *Peer, envelopes ...*Envelope) error {
	if len(envelopes) == 1 {
		return p2p.Send(peer.ws, p2pMessageCode, envelopes[0])
	}
	return p2p.Send(peer.ws, p2pMessageCode, envelopes)
}

// NewKeyPair generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption. Returns ID of the new key pair.
func (whisper *Whisper) NewKeyPair() (string, error) {
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

	id, err := toDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIDSize)
	if err != nil {
		return "", err
	}

	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()

	if whisper.privateKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	whisper.privateKeys[id] = key
	return id, nil
}

// DeleteKeyPair deletes the specified key if it exists.
func (whisper *Whisper) DeleteKeyPair(key string) bool {
	deterministicID, err := toDeterministicID(key, keyIDSize)
	if err != nil {
		return false
	}

	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()

	if whisper.privateKeys[deterministicID] != nil {
		delete(whisper.privateKeys, deterministicID)
		return true
	}
	return false
}

// AddKeyPair imports a asymmetric private key and returns it identifier.
func (whisper *Whisper) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	id, err := makeDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIDSize)
	if err != nil {
		return "", err
	}
	if whisper.HasKeyPair(id) {
		return id, nil // no need to re-inject
	}

	whisper.keyMu.Lock()
	whisper.privateKeys[id] = key
	whisper.keyMu.Unlock()
	log.Info("Whisper identity added", "id", id, "pubkey", common.ToHex(crypto.FromECDSAPub(&key.PublicKey)))

	return id, nil
}

// SelectKeyPair adds cryptographic identity, and makes sure
// that it is the only private key known to the node.
func (whisper *Whisper) SelectKeyPair(key *ecdsa.PrivateKey) error {
	id, err := makeDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIDSize)
	if err != nil {
		return err
	}

	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()

	whisper.privateKeys = make(map[string]*ecdsa.PrivateKey) // reset key store
	whisper.privateKeys[id] = key

	log.Info("Whisper identity selected", "id", id, "key", common.ToHex(crypto.FromECDSAPub(&key.PublicKey)))
	return nil
}

// DeleteKeyPairs removes all cryptographic identities known to the node
func (whisper *Whisper) DeleteKeyPairs() error {
	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()

	whisper.privateKeys = make(map[string]*ecdsa.PrivateKey)

	return nil
}

// HasKeyPair checks if the whisper node is configured with the private key
// of the specified public pair.
func (whisper *Whisper) HasKeyPair(id string) bool {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return false
	}

	whisper.keyMu.RLock()
	defer whisper.keyMu.RUnlock()
	return whisper.privateKeys[deterministicID] != nil
}

// GetPrivateKey retrieves the private key of the specified identity.
func (whisper *Whisper) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return nil, err
	}

	whisper.keyMu.RLock()
	defer whisper.keyMu.RUnlock()
	key := whisper.privateKeys[deterministicID]
	if key == nil {
		return nil, fmt.Errorf("invalid id")
	}
	return key, nil
}

// GenerateSymKey generates a random symmetric key and stores it under id,
// which is then returned. Will be used in the future for session key exchange.
func (whisper *Whisper) GenerateSymKey() (string, error) {
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

	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()

	if whisper.symKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	whisper.symKeys[id] = key
	return id, nil
}

// AddSymKey stores the key with a given id.
func (whisper *Whisper) AddSymKey(id string, key []byte) (string, error) {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return "", err
	}

	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()

	if whisper.symKeys[deterministicID] != nil {
		return "", fmt.Errorf("key already exists: %v", id)
	}
	whisper.symKeys[deterministicID] = key
	return deterministicID, nil
}

// AddSymKeyDirect stores the key, and returns its id.
func (whisper *Whisper) AddSymKeyDirect(key []byte) (string, error) {
	if len(key) != aesKeyLength {
		return "", fmt.Errorf("wrong key size: %d", len(key))
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}

	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()

	if whisper.symKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	whisper.symKeys[id] = key
	return id, nil
}

// AddSymKeyFromPassword generates the key from password, stores it, and returns its id.
func (whisper *Whisper) AddSymKeyFromPassword(password string) (string, error) {
	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}
	if whisper.HasSymKey(id) {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	// kdf should run no less than 0.1 seconds on an average computer,
	// because it's an once in a session experience
	derived := pbkdf2.Key([]byte(password), nil, 65356, aesKeyLength, sha256.New)
	if err != nil {
		return "", err
	}

	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()

	// double check is necessary, because deriveKeyMaterial() is very slow
	if whisper.symKeys[id] != nil {
		return "", fmt.Errorf("critical error: failed to generate unique ID")
	}
	whisper.symKeys[id] = derived
	return id, nil
}

// HasSymKey returns true if there is a key associated with the given id.
// Otherwise returns false.
func (whisper *Whisper) HasSymKey(id string) bool {
	whisper.keyMu.RLock()
	defer whisper.keyMu.RUnlock()
	return whisper.symKeys[id] != nil
}

// DeleteSymKey deletes the key associated with the name string if it exists.
func (whisper *Whisper) DeleteSymKey(id string) bool {
	whisper.keyMu.Lock()
	defer whisper.keyMu.Unlock()
	if whisper.symKeys[id] != nil {
		delete(whisper.symKeys, id)
		return true
	}
	return false
}

// GetSymKey returns the symmetric key associated with the given id.
func (whisper *Whisper) GetSymKey(id string) ([]byte, error) {
	whisper.keyMu.RLock()
	defer whisper.keyMu.RUnlock()
	if whisper.symKeys[id] != nil {
		return whisper.symKeys[id], nil
	}
	return nil, fmt.Errorf("non-existent key ID")
}

// Subscribe installs a new message handler used for filtering, decrypting
// and subsequent storing of incoming messages.
func (whisper *Whisper) Subscribe(f *Filter) (string, error) {
	s, err := whisper.filters.Install(f)
	if err == nil {
		whisper.updateBloomFilter(f)
	}
	return s, err
}

// updateBloomFilter recalculates the new value of bloom filter,
// and informs the peers if necessary.
func (whisper *Whisper) updateBloomFilter(f *Filter) {
	aggregate := make([]byte, BloomFilterSize)
	for _, t := range f.Topics {
		top := BytesToTopic(t)
		b := TopicToBloom(top)
		aggregate = addBloom(aggregate, b)
	}

	if !BloomFilterMatch(whisper.BloomFilter(), aggregate) {
		// existing bloom filter must be updated
		aggregate = addBloom(whisper.BloomFilter(), aggregate)
		whisper.SetBloomFilter(aggregate)
	}
}

// GetFilter returns the filter by id.
func (whisper *Whisper) GetFilter(id string) *Filter {
	return whisper.filters.Get(id)
}

// Unsubscribe removes an installed message handler.
func (whisper *Whisper) Unsubscribe(id string) error {
	ok := whisper.filters.Uninstall(id)
	if !ok {
		return fmt.Errorf("Unsubscribe: Invalid ID")
	}
	return nil
}

// Send injects a message into the whisper send queue, to be distributed in the
// network in the coming cycles.
func (whisper *Whisper) Send(envelope *Envelope) error {
	ok, err := whisper.add(envelope, false)
	if err == nil && !ok {
		return fmt.Errorf("failed to add envelope")
	}
	return err
}

// Start implements node.Service, starting the background data propagation thread
// of the Whisper protocol.
func (whisper *Whisper) Start(*p2p.Server) error {
	log.Info("started whisper v." + ProtocolVersionStr)
	go whisper.update()

	numCPU := runtime.NumCPU()
	for i := 0; i < numCPU; i++ {
		go whisper.processQueue()
	}

	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Whisper protocol.
func (whisper *Whisper) Stop() error {
	close(whisper.quit)
	log.Info("whisper stopped")
	return nil
}

// HandlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (whisper *Whisper) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Create the new peer and start tracking it
	whisperPeer := newPeer(whisper, peer, rw)

	whisper.peerMu.Lock()
	whisper.peers[whisperPeer] = struct{}{}
	whisper.peerMu.Unlock()

	defer func() {
		whisper.peerMu.Lock()
		delete(whisper.peers, whisperPeer)
		whisper.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := whisperPeer.handshake(); err != nil {
		return err
	}
	whisperPeer.start()
	defer whisperPeer.stop()

	return whisper.runMessageLoop(whisperPeer, rw)
}

func (whisper *Whisper) sendConfirmation(peer enode.ID, rw p2p.MsgReadWriter, data []byte,
	envelopeErrors []EnvelopeError) {
	batchHash := crypto.Keccak256Hash(data)
	if err := p2p.Send(rw, messageResponseCode, NewMessagesResponse(batchHash, envelopeErrors)); err != nil {
		log.Warn("failed to deliver messages response", "hash", batchHash, "envelopes errors", envelopeErrors,
			"peer", peer, "error", err)
	}
	if err := p2p.Send(rw, batchAcknowledgedCode, batchHash); err != nil {
		log.Warn("failed to deliver confirmation", "hash", batchHash, "peer", peer, "error", err)
	}
}

// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (whisper *Whisper) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			log.Info("message loop", "peer", p.peer.ID(), "err", err)
			return err
		}
		if packet.Size > whisper.MaxMessageSize() {
			log.Warn("oversized message received", "peer", p.peer.ID())
			return errors.New("oversized message received")
		}

		switch packet.Code {
		case statusCode:
			// this should not happen, but no need to panic; just ignore this message.
			log.Warn("unxepected status message received", "peer", p.peer.ID())
		case messagesCode:
			// decode the contained envelopes
			data, err := ioutil.ReadAll(packet.Payload)
			if err != nil {
				log.Warn("failed to read envelopes data", "peer", p.peer.ID(), "error", err)
				return errors.New("invalid enveloopes")
			}

			var envelopes []*Envelope
			if err := rlp.DecodeBytes(data, &envelopes); err != nil {
				log.Warn("failed to decode envelopes, peer will be disconnected", "peer", p.peer.ID(), "err", err)
				return errors.New("invalid envelopes")
			}
			trouble := false
			envelopeErrors := []EnvelopeError{}
			for _, env := range envelopes {
				cached, err := whisper.add(env, whisper.LightClientMode())
				if err != nil {
					_, isTimeSyncError := err.(TimeSyncError)
					if !isTimeSyncError {
						trouble = true
						log.Error("bad envelope received, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					}
					envelopeErrors = append(envelopeErrors, ErrorToEnvelopeError(env.Hash(), err))
				}

				whisper.envelopeFeed.Send(EnvelopeEvent{
					Event: EventEnvelopeReceived,
					Hash:  env.Hash(),
					Peer:  p.peer.ID(),
				})
				if cached {
					p.mark(env)
				}
			}
			if !whisper.disableConfirmations {
				go whisper.sendConfirmation(p.peer.ID(), rw, data, envelopeErrors)
			}

			if trouble {
				return errors.New("invalid envelope")
			}
		case messageResponseCode:
			var multiResponse MultiVersionResponse
			if err := packet.Decode(&multiResponse); err != nil {
				log.Error("failed to decode messages response", "peer", p.peer.ID(), "error", err)
				return errors.New("invalid response message")
			}
			if multiResponse.Version == 1 {
				response, err := multiResponse.DecodeResponse1()
				if err != nil {
					log.Error("failed to decode messages response into first version of response", "peer", p.peer.ID(), "error", err)
				}
				whisper.envelopeFeed.Send(EnvelopeEvent{
					Batch: response.Hash,
					Event: EventBatchAcknowledged,
					Peer:  p.peer.ID(),
					Data:  response.Errors,
				})
			} else {
				log.Warn("unknown version of the messages response was received. response is ignored", "peer", p.peer.ID(), "version", multiResponse.Version)
			}
		case batchAcknowledgedCode:
			var batchHash common.Hash
			if err := packet.Decode(&batchHash); err != nil {
				log.Error("failed to decode confirmation into common.Hash", "peer", p.peer.ID(), "error", err)
				return errors.New("invalid confirmation message")
			}
			whisper.envelopeFeed.Send(EnvelopeEvent{
				Batch: batchHash,
				Event: EventBatchAcknowledged,
				Peer:  p.peer.ID(),
			})
		case powRequirementCode:
			s := rlp.NewStream(packet.Payload, uint64(packet.Size))
			i, err := s.Uint()
			if err != nil {
				log.Warn("failed to decode powRequirementCode message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
				return errors.New("invalid powRequirementCode message")
			}
			f := math.Float64frombits(i)
			if math.IsInf(f, 0) || math.IsNaN(f) || f < 0.0 {
				log.Warn("invalid value in powRequirementCode message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
				return errors.New("invalid value in powRequirementCode message")
			}
			p.powRequirement = f
		case bloomFilterExCode:
			var bloom []byte
			err := packet.Decode(&bloom)
			if err == nil && len(bloom) != BloomFilterSize {
				err = fmt.Errorf("wrong bloom filter size %d", len(bloom))
			}

			if err != nil {
				log.Warn("failed to decode bloom filter exchange message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
				return errors.New("invalid bloom filter exchange message")
			}
			p.setBloomFilter(bloom)
		case p2pMessageCode:
			// peer-to-peer message, sent directly to peer bypassing PoW checks, etc.
			// this message is not supposed to be forwarded to other peers, and
			// therefore might not satisfy the PoW, expiry and other requirements.
			// these messages are only accepted from the trusted peer.
			if p.trusted {
				var (
					envelope  *Envelope
					envelopes []*Envelope
					err       error
				)

				// Read all data as we will try to decode it possibly twice
				// to keep backward compatibility.
				data, err := ioutil.ReadAll(packet.Payload)
				if err != nil {
					return fmt.Errorf("invalid direct messages: %v", err)
				}
				r := bytes.NewReader(data)

				packet.Payload = r

				if err = packet.Decode(&envelopes); err == nil {
					for _, envelope := range envelopes {
						whisper.postEvent(envelope, true)
					}
					continue
				}

				// As we failed to decode envelopes, let's set the offset
				// to the beginning and try decode data again.
				// Decoding to a single Envelope is required
				// to be backward compatible.
				if _, err := r.Seek(0, io.SeekStart); err != nil {
					return fmt.Errorf("invalid direct messages: %v", err)
				}

				if err = packet.Decode(&envelope); err == nil {
					whisper.postEvent(envelope, true)
					continue
				}

				if err != nil {
					log.Warn("failed to decode direct message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					return fmt.Errorf("invalid direct message: %v", err)
				}
			}
		case p2pSyncRequestCode:
			// TODO(adam): should we limit who can send this request?
			if whisper.mailServer != nil {
				var request SyncMailRequest
				if err := packet.Decode(&request); err != nil {
					return fmt.Errorf("failed to decode p2pSyncRequestCode payload: %v", err)
				}

				if err := request.Validate(); err != nil {
					return fmt.Errorf("sync mail request was invalid: %v", err)
				}

				if err := whisper.mailServer.SyncMail(p, request); err != nil {
					log.Error("failed to sync envelopes", "peer", p.peer.ID().String())
				}
			} else {
				log.Debug("requested to sync messages but mail servers is not registered", "peer", p.peer.ID().String())
			}
		case p2pSyncResponseCode:
			// TODO(adam): currently, there is no feedback when a sync response
			// is received. An idea to fix this:
			//   1. Sending a request contains an ID,
			//   2. Each sync reponse contains this ID,
			//   3. There is a way to call whisper.SyncMessages() and wait for the response.Final to be received for that particular request ID.
			//   4. If Cursor is not empty, another p2pSyncRequestCode should be sent.
			if p.trusted && whisper.mailServer != nil {
				var resp SyncResponse
				if err = packet.Decode(&resp); err != nil {
					return fmt.Errorf("failed to decode p2pSyncResponseCode payload: %v", err)
				}

				log.Info("received sync response", "count", len(resp.Envelopes), "final", resp.Final, "err", resp.Error, "cursor", resp.Cursor)

				for _, envelope := range resp.Envelopes {
					whisper.mailServer.Archive(envelope)
				}

				if resp.Error != "" || resp.Final {
					whisper.envelopeFeed.Send(EnvelopeEvent{
						Event: EventMailServerSyncFinished,
						Peer:  p.peer.ID(),
						Data: SyncEventResponse{
							Cursor: resp.Cursor,
							Error:  resp.Error,
						},
					})
				}
			}
		case p2pRequestCode:
			// Must be processed if mail server is implemented. Otherwise ignore.
			if whisper.mailServer != nil {
				var request Envelope
				if err := packet.Decode(&request); err != nil {
					log.Warn("failed to decode p2p request message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					return errors.New("invalid p2p request")
				}

				whisper.mailServer.DeliverMail(p, &request)
			}
		case p2pRequestCompleteCode:
			if p.trusted {
				var payload []byte
				if err := packet.Decode(&payload); err != nil {
					log.Warn("failed to decode response message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					return errors.New("invalid request response message")
				}

				event, err := CreateMailServerEvent(p.peer.ID(), payload)

				if err != nil {
					log.Warn("error while parsing request complete code, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					return err
				}

				if event != nil {
					whisper.envelopeFeed.Send(*event)
				}

			}
		default:
			// New message types might be implemented in the future versions of Whisper.
			// For forward compatibility, just ignore.
		}

		packet.Discard()
	}
}

// add inserts a new envelope into the message pool to be distributed within the
// whisper network. It also inserts the envelope into the expiration pool at the
// appropriate time-stamp. In case of error, connection should be dropped.
// param isP2P indicates whether the message is peer-to-peer (should not be forwarded).
func (whisper *Whisper) add(envelope *Envelope, isP2P bool) (bool, error) {
	now := uint32(whisper.timeSource().Unix())
	sent := envelope.Expiry - envelope.TTL

	envelopeAddedCounter.Inc(1)
	if sent > now {
		if sent-DefaultSyncAllowance > now {
			envelopeErrFromFutureCounter.Inc(1)
			log.Warn("envelope created in the future", "hash", envelope.Hash())
			return false, TimeSyncError(errors.New("envelope from future"))
		}
		// recalculate PoW, adjusted for the time difference, plus one second for latency
		envelope.calculatePoW(sent - now + 1)
	}

	if envelope.Expiry < now {
		if envelope.Expiry+DefaultSyncAllowance*2 < now {
			envelopeErrVeryOldCounter.Inc(1)
			log.Warn("very old envelope", "hash", envelope.Hash())
			return false, TimeSyncError(errors.New("very old envelope"))
		}
		log.Debug("expired envelope dropped", "hash", envelope.Hash().Hex())
		envelopeErrExpiredCounter.Inc(1)
		return false, nil // drop envelope without error
	}

	if uint32(envelope.size()) > whisper.MaxMessageSize() {
		envelopeErrOversizedCounter.Inc(1)
		return false, fmt.Errorf("huge messages are not allowed [%x]", envelope.Hash())
	}

	if envelope.PoW() < whisper.MinPow() {
		// maybe the value was recently changed, and the peers did not adjust yet.
		// in this case the previous value is retrieved by MinPowTolerance()
		// for a short period of peer synchronization.
		if envelope.PoW() < whisper.MinPowTolerance() {
			envelopeErrLowPowCounter.Inc(1)
			return false, fmt.Errorf("envelope with low PoW received: PoW=%f, hash=[%v]", envelope.PoW(), envelope.Hash().Hex())
		}
	}

	if !BloomFilterMatch(whisper.BloomFilter(), envelope.Bloom()) {
		// maybe the value was recently changed, and the peers did not adjust yet.
		// in this case the previous value is retrieved by BloomFilterTolerance()
		// for a short period of peer synchronization.
		if !BloomFilterMatch(whisper.BloomFilterTolerance(), envelope.Bloom()) {
			envelopeErrNoBloomMatchCounter.Inc(1)
			return false, fmt.Errorf("envelope does not match bloom filter, hash=[%v], bloom: \n%x \n%x \n%x",
				envelope.Hash().Hex(), whisper.BloomFilter(), envelope.Bloom(), envelope.Topic)
		}
	}

	hash := envelope.Hash()

	whisper.poolMu.Lock()
	_, alreadyCached := whisper.envelopes[hash]
	if !alreadyCached {
		whisper.envelopes[hash] = envelope
		if whisper.expirations[envelope.Expiry] == nil {
			whisper.expirations[envelope.Expiry] = mapset.NewThreadUnsafeSet()
		}
		if !whisper.expirations[envelope.Expiry].Contains(hash) {
			whisper.expirations[envelope.Expiry].Add(hash)
		}
	}
	whisper.poolMu.Unlock()

	if alreadyCached {
		log.Trace("whisper envelope already cached", "hash", envelope.Hash().Hex())
	} else {
		log.Trace("cached whisper envelope", "hash", envelope.Hash().Hex())
		envelopeNewAddedCounter.Inc(1)
		envelopeSizeMeter.Mark(int64(envelope.size()))
		whisper.statsMu.Lock()
		whisper.stats.memoryUsed += envelope.size()
		whisper.statsMu.Unlock()
		whisper.postEvent(envelope, isP2P) // notify the local node about the new message
		if whisper.mailServer != nil {
			whisper.mailServer.Archive(envelope)
			whisper.envelopeFeed.Send(EnvelopeEvent{
				Hash:  envelope.Hash(),
				Event: EventMailServerEnvelopeArchived,
			})
		}
	}
	return true, nil
}

// postEvent queues the message for further processing.
func (whisper *Whisper) postEvent(envelope *Envelope, isP2P bool) {
	if isP2P {
		whisper.p2pMsgQueue <- envelope
	} else {
		whisper.checkOverflow()
		whisper.messageQueue <- envelope
	}
}

// checkOverflow checks if message queue overflow occurs and reports it if necessary.
func (whisper *Whisper) checkOverflow() {
	queueSize := len(whisper.messageQueue)

	if queueSize == messageQueueLimit {
		if !whisper.Overflow() {
			whisper.settings.Store(overflowIdx, true)
			log.Warn("message queue overflow")
		}
	} else if queueSize <= messageQueueLimit/2 {
		if whisper.Overflow() {
			whisper.settings.Store(overflowIdx, false)
			log.Warn("message queue overflow fixed (back to normal)")
		}
	}
}

// processQueue delivers the messages to the watchers during the lifetime of the whisper node.
func (whisper *Whisper) processQueue() {
	var e *Envelope
	for {
		select {
		case <-whisper.quit:
			return

		case e = <-whisper.messageQueue:
			whisper.filters.NotifyWatchers(e, false)
			whisper.envelopeFeed.Send(EnvelopeEvent{
				Hash:  e.Hash(),
				Event: EventEnvelopeAvailable,
			})

		case e = <-whisper.p2pMsgQueue:
			whisper.filters.NotifyWatchers(e, true)
			whisper.envelopeFeed.Send(EnvelopeEvent{
				Hash:  e.Hash(),
				Event: EventEnvelopeAvailable,
			})
		}
	}
}

// update loops until the lifetime of the whisper node, updating its internal
// state by expiring stale messages from the pool.
func (whisper *Whisper) update() {
	// Start a ticker to check for expirations
	expire := time.NewTicker(expirationCycle)

	// Repeat updates until termination is requested
	for {
		select {
		case <-expire.C:
			whisper.expire()

		case <-whisper.quit:
			return
		}
	}
}

// expire iterates over all the expiration timestamps, removing all stale
// messages from the pools.
func (whisper *Whisper) expire() {
	whisper.poolMu.Lock()
	defer whisper.poolMu.Unlock()

	whisper.statsMu.Lock()
	defer whisper.statsMu.Unlock()
	whisper.stats.reset()
	now := uint32(whisper.timeSource().Unix())
	for expiry, hashSet := range whisper.expirations {
		if expiry < now {
			// Dump all expired messages and remove timestamp
			hashSet.Each(func(v interface{}) bool {
				sz := whisper.envelopes[v.(common.Hash)].size()
				delete(whisper.envelopes, v.(common.Hash))
				envelopeClearedCounter.Inc(1)
				whisper.envelopeFeed.Send(EnvelopeEvent{
					Hash:  v.(common.Hash),
					Event: EventEnvelopeExpired,
				})
				whisper.stats.messagesCleared++
				whisper.stats.memoryCleared += sz
				whisper.stats.memoryUsed -= sz
				return false
			})
			whisper.expirations[expiry].Clear()
			delete(whisper.expirations, expiry)
		}
	}
}

// Stats returns the whisper node statistics.
func (whisper *Whisper) Stats() Statistics {
	whisper.statsMu.Lock()
	defer whisper.statsMu.Unlock()

	return whisper.stats
}

// Envelopes retrieves all the messages currently pooled by the node.
func (whisper *Whisper) Envelopes() []*Envelope {
	whisper.poolMu.RLock()
	defer whisper.poolMu.RUnlock()

	all := make([]*Envelope, 0, len(whisper.envelopes))
	for _, envelope := range whisper.envelopes {
		all = append(all, envelope)
	}
	return all
}

// isEnvelopeCached checks if envelope with specific hash has already been received and cached.
func (whisper *Whisper) isEnvelopeCached(hash common.Hash) bool {
	whisper.poolMu.Lock()
	defer whisper.poolMu.Unlock()

	_, exist := whisper.envelopes[hash]
	return exist
}

// reset resets the node's statistics after each expiry cycle.
func (s *Statistics) reset() {
	s.cycles++
	s.totalMessagesCleared += s.messagesCleared

	s.memoryCleared = 0
	s.messagesCleared = 0
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

// SelectedKeyPairID returns the id of currently selected key pair.
// It helps distinguish between different users w/o exposing the user identity itself.
func (whisper *Whisper) SelectedKeyPairID() string {
	whisper.keyMu.RLock()
	defer whisper.keyMu.RUnlock()

	for id := range whisper.privateKeys {
		return id
	}
	return ""
}
