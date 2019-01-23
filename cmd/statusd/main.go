package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	gethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	nodemetrics "github.com/status-im/status-go/metrics/node"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/profiling"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	serverClientName = "Statusd"
)

var (
	buildStamp = "N/A" // rely on linker: -ldflags -X main.buildStamp"
)

var (
	configFiles      configFlags
	logLevel         = flag.String("log", "", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logWithoutColors = flag.Bool("log-without-color", false, "Disables log colors")
	ipcEnabled       = flag.Bool("ipc", false, "Enable IPC RPC endpoint")
	ipcFile          = flag.String("ipcfile", "", "Set IPC file path")
	pprofEnabled     = flag.Bool("pprof", false, "Enable runtime profiling via pprof")
	pprofPort        = flag.Int("pprof-port", 52525, "Port for runtime profiling via pprof")
	version          = flag.Bool("version", false, "Print version and dump configuration")

	dataDir    = flag.String("dir", getDefaultDataDir(), "Directory used by node to store data")
	register   = flag.Bool("register", false, "Register and make the node discoverable by other nodes")
	mailserver = flag.Bool("mailserver", false, "Enable Mail Server with default configuration")
	networkID  = flag.Int(
		"network-id",
		params.RopstenNetworkID,
		fmt.Sprintf(
			"A network ID: %d (Mainnet), %d (Ropsten), %d (Rinkeby)",
			params.MainNetworkID, params.RopstenNetworkID, params.RinkebyNetworkID,
		),
	)

	// don't change the name of this flag, https://github.com/ethereum/go-ethereum/blob/master/metrics/metrics.go#L41
	metrics = flag.Bool("metrics", false, "Expose ethereum metrics with debug_metrics jsonrpc call")

	syncAndExit = flag.Int("sync-and-exit", -1, "Timeout in minutes for blockchain sync and exit, zero means no timeout unless sync is finished")
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/cmd/statusd")

func init() {
	flag.Var(&configFiles, "c", "JSON configuration file(s). Multiple configuration files can be specified, and will be merged in occurrence order")

	colors := terminal.IsTerminal(int(os.Stdin.Fd()))
	if err := logutils.OverrideRootLog(true, "ERROR", "", colors); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}

	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() > 0 {
		printUsage()
		logger.Error("Extra args in command line: %v", flag.Args())
		os.Exit(1)
	}
}

// nolint:gocyclo
func main() {
	opts := []params.Option{params.WithFleet(params.FleetBeta)}
	if *mailserver {
		opts = append(opts, params.WithMailserver())
	}

	config, err := params.NewNodeConfigWithDefaultsAndFiles(
		*dataDir,
		uint64(*networkID),
		opts,
		configFiles,
	)
	if err != nil {
		printUsage()
		if err != nil {
			logger.Error(err.Error())
		}
		os.Exit(1)
	}

	if *register && *mailserver {
		config.RegisterTopics = append(config.RegisterTopics, params.MailServerDiscv5Topic)
	} else if *register {
		config.RegisterTopics = append(config.RegisterTopics, params.WhisperDiscv5Topic)
	}

	// enable IPC RPC
	if *ipcEnabled {
		config.IPCEnabled = true
		config.IPCFile = *ipcFile
	}

	// set up logging options
	setupLogging(config)

	// We want statusd to be distinct from StatusIM client.
	config.Name = serverClientName

	if *version {
		printVersion(config, buildStamp)
		return
	}

	backend := api.NewStatusBackend()
	err = backend.StartNode(config)
	if err != nil {
		logger.Error("Node start failed", "error", err)
		return
	}

	// handle interrupt signals
	interruptCh := haltOnInterruptSignal(backend.StatusNode())

	// Check if profiling shall be enabled.
	if *pprofEnabled {
		profiling.NewProfiler(*pprofPort).Go()
	}

	// Start collecting metrics. Metrics can be enabled by providing `-metrics` flag
	// or setting `gethmetrics.Enabled` to true during compilation time:
	// https://github.com/status-im/go-ethereum/pull/76.
	if *metrics || gethmetrics.Enabled {
		go startCollectingNodeMetrics(interruptCh, backend.StatusNode())
		go gethmetrics.CollectProcessMetrics(3 * time.Second)
	}

	// Sync blockchain and stop.
	if *syncAndExit >= 0 {
		exitCode := syncAndStopNode(interruptCh, backend.StatusNode(), *syncAndExit)
		// Call was interrupted. Wait for graceful shutdown.
		if exitCode == -1 {
			if gethNode := backend.StatusNode().GethNode(); gethNode != nil {
				gethNode.Wait()
			}
			return
		}
		// Otherwise, exit immediately with a returned exit code.
		os.Exit(exitCode)
	}

	gethNode := backend.StatusNode().GethNode()
	if gethNode != nil {
		// wait till node has been stopped
		gethNode.Wait()
	}
}

func getDefaultDataDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".statusd")
	}
	return "./statusd-data"
}

