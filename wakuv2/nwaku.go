//go:build use_nwaku
// +build use_nwaku

package wakuv2

/*
	#cgo LDFLAGS: -L../third_party/nwaku/build/ -lnegentropy -lwaku
	#cgo LDFLAGS: -L../third_party/nwaku -Wl,-rpath,../third_party/nwaku/build/

	#include "../third_party/nwaku/library/libwaku.h"
	#include <stdio.h>
	#include <stdlib.h>

	extern void globalEventCallback(int ret, char* msg, size_t len, void* userData);

	typedef struct {
		int ret;
		char* msg;
		size_t len;
	} Resp;

	static void* allocResp() {
		return calloc(1, sizeof(Resp));
	}

	static void freeResp(void* resp) {
		if (resp != NULL) {
			free(resp);
		}
	}

	static char* getMyCharPtr(void* resp) {
		if (resp == NULL) {
			return NULL;
		}
		Resp* m = (Resp*) resp;
		return m->msg;
	}

	static size_t getMyCharLen(void* resp) {
		if (resp == NULL) {
			return 0;
		}
		Resp* m = (Resp*) resp;
		return m->len;
	}

	static int getRet(void* resp) {
		if (resp == NULL) {
			return 0;
		}
		Resp* m = (Resp*) resp;
		return m->ret;
	}

	// resp must be set != NULL in case interest on retrieving data from the callback
	static void callback(int ret, char* msg, size_t len, void* resp) {
		if (resp != NULL) {
			Resp* m = (Resp*) resp;
			m->ret = ret;
			m->msg = msg;
			m->len = len;
		}
	}

	#define WAKU_CALL(call)                                                        \
	do {                                                                           \
		int ret = call;                                                              \
		if (ret != 0) {                                                              \
			printf("Failed the call to: %s. Returned code: %d\n", #call, ret);         \
			exit(1);                                                                   \
		}                                                                            \
	} while (0)

	static void* cGoWakuNew(const char* configJson, void* resp) {
		// We pass NULL because we are not interested in retrieving data from this callback
		void* ret = waku_new(configJson, (WakuCallBack) callback, resp);
		return ret;
	}

	static void cGoWakuStart(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_start(wakuCtx, (WakuCallBack) callback, resp));
	}

	static void cGoWakuStop(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_stop(wakuCtx, (WakuCallBack) callback, resp));
	}

	static void cGoWakuDestroy(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_destroy(wakuCtx, (WakuCallBack) callback, resp));
	}

	static void cGoWakuStartDiscV5(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_start_discv5(wakuCtx, (WakuCallBack) callback, resp));
	}

	static void cGoWakuStopDiscV5(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_stop_discv5(wakuCtx, (WakuCallBack) callback, resp));
	}

	static void cGoWakuVersion(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_version(wakuCtx, (WakuCallBack) callback, resp));
	}

	static void cGoWakuSetEventCallback(void* wakuCtx) {
		// The 'globalEventCallback' Go function is shared amongst all possible Waku instances.

		// Given that the 'globalEventCallback' is shared, we pass again the
		// wakuCtx instance but in this case is needed to pick up the correct method
		// that will handle the event.

		// In other words, for every call the libwaku makes to globalEventCallback,
		// the 'userData' parameter will bring the context of the node that registered
		// that globalEventCallback.

		// This technique is needed because cgo only allows to export Go functions and not methods.

		waku_set_event_callback(wakuCtx, (WakuCallBack) globalEventCallback, wakuCtx);
	}

	static void cGoWakuContentTopic(void* wakuCtx,
							char* appName,
							int appVersion,
							char* contentTopicName,
							char* encoding,
							void* resp) {

		WAKU_CALL( waku_content_topic(wakuCtx,
							appName,
							appVersion,
							contentTopicName,
							encoding,
							(WakuCallBack) callback,
							resp) );
	}

	static void cGoWakuPubsubTopic(void* wakuCtx, char* topicName, void* resp) {
		WAKU_CALL( waku_pubsub_topic(wakuCtx, topicName, (WakuCallBack) callback, resp) );
	}

	static void cGoWakuDefaultPubsubTopic(void* wakuCtx, void* resp) {
		WAKU_CALL (waku_default_pubsub_topic(wakuCtx, (WakuCallBack) callback, resp));
	}

	static void cGoWakuRelayPublish(void* wakuCtx,
                       const char* pubSubTopic,
                       const char* jsonWakuMessage,
                       int timeoutMs,
					   void* resp) {

		WAKU_CALL (waku_relay_publish(wakuCtx,
                       pubSubTopic,
                       jsonWakuMessage,
                       timeoutMs,
                       (WakuCallBack) callback,
                       resp));
	}

	static void cGoWakuRelaySubscribe(void* wakuCtx, char* pubSubTopic, void* resp) {
		WAKU_CALL ( waku_relay_subscribe(wakuCtx,
							pubSubTopic,
							(WakuCallBack) callback,
							resp) );
	}

	static void cGoWakuRelayUnsubscribe(void* wakuCtx, char* pubSubTopic, void* resp) {

		WAKU_CALL ( waku_relay_unsubscribe(wakuCtx,
							pubSubTopic,
							(WakuCallBack) callback,
							resp) );
	}

	static void cGoWakuConnect(void* wakuCtx, char* peerMultiAddr, int timeoutMs, void* resp) {
		WAKU_CALL( waku_connect(wakuCtx,
						peerMultiAddr,
						timeoutMs,
						(WakuCallBack) callback,
						resp) );
	}

	static void cGoWakuDialPeerById(void* wakuCtx,
									char* peerId,
									char* protocol,
									int timeoutMs,
									void* resp) {

		WAKU_CALL( waku_dial_peer_by_id(wakuCtx,
						peerId,
						protocol,
						timeoutMs,
						(WakuCallBack) callback,
						resp) );
	}

	static void cGoWakuDisconnectPeerById(void* wakuCtx, char* peerId, void* resp) {
		WAKU_CALL( waku_disconnect_peer_by_id(wakuCtx,
						peerId,
						(WakuCallBack) callback,
						resp) );
	}

	static void cGoWakuListenAddresses(void* wakuCtx, void* resp) {
		WAKU_CALL (waku_listen_addresses(wakuCtx, (WakuCallBack) callback, resp) );
	}

	static void cGoWakuGetMyENR(void* ctx, void* resp) {
		WAKU_CALL (waku_get_my_enr(ctx, (WakuCallBack) callback, resp) );
	}

	static void cGoWakuPingPeer(void* ctx, char* peerAddr, int timeoutMs, void* resp) {
		WAKU_CALL (waku_ping_peer(ctx, peerAddr, timeoutMs, (WakuCallBack) callback, resp) );
	}

	static void cGoWakuListPeersInMesh(void* ctx, char* pubSubTopic, void* resp) {
		WAKU_CALL (waku_relay_get_num_peers_in_mesh(ctx, pubSubTopic, (WakuCallBack) callback, resp) );
	}

	static void cGoWakuGetNumConnectedPeers(void* ctx, char* pubSubTopic, void* resp) {
		WAKU_CALL (waku_relay_get_num_connected_peers(ctx, pubSubTopic, (WakuCallBack) callback, resp) );
	}

	static void cGoWakuLightpushPublish(void* wakuCtx,
					const char* pubSubTopic,
					const char* jsonWakuMessage,
					void* resp) {

		WAKU_CALL (waku_lightpush_publish(wakuCtx,
						pubSubTopic,
						jsonWakuMessage,
						(WakuCallBack) callback,
						resp));
	}

	static void cGoWakuStoreQuery(void* wakuCtx,
					const char* jsonQuery,
					const char* peerAddr,
					int timeoutMs,
					void* resp) {

		WAKU_CALL (waku_store_query(wakuCtx,
									jsonQuery,
									peerAddr,
									timeoutMs,
									(WakuCallBack) callback,
									resp));
	}

	static void cGoWakuPeerExchangeQuery(void* wakuCtx,
								uint64_t numPeers,
								void* resp) {

		WAKU_CALL (waku_peer_exchange_request(wakuCtx,
									numPeers,
									(WakuCallBack) callback,
									resp));
	}

	static void cGoWakuGetPeerIdsByProtocol(void* wakuCtx,
									 const char* protocol,
									 void* resp) {

		WAKU_CALL (waku_get_peerids_by_protocol(wakuCtx,
									protocol,
									(WakuCallBack) callback,
									resp));
	}

*/
import "C"

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/golang/protobuf/proto"
	"github.com/jellydator/ttlcache/v3"
	"github.com/libp2p/go-libp2p/core/peer"
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

	"github.com/libp2p/go-libp2p/core/metrics"

	commonapi "github.com/waku-org/go-waku/waku/v2/api/common"

	filterapi "github.com/waku-org/go-waku/waku/v2/api/filter"
	"github.com/waku-org/go-waku/waku/v2/api/history"
	"github.com/waku-org/go-waku/waku/v2/api/missing"
	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"

	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
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

