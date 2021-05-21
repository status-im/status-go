package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/mailserver"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/services/nodebridge"
	"github.com/status-im/status-go/services/peer"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/wakuext"
	"github.com/status-im/status-go/static"
	"github.com/status-im/status-go/timesource"
	"github.com/status-im/status-go/waku"
	wakucommon "github.com/status-im/status-go/waku/common"
)

// Errors related to node and services creation.
var (
	ErrNodeMakeFailureFormat                      = "error creating p2p node: %s"
	ErrWakuServiceRegistrationFailure             = errors.New("failed to register the Waku service")
	ErrLightEthRegistrationFailure                = errors.New("failed to register the LES service")
	ErrLightEthRegistrationFailureUpstreamEnabled = errors.New("failed to register the LES service, upstream is also configured")
	ErrPersonalServiceRegistrationFailure         = errors.New("failed to register the personal api service")
	ErrStatusServiceRegistrationFailure           = errors.New("failed to register the Status service")
	ErrPeerServiceRegistrationFailure             = errors.New("failed to register the Peer service")
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/node")

// MakeNode creates a geth node entity
func MakeNode(config *params.NodeConfig, accs *accounts.Manager, db *leveldb.DB) (*node.Node, error) {
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

	stackConfig, err := newGethNodeConfig(config)
	if err != nil {
		return nil, err
	}

	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, fmt.Errorf(ErrNodeMakeFailureFormat, err.Error())
	}

	err = activateServices(stack, config, accs, db)
	if err != nil {
		return nil, err
	}
	return stack, nil
}

func activateServices(stack *node.Node, config *params.NodeConfig, accs *accounts.Manager, db *leveldb.DB) error {
	if config.EnableNTPSync {
		err := stack.Register(func(*node.ServiceContext) (node.Service, error) {
			return timesource.Default(), nil
		})
		if err != nil {
			return fmt.Errorf("failed to register NTP time source: %v", err)
		}
	}

	// start Ethereum service if we are not expected to use an upstream server
	if !config.UpstreamConfig.Enabled {
		if err := activateLightEthService(stack, accs, config); err != nil {
			return fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
		}
	} else {
		if config.LightEthConfig.Enabled {
			return ErrLightEthRegistrationFailureUpstreamEnabled
		}

		logger.Info("LES protocol is disabled")

		// `personal_sign` and `personal_ecRecover` methods are important to
		// keep DApps working.
		// Usually, they are provided by an ETH or a LES service, but when using
		// upstream, we don't start any of these, so we need to start our own
		// implementation.
		if err := activatePersonalService(stack, accs, config); err != nil {
			return fmt.Errorf("%v: %v", ErrPersonalServiceRegistrationFailure, err)
		}
	}

	if err := activateNodeServices(stack, config, db); err != nil {
		return err
	}

	return nil
}

func activateNodeServices(stack *node.Node, config *params.NodeConfig, db *leveldb.DB) error {
	// Register eth-node node bridge
	err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return &nodebridge.NodeService{Node: gethbridge.NewNodeBridge(stack)}, nil
	})
	if err != nil {
		return fmt.Errorf("failed to register NodeBridge: %v", err)
	}

	// start Waku service
	if err := activateWakuService(stack, config, db); err != nil {
		return fmt.Errorf("%v: %v", ErrWakuServiceRegistrationFailure, err)
	}

	// start peer service
	if err := activatePeerService(stack); err != nil {
		return fmt.Errorf("%v: %v", ErrPeerServiceRegistrationFailure, err)
	}
	return nil
}

// newGethNodeConfig returns default stack configuration for mobile client node
func newGethNodeConfig(config *params.NodeConfig) (*node.Config, error) {
	nc := &node.Config{
		DataDir:           config.DataDir,
		KeyStoreDir:       config.KeyStoreDir,
		UseLightweightKDF: true,
		NoUSB:             true,
		Name:              config.Name,
		Version:           config.Version,
		P2P: p2p.Config{
			NoDiscovery:     true, // we always use only v5 server
			ListenAddr:      config.ListenAddr,
			NAT:             nat.Any(),
			MaxPeers:        config.MaxPeers,
			MaxPendingPeers: config.MaxPendingPeers,
		},
		HTTPModules: config.FormatAPIModules(),
	}

	if config.IPCEnabled {
		// use well-known defaults
		if config.IPCFile == "" {
			config.IPCFile = "geth.ipc"
		}

		nc.IPCPath = config.IPCFile
	}

	if config.HTTPEnabled {
		nc.HTTPHost = config.HTTPHost
		nc.HTTPPort = config.HTTPPort
		nc.HTTPVirtualHosts = config.HTTPVirtualHosts
		nc.HTTPCors = config.HTTPCors
	}

	if config.ClusterConfig.Enabled {
		nc.P2P.BootstrapNodesV5 = parseNodesV5(config.ClusterConfig.BootNodes)
		nc.P2P.StaticNodes = parseNodes(config.ClusterConfig.StaticNodes)
	}

	if config.NodeKey != "" {
		sk, err := crypto.HexToECDSA(config.NodeKey)
		if err != nil {
			return nil, err
		}
		// override node's private key
		nc.P2P.PrivateKey = sk
	}

	return nc, nil
}

