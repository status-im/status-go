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

func TestSubscriptionEthWithParamsDict(t *testing.T) {
	// a simple test to check the parameter parsing for eth_* filter subscriptions
	backend := NewStatusBackend()

	initNodeAndLogin(t, backend)

	defer func() { require.NoError(t, backend.StopNode()) }()

	createSubscription(t, backend, fmt.Sprintf(`"eth_newFilter", [
	{
	 "fromBlock":"earliest",
	 "address":["0xc55cf4b03948d7ebc8b9e8bad92643703811d162","0xdee43a267e8726efd60c2e7d5b81552dcd4fa35c","0x703d7dc0bc8e314d65436adf985dda51e09ad43b","0xe639e24346d646e927f323558e6e0031bfc93581","0x2e7cd05f437eb256f363417fd8f920e2efa77540","0x57cc9b83730e6d22b224e9dc3e370967b44a2de0"],
	 "topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000005dc6108dc6296b052bbd33000553afe0ea576b5e",null]
    }
	]`))
}

func TestSubscriptionPendingTransaction(t *testing.T) {
	backend := NewStatusBackend()
	backend.allowAllRPC = true

	account, _ := initNodeAndLogin(t, backend)

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

func TestSubscriptionWhisperEnvelopes(t *testing.T) {
	backend := NewStatusBackend()

	initNodeAndLogin(t, backend)

	defer func() { require.NoError(t, backend.StopNode()) }()

	signals := make(chan string)
	defer func() {
		signal.ResetDefaultNodeNotificationHandler()
		close(signals)
	}()

	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		signals <- jsonEvent
	})

	topic := "0x12341234"
	payload := "0x12312312"

	shhGenSymKeyJSONResponse, err := backend.CallPrivateRPC(`{"jsonrpc":"2.0","method":"shh_generateSymKeyFromPassword","params":["test"],"id":119}`)
	require.NoError(t, err)
	symKeyID := extractResult(t, shhGenSymKeyJSONResponse)

	subID := createSubscription(t, backend, fmt.Sprintf(`"shh_newMessageFilter", [{ "symKeyID": "%s", "topics": ["%s"] }]`, symKeyID, topic))

	sendMessageFmt := `
	{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [{
			"ttl": 7,
			"symKeyID": "%s",
			"topic": "%s",
			"powTarget": 2.01,
			"powTime": 2,
			"payload": "%s"
		}],
		"id":11
	}`

	numberOfEnvelopes := 5

	for i := 0; i < numberOfEnvelopes; i++ {
		_, err = backend.CallPrivateRPC(fmt.Sprintf(sendMessageFmt, symKeyID, topic, payload))
		require.NoError(t, err)
	}

	select {
	case event := <-signals:
		validateShhEvent(t, event, subID, numberOfEnvelopes, topic, payload)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timeout waiting for subscription")
	}
}

// * * * * * * * * * * utility methods below * * * * * * * * * * *

func validateShhEvent(t *testing.T, jsonEvent string, expectedSubID string, numberOfEnvelopes int, topic string, payload string) {
	result := struct {
		Event signal.SubscriptionDataEvent `json:"event"`
		Type  string                       `json:"type"`
	}{}

	require.NoError(t, json.Unmarshal([]byte(jsonEvent), &result))

	require.Equal(t, signal.EventSubscriptionsData, result.Type)
	require.Equal(t, expectedSubID, result.Event.FilterID)

	require.Equal(t, numberOfEnvelopes, len(result.Event.Data))

	for _, item := range result.Event.Data {
		dict := item.(map[string]interface{})
		require.Equal(t, dict["topic"], topic)
		require.Equal(t, dict["payload"], payload)
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
	require.True(t, ok, fmt.Sprintf("unexpected response: %s", jsonString))

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

func initNodeAndLogin(t *testing.T, backend *StatusBackend) (string, string) {
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)

	err = backend.StartNode(config)
	require.NoError(t, err)

	info, _, err := backend.AccountManager().CreateAccount(password)
	require.NoError(t, err)

	require.NoError(t, backend.AccountManager().SelectAccount(info.WalletAddress, info.ChatAddress, password))

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
