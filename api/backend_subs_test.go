package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
)

const (
	password = "abc"
)

// since `backend_test` grew too big, subscription tests are moved to its own part

func TestSubscriptionPendingTransaction(t *testing.T) {
	backend := NewStatusBackend()

	account := initNodeAndLogin(t, backend)

	defer func() { require.NoError(t, backend.StopNode()) }()

	signals := make(chan string)
	defer func() {
		signal.ResetDefaultNodeNotificationHandler()
		close(signals)
	}()

	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		signals <- jsonEvent
	})

	subID := createSubscription(t, backend, `"eth_newPendingTransactionFilter", []`)

	fmt.Println(subID)

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

	txJsonResponse, err := backend.CallPrivateRPC(fmt.Sprintf(createTxFmt, account))
	require.NoError(t, err)

	createdTxID := extractResult(t, txJsonResponse)

	select {
	case event := <-signals:
		validateTxEvent(t, subID, event, createdTxID)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timeout waiting for subscription")
	}
}

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
	require.True(t, ok)

	return value.(string)
}

func createSubscription(t *testing.T, backend *StatusBackend, params string) string {
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

func initNodeAndLogin(t *testing.T, backend *StatusBackend) string {
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)

	err = backend.StartNode(config)
	require.NoError(t, err)

	info, _, err := backend.AccountManager().CreateAccount(password)
	require.NoError(t, err)

	backend.AccountManager().SelectAccount(info.WalletAddress, info.ChatAddress, password)

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

	return info.WalletAddress
}
