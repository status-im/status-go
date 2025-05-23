package personal

import (
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
)

type MockedPersonalAPI struct {
}

func NewMockedAPI() *MockedPersonalAPI {
	return &MockedPersonalAPI{}
}

func (api *MockedPersonalAPI) Recover(rpcParams RecoverParams) (addr types.Address, err error) {
	sig := types.HexBytes(rpcParams.Signature)
	if len(sig) != 65 {
		return types.Address{}, ErrInvalidSignatureLength
	}
	if sig[64] != 27 && sig[64] != 28 {
		return types.Address{}, ErrInvalidSignatureV
	}
	sig[64] -= 27 // Transform yellow paper V from 27/28 to 0/1
	hash := crypto.TextHash(types.HexBytes(rpcParams.Message))
	rpk, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return types.Address{}, err
	}
	return crypto.PubkeyToAddress(*rpk), nil
}

func (api *MockedPersonalAPI) CanRecover(rpcParams RecoverParams, revealedAddress types.Address) (bool, error) {
	return true, nil
}

func (api *MockedPersonalAPI) Sign(rpcParams SignParams, verifiedAccount *account.SelectedExtKey) (result types.HexBytes, err error) {
	bytesArray := []byte(rpcParams.Address)
	bytesArray = append(bytesArray, []byte(rpcParams.Password)...)
	bytesArray = common.Shake256(bytesArray)
	return append([]byte{0}, bytesArray...), nil
}
