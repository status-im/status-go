package pairing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"

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
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	accservice "github.com/status-im/status-go/services/accounts"
	"github.com/status-im/status-go/services/browsers"
)

const (
	pathWalletRoot          = "m/44'/60'/0'/0"
	pathEIP1581             = "m/43'/60'/1581'"
	pathDefaultChat         = pathEIP1581 + "/0'/0"
	pathDefaultWallet       = pathWalletRoot + "/0"
	currentNetwork          = "mainnet_rpc"
	socialLinkURL           = "https://github.com/status-im"
	ensUsername             = "bob.stateofus.eth"
	ensChainID              = 1
	publicChatID            = "localpairtest"
	profileKeypairMnemonic  = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
	seedKeypairMnemonic     = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	profileKeypairMnemonic1 = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about about"
	seedKeypairMnemonic1    = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about abandon"
	path0                   = "m/44'/60'/0'/0/0"
	path1                   = "m/44'/60'/0'/0/1"
	expectedKDFIterations   = 1024
)

var paths = []string{pathWalletRoot, pathEIP1581, pathDefaultChat, pathDefaultWallet}

func TestSyncDeviceSuite(t *testing.T) {
	suite.Run(t, new(SyncDeviceSuite))
}

type SyncDeviceSuite struct {
	suite.Suite
	logger   *zap.Logger
	password string
	tmpdir   string
}

func (s *SyncDeviceSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	s.password = "password"
	s.tmpdir = s.T().TempDir()
}

func (s *SyncDeviceSuite) prepareBackendWithAccount(mnemonic, tmpdir string) *api.GethStatusBackend {
	backend := s.prepareBackendWithoutAccount(tmpdir)
	accountManager := backend.AccountManager()
	accGenerator := accountManager.AccountsGenerator()

	var (
		generatedAccountInfo generator.GeneratedAndDerivedAccountInfo
		err                  error
	)
	if len(mnemonic) > 0 {
		generatedAccountInfo.GeneratedAccountInfo, err = accGenerator.ImportMnemonic(mnemonic, "")
		require.NoError(s.T(), err)
		generatedAccountInfo.Derived, err = accGenerator.DeriveAddresses(generatedAccountInfo.ID, paths)
		require.NoError(s.T(), err)
	} else {
		generatedAccountInfos, err := accGenerator.GenerateAndDeriveAddresses(12, 1, "", paths)
		require.NoError(s.T(), err)
		generatedAccountInfo = generatedAccountInfos[0]
	}
	account := multiaccounts.Account{
		KeyUID:        generatedAccountInfo.KeyUID,
		KDFIterations: dbsetup.ReducedKDFIterationsNumber,
	}
	err = accountManager.InitKeystore(filepath.Join(tmpdir, keystoreDir, account.KeyUID))
	require.NoError(s.T(), err)
	err = backend.OpenAccounts()
	require.NoError(s.T(), err)
	derivedAddresses := generatedAccountInfo.Derived
	_, err = accGenerator.StoreDerivedAccounts(generatedAccountInfo.ID, s.password, paths)
	require.NoError(s.T(), err)

	settings, err := defaultSettings(generatedAccountInfo.GeneratedAccountInfo, derivedAddresses, nil)
	require.NoError(s.T(), err)

	account.Name = settings.Name

	nodeConfig, err := nodeConfigForLocalPairSync(settings.InstallationID, account.KeyUID, tmpdir)
	nodeConfig.RootDataDir = tmpdir
	require.NoError(s.T(), err)
	require.NoError(s.T(), setDefaultNodeConfig(nodeConfig))

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

	accounts := []*accounts.Account{walletAccount, chatAccount}
	err = backend.StartNodeWithAccountAndInitialConfig(account, s.password, *settings, nodeConfig, accounts)
	require.NoError(s.T(), err)
	multiaccounts, err := backend.GetAccounts()
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), multiaccounts[0].ColorHash)

	return backend
}

func (s *SyncDeviceSuite) prepareBackendWithoutAccount(tmpdir string) *api.GethStatusBackend {
	backend := api.NewGethStatusBackend()
	backend.UpdateRootDataDir(tmpdir)
	return backend
}

func (s *SyncDeviceSuite) pairAccounts(serverBackend *api.GethStatusBackend, serverDir string,
	clientBackend *api.GethStatusBackend, clientDir string) {

	// Start sender server

	serverActiveAccount, err := serverBackend.GetActiveAccount()
	require.NoError(s.T(), err)

	serverKeystorePath := filepath.Join(serverDir, keystoreDir, serverActiveAccount.KeyUID)
	serverConfig := &SenderServerConfig{
		SenderConfig: &SenderConfig{
			KeystorePath: serverKeystorePath,
			DeviceType:   "desktop",
			KeyUID:       serverActiveAccount.KeyUID,
			Password:     s.password,
		},
		ServerConfig: new(ServerConfig),
	}

	configBytes, err := json.Marshal(serverConfig)
	require.NoError(s.T(), err)

	connectionString, err := StartUpSenderServer(serverBackend, string(configBytes))
	require.NoError(s.T(), err)

	// Start receiving client

	err = clientBackend.AccountManager().InitKeystore(filepath.Join(clientDir, keystoreDir))
	require.NoError(s.T(), err)

	err = clientBackend.OpenAccounts()
	require.NoError(s.T(), err)

	clientNodeConfig, err := nodeConfigForLocalPairSync(uuid.New().String(), "", clientDir)
	require.NoError(s.T(), err)

	clientKeystoreDir := filepath.Join(clientDir, keystoreDir)
	clientPayloadSourceConfig := ReceiverClientConfig{
		ReceiverConfig: &ReceiverConfig{
			KeystorePath:          clientKeystoreDir,
			DeviceType:            "desktop",
			KDFIterations:         expectedKDFIterations,
			NodeConfig:            clientNodeConfig,
			SettingCurrentNetwork: currentNetwork,
		},
		ClientConfig: new(ClientConfig),
	}
	clientNodeConfig.RootDataDir = clientDir

	clientConfigBytes, err := json.Marshal(clientPayloadSourceConfig)
	require.NoError(s.T(), err)

	err = StartUpReceivingClient(clientBackend, connectionString, string(clientConfigBytes))
	require.NoError(s.T(), err)

	require.True(s.T(), serverBackend.Messenger().HasPairedDevices())
	require.True(s.T(), clientBackend.Messenger().HasPairedDevices())
}

