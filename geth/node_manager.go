package geth

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/params"
)

// SelectedExtKey is a container for currently selected (logged in) account
type SelectedExtKey struct {
	Address     common.Address
	AccountKey  *keystore.Key
	SubAccounts []accounts.Account
}

// NodeManager manages Status node (which abstracts contained geth node)
type NodeManager struct {
	node            *Node                 // reference to Status node
	services        *NodeServiceStack     // default stack of services running on geth node
	api             *node.PrivateAdminAPI // exposes collection of administrative API methods
	SelectedAccount *SelectedExtKey       // account that was processed during the last call to SelectAccount()
}

// NodeServiceStack contains "standard" node services (which are always available)
type NodeServiceStack struct {
	lightEthereum      *les.LightEthereum  // LES service
	whisperService     *whisper.Whisper    // Whisper service
	rpcClient          *rpc.Client         // RPC client
	jailedRequestQueue *JailedRequestQueue // bridge via which jail notifies node of incoming requests
}

// errors
var (
	ErrInvalidGethNode             = errors.New("no running geth node detected")
	ErrInvalidAccountManager       = errors.New("could not retrieve account manager")
	ErrInvalidWhisperService       = errors.New("whisper service is unavailable")
	ErrInvalidLightEthereumService = errors.New("can not retrieve LES service")
	ErrInvalidClient               = errors.New("RPC client is not properly initialized")
	ErrInvalidJailedRequestQueue   = errors.New("jailed request queue is not properly initialized")
	ErrNodeMakeFailure             = errors.New("error creating p2p node")
	ErrNodeStartFailure            = errors.New("error starting p2p node")
	ErrNodeRunFailure              = errors.New("error running p2p node")
	ErrInvalidNodeAPI              = errors.New("no node API connected")
	ErrAccountKeyStoreMissing      = errors.New("account key store is not set")
)

var (
	nodeManagerInstance *NodeManager
	createOnce          sync.Once
)

// CreateAndRunNode creates and starts running Geth node locally (exposing given RPC port along the way)
func CreateAndRunNode(config *params.NodeConfig) error {
	defer HaltOnPanic()

	nodeManager := NewNodeManager(config)

	if nodeManager.NodeInited() {
		nodeManager.RunNode()
		nodeManager.WaitNodeStarted()
		return nil
	}

	return ErrNodeStartFailure
}

// NewNodeManager makes new instance of node manager
func NewNodeManager(config *params.NodeConfig) *NodeManager {
	createOnce.Do(func() {
		nodeManagerInstance = &NodeManager{
			services: &NodeServiceStack{
				jailedRequestQueue: NewJailedRequestsQueue(),
			},
		}
		nodeManagerInstance.node = MakeNode(config)
	})

	return nodeManagerInstance
}

// NodeManagerInstance exposes node manager instance
func NodeManagerInstance() *NodeManager {
	return nodeManagerInstance
}

// RunNode starts Geth node
func (m *NodeManager) RunNode() {
	go func() {
		defer HaltOnPanic()

		m.StartNode()

		if _, err := m.AccountManager(); err != nil {
			log.Warn(ErrInvalidAccountManager.Error())
		}
		if err := m.node.geth.Service(&m.services.whisperService); err != nil {
			log.Warn("cannot get whisper service", "error", err)
		}
		if err := m.node.geth.Service(&m.services.lightEthereum); err != nil {
			log.Warn("cannot get light ethereum service", "error", err)
		}

		// setup handlers
		if lightEthereum, err := m.LightEthereumService(); err == nil {
			lightEthereum.StatusBackend.SetTransactionQueueHandler(onSendTransactionRequest)
			lightEthereum.StatusBackend.SetAccountsFilterHandler(onAccountsListRequest)
			lightEthereum.StatusBackend.SetTransactionReturnHandler(onSendTransactionReturn)
		}

		var err error
		m.services.rpcClient, err = m.node.geth.Attach()
		if err != nil {
			log.Warn("cannot get RPC client service", "error", ErrInvalidClient)
		}

		// expose API
		m.api = node.NewPrivateAdminAPI(m.node.geth)

		m.PopulateStaticPeers()

		m.onNodeStarted() // node started, notify listeners
		m.node.geth.Wait()

		log.Info("node stopped")
	}()
}

// StartNode starts running P2P node
func (m *NodeManager) StartNode() {
	if m == nil || !m.NodeInited() {
		panic(ErrInvalidGethNode)
	}

	if err := m.node.geth.Start(); err != nil {
		panic(fmt.Sprintf("%v: %v", ErrNodeStartFailure, err))
	}

	if server := m.node.geth.Server(); server != nil {
		if nodeInfo := server.NodeInfo(); nodeInfo != nil {
			log.Info(nodeInfo.Enode)
		}
	}

	// allow interrupting running nodes
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go m.node.geth.Stop() // nolint: errcheck
		for i := 3; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info(fmt.Sprintf("Already shutting down, interrupt %d more times for panic.", i-1))
			}
		}
		panic("interrupted!")
	}()
}

// StopNode stops running P2P node
func (m *NodeManager) StopNode() error {
	if m == nil || !m.NodeInited() {
		return ErrInvalidGethNode
	}

	if err := m.node.geth.Stop(); err != nil {
		return err
	}
	m.node.started = make(chan struct{})
	return nil
}

// RestartNode restarts P2P node
func (m *NodeManager) RestartNode() error {
	if m == nil || !m.NodeInited() {
		return ErrInvalidGethNode
	}

	if err := m.StopNode(); err != nil {
		return err
	}
	m.RunNode()
	m.WaitNodeStarted()

	return nil
}

