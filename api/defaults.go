package api

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"path/filepath"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

const (
	pathWalletRoot           = "m/44'/60'/0'/0"
	pathEIP1581              = "m/43'/60'/1581'"
	pathDefaultChat          = pathEIP1581 + "/0'/0"
	pathEncryption           = pathEIP1581 + "/1'/0"
	pathDefaultWallet        = pathWalletRoot + "/0"
	defaultMnemonicLength    = 12
	shardsTestClusterID      = 16
	walletAccountDefaultName = "Account 1"

	DefaultKeystoreRelativePath   = "keystore"
	DefaultKeycardPairingDataFile = "/ethereum/mainnet_rpc/keycard/pairings.json"
	DefaultDataDir                = "/ethereum/mainnet_rpc"
	DefaultNodeName               = "StatusIM"
	DefaultLogFile                = "geth.log"
	DefaultAPILogFile             = "api.log"

	DefaultLogLevel                   = "ERROR"
	DefaultMaxPeers                   = 20
	DefaultMaxPendingPeers            = 20
	DefaultListenAddr                 = ":0"
	DefaultMaxMessageDeliveryAttempts = 3
	DefaultVerifyTransactionChainID   = 1
	DefaultCurrentNetwork             = "mainnet_rpc"
)

var (
	paths = []string{pathWalletRoot, pathEIP1581, pathDefaultChat, pathDefaultWallet, pathEncryption}

	DefaultFleet = params.FleetStatusProd

	overrideApiConfig = overrideApiConfigProd
)

func defaultSettings(keyUID string, address string, derivedAddresses map[string]generator.AccountInfo) (*settings.Settings, error) {
	chatKeyString := derivedAddresses[pathDefaultChat].PublicKey

	s := &settings.Settings{}
	s.BackupEnabled = true
	logLevel := "INFO"
	s.LogLevel = &logLevel
	s.ProfilePicturesShowTo = settings.ProfilePicturesShowToEveryone
	s.ProfilePicturesVisibility = settings.ProfilePicturesVisibilityEveryone
	s.KeyUID = keyUID
	s.Address = types.HexToAddress(address)
	s.WalletRootAddress = types.HexToAddress(derivedAddresses[pathWalletRoot].Address)
	s.URLUnfurlingMode = settings.URLUnfurlingAlwaysAsk

	// Set chat key & name
	name, err := alias.GenerateFromPublicKeyString(chatKeyString)
	if err != nil {
		return nil, err
	}
	s.Name = name
	s.PublicKey = chatKeyString

	s.DappsAddress = types.HexToAddress(derivedAddresses[pathDefaultWallet].Address)
	s.EIP1581Address = types.HexToAddress(derivedAddresses[pathEIP1581].Address)

	signingPhrase, err := buildSigningPhrase()
	if err != nil {
		return nil, err
	}
	s.SigningPhrase = signingPhrase

	s.SendPushNotifications = true
	s.InstallationID = multidevice.GenerateInstallationID()
	s.UseMailservers = true

	s.PreviewPrivacy = true
	s.PeerSyncingEnabled = false
	s.Currency = "usd"
	s.LinkPreviewRequestEnabled = true

	visibleTokens := make(map[string][]string)
	visibleTokens["mainnet"] = []string{"SNT"}
	visibleTokensJSON, err := json.Marshal(visibleTokens)
	if err != nil {
		return nil, err
	}
	visibleTokenJSONRaw := json.RawMessage(visibleTokensJSON)
	s.WalletVisibleTokens = &visibleTokenJSONRaw

	// TODO: fix this
	networks := make([]map[string]string, 0)
	networksJSON, err := json.Marshal(networks)
	if err != nil {
		return nil, err
	}
	networkRawMessage := json.RawMessage(networksJSON)
	s.Networks = &networkRawMessage
	s.CurrentNetwork = DefaultCurrentNetwork

	s.TokenGroupByCommunity = false
	s.ShowCommunityAssetWhenSendingTokens = true
	s.DisplayAssetsBelowBalance = false

	s.TestNetworksEnabled = false

	// Default user status
	currentUserStatus, err := json.Marshal(protocol.UserStatus{
		PublicKey:  chatKeyString,
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      0,
		CustomText: "",
	})
	if err != nil {
		return nil, err
	}
	userRawMessage := json.RawMessage(currentUserStatus)
	s.CurrentUserStatus = &userRawMessage

	return s, nil
}

