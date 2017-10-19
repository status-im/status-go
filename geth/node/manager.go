package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common/geth"
	"github.com/status-im/status-go/geth/common/services"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/signal"
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
	pending = 2
)

// NodeManager manages Status node (which abstracts contained geth node)
// uses interfaces to prevent races on exported objects.
type NodeManager struct {
	state *int32

	config atomic.Value // Status node configuration

	node     geth.Node // reference to Geth P2P stack/node
	nodeLock sync.RWMutex

	nodeStarted     chan struct{} // channel to wait for start up notifications
	nodeStartedLock sync.RWMutex

	nodeStopped     chan struct{} // channel to wait for termination notifications
	nodeStoppedLock sync.RWMutex

	whisperService     *whisperv5.Whisper // reference to Whisper service
	whisperServiceLock sync.RWMutex

	lesService     *les.LightEthereum // reference to LES service
	lesServiceLock sync.RWMutex

	rpcClient     geth.RPCClient // reference to RPC client
	rpcClientLock sync.RWMutex
}

// NewNodeManager makes new instance of node manager
func NewNodeManager() *NodeManager {
	var isStarted int32 = pending
	m := &NodeManager{state: &isStarted}

	go HaltOnInterruptSignal(m) // allow interrupting running nodes

	return m
}

// StartNode start Status node, fails if node is already started.
func (m *NodeManager) StartNode(config *params.NodeConfig) (<-chan struct{}, error) {
	if m.isNodeStarted() {
		return nil, ErrNodeExists
	}

	m.initLog(config)

	return m.startNode(config)
}

// StartNodeWait the same as StartNode, but works in sync mode.
func (m *NodeManager) StartNodeWait(config *params.NodeConfig) error {
	startedChan, err := m.StartNode(config)
	if err != nil {
		return err
	}
	<-startedChan

	return nil
}

// startNode start Status node, fails if node is already started.
func (m *NodeManager) startNode(config *params.NodeConfig) (<-chan struct{}, error) {
	m.setPendingState()

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
		m.initRPCClient(ethNode)

		// underlying node is started, every method can use it, we use it immediately
		go func() {
			if err := m.PopulateStaticPeers(); err != nil {
				log.Error("Static peers population", "error", err)
			}
		}()

		// notify all subscribers that Status node is started
		m.setStarted()
	}()

	return m.getNodeStarted(), nil
}

func (m *NodeManager) newNode(config *params.NodeConfig) (*node.Node, error) {
	ethNode, err := MakeNode(config)
	if err != nil {
		return nil, err
	}

	m.nodeLock.Lock()
	defer m.nodeLock.Unlock()

	if err = ethNode.Start(); err != nil {
		m.setFailed(fmt.Errorf("%v: %v", ErrNodeStartFailure, err))

		return nil, ErrInvalidNodeManager
	}

	return ethNode, nil
}

// initRPCClient up on given node, in case an error stops node.
func (m *NodeManager) initRPCClient(node *node.Node) {
	rpcClient, err := rpc.NewClient(node, m.getUpstreamConfig())
	if err != nil {
		log.Error("Init RPC client failed:", "error", err)
		m.setFailed(ErrRPCClient)

		return
	}

	m.setRPCClient(rpcClient)
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) StopNode() (<-chan struct{}, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	return m.stop()
}

// StopNodeWait the same as StopNode, but works in sync mode.
func (m *NodeManager) StopNodeWait() error {
	stoppedChan, err := m.StopNode()
	if err != nil {
		return err
	}
	<-stoppedChan

	return nil
}

func (m *NodeManager) stop() (<-chan struct{}, error) {
	err := m.stopNode()
	if err != nil {
		return nil, err
	}

	go func() {
		m.setStopped()
	}()

	return m.getNodeStopped(), nil
}

func (m *NodeManager) stopWait() error {
	stopped, err := m.stop()
	if err != nil {
		return err
	}

	<-stopped

	return nil
}

func (m *NodeManager) stopNode() error {
	m.nodeLock.Lock()
	err := m.node.Stop()
	m.nodeLock.Unlock()
	if err != nil {
		return err
	}

	return nil
}

// Node returns underlying Status node.
func (m *NodeManager) Node() (geth.Node, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	return m.getNode(), nil
}

// PopulateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster.
func (m *NodeManager) PopulateStaticPeers() error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	m.waitNodeStarted()

	return m.populateStaticPeers()
}

// populateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster.
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

// AddPeer adds new static peer node.
func (m *NodeManager) AddPeer(url string) error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	m.waitNodeStarted()

	return m.addPeer(url)
}

// addPeer adds new static peer node.
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

// ResetChainDataWait the same as ResetChainData, but works in sync mode.
func (m *NodeManager) ResetChainDataWait() error {
	resetChan, err := m.ResetChainData()
	if err != nil {
		return err
	}
	<-resetChan

	return nil
}

// resetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *NodeManager) resetChainData() (<-chan struct{}, error) {
	prevConfig := m.getConfig()

	err := m.stopWait()
	if err != nil {
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
	signal.Send(signal.Envelope{
		Type:  signal.EventChainDataRemoved,
		Event: struct{}{},
	})
	log.Info("Chain data has been removed", "dir", chainDataDir)

	return m.startNode(prevConfig)
}

// RestartNode restart running Status node, fails if node is not running.
func (m *NodeManager) RestartNode() (<-chan struct{}, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	return m.restartNode()
}

// RestartNodeWait the same as RestartNode, but works in sync mode.
func (m *NodeManager) RestartNodeWait() error {
	restartChan, err := m.RestartNode()
	if err != nil {
		return err
	}
	<-restartChan

	return nil
}

// restartNode restart running Status node, fails if node is not running.
func (m *NodeManager) restartNode() (<-chan struct{}, error) {
	prevConfig := m.getConfig()

	err := m.stopWait()
	if err != nil {
		return nil, err
	}

	return m.startNode(prevConfig)
}

// NodeConfig exposes reference to running node's configuration.
func (m *NodeManager) NodeConfig() (*params.NodeConfig, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	return m.getConfig(), nil
}

