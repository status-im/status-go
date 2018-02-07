package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/status-im/status-go/cmd/statusd/debug"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/metrics"
	nodemetrics "github.com/status-im/status-go/metrics/node"
)

var (
	gitCommit  = "N/A" // rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "N/A" // rely on linker: -ldflags -X main.buildStamp"
)

var (
	prodMode       = flag.Bool("production", false, "Whether production settings should be loaded")
	nodeKeyFile    = flag.String("nodekey", "", "P2P node key file (private key)")
	dataDir        = flag.String("datadir", params.DataDir, "Data directory for the databases and keystore")
	networkID      = flag.Int("networkid", params.RopstenNetworkID, "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby, 777=StatusChain)")
	lesEnabled     = flag.Bool("les", false, "LES protocol enabled (default is disabled)")
	whisperEnabled = flag.Bool("shh", false, "Whisper protocol enabled (default is disabled)")
	swarmEnabled   = flag.Bool("swarm", false, "Swarm protocol enabled")
	maxPeers       = flag.Int("maxpeers", 25, "maximum number of p2p peers (including all protocols)")
	httpEnabled    = flag.Bool("http", false, "HTTP RPC endpoint enabled (default: false)")
	httpPort       = flag.Int("httpport", params.HTTPPort, "HTTP RPC server's listening port")
	ipcEnabled     = flag.Bool("ipc", false, "IPC RPC endpoint enabled")
	cliEnabled     = flag.Bool("cli", false, "Enable debugging CLI server")
	cliPort        = flag.String("cliport", debug.CLIPort, "CLI server's listening port")
	logLevel       = flag.String("log", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logFile        = flag.String("logfile", "", "Path to the log file")
	version        = flag.Bool("version", false, "Print version")

	listenAddr = flag.String("listenaddr", ":30303", "IP address and port of this node (e.g. 127.0.0.1:30303)")
	standalone = flag.Bool("standalone", true, "Don't actively connect to peers, wait for incoming connections")
	bootnodes  = flag.String("bootnodes", "", "A list of bootnodes separated by comma")
	discovery  = flag.Bool("discovery", false, "Enable discovery protocol")

	// stats
	statsEnabled = flag.Bool("stats", false, "Expose node stats via /debug/vars expvar endpoint or Prometheus (log by default)")
	statsAddr    = flag.String("stats.addr", "0.0.0.0:8080", "HTTP address with /debug/vars endpoint")

	// don't change the name of this flag, https://github.com/ethereum/go-ethereum/blob/master/metrics/metrics.go#L41
	_ = flag.Bool("metrics", false, "Expose ethereum metrics with debug_metrics jsonrpc call.")
	// shh stuff
	identityFile = flag.String("shh.identityfile", "", "Protocol identity file (private key used for asymmetric encryption)")
	passwordFile = flag.String("shh.passwordfile", "", "Password file (password is used for symmetric encryption)")
	minPow       = flag.Float64("shh.pow", params.WhisperMinimumPoW, "PoW for messages to be added to queue, in float format")
	ttl          = flag.Int("shh.ttl", params.WhisperTTL, "Time to live for messages, in seconds")

	// MailServer
	enableMailServer = flag.Bool("shh.mailserver", false, "Delivers expired messages on demand")

	// Push Notification
	enablePN     = flag.Bool("shh.notify", false, "Node is capable of sending Push Notifications")
	firebaseAuth = flag.String("shh.firebaseauth", "", "FCM Authorization Key used for sending Push Notifications")

	syncAndExit = flag.Int("sync-and-exit", -1, "Timeout in minutes for blockchain sync and exit, zero means no timeout unless sync is finished")
)

func main() {
	flag.Usage = printUsage
	flag.Parse()

	config, err := makeNodeConfig()
	if err != nil {
		log.Fatalf("Making config failed: %v", err)
		return
	}

	if *version {
		printVersion(config, gitCommit, buildStamp)
		return
	}

	backend := api.NewStatusBackend()
	started, err := backend.StartNode(config)
	if err != nil {
		log.Fatalf("Node start failed: %v", err)
		return
	}

	// handle interrupt signals
	interruptCh := haltOnInterruptSignal(backend.NodeManager())

	// wait till node is started
	<-started

	// Check if debugging CLI connection shall be enabled.
	if *cliEnabled {
		err := startDebug(backend)
		if err != nil {
			log.Fatalf("Starting debugging CLI server failed: %v", err)
			return
		}
	}

	// Run stats server.
	if *statsEnabled {
		go startCollectingStats(interruptCh, backend.NodeManager())
	}

	// Sync blockchain and stop.
	if *syncAndExit >= 0 {
		exitCode := syncAndStopNode(interruptCh, backend.NodeManager(), *syncAndExit)
		// Call was interrupted. Wait for graceful shutdown.
		if exitCode == -1 {
			if node, err := backend.NodeManager().Node(); err == nil && node != nil {
				node.Wait()
			}
			return
		}
		// Otherwise, exit immediately with a returned exit code.
		os.Exit(exitCode)
	}

	node, err := backend.NodeManager().Node()
	if err != nil {
		log.Fatalf("Getting node failed: %v", err)
		return
	}

	// wait till node has been stopped
	node.Wait()
}

// startDebug starts the debugging API server.
func startDebug(backend *api.StatusBackend) error {
	statusAPI := api.NewStatusAPIWithBackend(backend)
	_, err := debug.New(statusAPI, *cliPort)
	return err
}

// startCollectingStats collects various stats about the node and other protocols like Whisper.
func startCollectingStats(interruptCh <-chan struct{}, nodeManager common.NodeManager) {
	log.Printf("Starting stats on %v", *statsAddr)

	node, err := nodeManager.Node()
	if err != nil {
		log.Printf("Failed to run metrics because could not get node: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := nodemetrics.SubscribeServerEvents(ctx, node); err != nil {
			log.Printf("Failed to subscribe server events: %v", err)
		}
	}()

	server := metrics.NewMetricsServer(*statsAddr)
	go func() {
		err := server.ListenAndServe()
		switch err {
		case http.ErrServerClosed:
		default:
			log.Printf("Metrics server failed: %v", err)
		}
	}()

	<-interruptCh

	if err := server.Shutdown(context.TODO()); err != nil {
		log.Printf("Failed to shutdown metrics server: %v", err)
	}
}

// syncAndStopNode tries to sync the blockchain and stop the node.
// It returns an exit code (`0` if successful or `1` in case of error)
// that can be used in `os.Exit` to exit immediately when the function returns.
// The special exit code `-1` is used if execution was interrupted.
func syncAndStopNode(interruptCh <-chan struct{}, nodeManager common.NodeManager, timeout int) (exitCode int) {
	log.Printf("syncAndStopNode: node will synchronize the chain and exit (timeout %d mins)", timeout)

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if timeout == 0 {
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Minute)
		defer cancel()
	}

	doneSync := make(chan struct{})
	errSync := make(chan error)
	go func() {
		if err := nodeManager.EnsureSync(ctx); err != nil {
			errSync <- err
		}

		close(doneSync)
	}()

	select {
	case err := <-errSync:
		fmt.Printf("syncAndStopNode: failed to sync the chain: %v", err)
		exitCode = 1
	case <-doneSync:
	case <-interruptCh:
		// cancel context and return immediately if interrupted
		// `-1` is used as a special exit code to denote interruption
		return -1
	}

	done, err := nodeManager.StopNode()
	if err != nil {
		log.Printf("syncAndStopNode: failed to stop the node: %v", err)
		return 1
	}
	<-done

	return
}

