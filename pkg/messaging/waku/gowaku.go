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

// Generate a mock for peerAddressHandler. Keep it in same dir and package, as it's a private type.
// Yet we name the file _test.go to keep it only available in testing environment.
//go:generate go tool mockgen -source=gowaku.go -destination=gowaku_mock_test.go -package=wakuv2

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math"
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/jellydator/ttlcache/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"

	"go.uber.org/zap"

	"golang.org/x/time/rate"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/metrics"

	filterapi "github.com/waku-org/go-waku/waku/v2/api/filter"
	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange"
	"github.com/waku-org/go-waku/waku/v2/utils"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/pkg/messaging/waku/common"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

const messageQueueLimit = 1024
const requestTimeout = 30 * time.Second
const bootnodesQueryBackoffMs = 200
const bootnodesMaxRetries = 7
const cacheTTL = 20 * time.Minute
const maxRelayPeers = 300
const randomPeersKeepAliveInterval = 5 * time.Second
const allPeersKeepAliveInterval = 5 * time.Minute
const pausedModePeerExchangeMinPeers = 3

type SentEnvelope struct {
	Envelope      *protocol.Envelope
	PublishMethod publish.PublishMethod
}

type ErrorSendingEnvelope struct {
	Error        error
	SentEnvelope SentEnvelope
}

type IMetricsHandler interface {
	PushSentEnvelope(sentEnvelope SentEnvelope)
	PushErrorSendingEnvelope(errorSendingEnvelope ErrorSendingEnvelope)
	PushPeerConnFailures(peerConnFailures map[string]int)
	PushMessageCheckSuccess()
	PushMessageCheckFailure()
	PushPeerCountByShard(peerCountByShard map[uint16]uint)
	PushPeerCountByOrigin(peerCountByOrigin map[wps.Origin]uint)
	PushDialFailure(dialFailure common.DialError)
	PushMissedMessage(envelope common.Envelope)
	PushMissedRelevantMessage(message *common.ReceivedMessage)
	PushMessageDeliveryConfirmed()
	PushSentMessageTotal(messageSize uint32, publishMethod string)
	PushRawMessageByType(pubsubTopic string, contentTopic string, messageType string, messageSize uint32)
}

// Waku represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Waku struct {
	gocommon.PauseBroadcaster

	node *node.WakuNode // reference to a libp2p waku node

	dnsAddressCache             map[string][]dnsdisc.DiscoveredNode // Map to store the multiaddresses returned by dns discovery
	dnsAddressCacheLock         *sync.RWMutex                       // lock to handle access to the map
	dnsDiscAsyncRetrievedSignal chan struct{}

	// Light-client filter protocol; tracks the wire subscriptions declared via
	// SyncSubscriptions.
	filterManager *filterapi.FilterManager

	// subscriptions maps each (pubsub, content) topic pair declared by the
	// transport via Subscribe to the internal subscription id it is registered
	// under in the filterManager — go-waku's filter API is id-based, so the
	// adapter has to remember the id to honor Unsubscribe. The id never leaves
	// the adapter; the logos-delivery backend is topic-keyed and needs no such
	// table.
	subscriptionsMu sync.Mutex
	subscriptions   map[types.TopicSubscription]string

	envelopeCache *ttlcache.Cache[gethcommon.Hash, struct{}] // short-lived set of seen envelope hashes; feeds hit/miss metrics and self-send suppression only. De-duplication is owned by the transport's persistent processed-message cache (status-im/status-go#7464).
	poolMu        sync.RWMutex                               // Mutex to sync the message and expiration pools

	bandwidthCounter *metrics.BandwidthCounter

	sendQueue *publish.MessageQueue

	msgQueue chan *common.ReceivedMessage // Message queue for waku messages that havent been decoded

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cfg     *Config
	options []node.WakuNodeOption

	envelopeFeed event.Feed

	messageSender *publish.MessageSender

	topicHealthStatusChan   chan peermanager.TopicHealthStatus
	connectionNotifChan     chan node.PeerConnection
	connStatusSubscriptions map[string]*types.ConnStatusSubscription
	connStatusMu            sync.Mutex
	onlineChecker           *onlinechecker.DefaultOnlineChecker
	// stateMu guards state and stateInitialized. ConnectionChanged is invoked
	// from the OS/mobile path while checkForConnectionChanges and
	// handleNetworkChangeFromApp run on the internal poller goroutine, so all
	// three accesses must be synchronized.
	stateMu sync.RWMutex
	state   connection.State
	// stateInitialized distinguishes "no ConnectionChange received yet" from
	// "ConnectionChange received with Offline=false"
	stateInitialized bool

	storeClient *StoreClient

	logger *zap.Logger

	timesource timesource.Provider

	// seededBootnodesForDiscV5 indicates whether we manage to retrieve discovery
	// bootnodes successfully
	seededBootnodesForDiscV5 bool

	// goingOnline is channel that notifies when connectivity has changed from offline to online
	goingOnline chan struct{}

	metricsHandler IMetricsHandler

	defaultShardInfo protocol.RelayShards
}

