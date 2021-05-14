package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/utils"
)

const (
	password = "abc"
)

// since `backend_test` grew too big, subscription tests are moved to its own part

func TestSubscriptionEthWithParamsDict(t *testing.T) {
	// a simple test to check the parameter parsing for eth_* filter subscriptions
	backend := NewGethStatusBackend()
	// initNodeAndLogin can fail and terminate the test, in that case stopNode must be executed anyway.
	defer func() {
		err := backend.StopNode()
		if err != node.ErrNoRunningNode {
			require.NoError(t, err)
		}
	}()

	initNodeAndLogin(t, backend)

	createSubscription(t, backend, `"eth_newFilter", [
	{
	 "fromBlock":"earliest",
	 "address":["0xc55cf4b03948d7ebc8b9e8bad92643703811d162","0xdee43a267e8726efd60c2e7d5b81552dcd4fa35c","0x703d7dc0bc8e314d65436adf985dda51e09ad43b","0xe639e24346d646e927f323558e6e0031bfc93581","0x2e7cd05f437eb256f363417fd8f920e2efa77540","0x57cc9b83730e6d22b224e9dc3e370967b44a2de0"],
	 "topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000005dc6108dc6296b052bbd33000553afe0ea576b5e",null]
    }
	]`)
}

func TestSubscriptionPendingTransaction(t *testing.T) {
	backend := NewGethStatusBackend()
	backend.allowAllRPC = true
	defer func() {
		err := backend.StopNode()
		if err != node.ErrNoRunningNode {
			require.NoError(t, err)
		}
	}()

	account, _ := initNodeAndLogin(t, backend)

	signals := make(chan string)
	defer func() {
		signal.ResetDefaultNodeNotificationHandler()
		close(signals)
	}()

	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		signals <- jsonEvent
	})

	subID := createSubscription(t, backend, `"eth_newPendingTransactionFilter", []`)

	createTxFmt := `
    {
		"jsonrpc":"2.0",
		"method":"eth_sendTransaction",
		"params":[
		{
		  "from": "%s",
		  "to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
		  "gas": "0x100000",
		  "gasPrice": "0x0",
		  "value": "0x0",
		  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
		}],
		"id":99
	}`

	txJSONResponse, err := backend.CallPrivateRPC(fmt.Sprintf(createTxFmt, account))
	require.NoError(t, err)

	createdTxID := extractResult(t, txJSONResponse)

	select {
	case event := <-signals:
		validateTxEvent(t, subID, event, createdTxID)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timeout waiting for subscription")
	}
}

// * * * * * * * * * * utility methods below * * * * * * * * * * *

func validateTxEvent(t *testing.T, expectedSubID string, jsonEvent string, txID string) {
	result := struct {
		Event signal.SubscriptionDataEvent `json:"event"`
		Type  string                       `json:"type"`
	}{}

	expectedData := []interface{}{
		txID,
	}

	require.NoError(t, json.Unmarshal([]byte(jsonEvent), &result))

	require.Equal(t, signal.EventSubscriptionsData, result.Type)
	require.Equal(t, expectedSubID, result.Event.FilterID)
	require.Equal(t, expectedData, result.Event.Data)
}

func extractResult(t *testing.T, jsonString string) string {
	resultMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonString), &resultMap)
	require.NoError(t, err)

	value, ok := resultMap["result"]
	require.True(t, ok, fmt.Sprintf("unexpected response: %s", jsonString))

	return value.(string)
}

func createSubscription(t *testing.T, backend StatusBackend, params string) string {
	createSubFmt := `
	{
		"jsonrpc": "2.0",
	"id": 10,
	    "method": "eth_subscribeSignal",
	"params": [%s]

	}`

	jsonResponse, err := backend.CallPrivateRPC(fmt.Sprintf(createSubFmt, params))
	require.NoError(t, err)

	return extractResult(t, jsonResponse)
}

func initNodeAndLogin(t *testing.T, backend *GethStatusBackend) (string, string) {
	utils.Init()

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)

	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
	err = backend.StartNode(config)
	require.NoError(t, err)
	_, info, _, err := backend.AccountManager().CreateAccount(password)
	require.NoError(t, err)

	loginParams := account.LoginParams{
		MainAccount: types.HexToAddress(info.WalletAddress),
		ChatAddress: types.HexToAddress(info.ChatAddress),
		Password:    password,
	}
	require.NoError(t, backend.AccountManager().SelectAccount(loginParams))

	unlockFmt := `
	{
		"jsonrpc": "2.0",
	"id": 11,
	    "method": "personal_unlockAccount",
		"params": ["%s", "%s"]
	}`

	unlockResult, err := backend.CallPrivateRPC(fmt.Sprintf(unlockFmt, info.WalletAddress, password))
	require.NoError(t, err)

	require.NotContains(t, unlockResult, "err")

	return info.WalletAddress, info.ChatPubKey
}
