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
	maxMsgSizeIdx                            = iota // Maximal message length allowed by the waku node
	overflowIdx                                     // Indicator of message queue overflow
	minPowIdx                                       // Minimal PoW required by the waku node
	minPowToleranceIdx                              // Minimal PoW tolerated by the waku node for a limited time
	bloomFilterIdx                                  // Bloom filter for topics of interest for this node
	bloomFilterToleranceIdx                         // Bloom filter tolerated by the waku node for a limited time
	lightClientModeIdx                              // Light client mode. (does not forward any messages)
	restrictConnectionBetweenLightClientsIdx        // Restrict connection between two light clients
)

// MailServerResponse is the response payload sent by the mailserver
type MailServerResponse struct {
	LastEnvelopeHash common.Hash
	Cursor           []byte
	Error            error
}

// Waku represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Waku struct {
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

	messageQueue chan *Envelope   // Message queue for normal waku messages
	p2pMsgQueue  chan interface{} // Message queue for peer-to-peer messages (not to be forwarded any further) and history delivery confirmations.
	quit         chan struct{}    // Channel used for graceful exit

	settings syncmap.Map // holds configuration settings that can be dynamically changed

	disableConfirmations bool // do not reply with confirmations

	syncAllowance int // maximum time in seconds allowed to process the waku-related messages

	statsMu sync.Mutex // guard stats
	stats   Statistics // Statistics of waku node

	mailServer MailServer // MailServer interface

	rateLimiter *PeerRateLimiter

	messageStoreFabric func() MessageStore

	envelopeFeed event.Feed

	timeSource func() time.Time // source of time for waku
}

// New creates a Waku client ready to communicate through the Ethereum P2P network.
func New(cfg *Config) *Waku {
	if cfg == nil {
		cfg = &DefaultConfig
	}

	waku := &Waku{
		privateKeys:          make(map[string]*ecdsa.PrivateKey),
		symKeys:              make(map[string][]byte),
		envelopes:            make(map[common.Hash]*Envelope),
		expirations:          make(map[uint32]mapset.Set),
		peers:                make(map[*Peer]struct{}),
		messageQueue:         make(chan *Envelope, messageQueueLimit),
		p2pMsgQueue:          make(chan interface{}, messageQueueLimit),
		quit:                 make(chan struct{}),
		syncAllowance:        DefaultSyncAllowance,
		timeSource:           time.Now,
		disableConfirmations: !cfg.EnableConfirmations,
	}

	waku.filters = NewFilters(waku)

	waku.settings.Store(minPowIdx, cfg.MinimumAcceptedPOW)
	waku.settings.Store(maxMsgSizeIdx, cfg.MaxMessageSize)
	waku.settings.Store(overflowIdx, false)
	waku.settings.Store(restrictConnectionBetweenLightClientsIdx, cfg.RestrictConnectionBetweenLightClients)

	// p2p waku sub protocol handler
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

// NewMessageStore returns object that implements MessageStore.
func (waku *Waku) NewMessageStore() MessageStore {
	if waku.messageStoreFabric != nil {
		return waku.messageStoreFabric()
	}
	return NewMemoryMessageStore()
}

// SetMessageStore allows to inject custom implementation of the message store.
func (waku *Waku) SetMessageStore(fabric func() MessageStore) {
	waku.messageStoreFabric = fabric
}

// SetTimeSource assigns a particular source of time to a waku object.
func (waku *Waku) SetTimeSource(timesource func() time.Time) {
	waku.timeSource = timesource
}

// SubscribeEnvelopeEvents subscribes to envelopes feed.
// In order to prevent blocking waku producers events must be amply buffered.
func (waku *Waku) SubscribeEnvelopeEvents(events chan<- EnvelopeEvent) event.Subscription {
	return waku.envelopeFeed.Subscribe(events)
}

// MinPow returns the PoW value required by this node.
func (waku *Waku) MinPow() float64 {
	val, exist := waku.settings.Load(minPowIdx)
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
func (waku *Waku) MinPowTolerance() float64 {
	val, exist := waku.settings.Load(minPowToleranceIdx)
	if !exist || val == nil {
		return DefaultMinimumPoW
	}
	return val.(float64)
}

// BloomFilter returns the aggregated bloom filter for all the topics of interest.
// The nodes are required to send only messages that match the advertised bloom filter.
// If a message does not match the bloom, it will tantamount to spam, and the peer will
// be disconnected.
func (waku *Waku) BloomFilter() []byte {
	val, exist := waku.settings.Load(bloomFilterIdx)
	if !exist || val == nil {
		return nil
	}
	return val.([]byte)
}

// BloomFilterTolerance returns the bloom filter which is tolerated for a limited
// time after new bloom was advertised to the peers. If sufficient time have elapsed
// or no change of bloom filter have ever occurred, the return value will be the same
// as return value of BloomFilter().
func (waku *Waku) BloomFilterTolerance() []byte {
	val, exist := waku.settings.Load(bloomFilterToleranceIdx)
	if !exist || val == nil {
		return nil
	}
	return val.([]byte)
}

// MaxMessageSize returns the maximum accepted message size.
func (waku *Waku) MaxMessageSize() uint32 {
	val, _ := waku.settings.Load(maxMsgSizeIdx)
	return val.(uint32)
}

// Overflow returns an indication if the message queue is full.
func (waku *Waku) Overflow() bool {
	val, _ := waku.settings.Load(overflowIdx)
	return val.(bool)
}

// APIs returns the RPC descriptors the Waku implementation offers
func (waku *Waku) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   NewPublicWakuAPI(waku),
			Public:    true,
		},
	}
}

