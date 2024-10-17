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
	"math"
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"

	"go.uber.org/zap"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/time/rate"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"

	filterapi "github.com/waku-org/go-waku/waku/v2/api/filter"
	"github.com/waku-org/go-waku/waku/v2/api/missing"
	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/utils"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/timesource"
	"github.com/status-im/status-go/wakuv2/common"
	"github.com/status-im/status-go/wakuv2/persistence"

	node "github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

const messageQueueLimit = 1024
const requestTimeout = 30 * time.Second
const bootnodesQueryBackoffMs = 200
const bootnodesMaxRetries = 7
const cacheTTL = 20 * time.Minute
const maxRelayPeers = 300
const randomPeersKeepAliveInterval = 5 * time.Second
const allPeersKeepAliveInterval = 5 * time.Minute

type SentEnvelope struct {
	Envelope      *protocol.Envelope
	PublishMethod publish.PublishMethod
}

type ErrorSendingEnvelope struct {
	Error        error
	SentEnvelope SentEnvelope
}

type ITelemetryClient interface {
	SetDeviceType(deviceType string)
	PushSentEnvelope(ctx context.Context, sentEnvelope SentEnvelope)
	PushErrorSendingEnvelope(ctx context.Context, errorSendingEnvelope ErrorSendingEnvelope)
	PushPeerCount(ctx context.Context, peerCount int)
	PushPeerConnFailures(ctx context.Context, peerConnFailures map[string]int)
	PushMessageCheckSuccess(ctx context.Context, messageHash string)
	PushMessageCheckFailure(ctx context.Context, messageHash string)
	PushPeerCountByShard(ctx context.Context, peerCountByShard map[uint16]uint)
	PushPeerCountByOrigin(ctx context.Context, peerCountByOrigin map[wps.Origin]uint)
}

// Waku represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Waku struct {
	node  *node.WakuNode // reference to a libp2p waku node
	appDB *sql.DB

	dnsAddressCache             map[string][]dnsdisc.DiscoveredNode // Map to store the multiaddresses returned by dns discovery
	dnsAddressCacheLock         *sync.RWMutex                       // lock to handle access to the map
	dnsDiscAsyncRetrievedSignal chan struct{}

	// Filter-related
	filters       *common.Filters // Message filters installed with Subscribe function
	filterManager *filterapi.FilterManager

	privateKeys map[string]*ecdsa.PrivateKey // Private key storage
	symKeys     map[string][]byte            // Symmetric key storage
	keyMu       sync.RWMutex                 // Mutex associated with key stores

	envelopeCache *ttlcache.Cache[gethcommon.Hash, *common.ReceivedMessage] // Pool of envelopes currently tracked by this node
	poolMu        sync.RWMutex                                              // Mutex to sync the message and expiration pools

	bandwidthCounter *metrics.BandwidthCounter

	protectedTopicStore *persistence.ProtectedTopicsStore

	sendQueue *publish.MessageQueue

	missingMsgVerifier *missing.MissingMessageVerifier

	msgQueue chan *common.ReceivedMessage // Message queue for waku messages that havent been decoded

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cfg     *Config
	options []node.WakuNodeOption

	envelopeFeed event.Feed

	storeMsgIDs   map[gethcommon.Hash]bool // Map of the currently processing ids
	storeMsgIDsMu sync.RWMutex

	messageSender *publish.MessageSender

	topicHealthStatusChan   chan peermanager.TopicHealthStatus
	connectionNotifChan     chan node.PeerConnection
	connStatusSubscriptions map[string]*types.ConnStatusSubscription
	connStatusMu            sync.Mutex
	onlineChecker           *onlinechecker.DefaultOnlineChecker
	state                   connection.State

	logger *zap.Logger

	// NTP Synced timesource
	timesource *timesource.NTPTimeSource

	// seededBootnodesForDiscV5 indicates whether we manage to retrieve discovery
	// bootnodes successfully
	seededBootnodesForDiscV5 bool

	// goingOnline is channel that notifies when connectivity has changed from offline to online
	goingOnline chan struct{}

	// discV5BootstrapNodes is the ENR to be used to fetch bootstrap nodes for discovery
	discV5BootstrapNodes []string

	onHistoricMessagesRequestFailed func([]byte, peer.ID, error)
	onPeerStats                     func(types.ConnStatus)

	statusTelemetryClient ITelemetryClient

	defaultShardInfo protocol.RelayShards
}

func (w *Waku) SetStatusTelemetryClient(client ITelemetryClient) {
	w.statusTelemetryClient = client
}

func newTTLCache() *ttlcache.Cache[gethcommon.Hash, *common.ReceivedMessage] {
	cache := ttlcache.New[gethcommon.Hash, *common.ReceivedMessage](ttlcache.WithTTL[gethcommon.Hash, *common.ReceivedMessage](cacheTTL))
	go func() {
		defer gocommon.LogOnPanic()
		cache.Start()
	}()
	return cache
}

