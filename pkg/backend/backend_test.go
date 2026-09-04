package backend

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/brianvoe/gofakeit/v6"

	accsmanagement "github.com/status-im/status-go/internal/accounts-management"
	accsmanagementcommon "github.com/status-im/status-go/internal/accounts-management/common"
	"github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/crypto"
	types "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	settings "github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/backend/node"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/typeddata"
	"github.com/status-im/status-go/pkg/services/wallet"
	walletservice "github.com/status-im/status-go/pkg/services/wallet"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

func setupTestDB() (*sql.DB, func() error, error) {
	return testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "tests")
}

func setupTestWalletDB() (*sql.DB, func() error, error) {
	return testutils.SetupTestSQLDB(walletdb.DbInitializer{}, "tests")
}

func setupTestMultiDB() (*multiaccounts.Database, func() error, error) {
	tmpfile, err := os.CreateTemp("", "tests")
	if err != nil {
		return nil, nil, err
	}
	db, err := multiaccounts.InitializeDB(tmpfile.Name())
	if err != nil {
		return nil, nil, err
	}
	return db, func() error {
		err := db.Close()
		if err != nil {
			return err
		}
		return os.Remove(tmpfile.Name())
	}, nil
}

func setupGethStatusBackend() (*StatusBackend, func() error, func() error, func() error, error) {
	db, stop1, err := setupTestDB()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	backend := NewStatusBackend(testutils.MustCreateTestLogger())
	if err != nil {
		return nil, nil, nil, nil, err
	}
	backend.StatusNode().SetAppDB(db)

	ma, stop2, err := setupTestMultiDB()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	backend.StatusNode().SetMultiaccountsDB(ma)

	walletDb, stop3, err := setupTestWalletDB()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	backend.StatusNode().SetWalletDB(walletDb)

	return backend, stop1, stop2, stop3, err
}

func TestEnsureDBsOpenedReopensEstablishedDatabases(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, true)
	backend := testContext.backend

	require.NoError(t, backend.closeDBs())
	require.NoError(t, backend.ensureDBsOpened(*testContext.multiAcc, testPassword))
	t.Cleanup(func() {
		require.NoError(t, backend.closeDBs())
	})

	require.NotNil(t, backend.appDB)
	require.NotNil(t, backend.walletDB)
	require.NoError(t, backend.appDB.Ping())
	require.NoError(t, backend.walletDB.Ping())
	require.Same(t, backend.appDB, backend.statusNode.GetAppDB())
	require.Same(t, backend.walletDB, backend.statusNode.GetWalletDB())

	accountsDB, err := accounts.NewDB(backend.appDB)
	require.NoError(t, err)
	profileKeypair, err := accountsDB.GetProfileKeypair()
	require.NoError(t, err)
	require.Equal(t, testContext.profileKeypair.KeyUID, profileKeypair.KeyUID)
}

func TestEnsureDBsOpenedEstablishedDatabasesRejectsWrongPassword(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, true)
	backend := testContext.backend

	require.NoError(t, backend.closeDBs())
	err := backend.ensureDBsOpened(*testContext.multiAcc, "wrong password")
	require.Error(t, err)
	require.Nil(t, backend.appDB)
	require.Nil(t, backend.walletDB)
}

func TestEnsureDBsOpenedEstablishedDatabasesDoesNotRegisterDBsOnAccountsDBFailure(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, true)
	backend := testContext.backend

	require.NoError(t, backend.closeDBs())
	backend.statusNode.SetAppDB(nil)
	backend.statusNode.SetWalletDB(nil)

	previousNewAccountsDB := newAccountsDB
	newAccountsDB = func(*sql.DB) (*accounts.Database, error) {
		return nil, fmt.Errorf("failed to create accounts db")
	}
	t.Cleanup(func() {
		newAccountsDB = previousNewAccountsDB
	})

	err := backend.ensureDBsOpened(*testContext.multiAcc, testPassword)
	require.EqualError(t, err, "failed to create accounts db")
	require.Nil(t, backend.appDB)
	require.Nil(t, backend.walletDB)
	require.Nil(t, backend.statusNode.GetAppDB())
	require.Nil(t, backend.statusNode.GetWalletDB())
}

func handleError(t *testing.T, err error) {
	if err != nil {
		t.Logf("deferred function error: '%s'", err)
	}
}

func TestBackendStartNodeConcurrently(t *testing.T) {
	backend, stop1, stop2, stop3, err := setupGethStatusBackend()
	defer func() {
		err := stop1()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stop2()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stop3()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	require.NoError(t, err)

	config, err := makeTestNodeConfig(t)
	require.NoError(t, err)

	count := 2
	resultCh := make(chan error)

	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func() {
			resultCh <- backend.StartNode(config)
			wg.Done()
		}()
	}

	// close channel as otherwise for loop never finishes
	go func() { wg.Wait(); close(resultCh) }()

	var results []error
	for err := range resultCh {
		results = append(results, err)
	}

	require.Contains(t, results, nil)
	require.Contains(t, results, node.ErrNodeRunning)

	err = backend.StopNode()
	require.NoError(t, err)
}

func TestBackendRestartNodeConcurrently(t *testing.T) {
	backend, stop1, stop2, stopWallet, err := setupGethStatusBackend()
	defer func() {
		err := stop1()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stop2()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stopWallet()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	require.NoError(t, err)

	config, err := makeTestNodeConfig(t)
	require.NoError(t, err)
	const count = 3

	require.NoError(t, backend.StartNode(config))
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(idx int) {
			assert.NoError(t, backend.RestartNode())
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestBackendGettersConcurrently(t *testing.T) {
	backend, stop1, stop2, stopWallet, err := setupGethStatusBackend()
	defer func() {
		err := stop1()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stop2()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stopWallet()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	require.NoError(t, err)

	config, err := makeTestNodeConfig(t)
	require.NoError(t, err)

	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.StatusNode())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.AccountsManager())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.signer)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.Transactor())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.True(t, backend.IsNodeRunning())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.True(t, backend.IsNodeRunning())
		wg.Done()
	}()

	wg.Wait()
}