// GetCurrentTime returns current time.
func (waku *Waku) GetCurrentTime() time.Time {
	return waku.timeSource()
}

// RegisterServer registers MailServer interface.
// MailServer will process all the incoming messages with p2pRequestCode.
func (waku *Waku) RegisterServer(server MailServer) {
	waku.mailServer = server
}

// Protocols returns the waku sub-protocols ran by this particular client.
func (waku *Waku) Protocols() []p2p.Protocol {
	return []p2p.Protocol{waku.protocol}
}

// Version returns the waku sub-protocols version number.
func (waku *Waku) Version() uint {
	return waku.protocol.Version
}

// SetMaxMessageSize sets the maximal message size allowed by this node
func (waku *Waku) SetMaxMessageSize(size uint32) error {
	if size > MaxMessageSize {
		return fmt.Errorf("message size too large [%d>%d]", size, MaxMessageSize)
	}
	waku.settings.Store(maxMsgSizeIdx, size)
	return nil
}

// SetBloomFilter sets the new bloom filter
func (waku *Waku) SetBloomFilter(bloom []byte) error {
	if len(bloom) != BloomFilterSize {
		return fmt.Errorf("invalid bloom filter size: %d", len(bloom))
	}

	b := make([]byte, BloomFilterSize)
	copy(b, bloom)

	waku.settings.Store(bloomFilterIdx, b)
	waku.notifyPeersAboutBloomFilterChange(b)

	go func() {
		// allow some time before all the peers have processed the notification
		time.Sleep(time.Duration(waku.syncAllowance) * time.Second)
		waku.settings.Store(bloomFilterToleranceIdx, b)
	}()

	return nil
}

// SetMinimumPoW sets the minimal PoW required by this node
func (waku *Waku) SetMinimumPoW(val float64) error {
	if val < 0.0 {
		return fmt.Errorf("invalid PoW: %f", val)
	}

	waku.settings.Store(minPowIdx, val)
	waku.notifyPeersAboutPowRequirementChange(val)

	go func() {
		// allow some time before all the peers have processed the notification
		time.Sleep(time.Duration(waku.syncAllowance) * time.Second)
		waku.settings.Store(minPowToleranceIdx, val)
	}()

	return nil
}

// SetMinimumPowTest sets the minimal PoW in test environment
func (waku *Waku) SetMinimumPowTest(val float64) {
	waku.settings.Store(minPowIdx, val)
	waku.notifyPeersAboutPowRequirementChange(val)
	waku.settings.Store(minPowToleranceIdx, val)
}

//SetLightClientMode makes node light client (does not forward any messages)
func (waku *Waku) SetLightClientMode(v bool) {
	waku.settings.Store(lightClientModeIdx, v)
}

func (waku *Waku) SetRateLimiter(r *PeerRateLimiter) {
	waku.rateLimiter = r
}

//LightClientMode indicates is this node is light client (does not forward any messages)
func (waku *Waku) LightClientMode() bool {
	val, exist := waku.settings.Load(lightClientModeIdx)
	if !exist || val == nil {
		return false
	}
	v, ok := val.(bool)
	return v && ok
}

//LightClientModeConnectionRestricted indicates that connection to light client in light client mode not allowed
func (waku *Waku) LightClientModeConnectionRestricted() bool {
	val, exist := waku.settings.Load(restrictConnectionBetweenLightClientsIdx)
	if !exist || val == nil {
		return false
	}
	v, ok := val.(bool)
	return v && ok
}

