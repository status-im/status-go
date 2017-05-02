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

package whisperv5

import (
	"bytes"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"golang.org/x/crypto/pbkdf2"
	set "gopkg.in/fatih/set.v0"
)

type Statistics struct {
	messagesCleared      int
	memoryCleared        int
	memoryUsed           int
	cycles               int
	totalMessagesCleared int
}

// Whisper represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Whisper struct {
	protocol p2p.Protocol // Protocol description and parameters
	filters  *Filters     // Message filters installed with Subscribe function

	privateKeys map[string]*ecdsa.PrivateKey // Private key storage
	symKeys     map[string][]byte            // Symmetric key storage
	keyMu       sync.RWMutex                 // Mutex associated with key storages

	envelopes   map[common.Hash]*Envelope // Pool of envelopes currently tracked by this node
	expirations map[uint32]*set.SetNonTS  // Message expiration pool
	poolMu      sync.RWMutex              // Mutex to sync the message and expiration pools

	peers  map[*Peer]struct{} // Set of currently active peers
	peerMu sync.RWMutex       // Mutex to sync the active peer set

	messageQueue chan *Envelope // Message queue for normal whisper messages
	p2pMsgQueue  chan *Envelope // Message queue for peer-to-peer messages (not to be forwarded any further)
	quit         chan struct{}  // Channel used for graceful exit

	minPoW       float64 // Minimal PoW required by the whisper node
	maxMsgLength int     // Maximal message length allowed by the whisper node
	overflow     bool    // Indicator of message queue overflow

	stats Statistics // Statistics of whisper node

	mailServer         MailServer // MailServer interface
	notificationServer NotificationServer
}

// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func New() *Whisper {
	whisper := &Whisper{
		privateKeys:  make(map[string]*ecdsa.PrivateKey),
		symKeys:      make(map[string][]byte),
		envelopes:    make(map[common.Hash]*Envelope),
		expirations:  make(map[uint32]*set.SetNonTS),
		peers:        make(map[*Peer]struct{}),
		messageQueue: make(chan *Envelope, messageQueueLimit),
		p2pMsgQueue:  make(chan *Envelope, messageQueueLimit),
		quit:         make(chan struct{}),
		minPoW:       DefaultMinimumPoW,
		maxMsgLength: DefaultMaxMessageLength,
	}
	whisper.filters = NewFilters(whisper)

	// p2p whisper sub protocol handler
	whisper.protocol = p2p.Protocol{
		Name:    ProtocolName,
		Version: uint(ProtocolVersion),
		Length:  NumberOfMessageCodes,
		Run:     whisper.HandlePeer,
	}

	return whisper
}

// APIs returns the RPC descriptors the Whisper implementation offers
func (w *Whisper) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   NewPublicWhisperAPI(w),
			Public:    true,
		},
	}
}

// RegisterServer registers MailServer interface.
// MailServer will process all the incoming messages with p2pRequestCode.
func (w *Whisper) RegisterServer(server MailServer) {
	w.mailServer = server
}

// RegisterNotificationServer registers notification server with Whisper
func (w *Whisper) RegisterNotificationServer(server NotificationServer) {
	w.notificationServer = server
}

// Protocols returns the whisper sub-protocols ran by this particular client.
func (w *Whisper) Protocols() []p2p.Protocol {
	return []p2p.Protocol{w.protocol}
}

// Version returns the whisper sub-protocols version number.
func (w *Whisper) Version() uint {
	return w.protocol.Version
}

// SetMaxMessageLength sets the maximal message length allowed by this node
func (w *Whisper) SetMaxMessageLength(val int) error {
	if val <= 0 {
		return fmt.Errorf("invalid message length: %d", val)
	}
	w.maxMsgLength = val
	return nil
}

// SetMinimumPoW sets the minimal PoW required by this node
func (w *Whisper) SetMinimumPoW(val float64) error {
	if val <= 0.0 {
		return fmt.Errorf("invalid PoW: %f", val)
	}
	w.minPoW = val
	return nil
}

