package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions"
)

var (
	networks     = json.RawMessage("{}")
	testSettings = settings.Settings{
		Address:           types.HexToAddress("0xeC540f3745Ff2964AFC1171a5A0DD726d1F6B472"),
		DisplayName:       "UserDisplayName",
		CurrentNetwork:    "mainnet_rpc",
		DappsAddress:      types.HexToAddress("0xe1300f99fDF7346986CbC766903245087394ecd0"),
		EIP1581Address:    types.HexToAddress("0xe1DDDE9235a541d1344550d969715CF43982de9f"),
		InstallationID:    "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:            "0x4e8129f3edfc004875be17bf468a784098a9f69b53c095be1f52deff286935ab",
		LatestDerivedPath: 0,
		Name:              "Jittery Cornflowerblue Kingbird",
		Networks:          &networks,
		PhotoPath:         "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:    false,
		PublicKey:         "0x04211fe0f69772ecf7eb0b5bfc7678672508a9fb01f2d699096f0d59ef7fe1a0cb1e648a80190db1c0f5f088872444d846f2956d0bd84069f3f9f69335af852ac0",
		SigningPhrase:     "yurt joey vibe",
		WalletRootAddress: types.HexToAddress("0xeB591fd819F86D0A6a2EF2Bcb94f77807a7De1a6")}
)

func setupTestDB() (*sql.DB, func() error, error) {
	tmpfile, err := ioutil.TempFile("", "tests")
	if err != nil {
		return nil, nil, err
	}
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "tests", sqlite.ReducedKDFIterationsNumber)
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

func setupTestMultiDB() (*multiaccounts.Database, func() error, error) {
	tmpfile, err := ioutil.TempFile("", "tests")
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

func setupGethStatusBackend() (*GethStatusBackend, func() error, func() error, error) {
	db, stop1, err := setupTestDB()
	if err != nil {
		return nil, nil, nil, err
	}
	backend := NewGethStatusBackend()
	backend.StatusNode().SetAppDB(db)

	ma, stop2, err := setupTestMultiDB()
	backend.StatusNode().SetMultiaccountsDB(ma)

	return backend, stop1, stop2, err
}

func TestBackendStartNodeConcurrently(t *testing.T) {
	utils.Init()

	backend, stop1, stop2, err := setupGethStatusBackend()
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
	require.NoError(t, err)

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
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

	backend, stop1, stop2, err := setupGethStatusBackend()
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
	require.NoError(t, err)

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	count := 3
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
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

	backend, stop1, stop2, err := setupGethStatusBackend()
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
	require.NoError(t, err)

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
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
		assert.NotNil(t, backend.AccountManager())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.personalAPI)
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
	backend := NewGethStatusBackend()
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
	b := NewGethStatusBackend()
	b.ConnectionChange(connection.None, false)
	assert.True(t, b.connectionState.Offline)

	b.ConnectionChange(connection.Wifi, false)
	assert.False(t, b.connectionState.Offline)

	b.ConnectionChange("unknown-state", false)
	assert.False(t, b.connectionState.Offline)
}

func TestBackendCallRPCConcurrently(t *testing.T) {
	utils.Init()

	backend, stop1, stop2, err := setupGethStatusBackend()
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
	require.NoError(t, err)

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
	count := 3

	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wg sync.WaitGroup

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
	backend := NewGethStatusBackend()

	var testCases = []struct {
		name          string
		fromState     appState
		toState       appState
		expectedState appState
	}{
		{
			name:          "success",
			fromState:     appStateInactive,
			toState:       appStateBackground,
			expectedState: appStateBackground,
		},
		{
			name:          "invalid state",
			fromState:     appStateInactive,
			toState:       "unexisting",
			expectedState: appStateInactive,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			backend.appState = tc.fromState
			backend.AppStateChange(tc.toState.String())
			assert.Equal(t, tc.expectedState.String(), backend.appState.String())
		})
	}
}