func (waku *Waku) notifyPeersAboutPowRequirementChange(pow float64) {
	arr := waku.getPeers()
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

func (waku *Waku) notifyPeersAboutBloomFilterChange(bloom []byte) {
	arr := waku.getPeers()
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

func (waku *Waku) getPeers() []*Peer {
	arr := make([]*Peer, len(waku.peers))
	i := 0
	waku.peerMu.Lock()
	for p := range waku.peers {
		arr[i] = p
		i++
	}
	waku.peerMu.Unlock()
	return arr
}

// getPeer retrieves peer by ID
func (waku *Waku) getPeer(peerID []byte) (*Peer, error) {
	waku.peerMu.Lock()
	defer waku.peerMu.Unlock()
	for p := range waku.peers {
		id := p.peer.ID()
		if bytes.Equal(peerID, id[:]) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("Could not find peer with ID: %x", peerID)
}

// AllowP2PMessagesFromPeer marks specific peer trusted,
// which will allow it to send historic (expired) messages.
func (waku *Waku) AllowP2PMessagesFromPeer(peerID []byte) error {
	p, err := waku.getPeer(peerID)
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
func (waku *Waku) RequestHistoricMessages(peerID []byte, envelope *Envelope) error {
	return waku.RequestHistoricMessagesWithTimeout(peerID, envelope, 0)
}

func (waku *Waku) RequestHistoricMessagesWithTimeout(peerID []byte, envelope *Envelope, timeout time.Duration) error {
	p, err := waku.getPeer(peerID)
	if err != nil {
		return err
	}
	waku.envelopeFeed.Send(EnvelopeEvent{
		Peer:  p.peer.ID(),
		Topic: envelope.Topic,
		Hash:  envelope.Hash(),
		Event: EventMailServerRequestSent,
	})
	p.trusted = true
	err = p2p.Send(p.ws, p2pRequestCode, envelope)
	if timeout != 0 {
		go waku.expireRequestHistoricMessages(p.peer.ID(), envelope.Hash(), timeout)
	}
	return err
}

func (waku *Waku) SendMessagesRequest(peerID []byte, request MessagesRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	p, err := waku.getPeer(peerID)
	if err != nil {
		return err
	}
	p.trusted = true
	if err := p2p.Send(p.ws, p2pRequestCode, request); err != nil {
		return err
	}
	waku.envelopeFeed.Send(EnvelopeEvent{
		Peer:  p.peer.ID(),
		Hash:  common.BytesToHash(request.ID),
		Event: EventMailServerRequestSent,
	})
	return nil
}

func (waku *Waku) expireRequestHistoricMessages(peer enode.ID, hash common.Hash, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-waku.quit:
		return
	case <-timer.C:
		waku.envelopeFeed.Send(EnvelopeEvent{
			Peer:  peer,
			Hash:  hash,
			Event: EventMailServerRequestExpired,
		})
	}
}

func (waku *Waku) SendHistoricMessageResponse(peer *Peer, payload []byte) error {
	size, r, err := rlp.EncodeToReader(payload)
	if err != nil {
		return err
	}

	return peer.ws.WriteMsg(p2p.Msg{Code: p2pRequestCompleteCode, Size: uint32(size), Payload: r})
}

// SendP2PMessage sends a peer-to-peer message to a specific peer.
func (waku *Waku) SendP2PMessage(peerID []byte, envelopes ...*Envelope) error {
	p, err := waku.getPeer(peerID)
	if err != nil {
		return err
	}
	return waku.SendP2PDirect(p, envelopes...)
}

// SendP2PDirect sends a peer-to-peer message to a specific peer.
// If only a single envelope is given, data is sent as a single object
// rather than a slice. This is important to keep this method backward compatible
// as it used to send only single envelopes.
func (waku *Waku) SendP2PDirect(peer *Peer, envelopes ...*Envelope) error {
	if len(envelopes) == 1 {
		return p2p.Send(peer.ws, p2pMessageCode, envelopes[0])
	}
	return p2p.Send(peer.ws, p2pMessageCode, envelopes)
}

// SendRawP2PDirect sends a peer-to-peer message to a specific peer.
// If only a single envelope is given, data is sent as a single object
// rather than a slice. This is important to keep this method backward compatible
// as it used to send only single envelopes.
func (waku *Waku) SendRawP2PDirect(peer *Peer, envelopes ...rlp.RawValue) error {
	if len(envelopes) == 1 {
		return p2p.Send(peer.ws, p2pMessageCode, envelopes[0])
	}
	return p2p.Send(peer.ws, p2pMessageCode, envelopes)
}

// NewKeyPair generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption. Returns ID of the new key pair.
func (waku *Waku) NewKeyPair() (string, error) {
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

	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()

	if waku.privateKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	waku.privateKeys[id] = key
	return id, nil
}

// DeleteKeyPair deletes the specified key if it exists.
func (waku *Waku) DeleteKeyPair(key string) bool {
	deterministicID, err := toDeterministicID(key, keyIDSize)
	if err != nil {
		return false
	}

	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()

	if waku.privateKeys[deterministicID] != nil {
		delete(waku.privateKeys, deterministicID)
		return true
	}
	return false
}

// AddKeyPair imports a asymmetric private key and returns it identifier.
func (waku *Waku) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	id, err := makeDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIDSize)
	if err != nil {
		return "", err
	}
	if waku.HasKeyPair(id) {
		return id, nil // no need to re-inject
	}

	waku.keyMu.Lock()
	waku.privateKeys[id] = key
	waku.keyMu.Unlock()
	log.Info("Waku identity added", "id", id, "pubkey", common.ToHex(crypto.FromECDSAPub(&key.PublicKey)))

	return id, nil
}

// SelectKeyPair adds cryptographic identity, and makes sure
// that it is the only private key known to the node.
func (waku *Waku) SelectKeyPair(key *ecdsa.PrivateKey) error {
	id, err := makeDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIDSize)
	if err != nil {
		return err
	}

	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()

	waku.privateKeys = make(map[string]*ecdsa.PrivateKey) // reset key store
	waku.privateKeys[id] = key

	log.Info("Waku identity selected", "id", id, "key", common.ToHex(crypto.FromECDSAPub(&key.PublicKey)))
	return nil
}

