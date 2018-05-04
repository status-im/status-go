package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	acc "github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/signal"
	"github.com/stretchr/testify/suite"

	. "github.com/status-im/status-go/t/utils"
)

const (
	signDataString   = "0xBAADBEEF"
	accountNotExists = "0x00164ca341326a03b547c05B343b2E21eFAe2400"

	// see vendor/github.com/ethereum/go-ethereum/rpc/errors.go:L27
	methodNotFoundErrorCode = -32601
)

type rpcError struct {
	Code int `json:"code"`
}

type testParams struct {
	Title             string
	EnableUpstream    bool
	Account           string
	Password          string
	HandlerFactory    func(string, string) func(string)
	ExpectedError     error
	DontSelectAccount bool // to take advantage of the fact, that the default is `false`
}

func TestPersonalSignSuite(t *testing.T) {
	s := new(PersonalSignSuite)
	s.upstream = false
	suite.Run(t, s)
}

func TestPersonalSignSuiteUpstream(t *testing.T) {
	s := new(PersonalSignSuite)
	s.upstream = true
	suite.Run(t, s)
}

type PersonalSignSuite struct {
	BaseJSONRPCSuite
	upstream bool
}

func (s *PersonalSignSuite) TestRestrictedPersonalAPIs() {
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(s.upstream, false)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()
	// These personal APIs should be available
	s.AssertAPIMethodExported("personal_sign")
	s.AssertAPIMethodExported("personal_ecRecover")
	// These personal APIs shouldn't be exported
	s.AssertAPIMethodUnexported("personal_sendTransaction")
	s.AssertAPIMethodUnexported("personal_unlockAccount")
	s.AssertAPIMethodUnexported("personal_newAccount")
	s.AssertAPIMethodUnexported("personal_lockAccount")
	s.AssertAPIMethodUnexported("personal_listAccounts")
	s.AssertAPIMethodUnexported("personal_importRawKey")
}

func (s *PersonalSignSuite) TestPersonalSignSuccess() {
	s.testPersonalSign(testParams{
		Account:  TestConfig.Account1.Address,
		Password: TestConfig.Account1.Password,
	})
}

func (s *PersonalSignSuite) TestPersonalSignWrongPassword() {
	s.testPersonalSign(testParams{
		Account:        TestConfig.Account1.Address,
		Password:       TestConfig.Account1.Password,
		HandlerFactory: s.notificationHandlerWrongPassword,
	})
}

func (s *PersonalSignSuite) TestPersonalSignNoSuchAccount() {
	s.testPersonalSign(testParams{
		Account:        accountNotExists,
		Password:       TestConfig.Account1.Password,
		ExpectedError:  personal.ErrInvalidPersonalSignAccount,
		HandlerFactory: s.notificationHandlerNoAccount,
	})
}

func (s *PersonalSignSuite) TestPersonalSignWrongAccount() {
	s.testPersonalSign(testParams{
		Account:        TestConfig.Account2.Address,
		Password:       TestConfig.Account2.Password,
		ExpectedError:  personal.ErrInvalidPersonalSignAccount,
		HandlerFactory: s.notificationHandlerInvalidAccount,
	})
}

func (s *PersonalSignSuite) TestPersonalSignNoAccountSelected() {
	s.testPersonalSign(testParams{
		Account:           TestConfig.Account1.Address,
		Password:          TestConfig.Account1.Password,
		HandlerFactory:    s.notificationHandlerNoAccountSelected,
		DontSelectAccount: true,
	})
}

// Utility methods
func (s *PersonalSignSuite) notificationHandlerWrongPassword(account string, pass string) func(string) {
	return func(jsonEvent string) {
		s.notificationHandler(account, pass+"wrong", keystore.ErrDecrypt)(jsonEvent)
		s.notificationHandlerSuccess(account, pass)(jsonEvent)
	}
}

func (s *PersonalSignSuite) notificationHandlerNoAccount(account string, pass string) func(string) {
	return func(jsonEvent string) {
		s.notificationHandler(account, pass, personal.ErrInvalidPersonalSignAccount)(jsonEvent)
	}
}