func TestBlockedRPCMethods(t *testing.T) {
	utils.Init()

	backend, stop1, stop2, err := setupGethStatusBackend()
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
	require.NoError(t, err)

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
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
	backend := NewGethStatusBackend()

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

	backend, stop1, stop2, err := setupGethStatusBackend()
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
	require.NoError(t, err)

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
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

	backend, stop1, stop2, err := setupGethStatusBackend()
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
	require.NoError(t, err)

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
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

	password := "test"
	backend, defers, err := setupWalletTest(t, password)
	require.NoError(t, err)
	defer defers()

	t.Run("AccountDoesntExist", func(t *testing.T) {
		pkey, err := gethcrypto.GenerateKey()
		require.NoError(t, err)
		address := gethcrypto.PubkeyToAddress(pkey.PublicKey)
		key, err := backend.getVerifiedWalletAccount(address.String(), password)
		require.EqualError(t, err, transactions.ErrAccountDoesntExist.Error())
		require.Nil(t, key)
	})

	t.Run("PasswordDoesntMatch", func(t *testing.T) {
		pkey, err := crypto.GenerateKey()
		require.NoError(t, err)
		address := crypto.PubkeyToAddress(pkey.PublicKey)
		db, err := accounts.NewDB(backend.appDB)

		require.NoError(t, err)
		_, err = backend.AccountManager().ImportAccount(pkey, password)
		require.NoError(t, err)
		require.NoError(t, db.SaveAccounts([]*accounts.Account{{Address: address}}))
		key, err := backend.getVerifiedWalletAccount(address.String(), "wrong-password")
		require.EqualError(t, err, "could not decrypt key with given password")
		require.Nil(t, key)
	})

	t.Run("PartialAccount", func(t *testing.T) {
		// Create a derived wallet account without storing the keys
		db, err := accounts.NewDB(backend.appDB)
		require.NoError(t, err)
		newPath := "m/0"
		walletRootAddress, err := db.GetWalletRootAddress()
		require.NoError(t, err)

		walletInfo, err := backend.AccountManager().AccountsGenerator().LoadAccount(walletRootAddress.String(), password)
		require.NoError(t, err)
		derivedInfos, err := backend.AccountManager().AccountsGenerator().DeriveAddresses(walletInfo.ID, []string{newPath})
		require.NoError(t, err)
		derivedInfo := derivedInfos[newPath]

		partialAcc := &accounts.Account{
			Address:   types.HexToAddress(derivedInfo.Address),
			Type:      accounts.AccountTypeGenerated,
			PublicKey: types.Hex2Bytes(derivedInfo.PublicKey),
			Path:      newPath,
			Wallet:    false,
			Name:      "PartialAccount",
		}
		require.NoError(t, db.SaveAccounts([]*accounts.Account{partialAcc}))

		// With partial account we need to dynamically generate private key
		key, err := backend.getVerifiedWalletAccount(partialAcc.Address.Hex(), password)
		require.NoError(t, err)
		require.Equal(t, partialAcc.Address, key.Address)
	})

	t.Run("Success", func(t *testing.T) {
		pkey, err := crypto.GenerateKey()
		require.NoError(t, err)
		address := crypto.PubkeyToAddress(pkey.PublicKey)
		db, err := accounts.NewDB(backend.appDB)
		require.NoError(t, err)
		defer db.Close()
		_, err = backend.AccountManager().ImportAccount(pkey, password)
		require.NoError(t, err)
		require.NoError(t, db.SaveAccounts([]*accounts.Account{{Address: address}}))
		key, err := backend.getVerifiedWalletAccount(address.String(), password)
		require.NoError(t, err)
		require.Equal(t, address, key.Address)
	})
}

func TestLoginWithKey(t *testing.T) {
	utils.Init()

	b := NewGethStatusBackend()
	chatKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	walletKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	keyUIDHex := sha256.Sum256(gethcrypto.FromECDSAPub(&chatKey.PublicKey))
	keyUID := types.EncodeHex(keyUIDHex[:])
	main := multiaccounts.Account{
		KeyUID: keyUID,
	}
	tmpdir, err := os.MkdirTemp("", "login-with-key-test-")
	require.NoError(t, err)
	defer os.Remove(tmpdir)
	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)
	keyhex := hex.EncodeToString(gethcrypto.FromECDSA(chatKey))

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)
	require.NoError(t, b.OpenAccounts())

	address := crypto.PubkeyToAddress(walletKey.PublicKey)
	require.NoError(t, b.SaveAccountAndStartNodeWithKey(main, "test-pass", testSettings, conf, []*accounts.Account{{Address: address, Wallet: true}}, keyhex))
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)
	require.NoError(t, b.OpenAccounts())

	require.NoError(t, b.StartNodeWithKey(main, "test-pass", keyhex))
	defer func() {
		assert.NoError(t, b.Logout())
		assert.NoError(t, b.StopNode())
	}()
	extkey, err := b.accountManager.SelectedChatAccount()
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(chatKey.PublicKey), extkey.Address)
}

