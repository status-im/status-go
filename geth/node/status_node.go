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
)

// errors
var (
	ErrNodeExists             = errors.New("node is already running")
	ErrNoRunningNode          = errors.New("there is no running node")
	ErrInvalidStatusNode      = errors.New("status node is not properly initialized")
	ErrInvalidService         = errors.New("service is unavailable")
	ErrInvalidAccountManager  = errors.New("could not retrieve account manager")
	ErrAccountKeyStoreMissing = errors.New("account key store is not set")
	ErrRPCClient              = errors.New("failed to init RPC client")
)

// RPCClientError reported when rpc client is initialized.
type RPCClientError error

// EthNodeError is reported when node crashed on start up.
type EthNodeError error

// StatusNode abstracts contained geth node and provides helper methods to
// interact with it.
type StatusNode struct {
	mu sync.RWMutex

	config    *params.NodeConfig // Status node configuration
	gethNode  *node.Node         // reference to Geth P2P stack/node
	rpcClient *rpc.Client        // reference to public RPC client
	register  *peers.Register
	peerPool  *peers.PeerPool
	db        *leveldb.DB // used as a cache for PeerPool

	log log.Logger
}

// New makes new instance of StatusNode.
func New() *StatusNode {
	return &StatusNode{
		log: log.New("package", "status-go/geth/node.StatusNode"),
	}
}

// Start starts current StatusNode, will fail if it's already started.
func (n *StatusNode) Start(config *params.NodeConfig, services ...node.ServiceConstructor) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if err := n.isAvailable(); err == nil {
		return ErrNodeExists
	}

	return n.start(config, services)
}

// start starts current StatusNode, will fail if it's already started.
func (n *StatusNode) start(config *params.NodeConfig, services []node.ServiceConstructor) error {
	ethNode, err := MakeNode(config)
	if err != nil {
		return err
	}
	n.gethNode = ethNode
	n.config = config

	for _, service := range services {
		if err := ethNode.Register(service); err != nil {
			return err
		}
	}

	// start underlying node
	if err := ethNode.Start(); err != nil {
		return EthNodeError(err)
	}
	// init RPC client for this node
	localRPCClient, err := n.gethNode.AttachPublic()
	if err == nil {
		n.rpcClient, err = rpc.NewClient(localRPCClient, n.config.UpstreamConfig)
	}
	if err != nil {
		n.log.Error("Failed to create an RPC client", "error", err)
		return RPCClientError(err)
	}
	if ethNode.Server().DiscV5 != nil {
		return n.startPeerPool()
	}
	return nil
}

func (n *StatusNode) startPeerPool() error {
	statusDB, err := db.Create(filepath.Join(n.config.DataDir, params.StatusDatabase))
	if err != nil {
		return err
	}
	n.db = statusDB
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
	return n.stop()
}

// stop will stop current StatusNode. A stopped node cannot be resumed.
func (n *StatusNode) stop() error {
	if err := n.isAvailable(); err != nil {
		return err
	}
	if n.gethNode.Server().DiscV5 != nil {
		n.stopPeerPool()
	}
	if err := n.gethNode.Stop(); err != nil {
		return err
	}
	n.gethNode = nil
	n.config = nil
	n.rpcClient = nil
	return nil
}

func (n *StatusNode) stopPeerPool() {
	n.register.Stop()
	n.peerPool.Stop()
	if err := n.db.Close(); err != nil {
		n.log.Error("error closing status db", "error", err)
	}
}

