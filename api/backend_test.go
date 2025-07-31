package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/status-im/status-go/accounts-management/keystore/geth"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/services/wallet"
	walletservice "github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/walletdatabase"
)

func setupTestDB() (*sql.DB, func() error, error) {
	return helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "tests")
}

func setupTestWalletDB() (*sql.DB, func() error, error) {
	return helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "tests")
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

func setupGethStatusBackend() (*GethStatusBackend, func() error, func() error, func() error, error) {
	db, stop1, err := setupTestDB()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	backend := NewGethStatusBackend(tt.MustCreateTestLogger())
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

func handleError(t *testing.T, err error) {
	if err != nil {
		t.Logf("deferred function error: '%s'", err)
	}
}

func TestBackendStartNodeConcurrently(t *testing.T) {
	utils.Init()

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

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
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
	utils.Init()

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

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
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

// TODO(adam): add concurrent tests for ResetChainData()

func TestBackendGettersConcurrently(t *testing.T) {
	utils.Init()

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

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
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

func TestBackendConnectionChangesConcurrently(t *testing.T) {
	connections := [...]string{connection.Wifi, connection.Cellular, connection.Unknown}
	backend := NewGethStatusBackend(tt.MustCreateTestLogger())

	count := 3

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			connIdx := rand.Intn(len(connections)) // nolint: gosec
			backend.ConnectionChange(connections[connIdx], false)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestBackendConnectionChangesToOffline(t *testing.T) {
	b := NewGethStatusBackend(tt.MustCreateTestLogger())

	b.ConnectionChange(connection.None, false)
	assert.True(t, b.connectionState.Offline)

	b.ConnectionChange(connection.Wifi, false)
	assert.False(t, b.connectionState.Offline)

	b.ConnectionChange("unknown-state", false)
	assert.False(t, b.connectionState.Offline)
}

func TestBackendCallRPCConcurrently(t *testing.T) {
	utils.Init()

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

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
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
			result, err := backend.CallRPC(fmt.Sprintf(
				`{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":%d}`,
				idx+1,
			))
			assert.NoError(t, err)
			assert.NotContains(t, result, "error")
			wg.Done()
		}(i)

		wg.Add(1)
		go func(idx int) {
			result, err := backend.CallPrivateRPC(fmt.Sprintf(
				`{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":%d}`,
				idx+1,
			))
			assert.NoError(t, err)
			assert.NotContains(t, result, "error")
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestAppStateChange(t *testing.T) {
	backend := NewGethStatusBackend(tt.MustCreateTestLogger())

	var testCases = []struct {
		name          string
		fromState     AppState
		toState       AppState
		expectedState AppState
	}{
		{
			name:          "success",
			fromState:     AppStateInactive,
			toState:       AppStateBackground,
			expectedState: AppStateBackground,
		},
		{
			name:          "invalid state",
			fromState:     AppStateInvalid,
			toState:       "unexisting",
			expectedState: AppStateInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			backend.appState = tc.fromState
			backend.AppStateChange(tc.toState)
			assert.Equal(t, tc.expectedState.String(), backend.appState.String())
		})
	}
}

func TestBlockedRPCMethods(t *testing.T) {
	utils.Init()

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

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)

	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() { require.NoError(t, backend.StopNode()) }()

	for idx, m := range rpc.BlockedMethods() {
		result, err := backend.CallRPC(fmt.Sprintf(
			`{"jsonrpc":"2.0","method":"%s","params":[],"id":%d}`,
			m,
			idx+1,
		))
		assert.NoError(t, err)
		assert.Contains(t, result, fmt.Sprintf(`{"code":-32700,"message":"%s"}`, rpc.ErrMethodNotFound))
	}
}

func TestCallRPCWithStoppedNode(t *testing.T) {
	backend := NewGethStatusBackend(tt.MustCreateTestLogger())

	resp, err := backend.CallRPC(
		`{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}`,
	)
	assert.Equal(t, ErrRPCClientUnavailable, err)
	assert.Equal(t, "", resp)

	resp, err = backend.CallPrivateRPC(
		`{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}`,
	)
	assert.Equal(t, ErrRPCClientUnavailable, err)
	assert.Equal(t, "", resp)
}

// TODO(adam): add concurrent tests for: SendTransaction

func TestStartStopMultipleTimes(t *testing.T) {
	utils.Init()

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

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)

	config.NoDiscovery = false
	// doesn't have to be running. just any valid enode to bypass validation.
	config.ClusterConfig.BootNodes = []string{
		"enode://e8a7c03b58911e98bbd66accb2a55d57683f35b23bf9dfca89e5e244eb5cc3f25018b4112db507faca34fb69ffb44b362f79eda97a669a8df29c72e654416784@0.0.0.0:30404",
	}
	require.NoError(t, err)
	require.NoError(t, backend.StartNode(config))
	require.NoError(t, backend.StopNode())
	require.NoError(t, backend.StartNode(config))
	require.NoError(t, backend.StopNode())
}

func TestHashTypedData(t *testing.T) {
	utils.Init()

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

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
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
		"chainId":           json.RawMessage(fmt.Sprintf("%d", params.StatusChainNetworkID)),
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
	utils.Init()

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
		require.EqualError(t, err, wallettypes.ErrAccountDoesntExist.Error())
		require.Nil(t, key)
	})

	t.Run("PasswordDoesntMatch", func(t *testing.T) {
		pkey, err := crypto.GenerateKey()
		require.NoError(t, err)
		privateKeyHex := types.EncodeHex(crypto.FromECDSA(pkey))
		address := crypto.PubkeyToAddress(pkey.PublicKey)
		keyUIDHex := sha256.Sum256(gethcrypto.FromECDSAPub(&pkey.PublicKey))
		keyUID := types.EncodeHex(keyUIDHex[:])

		db, err := accounts.NewDB(testContext.backend.appDB)
		require.NoError(t, err)

		_, err = testContext.backend.AccountsManager().CreateFromPrivateKeyAndStoreAccount(privateKeyHex, testPassword)
		require.NoError(t, err)

		require.NoError(t, db.SaveOrUpdateKeypair(&accounts.Keypair{
			KeyUID: keyUID,
			Name:   "private key keypair",
			Type:   accounts.KeypairTypeKey,
			Accounts: []*accounts.Account{
				{
					Address: address,
					KeyUID:  keyUID,
				},
			},
		}))
		key, err := testContext.backend.getVerifiedWalletAccount(address.String(), "wrong-password")
		require.EqualError(t, err, geth.ErrDecrypt.Error())
		require.Nil(t, key)
	})

	t.Run("PartialAccount", func(t *testing.T) {
		// Create a derived wallet account without storing the keys
		db, err := accounts.NewDB(testContext.backend.appDB)
		require.NoError(t, err)
		newPath := "m/0"
		walletRootAddress, err := db.GetWalletRootAddress()
		require.NoError(t, err)

		genAcc, err := testContext.backend.AccountsManager().LoadAccount(walletRootAddress, testPassword)
		require.NoError(t, err)

		walletInfo := genAcc.ToIdentifiedAccountInfo()

		derivedAcc, err := testContext.backend.AccountsManager().DeriveChildAccountForPathAndStore(walletRootAddress, newPath, testPassword)
		require.NoError(t, err)

		derivedInfo := derivedAcc.ToAccountInfo()

		keypair := &accounts.Keypair{
			KeyUID: walletInfo.KeyUID,
			Name:   "profile keypair",
			Type:   accounts.KeypairTypeProfile,
			Accounts: []*accounts.Account{
				{
					Address:   types.HexToAddress(derivedInfo.Address),
					KeyUID:    walletInfo.KeyUID,
					Type:      accounts.AccountTypeGenerated,
					PublicKey: types.Hex2Bytes(derivedInfo.PublicKey),
					Path:      newPath,
					Wallet:    false,
					Name:      "PartialAccount",
				},
			},
		}
		require.NoError(t, db.SaveOrUpdateKeypair(keypair))

		// With partial account we need to dynamically generate private key
		acc, err := testContext.backend.getVerifiedWalletAccount(keypair.Accounts[0].Address.Hex(), testPassword)
		require.NoError(t, err)
		require.Equal(t, keypair.Accounts[0].Address, acc.Address())
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

		_, err = testContext.backend.AccountsManager().CreateFromPrivateKeyAndStoreAccount(privateKeyHex, testPassword)
		require.NoError(t, err)

		require.NoError(t, db.SaveOrUpdateKeypair(&accounts.Keypair{
			KeyUID: keyUID,
			Name:   "private key keypair",
			Type:   accounts.KeypairTypeKey,
			Accounts: []*accounts.Account{
				{
					Address: address,
					KeyUID:  keyUID,
				},
			},
		}))
		acc, err := testContext.backend.getVerifiedWalletAccount(address.String(), testPassword)
		require.NoError(t, err)
		require.Equal(t, address, acc.Address())
	})
}

