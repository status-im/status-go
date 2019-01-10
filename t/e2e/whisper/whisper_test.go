package whisper

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/account"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
)

func TestWhisperTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperTestSuite))
}

type WhisperTestSuite struct {
	e2e.BackendTestSuite
}

// TODO(adam): can anyone explain what this test is testing?
// I don't see any race condition testing here.
func (s *WhisperTestSuite) TestWhisperFilterRace() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	whisperService, err := s.Backend.StatusNode().WhisperService()
	s.NoError(err)

	accountManager := account.NewManager(s.Backend.StatusNode())
	s.NotNil(accountManager)

	whisperAPI := whisper.NewPublicWhisperAPI(whisperService)

	// account1
	_, accountKey1, err := accountManager.AddressToDecryptedAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)
	accountKey1Byte := crypto.FromECDSAPub(&accountKey1.PrivateKey.PublicKey)

	key1ID, err := whisperService.AddKeyPair(accountKey1.PrivateKey)
	s.NoError(err)
	ok := whisperAPI.HasKeyPair(context.Background(), key1ID)
	s.True(ok, "identity not injected")

	// account2
	_, accountKey2, err := accountManager.AddressToDecryptedAccount(TestConfig.Account2.Address, TestConfig.Account2.Password)
	s.NoError(err)
	key2ID, err := whisperService.AddKeyPair(accountKey2.PrivateKey)
	s.NoError(err)
	ok = whisperAPI.HasKeyPair(context.Background(), key2ID)
	s.True(ok, "identity not injected")

	// race filter addition
	filterAdded := make(chan struct{})
	allFiltersAdded := make(chan struct{})

	go func() {
		counter := 10
		for range filterAdded {
			counter--
			if counter <= 0 {
				break
			}
		}

		close(allFiltersAdded)
	}()

	for i := 0; i < 10; i++ {
		go func() {
			// nolint: errcheck
			whisperAPI.NewMessageFilter(whisper.Criteria{
				Sig:          accountKey1Byte,
				PrivateKeyID: key2ID,
				Topics: []whisper.TopicType{
					{0x4e, 0x03, 0x65, 0x7a},
					{0x34, 0x60, 0x7c, 0x9b},
					{0x21, 0x41, 0x7d, 0xf9},
				},
			})
			filterAdded <- struct{}{}
		}()
	}

	<-allFiltersAdded
}

func (s *WhisperTestSuite) TestSelectAccount() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	whisperService, err := s.Backend.StatusNode().WhisperService()
	s.NoError(err)

	// create an acc
	address, pubKey, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// make sure that identity is not (yet injected)
	s.False(whisperService.HasKeyPair(pubKey), "identity already present in whisper")

	// try selecting with wrong password
	err = s.Backend.SelectAccount(address, "wrongPassword")
	s.NotNil(err)

	// select another account, make sure that previous account is wiped out from Whisper cache
	s.False(whisperService.HasKeyPair(pubKey), "identity already present in whisper")
	s.NoError(s.Backend.SelectAccount(address, TestConfig.Account1.Password))
	s.True(whisperService.HasKeyPair(pubKey), "identity not injected into whisper")
}

func (s *WhisperTestSuite) TestLogout() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	whisperService, err := s.Backend.StatusNode().WhisperService()
	s.NoError(err)

	// create an account
	address, pubKey, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// make sure that identity doesn't exist (yet) in Whisper
	s.False(whisperService.HasKeyPair(pubKey), "identity already present in whisper")
	s.NoError(s.Backend.SelectAccount(address, TestConfig.Account1.Password))
	s.True(whisperService.HasKeyPair(pubKey), "identity not injected into whisper")

	s.NoError(s.Backend.Logout())
	s.False(whisperService.HasKeyPair(pubKey), "identity not cleared from whisper")
}

