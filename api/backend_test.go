package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendStartNodeConcurrently(t *testing.T) {
	backend := NewStatusBackend()
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
	backend := NewStatusBackend()
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
	backend := NewStatusBackend()
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
	backend := NewStatusBackend()
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
			accountInfo, _, err := backend.AccountManager().CreateAccount(pass)
			assert.NoError(t, err)
			addressCh <- [...]string{accountInfo.WalletAddress, accountInfo.ChatAddress, pass}
			wgCreateAccounts.Done()
		}("password-00" + string(i))
	}

	// close addressCh as otherwise for loop never finishes
	go func() { wgCreateAccounts.Wait(); close(addressCh) }()

	// select, reselect or logout concurrently
	var wg sync.WaitGroup

	for tuple := range addressCh {
		wg.Add(1)
		go func(tuple [3]string) {
			loginParams := account.LoginParams{
				MainAccount: common.HexToAddress(tuple[0]),
				ChatAddress: common.HexToAddress(tuple[1]),
				Password:    tuple[2],
			}
			assert.NoError(t, backend.SelectAccount(loginParams))
			wg.Done()
		}(tuple)

		wg.Add(1)
		go func() {
			assert.NoError(t, backend.reSelectAccount())
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			assert.NoError(t, backend.Logout())
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestBackendInjectChatAccount(t *testing.T) {
	backend := NewStatusBackend()
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	chatPrivKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	encryptionPrivKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	chatPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(chatPrivKey))
	chatPubKeyHex := hexutil.Encode(crypto.FromECDSAPub(&chatPrivKey.PublicKey))
	encryptionPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(encryptionPrivKey))

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
	require.Equal(t, common.Address{}, mainAccountAddress)
	require.Equal(t, account.ErrNoAccountSelected, err)

	// selected chat account should have the key injected previously
	chatAcc, err := backend.AccountManager().SelectedChatAccount()
	require.Nil(t, err)
	require.Equal(t, chatPrivKey, chatAcc.AccountKey.PrivateKey)
}

func TestBackendConnectionChangesConcurrently(t *testing.T) {
	connections := [...]string{wifi, cellular, unknown}
	backend := NewStatusBackend()
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
	b := NewStatusBackend()
	b.ConnectionChange(none, false)
	assert.True(t, b.connectionState.Offline)

	b.ConnectionChange(wifi, false)
	assert.False(t, b.connectionState.Offline)

	b.ConnectionChange("unknown-state", false)
	assert.False(t, b.connectionState.Offline)
}

func TestBackendCallRPCConcurrently(t *testing.T) {
	backend := NewStatusBackend()
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
	backend := NewStatusBackend()

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
	backend := NewStatusBackend()
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
	backend := NewStatusBackend()

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
	backend := NewStatusBackend()
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
	backend := NewStatusBackend()
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
	backend := NewStatusBackend()
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
	assert.NotEqual(t, common.Hash{}, hash)
}
