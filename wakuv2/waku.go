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

package wakuv2

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	"github.com/multiformats/go-multiaddr"

	"go.uber.org/zap"

	mapset "github.com/deckarep/golang-set"
	"golang.org/x/crypto/pbkdf2"

	dssql "github.com/ipfs/go-ds-sql"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/metrics"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	libp2pproto "github.com/libp2p/go-libp2p-core/protocol"

	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/status-im/go-waku/waku/v2/dnsdisc"
	"github.com/status-im/go-waku/waku/v2/protocol"
	wakuprotocol "github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/wakuv2/common"
	"github.com/status-im/status-go/wakuv2/persistence"

	"github.com/libp2p/go-libp2p-core/discovery"
	node "github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
)

const messageQueueLimit = 1024
const requestTimeout = 5 * time.Second

const PeerStoreTable = "peerstore"

type settings struct {
	LightClient         bool   // Indicates if the node is a light client
	MinPeersForRelay    int    // Indicates the minimum number of peers required for using Relay Protocol instead of Lightpush
	MaxMsgSize          uint32 // Maximal message length allowed by the waku node
	EnableConfirmations bool   // Enable sending message confirmations
	PersistPeers        bool   // Indicates if the node will persist peers
}

// Waku represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Waku struct {
	node  *node.WakuNode // reference to a libp2p waku node
	appDB *sql.DB

	dnsAddressCache     map[string][]multiaddr.Multiaddr // Map to store the multiaddresses returned by dns discovery
	dnsAddressCacheLock *sync.RWMutex                    // lock to handle access to the map

	filters          *common.Filters         // Message filters installed with Subscribe function
	filterMsgChannel chan *protocol.Envelope // Channel for wakuv2 filter messages

	privateKeys map[string]*ecdsa.PrivateKey // Private key storage
	symKeys     map[string][]byte            // Symmetric key storage
	keyMu       sync.RWMutex                 // Mutex associated with key stores

	envelopes   map[gethcommon.Hash]*common.ReceivedMessage // Pool of envelopes currently tracked by this node
	expirations map[uint32]mapset.Set                       // Message expiration pool
	poolMu      sync.RWMutex                                // Mutex to sync the message and expiration pools

	bandwidthCounter *metrics.BandwidthCounter

	sendQueue chan *pb.WakuMessage
	msgQueue  chan *common.ReceivedMessage // Message queue for waku messages that havent been decoded
	quit      chan struct{}                // Channel used for graceful exit

	settings   settings     // Holds configuration settings that can be dynamically changed
	settingsMu sync.RWMutex // Mutex to sync the settings access

	envelopeFeed event.Feed

	storeMsgIDs   map[gethcommon.Hash]bool // Map of the currently processing ids
	storeMsgIDsMu sync.RWMutex

	connStatusSubscriptions map[string]*types.ConnStatusSubscription
	connStatusMu            sync.Mutex

	timeSource func() time.Time // source of time for waku

	logger *zap.Logger
}

