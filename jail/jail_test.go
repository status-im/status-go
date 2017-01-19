package jail_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/jail"
)

const (
	TEST_ADDRESS          = "0xadaf150b905cf5e6a778e553e15a139b6618bbb7"
	TEST_ADDRESS_PASSWORD = "asdfasdf"
	CHAT_ID_INIT          = "CHAT_ID_INIT_TEST"
	CHAT_ID_CALL          = "CHAT_ID_CALL_TEST"
	CHAT_ID_SEND          = "CHAT_ID_CALL_SEND"
	CHAT_ID_NON_EXISTENT  = "CHAT_IDNON_EXISTENT"

	TESTDATA_STATUS_JS  = "testdata/status.js"
	TESTDATA_TX_SEND_JS = "testdata/tx-send/"
)

func TestJailUnInited(t *testing.T) {
	errorWrapper := func(err error) string {
		return `{"error":"` + err.Error() + `"}`
	}

	expectedError := errorWrapper(jail.ErrInvalidJail)

	var jailInstance *jail.Jail
	response := jailInstance.Parse(CHAT_ID_CALL, ``)
	if response != expectedError {
		t.Errorf("error expected, but got: %v", response)
	}

	response = jailInstance.Call(CHAT_ID_CALL, `["commands", "testCommand"]`, `{"val": 12}`)
	if response != expectedError {
		t.Errorf("error expected, but got: %v", response)
	}

	_, err := jailInstance.GetVM(CHAT_ID_CALL)
	if err != jail.ErrInvalidJail {
		t.Errorf("error expected, but got: %v", err)
	}

	_, err = jailInstance.RPCClient()
	if err != jail.ErrInvalidJail {
		t.Errorf("error expected, but got: %v", err)
	}

	// now make sure that if Init is called, then Parse doesn't produce any error
	jailInstance = jail.Init(``)
	if jailInstance == nil {
		t.Error("jail instance shouldn't be nil at this point")
		return
	}
	statusJS := geth.LoadFromFile(TESTDATA_STATUS_JS) + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	response = jailInstance.Parse(CHAT_ID_CALL, statusJS)
	expectedResponse := `{"result": {"commands":{},"responses":{}}}`
	if response != expectedResponse {
		t.Errorf("unexpected response received: %v", response)
	}

	// however, we still expect issue voiced if somebody tries to execute code with Call
	response = jailInstance.Call(CHAT_ID_CALL, `["commands", "testCommand"]`, `{"val": 12}`)
	if response != errorWrapper(geth.ErrInvalidGethNode) {
		t.Errorf("error expected, but got: %v", response)
	}

	// make sure that Call() succeeds when node is started
	err = geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}
	response = jailInstance.Call(CHAT_ID_CALL, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse = `{"result": 144}`
	if response != expectedResponse {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		return
	}
}

func TestJailInit(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	initCode := `
	var _status_catalog = {
		foo: 'bar'
	};
	`
	jailInstance := jail.Init(initCode)

	extraCode := `
	var extraFunc = function (x) {
	  return x * x;
	};
	`
	response := jailInstance.Parse(CHAT_ID_INIT, extraCode)

	expectedResponse := `{"result": {"foo":"bar"}}`

	if !reflect.DeepEqual(expectedResponse, response) {
		t.Error("Expected output not returned from jail.Parse()")
		return
	}
}