func TestRuntimeLogLevelIsNotWrittenToDatabase(t *testing.T) {
	utils.Init()

	testContext := setupTestContext(t, testPassword, false, false, false)

	json := `{
		"NetworkId": 3,
		"DataDir": "` + testContext.config.DataDir + `",
		"KeycardPairingDataFile": "` + path.Join(testContext.config.DataDir, "keycard/pairings.json") + `",
		"NoDiscovery": true,
		"TorrentConfig": {
			"Port": 9025,
			"Enabled": false,
			"DataDir": "` + testContext.config.DataDir + `/archivedata",
			"TorrentDir": "` + testContext.config.DataDir + `/torrents"
		},
		"RuntimeLogLevel": "INFO",
		"LogLevel": "DEBUG"
	}`

	newConf, err := params.NewConfigFromJSON(json)
	require.NoError(t, err)
	require.Equal(t, "INFO", newConf.RuntimeLogLevel)

	require.NoError(t, testContext.backend.OpenAccounts())
	require.NotNil(t, testContext.backend.statusNode.HTTPServer())

	err = testContext.backend.ensureDBsOpened(*testContext.multiAcc, testPassword)
	require.NoError(t, err)

	require.NoError(t, testContext.backend.StartNodeWithAccountAndInitialConfig(testContext.mnemonic, *testContext.multiAcc, testPassword, testContext.settings, newConf, testContext.profileKeypair, nil))
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
	utils.Init()

	testContext := setupTestContext(t, testPassword, false, false, false)

	nameserver := "8.8.8.8"

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.DataDir,
		LogFilePath:        testContext.config.DataDir + "/log",
		WakuV2Nameserver:   &nameserver,
		WakuV2Fleet:        "status.staging",
	}

	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), signal.EventLoggedIn) {
			require.Contains(t, string(data), "status.staging")
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetMobileSignalHandler)
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

	accountsDB, err := testContext.backend.accountsDB()
	require.NoError(t, err)
	backupFecthed, err := accountsDB.BackupFetched()
	require.NoError(t, err)
	require.True(t, backupFecthed)

	require.True(t, acc.HasAcceptedTerms)

	waitForLogin(c)
	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	testContext.backend.UpdateRootDataDir(testContext.config.DataDir)

	accounts, err := testContext.backend.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)

	require.NotEmpty(t, accounts[0].KeyUID)
	require.Equal(t, acc.KeyUID, accounts[0].KeyUID)

	loginAccountRequest := &requests.Login{
		KeyUID:           accounts[0].KeyUID,
		Password:         testPassword,
		WakuV2Nameserver: nameserver,
	}
	err = testContext.backend.LoginAccount(loginAccountRequest)
	require.NoError(t, err)
	waitForLogin(c)
	require.Equal(t, nameserver, testContext.backend.config.WakuV2Config.Nameserver)
}

