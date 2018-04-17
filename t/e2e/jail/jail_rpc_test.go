package jail

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/sign"
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

// regression test: eth_getTransactionReceipt with invalid transaction hash should return "error":{"code":-32000,"message":"unknown transaction"}
func (s *JailRPCTestSuite) TestRegressionGetTransactionReceipt() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	rpcClient := s.Backend.StatusNode().RPCClient()
	s.NotNil(rpcClient)

	// note: transaction hash is assumed to be invalid
	got := rpcClient.CallRaw(`{"jsonrpc":"2.0","method":"eth_getTransactionReceipt","params":["0xbbebf28d0a3a3cbb38e6053a5b21f08f82c62b0c145a17b1c4313cac3f68ae7c"],"id":7}`)
	expected := `{"jsonrpc":"2.0","id":7,"error":{"code":-32000,"message":"unknown transaction"}}`
	s.Equal(expected, got)
}

func (s *JailRPCTestSuite) TestContractDeployment() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.StatusNode().EnsureSync)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	s.jail.CreateAndInitCell(testChatID)

	cell, err := s.jail.Cell(testChatID)
	s.NoError(err)

	completeQueuedTransaction := make(chan struct{})

	var txHash gethcommon.Hash
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		unmarshalErr := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(unmarshalErr, "cannot unmarshal JSON: %s", jsonEvent)

		if envelope.Type == sign.EventSignRequestAdded {
			event := envelope.Event.(map[string]interface{})
			s.T().Logf("transaction queued and will be completed shortly, id: %v", event["id"])

			s.NoError(s.Backend.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

			txID := event["id"].(string)
			result := s.Backend.ApproveSignRequest(txID, TestConfig.Account1.Password)
			txHash.SetBytes(result.Response.Bytes())
			if s.NoError(result.Error, event["id"]) {
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
			if (e) {
				errorValue = e;
				return;
			}
			// Once the contract has the transactionHash property set and once its deployed on an address.
			if (!contract.address) {
				responseValue = contract.transactionHash;
			}
		})
	`)
	s.NoError(err)

	select {
	case <-completeQueuedTransaction:
	case <-time.After(time.Minute):
		s.FailNow("test timed out")
	}

	// Wait until callback is fired and `responseValue` is set. Hacky but simple.
	time.Sleep(5 * time.Second)

	errorValue, err := cell.Get("errorValue")
	s.NoError(err)
	s.Equal("null", errorValue.Value().String())

	responseValue, err := cell.Get("responseValue")
	s.NoError(err)

	response, err := responseValue.Value().ToString()
	s.NoError(err)

	expectedResponse := txHash.Hex()
	s.Equal(expectedResponse, response)
}

func (s *JailRPCTestSuite) TestJailVMPersistence() {
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

		_status_catalog;
	`)
	s.NotContains(parseResult, "error", "further will fail if initial parsing failed")

	var wg sync.WaitGroup
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		if e := json.Unmarshal([]byte(jsonEvent), &envelope); e != nil {
			s.T().Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == sign.EventSignRequestAdded {
			event := envelope.Event.(map[string]interface{})
			s.T().Logf("Transaction queued (will be completed shortly): {id: %s}\n", event["id"].(string))

			var txHash gethcommon.Hash
			txID := event["id"].(string)
			result := s.Backend.ApproveSignRequest(txID, TestConfig.Account1.Password)
			s.NoError(result.Error, "cannot complete queued transaction[%v]: %v", event["id"], result.Error)

			txHash.SetBytes(result.Response.Bytes())

			s.T().Logf("Transaction complete: https://ropsten.etherscan.io/tx/%s", txHash.Hex())
		}
	})

	// run commands concurrently
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

	// Wait till eth_sendTransaction callbacks have been executed.
	// FIXME(tiabc): more reliable means of testing that.
	time.Sleep(5 * time.Second)

	// Validate total.
	cell, err := jail.Cell(testChatID)
	s.NoError(err)

	totalOtto, err := cell.Get("total")
	s.NoError(err)

	total, err := totalOtto.Value().ToFloat()
	s.NoError(err)

	s.T().Log(total)
	s.InDelta(0.840003, total, 0.0000001)
}