// New creates a WakuV2 client ready to communicate through the LibP2P network.
func New(nodeKey string, cfg *Config, logger *zap.Logger, appDB *sql.DB) (*Waku, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	cfg = setDefaults(cfg)

	logger.Debug("starting wakuv2 with config", zap.Any("config", cfg))

	waku := &Waku{
		appDB:                   appDB,
		privateKeys:             make(map[string]*ecdsa.PrivateKey),
		symKeys:                 make(map[string][]byte),
		envelopes:               make(map[gethcommon.Hash]*common.ReceivedMessage),
		expirations:             make(map[uint32]mapset.Set),
		msgQueue:                make(chan *common.ReceivedMessage, messageQueueLimit),
		sendQueue:               make(chan *pb.WakuMessage, 1000),
		connStatusSubscriptions: make(map[string]*types.ConnStatusSubscription),
		quit:                    make(chan struct{}),
		dnsAddressCache:         make(map[string][]multiaddr.Multiaddr),
		dnsAddressCacheLock:     &sync.RWMutex{},
		storeMsgIDs:             make(map[gethcommon.Hash]bool),
		storeMsgIDsMu:           sync.RWMutex{},
		timeSource:              time.Now,
		logger:                  logger,
	}

	waku.settings = settings{
		MaxMsgSize:       cfg.MaxMessageSize,
		LightClient:      cfg.LightClient,
		MinPeersForRelay: cfg.MinPeersForRelay,
		PersistPeers:     cfg.PersistPeers,
	}

	waku.filters = common.NewFilters()
	waku.bandwidthCounter = metrics.NewBandwidthCounter()
	waku.filterMsgChannel = make(chan *protocol.Envelope, 1024)

	var privateKey *ecdsa.PrivateKey
	var err error
	if nodeKey != "" {
		privateKey, err = crypto.HexToECDSA(nodeKey)
	} else {
		// If no nodekey is provided, create an ephemeral key
		privateKey, err = crypto.GenerateKey()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to setup the go-waku private key: %v", err)
	}

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprint(cfg.Host, ":", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to setup the network interface: %v", err)
	}

	ctx := context.Background()

	connStatusChan := make(chan node.ConnStatus, 100)

	if cfg.KeepAliveInterval == 0 {
		cfg.KeepAliveInterval = DefaultConfig.KeepAliveInterval
	}

	libp2pOpts := node.DefaultLibP2POptions
	libp2pOpts = append(libp2pOpts, libp2p.BandwidthReporter(waku.bandwidthCounter))

	if cfg.PersistPeers {
		if appDB == nil {
			return nil, fmt.Errorf("a db connection must be provided in order to persist the peers")
		}

		// Create persistent peerstore
		queries, err := persistence.NewQueries(PeerStoreTable, appDB)
		if err != nil {
			return nil, fmt.Errorf("failed to setup peerstore table: %v", err)
		}

		datastore := dssql.NewDatastore(appDB, queries)
		opts := pstoreds.DefaultOpts()
		peerStore, err := pstoreds.NewPeerstore(ctx, datastore, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to setup peerstore: %v", err)
		}
		libp2pOpts = append(libp2pOpts, libp2p.Peerstore(peerStore))
	}

	opts := []node.WakuNodeOption{
		node.WithLibP2POptions(libp2pOpts...),
		node.WithPrivateKey(privateKey),
		node.WithHostAddress(hostAddr),
		node.WithConnectionStatusChannel(connStatusChan),
		node.WithKeepAlive(time.Duration(cfg.KeepAliveInterval) * time.Second),
	}

	if cfg.Rendezvous {
		opts = append(opts, node.WithRendezvous(pubsub.WithDiscoveryOpts(discovery.Limit(cfg.DiscoveryLimit))))
	}

	if cfg.EnableDiscV5 {
		var bootnodes []*enode.Node
		for _, addr := range cfg.DiscV5BootstrapNodes {
			bootnode, err := enode.Parse(enode.ValidSchemes, addr)
			if err != nil {
				return nil, err
			}
			bootnodes = append(bootnodes, bootnode)
		}
		opts = append(opts, node.WithDiscoveryV5(cfg.UDPPort, bootnodes, cfg.AutoUpdate, pubsub.WithDiscoveryOpts(discovery.Limit(cfg.DiscoveryLimit))))
	}

	if cfg.LightClient {
		opts = append(opts, node.WithWakuFilter(false))
	} else {
		relayOpts := []pubsub.Option{
			pubsub.WithMaxMessageSize(int(waku.settings.MaxMsgSize)),
			pubsub.WithPeerExchange(cfg.PeerExchange),
		}

		if cfg.PeerExchange {
			relayOpts = append(relayOpts, pubsub.WithPeerExchange(true))
		}

		opts = append(opts, node.WithWakuRelay(relayOpts...))
	}

	if waku.node, err = node.New(ctx, opts...); err != nil {
		return nil, fmt.Errorf("failed to create a go-waku node: %v", err)
	}

	waku.addWakuV2Peers(cfg)

	if err = waku.node.Start(); err != nil {
		return nil, fmt.Errorf("failed to start go-waku node: %v", err)
	}

	go func() {
		for {
			select {
			case <-waku.quit:
				return
			case c := <-connStatusChan:
				waku.connStatusMu.Lock()
				latestConnStatus := formatConnStatus(c)
				for k, subs := range waku.connStatusSubscriptions {
					if subs.Active() {
						subs.C <- latestConnStatus
					} else {
						delete(waku.connStatusSubscriptions, k)
					}
				}
				waku.connStatusMu.Unlock()
				signal.SendPeerStats(latestConnStatus)
			}
		}
	}()

	go waku.runFilterMsgLoop()
	go waku.runRelayMsgLoop()

	log.Info("setup the go-waku node successfully")

	return waku, nil
}