// New creates a WakuV2 client ready to communicate through the LibP2P network.
func New(nodeKey *ecdsa.PrivateKey, fleet string, cfg *Config, logger *zap.Logger, appDB *sql.DB, ts *timesource.NTPTimeSource, onHistoricMessagesRequestFailed func([]byte, peer.ID, error), onPeerStats func(types.ConnStatus)) (*Waku, error) {
	var err error
	if logger == nil {
		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	}

	if ts == nil {
		ts = timesource.Default()
	}

	cfg = setDefaults(cfg)
	if err = cfg.Validate(logger); err != nil {
		return nil, err
	}

	logger.Info("starting wakuv2 with config", zap.Any("config", cfg))

	ctx, cancel := context.WithCancel(context.Background())

	waku := &Waku{
		appDB:                           appDB,
		cfg:                             cfg,
		privateKeys:                     make(map[string]*ecdsa.PrivateKey),
		symKeys:                         make(map[string][]byte),
		envelopeCache:                   newTTLCache(),
		msgQueue:                        make(chan *common.ReceivedMessage, messageQueueLimit),
		topicHealthStatusChan:           make(chan peermanager.TopicHealthStatus, 100),
		connectionNotifChan:             make(chan node.PeerConnection, 20),
		connStatusSubscriptions:         make(map[string]*types.ConnStatusSubscription),
		ctx:                             ctx,
		cancel:                          cancel,
		wg:                              sync.WaitGroup{},
		dnsAddressCache:                 make(map[string][]dnsdisc.DiscoveredNode),
		dnsAddressCacheLock:             &sync.RWMutex{},
		dnsDiscAsyncRetrievedSignal:     make(chan struct{}),
		storeMsgIDs:                     make(map[gethcommon.Hash]bool),
		timesource:                      ts,
		storeMsgIDsMu:                   sync.RWMutex{},
		logger:                          logger,
		discV5BootstrapNodes:            cfg.DiscV5BootstrapNodes,
		onHistoricMessagesRequestFailed: onHistoricMessagesRequestFailed,
		onPeerStats:                     onPeerStats,
		onlineChecker:                   onlinechecker.NewDefaultOnlineChecker(false).(*onlinechecker.DefaultOnlineChecker),
		sendQueue:                       publish.NewMessageQueue(1000, cfg.UseThrottledPublish),
	}

	waku.filters = common.NewFilters(waku.cfg.DefaultShardPubsubTopic, waku.logger)
	waku.bandwidthCounter = metrics.NewBandwidthCounter()

	if nodeKey == nil {
		// No nodekey is provided, create an ephemeral key
		nodeKey, err = crypto.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate a random go-waku private key: %v", err)
		}
	}

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprint(cfg.Host, ":", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to setup the network interface: %v", err)
	}

	libp2pOpts := node.DefaultLibP2POptions
	libp2pOpts = append(libp2pOpts, libp2p.BandwidthReporter(waku.bandwidthCounter))
	libp2pOpts = append(libp2pOpts, libp2p.NATPortMap())

	opts := []node.WakuNodeOption{
		node.WithLibP2POptions(libp2pOpts...),
		node.WithPrivateKey(nodeKey),
		node.WithHostAddress(hostAddr),
		node.WithConnectionNotification(waku.connectionNotifChan),
		node.WithTopicHealthStatusChannel(waku.topicHealthStatusChan),
		node.WithKeepAlive(randomPeersKeepAliveInterval, allPeersKeepAliveInterval),
		node.WithLogger(logger),
		node.WithLogLevel(logger.Level()),
		node.WithClusterID(cfg.ClusterID),
		node.WithMaxMsgSize(1024 * 1024),
	}

	if cfg.EnableDiscV5 {
		bootnodes, err := waku.getDiscV5BootstrapNodes(waku.ctx, cfg.DiscV5BootstrapNodes, false)
		if err != nil {
			logger.Error("failed to get bootstrap nodes", zap.Error(err))
			return nil, err
		}
		opts = append(opts, node.WithDiscoveryV5(uint(cfg.UDPPort), bootnodes, cfg.AutoUpdate))
	}
	shards, err := protocol.TopicsToRelayShards(cfg.DefaultShardPubsubTopic)
	if err != nil {
		logger.Error("FATAL ERROR: failed to parse relay shards", zap.Error(err))
		return nil, errors.New("failed to parse relay shard, invalid pubsubTopic configuration")
	}
	if len(shards) == 0 { //Hack so that tests don't fail. TODO: Need to remove this once tests are changed to use proper cluster and shard.
		shardInfo := protocol.RelayShards{ClusterID: 0, ShardIDs: []uint16{0}}
		shards = append(shards, shardInfo)
	}
	waku.defaultShardInfo = shards[0]
	if cfg.LightClient {
		opts = append(opts, node.WithWakuFilterLightNode())
		waku.defaultShardInfo = shards[0]
		opts = append(opts, node.WithMaxPeerConnections(cfg.DiscoveryLimit))
		cfg.EnableStoreConfirmationForMessagesSent = false
		//TODO: temporary work-around to improve lightClient connectivity, need to be removed once community sharding is implemented
		opts = append(opts, node.WithShards(waku.defaultShardInfo.ShardIDs))
	} else {
		relayOpts := []pubsub.Option{
			pubsub.WithMaxMessageSize(int(waku.cfg.MaxMessageSize)),
		}

		if testing.Testing() {
			relayOpts = append(relayOpts, pubsub.WithEventTracer(waku))
		}

		opts = append(opts, node.WithWakuRelayAndMinPeers(waku.cfg.MinPeersForRelay, relayOpts...))
		opts = append(opts, node.WithMaxPeerConnections(maxRelayPeers))
		cfg.EnablePeerExchangeClient = true //Enabling this until discv5 issues are resolved. This will enable more peers to be connected for relay mesh.
		cfg.EnableStoreConfirmationForMessagesSent = true
	}

	if cfg.EnableStore {
		if appDB == nil {
			return nil, errors.New("appDB is required for store")
		}
		opts = append(opts, node.WithWakuStore())
		dbStore, err := persistence.NewDBStore(logger, persistence.WithDB(appDB), persistence.WithRetentionPolicy(cfg.StoreCapacity, time.Duration(cfg.StoreSeconds)*time.Second))
		if err != nil {
			return nil, err
		}
		opts = append(opts, node.WithMessageProvider(dbStore))
	}

	if !cfg.LightClient {
		opts = append(opts, node.WithWakuFilterFullNode(filter.WithMaxSubscribers(20)))
		opts = append(opts, node.WithLightPush(lightpush.WithRateLimiter(1, 1)))
	}

	if appDB != nil {
		waku.protectedTopicStore, err = persistence.NewProtectedTopicsStore(logger, appDB)
		if err != nil {
			return nil, err
		}
	}

	if cfg.EnablePeerExchangeServer {
		opts = append(opts, node.WithPeerExchange(peer_exchange.WithRateLimiter(1, 1)))
	}

	waku.options = opts
	waku.logger.Info("setup the go-waku node successfully")

	return waku, nil
}

func (w *Waku) SubscribeToConnStatusChanges() *types.ConnStatusSubscription {
	w.connStatusMu.Lock()
	defer w.connStatusMu.Unlock()
	subscription := types.NewConnStatusSubscription()
	w.connStatusSubscriptions[subscription.ID] = subscription
	return subscription
}

func (w *Waku) GetNodeENRString() (string, error) {
	if w.node == nil {
		return "", errors.New("node not initialized")
	}
	return w.node.ENR().String(), nil
}

