package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions"
)

var (
	networks = json.RawMessage("{}")
	settings = accounts.Settings{
		Address:           types.HexToAddress("0xeC540f3745Ff2964AFC1171a5A0DD726d1F6B472"),
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

func TestBackendStartNodeConcurrently(t *testing.T) {
	utils.Init()

	backend := NewGethStatusBackend()
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

	backend := NewGethStatusBackend()
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

	backend := NewGethStatusBackend()
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

func TestBackendAccountsConcurrently(t *testing.T) {
	utils.Init()

	backend := NewGethStatusBackend()
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wgCreateAccounts sync.WaitGroup
	count := 3
	addressCh := make(chan [3]string, count) // use buffered channel to avoid blocking

	// create new accounts concurrently
	for i := 0; i < count; i++ {
		wgCreateAccounts.Add(1)
		go func(pass string) {
			_, accountInfo, _, err := backend.AccountManager().CreateAccount(pass)
			assert.NoError(t, err)
			addressCh <- [...]string{accountInfo.WalletAddress, accountInfo.ChatAddress, pass}
			wgCreateAccounts.Done()
		}("password-00" + fmt.Sprint(i))
	}

	// close addressCh as otherwise for loop never finishes
	go func() { wgCreateAccounts.Wait(); close(addressCh) }()

	// select, reselect or logout concurrently
	var wg sync.WaitGroup

	for tuple := range addressCh {
		wg.Add(1)
		go func(tuple [3]string) {
			loginParams := account.LoginParams{
				MainAccount: types.HexToAddress(tuple[0]),
				ChatAddress: types.HexToAddress(tuple[1]),
				Password:    tuple[2],
			}
			assert.NoError(t, backend.SelectAccount(loginParams))
			wg.Done()
		}(tuple)

		wg.Add(1)
		go func() {
			assert.NoError(t, backend.Logout())
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestBackendInjectChatAccount(t *testing.T) {
	utils.Init()

	backend := NewGethStatusBackend()
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	chatPrivKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	encryptionPrivKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)

	chatPrivKeyHex := hex.EncodeToString(gethcrypto.FromECDSA(chatPrivKey))
	chatPubKeyHex := types.EncodeHex(gethcrypto.FromECDSAPub(&chatPrivKey.PublicKey))
	encryptionPrivKeyHex := hex.EncodeToString(gethcrypto.FromECDSA(encryptionPrivKey))

	whisperService, err := backend.StatusNode().WhisperService()
	require.NoError(t, err)

	// public key should not be already in whisper
	require.False(t, whisperService.HasKeyPair(chatPubKeyHex), "identity already present in whisper")

	// call InjectChatAccount
	require.NoError(t, backend.InjectChatAccount(chatPrivKeyHex, encryptionPrivKeyHex))

	// public key should now be in whisper
	require.True(t, whisperService.HasKeyPair(chatPubKeyHex), "identity not injected into whisper")

	// wallet account should not be selected
	mainAccountAddress, err := backend.AccountManager().MainAccountAddress()
	require.Equal(t, types.Address{}, mainAccountAddress)
	require.Equal(t, account.ErrNoAccountSelected, err)

	// selected chat account should have the key injected previously
	chatAcc, err := backend.AccountManager().SelectedChatAccount()
	require.Nil(t, err)
	require.Equal(t, chatPrivKey, chatAcc.AccountKey.PrivateKey)
}

func TestBackendConnectionChangesConcurrently(t *testing.T) {
	connections := [...]string{wifi, cellular, unknown}
	backend := NewGethStatusBackend()
	count := 3

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			connIdx := rand.Intn(len(connections))
			backend.ConnectionChange(connections[connIdx], false)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestBackendConnectionChangesToOffline(t *testing.T) {
	b := NewGethStatusBackend()
	b.ConnectionChange(none, false)
	assert.True(t, b.connectionState.Offline)

	b.ConnectionChange(wifi, false)
	assert.False(t, b.connectionState.Offline)

	b.ConnectionChange("unknown-state", false)
	assert.False(t, b.connectionState.Offline)
}

func TestBackendCallRPCConcurrently(t *testing.T) {
	utils.Init()

	backend := NewGethStatusBackend()
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

	backend := NewGethStatusBackend()
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

	backend := NewGethStatusBackend()
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

func TestSignHash(t *testing.T) {
	utils.Init()

	backend := NewGethStatusBackend()
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))

	require.NoError(t, backend.StartNode(config))
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var testCases = []struct {
		name                 string
		chatPrivKeyHex       string
		hashHex              string
		expectedSignatureHex string
	}{
		{
			name:                 "tc1",
			chatPrivKeyHex:       "facadefacadefacadefacadefacadefacadefacadefacadefacadefacadefaca",
			hashHex:              "0xa4735de5193362fe856416000105cdfa6ce56265607311cebae93b26e5adf438",
			expectedSignatureHex: "0x176c971bae188c663614fc535ac9dbf62871dfeaadb38645809a510d28b3c4b0245415d5547c1b27f7cfea3341564f9c6981421144d3606b455346be69bd078c01",
		},
	}

	const dummyEncKey = "facadefacadefacadefacadefacadefacadefacadefacadefacadefacadefaca"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, backend.InjectChatAccount(tc.chatPrivKeyHex, dummyEncKey))

			signature, err := backend.SignHash(tc.hashHex)
			require.NoError(t, err)
			require.Equal(t, signature, tc.expectedSignatureHex)
		})
	}
}

func TestHashTypedData(t *testing.T) {
	utils.Init()

	backend := NewGethStatusBackend()
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
	tmpdir, err := ioutil.TempDir("", "verified-account-test-")
	require.NoError(t, err)
	defer os.Remove(tmpdir)
	backend := NewGethStatusBackend()
	backend.UpdateRootDataDir(tmpdir)
	require.NoError(t, backend.AccountManager().InitKeystore(filepath.Join(tmpdir, "keystore")))
	require.NoError(t, backend.ensureAppDBOpened(multiaccounts.Account{KeyUID: "0x1"}, password))
	config, err := params.NewNodeConfig(tmpdir, 178733)
	require.NoError(t, err)
	// this is for StatusNode().Config() call inside of the getVerifiedWalletAccount
	require.NoError(t, backend.StartNode(config))
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

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
		db := accounts.NewDB(backend.appDB)
		_, err = backend.AccountManager().ImportAccount(pkey, password)
		require.NoError(t, err)
		require.NoError(t, db.SaveAccounts([]accounts.Account{{Address: address}}))
		key, err := backend.getVerifiedWalletAccount(address.String(), "wrong-password")
		require.EqualError(t, err, "could not decrypt key with given password")
		require.Nil(t, key)
	})

	t.Run("Success", func(t *testing.T) {
		pkey, err := crypto.GenerateKey()
		require.NoError(t, err)
		address := crypto.PubkeyToAddress(pkey.PublicKey)
		db := accounts.NewDB(backend.appDB)
		_, err = backend.AccountManager().ImportAccount(pkey, password)
		require.NoError(t, err)
		require.NoError(t, db.SaveAccounts([]accounts.Account{{Address: address}}))
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
	tmpdir, err := ioutil.TempDir("", "login-with-key-test-")
	require.NoError(t, err)
	defer os.Remove(tmpdir)
	conf, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)
	keyhex := hex.EncodeToString(gethcrypto.FromECDSA(chatKey))

	require.NoError(t, b.AccountManager().InitKeystore(conf.KeyStoreDir))
	b.UpdateRootDataDir(conf.DataDir)
	require.NoError(t, b.OpenAccounts())

	address := crypto.PubkeyToAddress(walletKey.PublicKey)
	require.NoError(t, b.SaveAccountAndStartNodeWithKey(main, "test-pass", settings, conf, []accounts.Account{{Address: address, Wallet: true}}, keyhex))
	require.NoError(t, b.Logout())
	require.NoError(t, b.StopNode())

	require.NoError(t, b.StartNodeWithKey(main, "test-pass", keyhex))
	defer func() {
		assert.NoError(t, b.Logout())
		assert.NoError(t, b.StopNode())
	}()
	extkey, err := b.accountManager.SelectedChatAccount()
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(chatKey.PublicKey), extkey.Address)
}

func TestDeleteMulticcount(t *testing.T) {
	backend := NewGethStatusBackend()

	rootDataDir, err := ioutil.TempDir("", "test-keystore-dir")
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

	settings := accounts.Settings{
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
		settings,
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

	err = backend.DeleteMulticcount(account.KeyUID, keyStoreDir)
	require.NoError(t, err)

	files, err = ioutil.ReadDir(rootDataDir)
	require.NoError(t, err)
	require.Equal(t, 3, len(files))
}
