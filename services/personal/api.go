package personal

import (
	"errors"

	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/eth-node/types"
)

var (
	ErrInvalidSignatureLength = errors.New("invalid signature, must be 65 bytes long")
	ErrInvalidSignatureV      = errors.New("invalid Ethereum signature (V is not 27 or 28)")
)

// PublicAPI represents a set of APIs from the `web3.personal` namespace.
type PublicAPI struct {
	s *Service
}

// NewAPI creates an instance of the personal API.
func NewAPI(s *Service) *PublicAPI {
	return &PublicAPI{s}
}

// Recover is an implementation of `personal_ecRecover` or `web3.personal.ecRecover` API
func (api *PublicAPI) Recover(rpcParams RecoverParams) (addr types.Address, err error) {
	return api.s.Recover(rpcParams)
}

// CanRecover is an implementation of `personal_ecRecover` or `web3.personal.ecRecover` API
func (api *PublicAPI) CanRecover(rpcParams RecoverParams, revealedAddress types.Address) (bool, error) {
	return api.s.CanRecover(rpcParams, revealedAddress)
}

// Sign is an implementation of `personal_sign` or `web3.personal.sign` API
func (api *PublicAPI) Sign(rpcParams SignParams, verifiedAccount *generator.Account) (result types.HexBytes, err error) {
	return api.s.Sign(rpcParams, verifiedAccount)
}
