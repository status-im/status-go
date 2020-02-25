// +build nimbus

package node

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/db"
	nimbusbridge "github.com/status-im/status-go/eth-node/bridge/nimbus"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	nimbussvc "github.com/status-im/status-go/services/nimbus"
	"github.com/status-im/status-go/services/nodebridge"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/status"
)

// // tickerResolution is the delta to check blockchain sync progress.
// const tickerResolution = time.Second

// errors
var (
	ErrNodeRunning    = errors.New("node is already running")
	ErrNodeStopped    = errors.New("node not started")
	ErrNoRunningNode  = errors.New("there is no running node")
	ErrServiceUnknown = errors.New("service unknown")
)

// NimbusStatusNode abstracts contained geth node and provides helper methods to
// interact with it.
type NimbusStatusNode struct {
	mu sync.RWMutex

	//eventmux *event.TypeMux // Event multiplexer used between the services of a stack

	config           *params.NodeConfig // Status node configuration
	privateKey       *ecdsa.PrivateKey
	node             nimbusbridge.Node
	nodeRunning      bool
	rpcClient        *rpc.Client // reference to public RPC client
	rpcPrivateClient *rpc.Client // reference to private RPC client (can call private APIs)

	rpcAPIs             []gethrpc.API   // List of APIs currently provided by the node
	inprocHandler       *gethrpc.Server // In-process RPC request handler to process the API requests
	inprocPublicHandler *gethrpc.Server // In-process RPC request handler to process the public API requests

	serviceFuncs []nimbussvc.ServiceConstructor     // Service constructors (in dependency order)
	services     map[reflect.Type]nimbussvc.Service // Currently running services

	// discovery discovery.Discovery
	// register  *peers.Register
	// peerPool  *peers.PeerPool
	db *leveldb.DB // used as a cache for PeerPool

	//stop chan struct{} // Channel to wait for termination notifications
	lock sync.RWMutex

	log log.Logger
}

// NewNimbus makes new instance of NimbusStatusNode.
func NewNimbus() *NimbusStatusNode {
	return &NimbusStatusNode{
		//eventmux:          new(event.TypeMux),
		log: log.New("package", "status-go/node.NimbusStatusNode"),
	}
}

// Config exposes reference to running node's configuration
func (n *NimbusStatusNode) Config() *params.NodeConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.config
}

// GethNode returns underlying geth node.
// func (n *NimbusStatusNode) GethNode() *node.Node {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()

// 	return n.gethNode
// }

// Server retrieves the currently running P2P network layer.
// func (n *NimbusStatusNode) Server() *p2p.Server {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()

// 	if n.gethNode == nil {
// 		return nil
// 	}

// 	return n.gethNode.Server()
// }

// Start starts current NimbusStatusNode, failing if it's already started.
// It accepts a list of services that should be added to the node.
func (n *NimbusStatusNode) Start(config *params.NodeConfig, services ...nimbussvc.ServiceConstructor) error {
	panic("Start")
	return n.StartWithOptions(config, NimbusStartOptions{
		Services:       services,
		StartDiscovery: true,
		// AccountsManager: accs,
	})
}

// NimbusStartOptions allows to control some parameters of Start() method.
type NimbusStartOptions struct {
	Node           types.Node
	Services       []nimbussvc.ServiceConstructor
	StartDiscovery bool
	// AccountsManager *accounts.Manager
}

