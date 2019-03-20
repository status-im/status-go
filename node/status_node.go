package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/discovery"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/peers"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/peer"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/status"
)

// tickerResolution is the delta to check blockchain sync progress.
const tickerResolution = time.Second

// errors
var (
	ErrNodeRunning            = errors.New("node is already running")
	ErrNoGethNode             = errors.New("geth node is not available")
	ErrNoRunningNode          = errors.New("there is no running node")
	ErrAccountKeyStoreMissing = errors.New("account key store is not set")
	ErrServiceUnknown         = errors.New("service unknown")
	ErrDiscoveryRunning       = errors.New("discovery is already running")
)

// StatusNode abstracts contained geth node and provides helper methods to
// interact with it.
type StatusNode struct {
	mu sync.RWMutex

	config           *params.NodeConfig // Status node configuration
	gethNode         *node.Node         // reference to Geth P2P stack/node
	rpcClient        *rpc.Client        // reference to public RPC client
	rpcPrivateClient *rpc.Client        // reference to private RPC client (can call private APIs)

	discovery discovery.Discovery
	register  *peers.Register
	peerPool  *peers.PeerPool
	db        *leveldb.DB // used as a cache for PeerPool

	log log.Logger
}

// New makes new instance of StatusNode.
func New() *StatusNode {
	return &StatusNode{
		log: log.New("package", "status-go/node.StatusNode"),
	}
}

// Config exposes reference to running node's configuration
func (n *StatusNode) Config() *params.NodeConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.config
}

// GethNode returns underlying geth node.
func (n *StatusNode) GethNode() *node.Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.gethNode
}

// Server retrieves the currently running P2P network layer.
func (n *StatusNode) Server() *p2p.Server {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.gethNode == nil {
		return nil
	}

	return n.gethNode.Server()
}

// Start starts current StatusNode, failing if it's already started.
// It accepts a list of services that should be added to the node.
func (n *StatusNode) Start(config *params.NodeConfig, services ...node.ServiceConstructor) error {
	return n.StartWithOptions(config, StartOptions{
		Services:       services,
		StartDiscovery: true,
	})
}

// StartOptions allows to control some parameters of Start() method.
type StartOptions struct {
	Services       []node.ServiceConstructor
	StartDiscovery bool
}

// StartWithOptions starts current StatusNode, failing if it's already started.
// It takes some options that allows to further configure starting process.
func (n *StatusNode) StartWithOptions(config *params.NodeConfig, options StartOptions) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.isRunning() {
		n.log.Debug("cannot start, node already running")
		return ErrNodeRunning
	}

	n.log.Debug("starting with NodeConfig", "ClusterConfig", config.ClusterConfig)

	db, err := db.Create(config.DataDir, params.StatusDatabase)
	if err != nil {
		return err
	}

	n.db = db

	err = n.startWithDB(config, db, options.Services)

	// continue only if there was no error when starting node with a db
	if err == nil && options.StartDiscovery && n.discoveryEnabled() {
		err = n.startDiscovery()
	}

	if err != nil {
		if dberr := db.Close(); dberr != nil {
			n.log.Error("error while closing leveldb after node crash", "error", dberr)
		}
		n.db = nil
		return err
	}

	return nil
}

func (n *StatusNode) startWithDB(config *params.NodeConfig, db *leveldb.DB, services []node.ServiceConstructor) error {
	if err := n.createNode(config, db); err != nil {
		return err
	}
	n.config = config

	if err := n.start(services); err != nil {
		return err
	}

	if err := n.setupRPCClient(); err != nil {
		return err
	}

	return nil
}

func (n *StatusNode) createNode(config *params.NodeConfig, db *leveldb.DB) (err error) {
	n.gethNode, err = MakeNode(config, db)
	return err
}

// start starts current StatusNode, will fail if it's already started.
func (n *StatusNode) start(services []node.ServiceConstructor) error {
	for _, service := range services {
		if err := n.gethNode.Register(service); err != nil {
			return err
		}
	}

	return n.gethNode.Start()
}

func (n *StatusNode) setupRPCClient() (err error) {
	// setup public RPC client
	gethNodeClient, err := n.gethNode.AttachPublic()
	if err != nil {
		return
	}
	n.rpcClient, err = rpc.NewClient(gethNodeClient, n.config.UpstreamConfig)
	if err != nil {
		return
	}

	// setup private RPC client
	gethNodePrivateClient, err := n.gethNode.Attach()
	if err != nil {
		return
	}
	n.rpcPrivateClient, err = rpc.NewClient(gethNodePrivateClient, n.config.UpstreamConfig)

	return
}

