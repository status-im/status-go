package whisper

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestWhisperTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperTestSuite))
}

type WhisperTestSuite struct {
	e2e.BackendTestSuite
}

func (s *WhisperTestSuite) SetupTest() {
	s.BackendTestSuite.SetupTest()
}

// TODO(adam): can anyone explain what this test is testing?
// I don't see any race condition testing here.
func (s *WhisperTestSuite) TestWhisperFilterRace() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	address, _, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)
	// select account (sub-accounts will be created for this key)
	err = s.Backend.SelectAccount(address, TestConfig.Account1.Password)
	s.NoError(err, "account selection failed")

	whisperService, err := s.Backend.Whisper()
	s.NoError(err)
	whisperAPI := whisper.NewPublicWhisperAPI(whisperService)

	// account1
	_, accountKey1, err := s.Backend.AccountManager().AddressToDecryptedAccount(address, TestConfig.Account1.Password)
	s.NoError(err)
	accountKey1Byte := crypto.FromECDSAPub(&accountKey1.PrivateKey.PublicKey)

	key1ID, err := whisperService.AddKeyPair(accountKey1.PrivateKey)
	s.NoError(err)
	ok := whisperAPI.HasKeyPair(context.Background(), key1ID)
	s.True(ok, "identity not injected")

	// account2
	_, accountKey2, err := s.Backend.AccountManager().AddressToDecryptedAccount(address, TestConfig.Account2.Password)
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

func (s *WhisperTestSuite) TestLogout() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	whisperService, err := s.Backend.Whisper()
	s.NoError(err)

	accountManager := s.Backend.AccountManager()
	s.Nil(err)
	s.NotNil(accountManager)

	// create an account
	address, pubKey, _, err := accountManager.CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	// make sure that identity doesn't exist (yet) in Whisper
	s.False(whisperService.HasKeyPair(pubKey), "identity already present in whisper")
	s.NoError(s.Backend.SelectAccount(address, TestConfig.Account1.Password))
	s.True(whisperService.HasKeyPair(pubKey), "identity not injected into whisper")

	s.NoError(s.Backend.Logout())
	s.False(whisperService.HasKeyPair(pubKey), "identity not cleared from whisper")
}