func (w *Waku) SubscribeToConnStatusChanges() *types.ConnStatusSubscription {
	w.connStatusMu.Lock()
	defer w.connStatusMu.Unlock()
	subscription := types.NewConnStatusSubscription()
	w.connStatusSubscriptions[subscription.ID] = subscription
	return subscription
}

type fnApplyToEachPeer func(ma multiaddr.Multiaddr, protocol libp2pproto.ID)

func (w *Waku) addPeers(addresses []string, protocol libp2pproto.ID, apply fnApplyToEachPeer) {
	for _, addrString := range addresses {
		if addrString == "" {
			continue
		}

		if strings.HasPrefix(addrString, "enrtree://") {
			// Use DNS Discovery
			go w.dnsDiscover(addrString, protocol, apply)
		} else {
			// It's a normal multiaddress
			w.addPeerFromString(addrString, protocol, apply)
		}
	}
}

func (w *Waku) dnsDiscover(enrtreeAddress string, protocol libp2pproto.ID, apply fnApplyToEachPeer) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	w.dnsAddressCacheLock.RLock()
	multiaddresses, ok := w.dnsAddressCache[enrtreeAddress]
	w.dnsAddressCacheLock.RUnlock()

	if !ok {
		w.dnsAddressCacheLock.Lock()
		var err error
		multiaddresses, err = dnsdisc.RetrieveNodes(ctx, enrtreeAddress)
		w.dnsAddressCache[enrtreeAddress] = multiaddresses
		w.dnsAddressCacheLock.Unlock()
		if err != nil {
			log.Warn("dns discovery error ", err)
			return
		}
	}

	for _, m := range multiaddresses {
		apply(m, protocol)
	}
}

func (w *Waku) addPeerFromString(addrString string, protocol libp2pproto.ID, apply fnApplyToEachPeer) {
	addr, err := multiaddr.NewMultiaddr(addrString)
	if err != nil {
		log.Warn("invalid peer multiaddress", addrString, err)
		return
	}

	apply(addr, protocol)
}

func (w *Waku) addWakuV2Peers(cfg *Config) {
	if !cfg.LightClient {
		addRelayPeer := func(m multiaddr.Multiaddr, protocol libp2pproto.ID) {
			go func(node multiaddr.Multiaddr) {
				ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
				defer cancel()
				err := w.node.DialPeerWithMultiAddress(ctx, node)
				if err != nil {
					log.Warn("could not dial peer", err)
				} else {
					log.Info("relay peer dialed successfully", "multiaddr", node)
				}
			}(m)
		}
		w.addPeers(cfg.RelayNodes, relay.WakuRelayID_v200, addRelayPeer)
	}

	addToStore := func(m multiaddr.Multiaddr, protocol libp2pproto.ID) {
		peerID, err := w.node.AddPeer(m, protocol)
		if err != nil {
			log.Warn("could not add peer", "multiaddr", m, "err", err)
			return
		}
		log.Info("peer added successfully", "peerId", peerID)
	}

	w.addPeers(cfg.StoreNodes, store.StoreID_v20beta4, addToStore)
	w.addPeers(cfg.FilterNodes, filter.FilterID_v20beta1, addToStore)
	w.addPeers(cfg.LightpushNodes, lightpush.LightPushID_v20beta1, addToStore)
	w.addPeers(cfg.WakuRendezvousNodes, rendezvous.RendezvousID_v001, addToStore)
}

func (w *Waku) GetStats() types.StatsSummary {
	stats := w.bandwidthCounter.GetBandwidthTotals()
	return types.StatsSummary{
		UploadRate:   uint64(stats.RateOut),
		DownloadRate: uint64(stats.RateIn),
	}
}