func TestBackendCallRPCConcurrently(t *testing.T) {
	backend, stop1, stop2, stopWallet, err := setupGethStatusBackend()
	defer func() {
		err := stop1()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stop2()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stopWallet()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	require.NoError(t, err)

	config, err := makeTestNodeConfig(t)
	require.NoError(t, err)

	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wg sync.WaitGroup

	const count = 3
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			result := backend.CallInProcessRPC(fmt.Sprintf(
				`{"jsonrpc":"2.0","method":"appgeneral_version","params":[],"id":%d}`,
				idx+1,
			))
			assert.NotContains(t, result, "error")
			wg.Done()
		}(i)

		wg.Add(1)
		go func(idx int) {
			result := backend.CallInProcessRPC(fmt.Sprintf(
				`{"jsonrpc":"2.0","method":"appgeneral_version","params":[],"id":%d}`,
				idx+1,
			))
			assert.NotContains(t, result, "error")
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestCallRPCWithStoppedNode(t *testing.T) {
	backend := NewStatusBackend(testutils.MustCreateTestLogger())

	resp := backend.CallInProcessRPC(
		`{"jsonrpc":"2.0","method":"appgeneral_version","params":[],"id":1}`,
	)
	assert.Contains(t, resp, "error")

	resp = backend.CallInProcessRPC(
		`{"jsonrpc":"2.0","method":"appgeneral_version","params":[],"id":1}`,
	)
	assert.Contains(t, resp, "error")
}

// TODO(adam): add concurrent tests for: SendTransaction

func TestStartStopMultipleTimes(t *testing.T) {
	backend, stop1, stop2, stopWallet, err := setupGethStatusBackend()
	defer func() {
		err := stop1()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stop2()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stopWallet()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	require.NoError(t, err)

	config, err := makeTestNodeConfig(t)
	require.NoError(t, err)
	require.NoError(t, backend.StartNode(config))
	require.NoError(t, backend.StopNode())
	require.NoError(t, backend.StartNode(config))
	require.NoError(t, backend.StopNode())
}

func TestHashTypedData(t *testing.T) {
	backend, stop1, stop2, stopWallet, err := setupGethStatusBackend()
	defer func() {
		err := stop1()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stop2()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stopWallet()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	require.NoError(t, err)

	config, err := makeTestNodeConfig(t)
	require.NoError(t, err)

	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	eip712Domain := "EIP712Domain"
	mytypes := typeddata.Types{
		eip712Domain: []typeddata.Field{
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Text": []typeddata.Field{
			{Name: "body", Type: "string"},
		},
	}

	domain := map[string]json.RawMessage{
		"name":              json.RawMessage(`"Ether Text"`),
		"version":           json.RawMessage(`"1"`),
		"chainId":           json.RawMessage(fmt.Sprintf("%d", walletcommon.EthereumSepolia)),
		"verifyingContract": json.RawMessage(`"0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"`),
	}
	msg := map[string]json.RawMessage{
		"body": json.RawMessage(`"Hello, Bob!"`),
	}

	typed := typeddata.TypedData{
		Types:       mytypes,
		PrimaryType: "Text",
		Domain:      domain,
		Message:     msg,
	}

	hash, err := backend.HashTypedData(typed)
	require.NoError(t, err)
	assert.NotEqual(t, types.Hash{}, hash)
}

func TestBackendGetVerifiedAccount(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)

	err := testContext.backend.StartNode(testContext.config)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testContext.backend.StopNode())
	}()

	t.Run("AccountDoesntExist", func(t *testing.T) {
		pkey, err := gethcrypto.GenerateKey()
		require.NoError(t, err)
		address := gethcrypto.PubkeyToAddress(pkey.PublicKey)
		key, err := testContext.backend.getVerifiedWalletAccount(address.String(), testPassword)
		require.EqualError(t, err, accsmanagement.ErrAccountDoesNotExist.Error())
		require.Nil(t, key)
	})

	t.Run("PasswordDoesntMatch", func(t *testing.T) {
		pkey, err := gethcrypto.GenerateKey()
		require.NoError(t, err)
		privateKeyHex := types.EncodeHex(gethcrypto.FromECDSA(pkey))
		address := gethcrypto.PubkeyToAddress(pkey.PublicKey)

		_, err = testContext.backend.AccountsManager().CreateKeypairFromPrivateKeyAndStore(privateKeyHex, testPassword, "private key keypair", &accsmanagementtypes.AccountCreationDetails{
			Path: accsmanagementcommon.PathMaster,
		}, 0)
		require.NoError(t, err)

		key, err := testContext.backend.getVerifiedWalletAccount(address.String(), "wrong-password")
		require.EqualError(t, err, keystore.ErrIncorrectPasswordProvided.Error())
		require.Nil(t, key)
	})

	t.Run("PartialAccount", func(t *testing.T) { // This is mobile app specific test and can be removed
		// Create a derived wallet account without storing the keys
		db, err := accounts.NewDB(testContext.backend.appDB)
		require.NoError(t, err)

		err = db.CreateSettings(testContext.settings, *testContext.config)
		require.NoError(t, err)

		walletRootAddress, err := db.GetWalletRootAddress()
		require.NoError(t, err)

		// Store keystore file for the wallet root address
		masterAcc, derivedAccs, err := testContext.backend.AccountsManager().StoreKeystoreFilesForMnemonic(testContext.mnemonic, testPassword,
			[]string{accsmanagementcommon.PathWalletRoot})
		require.NoError(t, err)

		walletRootGeneratedAccount := derivedAccs[accsmanagementcommon.PathWalletRoot]
		require.Equal(t, walletRootAddress, walletRootGeneratedAccount.Address())

		// check the number of wallet addresses in the db
		walletAddresses, err := db.GetWalletAddresses()
		require.NoError(t, err)
		require.Equal(t, 2, len(walletAddresses)) // should be 1, but because of the tests `Run` before this one it's 2

		// Create a new wallet account and store it to db only, without storing the keystore file
		derivedWalletAcc1, err := generator.DeriveChildFromAccount(masterAcc, accsmanagementcommon.CustomWalletPath1)
		require.NoError(t, err)

		err = db.SaveOrUpdateAccounts([]*accsmanagementtypes.Account{
			{
				Address: derivedWalletAcc1.Address(),
				KeyUID:  masterAcc.KeyUID(),
				Path:    accsmanagementcommon.CustomWalletPath1,
			},
		}, false)
		require.NoError(t, err)

		// check the number of wallet addresses in the db, to ensure that the account was saved
		walletAddresses, err = db.GetWalletAddresses()
		require.NoError(t, err)
		require.Equal(t, 3, len(walletAddresses)) // should be 2, but because of the tests `Run` before this one it's 3

		// try to load the account, it should fail because the account is not in the keystore, just in the db
		loadedWalletAcc1, err := testContext.backend.AccountsManager().LoadAccount(derivedWalletAcc1.Address(), testPassword)
		require.Error(t, err)
		// Check if the error contains the expected message (new structured error format includes context)
		require.Contains(t, err.Error(), "keystore file is missing")
		require.Nil(t, loadedWalletAcc1)

		// try to get verified wallet account, it should generate a keystore for the wallet account from the wallet root address
		verifiedWalletAcc1, err := testContext.backend.AccountsManager().GetVerifiedWalletAccount(derivedWalletAcc1.Address(),
			testPassword)
		require.NoError(t, err)

		loadedWalletAcc1, err = testContext.backend.AccountsManager().LoadAccount(derivedWalletAcc1.Address(), testPassword)
		require.NoError(t, err)

		// derive, verified and loaded wallet account should be the same
		require.Equal(t, derivedWalletAcc1.Address(), verifiedWalletAcc1.Address())
		require.Equal(t, derivedWalletAcc1.Address(), loadedWalletAcc1.Address())
	})

	t.Run("Success", func(t *testing.T) {
		pkey, err := crypto.GenerateKey()
		require.NoError(t, err)
		privateKeyHex := types.EncodeHex(crypto.FromECDSA(pkey))
		address := crypto.PubkeyToAddress(pkey.PublicKey)
		keyUIDHex := sha256.Sum256(gethcrypto.FromECDSAPub(&pkey.PublicKey))
		keyUID := types.EncodeHex(keyUIDHex[:])

		db, err := accounts.NewDB(testContext.backend.appDB)
		require.NoError(t, err)
		defer func() {
			handleError(t, db.Close())
		}()

		keypair, err := testContext.backend.AccountsManager().CreateKeypairFromPrivateKeyAndStore(privateKeyHex, testPassword,
			"private key keypair", &accsmanagementtypes.AccountCreationDetails{
				Path: accsmanagementcommon.PathMaster,
			}, 0)
		require.NoError(t, err)

		require.Equal(t, keypair.KeyUID, keyUID)
		require.Len(t, keypair.Accounts, 1)
		require.Equal(t, keypair.Accounts[0].Address, address)

		acc, err := testContext.backend.getVerifiedWalletAccount(address.String(), testPassword)
		require.NoError(t, err)
		require.Equal(t, address, acc.Address())
	})
}

func TestRuntimeLogLevelIsNotWrittenToDatabase(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, false)

	json := `{
		"NetworkId": 3,
		"KeycardPairingDataFile": "` + path.Join(testContext.config.RootDataDir, "keycard/pairings.json") + `",
		"NoDiscovery": true,
		"TorrentConfig": {
			"Port": 9025,
			"Enabled": false,
			"DataDir": "` + testContext.config.RootDataDir + `/archivedata",
			"TorrentDir": "` + testContext.config.RootDataDir + `/torrents"
		},
		"RuntimeLogLevel": "INFO",
		"LogLevel": "DEBUG"
	}`

	newConf, err := params.NewConfigFromJSON(json)
	require.NoError(t, err)
	require.Equal(t, "INFO", newConf.RuntimeLogLevel)

	require.NoError(t, testContext.backend.OpenAccounts())
	require.NotNil(t, testContext.backend.statusNode.MediaServer())

	err = testContext.backend.ensureDBsOpened(*testContext.multiAcc, testPassword)
	require.NoError(t, err)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err = testContext.backend.StartNodeWithChatKeyOrMnemonic(
		request,
		testContext.mnemonic,
		nil,
		false,
	)
	require.NoError(t, err)

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	chatPrivKey := strings.TrimPrefix(testContext.chatPrivateKey, "0x")
	require.NoError(t, testContext.backend.StartNodeWithKey(*testContext.multiAcc, testPassword, chatPrivKey, newConf))

	defer func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	}()

	c, err := testContext.backend.GetNodeConfig()
	require.NoError(t, err)
	require.Equal(t, "", c.RuntimeLogLevel)
	require.Equal(t, "DEBUG", c.LogLevel)
}

func TestLoginAccount(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, false)

	nameserver := "8.8.8.8"

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.RootDataDir,
		LogFilePath:        testContext.config.RootDataDir + "/log",
		WakuV2Nameserver:   &nameserver,
		WakuV2Fleet:        "status.staging",
	}

	c := make(chan interface{}, 10)
	signal.SetHandler(func(data []byte) {
		if strings.Contains(string(data), signal.EventLoggedIn) {
			require.Contains(t, string(data), "status.staging")
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetHandler)
	waitForLogin := func(chan interface{}) {
		select {
		case <-c:
			break
		case <-time.After(5 * time.Second):
			t.FailNow()
		}
	}

	acc, err := testContext.backend.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	require.Equal(t, nameserver, testContext.backend.config.WakuV2Config.Nameserver)

	require.True(t, acc.HasAcceptedTerms)

	waitForLogin(c)
	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	testContext.backend.UpdateRootDataDir(testContext.config.RootDataDir)

	accounts, err := testContext.backend.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)

	require.NotEmpty(t, accounts[0].KeyUID)
	require.Equal(t, acc.KeyUID, accounts[0].KeyUID)

	migratedLogDir := testContext.config.RootDataDir + "/logs-after-login"
	loginAccountRequest := &requests.Login{
		KeyUID:           accounts[0].KeyUID,
		Password:         testPassword,
		WakuV2Nameserver: nameserver,
		LogFilePath:      migratedLogDir,
	}
	err = testContext.backend.LoginAccount(loginAccountRequest)
	require.NoError(t, err)
	waitForLogin(c)
	require.Equal(t, nameserver, testContext.backend.config.WakuV2Config.Nameserver)

	// A non-empty LogFilePath overrides and persists the profile's log directory
	require.Equal(t, migratedLogDir, testContext.backend.config.LogDir)
	persistedConfig, err := testContext.backend.GetNodeConfig()
	require.NoError(t, err)
	require.Equal(t, migratedLogDir, persistedConfig.LogDir)

	// The max-backups setter updates the DB and the live config in one call
	require.NoError(t, testContext.backend.SetProfileLogMaxBackups(7))
	require.Equal(t, 7, testContext.backend.config.LogMaxBackups)
	persistedConfig, err = testContext.backend.GetNodeConfig()
	require.NoError(t, err)
	require.Equal(t, 7, persistedConfig.LogMaxBackups)
}