// DeleteKeyPairs removes all cryptographic identities known to the node
func (waku *Waku) DeleteKeyPairs() error {
	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()

	waku.privateKeys = make(map[string]*ecdsa.PrivateKey)

	return nil
}

// HasKeyPair checks if the waku node is configured with the private key
// of the specified public pair.
func (waku *Waku) HasKeyPair(id string) bool {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return false
	}

	waku.keyMu.RLock()
	defer waku.keyMu.RUnlock()
	return waku.privateKeys[deterministicID] != nil
}

// GetPrivateKey retrieves the private key of the specified identity.
func (waku *Waku) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return nil, err
	}

	waku.keyMu.RLock()
	defer waku.keyMu.RUnlock()
	key := waku.privateKeys[deterministicID]
	if key == nil {
		return nil, fmt.Errorf("invalid id")
	}
	return key, nil
}

// GenerateSymKey generates a random symmetric key and stores it under id,
// which is then returned. Will be used in the future for session key exchange.
func (waku *Waku) GenerateSymKey() (string, error) {
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

	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()

	if waku.symKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	waku.symKeys[id] = key
	return id, nil
}

// AddSymKey stores the key with a given id.
func (waku *Waku) AddSymKey(id string, key []byte) (string, error) {
	deterministicID, err := toDeterministicID(id, keyIDSize)
	if err != nil {
		return "", err
	}

	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()

	if waku.symKeys[deterministicID] != nil {
		return "", fmt.Errorf("key already exists: %v", id)
	}
	waku.symKeys[deterministicID] = key
	return deterministicID, nil
}

// AddSymKeyDirect stores the key, and returns its id.
func (waku *Waku) AddSymKeyDirect(key []byte) (string, error) {
	if len(key) != aesKeyLength {
		return "", fmt.Errorf("wrong key size: %d", len(key))
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}

	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()

	if waku.symKeys[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}
	waku.symKeys[id] = key
	return id, nil
}

// AddSymKeyFromPassword generates the key from password, stores it, and returns its id.
func (waku *Waku) AddSymKeyFromPassword(password string) (string, error) {
	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}
	if waku.HasSymKey(id) {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	// kdf should run no less than 0.1 seconds on an average computer,
	// because it's an once in a session experience
	derived := pbkdf2.Key([]byte(password), nil, 65356, aesKeyLength, sha256.New)
	if err != nil {
		return "", err
	}

	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()

	// double check is necessary, because deriveKeyMaterial() is very slow
	if waku.symKeys[id] != nil {
		return "", fmt.Errorf("critical error: failed to generate unique ID")
	}
	waku.symKeys[id] = derived
	return id, nil
}

// HasSymKey returns true if there is a key associated with the given id.
// Otherwise returns false.
func (waku *Waku) HasSymKey(id string) bool {
	waku.keyMu.RLock()
	defer waku.keyMu.RUnlock()
	return waku.symKeys[id] != nil
}

