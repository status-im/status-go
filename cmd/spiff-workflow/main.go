package main

import (
	"encoding/json"
	"flag"
	"fmt"
	stdlog "log"
	"math/rand"
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
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
	waku2extn "github.com/status-im/status-go/services/wakuv2ext"
	"github.com/status-im/status-go/sqlite"
)

type testTimeSource struct{}

func (t *testTimeSource) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix()) * 1000
}

const (
	serverClientName = "Statusd"
)

var (
	configFiles      configFlags
	logLevel         = flag.String("log", "", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logWithoutColors = flag.Bool("log-without-color", false, "Disables log colors")
	seedPhrase       = flag.String("seed-phrase", "", "Seed phrase")
	version          = flag.Bool("version", false, "Print version and dump configuration")

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

	wakuextservice := backend.StatusNode().WakuV2ExtService()
	if wakuextservice == nil {
		logger.Error("wakuext not available")
		return
	}

	wakuext := waku2extn.NewPublicAPI(wakuextservice)

	// This will start the push notification server as well as
	// the config is set to Enabled
	_, err = wakuext.StartMessenger()
	if err != nil {
		logger.Error("failed to start messenger", "error", err)
		return
	}

	retrieveMessagesLoop(wakuext.Messenger(), 300*time.Millisecond)

}

func getDefaultDataDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".statusd")
	}
	return "./statusd-data"
}

func setupLogging(config *params.NodeConfig) {
	if *logLevel != "" {
		config.LogLevel = "DEBUG"
	}

	logSettings := logutils.LogSettings{
		Enabled:         config.LogEnabled,
		MobileSystem:    config.LogMobileSystem,
		Level:           "DEBUG",
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

	settings := &settings.Settings{}
	settings.KeyUID = generatedAccountInfo.KeyUID
	settings.Address = types.HexToAddress(generatedAccountInfo.Address)
	settings.WalletRootAddress = types.HexToAddress(derivedAddresses[pathWalletRoot].Address)

	// Set chat key & name
	name, err := alias.GenerateFromPublicKeyString(chatKeyString)
	if err != nil {
		return nil, err
	}
	settings.Name = name
	settings.PublicKey = chatKeyString

	settings.DappsAddress = types.HexToAddress(derivedAddresses[pathDefaultWallet].Address)
	settings.EIP1581Address = types.HexToAddress(derivedAddresses[pathEIP1581].Address)
	settings.Mnemonic = mnemonic

	signingPhrase, err := buildSigningPhrase()
	if err != nil {
		return nil, err
	}
	settings.SigningPhrase = signingPhrase

	settings.SendPushNotifications = true
	settings.InstallationID = uuid.New().String()
	settings.UseMailservers = true

	settings.PreviewPrivacy = true
	settings.Currency = "usd"
	settings.ProfilePicturesVisibility = 1
	settings.LinkPreviewRequestEnabled = true

	visibleTokens := make(map[string][]string)
	visibleTokens["mainnet"] = []string{"SNT"}
	visibleTokensJSON, err := json.Marshal(visibleTokens)
	if err != nil {
		return nil, err
	}
	visibleTokenJSONRaw := json.RawMessage(visibleTokensJSON)
	settings.WalletVisibleTokens = &visibleTokenJSONRaw

	// TODO: fix this
	networks := make([]map[string]string, 0)
	networksJSON, err := json.Marshal(networks)
	if err != nil {
		return nil, err
	}
	networkRawMessage := json.RawMessage(networksJSON)
	settings.Networks = &networkRawMessage
	settings.CurrentNetwork = "mainnet_rpc"

	return settings, nil
}

func defaultNodeConfig(installationID string) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.NetworkID = 1
	nodeConfig.LogLevel = "DEBUG"
	nodeConfig.DataDir = "/ethereum/mainnet_rpc"
	nodeConfig.HTTPEnabled = true
	nodeConfig.HTTPPort = 8545
	// FIXME: This should be taken from CLI flags.
	nodeConfig.HTTPHost = "0.0.0.0"
	// FIXME: This should be taken from CLI flags.
	nodeConfig.HTTPVirtualHosts = []string{"localhost", "wakunode"}
	nodeConfig.APIModules = "wakuext,ext,waku"

	nodeConfig.UpstreamConfig = params.UpstreamRPCConfig{
		Enabled: true,
		URL:     "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938",
	}

	nodeConfig.Name = "StatusIM"
	nodeConfig.Rendezvous = false
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
	nodes := []string{"enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@prod.nodes.status.im"}
	nodeConfig.ClusterConfig.WakuNodes = nodes
	nodeConfig.ClusterConfig.DiscV5BootstrapNodes = nodes

	nodeConfig.EnableNTPSync = true
	nodeConfig.WakuV2Config = params.WakuV2Config{
		Enabled:        true,
		EnableDiscV5:   true,
		DiscoveryLimit: 20,
		UDPPort:        9002,
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
		KDFIterations: sqlite.ReducedKDFIterationsNumber,
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
		Color:     "",
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
		return backend.StartNodeWithAccountAndInitialConfig(account, "", *settings, nodeConfig, accounts)
	}
	return backend.StartNodeWithAccount(account, "", nodeConfig)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func buildMessage(chat *protocol.Chat, count int) *common.Message {
	key, err := crypto.GenerateKey()
	if err != nil {
		logger.Error("failed build message", err)
		return nil
	}

	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	clock += uint64(count)
	message := &common.Message{}
	message.Text = fmt.Sprintf("test message %d", count)
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.From = common.PubkeyToHex(&key.PublicKey)
	data := []byte(uuid.New().String())
	message.ID = types.HexBytes(crypto.Keccak256(data)).String()
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	switch chat.ChatType {
	case protocol.ChatTypePublic, protocol.ChatTypeProfile:
		message.MessageType = protobuf.MessageType_PUBLIC_GROUP
	case protocol.ChatTypeOneToOne:
		message.MessageType = protobuf.MessageType_ONE_TO_ONE
	case protocol.ChatTypePrivateGroupChat:
		message.MessageType = protobuf.MessageType_PRIVATE_GROUP
	}

	_ = message.PrepareContent("")
	return message
}

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))] // nolint: gosec
	}
	return string(b)
}

func retrieveMessagesLoop(messenger *protocol.Messenger, tick time.Duration) {
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
		}
	}
}