// makeNodeConfig parses incoming CLI options and returns node configuration object
func makeNodeConfig() (*params.NodeConfig, error) {
	devMode := !*prodMode
	nodeConfig, err := params.NewNodeConfig(*dataDir, uint64(*networkID), devMode)
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

	nodeConfig.RPCEnabled = *httpEnabled
	nodeConfig.WhisperConfig.Enabled = *whisperEnabled
	nodeConfig.MaxPeers = *maxPeers

	nodeConfig.HTTPPort = *httpPort
	nodeConfig.IPCEnabled = *ipcEnabled

	nodeConfig.LightEthConfig.Enabled = *lesEnabled
	nodeConfig.SwarmConfig.Enabled = *swarmEnabled

	if *standalone {
		nodeConfig.BootClusterConfig.Enabled = false
		nodeConfig.BootClusterConfig.BootNodes = nil
	}

	nodeConfig.Discovery = *discovery

	// Even if standalone is true and discovery is disabled,
	// it's possible to use bootnodes in NodeManager.PopulateStaticPeers().
	// TODO(adam): research if we need NodeManager.PopulateStaticPeers() at all.
	if *bootnodes != "" {
		nodeConfig.BootClusterConfig.BootNodes = strings.Split(*bootnodes, ",")
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
	fmt.Fprint(os.Stderr, usage) // nolint: gas
	flag.PrintDefaults()
}

// haltOnInterruptSignal catches interrupt signal (SIGINT) and
// stops the node. It times out after 5 seconds
// if the node can not be stopped.
func haltOnInterruptSignal(nodeManager common.NodeManager) <-chan struct{} {
	interruptCh := make(chan struct{})

	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)
		defer signal.Stop(signalCh)
		<-signalCh

		close(interruptCh)

		log.Println("Got interrupt, shutting down...")

		nodeStopped, err := nodeManager.StopNode()
		if err != nil {
			log.Printf("Failed to stop node: %v", err)
			os.Exit(1)
		}

		select {
		case <-nodeStopped:
		case <-time.After(time.Second * 5):
			log.Printf("Stopping node timed out")
			os.Exit(1)
		}
	}()

	return interruptCh
}
