package main

import (
	"context"
	"encoding/json"
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
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ethereum/go-ethereum/log"
	gethmetrics "github.com/ethereum/go-ethereum/metrics"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/metrics"
	nodemetrics "github.com/status-im/status-go/metrics/node"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/profiling"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/identity/alias"
	waku2extn "github.com/status-im/status-go/services/wakuv2ext"
)

const (
	serverClientName = "Statusd"
)

var (
	configFiles      configFlags
	logLevel         = flag.String("log", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logWithoutColors = flag.Bool("log-without-color", false, "Disables log colors")
	seedPhrase       = flag.String("seed-phrase", "", "Seed phrase")
	version          = flag.Bool("version", false, "Print version and dump configuration")
	apiModules       = flag.String("api-modules", "wakuext,ext,waku,ens", "API modules to enable in the HTTP server")
	pprofEnabled     = flag.Bool("pprof", false, "Enable runtime profiling via pprof")
	pprofPort        = flag.Int("pprof-port", 52525, "Port for runtime profiling via pprof")
	metricsEnabled   = flag.Bool("metrics", false, "Expose ethereum metrics with debug_metrics jsonrpc call")
	metricsPort      = flag.Int("metrics-port", 9305, "Port for the Prometheus /metrics endpoint")

	dataDir   = flag.String("dir", getDefaultDataDir(), "Directory used by node to store data")
	networkID = flag.Int(
		"network-id",
		params.SepoliaNetworkID,
		fmt.Sprintf(
			"A network ID: %d (Mainnet), %d (Sepolia)",
			params.MainNetworkID, params.SepoliaNetworkID,
		),
	)
	listenAddr = flag.String("addr", "", "address to bind listener to")
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/cmd/statusd")

func init() {
	flag.Var(&configFiles, "c", "JSON configuration file(s). Multiple configuration files can be specified, and will be merged in occurrence order")
}

// nolint:gocyclo
func main() {
	colors := terminal.IsTerminal(int(os.Stdin.Fd()))
	if err := logutils.OverrideRootLog(true, "ERROR", logutils.FileOptions{}, colors); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}

	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() > 0 {
		printUsage()
		logger.Error("Extra args in command line: %v", flag.Args())
		os.Exit(1)
	}

	opts := []params.Option{}

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

	if *logLevel != "" {
		config.LogLevel = *logLevel
	}

	// set up logging options
	setupLogging(config)

	// We want statusd to be distinct from StatusIM client.
	config.Name = serverClientName

	if *version {
		printVersion(config)
		return
	}

	// Check if profiling shall be enabled.
	if *pprofEnabled {
		profiling.NewProfiler(*pprofPort).Go()
	}

	backend := api.NewGethStatusBackend(logutils.ZapLogger())
	err = ImportAccount(*seedPhrase, backend)
	if err != nil {
		logger.Error("failed import account", "err", err)
		return
	}

	// handle interrupt signals
	interruptCh := exitOnInterruptSignal(backend.StatusNode())

	// Start collecting metrics. Metrics can be enabled by providing `-metrics` flag
	// or setting `gethmetrics.Enabled` to true during compilation time:
	// https://github.com/status-im/go-ethereum/pull/76.
	if *metricsEnabled || gethmetrics.Enabled {
		go startNodeMetrics(interruptCh, backend.StatusNode())
		go gethmetrics.CollectProcessMetrics(3 * time.Second)
		go metrics.NewMetricsServer(*metricsPort, gethmetrics.DefaultRegistry).Listen()
	}

	wakuextservice := backend.StatusNode().WakuV2ExtService()
	if wakuextservice == nil {
		logger.Error("wakuext not available")
		return
	}

	wakuext := waku2extn.NewPublicAPI(wakuextservice)

	messenger := wakuext.Messenger()
	messenger.DisableStoreNodes()
	// This will start the push notification server as well as
	// the config is set to Enabled
	_, err = wakuext.StartMessenger()
	if err != nil {
		logger.Error("failed to start messenger", "error", err)
		return
	}

	retrieveMessagesLoop(messenger, 300*time.Millisecond)

}

func getDefaultDataDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".statusd")
	}
	return "./statusd-data"
}

func setupLogging(config *params.NodeConfig) {
	logSettings := logutils.LogSettings{
		Enabled:         config.LogEnabled,
		MobileSystem:    config.LogMobileSystem,
		Level:           config.LogLevel,
		File:            config.LogFile,
		MaxSize:         config.LogMaxSize,
		MaxBackups:      config.LogMaxBackups,
		CompressRotated: config.LogCompressRotated,
	}
	colors := !(*logWithoutColors) && terminal.IsTerminal(int(os.Stdin.Fd()))
	if err := logutils.OverrideRootLogWithConfig(logSettings, colors); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}
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