func SetDefaultFleet(nodeConfig *params.NodeConfig) error {
	return SetFleet(DefaultFleet, nodeConfig)
}

func SetFleet(fleet string, nodeConfig *params.NodeConfig) error {
	specifiedWakuV2Config := nodeConfig.WakuV2Config
	nodeConfig.WakuV2Config = params.WakuV2Config{
		Enabled:        true,
		EnableDiscV5:   true,
		DiscoveryLimit: 20,
		Host:           "0.0.0.0",
		AutoUpdate:     true,
		// mobile may need override following options
		LightClient:                            specifiedWakuV2Config.LightClient,
		EnableStoreConfirmationForMessagesSent: specifiedWakuV2Config.EnableStoreConfirmationForMessagesSent,
		EnableMissingMessageVerification:       specifiedWakuV2Config.EnableMissingMessageVerification,
		Nameserver:                             specifiedWakuV2Config.Nameserver,
	}

	clusterConfig, err := params.LoadClusterConfigFromFleet(fleet)
	if err != nil {
		return err
	}
	nodeConfig.ClusterConfig = *clusterConfig
	nodeConfig.ClusterConfig.Fleet = fleet
	nodeConfig.ClusterConfig.WakuNodes = params.DefaultWakuNodes(fleet)
	nodeConfig.ClusterConfig.DiscV5BootstrapNodes = params.DefaultDiscV5Nodes(fleet)

	if fleet == params.FleetStatusProd {
		nodeConfig.ClusterConfig.ClusterID = shardsTestClusterID
	}

	return nil
}

func buildWalletConfig(request *requests.WalletSecretsConfig, statusProxyEnabled bool) params.WalletConfig {
	walletConfig := params.WalletConfig{
		Enabled:        true,
		AlchemyAPIKeys: make(map[uint64]string),
	}

	if request.StatusProxyStageName != "" {
		walletConfig.StatusProxyStageName = request.StatusProxyStageName
	}

	if request.OpenseaAPIKey != "" {
		walletConfig.OpenseaAPIKey = request.OpenseaAPIKey
	}

	if request.RaribleMainnetAPIKey != "" {
		walletConfig.RaribleMainnetAPIKey = request.RaribleMainnetAPIKey
	}

	if request.RaribleTestnetAPIKey != "" {
		walletConfig.RaribleTestnetAPIKey = request.RaribleTestnetAPIKey
	}

	if request.InfuraToken != "" {
		walletConfig.InfuraAPIKey = request.InfuraToken
	}

	if request.InfuraSecret != "" {
		walletConfig.InfuraAPIKeySecret = request.InfuraSecret
	}

	if request.AlchemyEthereumMainnetToken != "" {
		walletConfig.AlchemyAPIKeys[mainnetChainID] = request.AlchemyEthereumMainnetToken
	}
	if request.AlchemyEthereumSepoliaToken != "" {
		walletConfig.AlchemyAPIKeys[sepoliaChainID] = request.AlchemyEthereumSepoliaToken
	}
	if request.AlchemyArbitrumMainnetToken != "" {
		walletConfig.AlchemyAPIKeys[arbitrumChainID] = request.AlchemyArbitrumMainnetToken
	}
	if request.AlchemyArbitrumSepoliaToken != "" {
		walletConfig.AlchemyAPIKeys[arbitrumSepoliaChainID] = request.AlchemyArbitrumSepoliaToken
	}
	if request.AlchemyOptimismMainnetToken != "" {
		walletConfig.AlchemyAPIKeys[optimismChainID] = request.AlchemyOptimismMainnetToken
	}
	if request.AlchemyOptimismSepoliaToken != "" {
		walletConfig.AlchemyAPIKeys[optimismSepoliaChainID] = request.AlchemyOptimismSepoliaToken
	}
	if request.StatusProxyMarketUser != "" {
		walletConfig.StatusProxyMarketUser = request.StatusProxyMarketUser
	}
	if request.StatusProxyMarketPassword != "" {
		walletConfig.StatusProxyMarketPassword = request.StatusProxyMarketPassword
	}
	if request.StatusProxyBlockchainUser != "" {
		walletConfig.StatusProxyBlockchainUser = request.StatusProxyBlockchainUser
	}
	if request.StatusProxyBlockchainPassword != "" {
		walletConfig.StatusProxyBlockchainPassword = request.StatusProxyBlockchainPassword
	}

	walletConfig.StatusProxyEnabled = statusProxyEnabled

	return walletConfig
}

