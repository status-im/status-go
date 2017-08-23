package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/status-im/status-go/static"
)

const (
	whisperMessage1 = `test message 1 (K1 -> K2, signed+encrypted, from us)`
	whisperMessage2 = `test message 2 (K1 -> K1, signed+encrypted to ourselves)`
	whisperMessage3 = `test message 3 (K1 -> "", signed broadcast)`
	whisperMessage4 = `test message 4 ("" -> "", anon broadcast)`
	whisperMessage5 = `test message 5 ("" -> K1, encrypted anon broadcast)`
	whisperMessage6 = `test message 6 (K2 -> K1, signed+encrypted, to us)`
	txSendFolder    = "testdata/jail/tx-send/"
	testChatID      = "testChat"
)

func (s *BackendTestSuite) TestJailSendQueuedTransaction() {
	require := s.Require()

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	txParams := `{
  		"from": "` + TestConfig.Account1.Address + `",
  		"to": "` + TestConfig.Account2.Address + `",
  		"value": "0.000001"
	}`

	txCompletedSuccessfully := make(chan struct{})
	txHashes := make(chan gethcommon.Hash)

	// replace transaction notification handler
	requireMessageId := false
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == node.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			messageId, ok := event["message_id"].(string)
			s.True(ok, "Message id is required, but not found")
			if requireMessageId {
				if len(messageId) == 0 {
					s.Fail("Message id is required, but not provided")
					return
				}
			} else {
				if len(messageId) != 0 {
					s.Fail("Message id is not required, but provided")
					return
				}
			}
			log.Info("Transaction queued (will be completed shortly)", "id", event["id"].(string))

			var txHash gethcommon.Hash
			if txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password); err != nil {
				s.Fail(fmt.Sprintf("cannot complete queued transaction[%v]: %v", event["id"], err))
			} else {
				log.Info("Transaction complete", "URL", "https://ropsten.etherscan.io/tx/%s"+txHash.Hex())
			}

			txCompletedSuccessfully <- struct{}{} // so that timeout is aborted
			txHashes <- txHash
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
					`{"address": "` + TestConfig.Account1.Address + `"}`,
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
					`{"result": {"context":{"` + params.SendTransactionMethodName + `":true},"result":{"transaction-hash":"TX_HASH"}}}`,
				},
				{
					`["commands", "getBalance"]`,
					`{"address": "` + TestConfig.Account1.Address + `"}`,
					`{"result": {"context":{},"result":{"balance":42}}}`, // note empty (but present) context!
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
					`{"address": "` + TestConfig.Account1.Address + `"}`,
					`{"result": {"balance":42}}`, // note empty context!
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
					`["commands", "getBalance"]`,
					`{"address": "` + TestConfig.Account1.Address + `"}`,
					`{"result": {"context":{"message_id":"42"},"result":{"balance":42}}}`, // message id in context, but default one is used!
				},
				{
					`["commands", "send"]`,
					txParams,
					`{"result": {"context":{"eth_sendTransaction":true,"message_id":"foobar"},"result":{"transaction-hash":"TX_HASH"}}}`,
				},
			},
		},
	}

	jailInstance := s.backend.JailManager()
	for _, test := range tests {
		jailInstance.BaseJS(string(static.MustAsset(txSendFolder + test.file)))
		common.PanicAfter(1*time.Minute, txCompletedSuccessfully, test.name)
		jailInstance.Parse(testChatID, ``)

		requireMessageId = test.requireMessageId

		for _, command := range test.commands {
			s.T().Logf("%s: %s", test.name, command.command)
			response := jailInstance.Call(testChatID, command.command, command.params)
			var txHash gethcommon.Hash
			if command.command == `["commands", "send"]` {
				txHash = <-txHashes
			}
			expectedResponse := strings.Replace(command.expectedResponse, "TX_HASH", txHash.Hex(), 1)
			s.Require().Equal(expectedResponse, response)
		}
	}
}