func TestVerifyDatabasePassword(t *testing.T) {
	utils.Init()

	testContext := setupTestContext(t, testPassword, false, false, false)

	require.NoError(t, testContext.backend.StartNodeWithAccountAndInitialConfig(testContext.mnemonic, *testContext.multiAcc, testPassword,
		testContext.settings, testContext.config, testContext.profileKeypair, nil))
	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	require.Error(t, testContext.backend.VerifyDatabasePassword(testContext.profileKeypair.KeyUID, "wrong-pass"))
	require.NoError(t, testContext.backend.VerifyDatabasePassword(testContext.profileKeypair.KeyUID, testPassword))
}

func TestConvertAccount(t *testing.T) {
	testContext := setupTestContext(t, testPassword, false, false, true)

	err := testContext.backend.StartNodeWithAccountAndInitialConfig(testContext.mnemonic, *testContext.multiAcc, testPassword,
		testContext.settings, testContext.config, testContext.profileKeypair, nil)
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
	err = testContext.backend.ensureAppDBOpened(*testContext.multiAcc, testPassword)
	require.NoError(t, err)

	// db creation
	db, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)

	// Check that there is no registered keycards
	keycards, err := db.GetKeycardsWithSameKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 0, len(keycards))

	keycardAccount := *testContext.multiAcc
	keycardAccount.KeycardPairing = "pairing"

	keycardSettings := settings.Settings{
		KeycardInstanceUID: "0xdeadbeef",
		KeycardPairedOn:    1,
		KeycardPairing:     "pairing",
	}

	// Converting to a keycard account
	const keycardPassword = "222222" // represents password for a keycard user
	err = testContext.backend.ConvertToKeycardAccount(keycardAccount, keycardSettings, testContext.profileKeypair.KeyUID, testPassword, keycardPassword)
	require.NoError(t, err)

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

	require.NoError(t, testContext.backend.Logout())
	require.NoError(t, testContext.backend.StopNode())

	chatPrivKey := strings.TrimPrefix(testContext.chatPrivateKey, "0x")
	require.NoError(t, testContext.backend.StartNodeWithKey(*testContext.multiAcc, keycardPassword, chatPrivKey, testContext.config))

	defer func() {
		assert.NoError(t, testContext.backend.Logout())
		assert.NoError(t, testContext.backend.StopNode())
	}()

	// Ensure we're able to open the DB
	err = testContext.backend.ensureAppDBOpened(keycardAccount, keycardPassword)
	require.NoError(t, err)

	// db creation after re-encryption
	db1, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)

	// Check that there is a registered keycard
	keycards, err = db1.GetKeycardsWithSameKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 1, len(keycards))

	// Converting to a regular account
	err = testContext.backend.ConvertToRegularAccount(testContext.mnemonic, keycardPassword, testPassword)
	require.NoError(t, err)

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
	err = testContext.backend.ensureAppDBOpened(keycardAccount, testPassword)
	require.NoError(t, err)

	// db creation after re-encryption
	db2, err := accounts.NewDB(testContext.backend.appDB)
	require.NoError(t, err)

	// Check that there is no registered keycards
	keycards, err = db2.GetKeycardsWithSameKeyUID(testContext.profileKeypair.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 0, len(keycards))
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

	b := NewGethStatusBackend(tt.MustCreateTestLogger())

	b.UpdateRootDataDir(conf.DataDir)

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
	require.NotNil(t, b.statusNode.HTTPServer())
	require.NoError(t, b.StopNode())

}