func (n *StatusNode) discoveryEnabled() bool {
	return n.config != nil && (!n.config.NoDiscovery || n.config.Rendezvous) && n.config.ClusterConfig.Enabled
}

func (n *StatusNode) discoverNode() (*enode.Node, error) {
	if !n.isRunning() {
		return nil, nil
	}

	server := n.gethNode.Server()
	discNode := server.Self()

	if n.config.AdvertiseAddr == "" {
		return discNode, nil
	}

	n.log.Info("Using AdvertiseAddr for rendezvous", "addr", n.config.AdvertiseAddr)

	r := discNode.Record()
	r.Set(enr.IP(net.ParseIP(n.config.AdvertiseAddr)))
	if err := enode.SignV4(r, server.PrivateKey); err != nil {
		return nil, err
	}
	return enode.New(enode.ValidSchemes[r.IdentityScheme()], r)
}

func (n *StatusNode) startRendezvous() (discovery.Discovery, error) {
	if !n.config.Rendezvous {
		return nil, errors.New("rendezvous is not enabled")
	}
	if len(n.config.ClusterConfig.RendezvousNodes) == 0 {
		return nil, errors.New("rendezvous node must be provided if rendezvous discovery is enabled")
	}
	maddrs := make([]ma.Multiaddr, len(n.config.ClusterConfig.RendezvousNodes))
	for i, addr := range n.config.ClusterConfig.RendezvousNodes {
		var err error
		maddrs[i], err = ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rendezvous node %s: %v", n.config.ClusterConfig.RendezvousNodes[0], err)
		}
	}
	node, err := n.discoverNode()
	if err != nil {
		return nil, fmt.Errorf("failed to get a discover node: %v", err)
	}

	return discovery.NewRendezvous(maddrs, n.gethNode.Server().PrivateKey, node)
}

// StartDiscovery starts the peers discovery protocols depending on the node config.
func (n *StatusNode) StartDiscovery() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.discoveryEnabled() {
		return n.startDiscovery()
	}

	return nil
}

func (n *StatusNode) startDiscovery() error {
	if n.isDiscoveryRunning() {
		return ErrDiscoveryRunning
	}

	discoveries := []discovery.Discovery{}
	if !n.config.NoDiscovery {
		discoveries = append(discoveries, discovery.NewDiscV5(
			n.gethNode.Server().PrivateKey,
			n.config.ListenAddr,
			parseNodesV5(n.config.ClusterConfig.BootNodes)))
	}
	if n.config.Rendezvous {
		d, err := n.startRendezvous()
		if err != nil {
			return err
		}
		discoveries = append(discoveries, d)
	}
	if len(discoveries) == 0 {
		return errors.New("wasn't able to register any discovery")
	} else if len(discoveries) > 1 {
		n.discovery = discovery.NewMultiplexer(discoveries)
	} else {
		n.discovery = discoveries[0]
	}
	log.Debug(
		"using discovery",
		"instance", reflect.TypeOf(n.discovery),
		"registerTopics", n.config.RegisterTopics,
		"requireTopics", n.config.RequireTopics,
	)
	n.register = peers.NewRegister(n.discovery, n.config.RegisterTopics...)
	options := peers.NewDefaultOptions()
	// TODO(dshulyak) consider adding a flag to define this behaviour
	options.AllowStop = len(n.config.RegisterTopics) == 0
	options.TrustedMailServers = parseNodesToNodeID(n.config.ClusterConfig.TrustedMailServers)

	options.MailServerRegistryAddress = n.config.MailServerRegistryAddress

	n.peerPool = peers.NewPeerPool(
		n.discovery,
		n.config.RequireTopics,
		peers.NewCache(n.db),
		options,
	)
	if err := n.discovery.Start(); err != nil {
		return err
	}
	if err := n.register.Start(); err != nil {
		return err
	}
	return n.peerPool.Start(n.gethNode.Server(), n.rpcClient)
}

// Stop will stop current StatusNode. A stopped node cannot be resumed.
func (n *StatusNode) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.isRunning() {
		return ErrNoRunningNode
	}

	return n.stop()
}

// stop will stop current StatusNode. A stopped node cannot be resumed.
func (n *StatusNode) stop() error {
	if n.isDiscoveryRunning() {
		if err := n.stopDiscovery(); err != nil {
			n.log.Error("Error stopping the discovery components", "error", err)
		}
		n.register = nil
		n.peerPool = nil
		n.discovery = nil
	}

	if err := n.gethNode.Stop(); err != nil {
		return err
	}

	n.rpcClient = nil
	n.rpcPrivateClient = nil
	// We need to clear `gethNode` because config is passed to `Start()`
	// and may be completely different. Similarly with `config`.
	n.gethNode = nil
	n.config = nil

	if n.db != nil {
		err := n.db.Close()

		n.db = nil

		return err
	}

	return nil
}