func (s *BackendTestSuite) TestContractDeployment() {
	require := s.Require()

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	// Allow to sync, otherwise you'll get "Nonce too low."
	time.Sleep(TestConfig.Node.SyncSeconds * time.Second)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	jailInstance := s.backend.JailManager()
	jailInstance.Parse(testChatID, "")

	cell, err := jailInstance.GetJailCell(testChatID)
	require.NoError(err)

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	common.PanicAfter(1*time.Minute, completeQueuedTransaction, s.T().Name())

	// replace transaction notification handler
	var txHash gethcommon.Hash
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		require.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == node.EventTransactionQueued {
			// Use s.* for assertions - require leaves the channel unclosed.

			event := envelope.Event.(map[string]interface{})
			s.T().Logf("transaction queued and will be completed shortly, id: %v", event["id"])

			s.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

			var err error
			txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			if s.NoError(err, event["id"]) {
				s.T().Logf("contract transaction complete, URL: %s", "https://ropsten.etherscan.io/tx/"+txHash.Hex())
			}

			close(completeQueuedTransaction)
		}
	})

	_, err = cell.Run(`
		var responseValue = null;
		var testContract = web3.eth.contract([{"constant":true,"inputs":[{"name":"a","type":"int256"}],"name":"double","outputs":[{"name":"","type":"int256"}],"payable":false,"type":"function"}]);
		var test = testContract.new(
		{
			from: '` + TestConfig.Account1.Address + `',
			data: '0x6060604052341561000c57fe5b5b60a58061001b6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa14603a575bfe5b3415604157fe5b60556004808035906020019091905050606b565b6040518082815260200191505060405180910390f35b60008160020290505b9190505600a165627a7a72305820ccdadd737e4ac7039963b54cee5e5afb25fa859a275252bdcf06f653155228210029',
			gas: '` + strconv.Itoa(params.DefaultGas) + `'
		}, function (e, contract){
			if (!e) {
				responseValue = contract.transactionHash
			}
		})
	`)
	require.NoError(err)

	<-completeQueuedTransaction

	responseValue, err := cell.Get("responseValue")
	require.NoError(err)

	response, err := responseValue.ToString()
	require.NoError(err)

	expectedResponse := txHash.Hex()
	require.Equal(expectedResponse, response)
	s.T().Logf("estimation complete: %s", response)
}

