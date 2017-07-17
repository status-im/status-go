package jail_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/proxy"
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

	s.jail = jail.New(proxy.NewRPCRouter(s.NodeManager))

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

// TestSendTransactionWithJail attemps to validate usage of upstream for transaction
// processing.
// TODO(influx6): Ensure upstream gets Account nodes so this tests passes.
func (s *JailTestSuite) TestSendTransactionWithJail() {
	require := s.Require()
	require.NotNil(s.jail)

	s.StartTestNode(params.RopstenNetworkID, true)
	defer s.StopTestNode()

	// load Status JS and add test command to it
	s.jail.BaseJS(baseStatusJSCode)
	s.jail.Parse(testChatID, ``)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := s.jail.JailCellVM(testChatID)
	require.NoError(err)
	require.NotNil(vm)

	_, err = vm.Run(`
	    var accountId = "` + TestConfig.Account1.Address + `";
	    var transactionCount = web3.eth.getTransactionCount(accountId, "latest");
	`)

	require.NoError(err)

	transactionCount, err := vm.Get("transanctionCount")
	require.NoError(err, "cannot obtain new transaction nounce")

	s.T().Logf("Received Transaction Nounce: %+q", transactionCount)

	_, err = vm.Run(`
	    var fromAccount = "` + TestConfig.Account1.Address + `";
	    var toAccount = "` + TestConfig.Account2.Address + `";
	    var nonce = "` + transactionCount.String() + `";

			var signedData = web3.eth.sign(fromAccount, "{message:'levitate in the void'}")

	    var sendResponse = web3.eth.sendTransaction({
				to: toAccount,
				from: fromAccount,
				gas: "0x76c0",
				nonce: nounce,
				data: signedData,
				gasPrice: "0x9184e72a000",
				value: web3.toWei(0.05, "ether"),
			});
	`)

	require.NoError(err)

	sendResponse, err := vm.Get("sendResponse")
	require.NoError(err, "cannot obtain response for sendtransaction")

	s.T().Logf("Received Transaction response: %+q", sendResponse)
}

func (s *JailTestSuite) TestJailRPCSend() {
	require := s.Require()
	require.NotNil(s.jail)

	s.StartTestNode(params.RopstenNetworkID, false)
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

	s.StartTestNode(params.RopstenNetworkID, false)
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
