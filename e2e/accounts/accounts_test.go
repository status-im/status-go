package accounts

import (
	"errors"
	"fmt"
	"testing"

	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/accounts"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/suite"
)

func TestAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(AccountsTestSuite))
}

type AccountsTestSuite struct {
	e2e.BackendTestSuite
}

func (s *AccountsTestSuite) TestAccountsList() {
	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	accounts, err := s.Backend.AccountManager().Accounts()
	s.NoError(err)

	// make sure that we start with empty accounts list (nobody has logged in yet)
	s.Zero(len(accounts), "accounts returned, while there should be none (we haven't logged in yet)")

	// create an account
	address, _, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// ensure that there is still no accounts returned
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Zero(len(accounts), "accounts returned, while there should be none (we haven't logged in yet)")

	// select account (sub-accounts will be created for this key)
	err = s.Backend.AccountManager().SelectAccount(address, TestConfig.Account1.Password)
	s.NoError(err, "account selection failed")

	// at this point main account should show up
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Equal(1, len(accounts), "exactly single account is expected (main account)")
	s.Equal(string(accounts[0].Hex()), "0x"+address,
		fmt.Sprintf("main account is not retured as the first key: got %s, expected %s", accounts[0].Hex(), "0x"+address))

	// create sub-account 1
	subAccount1, subPubKey1, err := s.Backend.AccountManager().CreateChildAccount("", TestConfig.Account1.Password)
	s.NoError(err, "cannot create sub-account")

	// now we expect to see both main account and sub-account 1
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Equal(2, len(accounts), "exactly 2 accounts are expected (main + sub-account 1)")
	s.Equal(string(accounts[0].Hex()), "0x"+address, "main account is not retured as the first key")
	s.Equal(string(accounts[1].Hex()), "0x"+subAccount1, "subAcount1 not returned")

	// create sub-account 2, index automatically progresses
	subAccount2, subPubKey2, err := s.Backend.AccountManager().CreateChildAccount("", TestConfig.Account1.Password)
	s.NoError(err, "cannot create sub-account")
	s.False(subAccount1 == subAccount2 || subPubKey1 == subPubKey2, "sub-account index auto-increament failed")

	// finally, all 3 accounts should show up (main account, sub-accounts 1 and 2)
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Equal(3, len(accounts), "unexpected number of accounts")
	s.Equal(string(accounts[0].Hex()), "0x"+address, "main account is not retured as the first key")

	subAccount1MatchesKey1 := string(accounts[1].Hex()) != "0x"+subAccount1
	subAccount1MatchesKey2 := string(accounts[2].Hex()) != "0x"+subAccount1
	s.False(!subAccount1MatchesKey1 && !subAccount1MatchesKey2, "subAcount1 not returned")

	subAccount2MatchesKey1 := string(accounts[1].Hex()) != "0x"+subAccount2
	subAccount2MatchesKey2 := string(accounts[2].Hex()) != "0x"+subAccount2
	s.False(!subAccount2MatchesKey1 && !subAccount2MatchesKey2, "subAcount2 not returned")
}

