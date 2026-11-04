package main

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/pbnjay/memory"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"

	dbutils "github.com/waku-org/go-waku/waku/persistence/utils"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	wakupeerstore "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/rendezvous"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	dssql "github.com/ipfs/go-ds-sql"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds" // nolint: staticcheck
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/cmd/waku/server/rest"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/metrics"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/v2/node"
	wprotocol "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"

	humanize "github.com/dustin/go-humanize"
)

func requiresDB(options NodeOptions) bool {
	return options.Store.Enable || options.Rendezvous.Enable
}

func scalePerc(value float64) float64 {
	if value > 100 {
		return 100
	}

	if value < 0.1 {
		return 0.1
	}

	return value
}

const dialTimeout = 7 * time.Second

func nonRecoverErrorMsg(format string, a ...any) error {
	err := fmt.Errorf(format, a...)
	return nonRecoverError(err)
}

func nonRecoverError(err error) error {
	return cli.Exit(err.Error(), 166)
}

// Execute starts a go-waku node with settings determined by the Options parameter
func Execute(options NodeOptions) error {
	// Set encoding for logs (console, json, ...)
	// Note that libp2p reads the encoding from GOLOG_LOG_FMT env var.
	lvl, err := zapcore.ParseLevel(options.LogLevel)
	if err != nil {
		return err
	}
	utils.InitLogger(options.LogEncoding, options.LogOutput, "gowaku", lvl)

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", options.Address, options.Port))
	if err != nil {
		return nonRecoverErrorMsg("invalid host address: %w", err)
	}

	prvKey, err := getPrivKey(options)
	if err != nil {
		return err
	}

	p2pPrvKey := utils.EcdsaPrivKeyToSecp256k1PrivKey(prvKey)
	id, err := peer.IDFromPublicKey(p2pPrvKey.GetPublic())
	if err != nil {
		return err
	}

	logger := utils.Logger().With(logging.HostID("node", id))

	var db *sql.DB
	var migrationFn func(*sql.DB, *zap.Logger) error
	if requiresDB(options) && options.Store.Migration {
		dbSettings := dbutils.DBSettings{}
		db, migrationFn, err = dbutils.ParseURL(options.Store.DatabaseURL, dbSettings, logger)
		if err != nil {
			return nonRecoverErrorMsg("could not connect to DB: %w", err)
		}
	}

	ctx := context.Background()

	var metricsServer *metrics.Server
	if options.Metrics.Enable {
		metricsServer = metrics.NewMetricsServer(options.Metrics.Address, options.Metrics.Port, logger)
		go metricsServer.Start()
	}

	nodeOpts := []node.WakuNodeOption{
		node.WithLogger(logger),
		node.WithLogLevel(lvl),
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithKeepAlive(10*time.Second, options.KeepAlive),
		node.WithMaxPeerConnections(options.MaxPeerConnections),
		node.WithPrometheusRegisterer(prometheus.DefaultRegisterer),
		node.WithPeerStoreCapacity(options.PeerStoreCapacity),
		node.WithMaxConnectionsPerIP(options.IPColocationLimit),
		node.WithClusterID(uint16(options.ClusterID)),
	}
	if len(options.AdvertiseAddresses) != 0 {
		nodeOpts = append(nodeOpts, node.WithAdvertiseAddresses(options.AdvertiseAddresses...))
	}

	if options.ExtIP != "" {
		ip := net.ParseIP(options.ExtIP)
		if ip == nil {
			return nonRecoverErrorMsg("could not set external IP address: invalid IP")
		}

		nodeOpts = append(nodeOpts, node.WithExternalIP(ip))
	}

	if options.DNS4DomainName != "" {
		nodeOpts = append(nodeOpts, node.WithDNS4Domain(options.DNS4DomainName))
	}

	libp2pOpts := node.DefaultLibP2POptions

	libp2pOpts = append(libp2pOpts, libp2p.PrometheusRegisterer(prometheus.DefaultRegisterer))

	memPerc := scalePerc(options.ResourceScalingMemoryPercent)
	fdPerc := scalePerc(options.ResourceScalingFDPercent)
	limits := rcmgr.DefaultLimits // Default memory limit: 1/8th of total memory, minimum 128MB, maximum 1GB
	scaledLimits := limits.Scale(int64(float64(memory.TotalMemory())*memPerc/100), int(float64(getNumFDs())*fdPerc/100))
	resourceManager, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(scaledLimits))
	if err != nil {
		return fmt.Errorf("could not set resource limits: %w", err)
	}

	libp2pOpts = append(libp2pOpts, libp2p.ResourceManager(resourceManager))
	libp2p.SetDefaultServiceLimits(&limits)

	if len(options.AdvertiseAddresses) == 0 {
		libp2pOpts = append(libp2pOpts, libp2p.NATPortMap()) // Attempt to open ports using uPNP for NATed hosts.)
	}

	// Node can be a circuit relay server
	if options.CircuitRelay {
		libp2pOpts = append(libp2pOpts, libp2p.EnableRelayService())
	}

	if options.ForceReachability != "" {
		libp2pOpts = append(libp2pOpts, libp2p.EnableRelay())
		nodeOpts = append(nodeOpts, node.WithCircuitRelayParams(2*time.Second, 2*time.Second))
		if options.ForceReachability == "private" {
			logger.Warn("node forced to be unreachable!")
			libp2pOpts = append(libp2pOpts, libp2p.ForceReachabilityPrivate())
		} else if options.ForceReachability == "public" {
			logger.Warn("node forced to be publicly reachable!")
			libp2pOpts = append(libp2pOpts, libp2p.ForceReachabilityPublic())
		} else {
			return nonRecoverErrorMsg("invalid reachability value")
		}
	}

	if options.UserAgent != "" {
		libp2pOpts = append(libp2pOpts, libp2p.UserAgent(options.UserAgent))
	}

	if options.Websocket.Enable {
		nodeOpts = append(nodeOpts, node.WithWebsockets(options.Websocket.Address, options.Websocket.WSPort))
	}

	if options.Websocket.Secure {
		nodeOpts = append(nodeOpts, node.WithSecureWebsockets(options.Websocket.Address, options.Websocket.WSSPort, options.Websocket.CertPath, options.Websocket.KeyPath))
	}

	if options.ShowAddresses {
		printListeningAddresses(ctx, nodeOpts, options)
		return nil
	}

	if options.Store.Enable && options.PersistPeers {
		// Create persistent peerstore
		queries, err := dbutils.NewQueries("peerstore", db)
		if err != nil {
			return nonRecoverErrorMsg("could not setup persistent peerstore database: %w", err)

		}

		datastore := dssql.NewDatastore(db, queries)
		opts := pstoreds.DefaultOpts()
		peerStore, err := pstoreds.NewPeerstore(ctx, datastore, opts)
		if err != nil {
			return nonRecoverErrorMsg("could not create persistent peerstore: %w", err)
		}

		nodeOpts = append(nodeOpts, node.WithPeerStore(peerStore))
	}

	nodeOpts = append(nodeOpts, node.WithLibP2POptions(libp2pOpts...))
	nodeOpts = append(nodeOpts, node.WithNTP())

	maxMsgSize := parseMsgSizeConfig(options.Relay.MaxMsgSize)

	if options.Relay.Enable {
		var wakurelayopts []pubsub.Option
		wakurelayopts = append(wakurelayopts, pubsub.WithPeerExchange(options.Relay.PeerExchange))
		wakurelayopts = append(wakurelayopts, pubsub.WithMaxMessageSize(maxMsgSize))

		nodeOpts = append(nodeOpts, node.WithWakuRelayAndMinPeers(options.Relay.MinRelayPeersToPublish, wakurelayopts...))
		nodeOpts = append(nodeOpts, node.WithMaxMsgSize(maxMsgSize))
	}

	nodeOpts = append(nodeOpts, node.WithWakuFilterLightNode())

	if options.Filter.Enable {
		nodeOpts = append(nodeOpts, node.WithWakuFilterFullNode(filter.WithTimeout(options.Filter.Timeout)))
	}

	var dbStore *persistence.DBStore
	if requiresDB(options) {
		dbOptions := []persistence.DBOption{
			persistence.WithDB(db),
			persistence.WithRetentionPolicy(options.Store.RetentionMaxMessages, options.Store.RetentionTime),
		}

		if options.Store.Migration {
			dbOptions = append(dbOptions, persistence.WithMigrations(migrationFn)) // TODO: refactor migrations out of DBStore, or merge DBStore with rendezvous DB
		}

		dbStore, err = persistence.NewDBStore(prometheus.DefaultRegisterer, logger, dbOptions...)
		if err != nil {
			return nonRecoverErrorMsg("error setting up db store: %w", err)
		}

		nodeOpts = append(nodeOpts, node.WithMessageProvider(dbStore))
	}

	if options.Store.Enable {
		nodeOpts = append(nodeOpts, node.WithWakuStore())
		nodeOpts = append(nodeOpts, node.WithMessageProvider(dbStore))
	}

	if options.LightPush.Enable {
		nodeOpts = append(nodeOpts, node.WithLightPush())
	}

	if options.PeerExchange.Enable {
		nodeOpts = append(nodeOpts, node.WithPeerExchange())
	}

	if options.Rendezvous.Enable {
		rdb := rendezvous.NewDB(db, logger)
		nodeOpts = append(nodeOpts, node.WithRendezvous(rdb))
	}

	utils.Logger().Info("Version details ", zap.String("version", node.Version), zap.String("commit", node.GitCommit))

	if err = checkForRLN(logger, options, &nodeOpts); err != nil {
		return nonRecoverError(err)
	}

	var discoveredNodes []dnsdisc.DiscoveredNode
	if options.DNSDiscovery.Enable {
		if len(options.DNSDiscovery.URLs.Value()) == 0 {
			return nonRecoverErrorMsg("DNS discovery URL is required")
		}
		discoveredNodes = node.GetNodesFromDNSDiscovery(logger, ctx, options.DNSDiscovery.Nameserver, options.DNSDiscovery.URLs.Value())
	}
	if options.DiscV5.Enable {
		discv5Opts, err := node.GetDiscv5Option(discoveredNodes, options.DiscV5.Nodes.Value(), options.DiscV5.Port, options.DiscV5.AutoUpdate)
		if err != nil {
			logger.Fatal("parsing ENR", zap.Error(err))
		}
		nodeOpts = append(nodeOpts, discv5Opts)
	}

	//Process pubSub and contentTopics specified and arrive at all corresponding pubSubTopics
	pubSubTopicMap, err := processTopics(options)
	if err != nil {
		return nonRecoverError(err)
	}

	pubSubTopicMapKeys := make([]string, 0, len(pubSubTopicMap))
	for k := range pubSubTopicMap {
		pubSubTopicMapKeys = append(pubSubTopicMapKeys, k)
	}

	rs, err := wprotocol.TopicsToRelayShards(pubSubTopicMapKeys...)
	if err == nil {
		if len(rs) == 1 {
			nodeOpts = append(nodeOpts, node.WithShards(rs[0].ShardIDs))
		} else {
			logger.Warn("could not set ENR shard info", zap.String("error", "invalid number of clusters found"), zap.Int("numClusters", len(rs)))
		}
	} else {
		logger.Warn("could not obtain list of shards", zap.Error(err))
	}

	wakuNode, err := node.New(nodeOpts...)
	if err != nil {
		return fmt.Errorf("could not instantiate waku: %w", err)
	}

	if err = wakuNode.Start(ctx); err != nil {
		return nonRecoverError(err)
	}

	for _, d := range discoveredNodes {
		wakuNode.AddDiscoveredPeer(d.PeerID, d.PeerInfo.Addrs, wakupeerstore.DNSDiscovery, nil, d.ENR, true)
	}

	//For now assuming that static peers added support/listen on all topics specified via commandLine.
	staticPeers := map[protocol.ID][]multiaddr.Multiaddr{
		legacy_store.StoreID_v20beta4:     options.Store.Nodes,
		lightpush.LightPushID_v20beta1:    options.LightPush.Nodes,
		rendezvous.RendezvousID:           options.Rendezvous.Nodes,
		filter.FilterSubscribeID_v20beta1: options.Filter.Nodes,
	}
	for protocolID, peers := range staticPeers {
		if err = addStaticPeers(wakuNode, peers, pubSubTopicMapKeys, protocolID); err != nil {
			return err
		}
	}

	var wg sync.WaitGroup

	if options.Relay.Enable {
		if err = handleRelayTopics(ctx, &wg, wakuNode, pubSubTopicMap); err != nil {
			return err
		}
	}

	for _, n := range options.StaticNodes {
		go func(ctx context.Context, node multiaddr.Multiaddr) {
			ctx, cancel := context.WithTimeout(ctx, dialTimeout)
			defer cancel()
			err = wakuNode.DialPeerWithMultiAddress(ctx, node)
			if err != nil {
				logger.Error("dialing peer", zap.Error(err))
			}
		}(ctx, n)
	}

	if options.DiscV5.Enable {
		if err = wakuNode.DiscV5().Start(ctx); err != nil {
			logger.Fatal("starting discovery v5", zap.Error(err))
		}
	}

	// retrieve and connect to peer exchange peers
	if options.PeerExchange.Enable && options.PeerExchange.Node != nil {
		logger.Info("retrieving peer info via peer exchange protocol")

		peerID, err := wakuNode.AddPeer([]multiaddr.Multiaddr{*options.PeerExchange.Node}, wakupeerstore.Static,
			pubSubTopicMapKeys, peer_exchange.PeerExchangeID_v20alpha1)
		if err != nil {
			logger.Error("adding peer exchange peer", logging.MultiAddrs("node", *options.PeerExchange.Node), zap.Error(err))
		} else {
			desiredOutDegree := wakuNode.Relay().Params().D
			if err = wakuNode.PeerExchange().Request(ctx, desiredOutDegree, peer_exchange.WithPeer(peerID)); err != nil {
				logger.Error("requesting peers via peer exchange", zap.Error(err))
			}
		}
	}

	var restServer *rest.WakuRest
	if options.RESTServer.Enable {
		wg.Add(1)
		restConfig := rest.RestConfig{Address: options.RESTServer.Address,
			Port:                uint(options.RESTServer.Port),
			EnablePProf:         options.PProf,
			EnableAdmin:         options.RESTServer.Admin,
			RelayCacheCapacity:  uint(options.RESTServer.RelayCacheCapacity),
			FilterCacheCapacity: uint(options.RESTServer.FilterCacheCapacity)}

		restServer = rest.NewWakuRest(wakuNode, restConfig, logger)
		restServer.Start(ctx, &wg)
	}

	wg.Wait()
	logger.Info("Node setup complete")

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	logger.Info("Received signal, shutting down...")

	// shut the node down
	wakuNode.Stop()

	if options.RESTServer.Enable {
		if err := restServer.Stop(ctx); err != nil {
			return err
		}
	}

	if options.Metrics.Enable {
		if err = metricsServer.Stop(ctx); err != nil {
			return err
		}
	}

	if db != nil {
		if err = db.Close(); err != nil {
			return err
		}
	}

	return nil
}