func TestVerifyDatabasePassword(t *testing.T) {
	utils.Init()

	b := NewGethStatusBackend()
	chatKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	walletKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	keyUIDHex := sha256.Sum256(gethcrypto.FromECDSAPub(&chatKey.PublicKey))
	keyUID := types.EncodeHex(keyUIDHex[:])
	main := multiaccounts.Account{
		KeyUID: keyUID,
	}
	tmpdir, err := os.MkdirTemp("", "verify-database-password-")
	require.NoError(t, err)
	defer os.Remove(tmpdir)
	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)
	keyhex := hex.EncodeToString(gethcrypto.FromECDSA(chatKey))

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)
	require.NoError(t, b.OpenAccounts())

	address := crypto.PubkeyToAddress(walletKey.PublicKey)
	require.NoError(t, b.SaveAccountAndStartNodeWithKey(main, "test-pass", testSettings, conf, []*accounts.Account{{Address: address, Wallet: true}}, keyhex))
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())

	require.Error(t, b.VerifyDatabasePassword(main.KeyUID, "wrong-pass"))
	require.NoError(t, b.VerifyDatabasePassword(main.KeyUID, "test-pass"))
}

func TestDeleteMultiaccount(t *testing.T) {
	backend := NewGethStatusBackend()

	rootDataDir, err := os.MkdirTemp("", "test-keystore-dir")
	require.NoError(t, err)
	defer os.Remove(rootDataDir)

	keyStoreDir := filepath.Join(rootDataDir, "keystore")

	backend.rootDataDir = rootDataDir

	err = backend.AccountManager().InitKeystore(keyStoreDir)
	require.NoError(t, err)

	backend.AccountManager()
	accs, err := backend.AccountManager().
		AccountsGenerator().
		GenerateAndDeriveAddresses(12, 1, "", []string{"m/44'/60'/0'/0"})
	require.NoError(t, err)

	generateAccount := accs[0]
	accountInfo, err := backend.AccountManager().
		AccountsGenerator().
		StoreAccount(generateAccount.ID, "123123")
	require.NoError(t, err)

	account := multiaccounts.Account{
		Name:           "foo",
		Timestamp:      1,
		KeycardPairing: "pairing",
		KeyUID:         generateAccount.KeyUID,
	}

	err = backend.ensureAppDBOpened(account, "123123")
	require.NoError(t, err)

	s := settings.Settings{
		Address:           types.HexToAddress(accountInfo.Address),
		CurrentNetwork:    "mainnet_rpc",
		DappsAddress:      types.HexToAddress(accountInfo.Address),
		EIP1581Address:    types.HexToAddress(accountInfo.Address),
		InstallationID:    "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:            account.KeyUID,
		LatestDerivedPath: 0,
		Name:              "Jittery Cornflowerblue Kingbird",
		Networks:          &networks,
		PhotoPath:         "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:    false,
		PublicKey:         accountInfo.PublicKey,
		SigningPhrase:     "yurt joey vibe",
		WalletRootAddress: types.HexToAddress(accountInfo.Address)}

	err = backend.saveAccountsAndSettings(
		s,
		&params.NodeConfig{},
		nil)
	require.NoError(t, err)

	err = backend.OpenAccounts()
	require.NoError(t, err)

	err = backend.SaveAccount(account)
	require.NoError(t, err)

	files, err := ioutil.ReadDir(rootDataDir)
	require.NoError(t, err)
	require.NotEqual(t, 3, len(files))

	err = backend.DeleteMultiaccount(account.KeyUID, keyStoreDir)
	require.NoError(t, err)

	files, err = ioutil.ReadDir(rootDataDir)
	require.NoError(t, err)
	require.Equal(t, 3, len(files))
}