// calculateGenesis retrieves genesis value for given network
func calculateGenesis(networkID uint64) (*core.Genesis, error) {
	var genesis *core.Genesis

	switch networkID {
	case params.MainNetworkID:
		genesis = core.DefaultGenesisBlock()
	case params.RopstenNetworkID:
		genesis = core.DefaultTestnetGenesisBlock()
	case params.RinkebyNetworkID:
		genesis = core.DefaultRinkebyGenesisBlock()
	case params.GoerliNetworkID:
		genesis = core.DefaultGoerliGenesisBlock()
	case params.StatusChainNetworkID:
		var err error
		if genesis, err = defaultStatusChainGenesisBlock(); err != nil {
			return nil, err
		}
	default:
		return nil, nil
	}

	return genesis, nil
}

// defaultStatusChainGenesisBlock returns the StatusChain network genesis block.
func defaultStatusChainGenesisBlock() (*core.Genesis, error) {
	genesisJSON, err := static.ConfigStatusChainGenesisJsonBytes()
	if err != nil {
		return nil, fmt.Errorf("status-chain-genesis.json could not be loaded: %s", err)
	}

	var genesis *core.Genesis
	err = json.Unmarshal(genesisJSON, &genesis)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal status-chain-genesis.json: %s", err)
	}
	return genesis, nil
}

// activateLightEthService configures and registers the eth.Ethereum service with a given node.
func activateLightEthService(stack *node.Node, accs *accounts.Manager, config *params.NodeConfig) error {
	if !config.LightEthConfig.Enabled {
		logger.Info("LES protocol is disabled")
		return nil
	}

	genesis, err := calculateGenesis(config.NetworkID)
	if err != nil {
		return err
	}

	ethConf := eth.DefaultConfig
	ethConf.Genesis = genesis
	ethConf.SyncMode = downloader.LightSync
	ethConf.NetworkId = config.NetworkID
	ethConf.DatabaseCache = config.LightEthConfig.DatabaseCache
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		// NOTE(dshulyak) here we set our instance of the accounts manager.
		// without sharing same instance selected account won't be visible for personal_* methods.
		nctx := &node.ServiceContext{}
		*nctx = *ctx
		nctx.AccountManager = accs
		return les.New(nctx, &ethConf)
	})
}

func activatePersonalService(stack *node.Node, accs *accounts.Manager, config *params.NodeConfig) error {
	return stack.Register(func(*node.ServiceContext) (node.Service, error) {
		svc := personal.New(accs)
		return svc, nil
	})
}

func activatePeerService(stack *node.Node) error {
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		svc := peer.New()
		return svc, nil
	})
}

func registerWakuMailServer(wakuService *waku.Waku, config *params.WakuConfig) (err error) {
	var mailServer mailserver.WakuMailServer
	wakuService.RegisterMailServer(&mailServer)

	return mailServer.Init(wakuService, config)
}

// activateWakuService configures Waku and adds it to the given node.
func activateWakuService(stack *node.Node, config *params.NodeConfig, db *leveldb.DB) (err error) {
	if !config.WakuConfig.Enabled {
		logger.Info("Waku protocol is disabled")
		return nil
	}

	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return createWakuService(ctx, &config.WakuConfig, &config.ClusterConfig)
	})
	if err != nil {
		return
	}

	// Register Whisper eth-node bridge
	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		var ethnode *nodebridge.NodeService
		if err := ctx.Service(&ethnode); err != nil {
			return nil, err
		}
		w, err := ethnode.Node.GetWaku(ctx)
		if err != nil {
			return nil, err
		}
		return &nodebridge.WakuService{Waku: w}, nil
	})
	if err != nil {
		return
	}

	// TODO(dshulyak) add a config option to enable it by default, but disable if app is started from statusd
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		var ethnode *nodebridge.NodeService
		if err := ctx.Service(&ethnode); err != nil {
			return nil, err
		}
		return wakuext.New(config.ShhextConfig, ethnode.Node, ctx, ext.EnvelopeSignalHandler{}, db), nil
	})
}