// DeleteSymKey deletes the key associated with the name string if it exists.
func (waku *Waku) DeleteSymKey(id string) bool {
	waku.keyMu.Lock()
	defer waku.keyMu.Unlock()
	if waku.symKeys[id] != nil {
		delete(waku.symKeys, id)
		return true
	}
	return false
}

// GetSymKey returns the symmetric key associated with the given id.
func (waku *Waku) GetSymKey(id string) ([]byte, error) {
	waku.keyMu.RLock()
	defer waku.keyMu.RUnlock()
	if waku.symKeys[id] != nil {
		return waku.symKeys[id], nil
	}
	return nil, fmt.Errorf("non-existent key ID")
}

// Subscribe installs a new message handler used for filtering, decrypting
// and subsequent storing of incoming messages.
func (waku *Waku) Subscribe(f *Filter) (string, error) {
	s, err := waku.filters.Install(f)
	if err == nil {
		waku.updateBloomFilter(f)
	}
	return s, err
}

// updateBloomFilter recalculates the new value of bloom filter,
// and informs the peers if necessary.
func (waku *Waku) updateBloomFilter(f *Filter) {
	aggregate := make([]byte, BloomFilterSize)
	for _, t := range f.Topics {
		top := BytesToTopic(t)
		b := TopicToBloom(top)
		aggregate = addBloom(aggregate, b)
	}

	if !BloomFilterMatch(waku.BloomFilter(), aggregate) {
		// existing bloom filter must be updated
		aggregate = addBloom(waku.BloomFilter(), aggregate)
		waku.SetBloomFilter(aggregate)
	}
}

// GetFilter returns the filter by id.
func (waku *Waku) GetFilter(id string) *Filter {
	return waku.filters.Get(id)
}

// Unsubscribe removes an installed message handler.
func (waku *Waku) Unsubscribe(id string) error {
	ok := waku.filters.Uninstall(id)
	if !ok {
		return fmt.Errorf("Unsubscribe: Invalid ID")
	}
	return nil
}

// Send injects a message into the waku send queue, to be distributed in the
// network in the coming cycles.
func (waku *Waku) Send(envelope *Envelope) error {
	ok, err := waku.add(envelope, false)
	if err == nil && !ok {
		return fmt.Errorf("failed to add envelope")
	}
	return err
}