func TestConvertAccount(t *testing.T) {
	const mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	const password = "111111"        // represents password for a regular user
	const keycardPassword = "222222" // represents password for a keycard user
	const pathMaster = "m"
	const pathWalletRoot = "m/44'/60'/0'/0"
	const pathEIP1581Root = "m/43'/60'/1581'"
	const pathEIP1581Chat = "m/43'/60'/1581'/0/0"
	const pathDefaultWalletAccount = "m/44'/60'/0'/0/0"
	const customWalletPath1 = "m/44'/60'/0'/0/1"
	const customWalletPath2 = "m/44'/60'/0'/0/2"
	var allGeneratedPaths []string
	allGeneratedPaths = append(allGeneratedPaths, customWalletPath1, customWalletPath2, pathWalletRoot, pathEIP1581Root, pathDefaultWalletAccount)

	var err error

	keystoreContainsFileForAccount := func(keyStoreDir string, hexAddress string) bool {
		addrWithoutPrefix := strings.ToLower(hexAddress[2:])
		found := false
		err = filepath.Walk(keyStoreDir, func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fileInfo.IsDir() && strings.Contains(strings.ToUpper(path), strings.ToUpper(addrWithoutPrefix)) {
				found = true
			}
			return nil
		})
		return found
	}

	rootDataDir, err := os.MkdirTemp("", "test-keystore-dir")
	require.NoError(t, err)
	defer os.Remove(rootDataDir)

	keyStoreDir := filepath.Join(rootDataDir, "keystore")

	utils.Init()

	backend, stop1, stop2, err := setupGethStatusBackend()
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
	require.NoError(t, err)

	backend.rootDataDir = rootDataDir
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	config.DataDir = rootDataDir
	config.KeyStoreDir = keyStoreDir
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
	require.NoError(t, backend.StartNode(config))
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	genAccInfo, err := backend.AccountManager().AccountsGenerator().ImportMnemonic(mnemonic, "")
	assert.NoError(t, err)

	accountInfo, err := backend.AccountManager().AccountsGenerator().StoreAccount(genAccInfo.ID, password)
	assert.NoError(t, err)

	found := keystoreContainsFileForAccount(keyStoreDir, accountInfo.Address)
	require.NoError(t, err)
	require.True(t, found)

	derivedAccounts, err := backend.AccountManager().AccountsGenerator().StoreDerivedAccounts(genAccInfo.ID, password, allGeneratedPaths)
	assert.NoError(t, err)

	accs, err := backend.AccountManager().AccountsGenerator().DeriveAddresses(genAccInfo.ID, []string{pathMaster})
	require.NoError(t, err)
	require.Equal(t, 1, len(accs))
	masterAddress := accs[pathMaster].Address
	found = keystoreContainsFileForAccount(keyStoreDir, masterAddress)
	require.NoError(t, err)
	require.True(t, found)

	var accountsToStore []*accounts.Account
	accountsToStore = append(accountsToStore, &accounts.Account{
		Address:   types.HexToAddress(masterAddress),
		KeyUID:    genAccInfo.KeyUID,
		Type:      accounts.AccountTypeGenerated,
		PublicKey: types.Hex2Bytes(accountInfo.PublicKey),
		Path:      pathEIP1581Chat,
		Wallet:    false,
		Chat:      true,
		Name:      "GeneratedAccount",
	})

	for p, dAccInfo := range derivedAccounts {
		found = keystoreContainsFileForAccount(keyStoreDir, dAccInfo.Address)
		require.NoError(t, err)
		require.True(t, found)

		if p == pathDefaultWalletAccount ||
			p == customWalletPath1 ||
			p == customWalletPath2 {
			accountsToStore = append(accountsToStore, &accounts.Account{
				Address:     types.HexToAddress(dAccInfo.Address),
				KeyUID:      genAccInfo.KeyUID,
				Wallet:      false,
				Chat:        false,
				Type:        accounts.AccountTypeGenerated,
				Path:        p,
				Name:        "derivacc" + p,
				Hidden:      false,
				DerivedFrom: masterAddress,
				Removed:     false,
			})
		}
	}

	account := multiaccounts.Account{
		Name:      "foo",
		Timestamp: 1,
		KeyUID:    genAccInfo.KeyUID,
	}

	err = backend.ensureAppDBOpened(account, password)
	require.NoError(t, err)

	s := settings.Settings{
		Address:           types.HexToAddress(masterAddress),
		DisplayName:       "UserDisplayName",
		CurrentNetwork:    "mainnet_rpc",
		DappsAddress:      types.HexToAddress(derivedAccounts[pathDefaultWalletAccount].Address),
		EIP1581Address:    types.HexToAddress(derivedAccounts[pathEIP1581Root].Address),
		InstallationID:    "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:            account.KeyUID,
		LatestDerivedPath: 0,
		Name:              "Jittery Cornflowerblue Kingbird",
		Networks:          &networks,
		PhotoPath:         "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:    false,
		PublicKey:         accountInfo.PublicKey,
		SigningPhrase:     "yurt joey vibe",
		WalletRootAddress: types.HexToAddress(derivedAccounts[pathWalletRoot].Address),
	}

	err = backend.saveAccountsAndSettings(
		s,
		&params.NodeConfig{},
		accountsToStore)
	require.NoError(t, err)

	err = backend.OpenAccounts()
	require.NoError(t, err)

	err = backend.SaveAccount(account)
	require.NoError(t, err)

	files, err := ioutil.ReadDir(rootDataDir)
	require.NoError(t, err)
	require.NotEqual(t, 3, len(files))

	keycardAccount := account
	keycardAccount.KeycardPairing = "pairing"

	keycardSettings := settings.Settings{
		KeycardInstanceUID: "0xdeadbeef",
		KeycardPairedOn:    1,
		KeycardPairing:     "pairing",
	}

	// Converting to a keycard account
	err = backend.ConvertToKeycardAccount(keycardAccount, keycardSettings, password, keycardPassword)
	require.NoError(t, err)

	// Validating results of converting to a keycard account.
	// All keystore files for the account which is migrated need to be removed.
	found = keystoreContainsFileForAccount(keyStoreDir, accountInfo.Address)
	require.NoError(t, err)
	require.False(t, found)

	for _, dAccInfo := range derivedAccounts {
		found = keystoreContainsFileForAccount(keyStoreDir, dAccInfo.Address)
		require.NoError(t, err)
		require.False(t, found)
	}

	found = keystoreContainsFileForAccount(keyStoreDir, masterAddress)
	require.NoError(t, err)
	require.False(t, found)

	// Ensure we're able to open the DB
	err = backend.ensureAppDBOpened(keycardAccount, keycardPassword)
	require.NoError(t, err)

	b1 := NewGethStatusBackend()
	require.NoError(t, b1.OpenAccounts())

	// Converting to a regular account
	err = backend.ConvertToRegularAccount(mnemonic, keycardPassword, password)
	require.NoError(t, err)

	// Validating results of converting to a regular account.
	// All keystore files for need to be created.
	found = keystoreContainsFileForAccount(keyStoreDir, accountInfo.Address)
	require.NoError(t, err)
	require.True(t, found)

	for _, dAccInfo := range derivedAccounts {
		found = keystoreContainsFileForAccount(keyStoreDir, dAccInfo.Address)
		require.NoError(t, err)
		require.True(t, found)
	}

	found = keystoreContainsFileForAccount(keyStoreDir, masterAddress)
	require.NoError(t, err)
	require.True(t, found)

	// Ensure we're able to open the DB
	err = backend.ensureAppDBOpened(keycardAccount, password)
	require.NoError(t, err)

	b2 := NewGethStatusBackend()
	require.NoError(t, b2.OpenAccounts())
}

