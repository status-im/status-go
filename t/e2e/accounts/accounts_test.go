package accounts

import (
	"errors"
	"fmt"
	"testing"

	"github.com/status-im/status-go/account"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(AccountsTestSuite))
}

type AccountsTestSuite struct {
	e2e.BackendTestSuite
}

func (s *AccountsTestSuite) TestAccountsList() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	accounts, err := s.Backend.AccountManager().Accounts()
	s.NoError(err)

	// make sure that we start with empty accounts list (nobody has logged in yet)
	s.Zero(len(accounts), "accounts returned, while there should be none (we haven't logged in yet)")

	// create an account
	walletAddress, _, chatAddress, _, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// ensure that there is still no accounts returned
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Zero(len(accounts), "accounts returned, while there should be none (we haven't logged in yet)")

	// select account (sub-accounts will be created for this key)
	err = s.Backend.SelectAccount(walletAddress, chatAddress, TestConfig.Account1.Password)
	s.NoError(err, "account selection failed")

	// at this point main account should show up
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Equal(1, len(accounts), "exactly single account is expected (main account)")
	s.Equal(accounts[0].Hex(), walletAddress,
		fmt.Sprintf("main account is not retured as the first key: got %s, expected %s", accounts[0].Hex(), "0x"+walletAddress))

	// create sub-account 1
	subAccount1, subPubKey1, err := s.Backend.AccountManager().CreateChildAccount("", TestConfig.Account1.Password)
	s.NoError(err, "cannot create sub-account")

	// now we expect to see both main account and sub-account 1
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Equal(2, len(accounts), "exactly 2 accounts are expected (main + sub-account 1)")
	s.Equal(accounts[0].Hex(), walletAddress, "main account is not retured as the first key")
	s.Equal(accounts[1].Hex(), subAccount1, "subAcount1 not returned")

	// create sub-account 2, index automatically progresses
	subAccount2, subPubKey2, err := s.Backend.AccountManager().CreateChildAccount("", TestConfig.Account1.Password)
	s.NoError(err, "cannot create sub-account")
	s.False(subAccount1 == subAccount2 || subPubKey1 == subPubKey2, "sub-account index auto-increament failed")

	// finally, all 3 accounts should show up (main account, sub-accounts 1 and 2)
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Equal(3, len(accounts), "unexpected number of accounts")
	s.Equal(accounts[0].Hex(), walletAddress, "main account is not retured as the first key")

	subAccount1MatchesKey1 := accounts[1].Hex() != "0x"+subAccount1
	subAccount1MatchesKey2 := accounts[2].Hex() != "0x"+subAccount1
	s.False(!subAccount1MatchesKey1 && !subAccount1MatchesKey2, "subAcount1 not returned")

	subAccount2MatchesKey1 := accounts[1].Hex() != "0x"+subAccount2
	subAccount2MatchesKey2 := accounts[2].Hex() != "0x"+subAccount2
	s.False(!subAccount2MatchesKey1 && !subAccount2MatchesKey2, "subAcount2 not returned")
}

func (s *AccountsTestSuite) TestCreateChildAccount() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	keyStore, err := s.Backend.StatusNode().AccountKeyStore()
	s.NoError(err)
	s.NotNil(keyStore)

	// create an account
	walletAddress, walletPubKey, chatAddress, chatPubKey, mnemonic, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {walletAddress: %s, walletKey: %s, chatAddress: %s, chatKey: %s, mnemonic:%s}",
		walletAddress, walletPubKey, chatAddress, chatPubKey, mnemonic)

	acct, err := account.ParseAccountString(walletAddress)
	s.NoError(err, "can not get account from address")

	// obtain decrypted key, and make sure that extended key (which will be used as root for sub-accounts) is present
	_, key, err := keyStore.AccountDecryptedKey(acct, TestConfig.Account1.Password)
	s.NoError(err, "can not obtain decrypted account key")
	s.NotNil(key.ExtendedKey, "CKD#2 has not been generated for new account")

	// try creating sub-account, w/o selecting main account i.e. w/o login to main account
	_, _, err = s.Backend.AccountManager().CreateChildAccount("", TestConfig.Account1.Password)
	s.EqualError(account.ErrNoAccountSelected, err.Error(), "expected error is not returned (tried to create sub-account w/o login)")

	err = s.Backend.SelectAccount(walletAddress, chatAddress, TestConfig.Account1.Password)
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
	s.StartTestBackend()
	defer s.StopTestBackend()

	keyStore, err := s.Backend.StatusNode().AccountKeyStore()
	s.NoError(err)

	// create an acc
	walletAddress, walletPubKey, chatAddress, chatPubKey, mnemonic, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {walletAddress: %s, walletKey: %s, chatAddress: %s, chatKey: %s, mnemonic:%s}",
		walletAddress, walletPubKey, chatAddress, chatPubKey, mnemonic)

	// try recovering using password + mnemonic
	walletAddressCheck, walletPubKeyCheck, chatAddressCheck, chatPubKeyCheck, err := s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover acc failed")

	s.True(walletAddress == walletAddressCheck &&
		walletPubKey == walletPubKeyCheck &&
		chatAddress == chatAddressCheck &&
		chatPubKey == chatPubKeyCheck,
		"incorrect accound details recovered")

	// now test recovering, but make sure that acc/key file is removed i.e. simulate recovering on a new device
	acc, err := account.ParseAccountString(walletAddress)
	s.NoError(err, "can not get acc from address")

	acc, key, err := keyStore.AccountDecryptedKey(acc, TestConfig.Account1.Password)
	s.NoError(err, "can not obtain decrypted acc key")
	extChild2String := key.ExtendedKey.String()

	s.NoError(keyStore.Delete(acc, TestConfig.Account1.Password), "cannot remove acc")

	walletAddressCheck, walletPubKeyCheck, chatAddressCheck, chatPubKeyCheck, err = s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover acc failed (for non-cached acc)")
	s.True(walletAddress == walletAddressCheck &&
		walletPubKey == walletPubKeyCheck &&
		chatAddress == chatAddressCheck &&
		chatPubKey == chatPubKeyCheck,
		"incorrect acc details recovered (for non-cached acc)")

	// make sure that extended key exists and is imported ok too
	_, key, err = keyStore.AccountDecryptedKey(acc, TestConfig.Account1.Password)
	s.NoError(err)
	s.Equal(extChild2String, key.ExtendedKey.String(), "CKD#2 key mismatch")

	// make sure that calling import several times, just returns from cache (no error is expected)
	walletAddressCheck, walletPubKeyCheck, chatAddressCheck, chatPubKeyCheck, err = s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover acc failed (for non-cached acc)")
	s.True(walletAddress == walletAddressCheck &&
		walletPubKey == walletPubKeyCheck &&
		chatAddress == chatAddressCheck &&
		chatPubKey == chatPubKeyCheck,
		"incorrect acc details recovered (for non-cached acc)")
}