// Start implements node.Service, starting the background data propagation thread
// of the Waku protocol.
func (waku *Waku) Start(*p2p.Server) error {
	log.Info("started waku v." + ProtocolVersionStr)
	go waku.update()

	numCPU := runtime.NumCPU()
	for i := 0; i < numCPU; i++ {
		go waku.processQueue()
	}
	go waku.processP2P()

	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Waku protocol.
func (waku *Waku) Stop() error {
	close(waku.quit)
	log.Info("waku stopped")
	return nil
}

// HandlePeer is called by the underlying P2P layer when the waku sub-protocol
// connection is negotiated.
func (waku *Waku) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Create the new peer and start tracking it
	wakuPeer := newPeer(waku, peer, rw)

	waku.peerMu.Lock()
	waku.peers[wakuPeer] = struct{}{}
	waku.peerMu.Unlock()

	defer func() {
		waku.peerMu.Lock()
		delete(waku.peers, wakuPeer)
		waku.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := wakuPeer.handshake(); err != nil {
		return err
	}
	wakuPeer.start()
	defer wakuPeer.stop()

	if waku.rateLimiter != nil {
		return waku.rateLimiter.decorate(wakuPeer, rw, waku.runMessageLoop)
	}
	return waku.runMessageLoop(wakuPeer, rw)
}

// TODO
//func (waku *Waku) sendConfirmation(peer enode.ID, rw p2p.MsgReadWriter, data []byte,
//	envelopeErrors []EnvelopeError) {
//	batchHash := crypto.Keccak256Hash(data)
//	if err := p2p.Send(rw, messageResponseCode, NewMessagesResponse(batchHash, envelopeErrors)); err != nil {
//		log.Warn("failed to deliver messages response", "hash", batchHash, "envelopes errors", envelopeErrors,
//			"peer", peer, "error", err)
//	}
//	if err := p2p.Send(rw, batchAcknowledgedCode, batchHash); err != nil {
//		log.Warn("failed to deliver confirmation", "hash", batchHash, "peer", peer, "error", err)
//	}
//}

// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (waku *Waku) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			log.Info("message loop", "peer", p.peer.ID(), "err", err)
			return err
		}
		if packet.Size > waku.MaxMessageSize() {
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
				envelopesRejectedCounter.WithLabelValues("failed_read").Inc()
				log.Warn("failed to read envelopes data", "peer", p.peer.ID(), "error", err)
				return errors.New("invalid enveloopes")
			}

			var envelopes []*Envelope
			if err := rlp.DecodeBytes(data, &envelopes); err != nil {
				envelopesRejectedCounter.WithLabelValues("invalid_data").Inc()
				log.Warn("failed to decode envelopes, peer will be disconnected", "peer", p.peer.ID(), "err", err)
				return errors.New("invalid envelopes")
			}
			trouble := false
			envelopeErrors := []EnvelopeError{}
			for _, env := range envelopes {
				cached, err := waku.add(env, waku.LightClientMode())
				if err != nil {
					_, isTimeSyncError := err.(TimeSyncError)
					if !isTimeSyncError {
						trouble = true
						log.Error("bad envelope received, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					}
					envelopeErrors = append(envelopeErrors, ErrorToEnvelopeError(env.Hash(), err))
				}

				waku.envelopeFeed.Send(EnvelopeEvent{
					Event: EventEnvelopeReceived,
					Topic: env.Topic,
					Hash:  env.Hash(),
					Peer:  p.peer.ID(),
				})
				envelopesValidatedCounter.Inc()
				if cached {
					p.mark(env)
				}
			}
			if !waku.disableConfirmations {
				// TODO
				//go waku.sendConfirmation(p.peer.ID(), rw, data, envelopeErrors)
			}

			if trouble {
				return errors.New("invalid envelope")
			}
		//case messageResponseCode:
		// TODO
		//	var multiResponse MultiVersionResponse
		//	if err := packet.Decode(&multiResponse); err != nil {
		//		envelopesRejectedCounter.WithLabelValues("failed_read").Inc()
		//		log.Error("failed to decode messages response", "peer", p.peer.ID(), "error", err)
		//		return errors.New("invalid response message")
		//	}
		//	if multiResponse.Version == 1 {
		//		response, err := multiResponse.DecodeResponse1()
		//		if err != nil {
		//			envelopesRejectedCounter.WithLabelValues("invalid_data").Inc()
		//			log.Error("failed to decode messages response into first version of response", "peer", p.peer.ID(), "error", err)
		//		}
		//		waku.envelopeFeed.Send(EnvelopeEvent{
		//			Batch: response.Hash,
		//			Event: EventBatchAcknowledged,
		//			Peer:  p.peer.ID(),
		//			Data:  response.Errors,
		//		})
		//	} else {
		//		log.Warn("unknown version of the messages response was received. response is ignored", "peer", p.peer.ID(), "version", multiResponse.Version)
		//	}
		//case batchAcknowledgedCode:
		// TODO
		//	var batchHash common.Hash
		//	if err := packet.Decode(&batchHash); err != nil {
		//		log.Error("failed to decode confirmation into common.Hash", "peer", p.peer.ID(), "error", err)
		//		return errors.New("invalid confirmation message")
		//	}
		//	waku.envelopeFeed.Send(EnvelopeEvent{
		//		Batch: batchHash,
		//		Event: EventBatchAcknowledged,
		//		Peer:  p.peer.ID(),
		//	})
		case powRequirementCode:
			s := rlp.NewStream(packet.Payload, uint64(packet.Size))
			i, err := s.Uint()
			if err != nil {
				envelopesRejectedCounter.WithLabelValues("invalid_pow_req").Inc()
				log.Warn("failed to decode powRequirementCode message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
				return errors.New("invalid powRequirementCode message")
			}
			f := math.Float64frombits(i)
			if math.IsInf(f, 0) || math.IsNaN(f) || f < 0.0 {
				envelopesRejectedCounter.WithLabelValues("invalid_pow_req").Inc()
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
				envelopesRejectedCounter.WithLabelValues("invalid_bloom").Inc()
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
						waku.postP2P(envelope)
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
					waku.postP2P(envelope)
					continue
				}

				if err != nil {
					log.Warn("failed to decode direct message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					return fmt.Errorf("invalid direct message: %v", err)
				}
			}
		case p2pRequestCode:
			// Must be processed if mail server is implemented. Otherwise ignore.
			if waku.mailServer != nil {
				// Read all data as we will try to decode it possibly twice.
				data, err := ioutil.ReadAll(packet.Payload)
				if err != nil {
					return fmt.Errorf("invalid direct messages: %v", err)
				}
				r := bytes.NewReader(data)
				packet.Payload = r

				var requestDeprecated Envelope
				errDepReq := packet.Decode(&requestDeprecated)
				if errDepReq == nil {
					waku.mailServer.DeliverMail(p, &requestDeprecated)
					continue
				} else {
					log.Info("failed to decode p2p request message (deprecated)", "peer", p.peer.ID(), "err", errDepReq)
				}

				// As we failed to decode the request, let's set the offset
				// to the beginning and try decode it again.
				if _, err := r.Seek(0, io.SeekStart); err != nil {
					return fmt.Errorf("invalid direct messages: %v", err)
				}

				var request MessagesRequest
				errReq := packet.Decode(&request)
				if errReq == nil {
					waku.mailServer.Deliver(p, request)
					continue
				} else {
					log.Info("failed to decode p2p request message", "peer", p.peer.ID(), "err", errReq)
				}

				return errors.New("invalid p2p request")
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
					waku.postP2P(*event)
				}

			}
		default:
			// New message types might be implemented in the future versions of Waku.
			// For forward compatibility, just ignore.
		}

		packet.Discard()
	}
}