// ResumeNode resumes previously stopped P2P node
func (m *NodeManager) ResumeNode() error {
	if m == nil || !m.NodeInited() {
		return ErrInvalidGethNode
	}

	m.RunNode()
	m.WaitNodeStarted()

	return ReSelectAccount()
}

// ResetChainData purges chain data (by removing data directory). Safe to apply on running P2P node.
func (m *NodeManager) ResetChainData() error {
	if m == nil || !m.NodeInited() {
		return ErrInvalidGethNode
	}

	if err := m.StopNode(); err != nil {
		return err
	}

	chainDataDir := filepath.Join(m.node.gethConfig.DataDir, m.node.gethConfig.Name, "lightchaindata")
	if _, err := os.Stat(chainDataDir); os.IsNotExist(err) {
		return err
	}
	if err := os.RemoveAll(chainDataDir); err != nil {
		return err
	}
	log.Info("chaindata removed", "dir", chainDataDir)

	return m.ResumeNode()
}

// StartNodeRPCServer starts HTTP RPC server
func (m *NodeManager) StartNodeRPCServer() (bool, error) {
	if m == nil || !m.NodeInited() {
		return false, ErrInvalidGethNode
	}

	if m.api == nil {
		return false, ErrInvalidNodeAPI
	}

	config := m.node.gethConfig
	modules := strings.Join(config.HTTPModules, ",")
	cors := strings.Join(config.HTTPCors, ",")

	return m.api.StartRPC(&config.HTTPHost, &config.HTTPPort, &cors, &modules)
}

// StopNodeRPCServer stops HTTP RPC server attached to node
func (m *NodeManager) StopNodeRPCServer() (bool, error) {
	if m == nil || !m.NodeInited() {
		return false, ErrInvalidGethNode
	}

	if m.api == nil {
		return false, ErrInvalidNodeAPI
	}

	return m.api.StopRPC()
}

// NodeInited checks whether manager has initialized node attached
func (m *NodeManager) NodeInited() bool {
	if m == nil || !m.node.Inited() {
		return false
	}

	return true
}

// Node returns attached node if it has been initialized
func (m *NodeManager) Node() *Node {
	if !m.NodeInited() {
		return nil
	}

	return m.node
}

// AccountManager exposes reference to accounts manager
func (m *NodeManager) AccountManager() (*accounts.Manager, error) {
	if m == nil || !m.NodeInited() {
		return nil, ErrInvalidGethNode
	}

	return m.node.geth.AccountManager(), nil
}

// AccountKeyStore exposes reference to accounts key store
func (m *NodeManager) AccountKeyStore() (*keystore.KeyStore, error) {
	if m == nil || !m.NodeInited() {
		return nil, ErrInvalidGethNode
	}

	accountManager, err := m.AccountManager()
	if err != nil {
		return nil, err
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

// LightEthereumService exposes LES
// nolint: dupl
func (m *NodeManager) LightEthereumService() (*les.LightEthereum, error) {
	if m == nil || !m.NodeInited() {
		return nil, ErrInvalidGethNode
	}

	if m.services.lightEthereum == nil {
		return nil, ErrInvalidLightEthereumService
	}

	return m.services.lightEthereum, nil
}

// WhisperService exposes Whisper service
// nolint: dupl
func (m *NodeManager) WhisperService() (*whisper.Whisper, error) {
	if m == nil || !m.NodeInited() {
		return nil, ErrInvalidGethNode
	}

	if m.services.whisperService == nil {
		return nil, ErrInvalidWhisperService
	}

	return m.services.whisperService, nil
}

// RPCClient exposes Geth's RPC client
// nolint: dupl
func (m *NodeManager) RPCClient() (*rpc.Client, error) {
	if m == nil || !m.NodeInited() {
		return nil, ErrInvalidGethNode
	}

	if m.services.rpcClient == nil {
		return nil, ErrInvalidClient
	}

	return m.services.rpcClient, nil
}

// JailedRequestQueue exposes reference to queue of jailed requests
func (m *NodeManager) JailedRequestQueue() (*JailedRequestQueue, error) {
	if m == nil || !m.NodeInited() {
		return nil, ErrInvalidGethNode
	}

	if m.services.jailedRequestQueue == nil {
		return nil, ErrInvalidJailedRequestQueue
	}

	return m.services.jailedRequestQueue, nil
}

// AddPeer adds new peer node
func (m *NodeManager) AddPeer(url string) (bool, error) {
	if m == nil || !m.NodeInited() {
		return false, ErrInvalidGethNode
	}

	server := m.node.geth.Server()
	if server == nil {
		return false, ErrInvalidGethNode
	}

	// Try to add the url as a static peer and return
	parsedNode, err := discover.ParseNode(url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.AddPeer(parsedNode)

	return true, nil
}

// WaitNodeStarted blocks until node is started (start channel gets notified)
func (m *NodeManager) WaitNodeStarted() {
	<-m.node.started // block until node is started
}

// onNodeStarted sends upward notification letting the app know that Geth node is ready to be used
func (m *NodeManager) onNodeStarted() {
	// notify local listener
	m.node.started <- struct{}{}
	close(m.node.started)

	// send signal up to native app
	SendSignal(SignalEnvelope{
		Type:  EventNodeStarted,
		Event: struct{}{},
	})
}

// PopulateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (m *NodeManager) PopulateStaticPeers() {
	for _, enode := range params.TestnetBootnodes {
		m.AddPeer(enode) // nolint: errcheck
	}
}

// Hex dumps address of a given extended key as hex string
func (k *SelectedExtKey) Hex() string {
	if k == nil {
		return "0x0"
	}

	return k.Address.Hex()
}