type WakuMessageHash = string
type WakuPubsubTopic = string
type WakuContentTopic = string

type WakuConfig struct {
	Host                 string   `json:"host,omitempty"`
	Port                 int      `json:"port,omitempty"`
	NodeKey              string   `json:"key,omitempty"`
	EnableRelay          bool     `json:"relay"`
	LogLevel             string   `json:"logLevel"`
	DnsDiscovery         bool     `json:"dnsDiscovery,omitempty"`
	DnsDiscoveryUrl      string   `json:"dnsDiscoveryUrl,omitempty"`
	MaxMessageSize       string   `json:"maxMessageSize,omitempty"`
	Staticnodes          []string `json:"staticnodes,omitempty"`
	Discv5BootstrapNodes []string `json:"discv5BootstrapNodes,omitempty"`
	Discv5Discovery      bool     `json:"discv5Discovery,omitempty"`
	ClusterID            uint16   `json:"clusterId,omitempty"`
	Shards               []uint16 `json:"shards,omitempty"`
}

// Waku represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Waku struct {
	wakuCtx unsafe.Pointer

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
	wakuCfg *WakuConfig

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

	StorenodeCycle   *history.StorenodeCycle
	HistoryRetriever *history.HistoryRetriever

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
	go cache.Start()
	return cache
}

// New creates a WakuV2 client ready to communicate through the LibP2P network.
func New(nodeKey *ecdsa.PrivateKey, fleet string, cfg *Config, nwakuCfg *WakuConfig, logger *zap.Logger, appDB *sql.DB, ts *timesource.NTPTimeSource, onHistoricMessagesRequestFailed func([]byte, peer.ID, error), onPeerStats func(types.ConnStatus)) (*Waku, error) {
	// Lock the main goroutine to its current OS thread
	runtime.LockOSThread()

	WakuSetup() // This should only be called once in the whole app's life

	node, err := wakuNew(nodeKey,
		fleet,
		cfg,
		nwakuCfg,
		logger, appDB, ts, onHistoricMessagesRequestFailed,
		onPeerStats)
	if err != nil {
		return nil, err
	}

	defaultPubsubTopic, err := node.WakuDefaultPubsubTopic()
	if err != nil {
		return nil, err
	}

	err = node.WakuRelaySubscribe(defaultPubsubTopic)
	if err != nil {
		return nil, err
	}

	node.WakuSetEventCallback()

	return node, nil

	// TODO-nwaku
	/*
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

		return waku, nil*/
}

