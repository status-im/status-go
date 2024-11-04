package main

import (
	"context"
	"database/sql"
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

	"github.com/google/uuid"
	"github.com/okzk/sdnotify"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ethereum/go-ethereum/log"
	gethmetrics "github.com/ethereum/go-ethereum/metrics"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/cmd/status-backend/server"
	"github.com/status-im/status-go/common/dbsetup"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/metrics"
	nodemetrics "github.com/status-im/status-go/metrics/node"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/profiling"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/walletdatabase"
)

const (
	serverClientName = "Statusd"
)

var (
	configFiles                    configFlags
	logLevel                       = flag.String("log", "", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logWithoutColors               = flag.Bool("log-without-color", false, "Disables log colors")
	ipcEnabled                     = flag.Bool("ipc", false, "Enable IPC RPC endpoint")
	ipcFile                        = flag.String("ipcfile", "", "Set IPC file path")
	pprofEnabled                   = flag.Bool("pprof", false, "Enable runtime profiling via pprof")
	pprofPort                      = flag.Int("pprof-port", 52525, "Port for runtime profiling via pprof")
	communityArchiveSupportEnabled = flag.Bool("community-archives", false, "Enable community history archive support")
	torrentClientPort              = flag.Int("torrent-client-port", 9025, "Port for BitTorrent protocol connections")
	version                        = flag.Bool("version", false, "Print version and dump configuration")

	dataDir    = flag.String("dir", getDefaultDataDir(), "Directory used by node to store data")
	register   = flag.Bool("register", false, "Register and make the node discoverable by other nodes")
	mailserver = flag.Bool("mailserver", false, "Enable Mail Server with default configuration")
	networkID  = flag.Int(
		"network-id",
		params.SepoliaNetworkID,
		fmt.Sprintf(
			"A network ID: %d (Mainnet), %d (Sepolia)",
			params.MainNetworkID, params.SepoliaNetworkID,
		),
	)
	fleet = flag.String(
		"fleet",
		params.FleetProd,
		fmt.Sprintf(
			"Select fleet: %s (default %s)",
			[]string{
				params.FleetProd,
				params.FleetStatusStaging,
				params.FleetStatusProd,
				params.FleetWakuSandbox,
				params.FleetWakuTest,
			},
			params.FleetProd,
		),
	)
	listenAddr = flag.String("addr", "", "address to bind listener to")
	serverAddr = flag.String("server", "", "Address `host:port` for HTTP API server of statusd")

	// don't change the name of this flag, https://github.com/ethereum/go-ethereum/blob/master/metrics/metrics.go#L41
	metricsEnabled = flag.Bool("metrics", false, "Expose ethereum metrics with debug_metrics jsonrpc call")
	metricsPort    = flag.Int("metrics-port", 9305, "Port for the Prometheus /metrics endpoint")
	seedPhrase     = flag.String("seed-phrase", "", "Seed phrase for account creation")
	password       = flag.String("password", "", "Password for account")
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/cmd/statusd")

func init() {
	flag.Var(&configFiles, "c", "JSON configuration file(s). Multiple configuration files can be specified, and will be merged in occurrence order")
}

// nolint:gocyclo
func main() {
	if err := logutils.OverrideRootLoggerWithConfig(logutils.LogSettings{
		Enabled:   true,
		Level:     "ERROR",
		Colorized: terminal.IsTerminal(int(os.Stdin.Fd())),
	}); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}

	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() > 0 {
		printUsage()
		logger.Error("Extra args in command line: %v", flag.Args())
		os.Exit(1)
	}

	if *seedPhrase != "" && *password == "" {
		printUsage()
		logger.Error("password is required when seed phrase is provided")
		os.Exit(1)
	}

	opts := []params.Option{params.WithFleet(*fleet)}
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
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Use listenAddr if and only if explicitly provided in the arguments.
	// The default value is set in params.NewNodeConfigWithDefaultsAndFiles().
	if *listenAddr != "" {
		config.ListenAddr = *listenAddr
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

	if *communityArchiveSupportEnabled {
		config.TorrentConfig.Enabled = true
		config.TorrentConfig.Port = *torrentClientPort
	}

	// set up logging options
	setupLogging(config)

	// We want statusd to be distinct from StatusIM client.
	config.Name = serverClientName

	if *version {
		printVersion(config)
		return
	}

	if serverAddr != nil && *serverAddr != "" {
		srv := server.NewServer()
		srv.Setup()
		err = srv.Listen(*serverAddr)
		if err != nil {
			logger.Error("failed to start server", "error", err)
			return
		}
		go srv.Serve()
		log.Info("server started", "address", srv.Address())
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Stop(ctx)
		}()
	}

	backend := api.NewGethStatusBackend(logutils.ZapLogger())
	if config.NodeKey == "" {
		logger.Error("node key needs to be set if running a push notification server")
		return
	}

	identity, err := crypto.HexToECDSA(config.NodeKey)
	if err != nil {
		logger.Error("node key is invalid", "error", err)
		return
	}

	// Generate installationID from public key, so it's always the same
	installationID, err := uuid.FromBytes(crypto.CompressPubkey(&identity.PublicKey)[:16])
	if err != nil {
		logger.Error("cannot create installation id", "error", err)
		return
	}

	if *seedPhrase != "" {
		// Remove data inside dir to avoid conflicts with existing data or account restoration fails
		if err := os.RemoveAll(config.DataDir); err != nil {
			logger.Error("failed to remove data dir", "error", err)
			return
		}

		if err := createDirsFromConfig(config); err != nil {
			logger.Error("failed to create directories", "error", err)
			return
		}

		request := requests.RestoreAccount{
			Mnemonic:    *seedPhrase,
			FetchBackup: false,
			CreateAccount: requests.CreateAccount{
				DisplayName:        "Account1",
				DeviceName:         "StatusIM",
				Password:           *password,
				CustomizationColor: "0x000000",
				RootDataDir:        config.DataDir,
				APIConfig: &requests.APIConfig{
					ConnectorEnabled: config.ClusterConfig.Enabled,
					HTTPEnabled:      config.HTTPEnabled,
					HTTPHost:         config.HTTPHost,
					HTTPPort:         config.HTTPPort,
					HTTPVirtualHosts: config.HTTPVirtualHosts,
					WSEnabled:        config.WSEnabled,
					WSHost:           config.WSHost,
					WSPort:           config.WSPort,
					APIModules:       config.APIModules,
				},
				NetworkID:            &config.NetworkID,
				TestOverrideNetworks: config.Networks,
			},
		}

		api.OverrideApiConfigTest()

		_, err := backend.RestoreAccountAndLogin(&request)
		if err != nil {
			logger.Error("failed to import account", "error", err)
			return
		}

		options := []protocol.Option{
			protocol.WithDatabase(backend.StatusNode().GetAppDB()),
			protocol.WithWalletDatabase(backend.StatusNode().GetWalletDB()),
			protocol.WithTorrentConfig(&config.TorrentConfig),
			protocol.WithWalletConfig(&config.WalletConfig),
			protocol.WithAccountManager(backend.AccountManager()),
		}

		messenger, err := protocol.NewMessenger(
			config.Name,
			identity,
			gethbridge.NewNodeBridge(backend.StatusNode().GethNode(), backend.StatusNode().WakuService(), backend.StatusNode().WakuV2Service()),
			installationID.String(),
			nil,
			config.Version,
			options...,
		)

		if err != nil {
			logger.Error("failed to create messenger", "error", err)
			return
		}

		_, err = messenger.Start()
		if err != nil {
			logger.Error("failed to start messenger", "error", err)
			return
		}

		interruptCh := haltOnInterruptSignal(backend.StatusNode())
		go retrieveMessagesLoop(messenger, 300*time.Millisecond, interruptCh)

	} else {
		appDB, walletDB, err := startNode(config, backend, installationID)
		if err != nil {
			logger.Error("failed to start node", "error", err)
			return
		}

		err = sdnotify.Ready()
		if err == sdnotify.ErrSdNotifyNoSocket {
			logger.Debug("sd_notify socket not available")
		} else if err != nil {
			logger.Warn("sd_notify READY call failed", "error", err)
		} else {
			// systemd aliveness notifications, affects only Linux
			go startSystemDWatchdog()
		}

		// handle interrupt signals
		interruptCh := haltOnInterruptSignal(backend.StatusNode())

		// Start collecting metrics. Metrics can be enabled by providing `-metrics` flag
		// or setting `gethmetrics.Enabled` to true during compilation time:
		// https://github.com/status-im/go-ethereum/pull/76.
		if *metricsEnabled || gethmetrics.Enabled {
			go startCollectingNodeMetrics(interruptCh, backend.StatusNode())
			go gethmetrics.CollectProcessMetrics(3 * time.Second)
			go metrics.NewMetricsServer(*metricsPort, gethmetrics.DefaultRegistry).Listen()
		}

		// Check if profiling shall be enabled.
		if *pprofEnabled {
			profiling.NewProfiler(*pprofPort).Go()
		}

		if config.PushNotificationServerConfig.Enabled {
			options := []protocol.Option{
				protocol.WithPushNotifications(),
				protocol.WithPushNotificationServerConfig(&pushnotificationserver.Config{
					Enabled:   config.PushNotificationServerConfig.Enabled,
					Identity:  config.PushNotificationServerConfig.Identity,
					GorushURL: config.PushNotificationServerConfig.GorushURL,
				}),
				protocol.WithDatabase(appDB),
				protocol.WithWalletDatabase(walletDB),
				protocol.WithTorrentConfig(&config.TorrentConfig),
				protocol.WithWalletConfig(&config.WalletConfig),
				protocol.WithAccountManager(backend.AccountManager()),
			}

			messenger, err := protocol.NewMessenger(
				config.Name,
				identity,
				gethbridge.NewNodeBridge(backend.StatusNode().GethNode(), backend.StatusNode().WakuService(), backend.StatusNode().WakuV2Service()),
				installationID.String(),
				nil,
				config.Version,
				options...,
			)
			if err != nil {
				logger.Error("failed to create messenger", "error", err)
				return
			}

			err = messenger.InitInstallations()
			if err != nil {
				logger.Error("failed to init messenger installations", "error", err)
				return
			}

			err = messenger.InitFilters()
			if err != nil {
				logger.Error("failed to init messenger filters", "error", err)
				return
			}

			// This will start the push notification server as well as
			// the config is set to Enabled
			_, err = messenger.Start()
			if err != nil {
				logger.Error("failed to start messenger", "error", err)
				return
			}
			go retrieveMessagesLoop(messenger, 300*time.Millisecond, interruptCh)
		}
	}

	gethNode := backend.StatusNode().GethNode()
	if gethNode != nil {
		// wait till node has been stopped
		gethNode.Wait()
		if err := sdnotify.Stopping(); err != nil {
			logger.Warn("sd_notify STOPPING call failed", "error", err)
		}
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

	logSettings := logutils.LogSettings{
		Enabled:         config.LogEnabled,
		MobileSystem:    config.LogMobileSystem,
		Level:           config.LogLevel,
		File:            config.LogFile,
		MaxSize:         config.LogMaxSize,
		MaxBackups:      config.LogMaxBackups,
		CompressRotated: config.LogCompressRotated,
		Colorized:       !(*logWithoutColors) && terminal.IsTerminal(int(os.Stdin.Fd())),
	}
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}
}

// loop for notifying systemd about process being alive
func startSystemDWatchdog() {
	for range time.Tick(30 * time.Second) {
		if err := sdnotify.Watchdog(); err != nil {
			logger.Warn("sd_notify WATCHDOG call failed", "error", err)
		}
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
func printVersion(config *params.NodeConfig) {
	fmt.Println(strings.Title(config.Name))
	fmt.Println("Version:", config.Version)
	fmt.Println("Network ID:", config.NetworkID)
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
	statusd -c ./default.json -log DEBUG --seed-phrase="test test test test test test test test test test test junk" --password=password # run node with configuration specified in ./default.json file, and import account with seed phrase and password

Options:
`
	fmt.Fprint(os.Stderr, usage)
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

// retrieveMessagesLoop fetches messages from a messenger so that they are processed
func retrieveMessagesLoop(messenger *protocol.Messenger, tick time.Duration, cancel <-chan struct{}) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := messenger.RetrieveAll()
			if err != nil {
				logger.Error("failed to retrieve raw messages", "err", err)
				continue
			}
		case <-cancel:
			return
		}
	}
}

func openDatabases(path string) (*sql.DB, *sql.DB, error) {
	walletDB, err := walletdatabase.InitializeDB(path+"-wallet.db", "", dbsetup.ReducedKDFIterationsNumber)
	if err != nil {
		logger.Error("failed to initialize wallet db", "error", err)
		return nil, nil, err
	}

	appDB, err := appdatabase.InitializeDB(path+".db", "", dbsetup.ReducedKDFIterationsNumber)
	if err != nil {
		logger.Error("failed to initialize app db", "error", err)
		return nil, nil, err
	}

	return appDB, walletDB, nil
}

func createDirsFromConfig(config *params.NodeConfig) error {
	// If DataDir is empty, it means we want to create an ephemeral node
	// keeping data only in memory.
	if config.DataDir != "" {
		// make sure data directory exists
		if err := os.MkdirAll(filepath.Clean(config.DataDir), os.ModePerm); err != nil {
			return fmt.Errorf("make node: make data directory: %v", err)
		}
	}

	if config.KeyStoreDir != "" {
		// make sure keys directory exists
		if err := os.MkdirAll(filepath.Clean(config.KeyStoreDir), os.ModePerm); err != nil {
			return fmt.Errorf("make node: make keys directory: %v", err)
		}
	}

	return nil
}

func startNode(config *params.NodeConfig, backend *api.GethStatusBackend, installationID uuid.UUID) (*sql.DB, *sql.DB, error) {
	err := backend.AccountManager().InitKeystore(config.KeyStoreDir)
	if err != nil {
		logger.Error("Failed to init keystore", "error", err)
		return nil, nil, err
	}

	err = createDirsFromConfig(config)
	if err != nil {
		logger.Error("failed to create directories", "error", err)
		return nil, nil, err
	}

	appDB, walletDB, err := openDatabases(config.DataDir + "/" + installationID.String())
	if err != nil {
		log.Error("failed to open databases")
		return nil, nil, err
	}

	backend.StatusNode().SetAppDB(appDB)
	backend.StatusNode().SetWalletDB(walletDB)

	err = backend.StartNode(config)
	if err != nil {
		logger.Error("Node start failed", "error", err)
		return nil, nil, err
	}

	return appDB, walletDB, nil
}