func TestVerifyDatabasePassword(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, false)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(
		request,
		testContext.mnemonic,
		nil,
		false,
	)
	require.NoError(t, err)

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	require.Error(t, testContext.backend.VerifyDatabasePassword(testContext.profileKeypair.KeyUID, "wrong-pass"))
	require.NoError(t, testContext.backend.VerifyDatabasePassword(testContext.profileKeypair.KeyUID, testPassword))
}

func TestConvertAccount(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(
		request,
		testContext.mnemonic,
		nil,
		false,
	)
	require.NoError(t, err)

	multiaccounts, err := testContext.backend.GetAccounts()
	require.NoError(t, err)
	require.NotEmpty(t, multiaccounts[0].ColorHash)
	serverMessenger := testContext.backend.Messenger()
	require.NotNil(t, serverMessenger)

	// Ensure all created accounts are in the keystore and can be loaded
	masterAddress := types.HexToAddress(testContext.profileKeypair.DerivedFrom)
	ok, err := testContext.backend.AccountsManager().VerifyAccountPassword(masterAddress, testPassword)
	require.NoError(t, err)
	require.True(t, ok)

	for _, acc := range testContext.profileKeypair.Accounts {
		ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(acc.Address, testPassword)
		require.NoError(t, err)
		require.True(t, ok)
	}

	// Ensure we're able to open the DB
	err = testContext.backend.ensureAppDBOpened(*testContext.multiAcc, dbCredentials{secret: testPassword, kdfIter: testContext.multiAcc.KDFIterations})
	require.NoError(t, err)

	// db creation
	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)

	// Check that the keypair is not yet marked as migrated to cold-wallet
	keypair, err := db.GetKeypairByKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.False(t, keypair.MigratedToColdWallet())

	require.NoError(t, db.SaveSettingField(settings.Mnemonic, testContext.mnemonic))
	// seeded true so the later assertion fails if the conversion stops clearing it;
	// nothing sets this flag locally, so it would otherwise pass on the default
	require.NoError(t, db.SaveSettingField(settings.ProfileMigrationNeeded, true))
	seededMigrationNeeded, err := db.ProfileMigrationNeeded()
	require.NoError(t, err)
	require.True(t, seededMigrationNeeded, "precondition: the flag is set before the conversion")

	keycardAccount := *testContext.multiAcc
	keycardAccount.KeycardPairing = "pairing"

	keycardSettings := settings.Settings{}

	// Converting to a keycard account
	const keycardPassword = "222222" // represents password for a keycard user
	err = testContext.backend.ConvertToKeycardAccount(keycardAccount, keycardSettings, testContext.profileKeypair.KeyUID, testPassword, keycardPassword)
	require.NoError(t, err)

	require.NotNil(t, testContext.backend.appDB)
	activeAccount, err := testContext.backend.GetActiveAccount()
	require.NoError(t, err)
	require.Equal(t, keycardAccount.KeycardPairing, activeAccount.KeycardPairing)

	// Validating results of converting to a keycard account.
	// All keystore files for the account which is migrated need to be removed.
	ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(masterAddress, testPassword)
	require.Error(t, err)
	require.False(t, ok)

	for _, acc := range testContext.profileKeypair.Accounts {
		ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(acc.Address, testPassword)
		require.Error(t, err)
		require.False(t, ok)
	}

	require.Zero(t, countKeystoreFilesForKeyUID(t, testContext.config.RootDataDir, testContext.profileKeypair.KeyUID),
		"Expected zero keystore files after conversion because a keycard profile must leave no orphan key files on disk")

	convertedMultiAcc, err := testContext.backend.multiaccountsDB.GetAccount(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Equal(t, "pairing", convertedMultiAcc.KeycardPairing,
		"Expected the keycard pairing persisted to multiaccounts DB because desktop re-login resolves the pairing from it")

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	chatPrivKey := strings.TrimPrefix(testContext.chatPrivateKey, "0x")
	require.NoError(t, testContext.backend.StartNodeWithKey(*testContext.multiAcc, keycardPassword, chatPrivKey, testContext.config))

	defer func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	}()

	// Ensure we're able to open the DB
	err = testContext.backend.ensureDBsOpened(keycardAccount, keycardPassword)
	require.NoError(t, err)

	// db creation after re-encryption
	db1, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)

	// Check that the keypair is now marked as cold-wallet (keycard) migrated
	keypair, err = db1.GetKeypairByKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.True(t, keypair.MigratedToColdWallet())

	storedMnemonic, err := db1.Mnemonic()
	require.NoError(t, err)
	require.Empty(t, storedMnemonic,
		"Expected the mnemonic wiped from settings because the seed phrase must not remain in a keycard profile DB")

	migrationNeeded, err := db1.ProfileMigrationNeeded()
	require.NoError(t, err)
	require.False(t, migrationNeeded,
		"Expected ProfileMigrationNeeded false because the conversion completes the profile migration")

	// Converting to a regular account
	err = testContext.backend.ConvertToRegularAccount(testContext.mnemonic, keycardPassword, testPassword)
	require.NoError(t, err)

	activeAccount, err = testContext.backend.GetActiveAccount()
	require.NoError(t, err)
	require.Empty(t, activeAccount.KeycardPairing)

	// Validating results of converting to a regular account.
	// All keystore files for need to be created.
	ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(masterAddress, testPassword)
	require.NoError(t, err)
	require.True(t, ok)

	for _, acc := range testContext.profileKeypair.Accounts {
		ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(acc.Address, testPassword)
		require.NoError(t, err)
		require.True(t, ok)
	}

	// Ensure we're able to open the DB
	err = testContext.backend.ensureDBsOpened(keycardAccount, testPassword)
	require.NoError(t, err)

	// db creation after re-encryption
	db2, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)

	// Check that the keypair is no longer marked as cold-wallet migrated
	keypair, err = db2.GetKeypairByKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.False(t, keypair.MigratedToColdWallet())

	regularMultiAcc, err := testContext.backend.multiaccountsDB.GetAccount(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Empty(t, regularMultiAcc.KeycardPairing,
		"Expected the keycard pairing cleared from multiaccounts DB because a regular account must not be treated as keycard-paired")
}

func copyFile(srcFolder string, dstFolder string, fileName string, t *testing.T) {
	data, err := os.ReadFile(path.Join(srcFolder, fileName))
	if err != nil {
		t.Fail()
	}

	err = os.WriteFile(path.Join(dstFolder, fileName), data, 0600)
	if err != nil {
		t.Fail()
	}
}

func copyDir(srcFolder string, dstFolder string, t *testing.T) {
	files, err := os.ReadDir(srcFolder)
	require.NoError(t, err)
	for _, file := range files {
		if !file.IsDir() {
			copyFile(srcFolder, dstFolder, file.Name(), t)
		} else {
			childFolder := path.Join(srcFolder, file.Name())
			newFolder := path.Join(dstFolder, file.Name())
			err = os.MkdirAll(newFolder, os.ModePerm)
			require.NoError(t, err)
			copyDir(childFolder, newFolder, t)
		}
	}
}

func loginDesktopUser(t *testing.T, conf *params.NodeConfig, keyUID string) {
	// The following passwords and DB used in this test unit are only
	// used to determine if login process works correctly after a migration

	// Expected account data:
	username := "TestUser"
	passwd := "0xC888C9CE9E098D5864D3DED6EBCC140A12142263BACE3A23A36F9905F12BD64A" // #nosec G101

	b := NewStatusBackend(testutils.MustCreateTestLogger())

	b.UpdateRootDataDir(conf.RootDataDir)

	require.NoError(t, b.OpenAccounts())

	accs, err := b.GetAccounts()
	require.NoError(t, err)

	require.Len(t, accs, 1)
	require.Equal(t, username, accs[0].Name)
	require.Equal(t, keyUID, accs[0].KeyUID)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := b.StartNodeWithAccount(accs[0], passwd, conf, nil)
		require.NoError(t, err)
	}()

	wg.Wait()
	require.NoError(t, b.Logout())
	require.NotNil(t, b.statusNode.MediaServer())
	require.NoError(t, b.StopNode())

}

func TestLoginAndMigrationsStillWorkWithExistingDesktopUser(t *testing.T) {
	keyUID := "0x7c46c8f6f059ab72d524f2a6d356904db30bb0392636172ab3929a6bd2220f84" // #nosec G101

	srcFolder := "testdata/test-0.132.0-account/"

	tmpdir := t.TempDir()
	copyDir(srcFolder, tmpdir, t)

	keystoreDir := path.Join(tmpdir, "keystore", keyUID)
	err := os.MkdirAll(keystoreDir, 0700)
	require.NoError(t, err)

	srcKeystoreFolder := "testdata/test-0.132.0-account/keystore/"
	copyDir(srcKeystoreFolder, keystoreDir, t)

	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	loginDesktopUser(t, conf, keyUID)
	loginDesktopUser(t, conf, keyUID) // Login twice to catch weird errors that only appear after logout
}