Options:
`
	fmt.Fprint(os.Stderr, usage)
	flag.PrintDefaults()
}

const pathWalletRoot = "m/44'/60'/0'/0"
const pathEIP1581 = "m/43'/60'/1581'"
const pathDefaultChat = pathEIP1581 + "/0'/0"
const pathDefaultWallet = pathWalletRoot + "/0"

var paths = []string{pathWalletRoot, pathEIP1581, pathDefaultChat, pathDefaultWallet}

func defaultSettings(generatedAccountInfo generator.GeneratedAccountInfo, derivedAddresses map[string]generator.AccountInfo, mnemonic *string) (*settings.Settings, error) {
	chatKeyString := derivedAddresses[pathDefaultChat].PublicKey

	defaultSettings := &settings.Settings{}
	defaultSettings.KeyUID = generatedAccountInfo.KeyUID
	defaultSettings.Address = types.HexToAddress(generatedAccountInfo.Address)
	defaultSettings.WalletRootAddress = types.HexToAddress(derivedAddresses[pathWalletRoot].Address)

	// Set chat key & name
	name, err := alias.GenerateFromPublicKeyString(chatKeyString)
	if err != nil {
		return nil, err
	}
	defaultSettings.Name = name
	defaultSettings.PublicKey = chatKeyString

	defaultSettings.DappsAddress = types.HexToAddress(derivedAddresses[pathDefaultWallet].Address)
	defaultSettings.EIP1581Address = types.HexToAddress(derivedAddresses[pathEIP1581].Address)
	defaultSettings.Mnemonic = mnemonic

	signingPhrase, err := buildSigningPhrase()
	if err != nil {
		return nil, err
	}
	defaultSettings.SigningPhrase = signingPhrase

	defaultSettings.SendPushNotifications = true
	defaultSettings.InstallationID = uuid.New().String()
	defaultSettings.UseMailservers = true

	defaultSettings.PreviewPrivacy = true
	defaultSettings.PeerSyncingEnabled = false
	defaultSettings.Currency = "usd"
	defaultSettings.ProfilePicturesVisibility = settings.ProfilePicturesVisibilityEveryone
	defaultSettings.LinkPreviewRequestEnabled = true

	defaultSettings.TestNetworksEnabled = false

	visibleTokens := make(map[string][]string)
	visibleTokens["mainnet"] = []string{"SNT"}
	visibleTokensJSON, err := json.Marshal(visibleTokens)
	if err != nil {
		return nil, err
	}
	visibleTokenJSONRaw := json.RawMessage(visibleTokensJSON)
	defaultSettings.WalletVisibleTokens = &visibleTokenJSONRaw

	// TODO: fix this
	networks := make([]map[string]string, 0)
	networksJSON, err := json.Marshal(networks)
	if err != nil {
		return nil, err
	}
	networkRawMessage := json.RawMessage(networksJSON)
	defaultSettings.Networks = &networkRawMessage
	defaultSettings.CurrentNetwork = "mainnet_rpc"

	return defaultSettings, nil
}

func defaultNodeConfig(installationID string) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.NetworkID = 1
	nodeConfig.LogLevel = "DEBUG"
	nodeConfig.DataDir = api.DefaultDataDir
	nodeConfig.HTTPEnabled = true
	nodeConfig.HTTPPort = 8545
	// FIXME: This should be taken from CLI flags.
	nodeConfig.HTTPHost = "0.0.0.0"
	// FIXME: This should be taken from CLI flags.
	nodeConfig.HTTPVirtualHosts = []string{"localhost", "wakunode"}
	nodeConfig.APIModules = *apiModules
	// Disable to avoid errors about empty ClusterConfig.BootNodes.
	nodeConfig.NoDiscovery = true

	nodeConfig.Name = "StatusIM"
	clusterConfig, err := params.LoadClusterConfigFromFleet("status.prod")
	if err != nil {
		return nil, err
	}
	nodeConfig.ClusterConfig = *clusterConfig

	nodeConfig.WalletConfig = params.WalletConfig{Enabled: true}
	nodeConfig.LocalNotificationsConfig = params.LocalNotificationsConfig{Enabled: true}
	nodeConfig.BrowsersConfig = params.BrowsersConfig{Enabled: true}
	nodeConfig.PermissionsConfig = params.PermissionsConfig{Enabled: true}
	nodeConfig.MailserversConfig = params.MailserversConfig{Enabled: true}
	err = api.SetDefaultFleet(nodeConfig)
	if err != nil {
		return nil, err
	}

	nodeConfig.WakuV2Config = params.WakuV2Config{
		Enabled:        true,
		EnableDiscV5:   true,
		DiscoveryLimit: 20,
		UDPPort:        9002,
	}

	nodeConfig.ShhextConfig = params.ShhextConfig{
		InstallationID:             installationID,
		MaxMessageDeliveryAttempts: api.DefaultMaxMessageDeliveryAttempts,
		MailServerConfirmations:    true,
		VerifyTransactionURL:       "",
		VerifyENSURL:               "",
		VerifyENSContractAddress:   "",
		VerifyTransactionChainID:   api.DefaultVerifyTransactionChainID,
		DataSyncEnabled:            true,
		PFSEnabled:                 true,
	}

	// TODO: check topics

	return nodeConfig, nil
}

func ImportAccount(seedPhrase string, backend *api.GethStatusBackend) error {
	backend.UpdateRootDataDir(*dataDir)
	manager := backend.AccountManager()
	if err := manager.InitKeystore(*dataDir); err != nil {
		return err
	}
	err := backend.OpenAccounts()
	if err != nil {
		logger.Error("failed open accounts", err)
		return err
	}
	generator := manager.AccountsGenerator()
	generatedAccountInfo, err := generator.ImportMnemonic(seedPhrase, "")
	if err != nil {
		logger.Error("failed import mnemonic", err)
		return err
	}

	derivedAddresses, err := generator.DeriveAddresses(generatedAccountInfo.ID, paths)
	if err != nil {
		logger.Error("failed derive", err)
		return err
	}

	var exist bool
	_, err = generator.StoreDerivedAccounts(generatedAccountInfo.ID, "", paths)
	if err != nil && err.Error() == "account already exists" {
		exist = true
	} else if err != nil {
		logger.Error("failed store derive", err)
		return err
	}

	account := multiaccounts.Account{
		KeyUID:        generatedAccountInfo.KeyUID,
		KDFIterations: dbsetup.ReducedKDFIterationsNumber,
	}
	settings, err := defaultSettings(generatedAccountInfo, derivedAddresses, &seedPhrase)
	if err != nil {
		return err
	}

	nodeConfig, err := defaultNodeConfig(settings.InstallationID)
	if err != nil {
		return err
	}

	walletDerivedAccount := derivedAddresses[pathDefaultWallet]
	walletAccount := &accounts.Account{
		PublicKey: types.Hex2Bytes(walletDerivedAccount.PublicKey),
		KeyUID:    generatedAccountInfo.KeyUID,
		Address:   types.HexToAddress(walletDerivedAccount.Address),
		ColorID:   "",
		Wallet:    true,
		Path:      pathDefaultWallet,
		Name:      "Ethereum account",
	}

	chatDerivedAccount := derivedAddresses[pathDefaultChat]
	chatAccount := &accounts.Account{
		PublicKey: types.Hex2Bytes(chatDerivedAccount.PublicKey),
		KeyUID:    generatedAccountInfo.KeyUID,
		Address:   types.HexToAddress(chatDerivedAccount.Address),
		Name:      settings.Name,
		Chat:      true,
		Path:      pathDefaultChat,
	}

	fmt.Println(nodeConfig)
	accounts := []*accounts.Account{walletAccount, chatAccount}
	if !exist {
		return backend.StartNodeWithAccountAndInitialConfig(account, "", *settings, nodeConfig, accounts, nil)
	}
	return backend.StartNodeWithAccount(account, "", nodeConfig, nil)
}

func retrieveMessagesLoop(messenger *protocol.Messenger, tick time.Duration) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for { //nolint: gosimple
		select {
		case <-ticker.C:
			_, err := messenger.RetrieveAll()
			if err != nil {
				logger.Error("failed to retrieve raw messages", "err", err)
				continue
			}
		}
	}
}

// exitOnInterruptSignal catches interrupt signal (SIGINT) and
// stops the node. It times out after 5 seconds
// if the node can not be stopped.
func exitOnInterruptSignal(statusNode *node.StatusNode) <-chan struct{} {
	interruptCh := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		defer signal.Stop(sigChan)
		<-sigChan
		close(interruptCh)
		logger.Info("Got interrupt, shutting down...")
		if err := statusNode.Stop(); err != nil {
			logger.Error("Failed to stop node", "error", err)
			os.Exit(1)
		}
	}()
	return interruptCh
}

// startCollectingStats collects various stats about the node and other protocols like Whisper.
func startNodeMetrics(interruptCh <-chan struct{}, statusNode *node.StatusNode) {
	logger.Info("Starting collecting node metrics")

	gNode := statusNode.GethNode()
	if gNode == nil {
		logger.Error("Failed to run metrics because it could not get the node")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		// Try to subscribe and collect metrics. In case of an error, retry.
		for {
			if err := nodemetrics.SubscribeServerEvents(ctx, gNode); err != nil {
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
