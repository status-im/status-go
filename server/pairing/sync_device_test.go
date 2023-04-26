package pairing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/status-im/status-go/protocol/identity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/sqlite"
)

const (
	pathWalletRoot    = "m/44'/60'/0'/0"
	pathEIP1581       = "m/43'/60'/1581'"
	pathDefaultChat   = pathEIP1581 + "/0'/0"
	pathDefaultWallet = pathWalletRoot + "/0"
	currentNetwork    = "mainnet_rpc"
	socialLinkURL     = "https://github.com/status-im"
	ensUsername       = "bob.stateofus.eth"
	ensChainID        = 1
)

var paths = []string{pathWalletRoot, pathEIP1581, pathDefaultChat, pathDefaultWallet}

func TestSyncDeviceSuite(t *testing.T) {
	suite.Run(t, new(SyncDeviceSuite))
}

type SyncDeviceSuite struct {
	suite.Suite
	password               string
	clientAsSenderTmpdir   string
	clientAsReceiverTmpdir string
}

func (s *SyncDeviceSuite) SetupTest() {
	s.password = "password"

	clientAsSenderTmpdir, err := os.MkdirTemp("", "TestPairingSyncDeviceClientAsSender")
	require.NoError(s.T(), err)
	s.clientAsSenderTmpdir = clientAsSenderTmpdir

	clientAsReceiverTmpdir, err := os.MkdirTemp("", "TestPairingSyncDeviceClientAsReceiver")
	require.NoError(s.T(), err)
	s.clientAsReceiverTmpdir = clientAsReceiverTmpdir
}

func (s *SyncDeviceSuite) TearDownTest() {
	os.RemoveAll(s.clientAsSenderTmpdir)
	os.RemoveAll(s.clientAsReceiverTmpdir)
}

func (s *SyncDeviceSuite) prepareBackendWithAccount(tmpdir string) *api.GethStatusBackend {
	backend := s.prepareBackendWithoutAccount(tmpdir)
	accountManager := backend.AccountManager()
	generator := accountManager.AccountsGenerator()
	generatedAccountInfos, err := generator.GenerateAndDeriveAddresses(12, 1, "", paths)
	require.NoError(s.T(), err)
	generatedAccountInfo := generatedAccountInfos[0]
	account := multiaccounts.Account{
		KeyUID:        generatedAccountInfo.KeyUID,
		KDFIterations: sqlite.ReducedKDFIterationsNumber,
	}
	err = accountManager.InitKeystore(filepath.Join(tmpdir, keystoreDir, account.KeyUID))
	require.NoError(s.T(), err)
	err = backend.OpenAccounts()
	require.NoError(s.T(), err)
	derivedAddresses := generatedAccountInfo.Derived
	_, err = generator.StoreDerivedAccounts(generatedAccountInfo.ID, s.password, paths)
	require.NoError(s.T(), err)

	settings, err := defaultSettings(generatedAccountInfo.GeneratedAccountInfo, derivedAddresses, nil)
	require.NoError(s.T(), err)

	account.Name = settings.Name

	nodeConfig, err := defaultNodeConfig(settings.InstallationID, account.KeyUID)
	require.NoError(s.T(), err)

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

	accounts := []*accounts.Account{walletAccount, chatAccount}
	err = backend.StartNodeWithAccountAndInitialConfig(account, s.password, *settings, nodeConfig, accounts)
	require.NoError(s.T(), err)

	return backend
}

func (s *SyncDeviceSuite) prepareBackendWithoutAccount(tmpdir string) *api.GethStatusBackend {
	backend := api.NewGethStatusBackend()
	backend.UpdateRootDataDir(tmpdir)
	return backend
}