func (w *Waku) SubscribeToConnStatusChanges() *types.ConnStatusSubscription {
	w.connStatusMu.Lock()
	defer w.connStatusMu.Unlock()
	subscription := types.NewConnStatusSubscription()
	w.connStatusSubscriptions[subscription.ID] = subscription
	return subscription
}

/* TODO-nwaku
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
} */

func (w *Waku) connect(peerInfo peer.AddrInfo, enr *enode.Node, origin wps.Origin) {
	defer gocommon.LogOnPanic()
	// Connection will be prunned eventually by the connection manager if needed
	// The peer connector in go-waku uses Connect, so it will execute identify as part of its
	addr := peerInfo.Addrs[0]
	w.WakuConnect(addr.String(), 1000)
}

/* TODO-nwaku
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
*/

func (w *Waku) GetPubsubTopic(topic string) string {
	if topic == "" {
		topic = w.cfg.DefaultShardPubsubTopic
	}

	return topic
}

/* TODO-nwaku
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
*/

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

// OnNewEnvelope is an interface from Waku FilterManager API that gets invoked when any new message is received by Filter.
func (w *Waku) OnNewEnvelope(env *protocol.Envelope) error {
	return w.OnNewEnvelopes(env, common.RelayedMessageType, false)
}

// Start implements node.Service, starting the background data propagation thread
// of the Waku protocol.
func (w *Waku) Start() error {
	err := w.WakuStart()
	if err != nil {
		return fmt.Errorf("failed to start nwaku node: %v", err)
	}

	/* TODO-nwaku
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
	*/
	w.StorenodeCycle = history.NewStorenodeCycle(w.logger, newPinger(w.wakuCtx))

	w.HistoryRetriever = history.NewHistoryRetriever(newStorenodeRequestor(w.wakuCtx, w.logger), NewHistoryProcessorWrapper(w), w.logger)
	w.StorenodeCycle.Start(w.ctx)

	w.logger.Info("WakuV2 PeerID", zap.Stringer("id", w.PeerID()))

	/* TODO-nwaku
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
	*/

	if w.cfg.EnableMissingMessageVerification {
		w.missingMsgVerifier = missing.NewMissingMessageVerifier(
			newStorenodeRequestor(w.wakuCtx, w.logger),
			w,
			w.timesource,
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

	/* TODO: nwaku
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
	*/

	w.wg.Add(1)

	go w.broadcast()

	go w.sendQueue.Start(w.ctx)

	err = w.startMessageSender()
	if err != nil {
		return err
	}

	/* TODO-nwaku
	// we should wait `seedBootnodesForDiscV5` shutdown smoothly before set w.ctx to nil within `w.Stop()`
	w.wg.Add(1)
	go w.seedBootnodesForDiscV5()
	*/

	return nil
}

func (w *Waku) checkForConnectionChanges() {

	/* TODO-nwaku
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
	}) */
}

/* TODO: nwaku
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
*/

func (w *Waku) startMessageSender() error {
	publishMethod := publish.Relay
	/* TODO-nwaku
	if w.cfg.LightClient {
		publishMethod = publish.LightPush
	}*/

	sender, err := publish.NewMessageSender(publishMethod, newPublisher(w.wakuCtx), w.logger)
	if err != nil {
		w.logger.Error("failed to create message sender", zap.Error(err))
		return err
	}

	if w.cfg.EnableStoreConfirmationForMessagesSent {
		msgStoredChan := make(chan gethcommon.Hash, 1000)
		msgExpiredChan := make(chan gethcommon.Hash, 1000)
		messageSentCheck := publish.NewMessageSentCheck(w.ctx, newStorenodeMessageVerifier(w.wakuCtx), w.StorenodeCycle, w.timesource, msgStoredChan, msgExpiredChan, w.logger)
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

/* TODO-nwaku
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
} */

// Stop implements node.Service, stopping the background data propagation thread
// of the Waku protocol.
func (w *Waku) Stop() error {
	w.cancel()

	w.envelopeCache.Stop()

	err := w.WakuStop()
	if err != nil {
		return err
	}

	/* TODO-nwaku
	if w.protectedTopicStore != nil {
		err := w.protectedTopicStore.Close()
		if err != nil {
			return err
		}
	}

	close(w.goingOnline)*/

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

// TODO-nwaku
func (w *Waku) PeerCount() int {
	return 0
	// return w.node.PeerCount()
}

// TODO-nwaku
func (w *Waku) Peers() types.PeerStats {
	return nil
	// return FormatPeerStats(w.node)
}

/* TODO-nwaku
func (w *Waku) RelayPeersByTopic(topic string) (*types.PeerList, error) {
	if w.cfg.LightClient {
		return nil, errors.New("only available in relay mode")
	}

	return &types.PeerList{
		FullMeshPeers: w.node.Relay().PubSub().MeshPeers(topic),
		AllPeers:      w.node.Relay().PubSub().ListPeers(topic),
	}, nil
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
} */

func (w *Waku) handleNetworkChangeFromApp(state connection.State) {
	// TODO-nwaku
	/*
		//If connection state is reported by something other than peerCount becoming 0 e.g from mobile app, disconnect all peers
		if (state.Offline && len(w.node.Host().Network().Peers()) > 0) ||
			(w.state.Type != state.Type && !w.state.Offline && !state.Offline) { // network switched between wifi and cellular
			w.logger.Info("connection switched or offline detected via mobile, disconnecting all peers")
			w.node.DisconnectAllPeers()
			if w.cfg.LightClient {
				w.filterManager.NetworkChange()
			}
		}
	*/
}

func (w *Waku) ConnectionChanged(state connection.State) {
	/* TODO-nwaku
	isOnline := !state.Offline
	if w.cfg.LightClient {
		//TODO: Update this as per  https://github.com/waku-org/go-waku/issues/1114
		go w.filterManager.OnConnectionStatusChange("", isOnline)
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
	*/
}

/* TODO-nwaku
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
				w.logger.Info("can't query bootnodes", zap.Int("peer-count", len(w.node.Host().Network().Peers())), zap.Int64("lastTry", lastTry), zap.Int64("now", now()), zap.Int64("backoff", bootnodesQueryBackoffMs*int64(math.Exp2(float64(retries)))), zap.Int("retries", retries))
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
*/

func (w *Waku) AddStorePeer(address multiaddr.Multiaddr) (peer.ID, error) {
	// TODO-nwaku
	/*
		peerID, err := w.node.AddPeer(address, wps.Static, w.cfg.DefaultShardedPubsubTopics, store.StoreQueryID_v300)
		if err != nil {
			return "", err
		}
		return peerID, nil */
	return "", nil
}

func (w *Waku) timestamp() int64 {
	return w.timesource.Now().UnixNano()
}

func (w *Waku) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	// TODO-nwaku
	/*
		peerID, err := w.node.AddPeer(address, wps.Static, w.cfg.DefaultShardedPubsubTopics, relay.WakuRelayID_v200)
		if err != nil {
			return "", err
		}
		return peerID, nil
	*/
	return "", nil
}

func (w *Waku) DialPeer(address multiaddr.Multiaddr) error {
	// TODO-nwaku
	/*
		ctx, cancel := context.WithTimeout(w.ctx, requestTimeout)
		defer cancel()
		return w.node.DialPeerWithMultiAddress(ctx, address) */
	return nil
}

func (w *Waku) DialPeerByID(peerID peer.ID) error {
	return w.WakuDialPeerById(peerID, string(relay.WakuRelayID_v200), 1000)
}

func (w *Waku) DropPeer(peerID peer.ID) error {
	// TODO-nwaku
	// return w.node.ClosePeerById(peerID)
	return nil
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

// TODO-nwaku
func (w *Waku) PeerID() peer.ID {
	// return w.node.Host().ID()
	return ""
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

// TODO-nwaku
func (w *Waku) StoreNode() *store.WakuStore {
	// return w.node.Store()
	return nil
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

// TODO-nwaku
func (w *Waku) LegacyStoreNode() legacy_store.Store {
	// return w.node.LegacyStore()
	return nil
}

func WakuSetup() {
	C.waku_setup()
}

func printStackTrace() {
	// Create a buffer to hold the stack trace
	buf := make([]byte, 102400)
	// Capture the stack trace into the buffer
	n := runtime.Stack(buf, false)
	// Print the stack trace
	fmt.Printf("Current stack trace:\n%s\n", buf[:n])
}

func wakuNew(nodeKey *ecdsa.PrivateKey,
	fleet string,
	cfg *Config, // TODO: merge Config and WakuConfig
	nwakuCfg *WakuConfig,
	logger *zap.Logger,
	appDB *sql.DB,
	ts *timesource.NTPTimeSource,
	onHistoricMessagesRequestFailed func([]byte, peer.ID, error), onPeerStats func(types.ConnStatus)) (*Waku, error) {

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

	nwakuCfg.NodeKey = hex.EncodeToString(crypto.FromECDSA(nodeKey))

	// TODO-nwaku
	// TODO: merge Config and WakuConfig
	cfg = setDefaults(cfg)
	if err = cfg.Validate(logger); err != nil {
		return nil, err
	}
	logger.Info("starting wakuv2 with config", zap.Any("nwakuCfg", nwakuCfg), zap.Any("wakuCfg", cfg))

	jsonConfig, err := json.Marshal(nwakuCfg)
	if err != nil {
		return nil, err
	}

	var cJsonConfig = C.CString(string(jsonConfig))
	var resp = C.allocResp()

	defer C.free(unsafe.Pointer(cJsonConfig))
	defer C.freeResp(resp)

	wakuCtx := C.cGoWakuNew(cJsonConfig, resp)
	// Notice that the events for self node are handled by the 'MyEventCallback' method

	if C.getRet(resp) == C.RET_OK {
		ctx, cancel := context.WithCancel(context.Background())
		return &Waku{
			wakuCtx:                         wakuCtx,
			wakuCfg:                         nwakuCfg,
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
			discV5BootstrapNodes:            nwakuCfg.Discv5BootstrapNodes,
			onHistoricMessagesRequestFailed: onHistoricMessagesRequestFailed,
			onPeerStats:                     onPeerStats,
			onlineChecker:                   onlinechecker.NewDefaultOnlineChecker(false).(*onlinechecker.DefaultOnlineChecker),
			sendQueue:                       publish.NewMessageQueue(1000, cfg.UseThrottledPublish),
		}, nil
	}

	errMsg := "error wakuNew: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, errors.New(errMsg)
}

func (self *Waku) WakuStart() error {

	var resp = C.allocResp()
	defer C.freeResp(resp)
	C.cGoWakuStart(self.wakuCtx, resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuStart: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (self *Waku) WakuStop() error {
	var resp = C.allocResp()
	defer C.freeResp(resp)
	C.cGoWakuStop(self.wakuCtx, resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuStop: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}
func (self *Waku) WakuDestroy() error {
	var resp = C.allocResp()
	defer C.freeResp(resp)
	C.cGoWakuDestroy(self.wakuCtx, resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuDestroy: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (self *Waku) StartDiscV5() error {
	var resp = C.allocResp()
	defer C.freeResp(resp)
	C.cGoWakuStartDiscV5(self.wakuCtx, resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuStartDiscV5: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (self *Waku) StopDiscV5() error {
	var resp = C.allocResp()
	defer C.freeResp(resp)
	C.cGoWakuStopDiscV5(self.wakuCtx, resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuStopDiscV5: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (self *Waku) WakuVersion() (string, error) {
	var resp = C.allocResp()
	defer C.freeResp(resp)

	C.cGoWakuVersion(self.wakuCtx, resp)

	if C.getRet(resp) == C.RET_OK {
		var version = C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		return version, nil
	}

	errMsg := "error WakuVersion: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return "", errors.New(errMsg)
}

//export globalEventCallback
func globalEventCallback(callerRet C.int, msg *C.char, len C.size_t, userData unsafe.Pointer) {
	// This is shared among all Golang instances
	self := Waku{wakuCtx: userData}
	self.MyEventCallback(callerRet, msg, len)
}

func (self *Waku) MyEventCallback(callerRet C.int, msg *C.char, len C.size_t) {
	fmt.Println("Event received:", C.GoStringN(msg, C.int(len)))
}

func (self *Waku) WakuSetEventCallback() {
	// Notice that the events for self node are handled by the 'MyEventCallback' method
	C.cGoWakuSetEventCallback(self.wakuCtx)
}

func (self *Waku) FormatContentTopic(
	appName string,
	appVersion int,
	contentTopicName string,
	encoding string) (WakuContentTopic, error) {

	var cAppName = C.CString(appName)
	var cContentTopicName = C.CString(contentTopicName)
	var cEncoding = C.CString(encoding)
	var resp = C.allocResp()

	defer C.free(unsafe.Pointer(cAppName))
	defer C.free(unsafe.Pointer(cContentTopicName))
	defer C.free(unsafe.Pointer(cEncoding))
	defer C.freeResp(resp)

	C.cGoWakuContentTopic(self.wakuCtx,
		cAppName,
		C.int(appVersion),
		cContentTopicName,
		cEncoding,
		resp)

	if C.getRet(resp) == C.RET_OK {
		var contentTopic = C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		return contentTopic, nil
	}

	errMsg := "error FormatContentTopic: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))

	return "", errors.New(errMsg)
}

func (self *Waku) FormatPubsubTopic(topicName string) (WakuPubsubTopic, error) {
	var cTopicName = C.CString(topicName)
	var resp = C.allocResp()

	defer C.free(unsafe.Pointer(cTopicName))
	defer C.freeResp(resp)

	C.cGoWakuPubsubTopic(self.wakuCtx, cTopicName, resp)
	if C.getRet(resp) == C.RET_OK {
		var pubsubTopic = C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		return pubsubTopic, nil
	}

	errMsg := "error FormatPubsubTopic: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))

	return "", errors.New(errMsg)
}

func (self *Waku) WakuDefaultPubsubTopic() (WakuPubsubTopic, error) {
	var resp = C.allocResp()
	defer C.freeResp(resp)
	C.cGoWakuDefaultPubsubTopic(self.wakuCtx, resp)
	if C.getRet(resp) == C.RET_OK {
		var defaultPubsubTopic = C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		return defaultPubsubTopic, nil
	}

	errMsg := "error WakuDefaultPubsubTopic: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))

	return "", errors.New(errMsg)
}

func (self *Waku) WakuRelaySubscribe(pubsubTopic string) error {
	var resp = C.allocResp()
	var cPubsubTopic = C.CString(pubsubTopic)

	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))

	if self.wakuCtx == nil {
		return errors.New("wakuCtx is nil")
	}
	// if self.cPubsubTopic == nil {
	// 	fmt.Println("cPubsubTopic is nil")
	// }
	// if self.resp == nil {
	// 	fmt.Println("resp is nil")
	// }

	C.cGoWakuRelaySubscribe(self.wakuCtx, cPubsubTopic, resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuRelaySubscribe: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (self *Waku) WakuRelayUnsubscribe(pubsubTopic string) error {
	var resp = C.allocResp()
	var cPubsubTopic = C.CString(pubsubTopic)
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))
	C.cGoWakuRelayUnsubscribe(self.wakuCtx, cPubsubTopic, resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuRelayUnsubscribe: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (self *Waku) WakuLightpushPublish(message *pb.WakuMessage, pubsubTopic string) (string, error) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	var cPubsubTopic = C.CString(pubsubTopic)
	var msg = C.CString(string(jsonMsg))
	var resp = C.allocResp()

	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))
	defer C.free(unsafe.Pointer(msg))

	C.cGoWakuLightpushPublish(self.wakuCtx, cPubsubTopic, msg, resp)
	if C.getRet(resp) == C.RET_OK {
		msg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		return msg, nil
	}
	errMsg := "error WakuLightpushPublish: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return "", errors.New(errMsg)
}

func wakuStoreQuery(
	wakuCtx unsafe.Pointer,
	jsonQuery string,
	peerAddr string,
	timeoutMs int) (string, error) {

	var cJsonQuery = C.CString(jsonQuery)
	var cPeerAddr = C.CString(peerAddr)
	var resp = C.allocResp()

	defer C.free(unsafe.Pointer(cJsonQuery))
	defer C.free(unsafe.Pointer(cPeerAddr))
	defer C.freeResp(resp)

	C.cGoWakuStoreQuery(wakuCtx, cJsonQuery, cPeerAddr, C.int(timeoutMs), resp)
	if C.getRet(resp) == C.RET_OK {
		msg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		return msg, nil
	}
	errMsg := "error WakuStoreQuery: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return "", errors.New(errMsg)
}

func (self *Waku) WakuPeerExchangeRequest(numPeers uint64) (string, error) {
	var resp = C.allocResp()
	defer C.freeResp(resp)

	C.cGoWakuPeerExchangeQuery(self.wakuCtx, C.uint64_t(numPeers), resp)
	if C.getRet(resp) == C.RET_OK {
		msg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		return msg, nil
	}
	errMsg := "error WakuPeerExchangeRequest: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return "", errors.New(errMsg)
}

func (self *Waku) WakuConnect(peerMultiAddr string, timeoutMs int) error {
	var resp = C.allocResp()
	var cPeerMultiAddr = C.CString(peerMultiAddr)
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPeerMultiAddr))

	C.cGoWakuConnect(self.wakuCtx, cPeerMultiAddr, C.int(timeoutMs), resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuConnect: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (self *Waku) WakuDialPeerById(peerId peer.ID, protocol string, timeoutMs int) error {
	var resp = C.allocResp()
	var cPeerId = C.CString(peerId.String())
	var cProtocol = C.CString(protocol)
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPeerId))
	defer C.free(unsafe.Pointer(cProtocol))

	C.cGoWakuDialPeerById(self.wakuCtx, cPeerId, cProtocol, C.int(timeoutMs), resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error DialPeerById: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (self *Waku) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	var resp = C.allocResp()
	defer C.freeResp(resp)
	C.cGoWakuListenAddresses(self.wakuCtx, resp)

	if C.getRet(resp) == C.RET_OK {

		var addrsRet []multiaddr.Multiaddr
		listenAddresses := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		addrss := strings.Split(listenAddresses, ",")
		for _, addr := range addrss {
			addr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				return nil, err
			}

			addrsRet = append(addrsRet, addr)
		}

		return addrsRet, nil
	}
	errMsg := "error WakuListenAddresses: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))

	return nil, errors.New(errMsg)
}

func (self *Waku) ENR() (*enode.Node, error) {
	var resp = C.allocResp()
	defer C.freeResp(resp)
	C.cGoWakuGetMyENR(self.wakuCtx, resp)

	if C.getRet(resp) == C.RET_OK {
		enrStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		n, err := enode.Parse(enode.ValidSchemes, enrStr)
		if err != nil {
			return nil, err
		}
		return n, nil
	}
	errMsg := "error WakuGetMyENR: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, errors.New(errMsg)
}

func (self *Waku) ListPeersInMesh(pubsubTopic string) (int, error) {
	var resp = C.allocResp()
	var cPubsubTopic = C.CString(pubsubTopic)
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))

	C.cGoWakuListPeersInMesh(self.wakuCtx, cPubsubTopic, resp)

	if C.getRet(resp) == C.RET_OK {
		numPeersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		numPeers, err := strconv.Atoi(numPeersStr)
		if err != nil {
			errMsg := "ListPeersInMesh - error converting string to int: " + err.Error()
			return 0, errors.New(errMsg)
		}
		return numPeers, nil
	}
	errMsg := "error ListPeersInMesh: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return 0, errors.New(errMsg)
}

func (self *Waku) GetNumConnectedPeers(paramPubsubTopic ...string) (int, error) {
	var pubsubTopic string
	if len(paramPubsubTopic) == 0 {
		pubsubTopic = ""
	} else {
		pubsubTopic = paramPubsubTopic[0]
	}

	var resp = C.allocResp()
	var cPubsubTopic = C.CString(pubsubTopic)
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))

	C.cGoWakuGetNumConnectedPeers(self.wakuCtx, cPubsubTopic, resp)

	if C.getRet(resp) == C.RET_OK {
		numPeersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		numPeers, err := strconv.Atoi(numPeersStr)
		if err != nil {
			errMsg := "GetNumConnectedPeers - error converting string to int: " + err.Error()
			return 0, errors.New(errMsg)
		}
		return numPeers, nil
	}
	errMsg := "error GetNumConnectedPeers: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return 0, errors.New(errMsg)
}

func (self *Waku) GetPeerIdsByProtocol(protocol string) (peer.IDSlice, error) {
	var resp = C.allocResp()
	var cProtocol = C.CString(protocol)
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cProtocol))

	C.cGoWakuGetPeerIdsByProtocol(self.wakuCtx, cProtocol, resp)

	if C.getRet(resp) == C.RET_OK {
		peersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		if peersStr == "" {
			return peer.IDSlice{}, nil
		}
		// peersStr contains a comma-separated list of peer ids
		itemsPeerIds := strings.Split(peersStr, ",")

		var peers peer.IDSlice
		for _, p := range itemsPeerIds {
			id, err := peer.Decode(p)
			if err != nil {
				errMsg := "GetPeerIdsByProtocol - error converting string to int: " + err.Error()
				return nil, errors.New(errMsg)
			}
			peers = append(peers, id)
		}

		return peers, nil
	}
	errMsg := "error GetPeerIdsByProtocol: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, errors.New(errMsg)
}

func (self *Waku) DisconnectPeerById(peerId peer.ID) error {
	var resp = C.allocResp()
	var cPeerId = C.CString(peerId.String())
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPeerId))

	C.cGoWakuDisconnectPeerById(self.wakuCtx, cPeerId, resp)

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error DisconnectPeerById: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

// func main() {

// 	config := WakuConfig{
// 		Host:        "0.0.0.0",
// 		Port:        30304,
// 		NodeKey:     "11d0dcea28e86f81937a3bd1163473c7fbc0a0db54fd72914849bc47bdf78710",
// 		EnableRelay: true,
// 		LogLevel:    "DEBUG",
// 	}

// 	node, err := wakuNew(config)
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	node.WakuSetEventCallback()

// 	defaultPubsubTopic, err := node.WakuDefaultPubsubTopic()
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	err = node.WakuRelaySubscribe(defaultPubsubTopic)
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	err = node.WakuConnect(
// 		// tries to connect to a localhost node with key: 0d714a1fada214dead6dc9c7274585eca0ff292451866e7d6d677dc818e8ccd2
// 		"/ip4/0.0.0.0/tcp/60000/p2p/16Uiu2HAmVFXtAfSj4EiR7mL2KvL4EE2wztuQgUSBoj2Jx2KeXFLN",
// 		10000)
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	err = node.WakuStart()
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	version, err := node.WakuVersion()
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	formattedContentTopic, err := node.FormatContentTopic("appName", 1, "cTopicName", "enc")
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	formattedPubsubTopic, err := node.FormatPubsubTopic("my-ctopic")
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	listenAddresses, err := node.WakuListenAddresses()
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	fmt.Println("Version:", version)
// 	fmt.Println("Custom content topic:", formattedContentTopic)
// 	fmt.Println("Custom pubsub topic:", formattedPubsubTopic)
// 	fmt.Println("Default pubsub topic:", defaultPubsubTopic)
// 	fmt.Println("Listen addresses:", listenAddresses)

// 	// Wait for a SIGINT or SIGTERM signal
// 	ch := make(chan os.Signal, 1)
// 	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
// 	<-ch

// 	err = node.WakuStop()
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}

// 	err = node.WakuDestroy()
// 	if err != nil {
// 		fmt.Println("Error happened:", err.Error())
// 		return
// 	}
// }

// MaxMessageSize returns the maximum accepted message size.
/* TODO-nwaku
func (w *Waku) MaxMessageSize() uint32 {
	return w.cfg.MaxMessageSize
} */

func newPublisher(wakuCtx unsafe.Pointer) publish.Publisher {
	return &nwakuPublisher{
		wakuCtx: wakuCtx,
	}
}

type nwakuPublisher struct {
	wakuCtx unsafe.Pointer
}

func (p *nwakuPublisher) RelayListPeers(pubsubTopic string) ([]peer.ID, error) {
	// TODO-nwaku
	return nil, nil
}

func (p *nwakuPublisher) RelayPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string) (pb.MessageHash, error) {
	timeoutMs := 1000

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return pb.MessageHash{}, err
	}

	var cPubsubTopic = C.CString(pubsubTopic)
	var msg = C.CString(string(jsonMsg))
	var resp = C.allocResp()

	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))
	defer C.free(unsafe.Pointer(msg))

	C.cGoWakuRelayPublish(p.wakuCtx, cPubsubTopic, msg, C.int(timeoutMs), resp)
	if C.getRet(resp) == C.RET_OK {
		msgHash := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		msgHashBytes, err := hexutil.Decode(msgHash)
		if err != nil {
			return pb.MessageHash{}, err
		}
		return pb.ToMessageHash(msgHashBytes), nil
	}
	errMsg := "error WakuRelayPublish: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return pb.MessageHash{}, errors.New(errMsg)
}

