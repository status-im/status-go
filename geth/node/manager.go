package node

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/params"
)

// errors
var (
	ErrNodeAlreadyExists           = errors.New("there is a running node already, stop it before starting another one")
	ErrNoRunningNode               = errors.New("there is no running node")
	ErrNodeOpTimedOut              = errors.New("operation takes too long, timed out")
	ErrInvalidRunningNode          = errors.New("running node is not correctly initialized")
	ErrInvalidNodeManager          = errors.New("node manager is not properly initialized")
	ErrInvalidWhisperService       = errors.New("whisper service is unavailable")
	ErrInvalidLightEthereumService = errors.New("LES service is unavailable")
	ErrInvalidAccountManager       = errors.New("could not retrieve account manager")
	ErrAccountKeyStoreMissing      = errors.New("account key store is not set")
	ErrInvalidRPCClient            = errors.New("RPC service is unavailable")
)

// NodeManager manages Status node (which abstracts contained geth node)
type NodeManager struct {
	sync.RWMutex
	config         *params.NodeConfig // Status node configuration
	node           *node.Node         // reference to Geth P2P stack/node
	nodeStopped    chan struct{}      // channel to wait for termination notifications
	whisperService *whisper.Whisper   // reference to Whisper service
	lesService     *les.LightEthereum // reference to LES service
	rpcClient      *rpc.Client        // reference to RPC client
}

// NewNodeManager makes new instance of node manager
func NewNodeManager() *NodeManager {
	m := &NodeManager{}

	// allow interrupting running nodes
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		defer signal.Stop(sigc)
		<-sigc
		if m.node == nil {
			return
		}
		log.Info("Got interrupt, shutting down...")
		go m.node.Stop() // nolint: errcheck
		for i := 3; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info(fmt.Sprintf("Already shutting down, interrupt %d more times for panic.", i-1))
			}
		}
		panic("interrupted!")
	}()

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
	if m.node != nil || m.nodeStopped != nil {
		return nil, ErrNodeAlreadyExists
	}

	var err error
	m.node, err = MakeNode(config)
	if err != nil {
		return nil, err
	}
	m.config = config // preserve config of successfully created node

	nodeStarted := make(chan struct{})
	m.nodeStopped = make(chan struct{})
	go func() {
		defer HaltOnPanic()

		if err := m.node.Start(); err != nil {
			m.Lock() // TODO potential deadlock (add test case to prove otherwise)
			m.config = nil
			m.lesService = nil
			m.whisperService = nil
			m.rpcClient = nil
			m.nodeStopped = nil
			m.node = nil
			m.Unlock()
			SendSignal(SignalEnvelope{
				Type: EventNodeCrashed,
				Event: NodeCrashEvent{
					Error: fmt.Errorf("%v: %v", ErrNodeStartFailure, err).Error(),
				},
			})
			close(nodeStarted)
			return
		}

		// node is ready, use it
		m.onNodeStarted(nodeStarted)
	}()

	return nodeStarted, nil
}

// onNodeStarted extra processing once we have running node
func (m *NodeManager) onNodeStarted(nodeStarted chan struct{}) {
	// post-start processing
	if err := m.populateStaticPeers(); err != nil {
		log.Error("Static peers population", "error", err)
	}

	// obtain node info
	enode := "none"
	if server := m.node.Server(); server != nil {
		if nodeInfo := server.NodeInfo(); nodeInfo != nil {
			enode = nodeInfo.Enode
			log.Info("Node is ready", "enode", enode)
		}
	}

	// notify all subscribers that node is started
	SendSignal(SignalEnvelope{
		Type:  EventNodeStarted,
		Event: struct{}{},
	})
	close(nodeStarted)

	// wait up until node is stopped
	m.node.Wait()
	SendSignal(SignalEnvelope{
		Type:  EventNodeStopped,
		Event: struct{}{},
	})
	close(m.nodeStopped)
	log.Info("Node is stopped", "enode", enode)
}

// IsNodeRunning confirm that node is running
func (m *NodeManager) IsNodeRunning() bool {
	if m == nil {
		return false
	}

	m.RLock()
	defer m.RUnlock()

	return m.node != nil && m.nodeStopped != nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) StopNode() error {
	if m == nil {
		return ErrInvalidNodeManager
	}

	m.Lock()
	defer m.Unlock()

	return m.stopNode()
}

// stopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) stopNode() error {
	if m.node == nil {
		return ErrNoRunningNode
	}

	if m.nodeStopped == nil { // node may be running, but required channel not set
		return ErrInvalidRunningNode
	}

	if err := m.node.Stop(); err != nil {
		return err
	}

	// wait till the previous node is fully stopped
	select {
	case <-m.nodeStopped:
		// pass
	case <-time.After(30 * time.Second):
		return ErrNodeOpTimedOut
	}

	m.config = nil
	m.lesService = nil
	m.whisperService = nil
	m.rpcClient = nil
	m.nodeStopped = nil
	m.node = nil
	return nil
}

// Node returns underlying Status node
func (m *NodeManager) Node() (*node.Node, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

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
	if m.node == nil {
		return ErrNoRunningNode
	}

	if !m.config.BootClusterConfig.Enabled {
		log.Info("Boot cluster is disabled")
		return nil
	}

	enodes, err := m.config.LoadBootClusterNodes()
	if err != nil {
		log.Warn("Can not load boot nodes", "error", err)
	}
	for _, enode := range enodes {
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
	if m == nil || m.node == nil {
		return ErrNoRunningNode
	}

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

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

	prevConfig := *m.config
	if err := m.stopNode(); err != nil {
		return nil, err
	}

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
	log.Info("chaindata removed", "dir", chainDataDir)

	return m.startNode(&prevConfig)
}

// RestartNode restart running Status node, fails if node is not running
func (m *NodeManager) RestartNode() (<-chan struct{}, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.Lock()
	defer m.Unlock()

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

	prevConfig := *m.config
	if err := m.stopNode(); err != nil {
		return nil, err
	}

	return m.startNode(&prevConfig)
}

// NodeConfig exposes reference to running node's configuration
func (m *NodeManager) NodeConfig() (*params.NodeConfig, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

	return m.config, nil
}

// LightEthereumService exposes reference to LES service running on top of the node
func (m *NodeManager) LightEthereumService() (*les.LightEthereum, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

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

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

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

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

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

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

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

// RPCClient exposes reference to RPC client connected to the running node
func (m *NodeManager) RPCClient() (*rpc.Client, error) {
	if m == nil {
		return nil, ErrInvalidNodeManager
	}

	m.RLock()
	defer m.RUnlock()

	if m.node == nil {
		return nil, ErrNoRunningNode
	}

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
