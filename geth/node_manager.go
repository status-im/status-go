package geth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv2"
)

// SelectedExtKey is a container for currently selected (logged in) account
type SelectedExtKey struct {
	Address     common.Address
	AccountKey  *accounts.Key
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

var (
	ErrDataDirPreprocessingFailed  = errors.New("failed to pre-process data directory")
	ErrInvalidGethNode             = errors.New("no running geth node detected")
	ErrInvalidAccountManager       = errors.New("could not retrieve account manager")
	ErrInvalidWhisperService       = errors.New("whisper service is unavailable")
	ErrInvalidLightEthereumService = errors.New("can not retrieve LES service")
	ErrInvalidClient               = errors.New("RPC client is not properly initialized")
	ErrInvalidJailedRequestQueue   = errors.New("jailed request queue is not properly initialized")
	ErrNodeMakeFailure             = errors.New("error creating p2p node")
	ErrNodeStartFailure            = errors.New("error starting p2p node")
	ErrInvalidNodeAPI              = errors.New("no node API connected")
)

var (
	nodeManagerInstance *NodeManager
	createOnce          sync.Once
)

// CreateAndRunNode creates and starts running Geth node locally (exposing given RPC port along the way)
func CreateAndRunNode(config *NodeConfig) error {
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
func NewNodeManager(config *NodeConfig) *NodeManager {
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
			glog.V(logger.Warn).Infoln(ErrInvalidAccountManager)
		}
		if err := m.node.geth.Service(&m.services.whisperService); err != nil {
			glog.V(logger.Warn).Infoln("cannot get whisper service:", err)
		}
		if err := m.node.geth.Service(&m.services.lightEthereum); err != nil {
			glog.V(logger.Warn).Infoln("cannot get light ethereum service:", err)
		}

		// setup handlers
		lightEthereum, err := m.LightEthereumService()
		if err != nil {
			panic("service stack misses LES")
		}

		lightEthereum.StatusBackend.SetTransactionQueueHandler(onSendTransactionRequest)
		lightEthereum.StatusBackend.SetAccountsFilterHandler(onAccountsListRequest)
		lightEthereum.StatusBackend.SetTransactionReturnHandler(onSendTransactionReturn)

		m.services.rpcClient, err = m.node.geth.Attach()
		if err != nil {
			glog.V(logger.Warn).Infoln("cannot get RPC client service:", ErrInvalidClient)
		}

		// expose API
		m.api = node.NewPrivateAdminAPI(m.node.geth)

		m.populateStaticPeers()

		m.onNodeStarted() // node started, notify listeners
		m.node.geth.Wait()

		glog.V(logger.Info).Infoln("node stopped")
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

	// allow interrupting running nodes
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		defer signal.Stop(sigc)
		<-sigc
		glog.V(logger.Info).Infoln("Got interrupt, shutting down...")
		go m.node.geth.Stop()
		for i := 3; i > 0; i-- {
			<-sigc
			if i > 1 {
				glog.V(logger.Info).Infof("Already shutting down, interrupt %d more times for panic.", i-1)
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

	m.node.geth.Stop()
	m.node.started = make(chan struct{})
	return nil
}

// RestartNode restarts P2P node
func (m *NodeManager) RestartNode() error {
	if m == nil || !m.NodeInited() {
		return ErrInvalidGethNode
	}

	m.StopNode()
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

	// re-select the previously selected account
	if err := ReSelectAccount(); err != nil {
		return err
	}

	return nil
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

	if err := m.ResumeNode(); err != nil {
		return err
	}

	return nil
}

func (m *NodeManager) StartNodeRPCServer() (bool, error) {
	if m == nil || !m.NodeInited() {
		return false, ErrInvalidGethNode
	}

	if m.api == nil {
		return false, ErrInvalidNodeAPI
	}

	config := m.node.gethConfig
	modules := strings.Join(config.HTTPModules, ",")

	return m.api.StartRPC(&config.HTTPHost, &config.HTTPPort, &config.HTTPCors, &modules)
}

// StopNodeRPCServer stops HTTP RPC service attached to node
func (m *NodeManager) StopNodeRPCServer() (bool, error) {
	if m == nil || !m.NodeInited() {
		return false, ErrInvalidGethNode
	}

	if m.api == nil {
		return false, ErrInvalidNodeAPI
	}

	return m.api.StopRPC()
}

// HasNode checks whether manager has initialized node attached
func (m *NodeManager) NodeInited() bool {
	if m == nil || !m.node.Inited() {
		return false
	}

	return true
}

// AccountManager exposes reference to accounts manager
func (m *NodeManager) AccountManager() (*accounts.Manager, error) {
	if m == nil || !m.NodeInited() {
		return nil, ErrInvalidGethNode
	}

	return m.node.geth.AccountManager(), nil
}

// LightEthereumService exposes LES
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

// populateStaticPeers connects current node with our publicly available LES cluster
func (m *NodeManager) populateStaticPeers() {
	// manually add static nodes (LES auto-discovery is not stable yet)
	configFile, err := ioutil.ReadFile(filepath.Join("../data", "static-nodes.json"))
	if err != nil {
		return
	}
	var enodes []string
	if err = json.Unmarshal(configFile, &enodes); err != nil {
		return
	}

	for _, enode := range enodes {
		m.AddPeer(enode)
	}
}

func (k *SelectedExtKey) Hex() string {
	if k == nil {
		return "0x0"
	}

	return k.Address.Hex()
}