func processTopics(options NodeOptions) (map[string][]string, error) {

	//Using a map to avoid duplicate pub-sub topics that can result from autosharding
	// or same-topic being passed twice.
	pubSubTopicMap := make(map[string][]string)

	for _, topic := range options.Relay.Topics.Value() {
		pubSubTopicMap[topic] = []string{}
	}

	for _, topic := range options.Relay.PubSubTopics.Value() {
		pubSubTopicMap[topic] = []string{}
	}

	//Get pubSub topics from contentTopics if they are as per autosharding
	for _, cTopic := range options.Relay.ContentTopics.Value() {
		contentTopic, err := wprotocol.StringToContentTopic(cTopic)
		if err != nil {
			return nil, err
		}
		pTopic := wprotocol.GetShardFromContentTopic(contentTopic, wprotocol.GenerationZeroShardsCount)
		if _, ok := pubSubTopicMap[pTopic.String()]; !ok {
			pubSubTopicMap[pTopic.String()] = []string{}
		}
		pubSubTopicMap[pTopic.String()] = append(pubSubTopicMap[pTopic.String()], cTopic)
	}
	//If no topics are passed, then use default waku topic.
	if len(pubSubTopicMap) == 0 && options.ClusterID == 0 {
		pubSubTopicMap[relay.DefaultWakuTopic] = []string{}
	}

	return pubSubTopicMap, nil
}

