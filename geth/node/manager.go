package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
)

// errors
var (
	ErrNodeExists                  = errors.New("node is already running")
	ErrNoRunningNode               = errors.New("there is no running node")
	ErrInvalidNodeManager          = errors.New("node manager is not properly initialized")
	ErrInvalidWhisperService       = errors.New("whisper service is unavailable")
	ErrInvalidLightEthereumService = errors.New("LES service is unavailable")
	ErrInvalidAccountManager       = errors.New("could not retrieve account manager")
	ErrAccountKeyStoreMissing      = errors.New("account key store is not set")
	ErrInvalidRPCClient            = errors.New("RPC client is unavailable")
	ErrInvalidRPCServer            = errors.New("RPC server is unavailable")
)

// NodeManager manages Status node (which abstracts contained geth node)
type NodeManager struct {
	sync.RWMutex
	config         *params.NodeConfig // Status node configuration
	node           *node.Node         // reference to Geth P2P stack/node
	nodeStarted    chan struct{}      // channel to wait for start up notifications
	nodeStopped    chan struct{}      // channel to wait for termination notifications
	whisperService *whisper.Whisper   // reference to Whisper service
	lesService     *les.LightEthereum // reference to LES service
	rpcClient      *rpc.Client        // reference to RPC client
	rpcServer      *rpc.Server        // reference to RPC server
}

// NewNodeManager makes new instance of node manager
func NewNodeManager() *NodeManager {
	m := &NodeManager{}
	go HaltOnInterruptSignal(m) // allow interrupting running nodes

	return m
}

// StartNode start Status node, fails if node is already started
func (m *NodeManager) StartNode(config *params.NodeConfig) (<-chan struct{}, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.Lock()
	defer m.Unlock()

	return m.startNode(config)
}

// startNode start Status node, fails if node is already started
func (m *NodeManager) startNode(config *params.NodeConfig) (<-chan struct{}, error) {
	if m.node != nil || m.nodeStarted != nil {
		return nil, ErrNodeExists
	}

	ethNode, err := MakeNode(config)
	if err != nil {
		return nil, err
	}

	m.nodeStarted = make(chan struct{}, 1)

	go func() {
		defer HaltOnPanic()

		// start underlying node
		if err := ethNode.Start(); err != nil {
			close(m.nodeStarted)
			m.Lock()
			m.nodeStarted = nil
			m.Unlock()
			SendSignal(SignalEnvelope{
				Type: EventNodeCrashed,
				Event: NodeCrashEvent{
					Error: fmt.Errorf("%v: %v", ErrNodeStartFailure, err).Error(),
				},
			})
			return
		}

		m.Lock()
		m.node = ethNode
		m.nodeStopped = make(chan struct{}, 1)
		m.config = config
		m.Unlock()

		// underlying node is started, every method can use it, we use it immediately
		go func() {
			if err := m.PopulateStaticPeers(); err != nil {
				log.Error("Static peers population", "error", err)
			}
		}()

		// notify all subscribers that Status node is started
		close(m.nodeStarted)
		SendSignal(SignalEnvelope{
			Type:  EventNodeStarted,
			Event: struct{}{},
		})

		// wait up until underlying node is stopped
		m.node.Wait()

		// notify m.Stop() that node has been stopped
		close(m.nodeStopped)
		log.Info("Node is stopped")
	}()

	return m.nodeStarted, nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) StopNode() (<-chan struct{}, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.Lock()
	defer m.Unlock()

	return m.stopNode()
}

// stopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) stopNode() (<-chan struct{}, error) {
	if m.node == nil || m.nodeStarted == nil || m.nodeStopped == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted // make sure you operate on fully started node

	// now attempt to stop
	if err := m.node.Stop(); err != nil {
		return nil, err
	}

	nodeStopped := make(chan struct{}, 1)
	go func() {
		<-m.nodeStopped // Status node is stopped (code after Wait() is executed)
		log.Info("Ready to reset node")

		// reset node params
		m.Lock()
		m.config = nil
		m.lesService = nil
		m.whisperService = nil
		m.rpcClient = nil
		m.rpcServer = nil
		m.nodeStarted = nil
		m.node = nil
		m.Unlock()

		close(nodeStopped) // Status node is stopped, and we can create another
		log.Info("Node manager resets node params")

		// notify application that it can send more requests now
		SendSignal(SignalEnvelope{
			Type:  EventNodeStopped,
			Event: struct{}{},
		})
		log.Info("Node manager notifed app, that node has stopped")
	}()

	return nodeStopped, nil
}

// IsNodeRunning confirm that node is running
func (m *NodeManager) IsNodeRunning() bool {
	if m == nil {
		return false
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return false
	}
	<-m.nodeStarted

	return true
}

// Node returns underlying Status node
func (m *NodeManager) Node() (*node.Node, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	return m.node, nil
}

// PopulateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (m *NodeManager) PopulateStaticPeers() error {
	if m == nil {
		return ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	return m.populateStaticPeers()
}

// populateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (m *NodeManager) populateStaticPeers() error {
	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return ErrNoRunningNode
	}
	<-m.nodeStarted

	if !m.config.BootClusterConfig.Enabled {
		log.Info("Boot cluster is disabled")
		return nil
	}

	for _, enode := range m.config.BootClusterConfig.BootNodes {
		err := m.addPeer(enode)
		if err != nil {
			log.Warn("Boot node addition failed", "error", err)
			continue
		}
		log.Info("Boot node added", "enode", enode)
	}

	return nil
}