func (s *SyncDeviceSuite) sendContactRequest(request *requests.SendContactRequest, messenger *protocol.Messenger) {
	senderPublicKey := common.PubkeyToHex(messenger.IdentityPublicKey())
	s.logger.Info("sendContactRequest", zap.String("sender", senderPublicKey), zap.String("receiver", request.ID))

	resp, err := messenger.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}

func (s *SyncDeviceSuite) receiveContactRequest(messageText string, messenger *protocol.Messenger) *common.Message {
	receiverPublicKey := types.EncodeHex(crypto.FromECDSAPub(messenger.IdentityPublicKey()))
	s.logger.Info("receiveContactRequest", zap.String("receiver", receiverPublicKey))

	// Wait for the message to reach its destination
	resp, err := protocol.WaitOnMessengerResponse(
		messenger,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) == 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().NotNil(resp)

	contactRequest := protocol.FindFirstByContentType(resp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	s.Require().NotNil(contactRequest)

	return contactRequest
}

func (s *SyncDeviceSuite) acceptContactRequest(contactRequest *common.Message, sender *protocol.Messenger, receiver *protocol.Messenger) {
	senderPublicKey := types.EncodeHex(crypto.FromECDSAPub(sender.IdentityPublicKey()))
	receiverPublicKey := types.EncodeHex(crypto.FromECDSAPub(receiver.IdentityPublicKey()))
	s.logger.Info("acceptContactRequest", zap.String("sender", senderPublicKey), zap.String("receiver", receiverPublicKey))

	_, err := receiver.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(contactRequest.ID)})
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err := protocol.WaitOnMessengerResponse(
		sender,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) == 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}

func (s *SyncDeviceSuite) checkMutualContact(backend *api.GethStatusBackend, contactPublicKey string) {
	messenger := backend.Messenger()
	contacts := messenger.MutualContacts()
	s.Require().Len(contacts, 1)
	contact := contacts[0]
	s.Require().Equal(contactPublicKey, contact.ID)
	s.Require().Equal(protocol.ContactRequestStateSent, contact.ContactRequestLocalState)
	s.Require().Equal(protocol.ContactRequestStateReceived, contact.ContactRequestRemoteState)
	s.Require().NotNil(contact.DisplayName)
}

