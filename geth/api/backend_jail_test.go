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
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/status-im/status-go/geth/txqueue"
	"github.com/status-im/status-go/static"
)

const (
	whisperMessage1 = `test message 1 (K1 -> K2, signed+encrypted, from us)`
	whisperMessage2 = `test message 3 (K1 -> "", signed broadcast)`
	whisperMessage3 = `test message 4 ("" -> "", anon broadcast)`
	whisperMessage4 = `test message 5 ("" -> K1, encrypted anon broadcast)`
	whisperMessage5 = `test message 6 (K2 -> K1, signed+encrypted, to us)`
	txSendFolder    = "testdata/jail/tx-send/"
	testChatID      = "testChat"
)

var (
	baseStatusJSCode = string(static.MustAsset("testdata/jail/status.js"))
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

	txHashes := make(chan gethcommon.Hash, 1)

	// replace transaction notification handler
	requireMessageId := false
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			messageId, ok := event["message_id"].(string)
			s.True(ok, "Message id is required, but not found")
			if requireMessageId {
				require.NotEmpty(messageId, "Message id is required, but not provided")
			} else {
				require.Empty(messageId, "Message id is not required, but provided")
			}

			txID := event["id"].(string)
			txHash, err := s.backend.CompleteTransaction(common.QueuedTxID(txID), TestConfig.Account1.Password)
			require.NoError(err, "cannot complete queued transaction[%v]", event["id"])

			log.Info("Transaction complete", "URL", "https://ropsten.etherscan.io/tx/%s"+txHash.Hex())

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
		jailInstance.Parse(testChatID, ``)

		// used by notification handler
		requireMessageId = test.requireMessageId

		for _, command := range test.commands {
			s.T().Logf("%s: %s", test.name, command.command)
			response := jailInstance.Call(testChatID, command.command, command.params)

			var txHash gethcommon.Hash
			if command.command == `["commands", "send"]` {
				select {
				case txHash = <-txHashes:
				case <-time.After(time.Minute):
					s.FailNow("test timed out: %s", test.name)
				}
			}

			expectedResponse := strings.Replace(command.expectedResponse, "TX_HASH", txHash.Hex(), 1)
			require.Equal(expectedResponse, response)
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

	cell, err := jailInstance.Cell(testChatID)
	require.NoError(err)

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	var txHash gethcommon.Hash
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		var err error
		err = json.Unmarshal([]byte(jsonEvent), &envelope)
		require.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			// Use s.* for assertions - require leaves the channel unclosed.

			event := envelope.Event.(map[string]interface{})
			s.T().Logf("transaction queued and will be completed shortly, id: %v", event["id"])

			s.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

			txID := event["id"].(string)
			txHash, err = s.backend.CompleteTransaction(common.QueuedTxID(txID), TestConfig.Account1.Password)
			if s.NoError(err, event["id"]) {
				s.T().Logf("contract transaction complete, URL: %s", "https://ropsten.etherscan.io/tx/"+txHash.Hex())
			}

			close(completeQueuedTransaction)
		}
	})

	_, err = cell.Run(`
		var responseValue = null;
		var errorValue = null;
		var testContract = web3.eth.contract([{"constant":true,"inputs":[{"name":"a","type":"int256"}],"name":"double","outputs":[{"name":"","type":"int256"}],"payable":false,"type":"function"}]);
		var test = testContract.new(
		{
			from: '` + TestConfig.Account1.Address + `',
			data: '0x6060604052341561000c57fe5b5b60a58061001b6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa14603a575bfe5b3415604157fe5b60556004808035906020019091905050606b565b6040518082815260200191505060405180910390f35b60008160020290505b9190505600a165627a7a72305820ccdadd737e4ac7039963b54cee5e5afb25fa859a275252bdcf06f653155228210029',
			gas: '` + strconv.Itoa(params.DefaultGas) + `'
		}, function (e, contract) {
			// NOTE: The callback will fire twice!
			errorValue = e;
			// Once the contract has the transactionHash property set and once its deployed on an address.
			if (!contract.address) {
				responseValue = contract.transactionHash;
			}
		})
	`)
	require.NoError(err)

	select {
	case <-completeQueuedTransaction:
	case <-time.After(time.Minute):
		s.FailNow("test timed out")
	}

	// Wait until callback is fired and `responseValue` is set. Hacky but simple.
	time.Sleep(2 * time.Second)

	errorValue, err := cell.Get("errorValue")
	require.NoError(err)
	require.Equal("null", errorValue.String())

	responseValue, err := cell.Get("responseValue")
	require.NoError(err)

	response, err := responseValue.ToString()
	require.NoError(err)

	expectedResponse := txHash.Hex()
	require.Equal(expectedResponse, response)
}