// add inserts a new envelope into the message pool to be distributed within the
// waku network. It also inserts the envelope into the expiration pool at the
// appropriate time-stamp. In case of error, connection should be dropped.
// param isP2P indicates whether the message is peer-to-peer (should not be forwarded).
func (waku *Waku) add(envelope *Envelope, isP2P bool) (bool, error) {
	now := uint32(waku.timeSource().Unix())
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

	if uint32(envelope.size()) > waku.MaxMessageSize() {
		envelopesCacheFailedCounter.WithLabelValues("oversized").Inc()
		return false, fmt.Errorf("huge messages are not allowed [%x]", envelope.Hash())
	}

	if envelope.PoW() < waku.MinPow() {
		// maybe the value was recently changed, and the peers did not adjust yet.
		// in this case the previous value is retrieved by MinPowTolerance()
		// for a short period of peer synchronization.
		if envelope.PoW() < waku.MinPowTolerance() {
			envelopesCacheFailedCounter.WithLabelValues("low_pow").Inc()
			return false, fmt.Errorf("envelope with low PoW received: PoW=%f, hash=[%v]", envelope.PoW(), envelope.Hash().Hex())
		}
	}

	if !BloomFilterMatch(waku.BloomFilter(), envelope.Bloom()) {
		// maybe the value was recently changed, and the peers did not adjust yet.
		// in this case the previous value is retrieved by BloomFilterTolerance()
		// for a short period of peer synchronization.
		if !BloomFilterMatch(waku.BloomFilterTolerance(), envelope.Bloom()) {
			envelopesCacheFailedCounter.WithLabelValues("no_bloom_match").Inc()
			return false, fmt.Errorf("envelope does not match bloom filter, hash=[%v], bloom: \n%x \n%x \n%x",
				envelope.Hash().Hex(), waku.BloomFilter(), envelope.Bloom(), envelope.Topic)
		}
	}

	hash := envelope.Hash()

	waku.poolMu.Lock()
	_, alreadyCached := waku.envelopes[hash]
	if !alreadyCached {
		waku.envelopes[hash] = envelope
		if waku.expirations[envelope.Expiry] == nil {
			waku.expirations[envelope.Expiry] = mapset.NewThreadUnsafeSet()
		}
		if !waku.expirations[envelope.Expiry].Contains(hash) {
			waku.expirations[envelope.Expiry].Add(hash)
		}
	}
	waku.poolMu.Unlock()

	if alreadyCached {
		log.Trace("waku envelope already cached", "hash", envelope.Hash().Hex())
		envelopesCachedCounter.WithLabelValues("hit").Inc()
	} else {
		log.Trace("cached waku envelope", "hash", envelope.Hash().Hex())
		envelopesCachedCounter.WithLabelValues("miss").Inc()
		envelopesSizeMeter.Observe(float64(envelope.size()))
		waku.statsMu.Lock()
		waku.stats.memoryUsed += envelope.size()
		waku.statsMu.Unlock()
		waku.postEvent(envelope, isP2P) // notify the local node about the new message
		if waku.mailServer != nil {
			waku.mailServer.Archive(envelope)
			waku.envelopeFeed.Send(EnvelopeEvent{
				Topic: envelope.Topic,
				Hash:  envelope.Hash(),
				Event: EventMailServerEnvelopeArchived,
			})
		}
	}
	return true, nil
}

func (waku *Waku) postP2P(event interface{}) {
	waku.p2pMsgQueue <- event
}