func (w *Waku) runRelayMsgLoop() {
	if w.settings.LightClient {
		return
	}

	sub, err := w.node.Relay().Subscribe(context.Background())
	if err != nil {
		fmt.Println("Could not subscribe:", err)
		return
	}

	for env := range sub.C {
		envelopeErrors, err := w.OnNewEnvelopes(env, common.RelayedMessageType)
		// TODO: should these be handled?
		_ = envelopeErrors
		_ = err
	}
}

func (w *Waku) runFilterMsgLoop() {
	if !w.settings.LightClient {
		return
	}

	for {
		select {
		case <-w.quit:
			return
		case env := <-w.filterMsgChannel:
			envelopeErrors, err := w.OnNewEnvelopes(env, common.RelayedMessageType)
			// TODO: should these be handled?
			_ = envelopeErrors
			_ = err
		}
	}
}

func (w *Waku) subscribeWakuFilterTopic(topics [][]byte) {
	var contentTopics []string
	for _, topic := range topics {
		contentTopics = append(contentTopics, common.BytesToTopic(topic).ContentTopic())
	}

	var err error
	contentFilter := filter.ContentFilter{
		Topic:         relay.DefaultWakuTopic,
		ContentTopics: contentTopics,
	}

	var wakuFilter filter.Filter
	_, wakuFilter, err = w.node.Filter().Subscribe(context.Background(), contentFilter)
	if err != nil {
		w.logger.Warn("could not add wakuv2 filter for topics", zap.Any("topics", topics))
		return
	}

	w.filterMsgChannel = wakuFilter.Chan
}

// MaxMessageSize returns the maximum accepted message size.
func (w *Waku) MaxMessageSize() uint32 {
	w.settingsMu.RLock()
	defer w.settingsMu.RUnlock()
	return w.settings.MaxMsgSize
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
			Namespace: Name,
			Version:   VersionStr,
			Service:   NewPublicWakuAPI(w),
			Public:    false,
		},
	}
}

// Protocols returns the waku sub-protocols ran by this particular client.
func (w *Waku) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

func (w *Waku) SendEnvelopeEvent(event common.EnvelopeEvent) int {
	return w.envelopeFeed.Send(event)
}

// SubscribeEnvelopeEvents subscribes to envelopes feed.
// In order to prevent blocking waku producers events must be amply buffered.
func (w *Waku) SubscribeEnvelopeEvents(events chan<- common.EnvelopeEvent) event.Subscription {
	return w.envelopeFeed.Subscribe(events)
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

	id, err := toDeterministicID(hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)), common.KeyIDSize)
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
	deterministicID, err := toDeterministicID(key, common.KeyIDSize)
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
	id, err := makeDeterministicID(hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)), common.KeyIDSize)
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
	id, err := makeDeterministicID(hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)), common.KeyIDSize)
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
	deterministicID, err := toDeterministicID(id, common.KeyIDSize)
	if err != nil {
		return false
	}

	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	return w.privateKeys[deterministicID] != nil
}

