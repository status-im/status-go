package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strings"

	"github.com/status-im/status-go/cmd/statusd/debug"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
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
	whisperEnabled = flag.Bool("shh", false, "SHH protocol enabled")
	swarmEnabled   = flag.Bool("swarm", false, "Swarm protocol enabled")
	httpEnabled    = flag.Bool("http", false, "HTTP RPC endpoint enabled (default: false)")
	httpPort       = flag.Int("httpport", params.HTTPPort, "HTTP RPC server's listening port")
	ipcEnabled     = flag.Bool("ipc", false, "IPC RPC endpoint enabled")
	cli            = flag.Bool("cli", false, "Enable debugging CLI connection")
	pprofPort      = flag.String("pprof", "", "Enable profiling on port <port>")
	logLevel       = flag.String("log", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logFile        = flag.String("logfile", "", "Path to the log file")
	version        = flag.Bool("version", false, "Print version")
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

	// wait till node is started
	<-started

	// Check if debugging CLI connection shall be enabled.
	if *cli {
		err := startDebug(backend)
		if err != nil {
			log.Fatalf("Starting debugging CLI server failed: %v", err)
			return
		}
	}

	// Check if pprof shall be enabled.
	if *pprofPort != "" {
		err := startPprof()
		if err != nil {
			log.Fatalf("Starting of profiling failed: %v", err)
			return
		}
	}

	// wait till node has been stopped
	node, err := backend.NodeManager().Node()
	if err != nil {
		log.Fatalf("Getting node failed: %v", err)
		return
	}

	node.Wait()
}

// startDebug starts the debugging API server.
func startDebug(backend *api.StatusBackend) error {
	statusAPI := api.NewStatusAPIWithBackend(backend)
	_, err := debug.New(statusAPI)
	return err
}

// startPprof starts the PPROF endpoints on the defined port.
func startPprof() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	s := http.Server{
		Addr:    ":" + (*pprofPort),
		Handler: mux,
	}
	return s.ListenAndServe()
}

// makeNodeConfig parses incoming CLI options and returns node configuration object
func makeNodeConfig() (*params.NodeConfig, error) {
	devMode := !*prodMode
	nodeConfig, err := params.NewNodeConfig(*dataDir, uint64(*networkID), devMode)
	if err != nil {
		return nil, err
	}

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

	nodeConfig.LightEthConfig.Enabled = true
	nodeConfig.RPCEnabled = *httpEnabled
	nodeConfig.WhisperConfig.Enabled = *whisperEnabled
	nodeConfig.SwarmConfig.Enabled = *swarmEnabled

	// RPC configuration
	if !*httpEnabled {
		nodeConfig.HTTPHost = "" // HTTP RPC is disabled
	}
	nodeConfig.HTTPPort = *httpPort
	nodeConfig.IPCEnabled = *ipcEnabled

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
	fmt.Fprintln(os.Stderr, "Usage: statusd [options]")
	fmt.Fprintf(os.Stderr, `
Examples:
  statusd                      # run status node with defaults
  statusd -networkid 4         # run node on Rinkeby network
  statusd -datadir /dir        # specify different dir for data
  statusd -ipc                 # enable IPC for usage with "geth attach"
  statusd -cli localhost:12345 # enable connection by local statusd-cli on port 12345

Options:
`)
	flag.PrintDefaults()
}