func (s *AccountsTestSuite) TestCreateChildAccount() {
	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	keyStore, err := s.Backend.NodeManager().AccountKeyStore()
	s.NoError(err)
	s.NotNil(keyStore)

	// create an account
	address, pubKey, mnemonic, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	acct, err := common.ParseAccountString(address)
	s.NoError(err, "can not get account from address")

	// obtain decrypted key, and make sure that extended key (which will be used as root for sub-accounts) is present
	_, key, err := keyStore.AccountDecryptedKey(acct, TestConfig.Account1.Password)
	s.NoError(err, "can not obtain decrypted account key")
	s.NotNil(key.ExtendedKey, "CKD#2 has not been generated for new account")

	// try creating sub-account, w/o selecting main account i.e. w/o login to main account
	_, _, err = s.Backend.AccountManager().CreateChildAccount("", TestConfig.Account1.Password)
	s.EqualError(accounts.ErrNoAccountSelected, err.Error(), "expected error is not returned (tried to create sub-account w/o login)")

	err = s.Backend.AccountManager().SelectAccount(address, TestConfig.Account1.Password)
	s.NoError(err, "cannot select account")

	// try to create sub-account with wrong password
	_, _, err = s.Backend.AccountManager().CreateChildAccount("", "wrong password")
	expectedErr := errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase")
	s.EqualError(expectedErr, err.Error(), "create sub-account with wrong password")

	// create sub-account (from implicit parent)
	subAccount1, subPubKey1, err := s.Backend.AccountManager().CreateChildAccount("", TestConfig.Account1.Password)
	s.NoError(err, "cannot create sub-account")

	// make sure that sub-account index automatically progresses
	subAccount2, subPubKey2, err := s.Backend.AccountManager().CreateChildAccount("", TestConfig.Account1.Password)
	s.NoError(err)
	s.False(subAccount1 == subAccount2 || subPubKey1 == subPubKey2, "sub-account index auto-increament failed")

	// create sub-account (from explicit parent)
	subAccount3, subPubKey3, err := s.Backend.AccountManager().CreateChildAccount(subAccount2, TestConfig.Account1.Password)
	s.NoError(err)
	s.False(subAccount1 == subAccount3 || subPubKey1 == subPubKey3 || subAccount2 == subAccount3 || subPubKey2 == subPubKey3)
}

func (s *AccountsTestSuite) TestRecoverAccount() {
	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	keyStore, err := s.Backend.NodeManager().AccountKeyStore()
	s.NoError(err)

	// create an account
	address, pubKey, mnemonic, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	// try recovering using password + mnemonic
	addressCheck, pubKeyCheck, err := s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover account failed")
	s.False(address != addressCheck || pubKey != pubKeyCheck, "incorrect accound details recovered")

	// now test recovering, but make sure that account/key file is removed i.e. simulate recovering on a new device
	account, err := common.ParseAccountString(address)
	s.NoError(err, "can not get account from address")

	account, key, err := keyStore.AccountDecryptedKey(account, TestConfig.Account1.Password)
	s.NoError(err, "can not obtain decrypted account key")
	extChild2String := key.ExtendedKey.String()

	s.NoError(keyStore.Delete(account, TestConfig.Account1.Password), "cannot remove account")

	addressCheck, pubKeyCheck, err = s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover account failed (for non-cached account)")
	s.False(address != addressCheck || pubKey != pubKeyCheck,
		"incorrect account details recovered (for non-cached account)")

	// make sure that extended key exists and is imported ok too
	_, key, err = keyStore.AccountDecryptedKey(account, TestConfig.Account1.Password)
	s.NoError(err)
	s.Equal(extChild2String, key.ExtendedKey.String(), "CKD#2 key mismatch")

	// make sure that calling import several times, just returns from cache (no error is expected)
	addressCheck, pubKeyCheck, err = s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover account failed (for non-cached account)")
	s.False(address != addressCheck || pubKey != pubKeyCheck,
		"incorrect account details recovered (for non-cached account)")

	// time to login with recovered data
	whisperService := s.WhisperService()

	// make sure that identity is not (yet injected)
	s.False(whisperService.HasKeyPair(pubKeyCheck), "identity already present in whisper")
	s.NoError(s.Backend.AccountManager().SelectAccount(addressCheck, TestConfig.Account1.Password))
	s.True(whisperService.HasKeyPair(pubKeyCheck), "identity not injected into whisper")
}