// LightpushPublish publishes a message via WakuLightPush
func (p *nwakuPublisher) LightpushPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string, maxPeers int) (pb.MessageHash, error) {
	// TODO-nwaku
	return pb.MessageHash{}, errors.New("not implemented yet")
}

func newStorenodeMessageVerifier(wakuCtx unsafe.Pointer) publish.StorenodeMessageVerifier {
	return &storenodeMessageVerifier{
		wakuCtx: wakuCtx,
	}
}

type storenodeMessageVerifier struct {
	wakuCtx unsafe.Pointer
}

func (d *storenodeMessageVerifier) MessageHashesExist(ctx context.Context, requestID []byte, peerID peer.ID, pageSize uint64, messageHashes []pb.MessageHash) ([]pb.MessageHash, error) {
	requestIDStr := hex.EncodeToString(requestID)
	storeRequest := &storepb.StoreQueryRequest{
		RequestId:         requestIDStr,
		MessageHashes:     make([][]byte, len(messageHashes)),
		IncludeData:       false,
		PaginationCursor:  nil,
		PaginationForward: false,
		PaginationLimit:   proto.Uint64(pageSize),
	}

	for i, mhash := range messageHashes {
		storeRequest.MessageHashes[i] = mhash.Bytes()
	}

	jsonQuery, err := json.Marshal(storeRequest)
	if err != nil {
		return nil, err
	}

	// TODO: timeouts need to be managed differently. For now we're using a 1m timeout
	jsonResponse, err := wakuStoreQuery(d.wakuCtx, string(jsonQuery), peerID.String(), int(time.Minute.Milliseconds()))
	if err != nil {
		return nil, err
	}

	response := &storepb.StoreQueryResponse{}
	err = json.Unmarshal([]byte(jsonResponse), response)
	if err != nil {
		return nil, err
	}

	if response.GetStatusCode() != http.StatusOK {
		return nil, fmt.Errorf("could not query storenode: %s %d %s", requestIDStr, response.GetStatusCode(), response.GetStatusDesc())
	}

	result := make([]pb.MessageHash, len(response.Messages))
	for i, msg := range response.Messages {
		result[i] = pb.ToMessageHash(msg.GetMessageHash())
	}

	return result, nil
}