func (w *Waku) getDiscV5BootstrapNodes(ctx context.Context, addresses []string, useOnlyDnsDiscCache bool) ([]*enode.Node, error) {
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	var result []*enode.Node

	w.seededBootnodesForDiscV5 = true

	retrieveENR := func(d dnsdisc.DiscoveredNode, wg *sync.WaitGroup) {
		mu.Lock()
		defer mu.Unlock()
		defer wg.Done()
		if d.ENR != nil {
			result = append(result, d.ENR)
		}
	}

	for _, addrString := range addresses {
		if addrString == "" {
			continue
		}

		if strings.HasPrefix(addrString, "enrtree://") {
			// Use DNS Discovery
			wg.Add(1)
			go func(addr string) {
				defer gocommon.LogOnPanic()
				defer wg.Done()
				if err := w.dnsDiscover(ctx, addr, retrieveENR, useOnlyDnsDiscCache); err != nil {
					go func() {
						defer gocommon.LogOnPanic()
						w.retryDnsDiscoveryWithBackoff(ctx, addr, w.dnsDiscAsyncRetrievedSignal)
					}()
				}
			}(addrString)
		} else {
			// It's a normal enr
			bootnode, err := enode.Parse(enode.ValidSchemes, addrString)
			if err != nil {
				return nil, err
			}
			mu.Lock()
			result = append(result, bootnode)
			mu.Unlock()
		}
	}
	wg.Wait()

	if len(result) == 0 {
		w.seededBootnodesForDiscV5 = false
	}

	return result, nil
}

type fnApplyToEachPeer func(d dnsdisc.DiscoveredNode, wg *sync.WaitGroup)

func (w *Waku) dnsDiscover(ctx context.Context, enrtreeAddress string, apply fnApplyToEachPeer, useOnlyCache bool) error {
	w.logger.Info("retrieving nodes", zap.String("enr", enrtreeAddress))
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	w.dnsAddressCacheLock.Lock()
	defer w.dnsAddressCacheLock.Unlock()

	discNodes, ok := w.dnsAddressCache[enrtreeAddress]
	if !ok && !useOnlyCache {
		nameserver := w.cfg.Nameserver
		resolver := w.cfg.Resolver

		var opts []dnsdisc.DNSDiscoveryOption
		if nameserver != "" {
			opts = append(opts, dnsdisc.WithNameserver(nameserver))
		}
		if resolver != nil {
			opts = append(opts, dnsdisc.WithResolver(resolver))
		}

		discoveredNodes, err := dnsdisc.RetrieveNodes(ctx, enrtreeAddress, opts...)
		if err != nil {
			w.logger.Warn("dns discovery error ", zap.Error(err))
			return err
		}

		if len(discoveredNodes) != 0 {
			w.dnsAddressCache[enrtreeAddress] = append(w.dnsAddressCache[enrtreeAddress], discoveredNodes...)
			discNodes = w.dnsAddressCache[enrtreeAddress]
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(discNodes))
	for _, d := range discNodes {
		apply(d, wg)
	}
	wg.Wait()

	return nil
}

func (w *Waku) retryDnsDiscoveryWithBackoff(ctx context.Context, addr string, successChan chan<- struct{}) {
	retries := 0
	for {
		err := w.dnsDiscover(ctx, addr, func(d dnsdisc.DiscoveredNode, wg *sync.WaitGroup) {}, false)
		if err == nil {
			select {
			case successChan <- struct{}{}:
			default:
			}

			break
		}

		retries++
		backoff := time.Second * time.Duration(math.Exp2(float64(retries)))
		if backoff > time.Minute {
			backoff = time.Minute
		}

		t := time.NewTimer(backoff)
		select {
		case <-w.ctx.Done():
			t.Stop()
			return
		case <-t.C:
			t.Stop()
		}
	}
}

func (w *Waku) discoverAndConnectPeers() {
	fnApply := func(d dnsdisc.DiscoveredNode, wg *sync.WaitGroup) {
		defer wg.Done()
		if len(d.PeerInfo.Addrs) != 0 {
			go w.connect(d.PeerInfo, d.ENR, wps.DNSDiscovery)
		}
	}

	for _, addrString := range w.cfg.WakuNodes {
		addrString := addrString
		if strings.HasPrefix(addrString, "enrtree://") {
			// Use DNS Discovery
			go func() {
				defer gocommon.LogOnPanic()
				if err := w.dnsDiscover(w.ctx, addrString, fnApply, false); err != nil {
					w.logger.Error("could not obtain dns discovery peers for ClusterConfig.WakuNodes", zap.Error(err), zap.String("dnsDiscURL", addrString))
				}
			}()
		} else {
			// It is a normal multiaddress
			addr, err := multiaddr.NewMultiaddr(addrString)
			if err != nil {
				w.logger.Warn("invalid peer multiaddress", zap.String("ma", addrString), zap.Error(err))
				continue
			}

			peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
			if err != nil {
				w.logger.Warn("invalid peer multiaddress", zap.Stringer("addr", addr), zap.Error(err))
				continue
			}

			go w.connect(*peerInfo, nil, wps.Static)
		}
	}
}

func (w *Waku) connect(peerInfo peer.AddrInfo, enr *enode.Node, origin wps.Origin) {
	defer gocommon.LogOnPanic()
	// Connection will be prunned eventually by the connection manager if needed
	// The peer connector in go-waku uses Connect, so it will execute identify as part of its
	w.node.AddDiscoveredPeer(peerInfo.ID, peerInfo.Addrs, origin, w.cfg.DefaultShardedPubsubTopics, enr, true)
}

func (w *Waku) telemetryBandwidthStats(telemetryServerURL string) {
	defer gocommon.LogOnPanic()
	defer w.wg.Done()

	if telemetryServerURL == "" {
		return
	}

	telemetry := NewBandwidthTelemetryClient(w.logger, telemetryServerURL)

	ticker := time.NewTicker(time.Second * 20)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			bandwidthPerProtocol := w.bandwidthCounter.GetBandwidthByProtocol()
			w.bandwidthCounter.Reset()
			go telemetry.PushProtocolStats(bandwidthPerProtocol)
		}
	}
}

func (w *Waku) GetStats() types.StatsSummary {
	stats := w.bandwidthCounter.GetBandwidthTotals()
	return types.StatsSummary{
		UploadRate:   uint64(stats.RateOut),
		DownloadRate: uint64(stats.RateIn),
	}
}

func (w *Waku) runPeerExchangeLoop() {
	defer gocommon.LogOnPanic()
	defer w.wg.Done()

	if !w.cfg.EnablePeerExchangeClient {
		// Currently peer exchange client is only used for light nodes
		return
	}

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Debug("Peer exchange loop stopped")
			return
		case <-ticker.C:
			w.logger.Info("Running peer exchange loop")

			// We select only the nodes discovered via DNS Discovery that support peer exchange
			// We assume that those peers are running peer exchange according to infra config,
			// If not, the peer selection process in go-waku will filter them out anyway
			w.dnsAddressCacheLock.RLock()
			var peers peer.IDSlice
			for _, record := range w.dnsAddressCache {
				for _, discoveredNode := range record {
					if len(discoveredNode.PeerInfo.Addrs) == 0 {
						continue
					}
					// Attempt to connect to the peers.
					// Peers will be added to the libp2p peer store thanks to identify
					go w.connect(discoveredNode.PeerInfo, discoveredNode.ENR, wps.DNSDiscovery)
					peers = append(peers, discoveredNode.PeerID)
				}
			}
			w.dnsAddressCacheLock.RUnlock()

			if len(peers) != 0 {
				err := w.node.PeerExchange().Request(w.ctx, w.cfg.DiscoveryLimit, peer_exchange.WithAutomaticPeerSelection(peers...),
					peer_exchange.FilterByShard(int(w.defaultShardInfo.ClusterID), int(w.defaultShardInfo.ShardIDs[0])))
				if err != nil {
					w.logger.Error("couldnt request peers via peer exchange", zap.Error(err))
				}
			}
		}
	}
}

