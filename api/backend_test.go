package api

import (
	"context"
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
	"time"

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
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/typeddata"
	walletservice "github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/walletdatabase"
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
	return helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "tests")
}

func setupTestWalletDB() (*sql.DB, func() error, error) {
	return helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "tests")
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

func setupGethStatusBackend() (*GethStatusBackend, func() error, func() error, func() error, error) {
	db, stop1, err := setupTestDB()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	backend := NewGethStatusBackend()
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
		keyUIDHex := sha256.Sum256(gethcrypto.FromECDSAPub(&pkey.PublicKey))
		keyUID := types.EncodeHex(keyUIDHex[:])

		db, err := accounts.NewDB(backend.appDB)

		require.NoError(t, err)
		_, err = backend.AccountManager().ImportAccount(pkey, password)
		require.NoError(t, err)
		require.NoError(t, db.SaveOrUpdateKeypair(&accounts.Keypair{
			KeyUID: keyUID,
			Name:   "private key keypair",
			Type:   accounts.KeypairTypeKey,
			Accounts: []*accounts.Account{
				&accounts.Account{
					Address: address,
					KeyUID:  keyUID,
				},
			},
		}))
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

		keypair := &accounts.Keypair{
			KeyUID: walletInfo.KeyUID,
			Name:   "profile keypair",
			Type:   accounts.KeypairTypeProfile,
			Accounts: []*accounts.Account{
				&accounts.Account{
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
		key, err := backend.getVerifiedWalletAccount(keypair.Accounts[0].Address.Hex(), password)
		require.NoError(t, err)
		require.Equal(t, keypair.Accounts[0].Address, key.Address)
	})

	t.Run("Success", func(t *testing.T) {
		pkey, err := crypto.GenerateKey()
		require.NoError(t, err)
		address := crypto.PubkeyToAddress(pkey.PublicKey)
		keyUIDHex := sha256.Sum256(gethcrypto.FromECDSAPub(&pkey.PublicKey))
		keyUID := types.EncodeHex(keyUIDHex[:])

		db, err := accounts.NewDB(backend.appDB)
		require.NoError(t, err)
		defer db.Close()
		_, err = backend.AccountManager().ImportAccount(pkey, password)
		require.NoError(t, err)
		require.NoError(t, db.SaveOrUpdateKeypair(&accounts.Keypair{
			KeyUID: keyUID,
			Name:   "private key keypair",
			Type:   accounts.KeypairTypeKey,
			Accounts: []*accounts.Account{
				&accounts.Account{
					Address: address,
					KeyUID:  keyUID,
				},
			},
		}))
		key, err := backend.getVerifiedWalletAccount(address.String(), password)
		require.NoError(t, err)
		require.Equal(t, address, key.Address)
	})
}

func TestRuntimeLogLevelIsNotWrittenToDatabase(t *testing.T) {
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

	tmpdir := t.TempDir()

	json := `{
		"NetworkId": 3,
		"DataDir": "` + tmpdir + `",
		"KeyStoreDir": "` + tmpdir + `",
		"KeycardPairingDataFile": "` + path.Join(tmpdir, "keycard/pairings.json") + `",
		"NoDiscovery": true,
		"TorrentConfig": {
			"Port": 9025,
			"Enabled": false,
			"DataDir": "` + tmpdir + `/archivedata",
			"TorrentDir": "` + tmpdir + `/torrents"
		},
		"RuntimeLogLevel": "INFO",
		"LogLevel": "DEBUG"
	}`

	conf, err := params.NewConfigFromJSON(json)
	require.NoError(t, err)
	require.Equal(t, "INFO", conf.RuntimeLogLevel)
	keyhex := hex.EncodeToString(gethcrypto.FromECDSA(chatKey))

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)
	require.NoError(t, b.OpenAccounts())
	require.NotNil(t, b.statusNode.HTTPServer())

	address := crypto.PubkeyToAddress(walletKey.PublicKey)

	settings := testSettings
	settings.KeyUID = keyUID
	settings.Address = crypto.PubkeyToAddress(walletKey.PublicKey)

	chatPubKey := crypto.FromECDSAPub(&chatKey.PublicKey)
	require.NoError(t, b.SaveAccountAndStartNodeWithKey(main, "test-pass", settings, conf,
		[]*accounts.Account{
			{Address: address, KeyUID: keyUID, Wallet: true},
			{Address: crypto.PubkeyToAddress(chatKey.PublicKey), KeyUID: keyUID, Chat: true, PublicKey: chatPubKey}}, keyhex))
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())

	require.NoError(t, b.StartNodeWithKey(main, "test-pass", keyhex, conf))
	defer func() {
		assert.NoError(t, b.Logout())
		assert.NoError(t, b.StopNode())
	}()

	c, err := b.GetNodeConfig()
	require.NoError(t, err)
	require.Equal(t, "", c.RuntimeLogLevel)
	require.Equal(t, "DEBUG", c.LogLevel)
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
	tmpdir := t.TempDir()
	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)
	keyhex := hex.EncodeToString(gethcrypto.FromECDSA(chatKey))

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)
	require.NoError(t, b.OpenAccounts())
	require.NotNil(t, b.statusNode.HTTPServer())

	address := crypto.PubkeyToAddress(walletKey.PublicKey)

	settings := testSettings
	settings.KeyUID = keyUID
	settings.Address = crypto.PubkeyToAddress(walletKey.PublicKey)

	chatPubKey := crypto.FromECDSAPub(&chatKey.PublicKey)
	require.NoError(t, b.SaveAccountAndStartNodeWithKey(main, "test-pass", settings, conf,
		[]*accounts.Account{
			{Address: address, KeyUID: keyUID, Wallet: true},
			{Address: crypto.PubkeyToAddress(chatKey.PublicKey), KeyUID: keyUID, Chat: true, PublicKey: chatPubKey}}, keyhex))
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)
	require.NoError(t, b.OpenAccounts())

	require.NoError(t, b.StartNodeWithKey(main, "test-pass", keyhex, conf))
	defer func() {
		assert.NoError(t, b.Logout())
		assert.NoError(t, b.StopNode())
	}()
	extkey, err := b.accountManager.SelectedChatAccount()
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(chatKey.PublicKey), extkey.Address)

	activeAccount, err := b.GetActiveAccount()
	require.NoError(t, err)
	require.NotNil(t, activeAccount.ColorHash)
}

