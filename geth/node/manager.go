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
	eles "github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common/geth"
	"github.com/status-im/status-go/geth/common/services"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
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
	state  *int32
	constr geth.NodeConstructor // Status node configuration
	lock   *sync.RWMutex

	node    geth.Node
	les     *les
	whisper *whisper
	rpc     rpcAccess

	logger logger
}

// NewNodeManager makes new instance of node manager
func NewNodeManager() *NodeManager {
	var isStarted int32 = pending

	m := &NodeManager{
		state:   &isStarted,
		lock:    &sync.RWMutex{},
		node:    newNode(),
		les:     newLES(),
		whisper: newWhisper(),
		rpc:     newRPC(),
		logger:  newLog(),
	}

	go HaltOnInterruptSignal(m) // allow interrupting running nodes

	return m
}

// StartNode start Status node, fails if node is already started.
func (m *NodeManager) StartNode(constr geth.NodeConstructor) error {
	if m.isNodeStarted() {
		return ErrNodeExists
	}

	m.setNodeConstructor(constr)
	m.initNodeLog()
	m.setPendingState()

	ethNode, err := m.newNode()
	if err != nil {
		return err
	}

	return m.startNode(ethNode)
}

func (m *NodeManager) setNodeConstructor(constr geth.NodeConstructor) {
	m.lock.Lock()
	m.constr = constr
	m.lock.Unlock()
}

// startNode start Status node, fails if node is already started.
func (m *NodeManager) startNode(ethNode geth.Node) error {
	m.setPendingState()

	defer HaltOnPanic()

	m.node.SetNode(ethNode.GetNode())
	m.constr.SetConfig(m.constr.Config())
	err := m.initRPCClient(ethNode)
	if err != nil {
		return err
	}

	// underlying node is started, every method can use it, we use it immediately
	go func() {
		if err := m.PopulateStaticPeers(); err != nil {
			log.Error("Static peers population", "error", err)
		}
	}()

	m.setStarted()

	return nil
}

func (m *NodeManager) newNode() (geth.Node, error) {
	var err error
	defer func() {
		if err != nil {
			m.setFailed(fmt.Errorf("%v: %v", ErrNodeStartFailure, err))
		}
	}()

	ethNode, err := m.constr.Make()
	if err != nil {
		return nil, err
	}

	if err = ethNode.Start(); err != nil {
		return nil, ErrInvalidNodeManager
	}

	return ethNode, nil
}

// initRPCClient up on given node, in case an error stops node.
func (m *NodeManager) initRPCClient(node geth.Node) error {
	err := m.rpc.Init(node.GetNode(), m.getUpstreamConfig())
	if err != nil {
		log.Error("Init RPC client failed:", "error", err)
		m.setFailed(ErrRPCClient)

		return ErrRPCClient
	}

	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) StopNode() error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	return m.stop()
}

func (m *NodeManager) stop() error {
	err := m.node.Stop()
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

	return m.node, nil
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
func (m *NodeManager) ResetChainData() error {
	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	return m.resetChainData()
}

// resetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *NodeManager) resetChainData() error {
	config := m.constr.Config()

	err := m.removeChainData(config.DataDir, config.Name)
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
	err := m.stop()
	if err != nil {
		return err
	}

	ethNode, err := m.newNode()
	if err != nil {
		return err
	}

	return m.startNode(ethNode)
}

// NodeConfig exposes reference to running node's configuration.
func (m *NodeManager) NodeConfig() (*params.NodeConfig, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	return m.constr.Config(), nil
}

// LightEthereumService exposes reference to LES service running on top of the node.
func (m *NodeManager) LightEthereumService() (services.LesService, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	les, err := m.getLesServices()
	if err != nil {
		log.Warn("Cannot obtain LES service", "error", err)
		return nil, err
	}

	les.Lock()
	defer les.Unlock()

	return les.l, nil
}

// WhisperService exposes reference to Whisper service running on top of the node.
func (m *NodeManager) WhisperService() (services.Whisper, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.whisper.Lock()
	defer m.whisper.Unlock()

	whisperService, err := m.getWhisperServices()
	if err != nil {
		log.Warn("Cannot obtain whisper service", "error", err)
		return nil, err
	}

	return whisperService.w, nil
}

// PublicWhisperAPI exposes reference to public Whisper API.
func (m *NodeManager) PublicWhisperAPI() (services.WhisperAPI, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	m.whisper.Lock()
	defer m.whisper.Unlock()

	whisperServices, err := m.getWhisperServices()
	if err != nil {
		log.Warn("Cannot obtain whisper service", "error", err)
		return nil, err
	}

	return whisperServices.api, nil
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
	return m.rpc.Client()
}

// GetStatusBackend exposes StatusBackend interface.
func (m *NodeManager) GetStatusBackend() (services.StatusBackend, error) {
	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	les, err := m.getLesServices()
	if err != nil {
		return nil, err
	}

	return les.back, nil
}

// initNodeLog initializes global logger parameters based on
// provided node configurations.
func (m *NodeManager) initNodeLog() {
	config := m.constr.Config()

	m.logger.Init(config.LogFile, config.LogLevel)
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

func (m *NodeManager) getBootClusterEnabled() bool {
	config := m.constr.Config()
	bootCluster := config.BootClusterConfig.Enabled

	return bootCluster
}

func (m *NodeManager) getUpstreamConfig() params.UpstreamRPCConfig {
	config := m.constr.Config()
	upstreamConfig := config.UpstreamConfig

	return upstreamConfig
}

func (m *NodeManager) getBootNodes() []string {
	config := m.constr.Config()
	nodes := make([]string, len(config.BootClusterConfig.BootNodes))
	copy(nodes, config.BootClusterConfig.BootNodes)

	return nodes
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

// getWhisperServices returns Whisper services or inits them
func (m *NodeManager) getWhisperServices() (*whisper, error) {
	if m.whisper.w != nil {
		return m.whisper, nil
	}

	var whisperObject *whisperv5.Whisper
	err := m.node.Service(&whisperObject)
	if err != nil {
		return nil, ErrInvalidWhisperService
	}

	m.whisper.w = whisperObject
	m.whisper.api = whisperv5.NewPublicWhisperAPI(whisperObject)

	return m.whisper, nil
}

// getLesService returns LES service or inits both LES and back
func (m *NodeManager) getLesServices() (*les, error) {
	m.les.Lock()
	defer m.les.Unlock()

	if m.les.l != nil {
		return m.les, nil
	}

	lesObject := &eles.LightEthereum{}
	err := m.node.Service(&lesObject)
	if err != nil {
		return nil, ErrInvalidLightEthereumService
	}

	m.les.l = lesObject
	m.les.back = lesObject.StatusBackend

	return m.les, nil
}