func TestLoginAndMigrationsStillWorkWithExistingDesktopUser(t *testing.T) {
	utils.Init()

	keyUID := "0x7c46c8f6f059ab72d524f2a6d356904db30bb0392636172ab3929a6bd2220f84" // #nosec G101

	srcFolder := "../static/test-0.132.0-account/"

	tmpdir := t.TempDir()
	copyDir(srcFolder, tmpdir, t)

	keystoreDir := path.Join(tmpdir, "keystore", keyUID)
	err := os.MkdirAll(keystoreDir, 0700)
	require.NoError(t, err)

	srcKeystoreFolder := "../static/test-0.132.0-account/keystore/"
	copyDir(srcKeystoreFolder, keystoreDir, t)

	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	loginDesktopUser(t, conf, keyUID)
	loginDesktopUser(t, conf, keyUID) // Login twice to catch weird errors that only appear after logout
}

func TestChangeDatabasePassword(t *testing.T) {
	utils.Init()

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
	err = testContext.backend.ChangeDatabasePassword(testContext.profileKeypair.KeyUID, testPassword, newPassword)
	require.NoError(t, err)

	testContext.backend.UpdateRootDataDir(testContext.config.DataDir)

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
	utils.Init()

	testContext := setupTestContext(t, testPassword, false, false, true)

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.DataDir,
		LogFilePath:        testContext.config.DataDir + "/log",
	}

	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetMobileSignalHandler)

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

	err = accountsAPI.AddAccount(context.Background(), testPassword, &accounts.Account{
		KeyUID:    account.KeyUID,
		Type:      accounts.AccountTypeGenerated,
		PublicKey: derivedAddress[0].PublicKey,
		Emoji:     "some",
		ColorID:   "so",
		Name:      "some name",
		Path:      derivedAddress[0].Path,
	})
	require.NoError(t, err)
}

