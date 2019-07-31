package accounts

import (
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/extkeys"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func buildLoginParams(mainAccountAddress, chatAddress, password string, watchAddresses []common.Address) account.LoginParams {
	return account.LoginParams{
		ChatAddress:    common.HexToAddress(chatAddress),
		Password:       password,
		MainAccount:    common.HexToAddress(mainAccountAddress),
		WatchAddresses: watchAddresses,
	}
}

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
	accountInfo, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// ensure that there is still no accounts returned
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Zero(len(accounts), "accounts returned, while there should be none (we haven't logged in yet)")

	// select account (sub-accounts will be created for this key)
	err = s.Backend.SelectAccount(buildLoginParams(accountInfo.WalletAddress, accountInfo.ChatAddress, TestConfig.Account1.Password, nil))
	s.NoError(err, "account selection failed")

	// at this point main account should show up
	accounts, err = s.Backend.AccountManager().Accounts()
	s.NoError(err)
	s.Equal(1, len(accounts), "exactly single account is expected (main account)")
	s.Equal(accounts[0].Hex(), accountInfo.WalletAddress,
		fmt.Sprintf("main account is not retured as the first key: got %s, expected %s", accounts[0].Hex(), "0x"+accountInfo.WalletAddress))
}

func (s *AccountsTestSuite) TestImportSingleExtendedKey() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	keyStore := s.Backend.AccountManager().GetKeystore()
	s.NotNil(keyStore)

	// create a master extended key
	mn := extkeys.NewMnemonic()
	mnemonic, err := mn.MnemonicPhrase(extkeys.EntropyStrength128, extkeys.EnglishLanguage)
	s.NoError(err)
	extKey, err := extkeys.NewMaster(mn.MnemonicSeed(mnemonic, ""))
	s.NoError(err)
	derivedExtendedKey, err := extKey.EthBIP44Child(0)
	s.NoError(err)

	// import single extended key
	password := "test-password-1"
	addr, _, err := s.Backend.AccountManager().ImportSingleExtendedKey(derivedExtendedKey, password)
	s.NoError(err)

	_, key, err := s.Backend.AccountManager().AddressToDecryptedAccount(addr, password)
	s.NoError(err)

	s.Equal(crypto.FromECDSA(key.PrivateKey), crypto.FromECDSA(key.ExtendedKey.ToECDSA()))
}

func (s *AccountsTestSuite) TestImportAccount() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	keyStore := s.Backend.AccountManager().GetKeystore()
	s.NotNil(keyStore)

	// create a private key
	privateKey, err := crypto.GenerateKey()
	s.NoError(err)

	// import as normal account
	password := "test-password-2"
	addr, err := s.Backend.AccountManager().ImportAccount(privateKey, password)
	s.Require().NoError(err)

	_, key, err := s.Backend.AccountManager().AddressToDecryptedAccount(addr.String(), password)
	s.NoError(err)

	s.Equal(crypto.FromECDSA(privateKey), crypto.FromECDSA(key.PrivateKey))
	s.True(key.ExtendedKey.IsZeroed())
}

func (s *AccountsTestSuite) TestRecoverAccount() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	keyStore := s.Backend.AccountManager().GetKeystore()
	s.NotNil(keyStore)

	// create an acc
	accountInfo, mnemonic, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {walletAddress: %s, walletKey: %s, chatAddress: %s, chatKey: %s, mnemonic:%s}",
		accountInfo.WalletAddress, accountInfo.WalletPubKey, accountInfo.ChatAddress, accountInfo.ChatPubKey, mnemonic)

	// try recovering using password + mnemonic
	accountInfoCheck, err := s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover acc failed")

	s.EqualValues(accountInfo, accountInfoCheck, "incorrect accound details recovered")

	// now test recovering, but make sure that acc/key file is removed i.e. simulate recovering on a new device
	acc, err := account.ParseAccountString(accountInfo.WalletAddress)
	s.NoError(err, "can not get acc from address")

	acc, key, err := keyStore.AccountDecryptedKey(acc, TestConfig.Account1.Password)
	s.NoError(err, "can not obtain decrypted acc key")
	extChild2String := key.ExtendedKey.String()

	s.NoError(keyStore.Delete(acc, TestConfig.Account1.Password), "cannot remove acc")

	accountInfoCheck, err = s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover acc failed (for non-cached acc)")
	s.EqualValues(accountInfo, accountInfoCheck, "incorrect acc details recovered (for non-cached acc)")

	// make sure that extended key exists and is imported ok too
	_, key, err = keyStore.AccountDecryptedKey(acc, TestConfig.Account1.Password)
	s.NoError(err)
	s.Equal(extChild2String, key.ExtendedKey.String(), "CKD#2 key mismatch")

	// make sure that calling import several times, just returns from cache (no error is expected)
	accountInfoCheck, err = s.Backend.AccountManager().RecoverAccount(TestConfig.Account1.Password, mnemonic)
	s.NoError(err, "recover acc failed (for non-cached acc)")
	s.EqualValues(accountInfo, accountInfoCheck, "incorrect acc details recovered (for non-cached acc)")
}

