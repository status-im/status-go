package pairing

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	accsmanagementcommon "github.com/status-im/status-go/internal/accounts-management/common"
	keystorepkg "github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/dbsetup"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/protocol"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/requests"
	testutils "github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/backend"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	accservice "github.com/status-im/status-go/pkg/services/accounts"
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
	s.logger = testutils.MustCreateTestLogger()
	s.password = "password"
	s.tmpdir = s.T().TempDir()
}

func (s *SyncDeviceSuite) prepareBackendWithAccount(mnemonic, tmpdir string) *backend.StatusBackend {
	err := os.MkdirAll(tmpdir, 0755) // making sure the dir is created
	s.Require().NoError(err)

	backend := s.prepareBackendWithoutAccount(tmpdir)

	displayName, err := common.RandomAlphabeticalString(8)
	s.Require().NoError(err)

	deviceName, err := common.RandomAlphanumericString(8)
	s.Require().NoError(err)

	createAccount := requests.CreateAccount{
		RootDataDir:        tmpdir,
		KdfIterations:      dbsetup.ReducedKDFIterationsNumber,
		DisplayName:        displayName,
		DeviceName:         deviceName,
		Password:           s.password,
		CustomizationColor: "primary",
	}

	if mnemonic == "" {
		_, err = backend.CreateAccountAndLogin(&createAccount)
	} else {
		_, err = backend.RestoreAccountAndLogin(&requests.RestoreAccount{
			Mnemonic:      mnemonic,
			CreateAccount: createAccount,
		})
	}
	s.Require().NoError(err)

	accs, err := backend.GetAccounts()
	s.Require().NoError(err)
	s.Require().NotEmpty(accs[0].ColorHash)

	return backend
}

func (s *SyncDeviceSuite) prepareBackendWithoutAccount(tmpdir string) *backend.StatusBackend {
	backend := backend.NewStatusBackend(s.logger)
	backend.UpdateRootDataDir(tmpdir)
	return backend
}

func (s *SyncDeviceSuite) reEncryptDBInPlace(path, oldKey string, oldIter int, newKey string, newIter int) {
	tmpPath := path + ".reencrypted"
	require.NoError(s.T(), sqlite.ExportDBWithKDFChange(path, oldKey, oldIter, tmpPath, newKey, newIter, nil, nil))
	require.NoError(s.T(), os.Rename(tmpPath, path))
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")
}