func TestJailFunctionCall(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")

	// load Status JS and add test command to it
	statusJS := geth.LoadFromFile(TESTDATA_STATUS_JS) + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	jailInstance.Parse(CHAT_ID_CALL, statusJS)

	// call with wrong chat id
	response := jailInstance.Call(CHAT_ID_NON_EXISTENT, "", "")
	expectedError := `{"error":"Cell[CHAT_IDNON_EXISTENT] doesn't exist."}`
	if response != expectedError {
		t.Errorf("expected error is not returned: expected %s, got %s", expectedError, response)
		return
	}

	// call extraFunc()
	response = jailInstance.Call(CHAT_ID_CALL, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse := `{"result": 144}`
	if response != expectedResponse {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		return
	}
}

func TestJailRPCSend(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")

	// load Status JS and add test command to it
	statusJS := geth.LoadFromFile(TESTDATA_STATUS_JS)
	jailInstance.Parse(CHAT_ID_CALL, statusJS)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := jailInstance.GetVM(CHAT_ID_CALL)
	if err != nil {
		t.Errorf("cannot get VM: %v", err)
		return
	}

	// internally (since we replaced `web3.send` with `jail.Send`)
	// all requests to web3 are forwarded to `jail.Send`
	_, err = vm.Run(`
	    var balance = web3.eth.getBalance("` + TEST_ADDRESS + `");
		var sendResult = web3.fromWei(balance, "ether")
	`)
	if err != nil {
		t.Errorf("cannot run custom code on VM: %v", err)
		return
	}

	value, err := vm.Get("sendResult")
	if err != nil {
		t.Errorf("cannot obtain result of balance check operation: %v", err)
		return
	}

	balance, err := value.ToFloat()
	if err != nil {
		t.Errorf("cannot obtain result of balance check operation: %v", err)
		return
	}

	if balance < 100 {
		t.Error("wrong balance (there should be lots of test Ether on that account)")
		return
	}

	t.Logf("Balance of %.2f ETH found on '%s' account", balance, TEST_ADDRESS)
}

func TestJailSendQueuedTransaction(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(TEST_ADDRESS, TEST_ADDRESS_PASSWORD); err != nil {
		t.Errorf("cannot select account: %v", TEST_ADDRESS)
		return
	}

	txParams := `{
  		"from": "` + TEST_ADDRESS + `",
  		"to": "0xf82da7547534045b4e00442bc89e16186cf8c272",
  		"value": "0.000001"
	}`

	txCompletedSuccessfully := make(chan struct{})
	txCompletedCounter := make(chan struct{})
	txHashes := make(chan common.Hash)

	// replace transaction notification handler
	requireMessageId := false
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			messageId, ok := event["message_id"].(string)
			if !ok {
				t.Error("Message id is required, but not found")
				return
			}
			if requireMessageId {
				if len(messageId) == 0 {
					t.Error("Message id is required, but not provided")
					return
				}
			} else {
				if len(messageId) != 0 {
					t.Error("Message id is not required, but provided")
					return
				}
			}
			t.Logf("Transaction queued (will be completed in 5 secs): {id: %s}\n", event["id"].(string))
			time.Sleep(5 * time.Second)

			var txHash common.Hash
			if txHash, err = geth.CompleteTransaction(event["id"].(string), TEST_ADDRESS_PASSWORD); err != nil {
				t.Errorf("cannot complete queued transation[%v]: %v", event["id"], err)
			} else {
				t.Logf("Transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			}

			txCompletedSuccessfully <- struct{}{} // so that timeout is aborted
			txHashes <- txHash
			txCompletedCounter <- struct{}{}
		}
	})

	type testCommand struct {
		command          string
		params           string
		expectedResponse string
	}
	type testCase struct {
		name             string
		file             string
		requireMessageId bool
		commands         []testCommand
	}

	tests := []testCase{
		{
			// no context or message id
			name:             "Case 1: no message id or context in inited JS",
			file:             "no-message-id-or-context.js",
			requireMessageId: false,
			commands: []testCommand{
				{
					`["commands", "send"]`,
					txParams,
					`{"result": {"transaction-hash":"TX_HASH"}}`,
				},
				{
					`["commands", "getBalance"]`,
					`{"address": "` + TEST_ADDRESS + `"}`,
					`{"result": {"balance":42}}`,
				},
			},
		},
		{
			// context is present in inited JS (but no message id is there)
			name:             "Case 2: context is present in inited JS (but no message id is there)",
			file:             "context-no-message-id.js",
			requireMessageId: false,
			commands: []testCommand{
				{
					`["commands", "send"]`,
					txParams,
					`{"result": {"context":{"` + geth.SendTransactionRequest + `":true},"result":{"transaction-hash":"TX_HASH"}}}`,
				},
				{
					`["commands", "getBalance"]`,
					`{"address": "` + TEST_ADDRESS + `"}`,
					`{"result": {"context":{},"result":{"balance":42}}}`, // note emtpy (but present) context!
				},
			},
		},
		{
			// message id is present in inited JS, but no context is there
			name:             "Case 3: message id is present, context is not present",
			file:             "message-id-no-context.js",
			requireMessageId: true,
			commands: []testCommand{
				{
					`["commands", "send"]`,
					txParams,
					`{"result": {"transaction-hash":"TX_HASH"}}`,
				},
				{
					`["commands", "getBalance"]`,
					`{"address": "` + TEST_ADDRESS + `"}`,
					`{"result": {"balance":42}}`, // note emtpy context!
				},
			},
		},
		{
			// both message id and context are present in inited JS (this UC is what we normally expect to see)
			name:             "Case 4: both message id and context are present",
			file:             "tx-send.js",
			requireMessageId: true,
			commands: []testCommand{
				{
					`["commands", "send"]`,
					txParams,
					`{"result": {"context":{"eth_sendTransaction":true,"message_id":"foobar"},"result":{"transaction-hash":"TX_HASH"}}}`,
				},
				{
					`["commands", "getBalance"]`,
					`{"address": "` + TEST_ADDRESS + `"}`,
					`{"result": {"context":{"message_id":"42"},"result":{"balance":42}}}`, // message id in context, but default one is used!
				},
			},
		},
	}

	for _, test := range tests {
		jailInstance := jail.Init(geth.LoadFromFile(TESTDATA_TX_SEND_JS + test.file))
		geth.PanicAfter(60*time.Second, txCompletedSuccessfully, test.name)
		jailInstance.Parse(CHAT_ID_SEND, ``)

		requireMessageId = test.requireMessageId

		for _, command := range test.commands {
			go func(jail *jail.Jail, test testCase, command testCommand) {
				t.Logf("->%s: %s", test.name, command.command)
				response := jail.Call(CHAT_ID_SEND, command.command, command.params)
				var txHash common.Hash
				if command.command == `["commands", "send"]` {
					txHash = <-txHashes
				}
				expectedResponse := strings.Replace(command.expectedResponse, "TX_HASH", txHash.Hex(), 1)
				if response != expectedResponse {
					t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
					return
				}
			}(jailInstance, test, command)
		}
		<-txCompletedCounter
	}
}