func copyFile(srcFolder string, dstFolder string, fileName string, t *testing.T) {
	data, err := ioutil.ReadFile(path.Join(srcFolder, fileName))
	if err != nil {
		t.Fail()
	}

	err = ioutil.WriteFile(path.Join(dstFolder, fileName), data, 0600)
	if err != nil {
		t.Fail()
	}
}

func copyDir(srcFolder string, dstFolder string, t *testing.T) {
	files, err := ioutil.ReadDir(srcFolder)
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

func login(t *testing.T, conf *params.NodeConfig) {
	// The following passwords and DB used in this test unit are only
	// used to determine if login process works correctly after a migration

	// Expected account data:
	keyUID := "0x7c46c8f6f059ab72d524f2a6d356904db30bb0392636172ab3929a6bd2220f84" // #nosec G101
	username := "TestUser"
	passwd := "0xC888C9CE9E098D5864D3DED6EBCC140A12142263BACE3A23A36F9905F12BD64A" // #nosec G101

	b := NewGethStatusBackend()

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)

	require.NoError(t, b.OpenAccounts())

	accounts, err := b.GetAccounts()
	require.NoError(t, err)

	require.Len(t, accounts, 1)
	require.Equal(t, username, accounts[0].Name)
	require.Equal(t, keyUID, accounts[0].KeyUID)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := b.StartNodeWithAccount(accounts[0], passwd, conf)
		require.NoError(t, err)
	}()

	wg.Wait()
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())

}

func TestLoginAndMigrationsStillWorkWithExistingUsers(t *testing.T) {
	utils.Init()

	srcFolder := "../static/test-0.97.3-account/"

	tmpdir, err := os.MkdirTemp("", "login-and-migrations-with-existing-users")
	require.NoError(t, err)
	defer os.Remove(tmpdir)

	copyDir(srcFolder, tmpdir, t)

	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	login(t, conf)
	login(t, conf) // Login twice to catch weird errors that only appear after logout
}
