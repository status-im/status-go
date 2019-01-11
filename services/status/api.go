package status

import (
	"context"
	"errors"
	"fmt"

	"github.com/status-im/status-go/account"
)

// PublicAPI represents a set of APIs from the `web3.status` namespace.
type PublicAPI struct {
	s *Service
}

// NewAPI creates an instance of the status API.
func NewAPI(s *Service) *PublicAPI {
	return &PublicAPI{s: s}
}

// LoginRequest : json request for status_login.
type LoginRequest struct {
	Addr     string `json:"address"`
	Password string `json:"password"`
}

// LoginResponse : json response returned by status_login.
type LoginResponse struct {
	AddressKeyID string `json:"address_key_id"`
}

// Login is an implementation of `status_login` or `web3.status.login` API
func (api *PublicAPI) Login(context context.Context, req LoginRequest) (res LoginResponse, err error) {
	_, accountKey, err := api.s.am.AddressToDecryptedAccount(req.Addr, req.Password)
	if err != nil {
		return
	}

	if res.AddressKeyID, err = api.s.w.AddKeyPair(accountKey.PrivateKey); err != nil {
		return
	}

	if err = api.s.am.SelectAccount(req.Addr, req.Addr, req.Password); err != nil {
		return
	}

	return
}

// SignupRequest : json request for status_signup.
type SignupRequest struct {
	Password string `json:"password"`
}

// SignupResponse : json response returned by status_signup.
type SignupResponse struct {
	WalletAddress string `json:"address"`
	WalletPubkey  string `json:"pubkey"`
	ChatAddress   string `json:"chatAddress"`
	ChatPubkey    string `json:"chatPubkey"`
	Mnemonic      string `json:"mnemonic"`
}

// Signup is an implementation of `status_signup` or `web3.status.signup` API
func (api *PublicAPI) Signup(context context.Context, req SignupRequest) (res SignupResponse, err error) {
	if res.WalletAddress, res.WalletPubkey, res.ChatAddress, res.ChatPubkey, res.Mnemonic, err = api.s.am.CreateAccount(req.Password); err != nil {
		err = errors.New("could not create the specified account : " + err.Error())
		return
	}

	return
}

// CreateAddressResponse : json response returned by status_createaccount
type CreateAddressResponse struct {
	Address string `json:"address"`
	Pubkey  string `json:"pubkey"`
	Privkey string `json:"privkey"`
}

// CreateAddress is an implementation of `status_createaccount` or `web3.status.createaccount` API
func (api *PublicAPI) CreateAddress(context context.Context) (res CreateAddressResponse, err error) {
	if res.Address, res.Pubkey, res.Privkey, err = account.CreateAddress(); err != nil {
		err = fmt.Errorf("could not create an address:  %v", err)
	}

	return
}
