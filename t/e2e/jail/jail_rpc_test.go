package jail

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/status-im/status-go/jail"
	"github.com/status-im/status-go/params"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestJailRPCTestSuite(t *testing.T) {
	suite.Run(t, new(JailRPCTestSuite))
}

type JailRPCTestSuite struct {
	e2e.BackendTestSuite

	jail jail.Manager
}

func (s *JailRPCTestSuite) SetupTest() {
	s.BackendTestSuite.SetupTest()
	s.jail = s.Backend.JailManager()
	s.NotNil(s.jail)
}

func (s *JailRPCTestSuite) TestJailRPCSend() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID)

	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.StatusNode().EnsureSync)

	// load Status JS and add test command to it
	s.jail.SetBaseJS(baseStatusJSCode)
	s.jail.CreateAndInitCell(testChatID)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	cell, err := s.jail.Cell(testChatID)
	s.NoError(err)
	s.NotNil(cell)

	// internally (since we replaced `web3.send` with `jail.Send`)
	// all requests to web3 are forwarded to `jail.Send`
	_, err = cell.Run(`
	    var balance = web3.eth.getBalance("` + TestConfig.Account1.Address + `");
		var sendResult = web3.fromWei(balance, "ether")
	`)
	s.NoError(err)

	value, err := cell.Get("sendResult")
	s.NoError(err, "cannot obtain result of balance check operation")

	balance, err := value.Value().ToFloat()
	s.NoError(err)

	s.T().Logf("Balance of %.2f ETH found on '%s' account", balance, TestConfig.Account1.Address)
	s.False(balance < 1, "wrong balance (there should be lots of test Ether on that account)")
}

func (s *JailRPCTestSuite) TestIsConnected() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	s.jail.CreateAndInitCell(testChatID)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	cell, err := s.jail.Cell(testChatID)
	s.NoError(err)

	_, err = cell.Run(`
	    var responseValue = web3.isConnected();
	    responseValue = JSON.stringify(responseValue);
	`)
	s.NoError(err)

	responseValue, err := cell.Get("responseValue")
	s.NoError(err, "cannot obtain result of isConnected()")

	response, err := responseValue.Value().ToBoolean()
	s.NoError(err, "cannot parse result")
	s.True(response)
}

// regression test: eth_getTransactionReceipt with invalid transaction hash should return "result":null.
func (s *JailRPCTestSuite) TestRegressionGetTransactionReceipt() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	rpcClient := s.Backend.StatusNode().RPCClient()
	s.NotNil(rpcClient)

	// note: transaction hash is assumed to be invalid
	got := rpcClient.CallRaw(`{"jsonrpc":"2.0","method":"eth_getTransactionReceipt","params":["0xbbebf28d0a3a3cbb38e6053a5b21f08f82c62b0c145a17b1c4313cac3f68ae7c"],"id":7}`)
	expected := `{"jsonrpc":"2.0","id":7,"result":null}`
	s.Equal(expected, got)
}

func (s *JailRPCTestSuite) TestJailVMPersistence() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID)

	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.StatusNode().EnsureSync)

	// log into account from which transactions will be sent
	err := s.Backend.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err, "cannot select account: %v", TestConfig.Account1.Address)

	type testCase struct {
		command   string
		params    string
		validator func(response string) error
	}
	var testCases = []testCase{
		{
			`["ping"]`,
			`{"pong": "Ping1", "amount": 0.42}`,
			func(response string) error {
				expectedResponse := `{"result":"Ping1"}`
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
				expectedResponse := `{"result":"Ping2"}`
				if response != expectedResponse {
					return fmt.Errorf("unexpected response, expected: %v, got: %v", expectedResponse, response)
				}
				return nil
			},
		},
	}

	jail := s.Backend.JailManager()
	jail.SetBaseJS(baseStatusJSCode)

	parseResult := jail.CreateAndInitCell(testChatID, `
		var total = 0;
		_status_catalog['ping'] = function(params) {
			total += Number(params.amount);
			return params.pong;
		}

		_status_catalog;
	`)
	s.NotContains(parseResult, "error", "further will fail if initial parsing failed")

	cell, err := jail.Cell(testChatID)
	s.NoError(err)

	// run commands concurrently
	var wg sync.WaitGroup
	for _, tc := range testCases {
		wg.Add(1)
		go func(tc testCase) {
			defer wg.Done() // ensure we don't forget it

			s.T().Logf("CALL START: %v %v", tc.command, tc.params)
			response := jail.Call(testChatID, tc.command, tc.params)
			if e := tc.validator(response); e != nil {
				s.T().Errorf("failed test validation: %v, err: %v", tc.command, e)
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

	// Validate total.
	totalOtto, err := cell.Get("total")
	s.NoError(err)

	total, err := totalOtto.Value().ToFloat()
	s.NoError(err)

	s.T().Log(total)
	// Should not have changed
	s.InDelta(0.840000, total, 0.0)
}