func (w *Waku) GetPubsubTopic(topic string) string {
	if topic == "" {
		topic = w.cfg.DefaultShardPubsubTopic
	}

	return topic
}

func (w *Waku) unsubscribeFromPubsubTopicWithWakuRelay(topic string) error {
	topic = w.GetPubsubTopic(topic)

	if !w.node.Relay().IsSubscribed(topic) {
		return nil
	}

	contentFilter := protocol.NewContentFilter(topic)

	return w.node.Relay().Unsubscribe(w.ctx, contentFilter)
}

func (w *Waku) subscribeToPubsubTopicWithWakuRelay(topic string, pubkey *ecdsa.PublicKey) error {
	if w.cfg.LightClient {
		return errors.New("only available for full nodes")
	}

	topic = w.GetPubsubTopic(topic)

	if w.node.Relay().IsSubscribed(topic) {
		return nil
	}

	if pubkey != nil {
		err := w.node.Relay().AddSignedTopicValidator(topic, pubkey)
		if err != nil {
			return err
		}
	}

	contentFilter := protocol.NewContentFilter(topic)

	sub, err := w.node.Relay().Subscribe(w.ctx, contentFilter)
	if err != nil {
		return err
	}

	w.wg.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer w.wg.Done()
		for {
			select {
			case <-w.ctx.Done():
				err := w.node.Relay().Unsubscribe(w.ctx, contentFilter)
				if err != nil && !errors.Is(err, context.Canceled) {
					w.logger.Error("could not unsubscribe", zap.Error(err))
				}
				return
			case env := <-sub[0].Ch:
				err := w.OnNewEnvelopes(env, common.RelayedMessageType, false)
				if err != nil {
					w.logger.Error("OnNewEnvelopes error", zap.Error(err))
				}
			}
		}
	}()

	return nil
}

// MaxMessageSize returns the maximum accepted message size.
func (w *Waku) MaxMessageSize() uint32 {
	return w.cfg.MaxMessageSize
}

// CurrentTime returns current time.
func (w *Waku) CurrentTime() time.Time {
	return w.timesource.Now()
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
	f.PubsubTopic = w.GetPubsubTopic(f.PubsubTopic)
	id, err := w.filters.Install(f)
	if err != nil {
		return id, err
	}

	if w.cfg.LightClient {
		cf := protocol.NewContentFilter(f.PubsubTopic, f.ContentTopics.ContentTopics()...)
		w.filterManager.SubscribeFilter(id, cf)
	}

	return id, nil
}

// Unsubscribe removes an installed message handler.
func (w *Waku) Unsubscribe(ctx context.Context, id string) error {
	ok := w.filters.Uninstall(id)
	if !ok {
		return fmt.Errorf("failed to unsubscribe: invalid ID '%s'", id)
	}

	if w.cfg.LightClient {
		w.filterManager.UnsubscribeFilter(id)
	}

	return nil
}

// GetFilter returns the filter by id.
func (w *Waku) GetFilter(id string) *common.Filter {
	return w.filters.Get(id)
}

// Unsubscribe removes an installed message handler.
func (w *Waku) UnsubscribeMany(ids []string) error {
	for _, id := range ids {
		w.logger.Info("cleaning up filter", zap.String("id", id))
		ok := w.filters.Uninstall(id)
		if !ok {
			w.logger.Warn("could not remove filter with id", zap.String("id", id))
		}
	}
	return nil
}

func (w *Waku) SkipPublishToTopic(value bool) {
	w.cfg.SkipPublishToTopic = value
}

func (w *Waku) ConfirmMessageDelivered(hashes []gethcommon.Hash) {
	w.messageSender.MessagesDelivered(hashes)
}

func (w *Waku) SetStorePeerID(peerID peer.ID) {
	w.messageSender.SetStorePeerID(peerID)
}

func (w *Waku) Query(ctx context.Context, peerID peer.ID, query store.FilterCriteria, cursor []byte, opts []store.RequestOption, processEnvelopes bool) ([]byte, int, error) {
	requestID := protocol.GenerateRequestID()

	opts = append(opts,
		store.WithRequestID(requestID),
		store.WithPeer(peerID),
		store.WithCursor(cursor))

	logger := w.logger.With(zap.String("requestID", hexutil.Encode(requestID)), zap.Stringer("peerID", peerID))

	logger.Debug("store.query",
		logutils.WakuMessageTimestamp("startTime", query.TimeStart),
		logutils.WakuMessageTimestamp("endTime", query.TimeEnd),
		zap.Strings("contentTopics", query.ContentTopics.ToList()),
		zap.String("pubsubTopic", query.PubsubTopic),
		zap.String("cursor", hexutil.Encode(cursor)),
	)

	queryStart := time.Now()
	result, err := w.node.Store().Query(ctx, query, opts...)
	queryDuration := time.Since(queryStart)
	if err != nil {
		logger.Error("error querying storenode", zap.Error(err))

		if w.onHistoricMessagesRequestFailed != nil {
			w.onHistoricMessagesRequestFailed(requestID, peerID, err)
		}
		return nil, 0, err
	}

	messages := result.Messages()
	envelopesCount := len(messages)
	w.logger.Debug("store.query response", zap.Duration("queryDuration", queryDuration), zap.Int("numMessages", envelopesCount), zap.Bool("hasCursor", result.IsComplete() && result.Cursor() != nil))
	for _, mkv := range messages {
		msg := mkv.Message

		// Temporarily setting RateLimitProof to nil so it matches the WakuMessage protobuffer we are sending
		// See https://github.com/vacp2p/rfc/issues/563
		mkv.Message.RateLimitProof = nil

		envelope := protocol.NewEnvelope(msg, msg.GetTimestamp(), query.PubsubTopic)

		err = w.OnNewEnvelopes(envelope, common.StoreMessageType, processEnvelopes)
		if err != nil {
			return nil, 0, err
		}
	}

	return result.Cursor(), envelopesCount, nil
}