func TestChangeDatabasePassword(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, false)

	err := testContext.backend.StartNode(testContext.config)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, testContext.backend.StopNode())
	}()

	masterAddress := types.HexToAddress(testContext.profileKeypair.DerivedFrom)
	ok, err := testContext.backend.AccountsManager().VerifyAccountPassword(masterAddress, testPassword)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(testContext.profileKeypair.Accounts[0].Address, testPassword)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(testContext.profileKeypair.Accounts[1].Address, testPassword)
	require.NoError(t, err)
	require.True(t, ok)

	db, err := testContext.backend.accountsDB()
	require.NoError(t, err)

	acc, err := db.GetAccountByAddress(testContext.profileKeypair.Accounts[0].Address)
	require.NoError(t, err)
	require.Equal(t, testContext.profileKeypair.Accounts[0].Address, acc.Address)

	acc, err = db.GetAccountByAddress(testContext.profileKeypair.Accounts[1].Address)
	require.NoError(t, err)
	require.Equal(t, testContext.profileKeypair.Accounts[1].Address, acc.Address)

	// Change password
	const newPassword = "newPassword"
	err = testContext.backend.ChangeDatabasePassword(testContext.profileKeypair.KeyUID, testPassword, newPassword, false)
	require.NoError(t, err)

	testContext.backend.UpdateRootDataDir(testContext.config.RootDataDir)

	// Test that keystore can be decrypted with the new password
	ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(masterAddress, newPassword)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(testContext.profileKeypair.Accounts[0].Address, newPassword)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(testContext.profileKeypair.Accounts[1].Address, newPassword)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCreateWallet(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.RootDataDir,
		LogFilePath:        testContext.config.RootDataDir + "/log",
	}

	c := make(chan interface{}, 10)
	signal.SetHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetHandler)

	account, err := testContext.backend.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	statusNode := testContext.backend.statusNode
	require.NotNil(t, statusNode)

	walletService := statusNode.WalletService()
	require.NotNil(t, walletService)
	walletAPI := walletservice.NewAPI(walletService)

	paths := []string{"m/44'/60'/0'/0/1"}

	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)
	masterAddress, err := db.GetMasterAddress()
	require.NoError(t, err)

	mnemonic, err := db.Mnemonic()
	require.NoError(t, err)
	require.NotEmpty(t, mnemonic)

	derivedAddress, err := walletAPI.GetDerivedAddresses(context.Background(), testPassword, masterAddress.String(), paths)
	require.NoError(t, err)
	require.Len(t, derivedAddress, 1)

	accountsService := statusNode.AccountService()
	require.NotNil(t, accountsService)
	accountsAPI := accountsService.AccountsAPI()

	err = accountsAPI.AddAccount(context.Background(), testPassword, &accsmanagementtypes.Account{
		Address:   derivedAddress[0].Address,
		KeyUID:    account.KeyUID,
		Type:      accsmanagementtypes.AccountTypeGenerated,
		PublicKey: derivedAddress[0].PublicKey,
		Emoji:     "some",
		ColorID:   "so",
		Name:      "some name",
		Path:      derivedAddress[0].Path,
	})
	require.NoError(t, err)
}

func TestSetFleet(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.RootDataDir,
		LogFilePath:        testContext.config.RootDataDir + "/log",
	}

	c := make(chan interface{}, 10)
	signal.SetHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetHandler)

	newAccount, err := testContext.backend.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	statusNode := testContext.backend.statusNode
	require.NotNil(t, statusNode)

	savedSettings, err := testContext.backend.GetSettings()
	require.NoError(t, err)
	require.Empty(t, savedSettings.Fleet)

	accountsDB, err := testContext.backend.accountsDB()
	require.NoError(t, err)
	err = accountsDB.SaveSettingField(settings.Fleet, params.FleetStatusProd)
	require.NoError(t, err)

	savedSettings, err = testContext.backend.GetSettings()
	require.NoError(t, err)
	require.NotEmpty(t, savedSettings.Fleet)
	require.Equal(t, params.FleetStatusProd, *savedSettings.Fleet)

	require.NoError(t, testContext.backend.Logout())

	testContext.backend.UpdateRootDataDir(testContext.config.RootDataDir)

	loginAccountRequest := &requests.Login{
		KeyUID:   newAccount.KeyUID,
		Password: testPassword,
	}
	require.NoError(t, testContext.backend.LoginAccount(loginAccountRequest))
	select {
	case <-c:
		break
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	// Check is using the right fleet
	require.Equal(t, params.FleetStatusProd, testContext.backend.config.ClusterConfig.Fleet)

	require.NoError(t, testContext.backend.Logout())
}

func fakeToken() security.SensitiveString {
	return security.NewSensitiveString(gofakeit.LetterN(10))
}

func TestWalletConfigOnLoginAccount(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	poktToken := fakeToken()
	infuraToken := fakeToken()
	alchemyAPIKey := fakeToken()
	raribleMainnetAPIKey := fakeToken()
	raribleTestnetAPIKey := fakeToken()
	coingeckoAPIKey := fakeToken()
	coingeckoDemoAPIKey := fakeToken()

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.RootDataDir,
		LogFilePath:        testContext.config.RootDataDir + "/log",
	}
	c := make(chan interface{}, 10)
	signal.SetHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetHandler)

	newAccount, err := testContext.backend.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	statusNode := testContext.backend.statusNode
	require.NotNil(t, statusNode)

	require.NoError(t, testContext.backend.Logout())

	loginAccountRequest := &requests.Login{
		KeyUID:   newAccount.KeyUID,
		Password: testPassword,
		WalletSecretsConfig: requests.WalletSecretsConfig{
			PoktToken:            poktToken,
			InfuraToken:          infuraToken,
			AlchemyAPIKey:        alchemyAPIKey,
			RaribleMainnetAPIKey: raribleMainnetAPIKey,
			RaribleTestnetAPIKey: raribleTestnetAPIKey,
			CoingeckoAPIKey:      coingeckoAPIKey,
			CoingeckoDemoAPIKey:  coingeckoDemoAPIKey,
		},
	}

	testContext.backend.UpdateRootDataDir(testContext.config.RootDataDir)

	require.NoError(t, testContext.backend.LoginAccount(loginAccountRequest))
	select {
	case <-c:
		break
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	walletConfig := testContext.backend.config.WalletConfig
	require.Equal(t, walletConfig.InfuraAPIKey, infuraToken)
	require.Equal(t, walletConfig.AlchemyAPIKey, alchemyAPIKey)
	require.Equal(t, walletConfig.RaribleMainnetAPIKey, raribleMainnetAPIKey)
	require.Equal(t, walletConfig.RaribleTestnetAPIKey, raribleTestnetAPIKey)
	require.Equal(t, walletConfig.CoingeckoAPIKey, coingeckoAPIKey)
	require.Equal(t, walletConfig.CoingeckoDemoAPIKey, coingeckoDemoAPIKey)

	require.NoError(t, testContext.backend.Logout())
}

func TestTestnetEnabledSettingOnCreateAccount(t *testing.T) {
	tmpdir := t.TempDir()

	b := NewStatusBackend(testutils.MustCreateTestLogger())

	// Creating an account with test networks enabled
	createAccountRequest1 := &requests.CreateAccount{
		DisplayName:         "User-1",
		CustomizationColor:  "#ffffff",
		Password:            "password123",
		RootDataDir:         tmpdir,
		LogFilePath:         tmpdir + "/log",
		TestNetworksEnabled: true,
	}
	_, err := b.CreateAccountAndLogin(createAccountRequest1)
	require.NoError(t, err)
	statusNode := b.statusNode
	require.NotNil(t, statusNode)

	settings, err := b.GetSettings()
	require.NoError(t, err)
	require.True(t, settings.TestNetworksEnabled)

	require.NoError(t, b.Logout())

	// Creating an account with test networks disabled
	createAccountRequest2 := &requests.CreateAccount{
		DisplayName:        "User-2",
		CustomizationColor: "#ffffff",
		Password:           "password",
		RootDataDir:        tmpdir,
		LogFilePath:        tmpdir + "/log",
	}
	_, err = b.CreateAccountAndLogin(createAccountRequest2)
	require.NoError(t, err)
	statusNode = b.statusNode
	require.NotNil(t, statusNode)

	settings, err = b.GetSettings()
	require.NoError(t, err)
	require.False(t, settings.TestNetworksEnabled)

	require.NoError(t, b.Logout())
}

func TestRestoreAccountAndLogin(t *testing.T) {
	tmpdir := t.TempDir()

	backend := NewStatusBackend(testutils.MustCreateTestLogger())

	// Test case 1: Valid restore account request
	restoreRequest := &requests.RestoreAccount{
		Mnemonic: "test test test test test test test test test test test test",
		CreateAccount: requests.CreateAccount{
			DisplayName:        "Account1",
			DeviceName:         "StatusIM",
			Password:           "password",
			CustomizationColor: "0x000000",
			RootDataDir:        tmpdir,
		},
	}
	account, err := backend.RestoreAccountAndLogin(restoreRequest)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, "Account1", account.Name)

	// Test case 2: Invalid restore account request
	invalidRequest := &requests.RestoreAccount{}
	_, err = backend.RestoreAccountAndLogin(invalidRequest)
	require.Error(t, err)

	db, err := accounts.NewDB(backend.appDB)
	require.NoError(t, err)
	mnemonic, err := db.Mnemonic()
	require.NoError(t, err)
	require.Empty(t, mnemonic)
}

func TestRestoreAccountAndLoginWithoutDisplayName(t *testing.T) {
	tmpdir := t.TempDir()

	backend := NewStatusBackend(testutils.MustCreateTestLogger())

	// Test case: Valid restore account request without DisplayName
	restoreRequest := &requests.RestoreAccount{
		Mnemonic: "test test test test test test test test test test test test",
		CreateAccount: requests.CreateAccount{
			DeviceName:         "StatusIM",
			Password:           "password",
			CustomizationColor: "0x000000",
			RootDataDir:        tmpdir,
		},
	}
	account, err := backend.RestoreAccountAndLogin(restoreRequest)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.NotEmpty(t, account.Name)
}