// GetPrivateKey retrieves the private key of the specified identity.
func (w *Waku) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	deterministicID, err := toDeterministicID(id, common.KeyIDSize)
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
	key, err := common.GenerateSecureRandomData(common.AESKeyLength)
	if err != nil {
		return "", err
	} else if !common.ValidateDataIntegrity(key, common.AESKeyLength) {
		return "", fmt.Errorf("error in GenerateSymKey: crypto/rand failed to generate random data")
	}

	id, err := common.GenerateRandomID()
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
	deterministicID, err := toDeterministicID(id, common.KeyIDSize)
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
	if len(key) != common.AESKeyLength {
		return "", fmt.Errorf("wrong key size: %d", len(key))
	}

	id, err := common.GenerateRandomID()
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
	id, err := common.GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %s", err)
	}
	if w.HasSymKey(id) {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	// kdf should run no less than 0.1 seconds on an average computer,
	// because it's an once in a session experience
	derived := pbkdf2.Key([]byte(password), nil, 65356, common.AESKeyLength, sha256.New)

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
func (w *Waku) Subscribe(f *common.Filter) (string, error) {
	s, err := w.filters.Install(f)
	if err != nil {
		return s, err
	}

	if w.settings.LightClient {
		w.subscribeWakuFilterTopic(f.Topics)
	}

	return s, nil
}

// GetFilter returns the filter by id.
func (w *Waku) GetFilter(id string) *common.Filter {
	return w.filters.Get(id)
}

// Unsubscribe removes an installed message handler.
func (w *Waku) Unsubscribe(id string) error {
	f := w.filters.Get(id)
	if f != nil && w.settings.LightClient {
		contentFilter := filter.ContentFilter{
			Topic: relay.DefaultWakuTopic,
		}
		for _, topic := range f.Topics {
			contentFilter.ContentTopics = append(contentFilter.ContentTopics, common.BytesToTopic(topic).ContentTopic())
		}

		if err := w.node.Filter().UnsubscribeFilter(context.Background(), contentFilter); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
	}

	ok := w.filters.Uninstall(id)
	if !ok {
		return fmt.Errorf("failed to unsubscribe: invalid ID '%s'", id)
	}
	return nil
}

// Unsubscribe removes an installed message handler.
func (w *Waku) UnsubscribeMany(ids []string) error {
	for _, id := range ids {
		w.logger.Debug("cleaning up filter", zap.String("id", id))
		ok := w.filters.Uninstall(id)
		if !ok {
			w.logger.Warn("could not remove filter with id", zap.String("id", id))
		}
	}
	return nil
}

func (w *Waku) notEnoughPeers() bool {
	numPeers := len(w.node.Relay().PubSub().ListPeers(relay.DefaultWakuTopic))
	return numPeers <= w.settings.MinPeersForRelay
}

func (w *Waku) broadcast() {
	for {
		select {
		case msg := <-w.sendQueue:

			hash, err := msg.Hash()
			if err != nil {
				log.Error("invalid message")
				continue
			}

			if w.settings.LightClient || w.notEnoughPeers() {
				log.Debug("publishing message via lightpush", zap.Any("hash", hexutil.Encode(hash)))
				_, err = w.node.Lightpush().Publish(context.Background(), msg)
			} else {
				log.Debug("publishing message via relay", zap.Any("hash", hexutil.Encode(hash)))
				_, err = w.node.Relay().Publish(context.Background(), msg)
			}

			if err != nil {
				log.Error("could not send message", zap.Any("hash", hexutil.Encode(hash)), zap.Error(err))
				w.envelopeFeed.Send(common.EnvelopeEvent{
					Hash:  gethcommon.BytesToHash(hash),
					Event: common.EventEnvelopeExpired,
				})

				continue
			}

			event := common.EnvelopeEvent{
				Event: common.EventEnvelopeSent,
				Hash:  gethcommon.BytesToHash(hash),
			}

			w.SendEnvelopeEvent(event)

		case <-w.quit:
			return
		}
	}
}

// Send injects a message into the waku send queue, to be distributed in the
// network in the coming cycles.
func (w *Waku) Send(msg *pb.WakuMessage) ([]byte, error) {
	hash, err := msg.Hash()
	if err != nil {
		return nil, err
	}

	w.sendQueue <- msg

	w.poolMu.Lock()
	_, alreadyCached := w.envelopes[gethcommon.BytesToHash(hash)]
	w.poolMu.Unlock()
	if !alreadyCached {
		envelope := wakuprotocol.NewEnvelope(msg, relay.DefaultWakuTopic)
		recvMessage := common.NewReceivedMessage(envelope, common.RelayedMessageType)
		w.postEvent(recvMessage) // notify the local node about the new message
		w.addEnvelope(recvMessage)
	}

	return hash, nil
}

func (w *Waku) Query(topics []common.TopicType, from uint64, to uint64, opts []store.HistoryRequestOption) (cursor *pb.Index, err error) {
	strTopics := make([]string, len(topics))
	for i, t := range topics {
		strTopics[i] = t.ContentTopic()
	}

	query := store.Query{
		StartTime:     int64(from) * int64(time.Second),
		EndTime:       int64(to) * int64(time.Second),
		ContentTopics: strTopics,
		Topic:         relay.DefaultWakuTopic,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := w.node.Store().Query(ctx, query, opts...)
	if err != nil {
		return
	}

	for _, msg := range result.Messages {
		envelope := wakuprotocol.NewEnvelope(msg, relay.DefaultWakuTopic)
		w.logger.Debug("received waku2 store message", zap.Any("envelopeHash", hexutil.Encode(envelope.Hash())))
		_, err = w.OnNewEnvelopes(envelope, common.StoreMessageType)
		if err != nil {
			return nil, err
		}
	}

	if len(result.Messages) != 0 {
		cursor = result.Cursor()
	}

	return
}

// Start implements node.Service, starting the background data propagation thread
// of the Waku protocol.
func (w *Waku) Start() error {
	numCPU := runtime.NumCPU()
	for i := 0; i < numCPU; i++ {
		go w.processQueue()
	}

	go w.broadcast()

	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Waku protocol.
func (w *Waku) Stop() error {
	w.node.Stop()
	close(w.quit)
	close(w.filterMsgChannel)
	return nil
}

func (w *Waku) OnNewEnvelopes(envelope *wakuprotocol.Envelope, msgType common.MessageType) ([]common.EnvelopeError, error) {
	recvMessage := common.NewReceivedMessage(envelope, msgType)
	envelopeErrors := make([]common.EnvelopeError, 0)

	w.logger.Debug("received new envelope")

	trouble := false

	_, err := w.add(recvMessage)
	if err != nil {
		w.logger.Info("invalid envelope received", zap.Error(err))
	}

	common.EnvelopesValidatedCounter.Inc()

	if trouble {
		return envelopeErrors, errors.New("received invalid envelope")
	}

	return envelopeErrors, nil
}

// addEnvelope adds an envelope to the envelope map, used for sending
func (w *Waku) addEnvelope(envelope *common.ReceivedMessage) {
	hash := envelope.Hash()

	w.poolMu.Lock()
	w.envelopes[hash] = envelope
	w.poolMu.Unlock()
}

func (w *Waku) add(recvMessage *common.ReceivedMessage) (bool, error) {
	common.EnvelopesReceivedCounter.Inc()

	hash := recvMessage.Hash()

	w.poolMu.Lock()
	_, alreadyCached := w.envelopes[hash]
	w.poolMu.Unlock()
	if !alreadyCached {
		w.addEnvelope(recvMessage)
	}

	if alreadyCached {
		log.Trace("w envelope already cached", "hash", recvMessage.Hash().Hex())
		common.EnvelopesCachedCounter.WithLabelValues("hit").Inc()
	} else {
		log.Trace("cached w envelope", "hash", recvMessage.Hash().Hex())
		common.EnvelopesCachedCounter.WithLabelValues("miss").Inc()
		common.EnvelopesSizeMeter.Observe(float64(recvMessage.Envelope.Size()))
		w.postEvent(recvMessage) // notify the local node about the new message
	}
	return true, nil
}

// postEvent queues the message for further processing.
func (w *Waku) postEvent(envelope *common.ReceivedMessage) {
	w.msgQueue <- envelope
}

// processQueue delivers the messages to the watchers during the lifetime of the waku node.
func (w *Waku) processQueue() {
	for {
		select {
		case <-w.quit:
			return
		case e := <-w.msgQueue:
			if e.MsgType == common.StoreMessageType {
				// We need to insert it first, and then remove it if not matched,
				// as messages are processed asynchronously
				w.storeMsgIDsMu.Lock()
				w.storeMsgIDs[e.Hash()] = true
				w.storeMsgIDsMu.Unlock()
			}

			matched := w.filters.NotifyWatchers(e)

			// If not matched we remove it
			if !matched {
				w.storeMsgIDsMu.Lock()
				delete(w.storeMsgIDs, e.Hash())
				w.storeMsgIDsMu.Unlock()
			}

			w.envelopeFeed.Send(common.EnvelopeEvent{
				Topic: e.Topic,
				Hash:  e.Hash(),
				Event: common.EventEnvelopeAvailable,
			})
		}
	}
}

func (w *Waku) ClearPeerCache() error {
	if !w.settings.PersistPeers {
		return nil
	}

	return persistence.Clean(w.appDB, PeerStoreTable)
}

// Envelopes retrieves all the messages currently pooled by the node.
func (w *Waku) Envelopes() []*common.ReceivedMessage {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	all := make([]*common.ReceivedMessage, 0, len(w.envelopes))
	for _, envelope := range w.envelopes {
		all = append(all, envelope)
	}
	return all
}

// GetEnvelope retrieves an envelope from the message queue by its hash.
// It returns nil if the envelope can not be found.
func (w *Waku) GetEnvelope(hash gethcommon.Hash) *common.ReceivedMessage {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()
	return w.envelopes[hash]
}

// isEnvelopeCached checks if envelope with specific hash has already been received and cached.
func (w *Waku) IsEnvelopeCached(hash gethcommon.Hash) bool {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	_, exist := w.envelopes[hash]
	return exist
}

func (w *Waku) PeerCount() int {
	return w.node.PeerCount()
}

func (w *Waku) Peers() map[string][]string {
	return FormatPeerStats(w.node.PeerStats())
}

func (w *Waku) StartDiscV5() error {
	if w.node.DiscV5() == nil {
		return errors.New("discv5 is not setup")
	}

	return w.node.DiscV5().Start()
}

func (w *Waku) StopDiscV5() error {
	if w.node.DiscV5() == nil {
		return errors.New("discv5 is not setup")
	}

	w.node.DiscV5().Stop()
	return nil
}

func (w *Waku) AddStorePeer(address string) (string, error) {
	addr, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return "", err
	}

	peerID, err := w.node.AddPeer(addr, store.StoreID_v20beta4)
	if err != nil {
		return "", err
	}
	return string(*peerID), nil
}

func (w *Waku) AddRelayPeer(address string) (string, error) {
	addr, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return "", err
	}

	peerID, err := w.node.AddPeer(addr, relay.WakuRelayID_v200)
	if err != nil {
		return "", err
	}
	return string(*peerID), nil
}

func (w *Waku) DialPeer(address string) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return w.node.DialPeer(ctx, address)
}