// OnNewEnvelope is an interface from Waku FilterManager API that gets invoked when any new message is received by Filter.
func (w *Waku) OnNewEnvelope(env *protocol.Envelope) error {
	return w.OnNewEnvelopes(env, common.RelayedMessageType, false)
}

// Start implements node.Service, starting the background data propagation thread
// of the Waku protocol.
func (w *Waku) Start() error {
	if w.ctx == nil {
		w.ctx, w.cancel = context.WithCancel(context.Background())
	}

	var err error
	if w.node, err = node.New(w.options...); err != nil {
		return fmt.Errorf("failed to create a go-waku node: %v", err)
	}

	w.goingOnline = make(chan struct{})

	if err = w.node.Start(w.ctx); err != nil {
		return fmt.Errorf("failed to start go-waku node: %v", err)
	}

	w.logger.Info("WakuV2 PeerID", zap.Stringer("id", w.node.Host().ID()))

	w.discoverAndConnectPeers()

	if w.cfg.EnableDiscV5 {
		err := w.node.DiscV5().Start(w.ctx)
		if err != nil {
			return err
		}
	}

	w.wg.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer w.wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.checkForConnectionChanges()
			case <-w.topicHealthStatusChan:
				// TODO: https://github.com/status-im/status-go/issues/4628
			case <-w.connectionNotifChan:
				w.checkForConnectionChanges()
			}
		}
	}()

	if w.cfg.TelemetryServerURL != "" {
		w.wg.Add(1)
		go func() {
			defer gocommon.LogOnPanic()
			defer w.wg.Done()
			peerTelemetryTickerInterval := time.Duration(w.cfg.TelemetryPeerCountSendPeriod) * time.Millisecond
			if peerTelemetryTickerInterval == 0 {
				peerTelemetryTickerInterval = 10 * time.Second
			}
			peerTelemetryTicker := time.NewTicker(peerTelemetryTickerInterval)
			defer peerTelemetryTicker.Stop()

			for {
				select {
				case <-w.ctx.Done():
					return
				case <-peerTelemetryTicker.C:
					w.reportPeerMetrics()
				}
			}
		}()
	}

	w.wg.Add(1)
	go w.telemetryBandwidthStats(w.cfg.TelemetryServerURL)
	//TODO: commenting for now so that only fleet nodes are used.
	//Need to uncomment once filter peer scoring etc is implemented.

	w.wg.Add(1)
	go w.runPeerExchangeLoop()

	if w.cfg.EnableMissingMessageVerification {

		w.missingMsgVerifier = missing.NewMissingMessageVerifier(
			w.node.Store(),
			w,
			w.node.Timesource(),
			w.logger)

		w.missingMsgVerifier.Start(w.ctx)

		w.wg.Add(1)
		go func() {
			defer gocommon.LogOnPanic()
			w.wg.Done()
			for {
				select {
				case <-w.ctx.Done():
					return
				case envelope := <-w.missingMsgVerifier.C:
					err = w.OnNewEnvelopes(envelope, common.MissingMessageType, false)
					if err != nil {
						w.logger.Error("OnNewEnvelopes error", zap.Error(err))
					}
				}
			}
		}()
	}

	if w.cfg.LightClient {
		// Create FilterManager that will main peer connectivity
		// for installed filters
		w.filterManager = filterapi.NewFilterManager(
			w.ctx,
			w.logger,
			w.cfg.MinPeersForFilter,
			w,
			w.node.FilterLightnode(),
			filterapi.WithBatchInterval(300*time.Millisecond))
	}

	err = w.setupRelaySubscriptions()
	if err != nil {
		return err
	}

	numCPU := runtime.NumCPU()
	for i := 0; i < numCPU; i++ {
		w.wg.Add(1)
		go w.processQueueLoop()
	}

	w.wg.Add(1)
	go w.broadcast()

	go func() {
		defer gocommon.LogOnPanic()
		w.sendQueue.Start(w.ctx)
	}()

	err = w.startMessageSender()
	if err != nil {
		return err
	}

	// we should wait `seedBootnodesForDiscV5` shutdown smoothly before set w.ctx to nil within `w.Stop()`
	w.wg.Add(1)
	go w.seedBootnodesForDiscV5()

	return nil
}

func (w *Waku) checkForConnectionChanges() {

	isOnline := len(w.node.Host().Network().Peers()) > 0

	w.connStatusMu.Lock()

	latestConnStatus := types.ConnStatus{
		IsOnline: isOnline,
		Peers:    FormatPeerStats(w.node),
	}

	w.logger.Debug("peer stats",
		zap.Int("peersCount", len(latestConnStatus.Peers)),
		zap.Any("stats", latestConnStatus))
	for k, subs := range w.connStatusSubscriptions {
		if !subs.Send(latestConnStatus) {
			delete(w.connStatusSubscriptions, k)
		}
	}

	w.connStatusMu.Unlock()

	if w.onPeerStats != nil {
		w.onPeerStats(latestConnStatus)
	}

	w.ConnectionChanged(connection.State{
		Type:    w.state.Type, //setting state type as previous one since there won't be a change here
		Offline: !latestConnStatus.IsOnline,
	})
}

func (w *Waku) reportPeerMetrics() {
	if w.statusTelemetryClient != nil {
		connFailures := FormatPeerConnFailures(w.node)
		w.statusTelemetryClient.PushPeerCount(w.ctx, w.PeerCount())
		w.statusTelemetryClient.PushPeerConnFailures(w.ctx, connFailures)

		peerCountByOrigin := make(map[wps.Origin]uint)
		peerCountByShard := make(map[uint16]uint)
		wakuPeerStore := w.node.Host().Peerstore().(wps.WakuPeerstore)

		for _, peerID := range w.node.Host().Network().Peers() {
			origin, err := wakuPeerStore.Origin(peerID)
			if err != nil {
				origin = wps.Unknown
			}

			peerCountByOrigin[origin]++
			pubsubTopics, err := wakuPeerStore.PubSubTopics(peerID)
			if err != nil {
				continue
			}

			keys := make([]string, 0, len(pubsubTopics))
			for k := range pubsubTopics {
				keys = append(keys, k)
			}
			relayShards, err := protocol.TopicsToRelayShards(keys...)
			if err != nil {
				continue
			}

			for _, shards := range relayShards {
				for _, shard := range shards.ShardIDs {
					peerCountByShard[shard]++
				}
			}
		}
		w.statusTelemetryClient.PushPeerCountByShard(w.ctx, peerCountByShard)
		w.statusTelemetryClient.PushPeerCountByOrigin(w.ctx, peerCountByOrigin)
	}
}