func (s *AccountsTestSuite) TestSelectAccount() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	// create an account
	accountInfo1, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {walletAddress: %s, walletKey: %s, chatAddress: %s, chatKey: %s}",
		accountInfo1.WalletAddress, accountInfo1.WalletPubKey, accountInfo1.ChatAddress, accountInfo1.ChatPubKey)

	accountInfo2, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	s.T().Logf("Account created: {walletAddress: %s, walletKey: %s, chatAddress: %s, chatKey: %s}",
		accountInfo2.WalletAddress, accountInfo2.WalletPubKey, accountInfo2.ChatAddress, accountInfo2.ChatPubKey)

	// try selecting with wrong password
	err = s.Backend.SelectAccount(buildLoginParams(accountInfo1.WalletAddress, accountInfo1.ChatAddress, "wrongPassword", nil))
	expectedErr := errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase")
	s.EqualError(expectedErr, err.Error(), "select account is expected to throw error: wrong password used")

	err = s.Backend.SelectAccount(buildLoginParams(accountInfo1.WalletAddress, accountInfo1.ChatAddress, TestConfig.Account1.Password, nil))
	s.NoError(err)

	// select another account, make sure that previous account is wiped out from Whisper cache
	s.NoError(s.Backend.SelectAccount(buildLoginParams(accountInfo2.WalletAddress, accountInfo2.ChatAddress, TestConfig.Account1.Password, nil)))
}

func (s *AccountsTestSuite) TestSetChatAccount() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	chatPrivKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	chatPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(chatPrivKey))

	encryptionPrivKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	encryptionPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(encryptionPrivKey))

	err = s.Backend.InjectChatAccount(chatPrivKeyHex, encryptionPrivKeyHex)
	s.Require().NoError(err)

	selectedChatAccount, err := s.Backend.AccountManager().SelectedChatAccount()
	s.Require().NoError(err)
	s.Equal(chatPrivKey, selectedChatAccount.AccountKey.PrivateKey)
}

func (s *AccountsTestSuite) TestSelectedAccountOnRestart() {
	s.StartTestBackend()

	// create test accounts
	accountInfo1, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	accountInfo2, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// make sure that no account is selected by default
	selectedWalletAccount, err := s.Backend.AccountManager().MainAccountAddress()
	s.EqualError(account.ErrNoAccountSelected, err.Error(), "account selected, but should not be")
	s.Equal(common.Address{}, selectedWalletAccount)
	selectedChatAccount, err := s.Backend.AccountManager().SelectedChatAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error(), "account selected, but should not be")
	s.Nil(selectedChatAccount)

	// select account
	err = s.Backend.SelectAccount(buildLoginParams(accountInfo1.WalletAddress, accountInfo1.ChatAddress, "wrongPassword", nil))
	expectedErr := errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase")
	s.EqualError(expectedErr, err.Error())

	watchAddresses := []common.Address{
		common.HexToAddress("0x00000000000000000000000000000000000001"),
		common.HexToAddress("0x00000000000000000000000000000000000002"),
	}
	s.NoError(s.Backend.SelectAccount(buildLoginParams(accountInfo2.WalletAddress, accountInfo2.ChatAddress, TestConfig.Account1.Password, watchAddresses)))

	// stop node (and all of its sub-protocols)
	nodeConfig := s.Backend.StatusNode().Config()
	s.NotNil(nodeConfig)
	preservedNodeConfig := *nodeConfig
	s.NoError(s.Backend.StopNode())

	// make sure that account is still selected
	selectedWalletAccount, err = s.Backend.AccountManager().MainAccountAddress()
	s.Require().NoError(err)
	s.NotNil(selectedWalletAccount)
	s.Equal(selectedWalletAccount.String(), accountInfo2.WalletAddress, "incorrect wallet address selected")
	selectedChatAccount, err = s.Backend.AccountManager().SelectedChatAccount()
	s.NoError(err)
	s.NotNil(selectedChatAccount)
	s.Equal(selectedChatAccount.Address.Hex(), accountInfo2.ChatAddress, "incorrect chat address selected")
	s.Equal(watchAddresses, s.Backend.AccountManager().WatchAddresses())

	// resume node
	s.Require().NoError(s.Backend.StartNode(&preservedNodeConfig))

	// re-check selected account (account2 MUST be selected)
	selectedWalletAccount, err = s.Backend.AccountManager().MainAccountAddress()
	s.NoError(err)
	s.NotNil(selectedWalletAccount)
	s.Equal(selectedWalletAccount.String(), accountInfo2.WalletAddress, "incorrect wallet address selected")
	selectedChatAccount, err = s.Backend.AccountManager().SelectedChatAccount()
	s.NoError(err)
	s.NotNil(selectedChatAccount)
	s.Equal(selectedChatAccount.Address.Hex(), accountInfo2.WalletAddress, "incorrect chat address selected")
	s.Equal(watchAddresses, s.Backend.AccountManager().WatchAddresses())

	// now restart node using RestartNode() method, and make sure that account is still available
	s.RestartTestNode()
	defer s.StopTestBackend()

	// now logout, and make sure that on restart no account is selected (i.e. logout works properly)
	s.NoError(s.Backend.Logout())
	s.RestartTestNode()

	selectedWalletAccount, err = s.Backend.AccountManager().MainAccountAddress()
	s.EqualError(account.ErrNoAccountSelected, err.Error())
	s.Equal(common.Address{}, selectedWalletAccount)
	selectedChatAccount, err = s.Backend.AccountManager().SelectedChatAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error())
	s.Nil(selectedChatAccount)
	s.Len(s.Backend.AccountManager().WatchAddresses(), 0)
}