// demigrateBackend converts a logged-in DEK-native backend to the legacy encryption scheme
// (as if the profile had been created by an older app version) and logs it back in.
func (s *SyncDeviceSuite) demigrateBackend(b *backend.StatusBackend, tmpdir, keyUID string) {
	require.NoError(s.T(), b.Logout())

	dek, dekIter, err := envelope.Unwrap(tmpdir, keyUID, s.password)
	require.NoError(s.T(), err)

	s.reEncryptDBInPlace(filepath.Join(tmpdir, keyUID+"-v4.db"), dek, dekIter, s.password, dbsetup.ReducedKDFIterationsNumber)
	s.reEncryptDBInPlace(filepath.Join(tmpdir, keyUID+"-wallet.db"), dek, dekIter, s.password, dbsetup.ReducedKDFIterationsNumber)
	require.NoError(s.T(), keystorepkg.ReEncryptKeyStoreDirAtPath(filepath.Join(tmpdir, backend.DefaultKeystoreRelativePath, keyUID), dek, s.password))
	require.NoError(s.T(), envelope.Remove(tmpdir, keyUID))

	require.NoError(s.T(), b.LoginAccount(&requests.Login{KeyUID: keyUID, Password: s.password}))
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

func countKeystoreFiles(directory, key string) int {
	files, err := os.ReadDir(directory)
	if err != nil {
		return 0
	}

	count := 0
	for _, file := range files {
		if strings.Contains(file.Name(), strings.ToLower(key)) {
			count++
		}
	}
	return count
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

	serverAccountsAPI := serverBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)
	walletAccounts := &accsmanagementtypes.AccountCreationDetails{
		Path:    accsmanagementcommon.PathDefaultWalletAccount,
		Name:    "Default Wallet Account",
		Emoji:   "💰",
		ColorID: "primary",
	}
	serverSeedPhraseKp, err := serverAccountsAPI.AddKeypairViaSeedPhrase(ctx, seedKeypairMnemonic, s.password, "Seed Phrase Keypair", accsmanagementtypes.ColdWalletTypeNone, walletAccounts)
	require.NoError(s.T(), err, "saving seed phrase keypair on server with keystore files created")

	clientAccountsAPI := clientBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)
	clientSeedPhraseKp, err := clientAccountsAPI.AddKeypairViaSeedPhrase(ctx, seedKeypairMnemonic, s.password, "Seed Phrase Keypair", accsmanagementtypes.ColdWalletTypeNone, walletAccounts)
	require.NoError(s.T(), err, "saving seed phrase keypair on client without keystore files")

	// check server - server should contain keystore files for imported seed phrase
	serverKeystorePath := filepath.Join(serverTmpDir, backend.DefaultKeystoreRelativePath, serverActiveAccount.KeyUID)
	require.True(s.T(), containsKeystoreFile(serverKeystorePath, serverSeedPhraseKp.DerivedFrom[2:]))
	for _, acc := range serverSeedPhraseKp.Accounts {
		require.True(s.T(), containsKeystoreFile(serverKeystorePath, acc.Address.String()[2:]))
	}

	accountsManager := clientBackend.AccountsManager()
	// need to delete keystore files for keypair in order to simulate the case where the keypair was restored and keystore files were not created yet
	clientKeystoreDir := filepath.Join(clientTmpDir, "keystore", clientActiveAccount.KeyUID)
	files, err := os.ReadDir(clientKeystoreDir)
	require.NoError(s.T(), err)

	for _, file := range files {
		if strings.Contains(strings.ToLower(file.Name()), strings.ToLower(clientSeedPhraseKp.DerivedFrom[2:])) {
			require.NoError(s.T(), os.RemoveAll(filepath.Join(clientKeystoreDir, file.Name())))
			continue
		}

		for _, acc := range clientSeedPhraseKp.Accounts {
			if !strings.Contains(strings.ToLower(file.Name()), strings.ToLower(acc.Address.String()[2:])) {
				continue
			}
			require.NoError(s.T(), os.RemoveAll(filepath.Join(clientKeystoreDir, file.Name())))
		}
	}

	// check client - client should not contain keystore files for imported seed phrase
	clientKeystorePath := filepath.Join(clientTmpDir, backend.DefaultKeystoreRelativePath, clientActiveAccount.KeyUID)
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
	err = StartUpKeystoreFilesReceivingClient(clientBackend, cs, &clientPayloadSourceConfig)
	require.NoError(s.T(), err)

	// check client - client should contain keystore files for imported seed phrase
	require.True(s.T(), containsKeystoreFile(clientKeystorePath, clientSeedPhraseKp.DerivedFrom[2:]))
	for _, acc := range clientSeedPhraseKp.Accounts {
		require.True(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}

	err = accountsManager.ReloadKeystore()
	require.NoError(s.T(), err)

	// both backends are DEK-native (created by this version): the received files must be
	// stored under the client's DEK, not the raw password (the manager's raw-password
	// fallback would mask that, so check the files directly)
	clientDek, _, err := envelope.Unwrap(clientTmpDir, clientActiveAccount.KeyUID, s.password)
	require.NoError(s.T(), err)
	require.NoError(s.T(), keystorepkg.VerifyKeyStoreDirAtPath(clientKeystorePath, clientDek))

	// check keystore on client
	genAcc, err := accountsManager.LoadAccount(types.HexToAddress(clientSeedPhraseKp.DerivedFrom), s.password)
	require.NoError(s.T(), err)

	accInfo := genAcc.ToIdentifiedAccountInfo()
	require.Equal(s.T(), clientSeedPhraseKp.KeyUID, accInfo.KeyUID)

	for _, acc := range clientSeedPhraseKp.Accounts {
		genAcc, err = accountsManager.LoadAccount(acc.Address, s.password)
		require.NoError(s.T(), err)
		accInfo := genAcc.ToIdentifiedAccountInfo()
		require.Equal(s.T(), acc.Address.String(), accInfo.Address)
	}
}