func (s *SyncDeviceSuite) TestPairingSyncDeviceClientAsSender() {
	clientTmpDir := filepath.Join(s.tmpdir, "client")
	clientBackend := s.prepareBackendWithAccount("", clientTmpDir)
	serverTmpDir := filepath.Join(s.tmpdir, "server")
	serverBackend := s.prepareBackendWithoutAccount(serverTmpDir)
	defer func() {
		require.NoError(s.T(), serverBackend.Logout())
		require.NoError(s.T(), clientBackend.Logout())
	}()
	ctx := context.TODO()

	err := serverBackend.AccountManager().InitKeystore(filepath.Join(serverTmpDir, keystoreDir))
	require.NoError(s.T(), err)
	err = serverBackend.OpenAccounts()
	require.NoError(s.T(), err)
	serverNodeConfig, err := nodeConfigForLocalPairSync(uuid.New().String(), "", serverTmpDir)
	require.NoError(s.T(), err)
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
	// generate bookmark
	clientBrowserAPI := clientBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	_, err = clientBrowserAPI.StoreBookmark(ctx, browsers.Bookmark{
		Name: "status.im",
		URL:  "https://status.im",
	})
	require.NoError(s.T(), err)
	// generate social link
	socialLinksToAdd := identity.SocialLinks{{Text: identity.GithubID, URL: socialLinkURL}}
	err = clientBackend.Messenger().AddOrReplaceSocialLinks(socialLinksToAdd)
	require.NoError(s.T(), err)
	// generate ens username
	err = clientBackend.StatusNode().EnsService().API().Add(ctx, ensChainID, ensUsername)
	require.NoError(s.T(), err)
	// generate profile showcase preferences
	profileShowcasePreferences := protocol.DummyProfileShowcasePreferences(false)
	err = clientBackend.Messenger().SetProfileShowcasePreferences(profileShowcasePreferences, false)
	require.NoError(s.T(), err)

	// startup sending client
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
	bookmarks, err := serverBrowserAPI.GetBookmarks(ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(bookmarks))
	require.Equal(s.T(), "status.im", bookmarks[0].Name)
	serverSocialLinks, err := serverBackend.Messenger().GetSocialLinks()
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(serverSocialLinks))
	require.True(s.T(), socialLinksToAdd.Equal(serverSocialLinks))
	uds, err := serverBackend.StatusNode().EnsService().API().GetEnsUsernames(ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(uds))
	require.Equal(s.T(), ensUsername, uds[0].Username)
	require.Equal(s.T(), uint64(ensChainID), uds[0].ChainID)
	require.False(s.T(), uds[0].Removed)
	require.Greater(s.T(), uds[0].Clock, uint64(0))
	serverProfileShowcasePreferences, err := serverBackend.Messenger().GetProfileShowcasePreferences()
	require.NoError(s.T(), err)
	require.True(s.T(), reflect.DeepEqual(profileShowcasePreferences, serverProfileShowcasePreferences))

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
	clientTmpDir := filepath.Join(s.tmpdir, "client")
	clientBackend := s.prepareBackendWithoutAccount(clientTmpDir)
	ctx := context.TODO()

	serverTmpDir := filepath.Join(s.tmpdir, "server")
	serverBackend := s.prepareBackendWithAccount("", serverTmpDir)
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
	// generate bookmark
	serverBrowserAPI := serverBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	_, err = serverBrowserAPI.StoreBookmark(ctx, browsers.Bookmark{
		Name: "status.im",
		URL:  "https://status.im",
	})
	require.NoError(s.T(), err)

	// generate social link
	serverMessenger := serverBackend.Messenger()
	socialLinksToAdd := identity.SocialLinks{{Text: identity.GithubID, URL: socialLinkURL}}
	err = serverMessenger.AddOrReplaceSocialLinks(socialLinksToAdd)
	require.NoError(s.T(), err)
	// generate ens username
	err = serverBackend.StatusNode().EnsService().API().Add(ctx, ensChainID, ensUsername)
	require.NoError(s.T(), err)

	// generate profile showcase preferences
	profileShowcasePreferences := protocol.DummyProfileShowcasePreferences(false)
	err = serverMessenger.SetProfileShowcasePreferences(profileShowcasePreferences, false)
	require.NoError(s.T(), err)

	// generate local deleted message
	_, err = serverMessenger.CreatePublicChat(&requests.CreatePublicChat{ID: publicChatID})
	require.NoError(s.T(), err)
	serverChat := serverMessenger.Chat(publicChatID)
	serverMessage := buildTestMessage(serverChat)
	serverMessengerResponse, err := serverMessenger.SendChatMessage(ctx, serverMessage)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(serverMessengerResponse.Messages()))
	serverMessageID := serverMessengerResponse.Messages()[0].ID
	_, err = serverMessenger.DeleteMessageForMeAndSync(ctx, publicChatID, serverMessageID)
	require.NoError(s.T(), err)

	err = clientBackend.AccountManager().InitKeystore(filepath.Join(clientTmpDir, keystoreDir))
	require.NoError(s.T(), err)
	err = clientBackend.OpenAccounts()
	require.NoError(s.T(), err)
	clientNodeConfig, err := nodeConfigForLocalPairSync(uuid.New().String(), "", clientTmpDir)
	require.NoError(s.T(), err)
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
	clientMessenger := clientBackend.Messenger()
	clientBrowserAPI := clientBackend.StatusNode().BrowserService().APIs()[0].Service.(*browsers.API)
	bookmarks, err := clientBrowserAPI.GetBookmarks(ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(bookmarks))
	require.Equal(s.T(), "status.im", bookmarks[0].Name)

	clientSocialLinks, err := clientMessenger.GetSocialLinks()
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(clientSocialLinks))
	require.True(s.T(), socialLinksToAdd.Equal(clientSocialLinks))

	clientProfileShowcasePreferences, err := clientMessenger.GetProfileShowcasePreferences()
	require.NoError(s.T(), err)
	require.True(s.T(), reflect.DeepEqual(profileShowcasePreferences, clientProfileShowcasePreferences))

	uds, err := clientBackend.StatusNode().EnsService().API().GetEnsUsernames(ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(uds))
	require.Equal(s.T(), ensUsername, uds[0].Username)
	require.Equal(s.T(), uint64(ensChainID), uds[0].ChainID)
	deleteForMeMessages, err := clientMessenger.GetDeleteForMeMessages()
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(deleteForMeMessages))

	clientActiveAccount, err := clientBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	require.Equal(s.T(), serverActiveAccount.Name, clientActiveAccount.Name)
	require.Equal(s.T(), clientActiveAccount.KDFIterations, expectedKDFIterations)

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

func (s *SyncDeviceSuite) TestPairingThreeDevices() {
	bobTmpDir := filepath.Join(s.tmpdir, "bob")
	bobBackend := s.prepareBackendWithAccount("", bobTmpDir)
	bobMessenger := bobBackend.Messenger()
	_, err := bobMessenger.Start()
	s.Require().NoError(err)

	alice1TmpDir := filepath.Join(s.tmpdir, "alice1")
	alice1Backend := s.prepareBackendWithAccount("", alice1TmpDir)
	alice1Messenger := alice1Backend.Messenger()
	_, err = alice1Messenger.Start()
	s.Require().NoError(err)

	alice2TmpDir := filepath.Join(s.tmpdir, "alice2")
	alice2Backend := s.prepareBackendWithoutAccount(alice2TmpDir)

	alice3TmpDir := filepath.Join(s.tmpdir, "alice3")
	alice3Backend := s.prepareBackendWithoutAccount(alice3TmpDir)

	defer func() {
		require.NoError(s.T(), bobBackend.Logout())
		require.NoError(s.T(), alice1Backend.Logout())
		require.NoError(s.T(), alice2Backend.Logout())
		require.NoError(s.T(), alice3Backend.Logout())
	}()

	// Make Alice and Bob mutual contacts
	bobPublicKey := bobMessenger.GetSelfContact().ID
	request := &requests.SendContactRequest{
		ID:      bobPublicKey,
		Message: protocol.RandomLettersString(5),
	}
	s.sendContactRequest(request, alice1Messenger)
	contactRequest := s.receiveContactRequest(request.Message, bobMessenger)
	s.acceptContactRequest(contactRequest, alice1Messenger, bobMessenger)
	s.checkMutualContact(alice1Backend, bobPublicKey)

	// We shouldn't sync ourselves as a contact, so we check there's only Bob
	// https://github.com/status-im/status-go/issues/3667
	s.Require().Equal(1, len(alice1Backend.Messenger().Contacts()))

	// Pair alice-1 <-> alice-2
	s.logger.Info("pairing Alice-1 and Alice-2")
	s.pairAccounts(alice1Backend, alice1TmpDir, alice2Backend, alice2TmpDir)

	s.checkMutualContact(alice2Backend, bobPublicKey)
	s.Require().Equal(1, len(alice2Backend.Messenger().Contacts()))

	// Pair Alice-2 <-> ALice-3
	s.logger.Info("pairing Alice-2 and Alice-3")
	s.pairAccounts(alice2Backend, alice2TmpDir, alice3Backend, alice3TmpDir)

	s.checkMutualContact(alice3Backend, bobPublicKey)
	s.Require().Equal(1, len(alice3Backend.Messenger().Contacts()))
}