func TestLoginAccount(t *testing.T) {
	utils.Init()
	password := "some-password"
	tmpdir := t.TempDir()
	nameserver := "8.8.8.8"

	b := NewGethStatusBackend()
	createAccountRequest := &requests.CreateAccount{
		DisplayName:           "some-display-name",
		CustomizationColor:    "#ffffff",
		Emoji:                 "some",
		Password:              password,
		BackupDisabledDataDir: tmpdir,
		LogFilePath:           tmpdir + "/log",
		WakuV2Nameserver:      &nameserver,
	}
	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), signal.EventLoggedIn) {
			c <- struct{}{}
		}
	})
	waitForLogin := func(chan interface{}) {
		select {
		case <-c:
			break
		case <-time.After(5 * time.Second):
			t.FailNow()
		}
	}

	_, err := b.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	require.Equal(t, nameserver, b.config.WakuV2Config.Nameserver)

	waitForLogin(c)
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())

	accounts, err := b.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)

	loginAccountRequest := &requests.Login{
		KeyUID:           accounts[0].KeyUID,
		Password:         password,
		WakuV2Nameserver: nameserver,
	}
	err = b.LoginAccount(loginAccountRequest)
	require.NoError(t, err)
	waitForLogin(c)

	require.Equal(t, nameserver, b.config.WakuV2Config.Nameserver)
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
	tmpdir := t.TempDir()
	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)
	keyhex := hex.EncodeToString(gethcrypto.FromECDSA(chatKey))

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)
	require.NoError(t, b.OpenAccounts())

	address := crypto.PubkeyToAddress(walletKey.PublicKey)

	settings := testSettings
	settings.KeyUID = keyUID
	settings.Address = crypto.PubkeyToAddress(walletKey.PublicKey)

	chatPubKey := crypto.FromECDSAPub(&chatKey.PublicKey)

	require.NoError(t, b.SaveAccountAndStartNodeWithKey(main, "test-pass", settings, conf, []*accounts.Account{
		{Address: address, KeyUID: keyUID, Wallet: true},
		{Address: crypto.PubkeyToAddress(chatKey.PublicKey), KeyUID: keyUID, Chat: true, PublicKey: chatPubKey}}, keyhex))
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())

	require.Error(t, b.VerifyDatabasePassword(main.KeyUID, "wrong-pass"))
	require.NoError(t, b.VerifyDatabasePassword(main.KeyUID, "test-pass"))
}