func (s *AccountsTestSuite) TestSelectAccount() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	// create an account
	walletAddress1, walletPubKey1, chatAddress1, chatPubKey1, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {walletAddress: %s, walletKey: %s, chatAddress: %s, chatKey: %s}",
		walletAddress1, walletPubKey1, chatAddress1, chatPubKey1)

	walletAddress2, walletPubKey2, chatAddress2, chatPubKey2, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {walletAddress: %s, walletKey: %s, chatAddress: %s, chatKey: %s}",
		walletAddress2, walletPubKey2, chatAddress2, chatPubKey2)

	// try selecting with wrong password
	err = s.Backend.SelectAccount(walletAddress1, chatAddress1, "wrongPassword")
	expectedErr := errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase")
	s.EqualError(expectedErr, err.Error(), "select account is expected to throw error: wrong password used")

	err = s.Backend.SelectAccount(walletAddress1, chatAddress1, TestConfig.Account1.Password)
	s.NoError(err)

	// select another account, make sure that previous account is wiped out from Whisper cache
	s.NoError(s.Backend.SelectAccount(walletAddress2, chatAddress2, TestConfig.Account1.Password))
}

func (s *AccountsTestSuite) TestSelectedAccountOnRestart() {
	s.StartTestBackend()

	// create test accounts
	walletAddress1, _, chatAddress1, _, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	walletAddress2, _, chatAddress2, _, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// make sure that no account is selected by default
	selectedWalletAccount, err := s.Backend.AccountManager().SelectedWalletAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error(), "account selected, but should not be")
	s.Nil(selectedWalletAccount)
	selectedChatAccount, err := s.Backend.AccountManager().SelectedChatAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error(), "account selected, but should not be")
	s.Nil(selectedChatAccount)

	// select account
	err = s.Backend.SelectAccount(walletAddress1, chatAddress1, "wrongPassword")
	expectedErr := errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase")
	s.EqualError(expectedErr, err.Error())

	s.NoError(s.Backend.SelectAccount(walletAddress2, chatAddress2, TestConfig.Account1.Password))

	// stop node (and all of its sub-protocols)
	nodeConfig := s.Backend.StatusNode().Config()
	s.NotNil(nodeConfig)
	preservedNodeConfig := *nodeConfig
	s.NoError(s.Backend.StopNode())

	// make sure that account is still selected
	selectedWalletAccount, err = s.Backend.AccountManager().SelectedWalletAccount()
	s.Require().NoError(err)
	s.NotNil(selectedWalletAccount)
	s.Equal(selectedWalletAccount.Address.Hex(), walletAddress2, "incorrect wallet address selected")
	selectedChatAccount, err = s.Backend.AccountManager().SelectedChatAccount()
	s.NoError(err)
	s.NotNil(selectedChatAccount)
	s.Equal(selectedChatAccount.Address.Hex(), chatAddress2, "incorrect chat address selected")

	// resume node
	s.Require().NoError(s.Backend.StartNode(&preservedNodeConfig))

	// re-check selected account (account2 MUST be selected)
	selectedWalletAccount, err = s.Backend.AccountManager().SelectedWalletAccount()
	s.NoError(err)
	s.NotNil(selectedWalletAccount)
	s.Equal(selectedWalletAccount.Address.Hex(), walletAddress2, "incorrect wallet address selected")
	selectedChatAccount, err = s.Backend.AccountManager().SelectedChatAccount()
	s.NoError(err)
	s.NotNil(selectedChatAccount)
	s.Equal(selectedChatAccount.Address.Hex(), walletAddress2, "incorrect chat address selected")

	// now restart node using RestartNode() method, and make sure that account is still available
	s.RestartTestNode()
	defer s.StopTestBackend()

	// now logout, and make sure that on restart no account is selected (i.e. logout works properly)
	s.NoError(s.Backend.Logout())
	s.RestartTestNode()

	selectedWalletAccount, err = s.Backend.AccountManager().SelectedWalletAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error())
	s.Nil(selectedWalletAccount)
	selectedChatAccount, err = s.Backend.AccountManager().SelectedChatAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error())
	s.Nil(selectedChatAccount)
}