func (w *Waku) startMessageSender() error {
	publishMethod := publish.Relay
	if w.cfg.LightClient {
		publishMethod = publish.LightPush
	}

	sender, err := publish.NewMessageSender(publishMethod, w.node.Lightpush(), w.node.Relay(), w.logger)
	if err != nil {
		w.logger.Error("failed to create message sender", zap.Error(err))
		return err
	}

	if w.cfg.EnableStoreConfirmationForMessagesSent {
		msgStoredChan := make(chan gethcommon.Hash, 1000)
		msgExpiredChan := make(chan gethcommon.Hash, 1000)
		messageSentCheck := publish.NewMessageSentCheck(w.ctx, w.node.Store(), w.node.Timesource(), msgStoredChan, msgExpiredChan, w.logger)
		sender.WithMessageSentCheck(messageSentCheck)

		w.wg.Add(1)
		go func() {
			defer gocommon.LogOnPanic()
			defer w.wg.Done()
			for {
				select {
				case <-w.ctx.Done():
					return
				case hash := <-msgStoredChan:
					w.SendEnvelopeEvent(common.EnvelopeEvent{
						Hash:  hash,
						Event: common.EventEnvelopeSent,
					})
					if w.statusTelemetryClient != nil {
						w.statusTelemetryClient.PushMessageCheckSuccess(w.ctx, hash.Hex())
					}
				case hash := <-msgExpiredChan:
					w.SendEnvelopeEvent(common.EnvelopeEvent{
						Hash:  hash,
						Event: common.EventEnvelopeExpired,
					})
					if w.statusTelemetryClient != nil {
						w.statusTelemetryClient.PushMessageCheckFailure(w.ctx, hash.Hex())
					}
				}
			}
		}()
	}

	if !w.cfg.UseThrottledPublish || testing.Testing() {
		// To avoid delaying the tests, or for when we dont want to rate limit, we set up an infinite rate limiter,
		// basically disabling the rate limit functionality
		limiter := publish.NewPublishRateLimiter(rate.Inf, 1)
		sender.WithRateLimiting(limiter)
	}

	w.messageSender = sender
	w.messageSender.Start()

	return nil
}

func (w *Waku) MessageExists(mh pb.MessageHash) (bool, error) {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()
	return w.envelopeCache.Has(gethcommon.Hash(mh)), nil
}

func (w *Waku) SetTopicsToVerifyForMissingMessages(peerID peer.ID, pubsubTopic string, contentTopics []string) {
	if !w.cfg.EnableMissingMessageVerification {
		return
	}

	w.missingMsgVerifier.SetCriteriaInterest(peerID, protocol.NewContentFilter(pubsubTopic, contentTopics...))
}

func (w *Waku) setupRelaySubscriptions() error {
	if w.cfg.LightClient {
		return nil
	}

	if w.protectedTopicStore != nil {
		protectedTopics, err := w.protectedTopicStore.ProtectedTopics()
		if err != nil {
			return err
		}

		for _, pt := range protectedTopics {
			// Adding subscription to protected topics
			err = w.subscribeToPubsubTopicWithWakuRelay(pt.Topic, pt.PubKey)
			if err != nil {
				return err
			}
		}
	}

	err := w.subscribeToPubsubTopicWithWakuRelay(w.cfg.DefaultShardPubsubTopic, nil)
	if err != nil {
		return err
	}

	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Waku protocol.
func (w *Waku) Stop() error {
	w.cancel()

	w.envelopeCache.Stop()

	w.node.Stop()

	if w.protectedTopicStore != nil {
		err := w.protectedTopicStore.Close()
		if err != nil {
			return err
		}
	}

	close(w.goingOnline)
	w.wg.Wait()

	w.ctx = nil
	w.cancel = nil

	return nil
}

func (w *Waku) OnNewEnvelopes(envelope *protocol.Envelope, msgType common.MessageType, processImmediately bool) error {
	if envelope == nil {
		return nil
	}

	recvMessage := common.NewReceivedMessage(envelope, msgType)
	if recvMessage == nil {
		return nil
	}

	logger := w.logger.With(
		zap.String("messageType", msgType),
		zap.Stringer("envelopeHash", envelope.Hash()),
		zap.String("pubsubTopic", envelope.PubsubTopic()),
		zap.String("contentTopic", envelope.Message().ContentTopic),
		logutils.WakuMessageTimestamp("timestamp", envelope.Message().Timestamp),
	)

	logger.Debug("received new envelope")
	trouble := false

	_, err := w.add(recvMessage, processImmediately)
	if err != nil {
		logger.Info("invalid envelope received", zap.Error(err))
		trouble = true
	}

	common.EnvelopesValidatedCounter.Inc()

	if trouble {
		return errors.New("received invalid envelope")
	}

	return nil
}

// addEnvelope adds an envelope to the envelope map, used for sending
func (w *Waku) addEnvelope(envelope *common.ReceivedMessage) {
	w.poolMu.Lock()
	w.envelopeCache.Set(envelope.Hash(), envelope, ttlcache.DefaultTTL)
	w.poolMu.Unlock()
}

func (w *Waku) add(recvMessage *common.ReceivedMessage, processImmediately bool) (bool, error) {
	common.EnvelopesReceivedCounter.Inc()

	w.poolMu.Lock()
	envelope := w.envelopeCache.Get(recvMessage.Hash())
	alreadyCached := envelope != nil
	w.poolMu.Unlock()

	if !alreadyCached {
		recvMessage.Processed.Store(false)
		w.addEnvelope(recvMessage)
	}

	logger := w.logger.With(zap.String("envelopeHash", recvMessage.Hash().Hex()))

	if alreadyCached {
		logger.Debug("w envelope already cached")
		common.EnvelopesCachedCounter.WithLabelValues("hit").Inc()
	} else {
		logger.Debug("cached w envelope")
		common.EnvelopesCachedCounter.WithLabelValues("miss").Inc()
		common.EnvelopesSizeMeter.Observe(float64(len(recvMessage.Envelope.Message().Payload)))
	}

	if !alreadyCached || !envelope.Value().Processed.Load() {
		if processImmediately {
			logger.Debug("immediately processing envelope")
			w.processMessage(recvMessage)
		} else {
			logger.Debug("posting event")
			w.postEvent(recvMessage) // notify the local node about the new message
		}
	}

	return true, nil
}

// postEvent queues the message for further processing.
func (w *Waku) postEvent(envelope *common.ReceivedMessage) {
	w.msgQueue <- envelope
}

// processQueueLoop delivers the messages to the watchers during the lifetime of the waku node.
func (w *Waku) processQueueLoop() {
	defer gocommon.LogOnPanic()
	defer w.wg.Done()
	if w.ctx == nil {
		return
	}
	for {
		select {
		case <-w.ctx.Done():
			return
		case e := <-w.msgQueue:
			w.processMessage(e)
		}
	}
}

func (w *Waku) processMessage(e *common.ReceivedMessage) {
	logger := w.logger.With(
		zap.Stringer("envelopeHash", e.Envelope.Hash()),
		zap.String("pubsubTopic", e.PubsubTopic),
		zap.String("contentTopic", e.ContentTopic.ContentTopic()),
		zap.Int64("timestamp", e.Envelope.Message().GetTimestamp()),
	)

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
		logger.Debug("filters did not match")
		w.storeMsgIDsMu.Lock()
		delete(w.storeMsgIDs, e.Hash())
		w.storeMsgIDsMu.Unlock()
	} else {
		logger.Debug("filters did match")
		e.Processed.Store(true)
	}

	w.envelopeFeed.Send(common.EnvelopeEvent{
		Topic: e.ContentTopic,
		Hash:  e.Hash(),
		Event: common.EventEnvelopeAvailable,
	})
}