// LightEthereumService exposes reference to LES service running on top of the node.
func (m *NodeManager) LightEthereumService() (services.LesService, error) {
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

// WhisperService exposes reference to Whisper service running on top of the node.
func (m *NodeManager) WhisperService() (services.Whisper, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	whisperService, err := m.getWhisperService()
	if err != nil {
		log.Warn("Cannot obtain whisper service", "error", err)
		return nil, err
	}

	m.whisperServiceLock.Lock()
	defer m.whisperServiceLock.Unlock()

	m.whisperService = whisperService
	return m.whisperService, nil
}

// PublicWhisperAPI exposes reference to public Whisper API.
func (m *NodeManager) PublicWhisperAPI() (*whisperv5.PublicWhisperAPI, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.waitNodeStarted()

	whisperService, err := m.getWhisperService()
	if err != nil {
		log.Warn("Cannot obtain whisper service", "error", err)
		return nil, err
	}

	return whisperv5.NewPublicWhisperAPI(whisperService), nil
}

// AccountManager exposes reference to node's accounts manager.
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

// AccountKeyStore exposes reference to accounts key store.
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
func (m *NodeManager) RPCClient() geth.RPCClient {
	return m.getRPCClient()
}

// GetStatusBackend exposes StatusBackend interface.
func (m *NodeManager) GetStatusBackend() (services.StatusBackend, error) {
	m.lesServiceLock.RLock()
	defer m.lesServiceLock.RUnlock()

	return m.lesService.StatusBackend, nil
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

// isNodeAvailable check if we have a node running and make sure is fully started.
func (m *NodeManager) isNodeAvailable() error {
	if !m.isNodeStarted() {
		return ErrNoRunningNode
	}

	return nil
}

func (m *NodeManager) setStoppedState() {
	atomic.StoreInt32(m.state, stopped)
}

func (m *NodeManager) setStartedState() {
	atomic.StoreInt32(m.state, started)
}

func (m *NodeManager) setPendingState() {
	atomic.StoreInt32(m.state, pending)
}

func (m *NodeManager) isNodeStarted() bool {
	return atomic.LoadInt32(m.state) == started
}

func (m *NodeManager) isNodeStopped() bool {
	return atomic.LoadInt32(m.state) == stopped
}

func (m *NodeManager) isNodePending() bool {
	return atomic.LoadInt32(m.state) == pending
}

func (m *NodeManager) setConfig(config *params.NodeConfig) {
	m.config.Store(*config)
}

func (m *NodeManager) getConfig() *params.NodeConfig {
	config := m.config.Load()
	configValue := config.(params.NodeConfig)

	return &configValue
}

func (m *NodeManager) getBootClusterEnabled() bool {
	config := m.getConfig()
	bootCluster := config.BootClusterConfig.Enabled

	return bootCluster
}

func (m *NodeManager) getUpstreamConfig() params.UpstreamRPCConfig {
	config := m.getConfig()
	upstreamConfig := config.UpstreamConfig

	return upstreamConfig
}

func (m *NodeManager) getBootNodes() []string {
	config := m.getConfig()
	nodes := make([]string, len(config.BootClusterConfig.BootNodes))
	copy(nodes, config.BootClusterConfig.BootNodes)

	return nodes
}

func (m *NodeManager) setNode(node geth.Node) {
	m.nodeLock.Lock()
	m.node = node
	m.nodeLock.Unlock()
}

func (m *NodeManager) getNode() geth.Node {
	m.nodeLock.RLock()
	defer m.nodeLock.RUnlock()

	return m.node
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

func (m *NodeManager) closeNodeStarted() {
	m.nodeStartedLock.Lock()
	close(m.nodeStarted)
	m.nodeStartedLock.Unlock()
}

func (m *NodeManager) waitNodeStarted() {
	m.nodeStartedLock.RLock()
	<-m.nodeStarted
	m.nodeStartedLock.RUnlock()
}

// setStarted sets node into started state: start channel closed, and send Started signal.
func (m *NodeManager) setStarted() {
	if !m.isNodePending() {
		return
	}

	m.closeNodeStarted()
	m.setStartedState()

	signal.Send(signal.Envelope{
		Type:  signal.EventNodeStarted,
		Event: struct{}{},
	})
}

// setStopped sets node into stopped state: start and stop channels closed, and send Stopped signal.
func (m *NodeManager) setStopped() {
	if m.isNodeStopped() {
		return
	}

	log.Info("Node is stopped")

	m.node.Wait()
	m.closeNodeStopped()
	m.setStoppedState()

	// notify application that it can send more requests now
	signal.Send(signal.Envelope{
		Type:  signal.EventNodeStopped,
		Event: struct{}{},
	})

	log.Info("Node manager notified app, that node has stopped")
}

// setFailed sets node into Failed state: stopped state, start and stop channels closed, and send Crash signal.
func (m *NodeManager) setFailed(err error) {
	if !m.isNodePending() {
		return
	}

	m.nodeStartedLock.Lock()
	close(m.nodeStarted)
	m.nodeStartedLock.Unlock()

	m.setStoppedState()

	signal.Send(signal.Envelope{
		Type: signal.EventNodeCrashed,
		Event: signal.NodeCrashEvent{
			Error: err.Error(),
		},
	})
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

func (m *NodeManager) closeNodeStopped() {
	m.nodeStoppedLock.Lock()
	close(m.nodeStopped)
	m.nodeStoppedLock.Unlock()
}

// getWhisperService returns Whisper service or inits it
func (m *NodeManager) getWhisperService() (*whisperv5.Whisper, error) {
	m.whisperServiceLock.Lock()
	defer m.whisperServiceLock.Unlock()

	if m.whisperService != nil {
		return m.whisperService, nil
	}

	var whisperObject *whisperv5.Whisper
	err := m.node.Service(&whisperObject)
	if err != nil {
		return nil, ErrInvalidWhisperService
	}
	m.whisperService = whisperObject

	return m.whisperService, nil
}

// getLesService returns LES service or inits both LES and backend
func (m *NodeManager) getLesService() (*les.LightEthereum, error) {
	m.lesServiceLock.Lock()
	defer m.lesServiceLock.Unlock()

	if m.lesService != nil {
		return m.lesService, nil
	}

	var lesObject *les.LightEthereum
	err := m.node.Service(&lesObject)
	if err != nil {
		return nil, ErrInvalidLightEthereumService
	}
	m.lesService = lesObject

	return m.lesService, nil
}

func (m *NodeManager) setRPCClient(rpcClient geth.RPCClient) {
	m.rpcClientLock.Lock()
	m.rpcClient = rpcClient
	m.rpcClientLock.Unlock()
}

func (m *NodeManager) getRPCClient() geth.RPCClient {
	m.rpcClientLock.RLock()
	defer m.rpcClientLock.RUnlock()

	if m.rpcClient == nil {
		return nil
	}

	return m.rpcClient
}
