package node

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
)

// node-related errors
var (
	ErrEthServiceRegistrationFailure     = errors.New("failed to register the Ethereum service")
	ErrWhisperServiceRegistrationFailure = errors.New("failed to register the Whisper service")
	ErrLightEthRegistrationFailure       = errors.New("failed to register the LES service")
	ErrNodeMakeFailure                   = errors.New("error creating p2p node")
	ErrNodeRunFailure                    = errors.New("error running p2p node")
	ErrNodeStartFailure                  = errors.New("error starting p2p node")
)

type StatusNode struct {
	sync.RWMutex
	Config  *params.NodeConfig
	Node    *node.Node
	Started chan struct{}
	Stopped chan struct{}
}

func New(config *params.NodeConfig) (*StatusNode, error) {
	stackConfig := defaultEmbeddedNodeConfig(config)
	n, err := node.New(stackConfig)

	if err != nil {
		return nil, err
	}

	return &StatusNode{
		Config:  config,
		Node:    n,
		Started: make(chan struct{}),
		Stopped: make(chan struct{}),
	}, nil
}

func (sn *StatusNode) Start() error {
	// TODO: make sure data directory exists
	// TODO: make sure keys directory exists
	// TODO: configure required node (should you need to update node's config, e.g. add bootstrap nodes, see node.Config)
	// TODO: Start Ethereum service if we are not expected to use an upstream server.
	// TODO: start Whisper service
	// TODO: return error if node already started
	// TODO: initialise logging
	// TODO: activate MailService required for Offline Inboxing

	sn.start()

	return nil
}

func (sn *StatusNode) start() {
	go func() {
		defer HaltOnPanic()
		// start underlying node
		if startErr := sn.Node.Start(); startErr != nil {
			close(sn.Started)
			sn.Lock()
			sn.Started = nil
			sn.Unlock()
			signal.Send(signal.Envelope{
				Type: signal.EventNodeCrashed,
				Event: signal.NodeCrashEvent{
					Error: fmt.Errorf("%v: %v", ErrNodeStartFailure, startErr).Error(),
				},
			})

			return
		}

		sn.Lock()
		// TODO: init RPC client for this node
		sn.Unlock()

		// underlying node is started, every method can use it, we use it immediately
		// TODO: PopulateStaticPeers

		// notify all subscribers that Status node is started
		close(sn.Started)
		signal.Send(signal.Envelope{
			Type:  signal.EventNodeStarted,
			Event: struct{}{},
		})

		// wait up until underlying node is stopped
		sn.Node.Wait()

		// notify sn.Stop() that node has been stopped
		close(sn.Stopped)
		log.Info("Node is stopped")
	}()
}

func defaultEmbeddedNodeConfig(config *params.NodeConfig) *node.Config {
	nc := &node.Config{
		DataDir:           config.DataDir,
		KeyStoreDir:       config.KeyStoreDir,
		UseLightweightKDF: true,
		NoUSB:             true,
		Name:              config.Name,
		Version:           config.Version,
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			DiscoveryV5Addr:  ":0",
			BootstrapNodes:   makeBootstrapNodes(),
			BootstrapNodesV5: makeBootstrapNodesV5(),
			ListenAddr:       config.ListenAddr,
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
			MaxPendingPeers:  config.MaxPendingPeers,
		},
		IPCPath:     makeIPCPath(config),
		HTTPCors:    []string{"*"},
		HTTPModules: strings.Split(config.APIModules, ","),
		WSHost:      makeWSHost(config),
		WSPort:      config.WSPort,
		WSOrigins:   []string{"*"},
		WSModules:   strings.Split(config.APIModules, ","),
	}

	if config.RPCEnabled {
		nc.HTTPHost = config.HTTPHost
		nc.HTTPPort = config.HTTPPort
	}

	if config.BootClusterConfig == nil || !config.BootClusterConfig.Enabled {
		nc.P2P.BootstrapNodes = nil
		nc.P2P.BootstrapNodesV5 = nil
	}

	return nc
}

// makeIPCPath returns IPC-RPC filename
func makeIPCPath(config *params.NodeConfig) string {
	if !config.IPCEnabled {
		return ""
	}

	return path.Join(config.DataDir, config.IPCFile)
}

// makeWSHost returns WS-RPC Server host, given enabled/disabled flag
func makeWSHost(config *params.NodeConfig) string {
	if !config.WSEnabled {
		return ""
	}

	return config.WSHost
}

// makeBootstrapNodes returns default (hence bootstrap) list of peers
func makeBootstrapNodes() []*discover.Node {
	// on desktops params.TestnetBootnodes and params.MainBootnodes,
	// on mobile client we deliberately keep this list empty
	enodes := []string{}

	var bootstrapNodes []*discover.Node
	for _, enode := range enodes {
		bootstrapNodes = append(bootstrapNodes, discover.MustParseNode(enode))
	}

	return bootstrapNodes
}

// makeBootstrapNodesV5 returns default (hence bootstrap) list of peers
func makeBootstrapNodesV5() []*discv5.Node {
	enodes := gethparams.DiscoveryV5Bootnodes

	var bootstrapNodes []*discv5.Node
	for _, enode := range enodes {
		bootstrapNodes = append(bootstrapNodes, discv5.MustParseNode(enode))
	}

	return bootstrapNodes
}
