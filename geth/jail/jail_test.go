package jail_test

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/status-im/status-go/static"
	"github.com/stretchr/testify/suite"
)

const (
	testChatID = "testChat"
)

var (
	baseStatusJSCode = string(static.MustAsset("testdata/jail/status.js"))
	txJSCode         = string(static.MustAsset("testdata/jail/tx-send/tx-send.js"))
)

func TestJailTestSuite(t *testing.T) {
	suite.Run(t, new(JailTestSuite))
}

type JailTestSuite struct {
	BaseTestSuite
	jail *jail.Jail
}

func (s *JailTestSuite) SetupTest() {
	require := s.Require()

	nodeManager := node.NewNodeManager()
	require.NotNil(nodeManager)

	accountManager := account.NewManager(nodeManager)
	require.NotNil(accountManager)

	jail := jail.New(nodeManager)
	require.NotNil(jail)

	s.jail = jail
	s.NodeManager = nodeManager
}

func (s *JailTestSuite) TestInit() {
	require := s.Require()

	errorWrapper := func(err error) string {
		return `{"error":"` + err.Error() + `"}`
	}

	// get cell VM w/o defining cell first
	cell, err := s.jail.Cell(testChatID)
	require.EqualError(err, "cell[testChat] doesn't exist")
	require.Nil(cell)

	// create VM (w/o properly initializing base JS script)
	err = errors.New("ReferenceError: '_status_catalog' is not defined")
	require.Equal(errorWrapper(err), s.jail.Parse(testChatID, ``))
	err = errors.New("ReferenceError: 'call' is not defined")
	require.Equal(errorWrapper(err), s.jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`))

	// get existing cell (even though we got errors, cell was still created)
	cell, err = s.jail.Cell(testChatID)
	require.NoError(err)
	require.NotNil(cell)

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

	// load Status JS and add test command to it
	statusJS := baseStatusJSCode + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	s.jail.Parse(testChatID, statusJS)

	// call with wrong chat id
	response := s.jail.Call("chatIDNonExistent", "", "")
	expectedError := `{"error":"cell[chatIDNonExistent] doesn't exist"}`
	require.Equal(expectedError, response)

	// call extraFunc()
	response = s.jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse := `{"result": 144}`
	require.Equal(expectedResponse, response)
}

// TestJailRPCAsyncSend was written to catch race conditions with a weird error message
// starting from `ReferenceError` as if otto vm were losing symbols.
func (s *JailTestSuite) TestJailRPCAsyncSend() {
	require := s.Require()

	// load Status JS and add test command to it
	s.jail.BaseJS(baseStatusJSCode)
	s.jail.Parse(testChatID, txJSCode)

	cell, err := s.jail.Cell(testChatID)
	require.NoError(err)
	require.NotNil(cell)

	// internally (since we replaced `web3.send` with `jail.Send`)
	// all requests to web3 are forwarded to `jail.Send`
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err = cell.Run(`_status_catalog.commands.sendAsync({
				"from": "` + TestConfig.Account1.Address + `",
				"to": "` + TestConfig.Account2.Address + `",
				"value": "0.000001"
			})`)
			require.NoError(err, "Request failed to process")
		}()
	}
	wg.Wait()
}

func (s *JailTestSuite) TestJailRPCSend() {
	require := s.Require()

	s.StartTestNode(params.RopstenNetworkID)
	defer s.StopTestNode()

	// load Status JS and add test command to it
	s.jail.BaseJS(baseStatusJSCode)
	s.jail.Parse(testChatID, ``)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	cell, err := s.jail.Cell(testChatID)
	require.NoError(err)
	require.NotNil(cell)

	// internally (since we replaced `web3.send` with `jail.Send`)
	// all requests to web3 are forwarded to `jail.Send`
	_, err = cell.Run(`
	    var balance = web3.eth.getBalance("` + TestConfig.Account1.Address + `");
		var sendResult = web3.fromWei(balance, "ether")
	`)
	require.NoError(err)

	value, err := cell.Get("sendResult")
	require.NoError(err, "cannot obtain result of balance check operation")

	balance, err := value.ToFloat()
	require.NoError(err)

	s.T().Logf("Balance of %.2f ETH found on '%s' account", balance, TestConfig.Account1.Address)
	require.False(balance < 100, "wrong balance (there should be lots of test Ether on that account)")
}

func (s *JailTestSuite) TestIsConnected() {
	require := s.Require()

	s.StartTestNode(params.RopstenNetworkID)
	defer s.StopTestNode()

	s.jail.Parse(testChatID, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	cell, err := s.jail.Cell(testChatID)
	require.NoError(err)

	_, err = cell.Run(`
	    var responseValue = web3.isConnected();
	    responseValue = JSON.stringify(responseValue);
	`)
	require.NoError(err)

	responseValue, err := cell.Get("responseValue")
	require.NoError(err, "cannot obtain result of isConnected()")

	response, err := responseValue.ToString()
	require.NoError(err, "cannot parse result")

	expectedResponse := `{"jsonrpc":"2.0","result":true}`
	require.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestEventSignal() {
	require := s.Require()

	s.jail.Parse(testChatID, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	cell, err := s.jail.Cell(testChatID)
	require.NoError(err)

	testData := "foobar"

	opCompletedSuccessfully := make(chan struct{}, 1)

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		require.NoError(err)

		if envelope.Type == jail.EventSignal {
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

	_, err = cell.Run(`
	    var responseValue = statusSignals.sendSignal("` + testData + `");
	    responseValue = JSON.stringify(responseValue);
	`)
	s.NoError(err)

	// make sure that signal is sent (and its parameters are correct)
	select {
	case <-opCompletedSuccessfully:
		// pass
	case <-time.After(3 * time.Second):
		require.Fail("operation timed out")
	}

	responseValue, err := cell.Get("responseValue")
	require.NoError(err, "cannot obtain result of localStorage.set()")

	response, err := responseValue.ToString()
	require.NoError(err, "cannot parse result")

	expectedResponse := `{"jsonrpc":"2.0","result":true}`
	require.Equal(expectedResponse, response)
}