func TestDeleteMultiaccount(t *testing.T) {
	backend := NewGethStatusBackend()

	rootDataDir := t.TempDir()

	keyStoreDir := filepath.Join(rootDataDir, "keystore")

	backend.rootDataDir = rootDataDir

	err := backend.AccountManager().InitKeystore(keyStoreDir)
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
	require.Error(t, err)
	require.True(t, err == accounts.ErrKeypairWithoutAccounts)

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
	const keycardUID = "1234"
	const pathEIP1581Root = "m/43'/60'/1581'"
	const pathEIP1581Chat = pathEIP1581Root + "/0'/0"
	const pathWalletRoot = "m/44'/60'/0'/0"
	const pathDefaultWalletAccount = pathWalletRoot + "/0"
	const customWalletPath1 = pathWalletRoot + "/1"
	const customWalletPath2 = pathWalletRoot + "/2"
	var allGeneratedPaths []string
	allGeneratedPaths = append(allGeneratedPaths, pathEIP1581Root, pathEIP1581Chat, pathWalletRoot, pathDefaultWalletAccount, customWalletPath1, customWalletPath2)

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

	rootDataDir := t.TempDir()

	keyStoreDir := filepath.Join(rootDataDir, "keystore")

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

	backend.rootDataDir = rootDataDir
	require.NoError(t, backend.AccountManager().InitKeystore(keyStoreDir))
	err = backend.OpenAccounts()
	require.NoError(t, err)

	genAccInfo, err := backend.AccountManager().AccountsGenerator().ImportMnemonic(mnemonic, "")
	assert.NoError(t, err)

	masterAddress := genAccInfo.Address

	accountInfo, err := backend.AccountManager().AccountsGenerator().StoreAccount(genAccInfo.ID, password)
	assert.NoError(t, err)

	found := keystoreContainsFileForAccount(keyStoreDir, accountInfo.Address)
	require.True(t, found)

	derivedAccounts, err := backend.AccountManager().AccountsGenerator().StoreDerivedAccounts(genAccInfo.ID, password, allGeneratedPaths)
	assert.NoError(t, err)

	chatKey := derivedAccounts[pathEIP1581Chat].PrivateKey[2:]
	chatAddress := derivedAccounts[pathEIP1581Chat].Address
	found = keystoreContainsFileForAccount(keyStoreDir, chatAddress)
	require.True(t, found)

	defaultSettings, err := defaultSettings(genAccInfo, derivedAccounts, nil)
	require.NoError(t, err)
	nodeConfig, err := defaultNodeConfig(defaultSettings.InstallationID, &requests.CreateAccount{
		LogLevel: defaultSettings.LogLevel,
	})
	require.NoError(t, err)
	nodeConfig.DataDir = rootDataDir
	nodeConfig.KeyStoreDir = keyStoreDir

	profileKeypair := &accounts.Keypair{
		KeyUID:      genAccInfo.KeyUID,
		Name:        "Profile Name",
		Type:        accounts.KeypairTypeProfile,
		DerivedFrom: masterAddress,
	}

	profileKeypair.Accounts = append(profileKeypair.Accounts, &accounts.Account{
		Address:   types.HexToAddress(chatAddress),
		KeyUID:    profileKeypair.KeyUID,
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
			wAcc := &accounts.Account{
				Address: types.HexToAddress(dAccInfo.Address),
				KeyUID:  genAccInfo.KeyUID,
				Wallet:  false,
				Chat:    false,
				Type:    accounts.AccountTypeGenerated,
				Path:    p,
				Name:    "derivacc" + p,
				Hidden:  false,
				Removed: false,
			}
			if p == pathDefaultWalletAccount {
				wAcc.Wallet = true
			}
			profileKeypair.Accounts = append(profileKeypair.Accounts, wAcc)
		}
	}

	account := multiaccounts.Account{
		Name:      profileKeypair.Name,
		Timestamp: 1,
		KeyUID:    profileKeypair.KeyUID,
	}

	err = backend.ensureAppDBOpened(account, password)
	require.NoError(t, err)

	err = backend.StartNodeWithAccountAndInitialConfig(account, password, *defaultSettings, nodeConfig, profileKeypair.Accounts)
	require.NoError(t, err)
	multiaccounts, err := backend.GetAccounts()
	require.NoError(t, err)
	require.NotEmpty(t, multiaccounts[0].ColorHash)
	serverMessenger := backend.Messenger()
	require.NotNil(t, serverMessenger)

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

	// Ensure we're able to open the DB
	err = backend.ensureAppDBOpened(keycardAccount, keycardPassword)
	require.NoError(t, err)

	// db creation
	db, err := accounts.NewDB(backend.appDB)
	require.NoError(t, err)

	// Check that there is no registered keycards
	keycards, err := db.GetKeycardsWithSameKeyUID(genAccInfo.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 0, len(keycards))

	// Converting to a keycard account
	err = backend.ConvertToKeycardAccount(keycardAccount, keycardSettings, keycardUID, password, keycardPassword)
	require.NoError(t, err)

	// Validating results of converting to a keycard account.
	// All keystore files for the account which is migrated need to be removed.
	found = keystoreContainsFileForAccount(keyStoreDir, masterAddress)
	require.False(t, found)

	for _, dAccInfo := range derivedAccounts {
		found = keystoreContainsFileForAccount(keyStoreDir, dAccInfo.Address)
		require.False(t, found)
	}

	require.NoError(t, backend.Logout())
	require.NoError(t, backend.StopNode())

	require.NoError(t, backend.AccountManager().InitKeystore(keyStoreDir))
	require.NoError(t, backend.OpenAccounts())

	require.NoError(t, backend.StartNodeWithKey(account, keycardPassword, chatKey, nodeConfig))
	defer func() {
		assert.NoError(t, backend.Logout())
		assert.NoError(t, backend.StopNode())
	}()

	// Ensure we're able to open the DB
	err = backend.ensureAppDBOpened(keycardAccount, keycardPassword)
	require.NoError(t, err)

	// db creation after re-encryption
	db1, err := accounts.NewDB(backend.appDB)
	require.NoError(t, err)

	// Check that there is a registered keycard
	keycards, err = db1.GetKeycardsWithSameKeyUID(genAccInfo.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 1, len(keycards))

	// Converting to a regular account
	err = backend.ConvertToRegularAccount(mnemonic, keycardPassword, password)
	require.NoError(t, err)

	// Validating results of converting to a regular account.
	// All keystore files for need to be created.
	found = keystoreContainsFileForAccount(keyStoreDir, accountInfo.Address)
	require.True(t, found)

	for _, dAccInfo := range derivedAccounts {
		found = keystoreContainsFileForAccount(keyStoreDir, dAccInfo.Address)
		require.True(t, found)
	}

	found = keystoreContainsFileForAccount(keyStoreDir, masterAddress)
	require.True(t, found)

	// Ensure we're able to open the DB
	err = backend.ensureAppDBOpened(keycardAccount, password)
	require.NoError(t, err)

	// db creation after re-encryption
	db2, err := accounts.NewDB(backend.appDB)
	require.NoError(t, err)

	// Check that there is no registered keycards
	keycards, err = db2.GetKeycardsWithSameKeyUID(genAccInfo.KeyUID)
	require.NoError(t, err)
	require.Equal(t, 0, len(keycards))
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

func loginDesktopUser(t *testing.T, conf *params.NodeConfig) {
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
	require.NotNil(t, b.statusNode.HTTPServer())
	require.NoError(t, b.StopNode())

}

func TestLoginAndMigrationsStillWorkWithExistingDesktopUser(t *testing.T) {
	utils.Init()

	srcFolder := "../static/test-0.132.0-account/"

	tmpdir := t.TempDir()

	copyDir(srcFolder, tmpdir, t)

	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	loginDesktopUser(t, conf)
	loginDesktopUser(t, conf) // Login twice to catch weird errors that only appear after logout
}

func loginMobileUser(t *testing.T, rootDataDir string) {
	keyUID := "0x24e820f007f6da0f56d394cdf1573d4c01ba716d5f39d0cf95e47bbf06e96d7d"
	passwd := "0x20756cad9b728c8225fd8cedb6badaf8731e174506950219ea657cd54f35f46c" // #nosec G101

	b := NewGethStatusBackend()
	b.UpdateRootDataDir(rootDataDir)
	require.NoError(t, b.OpenAccounts())

	// fixme(Frank) we need specify NoDiscovery to true to avoid error: "NoDiscovery is false, but ClusterConfig.BootNodes is empty"
	// relate mobile issue: https://github.com/status-im/status-mobile/issues/20140
	conf := &params.NodeConfig{NoDiscovery: true}
	require.NoError(t, b.StartNodeWithAccount(multiaccounts.Account{KeyUID: keyUID}, passwd, conf))
	require.NoError(t, b.Logout())
}

func TestLoginAndMigrationsStillWorkWithExistingMobileUser(t *testing.T) {
	utils.Init()

	srcFolder := "../static/test-release-1.20.x-aa6e4b2-account/"

	tmpdir := t.TempDir()

	copyDir(srcFolder, tmpdir, t)

	loginMobileUser(t, tmpdir)
	loginMobileUser(t, tmpdir) // Login twice to catch weird errors that only appear after logout
}

func TestChangeDatabasePassword(t *testing.T) {
	oldPassword := "password"
	newPassword := "newPassword"

	backend := NewGethStatusBackend()
	backend.UpdateRootDataDir(t.TempDir())

	// Setup keystore to test decryption of it
	keyStoreDir := t.TempDir()
	require.NoError(t, backend.accountManager.InitKeystore(keyStoreDir))

	_, accountInfo, _, err := backend.accountManager.CreateAccount(oldPassword)
	require.NoError(t, err)

	account := multiaccounts.Account{
		Name:          "TestAccount",
		Timestamp:     1,
		KeyUID:        "0x7c46c8f6f059ab72d524f2a6d356904db30bb0392636172ab3929a6bd2220f84",
		KDFIterations: 1,
	}

	// Initialize accounts DB
	err = backend.OpenAccounts()
	require.NoError(t, err)
	err = backend.SaveAccount(account)
	require.NoError(t, err)

	// Created DBs with old password
	err = backend.ensureDBsOpened(account, oldPassword)
	require.NoError(t, err)

	// Change password
	err = backend.ChangeDatabasePassword(account.KeyUID, oldPassword, newPassword)
	require.NoError(t, err)

	// Test that DBs can be opened with new password
	appDbPath, err := backend.getAppDBPath(account.KeyUID)
	require.NoError(t, err)
	appDb, err := sqlite.OpenDB(appDbPath, newPassword, account.KDFIterations)
	require.NoError(t, err)
	appDb.Close()

	walletDbPath, err := backend.getWalletDBPath(account.KeyUID)
	require.NoError(t, err)
	walletDb, err := sqlite.OpenDB(walletDbPath, newPassword, account.KDFIterations)
	require.NoError(t, err)
	walletDb.Close()

	// Test that keystore can be decrypted with the new password
	acc, key, err := backend.accountManager.AddressToDecryptedAccount(accountInfo.WalletAddress, newPassword)
	require.NoError(t, err)
	require.NotNil(t, acc)
	require.NotNil(t, key)
	require.Equal(t, acc.Address, key.Address)
}

func TestCreateWallet(t *testing.T) {
	utils.Init()
	password := "some-password2" // nolint: goconst
	tmpdir := t.TempDir()

	b := NewGethStatusBackend()
	createAccountRequest := &requests.CreateAccount{
		DisplayName:           "some-display-name",
		CustomizationColor:    "#ffffff",
		Emoji:                 "emoji",
		Password:              password,
		BackupDisabledDataDir: tmpdir,
		LogFilePath:           tmpdir + "/log",
	}
	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})

	account, err := b.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	statusNode := b.statusNode
	require.NotNil(t, statusNode)

	walletService := statusNode.WalletService()
	require.NotNil(t, walletService)
	walletAPI := walletservice.NewAPI(walletService)

	paths := []string{"m/44'/60'/0'/0/1"}

	db, err := accounts.NewDB(b.appDB)
	require.NoError(t, err)
	walletRootAddress, err := db.GetWalletRootAddress()
	require.NoError(t, err)

	require.NoError(t, err)

	derivedAddress, err := walletAPI.GetDerivedAddresses(context.Background(), password, walletRootAddress.String(), paths)
	require.NoError(t, err)
	require.Len(t, derivedAddress, 1)

	accountsService := statusNode.AccountService()
	require.NotNil(t, accountsService)
	accountsAPI := accountsService.AccountsAPI()

	err = accountsAPI.AddAccount(context.Background(), password, &accounts.Account{
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
	password := "some-password2" // nolint: goconst
	tmpdir := t.TempDir()

	b := NewGethStatusBackend()
	createAccountRequest := &requests.CreateAccount{
		DisplayName:           "some-display-name",
		CustomizationColor:    "#ffffff",
		Password:              password,
		BackupDisabledDataDir: tmpdir,
		LogFilePath:           tmpdir + "/log",
		Emoji:                 "some",
	}
	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})

	newAccount, err := b.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	statusNode := b.statusNode
	require.NotNil(t, statusNode)

	savedSettings, err := b.GetSettings()
	require.NoError(t, err)
	require.Empty(t, savedSettings.Fleet)

	accountsDB, err := b.accountsDB()
	require.NoError(t, err)
	err = accountsDB.SaveSettingField(settings.Fleet, params.FleetShardsTest)
	require.NoError(t, err)

	savedSettings, err = b.GetSettings()
	require.NoError(t, err)
	require.NotEmpty(t, savedSettings.Fleet)
	require.Equal(t, params.FleetShardsTest, *savedSettings.Fleet)

	require.NoError(t, b.Logout())

	loginAccountRequest := &requests.Login{
		KeyUID:   newAccount.KeyUID,
		Password: password,
	}
	require.NoError(t, b.LoginAccount(loginAccountRequest))
	select {
	case <-c:
		break
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	// Check is using the right fleet
	require.Equal(t, b.config.ClusterConfig.WakuNodes, params.DefaultWakuNodes(params.FleetShardsTest))

	require.NoError(t, b.Logout())
}