type pinger struct {
	wakuCtx unsafe.Pointer
}

func newPinger(wakuCtx unsafe.Pointer) commonapi.Pinger {
	return &pinger{
		wakuCtx: wakuCtx,
	}
}

func (p *pinger) PingPeer(ctx context.Context, peerID peer.ID) (time.Duration, error) {
	var resp = C.allocResp()
	var cPeerId = C.CString(peerID.String())
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPeerId))

	C.cGoWakuPingPeer(p.wakuCtx, cPeerId, C.int(time.Minute.Milliseconds()), resp)
	if C.getRet(resp) == C.RET_OK {
		rttStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		rttInt, err := strconv.ParseInt(rttStr, 10, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(rttInt), nil
	}

	errMsg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return 0, fmt.Errorf("PingPeer: %s", errMsg)
}

type storenodeRequestor struct {
	wakuCtx unsafe.Pointer
	logger  *zap.Logger
}

func newStorenodeRequestor(wakuCtx unsafe.Pointer, logger *zap.Logger) commonapi.StorenodeRequestor {
	return &storenodeRequestor{
		wakuCtx: wakuCtx,
		logger:  logger.Named("storenodeRequestor"),
	}
}

func (s *storenodeRequestor) GetMessagesByHash(ctx context.Context, peerID peer.ID, pageSize uint64, messageHashes []pb.MessageHash) (commonapi.StoreRequestResult, error) {
	requestIDStr := hex.EncodeToString(protocol.GenerateRequestID())

	logger := s.logger.With(zap.Stringer("peerID", peerID), zap.String("requestID", requestIDStr))

	logger.Debug("sending store request")

	storeRequest := &storepb.StoreQueryRequest{
		RequestId:         requestIDStr,
		MessageHashes:     make([][]byte, len(messageHashes)),
		IncludeData:       true,
		PaginationCursor:  nil,
		PaginationForward: false,
		PaginationLimit:   proto.Uint64(pageSize),
	}

	for i, mhash := range messageHashes {
		storeRequest.MessageHashes[i] = mhash.Bytes()
	}

	jsonQuery, err := json.Marshal(storeRequest)
	if err != nil {
		return nil, err
	}

	// TODO: timeouts need to be managed differently. For now we're using a 1m timeout
	jsonResponse, err := wakuStoreQuery(s.wakuCtx, string(jsonQuery), peerID.String(), int(time.Minute.Milliseconds()))
	if err != nil {
		return nil, err
	}

	storeResponse := &storepb.StoreQueryResponse{}
	err = json.Unmarshal([]byte(jsonResponse), storeResponse)
	if err != nil {
		return nil, err
	}

	if storeResponse.GetStatusCode() != http.StatusOK {
		return nil, fmt.Errorf("could not query storenode: %s %d %s", requestIDStr, storeResponse.GetStatusCode(), storeResponse.GetStatusDesc())
	}

	return newStoreResultImpl(s.wakuCtx, peerID, storeRequest, storeResponse), nil
}