// TestTransferringKeystoreFilesFromLegacySender covers the KeystoreFilesPayload flow with a
// LEGACY sender (an old app version) and a DEK-native receiver: the wire format is unchanged
// and the receiver stores the files under its own DEK.
func (s *SyncDeviceSuite) TestTransferringKeystoreFilesFromLegacySender() {
	ctx := context.TODO()

	serverTmpDir := filepath.Join(s.tmpdir, "server")
	serverBackend := s.prepareBackendWithAccount(profileKeypairMnemonic, serverTmpDir)

	clientTmpDir := filepath.Join(s.tmpdir, "client")
	clientBackend := s.prepareBackendWithAccount(profileKeypairMnemonic, clientTmpDir)
	defer func() {
		require.NoError(s.T(), clientBackend.Logout())
		require.NoError(s.T(), serverBackend.Logout())
	}()

	serverActiveAccount, err := serverBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	clientActiveAccount, err := clientBackend.GetActiveAccount()
	require.NoError(s.T(), err)
	require.True(s.T(), serverActiveAccount.KeyUID == clientActiveAccount.KeyUID)

	// Bring the server profile back to the legacy scheme, as an older app version would have it.
	s.demigrateBackend(serverBackend, serverTmpDir, serverActiveAccount.KeyUID)

	serverBackend.Messenger().SetLocalPairing(true)
	clientBackend.Messenger().SetLocalPairing(true)

	serverAccountsAPI := serverBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)
	walletAccounts := &accsmanagementtypes.AccountCreationDetails{
		Path:    accsmanagementcommon.PathDefaultWalletAccount,
		Name:    "Default Wallet Account",
		Emoji:   "💰",
		ColorID: "primary",
	}
	serverSeedPhraseKp, err := serverAccountsAPI.AddKeypairViaSeedPhrase(ctx, seedKeypairMnemonic, s.password, "Seed Phrase Keypair", accsmanagementtypes.ColdWalletTypeNone, walletAccounts)
	require.NoError(s.T(), err, "saving seed phrase keypair on legacy server with keystore files created")

	clientAccountsAPI := clientBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)
	clientSeedPhraseKp, err := clientAccountsAPI.AddKeypairViaSeedPhrase(ctx, seedKeypairMnemonic, s.password, "Seed Phrase Keypair", accsmanagementtypes.ColdWalletTypeNone, walletAccounts)
	require.NoError(s.T(), err, "saving seed phrase keypair on client without keystore files")

	// the legacy server keystore stays password-encrypted
	serverKeystorePath := filepath.Join(serverTmpDir, backend.DefaultKeystoreRelativePath, serverActiveAccount.KeyUID)
	require.False(s.T(), envelope.Exists(serverTmpDir, serverActiveAccount.KeyUID))
	require.NoError(s.T(), keystorepkg.VerifyKeyStoreDirAtPath(serverKeystorePath, s.password))
	require.True(s.T(), containsKeystoreFile(serverKeystorePath, serverSeedPhraseKp.DerivedFrom[2:]))

	// remove the client's keystore files for the seed keypair, simulating a restored keypair
	clientKeystorePath := filepath.Join(clientTmpDir, backend.DefaultKeystoreRelativePath, clientActiveAccount.KeyUID)
	files, err := os.ReadDir(clientKeystorePath)
	require.NoError(s.T(), err)
	for _, file := range files {
		if strings.Contains(strings.ToLower(file.Name()), strings.ToLower(clientSeedPhraseKp.DerivedFrom[2:])) {
			require.NoError(s.T(), os.RemoveAll(filepath.Join(clientKeystorePath, file.Name())))
			continue
		}
		for _, acc := range clientSeedPhraseKp.Accounts {
			if strings.Contains(strings.ToLower(file.Name()), strings.ToLower(acc.Address.String()[2:])) {
				require.NoError(s.T(), os.RemoveAll(filepath.Join(clientKeystorePath, file.Name())))
			}
		}
	}
	require.False(s.T(), containsKeystoreFile(clientKeystorePath, clientSeedPhraseKp.DerivedFrom[2:]))

	// transfer
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
	err = StartUpKeystoreFilesReceivingClient(clientBackend, cs, &clientPayloadSourceConfig)
	require.NoError(s.T(), err)

	// the client received the files and stored them under its own DEK
	require.True(s.T(), containsKeystoreFile(clientKeystorePath, clientSeedPhraseKp.DerivedFrom[2:]))
	for _, acc := range clientSeedPhraseKp.Accounts {
		require.True(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}

	clientDek, _, err := envelope.Unwrap(clientTmpDir, clientActiveAccount.KeyUID, s.password)
	require.NoError(s.T(), err)
	require.NoError(s.T(), keystorepkg.VerifyKeyStoreDirAtPath(clientKeystorePath, clientDek))

	// the server keystore is untouched, still legacy
	require.NoError(s.T(), keystorepkg.VerifyKeyStoreDirAtPath(serverKeystorePath, s.password))
	require.False(s.T(), envelope.Exists(serverTmpDir, serverActiveAccount.KeyUID))

	// keys remain loadable on the client through the resolver
	accountsManager := clientBackend.AccountsManager()
	require.NoError(s.T(), accountsManager.ReloadKeystore())
	genAcc, err := accountsManager.LoadAccount(types.HexToAddress(clientSeedPhraseKp.DerivedFrom), s.password)
	require.NoError(s.T(), err)
	require.Equal(s.T(), clientSeedPhraseKp.KeyUID, genAcc.ToIdentifiedAccountInfo().KeyUID)
}