func TestAcceptTerms(t *testing.T) {
	tmpdir := t.TempDir()
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	b.UpdateRootDataDir(conf.RootDataDir)
	require.NoError(t, b.OpenAccounts())
	nameserver := "8.8.8.8"
	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           "some-password",
		RootDataDir:        tmpdir,
		LogFilePath:        tmpdir + "/log",
		WakuV2Nameserver:   &nameserver,
		WakuV2Fleet:        "status.staging",
	}
	_, err = b.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	err = b.AcceptTerms()
	require.NoError(t, err)
}

func TestCreateAccountPathsValidation(t *testing.T) {
	tmpdir := t.TempDir()

	validation := &requests.CreateAccountValidation{
		AllowEmptyDisplayName: false,
	}

	request := &requests.CreateAccount{
		DisplayName:        "User-1",
		Password:           "password123",
		CustomizationColor: "#ffffff",
		RootDataDir:        tmpdir,
	}

	err := request.Validate(validation)
	require.NoError(t, err)

	request.RootDataDir = ""
	err = request.Validate(validation)
	require.ErrorIs(t, err, requests.ErrCreateAccountInvalidRootDataDir)
}

func TestRestoreKeycardAccountAndLogin(t *testing.T) {
	tmpdir := t.TempDir()

	exampleKeycardEvent := map[string]interface{}{
		"error":       "",
		"instanceUID": "a84599394887b742eed9a99d3834a797",
		"applicationInfo": map[string]interface{}{
			"initialized":    false,
			"instanceUID":    "",
			"version":        0,
			"availableSlots": 0,
			"keyUID":         "",
		},
		"seedPhraseIndexes": []interface{}{},
		"freePairingSlots":  0,
		"keyUid":            "0x579324c53f347e18961c775a00ec13ed7d59a225b1859d5125ff36b450b8778d",
		"pinRetries":        0,
		"pukRetries":        0,
		"cardMetadata": map[string]interface{}{
			"name":           "",
			"walletAccounts": []interface{}{},
		},
		"generatedWalletAccount": map[string]interface{}{
			"address":    "",
			"publicKey":  "",
			"privateKey": "",
		},
		"generatedWalletAccounts": []interface{}{},
		"txSignature": map[string]interface{}{
			"r": "",
			"s": "",
			"v": "",
		},
		"eip1581Key": map[string]interface{}{
			"address":    "0xA8d50f0B3bc581298446be8FBfF5c71684Ea6c01",
			"publicKey":  "0x040d7e6e3761ab3d17c220e484ede2f3fa02998b859d4d0e9d34216c6e41b03dc94996fdea23a9233092cee50a768e7428d5de7bd42e8e32c10d6b0e36b10f0e7a",
			"privateKey": "",
		},
		"encryptionKey": map[string]interface{}{
			"address":    "0x1ec12f2b323ddDD076A1127cEc8FA0B592c46cD3",
			"publicKey":  "0x04c4b16f670b51702dc130673bf9c64ffd1f69383cef2127dfa05031b9b1359120f7342134af9a350465126a85e87cb003b7c4f93d2ba2ff98bb73277b119c7a87",
			"privateKey": "68c830d5b327382a65e6c302594744ec0d28b01d1ea8124f49714f05c9625ddd"},
		"masterKey": map[string]interface{}{
			"address":    "0xbf9dE86774051537b2192Ce9c8d2496f129bA24b",
			"publicKey":  "0x040d909a07ecca18bbfa7d53d10a86bd956f54b8b446eabd94940e642ae18421b516ec5b63677c4ce65e0e266b58bdb716d8266b25356154eb61713ecb23824075",
			"privateKey": "",
		},
		"walletKey": map[string]interface{}{
			"address":    "0xB9E1998e1A8854887CA327D1aF5894B6CB0AC07D",
			"publicKey":  "0x04c16e7748f34e0ab2c9c13350d7872d928e942934dd8b8abd3fb12b8c742a5ee8cf0919731e800907068afec25f577bde3a9c534795e359ee48097e4e55f4aaca",
			"privateKey": "",
		},
		"walletRootKey": map[string]interface{}{
			"address":    "0xFf59db9F2f97Db7104A906C390D33C342a1309C8",
			"publicKey":  "0x04c436532398e19ed14b4eb41545b82014435d60e7db4449a371fd80d0d5cd557f60d81f6c2b35ca5440aa60934c23b70489b0e7963e63ec66b51a7e52db711262",
			"privateKey": "",
		},
		"whisperKey": map[string]interface{}{
			"address":    "0xBa122B9c0Ef560813b5D2C0961094aC36289f846",
			"publicKey":  "0x0441468c39b579259676350b9736b01cdadb740f67bfd022fa2b985123b1d66fc3191cfe73205e3d3d84148f0248f9a2978afeeda16d7c3db90bd2579f0de33459",
			"privateKey": "5a42b4f15ff1a5da95d116442ce11a31e9020f562224bf60b1d8d3a99d90653d",
		},
		"masterKeyAddress": "",
	}

	exampleRequest := map[string]interface{}{
		"mnemonic": "",
		"createAccountRequest": map[string]interface{}{
			"rootDataDir":   tmpdir,
			"kdfIterations": 256000,
			"deviceName":    "",
			"displayName":   "",
			"password":      "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
			"imagePath":     "",
			"imageCropRectangle": map[string]interface{}{
				"ax": 0, "ay": 0, "bx": 0, "by": 0},
			"customizationColor":       "primary",
			"emoji":                    "",
			"wakuV2Nameserver":         nil,
			"wakuV2LightClient":        false,
			"logLevel":                 "DEBUG",
			"logFilePath":              "",
			"logEnabled":               false,
			"previewPrivacy":           true,
			"verifyTransactionURL":     nil,
			"verifyENSURL":             nil,
			"verifyENSContractAddress": nil,
			"verifyTransactionChainID": nil,
			"upstreamConfig":           "",
			"networkID":                nil,
			"walletSecretsConfig": map[string]interface{}{
				"poktToken":                   "1234567890",
				"infuraToken":                 "1234567890",
				"infuraSecret":                "",
				"raribleMainnetApiKey":        "",
				"raribleTestnetApiKey":        "",
				"alchemyEthereumMainnetToken": "",
				"alchemyEthereumSepoliaToken": "",
				"alchemyArbitrumMainnetToken": "",
				"alchemyArbitrumSepoliaToken": "",
				"alchemyOptimismMainnetToken": "",
				"alchemyOptimismSepoliaToken": "",
				"alchemyBaseMainnetToken":     "",
				"alchemyBaseSepoliaToken":     "",
			},
			"torrentConfigEnabled":   false,
			"torrentConfigPort":      0,
			"keycardInstanceUID":     "a84599394887b742eed9a99d3834a797",
			"keycardPairingDataFile": path.Join(tmpdir, DefaultKeycardPairingDataFileRelativePath),
		},
	}

	require.NotNil(t, exampleKeycardEvent)
	require.NotNil(t, exampleRequest)

	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	backend := NewStatusBackend(testutils.MustCreateTestLogger())

	backend.UpdateRootDataDir(conf.RootDataDir)

	require.NoError(t, backend.OpenAccounts())

	keycardPairingDataFile := exampleRequest["createAccountRequest"].(map[string]interface{})["keycardPairingDataFile"].(string)

	kp := wallet.NewKeycardPairings()
	kp.SetKeycardPairingsFile(keycardPairingDataFile)

	err = kp.SetPairingsJSONFileContent([]byte(`{"a84599394887b742eed9a99d3834a797":{"key":"785d52957b05482477728380d9b4bbb5dc9a8ed978ab4a4098e1a279c855d3c6","index":1}}`))
	require.NoError(t, err)

	request := &requests.RestoreAccount{
		Keycard: &requests.KeycardData{
			KeyUID:              exampleKeycardEvent["keyUid"].(string),
			Address:             exampleKeycardEvent["masterKey"].(map[string]interface{})["address"].(string),
			WhisperPrivateKey:   exampleKeycardEvent["whisperKey"].(map[string]interface{})["privateKey"].(string),
			WhisperPublicKey:    exampleKeycardEvent["whisperKey"].(map[string]interface{})["publicKey"].(string),
			WhisperAddress:      exampleKeycardEvent["whisperKey"].(map[string]interface{})["address"].(string),
			WalletPublicKey:     exampleKeycardEvent["walletKey"].(map[string]interface{})["publicKey"].(string),
			WalletAddress:       exampleKeycardEvent["walletKey"].(map[string]interface{})["address"].(string),
			WalletRootAddress:   exampleKeycardEvent["walletRootKey"].(map[string]interface{})["address"].(string),
			Eip1581Address:      exampleKeycardEvent["eip1581Key"].(map[string]interface{})["address"].(string),
			EncryptionPublicKey: exampleKeycardEvent["encryptionKey"].(map[string]interface{})["publicKey"].(string),
		},
		CreateAccount: requests.CreateAccount{
			DisplayName:            "User-1",
			Password:               "password123",
			CustomizationColor:     "#ffffff",
			RootDataDir:            tmpdir,
			KeycardInstanceUID:     exampleKeycardEvent["instanceUID"].(string),
			KeycardPairingDataFile: &keycardPairingDataFile,
		},
	}

	acc, err := backend.RestoreKeycardAccountAndLogin(request)
	require.NoError(t, err)
	require.NotNil(t, acc)

	defer func() {
		assert.NoError(t, backend.Logout())
		assert.NoError(t, backend.StopNode())
	}()

	keycardKeyUID := exampleKeycardEvent["keyUid"].(string)

	require.Equal(t, "785d52957b05482477728380d9b4bbb5dc9a8ed978ab4a4098e1a279c855d3c6", acc.KeycardPairing,
		"Expected the pairing key resolved from the pairings file because prepareForKeycard looks it up by KeycardInstanceUID on desktop")

	// read back from the database, not from the returned struct: the field is set
	// before the account is saved, so the struct alone would not prove it persisted
	storedAcc, err := backend.multiaccountsDB.GetAccount(acc.KeyUID)
	require.NoError(t, err)
	require.Equal(t, acc.KeycardPairing, storedAcc.KeycardPairing,
		"Expected the pairing key persisted because the next login reads it from the multiaccounts database")

	require.Zero(t, countKeystoreFilesForKeyUID(t, tmpdir, keycardKeyUID),
		"Expected zero keystore files because a keycard restore must never write private keys to disk")

	db, err := accounts.NewDB(backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(keycardKeyUID)
	require.NoError(t, err)
	require.Equal(t, accsmanagementtypes.ColdWalletTypeStatusKeycard, keypair.ColdWallet,
		"Expected the restored profile keypair marked as status-keycard cold wallet because the restore request carried KeycardInstanceUID")
}

func countKeystoreFilesForKeyUID(t *testing.T, rootDataDir string, keyUID string) int {
	_, absolutePath := DefaultKeystorePath(rootDataDir, keyUID)
	entries, err := os.ReadDir(absolutePath)
	if os.IsNotExist(err) {
		return 0
	}
	require.NoError(t, err)
	return len(entries)
}

func keycardTestRestoreRequest(tmpdir string) *requests.RestoreAccount {
	pairingDataFile := path.Join(tmpdir, DefaultKeycardPairingDataFileRelativePath)
	return &requests.RestoreAccount{
		Keycard: &requests.KeycardData{
			KeyUID:              "0x579324c53f347e18961c775a00ec13ed7d59a225b1859d5125ff36b450b8778d",
			Address:             "0xbf9dE86774051537b2192Ce9c8d2496f129bA24b",
			WhisperPrivateKey:   "5a42b4f15ff1a5da95d116442ce11a31e9020f562224bf60b1d8d3a99d90653d",
			WhisperPublicKey:    "0x0441468c39b579259676350b9736b01cdadb740f67bfd022fa2b985123b1d66fc3191cfe73205e3d3d84148f0248f9a2978afeeda16d7c3db90bd2579f0de33459",
			WhisperAddress:      "0xBa122B9c0Ef560813b5D2C0961094aC36289f846",
			WalletPublicKey:     "0x04c16e7748f34e0ab2c9c13350d7872d928e942934dd8b8abd3fb12b8c742a5ee8cf0919731e800907068afec25f577bde3a9c534795e359ee48097e4e55f4aaca",
			WalletAddress:       "0xB9E1998e1A8854887CA327D1aF5894B6CB0AC07D",
			WalletRootAddress:   "0xFf59db9F2f97Db7104A906C390D33C342a1309C8",
			Eip1581Address:      "0xA8d50f0B3bc581298446be8FBfF5c71684Ea6c01",
			EncryptionPublicKey: "0x04c4b16f670b51702dc130673bf9c64ffd1f69383cef2127dfa05031b9b1359120f7342134af9a350465126a85e87cb003b7c4f93d2ba2ff98bb73277b119c7a87",
		},
		CreateAccount: requests.CreateAccount{
			DisplayName:            "User-1",
			Password:               "password123",
			CustomizationColor:     "#ffffff",
			RootDataDir:            tmpdir,
			KeycardInstanceUID:     "a84599394887b742eed9a99d3834a797",
			KeycardPairingDataFile: &pairingDataFile,
		},
	}
}

func TestRestoreKeycardAccountRejectedWhenKeycardNotInPairingsFile(t *testing.T) {
	tmpdir := t.TempDir()

	backend := NewStatusBackend(testutils.MustCreateTestLogger())
	backend.UpdateRootDataDir(tmpdir)
	require.NoError(t, backend.OpenAccounts())

	request := keycardTestRestoreRequest(tmpdir)

	kp := wallet.NewKeycardPairings()
	kp.SetKeycardPairingsFile(*request.KeycardPairingDataFile)
	err := kp.SetPairingsJSONFileContent([]byte(`{"someOtherInstanceUID":{"key":"785d52957b05482477728380d9b4bbb5dc9a8ed978ab4a4098e1a279c855d3c6","index":1}}`))
	require.NoError(t, err)

	// the restore opens the databases before it reaches the pairings-file check,
	// and StopNode does not close them
	t.Cleanup(func() { _ = backend.Logout() })

	_, err = backend.RestoreKeycardAccountAndLogin(request)
	require.ErrorContains(t, err, "keycard not found in pairings file",
		"Expected the restore to fail because the desktop pairing branch must reject a keycard absent from the pairings file")

	// Pins current behaviour: the databases and the wrapped-DEK file are written
	// before the pairings check runs, and a rejected restore does not remove them.
	keyUID := request.Keycard.KeyUID
	require.FileExists(t, envelope.Path(tmpdir, keyUID),
		"a rejected restore leaves the profile's wrapped-DEK file on disk")
	appDBPath, err := backend.getAppDBPath(keyUID)
	require.NoError(t, err)
	require.FileExists(t, appDBPath, "a rejected restore leaves the app database on disk")
	walletDBPath, err := backend.getWalletDBPath(keyUID)
	require.NoError(t, err)
	require.FileExists(t, walletDBPath, "a rejected restore leaves the wallet database on disk")

	// Sound either way: the leftover files are not reachable as a profile.
	accs, err := backend.GetAccounts()
	require.NoError(t, err)
	require.Empty(t, accs,
		"a rejected restore must write no multiaccount row, so the leftover files surface no profile")
}

func restoreSeedIntoKeycard(t *testing.T, tmpdir string, pairingKey string) (*StatusBackend, string, *multiaccounts.Account) {
	backend := NewStatusBackend(testutils.MustCreateTestLogger())
	backend.UpdateRootDataDir(tmpdir)
	require.NoError(t, backend.OpenAccounts())

	mnemonic, err := accsmanagementcommon.CreateRandomMnemonicWithDefaultLength()
	require.NoError(t, err)

	request := &requests.RestoreAccount{
		Mnemonic: mnemonic,
		CreateAccount: requests.CreateAccount{
			DisplayName:        "User-1",
			Password:           "password123",
			CustomizationColor: "#ffffff",
			RootDataDir:        tmpdir,
			KeycardInstanceUID: "a84599394887b742eed9a99d3834a797",
			KeycardPairingKey:  pairingKey,
		},
	}

	acc, err := backend.RestoreAccountAndLogin(request)
	require.NoError(t, err)
	require.NotNil(t, acc)

	return backend, mnemonic, acc
}

func TestRestoreSeedIntoKeycardUsesMobilePairingKey(t *testing.T) {
	tmpdir := t.TempDir()

	backend, _, acc := restoreSeedIntoKeycard(t, tmpdir, "mobile-pairing-key")
	defer func() {
		assert.NoError(t, backend.Logout())
		assert.NoError(t, backend.StopNode())
	}()

	require.Equal(t, "mobile-pairing-key", acc.KeycardPairing,
		"Expected KeycardPairingKey copied to the account because mobile has no pairings file")

	storedAcc, err := backend.multiaccountsDB.GetAccount(acc.KeyUID)
	require.NoError(t, err)
	require.Equal(t, "mobile-pairing-key", storedAcc.KeycardPairing,
		"Expected the pairing key persisted because the next login reads it from the multiaccounts database")

	require.Zero(t, countKeystoreFilesForKeyUID(t, tmpdir, acc.KeyUID),
		"Expected zero keystore files because a keycard account must never write private keys to disk")

	db, err := accounts.NewDB(backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(acc.KeyUID)
	require.NoError(t, err)
	require.Equal(t, accsmanagementtypes.ColdWalletTypeStatusKeycard, keypair.ColdWallet,
		"Expected the keypair marked as status-keycard cold wallet because the restore request carried KeycardInstanceUID")
}

func TestRestoreSeedIntoKeycardStoresWalletXPubAtWalletXPubPath(t *testing.T) {
	tmpdir := t.TempDir()

	backend, mnemonic, acc := restoreSeedIntoKeycard(t, tmpdir, "mobile-pairing-key")
	defer func() {
		assert.NoError(t, backend.Logout())
		assert.NoError(t, backend.StopNode())
	}()

	expectedXPub, err := generator.DeriveExtendedPublicKeyAtPath(mnemonic, "", accsmanagementcommon.PathWalletXPub)
	require.NoError(t, err)

	db, err := accounts.NewDB(backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(acc.KeyUID)
	require.NoError(t, err)
	require.Equal(t, expectedXPub, keypair.XPub,
		"Expected the stored xpub derived at m/44'/60'/0' because accounts later derived from it must match the wallet account addresses")
}

func setupLoggedOutAccountWithEmptyProfileXPub(t *testing.T) *setupContext {
	testContext := setupTestContext(t, testPassword, false, false, false)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)

	_, err = testContext.backend.appDB.Exec("UPDATE keypairs SET xpub = '' WHERE key_uid = ?", testContext.profileKeypair.KeyUID)
	require.NoError(t, err)

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	return testContext
}

// setupLoggedOutKeycardProfileWithoutXPub converts the profile to a keycard and
// clears its stored xpub, so nothing local can derive one and a login request is
// the only possible source.
func setupLoggedOutKeycardProfileWithoutXPub(t *testing.T) *setupContext {
	testContext := setupTestContext(t, testPassword, false, false, true)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}
	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)

	keycardAccount := *testContext.multiAcc
	keycardAccount.KeycardPairing = "pairing"
	require.NoError(t, testContext.backend.ConvertToKeycardAccount(keycardAccount, settings.Settings{},
		testContext.profileKeypair.KeyUID, testPassword, testKeycardPassword))

	_, err = testContext.backend.appDB.Exec("UPDATE keypairs SET xpub = '' WHERE key_uid = ?", testContext.profileKeypair.KeyUID)
	require.NoError(t, err)

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	require.Zero(t, countKeystoreFilesForKeyUID(t, testContext.config.RootDataDir, testContext.profileKeypair.KeyUID),
		"precondition: the conversion removed the keystore files, so nothing local can supply the xpub")

	return testContext
}