func (s *SyncDeviceSuite) createUser(name string) (*api.GethStatusBackend, string) {
	tmpDir := filepath.Join(s.tmpdir, name)
	backend := s.prepareBackendWithAccount("", tmpDir)
	_, err := backend.Messenger().Start()
	s.Require().NoError(err)
	return backend, tmpDir
}

func (s *SyncDeviceSuite) TestPairPendingContactRequest() {
	bobBackend, _ := s.createUser("bob")
	defer func() {
		s.Require().NoError(bobBackend.Logout())
	}()

	alice1Backend, alice1TmpDir := s.createUser("alice1")
	defer func() {
		s.Require().NoError(alice1Backend.Logout())
	}()

	// Create a pending CR from alice to bob
	bobPublicKey := bobBackend.Messenger().IdentityPublicKeyString()
	alicePublicKey := alice1Backend.Messenger().IdentityPublicKeyString()
	request := &requests.SendContactRequest{
		ID:      alicePublicKey,
		Message: protocol.RandomLettersString(5),
	}
	s.sendContactRequest(request, bobBackend.Messenger())
	contactRequest := s.receiveContactRequest(request.Message, alice1Backend.Messenger())
	s.Require().Equal(request.Message, contactRequest.Text)

	alice2TmpDir := filepath.Join(s.tmpdir, "alice2")
	alice2Backend := s.prepareBackendWithoutAccount(alice2TmpDir)
	defer func() {
		s.Require().NoError(alice2Backend.Logout())
	}()

	// Pair alice-1 <-> alice-2
	s.logger.Info("pairing Alice-1 and Alice-2")
	s.pairAccounts(alice1Backend, alice1TmpDir, alice2Backend, alice2TmpDir)

	s.logger.Debug("public keys",
		zap.String("alice", alice1Backend.Messenger().IdentityPublicKeyString()),
		zap.String("bob", bobBackend.Messenger().IdentityPublicKeyString()),
	)

	ensurePendingContact := func(m *protocol.Messenger) {
		contacts := m.Contacts()
		s.Require().Len(contacts, 1)

		c := contacts[0]
		s.Require().Equal(bobPublicKey, c.ID)
		s.Require().Equal(protocol.ContactRequestStateReceived, c.ContactRequestRemoteState)
		s.Require().Equal(protocol.ContactRequestStateNone, c.ContactRequestLocalState)

		acRequest := protocol.ActivityCenterNotificationsRequest{
			ActivityTypes: []protocol.ActivityCenterType{
				protocol.ActivityCenterNotificationTypeContactRequest,
			},
			ReadType: protocol.ActivityCenterQueryParamsReadAll,
			Limit:    10,
		}
		r, err := m.ActivityCenterNotifications(acRequest)
		s.Require().NoError(err)
		s.Require().Len(r.Notifications, 1)
	}

	// Ensure both devices have the pending Bob contact
	ensurePendingContact(alice1Backend.Messenger())
	ensurePendingContact(alice2Backend.Messenger())
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

	networks := `[{"id":"goerli_rpc","chain-explorer-link":"https://goerli.etherscan.io/address/","name":"Goerli with upstream RPC","config":{"NetworkId":5,"DataDir":"/ethereum/goerli_rpc","UpstreamConfig":{"Enabled":true,"URL":"https://goerli-archival.rpc.grove.city/v1/3ef2018191814b7e1009b8d9"}}},{"id":"mainnet_rpc","chain-explorer-link":"https://etherscan.io/address/","name":"Mainnet with upstream RPC","config":{"NetworkId":1,"DataDir":"/ethereum/mainnet_rpc","UpstreamConfig":{"Enabled":true,"URL":"https://eth-archival.rpc.grove.city/v1/3ef2018191814b7e1009b8d9"}}}]`
	var networksRawMessage json.RawMessage = []byte(networks)
	syncSettings.Networks = &networksRawMessage
	syncSettings.CurrentNetwork = currentNetwork

	return syncSettings, nil
}

func nodeConfigForLocalPairSync(installationID, keyUID, tmpDir string) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.LogEnabled = true
	nodeConfig.LogLevel = "DEBUG"
	nodeConfig.LogDir = tmpDir
	nodeConfig.KeyStoreDir = filepath.Join(keystoreDir, keyUID)
	nodeConfig.KeycardPairingDataFile = filepath.Join("keycard", "pairings.json")
	nodeConfig.ShhextConfig = params.ShhextConfig{
		InstallationID: installationID,
	}

	// need specify cluster config here, otherwise TestPairingThreeDevices will fail due to no messages(CR) received
	// TODO(frank) need to figure out why above happen
	clusterConfig, err := params.LoadClusterConfigFromFleet(params.FleetProd)
	if err != nil {
		return nil, err
	}
	nodeConfig.ClusterConfig = *clusterConfig

	return nodeConfig, nil
}