// StartWithOptions starts current NimbusStatusNode, failing if it's already started.
// It takes some options that allows to further configure starting process.
func (n *NimbusStatusNode) StartWithOptions(config *params.NodeConfig, options NimbusStartOptions) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.isRunning() {
		n.log.Debug("node is already running")
		return ErrNodeRunning
	}

	n.log.Debug("starting with NodeConfig", "ClusterConfig", config.ClusterConfig)

	db, err := db.Create(config.DataDir, params.StatusDatabase)
	if err != nil {
		return fmt.Errorf("failed to create database at %s: %v", config.DataDir, err)
	}

	n.db = db

	err = n.startWithDB(config, db, options.Services)

	// continue only if there was no error when starting node with a db
	if err == nil && options.StartDiscovery && n.discoveryEnabled() {
		// err = n.startDiscovery()
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

func (n *NimbusStatusNode) startWithDB(config *params.NodeConfig, db *leveldb.DB, services []nimbussvc.ServiceConstructor) error {
	if err := n.createNode(config, services, db); err != nil {
		return err
	}

	if err := n.setupRPCClient(); err != nil {
		return err
	}

	return nil
}

func (n *NimbusStatusNode) createNode(config *params.NodeConfig, services []nimbussvc.ServiceConstructor, db *leveldb.DB) error {
	var privateKey *ecdsa.PrivateKey
	if config.NodeKey != "" {
		var err error
		privateKey, err = crypto.HexToECDSA(config.NodeKey)
		if err != nil {
			return err
		}
	}

	n.privateKey = privateKey
	n.node = nimbusbridge.NewNodeBridge()

	err := n.activateServices(config, db)
	if err != nil {
		return err
	}

	if err = n.start(config, services); err != nil {
		return err
	}

	return nil
}

// start starts current NimbusStatusNode, will fail if it's already started.
func (n *NimbusStatusNode) start(config *params.NodeConfig, services []nimbussvc.ServiceConstructor) error {
	for _, service := range services {
		if err := n.Register(service); err != nil {
			return err
		}
	}

	n.config = config
	n.startServices()

	err := n.node.StartNimbus(n.privateKey, config.ListenAddr, true)
	n.nodeRunning = err == nil
	return err
}

func (n *NimbusStatusNode) setupRPCClient() (err error) {
	// setup public RPC client
	gethNodeClient := gethrpc.DialInProc(n.inprocPublicHandler)
	n.rpcClient, err = rpc.NewClient(gethNodeClient, n.config.UpstreamConfig)
	if err != nil {
		return
	}

	// setup private RPC client
	gethNodePrivateClient := gethrpc.DialInProc(n.inprocHandler)
	n.rpcPrivateClient, err = rpc.NewClient(gethNodePrivateClient, n.config.UpstreamConfig)

	return
}

func (n *NimbusStatusNode) discoveryEnabled() bool {
	return n.config != nil && (!n.config.NoDiscovery || n.config.Rendezvous) && n.config.ClusterConfig.Enabled
}

// func (n *NimbusStatusNode) discoverNode() (*enode.Node, error) {
// 	if !n.isRunning() {
// 		return nil, nil
// 	}

// 	server := n.gethNode.Server()
// 	discNode := server.Self()

// 	if n.config.AdvertiseAddr == "" {
// 		return discNode, nil
// 	}

// 	n.log.Info("Using AdvertiseAddr for rendezvous", "addr", n.config.AdvertiseAddr)

// 	r := discNode.Record()
// 	r.Set(enr.IP(net.ParseIP(n.config.AdvertiseAddr)))
// 	if err := enode.SignV4(r, server.PrivateKey); err != nil {
// 		return nil, err
// 	}
// 	return enode.New(enode.ValidSchemes[r.IdentityScheme()], r)
// }

// func (n *NimbusStatusNode) startRendezvous() (discovery.Discovery, error) {
// 	if !n.config.Rendezvous {
// 		return nil, errors.New("rendezvous is not enabled")
// 	}
// 	if len(n.config.ClusterConfig.RendezvousNodes) == 0 {
// 		return nil, errors.New("rendezvous node must be provided if rendezvous discovery is enabled")
// 	}
// 	maddrs := make([]ma.Multiaddr, len(n.config.ClusterConfig.RendezvousNodes))
// 	for i, addr := range n.config.ClusterConfig.RendezvousNodes {
// 		var err error
// 		maddrs[i], err = ma.NewMultiaddr(addr)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to parse rendezvous node %s: %v", n.config.ClusterConfig.RendezvousNodes[0], err)
// 		}
// 	}
// 	node, err := n.discoverNode()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get a discover node: %v", err)
// 	}

// 	return discovery.NewRendezvous(maddrs, n.gethNode.Server().PrivateKey, node)
// }

// StartDiscovery starts the peers discovery protocols depending on the node config.
func (n *NimbusStatusNode) StartDiscovery() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.discoveryEnabled() {
		// return n.startDiscovery()
	}

	return nil
}