func (s *setupContext) loginWithWalletXPub(t *testing.T, xpub string) error {
	return s.backend.loginAccount(&requests.Login{
		KeyUID:                   s.profileKeypair.KeyUID,
		Password:                 testKeycardPassword,
		KeycardWhisperPrivateKey: strings.TrimPrefix(s.chatPrivateKey, "0x"),
		WalletXPub:               xpub,
	})
}

func TestLoginStoresWalletXPubFromRequestWhenKeypairXPubMissing(t *testing.T) {
	testContext := setupLoggedOutKeycardProfileWithoutXPub(t)

	// the client reads this from the card and sends it at login
	requestXPub, err := generator.DeriveExtendedPublicKeyAtPath(testContext.mnemonic, "", accsmanagementcommon.PathWalletXPub)
	require.NoError(t, err)

	require.NoError(t, testContext.loginWithWalletXPub(t, requestXPub))
	defer func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	}()

	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Equal(t, requestXPub, keypair.XPub,
		"Expected the request-provided xpub stored because a keycard profile has no keystore file to derive it from")
}

func TestLoginRejectsGarbageWalletXPub(t *testing.T) {
	testContext := setupLoggedOutAccountWithEmptyProfileXPub(t)

	err := testContext.backend.loginAccount(&requests.Login{
		KeyUID:     testContext.profileKeypair.KeyUID,
		Password:   testPassword,
		WalletXPub: "not-an-extended-key",
	})
	defer func() {
		_ = testContext.backend.Logout()
		_ = testContext.backend.StopNode()
	}()
	require.ErrorContains(t, err, "invalid wallet xpub provided in the login request",
		"Expected the login to abort because an unparseable xpub must not be stored on the keypair")

	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Empty(t, keypair.XPub,
		"Expected the stored xpub untouched because the rejected login must not persist the invalid value")
}