func (s *BackendTestSuite) TestJailWhisper() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	jailInstance := s.backend.JailManager()
	require.NotNil(jailInstance)

	jailInstance.BaseJS(baseStatusJSCode)

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
	}
	installedFilters := map[string]string{
		whisperMessage1: "",
		whisperMessage2: "",
		whisperMessage3: "",
		whisperMessage4: "",
		whisperMessage5: "",
	}

	testCases := []struct {
		name      string
		testCode  string
		useFilter bool
	}{
		{
			"test 0: ensure correct version of Whisper is used",
			`
				var expectedVersion = '5.0';
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
				if (!shh.hasKeyPair(identity1)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}

				var identity2 = '` + accountKey2Hex + `';
				if (!shh.hasKeyPair(identity2)) {
					throw 'identitity "` + accountKey2Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage1 + `';

				// start watching for messages
				var filter = shh.newMessageFilter({
					sig: identity1,
					privateKeyID: identity2,
					topics: [topic]
				});

				// post message
				var message = {
					ttl: 20,
					powTarget: 0.01,
					powTime: 20,
					topic: topic,
					sig: identity1,
					pubKey: identity2,
			  		payload: web3.toHex(payload),
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + JSON.stringify(message);
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
			"test 2: signed (known sender) broadcast (From != nil && To == nil)",
			`
				var identity = '` + accountKey1Hex + `';
				if (!shh.hasKeyPair(identity)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage2 + `';

				// generate symmetric key
				var keyid = shh.newSymKey();
				if (!shh.hasSymKey(keyid)) {
					throw new Error('key not found');
				}

				// start watching for messages
				var filter = shh.newMessageFilter({
					sig: identity,
					topics: [topic],
					symKeyID: keyid
				});

				// post message
				var message = {
					ttl: 20,
					powTarget: 0.01,
					powTime: 20,
					topic: topic,
					sig: identity,
					symKeyID: keyid,
			  		payload: web3.toHex(payload),
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + JSON.stringify(message);
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
			"test 3: anonymous broadcast (From == nil && To == nil)",
			`
				var topic = makeTopic();
				var payload = '` + whisperMessage3 + `';

				// generate symmetric key
				var keyid = shh.newSymKey();
				if (!shh.hasSymKey(keyid)) {
					throw new Error('key not found');
				}

				// start watching for messages
				var filter = shh.newMessageFilter({
					topics: [topic],
					symKeyID: keyid
				});

				// post message
				var message = {
					ttl: 20,
					powTarget: 0.01,
					powTime: 20,
					topic: topic,
					symKeyID: keyid,
			  		payload: web3.toHex(payload),
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + JSON.stringify(message);
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
			"test 4: encrypted anonymous message (From == nil && To != nil)",
			`
				var identity = '` + accountKey2Hex + `';
				if (!shh.hasKeyPair(identity)) {
					throw 'idenitity "` + accountKey2Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage4 + `';

				// start watching for messages
				var filter = shh.newMessageFilter({
					privateKeyID: identity,
					topics: [topic],
				});

				// post message
				var message = {
					ttl: 20,
					powTarget: 0.01,
					powTime: 20,
					topic: topic,
					pubKey: identity,
			  		payload: web3.toHex(payload),
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + JSON.stringify(message);
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
			"test 5: encrypted signed response to us (From != nil && To != nil)",
			`
				var identity1 = '` + accountKey1Hex + `';
				if (!shh.hasKeyPair(identity1)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}
				var identity2 = '` + accountKey2Hex + `';
				if (!shh.hasKeyPair(identity2)) {
					throw 'idenitity "` + accountKey2Hex + `" not found in whisper';
				}
				var topic = makeTopic();
				var payload = '` + whisperMessage5 + `';
				// start watching for messages
				var filter = shh.newMessageFilter({
					privateKeyID: identity1,
					sig: identity2,
					topics: [topic],
				});

				// post message
				var message = {
				  	sig: identity2,
				  	pubKey: identity1,
				  	topic: topic,
				  	payload: web3.toHex(payload),
					ttl: 20,
					powTime: 20,
					powTarget: 0.01,
				};
				var sent = shh.post(message)
				if (!sent) {
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
	}

	for _, testCase := range testCases {
		s.T().Log(testCase.name)
		testCaseKey := crypto.Keccak256Hash([]byte(testCase.name)).Hex()

		jailInstance.Parse(testCaseKey, `
			var shh = web3.shh;
			// topic must be 4-byte long
			var makeTopic = function () {
				var topic = '0x';
				for (var i = 0; i < 8; i++) {
					topic += Math.floor(Math.random() * 16).toString(16);
				}
				return topic;
			};
		`)

		cell, err := jailInstance.Cell(testCaseKey)
		require.NoError(err, "cannot get VM")

		// post messages
		_, err = cell.Run(testCase.testCode)
		require.NoError(err)

		if !testCase.useFilter {
			continue
		}

		// update installed filters
		filterId, err := cell.Get("filterId")
		require.NoError(err, "cannot get filterId")

		filterName, err := cell.Get("filterName")
		require.NoError(err, "cannot get filterName")

		_, ok := installedFilters[filterName.String()]
		require.True(ok, "unrecognized filter")

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
		s.True(passedTest, "test not passed: %v", testName)
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
	jailInstance.BaseJS(baseStatusJSCode)

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
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			s.T().Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			s.T().Logf("Transaction queued (will be completed shortly): {id: %s}\n", event["id"].(string))

			//var txHash common.Hash
			txID := event["id"].(string)
			txHash, err := s.backend.CompleteTransaction(common.QueuedTxID(txID), TestConfig.Account1.Password)
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

	finishTestCases := make(chan struct{})
	go func() {
		wg.Wait()
		close(finishTestCases)
	}()

	select {
	case <-finishTestCases:
	case <-time.After(time.Minute):
		s.FailNow("some tests failed to finish in time")
	}

	// Wait till eth_sendTransaction callbacks have been executed.
	// FIXME(tiabc): more reliable means of testing that.
	time.Sleep(5 * time.Second)

	// Validate total.
	cell, err := jailInstance.Cell(testChatID)
	require.NoError(err)

	totalOtto, err := cell.Get("total")
	require.NoError(err)

	total, err := totalOtto.ToFloat()
	require.NoError(err)

	s.T().Log(total)
	require.InDelta(0.840003, total, 0.0000001)
}