// func (n *NimbusStatusNode) startDiscovery() error {
// 	if n.isDiscoveryRunning() {
// 		return ErrDiscoveryRunning
// 	}

// 	discoveries := []discovery.Discovery{}
// 	if !n.config.NoDiscovery {
// 		discoveries = append(discoveries, discovery.NewDiscV5(
// 			n.gethNode.Server().PrivateKey,
// 			n.config.ListenAddr,
// 			parseNodesV5(n.config.ClusterConfig.BootNodes)))
// 	}
// 	if n.config.Rendezvous {
// 		d, err := n.startRendezvous()
// 		if err != nil {
// 			return err
// 		}
// 		discoveries = append(discoveries, d)
// 	}
// 	if len(discoveries) == 0 {
// 		return errors.New("wasn't able to register any discovery")
// 	} else if len(discoveries) > 1 {
// 		n.discovery = discovery.NewMultiplexer(discoveries)
// 	} else {
// 		n.discovery = discoveries[0]
// 	}
// 	log.Debug(
// 		"using discovery",
// 		"instance", reflect.TypeOf(n.discovery),
// 		"registerTopics", n.config.RegisterTopics,
// 		"requireTopics", n.config.RequireTopics,
// 	)
// 	n.register = peers.NewRegister(n.discovery, n.config.RegisterTopics...)
// 	options := peers.NewDefaultOptions()
// 	// TODO(dshulyak) consider adding a flag to define this behaviour
// 	options.AllowStop = len(n.config.RegisterTopics) == 0
// 	options.TrustedMailServers = parseNodesToNodeID(n.config.ClusterConfig.TrustedMailServers)

// 	options.MailServerRegistryAddress = n.config.MailServerRegistryAddress

// 	n.peerPool = peers.NewPeerPool(
// 		n.discovery,
// 		n.config.RequireTopics,
// 		peers.NewCache(n.db),
// 		options,
// 	)
// 	if err := n.discovery.Start(); err != nil {
// 		return err
// 	}
// 	if err := n.register.Start(); err != nil {
// 		return err
// 	}
// 	return n.peerPool.Start(n.gethNode.Server(), n.rpcClient)
// }

// Stop will stop current NimbusStatusNode. A stopped node cannot be resumed.
func (n *NimbusStatusNode) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.isRunning() {
		return ErrNoRunningNode
	}

	var errs []error

	// Terminate all subsystems and collect any errors
	if err := n.stop(); err != nil && err != ErrNodeStopped {
		errs = append(errs, err)
	}
	// Report any errors that might have occurred
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return fmt.Errorf("%v", errs)
	}
}

// StopError is returned if a Node fails to stop either any of its registered
// services or itself.
type StopError struct {
	Server   error
	Services map[reflect.Type]error
}

// Error generates a textual representation of the stop error.
func (e *StopError) Error() string {
	return fmt.Sprintf("server: %v, services: %v", e.Server, e.Services)
}