func TestLoginRejectsPrivateExtendedKeyAsWalletXPub(t *testing.T) {
	testContext := setupLoggedOutAccountWithEmptyProfileXPub(t)

	otherMnemonic, err := accsmanagementcommon.CreateRandomMnemonicWithDefaultLength()
	require.NoError(t, err)
	otherMasterAcc, err := generator.CreateAccountFromMnemonic(otherMnemonic, "")
	require.NoError(t, err)
	privateExtendedKey := otherMasterAcc.ExtendedKey().String()

	err = testContext.backend.loginAccount(&requests.Login{
		KeyUID:     testContext.profileKeypair.KeyUID,
		Password:   testPassword,
		WalletXPub: privateExtendedKey,
	})
	defer func() {
		_ = testContext.backend.Logout()
		_ = testContext.backend.StopNode()
	}()
	require.ErrorContains(t, err, "private extended key provided as the wallet xpub",
		"Expected the login to abort because a private extended key in the xpub field would leak signing capability into the DB")

	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Empty(t, keypair.XPub,
		"Expected the stored xpub untouched because the rejected login must not persist the private key")
}

func TestKeycardReLoginViaLoginAccountAfterConversion(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)

	keycardAccount := *testContext.multiAcc
	keycardAccount.KeycardPairing = "pairing"
	err = testContext.backend.ConvertToKeycardAccount(keycardAccount, settings.Settings{}, testContext.profileKeypair.KeyUID, testPassword, testKeycardPassword)
	require.NoError(t, err)

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	c := make(chan interface{}, 10)
	signal.SetHandler(func(data []byte) {
		if strings.Contains(string(data), signal.EventLoggedIn) {
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetHandler)

	err = testContext.backend.LoginAccount(&requests.Login{
		KeyUID:                   testContext.profileKeypair.KeyUID,
		Password:                 testKeycardPassword,
		KeycardWhisperPrivateKey: strings.TrimPrefix(testContext.chatPrivateKey, "0x"),
	})
	require.NoError(t, err,
		"Expected the modern login request path to succeed for a converted keycard account because this is every keycard user's next app launch")
	defer func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	}()

	select {
	case <-c:
	case <-time.After(5 * time.Second):
		t.Fatal("Expected EventLoggedIn within 5s because LoginAccount must signal a successful keycard login")
	}

	require.Zero(t, countKeystoreFilesForKeyUID(t, testContext.config.RootDataDir, testContext.profileKeypair.KeyUID),
		"Expected zero keystore files because the keycard login must not rely on or recreate on-disk private keys")
}

func TestLostKeycardLoginWithMnemonicRecoversKeycardAccount(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)

	genAcc, _, err := testContext.backend.generateAccount(testContext.mnemonic)
	require.NoError(t, err)
	_, derivedInfo, err := testContext.backend.generateDerivedAddresses(genAcc, paths)
	require.NoError(t, err)
	encryptionPublicKey := derivedInfo[accsmanagementcommon.PathEIP1581Encryption].PublicKey

	keycardAccount := *testContext.multiAcc
	keycardAccount.KeycardPairing = "pairing"
	err = testContext.backend.ConvertToKeycardAccount(keycardAccount, settings.Settings{}, testContext.profileKeypair.KeyUID, testPassword, encryptionPublicKey)
	require.NoError(t, err)

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	err = testContext.backend.loginAccount(&requests.Login{
		KeyUID:   testContext.profileKeypair.KeyUID,
		Mnemonic: testContext.mnemonic,
	})
	require.NoError(t, err,
		"Expected seed-phrase login to succeed because it is the documented lost-keycard recovery route")
	defer func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	}()

	require.Equal(t, testContext.profileKeypair.KeyUID, testContext.backend.account.KeyUID,
		"Expected the logged-in account to be the keycard account because the mnemonic replaces password and whisper key")
}

func TestDeleteMultiaccount(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, true)

	rootDataDir := testContext.backend.rootDataDir
	keyStoreDir := filepath.Join(rootDataDir, "keystore")

	err := testContext.backend.OpenAccounts()
	require.NoError(t, err)

	files, err := os.ReadDir(rootDataDir)
	require.NoError(t, err)
	require.NotEqual(t, 3, len(files))

	err = testContext.backend.DeleteMultiaccount(testContext.profileKeypair.KeyUID, keyStoreDir)
	require.NoError(t, err)

	files, err = os.ReadDir(rootDataDir)
	require.NoError(t, err)
	require.Equal(t, 3, len(files))
}

func TestBackendConnectionChangesConcurrently(t *testing.T) {
	connections := [...]string{connection.Wifi, connection.Cellular, connection.Unknown}
	testContext := setupTestContext(t, testPassword, true, true, true)

	count := 3

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			connIdx := rand.Intn(len(connections)) // nolint: gosec
			testContext.backend.ConnectionChange(connections[connIdx], false)
		}()
	}

	wg.Wait()
}

func TestBackendConnectionChangesToOffline(t *testing.T) {
	testContext := setupTestContext(t, testPassword, true, true, true)

	testContext.backend.ConnectionChange(connection.None, false)
	assert.True(t, testContext.backend.connectionState.Offline)

	testContext.backend.ConnectionChange(connection.Wifi, false)
	assert.False(t, testContext.backend.connectionState.Offline)

	testContext.backend.ConnectionChange("unknown-state", false)
	assert.False(t, testContext.backend.connectionState.Offline)
}

const testKeycardPassword = "222222"

func setupKeycardConvertedContext(t *testing.T) *setupContext {
	testContext := setupTestContext(t, testPassword, false, false, true)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)

	keycardAccount := *testContext.multiAcc
	keycardAccount.KeycardPairing = "pairing"
	err = testContext.backend.ConvertToKeycardAccount(keycardAccount, settings.Settings{}, testContext.profileKeypair.KeyUID, testPassword, testKeycardPassword)
	require.NoError(t, err)

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	chatPrivKey := strings.TrimPrefix(testContext.chatPrivateKey, "0x")
	require.NoError(t, testContext.backend.StartNodeWithKey(*testContext.multiAcc, testKeycardPassword, chatPrivKey, testContext.config))

	t.Cleanup(func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	})

	return testContext
}

func TestConvertToKeycardAccountRequiresMessenger(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	keycardAccount := *testContext.multiAcc
	err := testContext.backend.ConvertToKeycardAccount(keycardAccount, settings.Settings{}, testContext.profileKeypair.KeyUID, testPassword, testKeycardPassword)
	require.ErrorContains(t, err, "cannot resolve messenger instance",
		"Expected the conversion to be rejected because without a logged-in messenger no keypair migration can run")
}