func (s *SyncDeviceSuite) TestPairingSyncDeviceClientAsSender() {
	clientTmpDir := filepath.Join(s.clientAsSenderTmpdir, "client")
	clientBackend := s.prepareBackendWithAccount(clientTmpDir)
	serverTmpDir := filepath.Join(s.clientAsSenderTmpdir, "server")
	serverBackend := s.prepareBackendWithoutAccount(serverTmpDir)
	defer func() {
		require.NoError(s.T(), serverBackend.Logout())
		require.NoError(s.T(), clientBackend.Logout())
	}()

	err := serverBackend.AccountManager().InitKeystore(filepath.Join(serverTmpDir, keystoreDir))
	require.NoError(s.T(), err)
	err = serverBackend.OpenAccounts()
	require.NoError(s.T(), err)
	serverNodeConfig, err := defaultNodeConfig(uuid.New().String(), "")
	require.NoError(s.T(), err)
	expectedKDFIterations := 1024
	serverKeystoreDir := filepath.Join(serverTmpDir, keystoreDir)
	serverPayloadSourceConfig := &ReceiverServerConfig{
		ReceiverConfig: &ReceiverConfig{
			NodeConfig:            serverNodeConfig,
			KeystorePath:          serverKeystoreDir,
			DeviceType:            "desktop",
			KDFIterations:         expectedKDFIterations,
			SettingCurrentNetwork: currentNetwork,
		},
		ServerConfig: new(ServerConfig),
	}
	serverNodeConfig.RootDataDir = serverTmpDir
	serverConfigBytes, err := json.Marshal(serverPayloadSourceConfig)
	require.NoError(s.T(), err)
	cs, err := StartUpReceiverServer(serverBackend, string(serverConfigBytes))
	require.NoError(s.T(), err)

	// generate some data for the client
	clientBrowserAPI := clientBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	_, err = clientBrowserAPI.StoreBookmark(context.TODO(), browsers.Bookmark{
		Name: "status.im",
		URL:  "https://status.im",
	})
	require.NoError(s.T(), err)
	err = clientBackend.Messenger().SetSocialLinks(&identity.SocialLinks{{Text: identity.GithubID, URL: socialLinkURL, Clock: 1}})
	require.NoError(s.T(), err)
	err = clientBackend.StatusNode().EnsService().API().Add(context.Background(), ensChainID, ensUsername)
	require.NoError(s.T(), err)

	clientActiveAccount, err := clientBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	clientKeystorePath := filepath.Join(clientTmpDir, keystoreDir, clientActiveAccount.KeyUID)
	clientPayloadSourceConfig := SenderClientConfig{
		SenderConfig: &SenderConfig{
			KeystorePath: clientKeystorePath,
			DeviceType:   "android",
			KeyUID:       clientActiveAccount.KeyUID,
			Password:     s.password,
		},
		ClientConfig: new(ClientConfig),
	}
	clientConfigBytes, err := json.Marshal(clientPayloadSourceConfig)
	require.NoError(s.T(), err)
	err = StartUpSendingClient(clientBackend, cs, string(clientConfigBytes))
	require.NoError(s.T(), err)

	// check that the server has the same data as the client
	serverBrowserAPI := serverBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	bookmarks, err := serverBrowserAPI.GetBookmarks(context.TODO())
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(bookmarks))
	require.Equal(s.T(), "status.im", bookmarks[0].Name)
	serverSocialLink, err := serverBackend.Messenger().GetSocialLink(identity.GithubID)
	require.NoError(s.T(), err)
	require.Equal(s.T(), socialLinkURL, serverSocialLink.URL)
	uds, err := serverBackend.StatusNode().EnsService().API().GetEnsUsernames(context.Background())
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(uds))
	require.Equal(s.T(), ensUsername, uds[0].Username)
	require.Equal(s.T(), uint64(ensChainID), uds[0].ChainID)
	require.False(s.T(), uds[0].Removed)
	require.Greater(s.T(), uds[0].Clock, uint64(0))

	serverActiveAccount, err := serverBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	require.Equal(s.T(), serverActiveAccount.Name, clientActiveAccount.Name)
	require.Equal(s.T(), serverActiveAccount.KDFIterations, expectedKDFIterations)

	serverMessenger := serverBackend.Messenger()
	clientMessenger := clientBackend.Messenger()
	require.True(s.T(), serverMessenger.HasPairedDevices())
	require.True(s.T(), clientMessenger.HasPairedDevices())

	err = clientMessenger.DisableInstallation(serverNodeConfig.ShhextConfig.InstallationID)
	require.NoError(s.T(), err)
	require.False(s.T(), clientMessenger.HasPairedDevices())
	clientNodeConfig, err := clientBackend.GetNodeConfig()
	require.NoError(s.T(), err)
	err = serverMessenger.DisableInstallation(clientNodeConfig.ShhextConfig.InstallationID)
	require.NoError(s.T(), err)
	require.False(s.T(), serverMessenger.HasPairedDevices())

	// repeat local pairing, we should expect no error after receiver logged in
	cs, err = StartUpReceiverServer(serverBackend, string(serverConfigBytes))
	require.NoError(s.T(), err)
	err = StartUpSendingClient(clientBackend, cs, string(clientConfigBytes))
	require.NoError(s.T(), err)
	require.True(s.T(), clientMessenger.HasPairedDevices())
	require.True(s.T(), serverMessenger.HasPairedDevices())

	// test if it's okay when account already exist but not logged in
	require.NoError(s.T(), serverBackend.Logout())
	cs, err = StartUpReceiverServer(serverBackend, string(serverConfigBytes))
	require.NoError(s.T(), err)
	err = StartUpSendingClient(clientBackend, cs, string(clientConfigBytes))
	require.NoError(s.T(), err)
}