func overrideApiConfigProd(nodeConfig *params.NodeConfig, config *requests.APIConfig) {
	nodeConfig.APIModules = config.APIModules
	nodeConfig.ConnectorConfig.Enabled = config.ConnectorEnabled

	nodeConfig.HTTPEnabled = config.HTTPEnabled
	nodeConfig.HTTPHost = config.HTTPHost
	nodeConfig.HTTPPort = config.HTTPPort
	nodeConfig.HTTPVirtualHosts = config.HTTPVirtualHosts

	nodeConfig.WSEnabled = config.WSEnabled
	nodeConfig.WSHost = config.WSHost
	nodeConfig.WSPort = config.WSPort
}

func DefaultNodeConfig(installationID string, request *requests.CreateAccount, opts ...params.Option) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.RootDataDir = request.RootDataDir
	nodeConfig.LogEnabled = request.LogEnabled
	nodeConfig.LogFile = DefaultLogFile
	nodeConfig.LogDir = request.LogFilePath
	nodeConfig.LogLevel = DefaultLogLevel
	nodeConfig.DataDir = DefaultDataDir
	nodeConfig.ProcessBackedupMessages = false
	nodeConfig.KeycardPairingDataFile = DefaultKeycardPairingDataFile
	if request.KeycardPairingDataFile != nil {
		nodeConfig.KeycardPairingDataFile = *request.KeycardPairingDataFile
	}

	if request.LogLevel != nil {
		nodeConfig.LogLevel = *request.LogLevel
		nodeConfig.LogEnabled = true
	} else {
		nodeConfig.LogEnabled = false
	}

	if request.TestOverrideNetworks != nil {
		nodeConfig.Networks = request.TestOverrideNetworks
	} else {
		nodeConfig.Networks = BuildDefaultNetworks(&request.WalletSecretsConfig)
	}

	if request.NetworkID != nil {
		nodeConfig.NetworkID = *request.NetworkID
	} else {
		nodeConfig.NetworkID = nodeConfig.Networks[0].ChainID
	}

	nodeConfig.Name = DefaultNodeName
	nodeConfig.NoDiscovery = true
	nodeConfig.MaxPeers = DefaultMaxPeers
	nodeConfig.MaxPendingPeers = DefaultMaxPendingPeers

	nodeConfig.WalletConfig = buildWalletConfig(&request.WalletSecretsConfig, request.StatusProxyEnabled)

	nodeConfig.LocalNotificationsConfig = params.LocalNotificationsConfig{Enabled: true}
	nodeConfig.BrowsersConfig = params.BrowsersConfig{Enabled: true}
	nodeConfig.PermissionsConfig = params.PermissionsConfig{Enabled: true}
	nodeConfig.MailserversConfig = params.MailserversConfig{Enabled: true}

	nodeConfig.ListenAddr = DefaultListenAddr

	fleet := request.WakuV2Fleet
	if fleet == "" {
		fleet = DefaultFleet
	}

	err := SetFleet(fleet, nodeConfig)
	if err != nil {
		return nil, err
	}

	if request.WakuV2LightClient {
		nodeConfig.WakuV2Config.LightClient = true
	}

	if request.WakuV2EnableMissingMessageVerification {
		nodeConfig.WakuV2Config.EnableMissingMessageVerification = true
	}

	if request.WakuV2EnableStoreConfirmationForMessagesSent {
		nodeConfig.WakuV2Config.EnableStoreConfirmationForMessagesSent = true
	}

	if request.WakuV2Nameserver != nil {
		nodeConfig.WakuV2Config.Nameserver = *request.WakuV2Nameserver
	}

	if request.TelemetryServerURL != "" {
		nodeConfig.WakuV2Config.TelemetryServerURL = request.TelemetryServerURL
	}

	nodeConfig.ShhextConfig = params.ShhextConfig{
		InstallationID:             installationID,
		MaxMessageDeliveryAttempts: DefaultMaxMessageDeliveryAttempts,
		MailServerConfirmations:    true,
		VerifyTransactionChainID:   DefaultVerifyTransactionChainID,
		DataSyncEnabled:            true,
		PFSEnabled:                 true,
	}

	if request.VerifyTransactionURL != nil {
		nodeConfig.ShhextConfig.VerifyTransactionURL = *request.VerifyTransactionURL
	} else {
		nodeConfig.ShhextConfig.VerifyTransactionURL = mainnet(request.WalletSecretsConfig.StatusProxyStageName).FallbackURL
	}

	if request.VerifyENSURL != nil {
		nodeConfig.ShhextConfig.VerifyENSURL = *request.VerifyENSURL
	} else {
		nodeConfig.ShhextConfig.VerifyENSURL = mainnet(request.WalletSecretsConfig.StatusProxyStageName).FallbackURL
	}

	if request.VerifyTransactionChainID != nil {
		nodeConfig.ShhextConfig.VerifyTransactionChainID = *request.VerifyTransactionChainID
	}

	if request.VerifyENSContractAddress != nil {
		nodeConfig.ShhextConfig.VerifyENSContractAddress = *request.VerifyENSContractAddress
	}

	if request.NetworkID != nil {
		nodeConfig.NetworkID = *request.NetworkID
	}

	nodeConfig.TorrentConfig = params.TorrentConfig{
		Enabled:    false,
		Port:       0,
		DataDir:    filepath.Join(nodeConfig.RootDataDir, params.ArchivesRelativePath),
		TorrentDir: filepath.Join(nodeConfig.RootDataDir, params.TorrentTorrentsRelativePath),
	}

	if request.TorrentConfigEnabled != nil {
		nodeConfig.TorrentConfig.Enabled = *request.TorrentConfigEnabled

	}
	if request.TorrentConfigPort != nil {
		nodeConfig.TorrentConfig.Port = *request.TorrentConfigPort
	}

	if request.APIConfig != nil {
		overrideApiConfig(nodeConfig, request.APIConfig)
	}

	for _, opt := range opts {
		if err := opt(nodeConfig); err != nil {
			return nil, err
		}
	}

	return nodeConfig, nil
}

