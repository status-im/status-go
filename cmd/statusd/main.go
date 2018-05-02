package main

import (
	"context"
	"flag"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/status-im/status-go/logutils"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/cmd/statusd/debug"
	"github.com/status-im/status-go/cmd/statusd/topics"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/metrics"
	nodemetrics "github.com/status-im/status-go/metrics/node"
	"github.com/status-im/status-go/profiling"
)

var (
	gitCommit  = "N/A" // rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "N/A" // rely on linker: -ldflags -X main.buildStamp"
)

var (
	clusterConfigFile = flag.String("clusterconfig", "", "Cluster configuration file")
	nodeKeyFile       = flag.String("nodekey", "", "P2P node key file (private key)")
	dataDir           = flag.String("datadir", params.DataDir, "Data directory for the databases and keystore")
	networkID         = flag.Int("networkid", params.RopstenNetworkID, "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby, 777=StatusChain)")
	lesEnabled        = flag.Bool("les", false, "Enable LES protocol")
	whisperEnabled    = flag.Bool("shh", false, "Enable Whisper protocol")
	swarmEnabled      = flag.Bool("swarm", false, "Enable Swarm protocol")
	maxPeers          = flag.Int("maxpeers", 25, "maximum number of p2p peers (including all protocols)")
	httpEnabled       = flag.Bool("http", false, "Enable HTTP RPC endpoint")
	httpHost          = flag.String("httphost", "127.0.0.1", "HTTP RPC host of the listening socket")
	httpPort          = flag.Int("httpport", params.HTTPPort, "HTTP RPC server's listening port")
	ipcEnabled        = flag.Bool("ipc", false, "Enable IPC RPC endpoint")
	cliEnabled        = flag.Bool("cli", false, "Enable debugging CLI server")
	cliPort           = flag.String("cliport", debug.CLIPort, "CLI server's listening port")
	pprofEnabled      = flag.Bool("pprof", false, "Enable runtime profiling via pprof")
	pprofPort         = flag.Int("pprofport", 52525, "Port for runtime profiling via pprof")
	logLevel          = flag.String("log", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logFile           = flag.String("logfile", "", "Path to the log file")
	version           = flag.Bool("version", false, "Print version")

	listenAddr = flag.String("listenaddr", ":30303", "IP address and port of this node (e.g. 127.0.0.1:30303)")
	standalone = flag.Bool("standalone", true, "Don't actively connect to peers, wait for incoming connections")
	bootnodes  = flag.String("bootnodes", "", "A list of bootnodes separated by comma")
	discovery  = flag.Bool("discovery", false, "Enable discovery protocol")

	// stats
	statsEnabled = flag.Bool("stats", false, "Expose node stats via /debug/vars expvar endpoint or Prometheus")
	statsAddr    = flag.String("stats.addr", "0.0.0.0:8080", "HTTP address with /debug/vars endpoint")

	// don't change the name of this flag, https://github.com/ethereum/go-ethereum/blob/master/metrics/metrics.go#L41
	_ = flag.Bool("metrics", false, "Expose ethereum metrics with debug_metrics jsonrpc call.")
	// shh stuff
	passwordFile = flag.String("shh.passwordfile", "", "Password file (password is used for symmetric encryption)")
	minPow       = flag.Float64("shh.pow", params.WhisperMinimumPoW, "PoW for messages to be added to queue, in float format")
	ttl          = flag.Int("shh.ttl", params.WhisperTTL, "Time to live for messages, in seconds")
	lightClient  = flag.Bool("shh.lightclient", false, "Start with empty bloom filter, and don't forward messages")

	// MailServer
	enableMailServer = flag.Bool("shh.mailserver", false, "Delivers expired messages on demand")

	// Push Notification
	firebaseAuth = flag.String("shh.firebaseauth", "", "FCM Authorization Key used for sending Push Notifications")

	syncAndExit = flag.Int("sync-and-exit", -1, "Timeout in minutes for blockchain sync and exit, zero means no timeout unless sync is finished")

	// Topics that will be search and registered by discovery v5.
	searchTopics   = topics.TopicLimitsFlag{}
	registerTopics = topics.TopicFlag{}
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/cmd/statusd")

func main() {
	flag.Var(&searchTopics, "topic.search", "Topic that will be searched in discovery v5, e.g (mailserver=1,1)")
	flag.Var(&registerTopics, "topic.register", "Topic that will be registered using discovery v5.")

	flag.Usage = printUsage
	flag.Parse()

	config, err := makeNodeConfig()
	if err != nil {
		stdlog.Fatalf("Making config failed %s", err)
	}

	if *version {
		printVersion(config, gitCommit, buildStamp)
		return
	}

	if err := logutils.OverrideRootLog(config.LogEnabled, config.LogLevel, config.LogFile, true); err != nil {
		stdlog.Fatalf("Error initializing logger: %s", err)
	}

	backend := api.NewStatusBackend()
	err = backend.StartNode(config)
	if err != nil {
		logger.Error("Node start failed", "error", err)
		return
	}

	// handle interrupt signals
	interruptCh := haltOnInterruptSignal(backend.StatusNode())

	// Check if debugging CLI connection shall be enabled.
	if *cliEnabled {
		err := startDebug(backend)
		if err != nil {
			logger.Error("Starting debugging CLI server failed", "error", err)
			return
		}
	}

	// Check if profiling shall be enabled.
	if *pprofEnabled {
		profiling.NewProfiler(*pprofPort).Go()
	}

	// Run stats server.
	if *statsEnabled {
		go startCollectingStats(interruptCh, backend.StatusNode())
	}

	// Sync blockchain and stop.
	if *syncAndExit >= 0 {
		exitCode := syncAndStopNode(interruptCh, backend.StatusNode(), *syncAndExit)
		// Call was interrupted. Wait for graceful shutdown.
		if exitCode == -1 {
			if node := backend.StatusNode().GethNode(); node != nil {
				node.Wait()
			}
			return
		}
		// Otherwise, exit immediately with a returned exit code.
		os.Exit(exitCode)
	}

	node := backend.StatusNode().GethNode()
	if node != nil {
		// wait till node has been stopped
		node.Wait()
	}
}

// startDebug starts the debugging API server.
func startDebug(backend *api.StatusBackend) error {
	statusAPI := api.NewStatusAPIWithBackend(backend)
	_, err := debug.New(statusAPI, *cliPort)
	return err
}

// startCollectingStats collects various stats about the node and other protocols like Whisper.
func startCollectingStats(interruptCh <-chan struct{}, statusNode *node.StatusNode) {

	logger.Info("Starting stats", "stats", *statsAddr)

	node := statusNode.GethNode()
	if node == nil {
		logger.Error("Failed to run metrics because it could not get the node")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := nodemetrics.SubscribeServerEvents(ctx, node); err != nil {
			logger.Error("Failed to subscribe server events", "error", err)
		}
	}()

	server := metrics.NewMetricsServer(*statsAddr)
	defer func() {
		// server may be nil if `-stats` flag is used
		// but the binary is compiled without metrics enabled
		if server == nil {
			return
		}

		if err := server.Shutdown(context.TODO()); err != nil {
			logger.Error("Failed to shutdown metrics server", "error", err)
		}
	}()
	go func() {
		// server may be nil if `-stats` flag is used
		// but the binary is compiled without metrics enabled
		if server == nil {
			return
		}

		err := server.ListenAndServe()
		switch err {
		case http.ErrServerClosed:
		default:
			logger.Error("Metrics server failed", "error", err)
		}
	}()

	<-interruptCh
}

// makeNodeConfig parses incoming CLI options and returns node configuration object
func makeNodeConfig() (*params.NodeConfig, error) {
	nodeConfig, err := params.NewNodeConfig(*dataDir, *clusterConfigFile, uint64(*networkID))
	if err != nil {
		return nil, err
	}

	nodeConfig.ListenAddr = *listenAddr

	// TODO(divan): move this logic into params package
	if *nodeKeyFile != "" {
		nodeConfig.NodeKeyFile = *nodeKeyFile
	}

	if *logLevel != "" {
		nodeConfig.LogLevel = *logLevel
	}

	if *logFile != "" {
		nodeConfig.LogFile = *logFile
	}

	if *logLevel != "" || *logFile != "" {
		nodeConfig.LogEnabled = true
	}

	nodeConfig.RPCEnabled = *httpEnabled
	nodeConfig.WhisperConfig.Enabled = *whisperEnabled
	nodeConfig.MaxPeers = *maxPeers

	nodeConfig.HTTPHost = *httpHost
	nodeConfig.HTTPPort = *httpPort
	nodeConfig.IPCEnabled = *ipcEnabled

	nodeConfig.LightEthConfig.Enabled = *lesEnabled
	nodeConfig.SwarmConfig.Enabled = *swarmEnabled

	if *standalone {
		nodeConfig.ClusterConfig.Enabled = false
		nodeConfig.ClusterConfig.BootNodes = nil
	}

	nodeConfig.NoDiscovery = !(*discovery)
	nodeConfig.RequireTopics = map[discv5.Topic]params.Limits(searchTopics)
	nodeConfig.RegisterTopics = []discv5.Topic(registerTopics)

	// Even if standalone is true and discovery is disabled,
	// it's possible to use bootnodes.
	if *bootnodes != "" {
		nodeConfig.ClusterConfig.BootNodes = strings.Split(*bootnodes, ",")
	}

	if *whisperEnabled {
		return whisperConfig(nodeConfig)
	}

	// RPC configuration
	if !*httpEnabled {
		nodeConfig.HTTPHost = "" // HTTP RPC is disabled
	}

	return nodeConfig, nil
}

// printVersion prints verbose output about version and config.
func printVersion(config *params.NodeConfig, gitCommit, buildStamp string) {
	if gitCommit != "" && len(gitCommit) > 8 {
		params.Version += "-" + gitCommit[:8]
	}

	fmt.Println(strings.Title(params.ClientIdentifier))
	fmt.Println("Version:", params.Version)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	if buildStamp != "" {
		fmt.Println("Build Stamp:", buildStamp)
	}

	fmt.Println("Network Id:", config.NetworkID)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())

	config.LightEthConfig.Genesis = "SKIP"
	fmt.Println("Loaded Config: ", config)
}

func printUsage() {
	usage := `
Usage: statusd [options]
Examples:
  statusd               # run status node with defaults
  statusd -networkid 4  # run node on Rinkeby network
  statusd -datadir /dir # specify different dir for data
  statusd -ipc          # enable IPC for usage with "geth attach"
  statusd -cli          # enable connection by statusd-cli on default port

Options:
`
	fmt.Fprintf(os.Stderr, usage)
	flag.PrintDefaults()
}

// haltOnInterruptSignal catches interrupt signal (SIGINT) and
// stops the node. It times out after 5 seconds
// if the node can not be stopped.
func haltOnInterruptSignal(statusNode *node.StatusNode) <-chan struct{} {
	interruptCh := make(chan struct{})
	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)
		defer signal.Stop(signalCh)
		<-signalCh
		close(interruptCh)
		logger.Info("Got interrupt, shutting down...")
		if err := statusNode.Stop(); err != nil {
			logger.Error("Failed to stop node", "error", err)
			os.Exit(1)
		}
	}()
	return interruptCh
}
