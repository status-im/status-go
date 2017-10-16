package jail

import (
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/signal"
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
	suite.Suite
	jail common.JailManager
}

func (s *JailTestSuite) SetupTest() {
	s.jail = jail.New(nil)
	s.NotNil(s.jail)
}

func (s *JailTestSuite) TestInit() {
	errorWrapper := func(err error) string {
		return `{"error":"` + err.Error() + `"}`
	}

	// get cell VM w/o defining cell first
	cell, err := s.jail.Cell(testChatID)

	s.EqualError(err, "cell["+testChatID+"] doesn't exist")
	s.Nil(cell)

	// create VM (w/o properly initializing base JS script)
	err = errors.New("ReferenceError: '_status_catalog' is not defined")
	s.Equal(errorWrapper(err), s.jail.Parse(testChatID, ``))
	err = errors.New("ReferenceError: 'call' is not defined")
	s.Equal(errorWrapper(err), s.jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`))

	// get existing cell (even though we got errors, cell was still created)
	cell, err = s.jail.Cell(testChatID)
	s.NoError(err)
	s.NotNil(cell)

	statusJS := baseStatusJSCode + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	s.jail.BaseJS(statusJS)

	// now no error should occur
	response := s.jail.Parse(testChatID, ``)
	expectedResponse := `{"result": {"commands":{},"responses":{}}}`
	s.Equal(expectedResponse, response)

	// make sure that Call succeeds even w/o running node
	response = s.jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse = `{"result": 144}`
	s.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestParse() {
	extraCode := `
	var _status_catalog = {
		foo: 'bar'
	};`
	response := s.jail.Parse("newChat", extraCode)
	expectedResponse := `{"result": {"foo":"bar"}}`
	s.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestFunctionCall() {
	// load Status JS and add test command to it
	statusJS := baseStatusJSCode + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	s.jail.Parse(testChatID, statusJS)

	// call with wrong chat id
	response := s.jail.Call("chatIDNonExistent", "", "")
	expectedError := `{"error":"cell[chatIDNonExistent] doesn't exist"}`
	s.Equal(expectedError, response)

	// call extraFunc()
	response = s.jail.Call(testChatID, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse := `{"result": 144}`
	s.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestEventSignal() {
	s.jail.Parse(testChatID, "")

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	cell, err := s.jail.Cell(testChatID)
	s.NoError(err)

	testData := "foobar"
	opCompletedSuccessfully := make(chan struct{}, 1)

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
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

	expectedResponse := `{"jsonrpc":"2.0","result":true}`
	s.Equal(expectedResponse, response)
}

func (s *JailTestSuite) TestJailCellsRemovedAfterStop() {
	const loopLen = 5

	getTestCellID := func(id int) string {
		return testChatID + strconv.Itoa(id)
	}
	require := s.Require()

	for i := 0; i < loopLen; i++ {
		s.jail.Parse(getTestCellID(i), "")
		cell, err := s.jail.Cell(getTestCellID(i))
		require.NoError(err)
		_, err = cell.Run(`
			var counter = 1;
			setInterval(function(){
				counter++;
			}, 1000);
		`)
	}

	s.jail.Stop()

	for i := 0; i < loopLen; i++ {
		_, err := s.jail.Cell(getTestCellID(i))
		require.Error(err, "Expected cells removing (from Jail) after stop")
	}
}
