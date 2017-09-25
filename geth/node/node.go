package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/whisper/mailserver"
	"github.com/ethereum/go-ethereum/whisper/notifications"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
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

// MakeNode create a geth node entity
func MakeNode(config *params.NodeConfig) (*node.Node, error) {
	// make sure data directory exists
	if err := os.MkdirAll(filepath.Join(config.DataDir), os.ModePerm); err != nil {
		return nil, err
	}

	// make sure keys directory exists
	if err := os.MkdirAll(filepath.Join(config.KeyStoreDir), os.ModePerm); err != nil {
		return nil, err
	}

	// configure required node (should you need to update node's config, e.g. add bootstrap nodes, see node.Config)
	stackConfig := defaultEmbeddedNodeConfig(config)

	if len(config.NodeKeyFile) > 0 {
		log.Info("Loading private key file", "file", config.NodeKeyFile)
		pk, err := crypto.LoadECDSA(config.NodeKeyFile)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed loading private key file '%s': %v", config.NodeKeyFile, err))
		}

		// override node's private key
		stackConfig.P2P.PrivateKey = pk
	}

	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, ErrNodeMakeFailure
	}

	// start Ethereum service if we are not expected to use an upstream server.
	if !config.UpstreamConfig.Enabled {
		if err := activateEthService(stack, config); err != nil {
			return nil, fmt.Errorf("%v: %v", ErrEthServiceRegistrationFailure, err)
		}
	} else {
		// TODO(divan): FIXME: this is rude workaround for #294 issue
		// we start activate LES service to have RPC handler for `eth_accounts` call
		// should be removed once proper own RPC and refactoring is completed
		config.MaxPeers = 0
		if err := activateEthService(stack, config); err != nil {
			return nil, fmt.Errorf("%v: %v", ErrEthServiceRegistrationFailure, err)
		}
		log.Info("Blockchain synchronization is switched off, RPC requests will be proxied to " + config.UpstreamConfig.URL)
	}

	// start Whisper service
	if err := activateShhService(stack, config); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrWhisperServiceRegistrationFailure, err)
	}

	return stack, nil
}

// defaultEmbeddedNodeConfig returns default stack configuration for mobile client node
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
			ListenAddr:       ":0",
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

	return nc
}

// updateCHT changes trusted canonical hash trie root
func updateCHT(eth *les.LightEthereum, config *params.NodeConfig) {
	if !config.BootClusterConfig.Enabled {
		return
	}

	if config.BootClusterConfig.RootNumber == 0 {
		return
	}

	if config.BootClusterConfig.RootHash == "" {
		return
	}

	eth.WriteTrustedCht(light.TrustedCht{
		Number: uint64(config.BootClusterConfig.RootNumber),
		Root:   gethcommon.HexToHash(config.BootClusterConfig.RootHash),
	})
	log.Info("Added trusted CHT",
		"develop", config.DevMode, "number", config.BootClusterConfig.RootNumber, "hash", config.BootClusterConfig.RootHash)
}

// activateEthService configures and registers the eth.Ethereum service with a given node.
func activateEthService(stack *node.Node, config *params.NodeConfig) error {
	if !config.LightEthConfig.Enabled {
		log.Info("LES protocol is disabled")
		return nil
	}

	var genesis *core.Genesis
	if config.LightEthConfig.Genesis != "" {
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.LightEthConfig.Genesis), genesis); err != nil {
			return fmt.Errorf("invalid genesis spec: %v", err)
		}
	}

	ethConf := eth.DefaultConfig
	ethConf.Genesis = genesis
	ethConf.SyncMode = downloader.LightSync
	ethConf.NetworkId = config.NetworkID
	ethConf.DatabaseCache = config.LightEthConfig.DatabaseCache
	ethConf.MaxPeers = config.MaxPeers

	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		lightEth, err := les.New(ctx, &ethConf)
		if err == nil {
			updateCHT(lightEth, config)

			// TODO(influx6): Find a more concrete solution for node sync.
			// This is a temporary solution to provide a fix for node synchronization
			// when upstream is enabled. We will need to find something more suitable
			// later.
			if config.UpstreamConfig.Enabled {
				lightEth.Downloader().Terminate()
				if block := lightEth.BlockChain(); block != nil {
					block.Stop()
				}
			}
		}
		return lightEth, err
	}); err != nil {
		return fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
	}

	return nil
}

// activateShhService configures Whisper and adds it to the given node.
func activateShhService(stack *node.Node, config *params.NodeConfig) error {
	if !config.WhisperConfig.Enabled {
		log.Info("SHH protocol is disabled")
		return nil
	}

	serviceConstructor := func(*node.ServiceContext) (node.Service, error) {
		whisperConfig := config.WhisperConfig
		whisperService := whisper.New(nil)

		// enable mail service
		if whisperConfig.MailServerNode {
			password, err := whisperConfig.ReadPasswordFile()
			if err != nil {
				return nil, err
			}

			var mailServer mailserver.WMailServer
			whisperService.RegisterServer(&mailServer)
			mailServer.Init(whisperService, whisperConfig.DataDir, string(password), whisperConfig.MinimumPoW)
		}

		// enable notification service
		if whisperConfig.NotificationServerNode {
			var notificationServer notifications.NotificationServer
			whisperService.RegisterNotificationServer(&notificationServer)

			notificationServer.Init(whisperService, whisperConfig)
		}

		return whisperService, nil
	}

	return stack.Register(serviceConstructor)
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