// GetEnvelope retrieves an envelope from the message queue by its hash.
// It returns nil if the envelope can not be found.
func (w *Waku) GetEnvelope(hash gethcommon.Hash) *common.ReceivedMessage {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	envelope := w.envelopeCache.Get(hash)
	if envelope == nil {
		return nil
	}

	return envelope.Value()
}

// isEnvelopeCached checks if envelope with specific hash has already been received and cached.
func (w *Waku) IsEnvelopeCached(hash gethcommon.Hash) bool {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	return w.envelopeCache.Has(hash)
}

func (w *Waku) ClearEnvelopesCache() {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	w.envelopeCache.Stop()
	w.envelopeCache = newTTLCache()
}

func (w *Waku) PeerCount() int {
	return w.node.PeerCount()
}

func (w *Waku) Peers() types.PeerStats {
	return FormatPeerStats(w.node)
}

func (w *Waku) RelayPeersByTopic(topic string) (*types.PeerList, error) {
	if w.cfg.LightClient {
		return nil, errors.New("only available in relay mode")
	}

	return &types.PeerList{
		FullMeshPeers: w.node.Relay().PubSub().MeshPeers(topic),
		AllPeers:      w.node.Relay().PubSub().ListPeers(topic),
	}, nil
}

func (w *Waku) ListenAddresses() []multiaddr.Multiaddr {
	return w.node.ListenAddresses()
}

func (w *Waku) ENR() (*enode.Node, error) {
	enr := w.node.ENR()
	if enr == nil {
		return nil, errors.New("enr not available")
	}

	return enr, nil
}

