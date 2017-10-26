package jail

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/static"
	"github.com/stretchr/testify/suite"
)

const (
	testChatID = "testChat"
)

var (
	baseStatusJSCode = string(static.MustAsset("testdata/jail/status.js"))
)

func TestJailTestSuite(t *testing.T) {
	suite.Run(t, new(JailTestSuite))
}

type JailTestSuite struct {
	e2e.NodeManagerTestSuite
	Jail common.JailManager
}

func (s *JailTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.Jail = jail.New(s.NodeManager)
}

func (s *JailTestSuite) TearDownTest() {
	s.Jail.Stop()
}

func (s *JailTestSuite) TestInitWithoutBaseJS() {
	errorWrapper := func(err error) string {
		return `{"error":"` + err.Error() + `"}`
	}

	// get cell VM w/o defining cell first
	cell, err := s.Jail.GetCell(testChatID)

	s.EqualError(err, "cell '"+testChatID+"' not found")
	s.Nil(cell)

	// create VM (w/o properly initializing base JS script)
	err = errors.New("ReferenceError: '_status_catalog' is not defined")
	s.Equal(errorWrapper(err), s.Jail.CreateAndInitCell(testChatID, ``))
	err = errors.New("ReferenceError: 'call' is not defined")
	s.Equal(errorWrapper(err), s.Jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`))

	// get existing cell (even though we got errors, cell was still created)
	cell, err = s.Jail.GetCell(testChatID)
	s.NoError(err)
	s.NotNil(cell)
}

func (s *JailTestSuite) TestInitWithBaseJS() {
	statusJS := baseStatusJSCode + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	s.Jail.SetBaseJS(statusJS)

	// now no error should occur
	response := s.Jail.CreateAndInitCell(testChatID, ``)
	expectedResponse := `{"result": {"commands":{},"responses":{}}}`
	s.Equal(expectedResponse, response)

	// make sure that Call succeeds even w/o running node
	response = s.Jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse = `{"result": 144}`
	s.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestMultipleInitError() {
	var response string

	response = s.Jail.CreateAndInitCell(testChatID, ``)
	s.Equal(`{"error":"ReferenceError: '_status_catalog' is not defined"}`, response)

	response = s.Jail.CreateAndInitCell(testChatID, ``)
	s.Equal(`{"error":"cell with id 'testChat' already exists"}`, response)
}

func (s *JailTestSuite) TestCreateAndInitResponse() {
	extraCode := `
	var _status_catalog = {
		foo: 'bar'
	};`
	response := s.Jail.CreateAndInitCell("newChat", extraCode)
	expectedResponse := `{"result": {"foo":"bar"}}`
	s.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestFunctionCall() {
	// load Status JS and add test command to it
	statusJS := baseStatusJSCode + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	s.Jail.CreateAndInitCell(testChatID, statusJS)

	// call with wrong chat id
	response := s.Jail.Call("chatIDNonExistent", "", "")
	expectedError := `{"error":"cell 'chatIDNonExistent' not found"}`
	s.Equal(expectedError, response)

	// call extraFunc()
	response = s.Jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse := `{"result": 144}`
	s.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestEventSignal() {
	s.StartTestNode(params.RinkebyNetworkID)
	defer s.StopTestNode()

	s.Jail.CreateAndInitCell(testChatID, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	cell, err := s.Jail.GetCell(testChatID)
	s.NoError(err)

	testData := "foobar"
	opCompletedSuccessfully := make(chan struct{}, 1)

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err = json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err)

		if envelope.Type == jail.EventSignal {
			event := envelope.Event.(map[string]interface{})
			chatID, ok := event["chat_id"].(string)
			s.True(ok, "chat id is required, but not found")
			s.Equal(testChatID, chatID, "incorrect chat ID")

			actualData, ok := event["data"].(string)
			s.True(ok, "data field is required, but not found")
			s.Equal(testData, actualData, "incorrect data")

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
	case <-time.After(5 * time.Second):
		s.FailNow("test timed out")
	}

	responseValue, err := cell.Get("responseValue")
	s.NoError(err, "cannot obtain result of localStorage.set()")

	response, err := responseValue.ToString()
	s.NoError(err, "cannot parse result")

	expectedResponse := `{"result":true}`
	s.Equal(expectedResponse, response)
}

// TestCallResponseOrder tests exactly the problem from
// https://github.com/status-im/status-go/issues/372
func (s *JailTestSuite) TestSendSyncResponseOrder() {
	s.StartTestNode(params.RinkebyNetworkID)
	defer s.StopTestNode()

	// `testCommand` is a simple JS function. `calculateGasPrice` makes
	// an implicit JSON-RPC call via `send` handler (it's a sync call).
	// `web3.eth.gasPrice` is chosen to call `send` handler under the hood
	// because it's a simple RPC method and does not require any params.
	statusJS := baseStatusJSCode + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};
	_status_catalog.commands["calculateGasPrice"] = function (n) {
		var gasMultiplicator = Math.pow(1.4, n).toFixed(3);
		var price = 211000000000;
		try {
			price = web3.eth.gasPrice;
		} catch (err) {}

		return price * gasMultiplicator;
	};
	`
	s.Jail.CreateAndInitCell(testChatID, statusJS)

	// Concurrently call `testCommand` and `calculateGasPrice` and do some assertions.
	// If the code executed in cell's VM is not thread-safe, this test will likely panic.
	N := 10
	errCh := make(chan error, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res := s.Jail.Call(testChatID, `["commands", "testCommand"]`, fmt.Sprintf(`{"val": %d}`, i))
			if !strings.Contains(string(res), fmt.Sprintf("result\": %d", i*i)) {
				errCh <- fmt.Errorf("result should be '%d', got %s", i*i, res)
			}
		}(i)

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res := s.Jail.Call(testChatID, `["commands", "calculateGasPrice"]`, fmt.Sprintf(`%d`, i))
			if strings.Contains(string(res), "error") {
				errCh <- fmt.Errorf("result should not contain 'error', got %s", res)
			}
		}(i)
	}

	wg.Wait()

	close(errCh)
	for e := range errCh {
		s.NoError(e)
	}
}

func (s *JailTestSuite) TestJailCellsRemovedAfterStop() {
	const loopLen = 5

	getTestCellID := func(id int) string {
		return testChatID + strconv.Itoa(id)
	}
	require := s.Require()

	for i := 0; i < loopLen; i++ {
		s.Jail.CreateAndInitCell(getTestCellID(i), "")
		cell, err := s.Jail.GetCell(getTestCellID(i))
		require.NoError(err)
		_, err = cell.Run(`
			var counter = 1;
			setInterval(function(){
				counter++;
			}, 1000);
		`)
		require.NoError(err)
	}

	s.Jail.Stop()

	for i := 0; i < loopLen; i++ {
		_, err := s.Jail.GetCell(getTestCellID(i))
		require.Error(err, "Expected cells removing (from Jail) after stop")
	}
}
