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
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/signal"
	"sync/atomic"
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
	ErrRPCClient                   = errors.New("failed to init RPC client")
)

const (
	started = 0
	stopped = 1
)

// NodeManager manages Status node (which abstracts contained geth node)
type NodeManager struct {
	isStarted *int32

	config     *params.NodeConfig // Status node configuration
	configLock sync.RWMutex

	node     *node.Node // reference to Geth P2P stack/node
	nodeLock sync.RWMutex

	nodeStarted     chan struct{} // channel to wait for start up notifications
	nodeStartedLock sync.RWMutex

	nodeStopped     chan struct{} // channel to wait for termination notifications
	nodeStoppedLock sync.RWMutex

	whisperService     *whisper.Whisper // reference to Whisper service
	whisperServiceLock sync.RWMutex

	lesService     *les.LightEthereum // reference to LES service
	lesServiceLock sync.RWMutex

	rpcClient     *rpc.Client // reference to RPC client
	rpcClientLock sync.RWMutex
}

// NewNodeManager makes new instance of node manager
func NewNodeManager() *NodeManager {
	var isStarted int32 = stopped
	m := &NodeManager{isStarted: &isStarted}

	go HaltOnInterruptSignal(m) // allow interrupting running nodes

	return m
}

// StartNode start Status node, fails if node is already started
func (m *NodeManager) StartNode(config *params.NodeConfig) (<-chan struct{}, error) {
	return m.startNode(config)
}

// startNode start Status node, fails if node is already started
func (m *NodeManager) startNode(config *params.NodeConfig) (<-chan struct{}, error) {
	if m.isNodeStarted() {
		return nil, ErrNodeExists
	}

	m.initLog(config)

	m.setNodeStarted(make(chan struct{}, 1))

	ethNode, err := m.newNode(config)
	if err != nil {
		return nil, err
	}

	go func() {
		defer HaltOnPanic()

		m.setNode(ethNode)
		m.setNodeStopped(make(chan struct{}, 1))
		m.setConfig(config)

		// init RPC client for this node
		rpcClient, err := rpc.NewClient(m.getNode(), m.getUpstreamConfig())
		if err != nil {
			log.Error("Init RPC client failed:", "error", err)

			m.setStarted()

			signal.Send(signal.Envelope{
				Type: signal.EventNodeCrashed,
				Event: signal.NodeCrashEvent{
					Error: ErrRPCClient.Error(),
				},
			})
			return
		}

		m.setRPCClient(rpcClient)

		// underlying node is started, every method can use it, we use it immediately
		go func() {
			if err := m.PopulateStaticPeers(); err != nil {
				log.Error("Static peers population", "error", err)
			}
		}()

		// notify all subscribers that Status node is started
		m.setStarted()
		signal.Send(signal.Envelope{
			Type:  signal.EventNodeStarted,
			Event: struct{}{},
		})

		// todo(@jeka): why we stop the underlying node
		// wait up until underlying node is stopped
		m.nodeLock.RLock()
		m.node.Wait()
		m.nodeLock.RUnlock()

		// notify m.Stop() that node has been stopped
		m.closeNodeStopped()
		log.Info("Node is stopped")
	}()

	return m.nodeStarted, nil
}

func (m *NodeManager) newNode(config *params.NodeConfig) (*node.Node, error) {
	ethNode, err := MakeNode(config)
	if err != nil {
		return nil, err
	}

	// start underlying node
	m.nodeLock.Lock()
	defer m.nodeLock.Unlock()

	if err = ethNode.Start(); err != nil {
		m.setStarted()

		signal.Send(signal.Envelope{
			Type: signal.EventNodeCrashed,
			Event: signal.NodeCrashEvent{
				Error: fmt.Errorf("%v: %v", ErrNodeStartFailure, err).Error(),
			},
		})
		return nil, ErrInvalidNodeManager
	}

	return ethNode, nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) StopNode() (<-chan struct{}, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	return m.stopNode()
}

// stopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) stopNode() (<-chan struct{}, error) {
	// now attempt to stop
	m.nodeLock.Lock()
	err := m.node.Stop()
	m.nodeLock.Unlock()
	if err != nil {
		return nil, err
	}

	nodeStopped := make(chan struct{}, 1)
	go func() {
		m.readNodeStopped() // Status node is stopped (code after Wait() is executed)

		m.setStopped() // Status node is stopped, and we can create another
		log.Info("Node manager resets node params")

		// notify application that it can send more requests now
		signal.Send(signal.Envelope{
			Type:  signal.EventNodeStopped,
			Event: struct{}{},
		})
		log.Info("Node manager notifed app, that node has stopped")
	}()

	return nodeStopped, nil
}

// IsNodeRunning confirm that node is running
func (m *NodeManager) IsNodeRunning() bool {
	if err := m.isNodeAvailable(); err != nil {
		return false
	}

	m.waitNodeStarted()

	return true
}

// Node returns underlying Status node
func (m *NodeManager) Node() (*node.Node, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	return m.getNode(), nil
}

// PopulateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (m *NodeManager) PopulateStaticPeers() error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	m.waitNodeStarted()

	return m.populateStaticPeers()
}

// populateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (m *NodeManager) populateStaticPeers() error {
	if !m.getBootClusterEnabled() {
		log.Info("Boot cluster is disabled")
		return nil
	}

	for _, enode := range m.getBootNodes() {
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
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	m.waitNodeStarted()

	return m.addPeer(url)
}

// addPeer adds new static peer node
func (m *NodeManager) addPeer(url string) error {
	m.nodeLock.Lock()
	server := m.node.Server()
	m.nodeLock.Unlock()
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
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	return m.resetChainData()
}

// resetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *NodeManager) resetChainData() (<-chan struct{}, error) {
	prevConfig := m.getConfig()
	nodeStopped, err := m.stopNode()
	if err != nil {
		return nil, err
	}

	<-nodeStopped

	chainDataDir := filepath.Join(prevConfig.DataDir, prevConfig.Name, "lightchaindata")
	if _, err := os.Stat(chainDataDir); os.IsNotExist(err) {
		return nil, err
	}

	if err := os.RemoveAll(chainDataDir); err != nil {
		return nil, err
	}

	// send signal up to native app
	signal.Send(signal.Envelope{
		Type:  signal.EventChainDataRemoved,
		Event: struct{}{},
	})
	log.Info("Chain data has been removed", "dir", chainDataDir)

	return m.startNode(prevConfig)
}

// RestartNode restart running Status node, fails if node is not running
func (m *NodeManager) RestartNode() (<-chan struct{}, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	return m.restartNode()
}

// restartNode restart running Status node, fails if node is not running
func (m *NodeManager) restartNode() (<-chan struct{}, error) {
	prevConfig := m.getConfig()
	nodeStopped, err := m.stopNode()
	if err != nil {
		return nil, err
	}

	<-nodeStopped

	return m.startNode(prevConfig)
}

// NodeConfig exposes reference to running node's configuration
func (m *NodeManager) NodeConfig() (*params.NodeConfig, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	return m.config, nil
}

// LightEthereumService exposes reference to LES service running on top of the node
func (m *NodeManager) LightEthereumService() (*les.LightEthereum, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	les, err := m.getLesService()
	if err != nil {
		log.Warn("Cannot obtain LES service", "error", err)
		return nil, err
	}

	m.lesService = les
	return m.lesService, nil
}

// WhisperService exposes reference to Whisper service running on top of the node
func (m *NodeManager) WhisperService() (*whisper.Whisper, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	whisperService, err := m.getWhisperService()
	if err != nil {
		log.Warn("Cannot obtain whisper service", "error", err)
		return nil, err
	}

	m.whisperService = whisperService
	return m.whisperService, nil
}

// AccountManager exposes reference to node's accounts manager
func (m *NodeManager) AccountManager() (*accounts.Manager, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	accountManager := m.node.AccountManager()
	if accountManager == nil {
		return nil, ErrInvalidAccountManager
	}

	return accountManager, nil
}

