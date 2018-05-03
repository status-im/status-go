package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/status-im/status-go/geth/db"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/peers"
	"github.com/status-im/status-go/geth/rpc"
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
)

// StatusNode abstracts contained geth node and provides helper methods to
// interact with it.
type StatusNode struct {
	mu sync.RWMutex

	config           *params.NodeConfig // Status node configuration
	gethNode         *node.Node         // reference to Geth P2P stack/node
	rpcClient        *rpc.Client        // reference to public RPC client
	rpcPrivateClient *rpc.Client        // reference to private RPC client (can call private APIs)

	register *peers.Register
	peerPool *peers.PeerPool
	db       *leveldb.DB // used as a cache for PeerPool

	log log.Logger
}

// New makes new instance of StatusNode.
func New() *StatusNode {
	return &StatusNode{
		log: log.New("package", "status-go/geth/node.StatusNode"),
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

// Start starts current StatusNode, will fail if it's already started.
func (n *StatusNode) Start(config *params.NodeConfig, services ...node.ServiceConstructor) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.isRunning() {
		return ErrNodeRunning
	}

	if err := n.createNode(config); err != nil {
		return err
	}
	n.config = config

	if err := n.start(services); err != nil {
		return err
	}

	if err := n.setupRPCClient(); err != nil {
		return err
	}

	statusDB, err := db.Create(n.config.DataDir, params.StatusDatabase)
	if err != nil {
		return err
	}

	n.db = statusDB

	if err := n.setupDeduplicator(); err != nil {
		return err
	}

	if n.config.Discovery {
		return n.startPeerPool()
	}

	return nil
}

func (n *StatusNode) setupDeduplicator() error {
	var s shhext.Service

	err := n.gethService(&s)
	if err == node.ErrServiceUnknown {
		return nil
	}
	if err != nil {
		return err
	}

	return s.Deduplicator.Start(n.db)
}

func (n *StatusNode) createNode(config *params.NodeConfig) (err error) {
	n.gethNode, err = MakeNode(config)
	return
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

func (n *StatusNode) startPeerPool() error {
	n.register = peers.NewRegister(n.config.RegisterTopics...)
	// TODO(dshulyak) consider adding a flag to define this behaviour
	stopOnMax := len(n.config.RegisterTopics) == 0
	n.peerPool = peers.NewPeerPool(n.config.RequireTopics,
		peers.DefaultFastSync,
		peers.DefaultSlowSync,
		peers.NewCache(n.db),
		stopOnMax,
	)
	if err := n.register.Start(n.gethNode.Server()); err != nil {
		return err
	}
	return n.peerPool.Start(n.gethNode.Server())
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
	if err := n.stopPeerPool(); err != nil {
		n.log.Error("Error stopping the PeerPool", "error", err)
	}
	n.register = nil
	n.peerPool = nil

	if err := n.gethNode.Stop(); err != nil {
		return err
	}

	n.rpcClient = nil
	n.rpcPrivateClient = nil
	// We need to clear `gethNode` because config is passed to `Start()`
	// and may be completely different. Similarly with `config`.
	n.gethNode = nil
	n.config = nil

	if err := n.db.Close(); err != nil {
		return err
	}
	n.db = nil

	return nil
}

func (n *StatusNode) stopPeerPool() error {
	if !n.config.Discovery {
		return nil
	}

	n.register.Stop()
	n.peerPool.Stop()
	return nil
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
	if n.config.ClusterConfig == nil || !n.config.ClusterConfig.Enabled {
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
	if n.config.ClusterConfig == nil || !n.config.ClusterConfig.Enabled {
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
	parsedNode, err := discover.ParseNode(url)
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
	parsedNode, err := discover.ParseNode(url)
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
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.rpcClient
}

// RPCPrivateClient exposes reference to RPC client connected to the running node
// that can call both public and private APIs.
func (n *StatusNode) RPCPrivateClient() *rpc.Client {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.rpcPrivateClient
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
