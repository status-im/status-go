package accounts

import (
	"strings"
	"testing"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestAccountsRPCTestSuite(t *testing.T) {
	suite.Run(t, new(AccountsRPCTestSuite))
}

type AccountsRPCTestSuite struct {
	e2e.BackendTestSuite
}

func (s *AccountsTestSuite) TestRPCEthAccounts() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	// log into test account
	err := s.Backend.SelectAccount(TestConfig.Account1.WalletAddress, TestConfig.Account1.ChatAddress, TestConfig.Account1.Password)
	s.NoError(err)

	rpcClient := s.Backend.StatusNode().RPCClient()
	s.NotNil(rpcClient)

	expectedResponse := `{"jsonrpc":"2.0","id":1,"result":["` + strings.ToLower(TestConfig.Account1.WalletAddress) + `"]}`
	resp := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_accounts",
		"params": []
    }`)

	s.Equal(expectedResponse, resp)
}

func (s *AccountsTestSuite) TestRPCEthAccountsWithUpstream() {
	if GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
	}

	addr, err := GetRemoteURL()
	s.NoError(err)
	s.StartTestBackend(e2e.WithUpstream(addr))
	defer s.StopTestBackend()

	// log into test account
	err = s.Backend.SelectAccount(TestConfig.Account1.WalletAddress, TestConfig.Account1.ChatAddress, TestConfig.Account1.Password)
	s.NoError(err)

	rpcClient := s.Backend.StatusNode().RPCClient()
	s.NotNil(rpcClient)

	expectedResponse := `{"jsonrpc":"2.0","id":1,"result":["` + strings.ToLower(TestConfig.Account1.WalletAddress) + `"]}`
	resp := rpcClient.CallRaw(`{
    	"jsonrpc": "2.0",
    	"id": 1,
    	"method": "eth_accounts",
    	"params": []
    }`)
	s.Equal(expectedResponse, resp)
}