func (s *SyncDeviceSuite) TestPairingSyncDeviceClientAsReceiver() {
	clientTmpDir := filepath.Join(s.clientAsReceiverTmpdir, "client")
	clientBackend := s.prepareBackendWithoutAccount(clientTmpDir)

	serverTmpDir := filepath.Join(s.clientAsReceiverTmpdir, "server")
	serverBackend := s.prepareBackendWithAccount(serverTmpDir)
	defer func() {
		require.NoError(s.T(), clientBackend.Logout())
		require.NoError(s.T(), serverBackend.Logout())
	}()

	serverActiveAccount, err := serverBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	serverKeystorePath := filepath.Join(serverTmpDir, keystoreDir, serverActiveAccount.KeyUID)
	var config = &SenderServerConfig{
		SenderConfig: &SenderConfig{
			KeystorePath: serverKeystorePath,
			DeviceType:   "desktop",
			KeyUID:       serverActiveAccount.KeyUID,
			Password:     s.password,
		},
		ServerConfig: new(ServerConfig),
	}
	configBytes, err := json.Marshal(config)
	require.NoError(s.T(), err)
	cs, err := StartUpSenderServer(serverBackend, string(configBytes))
	require.NoError(s.T(), err)

	// generate some data for the server
	serverBrowserAPI := serverBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	_, err = serverBrowserAPI.StoreBookmark(context.TODO(), browsers.Bookmark{
		Name: "status.im",
		URL:  "https://status.im",
	})
	require.NoError(s.T(), err)
	err = serverBackend.Messenger().SetSocialLinks(&identity.SocialLinks{{Text: identity.GithubID, URL: socialLinkURL, Clock: 1}})
	require.NoError(s.T(), err)
	err = serverBackend.StatusNode().EnsService().API().Add(context.Background(), ensChainID, ensUsername)
	require.NoError(s.T(), err)

	err = clientBackend.AccountManager().InitKeystore(filepath.Join(clientTmpDir, keystoreDir))
	require.NoError(s.T(), err)
	err = clientBackend.OpenAccounts()
	require.NoError(s.T(), err)
	clientNodeConfig, err := defaultNodeConfig(uuid.New().String(), "")
	require.NoError(s.T(), err)
	expectedKDFIterations := 2048
	clientKeystoreDir := filepath.Join(clientTmpDir, keystoreDir)
	clientPayloadSourceConfig := ReceiverClientConfig{
		ReceiverConfig: &ReceiverConfig{
			KeystorePath:          clientKeystoreDir,
			DeviceType:            "iphone",
			KDFIterations:         expectedKDFIterations,
			NodeConfig:            clientNodeConfig,
			SettingCurrentNetwork: currentNetwork,
		},
		ClientConfig: new(ClientConfig),
	}
	clientNodeConfig.RootDataDir = clientTmpDir
	clientConfigBytes, err := json.Marshal(clientPayloadSourceConfig)
	require.NoError(s.T(), err)
	err = StartUpReceivingClient(clientBackend, cs, string(clientConfigBytes))
	require.NoError(s.T(), err)

	// check that the client has the same data as the server
	clientBrowserAPI := clientBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	bookmarks, err := clientBrowserAPI.GetBookmarks(context.TODO())
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(bookmarks))
	require.Equal(s.T(), "status.im", bookmarks[0].Name)
	clientSocialLink, err := clientBackend.Messenger().GetSocialLink(identity.GithubID)
	require.NoError(s.T(), err)
	require.Equal(s.T(), socialLinkURL, clientSocialLink.URL)
	uds, err := clientBackend.StatusNode().EnsService().API().GetEnsUsernames(context.Background())
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(uds))
	require.Equal(s.T(), ensUsername, uds[0].Username)
	require.Equal(s.T(), uint64(ensChainID), uds[0].ChainID)

	clientActiveAccount, err := clientBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	require.Equal(s.T(), serverActiveAccount.Name, clientActiveAccount.Name)
	require.Equal(s.T(), clientActiveAccount.KDFIterations, expectedKDFIterations)

	serverMessenger := serverBackend.Messenger()
	clientMessenger := clientBackend.Messenger()
	require.True(s.T(), serverMessenger.HasPairedDevices())
	require.True(s.T(), clientMessenger.HasPairedDevices())

	err = serverMessenger.DisableInstallation(clientNodeConfig.ShhextConfig.InstallationID)
	require.NoError(s.T(), err)
	require.False(s.T(), serverMessenger.HasPairedDevices())
	serverNodeConfig, err := serverBackend.GetNodeConfig()
	require.NoError(s.T(), err)
	err = clientMessenger.DisableInstallation(serverNodeConfig.ShhextConfig.InstallationID)
	require.NoError(s.T(), err)
	require.False(s.T(), clientMessenger.HasPairedDevices())

	// repeat local pairing, we should expect no error after receiver logged in
	cs, err = StartUpSenderServer(serverBackend, string(configBytes))
	require.NoError(s.T(), err)
	err = StartUpReceivingClient(clientBackend, cs, string(clientConfigBytes))
	require.NoError(s.T(), err)
	require.True(s.T(), serverMessenger.HasPairedDevices())
	require.True(s.T(), clientMessenger.HasPairedDevices())

	// test if it's okay when account already exist but not logged in
	require.NoError(s.T(), clientBackend.Logout())
	cs, err = StartUpSenderServer(serverBackend, string(configBytes))
	require.NoError(s.T(), err)
	err = StartUpReceivingClient(clientBackend, cs, string(clientConfigBytes))
	require.NoError(s.T(), err)
}