func (w *Waku) DialPeerByID(peerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return w.node.DialPeerByID(ctx, peer.ID(peerID))
}

func (w *Waku) DropPeer(peerID string) error {
	return w.node.ClosePeerById(peer.ID(peerID))
}

func (w *Waku) ProcessingP2PMessages() bool {
	w.storeMsgIDsMu.Lock()
	defer w.storeMsgIDsMu.Unlock()
	return len(w.storeMsgIDs) != 0
}

func (w *Waku) MarkP2PMessageAsProcessed(hash gethcommon.Hash) {
	w.storeMsgIDsMu.Lock()
	defer w.storeMsgIDsMu.Unlock()
	delete(w.storeMsgIDs, hash)
}

// validatePrivateKey checks the format of the given private key.
func validatePrivateKey(k *ecdsa.PrivateKey) bool {
	if k == nil || k.D == nil || k.D.Sign() == 0 {
		return false
	}
	return common.ValidatePublicKey(&k.PublicKey)
}

// makeDeterministicID generates a deterministic ID, based on a given input
func makeDeterministicID(input string, keyLen int) (id string, err error) {
	buf := pbkdf2.Key([]byte(input), nil, 4096, keyLen, sha256.New)
	if !common.ValidateDataIntegrity(buf, common.KeyIDSize) {
		return "", fmt.Errorf("error in GenerateDeterministicID: failed to generate key")
	}
	id = gethcommon.Bytes2Hex(buf)
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

func FormatPeerStats(peers node.PeerStats) map[string][]string {
	p := make(map[string][]string)
	for k, v := range peers {
		p[k.Pretty()] = v
	}
	return p
}

func formatConnStatus(c node.ConnStatus) types.ConnStatus {
	return types.ConnStatus{
		IsOnline:   c.IsOnline,
		HasHistory: c.HasHistory,
		Peers:      FormatPeerStats(c.Peers),
	}
}
