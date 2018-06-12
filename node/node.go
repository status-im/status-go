package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

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
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/mailserver"
	shhmetrics "github.com/status-im/status-go/metrics/whisper"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/status"
	"github.com/status-im/status-go/timesource"
	"github.com/syndtr/goleveldb/leveldb"
)

// Errors related to node and services creation.
var (
	ErrNodeMakeFailure                    = errors.New("error creating p2p node")
	ErrWhisperServiceRegistrationFailure  = errors.New("failed to register the Whisper service")
	ErrLightEthRegistrationFailure        = errors.New("failed to register the LES service")
	ErrPersonalServiceRegistrationFailure = errors.New("failed to register the personal api service")
	ErrStatusServiceRegistrationFailure   = errors.New("failed to register the Status service")
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/node")

// MakeNode create a geth node entity
func MakeNode(config *params.NodeConfig, db *leveldb.DB) (*node.Node, error) {
	// If DataDir is empty, it means we want to create an ephemeral node
	// keeping data only in memory.
	if config.DataDir != "" {
		// make sure data directory exists
		if err := os.MkdirAll(filepath.Clean(config.DataDir), os.ModePerm); err != nil {
			return nil, fmt.Errorf("make node: make data directory: %v", err)
		}

		// make sure keys directory exists
		if err := os.MkdirAll(filepath.Clean(config.KeyStoreDir), os.ModePerm); err != nil {
			return nil, fmt.Errorf("make node: make keys directory: %v", err)
		}
	}

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

	// start Ethereum service if we are not expected to use an upstream server
	if !config.UpstreamConfig.Enabled {
		if err := activateLightEthService(stack, config); err != nil {
			return nil, fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
		}
	} else {
		// `personal_sign` and `personal_ecRecover` methods are important to
		// keep DApps working.
		// Usually, they are provided by an ETH or a LES service, but when using
		// upstream, we don't start any of these, so we need to start our own
		// implementation.
		if err := activatePersonalService(stack, config); err != nil {
			return nil, fmt.Errorf("%v: %v", ErrPersonalServiceRegistrationFailure, err)
		}
	}

	// start Whisper service.
	if err := activateShhService(stack, config, db); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrWhisperServiceRegistrationFailure, err)
	}

	// start status service.
	if err := activateStatusService(stack, config); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrStatusServiceRegistrationFailure, err)
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
			NoDiscovery:     true, // we always use only v5 server
			DiscoveryV5:     !config.NoDiscovery,
			ListenAddr:      config.ListenAddr,
			NAT:             nat.Any(),
			MaxPeers:        config.MaxPeers,
			MaxPendingPeers: config.MaxPendingPeers,
		},
		IPCPath:          makeIPCPath(config),
		HTTPCors:         nil,
		HTTPModules:      config.FormatAPIModules(),
		HTTPVirtualHosts: []string{"localhost"},
	}

	if config.RPCEnabled {
		nc.HTTPHost = config.HTTPHost
		nc.HTTPPort = config.HTTPPort
	}

	if config.ClusterConfig != nil && config.ClusterConfig.Enabled {
		nc.P2P.BootstrapNodesV5 = parseNodesV5(config.ClusterConfig.BootNodes)
		nc.P2P.StaticNodes = parseNodes(config.ClusterConfig.StaticNodes)
	}
	return nc
}

// activateLightEthService configures and registers the eth.Ethereum service with a given node.
func activateLightEthService(stack *node.Node, config *params.NodeConfig) error {
	if config.LightEthConfig == nil || !config.LightEthConfig.Enabled {
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
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return les.New(ctx, &ethConf)
	})
}

func activatePersonalService(stack *node.Node, config *params.NodeConfig) error {
	return stack.Register(func(*node.ServiceContext) (node.Service, error) {
		svc := personal.New(stack.AccountManager())
		return svc, nil
	})
}

func activateStatusService(stack *node.Node, config *params.NodeConfig) error {
	if !config.StatusServiceEnabled {
		logger.Info("Status service api is disabled")
		return nil
	}

	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		var whisper *whisper.Whisper
		if err := ctx.Service(&whisper); err != nil {
			return nil, err
		}
		svc := status.New(whisper)
		return svc, nil
	})
}

// activateShhService configures Whisper and adds it to the given node.
func activateShhService(stack *node.Node, config *params.NodeConfig, db *leveldb.DB) (err error) {
	if config.WhisperConfig == nil || !config.WhisperConfig.Enabled {
		logger.Info("SHH protocol is disabled")
		return nil
	}
	if config.WhisperConfig.EnableNTPSync {
		if err = stack.Register(func(*node.ServiceContext) (node.Service, error) {
			return timesource.Default(), nil
		}); err != nil {
			return
		}
	}

	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		whisperServiceConfig := &whisper.Config{
			MaxMessageSize:     whisper.DefaultMaxMessageSize,
			MinimumAcceptedPOW: 0.001,
			TimeSource:         time.Now,
		}

		if config.WhisperConfig.EnableNTPSync {
			if whisperServiceConfig.TimeSource, err = whisperTimeSource(ctx); err != nil {
				return nil, err
			}
		}

		whisperService := whisper.New(whisperServiceConfig)

		// enable metrics
		whisperService.RegisterEnvelopeTracer(&shhmetrics.EnvelopeTracer{})

		// enable mail service
		if config.WhisperConfig.EnableMailServer {
			if config.WhisperConfig.Password == "" {
				if err := config.WhisperConfig.ReadPasswordFile(); err != nil {
					return nil, err
				}
			}

			logger.Info("Register MailServer")

			var mailServer mailserver.WMailServer
			whisperService.RegisterServer(&mailServer)
			err := mailServer.Init(whisperService, config.WhisperConfig)
			if err != nil {
				return nil, err
			}
		}

		if config.WhisperConfig.LightClient {
			emptyBloomFilter := make([]byte, 64)
			if err := whisperService.SetBloomFilter(emptyBloomFilter); err != nil {
				return nil, err
			}
		}

		return whisperService, nil
	})
	if err != nil {
		return
	}

	// TODO(dshulyak) add a config option to enable it by default, but disable if app is started from statusd
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		var whisper *whisper.Whisper
		if err := ctx.Service(&whisper); err != nil {
			return nil, err
		}

		svc := shhext.New(whisper, shhext.EnvelopeSignalHandler{}, db)
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

// parseNodes creates list of discover.Node out of enode strings.
func parseNodes(enodes []string) []*discover.Node {
	var nodes []*discover.Node
	for _, enode := range enodes {
		parsedPeer, err := discover.ParseNode(enode)
		if err == nil {
			nodes = append(nodes, parsedPeer)
		} else {
			logger.Error("Failed to parse enode", "enode", enode, "err", err)
		}

	}
	return nodes
}

// parseNodesV5 creates list of discv5.Node out of enode strings.
func parseNodesV5(enodes []string) []*discv5.Node {
	var nodes []*discv5.Node
	for _, enode := range enodes {
		parsedPeer, err := discv5.ParseNode(enode)

		if err == nil {
			nodes = append(nodes, parsedPeer)
		} else {
			logger.Error("Failed to parse enode", "enode", enode, "err", err)
		}
	}
	return nodes
}

// whisperTimeSource get timeSource to be used by whisper
func whisperTimeSource(ctx *node.ServiceContext) (func() time.Time, error) {
	var timeSource *timesource.NTPTimeSource
	if err := ctx.Service(&timeSource); err != nil {
		return nil, err
	}
	return timeSource.Now, nil
}