func defaultSettings(generatedAccountInfo generator.GeneratedAccountInfo, derivedAddresses map[string]generator.AccountInfo, mnemonic *string) (*settings.Settings, error) {
	chatKeyString := derivedAddresses[pathDefaultChat].PublicKey

	syncSettings := &settings.Settings{}
	syncSettings.KeyUID = generatedAccountInfo.KeyUID
	syncSettings.Address = types.HexToAddress(generatedAccountInfo.Address)
	syncSettings.WalletRootAddress = types.HexToAddress(derivedAddresses[pathWalletRoot].Address)

	// Set chat key & name
	name, err := alias.GenerateFromPublicKeyString(chatKeyString)
	if err != nil {
		return nil, err
	}
	syncSettings.Name = name
	syncSettings.PublicKey = chatKeyString

	syncSettings.DappsAddress = types.HexToAddress(derivedAddresses[pathDefaultWallet].Address)
	syncSettings.EIP1581Address = types.HexToAddress(derivedAddresses[pathEIP1581].Address)
	syncSettings.Mnemonic = mnemonic

	syncSettings.SigningPhrase = "balabala"

	syncSettings.SendPushNotifications = true
	syncSettings.InstallationID = uuid.New().String()
	syncSettings.UseMailservers = true

	syncSettings.PreviewPrivacy = true
	syncSettings.Currency = "usd"
	syncSettings.ProfilePicturesVisibility = 1
	syncSettings.LinkPreviewRequestEnabled = true

	visibleTokens := make(map[string][]string)
	visibleTokens["mainnet"] = []string{"SNT"}
	visibleTokensJSON, err := json.Marshal(visibleTokens)
	if err != nil {
		return nil, err
	}
	visibleTokenJSONRaw := json.RawMessage(visibleTokensJSON)
	syncSettings.WalletVisibleTokens = &visibleTokenJSONRaw

	networks := `[{"id":"goerli_rpc","chain-explorer-link":"https://goerli.etherscan.io/address/","name":"Goerli with upstream RPC","config":{"NetworkId":5,"DataDir":"/ethereum/goerli_rpc","UpstreamConfig":{"Enabled":true,"URL":"https://goerli-archival.gateway.pokt.network/v1/lb/3ef2018191814b7e1009b8d9"}}},{"id":"mainnet_rpc","chain-explorer-link":"https://etherscan.io/address/","name":"Mainnet with upstream RPC","config":{"NetworkId":1,"DataDir":"/ethereum/mainnet_rpc","UpstreamConfig":{"Enabled":true,"URL":"https://eth-archival.gateway.pokt.network/v1/lb/3ef2018191814b7e1009b8d9"}}}]`
	var networksRawMessage json.RawMessage = []byte(networks)
	syncSettings.Networks = &networksRawMessage
	syncSettings.CurrentNetwork = currentNetwork

	return syncSettings, nil
}

func defaultNodeConfig(installationID, keyUID string) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.NetworkID = 1
	nodeConfig.LogLevel = "ERROR"
	nodeConfig.DataDir = filepath.Join("ethereum/mainnet_rpc")
	nodeConfig.KeyStoreDir = filepath.Join(keystoreDir, keyUID)
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

	nodeConfig.WalletConfig = params.WalletConfig{Enabled: false}
	nodeConfig.LocalNotificationsConfig = params.LocalNotificationsConfig{Enabled: true}
	nodeConfig.BrowsersConfig = params.BrowsersConfig{Enabled: false}
	nodeConfig.PermissionsConfig = params.PermissionsConfig{Enabled: true}
	nodeConfig.MailserversConfig = params.MailserversConfig{Enabled: true}
	nodeConfig.EnableNTPSync = true
	nodeConfig.WakuConfig = params.WakuConfig{
		Enabled:     true,
		LightClient: true,
		MinimumPoW:  0.000001,
	}

	nodeConfig.ShhextConfig = params.ShhextConfig{
		BackupDisabledDataDir:      nodeConfig.DataDir,
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

	return nodeConfig, nil
}