func (n *StatusNode) isDiscoveryRunning() bool {
	return n.register != nil || n.peerPool != nil || n.discovery != nil
}

func (n *StatusNode) stopDiscovery() error {
	n.register.Stop()
	n.peerPool.Stop()
	return n.discovery.Stop()
}

// ResetChainData removes chain data if node is not running.
func (n *StatusNode) ResetChainData(config *params.NodeConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.isRunning() {
		return ErrNodeRunning
	}

	chainDataDir := filepath.Join(config.DataDir, config.Name, "lightchaindata")
	if _, err := os.Stat(chainDataDir); os.IsNotExist(err) {
		return err
	}
	err := os.RemoveAll(chainDataDir)
	if err == nil {
		n.log.Info("Chain data has been removed", "dir", chainDataDir)
	}
	return err
}

// IsRunning confirm that node is running.
func (n *StatusNode) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.isRunning()
}

func (n *StatusNode) isRunning() bool {
	return n.gethNode != nil && n.gethNode.Server() != nil
}

// populateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (n *StatusNode) populateStaticPeers() error {
	if !n.config.ClusterConfig.Enabled {
		n.log.Info("Static peers are disabled")
		return nil
	}

	for _, enode := range n.config.ClusterConfig.StaticNodes {
		if err := n.addPeer(enode); err != nil {
			n.log.Error("Static peer addition failed", "error", err)
			return err
		}
		n.log.Info("Static peer added", "enode", enode)
	}

	return nil
}

func (n *StatusNode) removeStaticPeers() error {
	if !n.config.ClusterConfig.Enabled {
		n.log.Info("Static peers are disabled")
		return nil
	}

	for _, enode := range n.config.ClusterConfig.StaticNodes {
		if err := n.removePeer(enode); err != nil {
			n.log.Error("Static peer deletion failed", "error", err)
			return err
		}
		n.log.Info("Static peer deleted", "enode", enode)
	}
	return nil
}

// ReconnectStaticPeers removes and adds static peers to a server.
func (n *StatusNode) ReconnectStaticPeers() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.isRunning() {
		return ErrNoRunningNode
	}

	if err := n.removeStaticPeers(); err != nil {
		return err
	}

	return n.populateStaticPeers()
}

// AddPeer adds new static peer node
func (n *StatusNode) AddPeer(url string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.addPeer(url)
}

// addPeer adds new static peer node
func (n *StatusNode) addPeer(url string) error {
	parsedNode, err := enode.ParseV4(url)
	if err != nil {
		return err
	}

	if !n.isRunning() {
		return ErrNoRunningNode
	}

	n.gethNode.Server().AddPeer(parsedNode)

	return nil
}

func (n *StatusNode) removePeer(url string) error {
	parsedNode, err := enode.ParseV4(url)
	if err != nil {
		return err
	}

	if !n.isRunning() {
		return ErrNoRunningNode
	}

	n.gethNode.Server().RemovePeer(parsedNode)

	return nil
}

// PeerCount returns the number of connected peers.
func (n *StatusNode) PeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.isRunning() {
		return 0
	}

	return n.gethNode.Server().PeerCount()
}

// gethService is a wrapper for gethNode.Service which retrieves a currently
// running service registered of a specific type.
func (n *StatusNode) gethService(serviceInstance interface{}) error {
	if !n.isRunning() {
		return ErrNoRunningNode
	}

	if err := n.gethNode.Service(serviceInstance); err != nil {
		return err
	}

	return nil
}

// LightEthereumService exposes reference to LES service running on top of the node
func (n *StatusNode) LightEthereumService() (l *les.LightEthereum, err error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	err = n.gethService(&l)
	if err == node.ErrServiceUnknown {
		err = ErrServiceUnknown
	}

	return
}

// StatusService exposes reference to status service running on top of the node
func (n *StatusNode) StatusService() (st *status.Service, err error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	err = n.gethService(&st)
	if err == node.ErrServiceUnknown {
		err = ErrServiceUnknown
	}

	return
}

// PeerService exposes reference to peer service running on top of the node.
func (n *StatusNode) PeerService() (st *peer.Service, err error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	err = n.gethService(&st)
	if err == node.ErrServiceUnknown {
		err = ErrServiceUnknown
	}

	return
}

