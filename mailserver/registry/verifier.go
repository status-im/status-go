package registry

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

var logger = log.New("package", "mailserver/registry")

// Verifier verifies nodes based on a smart contract.
type Verifier struct {
	rc *RegistryCaller
}

// NewVerifier returns a new Verifier instance.
func NewVerifier(contractCaller bind.ContractCaller, contractAddress common.Address) (*Verifier, error) {
	logger.Debug("initializing mailserver registry verifier", "address", contractAddress.Hash())
	rc, err := NewRegistryCaller(contractAddress, contractCaller)
	if err != nil {
		logger.Debug("error initializing mailserver registry verifier", "address", fmt.Sprintf("%x", contractAddress))
		return nil, err
	}

	return &Verifier{
		rc: rc,
	}, nil
}

// VerifyNode checks if a given node is trusted using a smart contract.
func (v *Verifier) VerifyNode(_ context.Context, nodeID discover.NodeID) bool {
	res, err := v.rc.Exists(nil, nodeID.Bytes())
	logger.Debug("verifying node", "id", fmt.Sprintf("%x", nodeID.Bytes()), "verified", res)
	if err != nil {
		logger.Error("error verifying node", "id", fmt.Sprintf("%x", nodeID.Bytes()), "error", err)
		return false
	}

	return res
}
