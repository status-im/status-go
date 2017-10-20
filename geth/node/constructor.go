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
	eles "github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/light"
	enode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/whisper/mailserver"
	"github.com/ethereum/go-ethereum/whisper/notifications"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common/geth"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	"sync/atomic"
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

type nodeConstructor struct {
	config atomic.Value
}

func NewNodeConstructor(config *params.NodeConfig) *nodeConstructor {
	c := &nodeConstructor{}
	c.SetConfig(config)
	return c
}

// Make create a geth node entity
func (c *nodeConstructor) Make() (geth.Node, error) {
	config := c.getConfig()

	// make sure data directory exists
	if err := os.MkdirAll(filepath.Join(config.DataDir), os.ModePerm); err != nil {
		return nil, err
	}

	// make sure keys directory exists
	if err := os.MkdirAll(filepath.Join(config.KeyStoreDir), os.ModePerm); err != nil {
		return nil, err
	}

	// configure required node (should you need to update node's c, e.g. add bootstrap nodes, see node.Config)
	stackConfig := c.defaultEmbeddedNodeConfig()

	if len(config.NodeKeyFile) > 0 {
		log.Info("Loading private key file", "file", config.NodeKeyFile)
		pk, err := crypto.LoadECDSA(config.NodeKeyFile)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed loading private key file '%s': %v", config.NodeKeyFile, err))
		}

		// override node's private key
		stackConfig.P2P.PrivateKey = pk
	}

	stack, err := enode.New(stackConfig)
	if err != nil {
		return nil, ErrNodeMakeFailure
	}

	// Start Ethereum service if we are not expected to use an upstream server.
	if !config.UpstreamConfig.Enabled {
		if err := c.activateEthService(stack); err != nil {
			return nil, fmt.Errorf("%v: %v", ErrEthServiceRegistrationFailure, err)
		}
	}

	// start Whisper service
	if err := c.activateShhService(stack); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrWhisperServiceRegistrationFailure, err)
	}

	return newNode(stack), nil
}

func (c *nodeConstructor) Config() *params.NodeConfig {
	return c.getConfig()
}

func (c *nodeConstructor) SetConfig(config *params.NodeConfig) {
	c.config.Store(*config)
}

func (c *nodeConstructor) getConfig() *params.NodeConfig {
	config := c.config.Load()
	configValue := config.(params.NodeConfig)

	return &configValue
}

// defaultEmbeddedNodeConfig returns default stack configuration for mobile client node
func (c *nodeConstructor) defaultEmbeddedNodeConfig() *enode.Config {
	config := c.getConfig()

	nc := &enode.Config{
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
		IPCPath:     c.makeIPCPath(),
		HTTPCors:    []string{"*"},
		HTTPModules: strings.Split(config.APIModules, ","),
		WSHost:      c.makeWSHost(),
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
func (c *nodeConstructor) updateCHT(eth *eles.LightEthereum) {
	config := c.getConfig()

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
func (c *nodeConstructor) activateEthService(stack *enode.Node) error {
	config := c.getConfig()

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

	if err := stack.Register(func(ctx *enode.ServiceContext) (enode.Service, error) {
		lightEth, err := eles.New(ctx, &ethConf)
		if err == nil {
			c.updateCHT(lightEth)
		}

		return lightEth, err
	}); err != nil {
		return fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
	}

	return nil
}

// activateShhService configures Whisper and adds it to the given node.
func (c *nodeConstructor) activateShhService(stack *enode.Node) error {
	config := c.getConfig()

	if !config.WhisperConfig.Enabled {
		log.Info("SHH protocol is disabled")
		return nil
	}

	serviceConstructor := func(*enode.ServiceContext) (enode.Service, error) {
		whisperConfig := config.WhisperConfig
		whisperService := whisperv5.New(nil)

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
func (c *nodeConstructor) makeIPCPath() string {
	config := c.getConfig()

	if !config.IPCEnabled {
		return ""
	}

	return path.Join(config.DataDir, config.IPCFile)
}

// makeWSHost returns WS-RPC Server host, given enabled/disabled flag
func (c *nodeConstructor) makeWSHost() string {
	config := c.getConfig()

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
