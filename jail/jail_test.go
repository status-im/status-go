package jail_test

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/jail"
)

const (
	TEST_ADDRESS          = "0xadaf150b905cf5e6a778e553e15a139b6618bbb7"
	TEST_ADDRESS_PASSWORD = "asdfasdf"
	TEST_ADDRESS1         = "0xadd4d1d02e71c7360c53296968e59d57fd15e2ba"
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

func TestContractDeployment(t *testing.T) {
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

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(30*time.Second, completeQueuedTransaction, "TestContractDeployment")

	// replace transaction notification handler
	var txHash common.Hash
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})

			t.Logf("Transaction queued (will be completed in 5 secs): {id: %s}\n", event["id"].(string))
			time.Sleep(5 * time.Second)

			if err := geth.SelectAccount(TEST_ADDRESS, TEST_ADDRESS_PASSWORD); err != nil {
				t.Errorf("cannot select account: %v", TEST_ADDRESS)
				return
			}

			if txHash, err = geth.CompleteTransaction(event["id"].(string), TEST_ADDRESS_PASSWORD); err != nil {
				t.Errorf("cannot complete queued transation[%v]: %v", event["id"], err)
			} else {
				t.Logf("Contract created: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			}

			close(completeQueuedTransaction) // so that timeout is aborted
		}
	})

	_, err = vm.Run(`
		var responseValue = null;
		var testContract = web3.eth.contract([{"constant":true,"inputs":[{"name":"a","type":"int256"}],"name":"double","outputs":[{"name":"","type":"int256"}],"payable":false,"type":"function"}]);
		var test = testContract.new(
		{
			from: '` + TEST_ADDRESS + `',
			data: '0x6060604052341561000c57fe5b5b60a58061001b6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa14603a575bfe5b3415604157fe5b60556004808035906020019091905050606b565b6040518082815260200191505060405180910390f35b60008160020290505b9190505600a165627a7a72305820ccdadd737e4ac7039963b54cee5e5afb25fa859a275252bdcf06f653155228210029',
			gas: '` + strconv.Itoa(geth.DefaultGas) + `'
		}, function (e, contract){
			if (!e) {
				responseValue = contract.transactionHash
			}
		})
	`)
	if err != nil {
		t.Errorf("cannot run custom code on VM: %v", err)
		return
	}

	<-completeQueuedTransaction

	responseValue, err := vm.Get("responseValue")
	if err != nil {
		t.Errorf("cannot obtain result of custom code: %v", err)
		return
	}

	response, err := responseValue.ToString()
	if err != nil {
		t.Errorf("cannot parse result: %v", err)
		return
	}

	expectedResponse := txHash.Hex()
	if !reflect.DeepEqual(response, expectedResponse) {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		return
	}
}

