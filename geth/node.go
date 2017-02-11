package geth

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv2"
)

const (
	ClientIdentifier = "StatusIM" // Client identifier to advertise over the network
	VersionMajor     = 1          // Major version component of the current release
	VersionMinor     = 2          // Minor version component of the current release
	VersionPatch     = 0          // Patch version component of the current release
	VersionMeta      = "unstable" // Version metadata to append to the version string
	IPCFile          = "geth.ipc" // Filename of exposed IPC-RPC Server
	HTTPPort         = 8545       // HTTP-RPC port (replaced in unit tests)
	WSPort           = 8546       // WS-RPC port (replaced in unit tests)
	NetworkPort      = 30303
	MaxPeers         = 25
	MaxLightPeers    = 20
	MaxPendingPeers  = 0
	DefaultGas       = 180000

	ProcessFileDescriptorLimit = uint64(2048)
	DatabaseCacheSize          = 128 // Megabytes of memory allocated to internal caching (min 16MB / database forced)

	EventNodeStarted = "node.started"
	EventNodeCrashed = "node.crashed"
)

// Gas price settings
var (
	GasPrice                = new(big.Int).Mul(big.NewInt(20), common.Shannon)  // Minimal gas price to accept for mining a transactions
	GpoMinGasPrice          = new(big.Int).Mul(big.NewInt(20), common.Shannon)  // Minimum suggested gas price
	GpoMaxGasPrice          = new(big.Int).Mul(big.NewInt(500), common.Shannon) // Maximum suggested gas price
	GpoFullBlockRatio       = 80                                                // Full block threshold for gas price calculation (%)
	GpobaseStepDown         = 10                                                // Suggested gas price base step down ratio (1/1000)
	GpobaseStepUp           = 100                                               // Suggested gas price base step up ratio (1/1000)
	GpobaseCorrectionFactor = 110                                               // Suggested gas price base correction factor (%)
)

// default node configuration options
var (
	UseTestnetFlag = "true" // to be overridden via -ldflags '-X geth.UseTestnetFlag'
	UseTestnet     = false
)

func init() {
	if UseTestnetFlag == "true" { // set at compile time, here we make sure to set corresponding boolean flag
		UseTestnet = true
	}
}

// node-related errors
var (
	ErrEthServiceRegistrationFailure = errors.New("failed to register the Ethereum service")
	ErrSshServiceRegistrationFailure = errors.New("failed to register the Whisper service")
	ErrLightEthRegistrationFailure   = errors.New("failed to register the LES service")
)

// NodeConfig stores configuration options for a node
type NodeConfig struct {
	DataDir    string // base data directory
	IPCEnabled bool   // whether IPC-RPC Server is enabled or not
	HTTPPort   int    // HTTP-RPC Server port
	WSPort     int    // WS-RPC Server port
	WSEnabled  bool   // whether WS-RPC Server is enabled or not
	TLSEnabled bool   // whether TLS support should be enabled on node or not
}

// Node represents running node (serves as a wrapper around P2P node)
type Node struct {
	config     *NodeConfig   // configuration used to create Status node
	geth       *node.Node    // reference to the running Geth node
	gethConfig *node.Config  // configuration used to create P2P node
	started    chan struct{} // channel to wait for node to start
}

// Inited checks whether status node has been properly initialized
func (n *Node) Inited() bool {
	return n != nil && n.geth != nil
}

// MakeNode create a geth node entity
func MakeNode(config *NodeConfig) *Node {
	glog.CopyStandardLogTo("INFO")
	glog.SetToStderr(true)

	dataDir := config.DataDir
	if UseTestnet {
		dataDir = filepath.Join(config.DataDir, "testnet")
	}

	// exposed RPC APIs
	exposedAPIs := "db,eth,net,web3,shh,personal,admin" // TODO remove "admin" on main net

	// configure required node (should you need to update node's config, e.g. add bootstrap nodes, see node.Config)
	stackConfig := &node.Config{
		DataDir:           dataDir,
		UseLightweightKDF: true,
		Name:              ClientIdentifier,
		Version:           fmt.Sprintf("%d.%d.%d-%s", VersionMajor, VersionMinor, VersionPatch, VersionMeta),
		NoDiscovery:       true,
		DiscoveryV5:       true,
		DiscoveryV5Addr:   fmt.Sprintf(":%d", NetworkPort+1),
		BootstrapNodes:    makeBootstrapNodes(),
		BootstrapNodesV5:  makeBootstrapNodesV5(),
		ListenAddr:        fmt.Sprintf(":%d", NetworkPort),
		MaxPeers:          MaxPeers,
		MaxPendingPeers:   MaxPendingPeers,
		IPCPath:           makeIPCPath(dataDir, config.IPCEnabled),
		HTTPHost:          node.DefaultHTTPHost,
		HTTPPort:          config.HTTPPort,
		HTTPCors:          "*",
		HTTPModules:       strings.Split(exposedAPIs, ","),
		WSHost:            makeWSHost(config.WSEnabled),
		WSPort:            config.WSPort,
		WSOrigins:         "*",
		WSModules:         strings.Split(exposedAPIs, ","),
	}

	stack, err := node.New(stackConfig)
	if err != nil {
		Fatalf(ErrNodeMakeFailure)
	}

	// start Ethereum service
	if err := activateEthService(stack, makeDefaultExtra()); err != nil {
		Fatalf(fmt.Errorf("%v: %v", ErrEthServiceRegistrationFailure, err))
	}

	// start Whisper service
	if err := activateShhService(stack); err != nil {
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
func activateEthService(stack *node.Node, extra []byte) error {
	ethConf := &eth.Config{
		Etherbase:               common.Address{},
		ChainConfig:             makeChainConfig(stack),
		FastSync:                false,
		LightMode:               true,
		LightServ:               60,
		LightPeers:              MaxLightPeers,
		MaxPeers:                MaxPeers,
		DatabaseCache:           DatabaseCacheSize,
		DatabaseHandles:         makeDatabaseHandles(),
		NetworkId:               1, // Olympic
		MinerThreads:            runtime.NumCPU(),
		GasPrice:                GasPrice,
		GpoMinGasPrice:          GpoMinGasPrice,
		GpoMaxGasPrice:          GpoMaxGasPrice,
		GpoFullBlockRatio:       GpoFullBlockRatio,
		GpobaseStepDown:         GpobaseStepDown,
		GpobaseStepUp:           GpobaseStepUp,
		GpobaseCorrectionFactor: GpobaseCorrectionFactor,
		SolcPath:                "solc",
		AutoDAG:                 false,
	}

	if UseTestnet {
		ethConf.NetworkId = 3
		ethConf.Genesis = core.DefaultTestnetGenesisBlock()
	}

	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return les.New(ctx, ethConf)
	}); err != nil {
		return fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
	}

	return nil
}