func setupLogging(config *params.NodeConfig) {
	if *logLevel != "" {
		config.LogLevel = *logLevel
	}

	colors := !(*logWithoutColors) && terminal.IsTerminal(int(os.Stdin.Fd()))
	if err := logutils.OverrideRootLog(
		logEnabled(config),
		config.LogLevel,
		config.LogFile,
		colors,
	); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}
}

// startCollectingStats collects various stats about the node and other protocols like Whisper.
func startCollectingNodeMetrics(interruptCh <-chan struct{}, statusNode *node.StatusNode) {
	logger.Info("Starting collecting node metrics")

	gethNode := statusNode.GethNode()
	if gethNode == nil {
		logger.Error("Failed to run metrics because it could not get the node")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		// Try to subscribe and collect metrics. In case of an error, retry.
		for {
			if err := nodemetrics.SubscribeServerEvents(ctx, gethNode); err != nil {
				logger.Error("Failed to subscribe server events", "error", err)
			} else {
				// no error means that the subscription was terminated by purpose
				return
			}

			time.Sleep(time.Second)
		}
	}()

	<-interruptCh
}

func logEnabled(config *params.NodeConfig) bool {
	return config.LogLevel != "" || config.LogFile != ""
}

var (
	errStatusServiceRequiresIPC  = errors.New("to enable the StatusService on IPC, -ipc flag must be set")
	errStatusServiceRequiresHTTP = errors.New("to enable the StatusService on HTTP, -http flag must be set")
	errStatusServiceInvalidFlag  = errors.New("-status flag valid values are: ipc, http")
)

func configureStatusService(flagValue string, nodeConfig *params.NodeConfig) (*params.NodeConfig, error) {
	switch flagValue {
	case "ipc":
		if !nodeConfig.IPCEnabled {
			return nil, errStatusServiceRequiresIPC
		}
		nodeConfig.EnableStatusService = true
	case "http":
		if !nodeConfig.HTTPEnabled {
			return nil, errStatusServiceRequiresHTTP
		}
		nodeConfig.EnableStatusService = true
		nodeConfig.AddAPIModule("status")
	case "":
		nodeConfig.EnableStatusService = false
	default:
		return nil, errStatusServiceInvalidFlag
	}

	return nodeConfig, nil
}

// printVersion prints verbose output about version and config.
func printVersion(config *params.NodeConfig, buildStamp string) {
	fmt.Println(strings.Title(config.Name))
	fmt.Println("Version:", config.Version)

	if buildStamp != "" {
		fmt.Println("Build Stamp:", buildStamp)
	}

	fmt.Println("Network Id:", config.NetworkID)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())

	fmt.Println("Loaded Config: ", config)
}

func printUsage() {
	usage := `
Usage: statusd [options]
Examples:
  statusd                                        # run regular Whisper node that joins Status network
  statusd -c ./default.json                      # run node with configuration specified in ./default.json file
  statusd -c ./default.json -c ./standalone.json # run node with configuration specified in ./default.json file, after merging ./standalone.json file
  statusd -c ./default.json -metrics             # run node with configuration specified in ./default.json file, and expose ethereum metrics with debug_metrics jsonrpc call

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