// WhisperService exposes reference to Whisper service running on top of the node
func (n *StatusNode) WhisperService() (w *whisper.Whisper, err error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	err = n.gethService(&w)
	if err == node.ErrServiceUnknown {
		err = ErrServiceUnknown
	}

	return
}

// ShhExtService exposes reference to shh extension service running on top of the node
func (n *StatusNode) ShhExtService() (s *shhext.Service, err error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	err = n.gethService(&s)
	if err == node.ErrServiceUnknown {
		err = ErrServiceUnknown
	}

	return
}

// AccountManager exposes reference to node's accounts manager
func (n *StatusNode) AccountManager() (*accounts.Manager, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.gethNode == nil {
		return nil, ErrNoGethNode
	}

	return n.gethNode.AccountManager(), nil
}

// AccountKeyStore exposes reference to accounts key store
func (n *StatusNode) AccountKeyStore() (*keystore.KeyStore, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.gethNode == nil {
		return nil, ErrNoGethNode
	}

	accountManager := n.gethNode.AccountManager()
	backends := accountManager.Backends(keystore.KeyStoreType)
	if len(backends) == 0 {
		return nil, ErrAccountKeyStoreMissing
	}

	keyStore, ok := backends[0].(*keystore.KeyStore)
	if !ok {
		return nil, ErrAccountKeyStoreMissing
	}

	return keyStore, nil
}

// RPCClient exposes reference to RPC client connected to the running node.
func (n *StatusNode) RPCClient() *rpc.Client {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.rpcClient
}

// RPCPrivateClient exposes reference to RPC client connected to the running node
// that can call both public and private APIs.
func (n *StatusNode) RPCPrivateClient() *rpc.Client {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.rpcPrivateClient
}

// ChangeRPCClientsUpstreamURL updates RPCClient and RPCPrivateClient upstream URLs,
// if defined, without restarting the node.
// This is required for the Chaos Unicorn Day
func (n *StatusNode) ChangeRPCClientsUpstreamURL(url string) error {
	publicClient := n.RPCClient()
	if publicClient != nil {
		if err := publicClient.UpdateUpstreamURL(url); err != nil {
			return err
		}
	}

	privateClient := n.RPCPrivateClient()
	if privateClient != nil {
		if err := privateClient.UpdateUpstreamURL(url); err != nil {
			return err
		}
	}

	return nil
}

// EnsureSync waits until blockchain synchronization
// is complete and returns.
func (n *StatusNode) EnsureSync(ctx context.Context) error {
	// Don't wait for any blockchain sync for the
	// local private chain as blocks are never mined.
	if n.config.NetworkID == 0 || n.config.NetworkID == params.StatusChainNetworkID {
		return nil
	}

	return n.ensureSync(ctx)
}

func (n *StatusNode) ensureSync(ctx context.Context) error {
	les, err := n.LightEthereumService()
	if err != nil {
		return fmt.Errorf("failed to get LES service: %v", err)
	}

	downloader := les.Downloader()
	if downloader == nil {
		return errors.New("LightEthereumService downloader is nil")
	}

	progress := downloader.Progress()
	if n.PeerCount() > 0 && progress.CurrentBlock >= progress.HighestBlock {
		n.log.Debug("Synchronization completed", "current block", progress.CurrentBlock, "highest block", progress.HighestBlock)
		return nil
	}

	ticker := time.NewTicker(tickerResolution)
	defer ticker.Stop()

	progressTicker := time.NewTicker(time.Minute)
	defer progressTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout during node synchronization")
		case <-ticker.C:
			if n.PeerCount() == 0 {
				n.log.Debug("No established connections with any peers, continue waiting for a sync")
				continue
			}
			if downloader.Synchronising() {
				n.log.Debug("Synchronization is in progress")
				continue
			}
			progress = downloader.Progress()
			if progress.CurrentBlock >= progress.HighestBlock {
				n.log.Info("Synchronization completed", "current block", progress.CurrentBlock, "highest block", progress.HighestBlock)
				return nil
			}
			n.log.Debug("Synchronization is not finished", "current", progress.CurrentBlock, "highest", progress.HighestBlock)
		case <-progressTicker.C:
			progress = downloader.Progress()
			n.log.Warn("Synchronization is not finished", "current", progress.CurrentBlock, "highest", progress.HighestBlock)
		}
	}
}

// Discover sets up the discovery for a specific topic.
func (n *StatusNode) Discover(topic string, max, min int) (err error) {
	if n.peerPool == nil {
		return errors.New("peerPool not running")
	}
	return n.peerPool.UpdateTopic(topic, params.Limits{
		Max: max,
		Min: min,
	})
}
