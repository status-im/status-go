package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/common/shard"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
	wakuextn "github.com/status-im/status-go/services/wakuext"
)

const (
	serverClientName = "Statusd"
)

var (
	configFiles      configFlags
	logLevel         = flag.String("log", "", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logWithoutColors = flag.Bool("log-without-color", false, "Disables log colors")
	ipcEnabled       = flag.Bool("ipc", false, "Enable IPC RPC endpoint")
	ipcFile          = flag.String("ipcfile", "", "Set IPC file path")
	seedPhrase       = flag.String("seed-phrase", "", "Seed phrase")
	version          = flag.Bool("version", false, "Print version and dump configuration")
	communityID      = flag.String("community-id", "", "The id of the community")
	shardCluster     = flag.Int("shard-cluster", shard.UndefinedShardValue, "The shard cluster in which the of the community is published")
	shardIndex       = flag.Int("shard-index", shard.UndefinedShardValue, "The shard index in which the community is published")
	chatID           = flag.String("chat-id", "", "The id of the chat")

	dataDir   = flag.String("dir", getDefaultDataDir(), "Directory used by node to store data")
	networkID = flag.Int(
		"network-id",
		params.GoerliNetworkID,
		fmt.Sprintf(
			"A network ID: %d (Mainnet), %d (Goerli)",
			params.MainNetworkID, params.GoerliNetworkID,
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
		printVersion(config)
		return
	}

	backend := api.NewGethStatusBackend()
	err = ImportAccount(*seedPhrase, backend)
	if err != nil {
		logger.Error("failed import account", "err", err)
		return
	}

	wakuextservice := backend.StatusNode().WakuExtService()
	if wakuextservice == nil {
		logger.Error("wakuext not available")
		return
	}

	wakuext := wakuextn.NewPublicAPI(wakuextservice)

	// This will start the push notification server as well as
	// the config is set to Enabled
	_, err = wakuext.StartMessenger()
	if err != nil {
		logger.Error("failed to start messenger", "error", err)
		return
	}

	messenger := wakuextservice.Messenger()

	var s *shard.Shard = nil
	if shardCluster != nil && shardIndex != nil && *shardCluster != shard.UndefinedShardValue && *shardIndex != shard.UndefinedShardValue {
		s = &shard.Shard{
			Cluster: uint16(*shardCluster),
			Index:   uint16(*shardIndex),
		}
	}

	community, err := messenger.FetchCommunity(&protocol.FetchCommunityRequest{
		CommunityKey:    *communityID,
		Shard:           s,
		TryDatabase:     true,
		WaitForResponse: true,
	})
	if err != nil {

		logger.Error("community error", "error", err)
		return

	}
	chat := community.Chats()[*chatID]
	if chat == nil {
		logger.Warn("Chat not found")
		return
	}
	logger.Info("GOT community", "comm", chat)

	response, err := messenger.JoinCommunity(context.Background(), community.ID(), false)
	if err != nil {
		logger.Error("failed to join community", "err", err)
	}

	var targetChat *protocol.Chat

	for _, c := range response.Chats() {
		if strings.Contains(c.ID, *chatID) {
			targetChat = c
		}
	}

	if targetChat == nil {
		logger.Warn("chat not found")
		return
	}

	id := uuid.New().String()

	ticker := time.NewTicker(2 * time.Second)
	count := 0

	for { // nolint: gosimple
		select {
		case <-ticker.C:
			count++
			timestamp := time.Now().Format(time.RFC3339)
			logger.Info("Publishing", "id", id, "count", count, "time", timestamp)
			inputMessage := common.NewMessage()

			inputMessage.Text = fmt.Sprintf("%d\n%s\n%s", count, timestamp, id)
			inputMessage.LocalChatID = targetChat.ID
			inputMessage.ChatId = targetChat.ID
			inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN

			_, err := messenger.SendChatMessage(context.Background(), inputMessage)
			if err != nil {
				logger.Error("failed to send a message", "err", err)
			}

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
Usage: ping-community [options]
Example:
  ping-community --seed-phrase "your seed phrase" --community-id "community-id" --chat-id "chat-id"
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
	nodeConfig.LogLevel = "ERROR"
	nodeConfig.DataDir = "/ethereum/mainnet_rpc"
	nodeConfig.UpstreamConfig = params.UpstreamRPCConfig{
		Enabled: true,
		URL:     "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938",
	}

	nodeConfig.Name = "StatusIM"
	clusterConfig, err := params.LoadClusterConfigFromFleet("eth.prod")
	if err != nil {
		return nil, err
	}
	nodeConfig.ClusterConfig = *clusterConfig

	nodeConfig.WalletConfig = params.WalletConfig{Enabled: true}
	nodeConfig.LocalNotificationsConfig = params.LocalNotificationsConfig{Enabled: true}
	nodeConfig.BrowsersConfig = params.BrowsersConfig{Enabled: true}
	nodeConfig.PermissionsConfig = params.PermissionsConfig{Enabled: true}
	nodeConfig.MailserversConfig = params.MailserversConfig{Enabled: true}
	nodeConfig.WakuConfig = params.WakuConfig{
		Enabled:     true,
		LightClient: true,
		MinimumPoW:  0.000001,
	}

	nodeConfig.ShhextConfig = params.ShhextConfig{
		BackupDisabledDataDir:      "",
		InstallationID:             installationID,
		MaxMessageDeliveryAttempts: 6,
		MailServerConfirmations:    true,
		VerifyTransactionURL:       "",
		VerifyENSURL:               "",
		VerifyENSContractAddress:   "",
		VerifyTransactionChainID:   1,
		DataSyncEnabled:            true,
		PFSEnabled:                 true,
	}

	// TODO: check topics

	return nodeConfig, nil
}

func ImportAccount(seedPhrase string, backend *api.GethStatusBackend) error {
	backend.UpdateRootDataDir("./tmp")
	manager := backend.AccountManager()
	if err := manager.InitKeystore("./tmp"); err != nil {
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
		return err
	}

	derivedAddresses, err := generator.DeriveAddresses(generatedAccountInfo.ID, paths)
	if err != nil {
		return err
	}

	_, err = generator.StoreDerivedAccounts(generatedAccountInfo.ID, "", paths)
	if err != nil {
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
	err = backend.StartNodeWithAccountAndInitialConfig(account, "", *settings, nodeConfig, accounts)
	if err != nil {
		logger.Error("start node", err)
		return err
	}

	return nil
}
