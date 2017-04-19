package geth

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
	gethparams "github.com/ethereum/go-ethereum/params"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/params"
)

const (
	EventNodeStarted = "node.started"
	EventNodeCrashed = "node.crashed"
)

// node-related errors
var (
	ErrEthServiceRegistrationFailure = errors.New("failed to register the Ethereum service")
	ErrSshServiceRegistrationFailure = errors.New("failed to register the Whisper service")
	ErrLightEthRegistrationFailure   = errors.New("failed to register the LES service")
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

	// setup logging
	glog.CopyStandardLogTo("INFO")
	glog.SetToStderr(true)
	if _, err := params.SetupLogger(config); err != nil {
		Fatalf(err)
	}

	// configure required node (should you need to update node's config, e.g. add bootstrap nodes, see node.Config)
	stackConfig := &node.Config{
		DataDir:           config.DataDir,
		UseLightweightKDF: true,
		Name:              config.Name,
		Version:           config.Version,
		NoDiscovery:       true,
		DiscoveryV5:       false,
		DiscoveryV5Addr:   ":0",
		BootstrapNodes:    makeBootstrapNodes(),
		BootstrapNodesV5:  makeBootstrapNodesV5(),
		ListenAddr:        ":0",
		NAT:               nat.Any(),
		MaxPeers:          config.MaxPeers,
		MaxPendingPeers:   config.MaxPendingPeers,
		IPCPath:           makeIPCPath(config),
		HTTPHost:          config.HTTPHost,
		HTTPPort:          config.HTTPPort,
		HTTPCors:          "*",
		HTTPModules:       strings.Split(config.APIModules, ","),
		WSHost:            makeWSHost(config),
		WSPort:            config.WSPort,
		WSOrigins:         "*",
		WSModules:         strings.Split(config.APIModules, ","),
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
		Fatalf(fmt.Errorf("%v: %v", ErrSshServiceRegistrationFailure, err))
	}

	return &Node{
		geth:       stack,
		gethConfig: stackConfig,
		started:    make(chan struct{}),
		config:     config,
	}
}

// activateEthService configures and registers the eth.Ethereum service with a given node.
func activateEthService(stack *node.Node, config *params.NodeConfig) error {
	if !config.LightEthConfig.Enabled {
		glog.V(logger.Info).Infoln("LES protocol is disabled")
		return nil
	}

	ethConf := &eth.Config{
		Etherbase:               common.Address{},
		ChainConfig:             makeChainConfig(config),
		FastSync:                false,
		LightMode:               true,
		MaxPeers:                config.MaxPeers,
		DatabaseCache:           config.LightEthConfig.DatabaseCache,
		DatabaseHandles:         makeDatabaseHandles(),
		NetworkId:               config.NetworkId,
		Genesis:                 config.LightEthConfig.Genesis,
		MinerThreads:            runtime.NumCPU(),
		GasPrice:                params.GasPrice,
		GpoMinGasPrice:          params.GpoMinGasPrice,
		GpoMaxGasPrice:          params.GpoMaxGasPrice,
		GpoFullBlockRatio:       params.GpoFullBlockRatio,
		GpobaseStepDown:         params.GpobaseStepDown,
		GpobaseStepUp:           params.GpobaseStepUp,
		GpobaseCorrectionFactor: params.GpobaseCorrectionFactor,
		SolcPath:                "solc",
		AutoDAG:                 false,
	}

	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return les.New(ctx, ethConf)
	}); err != nil {
		return fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
	}

	return nil
}

// activateShhService configures Whisper and adds it to the given node.
func activateShhService(stack *node.Node, config *params.NodeConfig) error {
	if !config.WhisperConfig.Enabled {
		glog.V(logger.Info).Infoln("SHH protocol is disabled")
		return nil
	}
	serviceConstructor := func(*node.ServiceContext) (node.Service, error) {
		return whisper.New(), nil
	}
	if err := stack.Register(serviceConstructor); err != nil {
		return err
	}

	return nil
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

// makeChainConfig reads the chain configuration from the database in the datadir.
func makeChainConfig(config *params.NodeConfig) *gethparams.ChainConfig {
	chainConfig := new(gethparams.ChainConfig)

	// Homestead fork
	chainConfig.HomesteadBlock = config.HomesteadBlock

	// DAO fork
	chainConfig.DAOForkBlock = config.DAOForkBlock
	chainConfig.DAOForkSupport = config.DAOForkSupport

	// DoS reprice fork
	chainConfig.EIP150Block = config.EIP150Block
	chainConfig.EIP150Hash = config.EIP150Hash

	// DoS state cleanup fork
	chainConfig.EIP155Block = config.EIP155Block
	chainConfig.EIP158Block = config.EIP158Block
	chainConfig.ChainId = config.ChainId

	return chainConfig
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
		err := fmt.Errorf("%v: %v", ErrNodeStartFailure, r)

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