func (s *storenodeRequestor) Query(ctx context.Context, peerID peer.ID, storeRequest *storepb.StoreQueryRequest) (commonapi.StoreRequestResult, error) {
	jsonQuery, err := json.Marshal(storeRequest)
	if err != nil {
		return nil, err
	}

	// TODO: timeouts need to be managed differently. For now we're using a 1m timeout
	jsonResponse, err := wakuStoreQuery(s.wakuCtx, string(jsonQuery), peerID.String(), int(time.Minute.Milliseconds()))
	if err != nil {
		return nil, err
	}

	storeResponse := &storepb.StoreQueryResponse{}
	err = json.Unmarshal([]byte(jsonResponse), storeResponse)
	if err != nil {
		return nil, err
	}

	if storeResponse.GetStatusCode() != http.StatusOK {
		return nil, fmt.Errorf("could not query storenode: %s %d %s", storeRequest.RequestId, storeResponse.GetStatusCode(), storeResponse.GetStatusDesc())
	}

	return newStoreResultImpl(s.wakuCtx, peerID, storeRequest, storeResponse), nil
}

type storeResultImpl struct {
	done bool

	wakuCtx       unsafe.Pointer
	storeRequest  *storepb.StoreQueryRequest
	storeResponse *storepb.StoreQueryResponse
	peerID        peer.ID
}