func DefaultKeystorePath(rootDataDir string, keyUID string) (string, string) {
	relativePath := filepath.Join(DefaultKeystoreRelativePath, keyUID)
	absolutePath := filepath.Join(rootDataDir, relativePath)
	return relativePath, absolutePath
}

func buildSigningPhrase() (string, error) {
	length := big.NewInt(int64(len(dictionary)))
	a, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}
	b, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}
	c, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}

	return dictionary[a.Int64()] + " " + dictionary[b.Int64()] + " " + dictionary[c.Int64()], nil
}

func randomWalletEmoji() (string, error) {
	count := big.NewInt(int64(len(animalsAndNatureEmojis)))
	index, err := rand.Int(rand.Reader, count)
	if err != nil {
		return "", err
	}
	return animalsAndNatureEmojis[index.Int64()], nil
}

var animalsAndNatureEmojis = []string{
	"🐵", "🐒", "🦍", "🦧", "🦣", "🦏", "🦛", "🐪", "🐫", "🦙",
	"🐃", "🐂", "🐄", "🐎", "🦄", "🦓", "🦌", "🐐", "🐏", "🐑",
	"🦙", "🐘", "🦣", "🦛", "🦏", "🦒", "🐁", "🐀", "🐹", "🐰",
	"🐇", "🐿️", "🦔", "🦇", "🐻", "🐻‍❄️", "🐨", "🐼", "🦥", "🦦",
	"🦨", "🦘", "🦡", "🐾", "🐉", "🐲", "🌵", "🎄", "🌲", "🌳",
	"🌴", "🌱", "🌿", "☘️", "🍀", "🎍", "🎋", "🍃", "🍂", "🍁",
	"🍄", "🐚", "🪨", "🌾", "💐", "🌷", "🌹", "🥀", "🌺", "🌸",
	"🌼", "🌻", "🌞", "🌝", "🌛", "🌜", "🌚", "🌕", "🌖", "🌗",
	"🌘", "🌑", "🌒", "🌓", "🌔", "🌙", "🌎", "🌍", "🌏", "🪐",
	"💫", "⭐", "🌟", "✨", "⚡", "☄️", "💥", "🔥", "🌪️", "🌈",
	"☀️", "🌤️", "⛅", "🌥️", "☁️", "🌦️", "🌧️", "⛈️", "🌩️", "🌨️",
	"❄️", "☃️", "⛄", "🌬️", "💨", "💧", "💦", "🌊",
}
