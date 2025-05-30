package personal

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	accounttypes "github.com/status-im/status-go/account/types"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

// Make sure that Service implements node.Service interface.
var _ node.Lifecycle = (*Service)(nil)

// Service represents out own implementation of personal sign operations.
type Service struct {
}

// New returns a new Service.
func New() *Service {
	return &Service{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "personal",
			Version:   "1.0",
			Service:   NewAPI(s),
			Public:    false,
		},
	}
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Start() error {
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Stop() error {
	return nil
}

// Recover is an implementation of `personal_ecRecover` or `web3.personal.ecRecover` API
func (s *Service) Recover(rpcParams RecoverParams) (addr types.Address, err error) {
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
func (s *Service) CanRecover(rpcParams RecoverParams, revealedAddress types.Address) (bool, error) {
	recovered, err := s.Recover(rpcParams)
	if err != nil {
		return false, err
	}
	return recovered == revealedAddress, nil
}

// Sign is an implementation of `personal_sign` or `web3.personal.sign` API
func (s *Service) Sign(rpcParams SignParams, verifiedAccount *accounttypes.SelectedExtKey) (result types.HexBytes, err error) {
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
