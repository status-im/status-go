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

	whisperService     *whisperv5.Whisper // reference to Whisper service
	whisperServiceLock sync.RWMutex

	lesService     *les.LightEthereum // reference to LES service
	lesServiceLock sync.RWMutex

	rpcClient     *rpc.Client // reference to RPC client
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
func (m *NodeManager) StartNode(config *params.NodeConfig) error {
	if m.isNodeStarted() {
		return ErrNodeExists
	}

	m.initLog(config)

	return m.startNode(config)
}

// startNode start Status node, fails if node is already started.
func (m *NodeManager) startNode(config *params.NodeConfig) error {
	m.setPendingState()

	ethNode, err := m.newNode(config)
	if err != nil {
		return err
	}

	defer HaltOnPanic()

	m.setNode(ethNode)
	m.setConfig(config)
	m.initRPCClient(ethNode)

	// underlying node is started, every method can use it, we use it immediately
	go func() {
		if err := m.PopulateStaticPeers(); err != nil {
			log.Error("Static peers population", "error", err)
		}
	}()

	m.setStarted()

	return nil
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

	m.rpcClientLock.Lock()
	m.rpcClient = rpcClient
	m.rpcClientLock.Unlock()
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) StopNode() error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	return m.stop()
}

func (m *NodeManager) stop() error {
	m.nodeLock.Lock()
	err := m.node.Stop()
	m.nodeLock.Unlock()
	if err != nil {
		return err
	}

	m.setStopped()

	return nil
}

// Node returns underlying Status node.
func (m *NodeManager) Node() (geth.Node, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	return m.getNode(), nil
}

// PopulateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster.
func (m *NodeManager) PopulateStaticPeers() error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

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
func (m *NodeManager) ResetChainData() error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	return m.resetChainData()
}

// resetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *NodeManager) resetChainData() error {
	prevConfig := m.getConfig()

	err := m.removeChainData(prevConfig.DataDir, prevConfig.Name)
	if err != nil {
		return err
	}

	return m.restartNode()
}

func (m *NodeManager) removeChainData(dataDir, name string) error {
	chainDataDir := filepath.Join(dataDir, name, "lightchaindata")

	if _, err := os.Stat(chainDataDir); os.IsNotExist(err) {
		return err
	}

	if err := os.RemoveAll(chainDataDir); err != nil {
		return err
	}

	// send signal up to native app
	signal.Send(signal.Envelope{
		Type:  signal.EventChainDataRemoved,
		Event: struct{}{},
	})
	log.Info("Chain data has been removed", "dir", chainDataDir)

	return nil
}

// RestartNode restart running Status node, fails if node is not running.
func (m *NodeManager) RestartNode() error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	return m.restartNode()
}

// restartNode restart running Status node, fails if node is not running.
func (m *NodeManager) restartNode() error {
	prevConfig := m.getConfig()

	err := m.stop()
	if err != nil {
		return err
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

	les, err := m.getLesService()
	if err != nil {
		log.Warn("Cannot obtain LES service", "error", err)
		return nil, err
	}

	m.lesServiceLock.Lock()
	defer m.lesServiceLock.Unlock()

	m.lesService = les
	return m.lesService, nil
}

// WhisperService exposes reference to Whisper service running on top of the node.
func (m *NodeManager) WhisperService() (services.Whisper, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

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

	whisperService, err := m.getWhisperService()
	if err != nil {
		log.Warn("Cannot obtain whisper service", "error", err)
		return nil, err
	}

	m.whisperServiceLock.RLock()
	defer m.whisperServiceLock.RUnlock()

	return whisperv5.NewPublicWhisperAPI(whisperService), nil
}

// AccountManager exposes reference to node's accounts manager.
func (m *NodeManager) AccountManager() (*accounts.Manager, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

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
	m.rpcClientLock.RLock()
	defer m.rpcClientLock.RUnlock()

	if m.rpcClient == nil {
		return nil
	}

	return m.rpcClient
}

// GetStatusBackend exposes StatusBackend interface.
func (m *NodeManager) GetStatusBackend() (services.StatusBackend, error) {
	les, err := m.getLesService()
	if err != nil {
		return nil, err
	}

	return les.StatusBackend, nil
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

// setStarted sets node into started state: start channel closed, and send Started signal.
func (m *NodeManager) setStarted() {
	if !m.isNodePending() {
		return
	}

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

	m.setStoppedState()

	signal.Send(signal.Envelope{
		Type: signal.EventNodeCrashed,
		Event: signal.NodeCrashEvent{
			Error: err.Error(),
		},
	})
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