func (s *SyncDeviceSuite) TestTransferringKeystoreFilesSkipsAlreadyStoredKeystores() {
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

	serverAccountsAPI := serverBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)
	walletAccounts := &accsmanagementtypes.AccountCreationDetails{
		Path:    accsmanagementcommon.PathDefaultWalletAccount,
		Name:    "Default Wallet Account",
		Emoji:   "💰",
		ColorID: "primary",
	}
	serverSeedPhraseKp, err := serverAccountsAPI.AddKeypairViaSeedPhrase(ctx, seedKeypairMnemonic, s.password, "Seed Phrase Keypair", accsmanagementtypes.ColdWalletTypeNone, walletAccounts)
	require.NoError(s.T(), err, "saving seed phrase keypair on server with keystore files created")

	clientAccountsAPI := clientBackend.StatusNode().AccountService().APIs()[1].Service.(*accservice.API)
	clientSeedPhraseKp, err := clientAccountsAPI.AddKeypairViaSeedPhrase(ctx, seedKeypairMnemonic, s.password, "Seed Phrase Keypair", accsmanagementtypes.ColdWalletTypeNone, walletAccounts)
	require.NoError(s.T(), err, "saving seed phrase keypair on client with keystore files created")

	// unlike TestTransferringKeystoreFiles, the client's keystore files are deliberately kept, each account has exactly one file before the transfer
	clientKeystorePath := filepath.Join(clientTmpDir, backend.DefaultKeystoreRelativePath, clientActiveAccount.KeyUID)
	require.Equal(s.T(), 1, countKeystoreFiles(clientKeystorePath, clientSeedPhraseKp.DerivedFrom[2:]))
	for _, acc := range clientSeedPhraseKp.Accounts {
		require.Equal(s.T(), 1, countKeystoreFiles(clientKeystorePath, acc.Address.String()[2:]))
	}

	// prepare sender
	serverKeystorePath := filepath.Join(serverTmpDir, backend.DefaultKeystoreRelativePath, serverActiveAccount.KeyUID)
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
	err = StartUpKeystoreFilesReceivingClient(clientBackend, cs, &clientPayloadSourceConfig)
	require.NoError(s.T(), err)

	// still exactly one keystore file per account, the transferred duplicates were skipped
	require.Equal(s.T(), 1, countKeystoreFiles(clientKeystorePath, clientSeedPhraseKp.DerivedFrom[2:]))
	for _, acc := range clientSeedPhraseKp.Accounts {
		require.Equal(s.T(), 1, countKeystoreFiles(clientKeystorePath, acc.Address.String()[2:]))
	}

	// and the kept keystore files still load correctly
	accountsManager := clientBackend.AccountsManager()
	err = accountsManager.ReloadKeystore()
	require.NoError(s.T(), err)

	genAcc, err := accountsManager.LoadAccount(types.HexToAddress(clientSeedPhraseKp.DerivedFrom), s.password)
	require.NoError(s.T(), err)
	require.Equal(s.T(), clientSeedPhraseKp.KeyUID, genAcc.ToIdentifiedAccountInfo().KeyUID)

	for _, acc := range clientSeedPhraseKp.Accounts {
		genAcc, err = accountsManager.LoadAccount(acc.Address, s.password)
		require.NoError(s.T(), err)
		require.Equal(s.T(), acc.Address.String(), genAcc.ToIdentifiedAccountInfo().Address)
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
	im1 := &messagingtypes.InstallationMetadata{
		Name:       "client-device",
		DeviceType: "client-device-type",
	}
	settings, err := clientBackend.GetSettings()
	s.Require().NoError(err)
	err = clientMessenger.SetInstallationMetadata(settings.InstallationID, im1)
	s.Require().NoError(err)
	response, err := clientMessenger.SendPairInstallation(context.Background(), "", nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	response, err = protocol.WaitOnMessengerResponse(
		serverMessenger,
		func(r *protocol.MessengerResponse) bool {
			for _, i := range r.Installations() {
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
	for _, i := range response.Installations() {
		found = i.ID == settings.InstallationID &&
			i.InstallationMetadata != nil &&
			i.InstallationMetadata.Name == im1.Name &&
			i.InstallationMetadata.DeviceType == im1.DeviceType
		if found {
			break
		}
	}
	s.Require().True(found)

	_, err = serverMessenger.EnableInstallation(settings.InstallationID)
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
	walletAccounts := &accsmanagementtypes.AccountCreationDetails{
		Path:    accsmanagementcommon.PathDefaultWalletAccount,
		Name:    "Default Wallet Account",
		Emoji:   "💰",
		ColorID: "primary",
	}
	serverSeedPhraseKp, err := serverAccountsAPI.AddKeypairViaSeedPhrase(ctx, seedKeypairMnemonic1, s.password, "Seed Phrase Keypair", accsmanagementtypes.ColdWalletTypeNone, walletAccounts)
	require.NoError(s.T(), err, "saving seed phrase keypair on server with keystore files created")

	// Wait for sync messages to be received on client
	err = testutils.RetryWithBackOff(func() error {
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
		serverKp.ColdWallet == clientKp.ColdWallet)

	// Check server - server should contain keystore files for imported seed phrase
	serverKeystorePath := filepath.Join(serverTmpDir, backend.DefaultKeystoreRelativePath, serverActiveAccount.KeyUID)
	require.True(s.T(), containsKeystoreFile(serverKeystorePath, serverKp.DerivedFrom[2:]))
	for _, acc := range serverKp.Accounts {
		require.True(s.T(), containsKeystoreFile(serverKeystorePath, acc.Address.String()[2:]))
	}

	// Check client - client should not contain keystore files for imported seed phrase
	clientKeystorePath := filepath.Join(clientTmpDir, backend.DefaultKeystoreRelativePath, clientActiveAccount.KeyUID)
	require.False(s.T(), containsKeystoreFile(clientKeystorePath, clientKp.DerivedFrom[2:]))
	for _, acc := range clientKp.Accounts {
		require.False(s.T(), containsKeystoreFile(clientKeystorePath, acc.Address.String()[2:]))
	}

	//////////////////////////////////////////////////////////////////////////////
	// Convert it to a keycard keypair on server and sync it to client
	//////////////////////////////////////////////////////////////////////////////
	err = serverAccountsAPI.MigrateNonProfileKeypairToColdWallet(ctx, serverKp.KeyUID, s.password, accsmanagementtypes.ColdWalletTypeStatusKeycard)
	s.Require().NoError(err)

	// Wait for sync messages to be received on client
	err = testutils.RetryWithBackOff(func() error {
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
		serverKp.ColdWallet == clientKp.ColdWallet &&
		serverKp.ColdWallet == accsmanagementtypes.ColdWalletTypeStatusKeycard)

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
	err = serverAccountsAPI.MigrateNonProfileColdWalletKeypairToApp(ctx, seedKeypairMnemonic1, s.password)
	s.Require().NoError(err)

	// Wait for sync messages to be received on client
	err = testutils.RetryWithBackOff(func() error {
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
		serverKp.ColdWallet == clientKp.ColdWallet &&
		serverKp.ColdWallet == accsmanagementtypes.ColdWalletTypeNone)

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
	err = StartUpKeystoreFilesReceivingClient(clientBackend, cs, &clientPayloadSourceConfig)
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
