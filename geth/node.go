package geth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/whisper/mailserver"
	"github.com/ethereum/go-ethereum/whisper/notifications"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/params"
)

const (
	// EventNodeStarted is triggered when underlying node is fully started
	EventNodeStarted = "node.started"

	// EventNodeStopped is triggered when underlying node is fully stopped
	EventNodeStopped = "node.stopped"

	// EventChainDataRemoved is triggered when node's chain data is removed
	EventChainDataRemoved = "chaindata.removed"

	// EventNodeCrashed is triggered when node crashes
	EventNodeCrashed = "node.crashed"
)

// node-related errors
var (
	ErrEthServiceRegistrationFailure     = errors.New("failed to register the Ethereum service")
	ErrWhisperServiceRegistrationFailure = errors.New("failed to register the Whisper service")
	ErrLightEthRegistrationFailure       = errors.New("failed to register the LES service")
)

// Node represents running node (serves as a wrapper around P2P node)
type Node struct {
	config     *params.NodeConfig // configuration used to create Status node
	geth       *node.Node         // reference to the running Geth node
	gethConfig *node.Config       // configuration used to create P2P node
	started    chan struct{}      // channel to wait for node to start
}

// Inited checks whether status node has been properly initialized
func (n *Node) Inited() bool {
	return n != nil && n.geth != nil
}

// GethStack returns reference to Geth stack
func (n *Node) GethStack() *node.Node {
	return n.geth
}

// MakeNode create a geth node entity
func MakeNode(config *params.NodeConfig) *Node {
	// make sure data directory exists
	if err := os.MkdirAll(filepath.Join(config.DataDir), os.ModePerm); err != nil {
		Fatalf(err)
	}
	// make sure keys directory exists
	if err := os.MkdirAll(filepath.Join(config.KeyStoreDir), os.ModePerm); err != nil {
		Fatalf(err)
	}

	// setup logging
	if _, err := params.SetupLogger(config); err != nil {
		Fatalf(err)
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

	if len(config.NodeKeyFile) > 0 {
		log.Info("Loading private key file", "file", config.NodeKeyFile)
		pk, err := crypto.LoadECDSA(config.NodeKeyFile)
		if err != nil {
			log.Info("Failed loading private key file", "file", config.NodeKeyFile, "err", err)
		}

		// override node's private key
		stackConfig.P2P.PrivateKey = pk
	}

	stack, err := node.New(stackConfig)
	if err != nil {
		Fatalf(ErrNodeMakeFailure)
	}

	// start Ethereum service
	if err := activateEthService(stack, config); err != nil {
		Fatalf(fmt.Errorf("%v: %v", ErrEthServiceRegistrationFailure, err))
	}

	// start Whisper service
	if err := activateShhService(stack, config); err != nil {
		Fatalf(fmt.Errorf("%v: %v", ErrWhisperServiceRegistrationFailure, err))
	}

	return &Node{
		geth:       stack,
		gethConfig: stackConfig,
		started:    make(chan struct{}),
		config:     config,
	}
}

// defaultEmbeddedNodeConfig returns default stack configuration for mobile client node
func defaultEmbeddedNodeConfig(config *params.NodeConfig) *node.Config {
	return &node.Config{
		DataDir:           config.DataDir,
		KeyStoreDir:       config.KeyStoreDir,
		UseLightweightKDF: true,
		Name:              config.Name,
		Version:           config.Version,
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      false,
			DiscoveryV5Addr:  ":0",
			BootstrapNodes:   makeBootstrapNodes(),
			BootstrapNodesV5: makeBootstrapNodesV5(),
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
			MaxPendingPeers:  config.MaxPendingPeers,
		},
		IPCPath:     makeIPCPath(config),
		HTTPHost:    config.HTTPHost,
		HTTPPort:    config.HTTPPort,
		HTTPCors:    []string{"*"},
		HTTPModules: strings.Split(config.APIModules, ","),
		WSHost:      makeWSHost(config),
		WSPort:      config.WSPort,
		WSOrigins:   []string{"*"},
		WSModules:   strings.Split(config.APIModules, ","),
	}
}

