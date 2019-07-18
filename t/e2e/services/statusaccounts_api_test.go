package services

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestStatusAccountsAPISuite(t *testing.T) {
	s := new(StatusAccountsAPISuite)
	s.upstream = false
	suite.Run(t, s)
}

type StatusAccountsAPISuite struct {
	BaseJSONRPCSuite
	upstream bool
}

func (s *StatusAccountsAPISuite) TestAccessibleStatusAccountsAPIs() {
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(s.upstream, true, false)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	methods := []string{
		"statusaccounts_generate",
		"statusaccounts_importMnemonic",
		"statusaccounts_importPrivateKey",
		"statusaccounts_importJSONKey",
		"statusaccounts_deriveAddresses",
		"statusaccounts_storeAccount",
		"statusaccounts_storeDerivedAccounts",
		"statusaccounts_loadAccount",
	}

	for _, method := range methods {
		s.AssertAPIMethodUnexported(method)
		s.AssertAPIMethodExportedPrivately(method)
	}
}

func (s *StatusAccountsAPISuite) TestLoginLogout() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	checkAccounts := func(ids []string, errorExpected bool) {
		for _, id := range ids {
			s.tryDerivingAddresses(id, errorExpected)
		}
	}

	accountsCount := 2

	// generate accounts
	ids := s.generateAccounts(accountsCount)
	// all accounts should be in memory
	checkAccounts(ids, false)
	// Logout
	s.NoError(s.Backend.Logout())
	// accounts should not exist after logging-out
	checkAccounts(ids, true)

	// generate other accounts
	ids = s.generateAccounts(accountsCount)
	// all accounts should be in memory
	checkAccounts(ids, false)

	// try Login with empty credential
	// it won't really log in but it should remove all the account
	// from memory.
	err := s.Backend.SelectAccount(account.LoginParams{})
	s.Error(err)

	// all accounts should be in memory
	checkAccounts(ids, true)
}

func (s *StatusAccountsAPISuite) generateAccounts(n int) []string {
	generateCall := fmt.Sprintf(`{"jsonrpc":"2.0","method":"statusaccounts_generate","params":[12, %d, ""],"id":1}`, n)
	rawResp, err := s.Backend.CallPrivateRPC(generateCall)
	s.NoError(err)

	var resp struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}

	s.NoError(json.Unmarshal([]byte(rawResp), &resp))
	s.Equal(n, len(resp.Result))

	ids := make([]string, n)
	for i := 0; i < len(resp.Result); i++ {
		ids[i] = resp.Result[i].ID
	}

	return ids
}

func (s *StatusAccountsAPISuite) tryDerivingAddresses(accountID string, errorExpected bool) {
	deriveAddressesCall := fmt.Sprintf(`{"jsonrpc":"2.0","method":"statusaccounts_deriveAddresses","params":["%s", ["m/1/2"]],"id":1}`, accountID)
	rawResp, err := s.Backend.CallPrivateRPC(deriveAddressesCall)
	s.NoError(err)

	accountNotFoundMessage := "account not found"
	if errorExpected {
		s.Contains(rawResp, accountNotFoundMessage)
	} else {
		s.NotContains(rawResp, accountNotFoundMessage)
	}
}