func addStaticPeers(wakuNode *node.WakuNode, addresses []multiaddr.Multiaddr, pubSubTopics []string, protocols ...protocol.ID) error {
	for _, addr := range addresses {
		_, err := wakuNode.AddPeer([]multiaddr.Multiaddr{addr}, wakupeerstore.Static, pubSubTopics, protocols...)
		if err != nil {
			return fmt.Errorf("could not add static peer: %w", err)
		}
	}
	return nil
}

func loadPrivateKeyFromFile(path string, passwd string) (*ecdsa.PrivateKey, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var encryptedK keystore.CryptoJSON
	err = json.Unmarshal(src, &encryptedK)
	if err != nil {
		return nil, err
	}

	pKey, err := keystore.DecryptDataV3(encryptedK, passwd)
	if err != nil {
		return nil, err
	}

	return crypto.ToECDSA(pKey)
}

func getPrivKey(options NodeOptions) (*ecdsa.PrivateKey, error) {
	var prvKey *ecdsa.PrivateKey
	// get private key from nodeKey or keyFile
	if options.NodeKey != nil {
		prvKey = options.NodeKey
	} else {
		if _, err := os.Stat(options.KeyFile); err == nil {
			if prvKey, err = loadPrivateKeyFromFile(options.KeyFile, options.KeyPasswd); err != nil {
				return nil, fmt.Errorf("could not read keyfile: %w", err)
			}
		} else {
			if os.IsNotExist(err) {
				if prvKey, err = crypto.GenerateKey(); err != nil {
					return nil, fmt.Errorf("error generating key: %w", err)
				}
			} else {
				return nil, fmt.Errorf("could not read keyfile: %w", err)
			}
		}
	}
	return prvKey, nil
}

