package node

import (
	"fmt"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
)

// HaltOnPanic recovers from panic, logs issue, sends upward notification, and exits
func HaltOnPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%v: %v", ErrNodeRunFailure, r)

		// send signal up to native app
		signal.Send(signal.Envelope{
			Type: signal.EventNodeCrashed,
			Event: signal.NodeCrashEvent{
				Error: err.Error(),
			},
		})

		common.Fatalf(err) // os.exit(1) is called internally
	}
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