// AddPeer adds new static peer node
func (m *NodeManager) AddPeer(url string) error {
	if m == nil {
		return ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	return m.addPeer(url)
}

// addPeer adds new static peer node
func (m *NodeManager) addPeer(url string) error {
	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return ErrNoRunningNode
	}
	<-m.nodeStarted

	server := m.node.Server()
	if server == nil {
		return ErrNoRunningNode
	}

	// Try to add the url as a static peer and return
	parsedNode, err := discover.ParseNode(url)
	if err != nil {
		return err
	}
	server.AddPeer(parsedNode)

	return nil
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *NodeManager) ResetChainData() (<-chan struct{}, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.Lock()
	defer m.Unlock()

	return m.resetChainData()
}

// resetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *NodeManager) resetChainData() (<-chan struct{}, error) {
	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	prevConfig := *m.config
	nodeStopped, err := m.stopNode()
	if err != nil {
		return nil, err
	}

	m.Unlock()
	<-nodeStopped
	m.Lock()

	chainDataDir := filepath.Join(prevConfig.DataDir, prevConfig.Name, "lightchaindata")
	if _, err := os.Stat(chainDataDir); os.IsNotExist(err) {
		return nil, err
	}
	if err := os.RemoveAll(chainDataDir); err != nil {
		return nil, err
	}
	// send signal up to native app
	SendSignal(SignalEnvelope{
		Type:  EventChainDataRemoved,
		Event: struct{}{},
	})
	log.Info("Chain data has been removed", "dir", chainDataDir)

	return m.startNode(&prevConfig)
}

// RestartNode restart running Status node, fails if node is not running
func (m *NodeManager) RestartNode() (<-chan struct{}, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.Lock()
	defer m.Unlock()

	return m.restartNode()
}

// restartNode restart running Status node, fails if node is not running
func (m *NodeManager) restartNode() (<-chan struct{}, error) {
	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	prevConfig := *m.config
	nodeStopped, err := m.stopNode()
	if err != nil {
		return nil, err
	}

	m.Unlock()
	<-nodeStopped
	m.Lock()

	return m.startNode(&prevConfig)
}

// NodeConfig exposes reference to running node's configuration
func (m *NodeManager) NodeConfig() (*params.NodeConfig, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	return m.config, nil
}

// LightEthereumService exposes reference to LES service running on top of the node
func (m *NodeManager) LightEthereumService() (*les.LightEthereum, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	if m.lesService == nil {
		if err := m.node.Service(&m.lesService); err != nil {
			log.Warn("Cannot obtain LES service", "error", err)
			return nil, ErrInvalidLightEthereumService
		}
	}

	if m.lesService == nil {
		return nil, ErrInvalidLightEthereumService
	}

	return m.lesService, nil
}

// WhisperService exposes reference to Whisper service running on top of the node
func (m *NodeManager) WhisperService() (*whisper.Whisper, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	if m.whisperService == nil {
		if err := m.node.Service(&m.whisperService); err != nil {
			log.Warn("Cannot obtain whisper service", "error", err)
			return nil, ErrInvalidWhisperService
		}
	}

	if m.whisperService == nil {
		return nil, ErrInvalidWhisperService
	}

	return m.whisperService, nil
}

// AccountManager exposes reference to node's accounts manager
func (m *NodeManager) AccountManager() (*accounts.Manager, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	accountManager := m.node.AccountManager()
	if accountManager == nil {
		return nil, ErrInvalidAccountManager
	}

	return accountManager, nil
}

// AccountKeyStore exposes reference to accounts key store
func (m *NodeManager) AccountKeyStore() (*keystore.KeyStore, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	accountManager := m.node.AccountManager()
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

// RPCLocalClient exposes reference to RPC client connected to the running node.
func (m *NodeManager) RPCLocalClient() (*rpc.Client, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	config, err := m.NodeConfig()
	if err != nil {
		return nil, err
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}

	<-m.nodeStarted

	if m.rpcClient == nil {
		var err error
		m.rpcClient, err = m.node.Attach()
		if err != nil {
			log.Error("Cannot attach RPC client to node", "error", err)
			return nil, ErrInvalidRPCClient
		}
	}

	if m.rpcClient == nil {
		return nil, ErrInvalidRPCClient
	}

	return m.rpcClient, nil
}

// RPCUpstreamClient exposes reference to RPC client connected to the running node.
func (m *NodeManager) RPCUpstreamClient() (*rpc.Client, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	config, err := m.NodeConfig()
	if err != nil {
		return nil, err
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}

	<-m.nodeStarted

	if m.rpcClient == nil {
		m.rpcClient, err = rpc.Dial(config.UpstreamConfig.URL)
		if err != nil {
			log.Error("Failed to conect to upstream RPC server", "error", err)
			return nil, err
		}
	}

	if m.rpcClient == nil {
		return nil, ErrInvalidRPCClient
	}

	return m.rpcClient, nil
}

// RPCClient exposes reference to RPC client connected to the running node.
func (m *NodeManager) RPCClient() (*rpc.Client, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	config, err := m.NodeConfig()
	if err != nil {
		return nil, err
	}

	// Connect to upstream RPC server with new client and cache instance.
	if config.UpstreamConfig.Enabled {
		return m.RPCUpstreamClient()
	}

	return m.RPCUpstreamClient()
}

// RPCServer exposes reference to running node's in-proc RPC server/handler
func (m *NodeManager) RPCServer() (*rpc.Server, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	// make sure that node is fully started
	if m.node == nil || m.nodeStarted == nil {
		return nil, ErrNoRunningNode
	}
	<-m.nodeStarted

	if m.rpcServer == nil {
		var err error
		m.rpcServer, err = m.node.InProcRPC()
		if err != nil {
			log.Error("Cannot expose on-proc RPC server", "error", err)
			return nil, ErrInvalidRPCServer
		}
	}

	if m.rpcServer == nil {
		return nil, ErrInvalidRPCServer
	}

	return m.rpcServer, nil
}
