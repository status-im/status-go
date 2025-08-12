package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"

	accscommon "github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
)

const (
	defaultMnemonicLength    = 12
	walletAccountDefaultName = "Account 1"

	DefaultKeystoreRelativePath   = "keystore"
	DefaultKeycardPairingDataFile = "/ethereum/mainnet_rpc/keycard/pairings.json"
	DefaultDataDir                = "/ethereum/mainnet_rpc"
	DefaultAPILogFile             = "api.log"

	DefaultLogLevel                 = "ERROR"
	DefaultVerifyTransactionChainID = 1
	DefaultCurrentNetwork           = "mainnet_rpc"

	DefaultMaxMessageDeliveryAttempts = 3
)

var (
	paths = []string{accscommon.PathWalletRoot, accscommon.PathEIP1581Root, accscommon.PathEIP1581Chat, accscommon.PathDefaultWalletAccount, accscommon.PathEIP1581Encryption}

	DefaultFleet = params.FleetStatusProd
)

func defaultSettings(keyUID string, address string, derivedAddresses map[string]generator.AccountInfo) (*settings.Settings, error) {
	chatKeyString := derivedAddresses[accscommon.PathEIP1581Chat].PublicKey

	s := &settings.Settings{}
	s.BackupEnabled = true
	logLevel := "INFO"
	s.LogLevel = &logLevel
	s.ProfilePicturesShowTo = settings.ProfilePicturesShowToEveryone
	s.ProfilePicturesVisibility = settings.ProfilePicturesVisibilityEveryone
	s.KeyUID = keyUID
	s.Address = types.HexToAddress(address)
	s.WalletRootAddress = types.HexToAddress(derivedAddresses[accscommon.PathWalletRoot].Address)
	s.URLUnfurlingMode = settings.URLUnfurlingAlwaysAsk

	// Set chat key & name
	name, err := alias.GenerateFromPublicKeyString(chatKeyString)
	if err != nil {
		return nil, err
	}
	s.Name = name
	s.PublicKey = chatKeyString

	s.DappsAddress = types.HexToAddress(derivedAddresses[accscommon.PathDefaultWalletAccount].Address)
	s.EIP1581Address = types.HexToAddress(derivedAddresses[accscommon.PathEIP1581Root].Address)

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

	s.AutoRefreshTokensEnabled = true

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

func SetFleet(fleet string, nodeConfig *params.NodeConfig) error {
	specifiedWakuV2Config := nodeConfig.WakuV2Config
	nodeConfig.WakuV2Config = params.WakuV2Config{
		Enabled:        true,
		DiscoveryLimit: 20,
		Host:           "0.0.0.0",
		AutoUpdate:     true,
		// mobile may need override following options
		LightClient:                            specifiedWakuV2Config.LightClient,
		EnableStoreConfirmationForMessagesSent: specifiedWakuV2Config.EnableStoreConfirmationForMessagesSent,
		EnableMissingMessageVerification:       specifiedWakuV2Config.EnableMissingMessageVerification,
		Nameserver:                             specifiedWakuV2Config.Nameserver,
	}

	if !params.IsFleetSupported(fleet) {
		return fmt.Errorf("unknown fleet %s", fleet)
	}

	nodeConfig.ClusterConfig = params.DefaultClusterConfig(fleet)

	return nil
}

func buildWalletConfig(walletRequest *requests.WalletConfig, request *requests.WalletSecretsConfig) params.WalletConfig {
	walletConfig := params.WalletConfig{
		Enabled:                true,
		EnableMercuryoProvider: true,
		AlchemyAPIKeys:         make(map[uint64]security.SensitiveString),

		TokensListsAutoRefreshCheckInterval: walletRequest.TokensListsAutoRefreshCheckInterval,
		TokensListsAutoRefreshInterval:      walletRequest.TokensListsAutoRefreshInterval,
	}

	if request.StatusProxyStageName != "" {
		walletConfig.StatusProxyStageName = request.StatusProxyStageName
	}

	if !request.OpenseaAPIKey.Empty() {
		walletConfig.OpenseaAPIKey = request.OpenseaAPIKey
	}

	if !request.RaribleMainnetAPIKey.Empty() {
		walletConfig.RaribleMainnetAPIKey = request.RaribleMainnetAPIKey
	}

	if !request.RaribleTestnetAPIKey.Empty() {
		walletConfig.RaribleTestnetAPIKey = request.RaribleTestnetAPIKey
	}

	if !request.InfuraToken.Empty() {
		walletConfig.InfuraAPIKey = request.InfuraToken
	}

	if !request.InfuraSecret.Empty() {
		walletConfig.InfuraAPIKeySecret = request.InfuraSecret
	}

	if !request.AlchemyEthereumMainnetToken.Empty() {
		walletConfig.AlchemyAPIKeys[walletcommon.EthereumMainnet] = request.AlchemyEthereumMainnetToken
	}
	if !request.AlchemyEthereumSepoliaToken.Empty() {
		walletConfig.AlchemyAPIKeys[walletcommon.EthereumSepolia] = request.AlchemyEthereumSepoliaToken
	}
	if !request.AlchemyArbitrumMainnetToken.Empty() {
		walletConfig.AlchemyAPIKeys[walletcommon.ArbitrumMainnet] = request.AlchemyArbitrumMainnetToken
	}
	if !request.AlchemyArbitrumSepoliaToken.Empty() {
		walletConfig.AlchemyAPIKeys[walletcommon.ArbitrumSepolia] = request.AlchemyArbitrumSepoliaToken
	}
	if !request.AlchemyOptimismMainnetToken.Empty() {
		walletConfig.AlchemyAPIKeys[walletcommon.OptimismMainnet] = request.AlchemyOptimismMainnetToken
	}
	if !request.AlchemyOptimismSepoliaToken.Empty() {
		walletConfig.AlchemyAPIKeys[walletcommon.OptimismSepolia] = request.AlchemyOptimismSepoliaToken
	}
	if !request.AlchemyBaseMainnetToken.Empty() {
		walletConfig.AlchemyAPIKeys[walletcommon.BaseMainnet] = request.AlchemyBaseMainnetToken
	}
	if !request.AlchemyBaseSepoliaToken.Empty() {
		walletConfig.AlchemyAPIKeys[walletcommon.BaseSepolia] = request.AlchemyBaseSepoliaToken
	}
	if !request.StatusProxyMarketUser.Empty() {
		walletConfig.StatusProxyMarketUser = request.StatusProxyMarketUser
	}
	if !request.StatusProxyMarketPassword.Empty() {
		walletConfig.StatusProxyMarketPassword = request.StatusProxyMarketPassword
	}
	if !request.MarketDataProxyUser.Empty() {
		walletConfig.MarketDataProxyConfig.User = request.MarketDataProxyUser
	}
	if !request.MarketDataProxyPassword.Empty() {
		walletConfig.MarketDataProxyConfig.Password = request.MarketDataProxyPassword
	}
	if !request.MarketDataProxyUrl.Empty() {
		walletConfig.MarketDataProxyConfig.UrlOverride = request.MarketDataProxyUrl
	}
	if request.StatusProxyStageName != "" {
		walletConfig.MarketDataProxyConfig.StageName = request.StatusProxyStageName
	}
	if walletRequest.MarketDataFullDataRefreshInterval != 0 {
		walletConfig.MarketDataProxyConfig.FullDataRefreshInterval = walletRequest.MarketDataFullDataRefreshInterval
	}
	if walletRequest.MarketDataPriceRefreshInterval != 0 {
		walletConfig.MarketDataProxyConfig.PriceRefreshInterval = walletRequest.MarketDataPriceRefreshInterval
	}

	// FIXME: remove when EthRpcProxy* is integrated
	if !request.StatusProxyBlockchainUser.Empty() {
		walletConfig.StatusProxyBlockchainUser = request.StatusProxyBlockchainUser
	}
	if !request.StatusProxyBlockchainPassword.Empty() {
		walletConfig.StatusProxyBlockchainPassword = request.StatusProxyBlockchainPassword
	}

	if !request.EthRpcProxyUrl.Empty() {
		walletConfig.EthRpcProxyUrl = request.EthRpcProxyUrl
	}
	if !request.EthRpcProxyUser.Empty() {
		walletConfig.EthRpcProxyUser = request.EthRpcProxyUser
	}
	if !request.EthRpcProxyPassword.Empty() {
		walletConfig.EthRpcProxyPassword = request.EthRpcProxyPassword
	}

	return walletConfig
}

func overrideApiConfig(nodeConfig *params.NodeConfig, config *requests.APIConfig) {
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

// getMainnetRPCURL retuevrns URL of the first provider with TokenAuth from mainnet network
func getMainnetRPCURL(networks []params.Network) string {
	for _, network := range networks {
		if network.ChainID != walletcommon.EthereumMainnet {
			continue
		}
		for _, provider := range network.RpcProviders {
			if provider.AuthType == params.TokenAuth && provider.Enabled {
				return provider.GetFullURL().Reveal()
			}
		}
		break
	}
	return ""
}

func DefaultNodeConfig(installationID, keyUID string, request *requests.CreateAccount) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.RootDataDir = request.RootDataDir
	nodeConfig.LogEnabled = request.LogEnabled
	nodeConfig.LogToStderr = request.LogToStderr
	nodeConfig.LogFile = gocommon.TruncateWithDot(keyUID) + ".log"
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

	nodeConfig.WalletConfig = buildWalletConfig(&request.WalletConfig, &request.WalletSecretsConfig)

	nodeConfig.BrowsersConfig = params.BrowsersConfig{Enabled: true}
	nodeConfig.PermissionsConfig = params.PermissionsConfig{Enabled: true}

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

	nodeConfig.ShhextConfig = params.ShhextConfig{
		InstallationID:             installationID,
		MailServerConfirmations:    true,
		MaxMessageDeliveryAttempts: DefaultMaxMessageDeliveryAttempts,
		VerifyTransactionChainID:   DefaultVerifyTransactionChainID,
		DataSyncEnabled:            true,
		PFSEnabled:                 true,
	}

	if request.VerifyTransactionURL != nil {
		nodeConfig.ShhextConfig.VerifyTransactionURL = *request.VerifyTransactionURL
	} else {
		nodeConfig.ShhextConfig.VerifyTransactionURL = getMainnetRPCURL(nodeConfig.Networks)
	}

	if request.VerifyENSURL != nil {
		nodeConfig.ShhextConfig.VerifyENSURL = *request.VerifyENSURL
	} else {
		nodeConfig.ShhextConfig.VerifyENSURL = getMainnetRPCURL(nodeConfig.Networks)
	}

	if request.VerifyTransactionChainID != nil {
		nodeConfig.ShhextConfig.VerifyTransactionChainID = *request.VerifyTransactionChainID
	}

	if request.VerifyENSContractAddress != nil {
		nodeConfig.ShhextConfig.VerifyENSContractAddress = *request.VerifyENSContractAddress
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
