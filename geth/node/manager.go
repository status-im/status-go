package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethmessage "github.com/ethereum/go-ethereum/common/message"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/notifications"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
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
	ErrInvalidDeliveryService      = errors.New("delivery service is unavailable")
	ErrInvalidLightEthereumService = errors.New("LES service is unavailable")
	ErrInvalidAccountManager       = errors.New("could not retrieve account manager")
	ErrAccountKeyStoreMissing      = errors.New("account key store is not set")
	ErrRPCClient                   = errors.New("failed to init RPC client")
)

// NodeManager manages Status node (which abstracts contained geth node)
// nolint: golint
// should be fixed at https://github.com/status-im/status-go/issues/200
type NodeManager struct {
	sync.RWMutex
	config          *params.NodeConfig             // Status node configuration
	node            *node.Node                     // reference to Geth P2P stack/node
	nodeStarted     chan struct{}                  // channel to wait for start up notifications
	nodeStopped     chan struct{}                  // channel to wait for termination notifications
	whisperService  *whisper.Whisper               // reference to Whisper service
	deliveryService *notifications.DeliveryService // reference to Delivery service
	lesService      *les.LightEthereum             // reference to LES service
	rpcClient       *rpc.Client                    // reference to RPC client
}

// NewNodeManager makes new instance of node manager
func NewNodeManager() *NodeManager {
	m := &NodeManager{}
	go HaltOnInterruptSignal(m) // allow interrupting running nodes

	return m
}

// StartNode start Status node, fails if node is already started
func (m *NodeManager) StartNode(config *params.NodeConfig) (<-chan struct{}, error) {
	m.Lock()
	defer m.Unlock()

	return m.startNode(config)
}

// startNode start Status node, fails if node is already started
func (m *NodeManager) startNode(config *params.NodeConfig) (<-chan struct{}, error) {
	if m.node != nil || m.nodeStarted != nil {
		return nil, ErrNodeExists
	}

	m.initLog(config)

	deliveryManager := new(notifications.DeliveryService)

	ethNode, err := MakeNode(config)
	if err != nil {
		return nil, err
	}

	m.deliveryService = deliveryManager
	m.nodeStarted = make(chan struct{}, 1)

	// Subscribe for message delivery status and log out with special key.
	messageStateLoggingID := deliveryManager.Subscribe(func(state notifications.DeliveryState) {
		if state.IsP2P {
			var payload []byte
			var from, to string

			if state.P2P.Direction == gethmessage.IncomingMessage {
				if state.P2P.Received != nil {
					payload = state.P2P.Received.Payload

					if state.P2P.Received.Src != nil {
						from = gethcommon.ToHex(crypto.FromECDSAPub(state.P2P.Received.Src))
					}

					if state.P2P.Received.Dst != nil {
						to = gethcommon.ToHex(crypto.FromECDSAPub(state.P2P.Received.Dst))
					}
				}
			}

			if state.P2P.Direction == gethmessage.OutgoingMessage {
				from = state.P2P.Source.Sig

				if len(state.P2P.Source.PublicKey) == 0 {
					to = string(state.P2P.Source.PublicKey)
				} else {
					to = state.P2P.Source.TargetPeer
				}
			}

			if stat, err := json.Marshal(common.MessageStat{
				Payload:         payload,
				FromDevice:      from,
				ToDevice:        to,
				Source:          state.P2P.Source,
				RejectionReason: state.P2P.Reason,
				Envelope:        state.P2P.Envelope.Data,
				Status:          state.P2P.Status.String(),
				Type:            state.P2P.Direction.String(),
				Hash:            state.P2P.Envelope.Hash().String(),
				TimeSent:        state.P2P.Envelope.Expiry - state.P2P.Envelope.TTL,
			}); err == nil {
				log.Info(fmt.Sprintf("%s : P2P : %s : %s : %+s", params.MessageStatHeader, state.P2P.Direction.String(), state.P2P.Status.String(), string(stat)))
			}
			return
		}

		var payload []byte
		var from, to string

		if state.RPC.Direction == gethmessage.IncomingMessage {
			if state.RPC.Received != nil {
				payload = state.RPC.Received.Payload

				if state.RPC.Received.Src != nil {
					from = gethcommon.ToHex(crypto.FromECDSAPub(state.RPC.Received.Src))
				}

				if state.RPC.Received.Dst != nil {
					to = gethcommon.ToHex(crypto.FromECDSAPub(state.RPC.Received.Dst))
				}
			}
		}

		if state.RPC.Direction == gethmessage.OutgoingMessage {
			from = state.RPC.Source.Sig

			if len(state.RPC.Source.PublicKey) == 0 {
				to = string(state.RPC.Source.PublicKey)
			} else {
				to = state.RPC.Source.TargetPeer
			}
		}

		if stat, err := json.Marshal(common.MessageStat{
			Payload:         payload,
			FromDevice:      from,
			ToDevice:        to,
			Source:          state.RPC.Source,
			RejectionReason: state.RPC.Reason,
			Envelope:        state.RPC.Envelope.Data,
			Status:          state.RPC.Status.String(),
			Type:            state.RPC.Direction.String(),
			Hash:            state.RPC.Envelope.Hash().String(),
			TimeSent:        state.RPC.Envelope.Expiry - state.RPC.Envelope.TTL,
		}); err == nil {
			log.Info(fmt.Sprintf("%s : RPC : %s : %s : %+s", params.MessageStatHeader, state.RPC.Direction.String(), state.RPC.Status.String(), string(stat)))
		}
	})

	go func() {
		defer deliveryManager.Unsubscribe(messageStateLoggingID)
		defer HaltOnPanic()

		// start underlying node
		if err := ethNode.Start(); err != nil {
			close(m.nodeStarted)
			m.Lock()
			m.nodeStarted = nil
			m.Unlock()
			signal.Send(signal.Envelope{
				Type: signal.EventNodeCrashed,
				Event: signal.NodeCrashEvent{
					Error: fmt.Errorf("%v: %v", ErrNodeStartFailure, err).Error(),
				},
			})
			return
		}

		m.Lock()
		m.node = ethNode
		m.nodeStopped = make(chan struct{}, 1)
		m.config = config

		// init RPC client for this node
		m.rpcClient, err = rpc.NewClient(m.node, m.config.UpstreamConfig)
		if err != nil {
			log.Error("Init RPC client failed:", "error", err)
			m.Unlock()
			signal.Send(signal.Envelope{
				Type: signal.EventNodeCrashed,
				Event: signal.NodeCrashEvent{
					Error: ErrRPCClient.Error(),
				},
			})
			return
		}
		m.Unlock()

		// underlying node is started, every method can use it, we use it immediately
		go func() {
			if err := m.PopulateStaticPeers(); err != nil {
				log.Error("Static peers population", "error", err)
			}
		}()

		// notify all subscribers that Status node is started
		close(m.nodeStarted)
		signal.Send(signal.Envelope{
			Type:  signal.EventNodeStarted,
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
	m.Lock()
	defer m.Unlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}
	if m.nodeStopped == nil {
		return nil, ErrNoRunningNode
	}

	<-m.nodeStarted // make sure you operate on fully started node

	return m.stopNode()
}

// stopNode stop Status node. Stopped node cannot be resumed.
func (m *NodeManager) stopNode() (<-chan struct{}, error) {
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
		m.nodeStarted = nil
		m.node = nil
		m.Unlock()

		close(nodeStopped) // Status node is stopped, and we can create another
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
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return false
	}

	<-m.nodeStarted

	return true
}

// Node returns underlying Status node
func (m *NodeManager) Node() (*node.Node, error) {
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	<-m.nodeStarted

	return m.node, nil
}

// PopulateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (m *NodeManager) PopulateStaticPeers() error {
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	<-m.nodeStarted

	return m.populateStaticPeers()
}

// populateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (m *NodeManager) populateStaticPeers() error {
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
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return err
	}

	<-m.nodeStarted

	return m.addPeer(url)
}

// addPeer adds new static peer node
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
func (m *NodeManager) ResetChainData() (<-chan struct{}, error) {
	m.Lock()
	defer m.Unlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	<-m.nodeStarted

	return m.resetChainData()
}

// resetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *NodeManager) resetChainData() (<-chan struct{}, error) {
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
	signal.Send(signal.Envelope{
		Type:  signal.EventChainDataRemoved,
		Event: struct{}{},
	})
	log.Info("Chain data has been removed", "dir", chainDataDir)

	return m.startNode(&prevConfig)
}

// RestartNode restart running Status node, fails if node is not running
func (m *NodeManager) RestartNode() (<-chan struct{}, error) {
	m.Lock()
	defer m.Unlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	<-m.nodeStarted

	return m.restartNode()
}

// restartNode restart running Status node, fails if node is not running
func (m *NodeManager) restartNode() (<-chan struct{}, error) {
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
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	<-m.nodeStarted

	return m.config, nil
}

// LightEthereumService exposes reference to LES service running on top of the node
func (m *NodeManager) LightEthereumService() (*les.LightEthereum, error) {
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
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
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
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

// DeliveryService returns reference to running Whisper service
func (m *NodeManager) DeliveryService() (*notifications.DeliveryService, error) {
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
	}

	<-m.nodeStarted

	if m.deliveryService == nil {
		return nil, ErrInvalidDeliveryService
	}

	return m.deliveryService, nil
}

// AccountManager exposes reference to node's accounts manager
func (m *NodeManager) AccountManager() (*accounts.Manager, error) {
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
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
	m.RLock()
	defer m.RUnlock()

	if err := m.isNodeAvailable(); err != nil {
		return nil, err
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

// RPCClient exposes reference to RPC client connected to the running node.
func (m *NodeManager) RPCClient() *rpc.Client {
	m.Lock()
	defer m.Unlock()

	return m.rpcClient
}

// initLog initializes global logger parameters based on
// provided node configurations.
func (m *NodeManager) initLog(config *params.NodeConfig) {
	log.SetLevel(config.LogLevel)

	if config.LogFile != "" {
		err := log.SetLogFile(config.LogFile)
		if err != nil {
			fmt.Println("Failed to open log file, using stdout")
		}
	}
}

// isNodeAvailable check if we have a node running and make sure is fully started
func (m *NodeManager) isNodeAvailable() error {
	if m.nodeStarted == nil || m.node == nil {
		return ErrNoRunningNode
	}

	return nil
}