func TestConvertToKeycardAccountRejectsWrongOldPassword(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}

	_, err := testContext.backend.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)
	// the test logs out mid-way and then reopens the databases to read the keypair
	// back, so cleanup has to cover whichever state it ends in
	t.Cleanup(func() {
		_ = testContext.backend.Logout()
		_ = testContext.backend.StopNode()
	})

	keyUID := testContext.profileKeypair.KeyUID
	keystoreFilesBefore := countKeystoreFilesForKeyUID(t, testContext.config.RootDataDir, keyUID)
	require.Greater(t, keystoreFilesBefore, 0,
		"Expected keystore files before conversion because the profile keypair is password-based")

	keycardAccount := *testContext.multiAcc
	keycardAccount.KeycardPairing = "wrong-password-pairing"
	err = testContext.backend.ConvertToKeycardAccount(keycardAccount, settings.Settings{}, keyUID, "wrong-password", testKeycardPassword)
	require.ErrorIs(t, err, envelope.ErrInvalidKEK,
		"Expected the conversion to fail on the key-encryption envelope because the old password is wrong")

	// Pins current behaviour, which is the defect in #7698: the pairing is written
	// before anything is verified and is not rolled back. When #7698 is fixed this
	// assertion must flip to Empty - it is here so the fix cannot land unnoticed.
	storedAcc, err := testContext.backend.multiaccountsDB.GetAccount(keyUID)
	require.NoError(t, err)
	require.Equal(t, "wrong-password-pairing", storedAcc.KeycardPairing,
		"#7698: a failed conversion still leaves KeycardPairing set. If this fails, #7698 is fixed - change this to require.Empty")

	masterAddress := types.HexToAddress(testContext.profileKeypair.DerivedFrom)
	ok, err := testContext.backend.AccountsManager().VerifyAccountPassword(masterAddress, testPassword)
	require.NoError(t, err,
		"Expected the master keystore file to survive a failed conversion because deletion must come after password verification")
	require.True(t, ok)

	for _, acc := range testContext.profileKeypair.Accounts {
		ok, err = testContext.backend.AccountsManager().VerifyAccountPassword(acc.Address, testPassword)
		require.NoError(t, err,
			"Expected every account keystore file to survive a failed conversion")
		require.True(t, ok)
	}

	require.Equal(t, keystoreFilesBefore, countKeystoreFilesForKeyUID(t, testContext.config.RootDataDir, keyUID),
		"Expected the keystore file count unchanged because a failed conversion must not delete keys")

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	require.NoError(t, testContext.backend.VerifyDatabasePassword(keyUID, testPassword),
		"Expected the DB password unchanged because ChangeDatabasePassword must not run after a failed conversion")

	require.NoError(t, testContext.backend.ensureDBsOpened(*testContext.multiAcc, testPassword))
	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)

	keypair, err := db.GetKeypairByKeyUID(keyUID)
	require.NoError(t, err)
	require.False(t, keypair.MigratedToColdWallet(),
		"Expected the keypair to stay non-cold because the migration failed before flipping state")
}

func TestConvertToRegularAccountRejectsUnknownMnemonic(t *testing.T) {
	testContext := setupKeycardConvertedContext(t)
	keyUID := testContext.profileKeypair.KeyUID

	otherMnemonic, err := accsmanagementcommon.CreateRandomMnemonicWithDefaultLength()
	require.NoError(t, err)

	err = testContext.backend.ConvertToRegularAccount(otherMnemonic, testKeycardPassword, testPassword)
	require.ErrorIs(t, err, sql.ErrNoRows,
		"Expected the convert-back to fail looking up the unknown keyUID, not for some earlier unrelated reason")

	multiAccAfter, err := testContext.backend.multiaccountsDB.GetAccount(keyUID)
	require.NoError(t, err)
	require.Equal(t, "pairing", multiAccAfter.KeycardPairing,
		"Expected the keycard pairing intact because an unknown mnemonic must not touch this profile")

	require.Zero(t, countKeystoreFilesForKeyUID(t, testContext.config.RootDataDir, keyUID),
		"Expected zero keystore files because a failed convert-back must not recreate keys")

	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(keyUID)
	require.NoError(t, err)
	require.True(t, keypair.MigratedToColdWallet(),
		"Expected the keypair to stay cold-wallet migrated because the convert-back failed before any mutation")
}

func TestConvertToRegularAccountAcceptsWhitespacePaddedMnemonic(t *testing.T) {
	testContext := setupKeycardConvertedContext(t)
	keyUID := testContext.profileKeypair.KeyUID

	paddedMnemonic := "  " + strings.ReplaceAll(testContext.mnemonic, " ", "   ") + "  "
	err := testContext.backend.ConvertToRegularAccount(paddedMnemonic, testKeycardPassword, testPassword)
	require.NoError(t, err,
		"Expected the convert-back to succeed because mnemonic whitespace must be normalized before derivation")

	require.Greater(t, countKeystoreFilesForKeyUID(t, testContext.config.RootDataDir, keyUID), 0,
		"Expected keystore files recreated because whitespace-padded mnemonics must be normalized and accepted")

	require.NoError(t, testContext.backend.ensureDBsOpened(*testContext.multiAcc, testPassword))
	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(keyUID)
	require.NoError(t, err)
	require.False(t, keypair.MigratedToColdWallet(),
		"Expected the keypair back to password-based because the padded mnemonic resolves to the same account")
}

// Pins current behaviour: backfillKeypairsXPubOnLogin checks only that the value
// parses and is not private, so an xpub belonging to a different wallet tree is
// stored permanently. Since #7670, accounts derive from this xpub without a
// password, so a foreign value produces addresses whose keys nobody holds.
// If this fails, validation has been added - change it to require rejection.
func TestLoginAcceptsWalletXPubFromAnotherMnemonic(t *testing.T) {
	testContext := setupLoggedOutKeycardProfileWithoutXPub(t)

	otherMnemonic, err := accsmanagementcommon.CreateRandomMnemonicWithDefaultLength()
	require.NoError(t, err)
	foreignXPub, err := generator.DeriveExtendedPublicKeyAtPath(otherMnemonic, "", accsmanagementcommon.PathWalletXPub)
	require.NoError(t, err)

	ownXPub, err := generator.DeriveExtendedPublicKeyAtPath(testContext.mnemonic, "", accsmanagementcommon.PathWalletXPub)
	require.NoError(t, err)
	require.NotEqual(t, ownXPub, foreignXPub, "precondition: the two mnemonics derive different xpubs")

	require.NoError(t, testContext.loginWithWalletXPub(t, foreignXPub))
	defer func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	}()

	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)
	keypair, err := db.GetKeypairByKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Equal(t, foreignXPub, keypair.XPub,
		"an xpub from an unrelated mnemonic is accepted and stored: nothing checks it against the profile")
}

func TestLoginWithMnemonicRejectsMismatchedKeyUID(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, false)

	otherMnemonic, err := accsmanagementcommon.CreateRandomMnemonicWithDefaultLength()
	require.NoError(t, err)

	err = testContext.backend.loginAccount(&requests.Login{
		KeyUID:   testContext.profileKeypair.KeyUID,
		Mnemonic: otherMnemonic,
	})
	require.ErrorContains(t, err, "mnemonic does not match this account",
		"Expected the login to fail because a mnemonic deriving a different keyUID must not unlock this account")
}

// TestConvertToKeycardAccountRejectsWrongOldPassword fails at the envelope, before
// the settings writes. A legacy profile gets past them: ensureDBsOpened does not
// verify a legacy password while a session is open, so the seed phrase is cleared
// and only the keypair migration rejects the password. Pins the behaviour
// reported on #7698.
func TestConvertToKeycardAccountOnLegacyProfileWipesMnemonicOnWrongPassword(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)
	b := testContext.backend

	request := &requests.CreateAccount{
		RootDataDir:   testContext.config.RootDataDir,
		Password:      testPassword,
		KdfIterations: 1,
	}
	_, err := b.StartNodeWithChatKeyOrMnemonic(request, testContext.mnemonic, nil, false)
	require.NoError(t, err)

	accountsList, err := b.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accountsList, 1)
	multiAcc := accountsList[0]
	keyUID := multiAcc.KeyUID

	accountDB, err := accounts.NewDB(b.appDB)
	require.NoError(t, err)
	require.NoError(t, accountDB.SaveSettingField(settings.Mnemonic, testContext.mnemonic))

	// back to the legacy scheme, as an older app version would have left the profile
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())
	demigrateProfileForTest(t, b, keyUID, testPassword)
	require.False(t, b.ProfileEncryptionInfo(keyUID))

	chatPrivKey := strings.TrimPrefix(testContext.chatPrivateKey, "0x")
	require.NoError(t, b.StartNodeWithKey(multiAcc, testPassword, chatPrivKey, testContext.config))
	defer func() {
		_ = b.Logout()
		_ = b.StopNode()
	}()

	keycardAccount := multiAcc
	keycardAccount.KeycardPairing = "legacy-wrong-password-pairing"
	err = b.ConvertToKeycardAccount(keycardAccount, settings.Settings{}, keyUID, "wrong-password", testKeycardPassword)
	require.ErrorContains(t, err, "incorrect password provided",
		"a legacy profile only rejects the password at the keypair migration, two steps after the settings writes")

	db, err := accounts.NewDB(b.appDB)
	require.NoError(t, err)
	storedMnemonic, err := db.Mnemonic()
	require.NoError(t, err)
	require.Empty(t, storedMnemonic,
		"#7698: the seed phrase is cleared before the password is checked, so a wrong password loses it. If this fails, #7698 is fixed - change this to require.Equal against the seeded mnemonic")

	storedAcc, err := b.multiaccountsDB.GetAccount(keyUID)
	require.NoError(t, err)
	require.Equal(t, "legacy-wrong-password-pairing", storedAcc.KeycardPairing,
		"#7698: the pairing is written first and not rolled back")
}
