package registry

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type RegistryVerifier struct {
	rc         *RegistryCaller
	pendingOpt bool
}

func NewVerifier(contractCaller bind.ContractCaller, contractAddress common.Address) (*RegistryVerifier, error) {
	rc, err := NewRegistryCaller(contractAddress, contractCaller)
	if err != nil {
		return nil, err
	}

	return &RegistryVerifier{
		rc: rc,
	}, nil
}

func (v *RegistryVerifier) VerifyNode(_ context.Context, nodeID discover.NodeID) bool {
	res, err := v.rc.Exists(&bind.CallOpts{Pending: v.pendingOpt}, nodeID.Bytes())
	if err != nil {
		return false
	}

	return res
}