// postEvent queues the message for further processing.
func (waku *Waku) postEvent(envelope *Envelope, isP2P bool) {
	if isP2P {
		waku.postP2P(envelope)
	} else {
		waku.checkOverflow()
		waku.messageQueue <- envelope
	}

}

// checkOverflow checks if message queue overflow occurs and reports it if necessary.
func (waku *Waku) checkOverflow() {
	queueSize := len(waku.messageQueue)

	if queueSize == messageQueueLimit {
		if !waku.Overflow() {
			waku.settings.Store(overflowIdx, true)
			log.Warn("message queue overflow")
		}
	} else if queueSize <= messageQueueLimit/2 {
		if waku.Overflow() {
			waku.settings.Store(overflowIdx, false)
			log.Warn("message queue overflow fixed (back to normal)")
		}
	}
}

// processQueue delivers the messages to the watchers during the lifetime of the waku node.
func (waku *Waku) processQueue() {
	for {
		select {
		case <-waku.quit:
			return
		case e := <-waku.messageQueue:
			waku.filters.NotifyWatchers(e, false)
			waku.envelopeFeed.Send(EnvelopeEvent{
				Topic: e.Topic,
				Hash:  e.Hash(),
				Event: EventEnvelopeAvailable,
			})
		}
	}
}

func (waku *Waku) processP2P() {
	for {
		select {
		case <-waku.quit:
			return
		case e := <-waku.p2pMsgQueue:
			switch event := e.(type) {
			case *Envelope:
				waku.filters.NotifyWatchers(event, true)
				waku.envelopeFeed.Send(EnvelopeEvent{
					Topic: event.Topic,
					Hash:  event.Hash(),
					Event: EventEnvelopeAvailable,
				})
			case EnvelopeEvent:
				waku.envelopeFeed.Send(event)
			}
		}
	}
}

// update loops until the lifetime of the waku node, updating its internal
// state by expiring stale messages from the pool.
func (waku *Waku) update() {
	// Start a ticker to check for expirations
	expire := time.NewTicker(expirationCycle)

	// Repeat updates until termination is requested
	for {
		select {
		case <-expire.C:
			waku.expire()

		case <-waku.quit:
			return
		}
	}
}

// expire iterates over all the expiration timestamps, removing all stale
// messages from the pools.
func (waku *Waku) expire() {
	waku.poolMu.Lock()
	defer waku.poolMu.Unlock()

	waku.statsMu.Lock()
	defer waku.statsMu.Unlock()
	waku.stats.reset()
	now := uint32(waku.timeSource().Unix())
	for expiry, hashSet := range waku.expirations {
		if expiry < now {
			// Dump all expired messages and remove timestamp
			hashSet.Each(func(v interface{}) bool {
				sz := waku.envelopes[v.(common.Hash)].size()
				delete(waku.envelopes, v.(common.Hash))
				envelopesCachedCounter.WithLabelValues("clear").Inc()
				waku.envelopeFeed.Send(EnvelopeEvent{
					Hash:  v.(common.Hash),
					Event: EventEnvelopeExpired,
				})
				waku.stats.messagesCleared++
				waku.stats.memoryCleared += sz
				waku.stats.memoryUsed -= sz
				return false
			})
			waku.expirations[expiry].Clear()
			delete(waku.expirations, expiry)
		}
	}
}

// Stats returns the waku node statistics.
func (waku *Waku) Stats() Statistics {
	waku.statsMu.Lock()
	defer waku.statsMu.Unlock()

	return waku.stats
}

// Envelopes retrieves all the messages currently pooled by the node.
func (waku *Waku) Envelopes() []*Envelope {
	waku.poolMu.RLock()
	defer waku.poolMu.RUnlock()

	all := make([]*Envelope, 0, len(waku.envelopes))
	for _, envelope := range waku.envelopes {
		all = append(all, envelope)
	}
	return all
}

// isEnvelopeCached checks if envelope with specific hash has already been received and cached.
func (waku *Waku) isEnvelopeCached(hash common.Hash) bool {
	waku.poolMu.Lock()
	defer waku.poolMu.Unlock()

	_, exist := waku.envelopes[hash]
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
func (waku *Waku) SelectedKeyPairID() string {
	waku.keyMu.RLock()
	defer waku.keyMu.RUnlock()

	for id := range waku.privateKeys {
		return id
	}
	return ""
}

// GetEnvelope retrieves an envelope from the message queue by its hash.
// It returns nil if the envelope can not be found.
func (w *Waku) GetEnvelope(hash common.Hash) *Envelope {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()
	return w.envelopes[hash]
}
