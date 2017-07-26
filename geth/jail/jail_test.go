package jail_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/status-im/status-go/static"
	"github.com/stretchr/testify/suite"
)

const (
	testChatID = "testChat"
)

var baseStatusJSCode = string(static.MustAsset("testdata/jail/status.js"))

func TestJailTestSuite(t *testing.T) {
	suite.Run(t, new(JailTestSuite))
}

type JailTestSuite struct {
	BaseTestSuite
	jail *jail.Jail
}

func (s *JailTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.Require().NotNil(s.NodeManager)
	s.Require().IsType(&node.NodeManager{}, s.NodeManager)
	s.jail = jail.New(s.NodeManager)
	s.Require().NotNil(s.jail)
	s.Require().IsType(&jail.Jail{}, s.jail)
}

func (s *JailTestSuite) TestInit() {
	require := s.Require()
	require.NotNil(s.jail)

	errorWrapper := func(err error) string {
		return `{"error":"` + err.Error() + `"}`
	}

	// get cell VM w/o defining cell first
	vm, err := s.jail.JailCellVM(testChatID)
	require.EqualError(err, "cell[testChat] doesn't exist")
	require.Nil(vm)

	// create VM (w/o properly initializing base JS script)
	err = errors.New("ReferenceError: '_status_catalog' is not defined")
	require.Equal(errorWrapper(err), s.jail.Parse(testChatID, ``))
	err = errors.New("ReferenceError: 'call' is not defined")
	require.Equal(errorWrapper(err), s.jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`))

	// get existing cell (even though we got errors, cell was still created)
	vm, err = s.jail.JailCellVM(testChatID)
	require.NoError(err)
	require.NotNil(vm)

	statusJS := baseStatusJSCode + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	s.jail.BaseJS(statusJS)

	// now no error should occur
	response := s.jail.Parse(testChatID, ``)
	expectedResponse := `{"result": {"commands":{},"responses":{}}}`
	require.Equal(expectedResponse, response)

	// make sure that Call succeeds even w/o running node
	response = s.jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse = `{"result": 144}`
	require.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestParse() {
	require := s.Require()
	require.NotNil(s.jail)

	extraCode := `
	var _status_catalog = {
		foo: 'bar'
	};
	`
	response := s.jail.Parse("newChat", extraCode)
	expectedResponse := `{"result": {"foo":"bar"}}`
	require.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestFunctionCall() {
	require := s.Require()
	require.NotNil(s.jail)

	// load Status JS and add test command to it
	statusJS := baseStatusJSCode + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	s.jail.Parse(testChatID, statusJS)

	// call with wrong chat id
	response := s.jail.Call("chatIDNonExistent", "", "")
	expectedError := `{"error":"Cell[chatIDNonExistent] doesn't exist."}`
	require.Equal(expectedError, response)

	// call extraFunc()
	response = s.jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse := `{"result": 144}`
	require.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestJailTimeoutFailure() {
	require := s.Require()
	require.NotNil(s.jail)

	newCell := s.jail.NewJailCell(testChatID)
	require.NotNil(newCell)

	execr := newCell.Executor()

	// Attempt to run a timeout string against a JailCell.
	_, err := execr.Exec(`
		setTimeout(function(n){
			if(Date.now() - n < 50){
				throw new Error("Timedout early");
			}

			return n;
		}, 30, Date.now());
	`)

	require.NotNil(err)
}

func (s *JailTestSuite) TestJailTimeout() {
	require := s.Require()
	require.NotNil(s.jail)

	newCell := s.jail.NewJailCell(testChatID)
	require.NotNil(newCell)

	execr := newCell.Executor()

	// Attempt to run a timeout string against a JailCell.
	res, err := execr.Exec(`
		setTimeout(function(n){
			if(Date.now() - n < 50){
				throw new Error("Timedout early");
			}

			return n;
		}, 50, Date.now());
	`)

	require.NoError(err)
	require.NotNil(res)
}

func (s *JailTestSuite) TestJailFetch() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello World"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	require := s.Require()
	require.NotNil(s.jail)

	newCell := s.jail.NewJailCell(testChatID)
	require.NotNil(newCell)

	execr := newCell.Executor()

	wait := make(chan struct{})

	// Attempt to run a fetch resource.
	_, err := execr.Fetch(server.URL, func(res otto.Value) {
		go func() { wait <- struct{}{} }()
	})

	require.NoError(err)

	<-wait
}

func (s *JailTestSuite) TestJailRPCSend() {
	require := s.Require()
	require.NotNil(s.jail)

	s.StartTestNode(params.RopstenNetworkID)
	defer s.StopTestNode()

	// load Status JS and add test command to it
	s.jail.BaseJS(baseStatusJSCode)
	s.jail.Parse(testChatID, ``)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := s.jail.JailCellVM(testChatID)
	require.NoError(err)
	require.NotNil(vm)

	// internally (since we replaced `web3.send` with `jail.Send`)
	// all requests to web3 are forwarded to `jail.Send`
	_, err = vm.Run(`
	    var balance = web3.eth.getBalance("` + TestConfig.Account1.Address + `");
		var sendResult = web3.fromWei(balance, "ether")
	`)
	require.NoError(err)

	value, err := vm.Get("sendResult")
	require.NoError(err, "cannot obtain result of balance check operation")

	balance, err := value.ToFloat()
	require.NoError(err)

	s.T().Logf("Balance of %.2f ETH found on '%s' account", balance, TestConfig.Account1.Address)
	require.False(balance < 100, "wrong balance (there should be lots of test Ether on that account)")
}

func (s *JailTestSuite) TestGetJailCellVM() {
	expectedError := `cell[nonExistentChat] doesn't exist`
	_, err := s.jail.JailCellVM("nonExistentChat")
	s.EqualError(err, expectedError)

	// now let's create VM..
	s.jail.Parse(testChatID, ``)

	// ..and see if VM becomes available
	_, err = s.jail.JailCellVM(testChatID)
	s.NoError(err)
}

func (s *JailTestSuite) TestIsConnected() {
	require := s.Require()
	require.NotNil(s.jail)

	// TODO(tiabc): Is this required?
	s.StartTestNode(params.RopstenNetworkID)
	defer s.StopTestNode()

	s.jail.Parse(testChatID, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := s.jail.JailCellVM(testChatID)
	require.NoError(err)

	_, err = vm.Run(`
	    var responseValue = web3.isConnected();
	    responseValue = JSON.stringify(responseValue);
	`)
	require.NoError(err)

	responseValue, err := vm.Get("responseValue")
	require.NoError(err, "cannot obtain result of isConnected()")

	response, err := responseValue.ToString()
	require.NoError(err, "cannot parse result")

	expectedResponse := `{"jsonrpc":"2.0","result":true}`
	require.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestLocalStorageSet() {
	require := s.Require()
	require.NotNil(s.jail)

	s.jail.Parse(testChatID, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := s.jail.JailCellVM(testChatID)
	require.NoError(err)

	testData := "foobar"

	opCompletedSuccessfully := make(chan struct{}, 1)

	// replace transaction notification handler
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		require.NoError(err)

		if envelope.Type == jail.EventLocalStorageSet {
			event := envelope.Event.(map[string]interface{})
			chatID, ok := event["chat_id"].(string)
			require.True(ok, "chat id is required, but not found")
			require.Equal(testChatID, chatID, "incorrect chat ID")

			actualData, ok := event["data"].(string)
			require.True(ok, "data field is required, but not found")
			require.Equal(testData, actualData, "incorrect data")

			s.T().Logf("event processed: %s", jsonEvent)
			close(opCompletedSuccessfully)
		}
	})

	_, err = vm.Run(`
	    var responseValue = localStorage.set("` + testData + `");
	    responseValue = JSON.stringify(responseValue);
	`)
	s.NoError(err)

	// make sure that signal is sent (and its parameters are correct)
	select {
	case <-opCompletedSuccessfully:
		// pass
	case <-time.After(3 * time.Second):
		s.Fail("operation timed out")
	}

	responseValue, err := vm.Get("responseValue")
	s.NoError(err, "cannot obtain result of localStorage.set()")

	response, err := responseValue.ToString()
	s.NoError(err, "cannot parse result")

	expectedResponse := `{"jsonrpc":"2.0","result":true}`
	s.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestJailVMPersistence() {
	require := s.Require()

	s.StartTestNode(params.RopstenNetworkID)
	defer s.StopTestNode()

	accountManager := node.NewAccountManager(s.NodeManager)
	txQueueManager := node.NewTxQueueManager(s.NodeManager, accountManager)

	// log into account from which transactions will be sent
	err := accountManager.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
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
				return nil
			},
		},
		{
			`["sendTestTx"]`,
			`{"amount": "0.000002", "from": "` + TestConfig.Account1.Address + `"}`,
			func(response string) error {
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

	jailInstance := jail.New(s.NodeManager)
	vmID := "persistentVM" // we will send concurrent request to the very same VM
	jailInstance.Parse(vmID, `
		var total = 0;
		_status_catalog['ping'] = function(params) {
			total += params.amount;
			return params.pong;
		}

		_status_catalog['sendTestTx'] = function(params) {
		  var amount = params.amount;
		  var transaction = {
			"from": params.from,
			"to": "0xf82da7547534045b4e00442bc89e16186cf8c272",
			"value": web3.toWei(amount, "ether")
		  };
		  web3.eth.sendTransaction(transaction, function (error, result) {
		  	 console.log("eth.sendTransaction callback: 'total' variable (is it updated by concurrent routine): " + total);
			 if(!error)
				total += amount;
		  });
		}
	`)

	progress := make(chan string, len(testCases)+1)
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
			txHash, err := txQueueManager.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			require.NoError(err, "cannot complete queued transaction[%v]: %v", event["id"], err)

			s.T().Logf("Transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())

			progress <- "event queue notification processed"
		}
	})

	// run commands concurrently
	for _, tc := range testCases {
		go func(tc testCase) {
			s.T().Logf("CALL START: %v %v", tc.command, tc.params)
			response := jailInstance.Call(vmID, tc.command, tc.params)
			if err := tc.validator(response); err != nil {
				s.T().Errorf("failed test validation: %v, err: %v", tc.command, err)
			}
			s.T().Logf("CALL END: %v %v", tc.command, tc.params)
			progress <- tc.command
		}(tc)
	}

	// wait for all tests to finish
	cnt := len(testCases) + 1
	time.AfterFunc(5*time.Second, func() {
		// to long, allow main thread to return
		cnt = 0
	})

Loop:
	for {
		select {
		case <-progress:
			cnt--
			if cnt <= 0 {
				break Loop
			}
		case <-time.After(10 * time.Second): // timeout
			s.T().Error("test timed out")
			break Loop
		}
	}

	time.Sleep(2 * time.Second) // allow to propagate
}
