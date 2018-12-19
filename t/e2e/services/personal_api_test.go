package services

import (
	"fmt"
	"testing"

	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/suite"

	. "github.com/status-im/status-go/t/utils"
)

const (
	signDataString = "0xBAADBEEF"
)

type PersonalSignSuite struct {
	upstream bool
	BaseJSONRPCSuite
}

func TestPersonalSignSuiteUpstream(t *testing.T) {
	s := new(PersonalSignSuite)
	s.upstream = true
	suite.Run(t, s)
}

func (s *PersonalSignSuite) TestRestrictedPersonalAPIs() {
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(true, false, false)
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

func (s *PersonalSignSuite) TestPersonalSignUnsupportedMethod() {
	// Test upstream if that's not StatusChain
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
	}

	err := s.SetupTest(true, false, false)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"personal_sign","params":["%s", "%s"],"id":67}`,
		signDataString,
		TestConfig.Account1.Address)

	rawResult, err := s.Backend.CallRPC(basicCall)
	s.NoError(err)
	s.Contains(rawResult, `"error":{"code":-32700,"message":"method is unsupported by RPC interface"}`)
}

func (s *PersonalSignSuite) TestPersonalRecoverUnsupportedMethod() {

	// Test upstream if that's not StatusChain
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(true, false, false)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	// 2. Test recover
	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"personal_ecRecover","params":["%s", "%s"],"id":67}`,
		signDataString,
		"")

	rawResult, err := s.Backend.CallRPC(basicCall)
	s.NoError(err)
	s.Contains(rawResult, `"error":{"code":-32700,"message":"method is unsupported by RPC interface"}`)
}