// getPeer retrieves peer by ID
func (w *Whisper) getPeer(peerID []byte) (*Peer, error) {
	w.peerMu.Lock()
	defer w.peerMu.Unlock()
	for p := range w.peers {
		id := p.peer.ID()
		if bytes.Equal(peerID, id[:]) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("Could not find peer with ID: %x", peerID)
}

// AllowP2PMessagesFromPeer marks specific peer trusted,
// which will allow it to send historic (expired) messages.
func (w *Whisper) AllowP2PMessagesFromPeer(peerID []byte) error {
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
// The whisper protocol is agnostic of the format and contents of envelope.
func (w *Whisper) RequestHistoricMessages(peerID []byte, envelope *Envelope) error {
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	p.trusted = true
	return p2p.Send(p.ws, p2pRequestCode, envelope)
}

// SendP2PMessage sends a peer-to-peer message to a specific peer.
func (w *Whisper) SendP2PMessage(peerID []byte, envelope *Envelope) error {
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	return w.SendP2PDirect(p, envelope)
}

// SendP2PDirect sends a peer-to-peer message to a specific peer.
func (w *Whisper) SendP2PDirect(peer *Peer, envelope *Envelope) error {
	return p2p.Send(peer.ws, p2pCode, envelope)
}

// NewKeyPair generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption. Returns ID of the new key pair.
func (w *Whisper) NewKeyPair() (string, error) {
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

	id, err := toDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIdSize)
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

// AddIdentity adds cryptographic identity into the known
// identities list (for message decryption).
func (w *Whisper) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	id, err := makeDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIdSize)
	if err != nil {
		return "", err
	}
	if w.HasKeyPair(id) {
		return id, nil // no need to re-inject
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	w.privateKeys[id] = key
	log.Info("Whisper identity added", "id", id, "pubkey", common.ToHex(crypto.FromECDSAPub(&key.PublicKey)))

	return id, nil
}

// SelectKeyPair adds cryptographic identity, and makes sure
// that it is the only private key known to the node.
func (w *Whisper) SelectKeyPair(key *ecdsa.PrivateKey) error {
	id, err := makeDeterministicID(common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), keyIdSize)
	if err != nil {
		return err
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	w.privateKeys = make(map[string]*ecdsa.PrivateKey) // reset key store
	w.privateKeys[id] = key

	log.Info("Whisper identity selected", "id", id, "key", common.ToHex(crypto.FromECDSAPub(&key.PublicKey)))
	return nil
}

// DeleteKeyPairs removes all cryptographic identities known to the node
func (w *Whisper) DeleteKeyPairs() error {
	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	w.privateKeys = make(map[string]*ecdsa.PrivateKey)

	return nil
}

// DeleteKeyPair deletes the specified key if it exists.
func (w *Whisper) DeleteKeyPair(id string) bool {
	deterministicID, err := toDeterministicID(id, keyIdSize)
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

// HasKeyPair checks if the the whisper node is configured with the private key
// of the specified public pair.
func (w *Whisper) HasKeyPair(id string) bool {
	deterministicID, err := toDeterministicID(id, keyIdSize)
	if err != nil {
		return false
	}

	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	return w.privateKeys[deterministicID] != nil
}

// GetPrivateKey retrieves the private key of the specified identity.
func (w *Whisper) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	deterministicID, err := toDeterministicID(id, keyIdSize)
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
func (w *Whisper) GenerateSymKey() (string, error) {
	const size = aesKeyLength * 2
	buf := make([]byte, size)
	_, err := crand.Read(buf)
	if err != nil {
		return "", err
	} else if !validateSymmetricKey(buf) {
		return "", fmt.Errorf("error in GenerateSymKey: crypto/rand failed to generate random data")
	}

	key := buf[:aesKeyLength]
	salt := buf[aesKeyLength:]
	derived, err := DeriveOneTimeKey(key, salt, EnvelopeVersion)
	if err != nil {
		return "", err
	} else if !validateSymmetricKey(derived) {
		return "", fmt.Errorf("failed to derive valid key")
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
	w.symKeys[id] = derived
	return id, nil
}

// AddSymKey stores the key with a given id.
func (w *Whisper) AddSymKey(id string, key []byte) (string, error) {
	deterministicID, err := toDeterministicID(id, keyIdSize)
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
func (w *Whisper) AddSymKeyDirect(key []byte) (string, error) {
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
func (w *Whisper) AddSymKeyFromPassword(password string) (string, error) {
	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}
	if w.HasSymKey(id) {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	derived, err := deriveKeyMaterial([]byte(password), EnvelopeVersion)
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
func (w *Whisper) HasSymKey(id string) bool {
	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	return w.symKeys[id] != nil
}

// DeleteSymKey deletes the key associated with the name string if it exists.
func (w *Whisper) DeleteSymKey(id string) bool {
	w.keyMu.Lock()
	defer w.keyMu.Unlock()
	if w.symKeys[id] != nil {
		delete(w.symKeys, id)
		return true
	}
	return false
}

// GetSymKey returns the symmetric key associated with the given id.
func (w *Whisper) GetSymKey(id string) ([]byte, error) {
	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	if w.symKeys[id] != nil {
		return w.symKeys[id], nil
	}
	return nil, fmt.Errorf("non-existent key ID")
}

// Subscribe installs a new message handler used for filtering, decrypting
// and subsequent storing of incoming messages.
func (w *Whisper) Subscribe(f *Filter) (string, error) {
	return w.filters.Install(f)
}

// GetFilter returns the filter by id.
func (w *Whisper) GetFilter(id string) *Filter {
	return w.filters.Get(id)
}

// Unsubscribe removes an installed message handler.
func (w *Whisper) Unsubscribe(id string) error {
	ok := w.filters.Uninstall(id)
	if !ok {
		return fmt.Errorf("Unsubscribe: Invalid ID")
	}
	return nil
}

// Send injects a message into the whisper send queue, to be distributed in the
// network in the coming cycles.
func (w *Whisper) Send(envelope *Envelope) error {
	ok, err := w.add(envelope)
	if !ok {
		return fmt.Errorf("failed to add envelope")
	}
	return err
}

// Start implements node.Service, starting the background data propagation thread
// of the Whisper protocol.
func (w *Whisper) Start(stack *p2p.Server) error {
	log.Info("started whisper v." + ProtocolVersionStr)
	go w.update()

	numCPU := runtime.NumCPU()
	for i := 0; i < numCPU; i++ {
		go w.processQueue()
	}

	if w.notificationServer != nil {
		if err := w.notificationServer.Start(stack); err != nil {
			return err
		}
	}

	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Whisper protocol.
func (w *Whisper) Stop() error {
	close(w.quit)

	if w.notificationServer != nil {
		if err := w.notificationServer.Stop(); err != nil {
			return err
		}
	}

	log.Info("whisper stopped")
	return nil
}

// HandlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (wh *Whisper) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Create the new peer and start tracking it
	whisperPeer := newPeer(wh, peer, rw)

	wh.peerMu.Lock()
	wh.peers[whisperPeer] = struct{}{}
	wh.peerMu.Unlock()

	defer func() {
		wh.peerMu.Lock()
		delete(wh.peers, whisperPeer)
		wh.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := whisperPeer.handshake(); err != nil {
		return err
	}
	whisperPeer.start()
	defer whisperPeer.stop()

	return wh.runMessageLoop(whisperPeer, rw)
}

// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (wh *Whisper) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			log.Warn("message loop", "peer", p.peer.ID(), "err", err)
			return err
		}
		if packet.Size > uint32(wh.maxMsgLength) {
			log.Warn("oversized message received", "peer", p.peer.ID())
			return errors.New("oversized message received")
		}

		switch packet.Code {
		case statusCode:
			// this should not happen, but no need to panic; just ignore this message.
			log.Warn("unxepected status message received", "peer", p.peer.ID())
		case messagesCode:
			// decode the contained envelopes
			var envelopes []*Envelope
			if err := packet.Decode(&envelopes); err != nil {
				log.Warn("failed to decode envelope, peer will be disconnected", "peer", p.peer.ID(), "err", err)
				return errors.New("invalid envelope")
			}
			// inject all envelopes into the internal pool
			for _, envelope := range envelopes {
				cached, err := wh.add(envelope)
				if err != nil {
					log.Warn("bad envelope received, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					return errors.New("invalid envelope")
				}
				if cached {
					p.mark(envelope)
				}
			}
		case p2pCode:
			// peer-to-peer message, sent directly to peer bypassing PoW checks, etc.
			// this message is not supposed to be forwarded to other peers, and
			// therefore might not satisfy the PoW, expiry and other requirements.
			// these messages are only accepted from the trusted peer.
			if p.trusted {
				var envelope Envelope
				if err := packet.Decode(&envelope); err != nil {
					log.Warn("failed to decode direct message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					return errors.New("invalid direct message")
				}
				wh.postEvent(&envelope, true)
			}
		case p2pRequestCode:
			// Must be processed if mail server is implemented. Otherwise ignore.
			if wh.mailServer != nil {
				var request Envelope
				if err := packet.Decode(&request); err != nil {
					log.Warn("failed to decode p2p request message, peer will be disconnected", "peer", p.peer.ID(), "err", err)
					return errors.New("invalid p2p request")
				}
				wh.mailServer.DeliverMail(p, &request)
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
func (wh *Whisper) add(envelope *Envelope) (bool, error) {
	now := uint32(time.Now().Unix())
	sent := envelope.Expiry - envelope.TTL

	if sent > now {
		if sent-SynchAllowance > now {
			return false, fmt.Errorf("envelope created in the future [%x]", envelope.Hash())
		} else {
			// recalculate PoW, adjusted for the time difference, plus one second for latency
			envelope.calculatePoW(sent - now + 1)
		}
	}

	if envelope.Expiry < now {
		if envelope.Expiry+SynchAllowance*2 < now {
			return false, fmt.Errorf("very old message")
		} else {
			log.Debug("expired envelope dropped", "hash", envelope.Hash().Hex())
			return false, nil // drop envelope without error
		}
	}

	if envelope.size() > wh.maxMsgLength {
		return false, fmt.Errorf("huge messages are not allowed [%x]", envelope.Hash())
	}

	if len(envelope.Version) > 4 {
		return false, fmt.Errorf("oversized version [%x]", envelope.Hash())
	}

	if len(envelope.AESNonce) > AESNonceMaxLength {
		// the standard AES GSM nonce size is 12,
		// but const gcmStandardNonceSize cannot be accessed directly
		return false, fmt.Errorf("oversized AESNonce [%x]", envelope.Hash())
	}

	if len(envelope.Salt) > saltLength {
		return false, fmt.Errorf("oversized salt [%x]", envelope.Hash())
	}

	if envelope.PoW() < wh.minPoW {
		log.Debug("envelope with low PoW dropped", "PoW", envelope.PoW(), "hash", envelope.Hash().Hex())
		return false, nil // drop envelope without error
	}

	hash := envelope.Hash()

	wh.poolMu.Lock()
	_, alreadyCached := wh.envelopes[hash]
	if !alreadyCached {
		wh.envelopes[hash] = envelope
		if wh.expirations[envelope.Expiry] == nil {
			wh.expirations[envelope.Expiry] = set.NewNonTS()
		}
		if !wh.expirations[envelope.Expiry].Has(hash) {
			wh.expirations[envelope.Expiry].Add(hash)
		}
	}
	wh.poolMu.Unlock()

	if alreadyCached {
		log.Trace("whisper envelope already cached", "hash", envelope.Hash().Hex())
	} else {
		log.Trace("cached whisper envelope", "hash", envelope.Hash().Hex())
		wh.stats.memoryUsed += envelope.size()
		wh.postEvent(envelope, false) // notify the local node about the new message
		if wh.mailServer != nil {
			wh.mailServer.Archive(envelope)
		}
	}
	return true, nil
}

// postEvent queues the message for further processing.
func (w *Whisper) postEvent(envelope *Envelope, isP2P bool) {
	// if the version of incoming message is higher than
	// currently supported version, we can not decrypt it,
	// and therefore just ignore this message
	if envelope.Ver() <= EnvelopeVersion {
		if isP2P {
			w.p2pMsgQueue <- envelope
		} else {
			w.checkOverflow()
			w.messageQueue <- envelope
		}
	}
}

// checkOverflow checks if message queue overflow occurs and reports it if necessary.
func (w *Whisper) checkOverflow() {
	queueSize := len(w.messageQueue)

	if queueSize == messageQueueLimit {
		if !w.overflow {
			w.overflow = true
			log.Warn("message queue overflow")
		}
	} else if queueSize <= messageQueueLimit/2 {
		if w.overflow {
			w.overflow = false
			log.Warn("message queue overflow fixed (back to normal)")
		}
	}
}

// processQueue delivers the messages to the watchers during the lifetime of the whisper node.
func (w *Whisper) processQueue() {
	var e *Envelope
	for {
		select {
		case <-w.quit:
			return

		case e = <-w.messageQueue:
			w.filters.NotifyWatchers(e, false)

		case e = <-w.p2pMsgQueue:
			w.filters.NotifyWatchers(e, true)
		}
	}
}

// update loops until the lifetime of the whisper node, updating its internal
// state by expiring stale messages from the pool.
func (w *Whisper) update() {
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
func (w *Whisper) expire() {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	w.stats.reset()
	now := uint32(time.Now().Unix())
	for expiry, hashSet := range w.expirations {
		if expiry < now {
			// Dump all expired messages and remove timestamp
			hashSet.Each(func(v interface{}) bool {
				sz := w.envelopes[v.(common.Hash)].size()
				delete(w.envelopes, v.(common.Hash))
				w.stats.messagesCleared++
				w.stats.memoryCleared += sz
				w.stats.memoryUsed -= sz
				return true
			})
			w.expirations[expiry].Clear()
			delete(w.expirations, expiry)
		}
	}
}

// Stats returns the whisper node statistics.
func (w *Whisper) Stats() string {
	result := fmt.Sprintf("Memory usage: %d bytes. Average messages cleared per expiry cycle: %d. Total messages cleared: %d.",
		w.stats.memoryUsed, w.stats.totalMessagesCleared/w.stats.cycles, w.stats.totalMessagesCleared)
	if w.stats.messagesCleared > 0 {
		result += fmt.Sprintf(" Latest expiry cycle cleared %d messages (%d bytes).",
			w.stats.messagesCleared, w.stats.memoryCleared)
	}
	if w.overflow {
		result += " Message queue state: overflow."
	}
	return result
}

// Envelopes retrieves all the messages currently pooled by the node.
func (w *Whisper) Envelopes() []*Envelope {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	all := make([]*Envelope, 0, len(w.envelopes))
	for _, envelope := range w.envelopes {
		all = append(all, envelope)
	}
	return all
}

// Messages iterates through all currently floating envelopes
// and retrieves all the messages, that this filter could decrypt.
func (w *Whisper) Messages(id string) []*ReceivedMessage {
	result := make([]*ReceivedMessage, 0)
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	if filter := w.filters.Get(id); filter != nil {
		for _, env := range w.envelopes {
			msg := filter.processEnvelope(env)
			if msg != nil {
				result = append(result, msg)
			}
		}
	}
	return result
}

// isEnvelopeCached checks if envelope with specific hash has already been received and cached.
func (w *Whisper) isEnvelopeCached(hash common.Hash) bool {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	_, exist := w.envelopes[hash]
	return exist
}

// reset resets the node's statistics after each expiry cycle.
func (s *Statistics) reset() {
	s.cycles++
	s.totalMessagesCleared += s.messagesCleared

	s.memoryCleared = 0
	s.messagesCleared = 0
}

// ValidateKeyID checks the format of key id.
func ValidateKeyID(id string) error {
	const target = keyIdSize * 2
	if len(id) != target {
		return fmt.Errorf("wrong size of key ID (expected %d bytes, got %d): %s", target, len(id), id)
	}
	return nil
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

// validateSymmetricKey returns false if the key contains all zeros
func validateSymmetricKey(k []byte) bool {
	return len(k) > 0 && !containsOnlyZeros(k)
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

// deriveKeyMaterial derives symmetric key material from the key or password.
// pbkdf2 is used for security, in case people use password instead of randomly generated keys.
func deriveKeyMaterial(key []byte, version uint64) (derivedKey []byte, err error) {
	if version == 0 {
		// kdf should run no less than 0.1 seconds on average compute,
		// because it's a once in a session experience
		derivedKey := pbkdf2.Key(key, nil, 65356, aesKeyLength, sha256.New)
		return derivedKey, nil
	} else {
		return nil, unknownVersionError(version)
	}
}

// GenerateRandomID generates a random string, which is then returned to be used as a key id
func GenerateRandomID() (id string, err error) {
	buf := make([]byte, keyIdSize)
	_, err = crand.Read(buf)
	if err != nil {
		return "", err
	}
	if !validateSymmetricKey(buf) {
		return "", fmt.Errorf("error in generateRandomID: crypto/rand failed to generate random data")
	}
	id = common.Bytes2Hex(buf)
	return id, err
}

// makeDeterministicID generates a deterministic ID, based on a given input
func makeDeterministicID(input string, keyLen int) (id string, err error) {
	buf := pbkdf2.Key([]byte(input), nil, 4096, keyLen, sha256.New)
	if !validateSymmetricKey(buf) {
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