// stop will stop current NimbusStatusNode. A stopped node cannot be resumed.
func (n *NimbusStatusNode) stop() error {
	// if n.isDiscoveryRunning() {
	// 	if err := n.stopDiscovery(); err != nil {
	// 		n.log.Error("Error stopping the discovery components", "error", err)
	// 	}
	// n.register = nil
	// n.peerPool = nil
	// n.discovery = nil
	// }

	// if err := n.gethNode.Stop(); err != nil {
	// 	return err
	// }

	// Terminate the API, services and the p2p server.
	n.stopPublicInProc()
	n.stopInProc()
	n.rpcClient = nil
	n.rpcPrivateClient = nil
	n.rpcAPIs = nil

	failure := &StopError{
		Services: make(map[reflect.Type]error),
	}
	for kind, service := range n.services {
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	n.services = nil
	// We need to clear `node` because config is passed to `Start()`
	// and may be completely different. Similarly with `config`.
	if n.node != nil {
		n.node.Stop()
		n.node = nil
	}
	n.nodeRunning = false
	n.config = nil

	if n.db != nil {
		err := n.db.Close()

		n.db = nil

		return err
	}

	if len(failure.Services) > 0 {
		return failure
	}
	return nil
}

func (n *NimbusStatusNode) isDiscoveryRunning() bool {
	return false //n.register != nil || n.peerPool != nil || n.discovery != nil
}

// func (n *NimbusStatusNode) stopDiscovery() error {
// 	n.register.Stop()
// 	n.peerPool.Stop()
// 	return n.discovery.Stop()
// }

// ResetChainData removes chain data if node is not running.
func (n *NimbusStatusNode) ResetChainData(config *params.NodeConfig) error {
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
func (n *NimbusStatusNode) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.isRunning()
}

func (n *NimbusStatusNode) isRunning() bool {
	return n.node != nil && n.nodeRunning // && n.gethNode.Server() != nil
}

// populateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (n *NimbusStatusNode) populateStaticPeers() error {
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

func (n *NimbusStatusNode) removeStaticPeers() error {
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
func (n *NimbusStatusNode) ReconnectStaticPeers() error {
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
func (n *NimbusStatusNode) AddPeer(url string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.addPeer(url)
}

// addPeer adds new static peer node
func (n *NimbusStatusNode) addPeer(url string) error {
	if !n.isRunning() {
		return ErrNoRunningNode
	}

	n.node.AddPeer(url)

	return nil
}

func (n *NimbusStatusNode) removePeer(url string) error {
	if !n.isRunning() {
		return ErrNoRunningNode
	}

	n.node.RemovePeer(url)

	return nil
}

// PeerCount returns the number of connected peers.
func (n *NimbusStatusNode) PeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.isRunning() {
		return 0
	}

	return 1
	//return n.gethNode.Server().PeerCount()
}

// Service retrieves a currently running service registered of a specific type.
func (n *NimbusStatusNode) Service(service interface{}) error {
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Short circuit if the node's not running
	if !n.isRunning() {
		return ErrNodeStopped
	}
	// Otherwise try to find the service to return
	element := reflect.ValueOf(service).Elem()
	if running, ok := n.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
}

// // LightEthereumService exposes reference to LES service running on top of the node
// func (n *NimbusStatusNode) LightEthereumService() (l *les.LightEthereum, err error) {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()

// 	err = n.Service(&l)

// 	return
// }

// StatusService exposes reference to status service running on top of the node
func (n *NimbusStatusNode) StatusService() (st *status.Service, err error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	err = n.Service(&st)

	return
}

// // PeerService exposes reference to peer service running on top of the node.
// func (n *NimbusStatusNode) PeerService() (st *peer.Service, err error) {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()

// 	err = n.Service(&st)

// 	return
// }

// WhisperService exposes reference to Whisper service running on top of the node
func (n *NimbusStatusNode) WhisperService() (w *nodebridge.WhisperService, err error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	err = n.Service(&w)

	return
}

// ShhExtService exposes reference to shh extension service running on top of the node
func (n *NimbusStatusNode) ShhExtService() (s *shhext.NimbusService, err error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	err = n.Service(&s)

	return
}

// // WalletService returns wallet.Service instance if it was started.
// func (n *NimbusStatusNode) WalletService() (s *wallet.Service, err error) {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()
// 	err = n.Service(&s)
// 	return
// }

// // BrowsersService returns browsers.Service instance if it was started.
// func (n *NimbusStatusNode) BrowsersService() (s *browsers.Service, err error) {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()
// 	err = n.Service(&s)
// 	return
// }

// // PermissionsService returns browsers.Service instance if it was started.
// func (n *NimbusStatusNode) PermissionsService() (s *permissions.Service, err error) {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()
// 	err = n.Service(&s)
// 	return
// }

// // AccountManager exposes reference to node's accounts manager
// func (n *NimbusStatusNode) AccountManager() (*accounts.Manager, error) {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()

// 	if n.gethNode == nil {
// 		return nil, ErrNoGethNode
// 	}

// 	return n.gethNode.AccountManager(), nil
// }

// RPCClient exposes reference to RPC client connected to the running node.
func (n *NimbusStatusNode) RPCClient() *rpc.Client {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.rpcClient
}

// RPCPrivateClient exposes reference to RPC client connected to the running node
// that can call both public and private APIs.
func (n *NimbusStatusNode) RPCPrivateClient() *rpc.Client {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.rpcPrivateClient
}

// ChaosModeCheckRPCClientsUpstreamURL updates RPCClient and RPCPrivateClient upstream URLs,
// if defined, without restarting the node. This is required for the Chaos Unicorn Day.
// Additionally, if the passed URL is Infura, it changes it to httpbin.org/status/500.
func (n *NimbusStatusNode) ChaosModeCheckRPCClientsUpstreamURL(on bool) error {
	url := n.config.UpstreamConfig.URL

	if on {
		if strings.Contains(url, "infura.io") {
			url = "https://httpbin.org/status/500"
		}
	}

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
func (n *NimbusStatusNode) EnsureSync(ctx context.Context) error {
	// Don't wait for any blockchain sync for the
	// local private chain as blocks are never mined.
	if n.config.NetworkID == 0 || n.config.NetworkID == params.StatusChainNetworkID {
		return nil
	}

	return n.ensureSync(ctx)
}

func (n *NimbusStatusNode) ensureSync(ctx context.Context) error {
	return errors.New("Sync not implemented")
	// les, err := n.LightEthereumService()
	// if err != nil {
	// 	return fmt.Errorf("failed to get LES service: %v", err)
	// }

	// downloader := les.Downloader()
	// if downloader == nil {
	// 	return errors.New("LightEthereumService downloader is nil")
	// }

	// progress := downloader.Progress()
	// if n.PeerCount() > 0 && progress.CurrentBlock >= progress.HighestBlock {
	// 	n.log.Debug("Synchronization completed", "current block", progress.CurrentBlock, "highest block", progress.HighestBlock)
	// 	return nil
	// }

	// ticker := time.NewTicker(tickerResolution)
	// defer ticker.Stop()

	// progressTicker := time.NewTicker(time.Minute)
	// defer progressTicker.Stop()

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		return errors.New("timeout during node synchronization")
	// 	case <-ticker.C:
	// 		if n.PeerCount() == 0 {
	// 			n.log.Debug("No established connections with any peers, continue waiting for a sync")
	// 			continue
	// 		}
	// 		if downloader.Synchronising() {
	// 			n.log.Debug("Synchronization is in progress")
	// 			continue
	// 		}
	// 		progress = downloader.Progress()
	// 		if progress.CurrentBlock >= progress.HighestBlock {
	// 			n.log.Info("Synchronization completed", "current block", progress.CurrentBlock, "highest block", progress.HighestBlock)
	// 			return nil
	// 		}
	// 		n.log.Debug("Synchronization is not finished", "current", progress.CurrentBlock, "highest", progress.HighestBlock)
	// 	case <-progressTicker.C:
	// 		progress = downloader.Progress()
	// 		n.log.Warn("Synchronization is not finished", "current", progress.CurrentBlock, "highest", progress.HighestBlock)
	// 	}
	// }
}

// // Discover sets up the discovery for a specific topic.
// func (n *NimbusStatusNode) Discover(topic string, max, min int) (err error) {
// 	if n.peerPool == nil {
// 		return errors.New("peerPool not running")
// 	}
// 	return n.peerPool.UpdateTopic(topic, params.Limits{
// 		Max: max,
// 		Min: min,
// 	})
// }