func (s *WhisperTestSuite) TestSelectedAccountOnRestart() {
	s.StartTestBackend()

	// we need to make sure that selected account is injected as identity into Whisper
	whisperService := s.WhisperService()

	// create test accounts
	address1, pubKey1, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	address2, pubKey2, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account2.Password)
	s.NoError(err)

	// make sure that identity is not (yet injected)
	s.False(whisperService.HasKeyPair(pubKey1), "identity already present in whisper")

	// make sure that no account is selected by default
	selectedWalletAccount, err := s.Backend.AccountManager().SelectedWalletAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error(), "account selected, but should not be")
	s.Nil(selectedWalletAccount)

	// make sure that no chat account is selected by default
	selectedChatAccount, err := s.Backend.AccountManager().SelectedChatAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error(), "account selected, but should not be")
	s.Nil(selectedChatAccount)

	// select account with wrong password
	err = s.Backend.SelectAccount(address1, "wrongPassword")
	expectedErr := errors.New("cannot retrieve a valid key for a given account: could not decrypt key with given passphrase")
	s.EqualError(expectedErr, err.Error())

	// select account with right password
	s.NoError(s.Backend.SelectAccount(address1, TestConfig.Account1.Password))
	selectedChatAccount1, err := s.Backend.AccountManager().SelectedChatAccount()
	s.NoError(err)
	selectedChatPubKey1 := hexutil.Encode(crypto.FromECDSAPub(&selectedChatAccount1.AccountKey.PrivateKey.PublicKey))
	s.Equal(selectedChatPubKey1, pubKey1)
	s.True(whisperService.HasKeyPair(selectedChatPubKey1), "identity not injected into whisper")

	// select another account, make sure that previous account is wiped out from Whisper cache
	s.False(whisperService.HasKeyPair(pubKey2), "identity already present in whisper")
	s.NoError(s.Backend.SelectAccount(address2, TestConfig.Account2.Password))
	selectedChatAccount2, err := s.Backend.AccountManager().SelectedChatAccount()
	s.NoError(err)
	selectedChatPubKey2 := hexutil.Encode(crypto.FromECDSAPub(&selectedChatAccount2.AccountKey.PrivateKey.PublicKey))
	s.Equal(selectedChatPubKey2, pubKey2)
	s.True(whisperService.HasKeyPair(selectedChatPubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(selectedChatPubKey1), "identity should be removed, but it is still present in whisper")

	// stop node (and all of its sub-protocols)
	nodeConfig := s.Backend.StatusNode().Config()
	s.NotNil(nodeConfig)
	preservedNodeConfig := *nodeConfig
	s.NoError(s.Backend.StopNode())

	// resume node
	s.Require().NoError(s.Backend.StartNode(&preservedNodeConfig))

	// re-check selected account (account2 MUST be selected)
	selectedWalletAccount, err = s.Backend.AccountManager().SelectedWalletAccount()
	s.NoError(err)
	s.NotNil(selectedWalletAccount)
	s.Equal(selectedWalletAccount.Address.Hex(), address2, "incorrect address selected")

	// make sure that Whisper gets identity re-injected
	whisperService = s.WhisperService()
	s.True(whisperService.HasKeyPair(selectedChatPubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(selectedChatPubKey1), "identity should not be present, but it is still present in whisper")

	// now restart node using RestartNode() method, and make sure that account is still available
	s.RestartTestNode()
	defer s.StopTestBackend()

	whisperService = s.WhisperService()
	s.True(whisperService.HasKeyPair(selectedChatPubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(selectedChatPubKey1), "identity should not be present, but it is still present in whisper")

	// now logout, and make sure that on restart no account is selected (i.e. logout works properly)
	s.NoError(s.Backend.Logout())
	s.RestartTestNode()
	whisperService = s.WhisperService()
	s.False(whisperService.HasKeyPair(selectedChatPubKey2), "identity not injected into whisper")
	s.False(whisperService.HasKeyPair(selectedChatPubKey1), "identity should not be present, but it is still present in whisper")

	selectedWalletAccount, err = s.Backend.AccountManager().SelectedWalletAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error())
	s.Nil(selectedWalletAccount)

	selectedChatAccount, err = s.Backend.AccountManager().SelectedChatAccount()
	s.EqualError(account.ErrNoAccountSelected, err.Error())
	s.Nil(selectedChatAccount)
}

func (s *WhisperTestSuite) TestSelectedChatKeyIsUsedInWhisper() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	whisperService, err := s.Backend.StatusNode().WhisperService()
	s.NoError(err)

	// create an account
	address, _, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// select account
	s.NoError(s.Backend.SelectAccount(address, TestConfig.Account1.Password))

	// Get the chat account
	selectedChatAccount, err := s.Backend.AccountManager().SelectedChatAccount()
	s.NoError(err)

	// chat key should be injected in whisper
	selectedChatPubKey := hexutil.Encode(crypto.FromECDSAPub(&selectedChatAccount.AccountKey.PrivateKey.PublicKey))
	s.True(whisperService.HasKeyPair(selectedChatPubKey), "identity not injected in whisper")
}