// updateCHT changes trusted canonical hash trie root
func updateCHT(eth *les.LightEthereum, config *params.NodeConfig) {
	// 0xabaa042dec1ee30e0e8323d010a9c7d9a09b848631acdf66f66e966903b67755
	bc := eth.BlockChain()
	if bc.Genesis().Hash() == params.MainNetGenesisHash {
		eth.WriteTrustedCht(light.TrustedCht{
			Number: 805,
			Root:   common.HexToHash("85e4286fe0a730390245c49de8476977afdae0eb5530b277f62a52b12313d50f"),
		})
		log.Info("Added trusted CHT for mainnet")
	}
	if bc.Genesis().Hash() == params.RopstenNetGenesisHash {
		root := "28bcafd5504326a34995efc36d3a9ba0b6a22f5832e8e58bacb646b54cb8911a"
		if config.DevMode {
			root = "abaa042dec1ee30e0e8323d010a9c7d9a09b848631acdf66f66e966903b67755"
		}
		eth.WriteTrustedCht(light.TrustedCht{
			Number: 226,
			Root:   common.HexToHash(root),
		})
		log.Info("Added trusted CHT for Ropsten", "CHT", root)
	}
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
	ethConf.DatabaseHandles = makeDatabaseHandles()
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		lightEth, err := les.New(ctx, &ethConf)
		if err == nil {
			updateCHT(lightEth, config)
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
		whisperService := whisper.New()

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

// makeDatabaseHandles makes sure that enough file descriptors are available to the process
// (and returns half of them for node's database to use)
func makeDatabaseHandles() int {
	// current limit
	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		Fatalf(err)
	}

	// increase limit
	limit.Cur = limit.Max
	if limit.Cur > params.DefaultFileDescriptorLimit {
		limit.Cur = params.DefaultFileDescriptorLimit
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		Fatalf(err)
	}

	// re-query limit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		Fatalf(err)
	}

	// cap limit
	if limit.Cur > params.DefaultFileDescriptorLimit {
		limit.Cur = params.DefaultFileDescriptorLimit
	}

	return int(limit.Cur) / 2
}

// makeBootstrapNodes returns default (hence bootstrap) list of peers
func makeBootstrapNodes() []*discover.Node {
	// on desktops params.TestnetBootnodes and params.MainBootnodes,
	// on mobile client we deliberately keep this list empty
	enodes := []string{}

	var bootstapNodes []*discover.Node
	for _, enode := range enodes {
		bootstapNodes = append(bootstapNodes, discover.MustParseNode(enode))
	}

	return bootstapNodes
}

// makeBootstrapNodesV5 returns default (hence bootstrap) list of peers
func makeBootstrapNodesV5() []*discv5.Node {
	// on desktops params.DiscoveryV5Bootnodes,
	// on mobile client we deliberately keep this list empty
	enodes := []string{}

	var bootstapNodes []*discv5.Node
	for _, enode := range enodes {
		bootstapNodes = append(bootstapNodes, discv5.MustParseNode(enode))
	}

	return bootstapNodes
}

// Fatalf is used to halt the execution.
// When called the function prints stack end exits.
// Failure is logged into both StdErr and StdOut.
func Fatalf(reason interface{}, args ...interface{}) {
	// decide on output stream
	w := io.MultiWriter(os.Stdout, os.Stderr)
	outf, _ := os.Stdout.Stat()
	errf, _ := os.Stderr.Stat()
	if outf != nil && errf != nil && os.SameFile(outf, errf) {
		w = os.Stderr
	}

	// find out whether error or string has been passed as a reason
	r := reflect.ValueOf(reason)
	if r.Kind() == reflect.String {
		fmt.Fprintf(w, "Fatal Failure: "+reason.(string)+"\n", args)
	} else {
		fmt.Fprintf(w, "Fatal Failure: %v\n", reason.(error))
	}

	debug.PrintStack()

	os.Exit(1)
}

// HaltOnPanic recovers from panic, logs issue, sends upward notification, and exits
func HaltOnPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%v: %v", ErrNodeRunFailure, r)

		// send signal up to native app
		SendSignal(SignalEnvelope{
			Type: EventNodeCrashed,
			Event: NodeCrashEvent{
				Error: err.Error(),
			},
		})

		Fatalf(err) // os.exit(1) is called internally
	}
}