type testTimeSource struct{}

func (t *testTimeSource) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix())
}

func buildTestMessage(chat *protocol.Chat) *common.Message {
	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := common.NewMessage()
	message.Text = "text-input-message"
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
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

	return message
}

func (s *SyncDeviceSuite) getSeedPhraseKeypairForTest(backend *api.GethStatusBackend, mnemonic string, server bool) *accounts.Keypair {
	generatedAccount, err := backend.AccountManager().AccountsGenerator().ImportMnemonic(mnemonic, "")
	require.NoError(s.T(), err)
	generatedDerivedAccs, err := backend.AccountManager().AccountsGenerator().DeriveAddresses(generatedAccount.ID, []string{path0, path1})
	require.NoError(s.T(), err)

	seedPhraseKp := &accounts.Keypair{
		KeyUID:      generatedAccount.KeyUID,
		Name:        "SeedPhraseImported",
		Type:        accounts.KeypairTypeSeed,
		DerivedFrom: generatedAccount.Address,
	}
	i := 0
	for path, ga := range generatedDerivedAccs {
		acc := &accounts.Account{
			Address:   types.HexToAddress(ga.Address),
			KeyUID:    generatedAccount.KeyUID,
			Wallet:    false,
			Chat:      false,
			Type:      accounts.AccountTypeSeed,
			Path:      path,
			PublicKey: types.HexBytes(ga.PublicKey),
			Name:      fmt.Sprintf("Acc_%d", i),
			Operable:  accounts.AccountFullyOperable,
			Emoji:     fmt.Sprintf("Emoji_%d", i),
			ColorID:   "blue",
		}
		if !server {
			acc.Operable = accounts.AccountNonOperable
		}
		seedPhraseKp.Accounts = append(seedPhraseKp.Accounts, acc)
		i++
	}

	return seedPhraseKp
}

func containsKeystoreFile(directory, key string) bool {
	files, err := os.ReadDir(directory)
	if err != nil {
		return false
	}

	for _, file := range files {
		if strings.Contains(file.Name(), strings.ToLower(key)) {
			return true
		}
	}
	return false
}

