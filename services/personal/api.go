package personal

import (
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

var (
	ErrInvalidSignatureLength = errors.New("invalid signature, must be 65 bytes long")
	ErrInvalidSignatureV      = errors.New("invalid Ethereum signature (V is not 27 or 28)")
)

// PublicAPI represents a set of APIs from the `web3.personal` namespace.
type PublicAPI struct {
}

// NewAPI creates an instance of the personal API.
func NewAPI() *PublicAPI {
	return &PublicAPI{}
}

// Recover is an implementation of `personal_ecRecover` or `web3.personal.ecRecover` API
func (api *PublicAPI) Recover(rpcParams RecoverParams) (addr types.Address, err error) {
	message, err := hexutil.Decode(rpcParams.Message)
	if err != nil {
		return types.Address{}, err
	}
	sig, err := hexutil.Decode(rpcParams.Signature)
	if err != nil {
		return types.Address{}, err
	}

	if len(sig) != 65 {
		return types.Address{}, ErrInvalidSignatureLength
	}
	if sig[64] != 27 && sig[64] != 28 {
		return types.Address{}, ErrInvalidSignatureV
	}
	sig[64] -= 27 // Transform yellow paper V from 27/28 to 0/1
	hash := crypto.TextHash(message)
	rpk, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return types.Address{}, err
	}
	return crypto.PubkeyToAddress(*rpk), nil
}

// CanRecover is an implementation of `personal_ecRecover` or `web3.personal.ecRecover` API
func (api *PublicAPI) CanRecover(rpcParams RecoverParams, revealedAddress types.Address) (bool, error) {
	recovered, err := api.Recover(rpcParams)
	if err != nil {
		return false, err
	}
	return recovered == revealedAddress, nil
}

// Sign is an implementation of `personal_sign` or `web3.personal.sign` API
func (api *PublicAPI) Sign(rpcParams SignParams, verifiedAccount *account.SelectedExtKey) (result types.HexBytes, err error) {
	var dBytes []byte
	switch d := rpcParams.Data.(type) {
	case string:
		dBytes = []byte(d)
	case []byte:
		dBytes = d
	case byte:
		dBytes = []byte{d}
	}

	hash := crypto.TextHash(dBytes)

	sig, err := crypto.Sign(hash, verifiedAccount.AccountKey.PrivateKey)
	if err != nil {
		return types.HexBytes{}, err
	}
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

	return types.HexBytes(sig), err
}