func TestWalletConfigOnLoginAccount(t *testing.T) {
	utils.Init()
	password := "some-password2" // nolint: goconst
	tmpdir := t.TempDir()
	poktToken := "grove-token"    // nolint: goconst
	infuraToken := "infura-token" // nolint: goconst
	alchemyEthereumMainnetToken := "alchemy-ethereum-mainnet-token"
	alchemyEthereumSepoliaToken := "alchemy-ethereum-sepolia-token"
	alchemyArbitrumMainnetToken := "alchemy-arbitrum-mainnet-token"
	alchemyArbitrumSepoliaToken := "alchemy-arbitrum-sepolia-token"
	alchemyOptimismMainnetToken := "alchemy-optimism-mainnet-token"
	alchemyOptimismSepoliaToken := "alchemy-optimism-sepolia-token"
	raribleMainnetAPIKey := "rarible-mainnet-api-key" // nolint: gosec
	raribleTestnetAPIKey := "rarible-testnet-api-key" // nolint: gosec

	b := NewGethStatusBackend()
	createAccountRequest := &requests.CreateAccount{
		DisplayName:           "some-display-name",
		CustomizationColor:    "#ffffff",
		Password:              password,
		BackupDisabledDataDir: tmpdir,
		LogFilePath:           tmpdir + "/log",
		Emoji:                 "some",
	}
	c := make(chan interface{}, 10)
	signal.SetMobileSignalHandler(func(data []byte) {
		if strings.Contains(string(data), "node.login") {
			c <- struct{}{}
		}
	})

	newAccount, err := b.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	statusNode := b.statusNode
	require.NotNil(t, statusNode)

	require.NoError(t, b.Logout())

	loginAccountRequest := &requests.Login{
		KeyUID:   newAccount.KeyUID,
		Password: password,
		WalletSecretsConfig: requests.WalletSecretsConfig{
			PoktToken:                   poktToken,
			InfuraToken:                 infuraToken,
			AlchemyEthereumMainnetToken: alchemyEthereumMainnetToken,
			AlchemyEthereumSepoliaToken: alchemyEthereumSepoliaToken,
			AlchemyArbitrumMainnetToken: alchemyArbitrumMainnetToken,
			AlchemyArbitrumSepoliaToken: alchemyArbitrumSepoliaToken,
			AlchemyOptimismMainnetToken: alchemyOptimismMainnetToken,
			AlchemyOptimismSepoliaToken: alchemyOptimismSepoliaToken,
			RaribleMainnetAPIKey:        raribleMainnetAPIKey,
			RaribleTestnetAPIKey:        raribleTestnetAPIKey,
		},
	}

	require.NoError(t, b.LoginAccount(loginAccountRequest))
	select {
	case <-c:
		break
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	require.Equal(t, b.config.WalletConfig.InfuraAPIKey, infuraToken)
	require.Equal(t, b.config.WalletConfig.AlchemyAPIKeys[mainnetChainID], alchemyEthereumMainnetToken)
	require.Equal(t, b.config.WalletConfig.AlchemyAPIKeys[sepoliaChainID], alchemyEthereumSepoliaToken)
	require.Equal(t, b.config.WalletConfig.AlchemyAPIKeys[arbitrumChainID], alchemyArbitrumMainnetToken)
	require.Equal(t, b.config.WalletConfig.AlchemyAPIKeys[arbitrumSepoliaChainID], alchemyArbitrumSepoliaToken)
	require.Equal(t, b.config.WalletConfig.AlchemyAPIKeys[optimismChainID], alchemyOptimismMainnetToken)
	require.Equal(t, b.config.WalletConfig.AlchemyAPIKeys[optimismSepoliaChainID], alchemyOptimismSepoliaToken)
	require.Equal(t, b.config.WalletConfig.RaribleMainnetAPIKey, raribleMainnetAPIKey)
	require.Equal(t, b.config.WalletConfig.RaribleTestnetAPIKey, raribleTestnetAPIKey)

	require.NoError(t, b.Logout())
}

func TestTestnetEnabledSettingOnCreateAccount(t *testing.T) {
	utils.Init()
	tmpdir := t.TempDir()

	b := NewGethStatusBackend()

	// Creating an account with test networks enabled
	createAccountRequest1 := &requests.CreateAccount{
		DisplayName:           "User-1",
		CustomizationColor:    "#ffffff",
		Emoji:                 "some",
		Password:              "password123",
		BackupDisabledDataDir: tmpdir,
		LogFilePath:           tmpdir + "/log",
		TestNetworksEnabled:   true,
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
		DisplayName:           "User-2",
		CustomizationColor:    "#ffffff",
		Emoji:                 "some",
		Password:              "password",
		BackupDisabledDataDir: tmpdir,
		LogFilePath:           tmpdir + "/log",
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