func (s *BackendTestSuite) TestJailWhisper() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	jailInstance := s.backend.JailManager()
	require.NotNil(jailInstance)

	whisperService := s.WhisperService()
	whisperAPI := whisper.NewPublicWhisperAPI(whisperService)

	accountManager := s.backend.AccountManager()

	// account1
	_, accountKey1, err := accountManager.AddressToDecryptedAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	require.NoError(err)
	accountKey1Hex := gethcommon.ToHex(crypto.FromECDSAPub(&accountKey1.PrivateKey.PublicKey))

	_, err = whisperService.AddKeyPair(accountKey1.PrivateKey)
	require.NoError(err, fmt.Sprintf("identity not injected: %v", accountKey1Hex))

	if ok := whisperAPI.HasKeyPair(context.Background(), accountKey1Hex); !ok {
		require.FailNow(fmt.Sprintf("identity not injected: %v", accountKey1Hex))
	}

	// account2
	_, accountKey2, err := accountManager.AddressToDecryptedAccount(TestConfig.Account2.Address, TestConfig.Account2.Password)
	require.NoError(err)
	accountKey2Hex := gethcommon.ToHex(crypto.FromECDSAPub(&accountKey2.PrivateKey.PublicKey))

	_, err = whisperService.AddKeyPair(accountKey2.PrivateKey)
	require.NoError(err, fmt.Sprintf("identity not injected: %v", accountKey2Hex))

	if ok := whisperAPI.HasKeyPair(context.Background(), accountKey2Hex); !ok {
		require.FailNow(fmt.Sprintf("identity not injected: %v", accountKey2Hex))
	}

	passedTests := map[string]bool{
		whisperMessage1: false,
		whisperMessage2: false,
		whisperMessage3: false,
		whisperMessage4: false,
		whisperMessage5: false,
		whisperMessage6: false,
	}
	installedFilters := map[string]string{
		whisperMessage1: "",
		whisperMessage2: "",
		whisperMessage3: "",
		whisperMessage4: "",
		whisperMessage5: "",
		whisperMessage6: "",
	}

	testCases := []struct {
		name      string
		testCode  string
		useFilter bool
	}{
		{
			"test 0: ensure correct version of Whisper is used",
			`
				var expectedVersion = '0x5';
				if (web3.version.whisper != expectedVersion) {
					throw 'unexpected shh version, expected: ' + expectedVersion + ', got: ' + web3.version.whisper;
				}
			`,
			false,
		},
		{
			"test 1: encrypted signed message from us (From != nil && To != nil)",
			`
				var identity1 = '` + accountKey1Hex + `';
				if (!web3.shh.hasKeyPair(identity1)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}

				var identity2 = '` + accountKey2Hex + `';
				if (!web3.shh.hasKeyPair(identity2)) {
					throw 'idenitity "` + accountKey2Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage1 + `';

				// start watching for messages
				var filter = shh.filter({
					type: "asym",
					sig: identity1,
					key: identity2,
					topics: [topic]
				});
				console.log(JSON.stringify(filter));

				// post message
				var message = {
					type: "asym",
					sig: identity1,
					key: identity2,
					topic: topic,
					payload: payload,
					ttl: 20,
				};
				var err = shh.post(message)
				if (err !== null) {
					throw 'message not sent: ' + message;
				}

				var filterName = '` + whisperMessage1 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
				}
			`,
			true,
		},
		{
			"test 2: encrypted signed message to yourself (From != nil && To != nil)",
			`
				var identity = '` + accountKey1Hex + `';
				if (!web3.shh.hasKeyPair(identity)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage2 + `';

				// start watching for messages
				var filter = shh.filter({
					type: "asym",
					sig: identity,
					key: identity,
					topics: [topic],
				});

				// post message
				var message = {
					type: "asym",
				  	sig: identity,
				  	key: identity,
				  	topic: topic,
				  	payload: payload,
				  	ttl: 20,
				};
				var err = shh.post(message)
				if (err !== null) {
					throw 'message not sent: ' + message;
				}

				var filterName = '` + whisperMessage2 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
				}
			`,
			true,
		},
		{
			"test 3: signed (known sender) broadcast (From != nil && To == nil)",
			`
				var identity = '` + accountKey1Hex + `';
				if (!web3.shh.hasKeyPair(identity)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage3 + `';

				// generate symmetric key
				var keyid = shh.generateSymmetricKey();
				if (!shh.hasSymmetricKey(keyid)) {
					throw new Error('key not found');
				}

				// start watching for messages
				var filter = shh.filter({
					type: "sym",
					sig: identity,
					topics: [topic],
					key: keyid
				});

				// post message
				var message = {
					type: "sym",
					sig: identity,
					topic: topic,
					payload: payload,
					ttl: 20,
					key: keyid
				};
				var err = shh.post(message)
				if (err !== null) {
					throw 'message not sent: ' + message;
				}

				var filterName = '` + whisperMessage3 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
				}
			`,
			true,
		},
		{
			"test 4: anonymous broadcast (From == nil && To == nil)",
			`
				var topic = makeTopic();
				var payload = '` + whisperMessage4 + `';

				// generate symmetric key
				var keyid = shh.generateSymmetricKey();
				if (!shh.hasSymmetricKey(keyid)) {
					throw new Error('key not found');
				}

				// start watching for messages
				var filter = shh.filter({
					type: "sym",
					topics: [topic],
					key: keyid
				});

				// post message
				var message = {
					type: "sym",
					topic: topic,
					payload: payload,
					ttl: 20,
					key: keyid
				};
				var err = shh.post(message)
				if (err !== null) {
					throw 'message not sent: ' + err;
				}

				var filterName = '` + whisperMessage4 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
				}
			`,
			true,
		},
		{
			"test 5: encrypted anonymous message (From == nil && To != nil)",
			`
				var identity = '` + accountKey2Hex + `';
				if (!web3.shh.hasKeyPair(identity)) {
					throw 'idenitity "` + accountKey2Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage5 + `';

				// start watching for messages
				var filter = shh.filter({
					type: "asym",
					key: identity,
					topics: [topic],
				});

				// post message
				var message = {
					type: "asym",
					key: identity,
					topic: topic,
					payload: payload,
					ttl: 20
				};
				var err = shh.post(message)
				if (err !== null) {
					throw 'message not sent: ' + message;
				}

				var filterName = '` + whisperMessage5 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
				}
			`,
			true,
		},
		{
			"test 6: encrypted signed response to us (From != nil && To != nil)",
			`
				var identity1 = '` + accountKey1Hex + `';
				if (!web3.shh.hasKeyPair(identity1)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}

				var identity2 = '` + accountKey2Hex + `';
				if (!web3.shh.hasKeyPair(identity2)) {
					throw 'idenitity "` + accountKey2Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage6 + `';

				// start watching for messages
				var filter = shh.filter({
					type: "asym",
					sig: identity2,
					key: identity1,
					topics: [topic]
				});

				// post message
				var message = {
					type: "asym",
				  	sig: identity2,
				  	key: identity1,
				  	topic: topic,
				  	payload: payload,
				  	ttl: 20
				};
				var err = shh.post(message)
				if (err !== null) {
					throw 'message not sent: ' + message;
				}

				var filterName = '` + whisperMessage6 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
				}
			`,
			true,
		},
	}

	for _, testCase := range testCases {
		s.T().Log(testCase.name)
		testCaseKey := crypto.Keccak256Hash([]byte(testCase.name)).Hex()
		jailInstance.Parse(testCaseKey, `
			var shh = web3.shh;
			var makeTopic = function () {
				var min = 1;
				var max = Math.pow(16, 8);
				var randInt = Math.floor(Math.random() * (max - min + 1)) + min;
				return web3.toHex(randInt);
			};
		`)

		cell, err := jailInstance.GetJailCell(testCaseKey)
		require.NoError(err, "cannot get VM")

		// post messages
		if _, err := cell.Run(testCase.testCode); err != nil {
			require.Fail(err.Error())
			return
		}

		if !testCase.useFilter {
			continue
		}

		// update installed filters
		filterId, err := cell.Get("filterId")
		require.NoError(err, "cannot get filterId")

		filterName, err := cell.Get("filterName")
		require.NoError(err, "cannot get filterName")

		if _, ok := installedFilters[filterName.String()]; !ok {
			require.FailNow("unrecognized filter")
		}

		installedFilters[filterName.String()] = filterId.String()
	}

	time.Sleep(2 * time.Second) // allow whisper to poll

	for testKey, filter := range installedFilters {
		if filter != "" {
			s.T().Logf("filter found: %v", filter)
			messages, err := whisperAPI.GetFilterMessages(filter)
			require.NoError(err)
			for _, message := range messages {
				s.T().Logf("message found: %s", string(message.Payload))
				passedTests[testKey] = true
			}
		}
	}

	for testName, passedTest := range passedTests {
		if !passedTest {
			s.Fail(fmt.Sprintf("test not passed: %v", testName))
		}
	}
}

func (s *BackendTestSuite) TestJailVMPersistence() {
	require := s.Require()

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	// log into account from which transactions will be sent
	err := s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	require.NoError(err, "cannot select account: %v", TestConfig.Account1.Address)

	type testCase struct {
		command   string
		params    string
		validator func(response string) error
	}
	var testCases = []testCase{
		{
			`["sendTestTx"]`,
			`{"amount": "0.000001", "from": "` + TestConfig.Account1.Address + `"}`,
			func(response string) error {
				if strings.Contains(response, "error") {
					return fmt.Errorf("unexpected response: %v", response)
				}
				return nil
			},
		},
		{
			`["sendTestTx"]`,
			`{"amount": "0.000002", "from": "` + TestConfig.Account1.Address + `"}`,
			func(response string) error {
				if strings.Contains(response, "error") {
					return fmt.Errorf("unexpected response: %v", response)
				}
				return nil
			},
		},
		{
			`["ping"]`,
			`{"pong": "Ping1", "amount": 0.42}`,
			func(response string) error {
				expectedResponse := `{"result": "Ping1"}`
				if response != expectedResponse {
					return fmt.Errorf("unexpected response, expected: %v, got: %v", expectedResponse, response)
				}
				return nil
			},
		},
		{
			`["ping"]`,
			`{"pong": "Ping2", "amount": 0.42}`,
			func(response string) error {
				expectedResponse := `{"result": "Ping2"}`
				if response != expectedResponse {
					return fmt.Errorf("unexpected response, expected: %v, got: %v", expectedResponse, response)
				}
				return nil
			},
		},
	}

	jailInstance := s.backend.JailManager()
	jailInstance.BaseJS(string(static.MustAsset("testdata/jail/status.js")))
	parseResult := jailInstance.Parse(testChatID, `
		var total = 0;
		_status_catalog['ping'] = function(params) {
			total += Number(params.amount);
			return params.pong;
		}

		_status_catalog['sendTestTx'] = function(params) {
		  var amount = params.amount;
		  var transaction = {
			"from": params.from,
			"to": "`+TestConfig.Account2.Address+`",
			"value": web3.toWei(amount, "ether")
		  };
		  web3.eth.sendTransaction(transaction, function (error, result) {
			 if(!error) {
				total += Number(amount);
			 }
		  });
		}
	`)
	require.NotContains(parseResult, "error", "further will fail if initial parsing failed")

	var wg sync.WaitGroup
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope node.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			s.T().Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == node.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			s.T().Logf("Transaction queued (will be completed shortly): {id: %s}\n", event["id"].(string))

			time.Sleep(1 * time.Second)

			//if err := geth.DiscardTransaction(event["id"].(string)); err != nil {
			//	t.Errorf("cannot discard: %v", err)
			//	progress <- "tx discarded"
			//	return
			//}

			//var txHash common.Hash
			txHash, err := s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			require.NoError(err, "cannot complete queued transaction[%v]: %v", event["id"], err)

			s.T().Logf("Transaction complete: https://ropsten.etherscan.io/tx/%s", txHash.Hex())
		}
	})

	// run commands concurrently
	for _, tc := range testCases {
		wg.Add(1)
		go func(tc testCase) {
			defer wg.Done() // ensure we don't forget it

			s.T().Logf("CALL START: %v %v", tc.command, tc.params)
			response := jailInstance.Call(testChatID, tc.command, tc.params)
			if err := tc.validator(response); err != nil {
				s.T().Errorf("failed test validation: %v, err: %v", tc.command, err)
			}
			s.T().Logf("CALL END: %v %v", tc.command, tc.params)
		}(tc)
	}

	common.PanicAfter(60*time.Second, nil, "test timed out")
	wg.Wait()

	// Validate total.
	cell, err := jailInstance.GetJailCell(testChatID)
	require.NoError(err)

	totalOtto, err := cell.Get("total")
	require.NoError(err)

	total, err := totalOtto.ToFloat()
	require.NoError(err)

	s.T().Log(total)
	require.InDelta(0.840003, total, 0.0000001)
}
