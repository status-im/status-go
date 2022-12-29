package pairing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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

const pathWalletRoot = "m/44'/60'/0'/0"
const pathEIP1581 = "m/43'/60'/1581'"
const pathDefaultChat = pathEIP1581 + "/0'/0"
const pathDefaultWallet = pathWalletRoot + "/0"

var paths = []string{pathWalletRoot, pathEIP1581, pathDefaultChat, pathDefaultWallet}

const keystoreDir = "keystore"

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

	nodeConfig, err := defaultNodeConfig(tmpdir, settings.InstallationID, account.KeyUID)
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
	}()

	err := serverBackend.AccountManager().InitKeystore(filepath.Join(serverTmpDir, keystoreDir))
	require.NoError(s.T(), err)
	err = serverBackend.OpenAccounts()
	require.NoError(s.T(), err)
	serverKeystorePath := filepath.Join(serverTmpDir, keystoreDir)
	configJSON := fmt.Sprintf(`{"KeystorePath":"%s"}`, serverKeystorePath)
	cs, err := StartUpPairingServer(serverBackend, Receiving, configJSON)
	require.NoError(s.T(), err)

	// generate some data for the client
	clientBrowserAPI := clientBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	_, err = clientBrowserAPI.StoreBookmark(context.TODO(), browsers.Bookmark{
		Name: "status.im",
		URL:  "https://status.im",
	})
	require.NoError(s.T(), err)

	activeAccount, err := clientBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	clientKeystorePath := filepath.Join(clientTmpDir, keystoreDir, activeAccount.KeyUID)
	var config = PayloadSourceConfig{
		KeystorePath: clientKeystorePath,
		KeyUID:       activeAccount.KeyUID,
		Password:     s.password,
	}
	configBytes, err := json.Marshal(config)
	require.NoError(s.T(), err)
	err = StartUpPairingClient(clientBackend, cs, string(configBytes))
	require.NoError(s.T(), err)
	require.NoError(s.T(), clientBackend.Logout())

	serverBrowserAPI := serverBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	bookmarks, err := serverBrowserAPI.GetBookmarks(context.TODO())
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(bookmarks))
	require.Equal(s.T(), "status.im", bookmarks[0].Name)
}

func (s *SyncDeviceSuite) TestPairingSyncDeviceClientAsReceiver() {
	clientTmpDir := filepath.Join(s.clientAsReceiverTmpdir, "client")
	clientBackend := s.prepareBackendWithoutAccount(clientTmpDir)

	serverTmpDir := filepath.Join(s.clientAsReceiverTmpdir, "server")
	serverBackend := s.prepareBackendWithAccount(serverTmpDir)
	defer func() {
		require.NoError(s.T(), clientBackend.Logout())
	}()

	activeAccount, err := serverBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	serverKeystorePath := filepath.Join(serverTmpDir, keystoreDir, activeAccount.KeyUID)
	var config = PayloadSourceConfig{
		KeystorePath: serverKeystorePath,
		KeyUID:       activeAccount.KeyUID,
		Password:     s.password,
	}
	configBytes, err := json.Marshal(config)
	require.NoError(s.T(), err)
	cs, err := StartUpPairingServer(serverBackend, Sending, string(configBytes))
	require.NoError(s.T(), err)

	// generate some data for the server
	serverBrowserAPI := serverBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	_, err = serverBrowserAPI.StoreBookmark(context.TODO(), browsers.Bookmark{
		Name: "status.im",
		URL:  "https://status.im",
	})
	require.NoError(s.T(), err)

	err = clientBackend.AccountManager().InitKeystore(filepath.Join(clientTmpDir, keystoreDir))
	require.NoError(s.T(), err)
	err = clientBackend.OpenAccounts()
	require.NoError(s.T(), err)
	clientKeystorePath := filepath.Join(clientTmpDir, keystoreDir)
	configJSON := fmt.Sprintf(`{"KeystorePath":"%s"}`, clientKeystorePath)
	err = StartUpPairingClient(clientBackend, cs, configJSON)
	require.NoError(s.T(), err)

	require.NoError(s.T(), serverBackend.Logout())

	clientBrowserAPI := clientBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	bookmarks, err := clientBrowserAPI.GetBookmarks(context.TODO())
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(bookmarks))
	require.Equal(s.T(), "status.im", bookmarks[0].Name)
}

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

	settings.SigningPhrase = "balabala"

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

func defaultNodeConfig(tmpdir, installationID, keyUID string) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.NetworkID = 1
	nodeConfig.LogLevel = "ERROR"
	nodeConfig.DataDir = filepath.Join(tmpdir, "ethereum/mainnet_rpc")
	nodeConfig.KeyStoreDir = filepath.Join(tmpdir, keystoreDir, keyUID)
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

	return nodeConfig, nil
}