func TestJailMultipleInitSingletonJail(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance1 := jail.Init("")
	jailInstance2 := jail.Init("")
	jailInstance3 := jail.New()
	jailInstance4 := jail.GetInstance()

	if !reflect.DeepEqual(jailInstance1, jailInstance2) {
		t.Error("singleton property of jail instance is violated")
	}
	if !reflect.DeepEqual(jailInstance2, jailInstance3) {
		t.Error("singleton property of jail instance is violated")
	}
	if !reflect.DeepEqual(jailInstance3, jailInstance4) {
		t.Error("singleton property of jail instance is violated")
	}
}

func TestJailGetVM(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")

	expectedError := `Cell[` + CHAT_ID_NON_EXISTENT + `] doesn't exist.`
	_, err = jailInstance.GetVM(CHAT_ID_NON_EXISTENT)
	if err == nil || err.Error() != expectedError {
		t.Error("expected error, but call succeeded")
	}

	// now let's create VM..
	jailInstance.Parse(CHAT_ID_CALL, ``)
	// ..and see if VM becomes available
	_, err = jailInstance.GetVM(CHAT_ID_CALL)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIsConnected(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")
	jailInstance.Parse(CHAT_ID_CALL, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := jailInstance.GetVM(CHAT_ID_CALL)
	if err != nil {
		t.Errorf("cannot get VM: %v", err)
		return
	}

	_, err = vm.Run(`
	    var responseValue = web3.isConnected();
	    responseValue = JSON.stringify(responseValue);
	`)
	if err != nil {
		t.Errorf("cannot run custom code on VM: %v", err)
		return
	}

	responseValue, err := vm.Get("responseValue")
	if err != nil {
		t.Errorf("cannot obtain result of isConnected(): %v", err)
		return
	}

	response, err := responseValue.ToString()
	if err != nil {
		t.Errorf("cannot parse result: %v", err)
		return
	}

	expectedResponse := `{"jsonrpc":"2.0","result":true}`
	if !reflect.DeepEqual(response, expectedResponse) {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		return
	}
}

func TestLocalStorageSet(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")
	jailInstance.Parse(CHAT_ID_CALL, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := jailInstance.GetVM(CHAT_ID_CALL)
	if err != nil {
		t.Errorf("cannot get VM: %v", err)
		return
	}

	testData := "foobar"

	opCompletedSuccessfully := make(chan struct{}, 1)

	// replace transaction notification handler
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == jail.EventLocalStorageSet {
			event := envelope.Event.(map[string]interface{})
			chatId, ok := event["chat_id"].(string)
			if !ok {
				t.Error("Chat id is required, but not found")
				return
			}
			if chatId != CHAT_ID_CALL {
				t.Errorf("incorrect chat id: expected %q, got: %q", CHAT_ID_CALL, chatId)
				return
			}

			actualData, ok := event["data"].(string)
			if !ok {
				t.Error("Data field is required, but not found")
				return
			}

			if actualData != testData {
				t.Errorf("incorrect data: expected %q, got: %q", testData, actualData)
				return
			}

			t.Logf("event processed: %s", jsonEvent)
			opCompletedSuccessfully <- struct{}{} // so that timeout is aborted
		}
	})

	_, err = vm.Run(`
	    var responseValue = localStorage.set("` + testData + `");
	    responseValue = JSON.stringify(responseValue);
	`)
	if err != nil {
		t.Errorf("cannot run custom code on VM: %v", err)
		return
	}

	// make sure that signal is sent (and its parameters are correct)
	select {
	case <-opCompletedSuccessfully:
		// pass
	case <-time.After(3 * time.Second):
		t.Error("operation timed out")
	}

	responseValue, err := vm.Get("responseValue")
	if err != nil {
		t.Errorf("cannot obtain result of localStorage.set(): %v", err)
		return
	}

	response, err := responseValue.ToString()
	if err != nil {
		t.Errorf("cannot parse result: %v", err)
		return
	}

	expectedResponse := `{"jsonrpc":"2.0","result":true}`
	if !reflect.DeepEqual(response, expectedResponse) {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		return
	}
}

func TestLongRunningCallHalting(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")
	jailInstance.Config.LongRunningJobTimeout = 3 * time.Second

	statusJS := geth.LoadFromFile(TESTDATA_STATUS_JS) + `;
	_status_catalog.commands["neverEndingCommand"] = function () {
		while (true) {
			// loops forever
		}
	};`
	jailInstance.Parse(CHAT_ID_CALL, statusJS)

	correctlyHalted := make(chan struct{}, 1)
	go func() {
		response := jailInstance.Call(CHAT_ID_CALL, `["commands", "neverEndingCommand"]`, `""`)
		expectedResponse := `{"error":"long running job timed out"}`
		if response != expectedResponse {
			t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		}

		correctlyHalted <- struct{}{}
	}()

	select {
	case <-correctlyHalted:
	// pass
	case <-time.After((jail.JailedRuntimeRequestTimeout + 5) * time.Second):
		t.Error("long running jail.Call() request hasn't been halted")
	}
}

func TestLongRunningParseHalting(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")
	jailInstance.Config.LongRunningJobTimeout = 3 * time.Second

	statusJS := geth.LoadFromFile(TESTDATA_STATUS_JS) + `;
	while (true) {
		// loops forever
	}`

	correctlyHalted := make(chan struct{}, 1)
	go func() {
		response := jailInstance.Parse(CHAT_ID_CALL, statusJS)
		expectedResponse := `{"error":"long running job timed out"}`
		if response != expectedResponse {
			t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		}

		correctlyHalted <- struct{}{}
	}()

	select {
	case <-correctlyHalted:
		// pass
	case <-time.After((jail.JailedRuntimeRequestTimeout + 5) * time.Second):
		t.Error("long running jail.Parse() request hasn't been halted")
	}
}