func createWakuService(ctx *node.ServiceContext, wakuCfg *params.WakuConfig, clusterCfg *params.ClusterConfig) (*waku.Waku, error) {
	cfg := &waku.Config{
		MaxMessageSize:         wakucommon.DefaultMaxMessageSize,
		BloomFilterMode:        wakuCfg.BloomFilterMode,
		FullNode:               wakuCfg.FullNode,
		SoftBlacklistedPeerIDs: wakuCfg.SoftBlacklistedPeerIDs,
		MinimumAcceptedPoW:     params.WakuMinimumPoW,
		EnableConfirmations:    wakuCfg.EnableConfirmations,
	}

	if wakuCfg.MaxMessageSize > 0 {
		cfg.MaxMessageSize = wakuCfg.MaxMessageSize
	}
	if wakuCfg.MinimumPoW > 0 {
		cfg.MinimumAcceptedPoW = wakuCfg.MinimumPoW
	}

	w := waku.New(cfg, logutils.ZapLogger())

	if wakuCfg.EnableRateLimiter {
		r := wakuRateLimiter(wakuCfg, clusterCfg)
		w.RegisterRateLimiter(r)
	}

	if timesource, err := timeSource(ctx); err == nil {
		w.SetTimeSource(timesource)
	}

	// enable mail service
	if wakuCfg.EnableMailServer {
		if err := registerWakuMailServer(w, wakuCfg); err != nil {
			return nil, fmt.Errorf("failed to register WakuMailServer: %v", err)
		}
	}

	if wakuCfg.LightClient {
		emptyBloomFilter := make([]byte, 64)
		if err := w.SetBloomFilter(emptyBloomFilter); err != nil {
			return nil, err
		}
	}

	return w, nil
}

// parseNodes creates list of enode.Node out of enode strings.
func parseNodes(enodes []string) []*enode.Node {
	var nodes []*enode.Node
	for _, item := range enodes {
		parsedPeer, err := enode.ParseV4(item)
		if err == nil {
			nodes = append(nodes, parsedPeer)
		} else {
			logger.Error("Failed to parse enode", "enode", item, "err", err)
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

func parseNodesToNodeID(enodes []string) []enode.ID {
	nodeIDs := make([]enode.ID, 0, len(enodes))
	for _, node := range parseNodes(enodes) {
		nodeIDs = append(nodeIDs, node.ID())
	}
	return nodeIDs
}

// timeSource get timeSource to be used by whisper
func timeSource(ctx *node.ServiceContext) (func() time.Time, error) {
	var timeSource *timesource.NTPTimeSource
	if err := ctx.Service(&timeSource); err != nil {
		return nil, err
	}
	return timeSource.Now, nil
}

func wakuRateLimiter(wakuCfg *params.WakuConfig, clusterCfg *params.ClusterConfig) *wakucommon.PeerRateLimiter {
	enodes := append(
		parseNodes(clusterCfg.StaticNodes),
		parseNodes(clusterCfg.TrustedMailServers)...,
	)
	var (
		ips     []string
		peerIDs []enode.ID
	)
	for _, item := range enodes {
		ips = append(ips, item.IP().String())
		peerIDs = append(peerIDs, item.ID())
	}
	return wakucommon.NewPeerRateLimiter(
		&wakucommon.PeerRateLimiterConfig{
			PacketLimitPerSecIP:     wakuCfg.PacketRateLimitIP,
			PacketLimitPerSecPeerID: wakuCfg.PacketRateLimitPeerID,
			BytesLimitPerSecIP:      wakuCfg.BytesRateLimitIP,
			BytesLimitPerSecPeerID:  wakuCfg.BytesRateLimitPeerID,
			WhitelistedIPs:          ips,
			WhitelistedPeerIDs:      peerIDs,
		},
		&wakucommon.MetricsRateLimiterHandler{},
		&wakucommon.DropPeerRateLimiterHandler{
			Tolerance: wakuCfg.RateLimitTolerance,
		},
	)
}