// ResetChainData removes chain data if node is not running.
func (n *StatusNode) ResetChainData(config *params.NodeConfig) error {
	if n.IsRunning() {
		return ErrNodeExists
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	chainDataDir := filepath.Join(config.DataDir, config.Name, "lightchaindata")
	if _, err := os.Stat(chainDataDir); os.IsNotExist(err) {
		// is it really an error, if we want to remove it as next step?
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

	if err := n.isAvailable(); err != nil {
		return false
	}
	return true
}

// GethNode returns underlying geth node.
func (n *StatusNode) GethNode() (*node.Node, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if err := n.isAvailable(); err != nil {
		return nil, err
	}
	return n.gethNode, nil
}

// populateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (n *StatusNode) populateStaticPeers() error {
	if err := n.isAvailable(); err != nil {
		return err
	}
	if !n.config.ClusterConfig.Enabled {
		n.log.Info("Static peers are disabled")
		return nil
	}

	for _, enode := range n.config.ClusterConfig.StaticNodes {
		err := n.addPeer(enode)
		if err != nil {
			n.log.Warn("Static peer addition failed", "error", err)
			continue
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
	server := n.gethNode.Server()
	if server == nil {
		return ErrNoRunningNode
	}
	for _, enode := range n.config.ClusterConfig.StaticNodes {
		err := n.removePeer(enode)
		if err != nil {
			n.log.Warn("Static peer deletion failed", "error", err)
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
	if err := n.removeStaticPeers(); err != nil {
		return err
	}
	return n.populateStaticPeers()
}

// AddPeer adds new static peer node
func (n *StatusNode) AddPeer(url string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if err := n.isAvailable(); err != nil {
		return err
	}
	return n.addPeer(url)
}

// addPeer adds new static peer node
func (n *StatusNode) addPeer(url string) error {
	// Try to add the url as a static peer and return
	parsedNode, err := discover.ParseNode(url)
	if err != nil {
		return err
	}
	n.gethNode.Server().AddPeer(parsedNode)
	return nil
}

func (n *StatusNode) removePeer(url string) error {
	parsedNode, err := discover.ParseNode(url)
	if err != nil {
		return err
	}
	n.gethNode.Server().RemovePeer(parsedNode)
	return nil
}

// PeerCount returns the number of connected peers.
func (n *StatusNode) PeerCount() int {
	if !n.IsRunning() {
		return 0
	}
	return n.gethNode.Server().PeerCount()
}

// Config exposes reference to running node's configuration
func (n *StatusNode) Config() (*params.NodeConfig, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if err := n.isAvailable(); err != nil {
		return nil, err
	}
	return n.config, nil
}

// gethService is a wrapper for gethNode.Service which retrieves a currently
// running service registered of a specific type.
func (n *StatusNode) gethService(serviceInstance interface{}, serviceName string) error {
	if err := n.isAvailable(); err != nil {
		return err
	}
	if err := n.gethNode.Service(serviceInstance); err != nil || serviceInstance == nil {
		n.log.Warn("Cannot obtain ", serviceName, " service", "error", err)
		return ErrInvalidService
	}

	return nil
}

// LightEthereumService exposes reference to LES service running on top of the node
func (n *StatusNode) LightEthereumService() (l *les.LightEthereum, err error) {
	return l, n.gethService(&l, "LES")
}

// WhisperService exposes reference to Whisper service running on top of the node
func (n *StatusNode) WhisperService() (w *whisper.Whisper, err error) {
	return w, n.gethService(&w, "whisper")
}

// AccountManager exposes reference to node's accounts manager
func (n *StatusNode) AccountManager() (*accounts.Manager, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if err := n.isAvailable(); err != nil {
		return nil, err
	}
	accountManager := n.gethNode.AccountManager()
	if accountManager == nil {
		return nil, ErrInvalidAccountManager
	}
	return accountManager, nil
}

// AccountKeyStore exposes reference to accounts key store
func (n *StatusNode) AccountKeyStore() (*keystore.KeyStore, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if err := n.isAvailable(); err != nil {
		return nil, err
	}
	accountManager := n.gethNode.AccountManager()
	if accountManager == nil {
		return nil, ErrInvalidAccountManager
	}

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

// isAvailable check if we have a node running and make sure is fully started
func (n *StatusNode) isAvailable() error {
	if n.gethNode == nil || n.gethNode.Server() == nil {
		return ErrNoRunningNode
	}
	return nil
}

// tickerResolution is the delta to check blockchain sync progress.
const tickerResolution = time.Second

// EnsureSync waits until blockchain synchronization
// is complete and returns.
func (n *StatusNode) EnsureSync(ctx context.Context) error {
	// Don't wait for any blockchain sync for the
	// local private chain as blocks are never mined.
	if n.config.NetworkID == params.StatusChainNetworkID {
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
