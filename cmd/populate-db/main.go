package main

import (
	"context"
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
	"github.com/status-im/status-go/protocol/requests"
	wakuextn "github.com/status-im/status-go/services/wakuext"
)

type testTimeSource struct{}

func (t *testTimeSource) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix())
}

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
	nAddedContacts   = flag.Int("added-contacts", 100, "Number of added contacts to create")
	nContacts        = flag.Int("contacts", 100, "Number of contacts to create")
	nPublicChats     = flag.Int("public-chats", 5, "Number of public chats")
	nMessages        = flag.Int("number-of-messages", 0, "Number of messages for each chat")
	nOneToOneChats   = flag.Int("one-to-one-chats", 5, "Number of one to one chats")

	dataDir   = flag.String("dir", getDefaultDataDir(), "Directory used by node to store data")
	networkID = flag.Int(
		"network-id",
		params.RopstenNetworkID,
		fmt.Sprintf(
			"A network ID: %d (Mainnet), %d (Ropsten), %d (Rinkeby), %d (Goerli)",
			params.MainNetworkID, params.RopstenNetworkID, params.RinkebyNetworkID, params.GoerliNetworkID,
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

	logger.Info("Creating added contacts")

	for i := 0; i < *nAddedContacts; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			logger.Error("failed generate key", err)
			return
		}

		keyString := common.PubkeyToHex(&key.PublicKey)
		_, err = wakuext.AddContact(context.Background(), &requests.AddContact{ID: types.Hex2Bytes(keyString)})
		if err != nil {
			logger.Error("failed Add contact", "err", err)
			return
		}
	}

	logger.Info("Creating contacts")

	for i := 0; i < *nContacts; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			return
		}

		contact, err := protocol.BuildContactFromPublicKey(&key.PublicKey)
		if err != nil {
			return
		}

		_, err = wakuext.AddContact(context.Background(), &requests.AddContact{ID: types.Hex2Bytes(contact.ID)})
		if err != nil {
			return
		}
	}

	logger.Info("Creating public chats")

	for i := 0; i < *nPublicChats; i++ {
		chat := protocol.CreatePublicChat(randomString(10), &testTimeSource{})
		chat.SyncedTo = 0
		chat.SyncedFrom = 0

		err = wakuext.SaveChat(context.Background(), chat)
		if err != nil {
			return
		}

		var messages []*common.Message

		for i := 0; i < *nMessages; i++ {
			messages = append(messages, buildMessage(chat, i))

		}

		if len(messages) > 0 {
			if err := wakuext.SaveMessages(context.Background(), messages); err != nil {
				return
			}
		}

	}

	logger.Info("Creating one to one chats")

	for i := 0; i < *nOneToOneChats; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			return
		}

		keyString := common.PubkeyToHex(&key.PublicKey)
		chat := protocol.CreateOneToOneChat(keyString, &key.PublicKey, &testTimeSource{})
		chat.SyncedTo = 0
		chat.SyncedFrom = 0
		err = wakuext.SaveChat(context.Background(), chat)
		if err != nil {
			return
		}
		var messages []*common.Message

		for i := 0; i < *nMessages; i++ {
			messages = append(messages, buildMessage(chat, i))

		}

		if len(messages) > 0 {
			if err := wakuext.SaveMessages(context.Background(), messages); err != nil {
				return
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
	nodeConfig.LogLevel = "ERROR"
	nodeConfig.DataDir = "/ethereum/mainnet_rpc"
	nodeConfig.UpstreamConfig = params.UpstreamRPCConfig{
		Enabled: true,
		URL:     "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938",
	}

	nodeConfig.Name = "StatusIM"
	nodeConfig.Rendezvous = true
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
	nodeConfig.EnableNTPSync = true
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
		KeyUID: generatedAccountInfo.KeyUID,
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
	walletAccount := accounts.Account{
		PublicKey: types.Hex2Bytes(walletDerivedAccount.PublicKey),
		Address:   types.HexToAddress(walletDerivedAccount.Address),
		Color:     "",
		Wallet:    true,
		Path:      pathDefaultWallet,
		Name:      "Ethereum account",
	}

	chatDerivedAccount := derivedAddresses[pathDefaultChat]
	chatAccount := accounts.Account{
		PublicKey: types.Hex2Bytes(chatDerivedAccount.PublicKey),
		Address:   types.HexToAddress(chatDerivedAccount.Address),
		Name:      settings.Name,
		Chat:      true,
		Path:      pathDefaultChat,
	}

	fmt.Println(nodeConfig)
	accounts := []accounts.Account{walletAccount, chatAccount}
	err = backend.StartNodeWithAccountAndInitialConfig(account, "", *settings, nodeConfig, accounts)
	if err != nil {
		logger.Error("start node", err)
		return err
	}

	return nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func buildMessage(chat *protocol.Chat, count int) *common.Message {
	key, err := crypto.GenerateKey()
	if err != nil {
		logger.Error("failed build message", err)
		return nil
	}

	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
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
