package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/whisper/mailserver"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/params"
	shhmetrics "github.com/status-im/status-go/metrics/whisper"
	"github.com/status-im/status-go/shhext"
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

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/geth/node")

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
		logger.Info("Loading private key file", "file", config.NodeKeyFile)
		pk, err := crypto.LoadECDSA(config.NodeKeyFile)
		if err != nil {
			logger.Error("Failed loading private key file", "file", config.NodeKeyFile, "error", err)
		}

		// override node's private key
		stackConfig.P2P.PrivateKey = pk
	}

	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, ErrNodeMakeFailure
	}

	// Start Ethereum service if we are not expected to use an upstream server.
	if !config.UpstreamConfig.Enabled {
		if err := activateEthService(stack, config); err != nil {
			return nil, fmt.Errorf("%v: %v", ErrEthServiceRegistrationFailure, err)
		}
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
			NoDiscovery:     true,
			DiscoveryV5:     config.Discovery,
			ListenAddr:      config.ListenAddr,
			NAT:             nat.Any(),
			MaxPeers:        config.MaxPeers,
			MaxPendingPeers: config.MaxPendingPeers,
		},
		IPCPath:          makeIPCPath(config),
		HTTPCors:         []string{"*"},
		HTTPModules:      strings.Split(config.APIModules, ","),
		HTTPVirtualHosts: []string{"localhost"},
		WSHost:           makeWSHost(config),
		WSPort:           config.WSPort,
		WSOrigins:        []string{"*"},
		WSModules:        strings.Split(config.APIModules, ","),
	}

	if config.RPCEnabled {
		nc.HTTPHost = config.HTTPHost
		nc.HTTPPort = config.HTTPPort
	}

	if config.ClusterConfig.Enabled {
		nc.P2P.StaticNodes = parseNodes(config.ClusterConfig.StaticNodes)
		nc.P2P.BootstrapNodesV5 = parseNodesV5(config.ClusterConfig.BootNodes)
	}

	return nc
}

// activateEthService configures and registers the eth.Ethereum service with a given node.
func activateEthService(stack *node.Node, config *params.NodeConfig) error {
	if !config.LightEthConfig.Enabled {
		logger.Info("LES protocol is disabled")
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

	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return les.New(ctx, &ethConf)
	}); err != nil {
		return fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
	}

	return nil
}

// activateShhService configures Whisper and adds it to the given node.
func activateShhService(stack *node.Node, config *params.NodeConfig) error {

	if !config.WhisperConfig.Enabled {
		logger.Info("SHH protocol is disabled")
		return nil
	}
	var whisperService *whisper.Whisper
	if err := stack.Register(func(*node.ServiceContext) (node.Service, error) {
		whisperServiceConfig := &whisper.Config{
			MaxMessageSize:     whisper.DefaultMaxMessageSize,
			MinimumAcceptedPOW: 0.001,
		}
		whisperService = whisper.New(whisperServiceConfig)

		whisperConfig := config.WhisperConfig
		// enable metrics
		whisperService.RegisterEnvelopeTracer(&shhmetrics.EnvelopeTracer{})

		// enable mail service
		if whisperConfig.EnableMailServer {
			if whisperConfig.Password == "" {
				if err := whisperConfig.ReadPasswordFile(); err != nil {
					return nil, err
				}
			}

			logger.Info("Register MailServer")

			var mailServer mailserver.WMailServer
			whisperService.RegisterServer(&mailServer)
			mailServer.Init(whisperService, whisperConfig.DataDir, whisperConfig.Password, whisperConfig.MinimumPoW)
		}

		if whisperConfig.LightClient {
			emptyBloomFilter := make([]byte, 64)
			if err := whisperService.SetBloomFilter(emptyBloomFilter); err != nil {
				return nil, err
			}
		}

		return whisperService, nil
	}); err != nil {
		return err
	}
	// TODO(dshulyak) add a config option to enable it by default, but disable if app is started from statusd
	return stack.Register(func(*node.ServiceContext) (node.Service, error) {
		svc := shhext.New(shhext.SendEnvelopeSentSignal, whisperService)
		return svc, nil
	})
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

// parseNodes creates list of discover.Node out of enode strings.
func parseNodes(enodes []string) []*discover.Node {
	nodes := make([]*discover.Node, len(enodes))
	for i, enode := range enodes {
		nodes[i] = discover.MustParseNode(enode)
	}
	return nodes
}

// parseNodesV5 creates list of discv5.Node out of enode strings.
func parseNodesV5(enodes []string) []*discv5.Node {
	nodes := make([]*discv5.Node, len(enodes))
	for i, enode := range enodes {
		nodes[i] = discv5.MustParseNode(enode)
	}
	return nodes
}
