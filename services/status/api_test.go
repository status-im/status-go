package status

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/account"
	"github.com/stretchr/testify/suite"
)

func TestStatusSuite(t *testing.T) {
	suite.Run(t, new(StatusSuite))
}

type StatusSuite struct {
	suite.Suite
	am  *MockAccountManager
	w   *MockWhisperService
	api *PublicAPI
}

func (s *StatusSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.am = NewMockAccountManager(ctrl)
	s.w = NewMockWhisperService(ctrl)
	service := New(s.w)
	service.SetAccountManager(s.am)

	s.api = NewAPI(service)
}

var logintests = []struct {
	name                string
	expectedAddressKey  string
	expectedError       error
	prepareExpectations func(*StatusSuite)
}{
	{
		name:               "success login",
		expectedAddressKey: "addressKey",
		expectedError:      nil,
		prepareExpectations: func(s *StatusSuite) {
			key := keystore.Key{
				PrivateKey: &ecdsa.PrivateKey{},
			}
			s.am.EXPECT().AddressToDecryptedAccount("address...", "password").Return(accounts.Account{}, &key, nil)
			s.w.EXPECT().AddKeyPair(key.PrivateKey).Return("addressKey", nil)
			s.am.EXPECT().SelectAccount("address...", "address...", "password").Return(nil)
		},
	},
	{
		name:               "error when decrypting account from address",
		expectedAddressKey: "",
		expectedError:      errors.New("foo"),
		prepareExpectations: func(s *StatusSuite) {
			key := keystore.Key{
				PrivateKey: &ecdsa.PrivateKey{},
			}
			s.am.EXPECT().AddressToDecryptedAccount("address...", "password").Return(accounts.Account{}, &key, errors.New("foo"))
		},
	},
	{
		name:               "error when adding key pair to whisper",
		expectedAddressKey: "",
		expectedError:      errors.New("foo"),
		prepareExpectations: func(s *StatusSuite) {
			key := keystore.Key{
				PrivateKey: &ecdsa.PrivateKey{},
			}
			s.am.EXPECT().AddressToDecryptedAccount("address...", "password").Return(accounts.Account{}, &key, nil)
			s.w.EXPECT().AddKeyPair(key.PrivateKey).Return("", errors.New("foo"))
		},
	},
	{
		name:               "error when selecting account",
		expectedAddressKey: "",
		expectedError:      errors.New("foo"),
		prepareExpectations: func(s *StatusSuite) {
			key := keystore.Key{
				PrivateKey: &ecdsa.PrivateKey{},
			}
			s.am.EXPECT().AddressToDecryptedAccount("address...", "password").Return(accounts.Account{}, &key, nil)
			s.w.EXPECT().AddKeyPair(key.PrivateKey).Return("", nil)
			s.am.EXPECT().SelectAccount("address...", "address...", "password").Return(errors.New("foo"))
		},
	},
}

func (s *StatusSuite) TestLogin() {
	for _, t := range logintests {
		req := LoginRequest{Addr: "address...", Password: "password"}

		t.prepareExpectations(s)

		var ctx context.Context
		res, err := s.api.Login(ctx, req)
		s.Equal(t.expectedAddressKey, res.AddressKeyID, "failed scenario : "+t.name)
		s.Equal(t.expectedError, err, "failed scenario : "+t.name)
	}
}

var signuptests = []struct {
	name                string
	expectedResponse    SignupResponse
	expectedError       error
	prepareExpectations func(*StatusSuite)
}{
	{
		name: "success signup",
		expectedResponse: SignupResponse{
			WalletAddress: "addr",
			WalletPubkey:  "pubkey",
			Mnemonic:      "mnemonic",
		},
		expectedError: nil,
		prepareExpectations: func(s *StatusSuite) {
			accountInfo := account.Info{
				WalletAddress: "addr",
				WalletPubKey:  "pubkey",
				ChatAddress:   "addr",
				ChatPubKey:    "pubkey",
			}
			s.am.EXPECT().CreateAccount("password").Return(accountInfo, "mnemonic", nil)
		},
	},
	{
		name: "success signup",
		expectedResponse: SignupResponse{
			WalletAddress: "",
			WalletPubkey:  "",
			Mnemonic:      "",
		},
		expectedError: errors.New("could not create the specified account : foo"),
		prepareExpectations: func(s *StatusSuite) {
			s.am.EXPECT().CreateAccount("password").Return(account.Info{}, "", errors.New("foo"))
		},
	},
}

func (s *StatusSuite) TestSignup() {
	for _, t := range signuptests {
		t.prepareExpectations(s)

		var ctx context.Context
		res, err := s.api.Signup(ctx, SignupRequest{Password: "password"})
		s.Equal(t.expectedResponse.WalletAddress, res.WalletAddress, "failed scenario : "+t.name)
		s.Equal(t.expectedResponse.WalletPubkey, res.WalletPubkey, "failed scenario : "+t.name)
		s.Equal(t.expectedResponse.Mnemonic, res.Mnemonic, "failed scenario : "+t.name)
		s.Equal(t.expectedError, err, "failed scenario : "+t.name)
	}
}