func (s *PersonalSignSuite) notificationHandlerInvalidAccount(account string, pass string) func(string) {
	return func(jsonEvent string) {
		s.notificationHandler(account, pass, personal.ErrInvalidPersonalSignAccount)(jsonEvent)
	}
}

func (s *PersonalSignSuite) notificationHandlerNoAccountSelected(account string, pass string) func(string) {
	return func(jsonEvent string) {
		s.notificationHandler(account, pass, acc.ErrNoAccountSelected)(jsonEvent)
		envelope := unmarshalEnvelope(jsonEvent)
		if envelope.Type == signal.EventSignRequestAdded {
			err := s.Backend.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
			s.NoError(err)
		}
		s.notificationHandlerSuccess(account, pass)(jsonEvent)
	}
}

func (s *PersonalSignSuite) notificationHandler(account string, pass string, expectedError error) func(string) {
	return func(jsonEvent string) {
		envelope := unmarshalEnvelope(jsonEvent)
		if envelope.Type == signal.EventSignRequestAdded {
			event := envelope.Event.(map[string]interface{})
			id := event["id"].(string)
			s.T().Logf("Sign request added (will be completed shortly): {id: %s}\n", id)

			//check for the correct method name
			method := event["method"].(string)
			s.Equal(params.PersonalSignMethodName, method)
			//check the event data
			args := event["args"].(map[string]interface{})
			s.Equal(signDataString, args["data"].(string))
			s.Equal(account, args["account"].(string))

			e := s.Backend.ApproveSignRequest(id, pass).Error
			s.T().Logf("Sign request approved. {id: %s, acc: %s, err: %v}", id, account, e)
			if expectedError == nil {
				s.NoError(e, "cannot complete sign reauest[%v]: %v", id, e)
			} else {
				s.EqualError(e, expectedError.Error())
			}
		}
	}
}

func (s *PersonalSignSuite) testPersonalSign(testParams testParams) string {
	// Test upstream if that's not StatusChain
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return ""
	}

	if testParams.HandlerFactory == nil {
		testParams.HandlerFactory = s.notificationHandlerSuccess
	}

	err := s.SetupTest(s.upstream, false)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	signal.SetDefaultNodeNotificationHandler(testParams.HandlerFactory(testParams.Account, testParams.Password))

	if testParams.DontSelectAccount {
		s.NoError(s.Backend.Logout())
	} else {
		s.NoError(s.Backend.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))
	}

	// Parameters ordering here is MetaMask-compatible, not geth-compatible.
	// account *PRECEDES* data
	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"personal_sign","params":["%s", "%s"],"id":67}`,
		testParams.Account,
		signDataString)

	result := s.Backend.CallRPC(basicCall)
	if testParams.ExpectedError == nil {
		s.NotContains(result, "error")
		return s.extractResultFromRPCResponse(result)
	}

	s.Contains(result, testParams.ExpectedError.Error())
	return ""
}

func (s *PersonalSignSuite) extractResultFromRPCResponse(response string) string {
	var r struct {
		Result string `json:"result"`
	}
	s.NoError(json.Unmarshal([]byte(response), &r))

	return r.Result
}

func unmarshalEnvelope(jsonEvent string) signal.Envelope {
	var envelope signal.Envelope
	if e := json.Unmarshal([]byte(jsonEvent), &envelope); e != nil {
		panic(e)
	}
	return envelope
}

func (s *PersonalSignSuite) TestPersonalRecoverSuccess() {

	// 1. Sign
	signedData := s.testPersonalSign(testParams{
		Account:  TestConfig.Account1.Address,
		Password: TestConfig.Account1.Password,
	})

	// Test upstream if that's not StatusChain
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(s.upstream, false)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	// 2. Test recover
	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"personal_ecRecover","params":["%s", "%s"],"id":67}`,
		signDataString,
		signedData)

	response := s.Backend.CallRPC(basicCall)

	result := s.extractResultFromRPCResponse(response)

	s.True(strings.EqualFold(result, TestConfig.Account1.Address))
}

func (s *BaseJSONRPCSuite) notificationHandlerSuccess(account string, pass string) func(string) {
	return func(jsonEvent string) {
		s.notificationHandler(account, pass, nil)(jsonEvent)
	}
}