func printListeningAddresses(ctx context.Context, nodeOpts []node.WakuNodeOption, options NodeOptions) {
	params := new(node.WakuNodeParameters)
	for _, opt := range nodeOpts {
		err := opt(params)
		if err != nil {
			panic(err)
		}
	}

	var libp2pOpts []config.Option
	libp2pOpts = append(libp2pOpts,
		params.Identity(),
		libp2p.ListenAddrs(params.MultiAddresses()...),
	)

	if options.Websocket.Secure {
		transports := libp2p.ChainOptions(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Transport(ws.New, ws.WithTLSConfig(params.TLSConfig())),
		)
		libp2pOpts = append(libp2pOpts, transports)
	}

	addrFactory := params.AddressFactory()
	if addrFactory != nil {
		libp2pOpts = append(libp2pOpts, libp2p.AddrsFactory(addrFactory))
	}

	h, err := libp2p.New(libp2pOpts...)
	if err != nil {
		panic(err)
	}

	hostAddrs := utils.EncapsulatePeerID(h.ID(), h.Addrs()...)
	for _, addr := range hostAddrs {
		fmt.Println(addr)
	}

}

func parseMsgSizeConfig(msgSizeConfig string) int {

	msgSize, err := humanize.ParseBytes(msgSizeConfig)
	if err != nil {
		msgSize = 0
	}
	return int(msgSize)
}