func (s *SyncDeviceSuite) TestTransferringKeystoreFiles() {
	ctx := context.TODO()

	serverTmpDir := filepath.Join(s.tmpdir, "server")
	serverBackend := s.prepareBackendWithAccount(profileKeypairMnemonic, serverTmpDir)

	clientTmpDir := filepath.Join(s.tmpdir, "client")
	clientBackend := s.prepareBackendWithAccount(profileKeypairMnemonic, clientTmpDir)
	defer func() {
		require.NoError(s.T(), clientBackend.Logout())
		require.NoError(s.T(), serverBackend.Logout())
	}()

	serverBackend.Messenger().SetLocalPairing(true)
	clientBackend.Messenger().SetLocalPairing(true)

	serverActiveAccount, err := serverBackend.GetActiveAccount()
	require.NoError(s.T(), err)

	clientActiveAccount, err := clientBackend.GetActiveAccount()
	require.NoError(s.T(), err)

	require.True(s.T(), serverActiveAccount.KeyUID == clientActiveAccount.KeyUID)

	serverSeedPhraseKp := s.getSeedPhraseKeypairForTest(serverBackend, seedKeypairMnemonic, true)
	serverAccountsAPI := serverBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)
	err = serverAccountsAPI.ImportMnemonic(ctx, seedKeypairMnemonic, s.password)
	require.NoError(s.T(), err, "importing mnemonic for new keypair on server")
	err = serverAccountsAPI.AddKeypair(ctx, s.password, serverSeedPhraseKp)
	require.NoError(s.T(), err, "saving seed phrase keypair on server with keystore files created")

	clientSeedPhraseKp := s.getSeedPhraseKeypairForTest(serverBackend, seedKeypairMnemonic, true)
	clientAccountsAPI := clientBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)
	err = clientAccountsAPI.SaveKeypair(ctx, clientSeedPhraseKp)
	require.NoError(s.T(), err, "saving seed phrase keypair on client without keystore files")

	// check server - server should contain keystore files for imported seed phrase
	serverKeystorePath := filepath.Join(serverTmpDir, keystoreDir, serverActiveAccount.KeyUID)
	require.True(s.T(), containsKeystoreFile(serverKeystorePath, serverSeedPhraseKp.DerivedFrom[2:]))
	for _, acc := range serverSeedPhraseKp.Accounts {
		require.True(s.T(), containsKeystoreFile(serverKeystorePath, acc.Address.String()[2:]))
	}

	// check client - client should not contain keystore files for imported seed phrase
	clientKeystorePath := filepath.Join(clientTmpDir, keystoreDir, clientActiveAccount.KeyUID)
	require.False(s.T(), containsKeystoreFile(clientKeystorePath, clientSeedPhraseKp.DerivedFrom[2:]))
	for _, acc := range clientSeedPhraseKp.Accounts {
		require.False(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}

	// prepare sender
	var config = KeystoreFilesSenderServerConfig{
		SenderConfig: &KeystoreFilesSenderConfig{
			KeystoreFilesConfig: KeystoreFilesConfig{
				KeystorePath:   serverKeystorePath,
				LoggedInKeyUID: serverActiveAccount.KeyUID,
				Password:       s.password,
			},
			KeypairsToExport: []string{serverSeedPhraseKp.KeyUID},
		},
		ServerConfig: new(ServerConfig),
	}
	configBytes, err := json.Marshal(config)
	require.NoError(s.T(), err)
	cs, err := StartUpKeystoreFilesSenderServer(serverBackend, string(configBytes))
	require.NoError(s.T(), err)

	// prepare receiver
	clientPayloadSourceConfig := KeystoreFilesReceiverClientConfig{
		ReceiverConfig: &KeystoreFilesReceiverConfig{
			KeystoreFilesConfig: KeystoreFilesConfig{
				KeystorePath:   clientKeystorePath,
				LoggedInKeyUID: clientActiveAccount.KeyUID,
				Password:       s.password,
			},
			KeypairsToImport: []string{serverSeedPhraseKp.KeyUID},
		},
		ClientConfig: new(ClientConfig),
	}
	clientConfigBytes, err := json.Marshal(clientPayloadSourceConfig)
	require.NoError(s.T(), err)
	err = StartUpKeystoreFilesReceivingClient(clientBackend, cs, string(clientConfigBytes))
	require.NoError(s.T(), err)

	// check client - client should contain keystore files for imported seed phrase
	accountManager := clientBackend.AccountManager()
	accGenerator := accountManager.AccountsGenerator()
	require.True(s.T(), containsKeystoreFile(clientKeystorePath, clientSeedPhraseKp.DerivedFrom[2:]))
	for _, acc := range clientSeedPhraseKp.Accounts {
		require.True(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}

	// reinit keystore on client
	require.NoError(s.T(), accountManager.InitKeystore(clientKeystorePath))

	// check keystore on client
	genAccInfo, err := accGenerator.LoadAccount(clientSeedPhraseKp.DerivedFrom, s.password)
	require.NoError(s.T(), err)
	require.Equal(s.T(), clientSeedPhraseKp.KeyUID, genAccInfo.KeyUID)
	for _, acc := range clientSeedPhraseKp.Accounts {
		genAccInfo, err := accGenerator.LoadAccount(acc.Address.String(), s.password)
		require.NoError(s.T(), err)
		require.Equal(s.T(), acc.Address.String(), genAccInfo.Address)
	}
}

func (s *SyncDeviceSuite) TestTransferringKeystoreFilesAfterStopUisngKeycard() {
	s.T().Skip("flaky test")

	ctx := context.TODO()

	// Prepare server
	serverTmpDir := filepath.Join(s.tmpdir, "server")
	serverBackend := s.prepareBackendWithAccount(profileKeypairMnemonic1, serverTmpDir)
	serverMessenger := serverBackend.Messenger()
	serverAccountsAPI := serverBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)

	// Prepare client
	clientTmpDir := filepath.Join(s.tmpdir, "client")
	clientBackend := s.prepareBackendWithAccount(profileKeypairMnemonic1, clientTmpDir)
	clientMessenger := clientBackend.Messenger()
	clientAccountsAPI := clientBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)

	defer func() {
		require.NoError(s.T(), clientBackend.Logout())
		require.NoError(s.T(), serverBackend.Logout())
	}()

	// Pair server and client
	im1 := &multidevice.InstallationMetadata{
		Name:       "client-device",
		DeviceType: "client-device-type",
	}
	settings, err := clientBackend.GetSettings()
	s.Require().NoError(err)
	err = clientMessenger.SetInstallationMetadata(settings.InstallationID, im1)
	s.Require().NoError(err)
	response, err := clientMessenger.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	response, err = protocol.WaitOnMessengerResponse(
		serverMessenger,
		func(r *protocol.MessengerResponse) bool {
			for _, i := range r.Installations {
				if i.ID == settings.InstallationID {
					return true
				}
			}
			return false
		},
		"installation not received",
	)

	s.Require().NoError(err)

	found := false
	for _, i := range response.Installations {
		found = i.ID == settings.InstallationID &&
			i.InstallationMetadata != nil &&
			i.InstallationMetadata.Name == im1.Name &&
			i.InstallationMetadata.DeviceType == im1.DeviceType
		if found {
			break
		}
	}
	s.Require().True(found)

	err = serverMessenger.EnableInstallation(settings.InstallationID)
	s.Require().NoError(err)

	// Check if the logged in account is the same on server and client
	serverActiveAccount, err := serverBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	clientActiveAccount, err := clientBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	require.True(s.T(), serverActiveAccount.KeyUID == clientActiveAccount.KeyUID)

	//////////////////////////////////////////////////////////////////////////////
	// From this point this test is trying to simulate the following scenario:
	// - add a new seed phrase keypair on server
	// - sync it to client
	// - convert it to a keycard keypair on server
	// - sync it to client
	// - stop using keycard on server
	// - sync it to client
	// - try to transfer keystore files from server to client
	//////////////////////////////////////////////////////////////////////////////

	//////////////////////////////////////////////////////////////////////////////
	// Add new seed phrase keypair to server and sync it to client
	//////////////////////////////////////////////////////////////////////////////
	serverSeedPhraseKp := s.getSeedPhraseKeypairForTest(serverBackend, seedKeypairMnemonic1, true)
	err = serverAccountsAPI.ImportMnemonic(ctx, seedKeypairMnemonic1, s.password)
	require.NoError(s.T(), err, "importing mnemonic for new keypair on server")
	err = serverAccountsAPI.AddKeypair(ctx, s.password, serverSeedPhraseKp)
	require.NoError(s.T(), err, "saving seed phrase keypair on server with keystore files created")

	// Wait for sync messages to be received on client
	err = tt.RetryWithBackOff(func() error {
		response, err := clientMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		for _, kp := range response.Keypairs {
			if kp.KeyUID == serverSeedPhraseKp.KeyUID {
				return nil
			}
		}

		return errors.New("no sync keypair received")
	})
	s.Require().NoError(err)

	// Check if the keypair saved on client is the same as the one on server
	serverKp, err := serverAccountsAPI.GetKeypairByKeyUID(ctx, serverSeedPhraseKp.KeyUID)
	s.Require().NoError(err)
	clientKp, err := clientAccountsAPI.GetKeypairByKeyUID(ctx, serverSeedPhraseKp.KeyUID)
	s.Require().NoError(err)

	s.Require().True(serverKp.KeyUID == clientKp.KeyUID &&
		serverKp.Name == clientKp.Name &&
		serverKp.Type == clientKp.Type &&
		serverKp.DerivedFrom == clientKp.DerivedFrom &&
		serverKp.LastUsedDerivationIndex == clientKp.LastUsedDerivationIndex &&
		serverKp.Clock == clientKp.Clock &&
		len(serverKp.Accounts) == len(clientKp.Accounts) &&
		len(serverKp.Keycards) == len(clientKp.Keycards))

	// Check server - server should contain keystore files for imported seed phrase
	serverKeystorePath := filepath.Join(serverTmpDir, keystoreDir, serverActiveAccount.KeyUID)
	require.True(s.T(), containsKeystoreFile(serverKeystorePath, serverKp.DerivedFrom[2:]))
	for _, acc := range serverKp.Accounts {
		require.True(s.T(), containsKeystoreFile(serverKeystorePath, acc.Address.String()[2:]))
	}

	// Check client - client should not contain keystore files for imported seed phrase
	clientKeystorePath := filepath.Join(clientTmpDir, keystoreDir, clientActiveAccount.KeyUID)
	require.False(s.T(), containsKeystoreFile(clientKeystorePath, clientKp.DerivedFrom[2:]))
	for _, acc := range clientKp.Accounts {
		require.False(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}

	//////////////////////////////////////////////////////////////////////////////
	// Convert it to a keycard keypair on server and sync it to client
	//////////////////////////////////////////////////////////////////////////////
	err = serverAccountsAPI.SaveOrUpdateKeycard(ctx, &accounts.Keycard{
		KeycardUID:        "1234",
		KeycardName:       "new-keycard",
		KeyUID:            serverKp.KeyUID,
		AccountsAddresses: []types.Address{serverKp.Accounts[0].Address, serverKp.Accounts[1].Address},
	}, false)
	s.Require().NoError(err)

	// Wait for sync messages to be received on client
	err = tt.RetryWithBackOff(func() error {
		response, err := clientMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		for _, kp := range response.Keypairs {
			if kp.KeyUID == serverKp.KeyUID {
				return nil
			}
		}
		return errors.New("no sync keypair received")
	})
	s.Require().NoError(err)

	// Check if the keypair saved on client is the same as the one on server
	serverKp, err = serverAccountsAPI.GetKeypairByKeyUID(ctx, serverSeedPhraseKp.KeyUID)
	s.Require().NoError(err)
	clientKp, err = clientAccountsAPI.GetKeypairByKeyUID(ctx, serverSeedPhraseKp.KeyUID)
	s.Require().NoError(err)

	s.Require().True(serverKp.KeyUID == clientKp.KeyUID &&
		serverKp.Name == clientKp.Name &&
		serverKp.Type == clientKp.Type &&
		serverKp.DerivedFrom == clientKp.DerivedFrom &&
		serverKp.LastUsedDerivationIndex == clientKp.LastUsedDerivationIndex &&
		serverKp.Clock == clientKp.Clock &&
		len(serverKp.Accounts) == len(clientKp.Accounts) &&
		len(serverKp.Keycards) == len(clientKp.Keycards) &&
		len(serverKp.Keycards) == 1)

	// Check server - server should not contain keystore files for imported seed phrase
	require.False(s.T(), containsKeystoreFile(serverKeystorePath, serverKp.DerivedFrom[2:]))
	for _, acc := range serverKp.Accounts {
		require.False(s.T(), containsKeystoreFile(serverKeystorePath, acc.Address.String()[2:]))
	}

	// Check client - client should not contain keystore files for imported seed phrase
	require.False(s.T(), containsKeystoreFile(clientKeystorePath, clientKp.DerivedFrom[2:]))
	for _, acc := range clientKp.Accounts {
		require.False(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}

	//////////////////////////////////////////////////////////////////////////////
	// Stop using keycard on server and sync it to client
	//////////////////////////////////////////////////////////////////////////////
	err = serverAccountsAPI.MigrateNonProfileKeycardKeypairToApp(ctx, seedKeypairMnemonic1, s.password)
	s.Require().NoError(err)

	// Wait for sync messages to be received on client
	err = tt.RetryWithBackOff(func() error {
		response, err := clientMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		for _, kp := range response.Keypairs {
			if kp.KeyUID == serverKp.KeyUID {
				return nil
			}
		}
		return errors.New("no sync keypair received")
	})
	s.Require().NoError(err)

	// Check if the keypair saved on client is the same as the one on server
	serverKp, err = serverAccountsAPI.GetKeypairByKeyUID(ctx, serverSeedPhraseKp.KeyUID)
	s.Require().NoError(err)
	clientKp, err = clientAccountsAPI.GetKeypairByKeyUID(ctx, serverSeedPhraseKp.KeyUID)
	s.Require().NoError(err)

	s.Require().True(serverKp.KeyUID == clientKp.KeyUID &&
		serverKp.Name == clientKp.Name &&
		serverKp.Type == clientKp.Type &&
		serverKp.DerivedFrom == clientKp.DerivedFrom &&
		serverKp.LastUsedDerivationIndex == clientKp.LastUsedDerivationIndex &&
		serverKp.Clock == clientKp.Clock &&
		len(serverKp.Accounts) == len(clientKp.Accounts) &&
		len(serverKp.Keycards) == len(clientKp.Keycards) &&
		len(serverKp.Keycards) == 0)

	// Check server - server should contain keystore files for imported seed phrase
	require.True(s.T(), containsKeystoreFile(serverKeystorePath, serverKp.DerivedFrom[2:]))
	for _, acc := range serverKp.Accounts {
		require.True(s.T(), containsKeystoreFile(serverKeystorePath, acc.Address.String()[2:]))
	}

	// Check client - client should not contain keystore files for imported seed phrase
	require.False(s.T(), containsKeystoreFile(clientKeystorePath, clientKp.DerivedFrom[2:]))
	for _, acc := range clientKp.Accounts {
		require.False(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}

	//////////////////////////////////////////////////////////////////////////////
	// Try to transfer keystore files from server to client
	//////////////////////////////////////////////////////////////////////////////

	serverMessenger.SetLocalPairing(true)
	clientMessenger.SetLocalPairing(true)

	// prepare sender
	var config = KeystoreFilesSenderServerConfig{
		SenderConfig: &KeystoreFilesSenderConfig{
			KeystoreFilesConfig: KeystoreFilesConfig{
				KeystorePath:   serverKeystorePath,
				LoggedInKeyUID: serverActiveAccount.KeyUID,
				Password:       s.password,
			},
			KeypairsToExport: []string{serverKp.KeyUID},
		},
		ServerConfig: new(ServerConfig),
	}
	configBytes, err := json.Marshal(config)
	require.NoError(s.T(), err)
	cs, err := StartUpKeystoreFilesSenderServer(serverBackend, string(configBytes))
	require.NoError(s.T(), err)

	// prepare receiver
	clientPayloadSourceConfig := KeystoreFilesReceiverClientConfig{
		ReceiverConfig: &KeystoreFilesReceiverConfig{
			KeystoreFilesConfig: KeystoreFilesConfig{
				KeystorePath:   clientKeystorePath,
				LoggedInKeyUID: clientActiveAccount.KeyUID,
				Password:       s.password,
			},
			KeypairsToImport: []string{clientKp.KeyUID},
		},
		ClientConfig: new(ClientConfig),
	}
	clientConfigBytes, err := json.Marshal(clientPayloadSourceConfig)
	require.NoError(s.T(), err)
	err = StartUpKeystoreFilesReceivingClient(clientBackend, cs, string(clientConfigBytes))
	require.NoError(s.T(), err)

	// Check server - server should contain keystore files for imported seed phrase
	require.True(s.T(), containsKeystoreFile(serverKeystorePath, serverKp.DerivedFrom[2:]))
	for _, acc := range serverKp.Accounts {
		require.True(s.T(), containsKeystoreFile(serverKeystorePath, acc.Address.String()[2:]))
	}

	// Check client - client should contain keystore files for imported seed phrase
	require.True(s.T(), containsKeystoreFile(clientKeystorePath, clientKp.DerivedFrom[2:]))
	for _, acc := range clientKp.Accounts {
		require.True(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}
}

func (s *SyncDeviceSuite) TestPreventLoggedInAccountLocalPairingClientAsReceiver() {
	clientTmpDir := filepath.Join(s.tmpdir, "client")
	clientBackend := s.prepareBackendWithAccount("", clientTmpDir)
	serverTmpDir := filepath.Join(s.tmpdir, "server")
	serverBackend := s.prepareBackendWithAccount("", serverTmpDir)
	defer func() {
		s.NoError(serverBackend.Logout())
		s.NoError(clientBackend.Logout())
	}()

	serverActiveAccount, err := serverBackend.GetActiveAccount()
	s.NoError(err)
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
	s.NoError(err)
	cs, err := StartUpSenderServer(serverBackend, string(configBytes))
	s.NoError(err)

	clientKeystoreDir := filepath.Join(clientTmpDir, keystoreDir)
	clientNodeConfig, err := nodeConfigForLocalPairSync(uuid.New().String(), "", clientTmpDir)
	s.NoError(err)
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
	s.NoError(err)
	err = StartUpReceivingClient(clientBackend, cs, string(clientConfigBytes))
	s.ErrorIs(err, ErrLoggedInKeyUIDConflict)
}

func (s *SyncDeviceSuite) TestPreventLoggedInAccountLocalPairingClientAsSender() {
	clientTmpDir := filepath.Join(s.tmpdir, "client")
	clientBackend := s.prepareBackendWithAccount("", clientTmpDir)
	serverTmpDir := filepath.Join(s.tmpdir, "server")
	serverBackend := s.prepareBackendWithAccount("", serverTmpDir)
	defer func() {
		s.NoError(serverBackend.Logout())
		s.NoError(clientBackend.Logout())
	}()

	serverNodeConfig, err := nodeConfigForLocalPairSync(uuid.New().String(), "", serverTmpDir)
	s.NoError(err)
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
	s.NoError(err)
	cs, err := StartUpReceiverServer(serverBackend, string(serverConfigBytes))
	s.NoError(err)

	clientActiveAccount, err := clientBackend.GetActiveAccount()
	s.NoError(err)
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
	s.NoError(err)
	err = StartUpSendingClient(clientBackend, cs, string(clientConfigBytes))
	s.ErrorContains(err, "[client] status not ok when sending account data, received '500 Internal Server Error'")
}