func TestSetFleet(t *testing.T) {
	utils.Init()

	testContext := setupTestContext(t, testPassword, false, false, true)

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.DataDir,
		LogFilePath:        testContext.config.DataDir + "/log",
	}

	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetMobileSignalHandler)

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

	testContext.backend.UpdateRootDataDir(testContext.config.DataDir)

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
	require.Equal(t, testContext.backend.config.ClusterConfig.WakuNodes, params.DefaultWakuNodes(params.FleetStatusProd))

	require.NoError(t, testContext.backend.Logout())
}

func fakeToken() security.SensitiveString {
	return security.NewSensitiveString(gofakeit.LetterN(10))
}

func TestWalletConfigOnLoginAccount(t *testing.T) {
	utils.Init()

	testContext := setupTestContext(t, testPassword, false, false, true)

	poktToken := fakeToken()
	infuraToken := fakeToken()
	alchemyEthereumMainnetToken := fakeToken()
	alchemyEthereumSepoliaToken := fakeToken()
	alchemyArbitrumMainnetToken := fakeToken()
	alchemyArbitrumSepoliaToken := fakeToken()
	alchemyOptimismMainnetToken := fakeToken()
	alchemyOptimismSepoliaToken := fakeToken()
	alchemyBaseMainnetToken := fakeToken()
	alchemyBaseSepoliaToken := fakeToken()
	raribleMainnetAPIKey := fakeToken()
	raribleTestnetAPIKey := fakeToken()

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        testContext.config.DataDir,
		LogFilePath:        testContext.config.DataDir + "/log",
	}
	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})
	t.Cleanup(signal.ResetMobileSignalHandler)

	newAccount, err := testContext.backend.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	statusNode := testContext.backend.statusNode
	require.NotNil(t, statusNode)

	require.NoError(t, testContext.backend.Logout())

	loginAccountRequest := &requests.Login{
		KeyUID:   newAccount.KeyUID,
		Password: testPassword,
		WalletSecretsConfig: requests.WalletSecretsConfig{
			PoktToken:                   poktToken,
			InfuraToken:                 infuraToken,
			AlchemyEthereumMainnetToken: alchemyEthereumMainnetToken,
			AlchemyEthereumSepoliaToken: alchemyEthereumSepoliaToken,
			AlchemyArbitrumMainnetToken: alchemyArbitrumMainnetToken,
			AlchemyArbitrumSepoliaToken: alchemyArbitrumSepoliaToken,
			AlchemyOptimismMainnetToken: alchemyOptimismMainnetToken,
			AlchemyOptimismSepoliaToken: alchemyOptimismSepoliaToken,
			AlchemyBaseMainnetToken:     alchemyBaseMainnetToken,
			AlchemyBaseSepoliaToken:     alchemyBaseSepoliaToken,
			RaribleMainnetAPIKey:        raribleMainnetAPIKey,
			RaribleTestnetAPIKey:        raribleTestnetAPIKey,
		},
	}

	testContext.backend.UpdateRootDataDir(testContext.config.DataDir)

	require.NoError(t, testContext.backend.LoginAccount(loginAccountRequest))
	select {
	case <-c:
		break
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	walletConfig := testContext.backend.config.WalletConfig
	require.Equal(t, walletConfig.InfuraAPIKey, infuraToken)
	require.Equal(t, walletConfig.AlchemyAPIKeys[common.EthereumMainnet], alchemyEthereumMainnetToken)
	require.Equal(t, walletConfig.AlchemyAPIKeys[common.EthereumSepolia], alchemyEthereumSepoliaToken)
	require.Equal(t, walletConfig.AlchemyAPIKeys[common.ArbitrumMainnet], alchemyArbitrumMainnetToken)
	require.Equal(t, walletConfig.AlchemyAPIKeys[common.ArbitrumSepolia], alchemyArbitrumSepoliaToken)
	require.Equal(t, walletConfig.AlchemyAPIKeys[common.OptimismMainnet], alchemyOptimismMainnetToken)
	require.Equal(t, walletConfig.AlchemyAPIKeys[common.OptimismSepolia], alchemyOptimismSepoliaToken)
	require.Equal(t, walletConfig.AlchemyAPIKeys[common.BaseMainnet], alchemyBaseMainnetToken)
	require.Equal(t, walletConfig.AlchemyAPIKeys[common.BaseSepolia], alchemyBaseSepoliaToken)
	require.Equal(t, walletConfig.RaribleMainnetAPIKey, raribleMainnetAPIKey)
	require.Equal(t, walletConfig.RaribleTestnetAPIKey, raribleTestnetAPIKey)

	require.NoError(t, testContext.backend.Logout())
}