// activateShhService configures Whisper and adds it to the given node.
func activateShhService(stack *node.Node) error {
	serviceConstructor := func(*node.ServiceContext) (node.Service, error) {
		return whisper.New(), nil
	}
	if err := stack.Register(serviceConstructor); err != nil {
		return err
	}

	return nil
}

// makeIPCPath returns IPC-RPC filename
func makeIPCPath(dataDir string, ipcEnabled bool) string {
	if !ipcEnabled {
		return ""
	}

	return path.Join(dataDir, IPCFile)
}

// makeWSHost returns WS-RPC Server host, given enabled/disabled flag
func makeWSHost(wsEnabled bool) string {
	if !wsEnabled {
		return ""
	}

	return node.DefaultWSHost
}

// makeChainConfig reads the chain configuration from the database in the datadir.
func makeChainConfig(stack *node.Node) *params.ChainConfig {
	config := new(params.ChainConfig)

	if UseTestnet {
		config = params.TestnetChainConfig
	} else {
		// Homestead fork
		config.HomesteadBlock = params.MainNetHomesteadBlock
		// DAO fork
		config.DAOForkBlock = params.MainNetDAOForkBlock
		config.DAOForkSupport = true

		// DoS reprice fork
		config.EIP150Block = params.MainNetHomesteadGasRepriceBlock
		config.EIP150Hash = params.MainNetHomesteadGasRepriceHash

		// DoS state cleanup fork
		config.EIP155Block = params.MainNetSpuriousDragon
		config.EIP158Block = params.MainNetSpuriousDragon
		config.ChainId = params.MainNetChainID
	}

	return config
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
	if limit.Cur > ProcessFileDescriptorLimit {
		limit.Cur = ProcessFileDescriptorLimit
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		Fatalf(err)
	}

	// re-query limit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		Fatalf(err)
	}

	// cap limit
	if limit.Cur > ProcessFileDescriptorLimit {
		limit.Cur = ProcessFileDescriptorLimit
	}

	return int(limit.Cur) / 2
}

func makeDefaultExtra() []byte {
	var clientInfo = struct {
		Version   uint
		Name      string
		GoVersion string
		Os        string
	}{uint(VersionMajor<<16 | VersionMinor<<8 | VersionPatch), ClientIdentifier, runtime.Version(), runtime.GOOS}
	extra, err := rlp.EncodeToBytes(clientInfo)
	if err != nil {
		glog.V(logger.Warn).Infoln("error setting canonical miner information:", err)
	}

	if uint64(len(extra)) > params.MaximumExtraDataSize {
		glog.V(logger.Warn).Infoln("error setting canonical miner information: extra exceeds", params.MaximumExtraDataSize)
		glog.V(logger.Debug).Infof("extra: %x\n", extra)
		return nil
	}

	return extra
}

// makeBootstrapNodes returns default (hence bootstrap) list of peers
func makeBootstrapNodes() []*discover.Node {
	enodes := params.MainnetBootnodes
	if UseTestnet {
		enodes = params.TestnetBootnodes
	}

	var bootstapNodes []*discover.Node
	for _, enode := range enodes {
		bootstapNodes = append(bootstapNodes, discover.MustParseNode(enode))
	}

	return bootstapNodes
}

// makeBootstrapNodesV5 returns default (hence bootstrap) list of peers
func makeBootstrapNodesV5() []*discv5.Node {
	enodes := params.DiscoveryV5Bootnodes

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