func TestJailBlockingDuringSync(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// reset chain data, forcing re-sync
	if err := geth.NodeManagerInstance().ResetChainData(); err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second) // so that sync has enough time to start

	jailInstance := jail.Init("")
	jailInstance.Parse(CHAT_ID_CALL, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := jailInstance.GetVM(CHAT_ID_CALL)
	if err != nil {
		t.Errorf("cannot get VM: %v", err)
		return
	}

	// define helper types
	type ntfyHandler struct {
		waiter   chan interface{}
		activate func(waiter chan interface{})
	}

	type TestCase struct {
		name    string
		code    string
		handler ntfyHandler
		check   func(responseValue otto.Value, returnedByWaiter interface{}) bool
	}

	// make sure you panic if transaction complete doesn't return
	abortPanic := make(chan struct{})
	geth.PanicAfter(60*time.Second, abortPanic, "TestJailBlockingDuringSync")

	makeTxCompleteHandler := func() ntfyHandler {
		return ntfyHandler{
			waiter: make(chan interface{}, 1),
			activate: func(waiter chan interface{}) {
				geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
					var txHash common.Hash
					var envelope geth.SignalEnvelope
					if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
						t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
						return
					}
					if envelope.Type == geth.EventTransactionQueued {
						event := envelope.Event.(map[string]interface{})

						glog.V(logger.Info).Infof("transaction queued (will be completed immediately): {id: %s}\n", event["id"].(string))

						if err := geth.SelectAccount(TEST_ADDRESS, TEST_ADDRESS_PASSWORD); err != nil {
							t.Errorf("cannot select account: %v", TEST_ADDRESS)
							return
						}

						if txHash, err = geth.CompleteTransaction(event["id"].(string), TEST_ADDRESS_PASSWORD); err != nil {
							glog.V(logger.Info).Infof("cannot complete queued transation[%v]: %v", event["id"], err)
							waiter <- common.Hash{}
						} else {
							glog.V(logger.Info).Infof("tx created: https://testnet.etherscan.io/tx/%s", txHash.Hex())
							waiter <- txHash
						}

						close(waiter)
					}
				})
			},
		}
	}
	makeDoNothingHandler := func() ntfyHandler {
		return ntfyHandler{
			waiter: make(chan interface{}, 1),
			activate: func(waiter chan interface{}) {
				close(waiter)
			},
		}
	}

	testCases := []TestCase{
		{
			"getBalance",
			`
				var getBalanceResponse = web3.fromWei(web3.eth.getBalance('` + TEST_ADDRESS + `'), 'ether');
			`,
			makeDoNothingHandler(),
			func(responseValue otto.Value, returnedByWaiter interface{}) bool {
				response, err := responseValue.ToInteger()
				if err != nil {
					t.Errorf("cannot parse result: %v", err)
					return false
				}

				if response > 9000 {
					return true
				}

				return false
			},
		},
		{
			"getTransactionCount",
			`
				var getTransactionCountResponse = web3.eth.getTransactionCount('` + TEST_ADDRESS + `');
			`,
			makeDoNothingHandler(),
			func(responseValue otto.Value, returnedByWaiter interface{}) bool {
				response, err := responseValue.ToInteger()
				if err != nil {
					t.Errorf("cannot parse result: %v", err)
					return false
				}

				if response > 1000 {
					return true
				}

				return false
			},
		},
		{
			"sendEther",
			`
				var sendEtherResponse = "0x0000000000000000000000000000000000000000000000000000000000000000"
				web3.eth.sendTransaction(
				{
					from: '` + TEST_ADDRESS + `', to: '` + TEST_ADDRESS1 + `', value: web3.toWei(0.000002, "ether")
				}, function (e, hash) {
					if (!e) {
						sendEtherResponse = hash
					} else {
						sendEtherResponse = e
					}
				})
			`,
			makeTxCompleteHandler(),
			func(responseValue otto.Value, returnedByWaiter interface{}) bool {
				response, err := responseValue.ToString()
				if err != nil {
					t.Errorf("cannot parse result: %v", err)
					return false
				}

				expectedResponse1 := "0x0000000000000000000000000000000000000000000000000000000000000000"
				if txHash, ok := returnedByWaiter.(common.Hash); ok {
					expectedResponse1 = txHash.Hex()
				}

				expectedResponse2 := `Error: Nonce too low`
				if !reflect.DeepEqual(response, expectedResponse1) && !reflect.DeepEqual(response, expectedResponse2) {
					t.Errorf("expected response is not returned: expected %s or %s, got %s", expectedResponse1, expectedResponse2, response)
					return false
				}

				return true
			},
		},
		{
			"createContract",
			`
				var createContractResponse = "0x0000000000000000000000000000000000000000000000000000000000000000";
				var testContract = web3.eth.contract([{"constant":true,"inputs":[{"name":"a","type":"int256"}],"name":"double","outputs":[{"name":"","type":"int256"}],"payable":false,"type":"function"}]);
				var test = testContract.new(
				{
					from: '` + TEST_ADDRESS + `',
					data: '0x6060604052341561000c57fe5b5b60a58061001b6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa14603a575bfe5b3415604157fe5b60556004808035906020019091905050606b565b6040518082815260200191505060405180910390f35b60008160020290505b9190505600a165627a7a72305820ccdadd737e4ac7039963b54cee5e5afb25fa859a275252bdcf06f653155228210029',
					gas: '` + strconv.Itoa(geth.DefaultGas) + `'
				}, function (e, contract){
					if (!e) {
						createContractResponse = contract.transactionHash
					} else {
						createContractResponse = e
					}
				})
			`,
			makeTxCompleteHandler(),
			func(responseValue otto.Value, returnedByWaiter interface{}) bool {
				response, err := responseValue.ToString()
				if err != nil {
					t.Errorf("cannot parse result: %v", err)
					return false
				}

				expectedResponse1 := ""
				if txHash, ok := returnedByWaiter.(common.Hash); ok {
					expectedResponse1 = txHash.Hex()
				}

				expectedResponse2 := `Error: Nonce too low`
				if !reflect.DeepEqual(response, expectedResponse1) && !reflect.DeepEqual(response, expectedResponse2) {
					t.Errorf("expected response is not returned: expected %s or %s, got %s", expectedResponse1, expectedResponse2, response)
					return false
				}

				return true
			},
		},
	}

	for _, testCase := range testCases {
		// prepare notification handler
		testCase.handler.activate(testCase.handler.waiter)

		// execute test code
		vm.Run(testCase.code)

		// wait for handler (if necessary)
		returnedByWaiter := <-testCase.handler.waiter

		// process response
		responseValue, err := vm.Get(testCase.name + "Response")
		if err != nil {
			t.Errorf("cannot obtain result of custom code: %v", err)
			return
		}

		if !testCase.check(responseValue, returnedByWaiter) {
			t.Fatalf("case failed: %v", testCase.name)
			return
		}
	}

	abortPanic <- struct{}{}

	// allow to sync (for other tests)
	time.Sleep(120 * time.Second)

}