func (w *Waku) SubscribeToPubsubTopic(topic string, pubkey *ecdsa.PublicKey) error {
	topic = w.GetPubsubTopic(topic)

	if !w.cfg.LightClient {
		err := w.subscribeToPubsubTopicWithWakuRelay(topic, pubkey)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Waku) UnsubscribeFromPubsubTopic(topic string) error {
	topic = w.GetPubsubTopic(topic)

	if !w.cfg.LightClient {
		err := w.unsubscribeFromPubsubTopicWithWakuRelay(topic)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Waku) RetrievePubsubTopicKey(topic string) (*ecdsa.PrivateKey, error) {
	topic = w.GetPubsubTopic(topic)
	if w.protectedTopicStore == nil {
		return nil, nil
	}

	return w.protectedTopicStore.FetchPrivateKey(topic)
}

func (w *Waku) StorePubsubTopicKey(topic string, privKey *ecdsa.PrivateKey) error {
	topic = w.GetPubsubTopic(topic)
	if w.protectedTopicStore == nil {
		return nil
	}

	return w.protectedTopicStore.Insert(topic, privKey, &privKey.PublicKey)
}

func (w *Waku) RemovePubsubTopicKey(topic string) error {
	topic = w.GetPubsubTopic(topic)
	if w.protectedTopicStore == nil {
		return nil
	}

	return w.protectedTopicStore.Delete(topic)
}

func (w *Waku) StartDiscV5() error {
	if w.node.DiscV5() == nil {
		return errors.New("discv5 is not setup")
	}

	return w.node.DiscV5().Start(w.ctx)
}

func (w *Waku) StopDiscV5() error {
	if w.node.DiscV5() == nil {
		return errors.New("discv5 is not setup")
	}

	w.node.DiscV5().Stop()
	return nil
}

func (w *Waku) handleNetworkChangeFromApp(state connection.State) {
	//If connection state is reported by something other than peerCount becoming 0 e.g from mobile app, disconnect all peers
	if (state.Offline && len(w.node.Host().Network().Peers()) > 0) ||
		(w.state.Type != state.Type && !w.state.Offline && !state.Offline) { // network switched between wifi and cellular
		w.logger.Info("connection switched or offline detected via mobile, disconnecting all peers")
		w.node.DisconnectAllPeers()
		if w.cfg.LightClient {
			w.filterManager.NetworkChange()
		}
	}
}

func (w *Waku) ConnectionChanged(state connection.State) {
	isOnline := !state.Offline
	if w.cfg.LightClient {
		//TODO: Update this as per  https://github.com/waku-org/go-waku/issues/1114
		go func() {
			defer gocommon.LogOnPanic()
			w.filterManager.OnConnectionStatusChange("", isOnline)
		}()
		w.handleNetworkChangeFromApp(state)
	} else {
		// for lightClient state update and onlineChange is handled in filterManager.
		// going online
		if isOnline && !w.onlineChecker.IsOnline() {
			//TODO: analyze if we need to discover and connect to peers for relay.
			w.discoverAndConnectPeers()
			select {
			case w.goingOnline <- struct{}{}:
			default:
				w.logger.Warn("could not write on connection changed channel")
			}
		}
		// update state
		w.onlineChecker.SetOnline(isOnline)
	}
	w.state = state
}

// seedBootnodesForDiscV5 tries to fetch bootnodes
// from an ENR periodically.
// It backs off exponentially until maxRetries, at which point it restarts from 0
// It also restarts if there's a connection change signalled from the client
func (w *Waku) seedBootnodesForDiscV5() {
	defer gocommon.LogOnPanic()
	defer w.wg.Done()

	if !w.cfg.EnableDiscV5 || w.node.DiscV5() == nil {
		return
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var retries = 0

	now := func() int64 {
		return time.Now().UnixNano() / int64(time.Millisecond)

	}

	var lastTry = now()

	canQuery := func() bool {
		backoff := bootnodesQueryBackoffMs * int64(math.Exp2(float64(retries)))

		return lastTry+backoff < now()
	}

	for {
		select {
		case <-w.dnsDiscAsyncRetrievedSignal:
			if !canQuery() {
				continue
			}

			err := w.restartDiscV5(true)
			if err != nil {
				w.logger.Warn("failed to restart discv5", zap.Error(err))
			}
			retries = 0
			lastTry = now()
		case <-ticker.C:
			if w.seededBootnodesForDiscV5 && len(w.node.Host().Network().Peers()) > 3 {
				w.logger.Debug("not querying bootnodes", zap.Bool("seeded", w.seededBootnodesForDiscV5), zap.Int("peer-count", len(w.node.Host().Network().Peers())))
				continue
			}

			if !canQuery() {
				w.logger.Info("can't query bootnodes",
					zap.Int("peer-count", len(w.node.Host().Network().Peers())),
					zap.Int64("lastTry", lastTry), zap.Int64("now", now()),
					zap.Int64("backoff", bootnodesQueryBackoffMs*int64(math.Exp2(float64(retries)))),
					zap.Int("retries", retries),
				)
				continue
			}

			w.logger.Info("querying bootnodes to restore connectivity", zap.Int("peer-count", len(w.node.Host().Network().Peers())))
			err := w.restartDiscV5(false)
			if err != nil {
				w.logger.Warn("failed to restart discv5", zap.Error(err))
			}

			lastTry = now()
			retries++
			// We reset the retries after a while and restart
			if retries > bootnodesMaxRetries {
				retries = 0
			}

		// If we go online, trigger immediately
		case <-w.goingOnline:
			if !canQuery() {
				continue
			}

			err := w.restartDiscV5(false)
			if err != nil {
				w.logger.Warn("failed to restart discv5", zap.Error(err))
			}
			retries = 0
			lastTry = now()

		case <-w.ctx.Done():
			w.logger.Debug("bootnode seeding stopped")
			return
		}
	}
}

// Restart discv5, re-retrieving bootstrap nodes
func (w *Waku) restartDiscV5(useOnlyDNSDiscCache bool) error {
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()
	bootnodes, err := w.getDiscV5BootstrapNodes(ctx, w.discV5BootstrapNodes, useOnlyDNSDiscCache)
	if err != nil {
		return err
	}
	if len(bootnodes) == 0 {
		return errors.New("failed to fetch bootnodes")
	}

	if w.node.DiscV5().ErrOnNotRunning() != nil {
		w.logger.Info("is not started restarting")
		err := w.node.DiscV5().Start(w.ctx)
		if err != nil {
			w.logger.Error("Could not start DiscV5", zap.Error(err))
		}
	} else {
		w.node.DiscV5().Stop()
		w.logger.Info("is started restarting")

		select {
		case <-w.ctx.Done(): // Don't start discv5 if we are stopping waku
			return nil
		default:
		}

		err := w.node.DiscV5().Start(w.ctx)
		if err != nil {
			w.logger.Error("Could not start DiscV5", zap.Error(err))
		}
	}

	w.logger.Info("restarting discv5 with nodes", zap.Any("nodes", bootnodes))
	return w.node.SetDiscV5Bootnodes(bootnodes)
}

func (w *Waku) AddStorePeer(address multiaddr.Multiaddr) (peer.ID, error) {
	peerID, err := w.node.AddPeer(address, wps.Static, w.cfg.DefaultShardedPubsubTopics, store.StoreQueryID_v300)
	if err != nil {
		return "", err
	}
	return peerID, nil
}

func (w *Waku) timestamp() int64 {
	return w.timesource.Now().UnixNano()
}

func (w *Waku) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	peerID, err := w.node.AddPeer(address, wps.Static, w.cfg.DefaultShardedPubsubTopics, relay.WakuRelayID_v200)
	if err != nil {
		return "", err
	}
	return peerID, nil
}

func (w *Waku) DialPeer(address multiaddr.Multiaddr) error {
	ctx, cancel := context.WithTimeout(w.ctx, requestTimeout)
	defer cancel()
	return w.node.DialPeerWithMultiAddress(ctx, address)
}

func (w *Waku) DialPeerByID(peerID peer.ID) error {
	ctx, cancel := context.WithTimeout(w.ctx, requestTimeout)
	defer cancel()
	return w.node.DialPeerByID(ctx, peerID)
}

func (w *Waku) DropPeer(peerID peer.ID) error {
	return w.node.ClosePeerById(peerID)
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

func (w *Waku) Clean() error {
	w.msgQueue = make(chan *common.ReceivedMessage, messageQueueLimit)

	for _, f := range w.filters.All() {
		f.Messages = common.NewMemoryMessageStore()
	}

	return nil
}

func (w *Waku) PeerID() peer.ID {
	return w.node.Host().ID()
}

func (w *Waku) PingPeer(ctx context.Context, peerID peer.ID) (time.Duration, error) {
	pingResultCh := ping.Ping(ctx, w.node.Host(), peerID)
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case r := <-pingResultCh:
		if r.Error != nil {
			return 0, r.Error
		}
		return r.RTT, nil
	}
}

func (w *Waku) Peerstore() peerstore.Peerstore {
	return w.node.Host().Peerstore()
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

func FormatPeerStats(wakuNode *node.WakuNode) types.PeerStats {
	p := make(types.PeerStats)
	for k, v := range wakuNode.PeerStats() {
		p[k] = types.WakuV2Peer{
			Addresses: utils.EncapsulatePeerID(k, wakuNode.Host().Peerstore().PeerInfo(k).Addrs...),
			Protocols: v,
		}
	}
	return p
}

func (w *Waku) StoreNode() *store.WakuStore {
	return w.node.Store()
}

func FormatPeerConnFailures(wakuNode *node.WakuNode) map[string]int {
	p := make(map[string]int)
	for _, peerID := range wakuNode.Host().Network().Peers() {
		peerInfo := wakuNode.Host().Peerstore().PeerInfo(peerID)
		connFailures := wakuNode.Host().Peerstore().(wps.WakuPeerstore).ConnFailures(peerInfo.ID)
		if connFailures > 0 {
			p[peerID.String()] = connFailures
		}
	}
	return p
}

func (w *Waku) LegacyStoreNode() legacy_store.Store {
	return w.node.LegacyStore()
}