var _ types.Waku = (*Waku)(nil)

func (w *Waku) SetMetricsHandler(client IMetricsHandler) {
	w.metricsHandler = client
}

func newTTLCache() *ttlcache.Cache[gethcommon.Hash, struct{}] {
	cache := ttlcache.New(ttlcache.WithTTL[gethcommon.Hash, struct{}](cacheTTL))
	go func() {
		defer gocommon.LogOnPanic()
		cache.Start()
	}()
	return cache
}

// New creates a WakuV2 client ready to communicate through the LibP2P network.
func New(nodeKey *ecdsa.PrivateKey, cfg *Config, logger *zap.Logger, ts timesource.Provider) (*Waku, error) {
	var err error
	if logger == nil {
		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	}

	if ts == nil {
		ts = timesource.DefaultService()
	}

	cfg = setDefaults(cfg)
	if err = cfg.Validate(logger); err != nil {
		return nil, err
	}

	logger.Info("starting wakuv2")

	ctx, cancel := context.WithCancel(context.Background())

	waku := &Waku{
		cfg:                         cfg,
		subscriptions:               make(map[types.TopicSubscription]string),
		envelopeCache:               newTTLCache(),
		msgQueue:                    make(chan *common.ReceivedMessage, messageQueueLimit),
		topicHealthStatusChan:       make(chan peermanager.TopicHealthStatus, 100),
		connectionNotifChan:         make(chan node.PeerConnection, 20),
		connStatusSubscriptions:     make(map[string]*types.ConnStatusSubscription),
		ctx:                         ctx,
		cancel:                      cancel,
		wg:                          sync.WaitGroup{},
		dnsAddressCache:             make(map[string][]dnsdisc.DiscoveredNode),
		dnsAddressCacheLock:         &sync.RWMutex{},
		dnsDiscAsyncRetrievedSignal: make(chan struct{}),
		timesource:                  ts,
		logger:                      logger,
		onlineChecker:               onlinechecker.NewDefaultOnlineChecker(false).(*onlinechecker.DefaultOnlineChecker),
		sendQueue:                   publish.NewMessageQueue(1000, cfg.UseThrottledPublish),
	}

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
		node.WithLogger(logger.Named("wakunode")),
		node.WithLogLevel(logger.Level()),
		node.WithClusterID(cfg.ClusterID),
		node.WithMaxMsgSize(1024 * 1024),
		node.WithPrometheusRegisterer(prometheus.DefaultRegisterer),
	}

	if cfg.EnableDiscV5 {
		bootnodes, err := waku.getDiscV5BootstrapNodes(waku.ctx, false)
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
	if cfg.IsLightClient() {
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

	if !cfg.IsLightClient() {
		opts = append(opts, node.WithWakuFilterFullNode(filter.WithMaxSubscribers(20)))
		opts = append(opts, node.WithLightPush(lightpush.WithRateLimiter(5, 10)))
	}

	if cfg.EnablePeerExchangeServer {
		opts = append(opts, node.WithPeerExchange(peer_exchange.WithRateLimiter(1, 1)))
	}

	waku.options = opts

	waku.logger.Info("setup the go-waku node successfully")

	return waku, nil
}

func (w *Waku) SubscribeToConnStatusChanges() (*types.ConnStatusSubscription, error) {
	w.connStatusMu.Lock()
	defer w.connStatusMu.Unlock()
	subscription := types.NewConnStatusSubscription()
	w.connStatusSubscriptions[subscription.ID] = subscription
	return subscription, nil
}

func (w *Waku) GetNodeENRString() (string, error) {
	if w.node == nil {
		return "", errors.New("node not initialized")
	}
	return w.node.ENR().String(), nil
}

func (w *Waku) getDiscV5BootstrapNodes(ctx context.Context, useOnlyDnsDiscCache bool) ([]*enode.Node, error) {
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

	for _, addrString := range w.cfg.DiscV5BootstrapNodes {
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
					// prevent w.ctx in retryDnsDiscoveryWithBackoff from set to nil when w.Stop() is called
					w.wg.Add(1)
					go func() {
						defer gocommon.LogOnPanic()
						defer w.wg.Done()
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
	applyFn := func(_ dnsdisc.DiscoveredNode, wg *sync.WaitGroup) {
		wg.Done()
	}
	for {
		err := w.dnsDiscover(ctx, addr, applyFn, false)
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

type peerAddressHandler interface {
	discoverAndConnect(addr string)
	connect(peerInfo peer.AddrInfo, node *enode.Node, origin wps.Origin)
}

func (w *Waku) discoverAndConnectPeers() {
	for _, addrString := range w.cfg.WakuNodes {
		err := handlePeerAddress(addrString, w)
		if err != nil {
			w.logger.Warn("failed to handle peer address", zap.String("addr", addrString), zap.Error(err))
		}
	}
}

// dnsDiscoveryBackoff returns the wait duration before the next DNS discovery
// retry attempt. attempt is the number of failures so far (0 = sentinel
// meaning "no retry, no wait"; 1 = first failure just happened, wait 1s
// before retry; 2 = second failure, 2s; etc). Exponential, capped at 60s.
// Capped attempt at 30 to avoid shift overflow.
func dnsDiscoveryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		return 0
	}
	if attempt > 30 {
		attempt = 30
	}
	backoff := 500 * time.Millisecond * time.Duration(1<<attempt)
	if backoff > 60*time.Second {
		return 60 * time.Second
	}
	return backoff
}

func (w *Waku) discoverAndConnect(address string) {
	fnApply := func(d dnsdisc.DiscoveredNode, wg *sync.WaitGroup) {
		defer wg.Done()
		if len(d.PeerInfo.Addrs) != 0 {
			go w.connect(d.PeerInfo, d.ENR, wps.DNSDiscovery)
		}
	}

	go func() {
		defer gocommon.LogOnPanic()
		// Retry on failure with exponential backoff: at cold start on Android,
		// the device DNS resolver isn't always ready when statusgo starts and
		// the enrtree:// lookup hits ::1:53 with "connection refused". Without
		// retry the node stays on whatever DNS-cached store peers it has until
		// the next real network event (e.g. airplane toggle) calls
		// discoverAndConnectPeers again. The loop terminates on success or
		// context cancellation (statusgo shutdown).
		for attempt := 1; ; attempt++ {
			err := w.dnsDiscover(w.ctx, address, fnApply, false)
			if err == nil {
				return
			}
			if w.ctx.Err() != nil {
				return
			}
			w.logger.Error("dns discovery failed",
				zap.String("dnsDiscURL", address),
				zap.Int("attempt", attempt),
				zap.Error(err))
			t := time.NewTimer(dnsDiscoveryBackoff(attempt))
			select {
			case <-t.C:
			case <-w.ctx.Done():
				t.Stop()
				return
			}
		}
	}()
}

func handlePeerAddress(addr string, handler peerAddressHandler) error {
	if strings.HasPrefix(addr, "enrtree://") {
		handler.discoverAndConnect(addr)
		return nil
	}

	if node, err := enode.Parse(enode.ValidSchemes, addr); err == nil {
		id, addrs, err := enr.Multiaddress(node)
		if err != nil {
			return pkgerrors.Wrap(err, "invalid enr contents")
		}

		peerInfo := peer.AddrInfo{
			ID:    id,
			Addrs: addrs,
		}
		handler.connect(peerInfo, node, wps.Static)
		return nil
	}

	if maddr, err := multiaddr.NewMultiaddr(addr); err == nil {
		peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return pkgerrors.Wrap(err, "invalid peer multiaddress")
		}

		handler.connect(*peerInfo, nil, wps.Static)
		return nil
	}

	return errors.New("unknown format of waku node address")
}

func (w *Waku) connect(peerInfo peer.AddrInfo, enr *enode.Node, origin wps.Origin) {
	defer gocommon.LogOnPanic()
	// Connection will be prunned eventually by the connection manager if needed
	// The peer connector in go-waku uses connect, so it will execute identify as part of its
	w.node.AddDiscoveredPeer(peerInfo.ID, peerInfo.Addrs, origin, w.cfg.DefaultShardedPubsubTopics, enr, true)
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
	sub := w.PauseBroadcaster.Subscribe()
	defer sub.Unsubscribe()
	paused := <-sub.C()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Debug("Peer exchange loop stopped")
			return
		case pausedState, ok := <-sub.C():
			if !ok {
				return
			}
			paused = pausedState
		case <-ticker.C:
			if paused {
				peerCount := len(w.node.Host().Network().Peers())
				if peerCount >= pausedModePeerExchangeMinPeers {
					// In paused mode, avoid proactive peer exchange unless peer count
					// drops below a minimal threshold needed to keep connectivity healthy.
					continue
				}
			}
			w.logger.Debug("Running peer exchange loop")
			err := w.node.PeerExchange().Request(
				w.ctx,
				w.cfg.DiscoveryLimit,
				peer_exchange.WithAutomaticPeerSelection(),
				peer_exchange.FilterByShard(int(w.defaultShardInfo.ClusterID), int(w.defaultShardInfo.ShardIDs[0])),
			)
			if err != nil {
				w.logger.Error("could not request peers via peer exchange", zap.Error(err))
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

func (w *Waku) subscribeToPubsubTopicWithWakuRelay(topic string) error {
	if w.cfg.IsLightClient() {
		return errors.New("only available for full nodes")
	}

	topic = w.GetPubsubTopic(topic)

	if w.node.Relay().IsSubscribed(topic) {
		return nil
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

func (w *Waku) SendEnvelopeEvent(event common.EnvelopeEvent) int {
	return w.envelopeFeed.Send(event)
}

// SubscribeEnvelopeEvents subscribes to envelopes feed.
// In order to prevent blocking waku producers events must be amply buffered.
func (w *Waku) subscribeEnvelopeEvents(events chan<- common.EnvelopeEvent) event.Subscription {
	return w.envelopeFeed.Subscribe(events)
}

func (w *Waku) SubscribeEnvelopeEvents(eventsProxy chan<- types.EnvelopeEvent) types.Subscription {
	events := make(chan common.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	go func() {
		defer gocommon.LogOnPanic()
		for e := range events {
			eventsProxy <- *NewWakuV2EnvelopeEventWrapper(&e)
		}
	}()

	return NewGethSubscriptionWrapper(w.subscribeEnvelopeEvents(events))
}

// newReceivedMessage builds the neutral, transport-agnostic view of a received
// envelope that travels on the envelope feed (EventEnvelopeAvailable.Data) for
// the transport to decode and route.
func newReceivedMessage(e *common.ReceivedMessage) *types.ReceivedMessage {
	msg := e.Envelope.Message()
	return &types.ReceivedMessage{
		Hash:         e.Hash().Bytes(),
		ContentTopic: msg.ContentTopic,
		Payload:      msg.Payload,
		Ephemeral:    msg.GetEphemeral(),
		Meta:         msg.Meta,
		PubsubTopic:  e.PubsubTopic,
		Version:      msg.GetVersion(),
		Timestamp:    msg.GetTimestamp(),
	}
}

// Subscribe registers a wire subscription for each given (pubsubTopic,
// contentTopic) pair. Subscribing an already-subscribed pair is a no-op.
func (w *Waku) Subscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	w.subscriptionsMu.Lock()
	defer w.subscriptionsMu.Unlock()

	for _, contentTopic := range contentTopics {
		sub := types.TopicSubscription{PubsubTopic: pubsubTopic, ContentTopic: contentTopic}
		if _, ok := w.subscriptions[sub]; ok {
			continue
		}

		id, err := common.GenerateRandomID()
		if err != nil {
			return err
		}
		if w.cfg.IsLightClient() && w.filterManager != nil {
			topics := [][]byte{types.TopicTypeToByteArray(contentTopic)}
			cf := protocol.NewContentFilter(w.GetPubsubTopic(pubsubTopic), common.NewTopicSetFromBytes(topics).ContentTopics()...)
			w.filterManager.SubscribeFilter(id, cf)
		}
		w.subscriptions[sub] = id
	}

	return nil
}

// Unsubscribe removes the wire subscriptions for the given pairs.
// Unsubscribing a pair that is not subscribed is a no-op.
func (w *Waku) Unsubscribe(ctx context.Context, pubsubTopic string, contentTopics []types.TopicType) error {
	w.subscriptionsMu.Lock()
	defer w.subscriptionsMu.Unlock()

	for _, contentTopic := range contentTopics {
		sub := types.TopicSubscription{PubsubTopic: pubsubTopic, ContentTopic: contentTopic}
		id, ok := w.subscriptions[sub]
		if !ok {
			continue
		}

		w.logger.Debug("cleaning up wire subscription", zap.String("id", id))
		if w.cfg.IsLightClient() && w.filterManager != nil {
			w.filterManager.UnsubscribeFilter(id)
		}
		delete(w.subscriptions, sub)
	}

	return nil
}

func (w *Waku) SkipPublishToTopic(value bool) {
	w.cfg.SkipPublishToTopic = value
}

func (w *Waku) ConfirmMessageDelivered(hashes []gethcommon.Hash) {
	w.messageSender.MessagesDelivered(hashes)
	if w.metricsHandler != nil {
		w.metricsHandler.PushMessageDeliveryConfirmed()
	}
}

// OnNewEnvelope is an interface from Waku FilterManager API that gets invoked when any new message is received by Filter.
func (w *Waku) OnNewEnvelope(env *protocol.Envelope) error {
	return w.OnNewEnvelopes(env, common.RelayedMessageType, false)
}

// Starts the background data propagation thread of the Waku protocol.
func (w *Waku) Start() error {
	if w.cancel == nil {
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

	selector := newStoreSelector()
	pager := newStorePager(wakuStoreRequestor{store: w.node.Store()}, NewHistoryProcessorWrapper(w), w.logger)
	w.storeClient = NewStoreClient(selector, pager, w.GetPubsubTopic, w.logger)
	// The store nodes belong to the fleet, so the waku node resolves them itself
	// instead of having a higher layer push them in.
	w.storeClient.SetStorenodes(w.fleetStorenodes())

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
		sub := w.PauseBroadcaster.Subscribe()
		defer sub.Unsubscribe()
		paused := <-sub.C()
		var tickerC <-chan time.Time
		if !paused {
			tickerC = ticker.C
		}
		for {
			select {
			case <-w.ctx.Done():
				return
			case pausedState, ok := <-sub.C():
				if !ok {
					return
				}
				paused = pausedState
				if paused {
					tickerC = nil
				} else {
					tickerC = ticker.C
				}
			case <-tickerC:
				w.checkForConnectionChanges()
			case <-w.topicHealthStatusChan:
				// TODO: https://github.com/status-im/status-go/issues/4628
			case <-w.connectionNotifChan:
				if !paused {
					w.checkForConnectionChanges()
				}
			}
		}
	}()

	if w.cfg.MetricsEnabled {
		w.wg.Add(1)
		go func() {
			defer gocommon.LogOnPanic()
			defer w.wg.Done()
			peerTelemetryTickerInterval := 10 * time.Second
			peerTelemetryTicker := time.NewTicker(peerTelemetryTickerInterval)
			defer peerTelemetryTicker.Stop()
			sub := w.PauseBroadcaster.Subscribe()
			defer sub.Unsubscribe()
			paused := <-sub.C()
			var telemetryTickerC <-chan time.Time
			if !paused {
				telemetryTickerC = peerTelemetryTicker.C
			}

			dialErrSub, err := w.node.Host().EventBus().Subscribe(new(utils.DialError))
			if err != nil {
				w.logger.Error("failed to subscribe to dial errors", zap.Error(err))
				return
			}
			defer dialErrSub.Close()

			messageSentSub, err := w.node.Host().EventBus().Subscribe(new(publish.MessageSent))
			if err != nil {
				w.logger.Error("failed to subscribe to message sent events", zap.Error(err))
				return
			}

			publishMethod := "relay"
			if w.cfg.IsLightClient() {
				publishMethod = "lightpush"
			}

			for {
				select {
				case <-w.ctx.Done():
					return
				case pausedState, ok := <-sub.C():
					if !ok {
						return
					}
					paused = pausedState
					if paused {
						telemetryTickerC = nil
					} else {
						telemetryTickerC = peerTelemetryTicker.C
					}
				case <-telemetryTickerC:
					w.reportPeerMetrics()
				case dialErr := <-dialErrSub.Out():
					errors := common.ParseDialErrors(dialErr.(utils.DialError).Err.Error())
					for _, dialError := range errors {
						w.metricsHandler.PushDialFailure(common.DialError{ErrType: dialError.ErrType, ErrMsg: dialError.ErrMsg, Protocols: dialError.Protocols})
					}
				case messageSent := <-messageSentSub.Out():
					w.metricsHandler.PushSentMessageTotal(messageSent.(publish.MessageSent).Size, publishMethod)
				}
			}
		}()
	}

	w.wg.Add(1)
	go w.runPeerExchangeLoop()

	if w.cfg.IsLightClient() {
		// Create FilterManager that will main peer connectivity
		// for installed filters
		w.filterManager = filterapi.NewFilterManager(
			w.ctx,
			w.logger,
			w.cfg.MaxPeersForFilter,
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

	w.MarkStarted()
	return nil
}

func (w *Waku) checkForConnectionChanges() {

	isOnline := len(w.node.Host().Network().Peers()) > 0

	w.connStatusMu.Lock()

	latestConnStatus := types.ConnStatus{
		IsOnline: isOnline,
	}

	w.logger.Debug("connection status", zap.Bool("isOnline", isOnline))
	for k, subs := range w.connStatusSubscriptions {
		if !subs.Send(latestConnStatus) {
			delete(w.connStatusSubscriptions, k)
		}
	}

	w.connStatusMu.Unlock()

	// Build the proposed next state. checkForConnectionChanges never originates
	// a Type change (the comment below acknowledges this), and it has no OS
	// visibility to learn about Expensive — both flow through the explicit
	// mobile.ConnectionChange path.
	prevState, _ := w.snapshotState()
	next := connection.State{
		Type:    prevState.Type, //setting state type as previous one since there won't be a change here
		Offline: !latestConnStatus.IsOnline,
	}
	if w.shouldFireConnectionChanged(next) {
		w.ConnectionChanged(next)
	}
}

func (w *Waku) reportPeerMetrics() {
	if w.metricsHandler != nil {
		connFailures := FormatPeerConnFailures(w.node)
		w.metricsHandler.PushPeerConnFailures(connFailures)

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
		w.metricsHandler.PushPeerCountByShard(peerCountByShard)
		w.metricsHandler.PushPeerCountByOrigin(peerCountByOrigin)
	}
}

func (w *Waku) startMessageSender() error {
	publishMethod := publish.Relay
	if w.cfg.IsLightClient() {
		publishMethod = publish.LightPush
	}

	sender, err := publish.NewMessageSender(publishMethod, publish.NewDefaultPublisher(w.node.Lightpush(), w.node.Relay()), nil, w.logger)
	if err != nil {
		w.logger.Error("failed to create message sender", zap.Error(err))
		return err
	}

	if w.cfg.MetricsEnabled {
		sender.WithMessageSentEmitter(w.node.Host())
	}

	if w.cfg.EnableStoreConfirmationForMessagesSent {
		msgStoredChan := make(chan gethcommon.Hash, 1000)
		msgExpiredChan := make(chan gethcommon.Hash, 1000)
		// The store node is selected on demand by the StoreClient (the go-waku
		// storenode cycle is gone); MessageSentCheck queries it by hash to confirm
		// that published messages propagated.
		messageSentCheck := NewMessageSentCheck(w.ctx, publish.NewDefaultStorenodeMessageVerifier(w.node.Store()), storeClientPeerProvider{w.storeClient}, w.node.Timesource(), msgStoredChan, msgExpiredChan, w.logger)
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
					if w.metricsHandler != nil {
						w.metricsHandler.PushMessageCheckSuccess()
					}
				case hash := <-msgExpiredChan:
					w.SendEnvelopeEvent(common.EnvelopeEvent{
						Hash:  hash,
						Event: common.EventEnvelopeExpired,
					})
					if w.metricsHandler != nil {
						w.metricsHandler.PushMessageCheckFailure()
					}
				}
			}
		}()
	}

	if !w.cfg.UseThrottledPublish || testing.Testing() {
		// To avoid delaying the tests, or for when we dont want to rate limit, we set up an infinite rate limiter,
		// basically disabling the rate limit functionality
		limiter := publish.NewDefaultRateLimiter(rate.Inf, 1)
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

func (w *Waku) setupRelaySubscriptions() error {
	if w.cfg.IsLightClient() {
		return nil
	}

	err := w.subscribeToPubsubTopicWithWakuRelay(w.cfg.DefaultShardPubsubTopic)
	if err != nil {
		return err
	}

	// Community control messages arrive on the non-protected pubsub topic.
	// TODO remove once fully migrated to Global Community Control and Content Topic
	// https://github.com/status-im/status-go/issues/6384
	err = w.subscribeToPubsubTopicWithWakuRelay(DefaultNonProtectedPubsubTopic())
	if err != nil {
		return err
	}

	return nil
}

// Stops the background data propagation thread of the Waku protocol.
func (w *Waku) Stop() error {
	// never started || already stopped
	if w.node == nil || w.cancel == nil {
		return nil
	}

	w.cancel()
	defer func() {
		w.cancel = nil
	}()

	w.envelopeCache.Stop()

	w.node.Stop()

	close(w.goingOnline)
	w.wg.Wait()

	w.MarkStopped()

	_ = w.logger.Sync()

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

	if w.metricsHandler != nil {
		if msgType == common.MissingMessageType {
			commonEnv := common.NewWakuEnvelope(envelope.Message(), envelope.PubsubTopic(), envelope.Hash())
			w.metricsHandler.PushMissedMessage(commonEnv)
		}
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
		logger.Debug("invalid envelope received", zap.Error(err))
		trouble = true
	}

	common.EnvelopesValidatedCounter.With(prometheus.Labels{"pubsubTopic": envelope.PubsubTopic(), "type": msgType}).Inc()

	if trouble {
		return errors.New("received invalid envelope")
	}

	return nil
}

// addEnvelope adds an envelope to the envelope map, used for sending
func (w *Waku) addEnvelope(envelope *common.ReceivedMessage) {
	w.poolMu.Lock()
	w.envelopeCache.Set(envelope.Hash(), struct{}{}, ttlcache.DefaultTTL)
	w.poolMu.Unlock()
}

func (w *Waku) add(recvMessage *common.ReceivedMessage, processImmediately bool) (bool, error) {
	common.EnvelopesReceivedCounter.Inc()

	w.poolMu.Lock()
	alreadyCached := w.envelopeCache.Has(recvMessage.Hash())
	w.poolMu.Unlock()

	logger := w.logger.With(zap.String("envelopeHash", recvMessage.Hash().Hex()))

	if alreadyCached {
		logger.Debug("w envelope already cached")
		common.EnvelopesCachedCounter.WithLabelValues("hit").Inc()
	} else {
		w.addEnvelope(recvMessage)
		logger.Debug("cached w envelope")
		common.EnvelopesCachedCounter.WithLabelValues("miss").Inc()
		common.EnvelopesSizeMeter.Observe(float64(len(recvMessage.Envelope.Message().Payload)))
	}

	// De-duplication is owned by the transport's persistent processed-message
	// cache (status-im/status-go#7464), so every received envelope is forwarded
	// regardless of the in-memory cache above.
	if processImmediately {
		logger.Debug("immediately processing envelope")
		w.processMessage(recvMessage)
	} else {
		logger.Debug("posting event")
		w.postEvent(recvMessage) // notify the local node about the new message
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
	// Decoding, content-topic routing and filter matching all happen in the
	// transport now (status-im/status-go#7464). The adapter just forwards every
	// received envelope on the envelope feed, carrying the neutral
	// ReceivedMessage as the EventEnvelopeAvailable payload; the transport
	// decodes and routes it. De-duplication is owned by the transport's
	// persistent processed-message cache, so nothing is marked processed here.
	w.envelopeFeed.Send(common.EnvelopeEvent{
		Topic: e.ContentTopic,
		Hash:  e.Hash(),
		Event: common.EventEnvelopeAvailable,
		Data:  newReceivedMessage(e),
	})
}

// Peers is retained only for the Python functional tests (see tests-functional);
// it is not used by status-app.
func (w *Waku) Peers() types.PeerStats {
	return FormatPeerStats(w.node)
}

// isNetworkSwitchEvent reports whether next represents a genuine wifi <-> cellular
// switch (while remaining online) that warrants tearing down peers
func isNetworkSwitchEvent(prev, next connection.State, prevInitialized bool) bool {
	if !prevInitialized {
		return false
	}
	if prev.Offline || next.Offline {
		return false
	}
	return prev.Type != next.Type
}

func (w *Waku) handleNetworkChangeFromApp(state connection.State) {
	prevState, prevInitialized := w.snapshotState()
	//If connection state is reported by something other than peerCount becoming 0 e.g from mobile app, disconnect all peers
	if (state.Offline && len(w.node.Host().Network().Peers()) > 0) ||
		isNetworkSwitchEvent(prevState, state, prevInitialized) {
		w.logger.Info("connection switched or offline detected via mobile, disconnecting all peers")
		w.node.DisconnectAllPeers()
		if w.cfg.IsLightClient() {
			w.filterManager.NetworkChange()
		}
	}
}

func (w *Waku) isGoingOnline(state connection.State) bool {
	return !state.Offline && !w.onlineChecker.IsOnline()
}

// shouldFireConnectionChanged reports whether next represents a change in
// connectivity. The first observation (no state recorded yet) always fires so
// downstream consumers like the LightClient FilterManager get initialized.
func (w *Waku) shouldFireConnectionChanged(next connection.State) bool {
	prev, prevInitialized := w.snapshotState()
	if !prevInitialized {
		return true
	}
	return prev.Offline != next.Offline || prev.Type != next.Type
}

// snapshotState returns a copy of the current connection state and the
// initialization flag under the stateMu read lock
func (w *Waku) snapshotState() (connection.State, bool) {
	w.stateMu.RLock()
	defer w.stateMu.RUnlock()
	return w.state, w.stateInitialized
}

func (w *Waku) ConnectionChanged(state connection.State) {
	if w.isGoingOnline(state) {
		w.discoverAndConnectPeers()
	}
	isOnline := !state.Offline

	if w.cfg.IsLightClient() {
		//TODO: Update this as per  https://github.com/waku-org/go-waku/issues/1114
		go func() {
			defer gocommon.LogOnPanic()
			w.filterManager.OnConnectionStatusChange("", isOnline, byte(state.Type))
		}()
		w.handleNetworkChangeFromApp(state)
	} else {
		// for lightClient state update and onlineChange is handled in filterManager.
		if w.isGoingOnline(state) {
			select {
			case w.goingOnline <- struct{}{}:
			default:
				w.logger.Warn("could not write on connection changed channel")
			}
		}
	}
	// update state
	w.onlineChecker.SetOnline(isOnline)
	w.stateMu.Lock()
	w.state = state
	w.stateInitialized = true
	w.stateMu.Unlock()
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
	sub := w.PauseBroadcaster.Subscribe()
	defer sub.Unsubscribe()
	paused := <-sub.C()
	var tickerC <-chan time.Time
	if !paused {
		tickerC = ticker.C
	}
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
		case pausedState, ok := <-sub.C():
			if !ok {
				return
			}
			paused = pausedState
			if paused {
				tickerC = nil
			} else {
				tickerC = ticker.C
			}
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
		case <-tickerC:
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
	bootnodes, err := w.getDiscV5BootstrapNodes(ctx, useOnlyDNSDiscCache)
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

func (w *Waku) timestamp() int64 {
	return w.timesource.Now().UnixNano()
}

func (w *Waku) Clean() error {
	w.msgQueue = make(chan *common.ReceivedMessage, messageQueueLimit)
	return nil
}

func (w *Waku) PeerID() peer.ID {
	return w.node.Host().ID()
}

func (w *Waku) Peerstore() peerstore.Peerstore {
	return w.node.Host().Peerstore()
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

// fleetStorenodes resolves the configured fleet's store nodes into dialable peer
// addresses. Nodes whose addressing info can't be resolved are skipped.
func (w *Waku) fleetStorenodes() []peer.AddrInfo {
	storeNodes := fleets.StoreNodes(w.cfg.Fleet)
	addrInfos := make([]peer.AddrInfo, 0, len(storeNodes))
	for _, node := range storeNodes {
		info, err := node.PeerInfo()
		if err != nil {
			w.logger.Warn("skipping storenode with unresolvable peer info",
				zap.String("id", node.ID), zap.String("name", node.Name), zap.Error(err))
			continue
		}
		addrInfos = append(addrInfos, info)
	}
	if len(addrInfos) == 0 && len(storeNodes) > 0 {
		w.logger.Warn("no usable storenodes after resolving peer info; history queries will fail",
			zap.Int("configured", len(storeNodes)))
	}
	return addrInfos
}

// StoreQuery retrieves historic messages for a single batch via the StoreClient
// facade, which selects the store node itself (no peer argument).
func (w *Waku) StoreQuery(
	ctx context.Context,
	batch types.MailserverBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return w.storeClient.Query(ctx, batch, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (w *Waku) Metrics() string {
	return ""
}