func newStoreResultImpl(wakuCtx unsafe.Pointer, peerID peer.ID, storeRequest *storepb.StoreQueryRequest, storeResponse *storepb.StoreQueryResponse) *storeResultImpl {
	return &storeResultImpl{
		wakuCtx:       wakuCtx,
		storeRequest:  storeRequest,
		storeResponse: storeResponse,
		peerID:        peerID,
	}
}

func (r *storeResultImpl) Cursor() []byte {
	return r.storeResponse.GetPaginationCursor()
}

func (r *storeResultImpl) IsComplete() bool {
	return r.done
}

func (r *storeResultImpl) PeerID() peer.ID {
	return r.peerID
}

func (r *storeResultImpl) Query() *storepb.StoreQueryRequest {
	return r.storeRequest
}

func (r *storeResultImpl) Response() *storepb.StoreQueryResponse {
	return r.storeResponse
}

func (r *storeResultImpl) Next(ctx context.Context, opts ...store.RequestOption) error {
	// TODO: opts is being ignored. Will require some changes in go-waku. For now using this
	// is not necessary

	if r.storeResponse.GetPaginationCursor() == nil {
		r.done = true
		return nil
	}

	r.storeRequest.RequestId = hex.EncodeToString(protocol.GenerateRequestID())
	r.storeRequest.PaginationCursor = r.storeResponse.PaginationCursor

	jsonQuery, err := json.Marshal(r.storeRequest)
	if err != nil {
		return err
	}

	// TODO: timeouts need to be managed differently. For now we're using a 1m timeout
	jsonResponse, err := wakuStoreQuery(r.wakuCtx, string(jsonQuery), r.peerID.String(), int(time.Minute.Milliseconds()))
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(jsonResponse), r.storeResponse)
	if err != nil {
		return err
	}

	return nil
}

func (r *storeResultImpl) Messages() []*storepb.WakuMessageKeyValue {
	return r.storeResponse.GetMessages()
}
