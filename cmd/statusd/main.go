package main

import (
	"flag"
	"fmt"
	"log"
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
	cliEnabled     = flag.Bool("cli", false, "Enable debugging CLI server")
	cliPort        = flag.String("cliport", debug.CLIPort, "CLI server's listening port")
	logLevel       = flag.String("log", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logFile        = flag.String("logfile", "", "Path to the log file")
	version        = flag.Bool("version", false, "Print version")
	les            = flag.Bool("les", false, "LES protocol")

	listenAddr = flag.String("listenaddr", ":30303", "IP address and port of this node (e.g. 127.0.0.1:30303)")
	standalone = flag.Bool("standalone", true, "Don't actively connect to peers, wait for incoming connections")

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
	if *cliEnabled {
		err := startDebug(backend)
		if err != nil {
			log.Fatalf("Starting debugging CLI server failed: %v", err)
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
	_, err := debug.New(statusAPI, *cliPort)
	return err
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

	nodeConfig.RPCEnabled = *httpEnabled
	nodeConfig.WhisperConfig.Enabled = *whisperEnabled

	nodeConfig.HTTPPort = *httpPort
	nodeConfig.IPCEnabled = *ipcEnabled

	nodeConfig.LightEthConfig.Enabled = *les
	nodeConfig.SwarmConfig.Enabled = *swarmEnabled

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
	fmt.Fprintf(os.Stderr, usage) // nolint: gas
	flag.PrintDefaults()
}
