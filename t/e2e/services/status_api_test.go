package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/status"
	"github.com/status-im/status-go/signal"
	"github.com/stretchr/testify/suite"

	. "github.com/status-im/status-go/t/utils"
)

type statusTestParams struct {
	Address        string
	Password       string
	HandlerFactory func(string, string) func(string)
	ExpectedError  error
	ChannelName    string
}

func TestStatusAPISuite(t *testing.T) {
	s := new(StatusAPISuite)
	s.upstream = false
	suite.Run(t, s)
}

func TestStatusAPISuiteUpstream(t *testing.T) {
	s := new(StatusAPISuite)
	s.upstream = true
	suite.Run(t, s)
}

type StatusAPISuite struct {
	BaseJSONRPCSuite
	upstream bool
}

func (s *StatusAPISuite) TestAccessibleStatusAPIs() {
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(s.upstream, true)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()
	// These status APIs should be unavailable
	s.AssertAPIMethodUnexported("status_login")
	s.AssertAPIMethodUnexported("status_signup")

	// These status APIs should be available only for IPC
	s.AssertAPIMethodExportedPrivately("status_login")
	s.AssertAPIMethodExportedPrivately("status_signup")
}

func (s *StatusAPISuite) TestStatusLoginSuccess() {
	addressKeyID := s.testStatusLogin(statusTestParams{
		Address:  TestConfig.Account1.Address,
		Password: TestConfig.Account1.Password,
	})
	s.NotEmpty(addressKeyID)
}

func (s *StatusAPISuite) TestStatusLoginInvalidAddress() {
	s.testStatusLogin(statusTestParams{
		Address:       "invalidaccount",
		Password:      TestConfig.Account1.Password,
		ExpectedError: account.ErrAddressToAccountMappingFailure,
	})
}

func (s *StatusAPISuite) TestStatusLoginInvalidPassword() {
	s.testStatusLogin(statusTestParams{
		Address:       "invalidaccount",
		Password:      TestConfig.Account1.Password,
		ExpectedError: account.ErrAddressToAccountMappingFailure,
	})
}

func (s *StatusAPISuite) TestStatusSignupSuccess() {
	var pwd = "randompassword"

	res := s.testStatusSignup(statusTestParams{
		Password: pwd,
	})
	s.NotEmpty(res.Address)
	s.NotEmpty(res.Pubkey)
	s.Equal(12, len(strings.Split(res.Mnemonic, " ")))

	// I should be able to login with the newly created account
	_ = s.testStatusLogin(statusTestParams{
		Address:  res.Address,
		Password: pwd,
	})
}

func (s *StatusAPISuite) testStatusLogin(testParams statusTestParams) *status.LoginResponse {
	// Test upstream if that's not StatusChain
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return nil
	}

	if testParams.HandlerFactory == nil {
		testParams.HandlerFactory = s.notificationHandlerSuccess
	}

	err := s.SetupTest(s.upstream, true)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	signal.SetDefaultNodeNotificationHandler(testParams.HandlerFactory(testParams.Address, testParams.Password))

	req := status.LoginRequest{
		Addr:     testParams.Address,
		Password: testParams.Password,
	}
	body, _ := json.Marshal(req)

	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"status_login","params":[%s],"id":67}`,
		body)

	result := s.Backend.CallPrivateRPC(basicCall)
	if testParams.ExpectedError == nil {
		var r struct {
			Error  string                `json:"error"`
			Result *status.LoginResponse `json:"result"`
		}
		s.NoError(json.Unmarshal([]byte(result), &r))
		s.Empty(r.Error)

		return r.Result
	}

	s.Contains(result, testParams.ExpectedError.Error())
	return nil
}

func (s *StatusAPISuite) testStatusSignup(testParams statusTestParams) *status.SignupResponse {
	// Test upstream if that's not StatusChain
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return nil
	}

	if testParams.HandlerFactory == nil {
		testParams.HandlerFactory = s.notificationHandlerSuccess
	}

	err := s.SetupTest(s.upstream, true)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	signal.SetDefaultNodeNotificationHandler(testParams.HandlerFactory(testParams.Address, testParams.Password))

	req := status.SignupRequest{
		Password: testParams.Password,
	}
	body, _ := json.Marshal(req)

	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"status_signup","params":[%s],"id":67}`,
		body)

	result := s.Backend.CallPrivateRPC(basicCall)

	if testParams.ExpectedError == nil {
		var r struct {
			Error  string                 `json:"error"`
			Result *status.SignupResponse `json:"result"`
		}
		s.NoError(json.Unmarshal([]byte(result), &r))
		s.Empty(r.Error)

		return r.Result
	}

	s.Contains(result, testParams.ExpectedError.Error())
	return nil
}
