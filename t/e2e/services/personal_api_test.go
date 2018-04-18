package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	acc "github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/sign"
	e2e "github.com/status-im/status-go/t/e2e"
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
	e2e.BackendTestSuite
	upstream bool
}

func (s *PersonalSignSuite) TestRestrictedPersonalAPIs() {
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.initTest(s.upstream)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()
	// These personal APIs should be available
	s.testAPIExported("personal_sign", true)
	s.testAPIExported("personal_ecRecover", true)
	// These personal APIs shouldn't be exported
	s.testAPIExported("personal_sendTransaction", false)
	s.testAPIExported("personal_unlockAccount", false)
	s.testAPIExported("personal_newAccount", false)
	s.testAPIExported("personal_lockAccount", false)
	s.testAPIExported("personal_listAccounts", false)
	s.testAPIExported("personal_importRawKey", false)
}

func (s *PersonalSignSuite) testAPIExported(method string, expectExported bool) {
	cmd := fmt.Sprintf(`{"jsonrpc":"2.0", "method": "%s", "params": []}`, method)

	result := s.Backend.CallRPC(cmd)

	var response struct {
		Error *rpcError `json:"error"`
	}

	s.NoError(json.Unmarshal([]byte(result), &response))

	hidden := (response.Error != nil && response.Error.Code == methodNotFoundErrorCode)

	s.Equal(expectExported, !hidden,
		"method %s should be %s, but it isn't",
		method, map[bool]string{true: "exported", false: "hidden"}[expectExported])
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
func (s *PersonalSignSuite) notificationHandlerSuccess(account string, pass string) func(string) {
	return func(jsonEvent string) {
		s.notificationHandler(account, pass, nil)(jsonEvent)
	}
}

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
		if envelope.Type == sign.EventSignRequestAdded {
			err := s.Backend.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
			s.NoError(err)
		}
		s.notificationHandlerSuccess(account, pass)(jsonEvent)
	}
}

func (s *PersonalSignSuite) notificationHandler(account string, pass string, expectedError error) func(string) {
	return func(jsonEvent string) {
		envelope := unmarshalEnvelope(jsonEvent)
		if envelope.Type == sign.EventSignRequestAdded {
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

	err := s.initTest(s.upstream)
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

	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"personal_sign","params":["%s", "%s"],"id":67}`,
		signDataString,
		testParams.Account)

	result := s.Backend.CallRPC(basicCall)
	if testParams.ExpectedError == nil {
		s.NotContains(result, "error")
		return s.extractResultFromRPCResponse(result)
	}

	s.Contains(result, testParams.ExpectedError.Error())
	return ""
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

	err := s.initTest(s.upstream)
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

func (s *PersonalSignSuite) initTest(upstreamEnabled bool) error {
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	nodeConfig.IPCEnabled = false
	nodeConfig.HTTPHost = "" // to make sure that no HTTP interface is started

	if upstreamEnabled {
		networkURL, err := GetRemoteURL()
		s.NoError(err)

		nodeConfig.UpstreamConfig.Enabled = true
		nodeConfig.UpstreamConfig.URL = networkURL
	}

	return s.Backend.StartNode(nodeConfig)
}

func (s *PersonalSignSuite) extractResultFromRPCResponse(response string) string {
	var r struct {
		Result string `json:"result"`
	}
	s.NoError(json.Unmarshal([]byte(response), &r))

	return r.Result
}