func TestTestnetEnabledSettingOnCreateAccount(t *testing.T) {
	utils.Init()
	tmpdir := t.TempDir()

	b := NewGethStatusBackend(tt.MustCreateTestLogger())

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
	utils.Init()
	tmpdir := t.TempDir()

	backend := NewGethStatusBackend(tt.MustCreateTestLogger())

	// Test case 1: Valid restore account request
	restoreRequest := &requests.RestoreAccount{
		Mnemonic:    "test test test test test test test test test test test test",
		FetchBackup: false,
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
	utils.Init()
	tmpdir := t.TempDir()

	backend := NewGethStatusBackend(tt.MustCreateTestLogger())

	// Test case: Valid restore account request without DisplayName
	restoreRequest := &requests.RestoreAccount{
		Mnemonic:    "test test test test test test test test test test test test",
		FetchBackup: false,
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
	b := NewGethStatusBackend(tt.MustCreateTestLogger())
	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	b.UpdateRootDataDir(conf.DataDir)
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
	utils.Init()
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
		"mnemonic":    "",
		"fetchBackup": true,
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
				"openseaApiKey":               "",
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
			"keycardPairingDataFile": path.Join(tmpdir, DefaultKeycardPairingDataFile),
		},
	}

	require.NotNil(t, exampleKeycardEvent)
	require.NotNil(t, exampleRequest)

	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	backend := NewGethStatusBackend(tt.MustCreateTestLogger())
	require.NoError(t, err)

	backend.UpdateRootDataDir(conf.DataDir)

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
}