func (s *AccountsTestSuite) TestSelectAccount() {
	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	// test to see if the account was injected in whisper
	whisperService := s.WhisperService()

	// create an account
	address1, pubKey1, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {address: %s, key: %s}", address1, pubKey1)

	address2, pubKey2, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {address: %s, key: %s}", address2, pubKey2)

	// make sure that identity is not (yet injected)
	s.False(whisperService.HasKeyPair(pubKey1), "identity already present in whisper")

	// try selecting with wrong password
	err = s.Backend.AccountManager().SelectAccount(address1, "wrongPassword")
	expectedErr := errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase")
	s.EqualError(expectedErr, err.Error(), "select account is expected to throw error: wrong password used")

	err = s.Backend.AccountManager().SelectAccount(address1, TestConfig.Account1.Password)
	s.NoError(err)
	s.True(whisperService.HasKeyPair(pubKey1), "identity not injected into whisper")

	// select another account, make sure that previous account is wiped out from Whisper cache
	s.False(whisperService.HasKeyPair(pubKey2), "identity already present in whisper")
	s.NoError(s.Backend.AccountManager().SelectAccount(address2, TestConfig.Account1.Password))
	s.True(whisperService.HasKeyPair(pubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(pubKey1), "identity should be removed, but it is still present in whisper")
}

func (s *AccountsTestSuite) TestSelectedAccountOnRestart() {
	s.StartTestBackend(params.RinkebyNetworkID)

	// we need to make sure that selected account is injected as identity into Whisper
	whisperService := s.WhisperService()

	// create test accounts
	address1, pubKey1, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	address2, pubKey2, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// make sure that identity is not (yet injected)
	s.False(whisperService.HasKeyPair(pubKey1), "identity already present in whisper")

	// make sure that no account is selected by default
	selectedAccount, err := s.Backend.AccountManager().SelectedAccount()
	s.EqualError(accounts.ErrNoAccountSelected, err.Error(), "account selected, but should not be")
	s.Nil(selectedAccount)

	// select account
	err = s.Backend.AccountManager().SelectAccount(address1, "wrongPassword")
	expectedErr := errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase")
	s.EqualError(expectedErr, err.Error())

	s.NoError(s.Backend.AccountManager().SelectAccount(address1, TestConfig.Account1.Password))
	s.True(whisperService.HasKeyPair(pubKey1), "identity not injected into whisper")

	// select another account, make sure that previous account is wiped out from Whisper cache
	s.False(whisperService.HasKeyPair(pubKey2), "identity already present in whisper")
	s.NoError(s.Backend.AccountManager().SelectAccount(address2, TestConfig.Account1.Password))
	s.True(whisperService.HasKeyPair(pubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(pubKey1), "identity should be removed, but it is still present in whisper")

	// stop node (and all of its sub-protocols)
	nodeConfig, err := s.Backend.NodeManager().NodeConfig()
	s.NoError(err)
	preservedNodeConfig := *nodeConfig
	nodeStoped, err := s.Backend.StopNode()
	s.NoError(err)
	<-nodeStoped

	// make sure that account is still selected
	selectedAccount, err = s.Backend.AccountManager().SelectedAccount()
	s.NoError(err)
	s.NotNil(selectedAccount)
	s.Equal(selectedAccount.Address.Hex(), "0x"+address2, "incorrect address selected")

	// resume node
	nodeStarted, err := s.Backend.StartNode(&preservedNodeConfig)
	s.NoError(err)
	<-nodeStarted

	// re-check selected account (account2 MUST be selected)
	selectedAccount, err = s.Backend.AccountManager().SelectedAccount()
	s.NoError(err)
	s.NotNil(selectedAccount)
	s.Equal(selectedAccount.Address.Hex(), "0x"+address2, "incorrect address selected")

	// make sure that Whisper gets identity re-injected
	whisperService = s.WhisperService()
	s.True(whisperService.HasKeyPair(pubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(pubKey1), "identity should not be present, but it is still present in whisper")

	// now restart node using RestartNode() method, and make sure that account is still available
	s.RestartTestNode()
	defer s.StopTestBackend()

	whisperService = s.WhisperService()
	s.True(whisperService.HasKeyPair(pubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(pubKey1), "identity should not be present, but it is still present in whisper")

	// now logout, and make sure that on restart no account is selected (i.e. logout works properly)
	s.NoError(s.Backend.AccountManager().Logout())
	s.RestartTestNode()
	whisperService = s.WhisperService()
	s.False(whisperService.HasKeyPair(pubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(pubKey1), "identity should not be present, but it is still present in whisper")

	selectedAccount, err = s.Backend.AccountManager().SelectedAccount()
	s.EqualError(accounts.ErrNoAccountSelected, err.Error())
	s.Nil(selectedAccount)
}