// AccountKeyStore exposes reference to accounts key store
func (m *NodeManager) AccountKeyStore() (*keystore.KeyStore, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

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

// RPCClient exposes reference to RPC client connected to the running node.
func (m *NodeManager) RPCClient() *rpc.Client {
	return m.getRPCClient()
}

// initLog initializes global logger parameters based on
// provided node configurations.
func (m *NodeManager) initLog(config *params.NodeConfig) {
	log.SetLevel(config.LogLevel)

	if config.LogFile == "" {
		return
	}

	err := log.SetLogFile(config.LogFile)
	if err != nil {
		fmt.Println("Failed to open log file, using stdout")
	}
}

// todo(@jeka): may be we need Wait flag for it to wait until node started
// isNodeAvailable check if we have a node running and make sure is fully started
func (m *NodeManager) isNodeAvailable() error {
	if !m.isNodeStarted() {
		return ErrNoRunningNode
	}

	return nil
}

func (m *NodeManager) setStopped() {
	atomic.CompareAndSwapInt32(m.isStarted, started, stopped)
}

func (m *NodeManager) setStartedStatus() {
	atomic.CompareAndSwapInt32(m.isStarted, stopped, started)
}

func (m *NodeManager) isNodeStarted() bool {
	return atomic.LoadInt32(m.isStarted) == started
}

//todo(@jeka): we should use copy generator
func (m *NodeManager) setConfig(config *params.NodeConfig) {
	m.configLock.Lock()
	m.config = config
	m.configLock.Unlock()
}

func (m *NodeManager) getConfig() *params.NodeConfig {
	m.configLock.RLock()
	defer m.configLock.RUnlock()

	if m.config == nil {
		return nil
	}

	config := *m.config
	return &config
}

func (m *NodeManager) getBootClusterEnabled() bool {
	m.configLock.RLock()
	bootCluster := m.config.BootClusterConfig.Enabled
	m.configLock.RUnlock()

	return bootCluster
}

func (m *NodeManager) getUpstreamConfig() params.UpstreamRPCConfig {
	m.configLock.RLock()
	upstreamConfig := m.config.UpstreamConfig
	m.configLock.RUnlock()

	return upstreamConfig
}

func (m *NodeManager) getBootNodes() []string {
	m.configLock.RLock()
	nodes := make([]string, len(m.config.BootClusterConfig.BootNodes))
	copy(nodes, m.config.BootClusterConfig.BootNodes)
	m.configLock.RUnlock()

	return nodes
}

func (m *NodeManager) setNode(node *node.Node) {
	m.nodeLock.Lock()
	m.node = node
	m.nodeLock.Unlock()
}

func (m *NodeManager) getNode() *node.Node {
	m.nodeLock.RLock()
	defer m.nodeLock.RUnlock()

	node := *m.node
	return &node
}

func (m *NodeManager) setNodeStarted(nodeStarted chan struct{}) {
	m.nodeStartedLock.Lock()
	m.nodeStarted = nodeStarted
	m.nodeStartedLock.Unlock()
}

func (m *NodeManager) getNodeStarted() chan struct{} {
	m.nodeStartedLock.RLock()
	defer m.nodeStartedLock.RUnlock()

	return m.nodeStarted
}

func (m *NodeManager) nodeStartedIsNil() bool {
	m.nodeStartedLock.RLock()
	ok := m.nodeStarted == nil
	m.nodeStartedLock.RUnlock()

	return ok
}

func (m *NodeManager) readNodeStarted() {
	m.nodeStartedLock.RLock()
	<-m.nodeStarted
	m.nodeStartedLock.RUnlock()
}

func (m *NodeManager) waitNodeStarted() {
	m.readNodeStarted()
}

func (m *NodeManager) setStarted() {
	if m.isNodeStarted() {
		return
	}

	m.nodeStartedLock.Lock()
	close(m.nodeStarted)
	m.nodeStartedLock.Unlock()

	m.setStartedStatus()
}

func (m *NodeManager) getNodeStopped() chan struct{} {
	m.nodeStoppedLock.RLock()
	defer m.nodeStoppedLock.RUnlock()

	return m.nodeStopped
}

func (m *NodeManager) setNodeStopped(nodeStopped chan struct{}) {
	m.nodeStoppedLock.Lock()
	m.nodeStopped = nodeStopped
	m.nodeStoppedLock.Unlock()
}

func (m *NodeManager) nodeStoppedIsNil() bool {
	m.nodeStoppedLock.RLock()
	ok := m.nodeStopped == nil
	m.nodeStoppedLock.RUnlock()

	return ok
}

func (m *NodeManager) readNodeStopped() {
	m.nodeStoppedLock.RLock()
	<-m.nodeStopped
	m.nodeStoppedLock.RUnlock()
}

func (m *NodeManager) closeNodeStopped() {
	m.nodeStoppedLock.Lock()
	close(m.nodeStopped)
	m.nodeStoppedLock.Unlock()
}

func (m *NodeManager) waitNodeStopped() {
	m.closeNodeStopped()
}

func (m *NodeManager) setWhisperService(whisper *whisper.Whisper) {
	m.whisperServiceLock.Lock()
	m.whisperService = whisper
	m.whisperServiceLock.Unlock()
}

func (m *NodeManager) getWhisperService() (*whisper.Whisper, error) {
	m.whisperServiceLock.RLock()
	defer m.whisperServiceLock.RUnlock()

	whisper := m.whisperService
	if whisper != nil {
		return whisper, nil
	}

	err := m.node.Service(&whisper)
	if err != nil {
		return nil, ErrInvalidWhisperService
	}


	return whisper, nil
}

func (m *NodeManager) whisperServiceIsNil() bool {
	m.whisperServiceLock.RLock()
	ok := m.whisperService == nil
	m.whisperServiceLock.RUnlock()

	return ok
}

func (m *NodeManager) setLesService(les *les.LightEthereum) {
	m.lesServiceLock.Lock()
	m.lesService = les
	m.lesServiceLock.Unlock()
}

func (m *NodeManager) getLesService() (*les.LightEthereum, error) {
	m.lesServiceLock.RLock()
	defer m.lesServiceLock.RUnlock()

	les := m.lesService
	if les != nil {
		return les, nil
	}

	err := m.node.Service(&les)
	if err != nil {
		return nil, ErrInvalidLightEthereumService
	}

	return les, nil
}

func (m *NodeManager) lesServiceIsNil() bool {
	m.lesServiceLock.RLock()
	ok := m.lesService == nil
	m.lesServiceLock.RUnlock()

	return ok
}

func (m *NodeManager) setRPCClient(rpcClient *rpc.Client) {
	m.rpcClientLock.Lock()
	m.rpcClient = rpcClient
	m.rpcClientLock.Unlock()
}

func (m *NodeManager) getRPCClient() *rpc.Client {
	m.rpcClientLock.RLock()
	defer m.rpcClientLock.RUnlock()

	if m.rpcClient == nil {
		return nil
	}

	rpcClient := *m.rpcClient
	return &rpcClient
}

func (m *NodeManager) rpcClientIsNil() bool {
	m.rpcClientLock.RLock()
	ok := m.rpcClient == nil
	m.rpcClientLock.RUnlock()

	return ok
}
