package registry

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// Verifier verifies nodes based on a smart contract.
type Verifier struct {
	rc *RegistryCaller
}

// NewVerifier returns a new Verifier instance.
func NewVerifier(contractCaller bind.ContractCaller, contractAddress common.Address) (*Verifier, error) {
	rc, err := NewRegistryCaller(contractAddress, contractCaller)
	if err != nil {
		return nil, err
	}

	return &Verifier{
		rc: rc,
	}, nil
}

// VerifyNode checks if a given node is trusted using a smart contract.
func (v *Verifier) VerifyNode(_ context.Context, nodeID discover.NodeID) bool {
	res, err := v.rc.Exists(nil, nodeID.Bytes())
	if err != nil {
		return false
	}

	return res
}
